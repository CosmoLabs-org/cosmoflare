package cosmoflare

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

// --- defaultCORSConfig tests ---

func TestDefaultCORSConfig_Defaults(t *testing.T) {
	cfg := defaultCORSConfig()
	if cfg.name != CORSDefaultRuleName {
		t.Errorf("name = %q, want %q", cfg.name, CORSDefaultRuleName)
	}
	if cfg.expression != corsDefaultExpr {
		t.Errorf("expression = %q, want %q", cfg.expression, corsDefaultExpr)
	}
	if cfg.maxAge != 86400 {
		t.Errorf("maxAge = %d, want 86400", cfg.maxAge)
	}
	if cfg.allowCredentials {
		t.Error("allowCredentials should default to false")
	}
	if len(cfg.allowOrigins) != 0 {
		t.Errorf("allowOrigins should be empty by default, got %v", cfg.allowOrigins)
	}
	if len(cfg.allowMethods) == 0 {
		t.Error("allowMethods should have default values")
	}
	if len(cfg.allowHeaders) == 0 {
		t.Error("allowHeaders should have default values")
	}
}

func TestDefaultCORSConfig_DefaultMethods(t *testing.T) {
	cfg := defaultCORSConfig()
	wantMethods := []string{"GET", "POST", "OPTIONS"}
	if len(cfg.allowMethods) != len(wantMethods) {
		t.Fatalf("allowMethods len = %d, want %d", len(cfg.allowMethods), len(wantMethods))
	}
	for i, m := range wantMethods {
		if cfg.allowMethods[i] != m {
			t.Errorf("allowMethods[%d] = %q, want %q", i, cfg.allowMethods[i], m)
		}
	}
}

func TestDefaultCORSConfig_DefaultHeaders(t *testing.T) {
	cfg := defaultCORSConfig()
	wantHeaders := []string{"Content-Type", "Authorization"}
	if len(cfg.allowHeaders) != len(wantHeaders) {
		t.Fatalf("allowHeaders len = %d, want %d", len(cfg.allowHeaders), len(wantHeaders))
	}
	for i, h := range wantHeaders {
		if cfg.allowHeaders[i] != h {
			t.Errorf("allowHeaders[%d] = %q, want %q", i, cfg.allowHeaders[i], h)
		}
	}
}

// --- CORSOption function tests ---

func TestWithCORSName(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSName("my-cors-rule")(cfg)
	if cfg.name != "my-cors-rule" {
		t.Errorf("name = %q, want %q", cfg.name, "my-cors-rule")
	}
}

func TestWithCORSOrigins_Single(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSOrigins("https://example.com")(cfg)
	if len(cfg.allowOrigins) != 1 || cfg.allowOrigins[0] != "https://example.com" {
		t.Errorf("allowOrigins = %v, want [https://example.com]", cfg.allowOrigins)
	}
}

func TestWithCORSOrigins_Multiple(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSOrigins("https://a.com", "https://b.com", "https://c.com")(cfg)
	if len(cfg.allowOrigins) != 3 {
		t.Fatalf("allowOrigins len = %d, want 3", len(cfg.allowOrigins))
	}
	if cfg.allowOrigins[1] != "https://b.com" {
		t.Errorf("allowOrigins[1] = %q, want https://b.com", cfg.allowOrigins[1])
	}
}

func TestWithCORSMethods(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSMethods("PUT", "DELETE")(cfg)
	if len(cfg.allowMethods) != 2 {
		t.Fatalf("allowMethods len = %d, want 2", len(cfg.allowMethods))
	}
	if cfg.allowMethods[0] != "PUT" || cfg.allowMethods[1] != "DELETE" {
		t.Errorf("allowMethods = %v, want [PUT DELETE]", cfg.allowMethods)
	}
}

func TestWithCORSHeaders(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSHeaders("X-Custom-Header", "X-Request-ID")(cfg)
	if len(cfg.allowHeaders) != 2 {
		t.Fatalf("allowHeaders len = %d, want 2", len(cfg.allowHeaders))
	}
	if cfg.allowHeaders[0] != "X-Custom-Header" {
		t.Errorf("allowHeaders[0] = %q, want X-Custom-Header", cfg.allowHeaders[0])
	}
}

