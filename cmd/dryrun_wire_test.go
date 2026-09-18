package cmd

import (
	"strings"
	"testing"
)

// TestDestructiveDryRun pins the FEAT-020 wave-2 consumer: dry-run defaults
// derive from the registry's destructive flag. Explicit --dry-run always
// stays dry; --force overrides the destructive default; unregistered and
// non-destructive paths keep the operator's flag choice verbatim.
func TestDestructiveDryRun(t *testing.T) {
	orig := DryRun
	defer func() { DryRun = orig }()

	cases := []struct {
		name  string
		path  []string
		dry   bool
		force bool
		want  bool
	}{
		{"destructive defaults dry", []string{"d1", "delete"}, false, false, true},
		{"destructive force executes", []string{"d1", "delete"}, false, true, false},
		{"explicit dry-run wins over force", []string{"d1", "delete"}, true, true, true},
		{"non-destructive defaults execute", []string{"dns", "create"}, false, false, false},
		{"non-destructive honors dry-run flag", []string{"dns", "create"}, true, false, true},
		{"unregistered path defaults execute", []string{"nope", "delete"}, false, false, false},
		{"unregistered path honors dry-run flag", []string{"nope", "delete"}, true, false, true},
		{"nested destructive path", []string{"kv", "namespace", "delete"}, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			DryRun = tc.dry
			if got := destructiveDryRun(tc.path, tc.force); got != tc.want {
				t.Errorf("destructiveDryRun(%q, force=%v) = %v, want %v",
					strings.Join(tc.path, " "), tc.force, got, tc.want)
			}
		})
	}
}
