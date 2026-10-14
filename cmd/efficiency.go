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

// Efficiency plan/JSON flags (bound in init).
var (
	efficiencyWindowDays int
	efficiencyJSON       bool
)

// Output seams for tests.
var (
	efficiencyJSONOut io.Writer = os.Stdout
	efficiencyTextOut io.Writer = os.Stdout
)

// efficiencyCollectFn is the collection seam (usageCollectFn pattern):
// production reads the configured account's real analytics; tests stub it.
// Workers+KV are always audited; if no R2 client is available the
// empty-bucket check is skipped rather than failing the audit.
var efficiencyCollectFn = func(windowDays int, now time.Time) (*cosmoflare.EfficiencyReport, error) {
	if AccountID == "" || APIToken == "" {
		return nil, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	workers, err := cosmoflare.NewWorkerServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}
	kv, err := cosmoflare.NewKVServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV service: %w", err)
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	w := cosmoflare.AnalyticsWindow{Start: now.Add(-time.Duration(windowDays) * 24 * time.Hour).UTC(), End: now.UTC()}
	r2, r2Err := getR2ClientForStatus()
	if r2Err != nil {
		return cosmoflare.NewEfficiencyService(workers, kv, analytics).Report(context.Background(), w)
	}
	return cosmoflare.NewEfficiencyServiceWithBuckets(workers, kv, analytics, r2).Report(context.Background(), w)
}

var efficiencyCmd = &cobra.Command{
	Use:   "efficiency",
	Short: "Find idle Cloudflare resources wasting your account",
	Long: `Find idle Cloudflare resources wasting your account.

Checks the whole account against real Cloudflare Analytics numbers and
reports resources that do nothing:

  idle workers          0 requests over the window (per-script GraphQL sums)
  dead KV namespaces    0 reads AND 0 writes over the window
  empty R2 buckets      0 objects at window end

Every finding carries its evidence (counts and the window it was measured
over) and an actionable fix. Numbers are always measured from the API —
never estimated or modeled.

Flags:
  --window   Look-back window in days (default 7, max 90)
  --json     Machine-readable envelope

Examples:
  cosmoflare efficiency                # audit the account for waste
  cosmoflare efficiency --window 30    # a month of patience before calling something idle
  cosmoflare efficiency --json | jq '.data.findings[] | select(.check=="idle-worker")'`,
	RunE: runEfficiency,
}

func init() {
	rootCmd.AddCommand(efficiencyCmd)
	efficiencyCmd.Flags().IntVar(&efficiencyWindowDays, "window", 7, "Look-back window in days (1-90)")
	efficiencyCmd.Flags().BoolVar(&efficiencyJSON, "json", false, "Output JSON envelope")
}

func runEfficiency(cmd *cobra.Command, args []string) error {
	if efficiencyWindowDays < 1 || efficiencyWindowDays > 90 {
		return fmt.Errorf("--window must be 1-90 days, got %d", efficiencyWindowDays)
	}
	rep, err := efficiencyCollectFn(efficiencyWindowDays, time.Now())
	if err != nil {
		return err
	}
	if efficiencyJSON {
		enc := json.NewEncoder(efficiencyJSONOut)
		return enc.Encode(map[string]interface{}{"status": "success", "data": rep})
	}
	return writeEfficiencyTable(rep)
}

func writeEfficiencyTable(rep *cosmoflare.EfficiencyReport) error {
	w := efficiencyTextOut
	fmt.Fprintf(w, "Efficiency audit — window %s → %s\n\n",
		rep.WindowStart.Format("Jan 02"), rep.WindowEnd.Format("Jan 02"))
	if len(rep.Findings) == 0 {
		fmt.Fprintf(w, "No waste found — checked %d workers, %d KV namespaces, %d R2 buckets.\n",
			rep.WorkersChecked, rep.KVNamespacesChecked, rep.BucketsChecked)
		return nil
	}
	fmt.Fprintf(w, "%-40s %-20s %-8s  %s\n", "RESOURCE", "CHECK", "SEV", "EVIDENCE")
	for _, f := range rep.Findings {
		sev := "⚠"
		if f.Severity == "info" {
			sev = "ℹ"
		}
		fmt.Fprintf(w, "%-40s %-20s %-8s  %s\n      fix: %s\n", f.Resource, f.Check, sev, f.Evidence, f.Fix)
	}
	fmt.Fprintf(w, "\nChecked %d workers, %d KV namespaces, %d R2 buckets — %d finding(s).\n",
		rep.WorkersChecked, rep.KVNamespacesChecked, rep.BucketsChecked, len(rep.Findings))
	return nil
}
