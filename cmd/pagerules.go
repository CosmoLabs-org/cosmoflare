package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var pagerulesCmd = &cobra.Command{
	Use:   "pagerules",
	Short: "Manage Cloudflare Page Rules",
	Long: `Page Rule management for URL-based settings and redirects.

Page Rules let you control Cloudflare features on a URL pattern basis.
Common uses: redirects, caching rules, SSL enforcement, and more.

This command is designed for both human operators and AI agents:
- --json provides machine-readable output for automated workflows
- All mutations support --dry-run for safe previewing
- Error messages include fix suggestions

Commands:
  list      List all page rules in a zone
  get       Get page rule details
  create    Create a new page rule
  update    Update an existing page rule
  delete    Delete a page rule

Examples:
  cosmoflare pagerules list ZONE_ID
  cosmoflare pagerules get ZONE_ID RULE_ID
  cosmoflare pagerules create ZONE_ID --url="*.example.com/old/*" --action=forwarding_url --action-value="https://example.com/new/$1" --status=active
  cosmoflare pagerules delete ZONE_ID RULE_ID --force
  cosmoflare pagerules list ZONE_ID --json | jq '.[].targets[0].constraint.value'`,
}

var (
	pageruleURL         string
	pageruleAction      string
	pageruleActionValue string
	pageruleStatus      string
	pagerulePriority    int
	pageruleForce       bool
)

var pagerulesListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List all page rules in a zone",
	Long: `List all page rules configured for a Cloudflare zone.

Examples:
  cosmoflare pagerules list ZONE_ID
  cosmoflare pagerules list ZONE_ID --json
  cosmoflare pagerules list ZONE_ID --json | jq '.[].targets[0].constraint.value'`,
	RunE: runPageRulesList,
}

var pagerulesGetCmd = &cobra.Command{
	Use:   "get [zone-id] [rule-id]",
	Short: "Get page rule details",
	Long: `Get full details of a single page rule by its ID.

Examples:
  cosmoflare pagerules get ZONE_ID RULE_ID
  cosmoflare pagerules get ZONE_ID RULE_ID --json`,
	RunE: runPageRulesGet,
}

var pagerulesCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a new page rule",
	Long: `Create a new page rule in a Cloudflare zone.

The --url flag specifies the URL pattern to match (e.g., "*.example.com/old/*").
The --action flag specifies what action to take (e.g., forwarding_url, always_https, cache_level).
The --action-value flag provides the value for the action (if needed).

Common actions:
  forwarding_url      Redirect to another URL (value: destination URL)
  always_https        Force HTTPS (no value needed)
  cache_level         Set cache level (value: bypass, basic, simplified, aggressive, cache_everything)
  ssl                 Set SSL mode (value: off, flexible, full, strict)
  browser_cache_ttl   Browser cache TTL (value: seconds as integer)

Examples:
  cosmoflare pagerules create ZONE_ID --url="*.example.com/old/*" --action=forwarding_url --action-value="https://example.com/new/$1"
  cosmoflare pagerules create ZONE_ID --url="example.com/secure/*" --action=always_https --status=active
  cosmoflare pagerules create ZONE_ID --url="example.com/static/*" --action=cache_level --action-value=cache_everything --priority=2`,
	RunE: runPageRulesCreate,
}

var pagerulesUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [rule-id]",
	Short: "Update an existing page rule",
	Long: `Update (replace) an existing page rule with new settings.

All fields are required when updating — this performs a full replacement.

Examples:
  cosmoflare pagerules update ZONE_ID RULE_ID --url="*.example.com/new/*" --action=forwarding_url --action-value="https://example.com/latest/$1"
  cosmoflare pagerules update ZONE_ID RULE_ID --url="example.com/*" --action=always_https --status=disabled
  cosmoflare pagerules update ZONE_ID RULE_ID --url="example.com/api/*" --action=cache_level --action-value=bypass --priority=1`,
	RunE: runPageRulesUpdate,
}

var pagerulesDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete a page rule",
	Long: `Delete a page rule from a zone.

WARNING: This action is irreversible.

Examples:
  cosmoflare pagerules delete ZONE_ID RULE_ID
  cosmoflare pagerules delete ZONE_ID RULE_ID --force`,
	RunE: runPageRulesDelete,
}

