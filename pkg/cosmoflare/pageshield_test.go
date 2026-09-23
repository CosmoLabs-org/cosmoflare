package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// newPageShieldTestService wires a PageShieldService at zoneID to an
// httptest server whose handler is closed over by the caller's test.
func newPageShieldTestService(t *testing.T, zoneID string, handler http.HandlerFunc) *PageShieldService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	svc, err := NewPageShieldService(cf, zoneID)
	if err != nil {
		t.Fatalf("failed to build service: %v", err)
	}
	return svc
}

// writeCloudflareResult writes a standard Cloudflare API success envelope
// with the given result payload.
func writeCloudflareResult(t *testing.T, w http.ResponseWriter, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"errors":  []any{},
		"result":  result,
	})
}

// TestPageShieldService_ListConnections verifies ListConnections issues a
// GET to the zone's page_shield connections endpoint and maps the response
// entries, including the malicious-domain flag.
func TestPageShieldService_ListConnections(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-test-123"
	malicious := true

	svc := newPageShieldTestService(t, zoneID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/zones/"+zoneID+"/page_shield/connections"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		writeCloudflareResult(t, w, []map[string]any{
			{
				"id":                        "conn-1",
				"url":                       "https://cdn.example.com/analytics.js",
				"host":                      "cdn.example.com",
				"first_seen_at":             "2026-01-02T03:04:05Z",
				"last_seen_at":              "2026-02-03T04:05:06Z",
				"page_urls":                 []string{"https://www.example.com/"},
				"domain_reported_malicious": malicious,
			},
		})
	})

	connections, err := svc.ListConnections(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}
	conn := connections[0]
	if conn.ID != "conn-1" || conn.Host != "cdn.example.com" {
		t.Errorf("unexpected mapping: %+v", conn)
	}
	if conn.DomainReportedMalicious == nil || !*conn.DomainReportedMalicious {
		t.Errorf("expected DomainReportedMalicious=true, got %+v", conn.DomainReportedMalicious)
	}
}

// TestPageShieldService_ListScripts verifies ListScripts issues a GET to
// the zone's page_shield scripts endpoint and maps response entries
// including the JS integrity score.
func TestPageShieldService_ListScripts(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-test-123"

	svc := newPageShieldTestService(t, zoneID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/zones/"+zoneID+"/page_shield/scripts"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		writeCloudflareResult(t, w, []map[string]any{
			{
				"id":                 "script-1",
				"url":                "https://cdn.example.com/lib.js",
				"host":               "cdn.example.com",
				"hash":               "abc123",
				"js_integrity_score": 42,
				"fetched_at":         "2026-02-03T04:05:06Z",
			},
		})
	})

	scripts, err := svc.ListScripts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}
	script := scripts[0]
	if script.ID != "script-1" || script.Hash != "abc123" || script.JSIntegrityScore != 42 {
		t.Errorf("unexpected mapping: %+v", script)
	}
}

// pageShieldPolicyStub is a stateful page_shield policies stub: create
// appends, patch merges over the stored state and returns the WRITTEN
// state, get/delete operate on the store.
type pageShieldPolicyStub struct {
	mu       sync.Mutex
	policies map[string]map[string]any
	nextID   int
}

