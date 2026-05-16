package r2go2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func cacheMockSetup(handler http.HandlerFunc) (*CacheService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewCacheService(cf, "zone-cache-123")
	return svc, server
}

func cacheWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewCacheServiceValidation(t *testing.T) {
	_, err := NewCacheService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewCacheService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewCacheService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewCacheServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewCacheService(cf, "zone123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewCacheServiceFromCredsValidation(t *testing.T) {
	_, err := NewCacheServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewCacheServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewCacheServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both zoneID and apiToken are empty")
	}
}

func TestNewCacheServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewCacheServiceFromCreds("zone123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestCacheOptions(t *testing.T) {
	cfg := &cacheConfig{}

	WithBrowserCacheTTL(3600)(cfg)
	if cfg.browserTTL == nil {
		t.Fatal("expected browserTTL to be set")
	}
	if *cfg.browserTTL != 3600 {
		t.Errorf("expected browserTTL=3600, got %d", *cfg.browserTTL)
	}

	WithDevMode(true)(cfg)
	if cfg.devMode == nil {
		t.Fatal("expected devMode to be set")
	}
	if !*cfg.devMode {
		t.Error("expected devMode=true")
	}

	WithCacheLevel("aggressive")(cfg)
	if cfg.cacheLevel == nil {
		t.Fatal("expected cacheLevel to be set")
	}
	if *cfg.cacheLevel != "aggressive" {
		t.Errorf("expected cacheLevel=aggressive, got %s", *cfg.cacheLevel)
	}
}

func TestCachePurgeByURLsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCacheService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.PurgeByURLs(ctx, nil)
	if err == nil {
		t.Error("expected error when urls is nil")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}

	_, err = svc.PurgeByURLs(ctx, []string{})
	if err == nil {
		t.Error("expected error when urls is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestCachePurgeByTagsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCacheService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.PurgeByTags(ctx, nil)
	if err == nil {
		t.Error("expected error when tags is nil")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}

	_, err = svc.PurgeByTags(ctx, []string{})
	if err == nil {
		t.Error("expected error when tags is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestCachePurgeByHostsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCacheService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.PurgeByHosts(ctx, nil)
	if err == nil {
		t.Error("expected error when hosts is nil")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}

	_, err = svc.PurgeByHosts(ctx, []string{})
	if err == nil {
		t.Error("expected error when hosts is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestCacheUpdateSettingsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCacheService(cf, "zone123")
	ctx := context.Background()

	err := svc.UpdateSettings(ctx)
	if err == nil {
		t.Error("expected error when no options provided")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestCacheConstructorErrorTypes(t *testing.T) {
	_, err := NewCacheService(nil, "zone123")
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError for nil API, got %T", err)
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewCacheService(cf, "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError for empty zoneID, got %T", err)
	}

	_, err = NewCacheServiceFromCreds("", "token")
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError for empty zoneID in FromCreds, got %T", err)
	}

	_, err = NewCacheServiceFromCreds("zone123", "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError for empty apiToken in FromCreds, got %T", err)
	}
}

func TestCacheTypes(t *testing.T) {
	result := &CachePurgeResult{ID: "purge-123"}
	if result.ID != "purge-123" {
		t.Errorf("expected ID=purge-123, got %s", result.ID)
	}

	settings := &CacheSettings{
		BrowserCacheTTL: 7200,
		DevelopmentMode: 1,
		CacheLevel:      "aggressive",
		MinifyCss:       true,
		MinifyJs:        false,
		MinifyHtml:      true,
	}
	if settings.BrowserCacheTTL != 7200 {
		t.Errorf("expected BrowserCacheTTL=7200, got %d", settings.BrowserCacheTTL)
	}
	if settings.DevelopmentMode != 1 {
		t.Errorf("expected DevelopmentMode=1, got %d", settings.DevelopmentMode)
	}
	if settings.CacheLevel != "aggressive" {
		t.Errorf("expected CacheLevel=aggressive, got %s", settings.CacheLevel)
	}
	if !settings.MinifyCss {
		t.Error("expected MinifyCss=true")
	}
	if settings.MinifyJs {
		t.Error("expected MinifyJs=false")
	}
	if !settings.MinifyHtml {
		t.Error("expected MinifyHtml=true")
	}
}

// --- httptest-based API mock tests ---

func TestCachePurgeAllWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "purge-all-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	result, err := svc.PurgeAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "purge-all-001" {
		t.Errorf("expected ID=purge-all-001, got %s", result.ID)
	}
}

func TestCachePurgeAllNotSuccessful(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "purge failed"}},
			"result":  map[string]interface{}{"id": ""},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.PurgeAll(ctx)
	if err == nil {
		t.Fatal("expected error when purge not successful")
	}
}

func TestCachePurgeByURLsWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "purge-url-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	result, err := svc.PurgeByURLs(ctx, []string{"https://example.com/page1", "https://example.com/page2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "purge-url-001" {
		t.Errorf("expected ID=purge-url-001, got %s", result.ID)
	}
}

func TestCachePurgeByTagsWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "purge-tag-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	result, err := svc.PurgeByTags(ctx, []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "purge-tag-001" {
		t.Errorf("expected ID=purge-tag-001, got %s", result.ID)
	}
}

func TestCachePurgeByHostsWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "purge-host-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	result, err := svc.PurgeByHosts(ctx, []string{"cdn.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "purge-host-001" {
		t.Errorf("expected ID=purge-host-001, got %s", result.ID)
	}
}

func TestCacheGetSettingsWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "browser_cache_ttl", "value": 14400, "editable": true},
				{"id": "development_mode", "value": 0, "editable": true},
				{"id": "cache_level", "value": "aggressive", "editable": true},
				{"id": "minify", "value": map[string]interface{}{"css": "on", "js": "on", "html": "off"}, "editable": true},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.BrowserCacheTTL != 14400 {
		t.Errorf("expected BrowserCacheTTL=14400, got %d", settings.BrowserCacheTTL)
	}
	if settings.CacheLevel != "aggressive" {
		t.Errorf("expected CacheLevel=aggressive, got %s", settings.CacheLevel)
	}
	if !settings.MinifyCss {
		t.Error("expected MinifyCss=true")
	}
	if !settings.MinifyJs {
		t.Error("expected MinifyJs=true")
	}
	if settings.MinifyHtml {
		t.Error("expected MinifyHtml=false")
	}
}

func TestCacheUpdateSettingsWithMock(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.UpdateSettings(ctx,
		WithBrowserCacheTTL(7200),
		WithDevMode(true),
		WithCacheLevel("basic"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCacheUpdateSettingsDevModeOff(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cacheWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.UpdateSettings(ctx, WithDevMode(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCachePurgeAPIError(t *testing.T) {
	svc, server := cacheMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		cacheWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "server error"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.PurgeAll(ctx)
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
}

func TestCachePurgeResultJSONMarshal(t *testing.T) {
	r := &CachePurgeResult{ID: "purge-json-001"}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded CachePurgeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "purge-json-001" {
		t.Errorf("ID mismatch: got %q", decoded.ID)
	}
}

func TestCacheSettingsJSONMarshal(t *testing.T) {
	s := &CacheSettings{
		BrowserCacheTTL: 7200, DevelopmentMode: 0,
		CacheLevel: "aggressive", MinifyCss: true, MinifyJs: false, MinifyHtml: true,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded CacheSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.BrowserCacheTTL != 7200 {
		t.Errorf("BrowserCacheTTL mismatch: got %d", decoded.BrowserCacheTTL)
	}
}

func TestCacheNewFromCredsSuccess(t *testing.T) {
	svc, err := NewCacheServiceFromCreds("zone123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
}
