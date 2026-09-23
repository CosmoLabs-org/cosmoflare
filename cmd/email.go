package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Manage Cloudflare Email Routing",
	Long: `Email Routing configuration for your domain.

Route incoming emails to specific destinations based on rules.
Manage routing rules and verified destination addresses.

This command is designed for both human operators and AI agents:
- --json provides structured output for automated email configuration
- Rules support pattern matching for flexible routing
- Destination management handles verification workflow

Commands:
  rules list        List all email routing rules
  rules get         Get rule details
  rules create      Create a routing rule
  rules update      Update a routing rule
  rules delete      Delete a routing rule
  destinations list List verified destination addresses
  destinations add  Add a destination address
  destinations get  Get destination details
  destinations delete Remove a destination
  catchall          Get the catch-all rule
  catchall update   Update the catch-all rule
  settings          View email routing settings
  enable            Enable email routing
  disable           Disable email routing

Examples:
  cosmoflare email rules list ZONE_ID
  cosmoflare email rules create ZONE_ID --match-to="support@example.com" --forward-to="team@company.com" --name="Support routing"
  cosmoflare email destinations list ZONE_ID
  cosmoflare email destinations add ZONE_ID --email="team@company.com"
  cosmoflare email rules list ZONE_ID --json | jq '.[].matchers[0].value'`,
}

// --- Rules subcommand group ---

var emailRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Email routing rule operations",
	Long: `Manage email routing rules for a zone.

Commands:
  list    List all routing rules
  get     Get a specific rule by ID
  create  Create a new routing rule
  update  Update an existing routing rule
  delete  Delete a routing rule

Examples:
  cosmoflare email rules list ZONE_ID
  cosmoflare email rules get ZONE_ID RULE_ID
  cosmoflare email rules create ZONE_ID --match-to="user@example.com" --forward-to="dest@example.com" --name="Forward"
  cosmoflare email rules create ZONE_ID --match-all --drop --name="Drop all"
  cosmoflare email rules update ZONE_ID RULE_ID --match-to="new@example.com" --forward-to="dest@example.com" --name="Updated"
  cosmoflare email rules delete ZONE_ID RULE_ID --force`,
}

var emailRulesListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List email routing rules",
	Long: `List all email routing rules in a zone.

Examples:
  cosmoflare email rules list ZONE_ID
  cosmoflare email rules list ZONE_ID --json`,
	RunE: runEmailRulesList,
}

var emailRulesGetCmd = &cobra.Command{
	Use:   "get [zone-id] [rule-id]",
	Short: "Get a specific email routing rule",
	Long: `Get details of a single email routing rule by its ID.

Examples:
  cosmoflare email rules get ZONE_ID RULE_ID
  cosmoflare email rules get ZONE_ID RULE_ID --json`,
	RunE: runEmailRulesGet,
}

var emailRulesCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create an email routing rule",
	Long: `Create a new email routing rule.

Use --match-to to match a specific email address, or --match-all for all incoming mail.
Use --forward-to to forward matched emails, or --drop to discard them.

Examples:
  cosmoflare email rules create ZONE_ID --match-to="support@example.com" --forward-to="team@company.com" --name="Support"
  cosmoflare email rules create ZONE_ID --match-all --forward-to="admin@company.com" --name="Forward all"
  cosmoflare email rules create ZONE_ID --match-to="spam@example.com" --drop --name="Drop spam"
  cosmoflare email rules create ZONE_ID --match-to="info@example.com" --forward-to="info@company.com" --name="Info" --priority=10 --enabled=false`,
	RunE: runEmailRulesCreate,
}

var emailRulesUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [rule-id]",
	Short: "Update an email routing rule",
	Long: `Update an existing email routing rule.

Examples:
  cosmoflare email rules update ZONE_ID RULE_ID --match-to="new@example.com" --forward-to="dest@example.com" --name="Updated"
  cosmoflare email rules update ZONE_ID RULE_ID --enabled=false
  cosmoflare email rules update ZONE_ID RULE_ID --priority=5 --json`,
	RunE: runEmailRulesUpdate,
}

var emailRulesDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete an email routing rule",
	Long: `Delete an email routing rule from a zone.

WARNING: This action is irreversible.

Examples:
  cosmoflare email rules delete ZONE_ID RULE_ID
  cosmoflare email rules delete ZONE_ID RULE_ID --force`,
	RunE: runEmailRulesDelete,
}

// --- Destinations subcommand group ---

