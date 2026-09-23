package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// lbStubServer spins up an httptest server standing in for the Cloudflare
// API and returns a client pointed at it (pattern: logpush_test.go
// logpushStubServer, itself from registrar_test.go registrarStubServer).
func lbStubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *cloudflare.API) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build stub cloudflare client: %v", err)
	}
	return server, cf
}

// lbEnvelope wraps a result in the standard Cloudflare response shape.
func lbEnvelope(result any) map[string]any {
	return map[string]any{
		"success": true,
		"errors":  []any{},
		"result":  result,
	}
}

// lbPoolJSON builds a raw API pool body with the fields the mapping layer
// reads back.
func lbPoolJSON(id, name, monitor string, enabled bool, origins []map[string]any) map[string]any {
	return map[string]any{
		"id":      id,
		"name":    name,
		"enabled": enabled,
		"monitor": monitor,
		"origins": origins,
	}
}

// lbMonitorJSON builds a raw API monitor body with the fields the mapping
// layer reads back.
func lbMonitorJSON(id, mtype, path, codes string, interval, retries, timeout int) map[string]any {
	return map[string]any{
		"id":             id,
		"type":           mtype,
		"path":           path,
		"expected_codes": codes,
		"interval":       interval,
		"retries":        retries,
		"timeout":        timeout,
	}
}

// TestLoadBalancerService_Create verifies Create posts the pool fields to
// the account pools endpoint and maps the API's pool back into the public
// view.
func TestLoadBalancerService_Create(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/pools"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["name"] != "primary" {
			t.Errorf("expected name primary, got %v", body["name"])
		}
		if body["monitor"] != "mon-1" {
			t.Errorf("expected monitor mon-1, got %v", body["monitor"])
		}
		if body["enabled"] != true {
			t.Errorf("expected enabled true (new pools start enabled), got %v", body["enabled"])
		}
		origins, ok := body["origins"].([]any)
		if !ok || len(origins) != 2 {
			t.Fatalf("expected 2 origins, got %v", body["origins"])
		}
		first, _ := origins[0].(map[string]any)
		if first["name"] != "web-1" || first["address"] != "10.0.0.1:80" || first["enabled"] != true {
			t.Errorf("first origin not mapped: %v", first)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(lbPoolJSON("pool-1", "primary", "mon-1", true, []map[string]any{
			{"name": "web-1", "address": "10.0.0.1:80", "enabled": true},
			{"name": "web-2", "address": "10.0.0.2:80", "enabled": false},
		})))
	})

	svc, err := NewLoadBalancerService(cf, accountID)
	if err != nil {
		t.Fatalf("unexpected error building service: %v", err)
	}
	pool, err := svc.Create(context.Background(), LoadBalancerPoolCreate{
		Name:    "primary",
		Monitor: "mon-1",
		Origins: []LoadBalancerOrigin{
			{Name: "web-1", Address: "10.0.0.1:80", Enabled: true},
			{Name: "web-2", Address: "10.0.0.2:80", Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.ID != "pool-1" || pool.Name != "primary" || pool.Monitor != "mon-1" || !pool.Enabled {
		t.Errorf("pool fields not mapped: %+v", pool)
	}
	if len(pool.Origins) != 2 || pool.Origins[1].Enabled {
		t.Errorf("origins not mapped: %+v", pool.Origins)
	}
}

// TestLoadBalancerService_CreateValidation verifies Create rejects a
// missing name, an empty origin set, and origins without addresses before
// any request is issued.
func TestLoadBalancerService_CreateValidation(t *testing.T) {
	t.Parallel()
	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected on validation failure, got %s %s", r.Method, r.URL.Path)
	})
	svc, _ := NewLoadBalancerService(cf, "account-test-123")

	if _, err := svc.Create(context.Background(), LoadBalancerPoolCreate{Origins: []LoadBalancerOrigin{{Name: "a", Address: "1.1.1.1"}}}); err == nil || !strings.Contains(err.Error(), "pool name is required") {
		t.Fatalf("expected name error, got %v", err)
	}
	if _, err := svc.Create(context.Background(), LoadBalancerPoolCreate{Name: "p"}); err == nil || !strings.Contains(err.Error(), "at least one origin is required") {
		t.Fatalf("expected origins error, got %v", err)
	}
	if _, err := svc.Create(context.Background(), LoadBalancerPoolCreate{Name: "p", Origins: []LoadBalancerOrigin{{Name: "a"}}}); err == nil || !strings.Contains(err.Error(), "missing an address") {
		t.Fatalf("expected address error, got %v", err)
	}
}

// TestLoadBalancerService_List verifies List issues a GET to the account
// pools endpoint and returns every mapped pool.
func TestLoadBalancerService_List(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/pools"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope([]map[string]any{
			lbPoolJSON("pool-1", "primary", "", true, []map[string]any{{"name": "web-1", "address": "10.0.0.1:80", "enabled": true}}),
			lbPoolJSON("pool-2", "fallback", "mon-9", false, nil),
		}))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	pools, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	if pools[0].ID != "pool-1" || !pools[0].Enabled {
		t.Errorf("first pool not mapped: %+v", pools[0])
	}
	if pools[1].Enabled || pools[1].Monitor != "mon-9" {
		t.Errorf("second pool not mapped: %+v", pools[1])
	}
}

