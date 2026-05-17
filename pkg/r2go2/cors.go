package r2go2

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

const (
	corsPhase = "http_response_headers_transform"
	corsDefaultExpr = "true"

	// CORSDefaultRuleName is the default identifier used for CORS rules managed by Cosmoflare.
	CORSDefaultRuleName = "cosmoflare-cors"
)

// ErrCORSRuleNotFound is returned when a CORS rule with the given name does not exist.
var ErrCORSRuleNotFound = errors.New("CORS rule not found")

// CORSRule represents a single CORS configuration rule in a Transform Ruleset.
type CORSRule struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	Expression       string   `json:"expression"`
	AllowOrigins     []string `json:"allow_origins"`
	AllowMethods     []string `json:"allow_methods"`
	AllowHeaders     []string `json:"allow_headers"`
	MaxAge           int      `json:"max_age,omitempty"`
	AllowCredentials bool     `json:"allow_credentials"`
}

// CORSOption is a functional option for building a CORS rule.
type CORSOption func(*corsConfig)

type corsConfig struct {
	name             string
	expression       string
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	maxAge           int
	allowCredentials bool
}

func defaultCORSConfig() *corsConfig {
	return &corsConfig{
		name:         CORSDefaultRuleName,
		expression:   corsDefaultExpr,
		allowMethods: []string{"GET", "POST", "OPTIONS"},
		allowHeaders: []string{"Content-Type", "Authorization"},
		maxAge:       86400,
	}
}

// WithCORSName sets the name/identifier for the CORS rule (maps to Description).
func WithCORSName(name string) CORSOption {
	return func(c *corsConfig) { c.name = name }
}

// WithCORSOrigins sets the allowed origins (e.g. "*", "https://example.com").
func WithCORSOrigins(origins ...string) CORSOption {
	return func(c *corsConfig) { c.allowOrigins = origins }
}

// WithCORSMethods sets the allowed HTTP methods.
func WithCORSMethods(methods ...string) CORSOption {
	return func(c *corsConfig) { c.allowMethods = methods }
}

// WithCORSHeaders sets the allowed request headers.
func WithCORSHeaders(headers ...string) CORSOption {
	return func(c *corsConfig) { c.allowHeaders = headers }
}

// WithCORSMaxAge sets the Access-Control-Max-Age value in seconds.
func WithCORSMaxAge(seconds int) CORSOption {
	return func(c *corsConfig) { c.maxAge = seconds }
}

// WithCORSCredentials sets whether Access-Control-Allow-Credentials is true.
func WithCORSCredentials(allow bool) CORSOption {
	return func(c *corsConfig) { c.allowCredentials = allow }
}

// WithCORSExpression sets the Wirefilter expression for the rule (default: "true").
func WithCORSExpression(expr string) CORSOption {
	return func(c *corsConfig) { c.expression = expr }
}

// CORSService manages CORS response headers via Cloudflare Transform Rules.
// CORS is zone-scoped — requires a zone ID.
type CORSService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewCORSService creates a new CORSService with an existing Cloudflare API client.
func NewCORSService(api *cloudflare.API, zoneID string) (*CORSService, error) {
	if api == nil {
		return nil, validationError("NewCORSService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewCORSService", "zone ID is required")
	}
	return &CORSService{cf: api, zoneID: zoneID}, nil
}

// NewCORSServiceFromCreds creates a CORSService from a zone ID and API token.
func NewCORSServiceFromCreds(zoneID, apiToken string) (*CORSService, error) {
	if zoneID == "" {
		return nil, validationError("NewCORSService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewCORSService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewCORSService", "failed to create Cloudflare API client", err)
	}
	return &CORSService{cf: cf, zoneID: zoneID}, nil
}

// GetCORSRules returns all CORS rules from the zone's response header transform ruleset.
// Returns an empty slice (not an error) if no transform ruleset exists yet.
func (s *CORSService) GetCORSRules(ctx context.Context) ([]*CORSRule, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	ruleset, err := s.cf.GetEntrypointRuleset(ctx, rc, corsPhase)
	if err != nil {
		if isNotFoundError(err) {
			return []*CORSRule{}, nil
		}
		return nil, newError("CORSService.GetCORSRules", "failed to get entrypoint ruleset", err)
	}

	var result []*CORSRule
	for _, rule := range ruleset.Rules {
		if cr, ok := parseCORSRule(rule); ok {
			result = append(result, cr)
		}
	}
	if result == nil {
		result = []*CORSRule{}
	}
	return result, nil
}

