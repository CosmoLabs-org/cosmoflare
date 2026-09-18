package cmd

// Offline unit tests for the cmd run-handlers of part 5 of the cmd test
// sweep: worker (list/get/delete/logs/settings), hyperdrive, cache purge,
// SSL custom hostnames, pages, dns, the ratelimit probe helpers, and the
// apply runners.
//
// Everything here runs without network access. Two strategies are used,
// matching the conventions already established in this package:
//
//   - Cred-validation and dry-run paths are driven through the package-level
//     flag variables (mirroring cache_run_test.go / apply_run_test.go).
//   - Service-dependent paths are exercised through httptest servers wired
//     into a real cosmoflare service via cloudflare.BaseURL, so request
//     mapping and error wrapping are verified without touching the live API.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	cloudflare "github.com/cloudflare/cloudflare-go"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// part5Globals snapshots and restores every package-level variable the
// runners under test read, so subtests cannot leak state into each other.
func part5Globals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldHdForce, oldHdHost, oldHdPort, oldHdScheme := hdForce, hdOriginHost, hdOriginPort, hdOriginScheme
	oldHdDB, oldHdUser, oldHdPass, oldHdName := hdDatabase, hdUser, hdPassword, hdName
	oldPagesForce, oldPagesBranch := pagesForce, pagesBranch
	oldDnsForce := dnsForce
	oldWorkerSince, oldWorkerInterval, oldWorkerLevel := workerLogSince, workerLogInterval, workerLogLevel
	oldWorkerCompat, oldWorkerUsage, oldWorkerBindings := workerCompatDate, workerUsageModel, workerBindings
	oldCacheURLs, oldCacheTags, oldCacheHosts := cachePurgeURLs, cachePurgeTags, cachePurgeHosts
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
		hdForce, hdOriginHost, hdOriginPort, hdOriginScheme = oldHdForce, oldHdHost, oldHdPort, oldHdScheme
		hdDatabase, hdUser, hdPassword, hdName = oldHdDB, oldHdUser, oldHdPass, oldHdName
		pagesForce, pagesBranch = oldPagesForce, oldPagesBranch
		dnsForce = oldDnsForce
		workerLogSince, workerLogInterval, workerLogLevel = oldWorkerSince, oldWorkerInterval, oldWorkerLevel
		workerCompatDate, workerUsageModel, workerBindings = oldWorkerCompat, oldWorkerUsage, oldWorkerBindings
		cachePurgeURLs, cachePurgeTags, cachePurgeHosts = oldCacheURLs, oldCacheTags, oldCacheHosts
	})
}

// part5CaptureStdout redirects os.Stdout into a pipe and returns a reader
// closure that drains it. The human-mode renderers (printInfo/printSuccess)
// write to os.Stdout, so this is how their output is asserted.
func part5CaptureStdout(t *testing.T) func() string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = old
		_ = w.Close()
	})
	return func() string {
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		return buf.String()
	}
}

// part5MarkChanged flips the given flags to "changed" for the duration of
// the test and restores both the value and the mark afterwards.
func part5MarkChanged(t *testing.T, cmd *cobra.Command, names ...string) {
	t.Helper()
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not registered on %q", name, cmd.Name())
		}
		f.Changed = true
		t.Cleanup(func() {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
}

// part5NewCommand returns a bare command whose output writer is buffered, so
// handlers that render to cmd.OutOrStdout() can be asserted without touching
// the real stdout.
func part5NewCommand() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	c := &cobra.Command{Use: "part5"}
	c.SetOut(&buf)
	return c, &buf
}

// --- worker runners ---

// TestPart5RunWorkerList_MissingCredentials verifies runWorkerList refuses
// to run before any network work when account ID or API token is missing.
func TestPart5RunWorkerList_MissingCredentials(t *testing.T) {
	cases := []struct {
		name, account, token string
	}{
		{"missing account", "", "tok-123"},
		{"missing token", "acct-123", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken, DryRun = tc.account, tc.token, false
			err := runWorkerList(workerListCmd, nil)
			if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
				t.Fatalf("expected worker service error, got %v", err)
			}
		})
	}
}

// TestPart5RunWorkerGet_MissingCredentials verifies runWorkerGet aborts at
// service construction with no credentials, even with a valid name argument.
func TestPart5RunWorkerGet_MissingCredentials(t *testing.T) {
	part5Globals(t)
	AccountID, APIToken, DryRun = "", "", false

	err := runWorkerGet(workerGetCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected worker service error, got %v", err)
	}
}

