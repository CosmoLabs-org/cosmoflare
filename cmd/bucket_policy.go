package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var bucketPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage R2 bucket policy documents",
	Long: `Get and set the access policy document on a Cloudflare R2 bucket.

Commands:
  get   Show the bucket's policy document
  set   Replace the bucket's policy document

Examples:
  cosmoflare bucket policy get my-bucket
  cosmoflare bucket policy set my-bucket --file policy.json`,
}

var bucketPolicyGetCmd = &cobra.Command{
	Use:   "get [bucket]",
	Short: "Show the bucket's policy document",
	Long: `Show the raw JSON access policy document configured on an R2 bucket.

Examples:
  cosmoflare bucket policy get my-bucket
  cosmoflare bucket policy get my-bucket --json`,
	Args: prefixedResourceArgs(cobra.ExactArgs(1)),
	RunE: runBucketPolicyGet,
}

var bucketPolicySetCmd = &cobra.Command{
	Use:   "set [bucket]",
	Short: "Replace the bucket's policy document",
	Long: `Replace the access policy document on an R2 bucket.

The file is validated as well-formed JSON locally before any API call.

Examples:
  cosmoflare bucket policy set my-bucket --file policy.json
  cosmoflare bucket policy set my-bucket --file policy.json --json`,
	Args: prefixedResourceArgs(cobra.ExactArgs(1)),
	RunE: runBucketPolicySet,
}

var bucketPolicyFile string

func init() {
	bucketCmd.AddCommand(bucketPolicyCmd)

	bucketPolicyCmd.AddCommand(bucketPolicyGetCmd)
	bucketPolicyCmd.AddCommand(bucketPolicySetCmd)

	bucketPolicySetCmd.Flags().StringVar(&bucketPolicyFile, "file", "", "Path to a JSON policy document")
}

// getBucketPolicyService creates the policy service using the same global
// credentials the bucket commands rely on.
func getBucketPolicyService() *cosmoflare.BucketPolicyService {
	return cosmoflare.NewBucketPolicyService(AccountID, APIToken)
}

func runBucketPolicyGet(cmd *cobra.Command, args []string) error {
	bucket := args[0]

	svc := getBucketPolicyService()
	policy, err := svc.GetBucketPolicy(cmd.Context(), bucket)
	if err != nil {
		return outErr("failed to get bucket policy", err)
	}

	return outResult(policy, func() {
		fmt.Println(string(policy))
	})
}

func runBucketPolicySet(cmd *cobra.Command, args []string) error {
	// Retained applyResourcePrefix wrap (idempotent with the Args
	// declaration above): TestBucketPolicySet_PrefixScoping drives this runner
	// directly, bypassing cobra, and pins runner-level prefixing.
	bucket := applyResourcePrefix(args[0])

	if bucketPolicyFile == "" {
		return fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(bucketPolicyFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", bucketPolicyFile, err)
	}

	if DryRun {
		return outResult(map[string]interface{}{"dry_run": true, "action": "set_bucket_policy", "bucket": bucket}, func() {
			printInfo("DRY RUN: Would set policy on bucket '%s'", bucket)
		})
	}

	svc := getBucketPolicyService()
	if err := svc.SetBucketPolicy(cmd.Context(), bucket, json.RawMessage(data)); err != nil {
		return outErr("failed to set bucket policy", err)
	}

	return outPayload("Bucket policy replaced", func() any {
		return map[string]string{"bucket": bucket}
	}, func() {
		printSuccess("Set policy on bucket '%s'", bucket)
	})
}
