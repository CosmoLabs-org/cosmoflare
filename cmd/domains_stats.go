package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Aggregate statistics across all domains",
	Long: `Summarize all domains into counts by NS status and SSL status, plus an
attention list of domains an operator should look at.

Examples:
  cosmoflare domains stats
  cosmoflare domains stats --json`,
	Args: cobra.NoArgs,
	RunE: runDomainsStats,
}

func runDomainsStats(cmd *cobra.Command, args []string) error {
	svc, err := newDomainService(false)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create domain service: %v", err))
		}
		return fmt.Errorf("failed to create domain service: %w", err)
	}

	ctx := context.Background()

	domains, _, err := svc.List(ctx, cosmoflare.DomainListOptions{})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list domains: %v", err))
		}
		return fmt.Errorf("failed to list domains: %w", err)
	}

	summary := cosmoflare.SummarizeDomains(domains)

	if JSONOutput {
		return printJSON(summary)
	}

	fmt.Printf("Total domains: %d\n", summary.Total)
	fmt.Printf("Needs attention: %d\n", summary.NeedsAttention)

	fmt.Println("By NS status:")
	printCountMap(summary.ByNSStatus)

	fmt.Println("By SSL status:")
	printCountMap(summary.BySSLStatus)

	if len(summary.Attention) > 0 {
		fmt.Println("Attention:")
		for _, name := range summary.Attention {
			fmt.Printf("  - %s\n", name)
		}
	}

	return nil
}

// printCountMap prints a map[string]int in deterministic (key-sorted) order.
func printCountMap(m map[string]int) {
	if len(m) == 0 {
		fmt.Println("  (none)")
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}
}
