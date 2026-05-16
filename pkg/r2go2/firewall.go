package r2go2

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// FirewallFilterRule represents a Cloudflare Firewall Rule (filter expression-based).
// This is distinct from FirewallRule (IP access rules) in waf.go.
type FirewallFilterRule struct {
	ID          string           `json:"id"`
	Description string           `json:"description"`
	Action      string           `json:"action"`
	Priority    int              `json:"priority,omitempty"`
	Paused      bool             `json:"paused"`
	Filter      FilterExpression `json:"filter"`
	CreatedOn   time.Time        `json:"created_on"`
	ModifiedOn  time.Time        `json:"modified_on"`
}

// FilterExpression represents a Cloudflare filter expression used by firewall rules.
type FilterExpression struct {
	ID          string `json:"id"`
	Expression  string `json:"expression"`
	Description string `json:"description,omitempty"`
	Paused      bool   `json:"paused"`
}

// FirewallService implements Cloudflare Firewall Rules operations.
// Firewall rules are zone-scoped — they use a zone ID, not an account ID.
type FirewallService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewFirewallService creates a new Firewall service client.
func NewFirewallService(api *cloudflare.API, zoneID string) (*FirewallService, error) {
	if api == nil {
		return nil, validationError("NewFirewallService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewFirewallService", "zone ID is required")
	}
	return &FirewallService{cf: api, zoneID: zoneID}, nil
}

// NewFirewallServiceFromCreds creates a FirewallService from zone ID and API token.
// Convenience helper for CLI usage.
func NewFirewallServiceFromCreds(zoneID, apiToken string) (*FirewallService, error) {
	if zoneID == "" {
		return nil, validationError("NewFirewallService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewFirewallService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewFirewallService", "failed to create Cloudflare API client", err)
	}
	return &FirewallService{cf: cf, zoneID: zoneID}, nil
}

// List returns all firewall rules in the zone.
func (s *FirewallService) List(ctx context.Context) ([]*FirewallFilterRule, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	params := cloudflare.FirewallRuleListParams{}

	results, _, err := s.cf.FirewallRules(ctx, rc, params)
	if err != nil {
		return nil, newError("FirewallService.List", "failed to list firewall rules", err)
	}

	rules := make([]*FirewallFilterRule, 0, len(results))
	for _, r := range results {
		rules = append(rules, cfFirewallToRule(r))
	}
	return rules, nil
}

// Get retrieves a single firewall rule by ID.
func (s *FirewallService) Get(ctx context.Context, ruleID string) (*FirewallFilterRule, error) {
	if ruleID == "" {
		return nil, validationError("FirewallService.Get", "rule ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.FirewallRule(ctx, rc, ruleID)
	if err != nil {
		return nil, notFound("FirewallService.Get", "", ruleID, err)
	}

	return cfFirewallToRule(result), nil
}

// Create creates a new firewall rule with the given filter expression and action.
func (s *FirewallService) Create(ctx context.Context, expression, action, description string) ([]*FirewallFilterRule, error) {
	if expression == "" {
		return nil, validationError("FirewallService.Create", "filter expression is required")
	}
	if action == "" {
		return nil, validationError("FirewallService.Create", "action is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	params := []cloudflare.FirewallRuleCreateParams{
		{
			Action:      action,
			Description: description,
			Filter: cloudflare.Filter{
				Expression:  expression,
				Description: description,
			},
		},
	}

	results, err := s.cf.CreateFirewallRules(ctx, rc, params)
	if err != nil {
		return nil, newError("FirewallService.Create", fmt.Sprintf("failed to create firewall rule with action %q", action), err)
	}

	rules := make([]*FirewallFilterRule, 0, len(results))
	for _, r := range results {
		rules = append(rules, cfFirewallToRule(r))
	}
	return rules, nil
}

// Update modifies an existing firewall rule.
func (s *FirewallService) Update(ctx context.Context, ruleID, expression, action, description string) (*FirewallFilterRule, error) {
	if ruleID == "" {
		return nil, validationError("FirewallService.Update", "rule ID is required")
	}
	if expression == "" {
		return nil, validationError("FirewallService.Update", "filter expression is required")
	}
	if action == "" {
		return nil, validationError("FirewallService.Update", "action is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	params := cloudflare.FirewallRuleUpdateParams{
		ID:          ruleID,
		Action:      action,
		Description: description,
		Filter: cloudflare.Filter{
			Expression:  expression,
			Description: description,
		},
	}

	result, err := s.cf.UpdateFirewallRule(ctx, rc, params)
	if err != nil {
		return nil, newError("FirewallService.Update", fmt.Sprintf("failed to update firewall rule %q", ruleID), err)
	}

	return cfFirewallToRule(result), nil
}

// Delete removes a firewall rule by ID.
func (s *FirewallService) Delete(ctx context.Context, ruleID string) error {
	if ruleID == "" {
		return validationError("FirewallService.Delete", "rule ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	err := s.cf.DeleteFirewallRule(ctx, rc, ruleID)
	if err != nil {
		return newError("FirewallService.Delete", fmt.Sprintf("failed to delete firewall rule %q", ruleID), err)
	}
	return nil
}

// cfFirewallToRule maps a cloudflare.FirewallRule to our FirewallFilterRule type.
func cfFirewallToRule(r cloudflare.FirewallRule) *FirewallFilterRule {
	priority := 0
	if r.Priority != nil {
		switch v := r.Priority.(type) {
		case float64:
			priority = int(v)
		case int:
			priority = v
		}
	}

	return &FirewallFilterRule{
		ID:          r.ID,
		Description: r.Description,
		Action:      r.Action,
		Priority:    priority,
		Paused:      r.Paused,
		Filter: FilterExpression{
			ID:          r.Filter.ID,
			Expression:  r.Filter.Expression,
			Description: r.Filter.Description,
			Paused:      r.Filter.Paused,
		},
		CreatedOn:  r.CreatedOn,
		ModifiedOn: r.ModifiedOn,
	}
}
