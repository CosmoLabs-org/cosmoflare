package knowledge

import (
	"strings"
	"testing"
)

// TestPermissionsPackLoadedAndValid mirrors the ratelimit pack test: the
// permissions pack must load, declare scopes/endpoints, scope its routes,
// and decode its evidence codes.
func TestPermissionsPackLoadedAndValid(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pack *Pack
	for _, p := range loaded {
		if p.Product == "permissions" {
			pack = p
		}
	}
	if pack == nil {
		t.Fatal("permissions pack not loaded")
	}
	if len(pack.Scopes) == 0 || len(pack.Endpoints) == 0 || len(pack.Errors) == 0 {
		t.Fatalf("permissions pack must declare scopes, endpoints and errors: %+v", pack)
	}

	// Scope check: the permission_groups catalog route is in scope.
	v := CheckRoute("GET", "/user/tokens/permission_groups")
	if !v.InScope || v.Blocked || v.Endpoint == nil {
		t.Fatalf("GET permission_groups must be registered: %+v", v)
	}
	if CheckRoute("GET", "/user/tokens").InScope {
		t.Fatal("unrelated user route must be out of scope")
	}

	// The dead classic rate_limits API is documented as deprecated-off.
	v = CheckRoute("GET", "/zones/z1/rate_limits")
	if v.Blocked != true || v.Endpoint == nil || v.Endpoint.Status != "deprecated-off" {
		t.Fatalf("GET classic rate_limits must be blocked as deprecated-off: %+v", v)
	}

	// The rulesets phase entrypoints are documented (registered endpoints).
	for _, path := range []string{
		"/zones/z1/rulesets/phases/http_ratelimit/entrypoint",
		"/zones/z1/rulesets/phases/http_request_firewall_custom/entrypoint",
	} {
		// The ratelimit pack also scopes these routes and is consulted
		// first (alphabetical pack order); assert the route is at least
		// registered somewhere and not blocked.
		if vr := CheckRoute("PUT", path); vr.Blocked || vr.Endpoint == nil {
			t.Fatalf("PUT %s must be a registered route: %+v", path, vr)
		}
	}

	// The 10405 rulesets-writes decode is retrievable with its context and
	// carries the missing-scope fix.
	d := LookupDecode(10405, "rulesets-writes")
	if d == nil {
		t.Fatal("code 10405 must decode under context rulesets-writes")
	}
	if d.Context != "rulesets-writes" || d.Cause == "" || d.Fix == "" {
		t.Fatalf("10405 decode incomplete: %+v", d)
	}
	if want := "Zone > Zone WAF > Edit"; !strings.Contains(d.Cause, want) || !strings.Contains(d.Fix, want) {
		t.Fatalf("10405 decode must name the %q permission: %+v", want, d)
	}

	// The classic-rate-limits decode for 10000 is retrievable too.
	d = LookupDecode(10000, "classic-rate-limits")
	if d == nil {
		t.Fatal("code 10000 must decode under context classic-rate-limits")
	}
	if !strings.Contains(d.Fix, "http_ratelimit") {
		t.Fatalf("10000 decode must point at the http_ratelimit phase: %+v", d)
	}
}