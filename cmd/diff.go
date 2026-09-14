package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
  cosmoflare diff                           # Compare all configured services
  cosmoflare diff --output=summary          # Show counts only
  cosmoflare diff workers                   # Compare only Workers
  cosmoflare diff dns                       # Compare only DNS records
  cosmoflare diff kv                        # Compare only KV namespaces
  cosmoflare diff r2                        # Compare only R2 buckets
  cosmoflare diff --json                    # Structured JSON output
  cosmoflare diff workers --json            # JSON output for Workers only`,
	RunE: runDiffAll,
}

var diffWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Compare Workers config against live state",
	Long: `Compare Workers defined in .cosmoflare.yaml against deployed Workers.

Shows which Workers exist in config but are not deployed (additions),
which are deployed but not in config (deletions).

Examples:
  cosmoflare diff workers
  cosmoflare diff workers --json
  cosmoflare diff workers --output=summary`,
	RunE: runDiffWorkers,
}

var diffDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Compare DNS records config against live state",
	Long: `Compare DNS records defined in .cosmoflare.yaml against live zone records.

Requires dns.zone_id to be set in .cosmoflare.yaml.

Shows additions, deletions, and modifications (TTL/proxied changes).

Examples:
  cosmoflare diff dns
  cosmoflare diff dns --json
  cosmoflare diff dns --output=summary`,
	RunE: runDiffDNS,
}

var diffKVCmd = &cobra.Command{
	Use:   "kv",
	Short: "Compare KV namespaces config against live state",
	Long: `Compare KV namespaces defined in .cosmoflare.yaml against live namespaces.

Shows which namespaces exist in config but not in Cloudflare (additions),
and which exist in Cloudflare but not in config (deletions).

Examples:
  cosmoflare diff kv
  cosmoflare diff kv --json
  cosmoflare diff kv --output=summary`,
	RunE: runDiffKV,
}

var diffR2Cmd = &cobra.Command{
	Use:   "r2",
	Short: "Compare R2 buckets config against live state",
	Long: `Compare R2 buckets defined in .cosmoflare.yaml against live buckets.

Shows which buckets exist in config but not in Cloudflare (additions),
and which exist in Cloudflare but not in config (deletions).

Examples:
  cosmoflare diff r2
  cosmoflare diff r2 --json
  cosmoflare diff r2 --output=summary`,
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

func getDiffService() (*cosmoflare.DiffService, error) {
	return cosmoflare.NewDiffService(AccountID, APIToken)
}

func loadConfig() (*cosmoflare.CosmoflareConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	cfg, err := cosmoflare.LoadCosmoflareConfig(dir)
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
		return outErr("failed to create diff service", err)
	}

	ctx := context.Background()
	summary, err := svc.CompareAll(ctx, cfg)
	if err != nil {
		return outErr("diff failed", err)
	}

	return outPayload("Diff complete", func() any {
		return summary
	}, func() {
		if !summary.HasChanges {
			printSuccess("No differences found — local config matches live state")
			return
		}

		for _, result := range summary.Results {
			printDiffResult(&result)
		}

		printDiffSummaryLine(summary)
	})
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
		return outErr("failed to create diff service", err)
	}

	ctx := context.Background()
	result, err := svc.CompareWorkers(ctx, cfg.Workers)
	if err != nil {
		return outErr("workers diff failed", err)
	}

	return outPayload("Workers diff complete", func() any {
		return result
	}, func() {
		if !result.HasChanges() {
			printSuccess("Workers: no differences found")
			return
		}

		printDiffResult(result)
	})
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
		return outErr("failed to create diff service", err)
	}

	ctx := context.Background()
	result, err := svc.CompareDNS(ctx, cfg.DNS)
	if err != nil {
		return outErr("dns diff failed", err)
	}

	return outPayload("DNS diff complete", func() any {
		return result
	}, func() {
		if !result.HasChanges() {
			printSuccess("DNS: no differences found")
			return
		}

		printDiffResult(result)
	})
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
		return outErr("failed to create diff service", err)
	}

	ctx := context.Background()
	result, err := svc.CompareKV(ctx, cfg.KV)
	if err != nil {
		return outErr("kv diff failed", err)
	}

	return outPayload("KV diff complete", func() any {
		return result
	}, func() {
		if !result.HasChanges() {
			printSuccess("KV: no differences found")
			return
		}

		printDiffResult(result)
	})
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
		return outErr("failed to create diff service", err)
	}

	ctx := context.Background()
	result, err := svc.CompareR2(ctx, cfg.R2)
	if err != nil {
		return outErr("r2 diff failed", err)
	}

	return outPayload("R2 diff complete", func() any {
		return result
	}, func() {
		if !result.HasChanges() {
			printSuccess("R2: no differences found")
			return
		}

		printDiffResult(result)
	})
}

// printDiffResult renders a single service diff in human-readable format.
func printDiffResult(result *cosmoflare.DiffResult) {
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
func printDiffSummaryLine(summary *cosmoflare.DiffSummary) {
	total := summary.TotalAdd + summary.TotalDel + summary.TotalMod
	fmt.Printf("\n%d change(s): +%d additions, -%d deletions, ~%d modifications\n",
		total, summary.TotalAdd, summary.TotalDel, summary.TotalMod)
}
