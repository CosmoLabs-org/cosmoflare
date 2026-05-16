package r2go2

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// PageRule represents a Cloudflare Page Rule.
type PageRule struct {
	ID         string           `json:"id"`
	Targets    []PageRuleTarget `json:"targets"`
	Actions    []PageRuleAction `json:"actions"`
	Priority   int              `json:"priority"`
	Status     string           `json:"status"`
	CreatedOn  time.Time        `json:"created_on"`
	ModifiedOn time.Time        `json:"modified_on"`
}

// PageRuleTarget describes which URLs a page rule applies to.
type PageRuleTarget struct {
	Target     string `json:"target"`
	Constraint struct {
		Operator string `json:"operator"`
		Value    string `json:"value"`
	} `json:"constraint"`
}

// PageRuleAction describes an action taken when a page rule matches.
type PageRuleAction struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
}

// PageRuleService implements Page Rule operations for a Cloudflare zone.
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
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewPageRuleService", "failed to create Cloudflare API client", err)
	}
	return &PageRuleService{cf: cf, zoneID: zoneID}, nil
}

// Create creates a new page rule in the zone.
func (s *PageRuleService) Create(ctx context.Context, target string, actions []PageRuleAction, priority int, status string) (*PageRule, error) {
	if target == "" {
		return nil, validationError("PageRuleService.Create", "target URL pattern is required")
	}
	if len(actions) == 0 {
		return nil, validationError("PageRuleService.Create", "at least one action is required")
	}
	if status == "" {
		status = "active"
	}

	cfActions := make([]cloudflare.PageRuleAction, 0, len(actions))
	for _, a := range actions {
		cfActions = append(cfActions, cloudflare.PageRuleAction{
			ID:    a.ID,
			Value: a.Value,
		})
	}

	rule := cloudflare.PageRule{
		Targets: []cloudflare.PageRuleTarget{
			{
				Target: "url",
				Constraint: struct {
					Operator string `json:"operator"`
					Value    string `json:"value"`
				}{
					Operator: "matches",
					Value:    target,
				},
			},
		},
		Actions:  cfActions,
		Priority: priority,
		Status:   status,
	}

	resp, err := s.cf.CreatePageRule(ctx, s.zoneID, rule)
	if err != nil {
		return nil, newError("PageRuleService.Create", fmt.Sprintf("failed to create page rule for target %q", target), err)
	}

	return cfPageRuleToPageRule(*resp), nil
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

// Update modifies an existing page rule.
func (s *PageRuleService) Update(ctx context.Context, ruleID string, target string, actions []PageRuleAction, priority int, status string) error {
	if ruleID == "" {
		return validationError("PageRuleService.Update", "rule ID is required")
	}

	rule := cloudflare.PageRule{}

	if target != "" {
		rule.Targets = []cloudflare.PageRuleTarget{
			{
				Target: "url",
				Constraint: struct {
					Operator string `json:"operator"`
					Value    string `json:"value"`
				}{
					Operator: "matches",
					Value:    target,
				},
			},
		}
	}

	if len(actions) > 0 {
		cfActions := make([]cloudflare.PageRuleAction, 0, len(actions))
		for _, a := range actions {
			cfActions = append(cfActions, cloudflare.PageRuleAction{
				ID:    a.ID,
				Value: a.Value,
			})
		}
		rule.Actions = cfActions
	}

	if priority > 0 {
		rule.Priority = priority
	}

	if status != "" {
		rule.Status = status
	}

	err := s.cf.ChangePageRule(ctx, s.zoneID, ruleID, rule)
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
		pt := PageRuleTarget{
			Target: t.Target,
		}
		pt.Constraint.Operator = t.Constraint.Operator
		pt.Constraint.Value = t.Constraint.Value
		targets = append(targets, pt)
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
		Targets:    targets,
		Actions:    actions,
		Priority:   r.Priority,
		Status:     r.Status,
		CreatedOn:  r.CreatedOn,
		ModifiedOn: r.ModifiedOn,
	}
}