// TestLoadBalancerService_Get verifies Get fetches a single pool by ID and
// maps timestamps into RFC 3339 strings.
func TestLoadBalancerService_Get(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/pools/pool-1"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		pool := lbPoolJSON("pool-1", "primary", "", true, []map[string]any{{"name": "web-1", "address": "10.0.0.1:80", "enabled": true}})
		pool["created_on"] = "2026-09-22T10:00:00Z"
		pool["healthy"] = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(pool))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	pool, err := svc.Get(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if pool.CreatedOn != want {
		t.Errorf("expected created_on %q, got %q", want, pool.CreatedOn)
	}
	if pool.Healthy == nil || !*pool.Healthy {
		t.Errorf("expected healthy=true, got %+v", pool.Healthy)
	}
}

// TestLoadBalancerService_Health verifies Health fetches the per-PoP health
// endpoint and maps the pool's health verdicts.
func TestLoadBalancerService_Health(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/pools/pool-1/health"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(map[string]any{
			"pool_id": "pool-1",
			"pop_health": map[string]any{
				"LAX": map[string]any{
					"healthy": true,
					"origins": []map[string]any{
						{"web-1": map[string]any{"healthy": true, "failure_reason": "", "response_code": 200}},
					},
				},
			},
		}))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	health, err := svc.Health(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.PoolID != "pool-1" {
		t.Errorf("expected pool_id pool-1, got %q", health.PoolID)
	}
	lax, ok := health.Pop["LAX"]
	if !ok || !lax.Healthy {
		t.Fatalf("LAX pop health not mapped: %+v", health.Pop)
	}
	if len(lax.Origins) != 1 {
		t.Fatalf("expected 1 origin probe result, got %d", len(lax.Origins))
	}
	if oh, ok := lax.Origins[0]["web-1"]; !ok || !oh.Healthy || oh.ResponseCode != 200 {
		t.Errorf("origin health not mapped: %+v", lax.Origins[0])
	}
}

// TestLoadBalancerService_Update verifies Update fetches the current pool,
// merges only the supplied changes onto it, and returns the WRITTEN state
// the API reports for the update (wave-1 stateful-stub lesson: what comes
// back must be the post-write pool, not a stale re-fetch).
func TestLoadBalancerService_Update(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/"+accountID+"/load_balancers/pools/pool-1":
			_ = json.NewEncoder(w).Encode(lbEnvelope(lbPoolJSON("pool-1", "primary", "mon-1", true, []map[string]any{
				{"name": "web-1", "address": "10.0.0.1:80", "enabled": true},
			})))
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/"+accountID+"/load_balancers/pools/pool-1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode PUT body: %v", err)
			}
			// merged: new name + disable, origins and monitor kept from GET
			if body["name"] != "primary-eu" {
				t.Errorf("expected merged name primary-eu, got %v", body["name"])
			}
			if body["enabled"] != false {
				t.Errorf("expected merged enabled=false, got %v", body["enabled"])
			}
			if body["monitor"] != "mon-1" {
				t.Errorf("expected monitor kept from current pool, got %v", body["monitor"])
			}
			origins, _ := body["origins"].([]any)
			if len(origins) != 1 {
				t.Errorf("expected origins kept from current pool, got %v", body["origins"])
			}
			// respond with the WRITTEN state
			_ = json.NewEncoder(w).Encode(lbEnvelope(lbPoolJSON("pool-1", "primary-eu", "mon-1", false, []map[string]any{
				{"name": "web-1", "address": "10.0.0.1:80", "enabled": true},
			})))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	disabled := false
	pool, err := svc.Update(context.Background(), "pool-1", LoadBalancerPoolUpdate{
		Name:    "primary-eu",
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.Name != "primary-eu" || pool.Enabled {
		t.Errorf("expected renamed disabled pool, got %+v", pool)
	}
}

// TestLoadBalancerService_UpdateValidation verifies Update rejects an
// invalid replacement origin set before any request is issued.
func TestLoadBalancerService_UpdateValidation(t *testing.T) {
	t.Parallel()
	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected on validation failure, got %s %s", r.Method, r.URL.Path)
	})
	svc, _ := NewLoadBalancerService(cf, "account-test-123")

	if _, err := svc.Update(context.Background(), "pool-1", LoadBalancerPoolUpdate{Origins: []LoadBalancerOrigin{{Name: "a"}}}); err == nil || !strings.Contains(err.Error(), "missing an address") {
		t.Fatalf("expected address error, got %v", err)
	}
}

