package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func hyperdriveMockSetup(handler http.HandlerFunc) (*HyperdriveService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewHyperdriveService(cf, "account-test-123")
	return svc, server
}

func hyperdriveWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewHyperdriveServiceValidation(t *testing.T) {
	_, err := NewHyperdriveService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewHyperdriveService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewHyperdriveServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewHyperdriveService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewHyperdriveServiceFromCredsValidation(t *testing.T) {
	_, err := NewHyperdriveServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewHyperdriveServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewHyperdriveServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewHyperdriveServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

// --- Method validation ---

func TestHyperdriveGetValidation(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when config ID is empty")
	}
}

func TestHyperdriveCreateValidation(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Create(context.Background(), "", HyperdriveOriginConfig{})
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestHyperdriveUpdateValidation(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Update(context.Background(), "", HyperdriveUpdateParams{})
	if err == nil {
		t.Error("expected error when config ID is empty")
	}
}

func TestHyperdriveDeleteValidation(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error when config ID is empty")
	}
}

// --- Mock API tests ---

func TestHyperdriveList(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		hyperdriveWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"id":   "cfg-001",
					"name": "my-db",
					"origin": map[string]interface{}{
						"database": "mydb",
						"host":     "db.example.com",
						"port":     5432,
						"scheme":   "postgres",
						"user":     "admin",
					},
				},
				{
					"id":   "cfg-002",
					"name": "staging-db",
					"origin": map[string]interface{}{
						"database": "staging",
						"host":     "staging.example.com",
						"port":     5432,
						"scheme":   "postgres",
						"user":     "reader",
					},
				},
			},
		})
	})
	defer server.Close()

	configs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	if configs[0].ID != "cfg-001" {
		t.Errorf("expected ID=cfg-001, got %s", configs[0].ID)
	}
	if configs[0].Name != "my-db" {
		t.Errorf("expected Name=my-db, got %s", configs[0].Name)
	}
	if configs[1].ID != "cfg-002" {
		t.Errorf("expected ID=cfg-002, got %s", configs[1].ID)
	}
}

func TestHyperdriveGet(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		hyperdriveWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":   "cfg-001",
				"name": "my-db",
				"origin": map[string]interface{}{
					"database": "mydb",
					"host":     "db.example.com",
					"port":     5432,
					"scheme":   "postgres",
					"user":     "admin",
				},
				"caching": map[string]interface{}{
					"disabled": false,
					"max_age":  60,
				},
			},
		})
	})
	defer server.Close()

	cfg, err := svc.Get(context.Background(), "cfg-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ID != "cfg-001" {
		t.Errorf("expected ID=cfg-001, got %s", cfg.ID)
	}
	if cfg.Origin.Database != "mydb" {
		t.Errorf("expected Origin.Database=mydb, got %s", cfg.Origin.Database)
	}
	if cfg.Origin.Host != "db.example.com" {
		t.Errorf("expected Origin.Host=db.example.com, got %s", cfg.Origin.Host)
	}
}

func TestHyperdriveDelete(t *testing.T) {
	svc, server := hyperdriveMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		hyperdriveWriteJSON(w, map[string]interface{}{
			"success": true,
			"result":  nil,
		})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "cfg-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
