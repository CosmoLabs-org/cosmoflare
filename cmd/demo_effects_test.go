package cmd

import (
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/visual"
)

// demoEffectsQuiet suppresses visual animations for the duration of the test
// and restores the previous quiet state afterwards, so demo routines do not
// spray animation frames while still exercising their real code paths.
func demoEffectsQuiet(t *testing.T) {
	t.Helper()
	wasQuiet := visual.IsQuiet()
	visual.DisableAnimations()
	t.Cleanup(func() {
		if wasQuiet {
			visual.DisableAnimations()
		} else {
			visual.EnableAnimations()
		}
	})
}

// demoEffectFns maps every visual demo routine from demo.go to its name so a
// single table-driven test can drive them without duplicating setup.
func demoEffectFns() map[string]func() {
	return map[string]func(){
		"progress":   demoProgress,
		"rainbow":    demoRainbow,
		"dashboard":  demoDashboard,
		"typewriter": demoTypewriter,
		"pulse":      demoPulse,
		"random":     demoRandom,
	}
}

// TestDemoEffects_DoNotPanic verifies each visual demo routine (progress,
// rainbow, dashboard, typewriter, pulse, random) runs to completion without
// panicking when animations are suppressed. These routines sleep for several
// seconds by design, so the whole test is skipped in -short mode.
func TestDemoEffects_DoNotPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("demo routines sleep for seconds by design; skipped in -short mode")
	}
	fns := demoEffectFns()
	names := []string{"progress", "rainbow", "dashboard", "typewriter", "pulse", "random"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			demoEffectsQuiet(t)
			fns[name]() // must not panic
		})
	}
}

// TestDemoEffects_AllTypesDocumentedInHelp verifies every routine reachable
// through demoEffectFns is also advertised in the demo command's help text,
// so the dispatch table and the documentation cannot drift apart.
func TestDemoEffects_AllTypesDocumentedInHelp(t *testing.T) {
	for name := range demoEffectFns() {
		t.Run(name, func(t *testing.T) {
			if !containsString(demoCmd.Long, name) {
				t.Errorf("demo type %q is runnable but missing from demoCmd.Long help text", name)
			}
		})
	}
}

// containsString reports whether substr appears in s. It mirrors the tiny
// helper pattern used elsewhere in the package without adding a dependency.
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOf(s, substr) >= 0)
}

// indexOf returns the first index of substr in s, or -1 when absent.
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
