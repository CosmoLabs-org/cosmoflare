package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestEmailCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("emailCmd not registered on rootCmd")
	}
}

func TestEmailCmd_Metadata(t *testing.T) {
	if emailCmd.Use != "email" {
		t.Errorf("emailCmd.Use = %q, want %q", emailCmd.Use, "email")
	}
	if emailCmd.Short == "" {
		t.Error("emailCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestEmailCmd_Subcommands(t *testing.T) {
	expected := []string{"rules", "destinations", "catchall", "settings", "enable", "disable"}
	for _, name := range expected {
		found := false
		for _, sub := range emailCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("email subcommand %q not registered", name)
		}
	}
}

func TestEmailRulesCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "get", "create", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range emailRulesCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("email rules subcommand %q not registered", name)
		}
	}
}

func TestEmailDestCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "add", "get", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range emailDestCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("email destinations subcommand %q not registered", name)
		}
	}
}

func TestEmailCatchallCmd_Subcommands(t *testing.T) {
	expected := []string{"update"}
	for _, name := range expected {
		found := false
		for _, sub := range emailCatchallCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("email catchall subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestEmailCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		emailRulesListCmd,
		emailRulesGetCmd,
		emailRulesCreateCmd,
		emailRulesUpdateCmd,
		emailRulesDeleteCmd,
		emailDestListCmd,
		emailDestAddCmd,
		emailDestGetCmd,
		emailDestDeleteCmd,
		emailCatchallCmd,
		emailCatchallUpdateCmd,
		emailSettingsCmd,
		emailEnableCmd,
		emailDisableCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestEmailRulesCreate_Flags(t *testing.T) {
	expected := []string{"name", "match-to", "match-all", "forward-to", "drop", "priority", "enabled"}
	for _, name := range expected {
		if emailRulesCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on emailRulesCreateCmd", name)
		}
	}
}

func TestEmailRulesCreate_RequiredFlags(t *testing.T) {
	f := emailRulesCreateCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("--name flag not found on emailRulesCreateCmd")
	}
}

func TestEmailRulesCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"match-all", "false"},
		{"drop", "false"},
		{"priority", "0"},
		{"enabled", "true"},
	}
	for _, tc := range cases {
		f := emailRulesCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestEmailRulesUpdate_Flags(t *testing.T) {
	expected := []string{"name", "match-to", "match-all", "forward-to", "drop", "priority", "enabled"}
	for _, name := range expected {
		if emailRulesUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on emailRulesUpdateCmd", name)
		}
	}
}

func TestEmailRulesDelete_ForceFlag(t *testing.T) {
	f := emailRulesDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on emailRulesDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestEmailDestAdd_Flags(t *testing.T) {
	f := emailDestAddCmd.Flags().Lookup("email")
	if f == nil {
		t.Fatal("--email flag not registered on emailDestAddCmd")
	}
}

func TestEmailDestDelete_ForceFlag(t *testing.T) {
	f := emailDestDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on emailDestDeleteCmd")
	}
}

func TestEmailCatchallUpdate_ForwardTo(t *testing.T) {
	f := emailCatchallUpdateCmd.Flags().Lookup("forward-to")
	if f == nil {
		t.Fatal("--forward-to flag not registered on emailCatchallUpdateCmd")
	}
}

// --- Arg validation (missing zone ID / rule ID) ---