var emailDestCmd = &cobra.Command{
	Use:   "destinations",
	Short: "Email routing destination addresses",
	Long: `Manage verified destination addresses for email routing.

Destination addresses must be verified via email before they can receive forwarded mail.

Commands:
  list    List all destination addresses
  add     Add a new destination address
  get     Get destination details
  delete  Remove a destination address

Examples:
  cosmoflare email destinations list ZONE_ID
  cosmoflare email destinations add ZONE_ID --email="team@company.com"
  cosmoflare email destinations get ZONE_ID ADDRESS_ID
  cosmoflare email destinations delete ZONE_ID ADDRESS_ID --force`,
}

var emailDestListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List destination addresses",
	Long: `List all verified destination addresses for the account.

Examples:
  cosmoflare email destinations list ZONE_ID
  cosmoflare email destinations list ZONE_ID --json`,
	RunE: runEmailDestList,
}

var emailDestAddCmd = &cobra.Command{
	Use:   "add [zone-id]",
	Short: "Add a destination address",
	Long: `Add a new destination address for email routing.

The address will receive a verification email that must be confirmed before it can be used.

Examples:
  cosmoflare email destinations add ZONE_ID --email="team@company.com"
  cosmoflare email destinations add ZONE_ID --email="admin@company.com" --json`,
	RunE: runEmailDestAdd,
}

var emailDestGetCmd = &cobra.Command{
	Use:   "get [zone-id] [address-id]",
	Short: "Get destination address details",
	Long: `Get details of a specific destination address.

Examples:
  cosmoflare email destinations get ZONE_ID ADDRESS_ID
  cosmoflare email destinations get ZONE_ID ADDRESS_ID --json`,
	RunE: runEmailDestGet,
}

var emailDestDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [address-id]",
	Short: "Delete a destination address",
	Long: `Remove a destination address from the account.

WARNING: This action is irreversible.

Examples:
  cosmoflare email destinations delete ZONE_ID ADDRESS_ID
  cosmoflare email destinations delete ZONE_ID ADDRESS_ID --force`,
	RunE: runEmailDestDelete,
}

// --- Catch-all and settings commands ---

var emailCatchallCmd = &cobra.Command{
	Use:   "catchall [zone-id]",
	Short: "Get the catch-all email routing rule",
	Long: `Get the current catch-all email routing rule for a zone.

The catch-all rule defines what happens to emails that don't match any specific routing rule.

Examples:
  cosmoflare email catchall ZONE_ID
  cosmoflare email catchall ZONE_ID --json`,
	RunE: runEmailCatchall,
}

var emailCatchallUpdateCmd = &cobra.Command{
	Use:   "update [zone-id]",
	Short: "Update the catch-all email routing rule",
	Long: `Update the catch-all email routing rule for a zone.

Examples:
  cosmoflare email catchall update ZONE_ID --forward-to="catchall@example.com"
  cosmoflare email catchall update ZONE_ID --forward-to="admin@example.com" --json`,
	RunE: runEmailCatchallUpdate,
}

var emailSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "Get email routing settings",
	Long: `Get email routing settings for a zone.

Examples:
  cosmoflare email settings ZONE_ID
  cosmoflare email settings ZONE_ID --json`,
	RunE: runEmailSettings,
}

var emailEnableCmd = &cobra.Command{
	Use:   "enable [zone-id]",
	Short: "Enable email routing for a zone",
	Long: `Enable email routing for a zone. This adds and locks the necessary MX and SPF DNS records.

Examples:
  cosmoflare email enable ZONE_ID
  cosmoflare email enable ZONE_ID --json`,
	RunE: runEmailEnable,
}

var emailDisableCmd = &cobra.Command{
	Use:   "disable [zone-id]",
	Short: "Disable email routing for a zone",
	Long: `Disable email routing for a zone. This removes the MX records previously required for email routing.

Examples:
  cosmoflare email disable ZONE_ID
  cosmoflare email disable ZONE_ID --json`,
	RunE: runEmailDisable,
}

// --- Flags ---

var (
	emailMatchTo   string
	emailMatchAll  bool
	emailForwardTo string
	emailDrop      bool
	emailRuleName  string
	emailPriority  int
	emailEnabled   bool
	emailForce     bool
	emailAddr      string
)

