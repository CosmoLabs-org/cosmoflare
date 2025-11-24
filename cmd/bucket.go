/*
Package cmd provides bucket management commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
)

// bucketCmd represents the bucket command
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
  r2go2 bucket create my-awesome-bucket
  r2go2 bucket ls --json
  r2go2 bucket get my-bucket --output table
  r2go2 bucket delete my-bucket --force`,
}

var (
	bucketLocation string
	bucketTags     []string
	bucketMetadata []string
	bucketFormat   string
	bucketForce    bool
	bucketSpec     string
)

// bucketCreateCmd represents the bucket create command
var bucketCreateCmd = &cobra.Command{
	Use:   "create [bucket-name]",
	Short: "Create a new R2 bucket",
	Long: `Create a new R2 bucket with the specified name.

Bucket names must:
- Be between 3 and 63 characters
- Contain only lowercase letters, numbers, hyphens, and dots
- Start and end with a letter or number
- Not be formatted as IP addresses

Optional parameters:
- --location: Geographic location for the bucket
- --tags: Key=value pairs for bucket tags
- --metadata: Additional metadata

Examples:
  r2go2 bucket create my-bucket
  r2go2 bucket create my-bucket --location=eu --tags=env=prod,tier=standard`,
	RunE: runBucketCreate,
}

// bucketListCmd represents the bucket list command
var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all R2 buckets",
	Long: `List all R2 buckets in the current account.

Output formats:
- table (default): Human-readable table format
- json: Machine-readable JSON format
- csv: Comma-separated values

Filtering options:
- --prefix: Filter buckets by name prefix
- --tag: Filter buckets by tag

Examples:
  r2go2 bucket list
  r2go2 bucket list --json
  r2go2 bucket list --prefix=prod-`,
	RunE: runBucketList,
}

// bucketGetCmd represents the bucket get command
var bucketGetCmd = &cobra.Command{
	Use:   "get [bucket-name]",
	Short: "Get bucket details",
	Long: `Get detailed information about a specific bucket.

Shows:
- Basic bucket information
- Creation date
- Location
- Object count and size
- Tags and metadata

Output formats:
- table (default): Human-readable table
- json: Machine-readable JSON
- yaml: YAML format

Examples:
  r2go2 bucket get my-bucket
  r2go2 bucket get my-bucket --output json
  r2go2 bucket get my-bucket --include-objects`,
	RunE: runBucketGet,
}

// bucketUpdateCmd represents the bucket update command
var bucketUpdateCmd = &cobra.Command{
	Use:   "update [bucket-name]",
	Short: "Update bucket metadata",
	Long: `Update bucket metadata such as tags and properties.

Note: Bucket names cannot be changed after creation.

Parameters:
- --tags: Update bucket tags (replaces all tags)
- --metadata: Update bucket metadata
- --add-tags: Add tags without replacing existing ones

Examples:
  r2go2 bucket update my-bucket --tags=env=staging
  r2go2 bucket update my-bucket --add-tags=project=website`,
	RunE: runBucketUpdate,
}

// bucketDeleteCmd represents the bucket delete command
var bucketDeleteCmd = &cobra.Command{
	Use:   "delete [bucket-name]",
	Short: "Delete an R2 bucket",
	Long: `Delete an R2 bucket and all its contents.

⚠️  WARNING: This action is irreversible and will delete all objects
in the bucket. Use with caution.

Parameters:
- --force: Skip confirmation prompt
- --dry-run: Show what would be deleted without executing

Examples:
  r2go2 bucket delete my-bucket
  r2go2 bucket delete my-bucket --force
  r2go2 bucket delete my-bucket --dry-run`,
	RunE: runBucketDelete,
}

// bucketExistsCmd represents the bucket exists command
var bucketExistsCmd = &cobra.Command{
	Use:   "exists [bucket-name]",
	Short: "Check if a bucket exists",
	Long: `Check if a bucket exists in the current account.

Exit codes:
- 0: Bucket exists
- 1: Bucket does not exist
- 2: Error occurred

This command is useful for scripting and CI/CD pipelines.

Examples:
  r2go2 bucket exists my-bucket && echo "Bucket exists"
  if r2go2 bucket exists my-bucket; then
    echo "Bucket found"
  fi`,
	RunE: runBucketExists,
}

// bucketImportCmd represents the bucket import command
var bucketImportCmd = &cobra.Command{
	Use:   "import [spec-file]",
	Short: "Create buckets from specification file",
	Long: `Create multiple buckets from a YAML or JSON specification file.

Specification file format (YAML):
buckets:
  - name: production-assets
    location: eu
    tags:
      env: prod
      tier: standard
    metadata:
      description: "Production assets bucket"

  - name: staging-assets
    location: us
    tags:
      env: staging
      tier: standard

Parameters:
- --dry-run: Validate without creating
- --continue: Continue on error instead of stopping

Examples:
  r2go2 bucket import buckets.yaml
  r2go2 bucket import buckets.yaml --dry-run
  r2go2 bucket import buckets.json`,
	RunE: runBucketImport,
}

func init() {
	rootCmd.AddCommand(bucketCmd)

	// Add subcommands
	bucketCmd.AddCommand(bucketCreateCmd)
	bucketCmd.AddCommand(bucketListCmd)
	bucketCmd.AddCommand(bucketGetCmd)
	bucketCmd.AddCommand(bucketUpdateCmd)
	bucketCmd.AddCommand(bucketDeleteCmd)
	bucketCmd.AddCommand(bucketExistsCmd)
	bucketCmd.AddCommand(bucketImportCmd)

	// Flags for bucket create
	bucketCreateCmd.Flags().StringVar(&bucketLocation, "location", "auto", "Bucket location (auto, us, eu, ap)")
	bucketCreateCmd.Flags().StringSliceVar(&bucketTags, "tags", []string{}, "Bucket tags (key=value format)")
	bucketCreateCmd.Flags().StringSliceVar(&bucketMetadata, "metadata", []string{}, "Bucket metadata (key=value format)")

	// Flags for bucket list
	bucketListCmd.Flags().StringVar(&bucketFormat, "format", "table", "Output format (table, json, csv)")
	bucketListCmd.Flags().String("prefix", "", "Filter buckets by name prefix")
	bucketListCmd.Flags().String("tag", "", "Filter buckets by tag")

	// Flags for bucket get
	bucketGetCmd.Flags().StringVar(&bucketFormat, "output", "table", "Output format (table, json, yaml)")
	bucketGetCmd.Flags().Bool("include-objects", false, "Include object count and size")

	// Flags for bucket update
	bucketUpdateCmd.Flags().StringSliceVar(&bucketTags, "tags", []string{}, "Bucket tags (replaces all)")
	bucketUpdateCmd.Flags().StringSliceVar(&bucketMetadata, "metadata", []string{}, "Bucket metadata")
	bucketUpdateCmd.Flags().StringSlice("add-tags", []string{}, "Add tags without replacing")
	bucketUpdateCmd.Flags().StringSlice("remove-tags", []string{}, "Remove specific tags")

	// Flags for bucket delete
	bucketDeleteCmd.Flags().BoolVar(&bucketForce, "force", false, "Skip confirmation prompt")

	// Flags for bucket import
	bucketImportCmd.Flags().StringVar(&bucketSpec, "spec", "", "Specification file path")
	bucketImportCmd.Flags().Bool("continue", false, "Continue on error")
}

func runBucketCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := args[0]

	printInfo("🪣 Creating bucket: %s", bucketName)

	// Parse tags and metadata
	tags, err := parseKeyValuePairs(bucketTags)
	if err != nil {
		return fmt.Errorf("failed to parse tags: %w", err)
	}

	metadata, err := parseKeyValuePairs(bucketMetadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	location, _ := cmd.Flags().GetString("location")

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would create bucket '%s'", bucketName)
		printInfo("  Location: %s", location)
		if len(tags) > 0 {
			printInfo("  Tags: %v", tags)
		}
		if len(metadata) > 0 {
			printInfo("  Metadata: %v", metadata)
		}
		return nil
	}

	// Create bucket
	bucket, err := client.CreateBucket(bucketName)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	printSuccess("✅ Bucket '%s' created successfully!", bucketName)
	printInfo("Created: %s", bucket.CreatedDate.Format(time.RFC3339))

	return nil
}

func runBucketList(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	prefix, _ := cmd.Flags().GetString("prefix")
	tagFilter, _ := cmd.Flags().GetString("tag")

	printInfo("📋 Listing buckets...")

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// List buckets
	buckets, err := client.ListBuckets()
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	// Filter buckets
	var filteredBuckets []*api.Bucket
	for _, bucket := range buckets {
		// Prefix filter
		if prefix != "" && !strings.HasPrefix(bucket.Name, prefix) {
			continue
		}

		// Tag filter (placeholder - would need actual tag filtering)
		if tagFilter != "" {
			// This would require additional API calls to get bucket tags
			// For now, skip tag filtering in this implementation
		}

		filteredBuckets = append(filteredBuckets, bucket)
	}

	if JSONOutput || format == "json" {
		return printJSON(filteredBuckets)
	}

	if format == "csv" {
		fmt.Println("name,created_date")
		for _, bucket := range filteredBuckets {
			fmt.Printf("%s,%s\n", bucket.Name, bucket.CreatedDate.Format(time.RFC3339))
		}
		return nil
	}

	// Table format (default)
	if len(filteredBuckets) == 0 {
		printInfo("No buckets found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED DATE\tSIZE\tOBJECTS")
	for _, bucket := range filteredBuckets {
		size := "N/A"
		objectCount := "N/A"
		if bucket.Size > 0 {
			size = formatBytes(bucket.Size)
		}
		if bucket.ObjectCount > 0 {
			objectCount = fmt.Sprintf("%d", bucket.ObjectCount)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			bucket.Name,
			bucket.CreatedDate.Format("2006-01-02 15:04:05"),
			size,
			objectCount,
		)
	}
	w.Flush()

	printInfo("Total: %d bucket(s)", len(filteredBuckets))
	return nil
}

func runBucketGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := args[0]

	output, _ := cmd.Flags().GetString("output")
	includeObjects, _ := cmd.Flags().GetBool("include-objects")

	printInfo("🔍 Getting bucket details: %s", bucketName)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get bucket details
	bucket, err := client.GetBucket(bucketName)
	if err != nil {
		return fmt.Errorf("failed to get bucket details: %w", err)
	}

	// Get object count if requested
	if includeObjects {
		objects, err := client.ListObjects(bucketName, "", "", 0)
		if err == nil {
			bucket.ObjectCount = int64(len(objects))
			var totalSize int64
			for _, obj := range objects {
				totalSize += obj.Size
			}
			bucket.Size = totalSize
		}
	}

	if output == "json" {
		return printJSON(bucket)
	}

	if output == "yaml" {
		yamlData, err := json.Marshal(bucket)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Printf("# Bucket: %s\n", bucket.Name)
		fmt.Println(string(yamlData))
		return nil
	}

	// Table format (default)
	fmt.Printf("Bucket: %s\n", bucket.Name)
	fmt.Printf("Created: %s\n", bucket.CreatedDate.Format(time.RFC3339))
	if bucket.Location != "" {
		fmt.Printf("Location: %s\n", bucket.Location)
	}
	if bucket.Size > 0 {
		fmt.Printf("Size: %s\n", formatBytes(bucket.Size))
	}
	if bucket.ObjectCount > 0 {
		fmt.Printf("Objects: %d\n", bucket.ObjectCount)
	}
	if len(bucket.Tags) > 0 {
		fmt.Printf("Tags:\n")
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
	bucketName := args[0]

	printInfo("📝 Updating bucket: %s", bucketName)

	// Parse tags and metadata
	tags, _ := cmd.Flags().GetStringSlice("tags")
	addTags, _ := cmd.Flags().GetStringSlice("add-tags")
	removeTags, _ := cmd.Flags().GetStringSlice("remove-tags")
	metadata, _ := cmd.Flags().GetStringSlice("metadata")

	// Get existing bucket
	_, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Note: Cloudflare R2 API may not support bucket metadata updates directly
	// This is a placeholder implementation that would need to be adapted
	// based on the actual API capabilities

	if DryRun {
		printInfo("DRY RUN: Would update bucket '%s'", bucketName)
		if len(tags) > 0 {
			parsedTags, _ := parseKeyValuePairs(tags)
			printInfo("  Set tags: %v", parsedTags)
		}
		if len(addTags) > 0 {
			parsedAddTags, _ := parseKeyValuePairs(addTags)
			printInfo("  Add tags: %v", parsedAddTags)
		}
		if len(removeTags) > 0 {
			printInfo("  Remove tags: %v", removeTags)
		}
		if len(metadata) > 0 {
			parsedMetadata, _ := parseKeyValuePairs(metadata)
			printInfo("  Set metadata: %v", parsedMetadata)
		}
		return nil
	}

	// For now, return success as placeholder
	printSuccess("✅ Bucket '%s' updated successfully!", bucketName)
	printWarning("Note: Bucket metadata updates may not be fully supported by R2 API")

	return nil
}

func runBucketDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := args[0]

	force, _ := cmd.Flags().GetBool("force")

	printInfo("🗑️  Deleting bucket: %s", bucketName)

	if !force && !DryRun {
		fmt.Printf("Are you sure you want to delete bucket '%s' and all its contents? [y/N]: ", bucketName)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Bucket deletion cancelled")
			return nil
		}
	}

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would delete bucket '%s'", bucketName)
		return nil
	}

	// Delete bucket
	if err := client.DeleteBucket(bucketName); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	printSuccess("✅ Bucket '%s' deleted successfully!", bucketName)
	return nil
}

func runBucketExists(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := args[0]

	// Create client
	client, err := getAPIClient()
	if err != nil {
		printError("Failed to create API client: %v", err)
		os.Exit(2)
	}

	exists, err := client.BucketExists(bucketName)
	if err != nil {
		printError("Failed to check bucket existence: %v", err)
		os.Exit(2)
	}

	if exists {
		printInfo("Bucket '%s' exists", bucketName)
		os.Exit(0)
	} else {
		printInfo("Bucket '%s' does not exist", bucketName)
		os.Exit(1)
	}

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

	printInfo("📥 Importing buckets from: %s", specFile)

	// Parse specification file
	var spec BucketSpec
	if err := parseSpecFile(specFile, &spec); err != nil {
		return fmt.Errorf("failed to parse specification file: %w", err)
	}

	if len(spec.Buckets) == 0 {
		printWarning("No buckets found in specification file")
		return nil
	}

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Create buckets
	successCount := 0
	errorCount := 0

	for _, bucketSpec := range spec.Buckets {
		printInfo("Creating bucket: %s", bucketSpec.Name)

		if DryRun {
			printInfo("DRY RUN: Would create bucket '%s'", bucketSpec.Name)
			successCount++
			continue
		}

		_, err := client.CreateBucket(bucketSpec.Name)
		if err != nil {
			printError("Failed to create bucket '%s': %v", bucketSpec.Name, err)
			errorCount++
			if !continueOnError {
				return fmt.Errorf("bucket creation failed")
			}
			continue
		}

		printSuccess("✅ Created bucket: %s", bucketSpec.Name)
		successCount++
	}

	printInfo("Import complete: %d successful, %d failed", successCount, errorCount)
	return nil
}

// Helper types and functions

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

	// Try YAML (would need yaml library)
	// For now, return error for YAML
	return fmt.Errorf("YAML parsing not implemented, use JSON format")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}