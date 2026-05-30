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

func TestSwitchCmd_LongDescription(t *testing.T) {
	if switchCmd.Long == "" {
		t.Error("switchCmd.Long description is empty")
	}
}

func TestSwitchCmd_DetailsFlagType(t *testing.T) {
	f := switchCmd.Flags().Lookup("details")
	if f == nil {
		t.Fatal("--details flag not found")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--details type = %q, want %q", f.Value.Type(), "bool")
	}
}

func TestSwitchCmd_DeleteFlagType(t *testing.T) {
	f := switchCmd.Flags().Lookup("delete")
	if f == nil {
		t.Fatal("--delete flag not found")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--delete type = %q, want %q", f.Value.Type(), "bool")
	}
}

func TestSwitchCmd_NoSubcommands(t *testing.T) {
	if len(switchCmd.Commands()) != 0 {
		t.Errorf("switchCmd has %d subcommands, expected 0", len(switchCmd.Commands()))
	}
}

func TestSwitchCmd_AcceptsAnyArgs(t *testing.T) {
	// switch has no Args validator set
	if switchCmd.Args != nil {
		if err := switchCmd.Args(switchCmd, []string{}); err != nil {
			t.Errorf("expected switch to accept zero args, got: %v", err)
		}
	}
}
