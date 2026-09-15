package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// diffRunGlobals snapshots and restores the package globals the diff runners
// read, so tests cannot leak state between each other.
func diffRunGlobals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldOutput := diffOutput
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
		diffOutput = oldOutput
		if f := diffCmd.PersistentFlags().Lookup("output"); f != nil {
			f.Changed = false
		}
	})
}

// diffRunChdir moves the test into a temp directory containing a
// .cosmoflare.yaml with the given content (empty string writes no file), and
// restores the original working directory on cleanup.
func diffRunChdir(t *testing.T, configContent string) string {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if configContent != "" {
		path := filepath.Join(dir, ".cosmoflare.yaml")
		if err := os.WriteFile(path, []byte(configContent), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldWD) })
	return dir
}

// TestRunDiffAll_MissingConfig verifies the diff runner fails with the
// user-facing config-missing error when no .cosmoflare.yaml is present.
func TestRunDiffAll_MissingConfig(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, "")

	err := runDiffAll(diffCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to load config") {
		t.Fatalf("expected config load error, got %v", err)
	}
}

// TestGetDiffService_RequiresCredentials verifies the diff service factory
// rejects missing account ID and API token before any network work.
func TestGetDiffService_RequiresCredentials(t *testing.T) {
	diffRunGlobals(t)
	cases := []struct {
		name, account, token, wantErr string
	}{
		{"missing account", "", "tok-123", "account ID is required"},
		{"missing token", "acct-123", "", "API token is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			AccountID, APIToken = tc.account, tc.token
			_, err := getDiffService()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunDiffAll_MissingCredentials verifies that with a valid config but no
// credentials the run aborts at diff-service creation, offline.
func TestRunDiffAll_MissingCredentials(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, "name: diff-test\n")
	AccountID = ""
	APIToken = ""

	err := runDiffAll(diffCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}

// TestRunDiffWorkers_NoWorkersConfigured verifies the workers diff reports an
// empty config and exits successfully without contacting the API.
func TestRunDiffWorkers_NoWorkersConfigured(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, "name: diff-test\n")

	if err := runDiffWorkers(diffWorkersCmd, nil); err != nil {
		t.Fatalf("expected nil error for empty workers config, got %v", err)
	}
}

// TestRunDiffDNS_RequiresZoneID verifies the DNS diff refuses to run when the
// config carries no dns.zone_id.
func TestRunDiffDNS_RequiresZoneID(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, "name: diff-test\n")

	err := runDiffDNS(diffDNSCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "dns.zone_id is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

// TestRunDiffKVAndR2_EmptyConfigSucceeds verifies the KV and R2 diffs short-
// circuit successfully when nothing is configured, staying offline.
func TestRunDiffKVAndR2_EmptyConfigSucceeds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, "name: diff-test\n")
	cases := []struct {
		name string
		run  func() error
	}{
		{"kv", func() error { return runDiffKV(diffKVCmd, nil) }},
		{"r2", func() error { return runDiffR2(diffR2Cmd, nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("expected nil error for empty config, got %v", err)
			}
		})
	}
}

// TestPrintDiffResult_SummaryAndFull verifies both output-detail modes render
// a populated result without panicking, and the summary line aggregates.
func TestPrintDiffResult_SummaryAndFull(t *testing.T) {
	diffRunGlobals(t)
	result := &cosmoflare.DiffResult{
		Service:   "workers",
		Additions: []cosmoflare.DiffEntry{{Resource: "edge-api", Detail: "not deployed"}},
		Deletions: []cosmoflare.DiffEntry{{Resource: "old-site", Detail: "not in config"}},
		Changes:   []cosmoflare.DiffEntry{{Resource: "cdn-worker", Detail: "pattern differs"}},
	}
	summary := &cosmoflare.DiffSummary{
		Results: []cosmoflare.DiffResult{*result}, TotalAdd: 1, TotalDel: 1, TotalMod: 1,
	}

	diffOutput = "full"
	printDiffResult(result)
	diffOutput = "summary"
	printDiffResult(result)
	printDiffSummaryLine(summary)
}
