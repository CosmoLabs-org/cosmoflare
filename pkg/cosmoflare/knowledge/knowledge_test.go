package knowledge

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		path string
		want bool
	}{
		{"exact", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets", true},
		{"param mismatch length", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets/extra", false},
		{"two params", "/zones/{z}/rulesets/{r}/rules", "/zones/z1/rulesets/r1/rules", true},
		{"literal segment differs", "/zones/{z}/dns", "/accounts/a1/dns", false},
		{"scope star consumes rest", "/zones/{zone_id}/rulesets*", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint", true},
		{"star requires prefix", "/zones/{zone_id}/rulesets*", "/zones/z1/other", false},
		{"empty param segment", "/zones/{z}/rulesets", "/zones//rulesets", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPath(tt.tmpl, tt.path); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.tmpl, tt.path, got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/client/v4/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/client/v4/client/v4/zones", "/client/v4/zones"}, // strip one prefix only
	}
	for _, tt := range tests {
		if got := NormalizePath(tt.in); got != tt.want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRatelimitPackLoadedAndValid(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pack *Pack
	for _, p := range loaded {
		if p.Product == "ratelimit" {
			pack = p
		}
	}
	if pack == nil {
		t.Fatal("ratelimit pack not loaded")
	}
	if len(pack.Scopes) == 0 || len(pack.Endpoints) == 0 {
		t.Fatalf("ratelimit pack must declare scopes and endpoints: %+v", pack)
	}

	// Scope check: rulesets paths are in scope, everything else is not.
	if !CheckRoute("GET", "/zones/z1/rulesets").InScope {
		t.Fatal("rulesets path must be in scope")
	}
	if CheckRoute("GET", "/zones/z1/dns_records").InScope {
		t.Fatal("dns_records path must be out of scope")
	}

	// The known-nonexistent route blocks pre-send.
	v := CheckRoute("POST", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint/rules")
	if !v.Blocked {
		t.Fatal("POST entrypoint/rules must be blocked (endpoint does not exist)")
	}

	// The documented routes pass.
	if v2 := CheckRoute("PUT", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint"); v2.Blocked || v2.Endpoint == nil {
		t.Fatalf("PUT entrypoint must be registered: %+v", v2)
	}

	// All four evidence codes decode.
	for _, code := range []int{10405, 20155, 10000} {
		if LookupDecode(code, "") == nil {
			t.Fatalf("code %d must have a global decode", code)
		}
	}
	if LookupDecode(1000, "phase-entrypoint") == nil {
		t.Fatal("code 1000 must decode under context phase-entrypoint")
	}

	// Free caps present.
	found := false
	for _, c := range pack.PlanCaps {
		if c.Plan == "free" {
			found = true
			if c.Caps["max:period_seconds"] != 10 || c.Caps["max:mitigation_timeout_seconds"] != 10 || c.Caps["max:rules_count"] != 1 {
				t.Fatalf("free caps wrong: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("free plan caps missing")
	}

	// The cf.colo.id invariant is present.
	hasInvariant := false
	for _, inv := range pack.Invariants {
		if inv.Field == "characteristics" && inv.Value == "cf.colo.id" {
			hasInvariant = true
		}
	}
	if !hasInvariant {
		t.Fatal("cf.colo.id invariant missing")
	}
}

// TestSkippedTrafficClasses verifies the ratelimit pack declares its
// not-counted traffic classes with evidence sources, and that unknown
// products return nothing (advisory when absent).
func TestSkippedTrafficClasses(t *testing.T) {
	skipped := SkippedTrafficClasses("ratelimit")
	if len(skipped) != 2 {
		t.Fatalf("ratelimit pack must declare exactly 2 skipped classes, got %d: %+v", len(skipped), skipped)
	}
	for _, tc := range skipped {
		if tc.Counted {
			t.Fatalf("skipped classes must have Counted=false: %+v", tc)
		}
		if tc.Source == "" {
			t.Fatalf("every matrix entry must carry a source: %+v", tc)
		}
	}
	if got := SkippedTrafficClasses("nosuch"); len(got) != 0 {
		t.Fatalf("unknown product must return empty, got %+v", got)
	}
}
