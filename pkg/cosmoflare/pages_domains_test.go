package cosmoflare

import (
	"context"
	"net/http"
	"testing"
)

// --- Validation ---

func TestPagesListDomainsValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.ListDomains(context.Background(), ""); err == nil {
		t.Error("expected error when project is empty")
	}
}

func TestPagesAttachDomainValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.AttachDomain(context.Background(), "", "example.com"); err == nil {
		t.Error("expected error when project is empty")
	}
	if _, err := svc.AttachDomain(context.Background(), "my-site", ""); err == nil {
		t.Error("expected error when domain is empty")
	}
}

func TestPagesDetachDomainValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.DetachDomain(context.Background(), "", "example.com"); err == nil {
		t.Error("expected error when project is empty")
	}
	if err := svc.DetachDomain(context.Background(), "my-site", ""); err == nil {
		t.Error("expected error when domain is empty")
	}
}

// --- Success paths ---

func TestPagesListDomainsSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/account-test-123/pages/projects/my-site/domains" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":                "domain-1",
					"name":              "example.com",
					"status":            "active",
					"verification_data": map[string]interface{}{"status": "active"},
					"validation_data":   map[string]interface{}{"status": "active", "method": "http"},
				},
			},
		})
	})
	defer server.Close()

	domains, err := svc.ListDomains(context.Background(), "my-site")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(domains))
	}
	if domains[0].Name != "example.com" || domains[0].Status != "active" {
		t.Errorf("unexpected domain: %+v", domains[0])
	}
}

func TestPagesAttachDomainSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                "domain-1",
				"name":              "example.com",
				"status":            "pending",
				"verification_data": map[string]interface{}{"status": "pending"},
				"validation_data":   map[string]interface{}{"status": "pending", "method": "http"},
			},
		})
	})
	defer server.Close()

	domain, err := svc.AttachDomain(context.Background(), "my-site", "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain.Name != "example.com" || domain.Status != "pending" {
		t.Errorf("unexpected domain: %+v", domain)
	}
	if domain.ValidationStatus != "pending" {
		t.Errorf("expected validation status surfaced, got %+v", domain)
	}
}

func TestPagesDetachDomainSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	if err := svc.DetachDomain(context.Background(), "my-site", "example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
