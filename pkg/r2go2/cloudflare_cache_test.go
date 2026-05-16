package r2go2

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

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
