package r2go2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// corsMockSetup creates a mock Cloudflare API server and CORSService for testing.
func corsMockSetup(handler http.HandlerFunc) (*CORSService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewCORSService(cf, "zone-test-123")
	return svc, server
}

func corsWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// buildCORSRulesetResponse produces a Cloudflare API envelope with rules.
func buildCORSRulesetResponse(rules []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"result": map[string]interface{}{
			"id":    "ruleset-abc",
			"name":  "default",
			"phase": corsPhase,
			"rules": rules,
		},
		"success":  true,
		"errors":   []interface{}{},
		"messages": []interface{}{},
	}
}

// --- Constructor tests ---

func TestNewCORSServiceValidation(t *testing.T) {
	_, err := NewCORSService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewCORSService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}
}

func TestNewCORSServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewCORSService(cf, "zone123")
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

func TestNewCORSServiceFromCredsValidation(t *testing.T) {
	_, err := NewCORSServiceFromCreds("", "test-token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewCORSServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

// --- GetCORSRules tests ---

func TestGetCORSRules_EmptyZone404(t *testing.T) {
	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		corsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 10001, "message": "not found"}},
		})
	})
	defer server.Close()

	rules, err := svc.GetCORSRules(context.Background())
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty rules, got %d", len(rules))
	}
}

func TestGetCORSRules_ParsesSingleRule(t *testing.T) {
	ruleJSON := map[string]interface{}{
		"id":          "rule-001",
		"action":      "rewrite",
		"expression":  "true",
		"description": "cosmoflare-cors",
		"enabled":     true,
		"action_parameters": map[string]interface{}{
			"headers": map[string]interface{}{
				"Access-Control-Allow-Origin": map[string]interface{}{
					"operation": "set", "value": "*",
				},
				"Access-Control-Allow-Methods": map[string]interface{}{
					"operation": "set", "value": "GET, POST, OPTIONS",
				},
				"Access-Control-Allow-Headers": map[string]interface{}{
					"operation": "set", "value": "Content-Type, Authorization",
				},
				"Access-Control-Max-Age": map[string]interface{}{
					"operation": "set", "value": "86400",
				},
			},
		},
	}

	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{ruleJSON}))
	})
	defer server.Close()

	rules, err := svc.GetCORSRules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	r := rules[0]
	if r.Name != "cosmoflare-cors" {
		t.Errorf("Name = %q, want %q", r.Name, "cosmoflare-cors")
	}
	if !r.Enabled {
		t.Error("expected Enabled=true")
	}
	if len(r.AllowOrigins) != 1 || r.AllowOrigins[0] != "*" {
		t.Errorf("AllowOrigins = %v, want [*]", r.AllowOrigins)
	}
	if len(r.AllowMethods) != 3 {
		t.Errorf("AllowMethods = %v, want 3 items", r.AllowMethods)
	}
	if r.MaxAge != 86400 {
		t.Errorf("MaxAge = %d, want 86400", r.MaxAge)
	}
}

// --- SetCORSHeaders tests ---

