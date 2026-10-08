package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// resetExcludeFlags gives the --exclude flag on create/update a fresh pflag
// value. pflag slice values remember that they were already set and APPEND
// on the next Set, and Flag.Changed stays true across Execute calls — so in
// one test process a second `--exclude ""` would append instead of clearing.
// A real CLI run parses once and is unaffected (verified with the built
// binary: update --exclude "" clears the stored list).
func resetExcludeFlags() {
	fresh := pflag.NewFlagSet("reset-exclude", pflag.ContinueOnError)
	fresh.StringSliceVar(&alertExclude, "exclude", nil, "")
	for _, c := range []*cobra.Command{alertsCreateCmd, alertsUpdateCmd} {
		if f := c.Flags().Lookup("exclude"); f != nil {
			f.Value = fresh.Lookup("exclude").Value
			f.Changed = false
		}
	}
}

// setupAlertTestEnv sets up a temporary environment for alert tests.
// It overrides getAlertService to use temp paths and returns a cleanup function.
func setupAlertTestEnv(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	historyPath := filepath.Join(dir, "alert-history.log")

	origFn := getAlertServiceFn
	getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
		return cosmoflare.NewAlertService(rulesPath, historyPath)
	}

	// Reset global flag vars between tests
	alertService = ""
	alertCondition = ""
	alertThreshold = 0
	alertAction = ""
	alertTarget = ""
	alertForce = false
	alertLimit = 0
	alertSince = ""
	alertExclude = nil
	resetExcludeFlags()

	return func() {
		getAlertServiceFn = origFn
	}
}

func executeAlertsCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestAlertsListEmpty(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "list")
	if err != nil {
		t.Fatalf("alerts list: %v", err)
	}
}

func TestAlertsCreateAndList(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "test-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "webhook",
		"--target", "https://hooks.example.com/alert",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	// List should show the rule
	JSONOutput = false
	_, err = executeAlertsCommand("alerts", "list")
	if err != nil {
		t.Fatalf("alerts list: %v", err)
	}
}

func TestAlertsCreateJSON(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	out, err := executeAlertsCommand("alerts", "create", "json-rule",
		"--service", "workers",
		"--condition", "failure-count",
		"--threshold", "10",
		"--action", "email",
		"--target", "admin@example.com",
		"--json",
	)
	if err != nil {
		t.Fatalf("alerts create --json: %v", err)
	}

	// Output should be valid JSON (printed to stdout, not captured by buf)
	_ = out
}

func TestAlertsGet(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "get-rule",
		"--service", "dns",
		"--condition", "latency",
		"--threshold", "500",
		"--action", "log",
		"--target", "/var/log/alerts.log",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	_, err = executeAlertsCommand("alerts", "get", "get-rule")
	if err != nil {
		t.Fatalf("alerts get: %v", err)
	}
}

func TestAlertsGetNotFound(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "get", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rule")
	}
}

func TestAlertsUpdate(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "update-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "webhook",
		"--target", "https://hooks.example.com/alert",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	_, err = executeAlertsCommand("alerts", "update", "update-rule",
		"--threshold", "10",
		"--action", "email",
		"--target", "admin@example.com",
	)
	if err != nil {
		t.Fatalf("alerts update: %v", err)
	}
}

