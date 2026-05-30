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

// --- Long description content ---

func TestThemeCmd_LongDescription(t *testing.T) {
	if themeCmd.Long == "" {
		t.Fatal("themeCmd.Long is empty")
	}
	// Must mention key themes and features
	keywords := []string{"theme", "Cosmic", "Ocean", "Forest"}
	for _, kw := range keywords {
		found := false
		for i := 0; i <= len(themeCmd.Long)-len(kw); i++ {
			if themeCmd.Long[i:i+len(kw)] == kw {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("themeCmd.Long should mention %q", kw)
		}
	}
}

// --- No subcommands ---

func TestThemeCmd_NoSubcommands(t *testing.T) {
	if len(themeCmd.Commands()) != 0 {
		t.Errorf("themeCmd has %d subcommands, expected 0", len(themeCmd.Commands()))
	}
}

// --- Flag types ---

func TestThemeCmd_FlagTypes(t *testing.T) {
	cases := []struct {
		name     string
		wantType string
	}{
		{"list", "bool"},
		{"set", "string"},
		{"create", "bool"},
		{"info", "bool"},
	}
	for _, tc := range cases {
		f := themeCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag --%s type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

// --- Flag usage strings ---

func TestThemeCmd_FlagUsageStrings(t *testing.T) {
	flags := []string{"list", "set", "create", "info"}
	for _, name := range flags {
		f := themeCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Usage == "" {
			t.Errorf("flag --%s has empty usage string", name)
		}
	}
}

// --- Short description quality ---

func TestThemeCmd_ShortDescriptionQuality(t *testing.T) {
	if len(themeCmd.Short) < 5 {
		t.Errorf("themeCmd.Short is too short (%d chars): %q", len(themeCmd.Short), themeCmd.Short)
	}
}
