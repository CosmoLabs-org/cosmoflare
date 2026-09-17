package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// wafRunSnapshot snapshots the globals the WAF handlers read and restores
// them when the test finishes.
func wafRunSnapshot(t *testing.T) {
	t.Helper()
	savedToken, savedDry, savedJSON := APIToken, DryRun, JSONOutput
	savedMode, savedIP := wafRuleMode, wafAccessIP
	savedAccessMode, savedNote, savedForce := wafAccessMode, wafAccessNote, wafForce
	t.Cleanup(func() {
		APIToken, DryRun, JSONOutput = savedToken, savedDry, savedJSON
		wafRuleMode, wafAccessIP = savedMode, savedIP
		wafAccessMode, wafAccessNote, wafForce = savedAccessMode, savedNote, savedForce
	})
}

// TestRunWAFArgValidation verifies each WAF handler rejects missing positional
// arguments before building a service or touching the network.
func TestRunWAFArgValidation(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false

	cases := []struct {
		name    string
		run     func(cmd *cobra.Command, args []string) error
		args    []string
		wantErr string
	}{
		{"packages without zone", runWAFPackages, nil, "zone ID is required"},
		{"rules without package", runWAFRules, []string{"zone"}, "zone ID and package ID are required"},
		{"rule without rule id", runWAFRule, []string{"zone", "pkg"}, "zone ID, package ID, and rule ID are required"},
		{"access list without zone", runWAFAccessList, nil, "zone ID is required"},
		{"access create without zone", runWAFAccessCreate, nil, "zone ID is required"},
		{"access delete without rule", runWAFAccessDelete, []string{"zone"}, "zone ID and rule ID are required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = capturePrint(t, func() {
				// handlers accept (*cobra.Command, []string); a nil command
				// value is acceptable because none of them dereference cmd.
				err := tc.run(nil, tc.args)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})
		})
	}
}

// TestRunWAFServiceError verifies handlers fail with the service-creation
// error when credentials are absent, before any API call.
func TestRunWAFServiceError(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false
	APIToken = ""
	DryRun = false

	cases := []struct {
		name string
		err  error
	}{
		{"packages", runWAFPackages(nil, []string{"zone-1"})},
		{"rules", runWAFRules(nil, []string{"zone-1", "pkg-1"})},
		{"rule", runWAFRule(nil, []string{"zone-1", "pkg-1", "rule-1"})},
		{"access list", runWAFAccessList(nil, []string{"zone-1"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || !strings.Contains(tc.err.Error(), "failed to create WAF service") {
				t.Errorf("expected service-creation error, got %v", tc.err)
			}
		})
	}
}

// TestRunWAFRule_DryRunUpdate verifies a mode update under --dry-run previews
// the change instead of calling the API.
func TestRunWAFRule_DryRunUpdate(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "unit-test-token-0123456789"
	wafRuleMode = "block"

	out := capturePrint(t, func() {
		err := runWAFRule(nil, []string{"zone-1", "pkg-1", "rule-1"})
		if err != nil {
			t.Errorf("dry-run rule update should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would update WAF rule 'rule-1' to mode 'block'") {
		t.Errorf("dry-run preview wrong: %q", out)
	}
}

// TestRunWAFAccessCreate_FlagValidation verifies --ip and --mode are required
// even when the zone argument is present.
func TestRunWAFAccessCreate_FlagValidation(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false

	t.Run("missing ip", func(t *testing.T) {
		wafAccessIP = ""
		wafAccessMode = "block"
		err := runWAFAccessCreate(nil, []string{"zone-1"})
		if err == nil || !strings.Contains(err.Error(), "--ip is required") {
			t.Errorf("expected --ip error, got %v", err)
		}
	})

	t.Run("missing mode", func(t *testing.T) {
		wafAccessIP = "1.2.3.4"
		wafAccessMode = ""
		err := runWAFAccessCreate(nil, []string{"zone-1"})
		if err == nil || !strings.Contains(err.Error(), "--mode is required") {
			t.Errorf("expected --mode error, got %v", err)
		}
	})
}

// TestRunWAFAccessCreate_DryRun verifies the access-rule dry-run preview.
func TestRunWAFAccessCreate_DryRun(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "unit-test-token-0123456789"
	wafAccessIP = "192.168.0.0/24"
	wafAccessMode = "whitelist"
	wafAccessNote = "office"

	out := capturePrint(t, func() {
		err := runWAFAccessCreate(nil, []string{"zone-1"})
		if err != nil {
			t.Errorf("dry-run access create should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would create access rule for 192.168.0.0/24 mode=whitelist") {
		t.Errorf("dry-run preview wrong: %q", out)
	}
}

// TestRunWAFAccessDelete_DryRun verifies deletion under --dry-run skips the
// confirmation prompt and previews the rule ID.
func TestRunWAFAccessDelete_DryRun(t *testing.T) {
	wafRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "unit-test-token-0123456789"
	wafForce = false

	out := capturePrint(t, func() {
		err := runWAFAccessDelete(nil, []string{"zone-1", "rule-9"})
		if err != nil {
			t.Errorf("dry-run access delete should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would delete access rule 'rule-9'") {
		t.Errorf("dry-run preview wrong: %q", out)
	}
	if strings.Contains(out, "Are you sure") {
		t.Errorf("dry-run must skip the confirmation prompt: %q", out)
	}
}
