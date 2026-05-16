package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

var firewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Manage Cloudflare Firewall Rules",
	Long: `Firewall rule management for Cloudflare WAF.

Create and manage firewall rules using Cloudflare filter expressions.
Rules can block, challenge, allow, or log traffic matching specific conditions.

Filter expressions use Cloudflare's expression language:
  (ip.src eq 1.2.3.4)
  (http.request.uri.path contains "/admin")
  (ip.geoip.country eq "CN")
  (cf.threat_score gt 50)

This command is designed for both human operators and AI agents:
- --json provides structured output for automated pipelines
- Filter expressions can be built programmatically
- Error messages include expression syntax hints

Commands:
  list      List all firewall rules
  get       Get rule details
  create    Create a new firewall rule
  update    Update an existing rule
  delete    Delete a firewall rule

Examples:
  cosmoflare firewall list ZONE_ID
  cosmoflare firewall create ZONE_ID --expression='(ip.src eq 1.2.3.4)' --action=block --description="Block bad IP"
  cosmoflare firewall create ZONE_ID --expression='(http.request.uri.path contains "/wp-admin")' --action=challenge
  cosmoflare firewall list ZONE_ID --json | jq '.[] | select(.action=="block")'
  cosmoflare firewall delete ZONE_ID RULE_ID --force`,
}

var (
	fwExpression  string
	fwAction      string
	fwDescription string
	fwPriority    int
	fwPaused      bool
	fwForce       bool
)

var firewallListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List all firewall rules",
	Long: `List all firewall rules in a zone.

Examples:
  cosmoflare firewall list ZONE_ID
  cosmoflare firewall list ZONE_ID --json
  cosmoflare firewall list ZONE_ID --json | jq '.[] | select(.action=="block")'`,
	RunE: runFirewallList,
}

var firewallGetCmd = &cobra.Command{
	Use:   "get [zone-id] [rule-id]",
	Short: "Get rule details",
	Long: `Get details of a single firewall rule by its ID.

Examples:
  cosmoflare firewall get ZONE_ID RULE_ID
  cosmoflare firewall get ZONE_ID RULE_ID --json`,
	RunE: runFirewallGet,
}

var firewallCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a new firewall rule",
	Long: `Create a new firewall rule with a filter expression and action.

Valid actions: block, challenge, js_challenge, allow, log, bypass

Filter expression examples:
  (ip.src eq 1.2.3.4)                          - Match specific IP
  (ip.src in {1.2.3.0/24})                     - Match IP range
  (ip.geoip.country eq "CN")                   - Match country
  (http.request.uri.path contains "/admin")     - Match URL path
  (cf.threat_score gt 50)                       - Match threat score
  (http.host eq "api.example.com")              - Match hostname

Examples:
  cosmoflare firewall create ZONE_ID --expression='(ip.src eq 1.2.3.4)' --action=block --description="Block bad IP"
  cosmoflare firewall create ZONE_ID --expression='(http.request.uri.path contains "/wp-admin")' --action=challenge
  cosmoflare firewall create ZONE_ID --expression='(cf.threat_score gt 50)' --action=js_challenge --description="Challenge high threat"
  cosmoflare firewall create ZONE_ID --expression='(ip.geoip.country eq "CN")' --action=block --json`,
	RunE: runFirewallCreate,
}

var firewallUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [rule-id]",
	Short: "Update an existing rule",
	Long: `Update an existing firewall rule. Both expression and action are required
since the Cloudflare API replaces the entire rule on update.

Examples:
  cosmoflare firewall update ZONE_ID RULE_ID --expression='(ip.src eq 5.6.7.8)' --action=block --description="Updated IP"
  cosmoflare firewall update ZONE_ID RULE_ID --expression='(cf.threat_score gt 30)' --action=challenge --json`,
	RunE: runFirewallUpdate,
}

var firewallDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete a firewall rule",
	Long: `Delete a firewall rule from a zone.

WARNING: This action is irreversible.

Examples:
  cosmoflare firewall delete ZONE_ID RULE_ID
  cosmoflare firewall delete ZONE_ID RULE_ID --force`,
	RunE: runFirewallDelete,
}

func init() {
	rootCmd.AddCommand(firewallCmd)

	firewallCmd.AddCommand(firewallListCmd)
	firewallCmd.AddCommand(firewallGetCmd)
	firewallCmd.AddCommand(firewallCreateCmd)
	firewallCmd.AddCommand(firewallUpdateCmd)
	firewallCmd.AddCommand(firewallDeleteCmd)

	// Create flags
	firewallCreateCmd.Flags().StringVar(&fwExpression, "expression", "", "Cloudflare filter expression (e.g., '(ip.src eq 1.2.3.4)')")
	firewallCreateCmd.Flags().StringVar(&fwAction, "action", "", "Rule action: block, challenge, js_challenge, allow, log, bypass")
	firewallCreateCmd.Flags().StringVar(&fwDescription, "description", "", "Human-readable rule description")
	firewallCreateCmd.Flags().IntVar(&fwPriority, "priority", 0, "Rule priority (lower = higher priority)")
	firewallCreateCmd.Flags().BoolVar(&fwPaused, "paused", false, "Create the rule in a paused state")
	_ = firewallCreateCmd.MarkFlagRequired("expression")
	_ = firewallCreateCmd.MarkFlagRequired("action")

	// Update flags
	firewallUpdateCmd.Flags().StringVar(&fwExpression, "expression", "", "Cloudflare filter expression")
	firewallUpdateCmd.Flags().StringVar(&fwAction, "action", "", "Rule action: block, challenge, js_challenge, allow, log, bypass")
	firewallUpdateCmd.Flags().StringVar(&fwDescription, "description", "", "Human-readable rule description")
	firewallUpdateCmd.Flags().IntVar(&fwPriority, "priority", 0, "Rule priority")
	firewallUpdateCmd.Flags().BoolVar(&fwPaused, "paused", false, "Whether the rule is paused")
	_ = firewallUpdateCmd.MarkFlagRequired("expression")
	_ = firewallUpdateCmd.MarkFlagRequired("action")

	// Delete flags
	firewallDeleteCmd.Flags().BoolVar(&fwForce, "force", false, "Skip confirmation prompt")
}

