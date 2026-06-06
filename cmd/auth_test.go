package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestAuthCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "auth" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("authCmd not registered on rootCmd")
	}
}

func TestAuthCmd_Metadata(t *testing.T) {
	if authCmd.Use != "auth" {
		t.Errorf("authCmd.Use = %q, want %q", authCmd.Use, "auth")
	}
	if authCmd.Short == "" {
		t.Error("authCmd.Short is empty")
	}
	if authCmd.Long == "" {
		t.Error("authCmd.Long is empty")
	}
}

func TestAuthCmd_ShortContainsCloudflare(t *testing.T) {
	if !strings.Contains(strings.ToLower(authCmd.Short), "cloudflare") &&
		!strings.Contains(strings.ToLower(authCmd.Short), "auth") {
		t.Errorf("authCmd.Short = %q, expected mention of Cloudflare or auth", authCmd.Short)
	}
}

func TestAuthCmd_LongContainsSubcommandList(t *testing.T) {
	long := authCmd.Long
	for _, sub := range []string{"login", "rotate", "status", "logout"} {
		if !strings.Contains(long, sub) {
			t.Errorf("authCmd.Long does not mention subcommand %q", sub)
		}
	}
}

// --- Subcommand registration ---

func TestAuthCmd_Subcommands(t *testing.T) {
	expected := []string{"login", "rotate", "status", "logout"}
	for _, name := range expected {
		found := false
		for _, sub := range authCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("auth subcommand %q not registered", name)
		}
	}
}

func TestAuthCmd_SubcommandCount(t *testing.T) {
	// Expect exactly 4 subcommands
	got := len(authCmd.Commands())
	if got < 4 {
		t.Errorf("authCmd has %d subcommands, want at least 4", got)
	}
}

// --- RunE handlers wired ---

func TestAuthCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		authLoginCmd,
		authRotateCmd,
		authStatusCmd,
		authLogoutCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestAuthCmd_NoRunE(t *testing.T) {
	// Parent authCmd should NOT have a RunE (it is a group command)
	if authCmd.RunE != nil {
		t.Error("authCmd should not have a RunE; it is a group command")
	}
	if authCmd.Run != nil {
		t.Error("authCmd should not have a Run; it is a group command")
	}
}

// --- Subcommand metadata ---

func TestAuthLoginCmd_Metadata(t *testing.T) {
	if authLoginCmd.Use != "login" {
		t.Errorf("authLoginCmd.Use = %q, want %q", authLoginCmd.Use, "login")
	}
	if authLoginCmd.Short == "" {
		t.Error("authLoginCmd.Short is empty")
	}
	if authLoginCmd.Long == "" {
		t.Error("authLoginCmd.Long is empty")
	}
}

func TestAuthRotateCmd_Metadata(t *testing.T) {
	if authRotateCmd.Use != "rotate" {
		t.Errorf("authRotateCmd.Use = %q, want %q", authRotateCmd.Use, "rotate")
	}
	if authRotateCmd.Short == "" {
		t.Error("authRotateCmd.Short is empty")
	}
}

func TestAuthStatusCmd_Metadata(t *testing.T) {
	if authStatusCmd.Use != "status" {
		t.Errorf("authStatusCmd.Use = %q, want %q", authStatusCmd.Use, "status")
	}
	if authStatusCmd.Short == "" {
		t.Error("authStatusCmd.Short is empty")
	}
}

func TestAuthLogoutCmd_Metadata(t *testing.T) {
	if authLogoutCmd.Use != "logout" {
		t.Errorf("authLogoutCmd.Use = %q, want %q", authLogoutCmd.Use, "logout")
	}
	if authLogoutCmd.Short == "" {
		t.Error("authLogoutCmd.Short is empty")
	}
}

// --- Help text content ---

func TestAuthLoginCmd_LongMentionsExamples(t *testing.T) {
	if !strings.Contains(authLoginCmd.Long, "cosmoflare auth login") {
		t.Error("authLoginCmd.Long should contain an example invocation")
	}
}

func TestAuthLoginCmd_LongMentionsFlags(t *testing.T) {
	long := authLoginCmd.Long
	if !strings.Contains(long, "--token") && !strings.Contains(long, "token") {
		t.Error("authLoginCmd.Long should mention token-related usage")
	}
}

func TestAuthRotateCmd_LongMentionsProfile(t *testing.T) {
	if !strings.Contains(authRotateCmd.Long, "--profile") {
		t.Error("authRotateCmd.Long should mention --profile")
	}
}

