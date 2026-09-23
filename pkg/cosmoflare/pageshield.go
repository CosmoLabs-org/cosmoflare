package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// PageShieldService implements Cloudflare Page Shield operations for a
// single zone.
//
// Page Shield detects malicious scripts and dynamically injected
// connections on proxied hostnames. It is zone-scoped: every call targets
// the zone passed to the constructor (the API path stays page_shield even
// though the CLI command group is named "page-shield").
//
// Required API token permissions (zone level): the "Client-side security"
// permission group — the token-UI group was renamed from "Page Shield" to
// "Client-side security" (verified against the Qwen dataset). Reads need
// Client-side security: Read; policy writes need Client-side security:
// Edit.
type PageShieldService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewPageShieldService creates a Page Shield service client for a zone.
func NewPageShieldService(api *cloudflare.API, zoneID string) (*PageShieldService, error) {
	if api == nil {
		return nil, validationError("NewPageShieldService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewPageShieldService", "zone ID is required")
	}
	return &PageShieldService{cf: api, zoneID: zoneID}, nil
}

// NewPageShieldServiceFromCreds creates a PageShieldService from a zone ID
// and API token.
func NewPageShieldServiceFromCreds(zoneID, apiToken string) (*PageShieldService, error) {
	if zoneID == "" {
		return nil, validationError("NewPageShieldServiceFromCreds", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewPageShieldServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewPageShieldServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &PageShieldService{cf: cf, zoneID: zoneID}, nil
}

// PageShieldConnection is a dynamically injected connection Page Shield
// detected on the zone (fetch/XHR endpoints, not script tags).
type PageShieldConnection struct {
	ID                      string   `json:"id"`
	URL                     string   `json:"url"`
	Host                    string   `json:"host"`
	AddedAt                 string   `json:"added_at,omitempty"`
	FirstSeenAt             string   `json:"first_seen_at,omitempty"`
	LastSeenAt              string   `json:"last_seen_at,omitempty"`
	FirstPageURL            string   `json:"first_page_url,omitempty"`
	PageURLs                []string `json:"page_urls,omitempty"`
	DomainReportedMalicious *bool    `json:"domain_reported_malicious,omitempty"`
	URLContainsCdnCgiPath   *bool    `json:"url_contains_cdn_cgi_path,omitempty"`
}

// PageShieldScript is a script Page Shield detected on the zone.
type PageShieldScript struct {
	ID                      string   `json:"id"`
	URL                     string   `json:"url"`
	Host                    string   `json:"host"`
	Hash                    string   `json:"hash,omitempty"`
	JSIntegrityScore        int      `json:"js_integrity_score,omitempty"`
	AddedAt                 string   `json:"added_at,omitempty"`
	FetchedAt               string   `json:"fetched_at,omitempty"`
	FirstSeenAt             string   `json:"first_seen_at,omitempty"`
	LastSeenAt              string   `json:"last_seen_at,omitempty"`
	FirstPageURL            string   `json:"first_page_url,omitempty"`
	PageURLs                []string `json:"page_urls,omitempty"`
	DomainReportedMalicious *bool    `json:"domain_reported_malicious,omitempty"`
	URLContainsCdnCgiPath   *bool    `json:"url_contains_cdn_cgi_path,omitempty"`
}

// PageShieldPolicy is a Page Shield policy that allows, logs, or blocks
// detected connections and scripts matching an expression.
type PageShieldPolicy struct {
	ID          string `json:"id"`
	Expression  string `json:"expression"`
	Action      string `json:"action"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// PageShieldPolicyOption is a functional option for policy create/update.
type PageShieldPolicyOption func(*pageShieldPolicyConfig)

type pageShieldPolicyConfig struct {
	expression  string
	action      string
	value       string
	description string
	enabled     *bool
}

// WithPageShieldPolicyExpression sets the policy expression (cloudflare
// filter expression selecting matching connections/scripts).
func WithPageShieldPolicyExpression(expression string) PageShieldPolicyOption {
	return func(c *pageShieldPolicyConfig) { c.expression = expression }
}

// WithPageShieldPolicyAction sets the policy action: allow, log, or block.
func WithPageShieldPolicyAction(action string) PageShieldPolicyOption {
	return func(c *pageShieldPolicyConfig) { c.action = action }
}

// WithPageShieldPolicyValue sets the policy value (action parameter, e.g.
// the warning shown by a block).
func WithPageShieldPolicyValue(value string) PageShieldPolicyOption {
	return func(c *pageShieldPolicyConfig) { c.value = value }
}

// WithPageShieldPolicyDescription sets the human-readable policy
// description.
func WithPageShieldPolicyDescription(description string) PageShieldPolicyOption {
	return func(c *pageShieldPolicyConfig) { c.description = description }
}

// WithPageShieldPolicyEnabled sets whether the policy is enabled.
func WithPageShieldPolicyEnabled(enabled bool) PageShieldPolicyOption {
	return func(c *pageShieldPolicyConfig) { c.enabled = boolPtr(enabled) }
}

func (c *pageShieldPolicyConfig) apply(opts []PageShieldPolicyOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// ListConnections lists the connections Page Shield detected in the zone.
func (s *PageShieldService) ListConnections(ctx context.Context) ([]*PageShieldConnection, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	results, _, err := s.cf.ListPageShieldConnections(ctx, rc, cloudflare.ListPageShieldConnectionsParams{})
	if err != nil {
		return nil, newError("PageShieldService.ListConnections", fmt.Sprintf("failed to list page shield connections in zone %q", s.zoneID), err)
	}

	connections := make([]*PageShieldConnection, 0, len(results))
	for _, conn := range results {
		connections = append(connections, mapPageShieldConnection(conn))
	}
	return connections, nil
}

// ListScripts lists the scripts Page Shield detected in the zone.
func (s *PageShieldService) ListScripts(ctx context.Context) ([]*PageShieldScript, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	results, _, err := s.cf.ListPageShieldScripts(ctx, rc, cloudflare.ListPageShieldScriptsParams{})
	if err != nil {
		return nil, newError("PageShieldService.ListScripts", fmt.Sprintf("failed to list page shield scripts in zone %q", s.zoneID), err)
	}

	scripts := make([]*PageShieldScript, 0, len(results))
	for _, script := range results {
		scripts = append(scripts, mapPageShieldScript(script))
	}
	return scripts, nil
}

// ListPolicies lists the Page Shield policies configured in the zone.
func (s *PageShieldService) ListPolicies(ctx context.Context) ([]*PageShieldPolicy, error) {
	rc := cloudflare.ZoneIdentifier(s.zoneID)
	results, _, err := s.cf.ListPageShieldPolicies(ctx, rc, cloudflare.ListPageShieldPoliciesParams{})
	if err != nil {
		return nil, newError("PageShieldService.ListPolicies", fmt.Sprintf("failed to list page shield policies in zone %q", s.zoneID), err)
	}

	policies := make([]*PageShieldPolicy, 0, len(results))
	for _, policy := range results {
		policies = append(policies, mapPageShieldPolicy(policy))
	}
	return policies, nil
}

// GetPolicy retrieves a single Page Shield policy by its ID.
func (s *PageShieldService) GetPolicy(ctx context.Context, policyID string) (*PageShieldPolicy, error) {
	if policyID == "" {
		return nil, validationError("PageShieldService.GetPolicy", "policy ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	policy, err := s.cf.GetPageShieldPolicy(ctx, rc, policyID)
	if err != nil {
		return nil, newError("PageShieldService.GetPolicy", fmt.Sprintf("failed to get page shield policy %q", policyID), err)
	}
	return mapPageShieldPolicy(*policy), nil
}

// CreatePolicy creates a Page Shield policy. The expression and action are
// required; everything else is optional.
func (s *PageShieldService) CreatePolicy(ctx context.Context, opts ...PageShieldPolicyOption) (*PageShieldPolicy, error) {
	var cfg pageShieldPolicyConfig
	cfg.apply(opts)
	if cfg.expression == "" {
		return nil, validationError("PageShieldService.CreatePolicy", "policy expression is required")
	}
	if cfg.action == "" {
		return nil, validationError("PageShieldService.CreatePolicy", "policy action is required (allow, log, or block)")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	policy, err := s.cf.CreatePageShieldPolicy(ctx, rc, cloudflare.CreatePageShieldPolicyParams{
		Expression:  cfg.expression,
		Action:      cfg.action,
		Value:       cfg.value,
		Description: cfg.description,
		Enabled:     cfg.enabled,
	})
	if err != nil {
		return nil, newError("PageShieldService.CreatePolicy", "failed to create page shield policy", err)
	}
	return mapPageShieldPolicy(*policy), nil
}

// UpdatePolicy updates a Page Shield policy. Only the fields covered by the
// given options change; unspecified fields keep their current values
// (fetch, merge, then write).
func (s *PageShieldService) UpdatePolicy(ctx context.Context, policyID string, opts ...PageShieldPolicyOption) (*PageShieldPolicy, error) {
	if policyID == "" {
		return nil, validationError("PageShieldService.UpdatePolicy", "policy ID is required")
	}

	var cfg pageShieldPolicyConfig
	cfg.apply(opts)
	if cfg.isEmpty() {
		return nil, validationError("PageShieldService.UpdatePolicy", "at least one update option is required (expression, action, value, description, enabled)")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	current, err := s.cf.GetPageShieldPolicy(ctx, rc, policyID)
	if err != nil {
		return nil, newError("PageShieldService.UpdatePolicy", fmt.Sprintf("failed to fetch page shield policy %q for update", policyID), err)
	}

	params := cloudflare.UpdatePageShieldPolicyParams{
		ID:          current.ID,
		Expression:  current.Expression,
		Action:      current.Action,
		Value:       current.Value,
		Description: current.Description,
		Enabled:     current.Enabled,
	}
	if cfg.expression != "" {
		params.Expression = cfg.expression
	}
	if cfg.action != "" {
		params.Action = cfg.action
	}
	if cfg.value != "" {
		params.Value = cfg.value
	}
	if cfg.description != "" {
		params.Description = cfg.description
	}
	if cfg.enabled != nil {
		params.Enabled = cfg.enabled
	}

	updated, err := s.cf.UpdatePageShieldPolicy(ctx, rc, params)
	if err != nil {
		return nil, newError("PageShieldService.UpdatePolicy", fmt.Sprintf("failed to update page shield policy %q", policyID), err)
	}
	return mapPageShieldPolicy(*updated), nil
}

// DeletePolicy deletes a Page Shield policy. This action is irreversible.
func (s *PageShieldService) DeletePolicy(ctx context.Context, policyID string) error {
	if policyID == "" {
		return validationError("PageShieldService.DeletePolicy", "policy ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	if err := s.cf.DeletePageShieldPolicy(ctx, rc, policyID); err != nil {
		return newError("PageShieldService.DeletePolicy", fmt.Sprintf("failed to delete page shield policy %q", policyID), err)
	}
	return nil
}

func (c *pageShieldPolicyConfig) isEmpty() bool {
	return c.expression == "" && c.action == "" && c.value == "" && c.description == "" && c.enabled == nil
}

func mapPageShieldConnection(conn cloudflare.PageShieldConnection) *PageShieldConnection {
	return &PageShieldConnection{
		ID:                      conn.ID,
		URL:                     conn.URL,
		Host:                    conn.Host,
		AddedAt:                 conn.AddedAt,
		FirstSeenAt:             conn.FirstSeenAt,
		LastSeenAt:              conn.LastSeenAt,
		FirstPageURL:            conn.FirstPageURL,
		PageURLs:                conn.PageURLs,
		DomainReportedMalicious: conn.DomainReportedMalicious,
		URLContainsCdnCgiPath:   conn.URLContainsCdnCgiPath,
	}
}

func mapPageShieldScript(script cloudflare.PageShieldScript) *PageShieldScript {
	return &PageShieldScript{
		ID:                      script.ID,
		URL:                     script.URL,
		Host:                    script.Host,
		Hash:                    script.Hash,
		JSIntegrityScore:        script.JSIntegrityScore,
		AddedAt:                 script.AddedAt,
		FetchedAt:               script.FetchedAt,
		FirstSeenAt:             script.FirstSeenAt,
		LastSeenAt:              script.LastSeenAt,
		FirstPageURL:            script.FirstPageURL,
		PageURLs:                script.PageURLs,
		DomainReportedMalicious: script.DomainReportedMalicious,
		URLContainsCdnCgiPath:   script.URLContainsCdnCgiPath,
	}
}

func mapPageShieldPolicy(policy cloudflare.PageShieldPolicy) *PageShieldPolicy {
	return &PageShieldPolicy{
		ID:          policy.ID,
		Expression:  policy.Expression,
		Action:      policy.Action,
		Value:       policy.Value,
		Description: policy.Description,
		Enabled:     policy.Enabled,
	}
}
