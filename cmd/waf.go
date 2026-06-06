package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var wafCmd = &cobra.Command{
	Use:   "waf",
	Short: "Manage Cloudflare WAF and firewall rules",
	Long: `WAF and firewall management for Cloudflare zones.

Commands:
  packages    List WAF packages
  rules       List WAF rules in a package
  rule        Get or update a single WAF rule
  access list     List IP access rules
  access create   Create an IP access rule
  access delete   Delete an IP access rule

WAF is zone-scoped, so a zone ID is required for all operations.

Examples:
  cosmoflare waf packages ZONE_ID --json
  cosmoflare waf rules ZONE_ID PACKAGE_ID
  cosmoflare waf rule ZONE_ID PACKAGE_ID RULE_ID --mode=block
  cosmoflare waf access list ZONE_ID
  cosmoflare waf access create ZONE_ID --ip=1.2.3.4 --mode=block`,
}

var wafAccessCmd = &cobra.Command{
	Use:   "access",
	Short: "Manage IP access rules",
}

var (
	wafRuleMode   string
	wafAccessIP   string
	wafAccessMode string
	wafAccessNote string
	wafForce      bool
)

var wafPackagesCmd = &cobra.Command{
	Use:   "packages [zone-id]",
	Short: "List WAF packages",
	Long: `List all WAF managed ruleset packages for a zone.

Examples:
  cosmoflare waf packages ZONE_ID
  cosmoflare waf packages ZONE_ID --json`,
	RunE: runWAFPackages,
}

var wafRulesCmd = &cobra.Command{
	Use:   "rules [zone-id] [package-id]",
	Short: "List WAF rules in a package",
	Long: `List all WAF rules within a specific package.

Examples:
  cosmoflare waf rules ZONE_ID PACKAGE_ID
  cosmoflare waf rules ZONE_ID PACKAGE_ID --json`,
	RunE: runWAFRules,
}

var wafRuleCmd = &cobra.Command{
	Use:   "rule [zone-id] [package-id] [rule-id]",
	Short: "Get or update a WAF rule",
	Long: `Get details of a WAF rule, or update its mode.

Valid modes: block, simulate, disable, default, challenge

Examples:
  cosmoflare waf rule ZONE_ID PACKAGE_ID RULE_ID
  cosmoflare waf rule ZONE_ID PACKAGE_ID RULE_ID --mode=block
  cosmoflare waf rule ZONE_ID PACKAGE_ID RULE_ID --mode=simulate --json`,
	RunE: runWAFRule,
}

var wafAccessListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List IP access rules",
	Long: `List all zone-level IP access rules.

Examples:
  cosmoflare waf access list ZONE_ID
  cosmoflare waf access list ZONE_ID --json`,
	RunE: runWAFAccessList,
}

var wafAccessCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create an IP access rule",
	Long: `Create a zone-level IP access rule.

Modes: block, challenge, whitelist, js_challenge

Examples:
  cosmoflare waf access create ZONE_ID --ip=1.2.3.4 --mode=block
  cosmoflare waf access create ZONE_ID --ip=192.168.0.0/24 --mode=whitelist --note="Office network"`,
	RunE: runWAFAccessCreate,
}

var wafAccessDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete an IP access rule",
	Long: `Delete a zone-level IP access rule.

Examples:
  cosmoflare waf access delete ZONE_ID RULE_ID
  cosmoflare waf access delete ZONE_ID RULE_ID --force`,
	RunE: runWAFAccessDelete,
}

func init() {
	rootCmd.AddCommand(wafCmd)

	wafCmd.AddCommand(wafPackagesCmd)
	wafCmd.AddCommand(wafRulesCmd)
	wafCmd.AddCommand(wafRuleCmd)
	wafCmd.AddCommand(wafAccessCmd)

	wafAccessCmd.AddCommand(wafAccessListCmd)
	wafAccessCmd.AddCommand(wafAccessCreateCmd)
	wafAccessCmd.AddCommand(wafAccessDeleteCmd)

	wafRuleCmd.Flags().StringVar(&wafRuleMode, "mode", "", "WAF rule mode (block, simulate, disable, default, challenge)")

	wafAccessCreateCmd.Flags().StringVar(&wafAccessIP, "ip", "", "IP address or CIDR range")
	wafAccessCreateCmd.Flags().StringVar(&wafAccessMode, "mode", "", "Access rule mode (block, challenge, whitelist, js_challenge)")
	wafAccessCreateCmd.Flags().StringVar(&wafAccessNote, "note", "", "Note for the access rule")

	wafAccessDeleteCmd.Flags().BoolVar(&wafForce, "force", false, "Skip confirmation prompt")
}

func getWAFService(zoneID string) (*cosmoflare.WAFService, error) {
	return cosmoflare.NewWAFServiceFromCreds(zoneID, APIToken)
}

