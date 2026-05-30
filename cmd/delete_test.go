package cmd

import (
	"testing"
)

// --- Command registration ---

func TestDeleteCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "delete" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("deleteCmd not registered on rootCmd")
	}
}

func TestDeleteCmd_Metadata(t *testing.T) {
	if deleteCmd.Use != "delete <bucket-name>" {
		t.Errorf("deleteCmd.Use = %q, want %q", deleteCmd.Use, "delete <bucket-name>")
	}
	if deleteCmd.Short == "" {
		t.Error("deleteCmd.Short is empty")
	}
}

// --- Run handler wired ---

func TestDeleteCmd_HasRunHandler(t *testing.T) {
	if deleteCmd.Run == nil {
		t.Error("deleteCmd.Run is nil — no handler wired")
	}
}

// --- Flag registration ---

func TestDeleteCmd_Flags(t *testing.T) {
	f := deleteCmd.Flags().Lookup("confirm")
	if f == nil {
		t.Fatal("flag --confirm not registered on deleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--confirm default = %q, want %q", f.DefValue, "false")
	}
}

// --- Args validation ---

func TestDeleteCmd_ArgsValidation(t *testing.T) {
	if deleteCmd.Args == nil {
		t.Fatal("deleteCmd.Args validator is nil")
	}

	// No args should fail
	if err := deleteCmd.Args(deleteCmd, []string{}); err == nil {
		t.Error("expected error with no args, got nil")
	}

	// One arg should succeed
	if err := deleteCmd.Args(deleteCmd, []string{"my-bucket"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}

	// Two args should fail
	if err := deleteCmd.Args(deleteCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args, got nil")
	}
}

func TestDeleteCmd_LongDescription(t *testing.T) {
	if deleteCmd.Long == "" {
		t.Error("deleteCmd.Long description is empty")
	}
}

func TestDeleteCmd_ConfirmFlagType(t *testing.T) {
	f := deleteCmd.Flags().Lookup("confirm")
	if f == nil {
		t.Fatal("--confirm flag not found")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--confirm type = %q, want %q", f.Value.Type(), "bool")
	}
}

func TestDeleteCmd_UsesRunNotRunE(t *testing.T) {
	if deleteCmd.Run == nil {
		t.Error("deleteCmd.Run should not be nil")
	}
	if deleteCmd.RunE != nil {
		t.Error("deleteCmd uses RunE instead of Run")
	}
}

func TestDeleteCmd_ArgsErrorMessage(t *testing.T) {
	if deleteCmd.Args == nil {
		t.Fatal("deleteCmd.Args validator is nil")
	}
	err := deleteCmd.Args(deleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error with no args")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
}

func TestDeleteCmd_NoSubcommands(t *testing.T) {
	if len(deleteCmd.Commands()) != 0 {
		t.Errorf("deleteCmd has %d subcommands, expected 0", len(deleteCmd.Commands()))
	}
}

func TestDeleteCmd_ThreeArgsFail(t *testing.T) {
	if deleteCmd.Args == nil {
		t.Fatal("deleteCmd.Args validator is nil")
	}
	err := deleteCmd.Args(deleteCmd, []string{"a", "b", "c"})
	if err == nil {
		t.Error("expected error with three args, got nil")
	}
}
