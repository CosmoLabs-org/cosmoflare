package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var corsCmd = &cobra.Command{
	Use:   "cors",
	Short: "Manage CORS response headers via Cloudflare Transform Rules",
	Long: `CORS response header management for Cloudflare zones via Transform Rules.

Cloudflare Transform Rules (response header modification) are used to inject
CORS headers — no Worker required. Rules are zone-scoped.

Note: Transform Rules require a Cloudflare Pro plan or higher.
Note: For multiple specific origins, only the first origin is used as a static
      header value. For dynamic origin reflection, deploy a Worker.

Commands:
  settings   Show all active CORS rules for the zone
  set        Create or replace a named CORS rule
  remove     Remove a CORS rule by name

Examples:
  cosmoflare cors settings ZONE_ID
  cosmoflare cors set ZONE_ID --origins "*" --methods "GET,POST,OPTIONS"
  cosmoflare cors set ZONE_ID --origins "https://app.example.com" --credentials --max-age 86400
  cosmoflare cors remove ZONE_ID
  cosmoflare cors remove ZONE_ID --rule-name my-cors-rule`,
}

var (
	corsOrigins     string
	corsMethods     string
	corsHeaders     string
	corsMaxAge      int
	corsCredentials bool
	corsRuleName    string
	corsExpression  string
)

var corsSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "Show all active CORS rules for the zone",
	Long: `Show all CORS rules currently active in the zone's response header transform ruleset.

Displays rule name, allowed origins, methods, credentials flag, and filter expression.

Examples:
  cosmoflare cors settings ZONE_ID
  cosmoflare cors settings ZONE_ID --json`,
	RunE: runCORSSettings,
}

var corsSetCmd = &cobra.Command{
	Use:   "set [zone-id]",
	Short: "Create or replace a named CORS rule",
	Long: `Create or replace a CORS rule in the zone's response header transform ruleset.

If a rule with --rule-name already exists it is replaced in-place. Other rules
in the ruleset are left untouched.

The --credentials flag cannot be combined with --origins "*" (CORS spec forbids it).

Examples:
  cosmoflare cors set ZONE_ID --origins "*" --methods "GET,POST,OPTIONS"
  cosmoflare cors set ZONE_ID --origins "https://app.example.com" --credentials
  cosmoflare cors set ZONE_ID --origins "https://app.example.com" --max-age 3600 --json
  cosmoflare cors set ZONE_ID --origins "*" --expression 'http.request.uri.path matches "^/api/"'`,
	RunE: runCORSSet,
}

var corsRemoveCmd = &cobra.Command{
	Use:   "remove [zone-id]",
	Short: "Remove a CORS rule by name",
	Long: `Remove a CORS rule from the zone's response header transform ruleset by name.

Other rules in the ruleset are preserved. Returns an error if no rule with the
given name exists.

Examples:
  cosmoflare cors remove ZONE_ID
  cosmoflare cors remove ZONE_ID --rule-name my-cors-rule
  cosmoflare cors remove ZONE_ID --json`,
	RunE: runCORSRemove,
}

func init() {
	rootCmd.AddCommand(corsCmd)

	corsCmd.AddCommand(corsSettingsCmd)
	corsCmd.AddCommand(corsSetCmd)
	corsCmd.AddCommand(corsRemoveCmd)

	// cors set flags
	corsSetCmd.Flags().StringVar(&corsOrigins, "origins", "", "Comma-separated allowed origins, e.g. \"*\" or \"https://app.example.com\" (required)")
	corsSetCmd.Flags().StringVar(&corsMethods, "methods", "GET, POST, OPTIONS", "Comma-separated HTTP methods")
	corsSetCmd.Flags().StringVar(&corsHeaders, "headers", "Content-Type, Authorization", "Comma-separated request header names")
	corsSetCmd.Flags().IntVar(&corsMaxAge, "max-age", 86400, "Preflight cache max-age in seconds (Access-Control-Max-Age)")
	corsSetCmd.Flags().BoolVar(&corsCredentials, "credentials", false, "Allow credentials (Access-Control-Allow-Credentials: true)")
	corsSetCmd.Flags().StringVar(&corsRuleName, "rule-name", cosmoflare.CORSDefaultRuleName, "Name/identifier for the rule (used to update or remove it later)")
	corsSetCmd.Flags().StringVar(&corsExpression, "expression", "true", "Wirefilter filter expression (default: all requests)")
	corsSetCmd.MarkFlagRequired("origins") //nolint:errcheck

	// cors remove flags
	corsRemoveCmd.Flags().StringVar(&corsRuleName, "rule-name", cosmoflare.CORSDefaultRuleName, "Name of the rule to remove")
}

func getCORSService(zoneID string) (*cosmoflare.CORSService, error) {
	return cosmoflare.NewCORSServiceFromCreds(zoneID, APIToken)
}

func runCORSSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getCORSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create CORS service: %w", err)
	}

	rules, err := svc.GetCORSRules(context.Background())
	if err != nil {
		return outErr("failed to get CORS rules", err)
	}

	if JSONOutput {
		return printJSON(rules)
	}

	if len(rules) == 0 {
		printInfo("No CORS rules configured for zone '%s'.", zoneID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tORIGINS\tMETHODS\tCREDENTIALS\tEXPRESSION")
	for _, r := range rules {
		origins := strings.Join(r.AllowOrigins, ", ")
		methods := strings.Join(r.AllowMethods, ", ")
		creds := "false"
		if r.AllowCredentials {
			creds = "true"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.Name, origins, methods, creds, r.Expression)
	}
	w.Flush()
	printInfo("Total: %d CORS rule(s) on zone '%s'", len(rules), zoneID)
	return nil
}

func runCORSSet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	// Parse origins
	origins := parseCORSList(corsOrigins)
	if len(origins) == 0 {
		return fmt.Errorf("--origins cannot be empty")
	}

	// Parse methods and headers
	methods := parseCORSList(corsMethods)
	headers := parseCORSList(corsHeaders)

	// Warn on multiple specific origins (static header limitation)
	if len(origins) > 1 {
		hasWildcard := false
		for _, o := range origins {
			if o == "*" {
				hasWildcard = true
			}
		}
		if !hasWildcard {
			printWarning("Multiple origins specified. Cloudflare Transform Rules use a static header value — only one origin can be set per rule. For dynamic origin reflection, use a Worker.")
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would set CORS rule", map[string]interface{}{
				"zone_id":  zoneID,
				"rule":     corsRuleName,
				"origins":  origins,
				"methods":  methods,
				"headers":  headers,
				"max_age":  corsMaxAge,
				"creds":    corsCredentials,
				"expr":     corsExpression,
			})
		}
		printInfo("DRY RUN: Would set CORS rule '%s' on zone '%s'", corsRuleName, zoneID)
		return nil
	}

	svc, err := getCORSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create CORS service: %w", err)
	}

	opts := []cosmoflare.CORSOption{
		cosmoflare.WithCORSName(corsRuleName),
		cosmoflare.WithCORSOrigins(origins...),
		cosmoflare.WithCORSMethods(methods...),
		cosmoflare.WithCORSHeaders(headers...),
		cosmoflare.WithCORSMaxAge(corsMaxAge),
		cosmoflare.WithCORSCredentials(corsCredentials),
		cosmoflare.WithCORSExpression(corsExpression),
	}

	rule, err := svc.SetCORSHeaders(context.Background(), opts...)
	if err != nil {
		return outErr("failed to set CORS rule", err)
	}

	if JSONOutput {
		return printJSON(rule)
	}

	printSuccess("CORS rule %q applied to zone %s.", rule.Name, zoneID)
	fmt.Printf("  Origins:      %s\n", strings.Join(rule.AllowOrigins, ", "))
	fmt.Printf("  Methods:      %s\n", strings.Join(rule.AllowMethods, ", "))
	fmt.Printf("  Headers:      %s\n", strings.Join(rule.AllowHeaders, ", "))
	fmt.Printf("  Max-Age:      %d\n", rule.MaxAge)
	fmt.Printf("  Credentials:  %v\n", rule.AllowCredentials)
	fmt.Printf("  Expression:   %s\n", rule.Expression)
	return nil
}

func runCORSRemove(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would remove CORS rule", map[string]string{
				"zone_id": zoneID,
				"rule":    corsRuleName,
			})
		}
		printInfo("DRY RUN: Would remove CORS rule '%s' from zone '%s'", corsRuleName, zoneID)
		return nil
	}

	svc, err := getCORSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create CORS service: %w", err)
	}

	err = svc.RemoveCORSRule(context.Background(), corsRuleName)
	if err != nil {
		if errors.Is(err, cosmoflare.ErrCORSRuleNotFound) {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("CORS rule %q not found on zone %s. Use \"cosmoflare cors settings %s\" to list active CORS rules.", corsRuleName, zoneID, zoneID))
			}
			return fmt.Errorf("CORS rule %q not found on zone %s.\n       Use \"cosmoflare cors settings %s\" to list active CORS rules", corsRuleName, zoneID, zoneID)
		}
		return outErr("failed to remove CORS rule", err)
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("CORS rule %q removed from zone %s.", corsRuleName, zoneID), nil)
	}
	printSuccess("CORS rule %q removed from zone %s.", corsRuleName, zoneID)
	return nil
}

// parseCORSList splits a comma-separated string into trimmed, non-empty items.
func parseCORSList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, v)
		}
	}
	return result
}
