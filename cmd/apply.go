package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var applyYes bool
var applyDeleteUnmanaged bool

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply local config changes to live Cloudflare state",
	Long: `Reconcile your local .cosmoflare.yaml against live Cloudflare state and
apply the required changes to make live match config.

This is the declarative "desired state" command — the counterpart to
'cosmoflare diff'. It computes what would change, shows you the plan,
and (after confirmation) executes the mutations.

For each service, apply will:
  + create  resources in config but not deployed
  - delete  resources deployed but not in config
  ~ update  resources that exist but with different settings

Subcommands:
  workers   Apply only Workers changes
  dns       Apply only DNS record changes
  kv        Apply only KV namespace changes
  r2        Apply only R2 bucket changes

Flags:
  --yes, -y     Skip confirmation prompt
  --dry-run     Show what would change without executing (same as diff)

Examples:
  cosmoflare apply                             # Apply all changes (with confirmation)
  cosmoflare apply --yes                       # Apply all changes without prompting
  cosmoflare apply --dry-run                   # Preview changes without applying
  cosmoflare apply workers                     # Apply only Workers changes
  cosmoflare apply dns --yes                   # Apply DNS changes without prompting
  cosmoflare apply kv                          # Apply KV namespace changes
  cosmoflare apply r2                          # Apply R2 bucket changes
  cosmoflare apply --json                      # Structured JSON output
  cosmoflare apply workers --json --yes        # JSON output, no prompt`,
	RunE: runApplyAll,
}

var applyWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Apply Workers config changes to live state",
	Long: `Apply Workers defined in .cosmoflare.yaml to live Cloudflare.

Creates workers that exist in config but are not deployed, and deletes
workers that are deployed but not in config.

Examples:
  cosmoflare apply workers
  cosmoflare apply workers --yes
  cosmoflare apply workers --dry-run
  cosmoflare apply workers --json`,
	RunE: runApplyWorkers,
}

var applyDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Apply DNS record config changes to live state",
	Long: `Apply DNS records defined in .cosmoflare.yaml to the live zone.

Requires dns.zone_id to be set in .cosmoflare.yaml.

Creates, updates, and deletes records to match the config.

Examples:
  cosmoflare apply dns
  cosmoflare apply dns --yes
  cosmoflare apply dns --dry-run
  cosmoflare apply dns --json`,
	RunE: runApplyDNS,
}

var applyKVCmd = &cobra.Command{
	Use:   "kv",
	Short: "Apply KV namespace config changes to live state",
	Long: `Apply KV namespaces defined in .cosmoflare.yaml to live Cloudflare.

Creates namespaces that exist in config but not in Cloudflare, and
deletes namespaces that exist in Cloudflare but not in config.

Examples:
  cosmoflare apply kv
  cosmoflare apply kv --yes
  cosmoflare apply kv --dry-run
  cosmoflare apply kv --json`,
	RunE: runApplyKV,
}

var applyR2Cmd = &cobra.Command{
	Use:   "r2",
	Short: "Apply R2 bucket config changes to live state",
	Long: `Apply R2 buckets defined in .cosmoflare.yaml to live Cloudflare.

Creates buckets that exist in config but not in Cloudflare, and
deletes buckets that exist in Cloudflare but not in config.

Examples:
  cosmoflare apply r2
  cosmoflare apply r2 --yes
  cosmoflare apply r2 --dry-run
  cosmoflare apply r2 --json`,
	RunE: runApplyR2,
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.AddCommand(applyWorkersCmd)
	applyCmd.AddCommand(applyDNSCmd)
	applyCmd.AddCommand(applyKVCmd)
	applyCmd.AddCommand(applyR2Cmd)

	applyCmd.PersistentFlags().BoolVarP(&applyYes, "yes", "y", false, "Skip confirmation prompt")
	applyCmd.PersistentFlags().BoolVar(&applyDeleteUnmanaged, "delete-unmanaged", false,
		"Delete live resources that are NOT declared in the config (unmanaged). Without this flag, such resources are skipped (BUG-049).")
}

func getApplyService() (*cosmoflare.ApplyService, error) {
	return cosmoflare.NewApplyService(AccountID, APIToken,
		cosmoflare.WithDeleteUnmanaged(applyDeleteUnmanaged))
}

