/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/spf13/cobra"
)

var (
	confirmFlag bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <bucket-name>",
	Short: "Delete an R2 bucket",
	Long: `Delete a Cloudflare R2 bucket. This operation is permanent and cannot be undone.

WARNING: Deleting a bucket will permanently delete all objects within it.
Please ensure you have backups of any important data before proceeding.

Examples:
  r2go2 delete my-bucket                    # Interactive confirmation
  r2go2 delete my-bucket --confirm          # Skip confirmation
  r2go2 delete my-bucket --dry-run          # Preview without deleting`,
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
			printInfo("Deleting bucket: %s", bucketName)
		}

		// Confirmation check (unless --confirm flag is used)
		if !confirmFlag && !DryRun {
			if !confirmDeletion(bucketName) {
				printInfo("Bucket deletion cancelled")
				return
			}
		}

		if DryRun {
			printSuccess("DRY RUN: Would delete bucket '%s' and all its contents", bucketName)
			return
		}

		// Delete the bucket
		err = client.DeleteBucket(bucketName)
		if err != nil {
			printErrorAndExit(err, "Failed to delete bucket")
		}

		if JSONOutput {
			response := map[string]interface{}{
				"success":      true,
				"message":      fmt.Sprintf("Bucket '%s' deleted successfully", bucketName),
				"bucket_name":  bucketName,
				"dry_run":      DryRun,
			}
			printJSON(response)
		} else {
			printSuccess("Bucket '%s' deleted successfully", bucketName)
			printWarning("This operation is permanent and cannot be undone")
		}
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Skip confirmation prompt")
}

// confirmDeletion prompts the user for confirmation before deleting
func confirmDeletion(bucketName string) bool {
	fmt.Printf("⚠️  WARNING: You are about to delete the bucket '%s' and ALL of its contents.\n", bucketName)
	fmt.Printf("⚠️  This action is PERMANENT and CANNOT be undone.\n")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("Type 'DELETE' to confirm, or 'cancel' to abort: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			printError("Error reading input: %v", err)
			return false
		}

		input = strings.TrimSpace(input)

		switch strings.ToLower(input) {
		case "delete":
			return true
		case "cancel", "":
			return false
		default:
			printWarning("Please type 'DELETE' to confirm or 'cancel' to abort")
		}
	}
}