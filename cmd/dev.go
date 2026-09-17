package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	devPort      int
	devWatch     bool
	devServices  string
	devNotifyURL string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start a local dev server for Cloudflare services (proxying in progress)",
	Long: `Start a local HTTP development server.

STATUS: the server scaffold is live — it binds the port, serves /health,
emits startup events (--json), and shuts down cleanly on SIGINT/SIGTERM.
Per-service proxying is NOT yet implemented: every service route currently
returns 502 {"error":"upstream not configured"} until the reverse proxy
lands (tracked as BUG-046).

Flags:
  --port       Local port to listen on (default: 8787, matches Wrangler)
  --watch      Watch .cosmoflare.yaml for changes and hot-reload (default: true)
  --services   Comma-separated list of services to enable (default: all)
               Valid: r2,kv,workers,dns,zones,ssl,cache,d1,pages,queues

Examples:
  cosmoflare dev                          # Start with defaults (all services, port 8787)
  cosmoflare dev --port 3000              # Listen on port 3000
  cosmoflare dev --services r2,kv         # Only proxy R2 and KV
  cosmoflare dev --env staging            # Use the 'staging' environment profile
  cosmoflare dev --json                   # Machine-readable startup events
  cosmoflare dev --watch=false            # Disable config hot-reload
  cosmoflare dev --notify                # POST lifecycle events to dev.notify_url
  cosmoflare dev --notify=https://hook.example.com/dev  # Explicit webhook URL

  # Use with other tools:
  curl http://localhost:8787/health       # Health check
  curl http://localhost:8787/r2/buckets   # Proxy to R2 API`,
	Args: cobra.NoArgs,
	RunE: runDev,
}

func init() {
	rootCmd.AddCommand(devCmd)

	devCmd.Flags().IntVar(&devPort, "port", 8787, "Local port to listen on")
	devCmd.Flags().BoolVar(&devWatch, "watch", true, "Watch config for changes and hot-reload")
	devCmd.Flags().StringVar(&devServices, "services", "", "Comma-separated services to enable (default: all)")
	devCmd.Flags().StringVar(&devNotifyURL, "notify", "", "Webhook URL for lifecycle events (or set dev.notify_url in .cosmoflare.yaml)")
}

func runDev(cmd *cobra.Command, args []string) error {
	var opts []cosmoflare.DevOption

	opts = append(opts, cosmoflare.WithDevPort(devPort))
	opts = append(opts, cosmoflare.WithDevWatch(devWatch))

	if devServices != "" {
		services := strings.Split(devServices, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
		opts = append(opts, cosmoflare.WithDevServices(services))
	}

	// The dev server follows the global --env selection (FEAT-026): the
	// per-command --profile flag was dropped in favor of the single selector.
	if ActiveProfile != nil {
		opts = append(opts, cosmoflare.WithDevProfile(ActiveProfile.Name))
	}

	if devNotifyURL != "" {
		opts = append(opts, cosmoflare.WithDevNotifyURL(devNotifyURL))
	}

	ds := cosmoflare.NewDevServer(opts...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if JSONOutput {
		event := ds.StartupEvent()
		b, _ := json.MarshalIndent(event, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "cosmoflare dev server starting on port %d\n", ds.Port())
		fmt.Fprintf(cmd.OutOrStdout(), "Services: %s\n", strings.Join(ds.Services(), ", "))
		if ds.Profile() != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Profile: %s\n", ds.Profile())
		}
		if ds.Watch() {
			fmt.Fprintf(cmd.OutOrStdout(), "Watching .cosmoflare.yaml for changes\n")
		}
	}

	err := ds.Start(ctx)
	if err != nil {
		return fmt.Errorf("dev server failed: %w", err)
	}

	return nil
}
