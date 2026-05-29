package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

var (
	costPeriod string
	costFormat string
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Estimate monthly Cloudflare costs",
	Long: `Estimate monthly Cloudflare costs based on current usage patterns.

Calculates approximate costs for R2 storage, Workers, and KV based on
Cloudflare published pricing. Provides per-service breakdowns and totals.

Subcommands:
  r2        R2 storage costs (storage GB, Class A/B operations)
  workers   Workers costs (requests, CPU time)
  kv        KV costs (reads, writes, storage)
  detail    Itemized breakdown by service

Flags:
  --period   Analysis period: 7d, 30d, or 90d (default 30d)
  --format   Output format: table, json, or csv (default table)

NOTE: Estimates are approximate. Actual costs depend on your Cloudflare
plan, contract terms, and real-time usage patterns.

Examples:
  cosmoflare cost                     # Show estimated monthly costs for all services
  cosmoflare cost --period 7d         # Estimate based on last 7 days
  cosmoflare cost --format csv        # Output as CSV
  cosmoflare cost --json              # JSON output (uses global --json flag)
  cosmoflare cost r2                  # R2 storage costs only
  cosmoflare cost workers             # Workers costs only
  cosmoflare cost kv                  # KV costs only
  cosmoflare cost detail              # Itemized per-service breakdown`,
	RunE: runCost,
}

var costR2Cmd = &cobra.Command{
	Use:   "r2",
	Short: "Estimate R2 storage costs",
	Long: `Estimate R2 storage costs based on current bucket sizes and operation counts.

Pricing (Cloudflare published rates):
  Storage:      $0.015/GB/month
  Class A ops:  $4.50 per million (PUT, POST, LIST)
  Class B ops:  $0.36 per million (GET, HEAD)

Examples:
  cosmoflare cost r2                  # R2 cost estimate
  cosmoflare cost r2 --period 7d     # Based on 7-day usage
  cosmoflare cost r2 --json          # JSON output`,
	RunE: runCostR2,
}

var costWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Estimate Workers costs",
	Long: `Estimate Workers costs based on request counts and CPU time.

Pricing (Cloudflare published rates):
  Requests:  $0.50 per million (after 100K free tier)
  CPU time:  Included in request pricing on paid plans

Examples:
  cosmoflare cost workers             # Workers cost estimate
  cosmoflare cost workers --json     # JSON output`,
	RunE: runCostWorkers,
}

var costKVCmd = &cobra.Command{
	Use:   "kv",
	Short: "Estimate KV costs",
	Long: `Estimate Workers KV costs based on read/write operations and storage.

Pricing (Cloudflare published rates):
  Reads:    $0.50 per million
  Writes:   $5.00 per million
  Storage:  $0.50/GB/month

Examples:
  cosmoflare cost kv                  # KV cost estimate
  cosmoflare cost kv --json          # JSON output`,
	RunE: runCostKV,
}

var costDetailCmd = &cobra.Command{
	Use:   "detail",
	Short: "Itemized cost breakdown by service",
	Long: `Show an itemized cost breakdown across all Cloudflare services with
per-line-item detail including usage quantities and unit prices.

Examples:
  cosmoflare cost detail              # Full itemized breakdown
  cosmoflare cost detail --json      # JSON output
  cosmoflare cost detail --format csv # CSV for spreadsheet import`,
	RunE: runCostDetail,
}

func init() {
	rootCmd.AddCommand(costCmd)

	costCmd.AddCommand(costR2Cmd)
	costCmd.AddCommand(costWorkersCmd)
	costCmd.AddCommand(costKVCmd)
	costCmd.AddCommand(costDetailCmd)

	// Persistent flags (inherited by subcommands)
	costCmd.PersistentFlags().StringVar(&costPeriod, "period", "30d", "Analysis period (7d, 30d, 90d)")
	costCmd.PersistentFlags().StringVar(&costFormat, "format", "table", "Output format (table, json, csv)")
}