func runApplyAll(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	svc, err := getApplyService()
	if err != nil {
		return fmt.Errorf("failed to create apply service: %w", err)
	}

	// First compute the diff to show the plan
	diffSvc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	diffSummary, err := diffSvc.CompareAll(ctx, cfg)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	if !diffSummary.HasChanges {
		if JSONOutput {
			return printSuccessJSON("No changes to apply", nil)
		}
		printSuccess("No changes to apply — local config matches live state")
		return nil
	}

	// Show the plan
	if !JSONOutput {
		fmt.Println("=== Apply Plan ===")
		for _, result := range diffSummary.Results {
			printDiffResult(&result)
		}
		printDiffSummaryLine(diffSummary)
		fmt.Println()
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("Dry run — no changes applied", diffSummary)
		}
		printWarning("Dry run — no changes were applied")
		return nil
	}

	// Confirm unless --yes
	if !applyYes && !JSONOutput {
		if !confirmApply() {
			printInfo("Apply cancelled")
			return nil
		}
	}

	summary, err := svc.ApplyAll(ctx, cfg, false)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	if JSONOutput {
		msg := "Apply complete"
		if summary.HasErrors {
			msg = "Apply completed with errors"
		}
		return printSuccessJSON(msg, summary)
	}

	printApplySummary(summary)
	return nil
}

func runApplyWorkers(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Workers) == 0 {
		printInfo("No workers configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getApplyService()
	if err != nil {
		return fmt.Errorf("failed to create apply service: %w", err)
	}

	// Show diff first
	diffSvc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	dr, err := diffSvc.CompareWorkers(ctx, cfg.Workers)
	if err != nil {
		return fmt.Errorf("workers diff failed: %w", err)
	}

	if !dr.HasChanges() {
		if JSONOutput {
			return printSuccessJSON("Workers: no changes to apply", nil)
		}
		printSuccess("Workers: no changes to apply")
		return nil
	}

	if !JSONOutput {
		printDiffResult(dr)
		fmt.Println()
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("Dry run — no changes applied", dr)
		}
		printWarning("Dry run — no changes were applied")
		return nil
	}

	if !applyYes && !JSONOutput {
		if !confirmApply() {
			printInfo("Apply cancelled")
			return nil
		}
	}

	result, err := svc.ApplyWorkers(ctx, cfg, false)
	if err != nil {
		return fmt.Errorf("workers apply failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Workers apply complete", result)
	}

	printApplyResult(result)
	return nil
}

func runApplyDNS(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.DNS.ZoneID == "" {
		return fmt.Errorf("dns.zone_id is required in .cosmoflare.yaml for DNS apply")
	}

	svc, err := getApplyService()
	if err != nil {
		return fmt.Errorf("failed to create apply service: %w", err)
	}

	diffSvc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	dr, err := diffSvc.CompareDNS(ctx, cfg.DNS)
	if err != nil {
		return fmt.Errorf("dns diff failed: %w", err)
	}

	if !dr.HasChanges() {
		if JSONOutput {
			return printSuccessJSON("DNS: no changes to apply", nil)
		}
		printSuccess("DNS: no changes to apply")
		return nil
	}

	if !JSONOutput {
		printDiffResult(dr)
		fmt.Println()
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("Dry run — no changes applied", dr)
		}
		printWarning("Dry run — no changes were applied")
		return nil
	}

	if !applyYes && !JSONOutput {
		if !confirmApply() {
			printInfo("Apply cancelled")
			return nil
		}
	}

	result, err := svc.ApplyDNS(ctx, cfg, false)
	if err != nil {
		return fmt.Errorf("dns apply failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("DNS apply complete", result)
	}

	printApplyResult(result)
	return nil
}

