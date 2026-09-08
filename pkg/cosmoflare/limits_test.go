package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLimitFor(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		plan     string
		want     uint64
		wantOK   bool
	}{
		{"workers scripts free", "workers.scripts", "free", 100, true},
		{"workers scripts paid", "workers.scripts", "paid", 500, true},
		{"workers daily requests free", "workers.daily_requests", "free", 100000, true},
		{"workers daily requests paid is unlimited", "workers.daily_requests", "paid", 0, true},
		{"r2 buckets plan-independent", "r2.buckets", "", 1000000, true},
		{"r2 custom domains plan-independent", "r2.custom_domains_per_bucket", "paid", 100, true},
		{"workers resource with unknown plan", "workers.scripts", "unknown", 0, false},
		{"unknown resource", "nope", "free", 0, false},
		{"workers resource with empty plan", "workers.scripts", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := limitFor(tt.resource, tt.plan)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("limitFor(%q, %q) = (%d, %v), want (%d, %v)",
					tt.resource, tt.plan, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDNSRecordsStaticLimit(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		created string // ISO date; empty = zero time
		want    uint64
		wantOK  bool
	}{
		{"pro", "pro", "", 3500, true},
		{"business", "business", "", 3500, true},
		{"free before cutoff", "free", "2024-01-15", 1000, true},
		{"free on cutoff", "free", "2024-09-01", 200, true},
		{"free after cutoff", "free", "2025-06-01", 200, true},
		{"enterprise has no per-zone limit", "enterprise", "", 0, false},
		{"unknown plan", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := time.Time{}
			if tt.created != "" {
				parsed, err := time.Parse("2006-01-02", tt.created)
				if err != nil {
					t.Fatalf("bad fixture date %q: %v", tt.created, err)
				}
				created = parsed
			}
			got, ok := dnsRecordsStaticLimit(tt.plan, created)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("dnsRecordsStaticLimit(%q, %v) = (%d, %v), want (%d, %v)",
					tt.plan, created, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNewLimitsServiceDefaults(t *testing.T) {
	s := NewLimitsService("acct", "tok")
	if s.accountID != "acct" || s.apiToken != "tok" {
		t.Fatal("credentials not stored")
	}
	if s.httpClient == nil {
		t.Fatal("default HTTP client missing")
	}
	if s.baseURL != "https://api.cloudflare.com/client/v4" {
		t.Fatalf("baseURL = %q", s.baseURL)
	}
}

func TestNewLimitsServiceValidation(t *testing.T) {
	if _, err := NewLimitsService("", "tok").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty account ID must fail validation")
	} else if !strings.Contains(err.Error(), "account ID is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewLimitsService("acct", "").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty token must fail validation")
	} else if !strings.Contains(err.Error(), "API token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testLimitServer(t *testing.T, handler http.HandlerFunc) (*LimitsService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewLimitsService("acct", "tok", WithLimitsBaseURL(srv.URL)), srv
}

func TestResolveWorkersPlanAuto(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct/subscriptions" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "cdn"}, "rate_plan": {"id": "cdn_pro"}},
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_paid", "public_name": "Workers Paid"}}
		]}`)
	})
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != "paid" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (paid, auto)", plan, source)
	}
}

func TestResolveWorkersPlanAutoFree(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_free"}}
		]}`)
	})
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != "free" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (free, auto)", plan, source)
	}
}

func TestResolveWorkersPlanFallbackChain(t *testing.T) {
	// Subscriptions endpoint 403s (scoped token) → config → flag ordering.
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success": false}`)
	})

	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "unknown" || source != "unknown" {
		t.Fatalf("no fallbacks: got (%q, %q, %v), want (unknown, unknown, nil)", plan, source, err)
	}

	s.configPlan = "paid"
	plan, source, err = s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "paid" || source != "config" {
		t.Fatalf("config fallback: got (%q, %q, %v), want (paid, config, nil)", plan, source, err)
	}

	s.flagPlan = "free"
	plan, source, err = s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "free" || source != "flag" {
		t.Fatalf("flag outranks config: got (%q, %q, %v), want (free, flag, nil)", plan, source, err)
	}
}

func TestResolveWorkersPlanInvalidValues(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	s.configPlan = "enterprise" // not a Workers tier — must be rejected, not trusted
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "unknown" || source != "unknown" {
		t.Fatalf("invalid config plan: got (%q, %q, %v), want (unknown, unknown, nil)", plan, source, err)
	}
}

func TestDNSUsageLive(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/z1/dns/usage" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": {"used": 180, "quota": 200}}`)
	})
	used, limit, source, err := s.dnsUsage(context.Background(), "z1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 180 || limit != 200 || source != "live-api" {
		t.Fatalf("dnsUsage = (%d, %d, %q), want (180, 200, live-api)", used, limit, source)
	}
}

func TestDNSUsageAlternateFieldNames(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": {"records_used": 10, "max_records": 3500}}`)
	})
	used, limit, source, err := s.dnsUsage(context.Background(), "z1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 10 || limit != 3500 || source != "live-api" {
		t.Fatalf("dnsUsage = (%d, %d, %q), want (10, 3500, live-api)", used, limit, source)
	}
}

func TestDNSUsageFallbackToStatic(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success": false}`)
	})
	// Zone created 2025-03-01 on free plan → 200 by cutoff rule.
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	used, limit, source, err := s.dnsUsageFallback(context.Background(), "z1", "free", created)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No live data → used falls back to a record count of 0 only when caller
	// passes it; here we assert the static limit and source.
	if limit != 200 || source != "static-docs" {
		t.Fatalf("dnsUsageFallback = (%d, %q, %v), want (200, static-docs, nil)", limit, source, err)
	}
	_ = used
}

func TestZonePlanLegacyID(t *testing.T) {
	z := Zone{Plan: ZonePlan{ID: "x", LegacyID: "pro", Name: "Pro"}}
	if z.Plan.LegacyID != "pro" {
		t.Fatalf("LegacyID = %q, want pro", z.Plan.LegacyID)
	}
}