// getCostService creates a CostService from global credentials.
func getCostService() (*r2go2.CostService, error) {
	return r2go2.NewCostService(AccountID, APIToken)
}

// validatePeriod checks that the period flag value is valid.
func validatePeriod(period string) error {
	switch period {
	case "7d", "30d", "90d":
		return nil
	default:
		return fmt.Errorf("invalid period %q: must be 7d, 30d, or 90d", period)
	}
}

// validateCostFormat checks that the format flag value is valid.
func validateCostFormat(format string) error {
	switch format {
	case "table", "json", "csv":
		return nil
	default:
		return fmt.Errorf("invalid format %q: must be table, json, or csv", format)
	}
}

func runCost(cmd *cobra.Command, args []string) error {
	if err := validatePeriod(costPeriod); err != nil {
		return err
	}
	if err := validateCostFormat(costFormat); err != nil {
		return err
	}

	svc, err := getCostService()
	if err != nil {
		return fmt.Errorf("failed to create cost service: %w", err)
	}

	// Estimate with placeholder usage (real API-based fetching is a future enhancement)
	r2Usage := r2go2.R2Usage{}
	workersUsage := r2go2.WorkersUsage{}
	kvUsage := r2go2.KVUsage{}

	total := svc.EstimateTotal(r2Usage, workersUsage, kvUsage)
	total.Period = costPeriod

	if JSONOutput || costFormat == "json" {
		return printJSON(total)
	}

	if costFormat == "csv" {
		return printCostCSV(total)
	}

	printCostSummary(total)
	return nil
}

func runCostR2(cmd *cobra.Command, args []string) error {
	if err := validatePeriod(costPeriod); err != nil {
		return err
	}
	if err := validateCostFormat(costFormat); err != nil {
		return err
	}

	svc, err := getCostService()
	if err != nil {
		return fmt.Errorf("failed to create cost service: %w", err)
	}

	usage := r2go2.R2Usage{}
	est := svc.EstimateR2Cost(usage)

	if JSONOutput || costFormat == "json" {
		return printJSON(est)
	}

	if costFormat == "csv" {
		return printR2CostCSV(est)
	}

	printR2CostTable(est)
	return nil
}

func runCostWorkers(cmd *cobra.Command, args []string) error {
	if err := validatePeriod(costPeriod); err != nil {
		return err
	}
	if err := validateCostFormat(costFormat); err != nil {
		return err
	}

	svc, err := getCostService()
	if err != nil {
		return fmt.Errorf("failed to create cost service: %w", err)
	}

	usage := r2go2.WorkersUsage{}
	est := svc.EstimateWorkersCost(usage)

	if JSONOutput || costFormat == "json" {
		return printJSON(est)
	}

	if costFormat == "csv" {
		return printWorkersCostCSV(est)
	}

	printWorkersCostTable(est)
	return nil
}

func runCostKV(cmd *cobra.Command, args []string) error {
	if err := validatePeriod(costPeriod); err != nil {
		return err
	}
	if err := validateCostFormat(costFormat); err != nil {
		return err
	}

	svc, err := getCostService()
	if err != nil {
		return fmt.Errorf("failed to create cost service: %w", err)
	}

	usage := r2go2.KVUsage{}
	est := svc.EstimateKVCost(usage)

	if JSONOutput || costFormat == "json" {
		return printJSON(est)
	}

	if costFormat == "csv" {
		return printKVCostCSV(est)
	}

	printKVCostTable(est)
	return nil
}

func runCostDetail(cmd *cobra.Command, args []string) error {
	if err := validatePeriod(costPeriod); err != nil {
		return err
	}
	if err := validateCostFormat(costFormat); err != nil {
		return err
	}

	svc, err := getCostService()
	if err != nil {
		return fmt.Errorf("failed to create cost service: %w", err)
	}

	r2Usage := r2go2.R2Usage{}
	workersUsage := r2go2.WorkersUsage{}
	kvUsage := r2go2.KVUsage{}

	total := svc.EstimateTotal(r2Usage, workersUsage, kvUsage)
	total.Period = costPeriod

	if JSONOutput || costFormat == "json" {
		return printJSON(total)
	}

	if costFormat == "csv" {
		return printDetailCSV(total)
	}

	printDetailTable(total)
	return nil
}

