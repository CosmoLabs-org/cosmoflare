package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show at-a-glance dashboard of your Cloudflare infrastructure",
	Long: `Infrastructure status dashboard showing resource counts and health.

Queries multiple Cloudflare services in parallel and displays a compact
summary of your account's resources: zones, workers, KV namespaces, R2
buckets, DNS records, and SSL status.

Examples:
  cosmoflare status              # Quick dashboard
  cosmoflare status --json       # Machine-readable output
  cosmoflare status --verbose    # Include per-zone breakdown`,
	RunE: runStatus,
}

var statusVerbose bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusVerbose, "verbose", false, "Show per-zone breakdown")
}

type StatusReport struct {
	Timestamp    string              `json:"timestamp"`
	AccountID    string              `json:"account_id"`
	Zones        StatusSection       `json:"zones"`
	Workers      StatusSection       `json:"workers"`
	KV           StatusSection       `json:"kv_namespaces"`
	Buckets      StatusSection       `json:"r2_buckets"`
	DNS          StatusSection       `json:"dns_records"`
	SSL          SSLStatusSection    `json:"ssl"`
	QueryTimeMs  int64               `json:"query_time_ms"`
	Errors       []string            `json:"errors,omitempty"`
}

type StatusSection struct {
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

type SSLStatusSection struct {
	Active   int    `json:"active"`
	Pending  int    `json:"pending"`
	Inactive int    `json:"inactive"`
	Error    string `json:"error,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	start := time.Now()
	ctx := context.Background()
	report := &StatusReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	report.AccountID = AccountID

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errors []string

	addError := func(msg string) {
		mu.Lock()
		errors = append(errors, msg)
		mu.Unlock()
	}

	// Query zones
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc, err := getZoneService()
		if err != nil {
			addError(fmt.Sprintf("zones: %v", err))
			report.Zones.Error = err.Error()
			return
		}
		zones, err := svc.List(ctx)
		if err != nil {
			addError(fmt.Sprintf("zones: %v", err))
			report.Zones.Error = err.Error()
			return
		}
		report.Zones.Count = len(zones)

		for _, z := range zones {
			if z.Status == "active" {
				report.SSL.Active++
			} else {
				report.SSL.Inactive++
			}
		}
	}()

	// Query workers
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc, err := getWorkerService()
		if err != nil {
			addError(fmt.Sprintf("workers: %v", err))
			report.Workers.Error = err.Error()
			return
		}
		workers, err := svc.List(ctx)
		if err != nil {
			addError(fmt.Sprintf("workers: %v", err))
			report.Workers.Error = err.Error()
			return
		}
		report.Workers.Count = len(workers)
	}()

	// Query KV namespaces
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc, err := getKVService()
		if err != nil {
			addError(fmt.Sprintf("kv: %v", err))
			report.KV.Error = err.Error()
			return
		}
		namespaces, err := svc.ListNamespaces(ctx)
		if err != nil {
			addError(fmt.Sprintf("kv: %v", err))
			report.KV.Error = err.Error()
			return
		}
		report.KV.Count = len(namespaces)
	}()

	// Query R2 buckets
	wg.Add(1)
	go func() {
		defer wg.Done()
		client, err := getR2ClientForStatus()
		if err != nil {
			addError(fmt.Sprintf("r2: %v", err))
			report.Buckets.Error = err.Error()
			return
		}
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			addError(fmt.Sprintf("r2: %v", err))
			report.Buckets.Error = err.Error()
			return
		}
		report.Buckets.Count = len(buckets)
	}()

	wg.Wait()
	report.QueryTimeMs = time.Since(start).Milliseconds()
	report.Errors = errors

	// outResult is byte-identical to the raw json.Encoder branch it replaced:
	// Encoder with SetIndent("", "  ") + trailing newline == MarshalIndent + Println.
	return outResult(report, func() {
		printStatusDashboard(report)
	})
}

func printStatusDashboard(r *StatusReport) {
	fmt.Println()
	fmt.Println("  Cosmoflare Infrastructure Status")
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Println()

	printStatusLine("Zones", r.Zones.Count, r.Zones.Error)
	printStatusLine("Workers", r.Workers.Count, r.Workers.Error)
	printStatusLine("KV Namespaces", r.KV.Count, r.KV.Error)
	printStatusLine("R2 Buckets", r.Buckets.Count, r.Buckets.Error)
	fmt.Println()

	if r.SSL.Active+r.SSL.Pending+r.SSL.Inactive > 0 {
		fmt.Printf("  SSL: %d active, %d pending, %d inactive\n",
			r.SSL.Active, r.SSL.Pending, r.SSL.Inactive)
		fmt.Println()
	}

	if len(r.Errors) > 0 {
		fmt.Printf("  Errors: %d service(s) unreachable\n", len(r.Errors))
	}

	fmt.Printf("  Query time: %dms\n", r.QueryTimeMs)
	fmt.Println()
}

func printStatusLine(label string, count int, errMsg string) {
	if errMsg != "" {
		fmt.Printf("  %-15s  --  (error: %s)\n", label, errMsg)
	} else {
		fmt.Printf("  %-15s  %d\n", label, count)
	}
}

func getR2ClientForStatus() (cosmoflare.R2Client, error) {
	return getAPIClient()
}
