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

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Manage Cloudflare Email Routing",
	Long: `Email Routing management for Cloudflare zones.

Commands:
  rules       List email routing rules
  rule get    Get a specific rule
  rule create Create a routing rule
  rule delete Delete a routing rule
  catchall    Get or update the catch-all rule
  settings    View email routing settings
  enable      Enable email routing
  disable     Disable email routing

Email routing is zone-scoped, so a zone ID is required for all operations.

Examples:
  r2go2 email rules ZONE_ID --json
  r2go2 email rule get ZONE_ID RULE_ID
  r2go2 email rule create ZONE_ID --match="user@example.com" --forward="dest@example.com"
  r2go2 email rule delete ZONE_ID RULE_ID --force
  r2go2 email catchall ZONE_ID
  r2go2 email catchall update ZONE_ID --forward="catchall@example.com"
  r2go2 email settings ZONE_ID
  r2go2 email enable ZONE_ID
  r2go2 email disable ZONE_ID`,
}

var (
	emailMatchAddr   string
	emailForwardAddr string
	emailRuleName    string
	emailForce       bool
)

// emailRulesCmd lists all email routing rules.
var emailRulesCmd = &cobra.Command{
	Use:   "rules [zone-id]",
	Short: "List email routing rules",
	Long: `List all email routing rules in a zone.

Examples:
  r2go2 email rules ZONE_ID
  r2go2 email rules ZONE_ID --json`,
	RunE: runEmailRules,
}

// emailRuleCmd is a parent command for rule sub-operations.
var emailRuleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Email routing rule operations",
	Long: `Operations on individual email routing rules.

Commands:
  get     Get a specific rule by ID
  create  Create a new routing rule
  delete  Delete a routing rule

Examples:
  r2go2 email rule get ZONE_ID RULE_ID
  r2go2 email rule create ZONE_ID --match="user@example.com" --forward="dest@example.com"
  r2go2 email rule delete ZONE_ID RULE_ID --force`,
}

var emailRuleGetCmd = &cobra.Command{
	Use:   "get [zone-id] [rule-id]",
	Short: "Get a specific email routing rule",
	Long: `Get details of a single email routing rule by its ID.

Examples:
  r2go2 email rule get ZONE_ID RULE_ID
  r2go2 email rule get ZONE_ID RULE_ID --json`,
	RunE: runEmailRuleGet,
}

var emailRuleCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create an email routing rule",
	Long: `Create a new email routing rule that forwards emails matching an address to a destination.

Examples:
  r2go2 email rule create ZONE_ID --match="user@example.com" --forward="dest@example.com"
  r2go2 email rule create ZONE_ID --match="info@example.com" --forward="team@company.com" --name="Info forwarding"
  r2go2 email rule create ZONE_ID --match="sales@example.com" --forward="crm@company.com" --name="Sales" --json`,
	RunE: runEmailRuleCreate,
}

var emailRuleDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete an email routing rule",
	Long: `Delete an email routing rule from a zone.

WARNING: This action is irreversible.

Examples:
  r2go2 email rule delete ZONE_ID RULE_ID
  r2go2 email rule delete ZONE_ID RULE_ID --force`,
	RunE: runEmailRuleDelete,
}

// emailCatchallCmd gets the catch-all rule.
var emailCatchallCmd = &cobra.Command{
	Use:   "catchall [zone-id]",
	Short: "Get the catch-all email routing rule",
	Long: `Get the current catch-all email routing rule for a zone.

The catch-all rule defines what happens to emails that don't match any specific routing rule.

Examples:
  r2go2 email catchall ZONE_ID
  r2go2 email catchall ZONE_ID --json`,
	RunE: runEmailCatchall,
}

// emailCatchallUpdateCmd updates the catch-all rule.
var emailCatchallUpdateCmd = &cobra.Command{
	Use:   "update [zone-id]",
	Short: "Update the catch-all email routing rule",
	Long: `Update the catch-all email routing rule for a zone.

Examples:
  r2go2 email catchall update ZONE_ID --forward="catchall@example.com"
  r2go2 email catchall update ZONE_ID --forward="admin@example.com" --json`,
	RunE: runEmailCatchallUpdate,
}

// emailSettingsCmd gets email routing settings.
var emailSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "Get email routing settings",
	Long: `Get email routing settings for a zone.

Examples:
  r2go2 email settings ZONE_ID
  r2go2 email settings ZONE_ID --json`,
	RunE: runEmailSettings,
}

// emailEnableCmd enables email routing.
var emailEnableCmd = &cobra.Command{
	Use:   "enable [zone-id]",
	Short: "Enable email routing for a zone",
	Long: `Enable email routing for a zone. This adds and locks the necessary MX and SPF DNS records.

Examples:
  r2go2 email enable ZONE_ID
  r2go2 email enable ZONE_ID --json`,
	RunE: runEmailEnable,
}

