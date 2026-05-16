package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

var pageruleCmd = &cobra.Command{
	Use:   "pagerule",
	Short: "Manage Cloudflare Page Rules",
	Long: `Page Rule management for Cloudflare zones.

Commands:
  create    Create a page rule
  list      List page rules
  get       Get a page rule
  update    Update a page rule
  delete    Delete a page rule

Page rules are zone-scoped, so a zone ID is required for all operations.

Examples:
  r2go2 pagerule create ZONE_ID --target="example.com/*" --action=forwarding_url --action-value="https://new.example.com/$1"
  r2go2 pagerule list ZONE_ID --json
  r2go2 pagerule get ZONE_ID RULE_ID
  r2go2 pagerule update ZONE_ID RULE_ID --status=disabled
  r2go2 pagerule delete ZONE_ID RULE_ID --force`,
}

var (
	pageruleTarget      string
	pageruleAction      string
	pageruleActionValue string
	pagerulePriority    int
	pageruleStatus      string
	pageruleForce       bool
)

var pageruleCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a page rule",
	Long: `Create a new page rule in a Cloudflare zone.

The target is a URL pattern that supports wildcards (*).
The action determines what happens when the pattern matches.

Common actions:
  forwarding_url           301/302 redirect (requires --action-value as JSON: {"status_code":301,"url":"https://..."})
  always_use_https         Force HTTPS (no value needed)
  cache_level              Set cache level (bypass, basic, simplified, aggressive, cache_everything)
  browser_cache_ttl        Browser cache TTL in seconds
  security_level           Security level (off, essentially_off, low, medium, high, under_attack)
  ssl                      SSL mode (off, flexible, full, strict)
  disable_apps             Disable Cloudflare Apps
  disable_performance      Disable performance features
  disable_security         Disable security features
  rocket_loader            Rocket Loader (on, off)
  minify                   Minify settings (JSON: {"html":"on","css":"on","js":"on"})
  email_obfuscation        Email obfuscation (on, off)

Examples:
  r2go2 pagerule create ZONE_ID --target="example.com/*" --action=forwarding_url --action-value='{"status_code":301,"url":"https://new.example.com/$1"}'
  r2go2 pagerule create ZONE_ID --target="example.com/api/*" --action=cache_level --action-value=bypass
  r2go2 pagerule create ZONE_ID --target="example.com/*" --action=always_use_https
  r2go2 pagerule create ZONE_ID --target="example.com/*" --action=ssl --action-value=full --priority=1 --status=active`,
	RunE: runPageRuleCreate,
}

var pageruleListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List page rules",
	Long: `List all page rules in a zone.

Examples:
  r2go2 pagerule list ZONE_ID
  r2go2 pagerule list ZONE_ID --json`,
	RunE: runPageRuleList,
}

var pageruleGetCmd = &cobra.Command{
	Use:   "get [zone-id] [rule-id]",
	Short: "Get a page rule",
	Long: `Get details of a single page rule by its ID.

Examples:
  r2go2 pagerule get ZONE_ID RULE_ID
  r2go2 pagerule get ZONE_ID RULE_ID --json`,
	RunE: runPageRuleGet,
}

var pageruleUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [rule-id]",
	Short: "Update a page rule",
	Long: `Update an existing page rule.

Provide only the fields you want to change.

Examples:
  r2go2 pagerule update ZONE_ID RULE_ID --status=disabled
  r2go2 pagerule update ZONE_ID RULE_ID --priority=2
  r2go2 pagerule update ZONE_ID RULE_ID --target="example.com/new/*" --action=cache_level --action-value=aggressive
  r2go2 pagerule update ZONE_ID RULE_ID --status=active --priority=1 --json`,
	RunE: runPageRuleUpdate,
}

var pageruleDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete a page rule",
	Long: `Delete a page rule from a zone.

WARNING: This action is irreversible.

Examples:
  r2go2 pagerule delete ZONE_ID RULE_ID
  r2go2 pagerule delete ZONE_ID RULE_ID --force`,
	RunE: runPageRuleDelete,
}

