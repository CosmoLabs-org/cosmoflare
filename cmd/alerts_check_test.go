package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// buckAlertsCheckEnv wires getAlertServiceFn at temp paths (same seam the
// other alert tests use) and snapshots the credential globals.
func buckAlertsCheckEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	historyPath := filepath.Join(dir, "alert-history.log")

	origFn := getAlertServiceFn
	oldAcct, oldTok := AccountID, APIToken
	oldJSON := JSONOutput
	getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
		return cosmoflare.NewAlertService(rulesPath, historyPath)
	}
	t.Cleanup(func() {
		getAlertServiceFn = origFn
		AccountID, APIToken = oldAcct, oldTok
		JSONOutput = oldJSON
	})
	AccountID, APIToken = "", ""
	JSONOutput = false
}

// TestRunAlertsCheck_NoRules verifies a check with no configured rules
// succeeds offline: nothing is evaluated so no metrics collection runs.
func TestRunAlertsCheck_NoRules(t *testing.T) {
	buckAlertsCheckEnv(t)

	if err := runAlertsCheck(alertsCheckCmd, nil); err != nil {
		t.Fatalf("runAlertsCheck with no rules: %v", err)
	}
}

// TestRunAlertsCheck_OnlyDisabledRules verifies disabled rules are not
// counted as evaluated, so no metrics collection (and no credentials) is
// required.
func TestRunAlertsCheck_OnlyDisabledRules(t *testing.T) {
	buckAlertsCheckEnv(t)

	svc, err := getAlertService()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(&cosmoflare.AlertRule{
		Name:      "disabled-rule",
		Service:   "r2",
		Condition: "storage-limit",
		Threshold: 80,
		Action:    "log",
		Target:    "/dev/null",
		Enabled:   false,
	}); err != nil {
		t.Fatalf("create disabled rule: %v", err)
	}

	if err := runAlertsCheck(alertsCheckCmd, nil); err != nil {
		t.Fatalf("runAlertsCheck with only disabled rules: %v", err)
	}
}

// TestRunAlertsCheck_MissingCreds verifies that with at least one enabled
// rule the check requires account credentials before touching the
// analytics API.
func TestRunAlertsCheck_MissingCreds(t *testing.T) {
	buckAlertsCheckEnv(t)

	svc, err := getAlertService()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(&cosmoflare.AlertRule{
		Name:      "enabled-rule",
		Service:   "r2",
		Condition: "storage-limit",
		Threshold: 80,
		Action:    "log",
		Target:    "/dev/null",
		Enabled:   true,
	}); err != nil {
		t.Fatalf("create enabled rule: %v", err)
	}

	err = runAlertsCheck(alertsCheckCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "account ID and API token are required") {
		t.Fatalf("expected credentials-required error, got %v", err)
	}
}

// TestRunAlertsUpdate_NotFound verifies updating an unknown rule surfaces
// the wrapped not-found error.
func TestRunAlertsUpdate_NotFound(t *testing.T) {
	buckAlertsCheckEnv(t)

	err := runAlertsUpdate(alertsUpdateCmd, []string{"ghost-rule"})
	if err == nil || !strings.Contains(err.Error(), "failed to update alert rule") {
		t.Fatalf("expected update failure for missing rule, got %v", err)
	}
}

// TestRunAlertsHistory_EmptyOffline verifies the history reader succeeds
// with an empty (nonexistent) history file.
func TestRunAlertsHistory_EmptyOffline(t *testing.T) {
	buckAlertsCheckEnv(t)
	alertLimit = 0
	alertSince = ""

	if err := runAlertsHistory(alertsHistoryCmd, nil); err != nil {
		t.Fatalf("runAlertsHistory empty: %v", err)
	}
}

// TestRunAlertsHistory_InvalidSince verifies a malformed --since value is
// rejected with the RFC3339 hint.
func TestRunAlertsHistory_InvalidSince(t *testing.T) {
	buckAlertsCheckEnv(t)
	alertSince = "not-a-timestamp"

	err := runAlertsHistory(alertsHistoryCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid --since format") {
		t.Fatalf("expected invalid --since error, got %v", err)
	}
}