func init() {
	rootCmd.AddCommand(emailCmd)

	// Rules subcommand group
	emailCmd.AddCommand(emailRulesCmd)
	emailRulesCmd.AddCommand(emailRulesListCmd)
	emailRulesCmd.AddCommand(emailRulesGetCmd)
	emailRulesCmd.AddCommand(emailRulesCreateCmd)
	emailRulesCmd.AddCommand(emailRulesUpdateCmd)
	emailRulesCmd.AddCommand(emailRulesDeleteCmd)

	// Destinations subcommand group
	emailCmd.AddCommand(emailDestCmd)
	emailDestCmd.AddCommand(emailDestListCmd)
	emailDestCmd.AddCommand(emailDestAddCmd)
	emailDestCmd.AddCommand(emailDestGetCmd)
	emailDestCmd.AddCommand(emailDestDeleteCmd)

	// Catch-all, settings, enable, disable
	emailCmd.AddCommand(emailCatchallCmd)
	emailCatchallCmd.AddCommand(emailCatchallUpdateCmd)
	emailCmd.AddCommand(emailSettingsCmd)
	emailCmd.AddCommand(emailEnableCmd)
	emailCmd.AddCommand(emailDisableCmd)

	// Rules create flags
	emailRulesCreateCmd.Flags().StringVar(&emailRuleName, "name", "", "Rule name")
	emailRulesCreateCmd.Flags().StringVar(&emailMatchTo, "match-to", "", "Email address to match (creates literal matcher on 'to' field)")
	emailRulesCreateCmd.Flags().BoolVar(&emailMatchAll, "match-all", false, "Match all incoming email (creates 'all' matcher)")
	emailRulesCreateCmd.Flags().StringVar(&emailForwardTo, "forward-to", "", "Destination email to forward to (creates 'forward' action)")
	emailRulesCreateCmd.Flags().BoolVar(&emailDrop, "drop", false, "Drop matching email (creates 'drop' action)")
	emailRulesCreateCmd.Flags().IntVar(&emailPriority, "priority", 0, "Rule priority (lower = higher priority)")
	emailRulesCreateCmd.Flags().BoolVar(&emailEnabled, "enabled", true, "Whether rule is enabled")
	_ = emailRulesCreateCmd.MarkFlagRequired("name")

	// Rules update flags
	emailRulesUpdateCmd.Flags().StringVar(&emailRuleName, "name", "", "Rule name")
	emailRulesUpdateCmd.Flags().StringVar(&emailMatchTo, "match-to", "", "Email address to match (creates literal matcher on 'to' field)")
	emailRulesUpdateCmd.Flags().BoolVar(&emailMatchAll, "match-all", false, "Match all incoming email (creates 'all' matcher)")
	emailRulesUpdateCmd.Flags().StringVar(&emailForwardTo, "forward-to", "", "Destination email to forward to (creates 'forward' action)")
	emailRulesUpdateCmd.Flags().BoolVar(&emailDrop, "drop", false, "Drop matching email (creates 'drop' action)")
	emailRulesUpdateCmd.Flags().IntVar(&emailPriority, "priority", 0, "Rule priority (lower = higher priority)")
	emailRulesUpdateCmd.Flags().BoolVar(&emailEnabled, "enabled", true, "Whether rule is enabled")

	// Rules delete flags
	emailRulesDeleteCmd.Flags().BoolVar(&emailForce, "force", false, "Skip confirmation prompt")

	// Destinations add flags
	emailDestAddCmd.Flags().StringVar(&emailAddr, "email", "", "Destination email address")
	_ = emailDestAddCmd.MarkFlagRequired("email")

	// Destinations delete flags
	emailDestDeleteCmd.Flags().BoolVar(&emailForce, "force", false, "Skip confirmation prompt")

	// Catch-all update flags
	emailCatchallUpdateCmd.Flags().StringVar(&emailForwardTo, "forward-to", "", "Destination email for catch-all forwarding")
	_ = emailCatchallUpdateCmd.MarkFlagRequired("forward-to")
}

func getEmailService(zoneID string) (*cosmoflare.EmailService, error) {
	return cosmoflare.NewEmailServiceFromCreds(zoneID, AccountID, APIToken)
}

// --- Rules handlers ---

