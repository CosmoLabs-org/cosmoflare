package knowledge

import (
	"maps"
	"testing"
)

// clonePayload returns a shallow copy of base with mut applied, so each
// subtest starts from a known-good free-tier payload and perturbs only the
// fields it exercises instead of re-declaring the whole map.
func clonePayload(base map[string]any, mut func(map[string]any)) map[string]any {
	p := maps.Clone(base)
	mut(p)
	return p
}

// TestValidatePayloadRatelimit validates preflight payload checks against
// the ratelimit pack: clean free-tier payloads pass, the cf.colo.id invariant
// fires regardless of plan knowledge, plan caps and the characteristics
// allow_set are enforced on known plans, and unknown plans/products degrade
// to advisory.
func TestValidatePayloadRatelimit(t *testing.T) {
	t.Parallel()
	base := map[string]any{
		"rules_count":                1,
		"period_seconds":             10,
		"mitigation_timeout_seconds": 10,
		"characteristics":            []string{"cf.colo.id", "ip.src"},
	}

	t.Run("clean on free", func(t *testing.T) {
		t.Parallel()
		if v := ValidatePayload("ratelimit", "free", base); len(v) != 0 {
			t.Fatalf("clean free payload must pass, got %+v", v)
		}
	})

	t.Run("missing cf.colo.id violates invariant", func(t *testing.T) {
		t.Parallel()
		p := clonePayload(base, func(p map[string]any) {
			p["characteristics"] = []string{"ip.src"}
		})
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one characteristics violation, got %+v", v)
		}
	})

	t.Run("free caps enforced", func(t *testing.T) {
		t.Parallel()
		p := clonePayload(base, func(p map[string]any) {
			// Two caps broken at once: rules_count (max 1) and
			// period_seconds (max 10 on free).
			p["rules_count"] = 2
			p["period_seconds"] = 60
		})
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 2 {
			t.Fatalf("want rules_count + period violations, got %+v", v)
		}
	})

	t.Run("characteristics allow_set enforced", func(t *testing.T) {
		t.Parallel()
		p := clonePayload(base, func(p map[string]any) {
			p["characteristics"] = []string{"cf.colo.id", "http.request.headers.x-tiny"}
		})
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one allow_set violation, got %+v", v)
		}
	})

	t.Run("unknown plan skips caps not invariants", func(t *testing.T) {
		t.Parallel()
		p := clonePayload(base, func(p map[string]any) {
			// All caps broken, but plan "" has no caps entry — only the
			// invariant (missing cf.colo.id) may fire.
			p["rules_count"] = 99
			p["period_seconds"] = 3600
			p["mitigation_timeout_seconds"] = 600
			p["characteristics"] = []string{"ip.src"}
		})
		v := ValidatePayload("ratelimit", "", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("unknown plan: only the invariant fires, got %+v", v)
		}
	})

	t.Run("unknown product passes", func(t *testing.T) {
		t.Parallel()
		if v := ValidatePayload("nosuch", "free", base); len(v) != 0 {
			t.Fatalf("unknown product must pass, got %+v", v)
		}
	})
}