// emailDisableCmd disables email routing.
var emailDisableCmd = &cobra.Command{
	Use:   "disable [zone-id]",
	Short: "Disable email routing for a zone",
	Long: `Disable email routing for a zone. This removes the MX records previously required for email routing.

Examples:
  r2go2 email disable ZONE_ID
  r2go2 email disable ZONE_ID --json`,
	RunE: runEmailDisable,
}

func init() {
	rootCmd.AddCommand(emailCmd)

	emailCmd.AddCommand(emailRulesCmd)
	emailCmd.AddCommand(emailRuleCmd)
	emailCmd.AddCommand(emailCatchallCmd)
	emailCmd.AddCommand(emailSettingsCmd)
	emailCmd.AddCommand(emailEnableCmd)
	emailCmd.AddCommand(emailDisableCmd)

	emailRuleCmd.AddCommand(emailRuleGetCmd)
	emailRuleCmd.AddCommand(emailRuleCreateCmd)
	emailRuleCmd.AddCommand(emailRuleDeleteCmd)

	emailCatchallCmd.AddCommand(emailCatchallUpdateCmd)

	// Create flags
	emailRuleCreateCmd.Flags().StringVar(&emailMatchAddr, "match", "", "Email address to match (e.g., user@example.com)")
	emailRuleCreateCmd.Flags().StringVar(&emailForwardAddr, "forward", "", "Destination email to forward to (e.g., dest@example.com)")
	emailRuleCreateCmd.Flags().StringVar(&emailRuleName, "name", "", "Optional name for the rule")
	_ = emailRuleCreateCmd.MarkFlagRequired("match")
	_ = emailRuleCreateCmd.MarkFlagRequired("forward")

	// Delete flags
	emailRuleDeleteCmd.Flags().BoolVar(&emailForce, "force", false, "Skip confirmation prompt")

	// Catchall update flags
	emailCatchallUpdateCmd.Flags().StringVar(&emailForwardAddr, "forward", "", "Destination email for catch-all forwarding")
	_ = emailCatchallUpdateCmd.MarkFlagRequired("forward")
}

func getEmailService(zoneID string) (*r2go2.EmailService, error) {
	return r2go2.NewEmailServiceFromCreds(zoneID, APIToken)
}

func runEmailRules(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	rules, err := svc.ListRules(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list email routing rules: %v", err))
		}
		return fmt.Errorf("failed to list email routing rules: %w", err)
	}

	if JSONOutput {
		return printJSON(rules)
	}

	if len(rules) == 0 {
		printInfo("No email routing rules found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMATCH\tFORWARD TO\tENABLED\tPRIORITY")
	for _, r := range rules {
		matchStr := ""
		if len(r.Matchers) > 0 {
			matchStr = r.Matchers[0].Value
		}
		forwardStr := ""
		if len(r.Actions) > 0 && len(r.Actions[0].Value) > 0 {
			forwardStr = strings.Join(r.Actions[0].Value, ", ")
		}
		enabledStr := "no"
		if r.Enabled {
			enabledStr = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			r.ID,
			r.Name,
			matchStr,
			forwardStr,
			enabledStr,
			r.Priority,
		)
	}
	w.Flush()

	printInfo("Total: %d rule(s)", len(rules))
	return nil
}

func runEmailRuleGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	rule, err := svc.GetRule(context.Background(), ruleID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get email routing rule: %v", err))
		}
		return fmt.Errorf("failed to get email routing rule: %w", err)
	}

	if JSONOutput {
		return printJSON(rule)
	}

	fmt.Printf("ID:         %s\n", rule.ID)
	fmt.Printf("Name:       %s\n", rule.Name)
	fmt.Printf("Enabled:    %v\n", rule.Enabled)
	fmt.Printf("Priority:   %d\n", rule.Priority)
	if len(rule.Matchers) > 0 {
		fmt.Printf("Matchers:\n")
		for _, m := range rule.Matchers {
			fmt.Printf("  - Type: %s  Field: %s  Value: %s\n", m.Type, m.Field, m.Value)
		}
	}
	if len(rule.Actions) > 0 {
		fmt.Printf("Actions:\n")
		for _, a := range rule.Actions {
			fmt.Printf("  - Type: %s  Value: %s\n", a.Type, strings.Join(a.Value, ", "))
		}
	}
	return nil
}

func runEmailRuleCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create email routing rule", map[string]interface{}{
				"zone_id": zoneID,
				"match":   emailMatchAddr,
				"forward": emailForwardAddr,
				"name":    emailRuleName,
			})
		}
		printInfo("DRY RUN: Would create email routing rule: %s -> %s", emailMatchAddr, emailForwardAddr)
		return nil
	}

	rule, err := svc.CreateRule(context.Background(), emailMatchAddr, emailForwardAddr, emailRuleName, true)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create email routing rule: %v", err))
		}
		return fmt.Errorf("failed to create email routing rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Email routing rule created successfully", rule)
	}

	printSuccess("Email routing rule created successfully!")
	printInfo("ID: %s", rule.ID)
	printInfo("Name: %s", rule.Name)
	printInfo("Match: %s", emailMatchAddr)
	printInfo("Forward: %s", emailForwardAddr)
	printInfo("Enabled: %v", rule.Enabled)
	return nil
}

func runEmailRuleDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	if !emailForce && !DryRun {
		fmt.Printf("Are you sure you want to delete email routing rule '%s'? [y/N]: ", ruleID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Email routing rule deletion cancelled")
			return nil
		}
	}

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete email routing rule", map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			})
		}
		printInfo("DRY RUN: Would delete email routing rule '%s' from zone '%s'", ruleID, zoneID)
		return nil
	}

	if _, err := svc.DeleteRule(context.Background(), ruleID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete email routing rule: %v", err))
		}
		return fmt.Errorf("failed to delete email routing rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Email routing rule deleted successfully", map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		})
	}
	printSuccess("Email routing rule '%s' deleted successfully!", ruleID)
	return nil
}

func runEmailCatchall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	catchall, err := svc.GetCatchAll(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get catch-all rule: %v", err))
		}
		return fmt.Errorf("failed to get catch-all rule: %w", err)
	}

	if JSONOutput {
		return printJSON(catchall)
	}

	fmt.Printf("ID:       %s\n", catchall.ID)
	fmt.Printf("Name:     %s\n", catchall.Name)
	fmt.Printf("Enabled:  %v\n", catchall.Enabled)
	if len(catchall.Matchers) > 0 {
		fmt.Printf("Matchers:\n")
		for _, m := range catchall.Matchers {
			fmt.Printf("  - Type: %s\n", m.Type)
		}
	}
	if len(catchall.Actions) > 0 {
		fmt.Printf("Actions:\n")
		for _, a := range catchall.Actions {
			fmt.Printf("  - Type: %s  Value: %s\n", a.Type, strings.Join(a.Value, ", "))
		}
	}
	return nil
}

func runEmailCatchallUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update catch-all rule", map[string]interface{}{
				"zone_id": zoneID,
				"forward": emailForwardAddr,
			})
		}
		printInfo("DRY RUN: Would update catch-all rule to forward to %s", emailForwardAddr)
		return nil
	}

	catchall, err := svc.UpdateCatchAll(context.Background(), emailForwardAddr, true)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update catch-all rule: %v", err))
		}
		return fmt.Errorf("failed to update catch-all rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Catch-all rule updated successfully", catchall)
	}

	printSuccess("Catch-all rule updated successfully!")
	printInfo("Forward to: %s", emailForwardAddr)
	printInfo("Enabled: %v", catchall.Enabled)
	return nil
}

func runEmailSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get email routing settings: %v", err))
		}
		return fmt.Errorf("failed to get email routing settings: %w", err)
	}

	if JSONOutput {
		return printJSON(settings)
	}

	fmt.Printf("ID:       %s\n", settings.ID)
	fmt.Printf("Name:     %s\n", settings.Name)
	fmt.Printf("Enabled:  %v\n", settings.Enabled)
	fmt.Printf("Status:   %s\n", settings.Status)
	if settings.Created != nil {
		fmt.Printf("Created:  %s\n", settings.Created.Format("2006-01-02 15:04:05"))
	}
	if settings.Modified != nil {
		fmt.Printf("Modified: %s\n", settings.Modified.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runEmailEnable(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would enable email routing", map[string]string{"zone_id": zoneID})
		}
		printInfo("DRY RUN: Would enable email routing for zone '%s'", zoneID)
		return nil
	}

	settings, err := svc.Enable(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to enable email routing: %v", err))
		}
		return fmt.Errorf("failed to enable email routing: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Email routing enabled successfully", settings)
	}

	printSuccess("Email routing enabled for zone '%s'!", zoneID)
	printInfo("Status: %s", settings.Status)
	return nil
}

func runEmailDisable(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create email service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would disable email routing", map[string]string{"zone_id": zoneID})
		}
		printInfo("DRY RUN: Would disable email routing for zone '%s'", zoneID)
		return nil
	}

	settings, err := svc.Disable(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to disable email routing: %v", err))
		}
		return fmt.Errorf("failed to disable email routing: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Email routing disabled successfully", settings)
	}

	printSuccess("Email routing disabled for zone '%s'!", zoneID)
	printInfo("Status: %s", settings.Status)
	return nil
}
