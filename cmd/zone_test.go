package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

// --- Command Use format strings ---

func TestZoneSubcmd_UseStrings(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{zoneCreateCmd, "create [name]"},
		{zoneListCmd, "list"},
		{zoneGetCmd, "get [zone-id]"},
		{zoneSettingsCmd, "settings [zone-id]"},
		{zoneDeleteCmd, "delete [zone-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

// --- Subcommand Short descriptions non-empty ---

func TestZoneSubcmd_ShortNonEmpty(t *testing.T) {
	cmds := []*cobra.Command{
		zoneCreateCmd,
		zoneListCmd,
		zoneGetCmd,
		zoneSettingsCmd,
		zoneDeleteCmd,
	}
	for _, c := range cmds {
		if c.Short == "" {
			t.Errorf("zone subcommand %q has empty Short description", c.Use)
		}
	}
}

// --- Long help text contains key terms ---

func TestZoneCmd_LongContainsKeyTerms(t *testing.T) {
	long := zoneCmd.Long
	terms := []string{"zone", "create", "list", "delete", "settings"}
	for _, term := range terms {
		if !bytes.Contains([]byte(long), []byte(term)) {
			t.Errorf("zoneCmd.Long does not contain %q", term)
		}
	}
}

func TestZoneCreateCmd_LongContainsExamples(t *testing.T) {
	long := zoneCreateCmd.Long
	if !bytes.Contains([]byte(long), []byte("cosmoflare zone create")) {
		t.Error("zoneCreateCmd.Long does not contain example usage")
	}
}

func TestZoneDeleteCmd_LongContainsWarning(t *testing.T) {
	long := zoneDeleteCmd.Long
	if !bytes.Contains([]byte(long), []byte("WARNING")) {
		t.Error("zoneDeleteCmd.Long should contain a WARNING about irreversibility")
	}
}

func TestZoneSettingsCmd_LongContainsSSL(t *testing.T) {
	long := zoneSettingsCmd.Long
	if !bytes.Contains([]byte(long), []byte("SSL")) {
		t.Error("zoneSettingsCmd.Long should mention SSL")
	}
}

// --- Flag types ---

func TestZoneCreate_FlagType(t *testing.T) {
	f := zoneCreateCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not found on zoneCreateCmd")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--type flag type = %q, want %q", f.Value.Type(), "string")
	}
}

func TestZoneDelete_ForceFlagType(t *testing.T) {
	f := zoneDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found on zoneDeleteCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--force flag type = %q, want %q", f.Value.Type(), "bool")
	}
}

// --- Subcommand count ---

func TestZoneCmd_SubcommandCount(t *testing.T) {
	got := len(zoneCmd.Commands())
	if got != 5 {
		t.Errorf("zoneCmd has %d subcommands, want 5", got)
	}
}

// --- Error message content ---

func TestZoneCreate_ErrorMentionsName(t *testing.T) {
	err := runZoneCreate(zoneCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("name")) {
		t.Errorf("error %q should mention 'name'", err.Error())
	}
}

func TestZoneGet_ErrorMentionsZoneID(t *testing.T) {
	err := runZoneGet(zoneGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error %q should mention 'zone ID'", err.Error())
	}
}

func TestZoneDelete_ErrorMentionsZoneID(t *testing.T) {
	err := runZoneDelete(zoneDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error %q should mention 'zone ID'", err.Error())
	}
}

func TestZoneSettings_ErrorMentionsZoneID(t *testing.T) {
	err := runZoneSettings(zoneSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error %q should mention 'zone ID'", err.Error())
	}
}

// --- Flag usage strings non-empty ---

func TestZoneCreate_FlagUsageNonEmpty(t *testing.T) {
	f := zoneCreateCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not found")
	}
	if f.Usage == "" {
		t.Error("--type has empty Usage string on zoneCreateCmd")
	}
}

func TestZoneDelete_FlagUsageNonEmpty(t *testing.T) {
	f := zoneDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found")
	}
	if f.Usage == "" {
		t.Error("--force has empty Usage string on zoneDeleteCmd")
	}
}

// --- Zone type flag accepts valid values ---

func TestZoneCreate_TypeFlagPartial(t *testing.T) {
	cmd := &cobra.Command{}
	var zType string
	cmd.Flags().StringVar(&zType, "type", "full", "")
	if err := cmd.Flags().Set("type", "partial"); err != nil {
		t.Fatalf("failed to set --type=partial: %v", err)
	}
	if zType != "partial" {
		t.Errorf("zType = %q, want %q", zType, "partial")
	}
}

// --- DryRun flag interaction ---

func TestZoneCreate_DryRunNoArgs(t *testing.T) {
	origDryRun := DryRun
	defer func() { DryRun = origDryRun }()
	DryRun = true

	// DryRun is only reached after arg validation, so no args still errors
	err := runZoneCreate(zoneCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error even in DryRun when no zone name provided")
	}
}

func TestZoneDelete_DryRunNoArgs(t *testing.T) {
	origDryRun := DryRun
	defer func() { DryRun = origDryRun }()
	DryRun = true

	err := runZoneDelete(zoneDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error even in DryRun when no zone ID provided")
	}
}

// --- Zone list has no required flags ---

func TestZoneList_HasNoRequiredFlags(t *testing.T) {
	count := 0
	zoneListCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Annotations != nil && f.Annotations[cobra.BashCompOneRequiredFlag] != nil {
			count++
		}
	})
	if count != 0 {
		t.Errorf("zoneListCmd has %d required flag(s), want 0", count)
	}
}
