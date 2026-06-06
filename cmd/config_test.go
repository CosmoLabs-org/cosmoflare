package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestConfigCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("configCmd not registered on rootCmd")
	}
}

func TestConfigCmd_Metadata(t *testing.T) {
	if configCmd.Use != "config" {
		t.Errorf("configCmd.Use = %q, want %q", configCmd.Use, "config")
	}
	if configCmd.Short == "" {
		t.Error("configCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestConfigCmd_Subcommands(t *testing.T) {
	expected := []string{"init", "validate", "list", "show", "set", "delete", "switch", "export"}
	for _, name := range expected {
		found := false
		for _, sub := range configCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("config subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestConfigCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		configInitCmd,
		configValidateCmd,
		configListCmd,
		configShowCmd,
		configSetCmd,
		configDeleteCmd,
		configSwitchCmd,
		configExportCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestConfigSet_Flags(t *testing.T) {
	expected := []string{"description", "account-id", "api-token", "endpoint", "access-key", "secret-key", "region", "interactive", "test-connection"}
	for _, name := range expected {
		if configSetCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on configSetCmd", name)
		}
	}
}

func TestConfigShow_Flags(t *testing.T) {
	f := configShowCmd.Flags().Lookup("show-secrets")
	if f == nil {
		t.Fatal("--show-secrets flag not registered on configShowCmd")
	}
}

func TestConfigSet_RequiredFlags(t *testing.T) {
	f := configSetCmd.Flags().Lookup("account-id")
	if f == nil {
		t.Fatal("--account-id flag not found on configSetCmd")
	}
}

// --- Flag defaults ---

func TestConfigSet_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"region", "auto"},
		{"interactive", "true"},
		{"test-connection", "false"},
	}
	for _, tc := range cases {
		f := configSetCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestConfigShow_FlagDefaults(t *testing.T) {
	f := configShowCmd.Flags().Lookup("show-secrets")
	if f == nil {
		t.Fatal("--show-secrets flag not found on configShowCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--show-secrets default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestConfigSet_NoProfileName(t *testing.T) {
	err := runConfigSet(configSetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no profile name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("profile name")) {
		t.Errorf("error = %q, want it to mention 'profile name'", err.Error())
	}
}

func TestConfigSwitch_NoArgs(t *testing.T) {
	err := runConfigSwitch(configSwitchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no profile name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("profile name")) {
		t.Errorf("error = %q, want it to mention 'profile name'", err.Error())
	}
}

func TestConfigDelete_NoArgs(t *testing.T) {
	err := runConfigDelete(configDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no profile name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("profile name")) {
		t.Errorf("error = %q, want it to mention 'profile name'", err.Error())
	}
}

// --- Flag variable wiring ---

func TestConfigFlagsParsing_AccountID(t *testing.T) {
	cmd := &cobra.Command{}
	var accountID string
	cmd.Flags().StringVar(&accountID, "account-id", "", "")
	if err := cmd.Flags().Set("account-id", "testaccount123"); err != nil {
		t.Fatalf("failed to set --account-id: %v", err)
	}
	if accountID != "testaccount123" {
		t.Errorf("accountID = %q, want %q", accountID, "testaccount123")
	}
}

func TestConfigFlagsParsing_Region(t *testing.T) {
	cmd := &cobra.Command{}
	var region string
	cmd.Flags().StringVar(&region, "region", "auto", "")
	if region != "auto" {
		t.Errorf("region default = %q, want %q", region, "auto")
	}
	if err := cmd.Flags().Set("region", "us-east-1"); err != nil {
		t.Fatalf("failed to set --region: %v", err)
	}
	if region != "us-east-1" {
		t.Errorf("region = %q, want %q", region, "us-east-1")
	}
}

func TestConfigFlagsParsing_Interactive(t *testing.T) {
	cmd := &cobra.Command{}
	var interactive bool
	cmd.Flags().BoolVar(&interactive, "interactive", true, "")
	if !interactive {
		t.Error("interactive should default to true")
	}
	if err := cmd.Flags().Set("interactive", "false"); err != nil {
		t.Fatalf("failed to set --interactive: %v", err)
	}
	if interactive {
		t.Error("interactive should be false after --interactive=false")
	}
}

// --- Long description content ---

func TestConfigCmd_LongContainsSubcommands(t *testing.T) {
	subcommands := []string{"init", "validate", "list", "show", "set", "delete", "switch", "export"}
	for _, sub := range subcommands {
		if !strings.Contains(configCmd.Long, sub) {
			t.Errorf("configCmd.Long does not mention subcommand %q", sub)
		}
	}
}

func TestConfigCmd_LongContainsProfilesPath(t *testing.T) {
	if !strings.Contains(configCmd.Long, "config.yaml") {
		t.Error("configCmd.Long does not mention config.yaml storage path")
	}
}

// --- Use patterns with optional args ---

func TestConfigValidateCmd_UsePattern(t *testing.T) {
	if configValidateCmd.Use == "" {
		t.Fatal("configValidateCmd.Use is empty")
	}
	if !strings.Contains(configValidateCmd.Use, "validate") {
		t.Errorf("configValidateCmd.Use = %q, expected to contain 'validate'", configValidateCmd.Use)
	}
}

func TestConfigShowCmd_UsePattern(t *testing.T) {
	if !strings.Contains(configShowCmd.Use, "show") {
		t.Errorf("configShowCmd.Use = %q, expected to contain 'show'", configShowCmd.Use)
	}
}

func TestConfigSetCmd_UsePattern(t *testing.T) {
	if !strings.Contains(configSetCmd.Use, "set") {
		t.Errorf("configSetCmd.Use = %q, expected to contain 'set'", configSetCmd.Use)
	}
}

func TestConfigDeleteCmd_UsePattern(t *testing.T) {
	if !strings.Contains(configDeleteCmd.Use, "delete") {
		t.Errorf("configDeleteCmd.Use = %q, expected to contain 'delete'", configDeleteCmd.Use)
	}
}

func TestConfigSwitchCmd_UsePattern(t *testing.T) {
	if !strings.Contains(configSwitchCmd.Use, "switch") {
		t.Errorf("configSwitchCmd.Use = %q, expected to contain 'switch'", configSwitchCmd.Use)
	}
}

func TestConfigExportCmd_UsePattern(t *testing.T) {
	if !strings.Contains(configExportCmd.Use, "export") {
		t.Errorf("configExportCmd.Use = %q, expected to contain 'export'", configExportCmd.Use)
	}
}

// --- configCmd has no RunE (group command) ---

func TestConfigCmd_NoRunE(t *testing.T) {
	if configCmd.RunE != nil {
		t.Error("configCmd.RunE should be nil (it is a group command)")
	}
	if configCmd.Run != nil {
		t.Error("configCmd.Run should be nil (it is a group command)")
	}
}

// --- Flag type correctness ---

func TestConfigSet_FlagTypes(t *testing.T) {
	boolFlags := []string{"interactive", "test-connection"}
	for _, name := range boolFlags {
		f := configSetCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on configSetCmd", name)
		}
		if f.Value.Type() != "bool" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "bool")
		}
	}

	stringFlags := []string{"description", "account-id", "api-token", "endpoint", "access-key", "secret-key", "region"}
	for _, name := range stringFlags {
		f := configSetCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on configSetCmd", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "string")
		}
	}
}

func TestConfigShow_FlagType(t *testing.T) {
	f := configShowCmd.Flags().Lookup("show-secrets")
	if f == nil {
		t.Fatal("--show-secrets flag not found on configShowCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--show-secrets type = %q, want %q", f.Value.Type(), "bool")
	}
}

// --- Flag usage strings non-empty ---

func TestConfigSet_FlagUsageStrings(t *testing.T) {
	flags := []string{"description", "account-id", "api-token", "endpoint", "access-key", "secret-key", "region", "interactive", "test-connection"}
	for _, name := range flags {
		f := configSetCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on configSetCmd", name)
		}
		if f.Usage == "" {
			t.Errorf("flag --%s has empty usage string", name)
		}
	}
}

