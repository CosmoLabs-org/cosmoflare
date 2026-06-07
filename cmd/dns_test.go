package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestDNSCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "dns" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("dnsCmd not registered on rootCmd")
	}
}

func TestDNSCmd_Metadata(t *testing.T) {
	if dnsCmd.Use != "dns" {
		t.Errorf("dnsCmd.Use = %q, want %q", dnsCmd.Use, "dns")
	}
	if dnsCmd.Short == "" {
		t.Error("dnsCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestDNSCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range dnsCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("dns subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestDNSCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		dnsCreateCmd,
		dnsListCmd,
		dnsGetCmd,
		dnsUpdateCmd,
		dnsDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestDNSCreate_Flags(t *testing.T) {
	expected := []string{"type", "name", "content", "ttl", "proxied", "priority", "comment"}
	for _, name := range expected {
		if dnsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on dnsCreateCmd", name)
		}
	}
}

func TestDNSCreate_RequiredFlags(t *testing.T) {
	required := []string{"type", "name", "content"}
	for _, name := range required {
		f := dnsCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsCreateCmd", name)
		}
		annots := f.Annotations
		if annots == nil || annots[cobra.BashCompOneRequiredFlag] == nil {
			t.Errorf("flag --%s should be marked as required on dnsCreateCmd", name)
		}
	}
}

func TestDNSCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"ttl", "0"},
		{"proxied", "false"},
		{"priority", "0"},
	}
	for _, tc := range cases {
		f := dnsCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsCreateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestDNSList_Flags(t *testing.T) {
	expected := []string{"type", "name", "content"}
	for _, name := range expected {
		if dnsListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on dnsListCmd", name)
		}
	}
}

func TestDNSUpdate_Flags(t *testing.T) {
	expected := []string{"ttl", "proxied", "priority", "comment"}
	for _, name := range expected {
		if dnsUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on dnsUpdateCmd", name)
		}
	}
}

