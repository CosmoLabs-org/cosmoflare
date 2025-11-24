/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all R2 buckets",
	Long: `List all Cloudflare R2 buckets in your account with their names and creation dates.

Examples:
  r2go2 list
  r2go2 list --json
  r2go2 list --account-id="your-account-id"`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Create API client
		opts := &api.ClientOptions{
			AccountID: AccountID,
			APIToken:  APIToken,
		}
		client, err := api.NewClient(opts)
		if err != nil {
			printErrorAndExit(err, "Failed to create API client")
		}

		if Verbose {
			printInfo("Listing buckets in account: %s", maskAccountID(AccountID))
		}

		// List buckets
		buckets, err := client.ListBuckets()
		if err != nil {
			printErrorAndExit(err, "Failed to list buckets")
		}

		// Sort buckets by creation date (newest first)
		sort.Slice(buckets, func(i, j int) bool {
			return buckets[i].CreatedDate.After(buckets[j].CreatedDate)
		})

		if JSONOutput {
			response := map[string]interface{}{
				"success": true,
				"buckets": buckets,
				"total":   len(buckets),
				"dry_run": DryRun,
			}
			printJSON(response)
		} else {
			printBucketsTable(buckets)
		}
	},
}

func init() {
	// No additional flags for list command
}

// printBucketsTable displays buckets in a formatted table
func printBucketsTable(buckets []*api.Bucket) {
	if len(buckets) == 0 {
		printInfo("No buckets found in account %s", maskAccountID(AccountID))
		return
	}

	fmt.Println()
	fmt.Println("🪣 Buckets:")
	fmt.Println()

	// Print each bucket
	for _, bucket := range buckets {
		timeAgo := formatTimeAgo(bucket.CreatedDate)
		fmt.Printf("  • %-20s (%s)\n", bucket.Name, timeAgo)
	}

	fmt.Println()
	fmt.Printf("Total: %d bucket%s\n", len(buckets), pluralize(len(buckets)))
	fmt.Println()
}

// formatTimeAgo returns a human-readable time difference
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("Created %d minute%s ago", minutes, pluralize(minutes))
	}

	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("Created %d hour%s ago", hours, pluralize(hours))
	}

	if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("Created %d day%s ago", days, pluralize(days))
	}

	// For older buckets, show the actual date
	return fmt.Sprintf("Created %s", t.Format("2006-01-02"))
}

// pluralize returns a plural suffix if the count is not 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// printErrorAndExit is a helper function to print errors and exit
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
}