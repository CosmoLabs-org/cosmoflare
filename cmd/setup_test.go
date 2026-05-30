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

// --- Long description content ---

func TestSetupCmd_LongDescription(t *testing.T) {
	if setupCmd.Long == "" {
		t.Fatal("setupCmd.Long is empty")
	}
	// Must mention key topics
	keywords := []string{"setup", "wizard", "profile", "token"}
	for _, kw := range keywords {
		found := false
		lower := setupCmd.Long
		for i := 0; i <= len(lower)-len(kw); i++ {
			match := true
			for j := 0; j < len(kw); j++ {
				c := lower[i+j]
				k := kw[j]
				if c != k && c != k-32 && c != k+32 {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("setupCmd.Long should mention %q", kw)
		}
	}
}

// --- No subcommands ---

func TestSetupCmd_NoSubcommands(t *testing.T) {
	if len(setupCmd.Commands()) != 0 {
		t.Errorf("setupCmd has %d subcommands, expected 0", len(setupCmd.Commands()))
	}
}

// --- Flag types ---

func TestSetupCmd_FlagTypes(t *testing.T) {
	cases := []struct {
		name     string
		wantType string
	}{
		{"profile", "string"},
		{"quiet", "bool"},
		{"skip-test", "bool"},
		{"auto-detect", "bool"},
		{"switch", "bool"},
		{"welcome", "bool"},
		{"backup", "bool"},
		{"restore", "bool"},
	}
	for _, tc := range cases {
		f := setupCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag --%s type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

// --- Flag usage strings ---

func TestSetupCmd_FlagUsageStrings(t *testing.T) {
	flags := []string{"profile", "quiet", "skip-test", "auto-detect", "switch", "welcome", "backup", "restore"}
	for _, name := range flags {
		f := setupCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Usage == "" {
			t.Errorf("flag --%s has empty usage string", name)
		}
	}
}
