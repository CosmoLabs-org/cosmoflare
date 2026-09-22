package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// errStubConn stands in for a connection error at the HTTP seam.
var errStubConn = errors.New("stub connection refused")

// mustMarshalForTags marshals v (or fails the test) so field-tag shape can
// be asserted without re-implementing the encoder.
func mustMarshalForTags(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// --- Command registration ---

func TestPagesDeployCmd_RegisteredOnPages(t *testing.T) {
	found := false
	for _, sub := range pagesCmd.Commands() {
		if sub.Name() == "deploy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pages deploy subcommand not registered on pagesCmd")
	}
	if pagesDeployCmd.RunE == nil {
		t.Fatal("pagesDeployCmd has nil RunE")
	}
}

func TestPagesDeployCmd_Flags(t *testing.T) {
	for _, name := range []string{"project", "branch", "verify", "build-id"} {
		if pagesDeployCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on pagesDeployCmd", name)
		}
	}
}

// --- URL list parsing ---

func TestParseVerifyPaths(t *testing.T) {
	cases := []struct {
		name  string
		specs []string
		want  []string
	}{
		{"nil specs", nil, []string{}},
		{"comma separated", []string{"/,/about,/health"}, []string{"/", "/about", "/health"}},
		{"repeatable", []string{"/", "/about", "/", "/about"}, []string{"/", "/about"}},
		{"strips empties and blanks", []string{" , /a ,,", ""}, []string{"/a"}},
		{"dedupes across specs", []string{"/a,/b", "/b,/c"}, []string{"/a", "/b", "/c"}},
		{"relative paths kept as-is", []string{"about", "health.json"}, []string{"about", "health.json"}},
		{"absolute url passes through parsing", []string{"https://x.dev/a"}, []string{"https://x.dev/a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseVerifyPaths(tc.specs)
			if len(got) != len(tc.want) {
				t.Fatalf("parseVerifyPaths(%v) = %v, want %v", tc.specs, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("parseVerifyPaths(%v) = %v, want %v", tc.specs, got, tc.want)
				}
			}
		})
	}
}

// --- Poll decision (pure) ---

func TestPagesPollDecision(t *testing.T) {
	cases := []struct {
		name        string
		attempt     int
		ok          bool
		maxAttempts int
		want        pagesPollOutcome
	}{
		{"success is live", 1, true, 10, pagesPollLive},
		{"late success is live", 10, true, 10, pagesPollLive},
		{"early failure keeps polling", 1, false, 10, pagesPollKeep},
		{"mid failure keeps polling", 9, false, 10, pagesPollKeep},
		{"last failure fails", 10, false, 10, pagesPollFailed},
		{"beyond max fails", 11, false, 10, pagesPollFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pagesPollDecision(tc.attempt, tc.ok, tc.maxAttempts); got != tc.want {
				t.Fatalf("pagesPollDecision(%d, %v, %d) = %v, want %v", tc.attempt, tc.ok, tc.maxAttempts, got, tc.want)
			}
		})
	}
}

// --- Table renderer + JSON envelope shape ---

func TestRenderVerifyTable(t *testing.T) {
	res := pagesVerifyResult{
		DeployedURL: "https://abc.pages.dev",
		BuildID:     "deadbee",
		Checks: []pagesVerifyCheck{
			{Path: "/", Status: 200, OK: true, Ms: 42},
			{Path: "/about", Status: 404, OK: false, Ms: 7},
			{Path: "/health", Status: 0, OK: false, Ms: 3},
		},
	}
	var buf bytes.Buffer
	renderVerifyTable(&buf, res)
	out := buf.String()
	for _, want := range []string{
		"PATH", "STATUS", "MS", "RESULT",
		"/", "200", "42", "PASS",
		"/about", "404", "FAIL",
		"/health", "0", "FAIL",
		"https://abc.pages.dev",
		"deadbee",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("renderVerifyTable output missing %q:\n%s", want, out)
		}
	}
}

