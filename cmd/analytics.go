package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
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
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if analyticsBucket != "" {
		return runSingleBucketAnalytics(client)
	}
	return runAllBucketsAnalytics(client)
}

func runSingleBucketAnalytics(client cosmoflare.R2Client) error {
	result, err := client.ListObjects(context.Background(), analyticsBucket, "", "", 0)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
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

	if JSONOutput {
		return printSuccessJSON("Analytics for bucket", map[string]interface{}{
			"bucket":       stat.Name,
			"object_count": stat.ObjectCount,
			"total_size":   stat.TotalSize,
			"avg_size":     avgSize,
			"period":       analyticsPeriod,
		})
	}

	fmt.Printf("Bucket: %s\n", stat.Name)
	fmt.Printf("Objects: %d\n", stat.ObjectCount)
	fmt.Printf("Total Size: %s\n", utils.FormatBytes(stat.TotalSize))
	if stat.ObjectCount > 0 {
		fmt.Printf("Avg Object Size: %s\n", utils.FormatBytes(avgSize))
	}
	return nil
}

func runAllBucketsAnalytics(client cosmoflare.R2Client) error {
	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	stats := make([]BucketStat, 0, len(buckets))
	var grandTotal int64

	for _, b := range buckets {
		result, err := client.ListObjects(context.Background(), b.Name, "", "", 0)
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

	if JSONOutput {
		return printSuccessJSON("Analytics summary", AnalyticsResult{
			Buckets: stats,
			Total:   BucketStat{Name: "TOTAL", ObjectCount: int64(len(buckets)), TotalSize: grandTotal},
			Period:  analyticsPeriod,
		})
	}

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

	return nil
}
