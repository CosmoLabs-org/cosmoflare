package cmd

import (
	"testing"
)

// --- Command registration ---

func TestDashboardCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "dashboard" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("dashboardCmd not registered on rootCmd")
	}
}

func TestDashboardCmd_Metadata(t *testing.T) {
	if dashboardCmd.Use != "dashboard" {
		t.Errorf("dashboardCmd.Use = %q, want %q", dashboardCmd.Use, "dashboard")
	}
	if dashboardCmd.Short == "" {
		t.Error("dashboardCmd.Short is empty")
	}
}

// --- No subcommands ---

func TestDashboardCmd_NoSubcommands(t *testing.T) {
	if len(dashboardCmd.Commands()) != 0 {
		t.Errorf("dashboardCmd has %d subcommands, expected 0", len(dashboardCmd.Commands()))
	}
}

// --- Run handler wired ---

func TestDashboardCmd_Run(t *testing.T) {
	if dashboardCmd.Run == nil {
		t.Error("dashboardCmd has nil Run")
	}
}

func TestDashboardCmd_LongDescription(t *testing.T) {
	if dashboardCmd.Long == "" {
		t.Error("dashboardCmd.Long description is empty")
	}
}

func TestDashboardCmd_NoRunE(t *testing.T) {
	// dashboard uses Run (not RunE), verify RunE is nil
	if dashboardCmd.RunE != nil {
		t.Error("dashboardCmd.RunE should be nil — dashboard uses Run, not RunE")
	}
}

func TestDashboardCmd_NoLocalFlags(t *testing.T) {
	// dashboard command has no local flags (only inherits persistent from root)
	// No local flags are defined in dashboard init()
	if dashboardCmd.HasLocalFlags() {
		t.Error("dashboardCmd should not have local flags")
	}
}

func TestDashboardCmd_AcceptsAnyArgs(t *testing.T) {
	// dashboard has no Args validator set, so it accepts any args by default
	if dashboardCmd.Args != nil {
		// If Args is set, make sure it still allows zero args
		if err := dashboardCmd.Args(dashboardCmd, []string{}); err != nil {
			t.Errorf("expected dashboard to accept zero args, got: %v", err)
		}
	}
}
