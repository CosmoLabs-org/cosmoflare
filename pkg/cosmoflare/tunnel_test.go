package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func tunnelMockSetup(handler http.HandlerFunc) (*TunnelService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewTunnelService(cf, "account-test-123")
	return svc, server
}

func tunnelWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewTunnelServiceValidation(t *testing.T) {
	_, err := NewTunnelService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewTunnelService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewTunnelServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewTunnelService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewTunnelServiceFromCredsValidation(t *testing.T) {
	_, err := NewTunnelServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewTunnelServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewTunnelServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewTunnelServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

// --- Method validation ---

func TestTunnelGetValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

func TestTunnelResolveValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Resolve(context.Background(), "")
	if err == nil {
		t.Error("expected error when tunnel ID or name is empty")
	}
}

func TestTunnelCreateValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Create(context.Background(), "", "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestTunnelDeleteValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.Delete(context.Background(), "", false); err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

func TestTunnelTokenValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Token(context.Background(), "")
	if err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

func TestTunnelConnectionsValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Connections(context.Background(), "")
	if err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

func TestTunnelCleanupConnectionsValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.CleanupTunnelConnections(context.Background(), ""); err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

func TestTunnelCleanupValidation(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.Cleanup(context.Background(), ""); err == nil {
		t.Error("expected error when tunnel ID is empty")
	}
}

// --- API success tests ---

func TestTunnelCreateSuccess(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/accounts/account-test-123/cfd_tunnel") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if name, _ := body["name"].(string); name != "my-tunnel" {
			t.Errorf("expected body name=my-tunnel, got %v", body["name"])
		}
		if secret, _ := body["tunnel_secret"].(string); secret == "" {
			t.Error("expected a generated tunnel_secret in the create request")
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":         "11111111-2222-3333-4444-555555555555",
				"name":       "my-tunnel",
				"created_at": "2026-09-20T10:00:00Z",
				"status":     "inactive",
			},
		})
	})
	defer server.Close()

	tun, err := svc.Create(context.Background(), "my-tunnel", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tun.ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected ID=11111111-..., got %s", tun.ID)
	}
	if tun.Name != "my-tunnel" {
		t.Errorf("expected Name=my-tunnel, got %s", tun.Name)
	}
	if tun.Status != "inactive" {
		t.Errorf("expected Status=inactive, got %s", tun.Status)
	}
	if tun.CreatedAt == nil || tun.CreatedAt.UTC().Year() != 2026 {
		t.Errorf("expected CreatedAt in 2026, got %v", tun.CreatedAt)
	}
}

func TestTunnelCreateError(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "my-tunnel", "")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestTunnelListSuccess(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/accounts/account-test-123/cfd_tunnel") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":     "11111111-2222-3333-4444-555555555555",
					"name":   "first",
					"status": "active",
					"connections": []map[string]interface{}{
						{"id": "conn-1", "colo_name": "SJC", "origin_ip": "203.0.113.10"},
					},
				},
				{"id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "name": "second"},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 100, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	tunnels, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(tunnels))
	}
	if tunnels[0].Name != "first" {
		t.Errorf("expected Name=first, got %s", tunnels[0].Name)
	}
	if len(tunnels[0].Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(tunnels[0].Connections))
	}
	if tunnels[0].Connections[0].ColoName != "SJC" {
		t.Errorf("expected ColoName=SJC, got %s", tunnels[0].Connections[0].ColoName)
	}
	if tunnels[1].Status != "" {
		t.Errorf("expected empty status for second tunnel, got %s", tunnels[1].Status)
	}
}

func TestTunnelListError(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestTunnelGetSuccess(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/cfd_tunnel/11111111-2222-3333-4444-555555555555") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":       "11111111-2222-3333-4444-555555555555",
				"name":     "my-tunnel",
				"status":   "active",
				"tun_type": "cfd_tunnel",
			},
		})
	})
	defer server.Close()

	tun, err := svc.Get(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tun.Name != "my-tunnel" {
		t.Errorf("expected Name=my-tunnel, got %s", tun.Name)
	}
	if tun.TunnelType != "cfd_tunnel" {
		t.Errorf("expected TunnelType=cfd_tunnel, got %s", tun.TunnelType)
	}
}

func TestTunnelResolveByUUID(t *testing.T) {
	uuid := "11111111-2222-3333-4444-555555555555"
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "" {
			t.Errorf("UUID input should not filter by name, got name=%q", r.URL.Query().Get("name"))
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":   uuid,
				"name": "my-tunnel",
			},
		})
	})
	defer server.Close()

	tun, err := svc.Resolve(context.Background(), uuid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tun.ID != uuid {
		t.Errorf("expected ID=%s, got %s", uuid, tun.ID)
	}
}

