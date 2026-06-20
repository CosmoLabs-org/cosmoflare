package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func TestRegistrarService_List(t *testing.T) {
	const accountID = "account-test-123"
	// A clearly-future expiry so the assertion below is meaningful.
	futureExpiry := time.Now().UTC().Add(365 * 24 * time.Hour).Truncate(time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/registrar/domains"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		// Mirror the real Cloudflare Registrar list response shape. The domain
		// name is carried in the "id" field (see cloudflare-go registrar.go).
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":                "locked-example.com",
					"current_registrar": "Cloudflare",
					"expires_at":        futureExpiry.Format(time.RFC3339),
					"locked":            true,
				},
				{
					"id":                "open-example.com",
					"current_registrar": "Cloudflare",
					"expires_at":        futureExpiry.Format(time.RFC3339),
					"locked":            false,
				},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 20, "count": 2, "total_count": 2,
			},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc := NewRegistrarService(cf, accountID)

	infos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 results, got %d", len(infos))
	}

	locked, ok := infos["locked-example.com"]
	if !ok {
		t.Fatalf("expected locked-example.com in results, got %+v", infos)
	}
	if !locked.TransferLock {
		t.Errorf("expected locked-example.com TransferLock=true, got false")
	}
	if locked.Registrar != "cloudflare" {
		t.Errorf("expected Registrar=cloudflare, got %q", locked.Registrar)
	}
	if locked.ExpiresAt == nil {
		t.Fatalf("expected locked-example.com ExpiresAt non-nil")
	}
	if !locked.ExpiresAt.After(time.Now()) {
		t.Errorf("expected ExpiresAt in the future, got %v", locked.ExpiresAt)
	}

	open, ok := infos["open-example.com"]
	if !ok {
		t.Fatalf("expected open-example.com in results, got %+v", infos)
	}
	if open.TransferLock {
		t.Errorf("expected open-example.com TransferLock=false, got true")
	}
	if open.Registrar != "cloudflare" {
		t.Errorf("expected Registrar=cloudflare, got %q", open.Registrar)
	}
	if open.ExpiresAt == nil {
		t.Fatalf("expected open-example.com ExpiresAt non-nil")
	}
	if !open.ExpiresAt.After(time.Now()) {
		t.Errorf("expected open-example.com ExpiresAt in the future, got %v", open.ExpiresAt)
	}
}

func TestRegistrarService_ListValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc := NewRegistrarService(cf, "")
	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestRegistrarService_ListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"errors":      []interface{}{},
			"result":      []map[string]interface{}{},
			"result_info": map[string]interface{}{"page": 1, "per_page": 20, "count": 0, "total_count": 0},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc := NewRegistrarService(cf, "account-test-123")

	infos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 results, got %d", len(infos))
	}
}

func TestNewRegistrarServiceFromCreds(t *testing.T) {
	if _, err := NewRegistrarServiceFromCreds("", "tok"); err == nil {
		t.Error("expected error for empty account ID")
	}
	if _, err := NewRegistrarServiceFromCreds("acct", ""); err == nil {
		t.Error("expected error for empty API token")
	}
	svc, err := NewRegistrarServiceFromCreds("acct", "tok")
	if err != nil || svc == nil {
		t.Fatalf("expected service, got svc=%v err=%v", svc, err)
	}
}
