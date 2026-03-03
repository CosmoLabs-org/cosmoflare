/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <bucket-name>",
	Short: "Create a new R2 bucket",
	Long: `Create a new Cloudflare R2 bucket with the specified name.

Bucket names must be globally unique across all Cloudflare customers
and follow S3 bucket naming conventions:
- Must be between 3 and 63 characters long
- Must start and end with a lowercase letter or number
- Can contain lowercase letters, numbers, hyphens, and periods

Examples:
  r2go2 create my-awesome-bucket
  r2go2 create my-backup-bucket --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("exactly one argument (bucket name) is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bucketName := args[0]

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
			printInfo("Creating bucket: %s", bucketName)
		}

		if DryRun {
			printSuccess("DRY RUN: Would create bucket '%s' in account %s", bucketName, utils.MaskAccountID(AccountID))
			return
		}

		// Create the bucket
		bucket, err := client.CreateBucket(bucketName)
		if err != nil {
			printErrorAndExit(err, "Failed to create bucket")
		}

		if JSONOutput {
			printSuccessJSON("Bucket created successfully", bucket)
		} else {
			printSuccess("Bucket '%s' created successfully", bucket.Name)
			printInfo("Creation date: %s", bucket.CreatedDate.Format("2006-01-02 15:04:05"))
		}
	},
}

func init() {
	// No additional flags for create command
}