package cmd

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"github.com/spf13/cobra"
)

const authTokenTestLongToken = "cf-super-secret-token-9876" // last 4: "9876"

// --- Command registration & metadata ---

func TestAuthTokenCmd_RegisteredOnAuth(t *testing.T) {
	found := false
	for _, sub := range authCmd.Commands() {
		if sub.Name() == "token" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("authTokenCmd not registered on authCmd")
	}
}

func TestAuthTokenCmd_FindableByName(t *testing.T) {
	sub, _, err := authCmd.Find([]string{"token"})
	if err != nil {
		t.Fatalf("authCmd.Find(token): %v", err)
	}
	if sub == nil || sub.Name() != "token" {
		t.Fatalf("authCmd.Find(token) = %v, want command named token", sub)
	}
}

func TestAuthTokenCmd_Metadata(t *testing.T) {
	if authTokenCmd.Use != "token" {
		t.Errorf("authTokenCmd.Use = %q, want %q", authTokenCmd.Use, "token")
	}
	if authTokenCmd.Short == "" {
		t.Error("authTokenCmd.Short is empty")
	}
	if authTokenCmd.Long == "" {
		t.Error("authTokenCmd.Long is empty")
	}
	if authTokenCmd.RunE == nil {
		t.Error("authTokenCmd has nil RunE")
	}
	if authTokenCmd.Parent() == nil || authTokenCmd.Parent().Use != "auth" {
		t.Errorf("authTokenCmd parent = %v, want auth", authTokenCmd.Parent())
	}
}

func TestAuthTokenCmd_LongWarnsAboutReveal(t *testing.T) {
	long := strings.ToLower(authTokenCmd.Long)
	if !strings.Contains(long, "warning") {
		t.Error("authTokenCmd.Long should warn about --reveal printing a secret")
	}
	if !strings.Contains(long, "pipe") && !strings.Contains(long, "piping") {
		t.Error("authTokenCmd.Long should suggest piping instead of printing the secret")
	}
	if !strings.Contains(long, "cosmoflare auth token --reveal") {
		t.Error("authTokenCmd.Long should contain a --reveal example invocation")
	}
}

// --- Flag registration & defaults ---

func TestAuthToken_Flags(t *testing.T) {
	f := authTokenCmd.Flags().Lookup("reveal")
	if f == nil {
		t.Fatal("flag --reveal not registered on authTokenCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("flag --reveal type = %q, want bool", f.Value.Type())
	}
	if f.DefValue != "false" {
		t.Errorf("flag --reveal default = %q, want false", f.DefValue)
	}
}

// --- redactTokenTail helper ---

func TestRedactTokenTail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{authTokenTestLongToken, "****9876"},
		{"abcd", "****"},
		{"abc", "****"},
		{"", "****"},
	}
	for _, tc := range cases {
		if got := redactTokenTail(tc.in); got != tc.want {
			t.Errorf("redactTokenTail(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- Test scaffolding ---

// tokenTestGlobals snapshots the auth token flag and JSON mode globals.
func tokenTestGlobals(t *testing.T) {
	t.Helper()
	oldReveal := authReveal
	oldJSON := JSONOutput
	t.Cleanup(func() {
		authReveal = oldReveal
		JSONOutput = oldJSON
	})
}

// tokenSeedProfile isolates HOME/keychain and persists a profile with the
// given token, mirroring authRunSeedProfile.
func tokenSeedProfile(t *testing.T, name, apiToken string) {
	t.Helper()
	t.Setenv("COSMOFLARE_NO_KEYCHAIN", "1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	mgr, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	if err := mgr.SetProfile(&config.Profile{
		Name:      name,
		AccountID: "acct-12345678",
		APIToken:  apiToken,
		Region:    "auto",
	}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
}

// newAuthTokenRunCmd builds a fresh token command with the real flag set so
// package flag state is never mutated.
func newAuthTokenRunCmd() *cobra.Command {
	c := &cobra.Command{Use: "token"}
	c.Flags().Bool("reveal", false, "")
	return c
}

// tokenReadOutput drains the captured stdout pipe into a string.
func tokenReadOutput(t *testing.T, r *os.File, restore func()) string {
	t.Helper()
	restore()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}
	return string(out)
}

// --- RunE: default redaction ---

func TestRunAuthToken_DefaultRedactsToken(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "default", authTokenTestLongToken)

	cmd := newAuthTokenRunCmd()
	r, restore := captureStdout(t)
	err := runAuthToken(cmd, nil)
	out := tokenReadOutput(t, r, restore)

	if err != nil {
		t.Fatalf("runAuthToken returned error: %v", err)
	}
	if strings.Contains(out, authTokenTestLongToken) {
		t.Errorf("default output contains the FULL token — redaction failed:\n%s", out)
	}
	if !strings.Contains(out, "****9876") {
		t.Errorf("default output should show the last 4 characters (****9876), got:\n%s", out)
	}
	if !strings.Contains(out, utils.MaskAccountID("acct-12345678")) {
		t.Errorf("default output should include the (masked) account ID, got:\n%s", out)
	}
	if !strings.Contains(out, "file") {
		t.Errorf("default output should mention the store backend (file under tests), got:\n%s", out)
	}
}

// --- RunE: --reveal plain mode ---

func TestRunAuthToken_RevealPlainPrintsOnlyToken(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "default", authTokenTestLongToken)

	cmd := newAuthTokenRunCmd()
	_ = cmd.Flags().Set("reveal", "true")
	r, restore := captureStdout(t)
	err := runAuthToken(cmd, nil)
	out := tokenReadOutput(t, r, restore)

	if err != nil {
		t.Fatalf("runAuthToken returned error: %v", err)
	}
	if got := strings.TrimSpace(out); got != authTokenTestLongToken {
		t.Errorf("--reveal plain output = %q, want exactly the token %q", got, authTokenTestLongToken)
	}
}