// SetCORSHeaders creates or replaces a CORS rule in the entrypoint ruleset.
// If a rule with the given name (description) already exists it is replaced in-place.
// If no ruleset exists yet, one is created via UpdateEntrypointRuleset.
func (s *CORSService) SetCORSHeaders(ctx context.Context, opts ...CORSOption) (*CORSRule, error) {
	cfg := defaultCORSConfig()
	for _, o := range opts {
		o(cfg)
	}

	if len(cfg.allowOrigins) == 0 {
		return nil, validationError("CORSService.SetCORSHeaders", "at least one origin is required (use WithCORSOrigins)")
	}

	// Reject credentials + wildcard combination per CORS spec
	if cfg.allowCredentials {
		for _, o := range cfg.allowOrigins {
			if o == "*" {
				return nil, validationError("CORSService.SetCORSHeaders",
					`cannot use AllowCredentials with wildcard origin "*" — specify an explicit origin`)
			}
		}
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)

	// 1. Read current state (404 = no ruleset yet, treat as empty)
	existing, err := s.cf.GetEntrypointRuleset(ctx, rc, corsPhase)
	var existingRules []cloudflare.RulesetRule
	if err != nil {
		if !isNotFoundError(err) {
			return nil, newError("CORSService.SetCORSHeaders", "failed to read entrypoint ruleset", err)
		}
		// 404 — start with empty rule list
	} else {
		existingRules = existing.Rules
	}

	// 2. Build new rule from config
	newRule := buildRulesetRule(cfg)

	// 3. Upsert — replace by description or append
	rules := upsertRule(existingRules, newRule, cfg.name)

	// 4. Write back
	updated, err := s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
		Phase: corsPhase,
		Rules: rules,
	})
	if err != nil {
		return nil, newError("CORSService.SetCORSHeaders", "failed to update entrypoint ruleset", err)
	}

	// Find and return the rule we just upserted
	for _, r := range updated.Rules {
		if r.Description == cfg.name {
			if cr, ok := parseCORSRule(r); ok {
				return cr, nil
			}
		}
	}

	// Fallback: build the CORSRule from our config
	return corsRuleFromConfig(cfg), nil
}

// RemoveCORSRule removes a CORS rule from the entrypoint ruleset by name (description).
// Returns ErrCORSRuleNotFound if no rule with that name exists.
func (s *CORSService) RemoveCORSRule(ctx context.Context, name string) error {
	if name == "" {
		name = CORSDefaultRuleName
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	ruleset, err := s.cf.GetEntrypointRuleset(ctx, rc, corsPhase)
	if err != nil {
		if isNotFoundError(err) {
			return ErrCORSRuleNotFound
		}
		return newError("CORSService.RemoveCORSRule", "failed to read entrypoint ruleset", err)
	}

	filtered := make([]cloudflare.RulesetRule, 0, len(ruleset.Rules))
	found := false
	for _, r := range ruleset.Rules {
		if r.Description == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}

	if !found {
		return ErrCORSRuleNotFound
	}

	_, err = s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
		Phase: corsPhase,
		Rules: filtered,
	})
	if err != nil {
		return newError("CORSService.RemoveCORSRule", "failed to update entrypoint ruleset", err)
	}
	return nil
}

// --- Internal helpers ---

// corsHeaderKeys are the CORS-related response headers we manage.
var corsHeaderKeys = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Max-Age",
	"Access-Control-Allow-Credentials",
}