// TestPart5RunWorkerSettings_MissingCredentials verifies the settings runner
// validates arguments and settings first, then aborts offline when the
// worker service cannot be built.
func TestPart5RunWorkerSettings_MissingCredentials(t *testing.T) {
	part5Globals(t)
	AccountID, APIToken, DryRun = "", "", false
	workerCompatDate = "2026-01-01"

	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected worker service error, got %v", err)
	}
}

// TestPart5RunWorkerSettings_InvalidBindingsRejectsDryRun verifies binding
// syntax is validated before the dry-run short-circuit, so a malformed
// --bindings value is never silently accepted.
func TestPart5RunWorkerSettings_InvalidBindingsRejectsDryRun(t *testing.T) {
	part5Globals(t)
	DryRun = true
	AccountID, APIToken = "acct-123", "tok-123"
	workerBindings = []string{"KV"} // missing ":type:id"

	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "invalid binding format") {
		t.Fatalf("expected binding format error, got %v", err)
	}
}

// TestPart5RunWorkerLogs_MissingCredentials verifies the logs runner aborts
// offline when credentials are absent, for both the one-shot and follow
// modes.
func TestPart5RunWorkerLogs_MissingCredentials(t *testing.T) {
	cases := []struct {
		name   string
		follow bool
	}{
		{"one shot", false},
		{"follow", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken, DryRun = "", "", false
			workerLogFollow = tc.follow
			t.Cleanup(func() { workerLogFollow = false })

			err := runWorkerLogs(workerLogsCmd, []string{"my-worker"})
			if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
				t.Fatalf("expected worker service error, got %v", err)
			}
		})
	}
}

// TestPart5RunWorkerLogsFollow_InvalidSince verifies an unparsable --since
// duration is rejected with a descriptive error before any tailing starts.
func TestPart5RunWorkerLogsFollow_InvalidSince(t *testing.T) {
	part5Globals(t)
	workerLogSince = "not-a-duration"

	svc, err := cosmoflare.NewWorkerServiceFromCreds("acct-123", "tok-123")
	if err != nil {
		t.Fatalf("fixture service: %v", err)
	}

	err = runWorkerLogsFollow(workerLogsCmd, svc, "my-worker")
	if err == nil || !strings.Contains(err.Error(), `invalid --since value "not-a-duration"`) {
		t.Fatalf("expected --since validation error, got %v", err)
	}
}

// TestPart5RunWorkerLogsFollow_EmptyNameFails verifies the tailing path maps
// the service-level name validation error onto the CLI error envelope.
func TestPart5RunWorkerLogsFollow_EmptyNameFails(t *testing.T) {
	part5Globals(t)
	workerLogSince = ""

	svc, err := cosmoflare.NewWorkerServiceFromCreds("acct-123", "tok-123")
	if err != nil {
		t.Fatalf("fixture service: %v", err)
	}

	err = runWorkerLogsFollow(workerLogsCmd, svc, "")
	if err == nil || !strings.Contains(err.Error(), "failed to start log tailing") {
		t.Fatalf("expected tailing error, got %v", err)
	}
}

