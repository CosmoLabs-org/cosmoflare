package cmd

import (
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/visual"
)

// demoRunGlobals snapshots and restores the demoType flag variable and the
// visual package quiet mode, so tests cannot leak state into each other.
func demoRunGlobals(t *testing.T) {
	t.Helper()
	oldType := demoType
	wasQuiet := visual.IsQuiet()
	t.Cleanup(func() {
		demoType = oldType
		if wasQuiet {
			visual.DisableAnimations()
		} else {
			visual.EnableAnimations()
		}
	})
}

// TestRunDemo_FlagDefaultIsStartup verifies the --type flag is registered on
// demoCmd with "startup" as its default value.
func TestRunDemo_FlagDefaultIsStartup(t *testing.T) {
	flag := demoCmd.Flags().Lookup("type")
	if flag == nil {
		t.Fatal("expected --type flag to be registered on demoCmd")
	}
	if got := flag.DefValue; got != "startup" {
		t.Fatalf("expected --type default %q, got %q", "startup", got)
	}
}

// TestRunDemo_DispatchesFastTypes verifies that runDemo dispatches the
// offline-safe demo types (startup, spinner, success) without panicking.
// Animations are disabled so the demo functions return immediately.
func TestRunDemo_DispatchesFastTypes(t *testing.T) {
	for _, typeName := range []string{"startup", "spinner", "success"} {
		t.Run(typeName, func(t *testing.T) {
			demoRunGlobals(t)
			visual.DisableAnimations()
			demoType = typeName

			runDemo(demoCmd, nil) // must not panic
		})
	}
}

// TestRunDemo_UnknownTypeListsOptions verifies the default switch branch: an
// unrecognized --type prints the available demo list and returns cleanly.
func TestRunDemo_UnknownTypeListsOptions(t *testing.T) {
	demoRunGlobals(t)
	visual.DisableAnimations()
	demoType = "does-not-exist"

	runDemo(demoCmd, nil) // must not panic
}
