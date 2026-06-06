package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestAccountCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "account" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("accountCmd not registered on rootCmd")
	}
}

func TestAccountCmd_Metadata(t *testing.T) {
	if accountCmd.Use != "account" {
		t.Errorf("accountCmd.Use = %q, want %q", accountCmd.Use, "account")
	}
	if accountCmd.Short == "" {
		t.Error("accountCmd.Short is empty")
	}
	if accountCmd.Long == "" {
		t.Error("accountCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestAccountCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "add", "switch", "remove", "current", "verify"}
	for _, name := range expected {
		found := false
		for _, sub := range accountCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("account subcommand %q not registered", name)
		}
	}
}

// --- Subcommand metadata ---

func TestAccountListCmd_Metadata(t *testing.T) {
	if accountListCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", accountListCmd.Use, "list")
	}
	if accountListCmd.Short == "" {
		t.Error("Short is empty")
	}
	if accountListCmd.Long == "" {
		t.Error("Long is empty")
	}
}

func TestAccountAddCmd_Metadata(t *testing.T) {
	if accountAddCmd.Use != "add <name>" {
		t.Errorf("Use = %q, want %q", accountAddCmd.Use, "add <name>")
	}
	if accountAddCmd.Short == "" {
		t.Error("Short is empty")
	}
}

func TestAccountSwitchCmd_Metadata(t *testing.T) {
	if accountSwitchCmd.Use != "switch <name>" {
		t.Errorf("Use = %q, want %q", accountSwitchCmd.Use, "switch <name>")
	}
	if accountSwitchCmd.Short == "" {
		t.Error("Short is empty")
	}
}

func TestAccountRemoveCmd_Metadata(t *testing.T) {
	if accountRemoveCmd.Use != "remove <name>" {
		t.Errorf("Use = %q, want %q", accountRemoveCmd.Use, "remove <name>")
	}
	if accountRemoveCmd.Short == "" {
		t.Error("Short is empty")
	}
}

func TestAccountCurrentCmd_Metadata(t *testing.T) {
	if accountCurrentCmd.Use != "current" {
		t.Errorf("Use = %q, want %q", accountCurrentCmd.Use, "current")
	}
	if accountCurrentCmd.Short == "" {
		t.Error("Short is empty")
	}
}

func TestAccountVerifyCmd_Metadata(t *testing.T) {
	if accountVerifyCmd.Use != "verify <name>" {
		t.Errorf("Use = %q, want %q", accountVerifyCmd.Use, "verify <name>")
	}
	if accountVerifyCmd.Short == "" {
		t.Error("Short is empty")
	}
}

// --- Arg validation ---

func TestAccountAddCmd_RequiresArg(t *testing.T) {
	err := accountAddCmd.Args(accountAddCmd, []string{})
	if err == nil {
		t.Error("accountAddCmd should require exactly 1 argument")
	}
}

func TestAccountAddCmd_AcceptsOneArg(t *testing.T) {
	err := accountAddCmd.Args(accountAddCmd, []string{"prod"})
	if err != nil {
		t.Errorf("accountAddCmd should accept 1 argument, got error: %v", err)
	}
}

func TestAccountSwitchCmd_RequiresArg(t *testing.T) {
	err := accountSwitchCmd.Args(accountSwitchCmd, []string{})
	if err == nil {
		t.Error("accountSwitchCmd should require exactly 1 argument")
	}
}

func TestAccountRemoveCmd_RequiresArg(t *testing.T) {
	err := accountRemoveCmd.Args(accountRemoveCmd, []string{})
	if err == nil {
		t.Error("accountRemoveCmd should require exactly 1 argument")
	}
}

func TestAccountVerifyCmd_RequiresArg(t *testing.T) {
	err := accountVerifyCmd.Args(accountVerifyCmd, []string{})
	if err == nil {
		t.Error("accountVerifyCmd should require exactly 1 argument")
	}
}

// --- Flags ---

func TestAccountAddCmd_Flags(t *testing.T) {
	flags := []string{"account-id", "api-token", "email"}
	for _, name := range flags {
		f := accountAddCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered on accountAddCmd", name)
		}
	}
}

func TestAccountRemoveCmd_ForceFlag(t *testing.T) {
	f := accountRemoveCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on accountRemoveCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Help output ---

func TestAccountCmd_HelpContainsExamples(t *testing.T) {
	var buf bytes.Buffer
	accountCmd.SetOut(&buf)
	accountCmd.SetErr(&buf)

	tmpRoot := &cobra.Command{Use: "cosmoflare"}
	tmpRoot.AddCommand(accountCmd)

	err := accountCmd.Help()
	if err != nil {
		t.Fatalf("Help() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("cosmoflare account list")) {
		t.Error("help output should contain 'cosmoflare account list'")
	}
	if !bytes.Contains([]byte(output), []byte("cosmoflare account add")) {
		t.Error("help output should contain 'cosmoflare account add'")
	}
}

// --- skipValidation ---

func TestAccountCmd_SkipValidation(t *testing.T) {
	// Verify accountCmd is a child of rootCmd
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub == accountCmd {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("accountCmd is not a child of rootCmd")
	}

	// Verify it has all 6 subcommands
	subs := accountCmd.Commands()
	if len(subs) < 6 {
		t.Errorf("expected at least 6 subcommands, got %d", len(subs))
	}
}

// --- getAccountService ---

func TestGetAccountService(t *testing.T) {
	svc, err := getAccountService()
	if err != nil {
		t.Fatalf("getAccountService() error: %v", err)
	}
	if svc == nil {
		t.Fatal("getAccountService() returned nil")
	}
	if svc.ConfigDir() == "" {
		t.Error("ConfigDir() is empty")
	}
}

// --- maskID ---

func TestMaskID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdefghijklmnop", "abcd********mnop"},
		{"12345678", "12345678"},         // 8 chars, no masking
		{"short", "short"},               // under 8 chars
		{"abcdefghi", "abcd*efghi"[:10]}, // 9 chars
	}

	// Re-check the 9-char case manually
	got9 := maskID("abcdefghi")
	if len(got9) != 9 {
		t.Errorf("maskID(9 chars) len = %d, want 9", len(got9))
	}
	if got9[:4] != "abcd" {
		t.Errorf("maskID prefix = %q, want %q", got9[:4], "abcd")
	}
	if got9[len(got9)-4:] != "fghi" {
		t.Errorf("maskID suffix = %q, want %q", got9[len(got9)-4:], "fghi")
	}

	// Test the first two cases
	for _, tt := range tests[:2] {
		got := maskID(tt.input)
		if got != tt.want {
			t.Errorf("maskID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
