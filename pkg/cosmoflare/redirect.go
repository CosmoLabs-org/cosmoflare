package cosmoflare

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
)

// redirectPhase is the Rulesets API phase used for modern (dynamic) Redirect
// Rules. A zone has at most one entrypoint ruleset for this phase, and that
// ruleset's Rules slice holds the individual redirect rules.
const redirectPhase = string(cloudflare.RulesetPhaseHTTPRequestDynamicRedirect)

// redirectAction is the ruleset rule action that performs a dynamic redirect.
const redirectAction = "redirect"

// RedirectRule is a domain-agnostic view of a Cloudflare Redirect Rule
// (Rulesets API, phase http_request_dynamic_redirect).
type RedirectRule struct {
	ID            string `json:"id"`
	ZoneID        string `json:"zone_id"`
	When          string `json:"when"`        // expression / URL pattern
	Destination   string `json:"destination"` // target URL (may use $1..$n captures)
	StatusCode    int    `json:"status_code"` // 301, 302, 307, 308
	PreserveQuery bool   `json:"preserve_query"`
	Enabled       bool   `json:"enabled"`
}

// RedirectRuleInput captures the fields required to create a redirect rule.
type RedirectRuleInput struct {
	ZoneID        string
	When          string
	Destination   string
	StatusCode    int
	PreserveQuery bool
}

// RedirectService manages Cloudflare Redirect Rules via the modern Rulesets API.
type RedirectService struct {
	cf        *cloudflare.API
	accountID string
}

// NewRedirectService builds a RedirectService bound to the given Cloudflare
// client and account ID.
func NewRedirectService(cf *cloudflare.API, accountID string) *RedirectService {
	return &RedirectService{cf: cf, accountID: accountID}
}

// List returns the redirect rules configured for a zone. If the zone has no
// dynamic-redirect phase ruleset, an empty slice is returned (not an error).
func (s *RedirectService) List(ctx context.Context, zoneID string) ([]RedirectRule, error) {
	if zoneID == "" {
		return nil, validationError("RedirectService.List", "zone ID is required")
	}
	rc := cloudflare.ZoneIdentifier(zoneID)
	sets, err := s.cf.ListRulesets(ctx, rc, cloudflare.ListRulesetsParams{})
	if err != nil {
		return nil, newError("RedirectService.List", "list rulesets", err)
	}
	var phaseID string
	for _, rs := range sets {
		if rs.Phase == redirectPhase {
			phaseID = rs.ID
			break
		}
	}
	if phaseID == "" {
		return []RedirectRule{}, nil
	}
	rs, err := s.cf.GetRuleset(ctx, rc, phaseID)
	if err != nil {
		return nil, newError("RedirectService.List", "get redirect ruleset", err)
	}
	out := make([]RedirectRule, 0, len(rs.Rules))
	for _, r := range rs.Rules {
		out = append(out, mapRedirectRule(r, zoneID))
	}
	return out, nil
}

// mapRedirectRule converts a cloudflare-go RulesetRule into the domain-agnostic
// RedirectRule. ActionParameters is a typed struct in this SDK version, so the
// redirect details live under FromValue.
func mapRedirectRule(r cloudflare.RulesetRule, zoneID string) RedirectRule {
	rr := RedirectRule{
		ID:      r.ID,
		ZoneID:  zoneID,
		When:    r.Expression,
		Enabled: r.Enabled != nil && *r.Enabled,
	}
	if r.ActionParameters != nil && r.ActionParameters.FromValue != nil {
		fv := r.ActionParameters.FromValue
		rr.Destination = fv.TargetURL.Value
		rr.StatusCode = int(fv.StatusCode)
		rr.PreserveQuery = fv.PreserveQueryString != nil && *fv.PreserveQueryString
	}
	return rr
}

