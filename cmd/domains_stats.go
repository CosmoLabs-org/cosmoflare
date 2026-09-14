package cmd

import (
	"context"
	"fmt"
	"sort"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
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

// domainsStatsCheckRedirects enables the opt-in redirect-destination probe
// pass. It doubles as the enrich argument to newDomainService so the service
// is built with WithRedirects/WithPageRules wiring — without it GetDetail
// returns no redirects and the probe pass would be a silent no-op.
var domainsStatsCheckRedirects bool

func runDomainsStats(cmd *cobra.Command, args []string) error {
	svc, err := newDomainService(domainsStatsCheckRedirects)
	if err != nil {
		return outErr("failed to create domain service", err)
	}

	ctx := context.Background()

	domains, _, err := svc.List(ctx, cosmoflare.DomainListOptions{})
	if err != nil {
		return outErr("failed to list domains", err)
	}

	if domainsStatsCheckRedirects {
		if err := checkDomainRedirects(ctx, svc, domains); err != nil {
			return outErr("redirect check failed", err)
		}
	}

	summary := cosmoflare.SummarizeDomains(domains)

	return outResult(summary, func() {
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
	})
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

func init() {
	domainsStatsCmd.Flags().BoolVar(&domainsStatsCheckRedirects, "check-redirects", false,
		"Probe redirect destinations (opt-in: live HTTP probes, 8 concurrent, 10s each)")
}

// checkDomainRedirects probes every domain's redirect destinations and sets
// RedirectIssue on each DomainStatus. Destinations are deduplicated across
// domains so each URL is probed once per run.
func checkDomainRedirects(ctx context.Context, svc *cosmoflare.DomainService, domains []*cosmoflare.DomainStatus) error {
	byZone := make(map[string]*cosmoflare.DomainStatus, len(domains))
	var zoneIDs []string
	for _, d := range domains {
		if d != nil && d.Zone != nil {
			byZone[d.Zone.ID] = d
			zoneIDs = append(zoneIDs, d.Zone.ID)
		}
	}

	perDomain := make(map[string][]cosmoflare.RedirectProbeResult, len(zoneIDs))
	var dests []string
	for _, zoneID := range zoneIDs {
		detail, err := svc.GetDetail(ctx, zoneID)
		if err != nil {
			continue // partial-failure: domains we cannot detail keep no verdict
		}
		for _, r := range detail.Redirects {
			if r.Destination != "" {
				dests = append(dests, r.Destination)
				perDomain[zoneID] = append(perDomain[zoneID], cosmoflare.RedirectProbeResult{Destination: r.Destination})
			}
		}
	}

	results := cosmoflare.NewRedirectProber().ProbeAll(ctx, dests)
	for zoneID, rs := range perDomain {
		withStatus := make([]cosmoflare.RedirectProbeResult, 0, len(rs))
		for _, r := range rs {
			if res, ok := results[r.Destination]; ok {
				withStatus = append(withStatus, res)
			}
		}
		if d := byZone[zoneID]; d != nil {
			d.RedirectIssue = cosmoflare.ClassifyRedirectIssues(withStatus)
		}
	}
	return nil
}
