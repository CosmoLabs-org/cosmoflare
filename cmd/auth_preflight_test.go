package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// preflightCaptureStdout swaps os.Stdout for a pipe while fn runs and
// returns everything fn printed. Preflight writes its human/JSON output to
// stdout, so tests assert on this capture.
func preflightCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	out := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	return string(out)
}

// preflightStubScopes points cfAPIBase at an httptest server whose
// /user/tokens/verify and /user/tokens/{id} responses report the given
// permission-group names, mimicking Cloudflare's token-verification API.
func preflightStubScopes(t *testing.T, groups ...string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/user/tokens/verify"):
			fmt.Fprint(w, `{"success":true,"result":{"id":"tok123","status":"active"}}`)
		case strings.Contains(r.URL.Path, "/user/tokens/tok123"):
			var b strings.Builder
			b.WriteString(`{"success":true,"result":{"policies":[{"permission_groups":[`)
			for i, g := range groups {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, `{"name":%q}`, g)
			}
			b.WriteString(`]}]}}`)
			fmt.Fprint(w, b.String())
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	old := cfAPIBase
	cfAPIBase = srv.URL
	t.Cleanup(func() { cfAPIBase = old })

	t.Setenv("CLOUDFLARE_API_TOKEN", "tok")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "acct")
}

// TestAuthPreflight_AllPresentCaptured is the real happy-path assertion
// (kept separate so the capture helper stays out of the error path).
func TestAuthPreflight_AllPresentCaptured(t *testing.T) {
	preflightStubScopes(t, "Zone Read", "DNS Write")

	oldJSON := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = oldJSON })

	var runErr error
	out := preflightCaptureStdout(t, func() {
		runErr = runAuthPreflight(authPreflightCmd, []string{"zones"})
	})
	if runErr != nil {
		t.Fatalf("zones preflight with all scopes present: unexpected error: %v", runErr)
	}
	if !strings.Contains(out, "Zone:Zones:Read") {
		t.Errorf("output must list the required scope, got:\n%s", out)
	}
	if !strings.Contains(out, "Token scopes") {
		t.Errorf("output must show the present token scopes, got:\n%s", out)
	}
	if strings.Contains(out, "missing scope") {
		t.Errorf("happy path must not report missing scopes, got:\n%s", out)
	}
}

// TestAuthPreflight_GapExits1: when the token lacks a required scope the
// command prints a one-line "missing scope: ..." list and returns an error
// (which makes the process exit 1).
func TestAuthPreflight_GapExits1(t *testing.T) {
	preflightStubScopes(t, "Zone Read") // no DNS Write / DNS Read

	oldJSON := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = oldJSON })

	var runErr error
	out := preflightCaptureStdout(t, func() {
		runErr = runAuthPreflight(authPreflightCmd, []string{"dns"})
	})
	if runErr == nil {
		t.Fatal("dns preflight with missing DNS scope must return an error (exit 1)")
	}
	if !strings.Contains(runErr.Error(), "missing scope") {
		t.Errorf("error must carry the missing-scope list, got %v", runErr)
	}
	if !strings.Contains(out, "missing scope: Zone:DNS:Read") {
		t.Errorf("stdout must contain one-line 'missing scope: Zone:DNS:Read', got:\n%s", out)
	}
}

// TestAuthPreflight_GapJSON: --json mode emits a machine-readable envelope
// carrying the missing scopes and still exits non-zero via the error.
func TestAuthPreflight_GapJSON(t *testing.T) {
	preflightStubScopes(t, "Zone Read")

	oldJSON := JSONOutput
	JSONOutput = true
	t.Cleanup(func() { JSONOutput = oldJSON })

	var runErr error
	out := preflightCaptureStdout(t, func() {
		runErr = runAuthPreflight(authPreflightCmd, []string{"dns"})
	})
	if runErr == nil {
		t.Fatal("JSON-mode gap must still return an error (exit 1)")
	}
	if !strings.Contains(out, `"missing"`) || !strings.Contains(out, "Zone:DNS:Read") {
		t.Errorf("JSON envelope must list missing scopes, got:\n%s", out)
	}
}

// TestAuthPreflight_UnknownFamily: an unrecognized family errors with the
// known-families list before any network call.
func TestAuthPreflight_UnknownFamily(t *testing.T) {
	oldJSON := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = oldJSON })

	var runErr error
	out := preflightCaptureStdout(t, func() {
		runErr = runAuthPreflight(authPreflightCmd, []string{"warp-tunnels-bogus"})
	})
	if runErr == nil {
		t.Fatal("unknown family must error")
	}
	if !strings.Contains(runErr.Error(), "unknown command family") {
		t.Errorf("error must name the unknown family, got %v", runErr)
	}
	if !strings.Contains(out+runErr.Error(), "zones") {
		t.Errorf("error must list known families, got %v", runErr)
	}
}

// TestMissingScopes_: unit-test the matcher in internal/cli/scopes.go via
// the re-exported helpers used by the command.
func TestMissingScopes_PreflightUnit(t *testing.T) {
	missing := preflightMissingScopes("dns", []string{"DNS Write"})
	if len(missing) != 0 {
		t.Errorf("DNS Write should satisfy dns, got missing %v", missing)
	}
	missing = preflightMissingScopes("zones", []string{"DNS Write"})
	if len(missing) != 1 || missing[0] != "Zone:Zones:Read" {
		t.Errorf("zones requires Zone:Zones:Read, got %v", missing)
	}
	if _, ok := preflightRequiredScopes("nope"); ok {
		t.Error("unknown family must report not-ok")
	}
}
