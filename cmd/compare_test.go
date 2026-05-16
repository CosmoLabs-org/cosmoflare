package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestCompareCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "compare [source-bucket] [dest-bucket]" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("compareCmd not registered on rootCmd")
	}
}

func TestCompareCmd_Metadata(t *testing.T) {
	if compareCmd.Short == "" {
		t.Error("compareCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestCompareCmd_HasRunE(t *testing.T) {
	if compareCmd.RunE == nil {
		t.Error("compareCmd.RunE is nil")
	}
}

// --- Flag registration ---

func TestCompareCmd_Flags(t *testing.T) {
	f := compareCmd.Flags().Lookup("prefix")
	if f == nil {
		t.Fatal("--prefix flag not registered on compareCmd")
	}
}

func TestCompareCmd_FlagDefaults(t *testing.T) {
	f := compareCmd.Flags().Lookup("prefix")
	if f == nil {
		t.Fatal("--prefix flag not found on compareCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--prefix default = %q, want empty string", f.DefValue)
	}
}

// --- Arg validation ---

func TestCompare_NoArgs(t *testing.T) {
	err := runCompare(compareCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket args provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("bucket")) {
		t.Errorf("error = %q, want it to mention 'bucket'", err.Error())
	}
}

func TestCompare_OneArg(t *testing.T) {
	err := runCompare(compareCmd, []string{"src-bucket"})
	if err == nil {
		t.Fatal("expected error when only one bucket arg provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("bucket")) {
		t.Errorf("error = %q, want it to mention 'bucket'", err.Error())
	}
}

// --- Flag variable wiring ---

func TestCompareFlagsParsing_Prefix(t *testing.T) {
	cmd := &cobra.Command{}
	var prefix string
	cmd.Flags().StringVar(&prefix, "prefix", "", "")
	if prefix != "" {
		t.Errorf("prefix default = %q, want empty string", prefix)
	}
	if err := cmd.Flags().Set("prefix", "images/"); err != nil {
		t.Fatalf("failed to set --prefix: %v", err)
	}
	if prefix != "images/" {
		t.Errorf("prefix = %q, want %q", prefix, "images/")
	}
}
