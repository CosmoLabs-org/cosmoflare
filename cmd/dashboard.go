/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/tui"
)

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch interactive TUI dashboard for R2 bucket management",
	Long: `Launch a beautiful, interactive terminal-based dashboard for managing Cloudflare R2 buckets.

Features:
  • Professional keyboard navigation
  • Real-time monitoring and statistics
  • Interactive file management
  • Beautiful visual design
  • Search and filtering capabilities
  • Settings management

The dashboard provides a GUI-like experience entirely within your terminal,
perfect for SSH connections and command-line workflows.

Examples:
  r2go2 dashboard                    # Launch dashboard
  r2go2 dashboard --help            # Show dashboard help`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch TUI dashboard
		if err := runDashboard(); err != nil {
			printError("Dashboard error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add dashboard command to root command
	rootCmd.AddCommand(dashboardCmd)
}

// runDashboard initializes and starts the TUI dashboard
func runDashboard() error {
	// Import and use the TUI package
	return tui.RunDashboard()
}