package cmd

import (
	"testing"
)

// --- Command registration ---

func TestThemeCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "theme" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("themeCmd not registered on rootCmd")
	}
}

func TestThemeCmd_Metadata(t *testing.T) {
	if themeCmd.Use != "theme" {
		t.Errorf("themeCmd.Use = %q, want %q", themeCmd.Use, "theme")
	}
	if themeCmd.Short == "" {
		t.Error("themeCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestThemeCmd_RunE(t *testing.T) {
	if themeCmd.RunE == nil {
		t.Error("themeCmd has nil RunE")
	}
}

// --- Flag registration ---

func TestThemeCmd_Flags(t *testing.T) {
	expected := []string{"list", "set", "create", "info"}
	for _, name := range expected {
		if themeCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on themeCmd", name)
		}
	}
}

func TestThemeCmd_FlagDefaults(t *testing.T) {
	f := themeCmd.Flags()

	if v, _ := f.GetBool("list"); v != false {
		t.Error("--list default should be false")
	}
	if v, _ := f.GetString("set"); v != "" {
		t.Errorf("--set default = %q, want empty", v)
	}
	if v, _ := f.GetBool("create"); v != false {
		t.Error("--create default should be false")
	}
	if v, _ := f.GetBool("info"); v != false {
		t.Error("--info default should be false")
	}
}