// phaseRuleset resolves the dynamic-redirect phase ruleset for a zone. It
// returns the ruleset ID (empty string if none exists) and the full ruleset.
func (s *RedirectService) phaseRuleset(ctx context.Context, zoneID string) (string, cloudflare.Ruleset, error) {
	rc := cloudflare.ZoneIdentifier(zoneID)
	sets, err := s.cf.ListRulesets(ctx, rc, cloudflare.ListRulesetsParams{})
	if err != nil {
		return "", cloudflare.Ruleset{}, newError("RedirectService", "list rulesets", err)
	}
	for _, rs := range sets {
		if rs.Phase == redirectPhase {
			full, err := s.cf.GetRuleset(ctx, rc, rs.ID)
			if err != nil {
				return "", cloudflare.Ruleset{}, newError("RedirectService", "get redirect ruleset", err)
			}
			return rs.ID, full, nil
		}
	}
	return "", cloudflare.Ruleset{}, nil
}

// Create adds a redirect rule to the zone's dynamic-redirect phase ruleset,
// creating the ruleset if it does not yet exist. The created rule is echoed
// back as a RedirectRule.
func (s *RedirectService) Create(ctx context.Context, in RedirectRuleInput) (RedirectRule, error) {
	if in.ZoneID == "" || in.Destination == "" {
		return RedirectRule{}, validationError("RedirectService.Create", "zone ID and destination required")
	}

	rc := cloudflare.ZoneIdentifier(in.ZoneID)

	preserve := in.PreserveQuery
	rule := cloudflare.RulesetRule{
		Action:     redirectAction,
		Expression: in.When,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			FromValue: &cloudflare.RulesetRuleActionParametersFromValue{
				StatusCode: uint16(in.StatusCode),
				TargetURL: cloudflare.RulesetRuleActionParametersTargetURL{
					Value: in.Destination,
				},
				PreserveQueryString: &preserve,
			},
		},
	}

	phaseID, existing, err := s.phaseRuleset(ctx, in.ZoneID)
	if err != nil {
		return RedirectRule{}, err
	}

	var result cloudflare.Ruleset
	if phaseID == "" {
		result, err = s.cf.CreateRuleset(ctx, rc, cloudflare.CreateRulesetParams{
			Name:  "Redirect Rules",
			Kind:  "zone",
			Phase: redirectPhase,
			Rules: []cloudflare.RulesetRule{rule},
		})
		if err != nil {
			return RedirectRule{}, newError("RedirectService.Create", "create redirect ruleset", err)
		}
	} else {
		rules := append(existing.Rules, rule)
		result, err = s.cf.UpdateRuleset(ctx, rc, cloudflare.UpdateRulesetParams{
			ID:          phaseID,
			Description: existing.Description,
			Rules:       rules,
		})
		if err != nil {
			return RedirectRule{}, newError("RedirectService.Create", "update redirect ruleset", err)
		}
	}

	// Echo the newly created rule. Prefer the last rule returned by the API
	// (it carries the server-assigned ID); fall back to the input on mismatch.
	out := RedirectRule{
		ZoneID:        in.ZoneID,
		When:          in.When,
		Destination:   in.Destination,
		StatusCode:    in.StatusCode,
		PreserveQuery: in.PreserveQuery,
	}
	if n := len(result.Rules); n > 0 {
		out = mapRedirectRule(result.Rules[n-1], in.ZoneID)
	}
	return out, nil
}

// Delete removes a redirect rule from the zone's dynamic-redirect phase ruleset.
func (s *RedirectService) Delete(ctx context.Context, zoneID, ruleID string) error {
	if zoneID == "" || ruleID == "" {
		return validationError("RedirectService.Delete", "zone ID and rule ID required")
	}
	phaseID, _, err := s.phaseRuleset(ctx, zoneID)
	if err != nil {
		return err
	}
	if phaseID == "" {
		return validationError("RedirectService.Delete", "no redirect phase configured")
	}
	rc := cloudflare.ZoneIdentifier(zoneID)
	if err := s.cf.DeleteRulesetRule(ctx, rc, cloudflare.DeleteRulesetRuleParams{
		RulesetID:     phaseID,
		RulesetRuleID: ruleID,
	}); err != nil {
		return newError("RedirectService.Delete", "delete rule", err)
	}
	return nil
}
