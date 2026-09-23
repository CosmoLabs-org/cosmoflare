package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Web Analytics (RUM) is account-scoped. Permission pending FEAT-011
// dataset (not in the Qwen token-UI catalog).

var webAnalyticsCmd = &cobra.Command{
	Use:   "web-analytics",
	Short: "Manage Web Analytics (RUM) sites",
	Long: `Web Analytics site management for an account.

Web Analytics measures pageviews and visits via a lightweight JS beacon;
each site carries a token embedded in the snippet.

Commands:
  site      Manage analytics sites (create, list, delete)

Examples:
  cosmoflare web-analytics site create --host=example.com
  cosmoflare web-analytics site list --json
  cosmoflare web-analytics site delete SITE_TAG --force`,
}

var webAnalyticsSiteDeleteCmd *cobra.Command

var webAnalyticsSiteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage Web Analytics sites",
	Long: `Manage Web Analytics sites for an account.

Commands:
  create    Create a site (beacon token shown once with the snippet)
  list      List sites
  delete    Delete a site

Examples:
  cosmoflare web-analytics site create --host=example.com`,
}

var (
	waHost        string
	waZoneTag     string
	waAutoInstall bool
	waForce       bool
)

var newWebAnalyticsService = func() (*cosmoflare.WebAnalyticsService, error) {
	return cosmoflare.NewWebAnalyticsServiceFromCreds(AccountID, APIToken)
}

func runWebAnalyticsSiteCreate(cmd *cobra.Command, args []string) error {
	if waHost == "" {
		return fmt.Errorf("--host is required")
	}
	var autoInstall *bool
	if cmd != nil && cmd.Flags().Changed("auto-install") {
		autoInstall = &waAutoInstall
	}
	svc, err := newWebAnalyticsService()
	if err != nil {
		return outErr("failed to create Web Analytics service", err)
	}
	site, err := svc.Create(context.Background(), waHost, waZoneTag, autoInstall)
	if err != nil {
		return outErr("failed to create Web Analytics site", err)
	}
	return outPayload("Web Analytics site created", func() any { return site }, func() {
		printSuccess("Web Analytics site %s created", site.SiteTag)
		printInfo("Beacon token (paste the snippet into your <head>; the token is the site credential):")
		if site.Snippet != "" {
			fmt.Printf("  %s\n", site.Snippet)
		} else {
			fmt.Printf("  token: %s\n", site.SiteToken)
		}
	})
}

func runWebAnalyticsSiteList(cmd *cobra.Command, args []string) error {
	svc, err := newWebAnalyticsService()
	if err != nil {
		return outErr("failed to create Web Analytics service", err)
	}
	sites, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list Web Analytics sites", err)
	}
	return outResult(sites, func() {
		if len(sites) == 0 {
			printInfo("No Web Analytics sites")
			return
		}
		for _, s := range sites {
			printInfo("%-36s  zone=%s  ruleset=%s", s.SiteTag, s.ZoneName, s.RulesetID)
		}
		printInfo("Total: %d site(s)", len(sites))
	})
}

func runWebAnalyticsSiteDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("site tag is required")
	}
	siteTag := args[0]

	// Not registry-flagged destructive yet (registry pass follows): the
	// confirmation prompt stays live; an explicit --dry-run still
	// short-circuits before any service call.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "web-analytics site delete"), waForce)
	if dry {
		return outPayload("DRY RUN: Would delete Web Analytics site", func() any {
			return map[string]string{"site_tag": siteTag}
		}, func() {
			printInfo("DRY RUN: Would delete Web Analytics site '%s'", siteTag)
		})
	}
	if !waForce && !ux.Confirm(fmt.Sprintf("Delete Web Analytics site '%s'? Its beacon stops recording.", siteTag)) {
		printInfo("Web Analytics site deletion cancelled")
		return nil
	}

	svc, err := newWebAnalyticsService()
	if err != nil {
		return outErr("failed to create Web Analytics service", err)
	}
	if err := svc.Delete(context.Background(), siteTag); err != nil {
		return outErr(fmt.Sprintf("failed to delete Web Analytics site %q", siteTag), err)
	}
	return outPayload("Web Analytics site deleted", func() any {
		return map[string]string{"site_tag": siteTag}
	}, func() { printSuccess("Web Analytics site %s deleted", siteTag) })
}

func init() {
	rootCmd.AddCommand(webAnalyticsCmd)
	webAnalyticsCmd.AddCommand(webAnalyticsSiteCmd)

	webAnalyticsSiteCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a site (token + snippet shown once)",
		RunE:  runWebAnalyticsSiteCreate,
	}
	webAnalyticsSiteCreateCmd.Flags().StringVar(&waHost, "host", "", "Hostname to measure (required)")
	webAnalyticsSiteCreateCmd.Flags().StringVar(&waZoneTag, "zone-tag", "", "Zone tag for auto-install (optional)")
	webAnalyticsSiteCreateCmd.Flags().BoolVar(&waAutoInstall, "auto-install", false, "Enable auto-install on the zone")

	webAnalyticsSiteListCmd := &cobra.Command{
		Use:   "list",
		Short: "List Web Analytics sites",
		RunE:  runWebAnalyticsSiteList,
	}

	webAnalyticsSiteDeleteCmd = &cobra.Command{
		Use:   "delete [site-tag]",
		Short: "Delete a Web Analytics site",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runWebAnalyticsSiteDelete,
	}
	webAnalyticsSiteDeleteCmd.Flags().BoolVar(&waForce, "force", false, "Skip confirmation prompt")

	webAnalyticsSiteCmd.AddCommand(webAnalyticsSiteCreateCmd, webAnalyticsSiteListCmd, webAnalyticsSiteDeleteCmd)
}
