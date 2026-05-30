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