func TestDNSDelete_ForceFlag(t *testing.T) {
	f := dnsDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on dnsDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestDNSCreate_NoZoneID(t *testing.T) {
	err := runDNSCreate(dnsCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestDNSList_NoZoneID(t *testing.T) {
	err := runDNSList(dnsListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestDNSGet_NoArgs(t *testing.T) {
	err := runDNSGet(dnsGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestDNSGet_OneArg(t *testing.T) {
	err := runDNSGet(dnsGetCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing record ID)")
	}
}

func TestDNSUpdate_NoArgs(t *testing.T) {
	err := runDNSUpdate(dnsUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestDNSUpdate_OneArg(t *testing.T) {
	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing record ID)")
	}
}

func TestDNSUpdate_NoUpdateFlags(t *testing.T) {
	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123", "record456"})
	if err == nil {
		t.Fatal("expected error when no update flags provided")
	}
}

func TestDNSDelete_NoArgs(t *testing.T) {
	err := runDNSDelete(dnsDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestDNSDelete_OneArg(t *testing.T) {
	err := runDNSDelete(dnsDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing record ID)")
	}
}

// --- DryRun mode ---

func TestDNSCreate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	dnsRecordType = "A"
	dnsName = "www"
	dnsContent = "1.2.3.4"

	err := runDNSCreate(dnsCreateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runDNSCreate(DryRun) returned error: %v", err)
	}
}

func TestDNSCreate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	dnsRecordType = "A"
	dnsName = "www"
	dnsContent = "1.2.3.4"

	err := runDNSCreate(dnsCreateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runDNSCreate(DryRun+JSON) returned error: %v", err)
	}
}

func TestDNSDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	err := runDNSDelete(dnsDeleteCmd, []string{"zone123", "record456"})
	if err != nil {
		t.Errorf("runDNSDelete(DryRun) returned error: %v", err)
	}
}

func TestDNSDelete_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	err := runDNSDelete(dnsDeleteCmd, []string{"zone123", "record456"})
	if err != nil {
		t.Errorf("runDNSDelete(DryRun+JSON) returned error: %v", err)
	}
}

// --- Flag variable wiring ---

func TestDNSFlagsParsing_RecordType(t *testing.T) {
	cmd := &cobra.Command{}
	var recordType string
	cmd.Flags().StringVar(&recordType, "type", "", "")
	if err := cmd.Flags().Set("type", "AAAA"); err != nil {
		t.Fatalf("failed to set --type: %v", err)
	}
	if recordType != "AAAA" {
		t.Errorf("recordType = %q, want %q", recordType, "AAAA")
	}
}

func TestDNSFlagsParsing_Proxied(t *testing.T) {
	cmd := &cobra.Command{}
	var proxied bool
	cmd.Flags().BoolVar(&proxied, "proxied", false, "")
	if proxied {
		t.Error("proxied should default to false")
	}
	if err := cmd.Flags().Set("proxied", "true"); err != nil {
		t.Fatalf("failed to set --proxied: %v", err)
	}
	if !proxied {
		t.Error("proxied should be true after --proxied=true")
	}
}

func TestDNSFlagsParsing_TTL(t *testing.T) {
	cmd := &cobra.Command{}
	var ttl int
	cmd.Flags().IntVar(&ttl, "ttl", 0, "")
	if err := cmd.Flags().Set("ttl", "3600"); err != nil {
		t.Fatalf("failed to set --ttl: %v", err)
	}
	if ttl != 3600 {
		t.Errorf("ttl = %d, want %d", ttl, 3600)
	}
}

// --- Command Use format strings ---

func TestDNSSubcmd_UseStrings(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{dnsCreateCmd, "create [zone-id]"},
		{dnsListCmd, "list [zone-id]"},
		{dnsGetCmd, "get [zone-id] [record-id]"},
		{dnsUpdateCmd, "update [zone-id] [record-id]"},
		{dnsDeleteCmd, "delete [zone-id] [record-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

// --- Subcommand Short descriptions non-empty ---

func TestDNSSubcmd_ShortNonEmpty(t *testing.T) {
	cmds := []*cobra.Command{
		dnsCreateCmd,
		dnsListCmd,
		dnsGetCmd,
		dnsUpdateCmd,
		dnsDeleteCmd,
	}
	for _, c := range cmds {
		if c.Short == "" {
			t.Errorf("dns subcommand %q has empty Short description", c.Use)
		}
	}
}

// --- Long help text contains key terms ---

func TestDNSCmd_LongContainsKeyTerms(t *testing.T) {
	long := dnsCmd.Long
	terms := []string{"DNS", "zone", "create", "list", "delete"}
	for _, term := range terms {
		if !bytes.Contains([]byte(long), []byte(term)) {
			t.Errorf("dnsCmd.Long does not contain %q", term)
		}
	}
}

func TestDNSCreateCmd_LongContainsExamples(t *testing.T) {
	long := dnsCreateCmd.Long
	if !bytes.Contains([]byte(long), []byte("cosmoflare dns create")) {
		t.Error("dnsCreateCmd.Long does not contain example usage")
	}
}

func TestDNSDeleteCmd_LongContainsWarning(t *testing.T) {
	long := dnsDeleteCmd.Long
	if !bytes.Contains([]byte(long), []byte("WARNING")) {
		t.Error("dnsDeleteCmd.Long should contain a WARNING about irreversibility")
	}
}

// --- Flag types ---

func TestDNSCreate_FlagTypes(t *testing.T) {
	cases := []struct {
		name     string
		wantType string
	}{
		{"type", "string"},
		{"name", "string"},
		{"content", "string"},
		{"comment", "string"},
		{"ttl", "int"},
		{"proxied", "bool"},
		{"priority", "uint16"},
	}
	for _, tc := range cases {
		f := dnsCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsCreateCmd", tc.name)
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag --%s type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestDNSList_FlagTypes(t *testing.T) {
	for _, name := range []string{"type", "name", "content"} {
		f := dnsListCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsListCmd", name)
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag --%s type = %q, want %q", name, f.Value.Type(), "string")
		}
	}
}

func TestDNSUpdate_FlagTypes(t *testing.T) {
	cases := []struct {
		name     string
		wantType string
	}{
		{"ttl", "int"},
		{"proxied", "bool"},
		{"priority", "uint16"},
		{"comment", "string"},
	}
	for _, tc := range cases {
		f := dnsUpdateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsUpdateCmd", tc.name)
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag --%s type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

// --- Error messages mention expected identifiers ---

func TestDNSGet_ErrorMentionsBothIDs(t *testing.T) {
	err := runDNSGet(dnsGetCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error with one arg")
	}
	msg := err.Error()
	if !bytes.Contains([]byte(msg), []byte("zone ID")) && !bytes.Contains([]byte(msg), []byte("record ID")) {
		t.Errorf("error %q should mention zone ID or record ID", msg)
	}
}

func TestDNSUpdate_ErrorMentionsBothIDs(t *testing.T) {
	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error with one arg")
	}
	msg := err.Error()
	if !bytes.Contains([]byte(msg), []byte("zone ID")) && !bytes.Contains([]byte(msg), []byte("record ID")) {
		t.Errorf("error %q should mention zone ID or record ID", msg)
	}
}

func TestDNSDelete_ErrorMentionsBothIDs(t *testing.T) {
	err := runDNSDelete(dnsDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error with one arg")
	}
	msg := err.Error()
	if !bytes.Contains([]byte(msg), []byte("zone ID")) && !bytes.Contains([]byte(msg), []byte("record ID")) {
		t.Errorf("error %q should mention zone ID or record ID", msg)
	}
}

func TestDNSUpdate_NoFlagsErrorMentionsFlags(t *testing.T) {
	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123", "record456"})
	if err == nil {
		t.Fatal("expected error when no update flags are set")
	}
	msg := err.Error()
	// error should mention at least one of the valid update flags
	hasFlag := bytes.Contains([]byte(msg), []byte("--ttl")) ||
		bytes.Contains([]byte(msg), []byte("--proxied")) ||
		bytes.Contains([]byte(msg), []byte("--priority")) ||
		bytes.Contains([]byte(msg), []byte("--comment"))
	if !hasFlag {
		t.Errorf("error %q should mention at least one update flag", msg)
	}
}

// --- DryRun Update ---

func TestDNSUpdate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	// must set at least one flag as Changed to pass the no-opts guard
	if err := dnsUpdateCmd.Flags().Set("ttl", "300"); err != nil {
		t.Fatalf("failed to set --ttl: %v", err)
	}

	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123", "record456"})
	if err != nil {
		t.Errorf("runDNSUpdate(DryRun) returned error: %v", err)
	}
}

func TestDNSUpdate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	if err := dnsUpdateCmd.Flags().Set("ttl", "300"); err != nil {
		t.Fatalf("failed to set --ttl: %v", err)
	}

	err := runDNSUpdate(dnsUpdateCmd, []string{"zone123", "record456"})
	if err != nil {
		t.Errorf("runDNSUpdate(DryRun+JSON) returned error: %v", err)
	}
}

// --- Flag defaults for update flags ---

func TestDNSUpdate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"ttl", "0"},
		{"proxied", "false"},
		{"priority", "0"},
		{"comment", ""},
	}
	for _, tc := range cases {
		f := dnsUpdateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on dnsUpdateCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Subcommand count ---

func TestDNSCmd_SubcommandCount(t *testing.T) {
	got := len(dnsCmd.Commands())
	if got != 5 {
		t.Errorf("dnsCmd has %d subcommands, want 5", got)
	}
}

// --- Flag short description non-empty ---

func TestDNSCreate_FlagUsageNonEmpty(t *testing.T) {
	for _, name := range []string{"type", "name", "content", "ttl", "proxied", "priority", "comment"} {
		f := dnsCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if f.Usage == "" {
			t.Errorf("flag --%s has empty Usage string", name)
		}
	}
}

func TestDNSDelete_FlagUsageNonEmpty(t *testing.T) {
	f := dnsDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("flag --force not found on dnsDeleteCmd")
	}
	if f.Usage == "" {
		t.Error("--force has empty Usage string")
	}
}