// TestPart5RunWorkerLogsFollow_StreamsEntriesAndStopsOnSignal verifies the
// follow loop renders each polled entry to the command writer and returns
// nil once the process is interrupted. The worker settings endpoint is
// stubbed with an httptest server so no live API is contacted.
func TestPart5RunWorkerLogsFollow_StreamsEntriesAndStopsOnSignal(t *testing.T) {
	part5Globals(t)
	JSONOutput = false
	workerLogSince, workerLogLevel, workerLogInterval = "", "", 1

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/workers/scripts/my-worker/settings") {
			http.Error(w, `{"success":false,"errors":[{"code":9999,"message":"unexpected path"}]}`, http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"w1","modified_on":"2026-01-02T15:04:05Z"}}`)
	}))
	defer ts.Close()

	cf, err := cloudflare.NewWithAPIToken("tok-123", cloudflare.BaseURL(ts.URL))
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	svc, err := cosmoflare.NewWorkerService(cf, "acct-123")
	if err != nil {
		t.Fatalf("worker service: %v", err)
	}

	cmd, buf := part5NewCommand()

	done := make(chan error, 1)
	go func() { done <- runWorkerLogsFollow(cmd, svc, "my-worker") }()

	// Give the first poll a moment to emit its entry, then interrupt.
	time.Sleep(500 * time.Millisecond)
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find process: %v", err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatalf("signal: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("follow returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("follow did not return after interrupt")
	}

	out := buf.String()
	if !strings.Contains(out, "metadata") || !strings.Contains(out, "[info]") {
		t.Fatalf("expected a rendered log entry in output, got %q", out)
	}
}

// --- hyperdrive runners ---

// TestPart5GetHyperdriveService_RequiresCredentials verifies the Hyperdrive
// service factory rejects a missing account ID and a missing API token.
func TestPart5GetHyperdriveService_RequiresCredentials(t *testing.T) {
	cases := []struct {
		name, account, token, wantErr string
	}{
		{"missing account", "", "tok-123", "account ID is required"},
		{"missing token", "acct-123", "", "API token is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken = tc.account, tc.token
			_, err := getHyperdriveService()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestPart5RunHyperdriveListGetUpdate_MissingCredentials verifies the
// read/update runners abort at service construction when credentials are
// absent, without contacting the API.
func TestPart5RunHyperdriveListGetUpdate_MissingCredentials(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"list", func() error { return runHyperdriveList(hyperdriveListCmd, nil) }},
		{"get", func() error { return runHyperdriveGet(hyperdriveGetCmd, []string{"cfg-1"}) }},
		{"update", func() error { return runHyperdriveUpdate(hyperdriveUpdateCmd, []string{"cfg-1"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken, DryRun = "", "", false

			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "failed to create Hyperdrive service") {
				t.Fatalf("expected Hyperdrive service error, got %v", err)
			}
		})
	}
}

// TestPart5RunHyperdriveCreate_DryRunOmitsPassword verifies the dry-run
// payload for create carries the origin parameters but never the password.
func TestPart5RunHyperdriveCreate_DryRunOmitsPassword(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput = true, true
	hdOriginHost, hdOriginPort, hdOriginScheme = "db.example.com", 5432, "postgres"
	hdDatabase, hdUser, hdPassword = "app", "appuser", "hunter2"

	read := part5CaptureStdout(t)
	if err := runHyperdriveCreate(hyperdriveCreateCmd, []string{"my-hd"}); err != nil {
		t.Fatalf("dry-run create returned error: %v", err)
	}
	out := read()

	if !strings.Contains(out, "my-hd") || !strings.Contains(out, "db.example.com") {
		t.Fatalf("dry-run payload missing name/origin host: %q", out)
	}
	if strings.Contains(out, "hunter2") {
		t.Fatalf("dry-run payload must not leak the origin password: %q", out)
	}
}

// TestPart5RunHyperdriveUpdate_DryRunJSON verifies the update dry-run
// payload identifies the config being updated.
func TestPart5RunHyperdriveUpdate_DryRunJSON(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput = true, true
	hdOriginHost, hdDatabase = "db.example.com", "app"

	read := part5CaptureStdout(t)
	if err := runHyperdriveUpdate(hyperdriveUpdateCmd, []string{"cfg-42"}); err != nil {
		t.Fatalf("dry-run update returned error: %v", err)
	}
	if out := read(); !strings.Contains(out, "cfg-42") {
		t.Fatalf("dry-run payload missing config id: %q", out)
	}
}

// TestPart5RunHyperdriveDelete_DryRunSkipsPrompt verifies that in dry-run
// mode the interactive confirmation prompt is skipped entirely.
func TestPart5RunHyperdriveDelete_DryRunSkipsPrompt(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput, hdForce = true, false, false

	read := part5CaptureStdout(t)
	if err := runHyperdriveDelete(hyperdriveDeleteCmd, []string{"cfg-42"}); err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	out := read()
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("expected dry-run notice, got %q", out)
	}
	if strings.Contains(out, "Are you sure") {
		t.Fatalf("dry-run must not prompt for confirmation: %q", out)
	}
}

// TestPart5RunHyperdriveDelete_MissingCredentials verifies that with the
// prompt skipped via --force the runner still aborts offline without
// credentials.
func TestPart5RunHyperdriveDelete_MissingCredentials(t *testing.T) {
	part5Globals(t)
	DryRun, hdForce, AccountID, APIToken = false, true, "", ""

	err := runHyperdriveDelete(hyperdriveDeleteCmd, []string{"cfg-42"})
	if err == nil || !strings.Contains(err.Error(), "failed to create Hyperdrive service") {
		t.Fatalf("expected Hyperdrive service error, got %v", err)
	}
}

// --- cache purge helpers ---

// TestPart5PurgeCacheDryRunZeroService verifies every purge helper
// short-circuits in dry-run mode with a zero-value service and empty
// selector slices: no panic, no network, a dry-run notice instead.
func TestPart5PurgeCacheDryRunZeroService(t *testing.T) {
	cases := []struct {
		name string
		run  func(svc *cosmoflare.CacheService, ctx context.Context, zoneID string) error
	}{
		{"all", purgeCacheAll},
		{"urls", purgeCacheByURLs},
		{"tags", purgeCacheByTags},
		{"hosts", purgeCacheByHosts},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			DryRun, JSONOutput = true, false

			read := part5CaptureStdout(t)
			err := tc.run(&cosmoflare.CacheService{}, context.Background(), "zone123")
			if err != nil {
				t.Fatalf("dry-run purge returned error: %v", err)
			}
			if out := read(); !strings.Contains(out, "DRY RUN") {
				t.Fatalf("expected dry-run notice, got %q", out)
			}
		})
	}
}

// TestPart5PurgeCacheByURLs_JSONDryRunPayload verifies the JSON dry-run
// payload echoes the exact URL list queued for purge.
func TestPart5PurgeCacheByURLs_JSONDryRunPayload(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput = true, true
	cachePurgeURLs = []string{"https://example.com/a.css", "https://example.com/b.js"}

	read := part5CaptureStdout(t)
	if err := purgeCacheByURLs(&cosmoflare.CacheService{}, context.Background(), "zone123"); err != nil {
		t.Fatalf("dry-run purge returned error: %v", err)
	}
	out := read()
	for _, want := range []string{"zone123", "https://example.com/a.css", "https://example.com/b.js"} {
		if !strings.Contains(out, want) {
			t.Fatalf("payload missing %q: %q", want, out)
		}
	}
}

// TestPart5PurgeCacheAll_Success verifies purgeCacheAll maps a successful
// purge-everything response onto the success envelope.
func TestPart5PurgeCacheAll_Success(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput = false, true

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/zones/zone123/purge_cache" {
			http.Error(w, `{"success":false,"errors":[{"code":9999,"message":"unexpected path"}]}`, http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"purge-abc"}}`)
	}))
	defer ts.Close()

	cf, err := cloudflare.NewWithAPIToken("tok-123", cloudflare.BaseURL(ts.URL))
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	svc, err := cosmoflare.NewCacheService(cf, "zone123")
	if err != nil {
		t.Fatalf("cache service: %v", err)
	}

	read := part5CaptureStdout(t)
	if err := purgeCacheAll(svc, context.Background(), "zone123"); err != nil {
		t.Fatalf("purge-all returned error: %v", err)
	}
	if out := read(); !strings.Contains(out, "purge-abc") {
		t.Fatalf("payload missing purge result id: %q", out)
	}
}