// --- RunE: --json default omits the token ---

func TestRunAuthToken_JSONDefaultOmitsToken(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "default", authTokenTestLongToken)
	JSONOutput = true

	cmd := newAuthTokenRunCmd()
	r, restore := captureStdout(t)
	err := runAuthToken(cmd, nil)
	out := tokenReadOutput(t, r, restore)

	if err != nil {
		t.Fatalf("runAuthToken returned error: %v", err)
	}
	if strings.Contains(out, authTokenTestLongToken) {
		t.Errorf("--json default output contains the FULL token — redaction failed:\n%s", out)
	}

	var resp OutputResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if !resp.Success {
		t.Error("JSON envelope success should be true")
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("JSON envelope data = %#v, want object", resp.Data)
	}
	if got := data["token"]; got != "****9876" {
		t.Errorf("data.token = %v, want redacted %q", got, "****9876")
	}
	if revealed, ok := data["revealed"].(bool); !ok || revealed {
		t.Errorf("data.revealed = %#v, want false", data["revealed"])
	}
	if got := data["backend"]; got != "file" {
		t.Errorf("data.backend = %v, want %q", got, "file")
	}
}

// --- RunE: --reveal --json includes the token ---

func TestRunAuthToken_JSONRevealIncludesToken(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "default", authTokenTestLongToken)
	JSONOutput = true

	cmd := newAuthTokenRunCmd()
	_ = cmd.Flags().Set("reveal", "true")
	r, restore := captureStdout(t)
	err := runAuthToken(cmd, nil)
	out := tokenReadOutput(t, r, restore)

	if err != nil {
		t.Fatalf("runAuthToken returned error: %v", err)
	}

	var resp OutputResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("JSON envelope data = %#v, want object", resp.Data)
	}
	if got := data["token"]; got != authTokenTestLongToken {
		t.Errorf("data.token = %v, want the full token", got)
	}
	if revealed, ok := data["revealed"].(bool); !ok || !revealed {
		t.Errorf("data.revealed = %#v, want true", data["revealed"])
	}
}

// --- RunE: missing credentials ---

func TestRunAuthToken_MissingCredentials(t *testing.T) {
	tokenTestGlobals(t)
	t.Setenv("COSMOFLARE_NO_KEYCHAIN", "1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	cmd := newAuthTokenRunCmd()
	err := runAuthToken(cmd, nil)
	if err == nil {
		t.Fatal("expected error when no credentials are stored")
	}
	msg := err.Error()
	// Actionable error: what failed, why, how to fix.
	if !strings.Contains(msg, "no stored credentials") {
		t.Errorf("error = %q, want it to say what failed", msg)
	}
	if !strings.Contains(msg, "auth login") {
		t.Errorf("error = %q, want it to suggest 'cosmoflare auth login'", msg)
	}
	if !strings.Contains(msg, "CLOUDFLARE_API_TOKEN") {
		t.Errorf("error = %q, want it to mention the environment variables", msg)
	}
}

func TestRunAuthToken_ProfileWithoutToken(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "empty-token", "")
	// Clear the token field so the profile has no credential.
	mgr, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	p, err := mgr.GetProfile("empty-token")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	p.APIToken = ""
	if err := mgr.SetProfile(p); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}

	cmd := newAuthTokenRunCmd()
	err = runAuthToken(cmd, nil)
	if err == nil {
		t.Fatal("expected error when profile has no token")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("error = %q, want it to suggest 'cosmoflare auth login'", err.Error())
	}
}

// --- RunE: environment credentials win and report the environment backend ---

func TestRunAuthToken_EnvironmentCredentialsBackend(t *testing.T) {
	tokenTestGlobals(t)
	tokenSeedProfile(t, "default", authTokenTestLongToken)
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token-4321")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "acct-12345678")

	cmd := newAuthTokenRunCmd()
	r, restore := captureStdout(t)
	err := runAuthToken(cmd, nil)
	out := tokenReadOutput(t, r, restore)

	if err != nil {
		t.Fatalf("runAuthToken returned error: %v", err)
	}
	if strings.Contains(out, "env-token-4321") {
		t.Errorf("default output leaked the environment token:\n%s", out)
	}
	if !strings.Contains(out, "****4321") {
		t.Errorf("default output should redact to ****4321, got:\n%s", out)
	}
	if !strings.Contains(out, "environment") {
		t.Errorf("default output should report the environment backend, got:\n%s", out)
	}
}