func TestWithCORSMaxAge(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSMaxAge(3600)(cfg)
	if cfg.maxAge != 3600 {
		t.Errorf("maxAge = %d, want 3600", cfg.maxAge)
	}
}

func TestWithCORSMaxAge_Zero(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSMaxAge(0)(cfg)
	if cfg.maxAge != 0 {
		t.Errorf("maxAge = %d, want 0", cfg.maxAge)
	}
}

func TestWithCORSCredentials_True(t *testing.T) {
	cfg := defaultCORSConfig()
	WithCORSCredentials(true)(cfg)
	if !cfg.allowCredentials {
		t.Error("allowCredentials should be true")
	}
}

func TestWithCORSCredentials_False(t *testing.T) {
	cfg := defaultCORSConfig()
	cfg.allowCredentials = true
	WithCORSCredentials(false)(cfg)
	if cfg.allowCredentials {
		t.Error("allowCredentials should be false after WithCORSCredentials(false)")
	}
}

func TestWithCORSExpression(t *testing.T) {
	cfg := defaultCORSConfig()
	expr := `http.request.uri.path matches "^/api/.*"`
	WithCORSExpression(expr)(cfg)
	if cfg.expression != expr {
		t.Errorf("expression = %q, want %q", cfg.expression, expr)
	}
}

func TestCORSOptions_Composable(t *testing.T) {
	cfg := defaultCORSConfig()
	opts := []CORSOption{
		WithCORSName("composed-rule"),
		WithCORSOrigins("https://x.com"),
		WithCORSMethods("GET"),
		WithCORSHeaders("X-Foo"),
		WithCORSMaxAge(7200),
		WithCORSCredentials(true),
		WithCORSExpression("true"),
	}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.name != "composed-rule" {
		t.Errorf("name = %q", cfg.name)
	}
	if len(cfg.allowOrigins) != 1 || cfg.allowOrigins[0] != "https://x.com" {
		t.Errorf("allowOrigins = %v", cfg.allowOrigins)
	}
	if cfg.maxAge != 7200 {
		t.Errorf("maxAge = %d", cfg.maxAge)
	}
	if !cfg.allowCredentials {
		t.Error("allowCredentials should be true")
	}
}

// --- CORSRule struct field tests ---

func TestCORSRule_Fields(t *testing.T) {
	rule := CORSRule{
		ID:               "rule-id-001",
		Name:             "test-rule",
		Enabled:          true,
		Expression:       "true",
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		MaxAge:           3600,
		AllowCredentials: false,
	}
	if rule.ID != "rule-id-001" {
		t.Errorf("ID = %q", rule.ID)
	}
	if rule.Name != "test-rule" {
		t.Errorf("Name = %q", rule.Name)
	}
	if !rule.Enabled {
		t.Error("Enabled should be true")
	}
	if rule.MaxAge != 3600 {
		t.Errorf("MaxAge = %d", rule.MaxAge)
	}
	if rule.AllowCredentials {
		t.Error("AllowCredentials should be false")
	}
}

func TestCORSRule_Credentials(t *testing.T) {
	rule := CORSRule{AllowCredentials: true}
	if !rule.AllowCredentials {
		t.Error("AllowCredentials should be true")
	}
}

// --- isCORSHeader tests ---

func TestIsCORSHeader_KnownHeaders(t *testing.T) {
	known := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
		"Access-Control-Allow-Credentials",
	}
	for _, h := range known {
		if !isCORSHeader(h) {
			t.Errorf("isCORSHeader(%q) = false, want true", h)
		}
	}
}

func TestIsCORSHeader_CaseInsensitive(t *testing.T) {
	if !isCORSHeader("access-control-allow-origin") {
		t.Error("isCORSHeader should be case-insensitive")
	}
	if !isCORSHeader("ACCESS-CONTROL-ALLOW-METHODS") {
		t.Error("isCORSHeader should be case-insensitive for uppercase")
	}
}

func TestIsCORSHeader_UnknownHeaders(t *testing.T) {
	unknown := []string{"Content-Type", "Authorization", "X-Custom-Header", ""}
	for _, h := range unknown {
		if isCORSHeader(h) {
			t.Errorf("isCORSHeader(%q) = true, want false", h)
		}
	}
}

// --- buildRulesetRule tests ---