func runEmailRulesList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	rules, err := svc.ListRules(context.Background())
	if err != nil {
		return outErr("failed to list email routing rules", err)
	}

	return outResult(rules, func() {
		if len(rules) == 0 {
			printInfo("No email routing rules found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tMATCH\tACTION\tENABLED\tPRIORITY")
		for _, r := range rules {
			matchStr := ""
			if len(r.Matchers) > 0 {
				if r.Matchers[0].Type == "all" {
					matchStr = "(all)"
				} else {
					matchStr = r.Matchers[0].Value
				}
			}
			actionStr := ""
			if len(r.Actions) > 0 {
				if r.Actions[0].Type == "drop" {
					actionStr = "drop"
				} else {
					actionStr = strings.Join(r.Actions[0].Value, ", ")
				}
			}
			enabledStr := "no"
			if r.Enabled {
				enabledStr = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
				r.ID,
				r.Name,
				matchStr,
				actionStr,
				enabledStr,
				r.Priority,
			)
		}
		w.Flush()

		printInfo("Total: %d rule(s)", len(rules))
	})
}

func runEmailRulesGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	rule, err := svc.GetRule(context.Background(), ruleID)
	if err != nil {
		return outErr("failed to get email routing rule", err)
	}

	return outResult(rule, func() {
		fmt.Printf("ID:         %s\n", rule.ID)
		fmt.Printf("Name:       %s\n", rule.Name)
		fmt.Printf("Enabled:    %v\n", rule.Enabled)
		fmt.Printf("Priority:   %d\n", rule.Priority)
		if len(rule.Matchers) > 0 {
			fmt.Printf("Matchers:\n")
			for _, m := range rule.Matchers {
				if m.Type == "all" {
					fmt.Printf("  - Type: all (match all incoming)\n")
				} else {
					fmt.Printf("  - Type: %s  Field: %s  Value: %s\n", m.Type, m.Field, m.Value)
				}
			}
		}
		if len(rule.Actions) > 0 {
			fmt.Printf("Actions:\n")
			for _, a := range rule.Actions {
				if a.Type == "drop" {
					fmt.Printf("  - Type: drop\n")
				} else {
					fmt.Printf("  - Type: %s  Value: %s\n", a.Type, strings.Join(a.Value, ", "))
				}
			}
		}
	})
}

func runEmailRulesCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	// Validate matcher flags
	if emailMatchTo == "" && !emailMatchAll {
		return fmt.Errorf("either --match-to or --match-all is required")
	}
	if emailMatchTo != "" && emailMatchAll {
		return fmt.Errorf("--match-to and --match-all are mutually exclusive")
	}

	// Validate action flags
	if emailForwardTo == "" && !emailDrop {
		return fmt.Errorf("either --forward-to or --drop is required")
	}
	if emailForwardTo != "" && emailDrop {
		return fmt.Errorf("--forward-to and --drop are mutually exclusive")
	}

	// Build matchers
	var matchers []cosmoflare.EmailRuleMatcher
	if emailMatchAll {
		matchers = []cosmoflare.EmailRuleMatcher{{Type: "all"}}
	} else {
		matchers = []cosmoflare.EmailRuleMatcher{{Type: "literal", Field: "to", Value: emailMatchTo}}
	}

	// Build actions
	var actions []cosmoflare.EmailRuleAction
	if emailDrop {
		actions = []cosmoflare.EmailRuleAction{{Type: "drop", Value: []string{}}}
	} else {
		actions = []cosmoflare.EmailRuleAction{{Type: "forward", Value: []string{emailForwardTo}}}
	}

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create email routing rule", func() any {
			return map[string]interface{}{
				"zone_id":  zoneID,
				"name":     emailRuleName,
				"matchers": matchers,
				"actions":  actions,
				"priority": emailPriority,
				"enabled":  emailEnabled,
			}
		}, func() {
			printInfo("DRY RUN: Would create email routing rule %q", emailRuleName)
		})
	}

	rule, err := svc.CreateRule(context.Background(), emailRuleName, matchers, actions, emailPriority, emailEnabled)
	if err != nil {
		return outErr("failed to create email routing rule", err)
	}

	return outPayload("Email routing rule created successfully", func() any {
		return rule
	}, func() {
		printSuccess("Email routing rule created successfully!")
		printInfo("ID: %s", rule.ID)
		printInfo("Name: %s", rule.Name)
		printInfo("Enabled: %v", rule.Enabled)
		printInfo("Priority: %d", rule.Priority)
	})
}

func runEmailRulesUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID, ruleID := args[0], args[1]

	// Validate matcher flags (at least one must be set for update)
	if emailMatchTo != "" && emailMatchAll {
		return fmt.Errorf("--match-to and --match-all are mutually exclusive")
	}
	if emailForwardTo != "" && emailDrop {
		return fmt.Errorf("--forward-to and --drop are mutually exclusive")
	}

	// Build matchers
	var matchers []cosmoflare.EmailRuleMatcher
	if emailMatchAll {
		matchers = []cosmoflare.EmailRuleMatcher{{Type: "all"}}
	} else if emailMatchTo != "" {
		matchers = []cosmoflare.EmailRuleMatcher{{Type: "literal", Field: "to", Value: emailMatchTo}}
	}

	// Build actions
	var actions []cosmoflare.EmailRuleAction
	if emailDrop {
		actions = []cosmoflare.EmailRuleAction{{Type: "drop", Value: []string{}}}
	} else if emailForwardTo != "" {
		actions = []cosmoflare.EmailRuleAction{{Type: "forward", Value: []string{emailForwardTo}}}
	}

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update email routing rule", func() any {
			return map[string]interface{}{
				"zone_id": zoneID,
				"rule_id": ruleID,
				"name":    emailRuleName,
			}
		}, func() {
			printInfo("DRY RUN: Would update email routing rule '%s'", ruleID)
		})
	}

	// Fetch existing rule to preserve unmodified fields (full-replacement API).
	existing, err := svc.GetRule(context.Background(), ruleID)
	if err != nil {
		return outErr(fmt.Sprintf("failed to fetch existing rule %s", ruleID), err)
	}

	// Merge: use flag value if explicitly set, otherwise keep existing.
	name := existing.Name
	if cmd.Flags().Changed("name") {
		name = emailRuleName
	}
	priority := existing.Priority
	if cmd.Flags().Changed("priority") {
		priority = emailPriority
	}
	enabled := existing.Enabled
	if cmd.Flags().Changed("enabled") {
		enabled = emailEnabled
	}
	if len(matchers) == 0 {
		matchers = existing.Matchers
	}
	if len(actions) == 0 {
		actions = existing.Actions
	}

	rule, err := svc.UpdateRule(context.Background(), ruleID, name, matchers, actions, priority, enabled)
	if err != nil {
		return outErr("failed to update email routing rule", err)
	}

	return outPayload("Email routing rule updated successfully", func() any {
		return rule
	}, func() {
		printSuccess("Email routing rule '%s' updated successfully!", ruleID)
		printInfo("Name: %s", rule.Name)
		printInfo("Enabled: %v", rule.Enabled)
		printInfo("Priority: %d", rule.Priority)
	})
}

func runEmailRulesDelete(cmd *cobra.Command, args []string) error {
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
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete email routing rule", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"rule_id": ruleID,
			}
		}, func() {
			printInfo("DRY RUN: Would delete email routing rule '%s' from zone '%s'", ruleID, zoneID)
		})
	}

	if err := svc.DeleteRule(context.Background(), ruleID); err != nil {
		return outErr("failed to delete email routing rule", err)
	}

	return outPayload("Email routing rule deleted successfully", func() any {
		return map[string]string{
			"zone_id": zoneID,
			"rule_id": ruleID,
		}
	}, func() {
		printSuccess("Email routing rule '%s' deleted successfully!", ruleID)
	})
}

// --- Destinations handlers ---

func runEmailDestList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	destinations, err := svc.ListDestinations(context.Background())
	if err != nil {
		return outErr("failed to list email destinations", err)
	}

	return outResult(destinations, func() {
		if len(destinations) == 0 {
			printInfo("No email destinations found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tEMAIL\tVERIFIED\tCREATED")
		for _, d := range destinations {
			verifiedStr := "pending"
			if !d.Verified.IsZero() {
				verifiedStr = d.Verified.Format("2006-01-02")
			}
			createdStr := ""
			if !d.Created.IsZero() {
				createdStr = d.Created.Format("2006-01-02 15:04:05")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				d.ID,
				d.Email,
				verifiedStr,
				createdStr,
			)
		}
		w.Flush()

		printInfo("Total: %d destination(s)", len(destinations))
	})
}

func runEmailDestAdd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would add email destination", func() any {
			return map[string]interface{}{
				"zone_id": zoneID,
				"email":   emailAddr,
			}
		}, func() {
			printInfo("DRY RUN: Would add email destination %q", emailAddr)
		})
	}

	dest, err := svc.CreateDestination(context.Background(), emailAddr)
	if err != nil {
		return outErr("failed to add email destination", err)
	}

	return outPayload("Email destination added successfully", func() any {
		return dest
	}, func() {
		printSuccess("Email destination added successfully!")
		printInfo("ID: %s", dest.ID)
		printInfo("Email: %s", dest.Email)
		printInfo("A verification email has been sent. Please confirm to activate.")
	})
}

func runEmailDestGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and address ID are required")
	}
	zoneID, addressID := args[0], args[1]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	dest, err := svc.GetDestination(context.Background(), addressID)
	if err != nil {
		return outErr("failed to get email destination", err)
	}

	return outResult(dest, func() {
		fmt.Printf("ID:       %s\n", dest.ID)
		fmt.Printf("Email:    %s\n", dest.Email)
		if !dest.Verified.IsZero() {
			fmt.Printf("Verified: %s\n", dest.Verified.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("Verified: pending\n")
		}
		if !dest.Created.IsZero() {
			fmt.Printf("Created:  %s\n", dest.Created.Format("2006-01-02 15:04:05"))
		}
		if !dest.Modified.IsZero() {
			fmt.Printf("Modified: %s\n", dest.Modified.Format("2006-01-02 15:04:05"))
		}
	})
}

func runEmailDestDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and address ID are required")
	}
	zoneID, addressID := args[0], args[1]

	if !emailForce && !DryRun {
		fmt.Printf("Are you sure you want to delete email destination '%s'? [y/N]: ", addressID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Email destination deletion cancelled")
			return nil
		}
	}

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete email destination", func() any {
			return map[string]string{
				"zone_id":    zoneID,
				"address_id": addressID,
			}
		}, func() {
			printInfo("DRY RUN: Would delete email destination '%s'", addressID)
		})
	}

	if err := svc.DeleteDestination(context.Background(), addressID); err != nil {
		return outErr("failed to delete email destination", err)
	}

	return outPayload("Email destination deleted successfully", func() any {
		return map[string]string{
			"zone_id":    zoneID,
			"address_id": addressID,
		}
	}, func() {
		printSuccess("Email destination '%s' deleted successfully!", addressID)
	})
}

// --- Catch-all handlers ---

func runEmailCatchall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	catchall, err := svc.GetCatchAll(context.Background())
	if err != nil {
		return outErr("failed to get catch-all rule", err)
	}

	return outResult(catchall, func() {
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
				if a.Type == "drop" {
					fmt.Printf("  - Type: drop\n")
				} else {
					fmt.Printf("  - Type: %s  Value: %s\n", a.Type, strings.Join(a.Value, ", "))
				}
			}
		}
	})
}

func runEmailCatchallUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update catch-all rule", func() any {
			return map[string]interface{}{
				"zone_id":    zoneID,
				"forward_to": emailForwardTo,
			}
		}, func() {
			printInfo("DRY RUN: Would update catch-all rule to forward to %s", emailForwardTo)
		})
	}

	catchall, err := svc.UpdateCatchAll(context.Background(), emailForwardTo, true)
	if err != nil {
		return outErr("failed to update catch-all rule", err)
	}

	return outPayload("Catch-all rule updated successfully", func() any {
		return catchall
	}, func() {
		printSuccess("Catch-all rule updated successfully!")
		printInfo("Forward to: %s", emailForwardTo)
		printInfo("Enabled: %v", catchall.Enabled)
	})
}

// --- Settings handlers ---

func runEmailSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		return outErr("failed to get email routing settings", err)
	}

	return outResult(settings, func() {
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
	})
}

func runEmailEnable(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would enable email routing", func() any {
			return map[string]string{"zone_id": zoneID}
		}, func() {
			printInfo("DRY RUN: Would enable email routing for zone '%s'", zoneID)
		})
	}

	settings, err := svc.Enable(context.Background())
	if err != nil {
		return outErr("failed to enable email routing", err)
	}

	return outPayload("Email routing enabled successfully", func() any {
		return settings
	}, func() {
		printSuccess("Email routing enabled for zone '%s'!", zoneID)
		printInfo("Status: %s", settings.Status)
	})
}

func runEmailDisable(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getEmailService(zoneID)
	if err != nil {
		return outErr("failed to create email service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would disable email routing", func() any {
			return map[string]string{"zone_id": zoneID}
		}, func() {
			printInfo("DRY RUN: Would disable email routing for zone '%s'", zoneID)
		})
	}

	settings, err := svc.Disable(context.Background())
	if err != nil {
		return outErr("failed to disable email routing", err)
	}

	return outPayload("Email routing disabled successfully", func() any {
		return settings
	}, func() {
		printSuccess("Email routing disabled for zone '%s'!", zoneID)
		printInfo("Status: %s", settings.Status)
	})
}