func init() {
	rootCmd.AddCommand(pagerulesCmd)

	pagerulesCmd.AddCommand(pagerulesListCmd)
	pagerulesCmd.AddCommand(pagerulesGetCmd)
	pagerulesCmd.AddCommand(pagerulesCreateCmd)
	pagerulesCmd.AddCommand(pagerulesUpdateCmd)
	pagerulesCmd.AddCommand(pagerulesDeleteCmd)

	// Create flags
	pagerulesCreateCmd.Flags().StringVar(&pageruleURL, "url", "", "URL match pattern (e.g., *.example.com/path/*)")
	pagerulesCreateCmd.Flags().StringVar(&pageruleAction, "action", "", "Action ID (forwarding_url, always_https, cache_level, ssl, etc.)")
	pagerulesCreateCmd.Flags().StringVar(&pageruleActionValue, "action-value", "", "Action value (URL for forwarding, cache level, etc.)")
	pagerulesCreateCmd.Flags().StringVar(&pageruleStatus, "status", "active", "Rule status: active or disabled")
	pagerulesCreateCmd.Flags().IntVar(&pagerulePriority, "priority", 1, "Rule priority (1 = highest, evaluated first)")
	_ = pagerulesCreateCmd.MarkFlagRequired("url")
	_ = pagerulesCreateCmd.MarkFlagRequired("action")

	// Update flags
	pagerulesUpdateCmd.Flags().StringVar(&pageruleURL, "url", "", "URL match pattern (e.g., *.example.com/path/*)")
	pagerulesUpdateCmd.Flags().StringVar(&pageruleAction, "action", "", "Action ID (forwarding_url, always_https, cache_level, ssl, etc.)")
	pagerulesUpdateCmd.Flags().StringVar(&pageruleActionValue, "action-value", "", "Action value (URL for forwarding, cache level, etc.)")
	pagerulesUpdateCmd.Flags().StringVar(&pageruleStatus, "status", "active", "Rule status: active or disabled")
	pagerulesUpdateCmd.Flags().IntVar(&pagerulePriority, "priority", 1, "Rule priority (1 = highest, evaluated first)")
	_ = pagerulesUpdateCmd.MarkFlagRequired("url")
	_ = pagerulesUpdateCmd.MarkFlagRequired("action")

	// Delete flags
	pagerulesDeleteCmd.Flags().BoolVar(&pageruleForce, "force", false, "Skip confirmation prompt")
}

func getPageRuleService(zoneID string) (*cosmoflare.PageRuleService, error) {
	return cosmoflare.NewPageRuleServiceFromCreds(zoneID, APIToken)
}

func runPageRulesList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return outErr("failed to create page rule service", err)
	}

	rules, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list page rules", err)
	}

	return outResult(rules, func() {
		if len(rules) == 0 {
			printInfo("No page rules found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tURL PATTERN\tACTION")
		for _, r := range rules {
			urlPattern := ""
			if len(r.Targets) > 0 {
				urlPattern = r.Targets[0].Constraint.Value
			}
			actionStr := ""
			if len(r.Actions) > 0 {
				actionStr = r.Actions[0].ID
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				r.ID,
				r.Status,
				r.Priority,
				urlPattern,
				actionStr,
			)
		}
		w.Flush()

		printInfo("Total: %d page rule(s)", len(rules))
	})
}

func runPageRulesGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return outErr("failed to create page rule service", err)
	}

	rule, err := svc.Get(context.Background(), ruleID)
	if err != nil {
		return outErr("failed to get page rule", err)
	}

	return outResult(rule, func() {
		fmt.Printf("ID:         %s\n", rule.ID)
		fmt.Printf("Status:     %s\n", rule.Status)
		fmt.Printf("Priority:   %d\n", rule.Priority)
		fmt.Printf("Created:    %s\n", rule.CreatedOn.Format("2006-01-02 15:04:05"))
		fmt.Printf("Modified:   %s\n", rule.ModifiedOn.Format("2006-01-02 15:04:05"))
		fmt.Println("Targets:")
		for _, t := range rule.Targets {
			fmt.Printf("  %s %s %s\n", t.Target, t.Constraint.Operator, t.Constraint.Value)
		}
		fmt.Println("Actions:")
		for _, a := range rule.Actions {
			if a.Value != nil {
				fmt.Printf("  %s = %v\n", a.ID, a.Value)
			} else {
				fmt.Printf("  %s\n", a.ID)
			}
		}
	})
}

func runPageRulesCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	targets := []cosmoflare.PageRuleTarget{
		{
			Target: "url",
			Constraint: cosmoflare.PageRuleConstraint{
				Operator: "matches",
				Value:    pageruleURL,
			},
		},
	}

	var actionValue interface{}
	if pageruleActionValue != "" {
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(pageruleActionValue), &jsonVal); err == nil {
			actionValue = jsonVal
		} else {
			actionValue = pageruleActionValue
		}
	}
	actions := []cosmoflare.PageRuleAction{
		{
			ID:    pageruleAction,
			Value: actionValue,
		},
	}

	if DryRun {
		return outPayload("DRY RUN: Would create page rule", func() any {
			return map[string]interface{}{
				"zone_id":  zoneID,
				"url":      pageruleURL,
				"action":   pageruleAction,
				"status":   pageruleStatus,
				"priority": pagerulePriority,
			}
		}, func() {
			printInfo("DRY RUN: Would create page rule matching '%s' with action '%s' in zone %s", pageruleURL, pageruleAction, zoneID)
		})
	}

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return outErr("failed to create page rule service", err)
	}

	rule, err := svc.Create(context.Background(), targets, actions, pageruleStatus, pagerulePriority)
	if err != nil {
		return outErr("failed to create page rule", err)
	}

	return outPayload("Page rule created successfully", func() any {
		return rule
	}, func() {
		printSuccess("Page rule created successfully!")
		printInfo("ID: %s", rule.ID)
		printInfo("Status: %s", rule.Status)
		printInfo("Priority: %d", rule.Priority)
		printInfo("URL: %s", pageruleURL)
		printInfo("Action: %s", pageruleAction)
	})
}

func runPageRulesUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	targets := []cosmoflare.PageRuleTarget{
		{
			Target: "url",
			Constraint: cosmoflare.PageRuleConstraint{
				Operator: "matches",
				Value:    pageruleURL,
			},
		},
	}

	var actionValue interface{}
	if pageruleActionValue != "" {
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(pageruleActionValue), &jsonVal); err == nil {
			actionValue = jsonVal
		} else {
			actionValue = pageruleActionValue
		}
	}
	actions := []cosmoflare.PageRuleAction{
		{
			ID:    pageruleAction,
			Value: actionValue,
		},
	}

	if DryRun {
		return outPayload("DRY RUN: Would update page rule", func() any {
			return map[string]interface{}{
				"zone_id":  zoneID,
				"rule_id":  ruleID,
				"url":      pageruleURL,
				"action":   pageruleAction,
				"status":   pageruleStatus,
				"priority": pagerulePriority,
			}
		}, func() {
			printInfo("DRY RUN: Would update page rule '%s' in zone '%s'", ruleID, zoneID)
		})
	}

	svc, err := getPageRuleService(zoneID)
	if err != nil {
		return outErr("failed to create page rule service", err)
	}

	err = svc.Update(context.Background(), ruleID, targets, actions, pageruleStatus, pagerulePriority)
	if err != nil {
		return outErr("failed to update page rule", err)
	}

	return outPayload("Page rule updated successfully", func() any {
		return map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		}
	}, func() {
		printSuccess("Page rule '%s' updated successfully!", ruleID)
	})
}

func runPageRulesDelete(cmd *cobra.Command, args []string) error {
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
		return outErr("failed to create page rule service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete page rule", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			}
		}, func() {
			printInfo("DRY RUN: Would delete page rule '%s' from zone '%s'", ruleID, zoneID)
		})
	}

	if err := svc.Delete(context.Background(), ruleID); err != nil {
		return outErr("failed to delete page rule", err)
	}

	return outPayload("Page rule deleted successfully", func() any {
		return map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		}
	}, func() {
		printSuccess("Page rule '%s' deleted successfully!", ruleID)
	})
}