// TestPart5PurgeCacheAll_APIErrorWrapped verifies an API failure surfaces as
// the wrapped CLI error rather than a panic or silent success.
func TestPart5PurgeCacheAll_APIErrorWrapped(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput = false, false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":9103,"message":"Unauthorized to access requested resource"}],"messages":[],"result":null}`)
	}))
	defer ts.Close()

	cf, err := cloudflare.NewWithAPIToken("tok-123", cloudflare.BaseURL(ts.URL))
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	svc, err := cosmoflare.NewCacheService(cf, "zone123")
	if err != nil {
		t.Fatalf("cache service: %v", err)
	}

	err = purgeCacheAll(svc, context.Background(), "zone123")
	if err == nil || !strings.Contains(err.Error(), "failed to purge cache") {
		t.Fatalf("expected wrapped purge error, got %v", err)
	}
}

// TestPart5RunCacheSettings_MissingToken verifies the settings runner aborts
// at service construction when no token is configured, whether or not update
// flags were supplied.
func TestPart5RunCacheSettings_MissingToken(t *testing.T) {
	cases := []struct {
		name  string
		flags []string
	}{
		{"view mode", nil},
		{"update mode", []string{"browser-ttl"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			APIToken, DryRun = "", false
			if len(tc.flags) > 0 {
				part5MarkChanged(t, cacheSettingsCmd, tc.flags...)
			}

			err := runCacheSettings(cacheSettingsCmd, []string{"zone123"})
			if err == nil || !strings.Contains(err.Error(), "failed to create cache service") {
				t.Fatalf("expected cache service error, got %v", err)
			}
		})
	}
}

