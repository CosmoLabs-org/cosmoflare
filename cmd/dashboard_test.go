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

// --- RunE handler wired ---

func TestDashboardCmd_RunE(t *testing.T) {
	if dashboardCmd.RunE == nil {
		t.Error("dashboardCmd has nil RunE")
	}
}

func TestDashboardCmd_LongDescription(t *testing.T) {
	if dashboardCmd.Long == "" {
		t.Error("dashboardCmd.Long description is empty")
	}
}

func TestDashboardCmd_NoRun(t *testing.T) {
	// dashboard uses RunE (not Run), verify Run is nil
	if dashboardCmd.Run != nil {
		t.Error("dashboardCmd.Run should be nil — dashboard uses RunE, not Run")
	}
}

func TestDashboardCmd_IntervalFlag(t *testing.T) {
	f := dashboardCmd.Flags().Lookup("interval")
	if f == nil {
		t.Fatal("--interval flag not registered on dashboard command")
	}
	if f.DefValue != "30s" {
		t.Errorf("expected default interval '30s', got %q", f.DefValue)
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
