package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestPageRulesCmd_NotNil(t *testing.T) {
	if pagerulesCmd == nil {
		t.Fatal("pagerulesCmd must not be nil")
	}
}

func TestPageRulesCmd_Metadata(t *testing.T) {
	if pagerulesCmd.Use != "pagerules" {
		t.Errorf("pagerulesCmd.Use = %q, want %q", pagerulesCmd.Use, "pagerules")
	}
	if pagerulesCmd.Short == "" {
		t.Error("pagerulesCmd.Short must not be empty")
	}
}

func TestPageRulesCmd_RegisteredOnRoot(t *testing.T) {
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "pagerules" {
			return
		}
	}
	t.Error("pagerulesCmd not found in rootCmd.Commands()")
}

func TestPageRulesCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "get", "create", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range pagerulesCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not registered on pagerulesCmd", name)
		}
	}
}

// --- RunE handlers ---

func TestPageRulesSubcommands_HaveRunE(t *testing.T) {
	cmds := []struct {
		name string
		runE func(cmd *cobra.Command, args []string) error
	}{
		{"list", pagerulesListCmd.RunE},
		{"get", pagerulesGetCmd.RunE},
		{"create", pagerulesCreateCmd.RunE},
		{"update", pagerulesUpdateCmd.RunE},
		{"delete", pagerulesDeleteCmd.RunE},
	}
	for _, c := range cmds {
		if c.runE == nil {
			t.Errorf("%s.RunE must not be nil", c.name)
		}
	}
}

// --- Create flags ---

func TestPageRulesCreateCmd_Flags(t *testing.T) {
	expected := []string{"url", "action", "action-value", "status", "priority"}
	for _, name := range expected {
		if pagerulesCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on create command", name)
		}
	}

	// Verify required flags
	requiredFlags := []string{"url", "action"}
	for _, name := range requiredFlags {
		f := pagerulesCreateCmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		// Check the annotation for required
		annotations := pagerulesCreateCmd.Flags().Lookup(name).Annotations
		if annotations == nil {
			t.Errorf("flag --%s should be marked required on create command", name)
		}
	}
}

func TestPageRulesCreateCmd_FlagDefaults(t *testing.T) {
	flags := pagerulesCreateCmd.Flags()

	stringCases := []struct {
		name string
		want string
	}{
		{"url", ""},
		{"action", ""},
		{"action-value", ""},
		{"status", "active"},
	}
	for _, tc := range stringCases {
		f := flags.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag --%s not registered", tc.name)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}

	priorityFlag := flags.Lookup("priority")
	if priorityFlag == nil {
		t.Error("flag --priority not registered")
	} else if priorityFlag.DefValue != "1" {
		t.Errorf("flag --priority default = %q, want %q", priorityFlag.DefValue, "1")
	}
}

// --- Update flags ---

func TestPageRulesUpdateCmd_Flags(t *testing.T) {
	expected := []string{"url", "action", "action-value", "status", "priority"}
	for _, name := range expected {
		if pagerulesUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on update command", name)
		}
	}
}

// --- Delete flags ---

func TestPageRulesDeleteCmd_Flags(t *testing.T) {
	f := pagerulesDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Error("flag --force not registered on delete command")
	} else if f.DefValue != "false" {
		t.Errorf("flag --force default = %q, want %q", f.DefValue, "false")
	}
}

// --- List command no-args validation ---

func TestPageRulesList_NoArgs(t *testing.T) {
	err := runPageRulesList(pagerulesListCmd, []string{})
	if err == nil {
		t.Error("expected error when no zone ID provided")
	}
}

// --- Get command no-args validation ---

func TestPageRulesGet_NoArgs(t *testing.T) {
	err := runPageRulesGet(pagerulesGetCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided")
	}

	err = runPageRulesGet(pagerulesGetCmd, []string{"zone123"})
	if err == nil {
		t.Error("expected error when only zone ID provided")
	}
}

// --- Create command no-args validation ---

func TestPageRulesCreate_NoArgs(t *testing.T) {
	err := runPageRulesCreate(pagerulesCreateCmd, []string{})
	if err == nil {
		t.Error("expected error when no zone ID provided")
	}
}

// --- Update command no-args validation ---

func TestPageRulesUpdate_NoArgs(t *testing.T) {
	err := runPageRulesUpdate(pagerulesUpdateCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided")
	}

	err = runPageRulesUpdate(pagerulesUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Error("expected error when only zone ID provided")
	}
}

// --- Delete command no-args validation ---

func TestPageRulesDelete_NoArgs(t *testing.T) {
	err := runPageRulesDelete(pagerulesDeleteCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided")
	}

	err = runPageRulesDelete(pagerulesDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Error("expected error when only zone ID provided")
	}
}