func TestPagesVerifyResultJSONShape(t *testing.T) {
	res := pagesVerifyResult{
		DeployedURL: "https://abc.pages.dev",
		BuildID:     "deadbee",
		Checks:      []pagesVerifyCheck{{Path: "/", Status: 200, OK: true, Ms: 42}},
	}
	// Field tags are the contract; verify via a targeted marshal of the
	// check struct and the envelope's exported shape.
	c := res.Checks[0]
	if c.Path != "/" || c.Status != 200 || !c.OK || c.Ms != 42 {
		t.Fatalf("check fields = %+v", c)
	}
	// JSON tags must match {deployed_url, build_id, checks:[{path,status,ok,ms}]}.
	importJSON := mustMarshalForTags(t, res)
	for _, want := range []string{
		`"deployed_url"`, `"build_id"`, `"checks"`, `"path"`, `"status"`, `"ok"`, `"ms"`,
	} {
		if !strings.Contains(importJSON, want) {
			t.Fatalf("JSON envelope missing key %s: %s", want, importJSON)
		}
	}
}

// --- Build-id fallback chain ---

func TestResolvePagesBuildID(t *testing.T) {
	t.Run("flag wins", func(t *testing.T) {
		restorePagesBuildIDSeams(t, "ENVVAL", "GITVAL")
		if got := resolvePagesBuildID("FLAGVAL"); got != "FLAGVAL" {
			t.Fatalf("resolvePagesBuildID(flag) = %q, want FLAGVAL", got)
		}
	})
	t.Run("env beats git", func(t *testing.T) {
		restorePagesBuildIDSeams(t, "ENVVAL", "GITVAL")
		if got := resolvePagesBuildID(""); got != "ENVVAL" {
			t.Fatalf("resolvePagesBuildID(env) = %q, want ENVVAL", got)
		}
	})
	t.Run("git fallback", func(t *testing.T) {
		restorePagesBuildIDSeams(t, "", "GITVAL")
		if got := resolvePagesBuildID(""); got != "GITVAL" {
			t.Fatalf("resolvePagesBuildID(git) = %q, want GITVAL", got)
		}
	})
	t.Run("empty when all missing", func(t *testing.T) {
		restorePagesBuildIDSeams(t, "", "")
		if got := resolvePagesBuildID(""); got != "" {
			t.Fatalf("resolvePagesBuildID(empty) = %q, want empty", got)
		}
	})
}

// restorePagesBuildIDSeams stubs the env and git seams for the build-id
// chain and restores them on cleanup.
func restorePagesBuildIDSeams(t *testing.T, envVal, gitVal string) {
	t.Helper()
	oldEnv, oldGit := pagesEnvLookup, pagesGitShortRev
	pagesEnvLookup = func(string) string { return envVal }
	pagesGitShortRev = func() string { return gitVal }
	t.Cleanup(func() {
		pagesEnvLookup = oldEnv
		pagesGitShortRev = oldGit
	})
}

// --- Poll loop behavior with stubbed HTTP + sleep ---

func TestPollPagesDeployment(t *testing.T) {
	oldGet, oldSleep := pagesVerifyGet, pagesVerifySleep
	t.Cleanup(func() { pagesVerifyGet, pagesVerifySleep = oldGet, oldSleep })
	pagesVerifySleep = func(time.Duration) {}

	t.Run("immediate 200 marks live", func(t *testing.T) {
		pagesVerifyGet = func(url string) (int, error) { return 200, nil }
		if !pollPagesDeployment("https://x.pages.dev", []string{"/"}) {
			t.Fatal("expected live on first 200")
		}
	})
	t.Run("any path 200 marks live", func(t *testing.T) {
		pagesVerifyGet = func(url string) (int, error) {
			if strings.HasSuffix(url, "/health") {
				return 200, nil
			}
			return 0, errStubConn
		}
		if !pollPagesDeployment("https://x.pages.dev", []string{"/", "/health"}) {
			t.Fatal("expected live when any verify path answers 200")
		}
	})
	t.Run("non-2xx never live", func(t *testing.T) {
		pagesVerifyGet = func(url string) (int, error) { return 503, nil }
		if pollPagesDeployment("https://x.pages.dev", []string{"/"}) {
			t.Fatal("expected timeout failure on persistent 503")
		}
	})
	t.Run("no paths polls deployment root", func(t *testing.T) {
		var seen []string
		pagesVerifyGet = func(url string) (int, error) {
			seen = append(seen, url)
			return 200, nil
		}
		if !pollPagesDeployment("https://x.pages.dev", nil) {
			t.Fatal("expected live with no paths given")
		}
		if len(seen) != 1 || seen[0] != "https://x.pages.dev/" {
			t.Fatalf("polled URLs = %v, want deployment root", seen)
		}
	})
}

