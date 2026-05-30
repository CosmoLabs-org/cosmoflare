package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage Cloudflare Workers",
	Long: `Worker management operations for Cloudflare Workers.

Commands:
  deploy    Deploy or update a Worker script
  list      List all Workers
  get       Get Worker script content
  delete    Delete a Worker
  logs      View Worker logs
  settings  Update Worker settings

Examples:
  cosmoflare worker deploy my-worker --script=worker.js
  cosmoflare worker list --json
  cosmoflare worker get my-worker
  cosmoflare worker delete my-worker --force`,
}

var (
	workerScript       string
	workerCompatDate   string
	workerBindings     []string
	workerTags         []string
	workerModule       bool
	workerForce        bool
	workerLogLimit     int
	workerLogFollow    bool
	workerLogInterval  int
	workerLogLevel     string
	workerLogSince     string
	workerUsageModel   string
)

var workerDeployCmd = &cobra.Command{
	Use:   "deploy [name]",
	Short: "Deploy or update a Worker script",
	Long: `Deploy a Worker script to Cloudflare.

The script can be provided via --script flag (file path) or stdin.

Examples:
  cosmoflare worker deploy my-worker --script=worker.js
  cosmoflare worker deploy my-worker --script=worker.js --compatibility-date=2024-01-01
  cosmoflare worker deploy my-worker --script=worker.js --bindings=kv:MY_KV:ns-123`,
	RunE: runWorkerDeploy,
}

var workerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Workers",
	Long: `List all Workers in the current account.

Examples:
  cosmoflare worker list
  cosmoflare worker list --json`,
	RunE: runWorkerList,
}

var workerGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get Worker script content",
	Long: `Get a Worker's script content and metadata.

Examples:
  cosmoflare worker get my-worker
  cosmoflare worker get my-worker --json`,
	RunE: runWorkerGet,
}

var workerDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a Worker",
	Long: `Delete a Worker script.

WARNING: This action is irreversible.

Examples:
  cosmoflare worker delete my-worker
  cosmoflare worker delete my-worker --force`,
	RunE: runWorkerDelete,
}

var workerLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "View Worker logs",
	Long: `View recent log events for a Worker.

Use --follow (-f) to continuously poll for new log entries. Combine with
--level to filter by severity and --since to set a time window.

Examples:
  cosmoflare worker logs my-worker
  cosmoflare worker logs my-worker --limit=50 --json
  cosmoflare worker logs my-worker --follow
  cosmoflare worker logs my-worker -f --level=error --since=15m
  cosmoflare worker logs my-worker -f --interval=5 --json`,
	RunE: runWorkerLogs,
}

var workerSettingsCmd = &cobra.Command{
	Use:   "settings [name]",
	Short: "Update Worker settings",
	Long: `Update a Worker's configuration settings.

Examples:
  cosmoflare worker settings my-worker --compatibility-date=2024-01-01
  cosmoflare worker settings my-worker --usage-model=bundled`,
	RunE: runWorkerSettings,
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.AddCommand(workerDeployCmd)
	workerCmd.AddCommand(workerListCmd)
	workerCmd.AddCommand(workerGetCmd)
	workerCmd.AddCommand(workerDeleteCmd)
	workerCmd.AddCommand(workerLogsCmd)
	workerCmd.AddCommand(workerSettingsCmd)

	workerDeployCmd.Flags().StringVarP(&workerScript, "script", "s", "", "Path to Worker script file")
	workerDeployCmd.Flags().StringVar(&workerCompatDate, "compatibility-date", "", "Workers runtime compatibility date (yyyy-mm-dd)")
	workerDeployCmd.Flags().StringSliceVar(&workerBindings, "bindings", []string{}, "Bindings in name:type:id format (e.g., MY_KV:kv:ns-123)")
	workerDeployCmd.Flags().StringSliceVar(&workerTags, "tags", []string{}, "Worker tags")
	workerDeployCmd.Flags().BoolVar(&workerModule, "module", false, "Treat script as ES module")

	workerDeleteCmd.Flags().BoolVar(&workerForce, "force", false, "Skip confirmation prompt")

	workerLogsCmd.Flags().IntVar(&workerLogLimit, "limit", 100, "Maximum number of log entries")
	workerLogsCmd.Flags().BoolVarP(&workerLogFollow, "follow", "f", false, "Continuously poll for new log entries")
	workerLogsCmd.Flags().IntVar(&workerLogInterval, "interval", 2, "Polling interval in seconds (with --follow)")
	workerLogsCmd.Flags().StringVar(&workerLogLevel, "level", "", "Filter by log level (error|warn|info|debug)")
	workerLogsCmd.Flags().StringVar(&workerLogSince, "since", "", "Show logs since duration (e.g. 15m, 1h)")

	workerSettingsCmd.Flags().StringVar(&workerCompatDate, "compatibility-date", "", "Workers runtime compatibility date")
	workerSettingsCmd.Flags().StringVar(&workerUsageModel, "usage-model", "", "Usage model (bundled or unbound)")
	workerSettingsCmd.Flags().StringSliceVar(&workerBindings, "bindings", []string{}, "Bindings in name:type:id format")
}