func TestEmailRulesList_NoZoneID(t *testing.T) {
	err := runEmailRulesList(emailRulesListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestEmailRulesGet_NoArgs(t *testing.T) {
	err := runEmailRulesGet(emailRulesGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestEmailRulesGet_OneArg(t *testing.T) {
	err := runEmailRulesGet(emailRulesGetCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

func TestEmailRulesCreate_NoZoneID(t *testing.T) {
	err := runEmailRulesCreate(emailRulesCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailRulesCreate_NoMatcher(t *testing.T) {
	// Reset flags
	emailMatchTo = ""
	emailMatchAll = false
	emailForwardTo = ""
	emailDrop = false
	emailRuleName = "test"

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no matcher flags set")
	}
}

func TestEmailRulesCreate_MutuallyExclusiveMatchers(t *testing.T) {
	emailMatchTo = "user@example.com"
	emailMatchAll = true
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "test"

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when both --match-to and --match-all are set")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("mutually exclusive")) {
		t.Errorf("error = %q, want 'mutually exclusive'", err.Error())
	}
}

func TestEmailRulesCreate_MutuallyExclusiveActions(t *testing.T) {
	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = true
	emailRuleName = "test"

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when both --forward-to and --drop are set")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("mutually exclusive")) {
		t.Errorf("error = %q, want 'mutually exclusive'", err.Error())
	}
}

func TestEmailRulesUpdate_NoArgs(t *testing.T) {
	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestEmailRulesUpdate_OneArg(t *testing.T) {
	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided")
	}
}

func TestEmailRulesDelete_NoArgs(t *testing.T) {
	err := runEmailRulesDelete(emailRulesDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestEmailDestList_NoZoneID(t *testing.T) {
	err := runEmailDestList(emailDestListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailDestAdd_NoZoneID(t *testing.T) {
	err := runEmailDestAdd(emailDestAddCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailDestGet_NoArgs(t *testing.T) {
	err := runEmailDestGet(emailDestGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestEmailDestGet_OneArg(t *testing.T) {
	err := runEmailDestGet(emailDestGetCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing address ID)")
	}
}

func TestEmailDestDelete_NoArgs(t *testing.T) {
	err := runEmailDestDelete(emailDestDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestEmailCatchall_NoZoneID(t *testing.T) {
	err := runEmailCatchall(emailCatchallCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailCatchallUpdate_NoZoneID(t *testing.T) {
	err := runEmailCatchallUpdate(emailCatchallUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailSettings_NoZoneID(t *testing.T) {
	err := runEmailSettings(emailSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailEnable_NoZoneID(t *testing.T) {
	err := runEmailEnable(emailEnableCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestEmailDisable_NoZoneID(t *testing.T) {
	err := runEmailDisable(emailDisableCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

// --- DryRun mode ---

func TestEmailRulesCreate_DryRun(t *testing.T) {
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

	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "test-rule"

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runEmailRulesCreate(DryRun) returned error: %v", err)
	}
}

func TestEmailRulesCreate_DryRunJSON(t *testing.T) {
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

	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "test-rule"

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runEmailRulesCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestEmailRulesDelete_DryRun(t *testing.T) {
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

	err := runEmailRulesDelete(emailRulesDeleteCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runEmailRulesDelete(DryRun) returned error: %v", err)
	}
}

func TestEmailDestAdd_DryRun(t *testing.T) {
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
	emailAddr = "test@example.com"

	err := runEmailDestAdd(emailDestAddCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runEmailDestAdd(DryRun) returned error: %v", err)
	}
}

func TestEmailCatchallUpdate_DryRun(t *testing.T) {
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
	emailForwardTo = "catch@example.com"

	err := runEmailCatchallUpdate(emailCatchallUpdateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runEmailCatchallUpdate(DryRun) returned error: %v", err)
	}
}

func TestEmailEnable_DryRun(t *testing.T) {
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

	err := runEmailEnable(emailEnableCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runEmailEnable(DryRun) returned error: %v", err)
	}
}

func TestEmailDisable_DryRun(t *testing.T) {
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

	err := runEmailDisable(emailDisableCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runEmailDisable(DryRun) returned error: %v", err)
	}
}

// --- Flag variable wiring ---

func TestEmailFlagsParsing_MatchTo(t *testing.T) {
	cmd := &cobra.Command{}
	var matchTo string
	cmd.Flags().StringVar(&matchTo, "match-to", "", "")
	if err := cmd.Flags().Set("match-to", "user@example.com"); err != nil {
		t.Fatalf("failed to set --match-to: %v", err)
	}
	if matchTo != "user@example.com" {
		t.Errorf("matchTo = %q, want %q", matchTo, "user@example.com")
	}
}

func TestEmailFlagsParsing_MatchAll(t *testing.T) {
	cmd := &cobra.Command{}
	var matchAll bool
	cmd.Flags().BoolVar(&matchAll, "match-all", false, "")
	if matchAll {
		t.Error("matchAll should default to false")
	}
	if err := cmd.Flags().Set("match-all", "true"); err != nil {
		t.Fatalf("failed to set --match-all: %v", err)
	}
	if !matchAll {
		t.Error("matchAll should be true after setting --match-all=true")
	}
}

func TestEmailFlagsParsing_Drop(t *testing.T) {
	cmd := &cobra.Command{}
	var drop bool
	cmd.Flags().BoolVar(&drop, "drop", false, "")
	if err := cmd.Flags().Set("drop", "true"); err != nil {
		t.Fatalf("failed to set --drop: %v", err)
	}
	if !drop {
		t.Error("drop should be true after --drop=true")
	}
}

// --- Use field verification for all leaf subcommands ---

func TestEmailRulesLeafCmds_UseFields(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{emailRulesListCmd, "list [zone-id]"},
		{emailRulesGetCmd, "get [zone-id] [rule-id]"},
		{emailRulesCreateCmd, "create [zone-id]"},
		{emailRulesUpdateCmd, "update [zone-id] [rule-id]"},
		{emailRulesDeleteCmd, "delete [zone-id] [rule-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

func TestEmailDestLeafCmds_UseFields(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{emailDestListCmd, "list [zone-id]"},
		{emailDestAddCmd, "add [zone-id]"},
		{emailDestGetCmd, "get [zone-id] [address-id]"},
		{emailDestDeleteCmd, "delete [zone-id] [address-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

func TestEmailTopLevelCmds_UseFields(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{emailCatchallCmd, "catchall [zone-id]"},
		{emailCatchallUpdateCmd, "update [zone-id]"},
		{emailSettingsCmd, "settings [zone-id]"},
		{emailEnableCmd, "enable [zone-id]"},
		{emailDisableCmd, "disable [zone-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

// --- Short description non-empty for all subcommands ---

func TestEmailAllSubcmds_ShortNotEmpty(t *testing.T) {
	cmds := []*cobra.Command{
		emailRulesCmd,
		emailRulesListCmd,
		emailRulesGetCmd,
		emailRulesCreateCmd,
		emailRulesUpdateCmd,
		emailRulesDeleteCmd,
		emailDestCmd,
		emailDestListCmd,
		emailDestAddCmd,
		emailDestGetCmd,
		emailDestDeleteCmd,
		emailCatchallCmd,
		emailCatchallUpdateCmd,
		emailSettingsCmd,
		emailEnableCmd,
		emailDisableCmd,
	}
	for _, c := range cmds {
		if c.Short == "" {
			t.Errorf("command %q has empty Short description", c.Use)
		}
	}
}

// --- Long description non-empty for key commands ---

func TestEmailKeyCmd_LongNotEmpty(t *testing.T) {
	cmds := []*cobra.Command{
		emailCmd,
		emailRulesCmd,
		emailRulesCreateCmd,
		emailDestCmd,
		emailCatchallCmd,
	}
	for _, c := range cmds {
		if c.Long == "" {
			t.Errorf("command %q has empty Long description", c.Use)
		}
	}
}

// --- Long help text content checks ---

func TestEmailCmd_LongContainsExamples(t *testing.T) {
	if !bytes.Contains([]byte(emailCmd.Long), []byte("cosmoflare email")) {
		t.Error("emailCmd.Long should contain example commands starting with 'cosmoflare email'")
	}
}

func TestEmailRulesCreateCmd_LongContainsMutuallyExclusiveHint(t *testing.T) {
	long := emailRulesCreateCmd.Long
	if !bytes.Contains([]byte(long), []byte("--match-to")) {
		t.Error("emailRulesCreateCmd.Long should document --match-to flag")
	}
	if !bytes.Contains([]byte(long), []byte("--forward-to")) {
		t.Error("emailRulesCreateCmd.Long should document --forward-to flag")
	}
}

// --- Arg validation: missing second argument ---

func TestEmailRulesDelete_OneArg(t *testing.T) {
	err := runEmailRulesDelete(emailRulesDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

func TestEmailDestDelete_OneArg(t *testing.T) {
	err := runEmailDestDelete(emailDestDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing address ID)")
	}
}

// --- runEmailRulesCreate: no action flag ---

func TestEmailRulesCreate_NoAction(t *testing.T) {
	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = ""
	emailDrop = false
	emailRuleName = "test"

	err := runEmailRulesCreate(emailRulesCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no action flag (--forward-to or --drop) set")
	}
}

// --- runEmailRulesUpdate: mutually exclusive flag validation ---

func TestEmailRulesUpdate_MutuallyExclusiveMatchers(t *testing.T) {
	emailMatchTo = "user@example.com"
	emailMatchAll = true
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "test"

	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123", "rule456"})
	if err == nil {
		t.Fatal("expected error when both --match-to and --match-all set on update")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("mutually exclusive")) {
		t.Errorf("error = %q, want 'mutually exclusive'", err.Error())
	}
}

func TestEmailRulesUpdate_MutuallyExclusiveActions(t *testing.T) {
	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = true
	emailRuleName = "test"

	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123", "rule456"})
	if err == nil {
		t.Fatal("expected error when both --forward-to and --drop set on update")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("mutually exclusive")) {
		t.Errorf("error = %q, want 'mutually exclusive'", err.Error())
	}
}

// --- Dry-run for update and dest-delete ---

func TestEmailRulesUpdate_DryRun(t *testing.T) {
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

	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "updated-rule"

	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runEmailRulesUpdate(DryRun) returned error: %v", err)
	}
}

func TestEmailRulesUpdate_DryRunJSON(t *testing.T) {
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

	emailMatchTo = "user@example.com"
	emailMatchAll = false
	emailForwardTo = "dest@example.com"
	emailDrop = false
	emailRuleName = "updated-rule"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123", "rule456"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runEmailRulesUpdate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestEmailDestDelete_DryRun(t *testing.T) {
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

	err := runEmailDestDelete(emailDestDeleteCmd, []string{"zone123", "addr456"})
	if err != nil {
		t.Errorf("runEmailDestDelete(DryRun) returned error: %v", err)
	}
}

func TestEmailDestDelete_DryRunJSON(t *testing.T) {
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEmailDestDelete(emailDestDeleteCmd, []string{"zone123", "addr456"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runEmailDestDelete(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

// --- Required flag annotation checks ---

func TestEmailDestAddCmd_EmailFlagRequired(t *testing.T) {
	f := emailDestAddCmd.Flags().Lookup("email")
	if f == nil {
		t.Fatal("--email flag not registered on emailDestAddCmd")
	}
	annotations := f.Annotations
	if annotations == nil {
		t.Fatal("--email flag has no annotations (should be marked required)")
	}
	if _, ok := annotations[cobra.BashCompOneRequiredFlag]; !ok {
		t.Error("--email flag should be marked as required")
	}
}

func TestEmailRulesCreateCmd_NameFlagRequired(t *testing.T) {
	f := emailRulesCreateCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("--name flag not registered on emailRulesCreateCmd")
	}
	annotations := f.Annotations
	if annotations == nil {
		t.Fatal("--name flag has no annotations (should be marked required)")
	}
	if _, ok := annotations[cobra.BashCompOneRequiredFlag]; !ok {
		t.Error("--name flag should be marked as required")
	}
}

func TestEmailCatchallUpdateCmd_ForwardToFlagRequired(t *testing.T) {
	f := emailCatchallUpdateCmd.Flags().Lookup("forward-to")
	if f == nil {
		t.Fatal("--forward-to flag not registered on emailCatchallUpdateCmd")
	}
	annotations := f.Annotations
	if annotations == nil {
		t.Fatal("--forward-to flag has no annotations (should be marked required)")
	}
	if _, ok := annotations[cobra.BashCompOneRequiredFlag]; !ok {
		t.Error("--forward-to flag should be marked as required")
	}
}

// --- Flag type checks ---

func TestEmailRulesCreateCmd_FlagTypes(t *testing.T) {
	stringFlags := []string{"name", "match-to", "forward-to"}
	for _, name := range stringFlags {
		f := emailRulesCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "string")
		}
	}

	boolFlags := []string{"match-all", "drop", "enabled"}
	for _, name := range boolFlags {
		f := emailRulesCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "bool" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "bool")
		}
	}

	intFlags := []string{"priority"}
	for _, name := range intFlags {
		f := emailRulesCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Value.Type() != "int" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "int")
		}
	}
}

func TestEmailRulesUpdateCmd_FlagTypes(t *testing.T) {
	stringFlags := []string{"name", "match-to", "forward-to"}
	for _, name := range stringFlags {
		f := emailRulesUpdateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on emailRulesUpdateCmd", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("emailRulesUpdateCmd flag --%s type = %q, want %q", name, f.Value.Type(), "string")
		}
	}
}

// --- emailRulesUpdateCmd flag defaults ---

func TestEmailRulesUpdate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"match-all", "false"},
		{"drop", "false"},
		{"priority", "0"},
		{"enabled", "true"},
	}
	for _, tc := range cases {
		f := emailRulesUpdateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on emailRulesUpdateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("emailRulesUpdateCmd flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- emailRulesCmd and emailDestCmd: Short contains key context ---

func TestEmailRulesCmd_ShortContainsKeyword(t *testing.T) {
	short := emailRulesCmd.Short
	if !bytes.Contains([]byte(short), []byte("rule")) && !bytes.Contains([]byte(short), []byte("Rule")) {
		t.Errorf("emailRulesCmd.Short %q does not mention 'rule'", short)
	}
}

func TestEmailDestCmd_ShortContainsKeyword(t *testing.T) {
	short := emailDestCmd.Short
	if !bytes.Contains([]byte(short), []byte("destination")) && !bytes.Contains([]byte(short), []byte("Destination")) {
		t.Errorf("emailDestCmd.Short %q does not mention 'destination'", short)
	}
}

// --- emailRulesUpdate: one arg returns error ---

func TestEmailRulesUpdate_TwoArgs_DryRun(t *testing.T) {
	origDryRun := DryRun
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	AccountID = "test-account"
	APIToken = "test-token"

	emailMatchTo = ""
	emailMatchAll = false
	emailForwardTo = ""
	emailDrop = false
	emailRuleName = ""

	// Provide both zone and rule ID — should succeed in dry-run (no API call)
	err := runEmailRulesUpdate(emailRulesUpdateCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runEmailRulesUpdate(DryRun, no flags) returned error: %v", err)
	}
}
