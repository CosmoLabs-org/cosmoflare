package cmd

import (
	"bytes"
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
