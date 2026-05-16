package r2go2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// --- Constructor validation ---

func TestNewFirewallServiceValidation(t *testing.T) {
	_, err := NewFirewallService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewFirewallService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewFirewallService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewFirewallServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewFirewallService(cf, "zone123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewFirewallServiceFromCredsValidation(t *testing.T) {
	_, err := NewFirewallServiceFromCreds("", "test-token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewFirewallServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewFirewallServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both zoneID and apiToken are empty")
	}
}

func TestNewFirewallServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewFirewallServiceFromCreds("zone123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

// --- Error type assertions ---

func TestFirewallConstructorValidationErrorType(t *testing.T) {
	_, err := NewFirewallService(nil, "zone123")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewFirewallService(cf, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestFirewallFromCredsValidationErrorType(t *testing.T) {
	_, err := NewFirewallServiceFromCreds("", "test-token")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty zoneID, got %T", err)
	}

	_, err = NewFirewallServiceFromCreds("zone123", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty apiToken, got %T", err)
	}
}

// --- Method input validation ---

func TestFirewallGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewFirewallService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Get(ctx, "")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestFirewallCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewFirewallService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "block", "test rule")
	if err == nil {
		t.Error("expected error when expression is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty expression, got %T", err)
	}

	_, err = svc.Create(ctx, "(ip.src eq 1.2.3.4)", "", "test rule")
	if err == nil {
		t.Error("expected error when action is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty action, got %T", err)
	}
}

func TestFirewallUpdateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewFirewallService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Update(ctx, "", "(ip.src eq 1.2.3.4)", "block", "desc")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty ruleID, got %T", err)
	}

	_, err = svc.Update(ctx, "rule-123", "", "block", "desc")
	if err == nil {
		t.Error("expected error when expression is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty expression, got %T", err)
	}

	_, err = svc.Update(ctx, "rule-123", "(ip.src eq 1.2.3.4)", "", "desc")
	if err == nil {
		t.Error("expected error when action is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty action, got %T", err)
	}
}

func TestFirewallDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewFirewallService(cf, "zone123")
	ctx := context.Background()

	err := svc.Delete(ctx, "")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

// --- Type and JSON tests ---

func TestFirewallFilterRuleType(t *testing.T) {
	now := time.Now()
	rule := &FirewallFilterRule{
		ID:          "rule-001",
		Description: "Block bad IP",
		Action:      "block",
		Priority:    1,
		Paused:      false,
		Filter: FilterExpression{
			ID:         "filter-001",
			Expression: "(ip.src eq 1.2.3.4)",
			Paused:     false,
		},
		CreatedOn:  now,
		ModifiedOn: now,
	}

	if rule.ID != "rule-001" {
		t.Errorf("unexpected ID: %s", rule.ID)
	}
	if rule.Action != "block" {
		t.Errorf("unexpected Action: %s", rule.Action)
	}
	if rule.Filter.Expression != "(ip.src eq 1.2.3.4)" {
		t.Errorf("unexpected Filter.Expression: %s", rule.Filter.Expression)
	}
	if rule.Priority != 1 {
		t.Errorf("unexpected Priority: %d", rule.Priority)
	}
	if rule.Paused {
		t.Error("expected Paused=false")
	}
}