func TestConfigShow_FlagUsageString(t *testing.T) {
	f := configShowCmd.Flags().Lookup("show-secrets")
	if f == nil {
		t.Fatal("--show-secrets flag not found on configShowCmd")
	}
	if f.Usage == "" {
		t.Error("--show-secrets has empty usage string")
	}
}

// --- Short descriptions non-empty on all subcommands ---

func TestConfigSubcommands_ShortDescriptions(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"init":     configInitCmd,
		"validate": configValidateCmd,
		"list":     configListCmd,
		"show":     configShowCmd,
		"set":      configSetCmd,
		"delete":   configDeleteCmd,
		"switch":   configSwitchCmd,
		"export":   configExportCmd,
	}
	for name, c := range cmds {
		if c.Short == "" {
			t.Errorf("config %s subcommand has empty Short description", name)
		}
	}
}

// --- Long descriptions non-empty on all subcommands ---

func TestConfigSubcommands_LongDescriptions(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"init":     configInitCmd,
		"validate": configValidateCmd,
		"list":     configListCmd,
		"show":     configShowCmd,
		"set":      configSetCmd,
		"delete":   configDeleteCmd,
		"switch":   configSwitchCmd,
		"export":   configExportCmd,
	}
	for name, c := range cmds {
		if c.Long == "" {
			t.Errorf("config %s subcommand has empty Long description", name)
		}
	}
}

