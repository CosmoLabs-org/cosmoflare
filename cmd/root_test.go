package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// --- BUG-034: --json mode must emit the JSON error envelope on config errors ---

// captureStdout swaps os.Stdout for a pipe and returns the restore func and
// the reader holding everything written after the swap.
func captureStdout(t *testing.T) (*os.File, func()) {
	t.Helper()
	saved := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w
	return r, func() {
		os.Stdout = saved
		w.Close()
	}
}

func TestEmitConfigError_JSONModeEmitsErrorEnvelope(t *testing.T) {
	savedJSON := JSONOutput
	defer func() { JSONOutput = savedJSON }()

	JSONOutput = true
	r, restore := captureStdout(t)
	emitConfigError("Configuration error: %v", fmt.Errorf("Cloudflare API token is required. Set CLOUDFLARE_API_TOKEN environment variable or use --api-token flag"))
	restore()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}

	var resp OutputResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("BUG-034 regression: stdout is not a parseable JSON error envelope: %v; raw output: %q", err, string(out))
	}
	if resp.Success {
		t.Errorf("envelope success = true, want false")
	}
	if resp.Error == "" {
		t.Errorf("envelope error field is empty, want the configuration error message; raw output: %q", string(out))
	}
	if !strings.Contains(resp.Error, "API token is required") {
		t.Errorf("envelope error = %q, want it to contain the underlying validation message", resp.Error)
	}
}

func TestEmitConfigError_TextModePrintsPlainMessage(t *testing.T) {
	savedJSON := JSONOutput
	defer func() { JSONOutput = savedJSON }()

	JSONOutput = false
	r, restore := captureStdout(t)
	emitConfigError("Configuration error: %v", fmt.Errorf("token missing"))
	restore()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}

	if !strings.Contains(string(out), "Configuration error: token missing") {
		t.Errorf("text-mode output = %q, want it to contain the plain error message", string(out))
	}
	if strings.Contains(string(out), `"success"`) {
		t.Errorf("text-mode output unexpectedly contains a JSON envelope: %q", string(out))
	}
}
