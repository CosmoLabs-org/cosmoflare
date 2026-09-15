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

func workerRouteMockSetup(handler http.HandlerFunc) (*WorkerService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWorkerService(cf, "acct-worker-routes")
	return svc, server
}

func workerRouteWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestWorkerRouteType(t *testing.T) {
	r := WorkerRoute{ID: "route-1", Pattern: "example.com/*", Script: "my-worker"}
	if r.ID != "route-1" || r.Pattern != "example.com/*" || r.Script != "my-worker" {
		t.Errorf("unexpected route: %+v", r)
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	for _, want := range []string{`"id":"route-1"`, `"pattern":"example.com/*"`, `"script":"my-worker"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("json %s missing %s", data, want)
		}
	}
}

// --- validation ---

func TestWorkerRouteListValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.RouteList(nil, "")
	if err == nil {
		t.Fatal("expected error when zone ID is empty")
	}
	if !strings.Contains(err.Error(), "zone ID is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorkerRouteCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	cases := []struct {
		zoneID, pattern, script, want string
	}{
		{"", "example.com/*", "my-worker", "zone ID is required"},
		{"zone-1", "", "my-worker", "pattern is required"},
		{"zone-1", "example.com/*", "", "script is required"},
	}
	for _, tc := range cases {
		_, err := svc.RouteCreate(nil, tc.zoneID, tc.pattern, tc.script)
		if err == nil {
			t.Fatalf("expected error for %+v", tc)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("error %q, want substring %q", err.Error(), tc.want)
		}
	}
}

func TestWorkerRouteUpdateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	cases := []struct {
		zoneID, routeID, pattern, script, want string
	}{
		{"", "route-1", "example.com/*", "my-worker", "zone ID is required"},
		{"zone-1", "", "example.com/*", "my-worker", "route ID is required"},
		{"zone-1", "route-1", "", "my-worker", "pattern is required"},
		{"zone-1", "route-1", "example.com/*", "", "script is required"},
	}
	for _, tc := range cases {
		_, err := svc.RouteUpdate(nil, tc.zoneID, tc.routeID, tc.pattern, tc.script)
		if err == nil {
			t.Fatalf("expected error for %+v", tc)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("error %q, want substring %q", err.Error(), tc.want)
		}
	}
}

func TestWorkerRouteDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	if err := svc.RouteDelete(nil, "", "route-1"); err == nil {
		t.Error("expected error when zone ID is empty")
	} else if !strings.Contains(err.Error(), "zone ID is required") {
		t.Errorf("unexpected error: %v", err)
	}

	if err := svc.RouteDelete(nil, "zone-1", ""); err == nil {
		t.Error("expected error when route ID is empty")
	} else if !strings.Contains(err.Error(), "route ID is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- httptest happy paths ---

func TestWorkerRouteListWithMock(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/zones/zone-1/workers/routes") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "route-1", "pattern": "example.com/*", "script": "worker-a"},
				{"id": "route-2", "pattern": "api.example.com/*", "script": "worker-b"},
			},
		})
	})
	defer server.Close()

	routes, err := svc.RouteList(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].ID != "route-1" || routes[0].Pattern != "example.com/*" || routes[0].Script != "worker-a" {
		t.Errorf("unexpected first route: %+v", routes[0])
	}
	if routes[1].Script != "worker-b" {
		t.Errorf("unexpected second route script: %s", routes[1].Script)
	}
}

func TestWorkerRouteListEmptyWithMock(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []interface{}{},
		})
	})
	defer server.Close()

	routes, err := svc.RouteList(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestWorkerRouteCreateWithMock(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":      "route-new",
				"pattern": "example.com/*",
				"script":  "my-worker",
			},
		})
	})
	defer server.Close()

	route, err := svc.RouteCreate(context.Background(), "zone-1", "example.com/*", "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.ID != "route-new" {
		t.Errorf("expected ID=route-new, got %s", route.ID)
	}
	if route.Pattern != "example.com/*" {
		t.Errorf("expected Pattern=example.com/*, got %s", route.Pattern)
	}
	if route.Script != "my-worker" {
		t.Errorf("expected Script=my-worker, got %s", route.Script)
	}
}

func TestWorkerRouteUpdateWithMock(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/zones/zone-1/workers/routes/route-1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":      "route-1",
				"pattern": "api.example.com/*",
				"script":  "other-worker",
			},
		})
	})
	defer server.Close()

	route, err := svc.RouteUpdate(context.Background(), "zone-1", "route-1", "api.example.com/*", "other-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.ID != "route-1" {
		t.Errorf("expected ID=route-1, got %s", route.ID)
	}
	if route.Pattern != "api.example.com/*" {
		t.Errorf("expected Pattern=api.example.com/*, got %s", route.Pattern)
	}
	if route.Script != "other-worker" {
		t.Errorf("expected Script=other-worker, got %s", route.Script)
	}
}

func TestWorkerRouteDeleteWithMock(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/zones/zone-1/workers/routes/route-1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "route-1"},
		})
	})
	defer server.Close()

	if err := svc.RouteDelete(context.Background(), "zone-1", "route-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- API error ---

func TestWorkerRouteListAPIError(t *testing.T) {
	svc, server := workerRouteMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerRouteWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1003, "message": "invalid or missing zone"}},
		})
	})
	defer server.Close()

	_, err := svc.RouteList(context.Background(), "zone-bad")
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to list worker routes") {
		t.Errorf("unexpected error message: %v", err)
	}
}