func (s *pageShieldPolicyStub) handle(t *testing.T) http.HandlerFunc {
	const zoneID = "zone-test-123"
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		path := r.URL.Path
		base := "/zones/" + zoneID + "/page_shield/policies"

		switch {
		case path == base && r.Method == http.MethodGet:
			results := make([]map[string]any, 0, len(s.policies))
			for _, p := range s.policies {
				results = append(results, p)
			}
			writeCloudflareResult(t, w, results)

		case path == base && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("bad create body: %v", err)
			}
			s.nextID++
			body["id"] = "policy-" + itoaForPageShield(s.nextID)
			s.policies[body["id"].(string)] = body
			// cloudflare-go decodes create/get/update policy responses raw
			// (no result envelope), so the policy IS the body here.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(body)

		case strings.HasPrefix(path, base+"/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(path, base+"/")
			p, ok := s.policies[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)

		case strings.HasPrefix(path, base+"/") && r.Method == http.MethodPut:
			id := strings.TrimPrefix(path, base+"/")
			p, ok := s.policies[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("bad patch body: %v", err)
			}
			for k, v := range body {
				p[k] = v
			}
			// Re-fetch semantics: the response after a write reflects the
			// WRITTEN state.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)

		case strings.HasPrefix(path, base+"/") && r.Method == http.MethodDelete:
			id := strings.TrimPrefix(path, base+"/")
			delete(s.policies, id)
			writeCloudflareResult(t, w, nil)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func itoaForPageShield(n int) string { return strconv.Itoa(n) }

// TestPageShieldService_PolicyLifecycle drives create, get, update and
// delete through the stateful stub, verifying the update merges only the
// changed fields (expression preserved, action replaced) and that the
// written state is what comes back.
func TestPageShieldService_PolicyLifecycle(t *testing.T) {
	t.Parallel()
	stub := &pageShieldPolicyStub{policies: map[string]map[string]any{}}
	svc := newPageShieldTestService(t, "zone-test-123", stub.handle(t))
	ctx := context.Background()

	created, err := svc.CreatePolicy(ctx,
		WithPageShieldPolicyExpression(`http.host eq "bad.example.com"`),
		WithPageShieldPolicyAction("log"),
		WithPageShieldPolicyDescription("watch bad host"),
		WithPageShieldPolicyEnabled(true),
	)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if created.ID == "" || created.Expression != `http.host eq "bad.example.com"` || created.Action != "log" {
		t.Fatalf("unexpected created policy: %+v", created)
	}
	if created.Enabled == nil || !*created.Enabled {
		t.Errorf("expected Enabled=true, got %+v", created.Enabled)
	}

	fetched, err := svc.GetPolicy(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if fetched.Action != "log" || fetched.Description != "watch bad host" {
		t.Fatalf("unexpected fetched policy: %+v", fetched)
	}

	// Update only the action; everything else must survive the merge.
	updated, err := svc.UpdatePolicy(ctx, created.ID, WithPageShieldPolicyAction("block"))
	if err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}
	if updated.Action != "block" {
		t.Errorf("expected Action=block after update, got %q", updated.Action)
	}
	if updated.Expression != created.Expression {
		t.Errorf("expected Expression preserved (%q), got %q", created.Expression, updated.Expression)
	}

	if err := svc.DeletePolicy(ctx, created.ID); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	policies, err := svc.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("expected no policies after delete, got %d", len(policies))
	}
}

// TestPageShieldService_Validation verifies constructor and method
// argument guards fail before any request is issued.
func TestPageShieldService_Validation(t *testing.T) {
	t.Parallel()

	if _, err := NewPageShieldService(nil, "zone-1"); err == nil {
		t.Error("expected error for nil API client")
	}
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewPageShieldService(cf, "zone-1")
	if err != nil {
		t.Fatalf("NewPageShieldService: %v", err)
	}
	ctx := context.Background()

	if _, err := svc.GetPolicy(ctx, ""); err == nil {
		t.Error("expected error for empty policy ID on GetPolicy")
	}
	if _, err := svc.CreatePolicy(ctx, WithPageShieldPolicyAction("block")); err == nil {
		t.Error("expected error for missing expression on CreatePolicy")
	}
	if _, err := svc.CreatePolicy(ctx, WithPageShieldPolicyExpression("true")); err == nil {
		t.Error("expected error for missing action on CreatePolicy")
	}
	if _, err := svc.UpdatePolicy(ctx, "policy-1"); err == nil {
		t.Error("expected error for no update options on UpdatePolicy")
	}
	if _, err := svc.UpdatePolicy(ctx, "", WithPageShieldPolicyAction("log")); err == nil {
		t.Error("expected error for empty policy ID on UpdatePolicy")
	}
	if err := svc.DeletePolicy(ctx, ""); err == nil {
		t.Error("expected error for empty policy ID on DeletePolicy")
	}
	if _, err := NewPageShieldServiceFromCreds("zone-1", ""); err == nil {
		t.Error("expected error for empty API token")
	}
}
