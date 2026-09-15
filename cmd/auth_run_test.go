package cmd

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/spf13/cobra"
)

// authRunGlobals snapshots the auth flag variables and the BUG-044 test seams
// (testCredentials / generateNewToken / revokeOldToken) and clears the auth
// environment variables so tests cannot leak state.
func authRunGlobals(t *testing.T) {
	t.Helper()
	oldProfile, oldToken, oldAccount := authProfile, authToken, authAccountID
	oldEmail, oldMethod, oldScope := authEmail, authMethod, authScope
	oldInteractive := authInteractive
	oldTest, oldGen, oldRevoke := testCredentials, generateNewToken, revokeOldToken
	t.Cleanup(func() {
		authProfile, authToken, authAccountID = oldProfile, oldToken, oldAccount
		authEmail, authMethod, authScope = oldEmail, oldMethod, oldScope
		authInteractive = oldInteractive
		testCredentials, generateNewToken, revokeOldToken = oldTest, oldGen, oldRevoke
	})
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_EMAIL", "")
}

// newAuthLoginRunCmd builds a fresh login command with the same flag set as
// the real registration so package flag state is never mutated.
func newAuthLoginRunCmd() *cobra.Command {
	c := &cobra.Command{Use: "login"}
	c.Flags().String("profile", "", "")
	c.Flags().String("token", "", "")
	c.Flags().String("account-id", "", "")
	c.Flags().String("email", "", "")
	c.Flags().String("method", "token", "")
	c.Flags().Bool("interactive", true, "")
	c.Flags().String("scope", "", "")
	return c
}

// newAuthRotateRunCmd builds a fresh rotate command with the real flag set.
func newAuthRotateRunCmd() *cobra.Command {
	c := &cobra.Command{Use: "rotate"}
	c.Flags().String("profile", "", "")
	c.Flags().Bool("revoke-old", false, "")
	return c
}

// authRunIsolateHome isolates HOME + keychain so a config manager can be
// created without touching the developer's real profile store.
func authRunIsolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("COSMOFLARE_NO_KEYCHAIN", "1")
	t.Setenv("HOME", t.TempDir())
}