func TestBuildRulesetRule_BasicFields(t *testing.T) {
	cfg := &corsConfig{
		name:         "test-rule",
		expression:   "true",
		allowOrigins: []string{"https://example.com"},
		allowMethods: []string{"GET", "POST"},
		allowHeaders: []string{"Content-Type"},
		maxAge:       3600,
	}
	rule := buildRulesetRule(cfg)
	if rule.Description != "test-rule" {
		t.Errorf("Description = %q, want test-rule", rule.Description)
	}
	if rule.Expression != "true" {
		t.Errorf("Expression = %q, want true", rule.Expression)
	}
	if rule.Enabled == nil || !*rule.Enabled {
		t.Error("Enabled should be true")
	}
	if rule.Action != string(cloudflare.RulesetRuleActionRewrite) {
		t.Errorf("Action = %q, want rewrite", rule.Action)
	}
}

func TestBuildRulesetRule_OriginHeader(t *testing.T) {
	cfg := &corsConfig{
		name:         "rule",
		expression:   "true",
		allowOrigins: []string{"https://a.com", "https://b.com"},
	}
	rule := buildRulesetRule(cfg)
	h, ok := rule.ActionParameters.Headers["Access-Control-Allow-Origin"]
	if !ok {
		t.Fatal("Access-Control-Allow-Origin header missing")
	}
	if h.Operation != "set" {
		t.Errorf("Operation = %q, want set", h.Operation)
	}
	if h.Value != "https://a.com, https://b.com" {
		t.Errorf("Value = %q, want 'https://a.com, https://b.com'", h.Value)
	}
}

func TestBuildRulesetRule_MaxAgeZeroOmitted(t *testing.T) {
	cfg := &corsConfig{
		name:         "rule",
		expression:   "true",
		allowOrigins: []string{"*"},
		maxAge:       0,
	}
	rule := buildRulesetRule(cfg)
	if _, ok := rule.ActionParameters.Headers["Access-Control-Max-Age"]; ok {
		t.Error("Access-Control-Max-Age should be absent when maxAge is 0")
	}
}

func TestBuildRulesetRule_CredentialsHeader(t *testing.T) {
	cfg := &corsConfig{
		name:             "rule",
		expression:       "true",
		allowOrigins:     []string{"https://example.com"},
		allowCredentials: true,
	}
	rule := buildRulesetRule(cfg)
	h, ok := rule.ActionParameters.Headers["Access-Control-Allow-Credentials"]
	if !ok {
		t.Fatal("Access-Control-Allow-Credentials header missing")
	}
	if h.Value != "true" {
		t.Errorf("Value = %q, want true", h.Value)
	}
}

func TestBuildRulesetRule_NoCredentialsHeader(t *testing.T) {
	cfg := &corsConfig{
		name:             "rule",
		expression:       "true",
		allowOrigins:     []string{"*"},
		allowCredentials: false,
	}
	rule := buildRulesetRule(cfg)
	if _, ok := rule.ActionParameters.Headers["Access-Control-Allow-Credentials"]; ok {
		t.Error("Access-Control-Allow-Credentials should be absent when allowCredentials is false")
	}
}

// --- upsertRule tests ---

func TestUpsertRule_AppendsWhenEmpty(t *testing.T) {
	newRule := cloudflare.RulesetRule{Description: "new-rule"}
	result := upsertRule(nil, newRule, "new-rule")
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Description != "new-rule" {
		t.Errorf("Description = %q", result[0].Description)
	}
}

func TestUpsertRule_AppendsWhenNoMatch(t *testing.T) {
	existing := []cloudflare.RulesetRule{
		{Description: "other-rule"},
	}
	newRule := cloudflare.RulesetRule{Description: "new-rule"}
	result := upsertRule(existing, newRule, "new-rule")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[1].Description != "new-rule" {
		t.Errorf("result[1].Description = %q", result[1].Description)
	}
}

func TestUpsertRule_ReplacesExisting(t *testing.T) {
	existing := []cloudflare.RulesetRule{
		{Description: "keep-rule"},
		{Description: "target-rule", Expression: "old-expr"},
	}
	newRule := cloudflare.RulesetRule{Description: "target-rule", Expression: "new-expr"}
	result := upsertRule(existing, newRule, "target-rule")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2 (no append on replace)", len(result))
	}
	if result[1].Expression != "new-expr" {
		t.Errorf("Expression after replace = %q, want new-expr", result[1].Expression)
	}
}