// --- account-id required annotation ---

func TestConfigSet_AccountIDIsRequired(t *testing.T) {
	f := configSetCmd.Flags().Lookup("account-id")
	if f == nil {
		t.Fatal("--account-id flag not found on configSetCmd")
	}
	requiredAnnotation, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	if !ok {
		t.Fatal("--account-id is not marked as required (no cobra.BashCompOneRequiredFlag annotation)")
	}
	if len(requiredAnnotation) == 0 || requiredAnnotation[0] != "true" {
		t.Errorf("--account-id required annotation = %v, want [\"true\"]", requiredAnnotation)
	}
}

// --- configSet error message specifics ---

func TestConfigSet_ErrorMessageContent(t *testing.T) {
	err := runConfigSet(configSetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when profile name is empty")
	}
	msg := err.Error()
	if !strings.Contains(msg, "required") && !strings.Contains(msg, "profile name") {
		t.Errorf("error = %q, expected mention of 'profile name' or 'required'", msg)
	}
}

func TestConfigDelete_ErrorMessageContent(t *testing.T) {
	err := runConfigDelete(configDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	msg := err.Error()
	if !strings.Contains(msg, "required") && !strings.Contains(msg, "profile name") {
		t.Errorf("error = %q, expected mention of 'profile name' or 'required'", msg)
	}
}

func TestConfigSwitch_ErrorMessageContent(t *testing.T) {
	err := runConfigSwitch(configSwitchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	msg := err.Error()
	if !strings.Contains(msg, "required") && !strings.Contains(msg, "profile name") {
		t.Errorf("error = %q, expected mention of 'profile name' or 'required'", msg)
	}
}

// --- configCmd parent/child relationship ---

func TestConfigSubcommands_ParentIsConfigCmd(t *testing.T) {
	cmds := []*cobra.Command{
		configInitCmd,
		configValidateCmd,
		configListCmd,
		configShowCmd,
		configSetCmd,
		configDeleteCmd,
		configSwitchCmd,
		configExportCmd,
	}
	for _, c := range cmds {
		if c.Parent() == nil {
			t.Errorf("command %q has nil parent", c.Use)
			continue
		}
		if c.Parent().Use != "config" {
			t.Errorf("command %q parent.Use = %q, want %q", c.Use, c.Parent().Use, "config")
		}
	}
}

// --- configSet region default value override ---

func TestConfigSet_RegionDefaultIsAuto(t *testing.T) {
	f := configSetCmd.Flags().Lookup("region")
	if f == nil {
		t.Fatal("--region flag not found on configSetCmd")
	}
	if f.DefValue != "auto" {
		t.Errorf("--region default = %q, want %q", f.DefValue, "auto")
	}
	// Ensure the default is a valid non-empty string
	if f.DefValue == "" {
		t.Error("--region default value should not be empty")
	}
}

// --- configSet test-connection default is false ---

func TestConfigSet_TestConnectionDefaultIsFalse(t *testing.T) {
	f := configSetCmd.Flags().Lookup("test-connection")
	if f == nil {
		t.Fatal("--test-connection flag not found on configSetCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--test-connection default = %q, want %q", f.DefValue, "false")
	}
}

// --- configExportCmd Long content ---

func TestConfigExportCmd_LongMentionsEval(t *testing.T) {
	if !strings.Contains(configExportCmd.Long, "eval") && !strings.Contains(configExportCmd.Long, "export") {
		t.Error("configExportCmd.Long should mention eval or export usage")
	}
}

// --- configSetCmd Long content (examples) ---

func TestConfigSetCmd_LongHasExample(t *testing.T) {
	if !strings.Contains(configSetCmd.Long, "cosmoflare") && !strings.Contains(configSetCmd.Long, "Example") {
		t.Error("configSetCmd.Long should contain usage examples")
	}
}
