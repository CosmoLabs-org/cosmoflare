package knowledge

import "testing"

// loadPackForProduct returns the embedded pack for one product, failing the
// test when it is missing. Shared by the per-pack regression tests so they
// do not each re-implement the Load + linear scan.
func loadPackForProduct(t *testing.T, product string) *Pack {
	t.Helper()
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, p := range loaded {
		if p.Product == product {
			return p
		}
	}
	t.Fatalf("pack %q not loaded (loaded: %d packs)", product, len(loaded))
	return nil
}

// TestMatchPath validates the template matcher that underpins scoping and
// endpoint registration: literal segments must match exactly, "{param}"
// segments match any single non-empty segment, a trailing "*" prefix-matches
// the remainder of the path, and segment counts must line up.
func TestMatchPath(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if got := matchPath(tt.tmpl, tt.path); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.tmpl, tt.path, got, tt.want)
			}
		})
	}
}

// TestNormalizePath validates that exactly one leading "/client/v4" is
// stripped so pack templates written against the bare API path also match
// requests made through the SDK's default BaseURL.
func TestNormalizePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"sdk prefix stripped", "/client/v4/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"bare path unchanged", "/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"only one prefix stripped", "/client/v4/client/v4/zones", "/client/v4/zones"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizePath(tt.in); got != tt.want {
				t.Fatalf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestRatelimitPackLoadedAndValid is the ratelimit pack regression test: the
// pack must load with populated sections, scope exactly its rulesets routes,
// block known-nonexistent routes while registering documented ones, expose
// decodes for its evidence codes, and encode the free-plan caps and the
// cf.colo.id invariant.
func TestRatelimitPackLoadedAndValid(t *testing.T) {
	t.Parallel()
	pack := loadPackForProduct(t, "ratelimit")

	t.Run("pack declares scopes and endpoints", func(t *testing.T) {
		t.Parallel()
		if len(pack.Scopes) == 0 || len(pack.Endpoints) == 0 {
			t.Fatalf("ratelimit pack must declare scopes and endpoints: %+v", pack)
		}
	})

	t.Run("scoping follows the rulesets scope", func(t *testing.T) {
		t.Parallel()
		if !CheckRoute("GET", "/zones/z1/rulesets").InScope {
			t.Fatal("rulesets path must be in scope")
		}
		// dns_records lives under /zones/{z} but outside the rulesets
		// scope — scoping is prefix-shaped, not account-shaped.
		if CheckRoute("GET", "/zones/z1/dns_records").InScope {
			t.Fatal("dns_records path must be out of scope")
		}
	})

	t.Run("known-nonexistent route blocks pre-send", func(t *testing.T) {
		t.Parallel()
		v := CheckRoute("POST", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint/rules")
		if !v.Blocked {
			t.Fatal("POST entrypoint/rules must be blocked (endpoint does not exist)")
		}
	})

	t.Run("documented routes pass", func(t *testing.T) {
		t.Parallel()
		if v := CheckRoute("PUT", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint"); v.Blocked || v.Endpoint == nil {
			t.Fatalf("PUT entrypoint must be registered: %+v", v)
		}
	})

	t.Run("evidence codes decode", func(t *testing.T) {
		t.Parallel()
		for _, code := range []int{10405, 20155, 10000} {
			if LookupDecode(code, "") == nil {
				t.Fatalf("code %d must have a global decode", code)
			}
		}
		// 1000 is ambiguous across products and only decodes when the
		// caller supplies the entrypoint context.
		if LookupDecode(1000, "phase-entrypoint") == nil {
			t.Fatal("code 1000 must decode under context phase-entrypoint")
		}
	})

	t.Run("free plan caps present", func(t *testing.T) {
		t.Parallel()
		for _, c := range pack.PlanCaps {
			if c.Plan != "free" {
				continue
			}
			if c.Caps["max:period_seconds"] != 10 || c.Caps["max:mitigation_timeout_seconds"] != 10 || c.Caps["max:rules_count"] != 1 {
				t.Fatalf("free caps wrong: %+v", c)
			}
			return
		}
		t.Fatal("free plan caps missing")
	})

	t.Run("cf.colo.id invariant present", func(t *testing.T) {
		t.Parallel()
		for _, inv := range pack.Invariants {
			if inv.Field == "characteristics" && inv.Value == "cf.colo.id" {
				return
			}
		}
		t.Fatal("cf.colo.id invariant missing")
	})
}

// TestSkippedTrafficClasses verifies the ratelimit pack declares its
// not-counted traffic classes with evidence sources, and that unknown
// products return nothing (advisory when absent).
func TestSkippedTrafficClasses(t *testing.T) {
	t.Parallel()
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