// TestLoadBalancerService_Delete verifies Delete issues a DELETE against
// the pool endpoint.
func TestLoadBalancerService_Delete(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/pools/pool-1"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(map[string]any{"id": "pool-1"}))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	if err := svc.Delete(context.Background(), "pool-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLoadBalancerService_CreateMonitor verifies monitor creation posts the
// probe fields and maps the API's monitor back into the public view.
func TestLoadBalancerService_CreateMonitor(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/monitors"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["type"] != "https" {
			t.Errorf("expected type https, got %v", body["type"])
		}
		if body["path"] != "/healthz" {
			t.Errorf("expected path /healthz, got %v", body["path"])
		}
		if body["expected_codes"] != "2xx" {
			t.Errorf("expected expected_codes 2xx, got %v", body["expected_codes"])
		}
		if body["interval"] != float64(60) || body["retries"] != float64(2) || body["timeout"] != float64(5) {
			t.Errorf("probe timing not posted: %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(lbMonitorJSON("mon-1", "https", "/healthz", "2xx", 60, 2, 5)))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	monitor, err := svc.CreateMonitor(context.Background(), LoadBalancerMonitorCreate{
		Type:          "https",
		Path:          "/healthz",
		ExpectedCodes: "2xx",
		Interval:      60,
		Retries:       2,
		Timeout:       5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if monitor.ID != "mon-1" || monitor.Type != "https" || monitor.Interval != 60 {
		t.Errorf("monitor fields not mapped: %+v", monitor)
	}
}

// TestLoadBalancerService_CreateMonitorValidation verifies monitor creation
// rejects a missing type before any request is issued.
func TestLoadBalancerService_CreateMonitorValidation(t *testing.T) {
	t.Parallel()
	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected on validation failure, got %s %s", r.Method, r.URL.Path)
	})
	svc, _ := NewLoadBalancerService(cf, "account-test-123")

	if _, err := svc.CreateMonitor(context.Background(), LoadBalancerMonitorCreate{Path: "/"}); err == nil || !strings.Contains(err.Error(), "monitor type is required") {
		t.Fatalf("expected type error, got %v", err)
	}
}

// TestLoadBalancerService_ListMonitors verifies monitor list issues a GET
// to the account monitors endpoint and returns every mapped monitor.
func TestLoadBalancerService_ListMonitors(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/monitors"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope([]map[string]any{
			lbMonitorJSON("mon-1", "https", "/healthz", "2xx", 60, 2, 5),
			lbMonitorJSON("mon-2", "http", "/ping", "200", 30, 3, 2),
		}))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	monitors, err := svc.ListMonitors(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monitors) != 2 {
		t.Fatalf("expected 2 monitors, got %d", len(monitors))
	}
	if monitors[1].Interval != 30 || monitors[1].ExpectedCodes != "200" {
		t.Errorf("second monitor not mapped: %+v", monitors[1])
	}
}

