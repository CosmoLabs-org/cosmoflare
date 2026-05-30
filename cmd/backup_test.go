package cmd

import (
	"testing"
)

// --- Command registration ---

func TestBackupCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "backup" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("backupCmd not registered on rootCmd")
	}
}

func TestBackupCmd_Metadata(t *testing.T) {
	if backupCmd.Use != "backup" {
		t.Errorf("backupCmd.Use = %q, want %q", backupCmd.Use, "backup")
	}
	if backupCmd.Short == "" {
		t.Error("backupCmd.Short is empty")
	}
}

// --- No subcommands ---

func TestBackupCmd_NoSubcommands(t *testing.T) {
	if len(backupCmd.Commands()) != 0 {
		t.Errorf("backupCmd has %d subcommands, expected 0", len(backupCmd.Commands()))
	}
}

// --- RunE handler wired ---

func TestBackupCmd_RunE(t *testing.T) {
	if backupCmd.RunE == nil {
		t.Error("backupCmd has nil RunE")
	}
}

// --- Flag registration ---

func TestBackupCmd_Flags(t *testing.T) {
	f := backupCmd.Flags().Lookup("format")
	if f == nil {
		t.Fatal("flag --format not registered on backupCmd")
	}
}

func TestBackupCmd_FlagDefaults(t *testing.T) {
	f := backupCmd.Flags().Lookup("format")
	if f == nil {
		t.Fatal("flag --format not found")
	}
	if f.DefValue != "" {
		t.Errorf("flag --format default = %q, want %q", f.DefValue, "")
	}
}