func TestFirewallFilterRuleJSON(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	rule := &FirewallFilterRule{
		ID:          "rule-abc",
		Description: "Challenge suspicious",
		Action:      "challenge",
		Priority:    5,
		Paused:      true,
		Filter: FilterExpression{
			ID:          "filter-xyz",
			Expression:  "(cf.threat_score gt 50)",
			Description: "High threat score",
			Paused:      false,
		},
		CreatedOn:  now,
		ModifiedOn: now,
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded FirewallFilterRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != rule.ID {
		t.Errorf("expected ID=%s, got %s", rule.ID, decoded.ID)
	}
	if decoded.Action != rule.Action {
		t.Errorf("expected Action=%s, got %s", rule.Action, decoded.Action)
	}
	if decoded.Description != rule.Description {
		t.Errorf("expected Description=%s, got %s", rule.Description, decoded.Description)
	}
	if decoded.Priority != rule.Priority {
		t.Errorf("expected Priority=%d, got %d", rule.Priority, decoded.Priority)
	}
	if decoded.Paused != rule.Paused {
		t.Errorf("expected Paused=%v, got %v", rule.Paused, decoded.Paused)
	}
	if decoded.Filter.ID != rule.Filter.ID {
		t.Errorf("expected Filter.ID=%s, got %s", rule.Filter.ID, decoded.Filter.ID)
	}
	if decoded.Filter.Expression != rule.Filter.Expression {
		t.Errorf("expected Filter.Expression=%s, got %s", rule.Filter.Expression, decoded.Filter.Expression)
	}
	if decoded.Filter.Description != rule.Filter.Description {
		t.Errorf("expected Filter.Description=%s, got %s", rule.Filter.Description, decoded.Filter.Description)
	}
}

func TestFirewallFilterRuleJSONOmitEmpty(t *testing.T) {
	rule := &FirewallFilterRule{
		ID:     "rule-001",
		Action: "block",
		Filter: FilterExpression{
			ID:         "filter-001",
			Expression: "(ip.src eq 1.2.3.4)",
		},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Priority with omitempty should be absent when zero
	if _, exists := raw["priority"]; exists {
		t.Error("expected priority to be omitted when zero")
	}

	// Filter.Description with omitempty should be absent when empty
	filterMap := raw["filter"].(map[string]interface{})
	if _, exists := filterMap["description"]; exists {
		t.Error("expected filter.description to be omitted when empty")
	}
}

// --- Conversion helper tests ---

func TestCfFirewallToRule(t *testing.T) {
	now := time.Now()
	priority := float64(3)

	cfRule := cloudflare.FirewallRule{
		ID:          "fw-rule-123",
		Description: "Block country",
		Action:      "block",
		Priority:    priority,
		Paused:      true,
		Filter: cloudflare.Filter{
			ID:          "filter-456",
			Expression:  `(ip.geoip.country eq "CN")`,
			Description: "China traffic",
			Paused:      false,
		},
		CreatedOn:  now,
		ModifiedOn: now,
	}

	rule := cfFirewallToRule(cfRule)

	if rule.ID != "fw-rule-123" {
		t.Errorf("expected ID=fw-rule-123, got %s", rule.ID)
	}
	if rule.Description != "Block country" {
		t.Errorf("expected Description='Block country', got %s", rule.Description)
	}
	if rule.Action != "block" {
		t.Errorf("expected Action=block, got %s", rule.Action)
	}
	if rule.Priority != 3 {
		t.Errorf("expected Priority=3, got %d", rule.Priority)
	}
	if !rule.Paused {
		t.Error("expected Paused=true")
	}
	if rule.Filter.ID != "filter-456" {
		t.Errorf("expected Filter.ID=filter-456, got %s", rule.Filter.ID)
	}
	if rule.Filter.Expression != `(ip.geoip.country eq "CN")` {
		t.Errorf("unexpected Filter.Expression: %s", rule.Filter.Expression)
	}
	if rule.Filter.Description != "China traffic" {
		t.Errorf("expected Filter.Description='China traffic', got %s", rule.Filter.Description)
	}
	if rule.Filter.Paused {
		t.Error("expected Filter.Paused=false")
	}
	if !rule.CreatedOn.Equal(now) {
		t.Errorf("expected CreatedOn=%v, got %v", now, rule.CreatedOn)
	}
	if !rule.ModifiedOn.Equal(now) {
		t.Errorf("expected ModifiedOn=%v, got %v", now, rule.ModifiedOn)
	}
}

func TestCfFirewallToRuleNilPriority(t *testing.T) {
	cfRule := cloudflare.FirewallRule{
		ID:       "fw-rule-nil",
		Action:   "allow",
		Priority: nil,
		Filter: cloudflare.Filter{
			Expression: "(http.host eq \"example.com\")",
		},
	}

	rule := cfFirewallToRule(cfRule)

	if rule.Priority != 0 {
		t.Errorf("expected Priority=0 when nil, got %d", rule.Priority)
	}
}

func TestCfFirewallToRuleIntPriority(t *testing.T) {
	cfRule := cloudflare.FirewallRule{
		ID:       "fw-rule-int",
		Action:   "log",
		Priority: 7,
		Filter: cloudflare.Filter{
			Expression: "(ip.src eq 10.0.0.1)",
		},
	}

	rule := cfFirewallToRule(cfRule)

	if rule.Priority != 7 {
		t.Errorf("expected Priority=7, got %d", rule.Priority)
	}
}