func TestSetCORSHeaders_CreatesOnEmptyZone(t *testing.T) {
	callCount := 0
	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == http.MethodGet {
			// Simulate 404 — no ruleset yet
			w.WriteHeader(http.StatusNotFound)
			corsWriteJSON(w, map[string]interface{}{
				"result":  nil,
				"success": false,
				"errors":  []map[string]interface{}{{"code": 10001, "message": "not found"}},
			})
			return
		}
		// PUT — return the created ruleset
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{
			{
				"id":          "new-rule",
				"action":      "rewrite",
				"expression":  "true",
				"description": "cosmoflare-cors",
				"enabled":     true,
				"action_parameters": map[string]interface{}{
					"headers": map[string]interface{}{
						"Access-Control-Allow-Origin": map[string]interface{}{
							"operation": "set", "value": "*",
						},
					},
				},
			},
		}))
	})
	defer server.Close()

	rule, err := svc.SetCORSHeaders(context.Background(),
		WithCORSOrigins("*"),
		WithCORSMethods("GET", "POST", "OPTIONS"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule == nil {
		t.Fatal("expected non-nil rule")
	}
	if rule.Name != "cosmoflare-cors" {
		t.Errorf("Name = %q, want %q", rule.Name, "cosmoflare-cors")
	}
}

func TestSetCORSHeaders_ReplacesExistingRule(t *testing.T) {
	existingRules := []map[string]interface{}{
		{
			"id":          "rule-old",
			"action":      "rewrite",
			"expression":  "true",
			"description": "cosmoflare-cors",
			"enabled":     true,
			"action_parameters": map[string]interface{}{
				"headers": map[string]interface{}{
					"Access-Control-Allow-Origin": map[string]interface{}{
						"operation": "set", "value": "https://old.example.com",
					},
				},
			},
		},
		{
			"id":          "rule-other",
			"action":      "rewrite",
			"expression":  "true",
			"description": "some-other-rule",
			"enabled":     true,
			"action_parameters": map[string]interface{}{
				"headers": map[string]interface{}{
					"X-Custom-Header": map[string]interface{}{
						"operation": "set", "value": "custom-value",
					},
				},
			},
		},
	}

	var capturedBody []byte
	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			corsWriteJSON(w, buildCORSRulesetResponse(existingRules))
			return
		}
		// Capture PUT body to verify rule replacement
		capturedBody = make([]byte, r.ContentLength)
		r.Body.Read(capturedBody)

		// Return updated ruleset
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{
			{
				"id":          "rule-old",
				"action":      "rewrite",
				"expression":  "true",
				"description": "cosmoflare-cors",
				"enabled":     true,
				"action_parameters": map[string]interface{}{
					"headers": map[string]interface{}{
						"Access-Control-Allow-Origin": map[string]interface{}{
							"operation": "set", "value": "https://new.example.com",
						},
					},
				},
			},
			existingRules[1],
		}))
	})
	defer server.Close()

	rule, err := svc.SetCORSHeaders(context.Background(),
		WithCORSOrigins("https://new.example.com"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule == nil {
		t.Fatal("expected non-nil rule")
	}
	// Verify the response shows the new origin
	if len(rule.AllowOrigins) == 0 || rule.AllowOrigins[0] != "https://new.example.com" {
		t.Errorf("AllowOrigins = %v, want [https://new.example.com]", rule.AllowOrigins)
	}
}

func TestSetCORSHeaders_AppendsNewNamedRule(t *testing.T) {
	existingRules := []map[string]interface{}{
		{
			"id":          "rule-existing",
			"action":      "rewrite",
			"expression":  "true",
			"description": "other-cors",
			"enabled":     true,
			"action_parameters": map[string]interface{}{
				"headers": map[string]interface{}{
					"Access-Control-Allow-Origin": map[string]interface{}{
						"operation": "set", "value": "https://other.example.com",
					},
				},
			},
		},
	}

	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			corsWriteJSON(w, buildCORSRulesetResponse(existingRules))
			return
		}
		// Return ruleset with both rules
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{
			existingRules[0],
			{
				"id":          "rule-new",
				"action":      "rewrite",
				"expression":  "true",
				"description": "my-cors",
				"enabled":     true,
				"action_parameters": map[string]interface{}{
					"headers": map[string]interface{}{
						"Access-Control-Allow-Origin": map[string]interface{}{
							"operation": "set", "value": "*",
						},
					},
				},
			},
		}))
	})
	defer server.Close()

	rule, err := svc.SetCORSHeaders(context.Background(),
		WithCORSName("my-cors"),
		WithCORSOrigins("*"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Name != "my-cors" {
		t.Errorf("Name = %q, want %q", rule.Name, "my-cors")
	}
}

func TestSetCORSHeaders_RejectsWildcardWithCredentials(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCORSService(cf, "zone123")

	_, err := svc.SetCORSHeaders(context.Background(),
		WithCORSOrigins("*"),
		WithCORSCredentials(true),
	)
	if err == nil {
		t.Error("expected error when combining wildcard origin with credentials")
	}
}

func TestSetCORSHeaders_RequiresOrigins(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewCORSService(cf, "zone123")

	_, err := svc.SetCORSHeaders(context.Background())
	if err == nil {
		t.Error("expected error when no origins provided")
	}
}

// --- RemoveCORSRule tests ---

func TestRemoveCORSRule_RemovesMatchingRule(t *testing.T) {
	existingRules := []map[string]interface{}{
		{
			"id":          "rule-001",
			"action":      "rewrite",
			"expression":  "true",
			"description": "cosmoflare-cors",
			"enabled":     true,
			"action_parameters": map[string]interface{}{
				"headers": map[string]interface{}{
					"Access-Control-Allow-Origin": map[string]interface{}{
						"operation": "set", "value": "*",
					},
				},
			},
		},
	}

	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			corsWriteJSON(w, buildCORSRulesetResponse(existingRules))
			return
		}
		// PUT with empty rules
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{}))
	})
	defer server.Close()

	err := svc.RemoveCORSRule(context.Background(), "cosmoflare-cors")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveCORSRule_NotFoundError(t *testing.T) {
	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Empty ruleset
			corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{}))
			return
		}
		corsWriteJSON(w, buildCORSRulesetResponse([]map[string]interface{}{}))
	})
	defer server.Close()

	err := svc.RemoveCORSRule(context.Background(), "cosmoflare-cors")
	if err == nil {
		t.Fatal("expected ErrCORSRuleNotFound")
	}
	if err != ErrCORSRuleNotFound {
		t.Errorf("expected ErrCORSRuleNotFound, got: %v", err)
	}
}

