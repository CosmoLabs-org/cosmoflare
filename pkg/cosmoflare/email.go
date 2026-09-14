package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// EmailRule represents a Cloudflare Email Routing rule.
type EmailRule struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Priority int                `json:"priority"`
	Enabled  bool               `json:"enabled"`
	Matchers []EmailRuleMatcher `json:"matchers"`
	Actions  []EmailRuleAction  `json:"actions"`
}

// EmailRuleMatcher defines criteria for matching incoming emails.
type EmailRuleMatcher struct {
	Type  string `json:"type"`  // "literal" or "all"
	Field string `json:"field"` // "to"
	Value string `json:"value"` // email address pattern
}

// EmailRuleAction defines what to do with matched emails.
type EmailRuleAction struct {
	Type  string   `json:"type"`  // "forward" or "drop"
	Value []string `json:"value"` // destination addresses
}

// EmailDestination represents a verified destination address for email routing.
type EmailDestination struct {
	ID       string     `json:"id"`
	Email    string     `json:"email"`
	Verified *time.Time `json:"verified,omitempty"`
	Created  time.Time  `json:"created"`
	Modified time.Time  `json:"modified"`
}

// EmailCatchAll represents the catch-all email routing rule.
type EmailCatchAll struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Enabled  bool               `json:"enabled"`
	Matchers []EmailRuleMatcher `json:"matchers"`
	Actions  []EmailRuleAction  `json:"actions"`
}

// EmailSettings represents email routing settings for a zone.
type EmailSettings struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Enabled  bool       `json:"enabled"`
	Status   string     `json:"status"`
	Created  *time.Time `json:"created,omitempty"`
	Modified *time.Time `json:"modified,omitempty"`
}

// EmailService implements Cloudflare Email Routing operations.
// Rules are zone-scoped; destinations are account-scoped.
type EmailService struct {
	cf        *cloudflare.API
	zoneID    string
	accountID string
}

// NewEmailService creates a new Email Routing service client.
func NewEmailService(api *cloudflare.API, zoneID string, accountID string) (*EmailService, error) {
	if api == nil {
		return nil, validationError("NewEmailService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewEmailService", "zone ID is required")
	}
	if accountID == "" {
		return nil, validationError("NewEmailService", "account ID is required")
	}
	return &EmailService{cf: api, zoneID: zoneID, accountID: accountID}, nil
}

// NewEmailServiceFromCreds creates an EmailService from zone ID, account ID, and API token.
// Convenience helper for CLI usage.
func NewEmailServiceFromCreds(zoneID, accountID, apiToken string) (*EmailService, error) {
	if zoneID == "" {
		return nil, validationError("NewEmailService", "zone ID is required")
	}
	if accountID == "" {
		return nil, validationError("NewEmailService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewEmailService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewEmailService", "failed to create Cloudflare API client", err)
	}
	return &EmailService{cf: cf, zoneID: zoneID, accountID: accountID}, nil
}

// --- Rules (zone-scoped) ---

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

// CreateRule creates a new email routing rule.
func (s *EmailService) CreateRule(ctx context.Context, name string, matchers []EmailRuleMatcher, actions []EmailRuleAction, priority int, enabled bool) (*EmailRule, error) {
	if name == "" {
		return nil, validationError("EmailService.CreateRule", "rule name is required")
	}
	if len(matchers) == 0 {
		return nil, validationError("EmailService.CreateRule", "at least one matcher is required")
	}
	if len(actions) == 0 {
		return nil, validationError("EmailService.CreateRule", "at least one action is required")
	}

	cfMatchers := make([]cloudflare.EmailRoutingRuleMatcher, 0, len(matchers))
	for _, m := range matchers {
		cfMatchers = append(cfMatchers, cloudflare.EmailRoutingRuleMatcher{
			Type:  m.Type,
			Field: m.Field,
			Value: m.Value,
		})
	}

	cfActions := make([]cloudflare.EmailRoutingRuleAction, 0, len(actions))
	for _, a := range actions {
		cfActions = append(cfActions, cloudflare.EmailRoutingRuleAction{
			Type:  a.Type,
			Value: a.Value,
		})
	}

	params := cloudflare.CreateEmailRoutingRuleParameters{
		Name:     name,
		Matchers: cfMatchers,
		Actions:  cfActions,
		Priority: priority,
		Enabled:  boolPtr(enabled),
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.CreateEmailRoutingRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.CreateRule", fmt.Sprintf("failed to create email routing rule %q", name), err)
	}

	return cfEmailRuleToRule(result), nil
}

