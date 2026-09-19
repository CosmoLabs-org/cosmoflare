package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"gopkg.in/yaml.v3"
)

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Manage R2 buckets",
	Long: `Bucket management operations for Cloudflare R2.

Commands:
  create     Create a new bucket
  list       List all buckets
  get        Get bucket details
  update     Update bucket metadata
  delete     Delete a bucket
  exists     Check if bucket exists
  import     Create buckets from spec file

Examples:
  cosmoflare bucket create my-awesome-bucket
  cosmoflare bucket list --json
  cosmoflare bucket get my-bucket --output table
  cosmoflare bucket delete my-bucket --force`,
}

var (
	bucketLocation string
	bucketTags     []string
	bucketMetadata []string
	bucketFormat   string
	bucketForce    bool
	bucketSpec     string
)

var bucketCreateCmd = &cobra.Command{
	Use:   "create [bucket-name]",
	Short: "Create a new R2 bucket",
	Long: `Create a new R2 bucket with the specified name.

Bucket names must:
- Be between 3 and 63 characters
- Contain only lowercase letters, numbers, hyphens, and dots
- Start and end with a letter or number
- Not be formatted as IP addresses

Examples:
  cosmoflare bucket create my-bucket
  cosmoflare bucket create my-bucket --location=eu --tags=env=prod,tier=standard`,
	RunE: runBucketCreate,
}

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all R2 buckets",
	Long: `List all R2 buckets in the current account.

Output formats:
- table (default): Human-readable table format
- json: Machine-readable JSON format
- csv: Comma-separated values

Examples:
  cosmoflare bucket list
  cosmoflare bucket list --json
  cosmoflare bucket list --prefix=prod-`,
	RunE: runBucketList,
}

var bucketGetCmd = &cobra.Command{
	Use:   "get [bucket-name]",
	Short: "Get bucket details",
	Long: `Get detailed information about a specific bucket.

Examples:
  cosmoflare bucket get my-bucket
  cosmoflare bucket get my-bucket --output json
  cosmoflare bucket get my-bucket --include-objects`,
	RunE: runBucketGet,
}

var bucketUpdateCmd = &cobra.Command{
	Use:   "update [bucket-name]",
	Short: "Update bucket metadata",
	Long: `Update bucket metadata such as tags and properties.

Note: Bucket names cannot be changed after creation.

Examples:
  cosmoflare bucket update my-bucket --tags=env=staging
  cosmoflare bucket update my-bucket --add-tags=project=website`,
	RunE: runBucketUpdate,
}

var bucketDeleteCmd = &cobra.Command{
	Use:   "delete [bucket-name]",
	Short: "Delete an R2 bucket",
	Long: `Delete an R2 bucket and all its contents.

WARNING: This action is irreversible and will delete all objects
in the bucket. Use with caution.

Examples:
  cosmoflare bucket delete my-bucket
  cosmoflare bucket delete my-bucket --force
  cosmoflare bucket delete my-bucket --dry-run`,
	RunE: runBucketDelete,
}

var bucketExistsCmd = &cobra.Command{
	Use:   "exists [bucket-name]",
	Short: "Check if a bucket exists",
	Long: `Check if a bucket exists in the current account.

Exit codes:
- 0: Bucket exists
- 1: Bucket does not exist
- 2: Error occurred

Examples:
  cosmoflare bucket exists my-bucket && echo "Bucket exists"`,
	RunE: runBucketExists,
}

var bucketImportCmd = &cobra.Command{
	Use:   "import [spec-file]",
	Short: "Create buckets from specification file",
	Long: `Create multiple buckets from a JSON specification file.

Specification file format (JSON):
{"buckets": [{"name": "production-assets", "location": "eu"}]}

Examples:
  cosmoflare bucket import buckets.json
  cosmoflare bucket import buckets.json --dry-run`,
	RunE: runBucketImport,
}

