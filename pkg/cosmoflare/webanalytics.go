package cosmoflare

import (
	"context"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// ---------------------------------------------------------------------------
// FEAT-037 wave 3 — Web Analytics (Cloudflare RUM).
//
// Account-scoped site management for Cloudflare Web Analytics (the RUM
// beacon): list, create and delete sites via the
// /accounts/{account_id}/rum/site_info endpoints. cloudflare-go v0.116.0
// ships typed RUM resources (web_analytics.go), so no raw seam is needed.
//
// Site credentials: the beacon site_token and the JS snippet are returned
// by the CREATE call only. The token is the credential the beacon uses to
// attribute traffic — it must never be repeated in list output, logs or
// any other read path. WebAnalyticsSiteInfo therefore omits it by design.
//
// Web Analytics permission pending FEAT-011 dataset.
// ---------------------------------------------------------------------------

// WebAnalyticsSiteInfo is the public read model for a Web Analytics site.
// The site_token is deliberately absent: the token is a beacon credential
// surfaced exactly once by Create and never again (see WebAnalyticsSiteCreated).
type WebAnalyticsSiteInfo struct {
	SiteTag        string     `json:"site_tag"`
	ZoneTag        string     `json:"zone_tag,omitempty"`
	ZoneName       string     `json:"zone_name,omitempty"`
	RulesetID      string     `json:"ruleset_id,omitempty"`
	RulesetEnabled bool       `json:"ruleset_enabled"`
	AutoInstall    bool       `json:"auto_install"`
	Created        *time.Time `json:"created,omitempty"`
}

// WebAnalyticsSiteCreated is the one-time result of creating a site. It
// carries the beacon token (site_token) and the JS snippet that must be
// pasted into the site's <head>. Neither is retrievable later — callers
// must treat this response as the only chance to persist the token.
type WebAnalyticsSiteCreated struct {
	SiteTag     string `json:"site_tag"`
	SiteToken   string `json:"site_token"`
	Snippet     string `json:"snippet"`
	AutoInstall bool   `json:"auto_install"`
}

// WebAnalyticsService implements account-scoped Web Analytics (RUM) site
// operations.
type WebAnalyticsService struct {
	cf        *cloudflare.API
	accountID string
}

// NewWebAnalyticsService creates a new Web Analytics service client.
func NewWebAnalyticsService(api *cloudflare.API, accountID string) (*WebAnalyticsService, error) {
	if api == nil {
		return nil, validationError("NewWebAnalyticsService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewWebAnalyticsService", "account ID is required")
	}
	return &WebAnalyticsService{cf: api, accountID: accountID}, nil
}

// NewWebAnalyticsServiceFromCreds creates a WebAnalyticsService from an
// account ID and API token. Convenience helper for CLI usage.
func NewWebAnalyticsServiceFromCreds(accountID, apiToken string) (*WebAnalyticsService, error) {
	if accountID == "" {
		return nil, validationError("NewWebAnalyticsServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewWebAnalyticsServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewWebAnalyticsServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &WebAnalyticsService{cf: cf, accountID: accountID}, nil
}

// validate checks the invariants shared by every operation.
func (s *WebAnalyticsService) validate() error {
	if s == nil || s.cf == nil {
		return validationError("WebAnalyticsService", "cloudflare API client is required")
	}
	if s.accountID == "" {
		return validationError("WebAnalyticsService", "account ID is required")
	}
	return nil
}

// webAnalyticsSiteInfoFromCF maps the SDK read model to the public read
// struct, normalising zero timestamps to nil. The SDK's SiteToken and
// Snippet fields are intentionally dropped.
func webAnalyticsSiteInfoFromCF(site cloudflare.WebAnalyticsSite) WebAnalyticsSiteInfo {
	return WebAnalyticsSiteInfo{
		SiteTag:        site.SiteTag,
		ZoneTag:        site.Ruleset.ZoneTag,
		ZoneName:       site.Ruleset.ZoneName,
		RulesetID:      site.Ruleset.ID,
		RulesetEnabled: site.Ruleset.Enabled,
		AutoInstall:    site.AutoInstall,
		Created:        site.Created,
	}
}

// List returns every Web Analytics site in the account. The beacon token
// is never part of the result — see WebAnalyticsSiteInfo.
func (s *WebAnalyticsService) List(ctx context.Context) ([]WebAnalyticsSiteInfo, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	sites, _, err := s.cf.ListWebAnalyticsSites(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListWebAnalyticsSitesParams{})
	if err != nil {
		return nil, newError("WebAnalyticsService.List", "list web analytics sites", err)
	}
	out := make([]WebAnalyticsSiteInfo, 0, len(sites))
	for _, site := range sites {
		out = append(out, webAnalyticsSiteInfoFromCF(site))
	}
	return out, nil
}

// Create registers a new Web Analytics site. Exactly one of host or
// zoneTag must be set: a host measures a non-Cloudflare-proxied origin
// via the JS beacon, a zoneTag ties the site to an orange-clouded zone
// (where --auto-install lets Cloudflare inject the snippet for you).
//
// The returned token and snippet are shown ONCE — persist them before the
// process exits; they cannot be fetched again.
func (s *WebAnalyticsService) Create(ctx context.Context, host, zoneTag string, autoInstall *bool) (*WebAnalyticsSiteCreated, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if host == "" && zoneTag == "" {
		return nil, validationError("WebAnalyticsService.Create", "exactly one of host or zone tag is required")
	}
	if host != "" && zoneTag != "" {
		return nil, validationError("WebAnalyticsService.Create", "host and zone tag are mutually exclusive — pass exactly one")
	}
	site, err := s.cf.CreateWebAnalyticsSite(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.CreateWebAnalyticsSiteParams{
		Host:        host,
		ZoneTag:     zoneTag,
		AutoInstall: autoInstall,
	})
	if err != nil {
		target := zoneTag
		if host != "" {
			target = host
		}
		return nil, newError("WebAnalyticsService.Create", "create web analytics site for "+target, err)
	}
	return &WebAnalyticsSiteCreated{
		SiteTag:     site.SiteTag,
		SiteToken:   site.SiteToken,
		Snippet:     site.Snippet,
		AutoInstall: site.AutoInstall,
	}, nil
}

// Delete removes a Web Analytics site by its site tag. Deleting a site
// stops data collection for it; the beacon token is dead afterwards and
// historical data is no longer reachable through it.
func (s *WebAnalyticsService) Delete(ctx context.Context, siteTag string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if siteTag == "" {
		return validationError("WebAnalyticsService.Delete", "site tag is required")
	}
	if _, err := s.cf.DeleteWebAnalyticsSite(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.DeleteWebAnalyticsSiteParams{SiteTag: siteTag}); err != nil {
		return newError("WebAnalyticsService.Delete", "delete web analytics site "+siteTag, err)
	}
	return nil
}
