package r2go2

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// EmailRule represents a Cloudflare Email Routing rule.
type EmailRule struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Priority int                 `json:"priority"`
	Enabled  bool                `json:"enabled"`
	Matchers []EmailRuleMatcher  `json:"matchers"`
	Actions  []EmailRuleAction   `json:"actions"`
}

// EmailRuleMatcher defines matching criteria for an email routing rule.
type EmailRuleMatcher struct {
	Type  string `json:"type"`
	Field string `json:"field"`
	Value string `json:"value"`
}

// EmailRuleAction defines the action taken when a rule matches.
type EmailRuleAction struct {
	Type  string   `json:"type"`
	Value []string `json:"value"`
}

// EmailCatchAll represents the catch-all email routing rule.
type EmailCatchAll struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Enabled  bool                `json:"enabled"`
	Matchers []EmailRuleMatcher  `json:"matchers"`
	Actions  []EmailRuleAction   `json:"actions"`
}

// EmailSettings represents email routing settings for a zone.
type EmailSettings struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"`
	Created    *time.Time `json:"created,omitempty"`
	Modified   *time.Time `json:"modified,omitempty"`
}

// EmailService implements Email Routing operations.
type EmailService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewEmailService creates a new Email Routing service client.
func NewEmailService(api *cloudflare.API, zoneID string) (*EmailService, error) {
	if api == nil {
		return nil, validationError("NewEmailService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewEmailService", "zone ID is required")
	}
	return &EmailService{cf: api, zoneID: zoneID}, nil
}

// NewEmailServiceFromCreds creates an EmailService from zone ID and API token.
// Convenience helper for CLI usage.
func NewEmailServiceFromCreds(zoneID, apiToken string) (*EmailService, error) {
	if zoneID == "" {
		return nil, validationError("NewEmailService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewEmailService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewEmailService", "failed to create Cloudflare API client", err)
	}
	return &EmailService{cf: cf, zoneID: zoneID}, nil
}

// ListRules returns all email routing rules for the zone.
func (s *EmailService) ListRules(ctx context.Context) ([]*EmailRule, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	results, _, err := s.cf.ListEmailRoutingRules(ctx, rc, cloudflare.ListEmailRoutingRulesParameters{})
	if err != nil {
		return nil, newError("EmailService.ListRules", "failed to list email routing rules", err)
	}

	rules := make([]*EmailRule, 0, len(results))
	for _, r := range results {
		rules = append(rules, cfEmailRuleToRule(r))
	}
	return rules, nil
}

// CreateRule creates a new email routing rule.
func (s *EmailService) CreateRule(ctx context.Context, matchAddress, forwardTo, name string, enabled bool) (*EmailRule, error) {
	if matchAddress == "" {
		return nil, validationError("EmailService.CreateRule", "match address is required")
	}
	if forwardTo == "" {
		return nil, validationError("EmailService.CreateRule", "forward-to address is required")
	}

	enabledPtr := &enabled
	params := cloudflare.CreateEmailRoutingRuleParameters{
		Matchers: []cloudflare.EmailRoutingRuleMatcher{
			{
				Type:  "literal",
				Field: "to",
				Value: matchAddress,
			},
		},
		Actions: []cloudflare.EmailRoutingRuleAction{
			{
				Type:  "forward",
				Value: []string{forwardTo},
			},
		},
		Name:    name,
		Enabled: enabledPtr,
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.CreateEmailRoutingRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.CreateRule", fmt.Sprintf("failed to create email routing rule for %s", matchAddress), err)
	}

	return cfEmailRuleToRule(result), nil
}

