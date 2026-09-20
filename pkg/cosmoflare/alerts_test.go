package cosmoflare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestAlertService(t *testing.T) *AlertService {
	t.Helper()
	dir := t.TempDir()
	svc, err := NewAlertService(
		filepath.Join(dir, ".cosmoflare-alerts.yaml"),
		filepath.Join(dir, "alert-history.log"),
	)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	return svc
}

func sampleRule(name string) *AlertRule {
	return &AlertRule{
		Name:      name,
		Service:   "r2",
		Condition: "error-rate",
		Threshold: 5.0,
		Action:    "webhook",
		Target:    "https://hooks.example.com/alert",
	}
}

// Pointer helpers for building AlertRuleUpdate values in tests. boolPtr is
// already provided by dns.go in this package.
func alertsTestStrPtr(s string) *string   { return &s }
func alertsTestF64Ptr(f float64) *float64 { return &f }

func TestAlertService_CreateAndGet(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("high-error-rate")
	created, err := svc.Create(rule)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "high-error-rate" {
		t.Errorf("expected name 'high-error-rate', got %q", created.Name)
	}
	if !created.Enabled {
		t.Error("expected rule to be enabled by default")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	got, err := svc.Get("high-error-rate")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Service != "r2" {
		t.Errorf("expected service 'r2', got %q", got.Service)
	}
	if got.Threshold != 5.0 {
		t.Errorf("expected threshold 5.0, got %f", got.Threshold)
	}
}

func TestAlertService_CreateDuplicate(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("dup-rule"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.Create(sampleRule("dup-rule"))
	if err == nil {
		t.Fatal("expected error creating duplicate rule")
	}
}

func TestAlertService_CreateValidation(t *testing.T) {
	svc := newTestAlertService(t)

	tests := []struct {
		name string
		rule *AlertRule
	}{
		{"nil rule", nil},
		{"empty name", &AlertRule{Service: "r2", Condition: "error-rate", Threshold: 1, Action: "log", Target: "/tmp/log"}},
		{"invalid service", &AlertRule{Name: "x", Service: "invalid", Condition: "error-rate", Threshold: 1, Action: "log", Target: "/tmp/log"}},
		{"invalid condition", &AlertRule{Name: "x", Service: "r2", Condition: "invalid", Threshold: 1, Action: "log", Target: "/tmp/log"}},
		{"zero threshold", &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 0, Action: "log", Target: "/tmp/log"}},
		{"negative threshold", &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: -1, Action: "log", Target: "/tmp/log"}},
		{"invalid action", &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 1, Action: "invalid", Target: "/tmp/log"}},
		{"empty target", &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 1, Action: "log", Target: ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(tc.rule)
			if err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestAlertService_List(t *testing.T) {
	svc := newTestAlertService(t)

	// Empty list
	rules, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}

	// Create some rules
	for _, name := range []string{"rule-a", "rule-b", "rule-c"} {
		if _, err := svc.Create(sampleRule(name)); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	rules, err = svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestAlertService_Update(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("update-me"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update("update-me", &AlertRuleUpdate{
		Service:   alertsTestStrPtr("workers"),
		Threshold: alertsTestF64Ptr(10.0),
		Action:    alertsTestStrPtr("email"),
		Target:    alertsTestStrPtr("admin@example.com"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Service != "workers" {
		t.Errorf("expected service 'workers', got %q", updated.Service)
	}
	if updated.Threshold != 10.0 {
		t.Errorf("expected threshold 10.0, got %f", updated.Threshold)
	}
	if updated.Action != "email" {
		t.Errorf("expected action 'email', got %q", updated.Action)
	}

	// Verify persisted
	got, err := svc.Get("update-me")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Service != "workers" {
		t.Errorf("persisted service should be 'workers', got %q", got.Service)
	}
}

// TestAlertService_CreateDisabled verifies that an explicit disabled state
// from the create input is honored instead of force-enabled (TASK-013).
func TestAlertService_CreateDisabled(t *testing.T) {
	svc := newTestAlertService(t)

	created, err := svc.Create(sampleRule("paused-rule"), WithEnabled(false))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Enabled {
		t.Error("expected rule to stay disabled when created with WithEnabled(false)")
	}

	// Verify persisted state survives a reload
	got, err := svc.Get("paused-rule")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Enabled {
		t.Error("persisted rule should remain disabled")
	}
}

// TestAlertService_UpdateEnableDisable verifies Update can flip the enabled
// state in both directions (TASK-013).
func TestAlertService_UpdateEnableDisable(t *testing.T) {
	svc := newTestAlertService(t)

	if _, err := svc.Create(sampleRule("toggle-rule")); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// enable -> disable
	updated, err := svc.Update("toggle-rule", &AlertRuleUpdate{Enabled: boolPtr(false)})
	if err != nil {
		t.Fatalf("Update disable: %v", err)
	}
	if updated.Enabled {
		t.Error("expected rule to be disabled after update")
	}
	got, err := svc.Get("toggle-rule")
	if err != nil {
		t.Fatalf("Get after disable: %v", err)
	}
	if got.Enabled {
		t.Error("persisted rule should be disabled")
	}

	// disable -> enable
	updated, err = svc.Update("toggle-rule", &AlertRuleUpdate{Enabled: boolPtr(true)})
	if err != nil {
		t.Fatalf("Update enable: %v", err)
	}
	if !updated.Enabled {
		t.Error("expected rule to be re-enabled after update")
	}
}

// TestAlertService_UpdateNilEnabledUnchanged verifies a nil Enabled field on
// the update payload leaves the stored enabled state untouched (TASK-013).
func TestAlertService_UpdateNilEnabledUnchanged(t *testing.T) {
	svc := newTestAlertService(t)

	// Disabled rule stays disabled across an unrelated update.
	if _, err := svc.Create(sampleRule("quiet-rule"), WithEnabled(false)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	updated, err := svc.Update("quiet-rule", &AlertRuleUpdate{Threshold: alertsTestF64Ptr(42.0)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Enabled {
		t.Error("nil Enabled must not re-enable a disabled rule")
	}
	if updated.Threshold != 42.0 {
		t.Errorf("expected threshold 42.0, got %f", updated.Threshold)
	}

	// Enabled rule stays enabled across an unrelated update.
	if _, err := svc.Create(sampleRule("loud-rule")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	updated, err = svc.Update("loud-rule", &AlertRuleUpdate{Threshold: alertsTestF64Ptr(7.0)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Enabled {
		t.Error("nil Enabled must not disable an enabled rule")
	}
}

func TestAlertService_UpdateNotFound(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Update("nonexistent", &AlertRuleUpdate{Service: alertsTestStrPtr("r2")})
	if err == nil {
		t.Fatal("expected error updating nonexistent rule")
	}
}

func TestAlertService_UpdateInvalidService(t *testing.T) {
	svc := newTestAlertService(t)
	_, _ = svc.Create(sampleRule("bad-update"))

	_, err := svc.Update("bad-update", &AlertRuleUpdate{Service: alertsTestStrPtr("invalid")})
	if err == nil {
		t.Fatal("expected validation error for invalid service")
	}
}

func TestAlertService_Delete(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("delete-me"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = svc.Delete("delete-me")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.Get("delete-me")
	if err == nil {
		t.Fatal("expected error getting deleted rule")
	}
}

func TestAlertService_DeleteNotFound(t *testing.T) {
	svc := newTestAlertService(t)

	err := svc.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent rule")
	}
}

func TestAlertService_DeleteEmptyName(t *testing.T) {
	svc := newTestAlertService(t)

	err := svc.Delete("")
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestAlertService_Test(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("test-rule"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry, err := svc.Test("test-rule")
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if entry.RuleName != "test-rule" {
		t.Errorf("expected rule name 'test-rule', got %q", entry.RuleName)
	}
	if !entry.IsTest {
		t.Error("expected IsTest to be true")
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}

	// Verify it was recorded in history
	history, err := svc.History(0, time.Time{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if !history[0].IsTest {
		t.Error("history entry should be marked as test")
	}
}

func TestAlertService_TestNotFound(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Test("nonexistent")
	if err == nil {
		t.Fatal("expected error testing nonexistent rule")
	}
}

func TestAlertService_Evaluate_Triggered(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("eval-rule"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Value meets threshold (5.0) -> should trigger
	entry, err := svc.Evaluate("eval-rule", 7.5)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry == nil {
		t.Fatal("expected alert to trigger")
	}
	if entry.Value != 7.5 {
		t.Errorf("expected value 7.5, got %f", entry.Value)
	}
	if entry.IsTest {
		t.Error("evaluate should not be marked as test")
	}
}

func TestAlertService_Evaluate_NotTriggered(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Create(sampleRule("eval-safe"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Value below threshold (5.0) -> should not trigger
	entry, err := svc.Evaluate("eval-safe", 2.0)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry != nil {
		t.Error("expected alert to not trigger when value is below threshold")
	}
}

func TestAlertService_History_LimitAndSince(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("hist-rule")
	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Generate multiple history entries
	for i := 0; i < 5; i++ {
		if _, err := svc.Test("hist-rule"); err != nil {
			t.Fatalf("Test %d: %v", i, err)
		}
	}

	// All entries
	all, err := svc.History(0, time.Time{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 entries, got %d", len(all))
	}

	// Limit to 2
	limited, err := svc.History(2, time.Time{})
	if err != nil {
		t.Fatalf("History with limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(limited))
	}

	// Since future time
	future := time.Now().Add(time.Hour)
	empty, err := svc.History(0, future)
	if err != nil {
		t.Fatalf("History with since: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 entries for future since, got %d", len(empty))
	}
}

func TestAlertService_History_EmptyFile(t *testing.T) {
	svc := newTestAlertService(t)

	history, err := svc.History(0, time.Time{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 entries for empty history, got %d", len(history))
	}
}

func TestAlertService_HistoryNDJSON(t *testing.T) {
	svc := newTestAlertService(t)

	if _, err := svc.Create(sampleRule("ndjson-rule")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Test("ndjson-rule"); err != nil {
		t.Fatalf("Test: %v", err)
	}

	// Read raw file and verify it's valid NDJSON
	data, err := os.ReadFile(svc.historyPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := splitLines(data)
	validLines := 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry AlertHistory
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Errorf("invalid NDJSON line: %v", err)
		}
		validLines++
	}
	if validLines != 1 {
		t.Errorf("expected 1 valid NDJSON line, got %d", validLines)
	}
}

func TestAlertService_GetEmptyName(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Get("")
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestAlertService_AllServices(t *testing.T) {
	svc := newTestAlertService(t)

	services := []string{"r2", "workers", "kv", "dns"}
	for _, s := range services {
		rule := sampleRule("rule-" + s)
		rule.Service = s
		_, err := svc.Create(rule)
		if err != nil {
			t.Errorf("Create with service %q: %v", s, err)
		}
	}

	rules, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 4 {
		t.Errorf("expected 4 rules, got %d", len(rules))
	}
}

func TestAlertService_AllConditions(t *testing.T) {
	svc := newTestAlertService(t)

	conditions := []string{"error-rate", "storage-limit", "latency", "failure-count"}
	for _, c := range conditions {
		rule := sampleRule("rule-" + c)
		rule.Condition = c
		_, err := svc.Create(rule)
		if err != nil {
			t.Errorf("Create with condition %q: %v", c, err)
		}
	}
}

func TestAlertService_AllActions(t *testing.T) {
	svc := newTestAlertService(t)

	actions := []string{"webhook", "email", "log"}
	for _, a := range actions {
		rule := sampleRule("rule-" + a)
		rule.Action = a
		_, err := svc.Create(rule)
		if err != nil {
			t.Errorf("Create with action %q: %v", a, err)
		}
	}
}

func TestNewAlertService_Defaults(t *testing.T) {
	svc, err := NewAlertService("", "")
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	if svc.rulesPath != ".cosmoflare-alerts.yaml" {
		t.Errorf("expected default rules path, got %q", svc.rulesPath)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cosmoflare", "alert-history.log")
	if svc.historyPath != expected {
		t.Errorf("expected history path %q, got %q", expected, svc.historyPath)
	}
}

// TestNewAlertService_ExplicitPaths verifies that explicitly supplied paths are
// stored as-is without any modification.
func TestNewAlertService_ExplicitPaths(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "my-alerts.yaml")
	histPath := filepath.Join(dir, "history.log")

	svc, err := NewAlertService(rulesPath, histPath)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	if svc.rulesPath != rulesPath {
		t.Errorf("expected rulesPath %q, got %q", rulesPath, svc.rulesPath)
	}
	if svc.historyPath != histPath {
		t.Errorf("expected historyPath %q, got %q", histPath, svc.historyPath)
	}
}

// TestValidateAlertRule_AllFields exercises every validation branch directly.
func TestValidateAlertRule_AllFields(t *testing.T) {
	cases := []struct {
		name    string
		rule    *AlertRule
		wantErr bool
	}{
		{
			name:    "nil rule",
			rule:    nil,
			wantErr: true,
		},
		{
			name:    "empty name",
			rule:    &AlertRule{Service: "r2", Condition: "error-rate", Threshold: 1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid service empty",
			rule:    &AlertRule{Name: "x", Service: "", Condition: "error-rate", Threshold: 1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid service unknown",
			rule:    &AlertRule{Name: "x", Service: "s3", Condition: "error-rate", Threshold: 1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid condition empty",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "", Threshold: 1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid condition unknown",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "cpu-usage", Threshold: 1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "zero threshold",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 0, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "negative threshold",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: -0.1, Action: "log", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid action empty",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 1, Action: "", Target: "x"},
			wantErr: true,
		},
		{
			name:    "invalid action unknown",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 1, Action: "sms", Target: "x"},
			wantErr: true,
		},
		{
			name:    "empty target",
			rule:    &AlertRule{Name: "x", Service: "r2", Condition: "error-rate", Threshold: 1, Action: "log", Target: ""},
			wantErr: true,
		},
		{
			name:    "valid minimal rule",
			rule:    &AlertRule{Name: "ok", Service: "r2", Condition: "error-rate", Threshold: 0.001, Action: "log", Target: "/tmp/alert.log"},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAlertRule(tc.rule)
			if tc.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSplitLines covers the splitLines helper with various inputs.
func TestSplitLines(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected int
	}{
		{"empty", []byte{}, 0},
		{"single line no newline", []byte("hello"), 1},
		{"single line with newline", []byte("hello\n"), 1},
		{"two lines", []byte("line1\nline2"), 2},
		{"two lines both newlines", []byte("line1\nline2\n"), 2},
		{"three lines", []byte("a\nb\nc"), 3},
		{"empty lines between", []byte("a\n\nb"), 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitLines(tc.input)
			if len(result) != tc.expected {
				t.Errorf("expected %d lines, got %d", tc.expected, len(result))
			}
		})
	}
}

// TestAlertService_UpdateEmptyName verifies that Update rejects an empty name.
func TestAlertService_UpdateEmptyName(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Update("", &AlertRuleUpdate{Service: alertsTestStrPtr("r2")})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

// TestAlertService_UpdateInvalidCondition verifies that Update rejects unknown conditions.
func TestAlertService_UpdateInvalidCondition(t *testing.T) {
	svc := newTestAlertService(t)
	if _, err := svc.Create(sampleRule("cond-update")); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := svc.Update("cond-update", &AlertRuleUpdate{Condition: alertsTestStrPtr("unknown-condition")})
	if err == nil {
		t.Fatal("expected validation error for invalid condition")
	}
}

// TestAlertService_UpdateInvalidAction verifies that Update rejects unknown actions.
func TestAlertService_UpdateInvalidAction(t *testing.T) {
	svc := newTestAlertService(t)
	if _, err := svc.Create(sampleRule("action-update")); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := svc.Update("action-update", &AlertRuleUpdate{Action: alertsTestStrPtr("pager")})
	if err == nil {
		t.Fatal("expected validation error for invalid action")
	}
}

// TestAlertService_UpdateConditionAndTarget verifies condition and target patching.
func TestAlertService_UpdateConditionAndTarget(t *testing.T) {
	svc := newTestAlertService(t)
	if _, err := svc.Create(sampleRule("patch-rule")); err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update("patch-rule", &AlertRuleUpdate{
		Condition: alertsTestStrPtr("storage-limit"),
		Target:    alertsTestStrPtr("https://new.hook.example.com"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Condition != "storage-limit" {
		t.Errorf("expected condition 'storage-limit', got %q", updated.Condition)
	}
	if updated.Target != "https://new.hook.example.com" {
		t.Errorf("expected new target, got %q", updated.Target)
	}
	// Original fields should be preserved
	if updated.Service != "r2" {
		t.Errorf("service should be unchanged 'r2', got %q", updated.Service)
	}
}

// TestAlertService_UpdateSetsUpdatedAt verifies that UpdatedAt is refreshed on update.
func TestAlertService_UpdateSetsUpdatedAt(t *testing.T) {
	svc := newTestAlertService(t)
	created, err := svc.Create(sampleRule("ts-rule"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	originalUpdatedAt := created.UpdatedAt

	// Small sleep to ensure time advances
	time.Sleep(2 * time.Millisecond)

	updated, err := svc.Update("ts-rule", &AlertRuleUpdate{Threshold: alertsTestF64Ptr(99.0)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be refreshed after update")
	}
}

// TestAlertService_CreateSetsTimestamps verifies both CreatedAt and UpdatedAt are set on create.
func TestAlertService_CreateSetsTimestamps(t *testing.T) {
	svc := newTestAlertService(t)

	before := time.Now()
	created, err := svc.Create(sampleRule("ts-create"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	after := time.Now()

	if created.CreatedAt.Before(before) || created.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", created.CreatedAt, before, after)
	}
	if created.UpdatedAt.Before(before) || created.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in expected range [%v, %v]", created.UpdatedAt, before, after)
	}
}

// TestAlertService_Evaluate_DisabledRule verifies that disabled rules are not triggered.
func TestAlertService_Evaluate_DisabledRule(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("disabled-rule")
	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Disable the rule by updating config directly through the service's Update+Disable
	// We patch the rule file to set enabled=false
	cfg, err := svc.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	for _, r := range cfg.Rules {
		if r.Name == "disabled-rule" {
			r.Enabled = false
		}
	}
	if err := svc.saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	// Value exceeds threshold but rule is disabled
	entry, err := svc.Evaluate("disabled-rule", 100.0)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry != nil {
		t.Error("expected nil return for disabled rule")
	}
}

// TestAlertService_Evaluate_AllConditions verifies all condition types trigger correctly.
func TestAlertService_Evaluate_AllConditions(t *testing.T) {
	conditions := []string{"error-rate", "storage-limit", "latency", "failure-count"}

	for _, cond := range conditions {
		t.Run(cond, func(t *testing.T) {
			svc := newTestAlertService(t)

			rule := sampleRule("rule-" + cond)
			rule.Condition = cond
			rule.Threshold = 10.0
			if _, err := svc.Create(rule); err != nil {
				t.Fatalf("Create: %v", err)
			}

			// Value above threshold — should trigger
			entry, err := svc.Evaluate("rule-"+cond, 15.0)
			if err != nil {
				t.Fatalf("Evaluate above threshold: %v", err)
			}
			if entry == nil {
				t.Errorf("condition %q: expected trigger at value 15.0 with threshold 10.0", cond)
			}
		})
	}
}

// TestAlertService_Evaluate_ExactThreshold verifies boundary: value == threshold triggers.
func TestAlertService_Evaluate_ExactThreshold(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("exact-thresh")
	rule.Threshold = 5.0
	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry, err := svc.Evaluate("exact-thresh", 5.0)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry == nil {
		t.Error("expected alert to trigger when value equals threshold")
	}
}

// TestAlertService_Evaluate_BelowThreshold verifies no trigger when value < threshold.
func TestAlertService_Evaluate_BelowThreshold(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("below-thresh")
	rule.Threshold = 5.0
	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry, err := svc.Evaluate("below-thresh", 4.99)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry != nil {
		t.Error("expected no trigger when value is strictly below threshold")
	}
}

// TestAlertService_Evaluate_NonExistent verifies error on unknown rule name.
func TestAlertService_Evaluate_NonExistent(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Evaluate("ghost-rule", 99.0)
	if err == nil {
		t.Fatal("expected error evaluating nonexistent rule")
	}
}

// TestAlertService_Evaluate_MessageFormat verifies the triggered message contains key fields.
func TestAlertService_Evaluate_MessageFormat(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("msg-rule")
	rule.Service = "workers"
	rule.Condition = "latency"
	rule.Threshold = 200.0
	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry, err := svc.Evaluate("msg-rule", 350.0)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry to be non-nil")
	}
	if entry.Message == "" {
		t.Error("expected non-empty message")
	}
	if entry.Service != "workers" {
		t.Errorf("expected service 'workers', got %q", entry.Service)
	}
	if entry.Condition != "latency" {
		t.Errorf("expected condition 'latency', got %q", entry.Condition)
	}
}

// TestAlertService_History_SinceFilter verifies that entries before "since" are excluded.
func TestAlertService_History_SinceFilter(t *testing.T) {
	svc := newTestAlertService(t)

	if _, err := svc.Create(sampleRule("since-rule")); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Fire a test alert to produce a history entry
	if _, err := svc.Test("since-rule"); err != nil {
		t.Fatalf("Test: %v", err)
	}

	// Filter with a "since" time just before now — should include the entry
	sinceBeforeNow := time.Now().Add(-time.Minute)
	entries, err := svc.History(0, sinceBeforeNow)
	if err != nil {
		t.Fatalf("History since past: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry since past, got %d", len(entries))
	}

	// Filter with a "since" time in the future — should exclude all
	sinceInFuture := time.Now().Add(time.Hour)
	entries, err = svc.History(0, sinceInFuture)
	if err != nil {
		t.Fatalf("History since future: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries since future, got %d", len(entries))
	}
}

// TestAlertService_History_MalformedLines verifies that invalid NDJSON lines are skipped.
func TestAlertService_History_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	histPath := filepath.Join(dir, "history.log")

	// Write a mix of valid and malformed NDJSON
	valid := `{"rule_name":"x","service":"r2","condition":"error-rate","threshold":5,"value":6,"action":"log","target":"/tmp","message":"ok","timestamp":"2026-01-01T00:00:00Z"}`
	content := "not-json\n" + valid + "\n{broken\n"
	if err := os.WriteFile(histPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	svc, err := NewAlertService(filepath.Join(dir, "alerts.yaml"), histPath)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}

	entries, err := svc.History(0, time.Time{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	// Only 1 valid line out of 3
	if len(entries) != 1 {
		t.Errorf("expected 1 valid entry, got %d", len(entries))
	}
	if entries[0].RuleName != "x" {
		t.Errorf("expected rule_name 'x', got %q", entries[0].RuleName)
	}
}

// TestAlertService_LoadConfig_EmptyFile verifies loadConfig returns empty config when file absent.
func TestAlertService_LoadConfig_EmptyFile(t *testing.T) {
	svc := newTestAlertService(t)

	cfg, err := svc.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", cfg.Version)
	}
	if len(cfg.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(cfg.Rules))
	}
}

// TestAlertService_LoadConfig_MissingVersion verifies that a config without version gets "1" set.
func TestAlertService_LoadConfig_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "alerts.yaml")

	// Write a config without a version field
	if err := os.WriteFile(rulesPath, []byte("rules: []\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	svc, err := NewAlertService(rulesPath, filepath.Join(dir, "hist.log"))
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}

	cfg, err := svc.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("expected version '1' defaulted, got %q", cfg.Version)
	}
}

// TestAlertService_Test_PopulatesAllFields checks that Test sets every AlertHistory field.
func TestAlertService_Test_PopulatesAllFields(t *testing.T) {
	svc := newTestAlertService(t)

	rule := sampleRule("full-fields")
	rule.Service = "kv"
	rule.Condition = "failure-count"
	rule.Threshold = 42.0
	rule.Action = "email"
	rule.Target = "ops@example.com"

	if _, err := svc.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry, err := svc.Test("full-fields")
	if err != nil {
		t.Fatalf("Test: %v", err)
	}

	if entry.RuleName != "full-fields" {
		t.Errorf("RuleName: expected 'full-fields', got %q", entry.RuleName)
	}
	if entry.Service != "kv" {
		t.Errorf("Service: expected 'kv', got %q", entry.Service)
	}
	if entry.Condition != "failure-count" {
		t.Errorf("Condition: expected 'failure-count', got %q", entry.Condition)
	}
	if entry.Threshold != 42.0 {
		t.Errorf("Threshold: expected 42.0, got %f", entry.Threshold)
	}
	if entry.Value != 42.0 {
		t.Errorf("Value: expected 42.0 (simulated threshold), got %f", entry.Value)
	}
	if entry.Action != "email" {
		t.Errorf("Action: expected 'email', got %q", entry.Action)
	}
	if entry.Target != "ops@example.com" {
		t.Errorf("Target: expected 'ops@example.com', got %q", entry.Target)
	}
	if entry.Message == "" {
		t.Error("expected non-empty Message")
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
	if !entry.IsTest {
		t.Error("expected IsTest=true")
	}
}

// TestAlertService_DeleteMiddle verifies deletion preserves surrounding rules.
func TestAlertService_DeleteMiddle(t *testing.T) {
	svc := newTestAlertService(t)

	for _, name := range []string{"first", "middle", "last"} {
		if _, err := svc.Create(sampleRule(name)); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	if err := svc.Delete("middle"); err != nil {
		t.Fatalf("Delete middle: %v", err)
	}

	rules, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules after deletion, got %d", len(rules))
	}
	for _, r := range rules {
		if r.Name == "middle" {
			t.Error("deleted rule 'middle' still present")
		}
	}
}

// TestAlertService_GetNotFound verifies Get returns an error for unknown names.
func TestAlertService_GetNotFound(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Get("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent rule")
	}
}

// TestAlertsConfigStructure verifies AlertsConfig initialises correctly.
func TestAlertsConfigStructure(t *testing.T) {
	cfg := &AlertsConfig{
		Version: "1",
		Rules:   []*AlertRule{},
	}
	if cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", cfg.Version)
	}
	if cfg.Rules == nil {
		t.Error("expected non-nil Rules slice")
	}
}

// TestAlertHistoryIsTestOmitEmpty verifies IsTest is omitted in JSON when false.
func TestAlertHistoryIsTestOmitEmpty(t *testing.T) {
	entry := &AlertHistory{
		RuleName:  "x",
		Timestamp: time.Now(),
		IsTest:    false,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// is_test should be omitted when false (omitempty tag)
	// We verify no "is_test" key at all
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := m["is_test"]; ok {
		t.Error("expected is_test to be omitted when false (omitempty)")
	}
}

// TestAlertHistoryIsTestPresent verifies IsTest appears in JSON when true.
func TestAlertHistoryIsTestPresent(t *testing.T) {
	entry := &AlertHistory{
		RuleName:  "x",
		Timestamp: time.Now(),
		IsTest:    true,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if v, ok := m["is_test"]; !ok || v != true {
		t.Error("expected is_test=true to be present in JSON")
	}
}
