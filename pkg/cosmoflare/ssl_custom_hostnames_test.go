package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func chMockSetup(handler http.HandlerFunc) (*SSLCustomHostnameService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewSSLCustomHostnameService(cf, "zone-ch-123")
	return svc, server
}

func chWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func chHostname(id, name, status string) cloudflare.CustomHostname {
	return cloudflare.CustomHostname{
		ID:                 id,
		Hostname:           name,
		Status:             cloudflare.CustomHostnameStatus(status),
		CustomOriginServer: "origin.example.com",
		SSL: &cloudflare.CustomHostnameSSL{
			Status: "pending_validation",
			Method: "http",
			ValidationErrors: []cloudflare.SSLValidationError{
				{Message: "validation failed"},
			},
		},
	}
}

func TestNewSSLCustomHostnameServiceValidation(t *testing.T) {
	_, err := NewSSLCustomHostnameService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewSSLCustomHostnameService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewSSLCustomHostnameService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewSSLCustomHostnameServiceFromCredsValidation(t *testing.T) {
	_, err := NewSSLCustomHostnameServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}
	if !strings.Contains(err.Error(), "zone ID is required") {
		t.Errorf("unexpected error message: %v", err)
	}

	_, err = NewSSLCustomHostnameServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
	if !strings.Contains(err.Error(), "API token is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCustomHostnameList_Empty(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		chWriteJSON(w, map[string]interface{}{
			"success": true,
			"result":  []cloudflare.CustomHostname{},
			"result_info": map[string]int{
				"page": 1, "per_page": 20, "count": 0, "total_count": 0, "total_pages": 1,
			},
		})
	})
	defer server.Close()

	hostnames, err := svc.List(context.Background(), CustomHostnameListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hostnames) != 0 {
		t.Errorf("expected 0 hostnames, got %d", len(hostnames))
	}
}

func TestCustomHostnameList_Filtered(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		chWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": []cloudflare.CustomHostname{
				chHostname("ch-1", "app.customer.com", "active"),
				chHostname("ch-2", "shop.other.com", "pending"),
			},
			"result_info": map[string]int{
				"page": 1, "per_page": 20, "count": 2, "total_count": 2, "total_pages": 1,
			},
		})
	})
	defer server.Close()

	hostnames, err := svc.List(context.Background(), CustomHostnameListOptions{Hostname: "customer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hostnames) != 1 {
		t.Fatalf("expected 1 hostname, got %d", len(hostnames))
	}
	if hostnames[0].Hostname != "app.customer.com" {
		t.Errorf("hostname = %q, want app.customer.com", hostnames[0].Hostname)
	}
	if hostnames[0].Status != "active" {
		t.Errorf("status = %q, want active", hostnames[0].Status)
	}
	if hostnames[0].SSLStatus != "pending_validation" {
		t.Errorf("ssl_status = %q, want pending_validation", hostnames[0].SSLStatus)
	}
}

func TestCustomHostnameCreate(t *testing.T) {
	var gotMethod string
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		chWriteJSON(w, map[string]interface{}{
			"success": true,
			"result":  chHostname("ch-new", "app.customer.com", "pending"),
		})
	})
	defer server.Close()

	created, err := svc.Create(context.Background(), "app.customer.com", "origin.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if created.ID != "ch-new" {
		t.Errorf("id = %q, want ch-new", created.ID)
	}
	if created.CustomOriginServer != "origin.example.com" {
		t.Errorf("custom_origin_server = %q, want origin.example.com", created.CustomOriginServer)
	}
}

func TestCustomHostnameCreate_RequiresHostname(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error when hostname is empty")
	}
	if !strings.Contains(err.Error(), "hostname is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCustomHostnameCreate_APIError(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		chWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "hostname already in use"}},
		})
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "app.customer.com", "")
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "failed to create custom hostname") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCustomHostnameGet_PendingVerification(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/ch-1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		chWriteJSON(w, map[string]interface{}{
			"success": true,
			"result":  chHostname("ch-1", "app.customer.com", "pending"),
		})
	})
	defer server.Close()

	got, err := svc.Get(context.Background(), "ch-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.VerificationStatus != "pending_validation" {
		t.Errorf("verification_status = %q, want pending_validation", got.VerificationStatus)
	}
	if got.VerificationType != "http" {
		t.Errorf("verification_type = %q, want http", got.VerificationType)
	}
	if len(got.VerificationErrors) != 1 || got.VerificationErrors[0] != "validation failed" {
		t.Errorf("verification_errors = %v, want [validation failed]", got.VerificationErrors)
	}
}

func TestCustomHostnameGet_RequiresID(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when id is empty")
	}
}

func TestCustomHostnameUpdate(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		chWriteJSON(w, map[string]interface{}{
			"success": true,
			"result":  chHostname("ch-1", "app.customer.com", "pending"),
		})
	})
	defer server.Close()

	origin := "new-origin.example.com"
	updated, err := svc.Update(context.Background(), "ch-1", CustomHostnameUpdateOptions{CustomOriginServer: &origin})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != "ch-1" {
		t.Errorf("id = %q, want ch-1", updated.ID)
	}
}

func TestCustomHostnameUpdate_RequiresID(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	defer server.Close()

	origin := "origin.example.com"
	_, err := svc.Update(context.Background(), "", CustomHostnameUpdateOptions{CustomOriginServer: &origin})
	if err == nil {
		t.Fatal("expected error when id is empty")
	}
}

func TestCustomHostnameUpdate_RequiresAtLeastOneField(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	defer server.Close()

	_, err := svc.Update(context.Background(), "ch-1", CustomHostnameUpdateOptions{})
	if err == nil {
		t.Fatal("expected error when no fields set")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCustomHostnameDelete(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		chWriteJSON(w, map[string]interface{}{"success": true})
	})
	defer server.Close()

	if err := svc.Delete(context.Background(), "ch-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomHostnameDelete_RequiresID(t *testing.T) {
	svc, server := chMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	defer server.Close()

	if err := svc.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected error when id is empty")
	}
}