// ---------------------------------------------------------------------------
// Table output
// ---------------------------------------------------------------------------

func printCostSummary(total *r2go2.TotalCostEstimate) {
	fmt.Printf("\nCloudflare Cost Estimate (period: %s)\n", total.Period)
	fmt.Println("============================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tESTIMATED COST")
	fmt.Fprintf(w, "R2 Storage\t$%.2f\n", total.R2.TotalCost)
	fmt.Fprintf(w, "Workers\t$%.2f\n", total.Workers.TotalCost)
	fmt.Fprintf(w, "KV\t$%.2f\n", total.KV.TotalCost)
	fmt.Fprintln(w, "\t")
	fmt.Fprintf(w, "TOTAL\t$%.2f/mo\n", total.TotalMonthlyCost)
	w.Flush()

	fmt.Printf("\n%s\n", total.Disclaimer)
}

func printR2CostTable(est *r2go2.R2CostEstimate) {
	fmt.Println("\nR2 Storage Cost Estimate")
	fmt.Println("========================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ITEM\tUSAGE\tRATE\tCOST")
	fmt.Fprintf(w, "Storage\t%.2f GB\t$%.3f/GB\t$%.2f\n", est.Usage.StorageGB, r2go2.R2StoragePerGB, est.StorageCost)
	fmt.Fprintf(w, "Class A Ops\t%d\t$%.2f/M\t$%.2f\n", est.Usage.ClassAOps, r2go2.R2ClassAPerMillion, est.ClassACost)
	fmt.Fprintf(w, "Class B Ops\t%d\t$%.2f/M\t$%.2f\n", est.Usage.ClassBOps, r2go2.R2ClassBPerMillion, est.ClassBCost)
	fmt.Fprintln(w, "\t\t\t")
	fmt.Fprintf(w, "TOTAL\t\t\t$%.2f\n", est.TotalCost)
	w.Flush()
}

func printWorkersCostTable(est *r2go2.WorkersCostEstimate) {
	fmt.Println("\nWorkers Cost Estimate")
	fmt.Println("=====================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ITEM\tUSAGE\tRATE\tCOST")
	fmt.Fprintf(w, "Requests\t%d\t$%.2f/M\t$%.2f\n", est.Usage.Requests, r2go2.WorkersRequestsPerMillion, est.RequestsCost)
	fmt.Fprintf(w, "Free Tier\t%d\t\t-$0.00\n", r2go2.WorkersFreeRequests)
	fmt.Fprintf(w, "Billable\t%d\t\t\n", est.BillableRequests)
	fmt.Fprintln(w, "\t\t\t")
	fmt.Fprintf(w, "TOTAL\t\t\t$%.2f\n", est.TotalCost)
	w.Flush()
}

func printKVCostTable(est *r2go2.KVCostEstimate) {
	fmt.Println("\nKV Cost Estimate")
	fmt.Println("=================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ITEM\tUSAGE\tRATE\tCOST")
	fmt.Fprintf(w, "Reads\t%d\t$%.2f/M\t$%.2f\n", est.Usage.Reads, r2go2.KVReadsPerMillion, est.ReadsCost)
	fmt.Fprintf(w, "Writes\t%d\t$%.2f/M\t$%.2f\n", est.Usage.Writes, r2go2.KVWritesPerMillion, est.WritesCost)
	fmt.Fprintf(w, "Storage\t%.2f GB\t$%.2f/GB\t$%.2f\n", est.Usage.StorageGB, r2go2.KVStoragePerGB, est.StorageCost)
	fmt.Fprintln(w, "\t\t\t")
	fmt.Fprintf(w, "TOTAL\t\t\t$%.2f\n", est.TotalCost)
	w.Flush()
}

