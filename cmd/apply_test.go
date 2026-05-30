package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestApplyCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "apply" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("applyCmd not registered on rootCmd")
	}
}

func TestApplyCmd_Metadata(t *testing.T) {
	if applyCmd.Use != "apply" {
		t.Errorf("applyCmd.Use = %q, want %q", applyCmd.Use, "apply")
	}
	if applyCmd.Short == "" {
		t.Error("applyCmd.Short is empty")
	}
	if applyCmd.Long == "" {
		t.Error("applyCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestApplyCmd_Subcommands(t *testing.T) {
	expected := []string{"workers", "dns", "kv", "r2"}
	for _, name := range expected {
		found := false
		for _, sub := range applyCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("apply subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestApplyCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		applyCmd,
		applyWorkersCmd,
		applyDNSCmd,
		applyKVCmd,
		applyR2Cmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestApplyCmd_YesFlag(t *testing.T) {
	f := applyCmd.PersistentFlags().Lookup("yes")
	if f == nil {
		t.Fatal("flag --yes not registered on applyCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default = %q, want %q", f.DefValue, "false")
	}
	if f.Shorthand != "y" {
		t.Errorf("--yes shorthand = %q, want %q", f.Shorthand, "y")
	}
}

// --- Subcommand metadata ---

func TestApplyWorkersCmd_Metadata(t *testing.T) {
	if applyWorkersCmd.Use != "workers" {
		t.Errorf("Use = %q, want %q", applyWorkersCmd.Use, "workers")
	}
	if applyWorkersCmd.Short == "" {
		t.Error("Short is empty")
	}
	if applyWorkersCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestApplyDNSCmd_Metadata(t *testing.T) {
	if applyDNSCmd.Use != "dns" {
		t.Errorf("Use = %q, want %q", applyDNSCmd.Use, "dns")
	}
	if applyDNSCmd.Short == "" {
		t.Error("Short is empty")
	}
	if applyDNSCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestApplyKVCmd_Metadata(t *testing.T) {
	if applyKVCmd.Use != "kv" {
		t.Errorf("Use = %q, want %q", applyKVCmd.Use, "kv")
	}
	if applyKVCmd.Short == "" {
		t.Error("Short is empty")
	}
	if applyKVCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestApplyR2Cmd_Metadata(t *testing.T) {
	if applyR2Cmd.Use != "r2" {
		t.Errorf("Use = %q, want %q", applyR2Cmd.Use, "r2")
	}
	if applyR2Cmd.Short == "" {
		t.Error("Short is empty")
	}
	if applyR2Cmd.Long == "" {
		t.Error("Long is empty")
	}
}

// --- Yes flag inheritance ---

func TestApplyCmd_YesFlagInherited(t *testing.T) {
	subcmds := []*cobra.Command{applyWorkersCmd, applyDNSCmd, applyKVCmd, applyR2Cmd}
	for _, sub := range subcmds {
		f := sub.InheritedFlags().Lookup("yes")
		if f == nil {
			t.Errorf("subcommand %q does not inherit --yes flag", sub.Name())
		}
	}
}
