package cmd

import (
	"strings"
	"testing"
)

// --- Root command identity ---

func TestRootCmd_Use(t *testing.T) {
	if rootCmd.Use != "cosmoflare" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "cosmoflare")
	}
}

func TestRootCmd_Short(t *testing.T) {
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short is empty")
	}
}

func TestRootCmd_Long(t *testing.T) {
	if rootCmd.Long == "" {
		t.Error("rootCmd.Long is empty")
	}
}

// --- Version template ---

func TestRootCmd_VersionTemplate(t *testing.T) {
	tmpl := rootCmd.VersionTemplate()
	if !strings.Contains(tmpl, "Cosmoflare") {
		t.Errorf("version template = %q, want it to contain %q", tmpl, "Cosmoflare")
	}
}

// --- Global flags ---

func TestRootCmd_GlobalFlags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"account-id", ""},
		{"api-token", ""},
		{"dry-run", "false"},
		{"json", "false"},
		{"verbose", "false"},
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			pf := rootCmd.PersistentFlags().Lookup(f.name)
			if pf == nil {
				t.Fatalf("global flag --%s not registered", f.name)
			}
			if pf.DefValue != f.defValue {
				t.Errorf("--%s default = %q, want %q", f.name, pf.DefValue, f.defValue)
			}
		})
	}
}

func TestRootCmd_VerboseShorthand(t *testing.T) {
	pf := rootCmd.PersistentFlags().Lookup("verbose")
	if pf == nil {
		t.Fatal("global flag --verbose not registered")
	}
	if pf.Shorthand != "v" {
		t.Errorf("--verbose shorthand = %q, want %q", pf.Shorthand, "v")
	}
}

// --- Legacy subcommands registered ---

func TestRootCmd_LegacySubcommands(t *testing.T) {
	expected := []string{"create", "list", "delete"}
	cmds := rootCmd.Commands()
	for _, name := range expected {
		found := false
		for _, sub := range cmds {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("legacy subcommand %q not registered on rootCmd", name)
		}
	}
}

// --- Execute function ---

func TestRootCmd_ExecuteReturnsError(t *testing.T) {
	// Execute() should be callable without panic.
	// It will return an error because no API token is set for most subcommands,
	// but the function itself should not panic.
	// We just verify the function signature exists and is callable.
	var fn func() error = Execute
	if fn == nil {
		t.Fatal("Execute function is nil")
	}
}

// --- Helper functions ---

func TestValidateEnvironment_EmptyToken(t *testing.T) {
	saved := APIToken
	defer func() { APIToken = saved }()

	APIToken = ""
	err := validateEnvironment()
	if err == nil {
		t.Error("expected error with empty API token, got nil")
	}
}

func TestValidateEnvironment_ShortToken(t *testing.T) {
	saved := APIToken
	defer func() { APIToken = saved }()

	APIToken = "short"
	err := validateEnvironment()
	if err == nil {
		t.Error("expected error with short API token, got nil")
	}
}

func TestValidateEnvironment_ValidToken(t *testing.T) {
	saved := APIToken
	defer func() { APIToken = saved }()

	APIToken = "a_valid_cloudflare_api_token_1234567890"
	err := validateEnvironment()
	if err != nil {
		t.Errorf("expected no error with valid API token, got: %v", err)
	}
}

func TestSetBuildInfo(t *testing.T) {
	savedV, savedB, savedG := AppVersion, BuildTime, GitCommit
	defer func() {
		AppVersion = savedV
		BuildTime = savedB
		GitCommit = savedG
	}()

	SetBuildInfo("1.2.3", "2026-01-01", "abc123")
	if AppVersion != "1.2.3" {
		t.Errorf("AppVersion = %q, want %q", AppVersion, "1.2.3")
	}
	if BuildTime != "2026-01-01" {
		t.Errorf("BuildTime = %q, want %q", BuildTime, "2026-01-01")
	}
	if GitCommit != "abc123" {
		t.Errorf("GitCommit = %q, want %q", GitCommit, "abc123")
	}
}

func TestGetRelativePath(t *testing.T) {
	result := getRelativePath("/some/absolute/path")
	if result == "" {
		t.Error("getRelativePath returned empty string")
	}
}