func getWorkerService() (*cosmoflare.WorkerService, error) {
	return cosmoflare.NewWorkerServiceFromCreds(AccountID, APIToken)
}

func runWorkerDeploy(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	if workerScript == "" {
		return fmt.Errorf("script file is required (--script or -s)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	f, err := os.Open(workerScript)
	if err != nil {
		return fmt.Errorf("failed to open script file: %w", err)
	}
	defer f.Close()

	var opts []cosmoflare.WorkerOption
	if workerCompatDate != "" {
		opts = append(opts, cosmoflare.WithWorkerCompatibilityDate(workerCompatDate))
	}
	if len(workerBindings) > 0 {
		bindings, err := parseWorkerBindings(workerBindings)
		if err != nil {
			return err
		}
		opts = append(opts, cosmoflare.WithWorkerBindings(bindings))
	}
	if len(workerTags) > 0 {
		opts = append(opts, cosmoflare.WithWorkerTags(workerTags))
	}
	if workerModule {
		opts = append(opts, cosmoflare.WithWorkerModule(true))
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would deploy worker", map[string]string{"name": name, "script": workerScript})
		}
		printInfo("DRY RUN: Would deploy worker '%s' from %s", name, workerScript)
		return nil
	}

	worker, err := svc.Deploy(context.Background(), name, f, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to deploy worker: %v", err))
		}
		return fmt.Errorf("failed to deploy worker: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Worker deployed successfully", worker)
	}

	printSuccess("Worker '%s' deployed successfully!", name)
	printInfo("Size: %d bytes", worker.Size)
	return nil
}

func runWorkerList(cmd *cobra.Command, args []string) error {
	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	workers, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list workers: %v", err))
		}
		return fmt.Errorf("failed to list workers: %w", err)
	}

	if JSONOutput {
		return printJSON(workers)
	}

	if len(workers) == 0 {
		printInfo("No workers found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMODIFIED\tSIZE")
	for _, worker := range workers {
		fmt.Fprintf(w, "%s\t%s\t%d\n",
			worker.Name,
			worker.Modified.Format("2006-01-02 15:04:05"),
			worker.Size,
		)
	}
	w.Flush()

	printInfo("Total: %d worker(s)", len(workers))
	return nil
}

func runWorkerGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	worker, err := svc.Get(context.Background(), name)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get worker: %v", err))
		}
		return fmt.Errorf("failed to get worker: %w", err)
	}

	if JSONOutput {
		return printJSON(worker)
	}

	fmt.Printf("Worker: %s\n", worker.Name)
	fmt.Printf("Modified: %s\n", worker.Modified.Format("2006-01-02 15:04:05"))
	fmt.Printf("Size: %d bytes\n", worker.Size)
	if worker.CompatibilityDate != "" {
		fmt.Printf("Compatibility Date: %s\n", worker.CompatibilityDate)
	}
	if len(worker.Bindings) > 0 {
		fmt.Println("Bindings:")
		for _, b := range worker.Bindings {
			fmt.Printf("  %s (%s): %s\n", b.Name, b.Type, b.ID)
		}
	}
	fmt.Println()
	fmt.Println("--- Script ---")
	fmt.Println(worker.Script)
	return nil
}

func runWorkerDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	if !workerForce && !DryRun {
		fmt.Printf("Are you sure you want to delete worker '%s'? [y/N]: ", name)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Worker deletion cancelled")
			return nil
		}
	}

	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete worker", map[string]string{"name": name})
		}
		printInfo("DRY RUN: Would delete worker '%s'", name)
		return nil
	}

	if err := svc.Delete(context.Background(), name); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete worker: %v", err))
		}
		return fmt.Errorf("failed to delete worker: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Worker deleted successfully", map[string]string{"name": name})
	}
	printSuccess("Worker '%s' deleted successfully!", name)
	return nil
}

func runWorkerLogs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	if workerLogFollow {
		return runWorkerLogsFollow(cmd, svc, name)
	}

	var opts []cosmoflare.LogOption
	if workerLogLimit > 0 {
		opts = append(opts, cosmoflare.WithLogLimit(workerLogLimit))
	}

	entries, err := svc.Logs(context.Background(), name, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get worker logs: %v", err))
		}
		return fmt.Errorf("failed to get worker logs: %w", err)
	}

	if JSONOutput {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		printInfo("No log entries found for worker '%s'", name)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tLEVEL\tEVENT\tMESSAGE")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Level,
			e.Event,
			e.Message,
		)
	}
	w.Flush()
	return nil
}

func runWorkerLogsFollow(cmd *cobra.Command, svc *cosmoflare.WorkerService, name string) error {
	tailOpts := &cosmoflare.TailOptions{
		Interval: time.Duration(workerLogInterval) * time.Second,
		Level:    workerLogLevel,
	}

	if workerLogSince != "" {
		d, err := time.ParseDuration(workerLogSince)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", workerLogSince, err)
		}
		tailOpts.Since = d
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	ch, err := svc.TailLogs(ctx, name, tailOpts)
	if err != nil {
		return fmt.Errorf("failed to start log tailing: %w", err)
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	for entry := range ch {
		if JSONOutput {
			enc.Encode(entry)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%s  [%s]  %s  %s\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.Level,
				entry.Event,
				entry.Message,
			)
		}
	}
	return nil
}

func runWorkerSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	if workerCompatDate == "" && workerUsageModel == "" && len(workerBindings) == 0 {
		return fmt.Errorf("at least one setting is required (--compatibility-date, --usage-model, or --bindings)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	settings := cosmoflare.WorkerSettings{
		CompatibilityDate: workerCompatDate,
		UsageModel:        workerUsageModel,
	}

	if len(workerBindings) > 0 {
		bindings, err := parseWorkerBindings(workerBindings)
		if err != nil {
			return err
		}
		settings.Bindings = bindings
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update worker settings", map[string]interface{}{"name": name, "settings": settings})
		}
		printInfo("DRY RUN: Would update settings for worker '%s'", name)
		return nil
	}

	if err := svc.UpdateSettings(context.Background(), name, settings); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update worker settings: %v", err))
		}
		return fmt.Errorf("failed to update worker settings: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Worker settings updated", map[string]string{"name": name})
	}
	printSuccess("Settings updated for worker '%s'", name)
	return nil
}

func parseWorkerBindings(raw []string) ([]cosmoflare.WorkerBinding, error) {
	bindings := make([]cosmoflare.WorkerBinding, 0, len(raw))
	for _, b := range raw {
		parts := strings.SplitN(b, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid binding format %q: expected name:type:id", b)
		}
		bindings = append(bindings, cosmoflare.WorkerBinding{
			Name: parts[0],
			Type: parts[1],
			ID:   parts[2],
		})
	}
	return bindings, nil
}
