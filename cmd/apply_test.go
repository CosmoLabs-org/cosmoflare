package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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

func TestApplyCmd_ExactSubcommandCount(t *testing.T) {
	// Should have exactly workers, dns, kv, r2 — no more, no less
	got := len(applyCmd.Commands())
	want := 4
	if got != want {
		names := make([]string, 0, got)
		for _, c := range applyCmd.Commands() {
			names = append(names, c.Name())
		}
		t.Errorf("applyCmd has %d subcommands %v, want %d", got, names, want)
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

func TestApplyCmd_YesFlagType(t *testing.T) {
	f := applyCmd.PersistentFlags().Lookup("yes")
	if f == nil {
		t.Fatal("flag --yes not registered on applyCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--yes type = %q, want %q", f.Value.Type(), "bool")
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

// --- Long help content ---

func TestApplyCmd_LongHelpMentionsSubcommands(t *testing.T) {
	keywords := []string{"workers", "dns", "kv", "r2"}
	for _, kw := range keywords {
		if !strings.Contains(applyCmd.Long, kw) {
			t.Errorf("applyCmd.Long does not mention subcommand %q", kw)
		}
	}
}

func TestApplyCmd_LongHelpMentionsExamples(t *testing.T) {
	if !strings.Contains(applyCmd.Long, "cosmoflare apply") {
		t.Error("applyCmd.Long does not contain usage example")
	}
}

func TestApplyWorkersCmd_LongHelpContent(t *testing.T) {
	if !strings.Contains(applyWorkersCmd.Long, "cosmoflare apply workers") {
		t.Error("applyWorkersCmd.Long does not contain usage example")
	}
}

func TestApplyDNSCmd_LongHelpContent(t *testing.T) {
	if !strings.Contains(applyDNSCmd.Long, "cosmoflare apply dns") {
		t.Error("applyDNSCmd.Long does not contain usage example")
	}
}

func TestApplyKVCmd_LongHelpContent(t *testing.T) {
	if !strings.Contains(applyKVCmd.Long, "cosmoflare apply kv") {
		t.Error("applyKVCmd.Long does not contain usage example")
	}
}

func TestApplyR2Cmd_LongHelpContent(t *testing.T) {
	if !strings.Contains(applyR2Cmd.Long, "cosmoflare apply r2") {
		t.Error("applyR2Cmd.Long does not contain usage example")
	}
}

// --- actionSymbol helper ---

func TestActionSymbol(t *testing.T) {
	cases := []struct {
		action cosmoflare.ApplyAction
		want   string
	}{
		{cosmoflare.ApplyCreate, "+"},
		{cosmoflare.ApplyDelete, "-"},
		{cosmoflare.ApplyUpdate, "~"},
		{cosmoflare.ApplyAction("unknown"), " "},
		{cosmoflare.ApplyAction(""), " "},
	}
	for _, tc := range cases {
		got := actionSymbol(tc.action)
		if got != tc.want {
			t.Errorf("actionSymbol(%q) = %q, want %q", tc.action, got, tc.want)
		}
	}
}

// --- statusSymbol helper ---

func TestStatusSymbol(t *testing.T) {
	cases := []struct {
		status cosmoflare.ApplyStatus
		want   string
	}{
		{cosmoflare.ApplyStatusSuccess, "OK"},
		{cosmoflare.ApplyStatusFailed, "FAIL"},
		{cosmoflare.ApplyStatusSkipped, "SKIP"},
		{cosmoflare.ApplyStatus("unknown"), "?"},
		{cosmoflare.ApplyStatus(""), "?"},
	}
	for _, tc := range cases {
		got := statusSymbol(tc.status)
		if got != tc.want {
			t.Errorf("statusSymbol(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// --- confirmApply helper ---

// confirmApply reads from stdin so we only verify it returns bool without panicking.
// Full integration of stdin injection would require restructuring the function.
func TestConfirmApply_ReturnsBool(t *testing.T) {
	// confirmApply reads from stdin; when stdin is closed/empty it returns false.
	// We just verify it doesn't panic and returns a bool value.
	// (A false return on empty stdin is correct defensive behaviour.)
	result := confirmApply()
	_ = result // bool, acceptable either way with no stdin
}

// --- apply subcommand parent relationship ---

func TestApplyCmd_SubcommandsHaveCorrectParent(t *testing.T) {
	subcmds := []*cobra.Command{applyWorkersCmd, applyDNSCmd, applyKVCmd, applyR2Cmd}
	for _, sub := range subcmds {
		if sub.Parent() == nil {
			t.Errorf("subcommand %q has no parent", sub.Name())
			continue
		}
		if sub.Parent().Use != "apply" {
			t.Errorf("subcommand %q parent = %q, want %q", sub.Name(), sub.Parent().Use, "apply")
		}
	}
}
