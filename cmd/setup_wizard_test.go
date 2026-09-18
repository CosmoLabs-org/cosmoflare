package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// setupWizardSnapshot snapshots the setup flag variables and the Cloudflare
// environment variables the wizard helpers read.
func setupWizardSnapshot(t *testing.T) {
	t.Helper()
	setupRunGlobals(t)
	t.Cleanup(setupRunResetFlags)
}

// TestRunQuietSetup_MissingEnvVars verifies quiet setup fails fast with the
// exact missing-variable errors before any validation or network access.
func TestRunQuietSetup_MissingEnvVars(t *testing.T) {
	setupWizardSnapshot(t)
	cases := []struct {
		name    string
		setEnv  func()
		wantErr string
	}{
		{
			name:    "missing api token",
			setEnv:  func() {},
			wantErr: "CLOUDFLARE_API_TOKEN environment variable is required in quiet mode",
		},
		{
			name: "missing account id",
			setEnv: func() {
				t.Setenv("CLOUDFLARE_API_TOKEN", "fake-token-1234567890abcdef")
			},
			wantErr: "CLOUDFLARE_ACCOUNT_ID environment variable is required in quiet mode",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupRunResetFlags()
			tc.setEnv()
			err := runQuietSetup(interactive.NewSetupWizard())
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunQuietSetup_InvalidAccountIDFailsFast verifies quiet setup validates
// the account ID before touching the network; a malformed ID is rejected by
// the local validator.
func TestRunQuietSetup_InvalidAccountIDFailsFast(t *testing.T) {
	setupWizardSnapshot(t)
	setupRunResetFlags()
	t.Setenv("CLOUDFLARE_API_TOKEN", "fake-token-1234567890abcdef")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "not!a!hex!id")

	err := runQuietSetup(interactive.NewSetupWizard())
	if err == nil || !strings.Contains(err.Error(), "account ID validation failed") {
		t.Fatalf("expected account-ID validation error, got %v", err)
	}
}

// TestPrintSetupNextSteps_OutputsGuidance verifies the post-setup helper
// prints the expected command pointers without panicking.
func TestPrintSetupNextSteps_OutputsGuidance(t *testing.T) {
	r, restore := captureStdout(t)
	printSetupNextSteps()
	restore()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	text := string(out)
	for _, want := range []string{"Next steps", "cosmoflare config list", "cosmoflare bucket list", "cosmoflare --help"} {
		if !strings.Contains(text, want) {
			t.Errorf("next-steps output missing %q, got: %s", want, text)
		}
	}
}

// TestSetupWizardAccountInfo_ManualFallbackWithEmptyToken verifies that when
// auto-detection cannot run (no token) and stdin is closed, the account-info
// step degrades to empty values instead of panicking — the interactive error
// is intentionally swallowed by the wrapper.
func TestSetupWizardAccountInfo_ManualFallbackWithEmptyToken(t *testing.T) {
	setupWizardSnapshot(t)
	setupRunResetFlags()
	setupAutoDetect = false

	accountID, accountName := setupWizardAccountInfo(interactive.NewSetupWizard(), "")
	if accountID != "" || accountName != "" {
		t.Errorf("expected empty account values on closed stdin, got (%q, %q)", accountID, accountName)
	}
}
