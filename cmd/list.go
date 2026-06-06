package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all R2 buckets",
	Long: `List all Cloudflare R2 buckets in your account with their names and creation dates.

Examples:
  cosmoflare list
  cosmoflare list --json
  cosmoflare list --account-id="your-account-id"`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getAPIClient()
		if err != nil {
			printErrorAndExit(err, "Failed to create API client")
		}

		if Verbose {
			printInfo("Listing buckets in account: %s", utils.MaskAccountID(AccountID))
		}

		buckets, err := client.ListBuckets(context.Background())
		if err != nil {
			printErrorAndExit(err, "Failed to list buckets")
		}

		sort.Slice(buckets, func(i, j int) bool {
			return buckets[i].CreatedAt.After(buckets[j].CreatedAt)
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
}

func printBucketsTable(buckets []*cosmoflare.Bucket) {
	if len(buckets) == 0 {
		printInfo("No buckets found in account %s", utils.MaskAccountID(AccountID))
		return
	}

	fmt.Println()
	fmt.Println("Buckets:")
	fmt.Println()

	for _, bucket := range buckets {
		timeAgo := formatTimeAgo(bucket.CreatedAt)
		fmt.Printf("  - %-20s (%s)\n", bucket.Name, timeAgo)
	}

	fmt.Println()
	fmt.Printf("Total: %d bucket%s\n", len(buckets), pluralize(len(buckets)))
	fmt.Println()
}

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

	return fmt.Sprintf("Created %s", t.Format("2006-01-02"))
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
