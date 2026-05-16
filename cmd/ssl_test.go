package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestSSLCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "ssl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sslCmd not registered on rootCmd")
	}
}

func TestSSLCmd_Metadata(t *testing.T) {
	if sslCmd.Use != "ssl" {
		t.Errorf("sslCmd.Use = %q, want %q", sslCmd.Use, "ssl")
	}
	if sslCmd.Short == "" {
		t.Error("sslCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestSSLCmd_Subcommands(t *testing.T) {
	expected := []string{"status", "settings", "update", "verify"}
	for _, name := range expected {
		found := false
		for _, sub := range sslCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ssl subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestSSLCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		sslStatusCmd,
		sslSettingsCmd,
		sslUpdateCmd,
		sslVerifyCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestSSLUpdate_Flags(t *testing.T) {
	expected := []string{"mode", "min-tls", "always-https", "auto-rewrites"}
	for _, name := range expected {
		if sslUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on sslUpdateCmd", name)
		}
	}
}

func TestSSLUpdate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"mode", ""},
		{"min-tls", ""},
		{"always-https", "false"},
		{"auto-rewrites", "false"},
	}
	for _, tc := range cases {
		f := sslUpdateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on sslUpdateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Arg validation ---

func TestSSLStatus_NoZoneID(t *testing.T) {
	err := runSSLStatus(sslStatusCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestSSLSettings_NoZoneID(t *testing.T) {
	err := runSSLSettings(sslSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestSSLUpdate_NoZoneID(t *testing.T) {
	err := runSSLUpdate(sslUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestSSLUpdate_NoFlags(t *testing.T) {
	origSSLMode := sslMode
	defer func() { sslMode = origSSLMode }()

	sslMode = ""

	err := runSSLUpdate(sslUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no update flags provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("update flag")) {
		t.Errorf("error = %q, want it to mention 'update flag'", err.Error())
	}
}

func TestSSLVerify_NoZoneID(t *testing.T) {
	err := runSSLVerify(sslVerifyCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

// --- DryRun mode ---

func TestSSLUpdate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origSSLMode := sslMode
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		sslMode = origSSLMode
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	sslMode = "full"

	err := runSSLUpdate(sslUpdateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runSSLUpdate(DryRun) returned error: %v", err)
	}
}

func TestSSLUpdate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origSSLMode := sslMode
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		sslMode = origSSLMode
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"
	sslMode = "strict"

	err := runSSLUpdate(sslUpdateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runSSLUpdate(DryRun+JSON) returned error: %v", err)
	}
}

// --- Flag variable wiring ---

func TestSSLFlagsParsing_Mode(t *testing.T) {
	cmd := &cobra.Command{}
	var mode string
	cmd.Flags().StringVar(&mode, "mode", "", "")
	if err := cmd.Flags().Set("mode", "full"); err != nil {
		t.Fatalf("failed to set --mode: %v", err)
	}
	if mode != "full" {
		t.Errorf("mode = %q, want %q", mode, "full")
	}
}

func TestSSLFlagsParsing_AlwaysHTTPS(t *testing.T) {
	cmd := &cobra.Command{}
	var alwaysHTTPS bool
	cmd.Flags().BoolVar(&alwaysHTTPS, "always-https", false, "")
	if alwaysHTTPS {
		t.Error("alwaysHTTPS should default to false")
	}
	if err := cmd.Flags().Set("always-https", "true"); err != nil {
		t.Fatalf("failed to set --always-https: %v", err)
	}
	if !alwaysHTTPS {
		t.Error("alwaysHTTPS should be true after --always-https=true")
	}
}

func TestSSLFlagsParsing_MinTLS(t *testing.T) {
	cmd := &cobra.Command{}
	var minTLS string
	cmd.Flags().StringVar(&minTLS, "min-tls", "", "")
	if err := cmd.Flags().Set("min-tls", "1.2"); err != nil {
		t.Fatalf("failed to set --min-tls: %v", err)
	}
	if minTLS != "1.2" {
		t.Errorf("minTLS = %q, want %q", minTLS, "1.2")
	}
}
