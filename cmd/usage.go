package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// usage plan/anchor flags (bound in init).
var (
	usagePlan      string
	usageAnchorDay int
	usageJSON      bool
)

// Output seams for tests.
var (
	usageJSONOut io.Writer = os.Stdout
	usageTextOut io.Writer = os.Stdout
)

// usageCollectFn is the collection seam (collectWatchMetricsFn pattern):
// production reads the configured account's analytics; tests stub it.
var usageCollectFn = func(ctx context.Context, plan string, anchor int, now time.Time) (*cosmoflare.UsageSnapshot, error) {
	if AccountID == "" || APIToken == "" {
		return nil, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	return cosmoflare.CollectUsage(ctx, analytics, plan, anchor, now)
}

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Monthly usage against plan allowances with cycle pacing",
	Long: `Monthly usage against plan allowances with cycle pacing.

Shows every usage dimension (Workers requests and CPU-ms, Durable Object
requests and duration, R2 storage and Class A/B operations) with its
cycle-to-date usage, the plan allowance, percent consumed, and the linear
projection to cycle end. A dimension projected past 100% will overrun the
cycle at the current burn rate.

The billing cycle defaults to the calendar month (UTC). Anniversary-billed
accounts pass --anchor-day N (the day of the month the cycle resets).

Flags:
  --plan        free | paid | enterprise (default paid; resolves allowance tiers)
  --anchor-day  Billing cycle reset day 1-31 (default 0 = calendar month)
  --json        Machine-readable envelope

Examples:
  cosmoflare usage
  cosmoflare usage --anchor-day 15
  cosmoflare usage --plan free --json`,
	RunE: runUsage,
}

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.Flags().StringVar(&usagePlan, "plan", "paid", "Plan tier for allowance resolution: free, paid, or enterprise")
	usageCmd.Flags().IntVar(&usageAnchorDay, "anchor-day", 0, "Billing cycle reset day (1-31); 0 = calendar month")
	usageCmd.Flags().BoolVar(&usageJSON, "json", false, "Output JSON envelope")
}

func runUsage(cmd *cobra.Command, args []string) error {
	if usageAnchorDay < 0 || usageAnchorDay > 31 {
		return fmt.Errorf("--anchor-day must be 0-31, got %d", usageAnchorDay)
	}
	switch usagePlan {
	case "free", "paid", "enterprise":
	default:
		return fmt.Errorf("--plan must be free, paid, or enterprise, got %q", usagePlan)
	}
	snap, err := usageCollectFn(context.Background(), usagePlan, usageAnchorDay, time.Now().UTC())
	if err != nil {
		return err
	}
	if usageJSON {
		return writeUsageJSON(snap)
	}
	return writeUsageTable(snap)
}

func writeUsageJSON(snap *cosmoflare.UsageSnapshot) error {
	payload := map[string]interface{}{"status": "success", "data": snap}
	enc := json.NewEncoder(usageJSONOut)
	return enc.Encode(payload)
}

func writeUsageTable(snap *cosmoflare.UsageSnapshot) error {
	w := usageTextOut
	fmt.Fprintf(w, "Monthly usage — cycle %s → %s (%.0f of %.0f days elapsed)\n\n",
		snap.CycleStart.Format("Jan 02"), snap.CycleEnd.Format("Jan 02"), snap.DaysElapsed, snap.DaysTotal)
	fmt.Fprintf(w, "%-38s %14s %14s %8s %10s  %s\n", "DIMENSION", "USED", "LIMIT", "PCT", "PROJECTED", "STATUS")
	for _, d := range snap.Dimensions {
		used := formatUsageValue(d.Used, d.Unit)
		status := usageStatus(d)
		var limit, pct, proj string
		if d.Limit > 0 {
			limit = formatUsageValue(d.Limit, d.Unit)
			pct = fmt.Sprintf("%.1f%%", d.Pct)
			proj = fmt.Sprintf("%.1f%%", d.ProjectedPct)
		} else {
			limit = "limit unknown"
		}
		fmt.Fprintf(w, "%-38s %14s %14s %8s %10s  %s\n", d.Name, used, limit, pct, proj, status)
	}
	fmt.Fprintf(w, "\nProjection is linear (usage to date ÷ days elapsed × cycle length). ")
	fmt.Fprintf(w, "Set usage-pct / usage-projected-pct alert rules to page before an overrun.\n")
	return nil
}

func usageStatus(d cosmoflare.UsageDimension) string {
	if d.Limit == 0 {
		return "unknown"
	}
	if d.Pct >= 100 {
		return "over"
	}
	if d.ProjectedPct > 100 {
		return "projected-over"
	}
	return "ok"
}

// formatUsageValue renders large counts compactly (2.9B, 8.0M, 400.0k) via
// the shared cosmoflare.HumanCount tiers, and byte-ish units as plain floats.
func formatUsageValue(v float64, unit string) string {
	switch unit {
	case "gb_month_per_month", "gb_seconds_per_month":
		return fmt.Sprintf("%.1f", v)
	}
	if v >= 1e3 {
		return cosmoflare.HumanCount(v)
	}
	return fmt.Sprintf("%.0f", v)
}
