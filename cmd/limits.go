package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	limitsBucket string
	limitsPlan   string
)

var limitsCmd = &cobra.Command{
	Use:   "limits",
	Short: "Show plan-limit proximity for the account",
	Long: `Show how close the account is to its Cloudflare plan limits.

Rows cover Workers (scripts, daily requests on the free plan), R2 (buckets,
per-bucket custom domains with --bucket), zones, and per-zone DNS record
quotas. Usage comes from live API counts; limit values come from documented
plan tables (workers limits, r2 limits) or the live DNS quota API.

Workers plan tier resolution order:
  1. Subscriptions API (auto — needs Billing Read)
  2. --plan flag (free|paid)
  3. workers_plan in .cosmoflare.yaml (config)
  4. unknown — plan-dependent rows show usage without a percent

Every source is independent: a failing source prints a warning after the
table and the rest of the snapshot is unaffected. Exit code is non-zero only
when every source fails.

Examples:
  cosmoflare limits
  cosmoflare limits --bucket=my-bucket
  cosmoflare limits --plan free --json`,
	RunE: runLimits,
}

func init() {
	rootCmd.AddCommand(limitsCmd)
	limitsCmd.Flags().StringVar(&limitsBucket, "bucket", "", "Include per-bucket rows for this bucket (custom domains)")
	limitsCmd.Flags().StringVar(&limitsPlan, "plan", "", "Workers plan tier override: free|paid")
}

// sortRowsByPercent orders rows by proximity to limit, descending. Rows
// without a percent sink to the end, keeping insertion order among equals.
func sortRowsByPercent(rows []cosmoflare.LimitRow) []cosmoflare.LimitRow {
	out := make([]cosmoflare.LimitRow, len(rows))
	copy(out, rows)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Percent > out[j].Percent })
	return out
}

func runLimits(cmd *cobra.Command, args []string) error {
	printInfo("Collecting plan limits...")

	cfg, cfgErr := cosmoflare.LoadProjectConfig(".")
	configPlan := ""
	if cfgErr == nil {
		configPlan = cfg.WorkersPlan
	}

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}
	workersSvc, err := cosmoflare.NewWorkerServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return outErr("failed to create workers service", err)
	}
	zonesSvc, err := cosmoflare.NewZoneServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return outErr("failed to create zone service", err)
	}
	domainsSvc := cosmoflare.NewBucketDomainService(AccountID, APIToken)
	analyticsSvc := cosmoflare.NewAnalyticsService(AccountID, APIToken)

	svc := cosmoflare.NewLimitsService(AccountID, APIToken,
		cosmoflare.WithLimitsR2(client),
		cosmoflare.WithLimitsWorkers(workersSvc),
		cosmoflare.WithLimitsZones(zonesSvc),
		cosmoflare.WithLimitsDomains(domainsSvc),
		cosmoflare.WithLimitsAnalytics(analyticsSvc),
		cosmoflare.WithLimitsConfigPlan(configPlan),
		cosmoflare.WithLimitsFlagPlan(limitsPlan),
	)

	snap, err := svc.Snapshot(context.Background(), limitsBucket)
	if err != nil {
		return outErr("failed to collect limits", err)
	}

	return outPayload("Plan limits", func() any {
		return snap
	}, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RESOURCE\tSCOPE\tUSED\tLIMIT\tUSED%\tSOURCE")
		for _, row := range sortRowsByPercent(snap.Rows) {
			limit := "unlimited"
			if row.LimitSource == "unknown" {
				limit = "unknown" // limit exists but is unresolved (e.g. unknown plan tier)
			} else if row.Limit > 0 {
				limit = fmt.Sprintf("%d", row.Limit)
			}
			pct := "—"
			if row.Limit > 0 { // 0.0% is a real value whenever a limit is known
				pct = fmt.Sprintf("%.1f%%", row.Percent)
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
				row.Resource, row.Scope, row.Used, limit, pct, row.LimitSource)
		}
		w.Flush()

		fmt.Printf("\nWorkers plan: %s (resolved via %s)\n", snap.WorkersPlan, snap.PlanSource)
		for _, se := range snap.Sources {
			printInfo("Source %s failed: %s", se.Source, se.Err)
		}
		if cfgErr != nil {
			printInfo("No project config found (%v) — workers_plan fallback unavailable", cfgErr)
		}
	})
}
