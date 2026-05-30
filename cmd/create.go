package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <bucket-name>",
	Short: "Create a new R2 bucket",
	Long: `Create a new Cloudflare R2 bucket with the specified name.

Bucket names must follow S3 naming conventions:
- Must be between 3 and 63 characters long
- Must start and end with a lowercase letter or number
- Can contain lowercase letters, numbers, hyphens, and periods

Examples:
  cosmoflare create my-awesome-bucket
  cosmoflare create my-backup-bucket --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("exactly one argument (bucket name) is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bucketName := args[0]

		client, err := getAPIClient()
		if err != nil {
			printErrorAndExit(err, "Failed to create API client")
		}

		if Verbose {
			printInfo("Creating bucket: %s", bucketName)
		}

		if DryRun {
			printSuccess("DRY RUN: Would create bucket '%s' in account %s", bucketName, utils.MaskAccountID(AccountID))
			return
		}

		bucket, err := client.CreateBucket(context.Background(), bucketName)
		if err != nil {
			printErrorAndExit(err, "Failed to create bucket")
		}

		if JSONOutput {
			printSuccessJSON("Bucket created successfully", bucket)
		} else {
			printSuccess("Bucket '%s' created successfully", bucket.Name)
			printInfo("Creation date: %s", bucket.CreatedAt.Format(time.RFC3339))
		}
	},
}

func init() {
	// No additional flags for create command
}