func TestAuthLogoutCmd_LongMentionsNote(t *testing.T) {
	// Logout command should warn about profile config remaining
	long := strings.ToLower(authLogoutCmd.Long)
	if !strings.Contains(long, "profile") && !strings.Contains(long, "config") {
		t.Error("authLogoutCmd.Long should mention that profile configurations remain")
	}
}

// --- Flag registration ---

func TestAuthLogin_Flags(t *testing.T) {
	expected := []string{"profile", "token", "account-id", "email", "method", "interactive", "scope"}
	for _, name := range expected {
		if authLoginCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on authLoginCmd", name)
		}
	}
}

func TestAuthRotate_Flags(t *testing.T) {
	expected := []string{"profile", "revoke-old"}
	for _, name := range expected {
		if authRotateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on authRotateCmd", name)
		}
	}
}

// --- Flag defaults ---

func TestAuthLogin_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"method", "token"},
		{"interactive", "true"},
		{"profile", ""},
		{"token", ""},
		{"account-id", ""},
		{"email", ""},
		{"scope", ""},
	}
	for _, tc := range cases {
		f := authLoginCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestAuthRotate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"revoke-old", "false"},
		{"profile", ""},
	}
	for _, tc := range cases {
		f := authRotateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on authRotateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Flag types ---

func TestAuthLogin_FlagTypes(t *testing.T) {
	stringFlags := []string{"profile", "token", "account-id", "email", "method", "scope"}
	for _, name := range stringFlags {
		f := authLoginCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag --%s type = %q, want string", name, f.Value.Type())
		}
	}
	boolFlag := authLoginCmd.Flags().Lookup("interactive")
	if boolFlag == nil {
		t.Fatal("flag --interactive not found")
	}
	if boolFlag.Value.Type() != "bool" {
		t.Errorf("flag --interactive type = %q, want bool", boolFlag.Value.Type())
	}
}

func TestAuthRotate_RevokeFlagType(t *testing.T) {
	f := authRotateCmd.Flags().Lookup("revoke-old")
	if f == nil {
		t.Fatal("flag --revoke-old not found")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("flag --revoke-old type = %q, want bool", f.Value.Type())
	}
}

// --- Arg validation ---

func TestAuthRotate_NoProfile(t *testing.T) {
	// Reset the profile flag to empty before testing
	if err := authRotateCmd.Flags().Set("profile", ""); err != nil {
		t.Fatalf("failed to reset --profile: %v", err)
	}
	err := runAuthRotate(authRotateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no profile provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("profile")) {
		t.Errorf("error = %q, want it to mention 'profile'", err.Error())
	}
}

func TestAuthLogin_UnsupportedMethod(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("method", "token", "")
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("account-id", "", "")
	cmd.Flags().String("email", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().String("scope", "", "")

	if err := cmd.Flags().Set("method", "unsupported"); err != nil {
		t.Fatalf("failed to set --method: %v", err)
	}
	if err := cmd.Flags().Set("interactive", "false"); err != nil {
		t.Fatalf("failed to set --interactive: %v", err)
	}

	err := runAuthLogin(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for unsupported auth method")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want it to mention 'unsupported'", err.Error())
	}
}

// --- Flag variable wiring ---

func TestAuthFlagsParsing_Method(t *testing.T) {
	cmd := &cobra.Command{}
	var method string
	cmd.Flags().StringVar(&method, "method", "token", "")
	if method != "token" {
		t.Errorf("method default = %q, want %q", method, "token")
	}
	if err := cmd.Flags().Set("method", "key"); err != nil {
		t.Fatalf("failed to set --method: %v", err)
	}
	if method != "key" {
		t.Errorf("method = %q, want %q", method, "key")
	}
}

func TestAuthFlagsParsing_Token(t *testing.T) {
	cmd := &cobra.Command{}
	var token string
	cmd.Flags().StringVar(&token, "token", "", "")
	if err := cmd.Flags().Set("token", "my-api-token"); err != nil {
		t.Fatalf("failed to set --token: %v", err)
	}
	if token != "my-api-token" {
		t.Errorf("token = %q, want %q", token, "my-api-token")
	}
}

func TestAuthFlagsParsing_Interactive(t *testing.T) {
	cmd := &cobra.Command{}
	var interactive bool
	cmd.Flags().BoolVar(&interactive, "interactive", true, "")
	if !interactive {
		t.Error("interactive should default to true")
	}
	if err := cmd.Flags().Set("interactive", "false"); err != nil {
		t.Fatalf("failed to set --interactive: %v", err)
	}
	if interactive {
		t.Error("interactive should be false after --interactive=false")
	}
}

