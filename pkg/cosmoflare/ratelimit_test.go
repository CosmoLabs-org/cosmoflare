package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// ratelimitTestServer serves the full rulesets flow. Phase is the
// http_ratelimit entrypoint ruleset "rl-rs" with `existing` rules.
func ratelimitTestServer(t *testing.T, existing []cloudflare.RulesetRule) (*httptest.Server, *cloudflare.API) {
	t.Helper()
	mux := http.NewServeMux()
	encode := func(w http.ResponseWriter, result any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
	}
	mux.HandleFunc("/zones/z1", func(w http.ResponseWriter, r *http.Request) {
		encode(w, map[string]any{"id": "z1", "name": "example.com", "plan": map[string]any{"legacy_id": "free"}})
	})
	mux.HandleFunc("/zones/z1/rulesets/phases/http_ratelimit/entrypoint", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if existing == nil {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"errors":  []map[string]any{{"code": 1000, "message": "not_found"}},
				})
				return
			}
			encode(w, cloudflare.Ruleset{ID: "rl-rs", Phase: "http_ratelimit", Rules: existing})
		case http.MethodPut:
			var body struct {
				Rules []cloudflare.RulesetRule `json:"rules"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode PUT body: %v", err)
			}
			for i := range body.Rules {
				if body.Rules[i].ID == "" {
					body.Rules[i].ID = "generated-id"
				}
			}
			encode(w, cloudflare.Ruleset{ID: "rl-rs", Phase: "http_ratelimit", Rules: body.Rules})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return srv, cf
}

func newRatelimitService(t *testing.T, cf *cloudflare.API) *RateLimitService {
	t.Helper()
	zones, err := NewZoneService(cf, "acct-test")
	if err != nil {
		t.Fatalf("NewZoneService: %v", err)
	}
	svc, err := NewRateLimitService(cf, zones)
	if err != nil {
		t.Fatalf("NewRateLimitService: %v", err)
	}
	return svc
}

// TestRateLimitListEmptyOnMissingEntrypoint
// TestRateLimitListEmptyOnMissingEntrypoint verifies that List treats a zone
// with no http_ratelimit entrypoint ruleset (404) as "zero rules" rather than
// an error, so freshly created zones work.
func TestRateLimitListEmptyOnMissingEntrypoint(t *testing.T) {
	t.Parallel()
	_, cf := ratelimitTestServer(t, nil)
	got, err := newRatelimitService(t, cf).List(context.Background(), "z1")
	if err != nil {
		t.Fatalf("List on fresh zone must return empty, not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 rules, got %+v", got)
	}
}

// TestRateLimitListReturnsRules TestRateLimitListReturnsRules verifies that
// List maps entrypoint ruleset rules onto the rate-limit view model,
// preserving rule ID, requests-per-period, and traffic characteristics.
func TestRateLimitListReturnsRules(t *testing.T) {
	t.Parallel()
	enabled := true
	_, cf := ratelimitTestServer(t, []cloudflare.RulesetRule{{
		ID: "r1", Action: "block", Expression: `path eq "/a"`, Enabled: &enabled,
		RateLimit: &cloudflare.RulesetRuleRateLimit{
			Characteristics:   []string{"cf.colo.id", "ip.src"},
			RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		},
	}})
	got, err := newRatelimitService(t, cf).List(context.Background(), "z1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "r1" || got[0].RequestsPerPeriod != 10 {
		t.Fatalf("mapping wrong: %+v", got)
	}
	if !containsStr(got[0].Characteristics, "cf.colo.id") {
		t.Fatalf("characteristics must carry through: %+v", got[0])
	}
}

// TestRateLimitCreatePreflightBlocksMissingColo
// TestRateLimitCreatePreflightBlocksMissingColo verifies that Create refuses
// a rule whose characteristics omit cf.colo.id (required by Cloudflare) and
// that the error names the missing field.
func TestRateLimitCreatePreflightBlocksMissingColo(t *testing.T) {
	t.Parallel()
	_, cf := ratelimitTestServer(t, nil)
	_, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/a"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"ip.src"}, // missing cf.colo.id
	})
	if err == nil {
		t.Fatal("preflight must block missing cf.colo.id")
	}
	if !strings.Contains(err.Error(), "cf.colo.id") {
		t.Fatalf("error must name the field: %v", err)
	}
}

// TestRateLimitCreatePreflightBlocksFreeCap
// TestRateLimitCreatePreflightBlocksFreeCap verifies that Create enforces the
// free-plan cap of one rate-limit rule: adding a second rule to a zone that
// already has one is rejected with an error naming the cap.
func TestRateLimitCreatePreflightBlocksFreeCap(t *testing.T) {
	t.Parallel()
	enabled := true
	_, cf := ratelimitTestServer(t, []cloudflare.RulesetRule{{
		ID: "existing", Action: "block", Expression: `path eq "/old"`, Enabled: &enabled,
	}}) // zone already has 1 rule; free cap is 1
	_, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/new"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"cf.colo.id", "ip.src"},
	})
	if err == nil {
		t.Fatal("free plan allows 1 rule — second create must block")
	}
	if !strings.Contains(err.Error(), "rules_count") {
		t.Fatalf("error must name the cap: %v", err)
	}
}

// TestRateLimitCreatePutsEntrypoint TestRateLimitCreatePutsEntrypoint
// verifies that Create PUTs a complete rule to the entrypoint ruleset and
// returns the persisted rule with its server-assigned ID and period.
func TestRateLimitCreatePutsEntrypoint(t *testing.T) {
	t.Parallel()
	_, cf := ratelimitTestServer(t, nil)
	rule, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/catalog.json"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"cf.colo.id", "ip.src"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID == "" || rule.Period != 10 {
		t.Fatalf("created rule wrong: %+v", rule)
	}
}

func containsStr(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
