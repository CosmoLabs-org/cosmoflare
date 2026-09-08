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

var (
	metricsInterval time.Duration
	metricsWindow   time.Duration
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Real-time metrics dashboard for Cloudflare services",
	Long: `Launch a TUI dashboard showing live metrics for R2 storage, Workers
invocations, and KV operation rates.

The dashboard polls the Cloudflare API at a configurable interval and
displays usage data in a terminal-based interface.

With --json, prints one snapshot instead of the TUI: account counts plus
usage analytics over --window (R2 storage and operation volume, Workers
invocations, per-zone HTTP traffic). Sources that fail are reported in
"errors" — counts always print.

Flags:
  --interval   Polling interval (default: 30s, TUI only)
  --window     Usage analytics window (default: 24h, --json only)

Examples:
  cosmoflare metrics                      # Default 30s interval
  cosmoflare metrics --interval 10s       # Poll every 10 seconds
  cosmoflare metrics --json               # Counts + 24h usage as JSON
  cosmoflare metrics --json --window 7d   # Week-long usage window`,
	RunE: runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().DurationVar(&metricsInterval, "interval", 30*time.Second, "Polling interval")
	metricsCmd.Flags().DurationVar(&metricsWindow, "window", 24*time.Hour, "Usage analytics window (--json only)")
}

type metricsSnapshot struct {
	Timestamp      time.Time                              `json:"timestamp"`
	Window         string                                 `json:"window"`
	R2Buckets      int                                    `json:"r2_buckets"`
	R2TotalSize    int64                                  `json:"r2_total_size_bytes"`
	R2TotalObjects int64                                  `json:"r2_total_objects"`
	Workers        int                                    `json:"workers"`
	KVNamespaces   int                                    `json:"kv_namespaces"`
	R2Storage      []cosmoflare.R2BucketStorage           `json:"r2_storage,omitempty"`
	R2Operations   []cosmoflare.R2OperationCount          `json:"r2_operations,omitempty"`
	WorkersUsage   []cosmoflare.WorkersSummary            `json:"workers_usage,omitempty"`
	ZoneHTTP       map[string]*cosmoflare.ZoneHTTPSummary `json:"zone_http,omitempty"`
	Errors         map[string]string                      `json:"errors,omitempty"`
}

func (s *metricsSnapshot) recordError(source string, err error) {
	if s.Errors == nil {
		s.Errors = make(map[string]string)
	}
	s.Errors[source] = err.Error()
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	snap := metricsSnapshot{Timestamp: time.Now(), Window: metricsWindow.String()}
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	snap.R2Buckets = len(buckets)
	for _, b := range buckets {
		snap.R2TotalSize += b.Size
		snap.R2TotalObjects += b.ObjectCount
	}

	recordAnalytics := func(source string, fn func() error) {
		if err := fn(); err != nil {
			snap.recordError(source, err)
		}
	}

	if AccountID == "" || APIToken == "" {
		snap.recordError("analytics", fmt.Errorf("account ID and API token are required for usage analytics (run 'cosmoflare account' to configure)"))
	} else {
		analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
		w := cosmoflare.AnalyticsWindow{Start: time.Now().Add(-metricsWindow), End: time.Now()}

		recordAnalytics("r2_storage", func() error {
			data, err := analytics.R2Storage(ctx, w)
			snap.R2Storage = data
			return err
		})
		recordAnalytics("r2_operations", func() error {
			data, err := analytics.R2Operations(ctx, w)
			snap.R2Operations = data
			return err
		})
		recordAnalytics("workers_usage", func() error {
			data, err := analytics.Workers(ctx, w)
			snap.WorkersUsage = data
			return err
		})
		recordAnalytics("zone_http", func() error {
			return fetchZoneHTTP(ctx, analytics, w, &snap)
		})
	}

	out, _ := json.MarshalIndent(snap, "", "  ")
	fmt.Println(string(out))
	return nil
}

// fetchZoneHTTP fills snap.ZoneHTTP with per-zone traffic over the window,
// capped at 25 zones to bound query count. Per-zone failures are recorded
// individually so one bad zone does not hide the rest.
func fetchZoneHTTP(ctx context.Context, analytics *cosmoflare.AnalyticsService, w cosmoflare.AnalyticsWindow, snap *metricsSnapshot) error {
	zoneSvc, err := getZoneService()
	if err != nil {
		return err
	}
	zones, err := zoneSvc.List(ctx)
	if err != nil {
		return err
	}
	if len(zones) > 25 {
		zones = zones[:25]
	}
	snap.ZoneHTTP = make(map[string]*cosmoflare.ZoneHTTPSummary, len(zones))
	for _, z := range zones {
		summary, err := analytics.ZoneHTTP(ctx, z.ID, w)
		if err != nil {
			snap.recordError("zone:"+z.Name, err)
			continue
		}
		snap.ZoneHTTP[z.Name] = summary
	}
	return nil
}