// TestLoadBalancerService_UpdateMonitor verifies monitor update fetches the
// current monitor, merges only the supplied changes onto it, and returns
// the WRITTEN state.
func TestLoadBalancerService_UpdateMonitor(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/"+accountID+"/load_balancers/monitors/mon-1":
			_ = json.NewEncoder(w).Encode(lbEnvelope(lbMonitorJSON("mon-1", "http", "/ping", "2xx", 60, 2, 5)))
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/"+accountID+"/load_balancers/monitors/mon-1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode PUT body: %v", err)
			}
			// merged: new interval and timeout, everything else kept
			if body["interval"] != float64(30) || body["timeout"] != float64(2) {
				t.Errorf("expected merged timing, got %v", body)
			}
			if body["path"] != "/ping" || body["type"] != "http" {
				t.Errorf("expected path/type kept from current monitor, got %v", body)
			}
			_ = json.NewEncoder(w).Encode(lbEnvelope(lbMonitorJSON("mon-1", "http", "/ping", "2xx", 30, 2, 2)))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	monitor, err := svc.UpdateMonitor(context.Background(), "mon-1", LoadBalancerMonitorUpdate{
		Interval: 30,
		Timeout:  2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if monitor.Interval != 30 || monitor.Timeout != 2 || monitor.Path != "/ping" {
		t.Errorf("expected merged monitor, got %+v", monitor)
	}
}

// TestLoadBalancerService_DeleteMonitor verifies monitor delete issues a
// DELETE against the monitor endpoint.
func TestLoadBalancerService_DeleteMonitor(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/load_balancers/monitors/mon-1"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(lbEnvelope(map[string]any{"id": "mon-1"}))
	})

	svc, _ := NewLoadBalancerService(cf, accountID)
	if err := svc.DeleteMonitor(context.Background(), "mon-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLoadBalancerService_MalformedEnvelope verifies a non-JSON response
// body surfaces as an error rather than a zero-value resource, across both
// the pool and the monitor surfaces.
func TestLoadBalancerService_MalformedEnvelope(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "this is not json")
	})

	svc, _ := NewLoadBalancerService(cf, accountID)

	if _, err := svc.List(context.Background()); err == nil {
		t.Error("expected error listing pools against malformed envelope")
	}
	if _, err := svc.Get(context.Background(), "pool-1"); err == nil {
		t.Error("expected error getting pool against malformed envelope")
	}
	if _, err := svc.ListMonitors(context.Background()); err == nil {
		t.Error("expected error listing monitors against malformed envelope")
	}
	if _, err := svc.GetMonitor(context.Background(), "mon-1"); err == nil {
		t.Error("expected error getting monitor against malformed envelope")
	}
}

// TestLoadBalancerService_EmptyAccountID verifies every method guards
// against a missing account ID before issuing a request.
func TestLoadBalancerService_EmptyAccountID(t *testing.T) {
	t.Parallel()
	_, cf := lbStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected with empty account ID, got %s %s", r.Method, r.URL.Path)
	})
	// The constructor rejects an empty account ID, so exercise the method
	// guards by building the service struct directly (in-package).
	svc := &LoadBalancerService{cf: cf, accountID: ""}

	if _, err := svc.Create(context.Background(), LoadBalancerPoolCreate{Name: "p", Origins: []LoadBalancerOrigin{{Name: "a", Address: "1.1.1.1"}}}); err == nil {
		t.Error("expected error on Create with empty account ID")
	}
	if _, err := svc.List(context.Background()); err == nil {
		t.Error("expected error on List with empty account ID")
	}
	if _, err := svc.Get(context.Background(), "pool-1"); err == nil {
		t.Error("expected error on Get with empty account ID")
	}
	if _, err := svc.Health(context.Background(), "pool-1"); err == nil {
		t.Error("expected error on Health with empty account ID")
	}
	if _, err := svc.Update(context.Background(), "pool-1", LoadBalancerPoolUpdate{Name: "x"}); err == nil {
		t.Error("expected error on Update with empty account ID")
	}
	if err := svc.Delete(context.Background(), "pool-1"); err == nil {
		t.Error("expected error on Delete with empty account ID")
	}
	if _, err := svc.CreateMonitor(context.Background(), LoadBalancerMonitorCreate{Type: "http"}); err == nil {
		t.Error("expected error on CreateMonitor with empty account ID")
	}
	if _, err := svc.ListMonitors(context.Background()); err == nil {
		t.Error("expected error on ListMonitors with empty account ID")
	}
	if _, err := svc.GetMonitor(context.Background(), "mon-1"); err == nil {
		t.Error("expected error on GetMonitor with empty account ID")
	}
	if _, err := svc.UpdateMonitor(context.Background(), "mon-1", LoadBalancerMonitorUpdate{Path: "/x"}); err == nil {
		t.Error("expected error on UpdateMonitor with empty account ID")
	}
	if err := svc.DeleteMonitor(context.Background(), "mon-1"); err == nil {
		t.Error("expected error on DeleteMonitor with empty account ID")
	}
}

// TestLoadBalancerService_FromCredsValidation verifies the credentials
// constructor rejects missing account ID or token offline.
func TestLoadBalancerService_FromCredsValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewLoadBalancerServiceFromCreds("", "token"); err == nil {
		t.Error("expected error for missing account ID")
	}
	if _, err := NewLoadBalancerServiceFromCreds("acct", ""); err == nil {
		t.Error("expected error for missing API token")
	}
}