// authRunSeedProfile isolates HOME and persists a profile for rotate tests.
func authRunSeedProfile(t *testing.T, name string) {
	t.Helper()
	authRunIsolateHome(t)
	mgr, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	if err := mgr.SetProfile(&config.Profile{
		Name:      name,
		AccountID: "acct-run",
		APIToken:  "OLD-RUN-TOKEN",
		Region:    "auto",
	}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
}

// TestRunAuthLogin_TokenMethod verifies the happy token path: credentials are
// built from flags, trimmed, and passed to the (stubbed) validator.
func TestRunAuthLogin_TokenMethod(t *testing.T) {
	authRunGlobals(t)
	cmd := newAuthLoginRunCmd()
	_ = cmd.Flags().Set("interactive", "false")
	_ = cmd.Flags().Set("token", "  tok-run-123  ")
	_ = cmd.Flags().Set("account-id", " acct-run-456 ")

	var seen *AuthCredentials
	testCredentials = func(c *AuthCredentials) error { seen = c; return nil }

	if err := runAuthLogin(cmd, nil); err != nil {
		t.Fatalf("runAuthLogin returned error: %v", err)
	}
	if seen == nil {
		t.Fatal("testCredentials was not invoked")
	}
	if seen.APIToken != "tok-run-123" || seen.AccountID != "acct-run-456" {
		t.Errorf("credentials = %+v, want trimmed flag values", seen)
	}
	if seen.Method != "token" {
		t.Errorf("Method = %q, want %q", seen.Method, "token")
	}
}

// TestRunAuthLogin_EarlyErrors verifies the fast-fail branches: missing
// credentials and failed credential validation.
func TestRunAuthLogin_EarlyErrors(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(*cobra.Command)
		wantErr string
	}{
		{"missing token", func(c *cobra.Command) {
			_ = c.Flags().Set("interactive", "false")
		}, "failed to get credentials: API token is required"},
		{"missing account id", func(c *cobra.Command) {
			_ = c.Flags().Set("interactive", "false")
			_ = c.Flags().Set("token", "tok-run")
		}, "failed to get credentials: Account ID is required"},
		{"validation failure", func(c *cobra.Command) {
			_ = c.Flags().Set("interactive", "false")
			_ = c.Flags().Set("token", "tok-run")
			_ = c.Flags().Set("account-id", "acct-run")
		}, "credential validation failed: connection refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authRunGlobals(t)
			testCredentials = func(*AuthCredentials) error {
				return errors.New("connection refused")
			}
			cmd := newAuthLoginRunCmd()
			tc.setup(cmd)
			err := runAuthLogin(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// TestRunAuthLogin_SavesProfile verifies that --profile persists the flag
// credentials via saveCredentialsToProfile.
func TestRunAuthLogin_SavesProfile(t *testing.T) {
	authRunGlobals(t)
	authRunIsolateHome(t)

	cmd := newAuthLoginRunCmd()
	_ = cmd.Flags().Set("interactive", "false")
	_ = cmd.Flags().Set("token", "tok-save-run")
	_ = cmd.Flags().Set("account-id", "acct-save-run")
	_ = cmd.Flags().Set("profile", "run-profile")
	testCredentials = func(*AuthCredentials) error { return nil }

	if err := runAuthLogin(cmd, nil); err != nil {
		t.Fatalf("runAuthLogin returned error: %v", err)
	}

	mgr, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	profile, err := mgr.GetProfile("run-profile")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.APIToken != "tok-save-run" || profile.AccountID != "acct-save-run" {
		t.Errorf("profile = %+v, want saved flag credentials", profile)
	}
	if profile.Region != "auto" {
		t.Errorf("Region = %q, want %q", profile.Region, "auto")
	}
}

// TestRunAuthRotate_EarlyErrors verifies the rotate fast-fail branches that
// never reach the network.
func TestRunAuthRotate_EarlyErrors(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T) *cobra.Command
		wantErr string
	}{
		{"unknown profile", func(t *testing.T) *cobra.Command {
			authRunIsolateHome(t)
			cmd := newAuthRotateRunCmd()
			_ = cmd.Flags().Set("profile", "ghost-profile")
			return cmd
		}, "failed to get profile"},
		{"token generation failure", func(t *testing.T) *cobra.Command {
			authRunSeedProfile(t, "prod")
			cmd := newAuthRotateRunCmd()
			_ = cmd.Flags().Set("profile", "prod")
			return cmd
		}, "failed to generate new token: automatic token generation not implemented"},
		{"new token validation failure", func(t *testing.T) *cobra.Command {
			authRunSeedProfile(t, "prod")
			generateNewToken = func(*config.Profile) (string, error) {
				return "NEW-RUN-TOKEN", nil
			}
			testCredentials = func(*AuthCredentials) error {
				return errors.New("bad new token")
			}
			cmd := newAuthRotateRunCmd()
			_ = cmd.Flags().Set("profile", "prod")
			return cmd
		}, "new token validation failed: bad new token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authRunGlobals(t)
			cmd := tc.setup(t)
			err := runAuthRotate(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// TestRunAuthStatus_MissingEnv verifies the status runner fails fast with the
// env-validation error when credentials are absent (no network attempted).
func TestRunAuthStatus_MissingEnv(t *testing.T) {
	authRunGlobals(t)
	authRunIsolateHome(t)

	err := runAuthStatus(authStatusCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "missing CLOUDFLARE_API_TOKEN") {
		t.Fatalf("error = %v, want env-missing error", err)
	}
}

// TestRunAuthLogout_ClearsEnv verifies logout unsets the credential-related
// environment variables and returns nil.
func TestRunAuthLogout_ClearsEnv(t *testing.T) {
	for _, kv := range [][2]string{
		{"CLOUDFLARE_API_TOKEN", "tok"},
		{"CLOUDFLARE_ACCOUNT_ID", "acct"},
		{"R2_ENDPOINT", "https://r2.example"},
		{"AWS_ACCESS_KEY_ID", "key"},
		{"AWS_SECRET_ACCESS_KEY", "secret"},
		{"AWS_REGION", "us-east-1"},
	} {
		t.Setenv(kv[0], kv[1])
	}

	if err := runAuthLogout(authLogoutCmd, nil); err != nil {
		t.Fatalf("runAuthLogout returned error: %v", err)
	}
	for _, name := range []string{
		"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID", "R2_ENDPOINT",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION",
	} {
		if got := os.Getenv(name); got != "" {
			t.Errorf("%s = %q after logout, want empty", name, got)
		}
	}
}
