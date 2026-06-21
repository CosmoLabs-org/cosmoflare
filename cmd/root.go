/*
Package cmd provides the Cobra CLI commands for Cosmoflare

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"github.com/spf13/cobra"
)

// Build information
var (
	AppVersion   = "dev"
	BuildTime    = "unknown"
	GitCommit    = "unknown"
	AccountID    string
	APIToken     string
	DryRun       bool
	JSONOutput   bool
	Verbose      bool
)

// SetBuildInfo sets the build information
func SetBuildInfo(version, buildTime, gitCommit string) {
	AppVersion = version
	BuildTime = buildTime
	GitCommit = gitCommit
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cosmoflare",
	Short: "Cosmoflare — CLI for the full Cloudflare developer platform",
	Long: `Cosmoflare manages the full Cloudflare developer platform from the terminal.

Services:
  • R2 Storage   — buckets, objects, uploads, downloads, presigned URLs
  • Workers      — deploy, list, logs, delete, settings, bindings
  • KV           — namespaces, get/put/delete, list keys
  • DNS Records  — create, list, get, update, delete (zone-scoped)
  • Zones        — create, list, get, settings, delete (account-scoped)
  • SSL/TLS      — encryption mode, certificates, verification, settings
  • Cache        — purge all/URL/tag/host, cache settings

Features:
  • JSON output on every command (--json)
  • Dry-run mode for safe operations (--dry-run)
  • Shell completion for Bash, Zsh, Fish, PowerShell
  • Agent-friendly: rich --help, predictable exit codes

Built by CosmoLabs (https://cosmolabs.org). Open-source, MIT licensed.
The 'r2go2' binary remains available as a backward-compatible alias.

Environment Variables:
  CLOUDFLARE_API_TOKEN    Your Cloudflare API token (required)
  CLOUDFLARE_ACCOUNT_ID   Your Cloudflare Account ID (or use --account-id)

Examples:
  cosmoflare bucket list                       # List all R2 buckets
  cosmoflare dns list ZONE_ID                  # List DNS records
  cosmoflare zone list --json                  # List zones as JSON
  cosmoflare ssl status ZONE_ID               # Check SSL/TLS mode
  cosmoflare cache purge ZONE_ID --all --force # Purge entire cache
  cosmoflare worker deploy my-worker -s w.js  # Deploy a Worker`,
	Version: AppVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip API validation for commands that don't need R2 access.
		// Parent commands in this list cause all subcommands to skip too.
		skipValidation := []string{
			"setup", "config", "auth", "completion", "help", "version",
			"theme", "demo", "backup", "plugin", "account", "serve",
		}
		for _, skip := range skipValidation {
			if cmd.Name() == skip {
				return
			}
			for p := cmd; p != nil; p = p.Parent() {
				if p.Name() == skip {
					return
				}
			}
		}

		// Get API token from flag first, then environment
		if APIToken == "" {
			APIToken = os.Getenv("CLOUDFLARE_API_TOKEN")
		}

		// Validate API token is available (from flag or env)
		if err := validateEnvironment(); err != nil {
			printError("Configuration error: %v", err)
			os.Exit(1)
		}

		// Get account ID from flag or environment
		if AccountID == "" {
			AccountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
			if AccountID == "" {
				printError("Cloudflare Account ID is required. Set CLOUDFLARE_ACCOUNT_ID environment variable or use --account-id flag")
				os.Exit(1)
			}
		}

		// Print dry-run warning
		if DryRun {
			printWarning("DRY RUN MODE: No actual changes will be made")
		}

		// Verbose output
		if Verbose {
			printInfo("Cosmoflare version: %s", AppVersion)
			printInfo("Account ID: %s", utils.MaskAccountID(AccountID))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set custom version template with proper branding
	rootCmd.SetVersionTemplate("Cosmoflare version {{.Version}}\n")

	// Global flags
	rootCmd.PersistentFlags().StringVar(&AccountID, "account-id", "", "Cloudflare Account ID (overrides CLOUDFLARE_ACCOUNT_ID)")
	rootCmd.PersistentFlags().StringVar(&APIToken, "api-token", "", "Cloudflare API token (overrides CLOUDFLARE_API_TOKEN)")
	rootCmd.PersistentFlags().BoolVar(&DryRun, "dry-run", false, "Show what would happen without executing")
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose output")

	// NOTE: Commands are registered via their own init() functions in each cmd/*.go file
	// Legacy commands for backward compatibility are added here only
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)

	// Add completion command
	rootCmd.AddCommand(completionCmd)
}

// validateEnvironment checks that an API token is available and valid.
// The token may come from the --api-token flag (already bound to APIToken)
// or from the CLOUDFLARE_API_TOKEN environment variable.
func validateEnvironment() error {
	token := APIToken
	if token == "" {
		return fmt.Errorf("Cloudflare API token is required. Set CLOUDFLARE_API_TOKEN environment variable or use --api-token flag")
	}

	// Basic token format validation
	if len(token) < 10 {
		return fmt.Errorf("API token appears to be invalid (too short)")
	}

	return nil
}

// Output helpers for consistent formatting

// printInfo prints an informational message
func printInfo(format string, args ...interface{}) {
	if JSONOutput {
		return // Skip in JSON mode
	}
	fmt.Printf("ℹ️  %s\n", fmt.Sprintf(format, args...))
}

// printSuccess prints a success message
func printSuccess(format string, args ...interface{}) {
	if JSONOutput {
		return // Skip in JSON mode
	}
	fmt.Printf("✅ %s\n", fmt.Sprintf(format, args...))
}

// printWarning prints a warning message
func printWarning(format string, args ...interface{}) {
	if JSONOutput {
		return // Skip in JSON mode
	}
	fmt.Printf("⚠️  %s\n", fmt.Sprintf(format, args...))
}

// printError prints an error message
func printError(format string, args ...interface{}) {
	if JSONOutput {
		return // Skip in JSON mode
	}
	fmt.Printf("❌ %s\n", fmt.Sprintf(format, args...))
}

// printJSON outputs data in JSON format
func printJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// OutputResponse represents a standard response format
type OutputResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	DryRun  bool        `json:"dry_run,omitempty"`
}

// printSuccessJSON prints a success response in JSON format
func printSuccessJSON(message string, data interface{}) error {
	response := OutputResponse{
		Success: true,
		Message: message,
		Data:    data,
		DryRun:  DryRun,
	}
	return printJSON(response)
}

// printErrorJSON prints an error response in JSON format
func printErrorJSON(message string) error {
	response := OutputResponse{
		Success: false,
		Error:   message,
		DryRun:  DryRun,
	}
	return printJSON(response)
}

// printErrorAndExit prints an error and exits with status 1
func printErrorAndExit(err error, context string) {
	if JSONOutput {
		printErrorJSON(fmt.Sprintf("%s: %v", context, err))
	} else {
		printError("%s: %v", context, err)
		fmt.Println()
		printInfo("Troubleshooting tips:")
		printInfo("1. Verify your CLOUDFLARE_API_TOKEN is correct")
		printInfo("2. Ensure your account ID is correct")
		printInfo("3. Check that your token has R2 permissions")
		printInfo("4. Verify your network connection")
		fmt.Println()
	}
	os.Exit(1)
}

// getRelativePath gets a relative path for display purposes
func getRelativePath(path string) string {
	if wd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(wd, path); err == nil {
			return relPath
		}
	}
	return path
}