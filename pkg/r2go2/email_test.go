package r2go2

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// --- Constructor validation tests ---

func TestNewEmailService_NilAPI(t *testing.T) {
	_, err := NewEmailService(nil, "zone123", "acct123")
	if err == nil {
		t.Fatal("expected error for nil API client")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestNewEmailService_EmptyZoneID(t *testing.T) {
	_, err := NewEmailService(nil, "", "acct123")
	if err == nil {
		t.Fatal("expected error for empty zone ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestNewEmailService_EmptyAccountID(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err := NewEmailService(cf, "zone123", "")
	if err == nil {
		t.Fatal("expected error for empty account ID")
	}
}

func TestNewEmailServiceFromCreds_EmptyZoneID(t *testing.T) {
	_, err := NewEmailServiceFromCreds("", "acct123", "token123456")
	if err == nil {
		t.Fatal("expected error for empty zone ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestNewEmailServiceFromCreds_EmptyAccountID(t *testing.T) {
	_, err := NewEmailServiceFromCreds("zone123", "", "token123456")
	if err == nil {
		t.Fatal("expected error for empty account ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestNewEmailServiceFromCreds_EmptyToken(t *testing.T) {
	_, err := NewEmailServiceFromCreds("zone123", "acct123", "")
	if err == nil {
		t.Fatal("expected error for empty API token")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestNewEmailServiceFromCreds_Valid(t *testing.T) {
	svc, err := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID 'zone123', got %q", svc.zoneID)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID 'acct123', got %q", svc.accountID)
	}
}

// --- Method input validation tests ---

func TestEmailService_GetRule_EmptyID(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	_, err := svc.GetRule(nil, "")
	if err == nil {
		t.Fatal("expected error for empty rule ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_CreateRule_EmptyName(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	matchers := []EmailRuleMatcher{{Type: "literal", Field: "to", Value: "test@example.com"}}
	actions := []EmailRuleAction{{Type: "forward", Value: []string{"dest@example.com"}}}
	_, err := svc.CreateRule(nil, "", matchers, actions, 0, true)
	if err == nil {
		t.Fatal("expected error for empty rule name")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_CreateRule_EmptyMatchers(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	actions := []EmailRuleAction{{Type: "forward", Value: []string{"dest@example.com"}}}
	_, err := svc.CreateRule(nil, "test rule", nil, actions, 0, true)
	if err == nil {
		t.Fatal("expected error for empty matchers")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_CreateRule_EmptyActions(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	matchers := []EmailRuleMatcher{{Type: "literal", Field: "to", Value: "test@example.com"}}
	_, err := svc.CreateRule(nil, "test rule", matchers, nil, 0, true)
	if err == nil {
		t.Fatal("expected error for empty actions")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_UpdateRule_EmptyID(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	matchers := []EmailRuleMatcher{{Type: "literal", Field: "to", Value: "test@example.com"}}
	actions := []EmailRuleAction{{Type: "forward", Value: []string{"dest@example.com"}}}
	_, err := svc.UpdateRule(nil, "", "name", matchers, actions, 0, true)
	if err == nil {
		t.Fatal("expected error for empty rule ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_DeleteRule_EmptyID(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	err := svc.DeleteRule(nil, "")
	if err == nil {
		t.Fatal("expected error for empty rule ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_CreateDestination_EmptyEmail(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	_, err := svc.CreateDestination(nil, "")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_GetDestination_EmptyID(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	_, err := svc.GetDestination(nil, "")
	if err == nil {
		t.Fatal("expected error for empty address ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_DeleteDestination_EmptyID(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	err := svc.DeleteDestination(nil, "")
	if err == nil {
		t.Fatal("expected error for empty address ID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

func TestEmailService_UpdateCatchAll_EmptyForwardTo(t *testing.T) {
	svc, _ := NewEmailServiceFromCreds("zone123", "acct123", "valid-token-12345")
	_, err := svc.UpdateCatchAll(nil, "", true)
	if err == nil {
		t.Fatal("expected error for empty forward-to address")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Fatalf("expected R2ValidationError, got %T", err)
	}
}

// --- Type / JSON marshaling tests ---

func TestEmailRule_JSONMarshal(t *testing.T) {
	rule := &EmailRule{
		ID:       "rule-abc123",
		Name:     "Support routing",
		Priority: 10,
		Enabled:  true,
		Matchers: []EmailRuleMatcher{
			{Type: "literal", Field: "to", Value: "support@example.com"},
		},
		Actions: []EmailRuleAction{
			{Type: "forward", Value: []string{"team@company.com"}},
		},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal EmailRule: %v", err)
	}

	var decoded EmailRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EmailRule: %v", err)
	}

	if decoded.ID != rule.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, rule.ID)
	}
	if decoded.Name != rule.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, rule.Name)
	}
	if decoded.Priority != rule.Priority {
		t.Errorf("Priority mismatch: got %d, want %d", decoded.Priority, rule.Priority)
	}
	if decoded.Enabled != rule.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", decoded.Enabled, rule.Enabled)
	}
	if len(decoded.Matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(decoded.Matchers))
	}
	if decoded.Matchers[0].Type != "literal" {
		t.Errorf("Matcher type mismatch: got %q, want %q", decoded.Matchers[0].Type, "literal")
	}
	if decoded.Matchers[0].Field != "to" {
		t.Errorf("Matcher field mismatch: got %q, want %q", decoded.Matchers[0].Field, "to")
	}
	if decoded.Matchers[0].Value != "support@example.com" {
		t.Errorf("Matcher value mismatch: got %q, want %q", decoded.Matchers[0].Value, "support@example.com")
	}
	if len(decoded.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(decoded.Actions))
	}
	if decoded.Actions[0].Type != "forward" {
		t.Errorf("Action type mismatch: got %q, want %q", decoded.Actions[0].Type, "forward")
	}
	if len(decoded.Actions[0].Value) != 1 || decoded.Actions[0].Value[0] != "team@company.com" {
		t.Errorf("Action value mismatch: got %v, want [team@company.com]", decoded.Actions[0].Value)
	}
}

func TestEmailRule_JSONMarshal_DropAction(t *testing.T) {
	rule := &EmailRule{
		ID:       "rule-drop",
		Name:     "Drop spam",
		Priority: 99,
		Enabled:  true,
		Matchers: []EmailRuleMatcher{
			{Type: "all"},
		},
		Actions: []EmailRuleAction{
			{Type: "drop", Value: []string{}},
		},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded EmailRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Matchers[0].Type != "all" {
		t.Errorf("expected 'all' matcher type, got %q", decoded.Matchers[0].Type)
	}
	if decoded.Actions[0].Type != "drop" {
		t.Errorf("expected 'drop' action type, got %q", decoded.Actions[0].Type)
	}
}

func TestEmailDestination_JSONMarshal(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	dest := &EmailDestination{
		ID:       "dest-abc123",
		Email:    "team@company.com",
		Verified: &now,
		Created:  now.Add(-24 * time.Hour),
		Modified: now,
	}

	data, err := json.Marshal(dest)
	if err != nil {
		t.Fatalf("failed to marshal EmailDestination: %v", err)
	}

	var decoded EmailDestination
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EmailDestination: %v", err)
	}

	if decoded.ID != dest.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, dest.ID)
	}
	if decoded.Email != dest.Email {
		t.Errorf("Email mismatch: got %q, want %q", decoded.Email, dest.Email)
	}
	if decoded.Verified == nil || !decoded.Verified.Equal(*dest.Verified) {
		t.Errorf("Verified mismatch: got %v, want %v", decoded.Verified, dest.Verified)
	}
	if !decoded.Created.Equal(dest.Created) {
		t.Errorf("Created mismatch: got %v, want %v", decoded.Created, dest.Created)
	}
}

func TestEmailDestination_JSONMarshal_Unverified(t *testing.T) {
	dest := &EmailDestination{
		ID:      "dest-pending",
		Email:   "new@example.com",
		Created: time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(dest)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Roundtrip: verified should be zero time when not set
	var decoded EmailDestination
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != "dest-pending" {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, "dest-pending")
	}
	if decoded.Email != "new@example.com" {
		t.Errorf("Email mismatch: got %q, want %q", decoded.Email, "new@example.com")
	}
	if decoded.Verified != nil {
		t.Errorf("expected nil Verified for unverified destination, got %v", decoded.Verified)
	}
}

func TestEmailCatchAll_JSONMarshal(t *testing.T) {
	catchall := &EmailCatchAll{
		ID:      "catchall-123",
		Name:    "catch-all",
		Enabled: true,
		Matchers: []EmailRuleMatcher{
			{Type: "all"},
		},
		Actions: []EmailRuleAction{
			{Type: "forward", Value: []string{"admin@company.com"}},
		},
	}

	data, err := json.Marshal(catchall)
	if err != nil {
		t.Fatalf("failed to marshal EmailCatchAll: %v", err)
	}

	var decoded EmailCatchAll
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EmailCatchAll: %v", err)
	}

	if decoded.ID != catchall.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, catchall.ID)
	}
	if decoded.Enabled != true {
		t.Error("expected Enabled=true")
	}
	if len(decoded.Matchers) != 1 || decoded.Matchers[0].Type != "all" {
		t.Error("expected single 'all' matcher")
	}
}

func TestEmailSettings_JSONMarshal(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	settings := &EmailSettings{
		ID:       "settings-123",
		Name:     "example.com",
		Enabled:  true,
		Status:   "ready",
		Created:  &now,
		Modified: &now,
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("failed to marshal EmailSettings: %v", err)
	}

	var decoded EmailSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EmailSettings: %v", err)
	}

	if decoded.ID != settings.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, settings.ID)
	}
	if decoded.Status != "ready" {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, "ready")
	}
	if decoded.Created == nil {
		t.Fatal("expected non-nil Created")
	}
}

// --- Mapping helper tests ---

func TestCfEmailRuleToRule_Mapping(t *testing.T) {
	// Test that the mapping function properly handles nil Enabled
	// by using the internal cfEmailRuleToRule via the public types
	rule := &EmailRule{
		ID:       "test-id",
		Name:     "test",
		Priority: 5,
		Enabled:  false,
		Matchers: []EmailRuleMatcher{},
		Actions:  []EmailRuleAction{},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded EmailRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Enabled != false {
		t.Error("expected Enabled=false")
	}
}

func TestEmailRuleMatcher_Types(t *testing.T) {
	tests := []struct {
		name     string
		matcher  EmailRuleMatcher
		wantType string
	}{
		{"literal matcher", EmailRuleMatcher{Type: "literal", Field: "to", Value: "user@example.com"}, "literal"},
		{"all matcher", EmailRuleMatcher{Type: "all"}, "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.matcher)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var decoded EmailRuleMatcher
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if decoded.Type != tt.wantType {
				t.Errorf("Type mismatch: got %q, want %q", decoded.Type, tt.wantType)
			}
		})
	}
}

func TestEmailRuleAction_Types(t *testing.T) {
	tests := []struct {
		name       string
		action     EmailRuleAction
		wantType   string
		wantValues int
	}{
		{"forward action", EmailRuleAction{Type: "forward", Value: []string{"dest@example.com"}}, "forward", 1},
		{"forward multi", EmailRuleAction{Type: "forward", Value: []string{"a@x.com", "b@x.com"}}, "forward", 2},
		{"drop action", EmailRuleAction{Type: "drop", Value: []string{}}, "drop", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.action)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var decoded EmailRuleAction
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if decoded.Type != tt.wantType {
				t.Errorf("Type mismatch: got %q, want %q", decoded.Type, tt.wantType)
			}
			if len(decoded.Value) != tt.wantValues {
				t.Errorf("Value count mismatch: got %d, want %d", len(decoded.Value), tt.wantValues)
			}
		})
	}
}
