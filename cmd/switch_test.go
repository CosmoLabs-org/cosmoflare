package cmd

import (
	"testing"
)

// --- Command registration ---

func TestSwitchCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "switch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("switchCmd not registered on rootCmd")
	}
}

func TestSwitchCmd_Metadata(t *testing.T) {
	if switchCmd.Use != "switch" {
		t.Errorf("switchCmd.Use = %q, want %q", switchCmd.Use, "switch")
	}
	if switchCmd.Short == "" {
		t.Error("switchCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestSwitchCmd_RunE(t *testing.T) {
	if switchCmd.RunE == nil {
		t.Error("switchCmd has nil RunE")
	}
}

// --- Flag registration ---

func TestSwitchCmd_Flags(t *testing.T) {
	expected := []string{"details", "delete"}
	for _, name := range expected {
		if switchCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on switchCmd", name)
		}
	}
}

func TestSwitchCmd_FlagDefaults(t *testing.T) {
	f := switchCmd.Flags()

	if v, _ := f.GetBool("details"); v != false {
		t.Error("--details default should be false")
	}
	if v, _ := f.GetBool("delete"); v != false {
		t.Error("--delete default should be false")
	}
}
