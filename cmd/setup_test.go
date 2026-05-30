package cmd

import (
	"testing"
)

// --- Command registration ---

func TestSetupCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "setup" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("setupCmd not registered on rootCmd")
	}
}

func TestSetupCmd_Metadata(t *testing.T) {
	if setupCmd.Use != "setup" {
		t.Errorf("setupCmd.Use = %q, want %q", setupCmd.Use, "setup")
	}
	if setupCmd.Short == "" {
		t.Error("setupCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestSetupCmd_RunE(t *testing.T) {
	if setupCmd.RunE == nil {
		t.Error("setupCmd has nil RunE")
	}
}

// --- Flag registration ---

func TestSetupCmd_Flags(t *testing.T) {
	expected := []string{"profile", "quiet", "skip-test", "auto-detect", "switch", "welcome", "backup", "restore"}
	for _, name := range expected {
		if setupCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on setupCmd", name)
		}
	}
}

func TestSetupCmd_FlagDefaults(t *testing.T) {
	f := setupCmd.Flags()

	if v, _ := f.GetString("profile"); v != "" {
		t.Errorf("--profile default = %q, want empty", v)
	}
	if v, _ := f.GetBool("quiet"); v != false {
		t.Error("--quiet default should be false")
	}
	if v, _ := f.GetBool("skip-test"); v != false {
		t.Error("--skip-test default should be false")
	}
	if v, _ := f.GetBool("auto-detect"); v != true {
		t.Error("--auto-detect default should be true")
	}
	if v, _ := f.GetBool("switch"); v != false {
		t.Error("--switch default should be false")
	}
	if v, _ := f.GetBool("welcome"); v != false {
		t.Error("--welcome default should be false")
	}
	if v, _ := f.GetBool("backup"); v != false {
		t.Error("--backup default should be false")
	}
	if v, _ := f.GetBool("restore"); v != false {
		t.Error("--restore default should be false")
	}
}
