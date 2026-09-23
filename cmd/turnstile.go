package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Turnstile is account-scoped. Token-UI permission: account "Turnstile"
// (verified Qwen dataset).

var turnstileCmd = &cobra.Command{
	Use:   "turnstile",
	Short: "Manage Turnstile widgets",
	Long: `Turnstile widget management for an account.

Turnstile is Cloudflare's CAPTCHA alternative; widgets issue site keys and
secret keys for Challenge rendering.

Commands:
  widget    Manage Turnstile widgets (create, list, get, update, delete)

Examples:
  cosmoflare turnstile widget create --name=login-guard --hostname=example.com
  cosmoflare turnstile widget list --json
  cosmoflare turnstile widget delete SITE_KEY --force`,
}

var turnstileWidgetDeleteCmd *cobra.Command

var turnstileWidgetCmd = &cobra.Command{
	Use:   "widget",
	Short: "Manage Turnstile widgets",
	Long: `Manage Turnstile widgets for an account.

Commands:
  create    Create a Turnstile widget (secret key shown ONCE)
  list      List Turnstile widgets
  get       Get a Turnstile widget
  update    Update a Turnstile widget
  delete    Delete a Turnstile widget

Examples:
  cosmoflare turnstile widget create --name=login-guard --hostname=example.com --hostname=www.example.com`,
}

var (
	twName         string
	twHostnames    []string
	twMode         string
	twBotFightMode bool
	twOffLabel     bool
	twForce        bool
)

var newTurnstileService = func() (*cosmoflare.TurnstileService, error) {
	return cosmoflare.NewTurnstileServiceFromCreds(AccountID, APIToken)
}

func twOptions() []cosmoflare.TurnstileWidgetOption {
	var opts []cosmoflare.TurnstileWidgetOption
	if twName != "" {
		opts = append(opts, cosmoflare.WithTurnstileName(twName))
	}
	if len(twHostnames) > 0 {
		opts = append(opts, cosmoflare.WithTurnstileHostnames(twHostnames))
	}
	if twMode != "" {
		opts = append(opts, cosmoflare.WithTurnstileMode(twMode))
	}
	opts = append(opts, cosmoflare.WithTurnstileBotFightMode(twBotFightMode))
	if twOffLabel {
		opts = append(opts, cosmoflare.WithTurnstileOffLabel(twOffLabel))
	}
	return opts
}

func runTurnstileWidgetCreate(cmd *cobra.Command, args []string) error {
	if twName == "" {
		return fmt.Errorf("--name is required")
	}
	if len(twHostnames) == 0 {
		return fmt.Errorf("at least one --hostname is required")
	}
	svc, err := newTurnstileService()
	if err != nil {
		return outErr("failed to create Turnstile service", err)
	}
	widget, err := svc.Create(context.Background(), twOptions()...)
	if err != nil {
		return outErr("failed to create Turnstile widget", err)
	}
	return outPayload("Turnstile widget created", func() any { return widget }, func() {
		printSuccess("Turnstile widget %s created", widget.SiteKey)
		if widget.Secret != "" {
			printInfo("Secret key (shown ONCE — store it now, it is never displayed again):")
			fmt.Printf("  %s\n", widget.Secret)
		}
	})
}

func runTurnstileWidgetList(cmd *cobra.Command, args []string) error {
	svc, err := newTurnstileService()
	if err != nil {
		return outErr("failed to create Turnstile service", err)
	}
	widgets, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list Turnstile widgets", err)
	}
	return outResult(widgets, func() {
		if len(widgets) == 0 {
			printInfo("No Turnstile widgets")
			return
		}
		for _, w := range widgets {
			printInfo("%-24s  %-10s  %s", w.SiteKey, w.Mode, w.Name)
		}
		printInfo("Total: %d widget(s)", len(widgets))
	})
}

func runTurnstileWidgetGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("site key is required")
	}
	svc, err := newTurnstileService()
	if err != nil {
		return outErr("failed to create Turnstile service", err)
	}
	widget, err := svc.Get(context.Background(), args[0])
	if err != nil {
		return outErr(fmt.Sprintf("failed to get Turnstile widget %q", args[0]), err)
	}
	return outResult(widget, func() {
		printInfo("%-24s  %-10s  %s", widget.SiteKey, widget.Mode, widget.Name)
	})
}

func runTurnstileWidgetUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("site key is required")
	}
	svc, err := newTurnstileService()
	if err != nil {
		return outErr("failed to create Turnstile service", err)
	}
	widget, err := svc.Update(context.Background(), args[0], twOptions()...)
	if err != nil {
		return outErr(fmt.Sprintf("failed to update Turnstile widget %q", args[0]), err)
	}
	return outPayload("Turnstile widget updated", func() any { return widget }, func() {
		printSuccess("Turnstile widget %s updated", widget.SiteKey)
	})
}

func runTurnstileWidgetDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("site key is required")
	}
	siteKey := args[0]

	// Not registry-flagged destructive yet (registry pass follows): the
	// confirmation prompt stays live; an explicit --dry-run still
	// short-circuits before any service call.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "turnstile widget delete"), twForce)
	if dry {
		return outPayload("DRY RUN: Would delete Turnstile widget", func() any {
			return map[string]string{"site_key": siteKey}
		}, func() {
			printInfo("DRY RUN: Would delete Turnstile widget '%s'", siteKey)
		})
	}
	if !twForce && !ux.Confirm(fmt.Sprintf("Delete Turnstile widget '%s'? Sites using its site key will stop verifying.", siteKey)) {
		printInfo("Turnstile widget deletion cancelled")
		return nil
	}

	svc, err := newTurnstileService()
	if err != nil {
		return outErr("failed to create Turnstile service", err)
	}
	if err := svc.Delete(context.Background(), siteKey); err != nil {
		return outErr(fmt.Sprintf("failed to delete Turnstile widget %q", siteKey), err)
	}
	return outPayload("Turnstile widget deleted", func() any {
		return map[string]string{"site_key": siteKey}
	}, func() { printSuccess("Turnstile widget %s deleted", siteKey) })
}

func init() {
	rootCmd.AddCommand(turnstileCmd)
	turnstileCmd.AddCommand(turnstileWidgetCmd)

	turnstileWidgetCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Turnstile widget (secret key shown once)",
		RunE:  runTurnstileWidgetCreate,
	}
	turnstileWidgetListCmd := &cobra.Command{
		Use:   "list",
		Short: "List Turnstile widgets",
		RunE:  runTurnstileWidgetList,
	}
	turnstileWidgetGetCmd := &cobra.Command{
		Use:   "get [site-key]",
		Short: "Get a Turnstile widget",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTurnstileWidgetGet,
	}
	turnstileWidgetUpdateCmd := &cobra.Command{
		Use:   "update [site-key]",
		Short: "Update a Turnstile widget",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTurnstileWidgetUpdate,
	}
	turnstileWidgetDeleteCmd = &cobra.Command{
		Use:   "delete [site-key]",
		Short: "Delete a Turnstile widget",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTurnstileWidgetDelete,
	}
	turnstileWidgetDeleteCmd.Flags().BoolVar(&twForce, "force", false, "Skip confirmation prompt")

	turnstileWidgetCmd.AddCommand(turnstileWidgetCreateCmd, turnstileWidgetListCmd, turnstileWidgetGetCmd, turnstileWidgetUpdateCmd, turnstileWidgetDeleteCmd)

	turnstileWidgetCreateCmd.Flags().StringVar(&twName, "name", "", "Widget name (required)")
	turnstileWidgetCreateCmd.Flags().StringSliceVar(&twHostnames, "hostname", nil, "Hostname the widget protects (repeatable, required)")
	turnstileWidgetCreateCmd.Flags().StringVar(&twMode, "mode", "", "Widget mode (invisible, non-interactive, managed)")
	turnstileWidgetCreateCmd.Flags().BoolVar(&twBotFightMode, "bot-fight-mode", false, "Enable bot fight mode")
	turnstileWidgetCreateCmd.Flags().BoolVar(&twOffLabel, "off-label", false, "Hide the Turnstile label")

	turnstileWidgetUpdateCmd.Flags().StringVar(&twName, "name", "", "Widget name")
	turnstileWidgetUpdateCmd.Flags().StringSliceVar(&twHostnames, "hostname", nil, "Hostnames the widget protects (repeatable)")
	turnstileWidgetUpdateCmd.Flags().StringVar(&twMode, "mode", "", "Widget mode (invisible, non-interactive, managed)")
	turnstileWidgetUpdateCmd.Flags().BoolVar(&twBotFightMode, "bot-fight-mode", false, "Enable bot fight mode")
	turnstileWidgetUpdateCmd.Flags().BoolVar(&twOffLabel, "off-label", false, "Hide the Turnstile label")
}

// twTrimHostnames is retained for potential whitespace normalization in
// future hostname handling.
func twTrimHostnames(in []string) []string {
	out := in[:0]
	for _, h := range in {
		if h = strings.TrimSpace(h); h != "" {
			out = append(out, h)
		}
	}
	return out
}