// isCORSHeader reports whether key is a CORS response header.
func isCORSHeader(key string) bool {
	for _, k := range corsHeaderKeys {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

// parseCORSRule converts a cloudflare.RulesetRule into a CORSRule.
// Returns (nil, false) if the rule has no CORS headers.
func parseCORSRule(r cloudflare.RulesetRule) (*CORSRule, bool) {
	if r.ActionParameters == nil || len(r.ActionParameters.Headers) == 0 {
		return nil, false
	}

	hasCORS := false
	for k := range r.ActionParameters.Headers {
		if isCORSHeader(k) {
			hasCORS = true
			break
		}
	}
	if !hasCORS {
		return nil, false
	}

	cr := &CORSRule{
		ID:         r.ID,
		Name:       r.Description,
		Expression: r.Expression,
	}
	if r.Enabled != nil {
		cr.Enabled = *r.Enabled
	}

	hdrs := r.ActionParameters.Headers
	if h, ok := hdrs["Access-Control-Allow-Origin"]; ok {
		cr.AllowOrigins = splitTrimmed(h.Value)
	}
	if h, ok := hdrs["Access-Control-Allow-Methods"]; ok {
		cr.AllowMethods = splitTrimmed(h.Value)
	}
	if h, ok := hdrs["Access-Control-Allow-Headers"]; ok {
		cr.AllowHeaders = splitTrimmed(h.Value)
	}
	if h, ok := hdrs["Access-Control-Max-Age"]; ok {
		if n, err := strconv.Atoi(h.Value); err == nil {
			cr.MaxAge = n
		}
	}
	if h, ok := hdrs["Access-Control-Allow-Credentials"]; ok {
		cr.AllowCredentials = h.Value == "true"
	}

	return cr, true
}

// buildRulesetRule builds a cloudflare.RulesetRule from a corsConfig.
func buildRulesetRule(cfg *corsConfig) cloudflare.RulesetRule {
	enabled := true
	headers := map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{}

	// Origin: join multiple with ", " (best-effort for static value)
	originVal := strings.Join(cfg.allowOrigins, ", ")
	headers["Access-Control-Allow-Origin"] = cloudflare.RulesetRuleActionParametersHTTPHeader{
		Operation: "set",
		Value:     originVal,
	}

	if len(cfg.allowMethods) > 0 {
		headers["Access-Control-Allow-Methods"] = cloudflare.RulesetRuleActionParametersHTTPHeader{
			Operation: "set",
			Value:     strings.Join(cfg.allowMethods, ", "),
		}
	}

	if len(cfg.allowHeaders) > 0 {
		headers["Access-Control-Allow-Headers"] = cloudflare.RulesetRuleActionParametersHTTPHeader{
			Operation: "set",
			Value:     strings.Join(cfg.allowHeaders, ", "),
		}
	}

	if cfg.maxAge > 0 {
		headers["Access-Control-Max-Age"] = cloudflare.RulesetRuleActionParametersHTTPHeader{
			Operation: "set",
			Value:     strconv.Itoa(cfg.maxAge),
		}
	}

	if cfg.allowCredentials {
		headers["Access-Control-Allow-Credentials"] = cloudflare.RulesetRuleActionParametersHTTPHeader{
			Operation: "set",
			Value:     "true",
		}
	}

	return cloudflare.RulesetRule{
		Action:      string(cloudflare.RulesetRuleActionRewrite),
		Expression:  cfg.expression,
		Description: cfg.name,
		Enabled:     &enabled,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: headers,
		},
	}
}

// upsertRule replaces an existing rule by description or appends a new one.
func upsertRule(rules []cloudflare.RulesetRule, newRule cloudflare.RulesetRule, name string) []cloudflare.RulesetRule {
	for i, r := range rules {
		if r.Description == name {
			rules[i] = newRule
			return rules
		}
	}
	return append(rules, newRule)
}

// corsRuleFromConfig builds a CORSRule from a corsConfig (fallback).
func corsRuleFromConfig(cfg *corsConfig) *CORSRule {
	return &CORSRule{
		Name:             cfg.name,
		Enabled:          true,
		Expression:       cfg.expression,
		AllowOrigins:     cfg.allowOrigins,
		AllowMethods:     cfg.allowMethods,
		AllowHeaders:     cfg.allowHeaders,
		MaxAge:           cfg.maxAge,
		AllowCredentials: cfg.allowCredentials,
	}
}

// splitTrimmed splits a comma-separated string and trims spaces from each element.
func splitTrimmed(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, v)
		}
	}
	return result
}

// isNotFoundError checks whether an error indicates a 404 / not-found response.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "404")
}
