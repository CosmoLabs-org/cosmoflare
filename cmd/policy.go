/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/spf13/cobra"
)

var policyDays int

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy <bucket> <command>",
	Short: "Manage lifecycle policies for R2 buckets",
	Long: `Manage lifecycle policies for Cloudflare R2 buckets to automatically delete objects after a specified time.

This command currently supports setting simple expiration policies that delete objects after a specified number of days.

Examples:
  r2go2 policy my-bucket set --days=30    # Delete objects after 30 days
  r2go2 policy my-bucket set --days=365   # Delete objects after 1 year
  r2go2 policy my-bucket set --days=7 --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("exactly two arguments required: <bucket> <command>")
		}
		command := args[1]
		if command != "set" {
			return fmt.Errorf("unsupported command '%s'. Supported commands: set", command)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bucketName := args[0]
		command := args[1]

		// Validate days parameter for set command
		if command == "set" && policyDays <= 0 {
			printErrorAndExit(fmt.Errorf("--days must be a positive number"), "Invalid policy parameters")
		}

		// Create API client
		client, err := api.NewClient(AccountID)
		if err != nil {
			printErrorAndExit(err, "Failed to create API client")
		}

		switch command {
		case "set":
			handleSetPolicy(client, bucketName, policyDays)
		}
	},
}

func init() {
	policyCmd.Flags().IntVar(&policyDays, "days", 0, "Number of days after which objects should be deleted")
	policyCmd.MarkFlagRequired("days")
}

// handleSetPolicy handles setting a lifecycle policy on a bucket
func handleSetPolicy(client *api.Client, bucketName string, days int) {
	if Verbose {
		printInfo("Setting lifecycle policy for bucket: %s", bucketName)
		printInfo("Objects will be deleted after %d days", days)
	}

	policy := &api.LifecyclePolicy{
		ExpirationDays: days,
	}

	if DryRun {
		printSuccess("DRY RUN: Would set lifecycle policy on bucket '%s'", bucketName)
		printInfo("Objects older than %d days will be automatically deleted", days)
		return
	}

	// Set the policy
	err := client.SetLifecyclePolicy(bucketName, policy)
	if err != nil {
		printErrorAndExit(err, "Failed to set lifecycle policy")
	}

	if JSONOutput {
		response := map[string]interface{}{
			"success":          true,
			"message":          "Lifecycle policy set successfully",
			"bucket_name":      bucketName,
			"expiration_days":  days,
			"dry_run":          DryRun,
		}
		printJSON(response)
	} else {
		printSuccess("Lifecycle policy set successfully for bucket '%s'", bucketName)
		printInfo("Objects will be automatically deleted after %d days", days)
		printWarning("This policy affects all current and future objects in the bucket")
	}
}