func init() {
	rootCmd.AddCommand(bucketCmd)

	bucketCmd.AddCommand(bucketCreateCmd)
	bucketCmd.AddCommand(bucketListCmd)
	bucketCmd.AddCommand(bucketGetCmd)
	bucketCmd.AddCommand(bucketUpdateCmd)
	bucketCmd.AddCommand(bucketDeleteCmd)
	bucketCmd.AddCommand(bucketExistsCmd)
	bucketCmd.AddCommand(bucketImportCmd)

	bucketCreateCmd.Flags().StringVar(&bucketLocation, "location", "auto", "Bucket location (auto, us, eu, ap)")
	bucketCreateCmd.Flags().StringSliceVar(&bucketTags, "tags", []string{}, "Bucket tags (key=value format)")
	bucketCreateCmd.Flags().StringSliceVar(&bucketMetadata, "metadata", []string{}, "Bucket metadata (key=value format)")

	bucketListCmd.Flags().StringVar(&bucketFormat, "format", "table", "Output format (table, json, csv)")
	bucketListCmd.Flags().String("prefix", "", "Filter buckets by name prefix")
	bucketListCmd.Flags().String("tag", "", "Filter buckets by tag")

	bucketGetCmd.Flags().StringVar(&bucketFormat, "output", "table", "Output format (table, json, yaml)")
	bucketGetCmd.Flags().Bool("include-objects", false, "Include object count and size")

	bucketUpdateCmd.Flags().StringSliceVar(&bucketTags, "tags", []string{}, "Bucket tags (replaces all)")
	bucketUpdateCmd.Flags().StringSliceVar(&bucketMetadata, "metadata", []string{}, "Bucket metadata")
	bucketUpdateCmd.Flags().StringSlice("add-tags", []string{}, "Add tags without replacing")
	bucketUpdateCmd.Flags().StringSlice("remove-tags", []string{}, "Remove specific tags")

	bucketDeleteCmd.Flags().BoolVar(&bucketForce, "force", false, "Skip confirmation prompt")

	bucketImportCmd.Flags().StringVar(&bucketSpec, "spec", "", "Specification file path")
	bucketImportCmd.Flags().Bool("continue", false, "Continue on error")

	// Register dynamic bucket name completion
	RegisterCompletionFlags()
}

// filterBuckets narrows buckets by the user's --prefix flag and by the active
// profile's resource prefix (FEAT-026). Both filters are no-ops when unset.
func filterBuckets(buckets []*cosmoflare.Bucket, namePrefix string) []*cosmoflare.Bucket {
	var filtered []*cosmoflare.Bucket
	for _, bucket := range buckets {
		if namePrefix != "" && !strings.HasPrefix(bucket.Name, namePrefix) {
			continue
		}
		if !matchesResourcePrefix(bucket.Name) {
			continue
		}
		filtered = append(filtered, bucket)
	}
	return filtered
}

func runBucketCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := applyResourcePrefix(args[0])
	printInfo("Creating bucket: %s", bucketName)

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would create bucket '%s'", bucketName)
		return nil
	}

	bucket, err := client.CreateBucket(context.Background(), bucketName)
	if err != nil {
		return outErr("failed to create bucket", err)
	}

	return outPayload("Bucket created successfully", func() any {
		return bucket
	}, func() {
		printSuccess("Bucket '%s' created successfully!", bucketName)
		printInfo("Created: %s", bucket.CreatedAt.Format(time.RFC3339))
	})
}

func runBucketList(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	prefix, _ := cmd.Flags().GetString("prefix")

	printInfo("Listing buckets...")

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		return outErr("failed to list buckets", err)
	}

	filtered := filterBuckets(buckets, prefix)

	if JSONOutput || format == "json" {
		return printJSON(filtered)
	}

	if format == "csv" {
		fmt.Println("name,created_date")
		for _, bucket := range filtered {
			fmt.Printf("%s,%s\n", bucket.Name, bucket.CreatedAt.Format(time.RFC3339))
		}
		return nil
	}

	if len(filtered) == 0 {
		printInfo("No buckets found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED\tSIZE\tOBJECTS")
	for _, bucket := range filtered {
		size := "N/A"
		objects := "N/A"
		if bucket.Size > 0 {
			size = utils.FormatBytes(bucket.Size)
		}
		if bucket.ObjectCount > 0 {
			objects = fmt.Sprintf("%d", bucket.ObjectCount)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			bucket.Name,
			bucket.CreatedAt.Format("2006-01-02 15:04:05"),
			size,
			objects,
		)
	}
	w.Flush()

	printInfo("Total: %d bucket(s)", len(filtered))
	return nil
}

func runBucketGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := applyResourcePrefix(args[0])
	output, _ := cmd.Flags().GetString("output")
	includeObjects, _ := cmd.Flags().GetBool("include-objects")

	printInfo("Getting bucket details: %s", bucketName)

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	bucket, err := client.GetBucket(context.Background(), bucketName)
	if err != nil {
		return outErr("failed to get bucket details", err)
	}

	if includeObjects {
		result, err := client.ListObjects(context.Background(), bucketName, "", "", 0, "")
		if err == nil {
			bucket.ObjectCount = int64(len(result.Items))
			var totalSize int64
			for _, obj := range result.Items {
				totalSize += obj.Size
			}
			bucket.Size = totalSize
		}
	}

	if output == "json" {
		return printJSON(bucket)
	}

	if output == "yaml" {
		yamlData, err := yaml.Marshal(bucket)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Printf("# Bucket: %s\n", bucket.Name)
		fmt.Println(string(yamlData))
		return nil
	}

	fmt.Printf("Bucket: %s\n", bucket.Name)
	fmt.Printf("Created: %s\n", bucket.CreatedAt.Format(time.RFC3339))
	if bucket.Location != "" {
		fmt.Printf("Location: %s\n", bucket.Location)
	}
	if bucket.Size > 0 {
		fmt.Printf("Size: %s\n", utils.FormatBytes(bucket.Size))
	}
	if bucket.ObjectCount > 0 {
		fmt.Printf("Objects: %d\n", bucket.ObjectCount)
	}
	if len(bucket.Tags) > 0 {
		fmt.Println("Tags:")
		for key, value := range bucket.Tags {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

func runBucketUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}

	msg := "bucket metadata updates are not supported by the Cloudflare R2 API. Use 'cosmoflare cors' for CORS settings or manage bucket configuration through the Cloudflare dashboard"
	return outErrf("%s", msg)
}

func runBucketDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := applyResourcePrefix(args[0])
	force, _ := cmd.Flags().GetBool("force")

	printInfo("Deleting bucket: %s", bucketName)

	cliPath := []string{"r2", "bucket", "delete"}
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, force)
	if dry {
		return outPayload("DRY RUN: Would delete bucket", func() any {
			return map[string]string{"bucket": bucketName}
		}, func() {
			printInfo("DRY RUN: Would delete bucket '%s'", bucketName)
		})
	}

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	if err := client.DeleteBucket(context.Background(), bucketName); err != nil {
		auditMutation(cliPath, bucketName, false)
		return outErr("failed to delete bucket", err)
	}
	auditMutation(cliPath, bucketName, true)

	return outPayload("Bucket deleted successfully", func() any {
		return map[string]string{"bucket": bucketName}
	}, func() {
		printSuccess("Bucket '%s' deleted successfully!", bucketName)
	})
}

func runBucketExists(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := applyResourcePrefix(args[0])

	client, err := getAPIClient()
	if err != nil {
		emitConfigError("failed to create API client: %v", err)
		os.Exit(2)
	}

	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		emitConfigError("failed to check bucket existence: %v", err)
		os.Exit(2)
	}

	_ = outResult(map[string]interface{}{"exists": exists, "bucket": bucketName}, func() {
		if exists {
			printInfo("Bucket '%s' exists", bucketName)
		}
	})

	if exists {
		os.Exit(0)
	}
	os.Exit(1)

	return nil
}

func runBucketImport(cmd *cobra.Command, args []string) error {
	specFile, _ := cmd.Flags().GetString("spec")
	continueOnError, _ := cmd.Flags().GetBool("continue")

	if specFile == "" {
		if len(args) == 0 {
			return fmt.Errorf("specification file is required")
		}
		specFile = args[0]
	}

	printInfo("Importing buckets from: %s", specFile)

	var spec BucketSpec
	if err := parseSpecFile(specFile, &spec); err != nil {
		return fmt.Errorf("failed to parse specification file: %w", err)
	}

	if len(spec.Buckets) == 0 {
		printWarning("No buckets found in specification file")
		return nil
	}

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	successCount := 0
	errorCount := 0

	for _, bucketSpec := range spec.Buckets {
		printInfo("Creating bucket: %s", bucketSpec.Name)

		if DryRun {
			printInfo("DRY RUN: Would create bucket '%s'", bucketSpec.Name)
			successCount++
			continue
		}

		_, err := client.CreateBucket(context.Background(), bucketSpec.Name)
		if err != nil {
			printError("Failed to create bucket '%s': %v", bucketSpec.Name, err)
			errorCount++
			if !continueOnError {
				return fmt.Errorf("bucket creation failed")
			}
			continue
		}

		printSuccess("Created bucket: %s", bucketSpec.Name)
		successCount++
	}

	return outPayload("Import complete", func() any {
		return map[string]interface{}{
			"successful": successCount,
			"failed":     errorCount,
			"total":      len(spec.Buckets),
		}
	}, func() {
		printInfo("Import complete: %d successful, %d failed", successCount, errorCount)
	})
}

type BucketSpec struct {
	Buckets []BucketSpecItem `json:"buckets" yaml:"buckets"`
}

type BucketSpecItem struct {
	Name        string            `json:"name" yaml:"name"`
	Location    string            `json:"location,omitempty" yaml:"location,omitempty"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key=value pair: %s", pair)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

func parseSpecFile(filename string, spec *BucketSpec) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	// Try JSON first
	if err := json.Unmarshal(data, spec); err == nil {
		return nil
	}

	// Try YAML
	if err := yaml.Unmarshal(data, spec); err == nil {
		return nil
	}

	return fmt.Errorf("failed to parse spec file as JSON or YAML")
}