func TestRemoveCORSRule_404RulesetReturnsNotFound(t *testing.T) {
	svc, server := corsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		corsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 10001, "message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.RemoveCORSRule(context.Background(), "cosmoflare-cors")
	if err != ErrCORSRuleNotFound {
		t.Errorf("expected ErrCORSRuleNotFound for 404 ruleset, got: %v", err)
	}
}

// --- parseCORSRule unit tests ---

func TestParseCORSRule_NilActionParameters(t *testing.T) {
	r := cloudflare.RulesetRule{Description: "test", ActionParameters: nil}
	_, ok := parseCORSRule(r)
	if ok {
		t.Error("expected ok=false for nil ActionParameters")
	}
}

func TestParseCORSRule_NoCORSHeaders(t *testing.T) {
	enabled := true
	r := cloudflare.RulesetRule{
		Description: "not-cors",
		Enabled:     &enabled,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{
				"X-Custom": {Operation: "set", Value: "val"},
			},
		},
	}
	_, ok := parseCORSRule(r)
	if ok {
		t.Error("expected ok=false for rule with no CORS headers")
	}
}

func TestParseCORSRule_AllFields(t *testing.T) {
	enabled := true
	r := cloudflare.RulesetRule{
		ID:          "rule-xyz",
		Description: "my-cors",
		Expression:  "true",
		Enabled:     &enabled,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{
				"Access-Control-Allow-Origin":      {Operation: "set", Value: "https://a.com"},
				"Access-Control-Allow-Methods":     {Operation: "set", Value: "GET, POST"},
				"Access-Control-Allow-Headers":     {Operation: "set", Value: "Content-Type"},
				"Access-Control-Max-Age":           {Operation: "set", Value: "3600"},
				"Access-Control-Allow-Credentials": {Operation: "set", Value: "true"},
			},
		},
	}

	cr, ok := parseCORSRule(r)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cr.ID != "rule-xyz" {
		t.Errorf("ID = %q, want %q", cr.ID, "rule-xyz")
	}
	if cr.MaxAge != 3600 {
		t.Errorf("MaxAge = %d, want 3600", cr.MaxAge)
	}
	if !cr.AllowCredentials {
		t.Error("expected AllowCredentials=true")
	}
	if len(cr.AllowOrigins) != 1 || cr.AllowOrigins[0] != "https://a.com" {
		t.Errorf("AllowOrigins = %v", cr.AllowOrigins)
	}
}

// --- splitTrimmed tests ---

func TestSplitTrimmed(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"GET, POST, OPTIONS", []string{"GET", "POST", "OPTIONS"}},
		{"*", []string{"*"}},
		{"", nil},
		{"Content-Type,Authorization", []string{"Content-Type", "Authorization"}},
	}
	for _, tc := range cases {
		got := splitTrimmed(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitTrimmed(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitTrimmed(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}
