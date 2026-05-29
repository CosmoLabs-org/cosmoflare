package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestCostCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "cost" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("costCmd not registered on rootCmd")
	}
}

func TestCostCmd_Metadata(t *testing.T) {
	if costCmd.Use != "cost" {
		t.Errorf("costCmd.Use = %q, want %q", costCmd.Use, "cost")
	}
	if costCmd.Short == "" {
		t.Error("costCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestCostCmd_Subcommands(t *testing.T) {
	expected := []string{"r2", "workers", "kv", "detail"}
	for _, name := range expected {
		found := false
		for _, sub := range costCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cost subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestCostCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		costCmd,
		costR2Cmd,
		costWorkersCmd,
		costKVCmd,
		costDetailCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestCostCmd_PeriodFlag(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("period")
	if f == nil {
		t.Fatal("--period flag not registered on costCmd")
	}
	if f.DefValue != "30d" {
		t.Errorf("--period default = %q, want %q", f.DefValue, "30d")
	}
}

func TestCostCmd_FormatFlag(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not registered on costCmd")
	}
	if f.DefValue != "table" {
		t.Errorf("--format default = %q, want %q", f.DefValue, "table")
	}
}

// --- Flag inheritance ---

func TestCostCmd_SubcommandsInheritFlags(t *testing.T) {
	subs := []*cobra.Command{costR2Cmd, costWorkersCmd, costKVCmd, costDetailCmd}
	for _, sub := range subs {
		pf := sub.InheritedFlags()
		if pf.Lookup("period") == nil {
			t.Errorf("subcommand %q does not inherit --period flag", sub.Name())
		}
		if pf.Lookup("format") == nil {
			t.Errorf("subcommand %q does not inherit --format flag", sub.Name())
		}
	}
}

// --- Period validation ---

func TestCostCmd_ValidatePeriod(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"7d", true},
		{"30d", true},
		{"90d", true},
		{"1d", false},
		{"365d", false},
		{"", false},
		{"invalid", false},
	}
	for _, tc := range cases {
		err := validatePeriod(tc.input)
		if tc.valid && err != nil {
			t.Errorf("validatePeriod(%q) returned error %v, want nil", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validatePeriod(%q) returned nil, want error", tc.input)
		}
	}
}

// --- Format validation ---

func TestCostCmd_ValidateFormat(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"table", true},
		{"json", true},
		{"csv", true},
		{"xml", false},
		{"", false},
		{"yaml", false},
	}
	for _, tc := range cases {
		err := validateCostFormat(tc.input)
		if tc.valid && err != nil {
			t.Errorf("validateCostFormat(%q) returned error %v, want nil", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validateCostFormat(%q) returned nil, want error", tc.input)
		}
	}
}
