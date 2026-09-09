package cosmoflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// redirectNewClient builds a Cloudflare client pointed at the test server,
// mirroring the repo's standard test setup.
func redirectNewClient(t *testing.T, serverURL string) *cloudflare.API {
	t.Helper()
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(serverURL))
	if err != nil {
		t.Fatalf("failed to build cloudflare client: %v", err)
	}
	return cf
}

func TestRedirectService_List(t *testing.T) {
	const zoneID = "zone123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets":
			// Listing returns ruleset summaries with id, phase, name.
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result": []map[string]any{
					{"id": "other-phase", "phase": "http_request_firewall_custom", "name": "WAF"},
					{"id": "redirect-rs", "phase": "http_request_dynamic_redirect", "name": "Redirect Rules"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets/redirect-rs":
			// Fetching the phase ruleset returns its rules.
			enabled := true
			preserve := true
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result": map[string]any{
					"id":    "redirect-rs",
					"phase": "http_request_dynamic_redirect",
					"name":  "Redirect Rules",
					"rules": []map[string]any{
						{
							"id":         "rule-1",
							"expression": `http.request.uri.path eq "/old"`,
							"enabled":    enabled,
							"action":     "redirect",
							"action_parameters": map[string]any{
								"from_value": map[string]any{
									"status_code":           301,
									"preserve_query_string": preserve,
									"target_url": map[string]any{
										"value": "https://example.com/new",
									},
								},
							},
						},
					},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewRedirectService(redirectNewClient(t, server.URL), "account-test")

	rules, err := svc.List(context.Background(), zoneID)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	got := rules[0]
	if got.ID != "rule-1" {
		t.Errorf("expected ID rule-1, got %q", got.ID)
	}
	if got.Destination != "https://example.com/new" {
		t.Errorf("expected destination https://example.com/new, got %q", got.Destination)
	}
	if got.StatusCode != 301 {
		t.Errorf("expected status code 301, got %d", got.StatusCode)
	}
	if !got.PreserveQuery {
		t.Error("expected PreserveQuery to be true")
	}
	if !got.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestRedirectService_List_NoPhase(t *testing.T) {
	const zoneID = "zone123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets" {
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result":   []map[string]any{},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc := NewRedirectService(redirectNewClient(t, server.URL), "account-test")
	rules, err := svc.List(context.Background(), zoneID)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules when no redirect phase exists, got %d", len(rules))
	}
}

func TestRedirectService_List_Validation(t *testing.T) {
	svc := NewRedirectService(redirectNewClient(t, "http://example.invalid"), "account-test")
	if _, err := svc.List(context.Background(), ""); err == nil {
		t.Error("expected validation error for empty zone ID")
	}
}

func TestRedirectService_Create(t *testing.T) {
	const zoneID = "zone123"
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets":
			// No existing redirect phase -> Create path.
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result":   []map[string]any{},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/"+zoneID+"/rulesets":
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &capturedBody); err != nil {
				t.Fatalf("failed to decode create body: %v", err)
			}
			// Echo back a ruleset containing the submitted rule with a server ID.
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result": map[string]any{
					"id":    "redirect-rs",
					"phase": "http_request_dynamic_redirect",
					"name":  "Redirect Rules",
					"rules": []map[string]any{
						{
							"id":         "new-rule-id",
							"expression": `http.request.uri.path eq "/from"`,
							"action":     "redirect",
							"action_parameters": map[string]any{
								"from_value": map[string]any{
									"status_code": 302,
									"target_url":  map[string]any{"value": "https://example.com/to"},
								},
							},
						},
					},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewRedirectService(redirectNewClient(t, server.URL), "account-test")

	out, err := svc.Create(context.Background(), RedirectRuleInput{
		ZoneID:      zoneID,
		When:        `http.request.uri.path eq "/from"`,
		Destination: "https://example.com/to",
		StatusCode:  302,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Assert the request body carried destination + status.
	if capturedBody == nil {
		t.Fatal("create request body was never captured")
	}
	rawRules, ok := capturedBody["rules"].([]any)
	if !ok || len(rawRules) != 1 {
		t.Fatalf("expected 1 rule in request body, got %#v", capturedBody["rules"])
	}
	rule := rawRules[0].(map[string]any)
	if rule["action"] != "redirect" {
		t.Errorf("expected action redirect, got %v", rule["action"])
	}
	ap, ok := rule["action_parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected action_parameters object, got %#v", rule["action_parameters"])
	}
	fv, ok := ap["from_value"].(map[string]any)
	if !ok {
		t.Fatalf("expected from_value object, got %#v", ap["from_value"])
	}
	if sc, _ := fv["status_code"].(float64); int(sc) != 302 {
		t.Errorf("expected status_code 302 in body, got %v", fv["status_code"])
	}
	target, ok := fv["target_url"].(map[string]any)
	if !ok {
		t.Fatalf("expected target_url object, got %#v", fv["target_url"])
	}
	if target["value"] != "https://example.com/to" {
		t.Errorf("expected destination https://example.com/to in body, got %v", target["value"])
	}

	// Assert the returned RedirectRule echoes the created rule.
	if out.ID != "new-rule-id" {
		t.Errorf("expected returned ID new-rule-id, got %q", out.ID)
	}
	if out.Destination != "https://example.com/to" {
		t.Errorf("expected returned destination https://example.com/to, got %q", out.Destination)
	}
	if out.StatusCode != 302 {
		t.Errorf("expected returned status code 302, got %d", out.StatusCode)
	}
}

func TestRedirectService_Create_Validation(t *testing.T) {
	svc := NewRedirectService(redirectNewClient(t, "http://example.invalid"), "account-test")
	if _, err := svc.Create(context.Background(), RedirectRuleInput{ZoneID: "", Destination: "x"}); err == nil {
		t.Error("expected validation error for empty zone ID")
	}
	if _, err := svc.Create(context.Background(), RedirectRuleInput{ZoneID: "z", Destination: ""}); err == nil {
		t.Error("expected validation error for empty destination")
	}
}

func TestRedirectService_Delete(t *testing.T) {
	const zoneID = "zone123"
	var deletePath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets":
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result": []map[string]any{
					{"id": "redirect-rs", "phase": "http_request_dynamic_redirect", "name": "Redirect Rules"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/rulesets/redirect-rs":
			// phaseRuleset fetches the full ruleset after locating the phase.
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"errors":   []any{},
				"messages": []any{},
				"result": map[string]any{
					"id":    "redirect-rs",
					"phase": "http_request_dynamic_redirect",
					"name":  "Redirect Rules",
					"rules": []map[string]any{},
				},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/zones/"+zoneID+"/rulesets/redirect-rs/rules/"):
			deletePath = r.URL.Path
			// Successful rule deletion returns an empty 200/204 body.
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewRedirectService(redirectNewClient(t, server.URL), "account-test")

	if err := svc.Delete(context.Background(), zoneID, "rule-xyz"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	want := "/zones/" + zoneID + "/rulesets/redirect-rs/rules/rule-xyz"
	if deletePath != want {
		t.Errorf("expected DELETE to %q, got %q", want, deletePath)
	}
}

func TestRedirectService_Delete_Validation(t *testing.T) {
	svc := NewRedirectService(redirectNewClient(t, "http://example.invalid"), "account-test")
	if err := svc.Delete(context.Background(), "", "rule"); err == nil {
		t.Error("expected validation error for empty zone ID")
	}
	if err := svc.Delete(context.Background(), "zone", ""); err == nil {
		t.Error("expected validation error for empty rule ID")
	}
}

func TestNewRedirectServiceFromCreds(t *testing.T) {
	if _, err := NewRedirectServiceFromCreds("", "tok"); err == nil {
		t.Error("expected error for empty account ID")
	}
	if _, err := NewRedirectServiceFromCreds("acct", ""); err == nil {
		t.Error("expected error for empty API token")
	}
	svc, err := NewRedirectServiceFromCreds("acct", "tok")
	if err != nil || svc == nil {
		t.Fatalf("expected service, got svc=%v err=%v", svc, err)
	}
}

func TestLegacyForwardingRules(t *testing.T) {
	rules := []*PageRule{
		{
			ID: "pr1", Status: "active", Priority: 1,
			Targets: []PageRuleTarget{{Target: "url", Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/old/*"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/new", "status_code": float64(302)}}},
		},
		{
			ID: "pr2", Status: "disabled", Priority: 2,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/gone"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/"}}},
		},
		{
			ID: "pr3", Status: "active", Priority: 3,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/cache"}}},
			Actions: []PageRuleAction{{ID: "cache_level", Value: map[string]interface{}{"value": "bypass"}}}, // not forwarding → dropped
		},
		{
			ID: "pr4", Status: "active", Priority: 4,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/broken"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"status_code": float64(301)}}}, // no url → dropped
		},
	}

	got := legacyForwardingRules("zone1", rules)
	if len(got) != 2 {
		t.Fatalf("legacyForwardingRules returned %d rules, want 2: %+v", len(got), got)
	}
	first := got[0]
	if first.ZoneID != "zone1" || first.When != "*example.com/old/*" || first.Destination != "https://example.com/new" {
		t.Fatalf("first mapping wrong: %+v", first)
	}
	if first.StatusCode != 302 || !first.Enabled || first.Source != "pagerules" {
		t.Fatalf("first flags wrong: %+v", first)
	}
	second := got[1]
	if second.When != "*example.com/gone" || second.Destination != "https://example.com/" || second.StatusCode != 301 {
		t.Fatalf("second mapping wrong (status default 301 expected): %+v", second)
	}
	if second.Enabled {
		t.Fatal("disabled page rule must map to Enabled=false")
	}
}

func TestLegacyForwardingRulesEmpty(t *testing.T) {
	if got := legacyForwardingRules("z", nil); len(got) != 0 {
		t.Fatalf("nil input must yield empty, got %+v", got)
	}
}