func TestUpsertRule_PreservesOtherRules(t *testing.T) {
	existing := []cloudflare.RulesetRule{
		{Description: "rule-a"},
		{Description: "rule-b"},
		{Description: "rule-c"},
	}
	newRule := cloudflare.RulesetRule{Description: "rule-b", Expression: "updated"}
	result := upsertRule(existing, newRule, "rule-b")
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0].Description != "rule-a" {
		t.Errorf("result[0] = %q, want rule-a", result[0].Description)
	}
	if result[2].Description != "rule-c" {
		t.Errorf("result[2] = %q, want rule-c", result[2].Description)
	}
}

// --- corsRuleFromConfig tests ---

func TestCORSRuleFromConfig_AllFields(t *testing.T) {
	cfg := &corsConfig{
		name:             "my-rule",
		expression:       "http.host eq \"example.com\"",
		allowOrigins:     []string{"https://example.com"},
		allowMethods:     []string{"GET", "PUT"},
		allowHeaders:     []string{"Authorization"},
		maxAge:           1800,
		allowCredentials: true,
	}
	rule := corsRuleFromConfig(cfg)
	if rule.Name != "my-rule" {
		t.Errorf("Name = %q", rule.Name)
	}
	if !rule.Enabled {
		t.Error("Enabled should be true")
	}
	if rule.Expression != cfg.expression {
		t.Errorf("Expression = %q", rule.Expression)
	}
	if len(rule.AllowOrigins) != 1 || rule.AllowOrigins[0] != "https://example.com" {
		t.Errorf("AllowOrigins = %v", rule.AllowOrigins)
	}
	if len(rule.AllowMethods) != 2 {
		t.Errorf("AllowMethods len = %d", len(rule.AllowMethods))
	}
	if rule.MaxAge != 1800 {
		t.Errorf("MaxAge = %d", rule.MaxAge)
	}
	if !rule.AllowCredentials {
		t.Error("AllowCredentials should be true")
	}
}

func TestCORSRuleFromConfig_ID_Empty(t *testing.T) {
	cfg := &corsConfig{name: "r", expression: "true"}
	rule := corsRuleFromConfig(cfg)
	if rule.ID != "" {
		t.Errorf("ID should be empty for new rule from config, got %q", rule.ID)
	}
}

// --- isNotFound string-shape tests (message matching via corsTestError) ---

func TestIsNotFound_StringShape_Nil(t *testing.T) {
	if isNotFound(nil) {
		t.Error("isNotFound(nil) should return false")
	}
}