func TestTunnelResolveByName(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "my-tunnel" {
			t.Errorf("expected name filter my-tunnel, got %q", got)
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "11111111-2222-3333-4444-555555555555", "name": "my-tunnel"},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 100, "total_count": 1, "count": 1,
			},
		})
	})
	defer server.Close()

	tun, err := svc.Resolve(context.Background(), "my-tunnel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tun.ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("unexpected ID: %s", tun.ID)
	}
}

func TestTunnelResolveByNameNotFound(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []map[string]interface{}{},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 100, "total_count": 0, "count": 0,
			},
		})
	})
	defer server.Close()

	_, err := svc.Resolve(context.Background(), "no-such-tunnel")
	if err == nil {
		t.Fatal("expected error when no tunnel matches the name")
	}
	if !strings.Contains(err.Error(), "no tunnel named") {
		t.Errorf("error should mention the missing name, got: %v", err)
	}
}

func TestTunnelResolveByNameAmbiguous(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "11111111-2222-3333-4444-555555555555", "name": "dup"},
				{"id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "name": "dup"},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 100, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	_, err := svc.Resolve(context.Background(), "dup")
	if err == nil {
		t.Fatal("expected error when multiple tunnels match the name")
	}
	if !strings.Contains(err.Error(), "use a tunnel ID") {
		t.Errorf("error should suggest using a tunnel ID, got: %v", err)
	}
}

func TestTunnelDeleteSuccess(t *testing.T) {
	calls := []string{}
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	if err := svc.Delete(context.Background(), "11111111-2222-3333-4444-555555555555", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0] != "DELETE /accounts/account-test-123/cfd_tunnel/11111111-2222-3333-4444-555555555555" {
		t.Errorf("expected a single DELETE of the tunnel, got %v", calls)
	}
}

func TestTunnelDeleteCascadeTearsDownConnectionsFirst(t *testing.T) {
	calls := []string{}
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	if err := svc.Delete(context.Background(), "11111111-2222-3333-4444-555555555555", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"DELETE /accounts/account-test-123/cfd_tunnel/11111111-2222-3333-4444-555555555555/connections",
		"DELETE /accounts/account-test-123/cfd_tunnel/11111111-2222-3333-4444-555555555555",
	}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("expected connections teardown before delete, got %v", calls)
	}
}

func TestTunnelCleanupCallsBothEndpoints(t *testing.T) {
	calls := []string{}
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	if err := svc.Cleanup(context.Background(), "11111111-2222-3333-4444-555555555555"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(calls), calls)
	}
	if !strings.HasSuffix(calls[0], "/connections") {
		t.Errorf("expected connections teardown first, got %v", calls)
	}
}

func TestTunnelDeleteError(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	if err := svc.Delete(context.Background(), "11111111-2222-3333-4444-555555555555", false); err == nil {
		t.Error("expected error on server failure")
	}
}

func TestTunnelTokenSuccess(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/token") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  "eyJhbGciOiJQUzUxIn0.test-token",
		})
	})
	defer server.Close()

	token, err := svc.Token(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "eyJhbGciOiJPUzUxIn0.test-token" && token != "eyJhbGciOiJQUzUxIn0.test-token" {
		t.Errorf("unexpected token: %s", token)
	}
}

func TestTunnelTokenError(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.Token(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestTunnelConnectionsSuccess(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/connections") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		tunnelWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":      "connector-1",
					"version": "2026.8.1",
					"arch":    "darwin-arm64",
					"conns": []map[string]interface{}{
						{"id": "conn-1", "colo_name": "SJC", "origin_ip": "203.0.113.10", "client_version": "2026.8.1"},
					},
				},
			},
		})
	})
	defer server.Close()

	connectors, err := svc.Connections(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	if connectors[0].ID != "connector-1" || connectors[0].Version != "2026.8.1" {
		t.Errorf("unexpected connector: %+v", connectors[0])
	}
	if len(connectors[0].Connections) != 1 {
		t.Fatalf("expected 1 edge connection, got %d", len(connectors[0].Connections))
	}
	if connectors[0].Connections[0].ColoName != "SJC" {
		t.Errorf("expected ColoName=SJC, got %s", connectors[0].Connections[0].ColoName)
	}
}

func TestTunnelConnectionsError(t *testing.T) {
	svc, server := tunnelMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		tunnelWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Connections(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

// --- Time mapping ---

func TestMapTunnelNilTimestamps(t *testing.T) {
	tun := mapTunnel(cloudflare.Tunnel{ID: "x", Name: "y"})
	if tun.CreatedAt != nil || tun.DeletedAt != nil {
		t.Errorf("expected nil timestamps, got %v / %v", tun.CreatedAt, tun.DeletedAt)
	}
	now := time.Now().UTC()
	tun = mapTunnel(cloudflare.Tunnel{ID: "x", Name: "y", CreatedAt: &now})
	if tun.CreatedAt == nil || !tun.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt=%v, got %v", now, tun.CreatedAt)
	}
}
