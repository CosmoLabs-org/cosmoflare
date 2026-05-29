package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestDiffCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "diff" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("diffCmd not registered on rootCmd")
	}
}

func TestDiffCmd_Metadata(t *testing.T) {
	if diffCmd.Use != "diff" {
		t.Errorf("diffCmd.Use = %q, want %q", diffCmd.Use, "diff")
	}
	if diffCmd.Short == "" {
		t.Error("diffCmd.Short is empty")
	}
	if diffCmd.Long == "" {
		t.Error("diffCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestDiffCmd_Subcommands(t *testing.T) {
	expected := []string{"workers", "dns", "kv", "r2"}
	for _, name := range expected {
		found := false
		for _, sub := range diffCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("diff subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestDiffCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		diffCmd,
		diffWorkersCmd,
		diffDNSCmd,
		diffKVCmd,
		diffR2Cmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestDiffCmd_OutputFlag(t *testing.T) {
	f := diffCmd.PersistentFlags().Lookup("output")
	if f == nil {
		t.Fatal("flag --output not registered on diffCmd")
	}
	if f.DefValue != "full" {
		t.Errorf("--output default = %q, want %q", f.DefValue, "full")
	}
}

// --- Subcommand metadata ---

func TestDiffWorkersCmd_Metadata(t *testing.T) {
	if diffWorkersCmd.Use != "workers" {
		t.Errorf("Use = %q, want %q", diffWorkersCmd.Use, "workers")
	}
	if diffWorkersCmd.Short == "" {
		t.Error("Short is empty")
	}
	if diffWorkersCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestDiffDNSCmd_Metadata(t *testing.T) {
	if diffDNSCmd.Use != "dns" {
		t.Errorf("Use = %q, want %q", diffDNSCmd.Use, "dns")
	}
	if diffDNSCmd.Short == "" {
		t.Error("Short is empty")
	}
	if diffDNSCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestDiffKVCmd_Metadata(t *testing.T) {
	if diffKVCmd.Use != "kv" {
		t.Errorf("Use = %q, want %q", diffKVCmd.Use, "kv")
	}
	if diffKVCmd.Short == "" {
		t.Error("Short is empty")
	}
	if diffKVCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestDiffR2Cmd_Metadata(t *testing.T) {
	if diffR2Cmd.Use != "r2" {
		t.Errorf("Use = %q, want %q", diffR2Cmd.Use, "r2")
	}
	if diffR2Cmd.Short == "" {
		t.Error("Short is empty")
	}
	if diffR2Cmd.Long == "" {
		t.Error("Long is empty")
	}
}

// --- Output flag inheritance ---

func TestDiffCmd_OutputFlagInherited(t *testing.T) {
	// PersistentFlags should be visible to subcommands
	subcmds := []*cobra.Command{diffWorkersCmd, diffDNSCmd, diffKVCmd, diffR2Cmd}
	for _, sub := range subcmds {
		// InheritedFlags includes parent's persistent flags
		f := sub.InheritedFlags().Lookup("output")
		if f == nil {
			t.Errorf("subcommand %q does not inherit --output flag", sub.Name())
		}
	}
}