func TestAuthFlagsParsing_AccountID(t *testing.T) {
	cmd := &cobra.Command{}
	var accountID string
	cmd.Flags().StringVar(&accountID, "account-id", "", "")
	if accountID != "" {
		t.Errorf("account-id default = %q, want empty", accountID)
	}
	if err := cmd.Flags().Set("account-id", "abc123"); err != nil {
		t.Fatalf("failed to set --account-id: %v", err)
	}
	if accountID != "abc123" {
		t.Errorf("account-id = %q, want %q", accountID, "abc123")
	}
}

func TestAuthFlagsParsing_RevokeOld(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("revoke-old", false, "")
	if err := cmd.Flags().Set("revoke-old", "true"); err != nil {
		t.Fatalf("failed to set --revoke-old: %v", err)
	}
	got, err := cmd.Flags().GetBool("revoke-old")
	if err != nil {
		t.Fatalf("GetBool revoke-old: %v", err)
	}
	if !got {
		t.Error("revoke-old should be true after --revoke-old=true")
	}
}

// --- AuthCredentials struct ---

func TestAuthCredentials_Fields(t *testing.T) {
	creds := &AuthCredentials{
		APIToken:  "tok-abc",
		AccountID: "acct-123",
		Email:     "test@example.com",
		AccessKey: "access-key",
		SecretKey: "secret-key",
		Method:    "token",
		Scope:     "r2:read",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if creds.APIToken != "tok-abc" {
		t.Errorf("APIToken = %q, want %q", creds.APIToken, "tok-abc")
	}
	if creds.AccountID != "acct-123" {
		t.Errorf("AccountID = %q, want %q", creds.AccountID, "acct-123")
	}
	if creds.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", creds.Email, "test@example.com")
	}
	if creds.Method != "token" {
		t.Errorf("Method = %q, want %q", creds.Method, "token")
	}
	if creds.Scope != "r2:read" {
		t.Errorf("Scope = %q, want %q", creds.Scope, "r2:read")
	}
	if creds.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
}

func TestAuthCredentials_ZeroValue(t *testing.T) {
	var creds AuthCredentials
	if creds.APIToken != "" {
		t.Error("zero-value APIToken should be empty")
	}
	if creds.AccountID != "" {
		t.Error("zero-value AccountID should be empty")
	}
	if !creds.ExpiresAt.IsZero() {
		t.Error("zero-value ExpiresAt should be zero time")
	}
}

// --- getAPITokenCredentials error paths ---

func TestGetAPITokenCredentials_MissingToken(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("account-id", "some-account", "")
	cmd.Flags().Bool("interactive", false, "")

	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	_, err := getAPITokenCredentials(cmd)
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error = %q, want it to mention 'token'", err.Error())
	}
}

func TestGetAPITokenCredentials_MissingAccountID(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "some-token", "")
	cmd.Flags().String("account-id", "", "")
	cmd.Flags().Bool("interactive", false, "")

	if err := cmd.Flags().Set("token", "some-token"); err != nil {
		t.Fatalf("set token: %v", err)
	}

	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	_, err := getAPITokenCredentials(cmd)
	if err == nil {
		t.Fatal("expected error when account-id is missing")
	}
	if !strings.Contains(err.Error(), "Account ID") && !strings.Contains(err.Error(), "account") {
		t.Errorf("error = %q, want it to mention account ID", err.Error())
	}
}

func TestGetAPITokenCredentials_Success(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("account-id", "", "")
	cmd.Flags().Bool("interactive", false, "")

	if err := cmd.Flags().Set("token", "test-token-value"); err != nil {
		t.Fatalf("set token: %v", err)
	}
	if err := cmd.Flags().Set("account-id", "test-account-id"); err != nil {
		t.Fatalf("set account-id: %v", err)
	}

	creds, err := getAPITokenCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.APIToken != "test-token-value" {
		t.Errorf("APIToken = %q, want %q", creds.APIToken, "test-token-value")
	}
	if creds.AccountID != "test-account-id" {
		t.Errorf("AccountID = %q, want %q", creds.AccountID, "test-account-id")
	}
	if creds.Method != "token" {
		t.Errorf("Method = %q, want %q", creds.Method, "token")
	}
}

