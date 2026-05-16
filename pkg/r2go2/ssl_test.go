package r2go2

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func TestNewSSLServiceValidation(t *testing.T) {
	_, err := NewSSLService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewSSLService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewSSLService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewSSLServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewSSLService(cf, "zone123")
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

func TestNewSSLServiceFromCredsValidation(t *testing.T) {
	_, err := NewSSLServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}
	if !strings.Contains(err.Error(), "zone ID is required") {
		t.Errorf("unexpected error message: %v", err)
	}

	_, err = NewSSLServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
	if !strings.Contains(err.Error(), "API token is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewSSLServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewSSLServiceFromCreds("zone123", "test-token")
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

func TestSSLOptions(t *testing.T) {
	cfg := &sslConfig{}

	WithMinTLSVersion("1.2")(cfg)
	if cfg.minTLSVersion == nil {
		t.Fatal("expected minTLSVersion to be set")
	}
	if *cfg.minTLSVersion != "1.2" {
		t.Errorf("expected minTLSVersion=1.2, got %s", *cfg.minTLSVersion)
	}

	WithAlwaysHTTPS(true)(cfg)
	if cfg.alwaysHTTPS == nil {
		t.Fatal("expected alwaysHTTPS to be set")
	}
	if !*cfg.alwaysHTTPS {
		t.Error("expected alwaysHTTPS=true")
	}

	WithAutoHTTPSRewrites(true)(cfg)
	if cfg.autoRewrites == nil {
		t.Fatal("expected autoRewrites to be set")
	}
	if !*cfg.autoRewrites {
		t.Error("expected autoRewrites=true")
	}
}

func TestSSLOptionsDisabled(t *testing.T) {
	cfg := &sslConfig{}

	WithAlwaysHTTPS(false)(cfg)
	if cfg.alwaysHTTPS == nil {
		t.Fatal("expected alwaysHTTPS to be set")
	}
	if *cfg.alwaysHTTPS {
		t.Error("expected alwaysHTTPS=false")
	}

	WithAutoHTTPSRewrites(false)(cfg)
	if cfg.autoRewrites == nil {
		t.Fatal("expected autoRewrites to be set")
	}
	if *cfg.autoRewrites {
		t.Error("expected autoRewrites=false")
	}
}

func TestSSLUpdateSSLValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewSSLService(cf, "zone123")

	_, err := svc.UpdateSSL(context.Background(), "")
	if err == nil {
		t.Error("expected error when SSL mode value is empty")
	}
	if !strings.Contains(err.Error(), "SSL mode value is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSLUpdateSettingsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewSSLService(cf, "zone123")

	err := svc.UpdateSettings(context.Background())
	if err == nil {
		t.Error("expected error when no options provided")
	}
	if !strings.Contains(err.Error(), "at least one setting option is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSLValidationErrorTypes(t *testing.T) {
	// NewSSLService nil API → R2ValidationError
	_, err := NewSSLService(nil, "zone123")
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// NewSSLService empty zoneID → R2ValidationError
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewSSLService(cf, "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// NewSSLServiceFromCreds empty zoneID → R2ValidationError
	_, err = NewSSLServiceFromCreds("", "token")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// NewSSLServiceFromCreds empty token → R2ValidationError
	_, err = NewSSLServiceFromCreds("zone123", "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// UpdateSSL empty value → R2ValidationError
	svc, _ := NewSSLService(cf, "zone123")
	_, err = svc.UpdateSSL(context.Background(), "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	// UpdateSettings no opts → R2ValidationError
	err = svc.UpdateSettings(context.Background())
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestSSLTypes(t *testing.T) {
	status := &SSLStatus{
		ID:                "ssl",
		Value:             "full",
		Editable:          true,
		CertificateStatus: "active",
		ModifiedOn:        "2026-01-01T00:00:00Z",
	}
	if status.ID != "ssl" {
		t.Errorf("unexpected ID: %s", status.ID)
	}
	if status.Value != "full" {
		t.Errorf("unexpected Value: %s", status.Value)
	}
	if !status.Editable {
		t.Error("expected Editable=true")
	}
	if status.CertificateStatus != "active" {
		t.Errorf("unexpected CertificateStatus: %s", status.CertificateStatus)
	}

	verification := &SSLVerification{
		CertificateStatus:  "active",
		VerificationType:   "cname",
		ValidationMethod:   "txt",
		CertPackUUID:       "uuid-123",
		VerificationStatus: true,
		BrandCheck:         false,
	}
	if verification.CertificateStatus != "active" {
		t.Errorf("unexpected CertificateStatus: %s", verification.CertificateStatus)
	}
	if !verification.VerificationStatus {
		t.Error("expected VerificationStatus=true")
	}

	settings := &SSLSettings{
		MinTLSVersion:          "1.2",
		AlwaysUseHTTPS:         true,
		AutomaticHTTPSRewrites: true,
		UniversalSSL:           true,
	}
	if settings.MinTLSVersion != "1.2" {
		t.Errorf("unexpected MinTLSVersion: %s", settings.MinTLSVersion)
	}
	if !settings.AlwaysUseHTTPS {
		t.Error("expected AlwaysUseHTTPS=true")
	}
	if !settings.AutomaticHTTPSRewrites {
		t.Error("expected AutomaticHTTPSRewrites=true")
	}
	if !settings.UniversalSSL {
		t.Error("expected UniversalSSL=true")
	}
}
