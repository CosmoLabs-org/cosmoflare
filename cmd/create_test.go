package cmd

import (
	"testing"
)

// --- Command registration ---

func TestCreateCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "create" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("createCmd not registered on rootCmd")
	}
}

func TestCreateCmd_Metadata(t *testing.T) {
	if createCmd.Use != "create <bucket-name>" {
		t.Errorf("createCmd.Use = %q, want %q", createCmd.Use, "create <bucket-name>")
	}
	if createCmd.Short == "" {
		t.Error("createCmd.Short is empty")
	}
}

// --- No subcommands ---

func TestCreateCmd_NoSubcommands(t *testing.T) {
	if len(createCmd.Commands()) != 0 {
		t.Errorf("createCmd has %d subcommands, expected 0", len(createCmd.Commands()))
	}
}

// --- Run handler wired ---

func TestCreateCmd_Run(t *testing.T) {
	if createCmd.Run == nil {
		t.Error("createCmd has nil Run")
	}
}

// --- Args validation ---

func TestCreateCmd_ArgsFunction(t *testing.T) {
	if createCmd.Args == nil {
		t.Error("createCmd has nil Args function")
	}
}

func TestCreateCmd_ArgsRejectsEmpty(t *testing.T) {
	err := createCmd.Args(createCmd, []string{})
	if err == nil {
		t.Error("createCmd.Args should reject empty args")
	}
}

func TestCreateCmd_ArgsAcceptsOne(t *testing.T) {
	err := createCmd.Args(createCmd, []string{"my-bucket"})
	if err != nil {
		t.Errorf("createCmd.Args should accept one arg, got error: %v", err)
	}
}

func TestCreateCmd_ArgsRejectsTwo(t *testing.T) {
	err := createCmd.Args(createCmd, []string{"bucket1", "bucket2"})
	if err == nil {
		t.Error("createCmd.Args should reject two args")
	}
}
