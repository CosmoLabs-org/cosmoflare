package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// --- Constructor validation ---

func TestNewPageRuleServiceValidation(t *testing.T) {
	_, err := NewPageRuleService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewPageRuleService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewPageRuleService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewPageRuleServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewPageRuleService(cf, "zone123")
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

func TestNewPageRuleServiceFromCredsValidation(t *testing.T) {
	_, err := NewPageRuleServiceFromCreds("", "test-token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewPageRuleServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewPageRuleServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both zoneID and apiToken are empty")
	}
}

func TestNewPageRuleServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewPageRuleServiceFromCreds("zone123", "test-token")
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

func TestPageRuleConstructorValidationErrorType(t *testing.T) {
	_, err := NewPageRuleService(nil, "zone123")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewPageRuleService(cf, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestPageRuleFromCredsValidationErrorType(t *testing.T) {
	_, err := NewPageRuleServiceFromCreds("", "test-token")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty zoneID, got %T", err)
	}

	_, err = NewPageRuleServiceFromCreds("zone123", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty apiToken, got %T", err)
	}
}

// --- Method input validation ---

func TestPageRuleGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewPageRuleService(cf, "zone123")
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

func TestPageRuleCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewPageRuleService(cf, "zone123")
	ctx := context.Background()

	// No targets
	_, err := svc.Create(ctx, nil, []PageRuleAction{{ID: "always_https"}}, "active", 1)
	if err == nil {
		t.Error("expected error when targets are empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty targets, got %T", err)
	}

	// No actions
	targets := []PageRuleTarget{{Target: "url", Constraint: PageRuleConstraint{Operator: "matches", Value: "*.example.com/*"}}}
	_, err = svc.Create(ctx, targets, nil, "active", 1)
	if err == nil {
		t.Error("expected error when actions are empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty actions, got %T", err)
	}
}

func TestPageRuleUpdateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewPageRuleService(cf, "zone123")
	ctx := context.Background()

	targets := []PageRuleTarget{{Target: "url", Constraint: PageRuleConstraint{Operator: "matches", Value: "*.example.com/*"}}}
	actions := []PageRuleAction{{ID: "always_https"}}

	// Empty ruleID
	err := svc.Update(ctx, "", targets, actions, "active", 1)
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// No targets
	err = svc.Update(ctx, "rule123", nil, actions, "active", 1)
	if err == nil {
		t.Error("expected error when targets are empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty targets, got %T", err)
	}

	// No actions
	err = svc.Update(ctx, "rule123", targets, nil, "active", 1)
	if err == nil {
		t.Error("expected error when actions are empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty actions, got %T", err)
	}
}

func TestPageRuleDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewPageRuleService(cf, "zone123")
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

// --- Type construction and JSON marshaling ---

func TestPageRuleTypeConstruction(t *testing.T) {
	now := time.Now()
	rule := &PageRule{
		ID:       "rule-abc123",
		Status:   "active",
		Priority: 1,
		Targets: []PageRuleTarget{
			{
				Target: "url",
				Constraint: PageRuleConstraint{
					Operator: "matches",
					Value:    "*.example.com/old/*",
				},
			},
		},
		Actions: []PageRuleAction{
			{
				ID:    "forwarding_url",
				Value: map[string]interface{}{"status_code": float64(301), "url": "https://example.com/new/$1"},
			},
		},
		CreatedOn:  now,
		ModifiedOn: now,
	}

	if rule.ID != "rule-abc123" {
		t.Errorf("expected ID=rule-abc123, got %s", rule.ID)
	}
	if rule.Status != "active" {
		t.Errorf("expected Status=active, got %s", rule.Status)
	}
	if rule.Priority != 1 {
		t.Errorf("expected Priority=1, got %d", rule.Priority)
	}
	if len(rule.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(rule.Targets))
	}
	if rule.Targets[0].Target != "url" {
		t.Errorf("expected Target=url, got %s", rule.Targets[0].Target)
	}
	if rule.Targets[0].Constraint.Operator != "matches" {
		t.Errorf("expected Operator=matches, got %s", rule.Targets[0].Constraint.Operator)
	}
	if rule.Targets[0].Constraint.Value != "*.example.com/old/*" {
		t.Errorf("expected Value=*.example.com/old/*, got %s", rule.Targets[0].Constraint.Value)
	}
	if len(rule.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(rule.Actions))
	}
	if rule.Actions[0].ID != "forwarding_url" {
		t.Errorf("expected action ID=forwarding_url, got %s", rule.Actions[0].ID)
	}
}

func TestPageRuleJSONMarshal(t *testing.T) {
	rule := &PageRule{
		ID:       "rule-001",
		Status:   "disabled",
		Priority: 5,
		Targets: []PageRuleTarget{
			{
				Target:     "url",
				Constraint: PageRuleConstraint{Operator: "matches", Value: "example.com/path/*"},
			},
		},
		Actions: []PageRuleAction{
			{ID: "always_https", Value: nil},
			{ID: "cache_level", Value: "aggressive"},
		},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded PageRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != "rule-001" {
		t.Errorf("expected ID=rule-001, got %s", decoded.ID)
	}
	if decoded.Status != "disabled" {
		t.Errorf("expected Status=disabled, got %s", decoded.Status)
	}
	if decoded.Priority != 5 {
		t.Errorf("expected Priority=5, got %d", decoded.Priority)
	}
	if len(decoded.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(decoded.Targets))
	}
	if decoded.Targets[0].Constraint.Value != "example.com/path/*" {
		t.Errorf("expected constraint value=example.com/path/*, got %s", decoded.Targets[0].Constraint.Value)
	}
	if len(decoded.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(decoded.Actions))
	}
	if decoded.Actions[0].ID != "always_https" {
		t.Errorf("expected action[0] ID=always_https, got %s", decoded.Actions[0].ID)
	}
	if decoded.Actions[1].ID != "cache_level" {
		t.Errorf("expected action[1] ID=cache_level, got %s", decoded.Actions[1].ID)
	}
	if decoded.Actions[1].Value != "aggressive" {
		t.Errorf("expected action[1] value=aggressive, got %v", decoded.Actions[1].Value)
	}
}