// UpdateRule updates an existing email routing rule.
func (s *EmailService) UpdateRule(ctx context.Context, ruleID string, name string, matchers []EmailRuleMatcher, actions []EmailRuleAction, priority int, enabled bool) (*EmailRule, error) {
	if ruleID == "" {
		return nil, validationError("EmailService.UpdateRule", "rule ID is required")
	}

	cfMatchers := make([]cloudflare.EmailRoutingRuleMatcher, 0, len(matchers))
	for _, m := range matchers {
		cfMatchers = append(cfMatchers, cloudflare.EmailRoutingRuleMatcher{
			Type:  m.Type,
			Field: m.Field,
			Value: m.Value,
		})
	}

	cfActions := make([]cloudflare.EmailRoutingRuleAction, 0, len(actions))
	for _, a := range actions {
		cfActions = append(cfActions, cloudflare.EmailRoutingRuleAction{
			Type:  a.Type,
			Value: a.Value,
		})
	}

	params := cloudflare.UpdateEmailRoutingRuleParameters{
		RuleID:   ruleID,
		Name:     name,
		Matchers: cfMatchers,
		Actions:  cfActions,
		Priority: priority,
		Enabled:  boolPtr(enabled),
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.UpdateEmailRoutingRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.UpdateRule", fmt.Sprintf("failed to update email routing rule %q", ruleID), err)
	}

	return cfEmailRuleToRule(result), nil
}

// DeleteRule removes an email routing rule.
func (s *EmailService) DeleteRule(ctx context.Context, ruleID string) error {
	if ruleID == "" {
		return validationError("EmailService.DeleteRule", "rule ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	_, err := s.cf.DeleteEmailRoutingRule(ctx, rc, ruleID)
	if err != nil {
		return newError("EmailService.DeleteRule", fmt.Sprintf("failed to delete email routing rule %q", ruleID), err)
	}
	return nil
}

// --- Destinations (account-scoped) ---

// ListDestinations returns all destination addresses for the account.
func (s *EmailService) ListDestinations(ctx context.Context) ([]*EmailDestination, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.ListEmailRoutingAddressParameters{}

	results, _, err := s.cf.ListEmailRoutingDestinationAddresses(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.ListDestinations", "failed to list email routing destinations", err)
	}

	destinations := make([]*EmailDestination, 0, len(results))
	for _, d := range results {
		destinations = append(destinations, cfEmailDestToDestination(d))
	}
	return destinations, nil
}

// CreateDestination adds a new destination address (requires email verification).
func (s *EmailService) CreateDestination(ctx context.Context, email string) (*EmailDestination, error) {
	if email == "" {
		return nil, validationError("EmailService.CreateDestination", "email address is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.CreateEmailRoutingAddressParameters{
		Email: email,
	}

	result, err := s.cf.CreateEmailRoutingDestinationAddress(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.CreateDestination", fmt.Sprintf("failed to create email destination %q", email), err)
	}

	return cfEmailDestToDestination(result), nil
}

// GetDestination retrieves a specific destination address by ID.
func (s *EmailService) GetDestination(ctx context.Context, addressID string) (*EmailDestination, error) {
	if addressID == "" {
		return nil, validationError("EmailService.GetDestination", "address ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetEmailRoutingDestinationAddress(ctx, rc, addressID)
	if err != nil {
		return nil, notFound("EmailService.GetDestination", "", addressID, err)
	}

	return cfEmailDestToDestination(result), nil
}

// DeleteDestination removes a destination address.
func (s *EmailService) DeleteDestination(ctx context.Context, addressID string) error {
	if addressID == "" {
		return validationError("EmailService.DeleteDestination", "address ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	_, err := s.cf.DeleteEmailRoutingDestinationAddress(ctx, rc, addressID)
	if err != nil {
		return newError("EmailService.DeleteDestination", fmt.Sprintf("failed to delete email destination %q", addressID), err)
	}
	return nil
}

// --- Catch-All (zone-scoped) ---

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

	params := cloudflare.EmailRoutingCatchAllRule{
		Name:    "catch-all",
		Enabled: boolPtr(enabled),
		Matchers: []cloudflare.EmailRoutingRuleMatcher{
			{Type: "all"},
		},
		Actions: []cloudflare.EmailRoutingRuleAction{
			{Type: "forward", Value: []string{forwardTo}},
		},
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	result, err := s.cf.UpdateEmailRoutingCatchAllRule(ctx, rc, params)
	if err != nil {
		return nil, newError("EmailService.UpdateCatchAll", "failed to update catch-all rule", err)
	}

	return cfCatchAllToRule(result), nil
}

// --- Settings (zone-scoped) ---

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

// --- Mapping helpers ---

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

func cfEmailDestToDestination(d cloudflare.EmailRoutingDestinationAddress) *EmailDestination {
	dest := &EmailDestination{
		ID:    d.Tag,
		Email: d.Email,
	}
	if d.Verified != nil {
		dest.Verified = d.Verified
	}
	if d.Created != nil {
		dest.Created = *d.Created
	}
	if d.Modified != nil {
		dest.Modified = *d.Modified
	}
	return dest
}
