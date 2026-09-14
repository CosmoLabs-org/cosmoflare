package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// PageRule represents a Cloudflare Page Rule.
type PageRule struct {
	ID         string           `json:"id"`
	Status     string           `json:"status"`
	Priority   int              `json:"priority"`
	Targets    []PageRuleTarget `json:"targets"`
	Actions    []PageRuleAction `json:"actions"`
	CreatedOn  time.Time        `json:"created_on"`
	ModifiedOn time.Time        `json:"modified_on"`
}

// PageRuleTarget specifies the URL pattern a page rule matches against.
type PageRuleTarget struct {
	Target     string             `json:"target"`
	Constraint PageRuleConstraint `json:"constraint"`
}

// PageRuleConstraint defines the match operator and URL pattern.
type PageRuleConstraint struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// PageRuleAction defines an action to take when a page rule matches.
type PageRuleAction struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
}

// PageRuleService implements Page Rule operations.
// Page Rules are zone-scoped — they use a zone ID, not an account ID.
type PageRuleService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewPageRuleService creates a new Page Rule service client.
func NewPageRuleService(api *cloudflare.API, zoneID string) (*PageRuleService, error) {
	if api == nil {
		return nil, validationError("NewPageRuleService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewPageRuleService", "zone ID is required")
	}
	return &PageRuleService{cf: api, zoneID: zoneID}, nil
}

// NewPageRuleServiceFromCreds creates a PageRuleService from zone ID and API token.
// Convenience helper for CLI usage.
func NewPageRuleServiceFromCreds(zoneID, apiToken string) (*PageRuleService, error) {
	if zoneID == "" {
		return nil, validationError("NewPageRuleService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewPageRuleService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewPageRuleService", "failed to create Cloudflare API client", err)
	}
	return &PageRuleService{cf: cf, zoneID: zoneID}, nil
}

// List returns all page rules in the zone.
func (s *PageRuleService) List(ctx context.Context) ([]*PageRule, error) {
	results, err := s.cf.ListPageRules(ctx, s.zoneID)
	if err != nil {
		return nil, newError("PageRuleService.List", "failed to list page rules", err)
	}

	rules := make([]*PageRule, 0, len(results))
	for _, r := range results {
		rules = append(rules, cfPageRuleToPageRule(r))
	}
	return rules, nil
}

// Get retrieves a single page rule by ID.
func (s *PageRuleService) Get(ctx context.Context, ruleID string) (*PageRule, error) {
	if ruleID == "" {
		return nil, validationError("PageRuleService.Get", "rule ID is required")
	}

	resp, err := s.cf.PageRule(ctx, s.zoneID, ruleID)
	if err != nil {
		return nil, notFound("PageRuleService.Get", "", ruleID, err)
	}

	return cfPageRuleToPageRule(resp), nil
}

// Create creates a new page rule.
func (s *PageRuleService) Create(ctx context.Context, targets []PageRuleTarget, actions []PageRuleAction, status string, priority int) (*PageRule, error) {
	if len(targets) == 0 {
		return nil, validationError("PageRuleService.Create", "at least one target is required")
	}
	if len(actions) == 0 {
		return nil, validationError("PageRuleService.Create", "at least one action is required")
	}
	if status == "" {
		status = "active"
	}
	if priority < 1 {
		priority = 1
	}

	cfTargets := make([]cloudflare.PageRuleTarget, 0, len(targets))
	for _, t := range targets {
		cfTargets = append(cfTargets, cloudflare.PageRuleTarget{
			Target: t.Target,
			Constraint: struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			}{
				Operator: t.Constraint.Operator,
				Value:    t.Constraint.Value,
			},
		})
	}

	cfActions := make([]cloudflare.PageRuleAction, 0, len(actions))
	for _, a := range actions {
		cfActions = append(cfActions, cloudflare.PageRuleAction{
			ID:    a.ID,
			Value: a.Value,
		})
	}

	rule := cloudflare.PageRule{
		Targets:  cfTargets,
		Actions:  cfActions,
		Status:   status,
		Priority: priority,
	}

	resp, err := s.cf.CreatePageRule(ctx, s.zoneID, rule)
	if err != nil {
		return nil, newError("PageRuleService.Create", "failed to create page rule", err)
	}

	return cfPageRuleToPageRule(*resp), nil
}

// Update replaces an existing page rule.
func (s *PageRuleService) Update(ctx context.Context, ruleID string, targets []PageRuleTarget, actions []PageRuleAction, status string, priority int) error {
	if ruleID == "" {
		return validationError("PageRuleService.Update", "rule ID is required")
	}
	if len(targets) == 0 {
		return validationError("PageRuleService.Update", "at least one target is required")
	}
	if len(actions) == 0 {
		return validationError("PageRuleService.Update", "at least one action is required")
	}
	if status == "" {
		status = "active"
	}
	if priority < 1 {
		priority = 1
	}

	cfTargets := make([]cloudflare.PageRuleTarget, 0, len(targets))
	for _, t := range targets {
		cfTargets = append(cfTargets, cloudflare.PageRuleTarget{
			Target: t.Target,
			Constraint: struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			}{
				Operator: t.Constraint.Operator,
				Value:    t.Constraint.Value,
			},
		})
	}

	cfActions := make([]cloudflare.PageRuleAction, 0, len(actions))
	for _, a := range actions {
		cfActions = append(cfActions, cloudflare.PageRuleAction{
			ID:    a.ID,
			Value: a.Value,
		})
	}

	rule := cloudflare.PageRule{
		Targets:  cfTargets,
		Actions:  cfActions,
		Status:   status,
		Priority: priority,
	}

	err := s.cf.UpdatePageRule(ctx, s.zoneID, ruleID, rule)
	if err != nil {
		return newError("PageRuleService.Update", fmt.Sprintf("failed to update page rule %q", ruleID), err)
	}
	return nil
}

// Delete removes a page rule.
func (s *PageRuleService) Delete(ctx context.Context, ruleID string) error {
	if ruleID == "" {
		return validationError("PageRuleService.Delete", "rule ID is required")
	}

	err := s.cf.DeletePageRule(ctx, s.zoneID, ruleID)
	if err != nil {
		return newError("PageRuleService.Delete", fmt.Sprintf("failed to delete page rule %q", ruleID), err)
	}
	return nil
}

// cfPageRuleToPageRule maps a cloudflare.PageRule to our PageRule type.
func cfPageRuleToPageRule(r cloudflare.PageRule) *PageRule {
	targets := make([]PageRuleTarget, 0, len(r.Targets))
	for _, t := range r.Targets {
		targets = append(targets, PageRuleTarget{
			Target: t.Target,
			Constraint: PageRuleConstraint{
				Operator: t.Constraint.Operator,
				Value:    t.Constraint.Value,
			},
		})
	}

	actions := make([]PageRuleAction, 0, len(r.Actions))
	for _, a := range r.Actions {
		actions = append(actions, PageRuleAction{
			ID:    a.ID,
			Value: a.Value,
		})
	}

	return &PageRule{
		ID:         r.ID,
		Status:     r.Status,
		Priority:   r.Priority,
		Targets:    targets,
		Actions:    actions,
		CreatedOn:  r.CreatedOn,
		ModifiedOn: r.ModifiedOn,
	}
}