func TestIsNotFound_StringShape_Messages(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"not found", true},
		{"could not find resource", true},
		{"404 Not Found", true},
		{"resource NOT FOUND in zone", true},
		{"internal server error", false},
		{"permission denied", false},
		{"unknown error occurred", false},
	}
	for _, tc := range cases {
		err := &corsTestError{tc.msg}
		got := isNotFound(err)
		if got != tc.want {
			t.Errorf("isNotFound(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

// corsTestError is a minimal error type for isNotFound string-shape tests.
type corsTestError struct{ msg string }

func (e *corsTestError) Error() string { return e.msg }

// --- CORSDefaultRuleName constant ---

func TestCORSDefaultRuleName(t *testing.T) {
	if CORSDefaultRuleName == "" {
		t.Error("CORSDefaultRuleName should not be empty")
	}
	if CORSDefaultRuleName != "cosmoflare-cors" {
		t.Errorf("CORSDefaultRuleName = %q, want cosmoflare-cors", CORSDefaultRuleName)
	}
}

// --- ErrCORSRuleNotFound ---

func TestErrCORSRuleNotFound_Message(t *testing.T) {
	if ErrCORSRuleNotFound == nil {
		t.Fatal("ErrCORSRuleNotFound should not be nil")
	}
	if ErrCORSRuleNotFound.Error() == "" {
		t.Error("ErrCORSRuleNotFound should have a non-empty message")
	}
}

// --- splitTrimmed edge cases ---

func TestSplitTrimmed_WhitespaceOnly(t *testing.T) {
	// A comma-separated string of only spaces should produce empty result
	got := splitTrimmed(",  ,  , ")
	for _, v := range got {
		if v != "" {
			t.Errorf("splitTrimmed should skip blank entries, got %q", v)
		}
	}
}

func TestSplitTrimmed_SingleNoComma(t *testing.T) {
	got := splitTrimmed("*")
	if len(got) != 1 || got[0] != "*" {
		t.Errorf("splitTrimmed('*') = %v, want [*]", got)
	}
}

// --- NewCORSServiceFromCreds success path ---

func TestNewCORSServiceFromCreds_Success(t *testing.T) {
	svc, err := NewCORSServiceFromCreds("zone-abc", "valid-api-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.zoneID != "zone-abc" {
		t.Errorf("zoneID = %q, want zone-abc", svc.zoneID)
	}
}

func TestNewCORSServiceFromCreds_EmptyZone(t *testing.T) {
	_, err := NewCORSServiceFromCreds("", "valid-token")
	if err == nil {
		t.Error("expected error when zone ID is empty")
	}
}

func TestNewCORSServiceFromCreds_EmptyToken(t *testing.T) {
	_, err := NewCORSServiceFromCreds("zone-abc", "")
	if err == nil {
		t.Error("expected error when API token is empty")
	}
}

// --- parseCORSRule edge cases ---

func TestParseCORSRule_MaxAgeInvalid(t *testing.T) {
	enabled := true
	r := cloudflare.RulesetRule{
		ID:         "r1",
		Expression: "true",
		Enabled:    &enabled,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{
				"Access-Control-Allow-Origin": {Operation: "set", Value: "*"},
				"Access-Control-Max-Age":      {Operation: "set", Value: "not-a-number"},
			},
		},
	}
	cr, ok := parseCORSRule(r)
	if !ok {
		t.Fatal("expected parseCORSRule to succeed")
	}
	// Invalid max-age should leave MaxAge at zero (default)
	if cr.MaxAge != 0 {
		t.Errorf("MaxAge = %d, want 0 for invalid value", cr.MaxAge)
	}
}

func TestParseCORSRule_EnabledFalse(t *testing.T) {
	disabled := false
	r := cloudflare.RulesetRule{
		ID:         "r2",
		Expression: "true",
		Enabled:    &disabled,
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{
				"Access-Control-Allow-Origin": {Operation: "set", Value: "https://example.com"},
			},
		},
	}
	cr, ok := parseCORSRule(r)
	if !ok {
		t.Fatal("expected parseCORSRule to succeed")
	}
	if cr.Enabled {
		t.Error("Enabled should be false when rule.Enabled points to false")
	}
}

func TestParseCORSRule_EmptyHeaders(t *testing.T) {
	r := cloudflare.RulesetRule{
		ActionParameters: &cloudflare.RulesetRuleActionParameters{
			Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{},
		},
	}
	_, ok := parseCORSRule(r)
	if ok {
		t.Error("expected parseCORSRule to return false for empty headers map")
	}
}

// --- buildRulesetRule methods/headers ---

func TestBuildRulesetRule_MethodsAndHeaders(t *testing.T) {
	cfg := &corsConfig{
		name:         "rule",
		expression:   "true",
		allowOrigins: []string{"*"},
		allowMethods: []string{"GET", "POST", "PUT"},
		allowHeaders: []string{"Content-Type", "X-Foo"},
		maxAge:       600,
	}
	rule := buildRulesetRule(cfg)
	if h, ok := rule.ActionParameters.Headers["Access-Control-Allow-Methods"]; !ok {
		t.Error("Access-Control-Allow-Methods missing")
	} else if h.Value != "GET, POST, PUT" {
		t.Errorf("Allow-Methods value = %q", h.Value)
	}
	if h, ok := rule.ActionParameters.Headers["Access-Control-Allow-Headers"]; !ok {
		t.Error("Access-Control-Allow-Headers missing")
	} else if h.Value != "Content-Type, X-Foo" {
		t.Errorf("Allow-Headers value = %q", h.Value)
	}
	if h, ok := rule.ActionParameters.Headers["Access-Control-Max-Age"]; !ok {
		t.Error("Access-Control-Max-Age missing")
	} else if h.Value != "600" {
		t.Errorf("Max-Age value = %q, want 600", h.Value)
	}
}
