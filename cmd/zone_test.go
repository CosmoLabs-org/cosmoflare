package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestZoneCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "zone" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("zoneCmd not registered on rootCmd")
	}
}

func TestZoneCmd_Metadata(t *testing.T) {
	if zoneCmd.Use != "zone" {
		t.Errorf("zoneCmd.Use = %q, want %q", zoneCmd.Use, "zone")
	}
	if zoneCmd.Short == "" {
		t.Error("zoneCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestZoneCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "settings", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range zoneCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("zone subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestZoneCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		zoneCreateCmd,
		zoneListCmd,
		zoneGetCmd,
		zoneSettingsCmd,
		zoneDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestZoneCreate_Flags(t *testing.T) {
	f := zoneCreateCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not registered on zoneCreateCmd")
	}
	if f.DefValue != "full" {
		t.Errorf("--type default = %q, want %q", f.DefValue, "full")
	}
}

func TestZoneDelete_ForceFlag(t *testing.T) {
	f := zoneDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on zoneDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestZoneCreate_NoName(t *testing.T) {
	err := runZoneCreate(zoneCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("name")) {
		t.Errorf("error = %q, want it to mention 'name'", err.Error())
	}
}

func TestZoneGet_NoZoneID(t *testing.T) {
	err := runZoneGet(zoneGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestZoneSettings_NoZoneID(t *testing.T) {
	err := runZoneSettings(zoneSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestZoneDelete_NoZoneID(t *testing.T) {
	err := runZoneDelete(zoneDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

// --- DryRun mode ---

func TestZoneCreate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	err := runZoneCreate(zoneCreateCmd, []string{"example.com"})
	if err != nil {
		t.Errorf("runZoneCreate(DryRun) returned error: %v", err)
	}
}

func TestZoneCreate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	AccountID = "test-account"
	APIToken = "test-token"

	err := runZoneCreate(zoneCreateCmd, []string{"example.com"})
	if err != nil {
		t.Errorf("runZoneCreate(DryRun+JSON) returned error: %v", err)
	}
}

func TestZoneDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	err := runZoneDelete(zoneDeleteCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runZoneDelete(DryRun) returned error: %v", err)
	}
}

func TestZoneDelete_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	AccountID = "test-account"
	APIToken = "test-token"

	err := runZoneDelete(zoneDeleteCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runZoneDelete(DryRun+JSON) returned error: %v", err)
	}
}

// --- Flag variable wiring ---

func TestZoneFlagsParsing_Type(t *testing.T) {
	cmd := &cobra.Command{}
	var zType string
	cmd.Flags().StringVar(&zType, "type", "full", "")
	if zType != "full" {
		t.Errorf("type default = %q, want %q", zType, "full")
	}
	if err := cmd.Flags().Set("type", "partial"); err != nil {
		t.Fatalf("failed to set --type: %v", err)
	}
	if zType != "partial" {
		t.Errorf("type = %q, want %q", zType, "partial")
	}
}

func TestZoneFlagsParsing_Force(t *testing.T) {
	cmd := &cobra.Command{}
	var force bool
	cmd.Flags().BoolVar(&force, "force", false, "")
	if force {
		t.Error("force should default to false")
	}
	if err := cmd.Flags().Set("force", "true"); err != nil {
		t.Fatalf("failed to set --force: %v", err)
	}
	if !force {
		t.Error("force should be true after --force=true")
	}
}
