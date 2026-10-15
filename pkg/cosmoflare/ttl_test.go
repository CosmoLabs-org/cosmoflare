package cosmoflare

import (
	"strings"
	"testing"
)

// corpusInvalidations mirrors the corpus's invalidateChurchCache body as the
// scanner's inventory would see it (Verified fact 4): a template literal
// yielding the static head "page:/church/", and a function-call argument
// yielding the bare name "churchCacheKey" (zero colon segments).
func corpusInvalidations() []InvalidationCall {
	return []InvalidationCall{
		{File: "middleware/cache.ts", Line: 89, KeyPattern: "page:/church/"},
		{File: "middleware/cache.ts", Line: 93, KeyPattern: "churchCacheKey"},
	}
}

func TestInvalidationCovered(t *testing.T) {
	tests := []struct {
		name          string
		routePrefix   string
		invalidations []InvalidationCall
		want          bool
	}{
		{name: "depth1 shared api is NOT covered", routePrefix: "api:meta:countries", invalidations: []InvalidationCall{{KeyPattern: "api:/api/churches/"}}, want: false},
		{name: "depth2 shared api:meta IS covered", routePrefix: "api:meta:countries", invalidations: []InvalidationCall{{KeyPattern: "api:meta:"}}, want: true},
		{name: "empty route prefix is never covered", routePrefix: "", invalidations: []InvalidationCall{{KeyPattern: "api:meta:"}}, want: false},
		{name: "no shared segments", routePrefix: "page:/church/", invalidations: []InvalidationCall{{KeyPattern: "api:meta:"}}, want: false},
		{name: "function-name pattern covers nothing", routePrefix: "api:meta:stats", invalidations: []InvalidationCall{{KeyPattern: "churchCacheKey"}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InvalidationCovered(tt.routePrefix, tt.invalidations); got != tt.want {
				t.Fatalf("InvalidationCovered(%q) = %v, want %v", tt.routePrefix, got, tt.want)
			}
		})
	}
}

// inv builds a minimal inventory containing one cached route.
func inv(routes []CachedRoute, invalidations []InvalidationCall) *ProjectCacheInventory {
	return &ProjectCacheInventory{
		Root:         "/tmp/project",
		Routes:       routes,
		Invalidation: invalidations,
	}
}

func TestGradeTTLsStaleRisk(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /countries", KeyPrefix: "api:meta:countries", TTL: 21600}}, nil), 500)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	v := got[0]
	if v.Verdict != "stale-risk" {
		t.Fatalf("verdict = %q, want stale-risk", v.Verdict)
	}
	if v.Covered {
		t.Fatal("covered = true, want false")
	}
	if !v.CoverageKnown {
		t.Fatal("coverage_known = false, want true (prefix is statically known)")
	}
	if v.Cadence != "daily" {
		t.Fatalf("cadence = %q, want daily", v.Cadence)
	}
	if v.WritesPerDay != 500 {
		t.Fatalf("writesPerDay = %d, want 500", v.WritesPerDay)
	}
}

func TestGradeTTLsNoInvalidation(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /stats", KeyPrefix: "stats:v1", TTL: 600}}, nil), 50)
	v := got[0]
	if v.Verdict != "no-invalidation" {
		t.Fatalf("verdict = %q, want no-invalidation", v.Verdict)
	}
	if v.Cadence != "daily" {
		t.Fatalf("cadence = %q, want daily", v.Cadence)
	}
}

func TestGradeTTLsRaise(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /meta", KeyPrefix: "api:meta:stats", TTL: 600}}, []InvalidationCall{{KeyPattern: "api:meta:"}}), 0)
	v := got[0]
	if v.Verdict != "raise" {
		t.Fatalf("verdict = %q, want raise", v.Verdict)
	}
	if !v.Covered || !v.CoverageKnown {
		t.Fatalf("covered=%v known=%v, want true/true", v.Covered, v.CoverageKnown)
	}
	if v.Cadence != "rare" {
		t.Fatalf("cadence = %q, want rare", v.Cadence)
	}
}

func TestGradeTTLsAppropriate(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /list", KeyPrefix: "api:list", TTL: 3600}}, []InvalidationCall{{KeyPattern: "api:list:"}}), 50)
	v := got[0]
	if v.Verdict != "appropriate" {
		t.Fatalf("verdict = %q, want appropriate", v.Verdict)
	}
	if !v.Covered || !v.CoverageKnown {
		t.Fatalf("covered=%v known=%v, want true/true", v.Covered, v.CoverageKnown)
	}
}

func TestGradeTTLsUnknownCoverage(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /church/x", KeyPrefix: "", TTL: 3600}}, nil), 50)
	v := got[0]
	if v.Verdict != "appropriate" {
		t.Fatalf("verdict = %q, want appropriate", v.Verdict)
	}
	if v.CoverageKnown {
		t.Fatal("coverage_known = true, want false")
	}
	if strings.Contains(v.Reason, "not statically determinable") == false {
		t.Fatalf("reason %q should note coverage not statically determinable", v.Reason)
	}
}

func TestGradeTTLsAsymmetryNote(t *testing.T) {
	routes := []CachedRoute{
		{Route: "GET /countries", KeyPrefix: "api:meta:countries", TTL: 21600},
		{Route: "GET /stats", KeyPrefix: "stats:v1", TTL: 600},
		{Route: "GET /meta", KeyPrefix: "api:meta:stats", TTL: 600},
		{Route: "GET /list", KeyPrefix: "api:list", TTL: 3600},
		{Route: "GET /church/x", KeyPrefix: "", TTL: 3600},
	}
	got := GradeTTLs(inv(routes, []InvalidationCall{{KeyPattern: "api:meta:"}}), 500)
	if len(got) != len(routes) {
		t.Fatalf("len = %d, want %d", len(got), len(routes))
	}
	for _, v := range got {
		if !strings.HasSuffix(v.Reason, TTLAsymmetryNote) {
			t.Fatalf("verdict %s: reason does not end with TTLAsymmetryNote: %q", v.Verdict, v.Reason)
		}
		if v.Route == "" {
			t.Fatal("route field empty")
		}
	}
}

// TestGradeTTLsZeroTTLNotSpecified pins the TTL==0 handling: 0 means the
// middleware default was not specified, so the route is graded AT the
// reference default (3600s) and the reason says so — it must not read as a
// known short TTL that can never be stale-risk.
func TestGradeTTLsZeroTTLNotSpecified(t *testing.T) {
	got := GradeTTLs(inv([]CachedRoute{{Route: "GET /cold", KeyPrefix: "cold:v1", TTL: 0}}, nil), 500)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	v := got[0]
	if v.Verdict != "no-invalidation" {
		t.Fatalf("verdict = %q, want no-invalidation (graded at the 3600s default)", v.Verdict)
	}
	if !strings.Contains(v.Reason, "not specified") {
		t.Fatalf("reason %q does not state the TTL was not specified", v.Reason)
	}
}
