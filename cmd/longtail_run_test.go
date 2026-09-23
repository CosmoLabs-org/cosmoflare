package cmd

import (
	"strings"
	"testing"
)

// TestLongTail_MissingCreds verifies every wave-3 long-tail runner fails
// fast with its service-construction error when credentials are absent.
func TestLongTail_MissingCreds(t *testing.T) {
	runGlobalsSnapshot(t)

	cases := []struct {
		name string
		call func() error
	}{
		{"page-shield connections", func() error { return runPageShieldConnections(nil, []string{"z1"}) }},
		{"page-shield scripts", func() error { return runPageShieldScripts(nil, []string{"z1"}) }},
		{"page-shield policy list", func() error { return runPageShieldPolicyList(nil, []string{"z1"}) }},
		{"turnstile widget list", func() error { return runTurnstileWidgetList(nil, nil) }},
		{"web-analytics site list", func() error { return runWebAnalyticsSiteList(nil, nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("expected missing-creds error, got nil")
			}
			if !strings.Contains(err.Error(), "failed to create") {
				t.Errorf("expected a service-construction error, got %v", err)
			}
		})
	}
}

// TestLongTail_ArgRequired verifies the direct-call arity guards fire
// before any service work.
func TestLongTail_ArgRequired(t *testing.T) {
	runGlobalsSnapshot(t)

	if err := runPageShieldConnections(nil, nil); err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Errorf("page-shield connections: got %v", err)
	}
	if err := runTurnstileWidgetDelete(nil, nil); err == nil || !strings.Contains(err.Error(), "site key is required") {
		t.Errorf("turnstile delete: got %v", err)
	}
	if err := runWebAnalyticsSiteDelete(nil, nil); err == nil || !strings.Contains(err.Error(), "site tag is required") {
		t.Errorf("web-analytics delete: got %v", err)
	}
}

// TestLongTail_DeleteDryRunPins pins the delete contract shape: with
// DryRun set, the deletes preview and exit cleanly regardless of the
// registry state (the destructiveDryRun call short-circuits first).
func TestLongTail_DeleteDryRunPins(t *testing.T) {
	runGlobalsSnapshot(t)
	DryRun = true

	out := capturePrint(t, func() {
		if err := runTurnstileWidgetDelete(nil, []string{"0x4AAA-sitekey"}); err != nil {
			t.Errorf("turnstile delete dry: %v", err)
		}
	})
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("turnstile delete: expected dry-run preview, got %q", out)
	}

	out = capturePrint(t, func() {
		if err := runWebAnalyticsSiteDelete(nil, []string{"site-tag"}); err != nil {
			t.Errorf("web-analytics delete dry: %v", err)
		}
	})
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("web-analytics delete: expected dry-run preview, got %q", out)
	}
}
