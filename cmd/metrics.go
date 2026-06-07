package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"

	"github.com/CosmoLabs-org/cosmoflare/internal/tui"
)

var metricsInterval time.Duration

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Real-time metrics dashboard for Cloudflare services",
	Long: `Launch a TUI dashboard showing live metrics for R2 storage, Workers
invocations, and KV operation rates.

The dashboard polls the Cloudflare API at a configurable interval and
displays usage data in a terminal-based interface.

Flags:
  --interval   Polling interval (default: 30s)

Examples:
  cosmoflare metrics                      # Default 30s interval
  cosmoflare metrics --interval 10s       # Poll every 10 seconds
  cosmoflare metrics --json               # Print metrics as JSON (no TUI)`,
	RunE: runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().DurationVar(&metricsInterval, "interval", 30*time.Second, "Polling interval")
}

type metricsSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	R2Buckets      int       `json:"r2_buckets"`
	R2TotalSize    int64     `json:"r2_total_size_bytes"`
	R2TotalObjects int64     `json:"r2_total_objects"`
	Workers        int       `json:"workers"`
	KVNamespaces   int       `json:"kv_namespaces"`
}

func runMetrics(cmd *cobra.Command, args []string) error {
	if JSONOutput {
		client, err := cosmoflare.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		return runMetricsJSON(client)
	}

	return tui.RunDashboardWithConfig(tui.DashboardConfig{
		Interval:     metricsInterval,
		StartSection: int(tui.SectionMonitoring),
	})
}

func runMetricsJSON(client cosmoflare.R2Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	snap := metricsSnapshot{Timestamp: time.Now()}
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	snap.R2Buckets = len(buckets)
	for _, b := range buckets {
		snap.R2TotalSize += b.Size
		snap.R2TotalObjects += b.ObjectCount
	}

	out, _ := json.MarshalIndent(snap, "", "  ")
	fmt.Println(string(out))
	return nil
}
