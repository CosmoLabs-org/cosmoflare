package r2go2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func sslMockSetup(handler http.HandlerFunc) (*SSLService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewSSLService(cf, "zone-ssl-123")
	return svc, server
}

func sslWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

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

// --- httptest-based API mock tests ---

func TestSSLGetSSLWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                 "ssl",
				"value":              "full",
				"editable":           true,
				"certificate_status": "active",
				"modified_on":        "2026-01-15T00:00:00Z",
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	status, err := svc.GetSSL(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Value != "full" {
		t.Errorf("expected Value=full, got %s", status.Value)
	}
	if status.CertificateStatus != "active" {
		t.Errorf("expected CertificateStatus=active, got %s", status.CertificateStatus)
	}
	if !status.Editable {
		t.Error("expected Editable=true")
	}
}

func TestSSLUpdateSSLWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                 "ssl",
				"value":              "strict",
				"editable":           true,
				"certificate_status": "active",
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	status, err := svc.UpdateSSL(ctx, "strict")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Value != "strict" {
		t.Errorf("expected Value=strict, got %s", status.Value)
	}
}

func TestSSLGetVerificationWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"certificate_status":  "active",
					"verification_type":   "cname",
					"validation_method":   "txt",
					"cert_pack_uuid":      "uuid-001",
					"verification_status": true,
					"brand_check":         false,
				},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	verifications, err := svc.GetVerification(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(verifications) != 1 {
		t.Fatalf("expected 1 verification, got %d", len(verifications))
	}
	if verifications[0].CertificateStatus != "active" {
		t.Errorf("expected CertificateStatus=active, got %s", verifications[0].CertificateStatus)
	}
	if !verifications[0].VerificationStatus {
		t.Error("expected VerificationStatus=true")
	}
}

func TestSSLGetSettingsWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "min_tls_version", "value": "1.2", "editable": true},
				{"id": "always_use_https", "value": "on", "editable": true},
				{"id": "automatic_https_rewrites", "value": "on", "editable": true},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.MinTLSVersion != "1.2" {
		t.Errorf("expected MinTLSVersion=1.2, got %s", settings.MinTLSVersion)
	}
	if !settings.AlwaysUseHTTPS {
		t.Error("expected AlwaysUseHTTPS=true")
	}
	if !settings.AutomaticHTTPSRewrites {
		t.Error("expected AutomaticHTTPSRewrites=true")
	}
}

func TestSSLGetSettingsWithUniversalSSL(t *testing.T) {
	callCount := 0
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "ssl/universal") {
			sslWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result":  map[string]interface{}{"enabled": true},
			})
			return
		}
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "min_tls_version", "value": "1.3", "editable": true},
				{"id": "always_use_https", "value": "off", "editable": true},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.MinTLSVersion != "1.3" {
		t.Errorf("expected MinTLSVersion=1.3, got %s", settings.MinTLSVersion)
	}
	if settings.AlwaysUseHTTPS {
		t.Error("expected AlwaysUseHTTPS=false when value is 'off'")
	}
	if !settings.UniversalSSL {
		t.Error("expected UniversalSSL=true")
	}
}

func TestSSLUpdateSettingsWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.UpdateSettings(ctx,
		WithMinTLSVersion("1.3"),
		WithAlwaysHTTPS(true),
		WithAutoHTTPSRewrites(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSLUpdateSettingsDisabledWithMock(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		sslWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.UpdateSettings(ctx,
		WithAlwaysHTTPS(false),
		WithAutoHTTPSRewrites(false),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSLGetSSLAPIError(t *testing.T) {
	svc, server := sslMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		sslWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "internal error"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.GetSSL(ctx)
	if err == nil {
		t.Fatal("expected error from API")
	}
}

func TestSSLStatusJSONMarshal(t *testing.T) {
	s := &SSLStatus{
		ID: "ssl", Value: "full", Editable: true,
		CertificateStatus: "active", ModifiedOn: "2026-01-01T00:00:00Z",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded SSLStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Value != "full" {
		t.Errorf("Value mismatch: got %q", decoded.Value)
	}
}

func TestSSLVerificationJSONMarshal(t *testing.T) {
	v := &SSLVerification{
		CertificateStatus: "active", VerificationType: "cname",
		ValidationMethod: "txt", CertPackUUID: "uuid-1",
		VerificationStatus: true, BrandCheck: false,
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded SSLVerification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !decoded.VerificationStatus {
		t.Error("expected VerificationStatus=true")
	}
}

func TestSSLSettingsJSONMarshal(t *testing.T) {
	s := &SSLSettings{
		MinTLSVersion: "1.3", AlwaysUseHTTPS: true,
		AutomaticHTTPSRewrites: true, UniversalSSL: true,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded SSLSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.MinTLSVersion != "1.3" {
		t.Errorf("MinTLSVersion mismatch: got %q", decoded.MinTLSVersion)
	}
}

func TestSSLNewFromCredsSuccess(t *testing.T) {
	svc, err := NewSSLServiceFromCreds("zone123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
}
