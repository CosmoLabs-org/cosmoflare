package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alert rules for Cloudflare services",
	Long: `Configure alert rules that notify on error rates, storage limits,
worker failures, and other conditions across Cloudflare services.

Alert rules are stored locally in .cosmoflare-alerts.yaml.
Alert history is logged to ~/.cosmoflare/alert-history.log (NDJSON).

Commands:
  list      List all configured alert rules
  create    Create a new alert rule
  get       Show alert rule details
  update    Update an existing alert rule
  delete    Delete an alert rule
  test      Trigger a test alert
  history   Show alert trigger history

Examples:
  cosmoflare alerts list
  cosmoflare alerts create high-errors --service r2 --condition error-rate --threshold 5 --action webhook --target https://hooks.example.com/alert
  cosmoflare alerts get high-errors
  cosmoflare alerts update high-errors --threshold 10
  cosmoflare alerts delete high-errors --force
  cosmoflare alerts test high-errors
  cosmoflare alerts history --limit 20`,
}

// Flag variables
var (
	alertService   string
	alertCondition string
	alertThreshold float64
	alertAction    string
	alertTarget    string
	alertForce     bool
	alertLimit     int
	alertSince     string
)

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured alert rules",
	Long: `List all alert rules configured in .cosmoflare-alerts.yaml.

Examples:
  cosmoflare alerts list
  cosmoflare alerts list --json`,
	RunE: runAlertsList,
}

var alertsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new alert rule",
	Long: `Create a new alert rule for monitoring Cloudflare services.

Required flags:
  --service     Service to monitor (r2, workers, kv, dns)
  --condition   Alert condition (error-rate, storage-limit, latency, failure-count)
  --threshold   Numeric threshold value that triggers the alert
  --action      Notification action (webhook, email, log)
  --target      Action target (URL for webhook, email address, or log path)

Examples:
  # Alert on high R2 error rate
  cosmoflare alerts create high-errors --service r2 --condition error-rate --threshold 5 --action webhook --target https://hooks.example.com/alert

  # Alert when KV storage approaches limit
  cosmoflare alerts create kv-storage-warn --service kv --condition storage-limit --threshold 80 --action email --target admin@example.com

  # Log worker failures
  cosmoflare alerts create worker-failures --service workers --condition failure-count --threshold 10 --action log --target /var/log/cosmoflare.log

  # JSON output
  cosmoflare alerts create dns-latency --service dns --condition latency --threshold 500 --action webhook --target https://hooks.example.com/dns --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsCreate,
}

var alertsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show alert rule details",
	Long: `Display the full configuration of a named alert rule.

Examples:
  cosmoflare alerts get high-errors
  cosmoflare alerts get high-errors --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsGet,
}

var alertsUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update an existing alert rule",
	Long: `Update one or more fields of an existing alert rule.
Only provided flags are modified; other fields remain unchanged.

Examples:
  cosmoflare alerts update high-errors --threshold 10
  cosmoflare alerts update high-errors --action email --target admin@example.com
  cosmoflare alerts update high-errors --service workers --condition failure-count --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsUpdate,
}

var alertsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an alert rule",
	Long: `Delete a named alert rule from .cosmoflare-alerts.yaml.

Requires --force flag to confirm deletion.

Examples:
  cosmoflare alerts delete high-errors --force
  cosmoflare alerts delete high-errors --force --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsDelete,
}

var alertsTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Trigger a test alert",
	Long: `Trigger a test alert for the named rule. The test alert is recorded
in alert history with is_test=true but does not actually send notifications.

Examples:
  cosmoflare alerts test high-errors
  cosmoflare alerts test high-errors --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAlertsTest,
}

var alertsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show alert trigger history",
	Long: `Display the alert trigger history from ~/.cosmoflare/alert-history.log.

Options:
  --limit   Maximum number of entries to show (most recent first)
  --since   Show entries after this time (RFC3339 format)

Examples:
  cosmoflare alerts history
  cosmoflare alerts history --limit 20
  cosmoflare alerts history --since 2026-01-01T00:00:00Z
  cosmoflare alerts history --json`,
	RunE: runAlertsHistory,
}

var alertsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Evaluate all alert rules once against live metrics and fire triggered alerts",
	Long: `Run one evaluation cycle: collect 24h usage analytics (Workers
invocations and R2 storage) from the configured account and evaluate every
enabled alert rule against them.

Triggered alerts are printed to the terminal (there is no running daemon to
bridge them to SSE). This is the same evaluation 'cosmoflare serve' runs
every 5 minutes.

