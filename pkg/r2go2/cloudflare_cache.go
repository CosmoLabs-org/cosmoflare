package r2go2

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// CachePurgeResult represents the result of a cache purge operation.
type CachePurgeResult struct {
	ID string `json:"id"`
}

// CacheSettings holds cache-related zone settings.
type CacheSettings struct {
	BrowserCacheTTL int    `json:"browser_cache_ttl"`
	DevelopmentMode int    `json:"development_mode"`
	CacheLevel      string `json:"cache_level"`
	MinifyCss       bool   `json:"minify_css"`
	MinifyJs        bool   `json:"minify_js"`
	MinifyHtml      bool   `json:"minify_html"`
}

// CacheOption is a functional option for cache settings updates.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	browserTTL *int
	devMode    *bool
	cacheLevel *string
}

// WithBrowserCacheTTL sets the browser cache TTL in seconds.
func WithBrowserCacheTTL(ttl int) CacheOption {
	return func(c *cacheConfig) { c.browserTTL = &ttl }
}

// WithDevMode enables or disables development mode.
func WithDevMode(enabled bool) CacheOption {
	return func(c *cacheConfig) { c.devMode = &enabled }
}

// WithCacheLevel sets the cache level ("aggressive", "basic", "simplified").
func WithCacheLevel(level string) CacheOption {
	return func(c *cacheConfig) { c.cacheLevel = &level }
}

// CacheService implements Cloudflare Cache operations.
// Cache is zone-scoped — it uses a zone ID, not an account ID.
type CacheService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewCacheService creates a new Cache service client.
func NewCacheService(api *cloudflare.API, zoneID string) (*CacheService, error) {
	if api == nil {
		return nil, validationError("NewCacheService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewCacheService", "zone ID is required")
	}
	return &CacheService{cf: api, zoneID: zoneID}, nil
}

// NewCacheServiceFromCreds creates a CacheService from zone ID and API token.
// Convenience helper for CLI usage.
func NewCacheServiceFromCreds(zoneID, apiToken string) (*CacheService, error) {
	if zoneID == "" {
		return nil, validationError("NewCacheService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewCacheService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewCacheService", "failed to create Cloudflare API client", err)
	}
	return &CacheService{cf: cf, zoneID: zoneID}, nil
}

// PurgeAll purges the entire cache for the zone.
//
// WARNING: This will substantially increase load on the origin server
// if there is a high cached vs. uncached request ratio.
func (s *CacheService) PurgeAll(ctx context.Context) (*CachePurgeResult, error) {
	resp, err := s.cf.PurgeEverything(ctx, s.zoneID)
	if err != nil {
		return nil, newError("CacheService.PurgeAll", "failed to purge all cache", err)
	}
	if !resp.Success {
		return nil, newError("CacheService.PurgeAll", "purge request was not successful", nil)
	}
	return &CachePurgeResult{ID: resp.Result.ID}, nil
}

// PurgeByURLs purges cached content for the specified URLs.
// Limit of 30 URLs per request.
func (s *CacheService) PurgeByURLs(ctx context.Context, urls []string) (*CachePurgeResult, error) {
	if len(urls) == 0 {
		return nil, validationError("CacheService.PurgeByURLs", "at least one URL is required")
	}
	resp, err := s.cf.PurgeCache(ctx, s.zoneID, cloudflare.PurgeCacheRequest{Files: urls})
	if err != nil {
		return nil, newError("CacheService.PurgeByURLs", "failed to purge cache by URLs", err)
	}
	return &CachePurgeResult{ID: resp.Result.ID}, nil
}

// PurgeByTags purges cached content matching the specified cache tags (Enterprise only).
func (s *CacheService) PurgeByTags(ctx context.Context, tags []string) (*CachePurgeResult, error) {
	if len(tags) == 0 {
		return nil, validationError("CacheService.PurgeByTags", "at least one tag is required")
	}
	resp, err := s.cf.PurgeCache(ctx, s.zoneID, cloudflare.PurgeCacheRequest{Tags: tags})
	if err != nil {
		return nil, newError("CacheService.PurgeByTags", "failed to purge cache by tags", err)
	}
	return &CachePurgeResult{ID: resp.Result.ID}, nil
}

// PurgeByHosts purges cached content for the specified hostnames.
func (s *CacheService) PurgeByHosts(ctx context.Context, hosts []string) (*CachePurgeResult, error) {
	if len(hosts) == 0 {
		return nil, validationError("CacheService.PurgeByHosts", "at least one host is required")
	}
	resp, err := s.cf.PurgeCache(ctx, s.zoneID, cloudflare.PurgeCacheRequest{Hosts: hosts})
	if err != nil {
		return nil, newError("CacheService.PurgeByHosts", "failed to purge cache by hosts", err)
	}
	return &CachePurgeResult{ID: resp.Result.ID}, nil
}

// GetSettings retrieves cache-related settings for the zone.
func (s *CacheService) GetSettings(ctx context.Context) (*CacheSettings, error) {
	resp, err := s.cf.ZoneSettings(ctx, s.zoneID)
	if err != nil {
		return nil, newError("CacheService.GetSettings", "failed to get zone settings", err)
	}

	settings := &CacheSettings{}
	for _, setting := range resp.Result {
		switch setting.ID {
		case "browser_cache_ttl":
			if v, ok := setting.Value.(float64); ok {
				settings.BrowserCacheTTL = int(v)
			}
		case "development_mode":
			if v, ok := setting.Value.(float64); ok {
				settings.DevelopmentMode = int(v)
			}
		case "cache_level":
			if v, ok := setting.Value.(string); ok {
				settings.CacheLevel = v
			}
		case "minify":
			if m, ok := setting.Value.(map[string]interface{}); ok {
				if css, ok := m["css"].(string); ok {
					settings.MinifyCss = css == "on"
				}
				if js, ok := m["js"].(string); ok {
					settings.MinifyJs = js == "on"
				}
				if html, ok := m["html"].(string); ok {
					settings.MinifyHtml = html == "on"
				}
			}
		}
	}
	return settings, nil
}

// UpdateSettings updates cache-related settings for the zone.
func (s *CacheService) UpdateSettings(ctx context.Context, opts ...CacheOption) error {
	if len(opts) == 0 {
		return validationError("CacheService.UpdateSettings", "at least one setting option is required")
	}

	cfg := &cacheConfig{}
	for _, o := range opts {
		o(cfg)
	}

	var settings []cloudflare.ZoneSetting

	if cfg.browserTTL != nil {
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "browser_cache_ttl",
			Value: *cfg.browserTTL,
		})
	}
	if cfg.devMode != nil {
		val := 0
		if *cfg.devMode {
			val = 1
		}
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "development_mode",
			Value: val,
		})
	}
	if cfg.cacheLevel != nil {
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "cache_level",
			Value: *cfg.cacheLevel,
		})
	}

	if len(settings) == 0 {
		return validationError("CacheService.UpdateSettings", "no settings to update")
	}

	_, err := s.cf.UpdateZoneSettings(ctx, s.zoneID, settings)
	if err != nil {
		return newError("CacheService.UpdateSettings", fmt.Sprintf("failed to update cache settings for zone %s", s.zoneID), err)
	}
	return nil
}
