package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// applyRunGlobals snapshots and restores the package-level variables the
// apply runners read, so tests cannot leak state between each other.
func applyRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldYes, oldDelete := applyYes, applyDeleteUnmanaged
	t.Cleanup(func() { applyYes, applyDeleteUnmanaged = oldYes, oldDelete })
}

// TestGetApplyService_RequiresCredentials verifies the apply service factory
// rejects a missing account ID and a missing API token before any work.
func TestGetApplyService_RequiresCredentials(t *testing.T) {
	applyRunGlobals(t)
	cases := []struct {
		name, account, token, wantErr string
	}{
		{"missing account", "", "tok-123", "account ID is required"},
		{"missing token", "acct-123", "", "API token is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			AccountID, APIToken = tc.account, tc.token
			_, err := getApplyService()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunApplyAll_MissingConfig verifies the apply runner fails with the
// user-facing config-missing error when no .cosmoflare.yaml is present.
func TestRunApplyAll_MissingConfig(t *testing.T) {
	applyRunGlobals(t)
	diffRunChdir(t, "")

	err := runApplyAll(applyCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to load config") {
		t.Fatalf("expected config load error, got %v", err)
	}
}

// TestRunApplyAll_MissingCredentials verifies that with a valid config but
// empty credentials the run aborts at apply-service creation, offline.
func TestRunApplyAll_MissingCredentials(t *testing.T) {
	applyRunGlobals(t)
	diffRunChdir(t, "name: apply-test\n")
	AccountID = ""
	APIToken = ""

	err := runApplyAll(applyCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create apply service") {
		t.Fatalf("expected apply service error, got %v", err)
	}
}

// TestRunApplySubcommands_EmptyConfigShortCircuits verifies the workers, kv
// and r2 runners report an empty config and return nil without any network.
func TestRunApplySubcommands_EmptyConfigShortCircuits(t *testing.T) {
	applyRunGlobals(t)
	diffRunChdir(t, "name: apply-test\n")
	cases := []struct {
		name string
		run  func() error
	}{
		{"workers", func() error { return runApplyWorkers(applyWorkersCmd, nil) }},
		{"kv", func() error { return runApplyKV(applyKVCmd, nil) }},
		{"r2", func() error { return runApplyR2(applyR2Cmd, nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("expected nil error for empty config, got %v", err)
			}
		})
	}
}

// TestRunApplyDNS_RequiresZoneID verifies the DNS apply runner refuses to run
// when the config carries no dns.zone_id.
func TestRunApplyDNS_RequiresZoneID(t *testing.T) {
	applyRunGlobals(t)
	diffRunChdir(t, "name: apply-test\n")

	err := runApplyDNS(applyDNSCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "dns.zone_id is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

// TestRunApplySubcommands_MissingCredentials verifies that populated configs
// with empty credentials abort at apply-service creation, offline.
func TestRunApplySubcommands_MissingCredentials(t *testing.T) {
	applyRunGlobals(t)
	configs := map[string]string{
		"workers": "name: apply-test\nworkers:\n  api:\n    script: dist/api.js\n",
		"dns":     "name: apply-test\ndns:\n  zone_id: zone-123\n",
		"kv":      "name: apply-test\nkv:\n  namespaces:\n    - name: cache\n",
		"r2":      "name: apply-test\nr2:\n  buckets:\n    - name: assets\n",
	}
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			applyRunGlobals(t)
			diffRunChdir(t, cfg)
			AccountID = ""
			APIToken = ""

			runners := map[string]func() error{
				"workers": func() error { return runApplyWorkers(applyWorkersCmd, nil) },
				"dns":     func() error { return runApplyDNS(applyDNSCmd, nil) },
				"kv":      func() error { return runApplyKV(applyKVCmd, nil) },
				"r2":      func() error { return runApplyR2(applyR2Cmd, nil) },
			}
			err := runners[name]()
			if err == nil || !strings.Contains(err.Error(), "failed to create apply service") {
				t.Fatalf("expected apply service error, got %v", err)
			}
		})
	}
}

// TestApplyDisplaySymbols verifies the action and status symbol mappings,
// including the fallback branches for unknown values.
func TestApplyDisplaySymbols(t *testing.T) {
	actions := []struct {
		action cosmoflare.ApplyAction
		want   string
	}{
		{cosmoflare.ApplyCreate, "+"},
		{cosmoflare.ApplyDelete, "-"},
		{cosmoflare.ApplyUpdate, "~"},
		{cosmoflare.ApplyAction("bogus"), " "},
	}
	for _, tc := range actions {
		if got := actionSymbol(tc.action); got != tc.want {
			t.Errorf("actionSymbol(%q) = %q, want %q", tc.action, got, tc.want)
		}
	}
	statuses := []struct {
		status cosmoflare.ApplyStatus
		want   string
	}{
		{cosmoflare.ApplyStatusSuccess, "OK"},
		{cosmoflare.ApplyStatusFailed, "FAIL"},
		{cosmoflare.ApplyStatusSkipped, "SKIP"},
		{cosmoflare.ApplyStatus("bogus"), "?"},
	}
	for _, tc := range statuses {
		if got := statusSymbol(tc.status); got != tc.want {
			t.Errorf("statusSymbol(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// TestPrintApplyResultAndSummary renders a populated single-service result
// (with and without an operation error) and an aggregate summary, checking
// the counts along the way.
func TestPrintApplyResultAndSummary(t *testing.T) {
	result := &cosmoflare.ApplyResult{
		Service: "workers",
		Operations: []cosmoflare.ApplyOperation{
			{Action: cosmoflare.ApplyCreate, Resource: "edge-api", Status: cosmoflare.ApplyStatusSuccess, Detail: "created"},
			{Action: cosmoflare.ApplyDelete, Resource: "old-site", Status: cosmoflare.ApplyStatusFailed, Detail: "delete", Error: "boom"},
			{Action: cosmoflare.ApplyUpdate, Resource: "cdn", Status: cosmoflare.ApplyStatusSkipped, Detail: "skipped"},
		},
	}
	s := result.Summary()
	if s.Succeeded != 1 || s.Failed != 1 || s.Skipped != 1 || s.Total != 3 {
		t.Fatalf("unexpected summary counts: %+v", s)
	}
	printApplyResult(result)
	printApplyResult(&cosmoflare.ApplyResult{Service: "empty"})

	summary := &cosmoflare.ApplySummary{Results: []cosmoflare.ApplyResult{*result}}
	summary.Aggregate()
	if summary.TotalSucceeded != 1 || summary.TotalFailed != 1 || summary.TotalSkipped != 1 {
		t.Fatalf("unexpected aggregate totals: %+v", summary)
	}
	printApplySummary(summary)
	printApplySummary(&cosmoflare.ApplySummary{})
}