// --- Joining paths onto the deployment URL ---

func TestJoinVerifyURL(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"https://x.pages.dev", "/", "https://x.pages.dev/"},
		{"https://x.pages.dev", "/about", "https://x.pages.dev/about"},
		{"https://x.pages.dev/", "/about", "https://x.pages.dev/about"},
		{"https://x.pages.dev", "about", "https://x.pages.dev/about"},
		{"https://x.pages.dev", "https://other.dev/a", "https://other.dev/a"},
		{"https://x.pages.dev", "http://other.dev/a", "http://other.dev/a"},
	}
	for _, tc := range cases {
		if got := joinVerifyURL(tc.base, tc.path); got != tc.want {
			t.Errorf("joinVerifyURL(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

// --- checkVerifyPaths with stubbed HTTP ---

func TestCheckVerifyPaths(t *testing.T) {
	oldGet := pagesVerifyGet
	t.Cleanup(func() { pagesVerifyGet = oldGet })
	pagesVerifyGet = func(url string) (int, error) {
		switch {
		case strings.HasSuffix(url, "/ok"):
			return 200, nil
		case strings.HasSuffix(url, "/redirect"):
			return 302, nil
		default:
			return 0, errStubConn
		}
	}
	checks := checkVerifyPaths("https://x.pages.dev", []string{"/ok", "/redirect", "/boom"})
	if len(checks) != 3 {
		t.Fatalf("got %d checks, want 3", len(checks))
	}
	if !checks[0].OK || checks[0].Status != 200 {
		t.Errorf("200 check = %+v, want ok", checks[0])
	}
	if checks[1].OK || checks[1].Status != 302 {
		t.Errorf("302 check = %+v, want not ok (non-2xx fails)", checks[1])
	}
	if checks[2].OK || checks[2].Status != 0 {
		t.Errorf("connection-error check = %+v, want not ok with status 0", checks[2])
	}
}

// --- Guards: missing creds and DryRun short-circuit ---

func TestRunPagesDeploy_MissingCreds(t *testing.T) {
	runGlobalsSnapshot(t)

	// Point at a real directory so the stat guard passes and the failure
	// comes from the creds guard, not the path check.
	err := runPagesDeploy(pagesDeployCmd, []string{"."})
	if err == nil {
		t.Fatal("runPagesDeploy with empty credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create Pages service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create Pages service")
	}
}

func TestRunPagesDeploy_MissingDirectory(t *testing.T) {
	runGlobalsSnapshot(t)

	err := runPagesDeploy(pagesDeployCmd, []string{"./no-such-dir-for-pages-deploy"})
	if err == nil {
		t.Fatal("runPagesDeploy with a missing directory should return an error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want it to mention the missing directory", err.Error())
	}
}

func TestRunPagesDeploy_DryRunShortCircuits(t *testing.T) {
	runGlobalsSnapshot(t)
	DryRun = true

	// Network seam that fails the test if hit: DryRun must not probe.
	oldGet := pagesVerifyGet
	pagesVerifyGet = func(string) (int, error) {
		t.Error("DryRun performed an HTTP request")
		return 0, errStubConn
	}
	t.Cleanup(func() { pagesVerifyGet = oldGet })

	// Flags are not parsed when RunE is invoked directly; set the values.
	oldVerify := pagesDeployVerify
	pagesDeployVerify = []string{"/,/about"}
	t.Cleanup(func() { pagesDeployVerify = oldVerify })

	out := capturePrint(t, func() {
		if err := runPagesDeploy(pagesDeployCmd, []string{"./dist"}); err != nil {
			t.Errorf("DryRun run returned error: %v", err)
		}
	})
	for _, want := range []string{"DRY RUN", "./dist", "dist", "/about", "/"} {
		if !strings.Contains(out, want) {
			t.Fatalf("DryRun output missing %q:\n%s", want, out)
		}
	}
}
