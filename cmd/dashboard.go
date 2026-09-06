/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/tui"
)

var dashboardInterval time.Duration

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch interactive TUI dashboard",
	Long: `Launch a beautiful, interactive terminal-based dashboard for monitoring
and managing your Cloudflare infrastructure.

Features:
  • Real-time monitoring with configurable poll interval
  • Bucket management (create, delete, list)
  • Interactive object listing with pagination
  • Professional keyboard navigation
  • Search and filtering capabilities
  • Settings management

The dashboard provides a GUI-like experience entirely within your terminal,
perfect for SSH connections and command-line workflows.

Examples:
  cosmoflare dashboard                    # Launch dashboard (30s poll interval)
  cosmoflare dashboard --interval 10s     # Poll every 10 seconds
  cosmoflare dashboard --help             # Show dashboard help`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunDashboardWithInterval(dashboardInterval)
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Flags().DurationVar(&dashboardInterval, "interval", 30*time.Second, "Monitoring poll interval")
}