// GetRule retrieves a single email routing rule by ID.
func (s *EmailService) GetRule(ctx context.Context, ruleID string) (*EmailRule, error) {
	if ruleID == "" {
		return nil, validationError("EmailService.GetRule", "rule ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.GetEmailRoutingRule(ctx, rc, ruleID)
	if err != nil {
		return nil, notFound("EmailService.GetRule", "", ruleID, err)
	}

	return cfEmailRuleToRule(result), nil
}

// UpdateRule updates an existing email routing rule.
func (s *EmailService) UpdateRule(ctx context.Context, ruleID string, matchers []EmailRuleMatcher, actions []EmailRuleAction, name string, enabled *bool) (*EmailRule, error) {
	if ruleID == "" {
		return nil, validationError("EmailService.UpdateRule", "rule ID is required")
	}

	params := cloudflare.UpdateEmailRoutingRuleParameters{
		RuleID: ruleID,
		Name:   name,
	}

	if enabled != nil {
		params.Enabled = enabled
	}

	if len(matchers) > 0 {
		cfMatchers := make([]cloudflare.EmailRoutingRuleMatcher, len(matchers))
		for i, m := range matchers {
			cfMatchers[i] = cloudflare.EmailRoutingRuleMatcher{
				Type:  m.Type,
				Field: m.Field,
				Value: m.Value,
			}
		}
		params.Matchers = cfMatchers
	}

	if len(actions) > 0 {
		cfActions := make([]cloudflare.EmailRoutingRuleAction, len(actions))
		for i, a := range actions {
			cfActions[i] = cloudflare.EmailRoutingRuleAction{
				Type:  a.Type,
				Value: a.Value,
			}
		}
		params.Actions = cfActions
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.UpdateEmailRoutingRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.UpdateRule", fmt.Sprintf("failed to update email routing rule %q", ruleID), err)
	}

	return cfEmailRuleToRule(result), nil
}

// DeleteRule deletes an email routing rule.
func (s *EmailService) DeleteRule(ctx context.Context, ruleID string) (*EmailRule, error) {
	if ruleID == "" {
		return nil, validationError("EmailService.DeleteRule", "rule ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.DeleteEmailRoutingRule(ctx, rc, ruleID)
	if err != nil {
		return nil, newError("EmailService.DeleteRule", fmt.Sprintf("failed to delete email routing rule %q", ruleID), err)
	}

	return cfEmailRuleToRule(result), nil
}

// GetCatchAll retrieves the catch-all email routing rule.
func (s *EmailService) GetCatchAll(ctx context.Context) (*EmailCatchAll, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.GetEmailRoutingCatchAllRule(ctx, rc)
	if err != nil {
		return nil, newError("EmailService.GetCatchAll", "failed to get catch-all rule", err)
	}

	return cfCatchAllToRule(result), nil
}

// UpdateCatchAll updates the catch-all email routing rule.
func (s *EmailService) UpdateCatchAll(ctx context.Context, forwardTo string, enabled bool) (*EmailCatchAll, error) {
	if forwardTo == "" {
		return nil, validationError("EmailService.UpdateCatchAll", "forward-to address is required")
	}

	enabledPtr := &enabled
	params := cloudflare.EmailRoutingCatchAllRule{
		Name:    "catch-all",
		Enabled: enabledPtr,
		Matchers: []cloudflare.EmailRoutingRuleMatcher{
			{
				Type: "all",
			},
		},
		Actions: []cloudflare.EmailRoutingRuleAction{
			{
				Type:  "forward",
				Value: []string{forwardTo},
			},
		},
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.UpdateEmailRoutingCatchAllRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.UpdateCatchAll", "failed to update catch-all rule", err)
	}

	return cfCatchAllToRule(result), nil
}

// GetSettings retrieves email routing settings for the zone.
func (s *EmailService) GetSettings(ctx context.Context) (*EmailSettings, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.GetEmailRoutingSettings(ctx, rc)
	if err != nil {
		return nil, newError("EmailService.GetSettings", "failed to get email routing settings", err)
	}

	return &EmailSettings{
		ID:       result.Tag,
		Name:     result.Name,
		Enabled:  result.Enabled,
		Status:   result.Status,
		Created:  result.Created,
		Modified: result.Modified,
	}, nil
}

// Enable enables email routing for the zone.
func (s *EmailService) Enable(ctx context.Context) (*EmailSettings, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.EnableEmailRouting(ctx, rc)
	if err != nil {
		return nil, newError("EmailService.Enable", "failed to enable email routing", err)
	}

	return &EmailSettings{
		ID:       result.Tag,
		Name:     result.Name,
		Enabled:  result.Enabled,
		Status:   result.Status,
		Created:  result.Created,
		Modified: result.Modified,
	}, nil
}

// Disable disables email routing for the zone.
func (s *EmailService) Disable(ctx context.Context) (*EmailSettings, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.DisableEmailRouting(ctx, rc)
	if err != nil {
		return nil, newError("EmailService.Disable", "failed to disable email routing", err)
	}

	return &EmailSettings{
		ID:       result.Tag,
		Name:     result.Name,
		Enabled:  result.Enabled,
		Status:   result.Status,
		Created:  result.Created,
		Modified: result.Modified,
	}, nil
}

// cfEmailRuleToRule maps a cloudflare.EmailRoutingRule to our EmailRule type.
func cfEmailRuleToRule(r cloudflare.EmailRoutingRule) *EmailRule {
	matchers := make([]EmailRuleMatcher, len(r.Matchers))
	for i, m := range r.Matchers {
		matchers[i] = EmailRuleMatcher{
			Type:  m.Type,
			Field: m.Field,
			Value: m.Value,
		}
	}

	actions := make([]EmailRuleAction, len(r.Actions))
	for i, a := range r.Actions {
		actions[i] = EmailRuleAction{
			Type:  a.Type,
			Value: a.Value,
		}
	}

	return &EmailRule{
		ID:       r.Tag,
		Name:     r.Name,
		Priority: r.Priority,
		Enabled:  boolVal(r.Enabled),
		Matchers: matchers,
		Actions:  actions,
	}
}

// cfCatchAllToRule maps a cloudflare.EmailRoutingCatchAllRule to our EmailCatchAll type.
func cfCatchAllToRule(r cloudflare.EmailRoutingCatchAllRule) *EmailCatchAll {
	matchers := make([]EmailRuleMatcher, len(r.Matchers))
	for i, m := range r.Matchers {
		matchers[i] = EmailRuleMatcher{
			Type:  m.Type,
			Field: m.Field,
			Value: m.Value,
		}
	}

	actions := make([]EmailRuleAction, len(r.Actions))
	for i, a := range r.Actions {
		actions[i] = EmailRuleAction{
			Type:  a.Type,
			Value: a.Value,
		}
	}

	return &EmailCatchAll{
		ID:       r.Tag,
		Name:     r.Name,
		Enabled:  boolVal(r.Enabled),
		Matchers: matchers,
		Actions:  actions,
	}
}