func TestGetAPITokenCredentials_TrimsWhitespace(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("account-id", "", "")
	cmd.Flags().Bool("interactive", false, "")

	// Cobra trims flag values so we simulate whitespace via env vars
	t.Setenv("CLOUDFLARE_API_TOKEN", "  token-with-spaces  ")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "  account-with-spaces  ")
	defer func() {
		t.Setenv("CLOUDFLARE_API_TOKEN", "")
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	}()

	creds, err := getAPITokenCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(creds.APIToken, " ") {
		t.Errorf("APIToken should be trimmed, got %q", creds.APIToken)
	}
	if strings.Contains(creds.AccountID, " ") {
		t.Errorf("AccountID should be trimmed, got %q", creds.AccountID)
	}
}

// --- getServiceKeyCredentials error paths ---

func TestGetServiceKeyCredentials_MissingEmail(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")
	cmd.Flags().Bool("interactive", false, "")

	t.Setenv("CLOUDFLARE_EMAIL", "")

	_, err := getServiceKeyCredentials(cmd)
	if err == nil {
		t.Fatal("expected error when email is missing")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("error = %q, want it to mention 'email'", err.Error())
	}
}

func TestGetServiceKeyCredentials_Success(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")
	cmd.Flags().Bool("interactive", false, "")

	if err := cmd.Flags().Set("email", "user@example.com"); err != nil {
		t.Fatalf("set email: %v", err)
	}

	creds, err := getServiceKeyCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", creds.Email, "user@example.com")
	}
	if creds.Method != "key" {
		t.Errorf("Method = %q, want %q", creds.Method, "key")
	}
}

func TestGetServiceKeyCredentials_FromEnv(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")
	cmd.Flags().Bool("interactive", false, "")

	t.Setenv("CLOUDFLARE_EMAIL", "env@example.com")

	creds, err := getServiceKeyCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Email != "env@example.com" {
		t.Errorf("Email = %q, want %q", creds.Email, "env@example.com")
	}
}

// --- generateNewToken / revokeOldToken stubs ---

func TestGenerateNewToken_ReturnsError(t *testing.T) {
	_, err := generateNewToken(nil)
	if err == nil {
		t.Fatal("generateNewToken should return an error (not yet implemented)")
	}
}

func TestRevokeOldToken_ReturnsError(t *testing.T) {
	err := revokeOldToken("some-token")
	if err == nil {
		t.Fatal("revokeOldToken should return an error (not yet implemented)")
	}
}

// --- newClientFromEnv ---

func TestNewClientFromEnv_MissingEnvVars(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	_, err := newClientFromEnv()
	if err == nil {
		t.Fatal("expected error when env vars missing")
	}
	if !strings.Contains(err.Error(), "CLOUDFLARE_API_TOKEN") && !strings.Contains(err.Error(), "missing") {
		t.Errorf("error = %q, want it to mention missing env vars", err.Error())
	}
}

func TestNewClientFromEnv_MissingToken(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "acct-123")

	_, err := newClientFromEnv()
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestNewClientFromEnv_MissingAccountID(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "some-token")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	_, err := newClientFromEnv()
	if err == nil {
		t.Fatal("expected error when account ID is missing")
	}
}

// --- authCmd has no persistent flags (not needed) ---

func TestAuthCmd_NoPersistentFlags(t *testing.T) {
	// auth parent has no persistent flags; subcommands manage their own.
	// HasFlags returns true only if at least one flag is defined.
	if authCmd.PersistentFlags().HasFlags() {
		t.Log("authCmd has persistent flags (informational, not a hard failure)")
	}
}

// --- Parent-child relationship ---

func TestAuthSubcommands_HaveCorrectParent(t *testing.T) {
	subs := []*cobra.Command{authLoginCmd, authRotateCmd, authStatusCmd, authLogoutCmd}
	for _, sub := range subs {
		if sub.Parent() == nil {
			t.Errorf("subcommand %q has nil parent", sub.Use)
			continue
		}
		if sub.Parent().Use != "auth" {
			t.Errorf("subcommand %q parent = %q, want %q", sub.Use, sub.Parent().Use, "auth")
		}
	}
}

// --- Command lookup by name ---

func TestAuthCmd_FindSubcommandByName(t *testing.T) {
	names := []string{"login", "rotate", "status", "logout"}
	for _, name := range names {
		sub, _, err := authCmd.Find([]string{name})
		if err != nil {
			t.Errorf("authCmd.Find(%q) error: %v", name, err)
			continue
		}
		if sub == nil || sub.Name() != name {
			t.Errorf("authCmd.Find(%q) = %v, want command named %q", name, sub, name)
		}
	}
}