func runWAFPackages(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	svc, err := getWAFService(args[0])
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}
	packages, err := svc.ListPackages(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list WAF packages: %v", err))
		}
		return fmt.Errorf("failed to list WAF packages: %w", err)
	}
	if JSONOutput {
		return printJSON(packages)
	}
	if len(packages) == 0 {
		printInfo("No WAF packages found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSENSITIVITY\tACTION MODE")
	for _, p := range packages {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Sensitivity, p.ActionMode)
	}
	w.Flush()
	printInfo("Total: %d package(s)", len(packages))
	return nil
}

func runWAFRules(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and package ID are required")
	}
	svc, err := getWAFService(args[0])
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}
	rules, err := svc.ListRules(context.Background(), args[1])
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list WAF rules: %v", err))
		}
		return fmt.Errorf("failed to list WAF rules: %w", err)
	}
	if JSONOutput {
		return printJSON(rules)
	}
	if len(rules) == 0 {
		printInfo("No WAF rules found in package '%s'", args[1])
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tMODE\tGROUP\tDESCRIPTION")
	for _, r := range rules {
		desc := r.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Mode, r.Group.Name, desc)
	}
	w.Flush()
	printInfo("Total: %d rule(s)", len(rules))
	return nil
}

func runWAFRule(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("zone ID, package ID, and rule ID are required")
	}
	zoneID, packageID, ruleID := args[0], args[1], args[2]

	svc, err := getWAFService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}

	if wafRuleMode != "" {
		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would update WAF rule", map[string]string{"rule_id": ruleID, "mode": wafRuleMode})
			}
			printInfo("DRY RUN: Would update WAF rule '%s' to mode '%s'", ruleID, wafRuleMode)
			return nil
		}
		rule, err := svc.UpdateRule(context.Background(), packageID, ruleID, wafRuleMode)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to update WAF rule: %v", err))
			}
			return fmt.Errorf("failed to update WAF rule: %w", err)
		}
		if JSONOutput {
			return printSuccessJSON("WAF rule updated", rule)
		}
		printSuccess("WAF rule '%s' mode set to '%s'", ruleID, rule.Mode)
		return nil
	}

	rule, err := svc.GetRule(context.Background(), packageID, ruleID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get WAF rule: %v", err))
		}
		return fmt.Errorf("failed to get WAF rule: %w", err)
	}
	if JSONOutput {
		return printJSON(rule)
	}
	fmt.Printf("ID:           %s\n", rule.ID)
	fmt.Printf("Description:  %s\n", rule.Description)
	fmt.Printf("Mode:         %s\n", rule.Mode)
	fmt.Printf("Default Mode: %s\n", rule.DefaultMode)
	fmt.Printf("Group:        %s (%s)\n", rule.Group.Name, rule.Group.ID)
	fmt.Printf("Allowed:      %s\n", strings.Join(rule.AllowedModes, ", "))
	return nil
}

func runWAFAccessList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	svc, err := getWAFService(args[0])
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}
	rules, err := svc.ListAccessRules(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list access rules: %v", err))
		}
		return fmt.Errorf("failed to list access rules: %w", err)
	}
	if JSONOutput {
		return printJSON(rules)
	}
	if len(rules) == 0 {
		printInfo("No IP access rules found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tMODE\tTARGET\tVALUE\tNOTES")
	for _, r := range rules {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Mode, r.Configuration.Target, r.Configuration.Value, r.Notes)
	}
	w.Flush()
	printInfo("Total: %d rule(s)", len(rules))
	return nil
}

func runWAFAccessCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	if wafAccessIP == "" {
		return fmt.Errorf("--ip is required")
	}
	if wafAccessMode == "" {
		return fmt.Errorf("--mode is required (block, challenge, whitelist, js_challenge)")
	}
	svc, err := getWAFService(args[0])
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}
	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create access rule", map[string]string{"ip": wafAccessIP, "mode": wafAccessMode})
		}
		printInfo("DRY RUN: Would create access rule for %s mode=%s", wafAccessIP, wafAccessMode)
		return nil
	}
	rule, err := svc.CreateAccessRule(context.Background(), "ip", wafAccessIP, wafAccessMode, wafAccessNote)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create access rule: %v", err))
		}
		return fmt.Errorf("failed to create access rule: %w", err)
	}
	if JSONOutput {
		return printSuccessJSON("Access rule created", rule)
	}
	printSuccess("Access rule created: %s %s %s", rule.Mode, rule.Configuration.Target, rule.Configuration.Value)
	return nil
}

func runWAFAccessDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	if !wafForce && !DryRun {
		fmt.Printf("Are you sure you want to delete access rule '%s'? [y/N]: ", ruleID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Access rule deletion cancelled")
			return nil
		}
	}
	svc, err := getWAFService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create WAF service: %w", err)
	}
	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete access rule", map[string]string{"rule_id": ruleID})
		}
		printInfo("DRY RUN: Would delete access rule '%s'", ruleID)
		return nil
	}
	if err := svc.DeleteAccessRule(context.Background(), ruleID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete access rule: %v", err))
		}
		return fmt.Errorf("failed to delete access rule: %w", err)
	}
	if JSONOutput {
		return printSuccessJSON("Access rule deleted", map[string]string{"rule_id": ruleID})
	}
	printSuccess("Access rule '%s' deleted", ruleID)
	return nil
}
