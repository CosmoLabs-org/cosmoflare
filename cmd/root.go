/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
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
	Use:   "r2go2",
	Short: "A production-ready CLI tool for managing Cloudflare R2 buckets",
	Long: `R2Go2 is a powerful command-line interface for managing Cloudflare R2 buckets.

Features:
  • Create, list, and delete R2 buckets
  • Upload files with custom object keys
  • Set lifecycle policies for automatic deletion
  • JSON output for scripting and automation
  • Dry-run mode for safe operations

Built by CosmoLabs for the CosmoDev ecosystem.

Environment Variables:
  CLOUDFLARE_API_TOKEN    Your Cloudflare API token (required)
  CLOUDFLARE_ACCOUNT_ID   Your Cloudflare Account ID (or use --account-id)

Examples:
  r2go2 list                              # List all buckets
  r2go2 create my-awesome-bucket          # Create a new bucket
  r2go2 upload my-bucket file.txt --key="remote/file.txt"  # Upload file`,
	Version: AppVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Validate environment variables
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
			printInfo("R2Go2 version: %s", AppVersion)
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
	rootCmd.SetVersionTemplate("R2Go2 version {{.Version}}\n")

	// Global flags
	rootCmd.PersistentFlags().StringVar(&AccountID, "account-id", "", "Cloudflare Account ID (overrides CLOUDFLARE_ACCOUNT_ID)")
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

// validateEnvironment checks required environment variables and settings
func validateEnvironment() error {
	// Check API token
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	if token == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN environment variable is required")
	}

	// Basic token format validation (should start with a pattern)
	if len(token) < 10 {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN appears to be invalid (too short)")
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

// getRelativePath gets a relative path for display purposes
func getRelativePath(path string) string {
	if wd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(wd, path); err == nil {
			return relPath
		}
	}
	return path
}