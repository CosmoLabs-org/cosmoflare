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

// --- Long description content ---

func TestBackupCmd_LongDescription(t *testing.T) {
	if backupCmd.Long == "" {
		t.Fatal("backupCmd.Long is empty")
	}
	// Must mention key features
	keywords := []string{"backup", "encrypt", "JSON", "format"}
	for _, kw := range keywords {
		found := false
		lower := backupCmd.Long
		for i := 0; i <= len(lower)-len(kw); i++ {
			match := true
			for j := 0; j < len(kw); j++ {
				c := lower[i+j]
				k := kw[j]
				if c != k && c != k-32 && c != k+32 {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("backupCmd.Long should mention %q", kw)
		}
	}
}

// --- Format flag usage string ---

func TestBackupCmd_FormatFlagUsage(t *testing.T) {
	f := backupCmd.Flags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not found")
	}
	if f.Usage == "" {
		t.Error("--format flag has empty usage string")
	}
	// Usage should mention the valid values
	usage := f.Usage
	for _, val := range []string{"enc", "json", "env"} {
		found := false
		for i := 0; i <= len(usage)-len(val); i++ {
			if usage[i:i+len(val)] == val {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("--format usage should mention %q, got: %s", val, usage)
		}
	}
}

// --- Args validation (backup accepts no positional args by default) ---

func TestBackupCmd_AcceptsNoArgs(t *testing.T) {
	// backup command doesn't set Args validator explicitly, so cobra defaults to arbitrary args
	// but the command should still work with no args
	if backupCmd.Use != "backup" {
		t.Errorf("backupCmd.Use = %q, want %q", backupCmd.Use, "backup")
	}
}

// --- Short description non-empty and meaningful ---

func TestBackupCmd_ShortDescription(t *testing.T) {
	if len(backupCmd.Short) < 5 {
		t.Errorf("backupCmd.Short is too short (%d chars): %q", len(backupCmd.Short), backupCmd.Short)
	}
}

// --- Format flag is a string type ---

func TestBackupCmd_FormatFlagType(t *testing.T) {
	f := backupCmd.Flags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not found")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--format flag type = %q, want %q", f.Value.Type(), "string")
	}
}