func init() {
	rootCmd.AddCommand(pageruleCmd)

	pageruleCmd.AddCommand(pageruleCreateCmd)
	pageruleCmd.AddCommand(pageruleListCmd)
	pageruleCmd.AddCommand(pageruleGetCmd)
	pageruleCmd.AddCommand(pageruleUpdateCmd)
	pageruleCmd.AddCommand(pageruleDeleteCmd)

	// Create flags
	pageruleCreateCmd.Flags().StringVar(&pageruleTarget, "target", "", "URL pattern to match (e.g., example.com/*)")
	pageruleCreateCmd.Flags().StringVar(&pageruleAction, "action", "", "Action ID (e.g., forwarding_url, always_use_https, cache_level)")
	pageruleCreateCmd.Flags().StringVar(&pageruleActionValue, "action-value", "", "Action value (string, or JSON for complex actions)")
	pageruleCreateCmd.Flags().IntVar(&pagerulePriority, "priority", 1, "Rule priority (1 = highest)")
	pageruleCreateCmd.Flags().StringVar(&pageruleStatus, "status", "active", "Rule status (active, disabled)")
	_ = pageruleCreateCmd.MarkFlagRequired("target")
	_ = pageruleCreateCmd.MarkFlagRequired("action")

	// Update flags
	pageruleUpdateCmd.Flags().StringVar(&pageruleTarget, "target", "", "URL pattern to match (e.g., example.com/*)")
	pageruleUpdateCmd.Flags().StringVar(&pageruleAction, "action", "", "Action ID (e.g., forwarding_url, always_use_https)")
	pageruleUpdateCmd.Flags().StringVar(&pageruleActionValue, "action-value", "", "Action value")
	pageruleUpdateCmd.Flags().IntVar(&pagerulePriority, "priority", 0, "Rule priority (1 = highest)")
	pageruleUpdateCmd.Flags().StringVar(&pageruleStatus, "status", "", "Rule status (active, disabled)")

	// Delete flags
	pageruleDeleteCmd.Flags().BoolVar(&pageruleForce, "force", false, "Skip confirmation prompt")
}

func getPageRuleService(zoneID string) (*r2go2.PageRuleService, error) {
	return r2go2.NewPageRuleServiceFromCreds(zoneID, APIToken)
}

// parseActionValue attempts to parse the action value as JSON for complex actions,
// otherwise returns the raw string value. Some actions (like always_use_https) need no value.
func parseActionValue(actionID, value string) interface{} {
	if value == "" {
		return nil
	}

	// For forwarding_url, the API expects a map with status_code and url.
	// Try to parse JSON for complex values.
	if strings.HasPrefix(strings.TrimSpace(value), "{") {
		// Return raw string — encoding/json will handle it on the wire.
		// The cloudflare-go library accepts interface{} and marshals it.
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}

	return value
}

func runPageRuleCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create page rule service: %w", err)
	}

	actionValue := parseActionValue(pageruleAction, pageruleActionValue)
	actions := []r2go2.PageRuleAction{
		{ID: pageruleAction, Value: actionValue},
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create page rule", map[string]interface{}{
				"zone_id":  zoneID,
				"target":   pageruleTarget,
				"action":   pageruleAction,
				"priority": pagerulePriority,
				"status":   pageruleStatus,
			})
		}
		printInfo("DRY RUN: Would create page rule for target '%s' with action '%s' in zone %s", pageruleTarget, pageruleAction, zoneID)
		return nil
	}

	rule, err := svc.Create(context.Background(), pageruleTarget, actions, pagerulePriority, pageruleStatus)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create page rule: %v", err))
		}
		return fmt.Errorf("failed to create page rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Page rule created successfully", rule)
	}

	printSuccess("Page rule created successfully!")
	printInfo("ID: %s", rule.ID)
	printInfo("Status: %s", rule.Status)
	printInfo("Priority: %d", rule.Priority)
	if len(rule.Targets) > 0 {
		printInfo("Target: %s", rule.Targets[0].Constraint.Value)
	}
	if len(rule.Actions) > 0 {
		printInfo("Action: %s", rule.Actions[0].ID)
	}
	return nil
}

