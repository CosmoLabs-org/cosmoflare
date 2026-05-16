package cmd

import (
	"bytes"
	"testing"

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
	f := authRotateCmd.Flags().Lookup("revoke-old")
	if f == nil {
		t.Fatal("--revoke-old flag not found on authRotateCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--revoke-old default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestAuthRotate_NoProfile(t *testing.T) {
	err := runAuthRotate(authRotateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no profile provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("profile")) {
		t.Errorf("error = %q, want it to mention 'profile'", err.Error())
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