func printDetailTable(total *r2go2.TotalCostEstimate) {
	fmt.Printf("\nItemized Cost Breakdown (period: %s)\n", total.Period)
	fmt.Println("============================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tITEM\tUSAGE\tRATE\tCOST")

	// R2
	fmt.Fprintf(w, "R2\tStorage\t%.2f GB\t$%.3f/GB\t$%.2f\n", total.R2.Usage.StorageGB, r2go2.R2StoragePerGB, total.R2.StorageCost)
	fmt.Fprintf(w, "R2\tClass A Ops\t%d\t$%.2f/M\t$%.2f\n", total.R2.Usage.ClassAOps, r2go2.R2ClassAPerMillion, total.R2.ClassACost)
	fmt.Fprintf(w, "R2\tClass B Ops\t%d\t$%.2f/M\t$%.2f\n", total.R2.Usage.ClassBOps, r2go2.R2ClassBPerMillion, total.R2.ClassBCost)

	// Workers
	fmt.Fprintf(w, "Workers\tRequests\t%d\t$%.2f/M\t$%.2f\n", total.Workers.Usage.Requests, r2go2.WorkersRequestsPerMillion, total.Workers.RequestsCost)

	// KV
	fmt.Fprintf(w, "KV\tReads\t%d\t$%.2f/M\t$%.2f\n", total.KV.Usage.Reads, r2go2.KVReadsPerMillion, total.KV.ReadsCost)
	fmt.Fprintf(w, "KV\tWrites\t%d\t$%.2f/M\t$%.2f\n", total.KV.Usage.Writes, r2go2.KVWritesPerMillion, total.KV.WritesCost)
	fmt.Fprintf(w, "KV\tStorage\t%.2f GB\t$%.2f/GB\t$%.2f\n", total.KV.Usage.StorageGB, r2go2.KVStoragePerGB, total.KV.StorageCost)

	fmt.Fprintln(w, "\t\t\t\t")
	fmt.Fprintf(w, "TOTAL\t\t\t\t$%.2f/mo\n", total.TotalMonthlyCost)
	w.Flush()

	fmt.Printf("\n%s\n", total.Disclaimer)
}

// ---------------------------------------------------------------------------
// CSV output
// ---------------------------------------------------------------------------

func printCostCSV(total *r2go2.TotalCostEstimate) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"service", "estimated_cost"})
	w.Write([]string{"R2 Storage", fmt.Sprintf("%.2f", total.R2.TotalCost)})
	w.Write([]string{"Workers", fmt.Sprintf("%.2f", total.Workers.TotalCost)})
	w.Write([]string{"KV", fmt.Sprintf("%.2f", total.KV.TotalCost)})
	w.Write([]string{"TOTAL", fmt.Sprintf("%.2f", total.TotalMonthlyCost)})

	return w.Error()
}

func printR2CostCSV(est *r2go2.R2CostEstimate) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"item", "usage", "rate", "cost"})
	w.Write([]string{"Storage", fmt.Sprintf("%.2f GB", est.Usage.StorageGB), fmt.Sprintf("$%.3f/GB", r2go2.R2StoragePerGB), fmt.Sprintf("%.2f", est.StorageCost)})
	w.Write([]string{"Class A Ops", fmt.Sprintf("%d", est.Usage.ClassAOps), fmt.Sprintf("$%.2f/M", r2go2.R2ClassAPerMillion), fmt.Sprintf("%.2f", est.ClassACost)})
	w.Write([]string{"Class B Ops", fmt.Sprintf("%d", est.Usage.ClassBOps), fmt.Sprintf("$%.2f/M", r2go2.R2ClassBPerMillion), fmt.Sprintf("%.2f", est.ClassBCost)})
	w.Write([]string{"TOTAL", "", "", fmt.Sprintf("%.2f", est.TotalCost)})

	return w.Error()
}

