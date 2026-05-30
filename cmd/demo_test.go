package cmd

import (
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
