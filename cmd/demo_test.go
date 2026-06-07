package cmd

import (
	"strings"
	"testing"
)

// --- Command registration ---

func TestDemoCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "demo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("demoCmd not registered on rootCmd")
	}
}

func TestDemoCmd_Metadata(t *testing.T) {
	if demoCmd.Use != "demo" {
		t.Errorf("demoCmd.Use = %q, want %q", demoCmd.Use, "demo")
	}
	if demoCmd.Short == "" {
		t.Error("demoCmd.Short is empty")
	}
}

func TestDemoCmd_ShortDescriptionContent(t *testing.T) {
	// Short should mention visual features
	short := strings.ToLower(demoCmd.Short)
	if !strings.Contains(short, "visual") && !strings.Contains(short, "cli") && !strings.Contains(short, "feature") {
		t.Errorf("demoCmd.Short %q does not describe the visual/CLI purpose", demoCmd.Short)
	}
}

// --- Run handler wired ---

func TestDemoCmd_HasRunHandler(t *testing.T) {
	if demoCmd.Run == nil {
		t.Error("demoCmd.Run is nil — no handler wired")
	}
}

// --- Flag registration ---

func TestDemoCmd_Flags(t *testing.T) {
	f := demoCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("flag --type not registered on demoCmd")
	}
	if f.DefValue != "startup" {
		t.Errorf("--type default = %q, want %q", f.DefValue, "startup")
	}
	if f.Shorthand != "t" {
		t.Errorf("--type shorthand = %q, want %q", f.Shorthand, "t")
	}
}

func TestDemoCmd_LongDescription(t *testing.T) {
	if demoCmd.Long == "" {
		t.Error("demoCmd.Long description is empty")
	}
}

func TestDemoCmd_LongDescriptionListsDemoTypes(t *testing.T) {
	expectedTypes := []string{
		"startup", "spinner", "progress", "success",
		"rainbow", "dashboard", "typewriter", "pulse", "random",
	}
	for _, dt := range expectedTypes {
		if !strings.Contains(demoCmd.Long, dt) {
			t.Errorf("demoCmd.Long does not mention demo type %q", dt)
		}
	}
}

func TestDemoCmd_LongDescriptionHasExamples(t *testing.T) {
	if !strings.Contains(demoCmd.Long, "cosmoflare demo") {
		t.Error("demoCmd.Long should contain usage examples with 'cosmoflare demo'")
	}
}

func TestDemoCmd_NoRunE(t *testing.T) {
	// demo uses Run (not RunE)
	if demoCmd.RunE != nil {
		t.Error("demoCmd.RunE should be nil — demo uses Run, not RunE")
	}
}

func TestDemoCmd_TypeFlagIsString(t *testing.T) {
	f := demoCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not found")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--type type = %q, want %q", f.Value.Type(), "string")
	}
}

func TestDemoCmd_TypeFlagUsageText(t *testing.T) {
	f := demoCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not found")
	}
	if f.Usage == "" {
		t.Error("--type flag has empty usage text")
	}
}

func TestDemoCmd_AcceptsAnyArgs(t *testing.T) {
	// demo has no Args validator set, so it accepts any args by default
	if demoCmd.Args != nil {
		if err := demoCmd.Args(demoCmd, []string{}); err != nil {
			t.Errorf("expected demo to accept zero args, got: %v", err)
		}
	}
}

func TestDemoCmd_NoSubcommands(t *testing.T) {
	if len(demoCmd.Commands()) != 0 {
		t.Errorf("demoCmd has %d subcommands, expected 0", len(demoCmd.Commands()))
	}
}

func TestDemoCmd_OnlyTypeFlag(t *testing.T) {
	// demo should only have the --type local flag
	if !demoCmd.HasLocalFlags() {
		t.Error("demoCmd should have at least --type as a local flag")
	}
	if demoCmd.Flags().Lookup("type") == nil {
		t.Error("demoCmd missing --type flag")
	}
}

func TestDemoCmd_TypeFlagDefaultIsValidDemoType(t *testing.T) {
	f := demoCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not found")
	}
	validTypes := map[string]bool{
		"startup": true, "spinner": true, "progress": true,
		"success": true, "rainbow": true, "dashboard": true,
		"typewriter": true, "pulse": true, "random": true,
	}
	if !validTypes[f.DefValue] {
		t.Errorf("--type default %q is not one of the documented demo types", f.DefValue)
	}
}

func TestDemoCmd_HasParentCommand(t *testing.T) {
	if demoCmd.Parent() == nil {
		t.Error("demoCmd has no parent — must be a subcommand of rootCmd")
	}
	if demoCmd.Parent().Name() != rootCmd.Name() {
		t.Errorf("demoCmd parent = %q, want %q", demoCmd.Parent().Name(), rootCmd.Name())
	}
}

func TestDemoCmd_NotHidden(t *testing.T) {
	if demoCmd.Hidden {
		t.Error("demoCmd.Hidden is true — demo should be visible in help")
	}
}

func TestDemoCmd_NoPersistentFlags(t *testing.T) {
	if demoCmd.PersistentFlags().HasFlags() {
		t.Error("demoCmd should not define persistent flags (--type should be local)")
	}
}
