package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// newRedirectService is a package-level factory var so tests can override the
// service constructor without live Cloudflare credentials.
var newRedirectService = func() (*cosmoflare.RedirectService, error) {
	return cosmoflare.NewRedirectServiceFromCreds(AccountID, APIToken)
}

var redirectsCmd = &cobra.Command{
	Use:   "redirects",
	Short: "Manage Cloudflare Redirect Rules (modern Rulesets API)",
	Long: `Manage modern Cloudflare Redirect Rules (Rulesets API,
phase http_request_dynamic_redirect).

Commands:
  list    List redirect rules for a zone
  create  Create a redirect rule in a zone
  delete  Delete a redirect rule from a zone

Examples:
  cosmoflare redirects list ZONE_ID
  cosmoflare redirects create ZONE_ID --when 'http.request.uri.path eq "/old"' --dest 'https://example.com/new'
  cosmoflare redirects create ZONE_ID --when '...' --dest '...' --status 302 --preserve-query
  cosmoflare redirects delete ZONE_ID RULE_ID`,
}

var (
	redirectWhen          string
	redirectDest          string
	redirectStatus        int
	redirectPreserveQuery bool
)

var redirectsListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List redirect rules for a zone",
	Long: `List all modern Redirect Rules configured for a zone.

If the zone has no redirect-phase ruleset, an empty list is returned.

Examples:
  cosmoflare redirects list abc123
  cosmoflare redirects list abc123 --json`,
	RunE: runRedirectsList,
}

var redirectsCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a redirect rule",
	Long: `Create a modern Redirect Rule in a zone.

The --when expression matches incoming requests (Cloudflare ruleset
expression syntax). The --dest destination is the target URL, which may
use $1..$n captures from the expression.

Examples:
  cosmoflare redirects create abc123 --when 'http.request.uri.path eq "/old"' --dest 'https://example.com/new'
  cosmoflare redirects create abc123 --when '...' --dest '...' --status 302
  cosmoflare redirects create abc123 --when '...' --dest '...' --preserve-query --json`,
	RunE: runRedirectsCreate,
}

var redirectsDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [rule-id]",
	Short: "Delete a redirect rule",
	Long: `Delete a modern Redirect Rule from a zone.

Examples:
  cosmoflare redirects delete abc123 rule456
  cosmoflare redirects delete abc123 rule456 --json`,
	RunE: runRedirectsDelete,
}

func init() {
	rootCmd.AddCommand(redirectsCmd)

	redirectsCmd.AddCommand(redirectsListCmd)
	redirectsCmd.AddCommand(redirectsCreateCmd)
	redirectsCmd.AddCommand(redirectsDeleteCmd)

	redirectsCreateCmd.Flags().StringVar(&redirectWhen, "when", "", "Match expression for incoming requests (Cloudflare ruleset expression)")
	redirectsCreateCmd.Flags().StringVar(&redirectDest, "dest", "", "Destination URL (may use $1..$n captures)")
	redirectsCreateCmd.Flags().IntVar(&redirectStatus, "status", 301, "HTTP redirect status code (301, 302, 307, 308)")
	redirectsCreateCmd.Flags().BoolVar(&redirectPreserveQuery, "preserve-query", false, "Preserve the original query string on redirect")
}

func runRedirectsList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := newRedirectService()
	if err != nil {
		return fmt.Errorf("failed to create redirect service: %w", err)
	}

	rules, err := svc.List(cmd.Context(), zoneID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list redirect rules: %v", err))
		}
		return fmt.Errorf("failed to list redirect rules: %w", err)
	}

	if JSONOutput {
		return printJSON(rules)
	}

	if len(rules) == 0 {
		printInfo("No redirect rules found for zone '%s'", zoneID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tENABLED\tPRESERVE-QUERY\tWHEN\tDESTINATION")
	for _, r := range rules {
		enabled := "no"
		if r.Enabled {
			enabled = "yes"
		}
		preserve := "no"
		if r.PreserveQuery {
			preserve = "yes"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.StatusCode,
			enabled,
			preserve,
			r.When,
			r.Destination,
		)
	}
	w.Flush()

	printInfo("Total: %d redirect rule(s)", len(rules))
	return nil
}

func runRedirectsCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	if redirectWhen == "" {
		return fmt.Errorf("--when expression is required")
	}
	if redirectDest == "" {
		return fmt.Errorf("--dest destination is required")
	}

	input := cosmoflare.RedirectRuleInput{
		ZoneID:        zoneID,
		When:          redirectWhen,
		Destination:   redirectDest,
		StatusCode:    redirectStatus,
		PreserveQuery: redirectPreserveQuery,
	}

	svc, err := newRedirectService()
	if err != nil {
		return fmt.Errorf("failed to create redirect service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create redirect rule", input)
		}
		printInfo("DRY RUN: Would create redirect rule in zone '%s' (when: %s, dest: %s, status: %d)",
			zoneID, redirectWhen, redirectDest, redirectStatus)
		return nil
	}

	rule, err := svc.Create(cmd.Context(), input)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create redirect rule: %v", err))
		}
		return fmt.Errorf("failed to create redirect rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Redirect rule created successfully", rule)
	}

	printSuccess("Redirect rule created (ID: %s)", rule.ID)
	printInfo("When: %s", rule.When)
	printInfo("Destination: %s", rule.Destination)
	printInfo("Status: %d", rule.StatusCode)
	return nil
}

func runRedirectsDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and rule ID are required")
	}
	zoneID := args[0]
	ruleID := args[1]

	svc, err := newRedirectService()
	if err != nil {
		return fmt.Errorf("failed to create redirect service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete redirect rule", map[string]string{"zone_id": zoneID, "rule_id": ruleID})
		}
		printInfo("DRY RUN: Would delete redirect rule '%s' from zone '%s'", ruleID, zoneID)
		return nil
	}

	if err := svc.Delete(cmd.Context(), zoneID, ruleID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete redirect rule: %v", err))
		}
		return fmt.Errorf("failed to delete redirect rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Redirect rule deleted successfully", map[string]string{"zone_id": zoneID, "rule_id": ruleID})
	}
	printSuccess("Redirect rule '%s' deleted from zone '%s'", ruleID, zoneID)
	return nil
}