Examples:
  cosmoflare alerts check
  cosmoflare alerts check --json`,
	RunE: runAlertsCheck,
}

func init() {
	rootCmd.AddCommand(alertsCmd)

	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsCreateCmd)
	alertsCmd.AddCommand(alertsGetCmd)
	alertsCmd.AddCommand(alertsUpdateCmd)
	alertsCmd.AddCommand(alertsDeleteCmd)
	alertsCmd.AddCommand(alertsTestCmd)
	alertsCmd.AddCommand(alertsHistoryCmd)
	alertsCmd.AddCommand(alertsCheckCmd)

	// Create flags
	alertsCreateCmd.Flags().StringVar(&alertService, "service", "", "Service to monitor (r2, workers, kv, dns)")
	alertsCreateCmd.Flags().StringVar(&alertCondition, "condition", "", "Alert condition (error-rate, storage-limit, latency, failure-count)")
	alertsCreateCmd.Flags().Float64Var(&alertThreshold, "threshold", 0, "Numeric threshold value")
	alertsCreateCmd.Flags().StringVar(&alertAction, "action", "", "Notification action (webhook, email, log)")
	alertsCreateCmd.Flags().StringVar(&alertTarget, "target", "", "Action target (URL, email, or log path)")
	_ = alertsCreateCmd.MarkFlagRequired("service")
	_ = alertsCreateCmd.MarkFlagRequired("condition")
	_ = alertsCreateCmd.MarkFlagRequired("threshold")
	_ = alertsCreateCmd.MarkFlagRequired("action")
	_ = alertsCreateCmd.MarkFlagRequired("target")

	// Update flags (same as create but not required)
	alertsUpdateCmd.Flags().StringVar(&alertService, "service", "", "Service to monitor (r2, workers, kv, dns)")
	alertsUpdateCmd.Flags().StringVar(&alertCondition, "condition", "", "Alert condition (error-rate, storage-limit, latency, failure-count)")
	alertsUpdateCmd.Flags().Float64Var(&alertThreshold, "threshold", 0, "Numeric threshold value")
	alertsUpdateCmd.Flags().StringVar(&alertAction, "action", "", "Notification action (webhook, email, log)")
	alertsUpdateCmd.Flags().StringVar(&alertTarget, "target", "", "Action target (URL, email, or log path)")

	// Delete flags
	alertsDeleteCmd.Flags().BoolVar(&alertForce, "force", false, "Confirm deletion")

	// History flags
	alertsHistoryCmd.Flags().IntVar(&alertLimit, "limit", 0, "Maximum number of entries to show")
	alertsHistoryCmd.Flags().StringVar(&alertSince, "since", "", "Show entries after this time (RFC3339)")
}

// firedAlert is one entry of 'alerts check --json' output.
type firedAlert struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
}

// runAlertsCheck evaluates every alert rule once against live metrics.
// Without a running daemon there is no SSE bridge, so notifications go to
// stdout via the manager's SetNotifier hook.
func runAlertsCheck(cmd *cobra.Command, args []string) error {
	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}
	rules, err := svc.List()
	if err != nil {
		return fmt.Errorf("failed to list alert rules: %w", err)
	}

	var fired []firedAlert
	evaluated := 0
	for _, r := range rules {
		if r != nil && r.Enabled {
			evaluated++
		}
	}

	var metrics webhook.EvalMetrics
	if evaluated > 0 {
		if AccountID == "" || APIToken == "" {
			return fmt.Errorf("account ID and API token are required for alert evaluation (run 'cosmoflare account' to configure)")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
		now := time.Now()
		metrics, err = webhook.CollectEvalMetrics(ctx, analytics, cosmoflare.AnalyticsWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now,
		})
		if err != nil {
			return fmt.Errorf("failed to collect metrics: %w", err)
		}
	}

	mgr := webhook.NewManager(nil, "")
	mgr.SetNotifier(func(p *webhook.NotificationPayload) {
		if p == nil || p.Alert == nil {
			return
		}
		printWarning("[ALERT] %s: %s value=%g threshold=%g", p.Alert.Name, p.Message, p.Value, p.Threshold)
		fired = append(fired, firedAlert{Name: p.Alert.Name, Value: p.Value, Threshold: p.Threshold})
	})

	eval := webhook.NewEvaluator(svc, mgr, 0) // one-shot: cooldown irrelevant
	firedNames := eval.Evaluate(metrics)

	if JSONOutput {
		if fired == nil {
			fired = []firedAlert{}
		}
		return printJSON(fired)
	}
	printInfo("%d rule(s) evaluated, %d fired", evaluated, len(firedNames))
	return nil
}

// getAlertServiceFn is the factory for AlertService. Tests override this
// to inject temp file paths.
var getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
	return cosmoflare.NewAlertService("", "")
}

func getAlertService() (*cosmoflare.AlertService, error) {
	return getAlertServiceFn()
}

func runAlertsList(cmd *cobra.Command, args []string) error {
	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	rules, err := svc.List()
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list alerts: %v", err))
		}
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Alert rules listed", rules)
	}

	if len(rules) == 0 {
		printInfo("No alert rules configured. Use 'cosmoflare alerts create' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSERVICE\tCONDITION\tTHRESHOLD\tACTION\tENABLED")
	for _, r := range rules {
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\t%v\n",
			r.Name, r.Service, r.Condition, r.Threshold, r.Action, r.Enabled)
	}
	w.Flush()
	return nil
}

func runAlertsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	rule := &cosmoflare.AlertRule{
		Name:      name,
		Service:   alertService,
		Condition: alertCondition,
		Threshold: alertThreshold,
		Action:    alertAction,
		Target:    alertTarget,
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create alert rule", rule)
		}
		printInfo("DRY RUN: Would create alert rule %q", name)
		return nil
	}

	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	created, err := svc.Create(rule)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create alert rule: %v", err))
		}
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Alert rule created", created)
	}
	printSuccess("Alert rule %q created", created.Name)
	return nil
}

func runAlertsGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	rule, err := svc.Get(name)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("alert rule %q not found: %v", name, err))
		}
		return fmt.Errorf("alert rule %q not found: %w", name, err)
	}

	if JSONOutput {
		return printJSON(rule)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Name:\t%s\n", rule.Name)
	fmt.Fprintf(w, "Service:\t%s\n", rule.Service)
	fmt.Fprintf(w, "Condition:\t%s\n", rule.Condition)
	fmt.Fprintf(w, "Threshold:\t%.2f\n", rule.Threshold)
	fmt.Fprintf(w, "Action:\t%s\n", rule.Action)
	fmt.Fprintf(w, "Target:\t%s\n", rule.Target)
	fmt.Fprintf(w, "Enabled:\t%v\n", rule.Enabled)
	fmt.Fprintf(w, "Created:\t%s\n", rule.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Updated:\t%s\n", rule.UpdatedAt.Format(time.RFC3339))
	w.Flush()
	return nil
}

func runAlertsUpdate(cmd *cobra.Command, args []string) error {
	name := args[0]

	update := &cosmoflare.AlertRule{}
	if cmd.Flags().Changed("service") {
		update.Service = alertService
	}
	if cmd.Flags().Changed("condition") {
		update.Condition = alertCondition
	}
	if cmd.Flags().Changed("threshold") {
		update.Threshold = alertThreshold
	}
	if cmd.Flags().Changed("action") {
		update.Action = alertAction
	}
	if cmd.Flags().Changed("target") {
		update.Target = alertTarget
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update alert rule", map[string]interface{}{"name": name, "updates": update})
		}
		printInfo("DRY RUN: Would update alert rule %q", name)
		return nil
	}

	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	updated, err := svc.Update(name, update)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update alert rule: %v", err))
		}
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Alert rule updated", updated)
	}
	printSuccess("Alert rule %q updated", updated.Name)
	return nil
}

func runAlertsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !alertForce && !DryRun {
		return fmt.Errorf("deletion requires --force flag\n\nUsage: cosmoflare alerts delete %s --force", name)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete alert rule", map[string]string{"name": name})
		}
		printInfo("DRY RUN: Would delete alert rule %q", name)
		return nil
	}

	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	if err := svc.Delete(name); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete alert rule: %v", err))
		}
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Alert rule deleted", map[string]string{"name": name})
	}
	printSuccess("Alert rule %q deleted", name)
	return nil
}

func runAlertsTest(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	entry, err := svc.Test(name)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to test alert: %v", err))
		}
		return fmt.Errorf("failed to test alert: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Test alert triggered", entry)
	}
	printSuccess("Test alert triggered for rule %q", name)
	fmt.Printf("  Service:   %s\n", entry.Service)
	fmt.Printf("  Condition: %s\n", entry.Condition)
	fmt.Printf("  Action:    %s -> %s\n", entry.Action, entry.Target)
	fmt.Printf("  Logged to: ~/.cosmoflare/alert-history.log\n")
	return nil
}

func runAlertsHistory(cmd *cobra.Command, args []string) error {
	svc, err := getAlertService()
	if err != nil {
		return fmt.Errorf("failed to create alert service: %w", err)
	}

	var since time.Time
	if alertSince != "" {
		parsed, err := time.Parse(time.RFC3339, alertSince)
		if err != nil {
			return fmt.Errorf("invalid --since format, expected RFC3339 (e.g. 2026-01-01T00:00:00Z): %w", err)
		}
		since = parsed
	}

	entries, err := svc.History(alertLimit, since)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to read alert history: %v", err))
		}
		return fmt.Errorf("failed to read alert history: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Alert history", entries)
	}

	if len(entries) == 0 {
		printInfo("No alert history found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tRULE\tSERVICE\tCONDITION\tVALUE\tTHRESHOLD\tTEST")
	for _, e := range entries {
		testStr := ""
		if e.IsTest {
			testStr = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f\t%.2f\t%s\n",
			e.Timestamp.Format(time.RFC3339), e.RuleName, e.Service,
			e.Condition, e.Value, e.Threshold, testStr)
	}
	w.Flush()
	return nil
}