// TestPart5RunCacheSettings_DryRunUpdate verifies an update invocation with
// changed flags short-circuits in dry-run mode without contacting the API.
func TestPart5RunCacheSettings_DryRunUpdate(t *testing.T) {
	part5Globals(t)
	APIToken, DryRun, JSONOutput = "tok-123", true, false
	cacheBrowserTTL, cacheDevMode, cacheCacheLevel = 3600, true, "aggressive"
	part5MarkChanged(t, cacheSettingsCmd, "browser-ttl", "dev-mode", "cache-level")

	read := part5CaptureStdout(t)
	if err := runCacheSettings(cacheSettingsCmd, []string{"zone123"}); err != nil {
		t.Fatalf("dry-run settings update returned error: %v", err)
	}
	if out := read(); !strings.Contains(out, "DRY RUN") {
		t.Fatalf("expected dry-run notice, got %q", out)
	}
}

// --- SSL custom hostname runners ---

// TestPart5RunCHRunners_MissingCredentials verifies the custom-hostname
// runners abort at service construction when no token is configured.
func TestPart5RunCHRunners_MissingCredentials(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"list", func() error { return runCHList(chListCmd, []string{"zone123"}) }},
		{"get", func() error { return runCHGet(chGetCmd, []string{"zone123", "hn-1"}) }},
		{"update", func() error {
			part5MarkChanged(t, chUpdateCmd, "origin")
			return runCHUpdate(chUpdateCmd, []string{"zone123", "hn-1"})
		}},
		{"delete", func() error { return runCHDelete(chDeleteCmd, []string{"zone123", "hn-1"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			APIToken, DryRun = "", false

			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "failed to create SSL custom hostname service") {
				t.Fatalf("expected SSL custom hostname service error, got %v", err)
			}
		})
	}
}

// TestPart5RunCHDelete_DryRunJSON verifies the delete dry-run payload
// identifies both the zone and the hostname being deleted.
func TestPart5RunCHDelete_DryRunJSON(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput, APIToken = true, true, "tok-123"

	read := part5CaptureStdout(t)
	if err := runCHDelete(chDeleteCmd, []string{"zone123", "hn-7"}); err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	out := read()
	if !strings.Contains(out, "zone123") || !strings.Contains(out, "hn-7") {
		t.Fatalf("payload missing zone/hostname id: %q", out)
	}
}

// --- pages runners ---

// TestPart5RunPagesRunners_MissingCredentials verifies the Pages runners
// abort at service construction when no credentials are configured.
func TestPart5RunPagesRunners_MissingCredentials(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"create", func() error { return runPagesCreate(pagesCreateCmd, []string{"site"}) }},
		{"list", func() error { return runPagesList(pagesListCmd, nil) }},
		{"get", func() error { return runPagesGet(pagesGetCmd, []string{"site"}) }},
		{"deployments", func() error { return runPagesDeployments(pagesDeploymentsCmd, []string{"site"}) }},
		{"delete", func() error { return runPagesDelete(pagesDeleteCmd, []string{"site"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken, DryRun = "", "", false
			pagesForce = true // skip the interactive prompt

			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "failed to create Pages service") {
				t.Fatalf("expected Pages service error, got %v", err)
			}
		})
	}
}

// TestPart5RunPagesCreate_DryRunIncludesBranch verifies the create dry-run
// payload carries the configured production branch.
func TestPart5RunPagesCreate_DryRunIncludesBranch(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput, AccountID, APIToken = true, true, "acct-123", "tok-123"
	pagesBranch = "main"

	read := part5CaptureStdout(t)
	if err := runPagesCreate(pagesCreateCmd, []string{"site"}); err != nil {
		t.Fatalf("dry-run create returned error: %v", err)
	}
	out := read()
	if !strings.Contains(out, "site") || !strings.Contains(out, "main") {
		t.Fatalf("payload missing project/branch: %q", out)
	}
}

// --- dns runners ---

