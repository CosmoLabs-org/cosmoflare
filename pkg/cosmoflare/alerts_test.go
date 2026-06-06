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

	updated, err := svc.Update("update-me", &AlertRule{
		Service:   "workers",
		Threshold: 10.0,
		Action:    "email",
		Target:    "admin@example.com",
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

func TestAlertService_UpdateNotFound(t *testing.T) {
	svc := newTestAlertService(t)

	_, err := svc.Update("nonexistent", &AlertRule{Service: "r2"})
	if err == nil {
		t.Fatal("expected error updating nonexistent rule")
	}
}

func TestAlertService_UpdateInvalidService(t *testing.T) {
	svc := newTestAlertService(t)
	_, _ = svc.Create(sampleRule("bad-update"))

	_, err := svc.Update("bad-update", &AlertRule{Service: "invalid"})
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