func printWorkersCostCSV(est *r2go2.WorkersCostEstimate) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"item", "usage", "rate", "cost"})
	w.Write([]string{"Requests", fmt.Sprintf("%d", est.Usage.Requests), fmt.Sprintf("$%.2f/M", r2go2.WorkersRequestsPerMillion), fmt.Sprintf("%.2f", est.RequestsCost)})
	w.Write([]string{"Free Tier", fmt.Sprintf("%d", r2go2.WorkersFreeRequests), "", "0.00"})
	w.Write([]string{"Billable", fmt.Sprintf("%d", est.BillableRequests), "", ""})
	w.Write([]string{"TOTAL", "", "", fmt.Sprintf("%.2f", est.TotalCost)})

	return w.Error()
}

func printKVCostCSV(est *r2go2.KVCostEstimate) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"item", "usage", "rate", "cost"})
	w.Write([]string{"Reads", fmt.Sprintf("%d", est.Usage.Reads), fmt.Sprintf("$%.2f/M", r2go2.KVReadsPerMillion), fmt.Sprintf("%.2f", est.ReadsCost)})
	w.Write([]string{"Writes", fmt.Sprintf("%d", est.Usage.Writes), fmt.Sprintf("$%.2f/M", r2go2.KVWritesPerMillion), fmt.Sprintf("%.2f", est.WritesCost)})
	w.Write([]string{"Storage", fmt.Sprintf("%.2f GB", est.Usage.StorageGB), fmt.Sprintf("$%.2f/GB", r2go2.KVStoragePerGB), fmt.Sprintf("%.2f", est.StorageCost)})
	w.Write([]string{"TOTAL", "", "", fmt.Sprintf("%.2f", est.TotalCost)})

	return w.Error()
}

func printDetailCSV(total *r2go2.TotalCostEstimate) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"service", "item", "usage", "rate", "cost"})
	w.Write([]string{"R2", "Storage", fmt.Sprintf("%.2f GB", total.R2.Usage.StorageGB), fmt.Sprintf("$%.3f/GB", r2go2.R2StoragePerGB), fmt.Sprintf("%.2f", total.R2.StorageCost)})
	w.Write([]string{"R2", "Class A Ops", fmt.Sprintf("%d", total.R2.Usage.ClassAOps), fmt.Sprintf("$%.2f/M", r2go2.R2ClassAPerMillion), fmt.Sprintf("%.2f", total.R2.ClassACost)})
	w.Write([]string{"R2", "Class B Ops", fmt.Sprintf("%d", total.R2.Usage.ClassBOps), fmt.Sprintf("$%.2f/M", r2go2.R2ClassBPerMillion), fmt.Sprintf("%.2f", total.R2.ClassBCost)})
	w.Write([]string{"Workers", "Requests", fmt.Sprintf("%d", total.Workers.Usage.Requests), fmt.Sprintf("$%.2f/M", r2go2.WorkersRequestsPerMillion), fmt.Sprintf("%.2f", total.Workers.RequestsCost)})
	w.Write([]string{"KV", "Reads", fmt.Sprintf("%d", total.KV.Usage.Reads), fmt.Sprintf("$%.2f/M", r2go2.KVReadsPerMillion), fmt.Sprintf("%.2f", total.KV.ReadsCost)})
	w.Write([]string{"KV", "Writes", fmt.Sprintf("%d", total.KV.Usage.Writes), fmt.Sprintf("$%.2f/M", r2go2.KVWritesPerMillion), fmt.Sprintf("%.2f", total.KV.WritesCost)})
	w.Write([]string{"KV", "Storage", fmt.Sprintf("%.2f GB", total.KV.Usage.StorageGB), fmt.Sprintf("$%.2f/GB", r2go2.KVStoragePerGB), fmt.Sprintf("%.2f", total.KV.StorageCost)})
	w.Write([]string{"TOTAL", "", "", "", fmt.Sprintf("%.2f", total.TotalMonthlyCost)})

	return w.Error()
}