func TestAlertsDeleteRequiresForce(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "del-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "log",
		"--target", "/tmp/log",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	// Without --force should fail
	_, err = executeAlertsCommand("alerts", "delete", "del-rule")
	if err == nil {
		t.Fatal("expected error without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}

	// With --force should succeed
	_, err = executeAlertsCommand("alerts", "delete", "del-rule", "--force")
	if err != nil {
		t.Fatalf("alerts delete --force: %v", err)
	}
}

func TestAlertsTest(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "test-alert-rule",
		"--service", "kv",
		"--condition", "storage-limit",
		"--threshold", "80",
		"--action", "webhook",
		"--target", "https://hooks.example.com/kv",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	_, err = executeAlertsCommand("alerts", "test", "test-alert-rule")
	if err != nil {
		t.Fatalf("alerts test: %v", err)
	}
}

func TestAlertsHistory(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "hist-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "log",
		"--target", "/tmp/log",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	// Generate test history
	_, err = executeAlertsCommand("alerts", "test", "hist-rule")
	if err != nil {
		t.Fatalf("alerts test: %v", err)
	}

	_, err = executeAlertsCommand("alerts", "history")
	if err != nil {
		t.Fatalf("alerts history: %v", err)
	}
}

func TestAlertsHistoryWithLimit(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "create", "limit-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "log",
		"--target", "/tmp/log",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	for i := 0; i < 5; i++ {
		_, err = executeAlertsCommand("alerts", "test", "limit-rule")
		if err != nil {
			t.Fatalf("alerts test %d: %v", i, err)
		}
	}

	_, err = executeAlertsCommand("alerts", "history", "--limit", "2")
	if err != nil {
		t.Fatalf("alerts history --limit: %v", err)
	}
}

func TestAlertsHistoryBadSince(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	_, err := executeAlertsCommand("alerts", "history", "--since", "not-a-date")
	if err == nil {
		t.Fatal("expected error for bad --since format")
	}
}

func TestAlertsDryRun(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	DryRun = true
	defer func() { DryRun = false }()

	// Create in dry run
	_, err := executeAlertsCommand("alerts", "create", "dry-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "log",
		"--target", "/tmp/log",
	)
	if err != nil {
		t.Fatalf("alerts create dry-run: %v", err)
	}

	// Delete in dry run (should not require --force)
	_, err = executeAlertsCommand("alerts", "delete", "dry-rule")
	if err != nil {
		t.Fatalf("alerts delete dry-run: %v", err)
	}
}

func TestAlertsListJSON(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	// Create a rule first
	_, err := executeAlertsCommand("alerts", "create", "json-list-rule",
		"--service", "r2",
		"--condition", "error-rate",
		"--threshold", "5",
		"--action", "log",
		"--target", "/tmp/log",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	// Capture JSON output directly from service
	svc, _ := getAlertServiceFn()
	rules, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	data, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed []cosmoflare.AlertRule
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 rule, got %d", len(parsed))
	}
}

// TestAlertsConfigPersistence verifies rules survive between service instances.
func TestAlertsConfigPersistence(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	historyPath := filepath.Join(dir, "alert-history.log")

	// First instance: create a rule
	svc1, err := cosmoflare.NewAlertService(rulesPath, historyPath)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}

	rule := &cosmoflare.AlertRule{
		Name:      "persist-rule",
		Service:   "r2",
		Condition: "error-rate",
		Threshold: 5.0,
		Action:    "webhook",
		Target:    "https://hooks.example.com",
	}
	if _, err := svc1.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Second instance: verify rule exists
	svc2, err := cosmoflare.NewAlertService(rulesPath, historyPath)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}

	got, err := svc2.Get("persist-rule")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Service != "r2" {
		t.Errorf("expected service 'r2', got %q", got.Service)
	}

	// Verify file exists on disk
	if _, err := os.Stat(rulesPath); err != nil {
		t.Errorf("rules file should exist: %v", err)
	}
}

// TestAlertsExcludeFlag verifies the --exclude flag on alerts create/update
// persists the rule's exclude list, and that --exclude "" clears it (FEAT-049).
func TestAlertsExcludeFlag(t *testing.T) {
	cleanup := setupAlertTestEnv(t)
	defer cleanup()

	JSONOutput = true
	defer func() { JSONOutput = false }()

	_, err := executeAlertsCommand("alerts", "create", "d1-hot",
		"--service", "d1",
		"--condition", "error-rate",
		"--threshold", "1",
		"--action", "log",
		"--target", "x",
		"--exclude", "a.example,b.example",
		"--json",
	)
	if err != nil {
		t.Fatalf("alerts create: %v", err)
	}

	svc, err := getAlertServiceFn()
	if err != nil {
		t.Fatalf("getAlertServiceFn: %v", err)
	}
	rule, err := svc.Get("d1-hot")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(rule.Exclude) != 2 || rule.Exclude[0] != "a.example" || rule.Exclude[1] != "b.example" {
		t.Fatalf("stored Exclude = %v; want [a.example b.example]", rule.Exclude)
	}

	resetExcludeFlags() // each Execute below stands for a separate CLI run
	_, err = executeAlertsCommand("alerts", "update", "d1-hot", "--exclude", "c.example")
	if err != nil {
		t.Fatalf("alerts update: %v", err)
	}
	rule, err = svc.Get("d1-hot")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if len(rule.Exclude) != 1 || rule.Exclude[0] != "c.example" {
		t.Fatalf("stored Exclude after update = %v; want [c.example]", rule.Exclude)
	}

	resetExcludeFlags()
	_, err = executeAlertsCommand("alerts", "update", "d1-hot", "--exclude", "")
	if err != nil {
		t.Fatalf("alerts update --exclude \"\": %v", err)
	}
	rule, err = svc.Get("d1-hot")
	if err != nil {
		t.Fatalf("Get after clear: %v", err)
	}
	if len(rule.Exclude) != 0 {
		t.Fatalf("stored Exclude after --exclude \"\" = %v; want cleared", rule.Exclude)
	}
}