func runApplyKV(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.KV.Namespaces) == 0 {
		printInfo("No KV namespaces configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getApplyService()
	if err != nil {
		return fmt.Errorf("failed to create apply service: %w", err)
	}

	diffSvc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	dr, err := diffSvc.CompareKV(ctx, cfg.KV)
	if err != nil {
		return fmt.Errorf("kv diff failed: %w", err)
	}

	if !dr.HasChanges() {
		if JSONOutput {
			return printSuccessJSON("KV: no changes to apply", nil)
		}
		printSuccess("KV: no changes to apply")
		return nil
	}

	if !JSONOutput {
		printDiffResult(dr)
		fmt.Println()
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("Dry run — no changes applied", dr)
		}
		printWarning("Dry run — no changes were applied")
		return nil
	}

	if !applyYes && !JSONOutput {
		if !confirmApply() {
			printInfo("Apply cancelled")
			return nil
		}
	}

	result, err := svc.ApplyKV(ctx, cfg, false)
	if err != nil {
		return fmt.Errorf("kv apply failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("KV apply complete", result)
	}

	printApplyResult(result)
	return nil
}

func runApplyR2(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.R2.Buckets) == 0 {
		printInfo("No R2 buckets configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getApplyService()
	if err != nil {
		return fmt.Errorf("failed to create apply service: %w", err)
	}

	diffSvc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	dr, err := diffSvc.CompareR2(ctx, cfg.R2)
	if err != nil {
		return fmt.Errorf("r2 diff failed: %w", err)
	}

	if !dr.HasChanges() {
		if JSONOutput {
			return printSuccessJSON("R2: no changes to apply", nil)
		}
		printSuccess("R2: no changes to apply")
		return nil
	}

	if !JSONOutput {
		printDiffResult(dr)
		fmt.Println()
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("Dry run — no changes applied", dr)
		}
		printWarning("Dry run — no changes were applied")
		return nil
	}

	if !applyYes && !JSONOutput {
		if !confirmApply() {
			printInfo("Apply cancelled")
			return nil
		}
	}

	result, err := svc.ApplyR2(ctx, cfg, false)
	if err != nil {
		return fmt.Errorf("r2 apply failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("R2 apply complete", result)
	}

	printApplyResult(result)
	return nil
}

// confirmApply prompts the user for confirmation before applying changes.
func confirmApply() bool {
	fmt.Print("Proceed with apply? [y/N] ")
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false
	}
	return response == "y" || response == "Y" || response == "yes" || response == "YES"
}

// printApplyResult renders a single service apply result in human-readable format.
func printApplyResult(result *cosmoflare.ApplyResult) {
	if len(result.Operations) == 0 {
		return
	}

	fmt.Printf("\n=== %s ===\n", result.Service)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, op := range result.Operations {
		symbol := actionSymbol(op.Action)
		status := statusSymbol(op.Status)
		if op.Error != "" {
			fmt.Fprintf(w, "  %s %s %s\t%s\t[error: %s]\n", symbol, status, op.Resource, op.Detail, op.Error)
		} else {
			fmt.Fprintf(w, "  %s %s %s\t%s\n", symbol, status, op.Resource, op.Detail)
		}
	}

	w.Flush()

	s := result.Summary()
	fmt.Printf("  %d succeeded, %d failed, %d skipped\n", s.Succeeded, s.Failed, s.Skipped)
}

// printApplySummary renders the aggregate apply summary.
func printApplySummary(summary *cosmoflare.ApplySummary) {
	for _, result := range summary.Results {
		printApplyResult(&result)
	}

	fmt.Printf("\nTotal: %d succeeded, %d failed, %d skipped\n",
		summary.TotalSucceeded, summary.TotalFailed, summary.TotalSkipped)

	if summary.HasErrors {
		printWarning("Some operations failed — review the output above")
	} else {
		printSuccess("All changes applied successfully")
	}
}

// actionSymbol returns a display symbol for an apply action.
func actionSymbol(action cosmoflare.ApplyAction) string {
	switch action {
	case cosmoflare.ApplyCreate:
		return "+"
	case cosmoflare.ApplyDelete:
		return "-"
	case cosmoflare.ApplyUpdate:
		return "~"
	default:
		return " "
	}
}

// statusSymbol returns a display symbol for an operation status.
func statusSymbol(status cosmoflare.ApplyStatus) string {
	switch status {
	case cosmoflare.ApplyStatusSuccess:
		return "OK"
	case cosmoflare.ApplyStatusFailed:
		return "FAIL"
	case cosmoflare.ApplyStatusSkipped:
		return "SKIP"
	default:
		return "?"
	}
}
