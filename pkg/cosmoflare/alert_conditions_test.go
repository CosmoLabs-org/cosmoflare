package cosmoflare

import (
	"strings"
	"testing"
)

// TestAlertConditionRegistryComplete pins the FEAT-015 contract: every
// registered condition carries a full descriptor, names are unique, and the
// legacy map-based validation accepts exactly the registry set.
func TestAlertConditionRegistryComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range AlertConditions() {
		if c.Name == "" {
			t.Fatalf("registry entry with empty name: %+v", c)
		}
		if seen[c.Name] {
			t.Fatalf("duplicate condition %q", c.Name)
		}
		seen[c.Name] = true
		for _, field := range map[string]string{
			"unit": c.Unit, "help": c.Help, "fed-by": c.FedBy, "data-key": c.DataKey, "service": c.Service,
		} {
			if strings.TrimSpace(field) == "" {
				t.Errorf("condition %q has an empty descriptor field: %+v", c.Name, c)
			}
		}
		if !validAlertCondition(c.Name) {
			t.Errorf("validAlertCondition(%q) = false, condition is registered", c.Name)
		}
	}
	if validAlertCondition("not-a-condition") {
		t.Error("validAlertCondition must reject unknown names")
	}
}

// TestAlertConditionListMatchesRegistry pins the error/help text derivation:
// the comma-joined list is exactly the registry names in registry order.
func TestAlertConditionListMatchesRegistry(t *testing.T) {
	reg := AlertConditions()
	want := make([]string, len(reg))
	for i, c := range reg {
		want[i] = c.Name
	}
	if got := AlertConditionList(); got != strings.Join(want, ", ") {
		t.Errorf("AlertConditionList() = %q, want %q", got, strings.Join(want, ", "))
	}
}

// TestAlertConditionScopes pins the FEAT-047 scope contract: stuck-work
// conditions fan out per script/do-object; legacy conditions stay account
// scope with an explicit value (never the zero-value ambiguity).
func TestAlertConditionScopes(t *testing.T) {
	for _, name := range []string{"worker-cpu", "worker-errors", "worker-requests", "worker-subrequests"} {
		desc, ok := LookupAlertCondition(name)
		if !ok {
			t.Fatalf("condition %s not registered", name)
		}
		if desc.Scope != "script" {
			t.Errorf("%s: Scope=%q, want script", name, desc.Scope)
		}
	}
	for _, name := range []string{"do-cpu", "do-requests"} {
		desc, ok := LookupAlertCondition(name)
		if !ok {
			t.Fatalf("condition %s not registered", name)
		}
		if desc.Scope != "do" {
			t.Errorf("%s: Scope=%q, want do", name, desc.Scope)
		}
	}
	for _, name := range []string{"error-rate", "storage-limit", "latency", "failure-count", "workers-script-count", "r2-bucket-count", "dns-record-quota"} {
		desc, ok := LookupAlertCondition(name)
		if !ok {
			t.Fatalf("condition %s not registered", name)
		}
		if desc.Scope != "account" {
			t.Errorf("%s: Scope=%q, want account (explicit, not zero value)", name, desc.Scope)
		}
	}
}
