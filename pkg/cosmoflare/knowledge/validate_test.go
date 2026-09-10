package knowledge

import "testing"

func TestValidatePayloadRatelimit(t *testing.T) {
	base := map[string]any{
		"rules_count":                1,
		"period_seconds":             10,
		"mitigation_timeout_seconds": 10,
		"characteristics":            []string{"cf.colo.id", "ip.src"},
	}

	t.Run("clean on free", func(t *testing.T) {
		if v := ValidatePayload("ratelimit", "free", base); len(v) != 0 {
			t.Fatalf("clean free payload must pass, got %+v", v)
		}
	})

	t.Run("missing cf.colo.id violates invariant", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 1, "period_seconds": 10, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"ip.src"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one characteristics violation, got %+v", v)
		}
	})

	t.Run("free caps enforced", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 2, "period_seconds": 60, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"cf.colo.id", "ip.src"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 2 {
			t.Fatalf("want rules_count + period violations, got %+v", v)
		}
	})

	t.Run("characteristics allow_set enforced", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 1, "period_seconds": 10, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"cf.colo.id", "http.request.headers.x-tiny"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one allow_set violation, got %+v", v)
		}
	})

	t.Run("unknown plan skips caps not invariants", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 99, "period_seconds": 3600, "mitigation_timeout_seconds": 600,
			"characteristics": []string{"ip.src"},
		}
		v := ValidatePayload("ratelimit", "", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("unknown plan: only the invariant fires, got %+v", v)
		}
	})

	t.Run("unknown product passes", func(t *testing.T) {
		if v := ValidatePayload("nosuch", "free", base); len(v) != 0 {
			t.Fatalf("unknown product must pass, got %+v", v)
		}
	})
}
