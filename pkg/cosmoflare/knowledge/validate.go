package knowledge

import "fmt"

// Violation is one preflight failure: the field, the rule, and the fix.
type Violation struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Fix   string `json:"fix"`
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s — %s", v.Field, v.Rule, v.Fix)
}

// ValidatePayload checks a product payload against its pack's plan caps and
// invariants. plan "" (unknown) skips caps but still runs invariants.
// Unknown products return nil (knowledge is advisory when absent).
func ValidatePayload(product, plan string, payload map[string]any) []Violation {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var pack *Pack
	for _, p := range loaded {
		if p.Product == product {
			pack = p
		}
	}
	if pack == nil {
		return nil
	}

	var out []Violation

	if plan != "" {
		for _, cap := range pack.PlanCaps {
			if cap.Plan != plan {
				continue
			}
			for key, limit := range cap.Caps {
				field, ok := strings_CutPrefix(key, "max:")
				if !ok {
					continue
				}
				if val, ok := payload[field].(int); ok && val > limit {
					out = append(out, Violation{
						Field: field,
						Rule:  fmt.Sprintf("plan %q caps %s at %d (got %d)", plan, field, limit, val),
						Fix:   fmt.Sprintf("reduce %s to <= %d or upgrade the plan", field, limit),
					})
				}
			}
			for key, allowed := range cap.Sets {
				field, ok := strings_CutPrefix(key, "allow_set:")
				if !ok {
					continue
				}
				vals, ok := payload[field].([]string)
				if !ok {
					continue
				}
				for _, v := range vals {
					if !contains(allowed, v) {
						out = append(out, Violation{
							Field: field,
							Rule:  fmt.Sprintf("plan %q allows only %v (got %q)", plan, allowed, v),
							Fix:   "remove the characteristic or upgrade the plan",
						})
					}
				}
			}
		}
	}

	for _, inv := range pack.Invariants {
		if inv.Op != "must_include" {
			continue
		}
		vals, ok := payload[inv.Field].([]string)
		if !ok || !contains(vals, inv.Value) {
			out = append(out, Violation{
				Field: inv.Field,
				Rule:  fmt.Sprintf("must include %q", inv.Value),
				Fix:   fmt.Sprintf("add %q to %s", inv.Value, inv.Field),
			})
		}
	}
	return out
}

// strings_CutPrefix avoids importing strings for one helper.
func strings_CutPrefix(s, prefix string) (string, bool) {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
