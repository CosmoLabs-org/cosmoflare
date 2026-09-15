package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// setupRunGlobals snapshots and restores the setup flag variables and the
// Cloudflare environment variables the setup runners read.
func setupRunGlobals(t *testing.T) {
	t.Helper()
	oldProfile, oldQuiet, oldSkip := setupProfile, setupQuiet, setupSkipTest
	oldAuto, oldSwitch, oldWelcome := setupAutoDetect, setupSwitch, setupWelcome
	oldBackup, oldRestore := setupBackup, setupRestore
	oldToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	oldAccount := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	t.Cleanup(func() {
		setupProfile, setupQuiet, setupSkipTest = oldProfile, oldQuiet, oldSkip
		setupAutoDetect, setupSwitch, setupWelcome = oldAuto, oldSwitch, oldWelcome
		setupBackup, setupRestore = oldBackup, oldRestore
		os.Setenv("CLOUDFLARE_API_TOKEN", oldToken)
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", oldAccount)
	})
}

// setupRunResetFlags clears all setup flags and Cloudflare env vars so each
// subtest starts from a pristine command state.
func setupRunResetFlags() {
	setupProfile = ""
	setupQuiet = false
	setupSkipTest = false
	setupAutoDetect = true
	setupSwitch = false
	setupWelcome = false
	setupBackup = false
	setupRestore = false
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
}

// TestRunSetup_QuietModeRequiresEnvVars verifies the quiet-mode guards fail
// fast with the exact missing-variable errors before any network access.
func TestRunSetup_QuietModeRequiresEnvVars(t *testing.T) {
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
				os.Setenv("CLOUDFLARE_API_TOKEN", "fake-token-1234567890abcdef")
			},
			wantErr: "CLOUDFLARE_ACCOUNT_ID environment variable is required in quiet mode",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupRunGlobals(t)
			setupRunResetFlags()
			setupQuiet = true
			tc.setEnv()

			err := runSetup(setupCmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunSetup_QuietModeInvalidAccountID verifies quiet setup rejects a
// malformed account ID offline via the format validator.
func TestRunSetup_QuietModeInvalidAccountID(t *testing.T) {
	setupRunGlobals(t)
	setupRunResetFlags()
	setupQuiet = true
	os.Setenv("CLOUDFLARE_API_TOKEN", "fake-token-1234567890abcdef")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "not-a-valid-id")

	err := runSetup(setupCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "account ID validation failed") {
		t.Fatalf("expected account ID validation error, got %v", err)
	}
}

// TestRunSetup_QuietModeShortToken verifies quiet setup rejects a token that
// fails the offline length check before any network validation happens.
func TestRunSetup_QuietModeShortToken(t *testing.T) {
	setupRunGlobals(t)
	setupRunResetFlags()
	setupQuiet = true
	os.Setenv("CLOUDFLARE_API_TOKEN", "short")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "0123456789abcdef0123456789abcdef")

	err := runSetup(setupCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "Token is too short") {
		t.Fatalf("expected short-token error, got %v", err)
	}
}

// TestRunSetup_WelcomePathReturnsNil verifies the --welcome branch prints the
// new-user welcome and returns success without touching configuration.
func TestRunSetup_WelcomePathReturnsNil(t *testing.T) {
	setupRunGlobals(t)
	setupRunResetFlags()
	setupWelcome = true

	if err := runSetup(setupCmd, nil); err != nil {
		t.Fatalf("welcome branch should return nil, got %v", err)
	}
}

// TestRunInteractiveSetup_AuthSelectionFailsOnClosedStdin verifies the
// interactive wizard fails fast when the auth-method prompt cannot read any
// input, instead of blocking forever.
func TestRunInteractiveSetup_AuthSelectionFailsOnClosedStdin(t *testing.T) {
	setupRunGlobals(t)
	setupRunResetFlags()

	err := runInteractiveSetup(interactive.NewSetupWizard())
	if err == nil || !strings.Contains(err.Error(), "authentication selection failed") {
		t.Fatalf("expected auth selection error on closed stdin, got %v", err)
	}
}

// TestRunSetup_InteractiveDispatchFailsFastOnEOF verifies runSetup dispatches
// to the interactive wizard by default and surfaces its prompt failure.
func TestRunSetup_InteractiveDispatchFailsFastOnEOF(t *testing.T) {
	setupRunGlobals(t)
	setupRunResetFlags()

	err := runSetup(setupCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "authentication selection failed") {
		t.Fatalf("expected auth selection error from interactive dispatch, got %v", err)
	}
}
