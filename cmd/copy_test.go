package cmd

import (
	"testing"
)

// --- Command registration ---

func TestCopyCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "copy [source] [destination]" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("copyCmd not registered on rootCmd")
	}
}

func TestCopyCmd_Metadata(t *testing.T) {
	if copyCmd.Short == "" {
		t.Error("copyCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestCopyCmd_HasRunE(t *testing.T) {
	if copyCmd.RunE == nil {
		t.Fatal("copyCmd.RunE is nil")
	}
}

// --- Flag registration ---

func TestCopyCmd_BasicFlags(t *testing.T) {
	expected := []string{"recursive", "resume", "verify", "overwrite", "preserve"}
	for _, name := range expected {
		if copyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on copyCmd", name)
		}
	}
}

func TestCopyCmd_ProgressFlags(t *testing.T) {
	expected := []string{"progress", "quiet", "stats"}
	for _, name := range expected {
		if copyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on copyCmd", name)
		}
	}
}

func TestCopyCmd_PerformanceFlags(t *testing.T) {
	expected := []string{"parallel", "chunk-size"}
	for _, name := range expected {
		if copyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on copyCmd", name)
		}
	}
}

func TestCopyCmd_BatchFlags(t *testing.T) {
	expected := []string{"batch", "format"}
	for _, name := range expected {
		if copyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on copyCmd", name)
		}
	}
}

func TestCopyCmd_AdvancedFlags(t *testing.T) {
	expected := []string{"interactive", "dry-run", "no-clobber", "retries", "timeout"}
	for _, name := range expected {
		if copyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on copyCmd", name)
		}
	}
}

// --- Flag defaults ---

func TestCopyCmd_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"recursive", "false"},
		{"resume", "false"},
		{"verify", "true"},
		{"overwrite", "false"},
		{"preserve", "true"},
		{"progress", "true"},
		{"quiet", "false"},
		{"stats", "false"},
		{"parallel", "4"},
		{"chunk-size", "8MB"},
		{"batch", "false"},
		{"format", "table"},
		{"interactive", "true"},
		{"dry-run", "false"},
		{"no-clobber", "false"},
		{"retries", "3"},
		{"timeout", "30m"},
	}
	for _, tc := range cases {
		f := copyCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Flag shorthand checks ---

func TestCopyCmd_FlagShorthands(t *testing.T) {
	shorthands := []struct {
		flag   string
		short  string
	}{
		{"recursive", "r"},
		{"verify", "V"},
		{"overwrite", "o"},
		{"preserve", "p"},
		{"progress", "P"},
		{"quiet", "q"},
		{"stats", "s"},
		{"parallel", "j"},
		{"batch", "b"},
	}
	for _, tc := range shorthands {
		f := copyCmd.Flags().Lookup(tc.flag)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.flag)
		}
		if f.Shorthand != tc.short {
			t.Errorf("flag --%s shorthand = %q, want %q", tc.flag, f.Shorthand, tc.short)
		}
	}
}

// --- Arg validation ---

func TestCopyCmd_NoArgs(t *testing.T) {
	err := runCopy(copyCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestCopyCmd_OneArg(t *testing.T) {
	err := runCopy(copyCmd, []string{"source.txt"})
	if err == nil {
		t.Fatal("expected error when only source provided (missing destination)")
	}
}