func TestPageRuleTargetConstruction(t *testing.T) {
	target := PageRuleTarget{
		Target: "url",
		Constraint: PageRuleConstraint{
			Operator: "matches",
			Value:    "*.example.com/images/*",
		},
	}

	if target.Target != "url" {
		t.Errorf("expected Target=url, got %s", target.Target)
	}
	if target.Constraint.Operator != "matches" {
		t.Errorf("expected Operator=matches, got %s", target.Constraint.Operator)
	}
	if target.Constraint.Value != "*.example.com/images/*" {
		t.Errorf("expected Value=*.example.com/images/*, got %s", target.Constraint.Value)
	}
}

func TestPageRuleActionConstruction(t *testing.T) {
	// Simple action (no value)
	action := PageRuleAction{ID: "always_https", Value: nil}
	if action.ID != "always_https" {
		t.Errorf("expected ID=always_https, got %s", action.ID)
	}

	// Action with string value
	action2 := PageRuleAction{ID: "cache_level", Value: "aggressive"}
	if action2.Value != "aggressive" {
		t.Errorf("expected Value=aggressive, got %v", action2.Value)
	}

	// Action with complex value (forwarding_url)
	fwdValue := map[string]interface{}{
		"status_code": float64(301),
		"url":         "https://new.example.com/$1",
	}
	action3 := PageRuleAction{ID: "forwarding_url", Value: fwdValue}
	if action3.ID != "forwarding_url" {
		t.Errorf("expected ID=forwarding_url, got %s", action3.ID)
	}
	valMap, ok := action3.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Value to be map[string]interface{}, got %T", action3.Value)
	}
	if valMap["status_code"] != float64(301) {
		t.Errorf("expected status_code=301, got %v", valMap["status_code"])
	}
	if valMap["url"] != "https://new.example.com/$1" {
		t.Errorf("expected url=https://new.example.com/$1, got %v", valMap["url"])
	}
}

// --- Converter test ---

func TestCfPageRuleToPageRule(t *testing.T) {
	now := time.Now()
	cfRule := cloudflare.PageRule{
		ID:       "cf-rule-001",
		Status:   "active",
		Priority: 3,
		Targets: []cloudflare.PageRuleTarget{
			{
				Target: "url",
				Constraint: struct {
					Operator string `json:"operator"`
					Value    string `json:"value"`
				}{
					Operator: "matches",
					Value:    "*.example.com/api/*",
				},
			},
		},
		Actions: []cloudflare.PageRuleAction{
			{ID: "ssl", Value: "full"},
			{ID: "cache_level", Value: "bypass"},
		},
		CreatedOn:  now,
		ModifiedOn: now,
	}

	rule := cfPageRuleToPageRule(cfRule)

	if rule.ID != "cf-rule-001" {
		t.Errorf("expected ID=cf-rule-001, got %s", rule.ID)
	}
	if rule.Status != "active" {
		t.Errorf("expected Status=active, got %s", rule.Status)
	}
	if rule.Priority != 3 {
		t.Errorf("expected Priority=3, got %d", rule.Priority)
	}
	if len(rule.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(rule.Targets))
	}
	if rule.Targets[0].Target != "url" {
		t.Errorf("expected Target=url, got %s", rule.Targets[0].Target)
	}
	if rule.Targets[0].Constraint.Operator != "matches" {
		t.Errorf("expected Operator=matches, got %s", rule.Targets[0].Constraint.Operator)
	}
	if rule.Targets[0].Constraint.Value != "*.example.com/api/*" {
		t.Errorf("expected Value=*.example.com/api/*, got %s", rule.Targets[0].Constraint.Value)
	}
	if len(rule.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(rule.Actions))
	}
	if rule.Actions[0].ID != "ssl" {
		t.Errorf("expected action[0] ID=ssl, got %s", rule.Actions[0].ID)
	}
	if rule.Actions[0].Value != "full" {
		t.Errorf("expected action[0] value=full, got %v", rule.Actions[0].Value)
	}
	if rule.Actions[1].ID != "cache_level" {
		t.Errorf("expected action[1] ID=cache_level, got %s", rule.Actions[1].ID)
	}
	if !rule.CreatedOn.Equal(now) {
		t.Errorf("expected CreatedOn=%v, got %v", now, rule.CreatedOn)
	}
	if !rule.ModifiedOn.Equal(now) {
		t.Errorf("expected ModifiedOn=%v, got %v", now, rule.ModifiedOn)
	}
}

func TestCfPageRuleToPageRuleEmpty(t *testing.T) {
	cfRule := cloudflare.PageRule{
		ID:       "empty-rule",
		Status:   "disabled",
		Priority: 1,
		Targets:  nil,
		Actions:  nil,
	}

	rule := cfPageRuleToPageRule(cfRule)

	if rule.ID != "empty-rule" {
		t.Errorf("expected ID=empty-rule, got %s", rule.ID)
	}
	if len(rule.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(rule.Targets))
	}
	if len(rule.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(rule.Actions))
	}
}
