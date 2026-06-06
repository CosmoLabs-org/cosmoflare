package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAICmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "ai" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("aiCmd not registered on rootCmd")
	}
}

func TestAICmd_Metadata(t *testing.T) {
	if aiCmd.Use != "ai" {
		t.Errorf("aiCmd.Use = %q, want %q", aiCmd.Use, "ai")
	}
	if aiCmd.Short == "" {
		t.Error("aiCmd.Short is empty")
	}
	if aiCmd.Long == "" {
		t.Error("aiCmd.Long is empty")
	}
}

func TestAICmd_SubcommandGroups(t *testing.T) {
	expected := []string{"models", "run", "gateway"}
	subs := aiCmd.Commands()
	for _, name := range expected {
		found := false
		for _, s := range subs {
			if s.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q on aiCmd", name)
		}
	}
}

func TestAICmd_SubcommandCount(t *testing.T) {
	subs := aiCmd.Commands()
	if len(subs) != 3 {
		t.Errorf("aiCmd has %d subcommands, want 3 (models, run, gateway)", len(subs))
	}
}

func TestAIModelsCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "get"}
	subs := aiModelsCmd.Commands()
	for _, name := range expected {
		found := false
		for _, s := range subs {
			if s.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q on aiModelsCmd", name)
		}
	}
}

func TestAIModelsListCmd_FilterFlag(t *testing.T) {
	f := aiModelsListCmd.Flags().Lookup("filter")
	if f == nil {
		t.Fatal("--filter flag not found on ai models list")
	}
	if f.DefValue != "" {
		t.Errorf("--filter default = %q, want empty", f.DefValue)
	}
}

func TestAIModelsGetCmd_ArgsValidation(t *testing.T) {
	if aiModelsGetCmd.Args == nil {
		t.Fatal("ai models get has nil Args validator")
	}
	if err := aiModelsGetCmd.Args(aiModelsGetCmd, []string{}); err == nil {
		t.Error("expected error with no args for ai models get")
	}
	if err := aiModelsGetCmd.Args(aiModelsGetCmd, []string{"@cf/meta/llama"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	if err := aiModelsGetCmd.Args(aiModelsGetCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for ai models get")
	}
}

func TestAIRunCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"prompt", ""},
		{"system", ""},
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := aiRunCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on ai run", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestAIRunCmd_ArgsValidation(t *testing.T) {
	if aiRunCmd.Args == nil {
		t.Fatal("ai run has nil Args validator")
	}
	if err := aiRunCmd.Args(aiRunCmd, []string{}); err == nil {
		t.Error("expected error with no args for ai run")
	}
	if err := aiRunCmd.Args(aiRunCmd, []string{"@cf/meta/llama"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
}

func TestAIGatewayCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "create", "delete", "logs"}
	subs := aiGatewayCmd.Commands()
	for _, name := range expected {
		found := false
		for _, s := range subs {
			if s.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q on aiGatewayCmd", name)
		}
	}
}

func TestAIGatewayCmd_SubcommandCount(t *testing.T) {
	subs := aiGatewayCmd.Commands()
	if len(subs) != 4 {
		t.Errorf("aiGatewayCmd has %d subcommands, want 4 (list, create, delete, logs)", len(subs))
	}
}

func TestAIGatewayCreateCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"cache-ttl", "0"},
		{"rate-limit", "0"},
		{"rate-window", "60"},
		{"collect-logs", "false"},
	}
	var createCmd *cobra.Command
	for _, sub := range aiGatewayCmd.Commands() {
		if sub.Name() == "create" {
			createCmd = sub
			break
		}
	}
	if createCmd == nil {
		t.Fatal("create subcommand not found")
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := createCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on gateway create", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestAIGatewayCreateCmd_ArgsValidation(t *testing.T) {
	if aiGatewayCreateCmd.Args == nil {
		t.Fatal("gateway create has nil Args validator")
	}
	if err := aiGatewayCreateCmd.Args(aiGatewayCreateCmd, []string{}); err == nil {
		t.Error("expected error with no args for gateway create")
	}
	if err := aiGatewayCreateCmd.Args(aiGatewayCreateCmd, []string{"my-gw"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
}

func TestAIGatewayDeleteCmd_ForceFlag(t *testing.T) {
	var deleteCmd *cobra.Command
	for _, sub := range aiGatewayCmd.Commands() {
		if sub.Name() == "delete" {
			deleteCmd = sub
			break
		}
	}
	if deleteCmd == nil {
		t.Fatal("delete subcommand not found")
	}
	f := deleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found on gateway delete")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestAIGatewayDeleteCmd_ArgsValidation(t *testing.T) {
	if aiGatewayDeleteCmd.Args == nil {
		t.Fatal("gateway delete has nil Args validator")
	}
	if err := aiGatewayDeleteCmd.Args(aiGatewayDeleteCmd, []string{}); err == nil {
		t.Error("expected error with no args for gateway delete")
	}
	if err := aiGatewayDeleteCmd.Args(aiGatewayDeleteCmd, []string{"my-gw"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	if err := aiGatewayDeleteCmd.Args(aiGatewayDeleteCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for gateway delete")
	}
}

func TestAIGatewayLogsCmd_LimitFlag(t *testing.T) {
	var logsCmd *cobra.Command
	for _, sub := range aiGatewayCmd.Commands() {
		if sub.Name() == "logs" {
			logsCmd = sub
			break
		}
	}
	if logsCmd == nil {
		t.Fatal("logs subcommand not found")
	}
	f := logsCmd.Flags().Lookup("limit")
	if f == nil {
		t.Fatal("--limit flag not found on gateway logs")
	}
	if f.DefValue != "25" {
		t.Errorf("--limit default = %q, want %q", f.DefValue, "25")
	}
}

func TestAIGatewayLogsCmd_ArgsValidation(t *testing.T) {
	if aiGatewayLogsCmd.Args == nil {
		t.Fatal("gateway logs has nil Args validator")
	}
	if err := aiGatewayLogsCmd.Args(aiGatewayLogsCmd, []string{}); err == nil {
		t.Error("expected error with no args for gateway logs")
	}
	if err := aiGatewayLogsCmd.Args(aiGatewayLogsCmd, []string{"my-gw"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
}

func TestAICmd_AllRunE(t *testing.T) {
	// Check that all leaf commands have RunE set
	leafCmds := []*cobra.Command{
		aiModelsListCmd,
		aiModelsGetCmd,
		aiRunCmd,
		aiGatewayListCmd,
		aiGatewayCreateCmd,
		aiGatewayDeleteCmd,
		aiGatewayLogsCmd,
	}
	for _, cmd := range leafCmds {
		if cmd.RunE == nil {
			t.Errorf("command %q has nil RunE", cmd.Use)
		}
	}
}