func getFirewallService(zoneID string) (*r2go2.FirewallService, error) {
	return r2go2.NewFirewallServiceFromCreds(zoneID, APIToken)
}

func runFirewallList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getFirewallService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create firewall service: %w", err)
	}

	rules, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list firewall rules: %v", err))
		}
		return fmt.Errorf("failed to list firewall rules: %w", err)
	}

	if JSONOutput {
		return printJSON(rules)
	}

	if len(rules) == 0 {
		printInfo("No firewall rules found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tACTION\tDESCRIPTION\tEXPRESSION\tPAUSED")
	for _, r := range rules {
		pausedStr := "no"
		if r.Paused {
			pausedStr = "yes"
		}
		// Truncate long expressions for table display
		expr := r.Filter.Expression
		if len(expr) > 50 {
			expr = expr[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.Action,
			r.Description,
			expr,
			pausedStr,
		)
	}
	w.Flush()

	printInfo("Total: %d rule(s)", len(rules))
	return nil
}

func runFirewallGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getFirewallService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create firewall service: %w", err)
	}

	rule, err := svc.Get(context.Background(), ruleID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get firewall rule: %v", err))
		}
		return fmt.Errorf("failed to get firewall rule: %w", err)
	}

	if JSONOutput {
		return printJSON(rule)
	}

	fmt.Printf("ID:          %s\n", rule.ID)
	fmt.Printf("Description: %s\n", rule.Description)
	fmt.Printf("Action:      %s\n", rule.Action)
	fmt.Printf("Priority:    %d\n", rule.Priority)
	fmt.Printf("Paused:      %v\n", rule.Paused)
	fmt.Printf("Filter ID:   %s\n", rule.Filter.ID)
	fmt.Printf("Expression:  %s\n", rule.Filter.Expression)
	if rule.Filter.Description != "" {
		fmt.Printf("Filter Desc: %s\n", rule.Filter.Description)
	}
	fmt.Printf("Created:     %s\n", rule.CreatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified:    %s\n", rule.ModifiedOn.Format("2006-01-02 15:04:05"))
	return nil
}

func runFirewallCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getFirewallService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create firewall service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create firewall rule", map[string]interface{}{
				"zone_id":     zoneID,
				"expression":  fwExpression,
				"action":      fwAction,
				"description": fwDescription,
			})
		}
		printInfo("DRY RUN: Would create firewall rule: action=%s expression=%s", fwAction, fwExpression)
		return nil
	}

	rules, err := svc.Create(context.Background(), fwExpression, fwAction, fwDescription)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create firewall rule: %v", err))
		}
		return fmt.Errorf("failed to create firewall rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Firewall rule created successfully", rules)
	}

	printSuccess("Firewall rule created successfully!")
	for _, r := range rules {
		printInfo("ID: %s", r.ID)
		printInfo("Action: %s", r.Action)
		printInfo("Expression: %s", r.Filter.Expression)
		if r.Description != "" {
			printInfo("Description: %s", r.Description)
		}
	}
	return nil
}

func runFirewallUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getFirewallService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create firewall service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update firewall rule", map[string]interface{}{
				"zone_id":     zoneID,
				"rule_id":     ruleID,
				"expression":  fwExpression,
				"action":      fwAction,
				"description": fwDescription,
			})
		}
		printInfo("DRY RUN: Would update firewall rule '%s' in zone '%s'", ruleID, zoneID)
		return nil
	}

	rule, err := svc.Update(context.Background(), ruleID, fwExpression, fwAction, fwDescription)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update firewall rule: %v", err))
		}
		return fmt.Errorf("failed to update firewall rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Firewall rule updated successfully", rule)
	}

	printSuccess("Firewall rule '%s' updated successfully!", ruleID)
	printInfo("Action: %s  Expression: %s", rule.Action, rule.Filter.Expression)
	return nil
}

func runFirewallDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	if !fwForce && !DryRun {
		fmt.Printf("Are you sure you want to delete firewall rule '%s'? [y/N]: ", ruleID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Firewall rule deletion cancelled")
			return nil
		}
	}

	svc, err := getFirewallService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create firewall service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete firewall rule", map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			})
		}
		printInfo("DRY RUN: Would delete firewall rule '%s' from zone '%s'", ruleID, zoneID)
		return nil
	}

	if err := svc.Delete(context.Background(), ruleID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete firewall rule: %v", err))
		}
		return fmt.Errorf("failed to delete firewall rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Firewall rule deleted successfully", map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		})
	}
	printSuccess("Firewall rule '%s' deleted successfully!", ruleID)
	return nil
}
