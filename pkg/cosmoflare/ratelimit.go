package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// RateLimitRule is one http_ratelimit phase rule in our shape.
type RateLimitRule struct {
	ID                string   `json:"id"`
	Expression        string   `json:"expression"`
	Description       string   `json:"description,omitempty"`
	Enabled           bool     `json:"enabled"`
	Characteristics   []string `json:"characteristics,omitempty"`
	RequestsPerPeriod int      `json:"requests_per_period"`
	Period            int      `json:"period_seconds"`
	MitigationTimeout int      `json:"mitigation_timeout_seconds"`
}

// RateLimitCreateInput captures what a caller supplies to create one rule.
type RateLimitCreateInput struct {
	ZoneID            string
	Expression        string
	Description       string
	RequestsPerPeriod int
	Period            int
	MitigationTimeout int
	Characteristics   []string
}

// RateLimitService manages zone rate-limiting rules via the Rulesets
// http_ratelimit phase. It is the reference consumer of the knowledge
// layer: preflight validation before any call, decoded errors after.
type RateLimitService struct {
	cf    *cloudflare.API
	zones *ZoneService
}

// NewRateLimitService builds the service over an existing client.
func NewRateLimitService(cf *cloudflare.API, zones *ZoneService) (*RateLimitService, error) {
	if cf == nil {
		return nil, validationError("NewRateLimitService", "cloudflare client is required")
	}
	if zones == nil {
		return nil, validationError("NewRateLimitService", "zone service is required")
	}
	return &RateLimitService{cf: cf, zones: zones}, nil
}

// NewRateLimitServiceFromCreds builds the service with the knowledge
// transport wired into its client.
func NewRateLimitServiceFromCreds(accountID, apiToken string) (*RateLimitService, error) {
	if accountID == "" {
		return nil, validationError("NewRateLimitService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewRateLimitService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken,
		cloudflare.HTTPClient(&http.Client{Transport: &knowledge.Transport{}}))
	if err != nil {
		return nil, newError("NewRateLimitService", "failed to create cloudflare client", err)
	}
	zones, err := NewZoneServiceFromCreds(accountID, apiToken)
	if err != nil {
		return nil, err
	}
	return NewRateLimitService(cf, zones)
}

// cfRatelimitPhase is the Rulesets phase owning zone rate limiting.
const cfRatelimitPhase = "http_ratelimit"

// List returns the zone's rate-limiting rules. A missing phase entrypoint
// (fresh zone) is an empty list, not an error — this is the disambiguation
// CF's API does not offer.
func (s *RateLimitService) List(ctx context.Context, zoneID string) ([]RateLimitRule, error) {
	if zoneID == "" {
		return nil, validationError("RateLimitService.List", "zone ID is required")
	}
	rs, err := s.cf.GetEntrypointRuleset(ctx, cloudflare.ZoneIdentifier(zoneID), cfRatelimitPhase)
	if err != nil {
		if isNotFoundCF(err) {
			return []RateLimitRule{}, nil
		}
		return nil, newError("RateLimitService.List",
			fmt.Sprintf("failed to list rate-limit rules for zone %q", zoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}
	return fromCFRules(rs.Rules), nil
}

// Create appends one rate-limiting rule via the entrypoint PUT (cloudflare-go
// v0.116.0 exposes no per-rule create). Preflight validates caps + invariants
// before any API call; a fresh zone's missing entrypoint is created by the
// same PUT.
func (s *RateLimitService) Create(ctx context.Context, in RateLimitCreateInput) (*RateLimitRule, error) {
	if in.ZoneID == "" {
		return nil, validationError("RateLimitService.Create", "zone ID is required")
	}
	if in.Expression == "" {
		return nil, validationError("RateLimitService.Create", "expression is required")
	}

	plan := ""
	if z, err := s.zones.Get(ctx, in.ZoneID); err == nil && z != nil {
		plan = z.Plan.LegacyID
	}

	existing, err := s.List(ctx, in.ZoneID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"rules_count":                len(existing) + 1,
		"period_seconds":             in.Period,
		"mitigation_timeout_seconds": in.MitigationTimeout,
		"characteristics":            in.Characteristics,
	}
	if violations := knowledge.ValidatePayload(cfKnowledgeProduct, plan, payload); len(violations) > 0 {
		msgs := make([]string, 0, len(violations))
		for _, v := range violations {
			msgs = append(msgs, v.String())
		}
		return nil, validationError("RateLimitService.Create",
			fmt.Sprintf("payload rejected before send (plan %q): %s", plan, strings.Join(msgs, "; ")))
	}

	rc := cloudflare.ZoneIdentifier(in.ZoneID)
	rs, err := s.cf.GetEntrypointRuleset(ctx, rc, cfRatelimitPhase)
	rules := make([]cloudflare.RulesetRule, 0, len(existing)+1)
	if err == nil {
		rules = append(rules, rs.Rules...)
	} else if !isNotFoundCF(err) {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("failed to read entrypoint for zone %q", in.ZoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}

	enabled := true
	rules = append(rules, cloudflare.RulesetRule{
		Action:      "block",
		Expression:  in.Expression,
		Description: in.Description,
		Enabled:     &enabled,
		RateLimit: &cloudflare.RulesetRuleRateLimit{
			Characteristics:   in.Characteristics,
			RequestsPerPeriod: in.RequestsPerPeriod,
			Period:            in.Period,
			MitigationTimeout: in.MitigationTimeout,
		},
	})

	updated, err := s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
		Phase: cfRatelimitPhase,
		Rules: rules,
	})
	if err != nil {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("failed to apply rate-limit rule to zone %q", in.ZoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}

	// The new rule is the one we appended.
	for _, r := range fromCFRules(updated.Rules) {
		if r.Expression == in.Expression && r.RequestsPerPeriod == in.RequestsPerPeriod {
			return &r, nil
		}
	}
	rules2 := fromCFRules(updated.Rules)
	if len(rules2) == 0 {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("zone %q entrypoint returned zero rules after update", in.ZoneID), nil)
	}
	return &rules2[len(rules2)-1], nil
}

// fromCFRules maps SDK rules into our shape.
func fromCFRules(in []cloudflare.RulesetRule) []RateLimitRule {
	out := make([]RateLimitRule, 0, len(in))
	for _, r := range in {
		rr := RateLimitRule{
			ID:          r.ID,
			Expression:  r.Expression,
			Description: r.Description,
			Enabled:     r.Enabled == nil || *r.Enabled,
		}
		if r.RateLimit != nil {
			rr.Characteristics = r.RateLimit.Characteristics
			rr.RequestsPerPeriod = r.RateLimit.RequestsPerPeriod
			rr.Period = r.RateLimit.Period
			rr.MitigationTimeout = r.RateLimit.MitigationTimeout
		}
		out = append(out, rr)
	}
	return out
}

// isNotFoundCF reports CF not-found shapes: HTTP 404 or code 1000
// (cloudflare-go v0.116.0 has no ErrNotFound sentinel).
func isNotFoundCF(err error) bool {
	if err == nil {
		return false
	}
	var cfErr *cloudflare.Error
	if errors.As(err, &cfErr) {
		if cfErr.StatusCode == http.StatusNotFound {
			return true
		}
		for _, c := range cfErr.ErrorCodes {
			if c == 1000 {
				return true
			}
		}
	}
	return false
}
