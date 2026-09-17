package cmd

import (
	"strings"
	"testing"
)

// pagerulesRunSnapshot snapshots the page-rule handler globals and restores
// them on cleanup.
func pagerulesRunSnapshot(t *testing.T) {
	t.Helper()
	savedURL, savedAction, savedValue := pageruleURL, pageruleAction, pageruleActionValue
	savedStatus, savedPriority, savedForce := pageruleStatus, pagerulePriority, pageruleForce
	savedDry, savedJSON, savedToken := DryRun, JSONOutput, APIToken
	t.Cleanup(func() {
		pageruleURL, pageruleAction, pageruleActionValue = savedURL, savedAction, savedValue
		pageruleStatus, pagerulePriority, pageruleForce = savedStatus, savedPriority, savedForce
		DryRun, JSONOutput, APIToken = savedDry, savedJSON, savedToken
	})
}

// TestRunPageRulesServiceError verifies list and get fail with the
// service-creation error when the API token is absent.
func TestRunPageRulesServiceError(t *testing.T) {
	pagerulesRunSnapshot(t)
	JSONOutput = false
	DryRun = false
	APIToken = ""

	t.Run("list", func(t *testing.T) {
		err := runPageRulesList(nil, []string{"zone-1"})
		if err == nil || !strings.Contains(err.Error(), "failed to create page rule service") {
			t.Errorf("expected service error, got %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		err := runPageRulesGet(nil, []string{"zone-1", "rule-1"})
		if err == nil || !strings.Contains(err.Error(), "failed to create page rule service") {
			t.Errorf("expected service error, got %v", err)
		}
	})
}

// TestRunPageRulesCreate_DryRun verifies the create preview renders the URL
// pattern, action, and zone without creating anything.
func TestRunPageRulesCreate_DryRun(t *testing.T) {
	pagerulesRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "" // dry-run happens before service creation
	pageruleURL = "*.example.com/old/*"
	pageruleAction = "forwarding_url"
	pageruleActionValue = `"https://example.com/new/$1"`
	pageruleStatus = "active"
	pagerulePriority = 1

	out := capturePrint(t, func() {
		err := runPageRulesCreate(nil, []string{"zone-1"})
		if err != nil {
			t.Errorf("dry-run create should succeed, got %v", err)
		}
	})

	for _, want := range []string{
		"DRY RUN: Would create page rule",
		"*.example.com/old/*",
		"forwarding_url",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("create preview missing %q: %q", want, out)
		}
	}
}

// TestRunPageRulesUpdate_DryRun verifies the update preview names the rule
// and zone being replaced.
func TestRunPageRulesUpdate_DryRun(t *testing.T) {
	pagerulesRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "" // dry-run happens before service creation
	pageruleURL = "example.com/*"
	pageruleAction = "always_https"
	pageruleStatus = "disabled"
	pagerulePriority = 1

	out := capturePrint(t, func() {
		err := runPageRulesUpdate(nil, []string{"zone-1", "rule-7"})
		if err != nil {
			t.Errorf("dry-run update should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would update page rule 'rule-7' in zone 'zone-1'") {
		t.Errorf("update preview wrong: %q", out)
	}
}

// TestRunPageRulesDelete_DryRun verifies deletion under --dry-run skips the
// confirmation prompt and previews the rule ID.
func TestRunPageRulesDelete_DryRun(t *testing.T) {
	pagerulesRunSnapshot(t)
	JSONOutput = false
	DryRun = true
	APIToken = "unit-test-token-0123456789" // service is built before the dry-run branch
	pageruleForce = false

	out := capturePrint(t, func() {
		err := runPageRulesDelete(nil, []string{"zone-1", "rule-9"})
		if err != nil {
			t.Errorf("dry-run delete should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would delete page rule 'rule-9'") {
		t.Errorf("delete preview wrong: %q", out)
	}
	if strings.Contains(out, "Are you sure") {
		t.Errorf("dry-run must skip the confirmation prompt: %q", out)
	}
}
