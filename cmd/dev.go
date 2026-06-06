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
	devPort     int
	devWatch    bool
	devServices string
	devProfile  string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start a local development proxy for Cloudflare services",
	Long: `Start a local HTTP server that proxies requests to Cloudflare services.

Enables offline-friendly development with a local URL rewrite layer and
optional hot-reload of configuration changes.

The server reads credentials from your active profile (or --profile flag)
and proxies requests to the real Cloudflare API through a local endpoint.

Flags:
  --port       Local port to listen on (default: 8787, matches Wrangler)
  --watch      Watch .cosmoflare.yaml for changes and hot-reload (default: true)
  --services   Comma-separated list of services to enable (default: all)
               Valid: r2,kv,workers,dns,zones,ssl,cache,d1,pages,queues
  --profile    Credential profile to use (default: active profile)

Examples:
  cosmoflare dev                          # Start with defaults (all services, port 8787)
  cosmoflare dev --port 3000              # Listen on port 3000
  cosmoflare dev --services r2,kv         # Only proxy R2 and KV
  cosmoflare dev --profile staging        # Use 'staging' credentials
  cosmoflare dev --json                   # Machine-readable startup events
  cosmoflare dev --watch=false            # Disable config hot-reload

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
	devCmd.Flags().StringVar(&devProfile, "profile", "", "Credential profile to use")
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

	if devProfile != "" {
		opts = append(opts, cosmoflare.WithDevProfile(devProfile))
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