// TestPart5RunDNSRunners_MissingToken verifies the DNS runners abort at
// service construction when no token is configured.
func TestPart5RunDNSRunners_MissingToken(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"create", func() error { return runDNSCreate(dnsCreateCmd, []string{"zone123"}) }},
		{"list", func() error { return runDNSList(dnsListCmd, []string{"zone123"}) }},
		{"get", func() error { return runDNSGet(dnsGetCmd, []string{"zone123", "rec-1"}) }},
		{"update", func() error {
			part5MarkChanged(t, dnsUpdateCmd, "ttl")
			return runDNSUpdate(dnsUpdateCmd, []string{"zone123", "rec-1"})
		}},
		{"delete", func() error {
			dnsForce = true
			return runDNSDelete(dnsDeleteCmd, []string{"zone123", "rec-1"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			APIToken, DryRun = "", false

			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "failed to create DNS service") {
				t.Fatalf("expected DNS service error, got %v", err)
			}
		})
	}
}

// TestPart5RunDNSDelete_DryRunJSON verifies the delete dry-run payload
// identifies both the zone and the record, and that the confirmation prompt
// is skipped.
func TestPart5RunDNSDelete_DryRunJSON(t *testing.T) {
	part5Globals(t)
	DryRun, JSONOutput, APIToken, dnsForce = true, true, "tok-123", false

	read := part5CaptureStdout(t)
	if err := runDNSDelete(dnsDeleteCmd, []string{"zone123", "rec-9"}); err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	out := read()
	if !strings.Contains(out, "zone123") || !strings.Contains(out, "rec-9") {
		t.Fatalf("payload missing zone/record id: %q", out)
	}
	if strings.Contains(out, "Are you sure") {
		t.Fatalf("dry-run must not prompt for confirmation: %q", out)
	}
}

// --- ratelimit probe helpers ---

// TestPart5RatelimitServiceAndZone_MissingCredentials verifies the shared
// probe helper refuses to build the rate-limit service without credentials.
func TestPart5RatelimitServiceAndZone_MissingCredentials(t *testing.T) {
	cases := []struct {
		name, account, token string
	}{
		{"missing account", "", "tok-123"},
		{"missing token", "acct-123", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			AccountID, APIToken = tc.account, tc.token

			_, _, err := ratelimitServiceAndZone(context.Background(), "example.com")
			if err == nil || !strings.Contains(err.Error(), "failed to create rate-limit service") {
				t.Fatalf("expected rate-limit service error, got %v", err)
			}
		})
	}
}

// TestPart5ZoneHostname_DottedTargetShortCircuits verifies a target that is
// already a hostname is returned verbatim without a zone lookup.
func TestPart5ZoneHostname_DottedTargetShortCircuits(t *testing.T) {
	part5Globals(t)
	APIToken = ""

	got, err := zoneHostname(context.Background(), "api.example.com", "zone123")
	if err != nil {
		t.Fatalf("dotted target must not hit the zone service: %v", err)
	}
	if got != "api.example.com" {
		t.Fatalf("zoneHostname = %q, want %q", got, "api.example.com")
	}
}

// TestPart5ZoneHostname_ZoneLookupRequiresCredentials verifies a bare zone
// target resolves through the zone service and fails offline without a
// token rather than returning an empty hostname.
func TestPart5ZoneHostname_ZoneLookupRequiresCredentials(t *testing.T) {
	part5Globals(t)
	APIToken = ""

	if _, err := zoneHostname(context.Background(), "myzone", "zone123"); err == nil {
		t.Fatal("expected zone lookup error without credentials, got nil")
	}
}

// TestPart5ReportProbe_JSONOutput verifies reportProbe renders the probe
// result as machine-readable JSON in JSON mode.
func TestPart5ReportProbe_JSONOutput(t *testing.T) {
	part5Globals(t)
	JSONOutput = true

	res := cosmoflare.RateLimitProbeResult{
		URL:      "https://api.example.com/login",
		Requests: 12,
		Statuses: map[int]int{429: 3, 200: 9},
		Verdict:  cosmoflare.VerdictTripped,
	}

	read := part5CaptureStdout(t)
	if err := reportProbe(res); err != nil {
		t.Fatalf("reportProbe returned error: %v", err)
	}
	out := read()
	for _, want := range []string{"https://api.example.com/login", "tripped", `"requests": 12`} {
		if !strings.Contains(out, want) {
			t.Fatalf("JSON probe report missing %q: %q", want, out)
		}
	}
}

