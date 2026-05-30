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