func runPageRuleList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create page rule service: %w", err)
	}

	rules, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list page rules: %v", err))
		}
		return fmt.Errorf("failed to list page rules: %w", err)
	}

	if JSONOutput {
		return printJSON(rules)
	}

	if len(rules) == 0 {
		printInfo("No page rules found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTARGET\tACTION\tPRIORITY\tSTATUS")
	for _, r := range rules {
		target := ""
		if len(r.Targets) > 0 {
			target = r.Targets[0].Constraint.Value
		}
		action := ""
		if len(r.Actions) > 0 {
			action = r.Actions[0].ID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			r.ID,
			target,
			action,
			r.Priority,
			r.Status,
		)
	}
	w.Flush()

	printInfo("Total: %d rule(s)", len(rules))
	return nil
}

func runPageRuleGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create page rule service: %w", err)
	}

	rule, err := svc.Get(context.Background(), ruleID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get page rule: %v", err))
		}
		return fmt.Errorf("failed to get page rule: %w", err)
	}

	if JSONOutput {
		return printJSON(rule)
	}

	fmt.Printf("ID:         %s\n", rule.ID)
	fmt.Printf("Status:     %s\n", rule.Status)
	fmt.Printf("Priority:   %d\n", rule.Priority)
	if len(rule.Targets) > 0 {
		fmt.Printf("Target:     %s\n", rule.Targets[0].Constraint.Value)
	}
	fmt.Println("Actions:")
	for _, a := range rule.Actions {
		if a.Value != nil {
			fmt.Printf("  - %s: %v\n", a.ID, a.Value)
		} else {
			fmt.Printf("  - %s\n", a.ID)
		}
	}
	fmt.Printf("Created:    %s\n", rule.CreatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified:   %s\n", rule.ModifiedOn.Format("2006-01-02 15:04:05"))
	return nil
}

func runPageRuleUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	// Check that at least one update flag was provided
	hasUpdate := cmd.Flags().Changed("target") || cmd.Flags().Changed("action") ||
		cmd.Flags().Changed("priority") || cmd.Flags().Changed("status")
	if !hasUpdate {
		return fmt.Errorf("at least one update flag is required (--target, --action, --priority, --status)")
	}

	var actions []r2go2.PageRuleAction
	if cmd.Flags().Changed("action") {
		actionValue := parseActionValue(pageruleAction, pageruleActionValue)
		actions = []r2go2.PageRuleAction{
			{ID: pageruleAction, Value: actionValue},
		}
	}

	target := ""
	if cmd.Flags().Changed("target") {
		target = pageruleTarget
	}

	priority := 0
	if cmd.Flags().Changed("priority") {
		priority = pagerulePriority
	}

	status := ""
	if cmd.Flags().Changed("status") {
		status = pageruleStatus
	}

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create page rule service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update page rule", map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			})
		}
		printInfo("DRY RUN: Would update page rule '%s' in zone '%s'", ruleID, zoneID)
		return nil
	}

	err = svc.Update(context.Background(), ruleID, target, actions, priority, status)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update page rule: %v", err))
		}
		return fmt.Errorf("failed to update page rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Page rule updated successfully", map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		})
	}

	printSuccess("Page rule '%s' updated successfully!", ruleID)
	return nil
}

func runPageRuleDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	if !pageruleForce && !DryRun {
		fmt.Printf("Are you sure you want to delete page rule '%s'? [y/N]: ", ruleID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Page rule deletion cancelled")
			return nil
		}
	}

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create page rule service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete page rule", map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			})
		}
		printInfo("DRY RUN: Would delete page rule '%s' from zone '%s'", ruleID, zoneID)
		return nil
	}

	if err := svc.Delete(context.Background(), ruleID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete page rule: %v", err))
		}
		return fmt.Errorf("failed to delete page rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Page rule deleted successfully", map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		})
	}
	printSuccess("Page rule '%s' deleted successfully!", ruleID)
	return nil
}