// TestPart5ReportProbe_HumanOutput verifies the human renderer prints the
// probe summary fields in plain mode.
func TestPart5ReportProbe_HumanOutput(t *testing.T) {
	part5Globals(t)
	JSONOutput = false

	res := cosmoflare.RateLimitProbeResult{
		URL:      "https://api.example.com/login",
		Requests: 12,
		Verdict:  cosmoflare.VerdictNotCounted,
	}

	read := part5CaptureStdout(t)
	if err := reportProbe(res); err != nil {
		t.Fatalf("reportProbe returned error: %v", err)
	}
	out := read()
	if !strings.Contains(out, "https://api.example.com/login") || !strings.Contains(out, "12 requests") {
		t.Fatalf("human probe report missing url/count: %q", out)
	}
}

// TestPart5ProbeBurstDefault_EmptyZoneID verifies the burst-default lookup
// propagates the service-level zone validation error.
func TestPart5ProbeBurstDefault_EmptyZoneID(t *testing.T) {
	part5Globals(t)

	cf, err := cloudflare.NewWithAPIToken("tok-123")
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	zones, err := cosmoflare.NewZoneService(cf, "acct-123")
	if err != nil {
		t.Fatalf("zone service: %v", err)
	}
	svc, err := cosmoflare.NewRateLimitService(cf, zones)
	if err != nil {
		t.Fatalf("rate-limit service: %v", err)
	}

	if _, err := probeBurstDefault(context.Background(), svc, "", "/login"); err == nil {
		t.Fatal("expected zone ID validation error, got nil")
	}
}

// TestPart5ProbeBurstDefault_NoMatchingRule verifies that a zone whose
// rate-limit entrypoint carries no matching rule reports the actionable
// "create one first" error instead of a bogus default.
func TestPart5ProbeBurstDefault_NoMatchingRule(t *testing.T) {
	part5Globals(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/rulesets/phases/http_ratelimit/entrypoint") {
			http.Error(w, `{"success":false,"errors":[{"code":9999,"message":"unexpected path"}]}`, http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"rs-1","name":"default","kind":"zone","phase":"http_ratelimit","rules":[]}}`)
	}))
	defer ts.Close()

	cf, err := cloudflare.NewWithAPIToken("tok-123", cloudflare.BaseURL(ts.URL))
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	zones, err := cosmoflare.NewZoneService(cf, "acct-123")
	if err != nil {
		t.Fatalf("zone service: %v", err)
	}
	svc, err := cosmoflare.NewRateLimitService(cf, zones)
	if err != nil {
		t.Fatalf("rate-limit service: %v", err)
	}

	_, err = probeBurstDefault(context.Background(), svc, "zone123", "/login")
	if err == nil || !strings.Contains(err.Error(), "no rate-limiting rule matches path") {
		t.Fatalf("expected no-matching-rule error, got %v", err)
	}
}

// --- apply runners ---

// TestPart5RunApplySubcommands_EmptyConfigNotice verifies the workers, kv
// and r2 runners report an empty config section and return nil, offline.
func TestPart5RunApplySubcommands_EmptyConfigNotice(t *testing.T) {
	cases := []struct {
		name, notice string
		run          func() error
	}{
		{"workers", "No workers configured", func() error { return runApplyWorkers(applyWorkersCmd, nil) }},
		{"kv", "No KV namespaces configured", func() error { return runApplyKV(applyKVCmd, nil) }},
		{"r2", "No R2 buckets configured", func() error { return runApplyR2(applyR2Cmd, nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part5Globals(t)
			applyRunGlobals(t)
			diffRunChdir(t, "name: apply-part5\n")

			read := part5CaptureStdout(t)
			if err := tc.run(); err != nil {
				t.Fatalf("expected nil error for empty config, got %v", err)
			}
			if out := read(); !strings.Contains(out, tc.notice) {
				t.Fatalf("expected %q notice in output, got %q", tc.notice, out)
			}
		})
	}
}

// TestPart5GetApplyService_ConstructsWithCredentials verifies the apply
// service factory builds a usable service offline once credentials exist.
func TestPart5GetApplyService_ConstructsWithCredentials(t *testing.T) {
	part5Globals(t)
	AccountID, APIToken = "acct-123", "tok-123"

	svc, err := getApplyService()
	if err != nil {
		t.Fatalf("expected service construction to succeed, got %v", err)
	}
	if svc == nil {
		t.Fatal("apply service is nil")
	}
}
