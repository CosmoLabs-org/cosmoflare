package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// ========== Firewall ==========

// --- Command registration ---

func TestFirewallCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "firewall" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("firewallCmd not registered on rootCmd")
	}
}

func TestFirewallCmd_Metadata(t *testing.T) {
	if firewallCmd.Use != "firewall" {
		t.Errorf("firewallCmd.Use = %q, want %q", firewallCmd.Use, "firewall")
	}
	if firewallCmd.Short == "" {
		t.Error("firewallCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestFirewallCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "get", "create", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range firewallCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("firewall subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestFirewallCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		firewallListCmd,
		firewallGetCmd,
		firewallCreateCmd,
		firewallUpdateCmd,
		firewallDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestFirewallCreate_Flags(t *testing.T) {
	expected := []string{"expression", "action", "description", "priority", "paused"}
	for _, name := range expected {
		if firewallCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on firewallCreateCmd", name)
		}
	}
}

func TestFirewallCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"paused", "false"},
		{"priority", "0"},
	}
	for _, tc := range cases {
		f := firewallCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestFirewallCreate_RequiredFlags(t *testing.T) {
	for _, name := range []string{"expression", "action"} {
		f := firewallCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("--%s flag not found on firewallCreateCmd", name)
		}
	}
}

func TestFirewallUpdate_Flags(t *testing.T) {
	expected := []string{"expression", "action", "description", "priority", "paused"}
	for _, name := range expected {
		if firewallUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on firewallUpdateCmd", name)
		}
	}
}

func TestFirewallDelete_ForceFlag(t *testing.T) {
	f := firewallDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on firewallDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestFirewallList_NoZoneID(t *testing.T) {
	err := runFirewallList(firewallListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestFirewallGet_NoArgs(t *testing.T) {
	err := runFirewallGet(firewallGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestFirewallGet_OneArg(t *testing.T) {
	err := runFirewallGet(firewallGetCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

func TestFirewallCreate_NoZoneID(t *testing.T) {
	err := runFirewallCreate(firewallCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestFirewallUpdate_NoArgs(t *testing.T) {
	err := runFirewallUpdate(firewallUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestFirewallUpdate_OneArg(t *testing.T) {
	err := runFirewallUpdate(firewallUpdateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

func TestFirewallDelete_NoArgs(t *testing.T) {
	err := runFirewallDelete(firewallDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestFirewallDelete_OneArg(t *testing.T) {
	err := runFirewallDelete(firewallDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

// --- DryRun mode ---

func TestFirewallCreate_DryRun(t *testing.T) {
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
	fwExpression = "(ip.src eq 1.2.3.4)"
	fwAction = "block"

	err := runFirewallCreate(firewallCreateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runFirewallCreate(DryRun) returned error: %v", err)
	}
}

func TestFirewallCreate_DryRunJSON(t *testing.T) {
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
	fwExpression = "(ip.src eq 1.2.3.4)"
	fwAction = "block"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runFirewallCreate(firewallCreateCmd, []string{"zone123"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runFirewallCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestFirewallUpdate_DryRun(t *testing.T) {
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

	err := runFirewallUpdate(firewallUpdateCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runFirewallUpdate(DryRun) returned error: %v", err)
	}
}

func TestFirewallDelete_DryRun(t *testing.T) {
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

	err := runFirewallDelete(firewallDeleteCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runFirewallDelete(DryRun) returned error: %v", err)
	}
}

// ========== WAF ==========

// --- Command registration ---

func TestWAFCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "waf" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("wafCmd not registered on rootCmd")
	}
}

func TestWAFCmd_Metadata(t *testing.T) {
	if wafCmd.Use != "waf" {
		t.Errorf("wafCmd.Use = %q, want %q", wafCmd.Use, "waf")
	}
	if wafCmd.Short == "" {
		t.Error("wafCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestWAFCmd_Subcommands(t *testing.T) {
	expected := []string{"packages", "rules", "rule", "access"}
	for _, name := range expected {
		found := false
		for _, sub := range wafCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("waf subcommand %q not registered", name)
		}
	}
}

func TestWAFAccessCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "create", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range wafAccessCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("waf access subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestWAFCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		wafPackagesCmd,
		wafRulesCmd,
		wafRuleCmd,
		wafAccessListCmd,
		wafAccessCreateCmd,
		wafAccessDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestWAFRule_ModeFlag(t *testing.T) {
	f := wafRuleCmd.Flags().Lookup("mode")
	if f == nil {
		t.Fatal("--mode flag not registered on wafRuleCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--mode default = %q, want empty string", f.DefValue)
	}
}

func TestWAFAccessCreate_Flags(t *testing.T) {
	expected := []string{"ip", "mode", "note"}
	for _, name := range expected {
		if wafAccessCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on wafAccessCreateCmd", name)
		}
	}
}

func TestWAFAccessDelete_ForceFlag(t *testing.T) {
	f := wafAccessDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on wafAccessDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestWAFPackages_NoZoneID(t *testing.T) {
	err := runWAFPackages(wafPackagesCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestWAFRules_NoArgs(t *testing.T) {
	err := runWAFRules(wafRulesCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestWAFRules_OneArg(t *testing.T) {
	err := runWAFRules(wafRulesCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing package ID)")
	}
}

func TestWAFRule_NoArgs(t *testing.T) {
	err := runWAFRule(wafRuleCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestWAFRule_OneArg(t *testing.T) {
	err := runWAFRule(wafRuleCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error with only zone ID")
	}
}

func TestWAFRule_TwoArgs(t *testing.T) {
	err := runWAFRule(wafRuleCmd, []string{"zone123", "pkg456"})
	if err == nil {
		t.Fatal("expected error with only zone ID and package ID (missing rule ID)")
	}
}

func TestWAFAccessList_NoZoneID(t *testing.T) {
	err := runWAFAccessList(wafAccessListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestWAFAccessCreate_NoZoneID(t *testing.T) {
	err := runWAFAccessCreate(wafAccessCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
}

func TestWAFAccessCreate_NoIP(t *testing.T) {
	wafAccessIP = ""
	wafAccessMode = "block"
	err := runWAFAccessCreate(wafAccessCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no --ip provided")
	}
}

func TestWAFAccessCreate_NoMode(t *testing.T) {
	wafAccessIP = "1.2.3.4"
	wafAccessMode = ""
	err := runWAFAccessCreate(wafAccessCreateCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no --mode provided")
	}
}

func TestWAFAccessDelete_NoArgs(t *testing.T) {
	err := runWAFAccessDelete(wafAccessDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestWAFAccessDelete_OneArg(t *testing.T) {
	err := runWAFAccessDelete(wafAccessDeleteCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when only zone ID provided (missing rule ID)")
	}
}

// --- DryRun mode ---

func TestWAFRule_DryRun(t *testing.T) {
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
	wafRuleMode = "block"

	err := runWAFRule(wafRuleCmd, []string{"zone123", "pkg456", "rule789"})
	if err != nil {
		t.Errorf("runWAFRule(DryRun) returned error: %v", err)
	}
}

func TestWAFAccessCreate_DryRun(t *testing.T) {
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
	wafAccessIP = "1.2.3.4"
	wafAccessMode = "block"

	err := runWAFAccessCreate(wafAccessCreateCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runWAFAccessCreate(DryRun) returned error: %v", err)
	}
}

func TestWAFAccessDelete_DryRun(t *testing.T) {
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

	err := runWAFAccessDelete(wafAccessDeleteCmd, []string{"zone123", "rule456"})
	if err != nil {
		t.Errorf("runWAFAccessDelete(DryRun) returned error: %v", err)
	}
}
