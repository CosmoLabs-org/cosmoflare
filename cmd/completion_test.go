package cmd

import (
	"testing"
)

// --- Command registration ---

func TestCompletionCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "completion" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("completionCmd not registered on rootCmd")
	}
}

func TestCompletionCmd_Metadata(t *testing.T) {
	if completionCmd.Use != "completion [bash|zsh|fish|powershell]" {
		t.Errorf("completionCmd.Use = %q, want %q", completionCmd.Use, "completion [bash|zsh|fish|powershell]")
	}
	if completionCmd.Short == "" {
		t.Error("completionCmd.Short is empty")
	}
}

// --- No subcommands ---

func TestCompletionCmd_NoSubcommands(t *testing.T) {
	if len(completionCmd.Commands()) != 0 {
		t.Errorf("completionCmd has %d subcommands, expected 0", len(completionCmd.Commands()))
	}
}

// --- Run handler wired ---

func TestCompletionCmd_Run(t *testing.T) {
	if completionCmd.Run == nil {
		t.Error("completionCmd has nil Run")
	}
}

// --- ValidArgs ---

func TestCompletionCmd_ValidArgs(t *testing.T) {
	expected := []string{"bash", "zsh", "fish", "powershell"}
	if len(completionCmd.ValidArgs) != len(expected) {
		t.Fatalf("completionCmd.ValidArgs length = %d, want %d", len(completionCmd.ValidArgs), len(expected))
	}
	for i, want := range expected {
		if completionCmd.ValidArgs[i] != want {
			t.Errorf("completionCmd.ValidArgs[%d] = %q, want %q", i, completionCmd.ValidArgs[i], want)
		}
	}
}

// --- Flag registration ---

func TestCompletionCmd_Flags(t *testing.T) {
	f := completionCmd.Flags().Lookup("no-descriptions")
	if f == nil {
		t.Fatal("flag --no-descriptions not registered on completionCmd")
	}
}

func TestCompletionCmd_FlagDefaults(t *testing.T) {
	f := completionCmd.Flags().Lookup("no-descriptions")
	if f == nil {
		t.Fatal("flag --no-descriptions not found")
	}
	if f.DefValue != "false" {
		t.Errorf("flag --no-descriptions default = %q, want %q", f.DefValue, "false")
	}
}

// --- DisableFlagsInUseLine ---

func TestCompletionCmd_DisableFlagsInUseLine(t *testing.T) {
	if !completionCmd.DisableFlagsInUseLine {
		t.Error("completionCmd.DisableFlagsInUseLine should be true")
	}
}
