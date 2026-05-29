package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

var diffOutput string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local config against live Cloudflare state",
	Long: `Show what would change if you applied your local .cosmoflare.yaml to live Cloudflare.

This is a read-only command — it never modifies anything. It compares
the resources defined in your project's .cosmoflare.yaml against what
currently exists in your Cloudflare account.

Output shows:
  + additions  — resources in config but not deployed
  - deletions  — resources deployed but not in config
  ~ changes    — resources that exist but with different settings

Subcommands:
  workers   Compare only Workers config
  dns       Compare only DNS records
  kv        Compare only KV namespaces
  r2        Compare only R2 buckets

Examples:
  r2go2 diff                           # Compare all configured services
  r2go2 diff --output=summary          # Show counts only
  r2go2 diff workers                   # Compare only Workers
  r2go2 diff dns                       # Compare only DNS records
  r2go2 diff kv                        # Compare only KV namespaces
  r2go2 diff r2                        # Compare only R2 buckets
  r2go2 diff --json                    # Structured JSON output
  r2go2 diff workers --json            # JSON output for Workers only`,
	RunE: runDiffAll,
}

var diffWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Compare Workers config against live state",
	Long: `Compare Workers defined in .cosmoflare.yaml against deployed Workers.

Shows which Workers exist in config but are not deployed (additions),
which are deployed but not in config (deletions).

Examples:
  r2go2 diff workers
  r2go2 diff workers --json
  r2go2 diff workers --output=summary`,
	RunE: runDiffWorkers,
}

var diffDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Compare DNS records config against live state",
	Long: `Compare DNS records defined in .cosmoflare.yaml against live zone records.

Requires dns.zone_id to be set in .cosmoflare.yaml.

Shows additions, deletions, and modifications (TTL/proxied changes).

Examples:
  r2go2 diff dns
  r2go2 diff dns --json
  r2go2 diff dns --output=summary`,
	RunE: runDiffDNS,
}

var diffKVCmd = &cobra.Command{
	Use:   "kv",
	Short: "Compare KV namespaces config against live state",
	Long: `Compare KV namespaces defined in .cosmoflare.yaml against live namespaces.

Shows which namespaces exist in config but not in Cloudflare (additions),
and which exist in Cloudflare but not in config (deletions).

Examples:
  r2go2 diff kv
  r2go2 diff kv --json
  r2go2 diff kv --output=summary`,
	RunE: runDiffKV,
}

var diffR2Cmd = &cobra.Command{
	Use:   "r2",
	Short: "Compare R2 buckets config against live state",
	Long: `Compare R2 buckets defined in .cosmoflare.yaml against live buckets.

Shows which buckets exist in config but not in Cloudflare (additions),
and which exist in Cloudflare but not in config (deletions).

Examples:
  r2go2 diff r2
  r2go2 diff r2 --json
  r2go2 diff r2 --output=summary`,
	RunE: runDiffR2,
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.AddCommand(diffWorkersCmd)
	diffCmd.AddCommand(diffDNSCmd)
	diffCmd.AddCommand(diffKVCmd)
	diffCmd.AddCommand(diffR2Cmd)

	diffCmd.PersistentFlags().StringVar(&diffOutput, "output", "full", "Output detail level: full or summary")
}

func getDiffService() (*r2go2.DiffService, error) {
	return r2go2.NewDiffService(AccountID, APIToken)
}

func loadConfig() (*r2go2.CosmoflareConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	cfg, err := r2go2.LoadCosmoflareConfig(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\n\nRun 'cosmoflare init' to create a .cosmoflare.yaml", err)
	}
	return cfg, nil
}

func runDiffAll(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	svc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	summary, err := svc.CompareAll(ctx, cfg)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Diff complete", summary)
	}

	if !summary.HasChanges {
		printSuccess("No differences found — local config matches live state")
		return nil
	}

	for _, result := range summary.Results {
		printDiffResult(&result)
	}

	printDiffSummaryLine(summary)
	return nil
}

func runDiffWorkers(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Workers) == 0 {
		printInfo("No workers configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	result, err := svc.CompareWorkers(ctx, cfg.Workers)
	if err != nil {
		return fmt.Errorf("workers diff failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Workers diff complete", result)
	}

	if !result.HasChanges() {
		printSuccess("Workers: no differences found")
		return nil
	}

	printDiffResult(result)
	return nil
}

func runDiffDNS(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.DNS.ZoneID == "" {
		return fmt.Errorf("dns.zone_id is required in .cosmoflare.yaml for DNS diff")
	}

	svc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	result, err := svc.CompareDNS(ctx, cfg.DNS)
	if err != nil {
		return fmt.Errorf("dns diff failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("DNS diff complete", result)
	}

	if !result.HasChanges() {
		printSuccess("DNS: no differences found")
		return nil
	}

	printDiffResult(result)
	return nil
}

func runDiffKV(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.KV.Namespaces) == 0 {
		printInfo("No KV namespaces configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	result, err := svc.CompareKV(ctx, cfg.KV)
	if err != nil {
		return fmt.Errorf("kv diff failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("KV diff complete", result)
	}

	if !result.HasChanges() {
		printSuccess("KV: no differences found")
		return nil
	}

	printDiffResult(result)
	return nil
}

func runDiffR2(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.R2.Buckets) == 0 {
		printInfo("No R2 buckets configured in .cosmoflare.yaml")
		return nil
	}

	svc, err := getDiffService()
	if err != nil {
		return fmt.Errorf("failed to create diff service: %w", err)
	}

	ctx := context.Background()
	result, err := svc.CompareR2(ctx, cfg.R2)
	if err != nil {
		return fmt.Errorf("r2 diff failed: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("R2 diff complete", result)
	}

	if !result.HasChanges() {
		printSuccess("R2: no differences found")
		return nil
	}

	printDiffResult(result)
	return nil
}

// printDiffResult renders a single service diff in human-readable format.
func printDiffResult(result *r2go2.DiffResult) {
	if !result.HasChanges() {
		return
	}

	fmt.Printf("\n=== %s ===\n", result.Service)

	if diffOutput == "summary" {
		fmt.Printf("  + %d additions, - %d deletions, ~ %d modifications\n",
			len(result.Additions), len(result.Deletions), len(result.Changes))
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, e := range result.Additions {
		fmt.Fprintf(w, "  + %s\t%s\n", e.Resource, e.Detail)
	}
	for _, e := range result.Deletions {
		fmt.Fprintf(w, "  - %s\t%s\n", e.Resource, e.Detail)
	}
	for _, e := range result.Changes {
		fmt.Fprintf(w, "  ~ %s\t%s\n", e.Resource, e.Detail)
	}

	w.Flush()
}

// printDiffSummaryLine prints the aggregate summary at the bottom.
func printDiffSummaryLine(summary *r2go2.DiffSummary) {
	total := summary.TotalAdd + summary.TotalDel + summary.TotalMod
	fmt.Printf("\n%d change(s): +%d additions, -%d deletions, ~%d modifications\n",
		total, summary.TotalAdd, summary.TotalDel, summary.TotalMod)
}
