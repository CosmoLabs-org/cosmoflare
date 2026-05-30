package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestWafCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "waf" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("wafCmd not registered on rootCmd")
	}
}

func TestWafCmd_Metadata(t *testing.T) {
	if wafCmd.Use != "waf" {
		t.Errorf("wafCmd.Use = %q, want %q", wafCmd.Use, "waf")
	}
	if wafCmd.Short == "" {
		t.Error("wafCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestWafCmd_Subcommands(t *testing.T) {
	expected := []string{"packages", "rules", "rule", "access"}
	for _, name := range expected {
		found := false
		for _, sub := range wafCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("waf subcommand %q not registered", name)
		}
	}
}

func TestWafAccessCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "create", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range wafAccessCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("waf access subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestWafCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		wafPackagesCmd,
		wafRulesCmd,
		wafRuleCmd,
		wafAccessListCmd,
		wafAccessCreateCmd,
		wafAccessDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestWafRule_Flags(t *testing.T) {
	if wafRuleCmd.Flags().Lookup("mode") == nil {
		t.Error("flag --mode not registered on wafRuleCmd")
	}
}

func TestWafAccessCreate_Flags(t *testing.T) {
	expected := []string{"ip", "mode", "note"}
	for _, name := range expected {
		if wafAccessCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on wafAccessCreateCmd", name)
		}
	}
}

func TestWafAccessDelete_Flags(t *testing.T) {
	if wafAccessDeleteCmd.Flags().Lookup("force") == nil {
		t.Error("flag --force not registered on wafAccessDeleteCmd")
	}
}

func TestWafCmd_FlagDefaults(t *testing.T) {
	if v, _ := wafRuleCmd.Flags().GetString("mode"); v != "" {
		t.Errorf("waf rule --mode default = %q, want empty", v)
	}
	if v, _ := wafAccessCreateCmd.Flags().GetString("ip"); v != "" {
		t.Errorf("waf access create --ip default = %q, want empty", v)
	}
	if v, _ := wafAccessCreateCmd.Flags().GetString("mode"); v != "" {
		t.Errorf("waf access create --mode default = %q, want empty", v)
	}
	if v, _ := wafAccessCreateCmd.Flags().GetString("note"); v != "" {
		t.Errorf("waf access create --note default = %q, want empty", v)
	}
	if v, _ := wafAccessDeleteCmd.Flags().GetBool("force"); v != false {
		t.Error("waf access delete --force default should be false")
	}
}
