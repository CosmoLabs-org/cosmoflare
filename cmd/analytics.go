package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	analyticsBucket string
	analyticsPeriod string
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Show R2 usage analytics",
	Long: `Show R2 usage statistics and analytics.

Displays bucket sizes, object counts, and storage distribution.
If --bucket is specified, shows detailed stats for that bucket.

Note: The --period flag is advisory. Real Cloudflare Analytics API
integration is planned for a future release. Currently stats are
computed from live bucket/object listings.

Examples:
  cosmoflare analytics
  cosmoflare analytics --bucket=my-bucket
  cosmoflare analytics --period=30d --json`,
	RunE: runAnalytics,
}

func init() {
	rootCmd.AddCommand(analyticsCmd)
	analyticsCmd.Flags().StringVar(&analyticsBucket, "bucket", "", "Specific bucket to analyze")
	analyticsCmd.Flags().StringVar(&analyticsPeriod, "period", "7d", "Time period (advisory)")
	analyticsCmd.AddCommand(analyticsKVCmd)
	analyticsKVCmd.Flags().IntVar(&analyticsKVWindow, "window", 7, "Look-back window in days (1-90)")
}

// --- analytics kv: per-namespace KV operation totals (FEAT-069) ---

var (
	analyticsKVWindow int
	analyticsKVJSON   bool
)

var (
	analyticsKVJSONOut io.Writer = os.Stdout
	analyticsKVTextOut io.Writer = os.Stdout
)

// analyticsKVCollectFn is the collection seam (usageCollectFn pattern):
// production queries the real GraphQL analytics dataset; tests stub it.
var analyticsKVCollectFn = func(windowDays int, now time.Time) ([]cosmoflare.KVOperationsSummary, error) {
	if AccountID == "" || APIToken == "" {
		return nil, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	w := cosmoflare.AnalyticsWindow{Start: now.Add(-time.Duration(windowDays) * 24 * time.Hour).UTC(), End: now.UTC()}
	return analytics.KVOperations(context.Background(), w)
}

var analyticsKVCmd = &cobra.Command{
	Use:   "kv",
	Short: "Show per-namespace KV operation totals",
	Long: `Show per-namespace KV operation totals from real Cloudflare Analytics.

Queries the kvOperationsAdaptiveGroups GraphQL dataset (one batched query)
and reports each namespace's total operations over the window, sorted by
traffic. The API exposes only a combined request count — there is no
reads/writes split on this dataset.

Flags:
  --window   Look-back window in days (default 7, max 90)
  --json     Machine-readable envelope

Examples:
  cosmoflare analytics kv                # per-namespace totals, last 7 days
  cosmoflare analytics kv --window 30    # a month of traffic
  cosmoflare analytics kv --json | jq -r '.data.namespaces[0]'`,
	RunE: runAnalyticsKV,
}

func runAnalyticsKV(cmd *cobra.Command, args []string) error {
	if analyticsKVWindow < 1 || analyticsKVWindow > 90 {
		return fmt.Errorf("--window must be 1-90 days, got %d", analyticsKVWindow)
	}
	rows, err := analyticsKVCollectFn(analyticsKVWindow, time.Now())
	if err != nil {
		return err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Requests > rows[j].Requests })

	if analyticsKVJSON || JSONOutput {
		enc := json.NewEncoder(analyticsKVJSONOut)
		var total uint64
		for _, r := range rows {
			total += r.Requests
		}
		return enc.Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"window_days":    analyticsKVWindow,
				"total_requests": total,
				"namespaces":     rows,
			},
		})
	}
	return writeAnalyticsKVTable(rows)
}

func writeAnalyticsKVTable(rows []cosmoflare.KVOperationsSummary) error {
	w := analyticsKVTextOut
	if len(rows) == 0 {
		fmt.Fprintf(w, "No KV operations in the last %d day(s) — every namespace is idle.\n", analyticsKVWindow)
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE ID\tOPERATIONS")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%d\n", r.NamespaceID, r.Requests)
	}
	tw.Flush()
	return nil
}

type BucketStat struct {
	Name        string  `json:"name"`
	ObjectCount int64   `json:"object_count"`
	TotalSize   int64   `json:"total_size"`
	Percentage  float64 `json:"percentage,omitempty"`
}

type AnalyticsResult struct {
	Buckets []BucketStat `json:"buckets"`
	Total   BucketStat   `json:"total"`
	Period  string       `json:"period"`
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	printInfo("Gathering R2 analytics (period: %s)...", analyticsPeriod)

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	if analyticsBucket != "" {
		return runSingleBucketAnalytics(client)
	}
	return runAllBucketsAnalytics(client)
}

func runSingleBucketAnalytics(client cosmoflare.R2Client) error {
	result, err := client.ListObjects(context.Background(), analyticsBucket, "", "", 0, "")
	if err != nil {
		return outErr("failed to list objects", err)
	}

	var totalSize int64
	for _, obj := range result.Items {
		totalSize += obj.Size
	}

	var avgSize int64
	if len(result.Items) > 0 {
		avgSize = totalSize / int64(len(result.Items))
	}

	stat := BucketStat{
		Name:        analyticsBucket,
		ObjectCount: int64(len(result.Items)),
		TotalSize:   totalSize,
	}

	return outPayload("Analytics for bucket", func() any {
		return map[string]interface{}{
			"bucket":       stat.Name,
			"object_count": stat.ObjectCount,
			"total_size":   stat.TotalSize,
			"avg_size":     avgSize,
			"period":       analyticsPeriod,
		}
	}, func() {
		fmt.Printf("Bucket: %s\n", stat.Name)
		fmt.Printf("Objects: %d\n", stat.ObjectCount)
		fmt.Printf("Total Size: %s\n", utils.FormatBytes(stat.TotalSize))
		if stat.ObjectCount > 0 {
			fmt.Printf("Avg Object Size: %s\n", utils.FormatBytes(avgSize))
		}
	})
}

func runAllBucketsAnalytics(client cosmoflare.R2Client) error {
	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		return outErr("failed to list buckets", err)
	}

	stats := make([]BucketStat, 0, len(buckets))
	var grandTotal int64

	for _, b := range buckets {
		result, err := client.ListObjects(context.Background(), b.Name, "", "", 0, "")
		if err != nil {
			printInfo("Skipping bucket %s: %v", b.Name, err)
			stats = append(stats, BucketStat{Name: b.Name})
			continue
		}

		var size int64
		for _, obj := range result.Items {
			size += obj.Size
		}
		grandTotal += size

		stats = append(stats, BucketStat{
			Name:        b.Name,
			ObjectCount: int64(len(result.Items)),
			TotalSize:   size,
		})
	}

	// Calculate percentages
	for i := range stats {
		if grandTotal > 0 {
			stats[i].Percentage = float64(stats[i].TotalSize) / float64(grandTotal) * 100
		}
	}

	return outPayload("Analytics summary", func() any {
		return AnalyticsResult{
			Buckets: stats,
			Total:   BucketStat{Name: "TOTAL", ObjectCount: int64(len(buckets)), TotalSize: grandTotal},
			Period:  analyticsPeriod,
		}
	}, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "BUCKET\tOBJECTS\tSIZE\t% OF TOTAL")
		for _, s := range stats {
			pct := fmt.Sprintf("%.1f%%", s.Percentage)
			if grandTotal == 0 {
				pct = "N/A"
			}
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", s.Name, s.ObjectCount, utils.FormatBytes(s.TotalSize), pct)
		}
		fmt.Fprintf(w, "TOTAL\t%d\t%s\t100%%\n", len(buckets), utils.FormatBytes(grandTotal))
		w.Flush()
	})
}
