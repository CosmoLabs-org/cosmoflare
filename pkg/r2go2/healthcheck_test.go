package r2go2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// --- Constructor validation ---

func TestNewHealthcheckServiceValidation(t *testing.T) {
	_, err := NewHealthcheckService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewHealthcheckService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewHealthcheckService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewHealthcheckServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewHealthcheckService(cf, "zone123")
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

func TestNewHealthcheckServiceFromCredsValidation(t *testing.T) {
	_, err := NewHealthcheckServiceFromCreds("", "test-token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewHealthcheckServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewHealthcheckServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both zoneID and apiToken are empty")
	}
}

func TestNewHealthcheckServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewHealthcheckServiceFromCreds("zone123", "test-token")
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

// --- Error type assertions ---

func TestHealthcheckConstructorValidationErrorType(t *testing.T) {
	_, err := NewHealthcheckService(nil, "zone123")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewHealthcheckService(cf, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestHealthcheckFromCredsValidationErrorType(t *testing.T) {
	_, err := NewHealthcheckServiceFromCreds("", "test-token")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty zoneID, got %T", err)
	}

	_, err = NewHealthcheckServiceFromCreds("zone123", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty apiToken, got %T", err)
	}
}

// --- Functional options ---

func TestHealthcheckOptions(t *testing.T) {
	cfg := &healthcheckConfig{}

	WithHealthcheckInterval(60)(cfg)
	if cfg.interval == nil || *cfg.interval != 60 {
		t.Errorf("expected interval=60, got %v", cfg.interval)
	}

	WithHealthcheckInterval(30)(cfg)
	if cfg.interval == nil || *cfg.interval != 30 {
		t.Errorf("expected interval=30, got %v", cfg.interval)
	}

	WithHealthcheckTimeout(10)(cfg)
	if cfg.timeout == nil || *cfg.timeout != 10 {
		t.Errorf("expected timeout=10, got %v", cfg.timeout)
	}

	WithHealthcheckRetries(3)(cfg)
	if cfg.retries == nil || *cfg.retries != 3 {
		t.Errorf("expected retries=3, got %v", cfg.retries)
	}

	WithHealthcheckSuspended(true)(cfg)
	if cfg.suspended == nil || !*cfg.suspended {
		t.Error("expected suspended=true")
	}

	WithHealthcheckSuspended(false)(cfg)
	if cfg.suspended == nil || *cfg.suspended {
		t.Error("expected suspended=false")
	}

	WithHealthcheckDescription("my health check")(cfg)
	if cfg.description == nil || *cfg.description != "my health check" {
		t.Errorf("expected description='my health check', got %v", cfg.description)
	}

	WithHealthcheckDescription("")(cfg)
	if cfg.description == nil || *cfg.description != "" {
		t.Error("expected description to be set to empty string")
	}

	WithHealthcheckType("TCP")(cfg)
	if cfg.hcType == nil || *cfg.hcType != "TCP" {
		t.Errorf("expected hcType=TCP, got %v", cfg.hcType)
	}

	WithHealthcheckType("HTTP")(cfg)
	if cfg.hcType == nil || *cfg.hcType != "HTTP" {
		t.Errorf("expected hcType=HTTP, got %v", cfg.hcType)
	}
}

// --- Method input validation ---

func TestHealthcheckGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewHealthcheckService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Get(ctx, "")
	if err == nil {
		t.Error("expected error when healthcheck ID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestHealthcheckCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewHealthcheckService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "example.com")
	if err == nil {
		t.Error("expected error when name is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty name, got %T", err)
	}

	_, err = svc.Create(ctx, "my-check", "")
	if err == nil {
		t.Error("expected error when address is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty address, got %T", err)
	}
}

func TestHealthcheckUpdateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewHealthcheckService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Update(ctx, "")
	if err == nil {
		t.Error("expected error when healthcheck ID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestHealthcheckDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewHealthcheckService(cf, "zone123")
	ctx := context.Background()

	err := svc.Delete(ctx, "")
	if err == nil {
		t.Error("expected error when healthcheck ID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

// --- Converter tests ---

func TestCfHealthcheckToHealthcheck(t *testing.T) {
	now := time.Now()

	cfHC := cloudflare.Healthcheck{
		ID:                   "hc-abc123",
		Name:                 "web-check",
		Address:              "example.com",
		Type:                 "HTTPS",
		Status:               "healthy",
		Suspended:            false,
		Interval:             60,
		Timeout:              5,
		Retries:              2,
		Description:          "Main site check",
		ConsecutiveSuccesses: 3,
		ConsecutiveFails:     1,
		FailureReason:        "",
		CreatedOn:            &now,
		ModifiedOn:           &now,
	}

	hc := cfHealthcheckToHealthcheck(cfHC)

	if hc.ID != "hc-abc123" {
		t.Errorf("expected ID=hc-abc123, got %s", hc.ID)
	}
	if hc.Name != "web-check" {
		t.Errorf("expected Name=web-check, got %s", hc.Name)
	}
	if hc.Address != "example.com" {
		t.Errorf("expected Address=example.com, got %s", hc.Address)
	}
	if hc.Type != "HTTPS" {
		t.Errorf("expected Type=HTTPS, got %s", hc.Type)
	}
	if hc.Status != "healthy" {
		t.Errorf("expected Status=healthy, got %s", hc.Status)
	}
	if hc.Suspended != false {
		t.Error("expected Suspended=false")
	}
	if hc.Interval != 60 {
		t.Errorf("expected Interval=60, got %d", hc.Interval)
	}
	if hc.Timeout != 5 {
		t.Errorf("expected Timeout=5, got %d", hc.Timeout)
	}
	if hc.Retries != 2 {
		t.Errorf("expected Retries=2, got %d", hc.Retries)
	}
	if hc.Description != "Main site check" {
		t.Errorf("expected Description='Main site check', got %s", hc.Description)
	}
	if hc.ConsecutiveSuccesses != 3 {
		t.Errorf("expected ConsecutiveSuccesses=3, got %d", hc.ConsecutiveSuccesses)
	}
	if hc.ConsecutiveFails != 1 {
		t.Errorf("expected ConsecutiveFails=1, got %d", hc.ConsecutiveFails)
	}
	if hc.FailureReason != "" {
		t.Errorf("expected empty FailureReason, got %s", hc.FailureReason)
	}
	if !hc.CreatedOn.Equal(now) {
		t.Errorf("expected CreatedOn=%v, got %v", now, hc.CreatedOn)
	}
	if !hc.ModifiedOn.Equal(now) {
		t.Errorf("expected ModifiedOn=%v, got %v", now, hc.ModifiedOn)
	}
}

func TestCfHealthcheckToHealthcheckNilTimes(t *testing.T) {
	cfHC := cloudflare.Healthcheck{
		ID:      "hc-def456",
		Name:    "tcp-check",
		Address: "10.0.0.1",
		Type:    "TCP",
		Status:  "unknown",
	}

	hc := cfHealthcheckToHealthcheck(cfHC)

	if hc.ID != "hc-def456" {
		t.Errorf("expected ID=hc-def456, got %s", hc.ID)
	}
	if !hc.CreatedOn.IsZero() {
		t.Errorf("expected zero CreatedOn when source is nil, got %v", hc.CreatedOn)
	}
	if !hc.ModifiedOn.IsZero() {
		t.Errorf("expected zero ModifiedOn when source is nil, got %v", hc.ModifiedOn)
	}
}

// --- Healthcheck type tests ---

func TestHealthcheckType(t *testing.T) {
	hc := &Healthcheck{
		ID:          "hc-001",
		Name:        "api-check",
		Address:     "api.example.com",
		Type:        "HTTP",
		Status:      "unhealthy",
		Suspended:   true,
		Interval:    120,
		Timeout:     10,
		Retries:     5,
		Description: "API endpoint monitor",
	}
	if hc.ID != "hc-001" {
		t.Errorf("unexpected ID: %s", hc.ID)
	}
	if hc.Type != "HTTP" {
		t.Errorf("unexpected Type: %s", hc.Type)
	}
	if hc.Status != "unhealthy" {
		t.Errorf("unexpected Status: %s", hc.Status)
	}
	if !hc.Suspended {
		t.Error("expected Suspended=true")
	}
	if hc.Interval != 120 {
		t.Errorf("unexpected Interval: %d", hc.Interval)
	}
	if hc.Description != "API endpoint monitor" {
		t.Errorf("unexpected Description: %s", hc.Description)
	}
}
