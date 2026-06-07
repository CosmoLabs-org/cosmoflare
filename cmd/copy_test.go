package cmd

import (
	"strings"
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
	if copyCmd.Long == "" {
		t.Error("copyCmd.Long is empty")
	}
	if copyCmd.Use != "copy [source] [destination]" {
		t.Errorf("copyCmd.Use = %q, want %q", copyCmd.Use, "copy [source] [destination]")
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
		flag  string
		short string
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

// --- Flag types ---

func TestCopyCmd_FlagTypes(t *testing.T) {
	boolFlags := []string{
		"recursive", "resume", "verify", "overwrite", "preserve",
		"progress", "quiet", "stats", "batch", "interactive", "dry-run", "no-clobber",
	}
	for _, name := range boolFlags {
		f := copyCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "bool" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "bool")
		}
	}

	intFlags := []string{"parallel", "retries"}
	for _, name := range intFlags {
		f := copyCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "int" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "int")
		}
	}

	stringFlags := []string{"chunk-size", "format", "timeout"}
	for _, name := range stringFlags {
		f := copyCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "string")
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

func TestCopyCmd_NoArgsErrorMessage(t *testing.T) {
	err := runCopy(copyCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	if !strings.Contains(err.Error(), "source") && !strings.Contains(err.Error(), "destination") && !strings.Contains(err.Error(), "required") {
		t.Errorf("error message %q should mention source/destination/required", err.Error())
	}
}

// --- parseChunkSize helper ---

func TestParseChunkSize_MB(t *testing.T) {
	cases := []struct {
		input string
		want  int64
	}{
		{"8MB", 8 * 1024 * 1024},
		{"1MB", 1 * 1024 * 1024},
		{"16MB", 16 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}
	for _, tc := range cases {
		got, err := parseChunkSize(tc.input)
		if err != nil {
			t.Errorf("parseChunkSize(%q) returned unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("parseChunkSize(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseChunkSize_GB(t *testing.T) {
	cases := []struct {
		input string
		want  int64
	}{
		{"1GB", 1 * 1024 * 1024 * 1024},
		{"2GB", 2 * 1024 * 1024 * 1024},
	}
	for _, tc := range cases {
		got, err := parseChunkSize(tc.input)
		if err != nil {
			t.Errorf("parseChunkSize(%q) returned unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("parseChunkSize(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseChunkSize_KB(t *testing.T) {
	cases := []struct {
		input string
		want  int64
	}{
		{"512KB", 512 * 1024},
		{"1KB", 1 * 1024},
	}
	for _, tc := range cases {
		got, err := parseChunkSize(tc.input)
		if err != nil {
			t.Errorf("parseChunkSize(%q) returned unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("parseChunkSize(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseChunkSize_CaseInsensitive(t *testing.T) {
	cases := []struct {
		lower string
		upper string
	}{
		{"8mb", "8MB"},
		{"1gb", "1GB"},
		{"512kb", "512KB"},
	}
	for _, tc := range cases {
		lower, err := parseChunkSize(tc.lower)
		if err != nil {
			t.Fatalf("parseChunkSize(%q) error: %v", tc.lower, err)
		}
		upper, err := parseChunkSize(tc.upper)
		if err != nil {
			t.Fatalf("parseChunkSize(%q) error: %v", tc.upper, err)
		}
		if lower != upper {
			t.Errorf("parseChunkSize case sensitivity: %q=%d, %q=%d", tc.lower, lower, tc.upper, upper)
		}
	}
}

func TestParseChunkSize_PlainBytes(t *testing.T) {
	got, err := parseChunkSize("4096")
	if err != nil {
		t.Fatalf("parseChunkSize(\"4096\") error: %v", err)
	}
	if got != 4096 {
		t.Errorf("parseChunkSize(\"4096\") = %d, want 4096", got)
	}
}

// --- Long help content ---

func TestCopyCmd_LongHelpContent(t *testing.T) {
	keywords := []string{
		"progress",
		"resume",
		"batch",
		"cosmoflare copy",
	}
	for _, kw := range keywords {
		if !strings.Contains(copyCmd.Long, kw) {
			t.Errorf("copyCmd.Long does not contain expected keyword %q", kw)
		}
	}
}

// --- No subcommands ---

func TestCopyCmd_HasNoSubcommands(t *testing.T) {
	if len(copyCmd.Commands()) != 0 {
		names := make([]string, 0, len(copyCmd.Commands()))
		for _, c := range copyCmd.Commands() {
			names = append(names, c.Name())
		}
		t.Errorf("copyCmd should have no subcommands, but has: %v", names)
	}
}
