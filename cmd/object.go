/*
Package cmd provides object management commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
)

// objectCmd represents the object command
var objectCmd = &cobra.Command{
	Use:   "object",
	Short: "Manage R2 objects",
	Long: `Object management operations for Cloudflare R2.

Commands:
  ls         List objects in a bucket
  get        Download an object
  put        Upload an object
  delete     Delete an object
  copy       Copy an object
  head       Get object metadata
  search     Search for objects
  batch      Batch operations

Examples:
  r2go2 object ls my-bucket --prefix=images/
  r2go2 object get my-bucket file.txt --output=local.txt
  r2go2 object put my-bucket file.txt --key=remote/file.txt
  r2go2 object delete my-bucket file.txt`,
}

var (
	objectPrefix     string
	objectDelimiter  string
	objectMaxKeys    int32
	objectRecursive  bool
	objectOutput     string
	objectRangeStart int64
	objectRangeEnd   int64
	objectKey        string
	objectContentType string
	objectCacheControl string
	objectMetadata   []string
	objectProgress   bool
	objectSpec       string
	objectPattern    string
	objectSearchType string
)

// objectListCmd represents the object list command
var objectListCmd = &cobra.Command{
	Use:   "ls [bucket-name]",
	Short: "List objects in a bucket",
	Long: `List objects in a bucket with filtering and formatting options.

Filtering options:
- --prefix: Filter by object key prefix
- --delimiter: Group objects by delimiter (for folder-like structure)
- --max-keys: Limit number of results
- --recursive: List all objects recursively

Output formats:
- table (default): Human-readable table
- json: Machine-readable JSON
- csv: Comma-separated values

Examples:
  r2go2 object ls my-bucket
  r2go2 object ls my-bucket --prefix=images/ --recursive
  r2go2 object ls my-bucket --delimiter=/ --json`,
	RunE: runObjectList,
}

// objectGetCmd represents the object get command
var objectGetCmd = &cobra.Command{
	Use:   "get [bucket-name] [object-key]",
	Short: "Download an object",
	Long: `Download an object from R2 to local storage.

Options:
- --output: Local file path (defaults to object key)
- --range: Download specific byte range
- --progress: Show progress bar

Examples:
  r2go2 object get my-bucket file.txt
  r2go2 object get my-bucket file.txt --output=local.txt
  r2go2 object get my-bucket large.zip --range=0-1023`,
	RunE: runObjectGet,
}

// objectPutCmd represents the object put command
var objectPutCmd = &cobra.Command{
	Use:   "put [bucket-name] [local-path]",
	Short: "Upload an object",
	Long: `Upload a local file to R2 with metadata options.

Options:
- --key: Remote object key (defaults to filename)
- --content-type: Content type (auto-detected if not specified)
- --cache-control: Cache control header
- --metadata: Additional metadata (key=value format)
- --progress: Show progress bar

Examples:
  r2go2 object put my-bucket file.txt
  r2go2 object put my-bucket file.txt --key=remote/file.txt
  r2go2 object put my-bucket image.jpg --content-type=image/jpeg --metadata=author=admin`,
	RunE: runObjectPut,
}

// objectDeleteCmd represents the object delete command
var objectDeleteCmd = &cobra.Command{
	Use:   "delete [bucket-name] [object-key]",
	Short: "Delete an object",
	Long: `Delete an object from R2.

Examples:
  r2go2 object delete my-bucket file.txt
  r2go2 object delete my-bucket folder/file.txt`,
	RunE: runObjectDelete,
}

// objectCopyCmd represents the object copy command
var objectCopyCmd = &cobra.Command{
	Use:   "copy [src-bucket]/[src-key] [dst-bucket]/[dst-key]",
	Short: "Copy an object",
	Long: `Copy an object within or between buckets.

Source and destination format: bucket/key

Options:
- --metadata: Update metadata during copy
- --content-type: Update content type

Examples:
  r2go2 object copy my-bucket/file.txt my-bucket/backup.txt
  r2go2 object copy source-bucket/img.jpg dest-bucket/images/img.jpg`,
	RunE: runObjectCopy,
}

// objectHeadCmd represents the object head command
var objectHeadCmd = &cobra.Command{
	Use:   "head [bucket-name] [object-key]",
	Short: "Get object metadata",
	Long: `Get object metadata without downloading the content.

Shows:
- Object size
- Last modified date
- ETag
- Content type
- Custom metadata
- Cache control

Output formats:
- table (default): Human-readable table
- json: Machine-readable JSON

Examples:
  r2go2 object head my-bucket file.txt
  r2go2 object head my-bucket file.txt --json`,
	RunE: runObjectHead,
}

// objectSearchCmd represents the object search command
var objectSearchCmd = &cobra.Command{
	Use:   "search [bucket-name] [query]",
	Short: "Search for objects",
	Long: `Search for objects in a bucket using different patterns.

Search types:
- prefix: Match key prefix (default)
- regex: Regular expression match
- glob: Glob pattern match
- exact: Exact key match

Examples:
  r2go2 object search my-bucket ".jpg"
  r2go2 object search my-bucket "image-*" --type=glob
  r2go2 object search my-bucket ".*\\.png$" --type=regex`,
	RunE: runObjectSearch,
}

// objectBatchCmd represents the object batch command
var objectBatchCmd = &cobra.Command{
	Use:   "batch [bucket-name] [spec-file]",
	Short: "Perform batch operations",
	Long: `Perform multiple operations from a specification file.

Specification file format (JSON):
{
  "operations": [
    {
      "action": "upload",
      "local_path": "local/file.txt",
      "object_key": "remote/file.txt"
    },
    {
      "action": "delete",
      "object_key": "old-file.txt"
    }
  ]
}

Examples:
  r2go2 object batch my-bucket operations.json
  r2go2 object batch my-bucket operations.json --dry-run`,
	RunE: runObjectBatch,
}

func init() {
	rootCmd.AddCommand(objectCmd)

	// Add subcommands
	objectCmd.AddCommand(objectListCmd)
	objectCmd.AddCommand(objectGetCmd)
	objectCmd.AddCommand(objectPutCmd)
	objectCmd.AddCommand(objectDeleteCmd)
	objectCmd.AddCommand(objectCopyCmd)
	objectCmd.AddCommand(objectHeadCmd)
	objectCmd.AddCommand(objectSearchCmd)
	objectCmd.AddCommand(objectBatchCmd)

	// Flags for object list
	objectListCmd.Flags().StringVar(&objectPrefix, "prefix", "", "Object key prefix filter")
	objectListCmd.Flags().StringVar(&objectDelimiter, "delimiter", "", "Delimiter for grouping")
	objectListCmd.Flags().Int32Var(&objectMaxKeys, "max-keys", 0, "Maximum number of keys to return")
	objectListCmd.Flags().BoolVar(&objectRecursive, "recursive", false, "List all objects recursively")

	// Flags for object get
	objectGetCmd.Flags().StringVar(&objectOutput, "output", "", "Local output file path")
	objectGetCmd.Flags().Int64Var(&objectRangeStart, "range-start", -1, "Range start byte")
	objectGetCmd.Flags().Int64Var(&objectRangeEnd, "range-end", -1, "Range end byte")

	// Flags for object put
	objectPutCmd.Flags().StringVar(&objectKey, "key", "", "Remote object key")
	objectPutCmd.Flags().StringVar(&objectContentType, "content-type", "", "Content type")
	objectPutCmd.Flags().StringVar(&objectCacheControl, "cache-control", "", "Cache control header")
	objectPutCmd.Flags().StringSliceVar(&objectMetadata, "metadata", []string{}, "Additional metadata (key=value)")
	objectPutCmd.Flags().BoolVar(&objectProgress, "progress", true, "Show progress bar")

	// Flags for object copy
	objectCopyCmd.Flags().StringSliceVar(&objectMetadata, "metadata", []string{}, "New metadata (key=value)")
	objectCopyCmd.Flags().StringVar(&objectContentType, "content-type", "", "New content type")

	// Flags for object head
	objectHeadCmd.Flags().StringVar(&objectOutput, "output", "table", "Output format (table, json)")

	// Flags for object search
	objectSearchCmd.Flags().StringVar(&objectSearchType, "type", "prefix", "Search type (prefix, regex, glob, exact)")

	// Flags for object batch
	objectBatchCmd.Flags().Bool("continue", false, "Continue on error")
}

func runObjectList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucketName := args[0]

	prefix, _ := cmd.Flags().GetString("prefix")
	delimiter, _ := cmd.Flags().GetString("delimiter")
	maxKeys, _ := cmd.Flags().GetInt32("max-keys")
	_, _ = cmd.Flags().GetBool("recursive")

	printInfo("📋 Listing objects in bucket: %s", bucketName)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// List objects
	result, err := client.ListObjects(context.Background(), bucketName, prefix, delimiter, int32(maxKeys))
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}
	objects := result.Items

	if JSONOutput {
		return printJSON(objects)
	}

	// Table format
	if len(objects) == 0 {
		printInfo("No objects found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tSIZE\tLAST MODIFIED\tETAG")
	for _, obj := range objects {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			obj.Key,
			utils.FormatBytes(obj.Size),
			obj.LastModified.Format("2006-01-02 15:04:05"),
			etagDisplay(obj.ETag),
		)
	}
	w.Flush()

	printInfo("Total: %d object(s)", len(objects))
	return nil
}

func runObjectGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and object key are required")
	}
	bucketName := args[0]
	objectKey := args[1]

	output, _ := cmd.Flags().GetString("output")

	if output == "" {
		output = filepath.Base(objectKey)
	}

	printInfo("⬇️  Downloading object: %s/%s", bucketName, objectKey)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get object
	obj, err := client.GetObject(context.Background(), bucketName, objectKey)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Content.Close()

	// Create output file
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if Verbose {
		printInfo("Size: %s", utils.FormatBytes(obj.Size))
	}

	size, err := io.Copy(file, obj.Content)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	printSuccess("✅ Downloaded: %s (%s)", output, utils.FormatBytes(size))
	return nil
}

func runObjectPut(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and local path are required")
	}
	bucketName := args[0]
	localPath := args[1]

	key, _ := cmd.Flags().GetString("key")
	contentType, _ := cmd.Flags().GetString("content-type")
	cacheControl, _ := cmd.Flags().GetString("cache-control")
	metadata, _ := cmd.Flags().GetStringSlice("metadata")

	if key == "" {
		key = filepath.Base(localPath)
	}

	printInfo("⬆️  Uploading: %s -> %s/%s", localPath, bucketName, key)

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to access local file: %w", err)
	}

	metadataMap, err := parseKeyValuePairs(metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would upload file")
		printInfo("  Bucket: %s", bucketName)
		printInfo("  Key: %s", key)
		printInfo("  Size: %s", utils.FormatBytes(fileInfo.Size()))
		if contentType != "" {
			printInfo("  Content-Type: %s", contentType)
		}
		if cacheControl != "" {
			printInfo("  Cache-Control: %s", cacheControl)
		}
		if len(metadataMap) > 0 {
			printInfo("  Metadata: %v", metadataMap)
		}
		return nil
	}

	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	opts := []r2go2.UploadOption{}
	if contentType != "" {
		opts = append(opts, r2go2.WithContentType(contentType))
	}
	if cacheControl != "" {
		opts = append(opts, r2go2.WithUploadCacheControl(cacheControl))
	}
	if len(metadataMap) > 0 {
		opts = append(opts, r2go2.WithMetadata(metadataMap))
	}

	result, err := client.Upload(context.Background(), bucketName, key, file, fileInfo.Size(), opts...)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	printSuccess("✅ Uploaded successfully!")
	printInfo("Key: %s", result.Key)
	printInfo("Size: %s", utils.FormatBytes(result.Size))
	printInfo("ETag: %s", result.ETag)
	return nil
}

func runObjectDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and object key are required")
	}
	bucketName := args[0]
	objectKey := args[1]

	printInfo("🗑️  Deleting object: %s/%s", bucketName, objectKey)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would delete object: %s/%s", bucketName, objectKey)
		return nil
	}

	// Delete object
	if err := client.DeleteObject(context.Background(), bucketName, objectKey); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	printSuccess("✅ Object deleted successfully!")
	return nil
}

func runObjectCopy(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("source and destination are required")
	}

	src := args[0]
	dst := args[1]

	srcParts := strings.SplitN(src, "/", 2)
	if len(srcParts) != 2 {
		return fmt.Errorf("invalid source format, expected: bucket/key")
	}

	dstParts := strings.SplitN(dst, "/", 2)
	if len(dstParts) != 2 {
		return fmt.Errorf("invalid destination format, expected: bucket/key")
	}

	srcBucket, srcKey := srcParts[0], srcParts[1]
	dstBucket, dstKey := dstParts[0], dstParts[1]

	printInfo("📋 Copying object: %s/%s -> %s/%s", srcBucket, srcKey, dstBucket, dstKey)

	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would copy object")
		printInfo("  Source: %s/%s", srcBucket, srcKey)
		printInfo("  Destination: %s/%s", dstBucket, dstKey)
		return nil
	}

	result, err := client.CopyObject(context.Background(), srcBucket, srcKey, dstBucket, dstKey)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	printSuccess("✅ Object copied successfully!")
	printInfo("Source: %s/%s", srcBucket, srcKey)
	printInfo("Destination: %s/%s", dstBucket, dstKey)
	printInfo("ETag: %s", result.ETag)
	return nil
}

func runObjectHead(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and object key are required")
	}
	bucketName := args[0]
	objectKey := args[1]

	output, _ := cmd.Flags().GetString("output")

	printInfo("🔍 Getting object metadata: %s/%s", bucketName, objectKey)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get object metadata
	obj, err := client.HeadObject(context.Background(), bucketName, objectKey)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}

	if output == "json" {
		return printJSON(obj)
	}

	// Table format
	fmt.Printf("Object: %s/%s\n", bucketName, objectKey)
	fmt.Printf("Size: %s\n", utils.FormatBytes(obj.Size))
	fmt.Printf("Last Modified: %s\n", obj.LastModified.Format(time.RFC3339))
	fmt.Printf("ETag: %s\n", obj.ETag)
	fmt.Printf("Cache-Control: %s\n", obj.CacheControl)

	if len(obj.Metadata) > 0 {
		fmt.Printf("Metadata:\n")
		for key, value := range obj.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

func runObjectSearch(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and search query are required")
	}
	bucketName := args[0]
	query := args[1]

	searchType, _ := cmd.Flags().GetString("type")

	printInfo("🔍 Searching in bucket %s: %s (type: %s)", bucketName, query, searchType)

	// Create client
	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// List all objects (would be more efficient with server-side filtering)
	listResult, err := client.ListObjects(context.Background(), bucketName, "", "", 0)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}
	objects := listResult.Items

	// Compile regex once if needed (before the loop)
	var regex *regexp.Regexp
	if searchType == "regex" {
		regex, err = regexp.Compile(query)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Filter objects based on search type
	var results []*r2go2.Object
	for _, obj := range objects {
		var match bool
		switch searchType {
		case "prefix":
			match = strings.HasPrefix(obj.Key, query)
		case "exact":
			match = obj.Key == query
		case "glob":
			var matchErr error
			match, matchErr = filepath.Match(query, obj.Key)
			if matchErr != nil {
				return fmt.Errorf("invalid glob pattern: %w", matchErr)
			}
		case "regex":
			match = regex.MatchString(obj.Key)
		default:
			match = strings.HasPrefix(obj.Key, query)
		}

		if match {
			results = append(results, obj)
		}
	}

	if JSONOutput {
		return printJSON(results)
	}

	// Table format
	if len(results) == 0 {
		printInfo("No objects found matching: %s", query)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tSIZE\tLAST MODIFIED")
	for _, obj := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			obj.Key,
			utils.FormatBytes(obj.Size),
			obj.LastModified.Format("2006-01-02 15:04:05"),
		)
	}
	w.Flush()

	printInfo("Found: %d object(s) matching '%s'", len(results), query)
	return nil
}

func runObjectBatch(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and spec file are required")
	}
	bucketName := args[0]
	specFile := args[1]

	continueOnError, _ := cmd.Flags().GetBool("continue")

	printInfo("📦 Processing batch operations from: %s", specFile)

	var spec BatchSpec
	if err := parseBatchSpec(specFile, &spec); err != nil {
		return fmt.Errorf("failed to parse batch spec: %w", err)
	}

	if len(spec.Operations) == 0 {
		printWarning("No operations found in specification file")
		return nil
	}

	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	successCount := 0
	errorCount := 0

	for i, op := range spec.Operations {
		printInfo("Operation %d/%d: %s", i+1, len(spec.Operations), op.Action)

		if DryRun {
			printInfo("DRY RUN: Would execute operation")
			successCount++
			continue
		}

		switch op.Action {
		case "upload":
			if op.LocalPath == "" || op.ObjectKey == "" {
				printError("Upload operation requires local_path and object_key")
				errorCount++
				continue
			}
			file, err := os.Open(op.LocalPath)
			if err != nil {
				printError("Failed to open file: %v", err)
				errorCount++
				if !continueOnError {
					return err
				}
				continue
			}
			defer file.Close()

			info, _ := file.Stat()
			_, err = client.Upload(context.Background(), bucketName, op.ObjectKey, file, info.Size())
			if err != nil {
				printError("Upload failed: %v", err)
				errorCount++
				if !continueOnError {
					return err
				}
				continue
			}
			printSuccess("Uploaded: %s", op.ObjectKey)
			successCount++

		case "delete":
			if op.ObjectKey == "" {
				printError("Delete operation requires object_key")
				errorCount++
				continue
			}
			err := client.DeleteObject(context.Background(), bucketName, op.ObjectKey)
			if err != nil {
				printError("Delete failed: %v", err)
				errorCount++
				if !continueOnError {
					return err
				}
				continue
			}
			printSuccess("Deleted: %s", op.ObjectKey)
			successCount++

		case "copy":
			if op.ObjectKey == "" || op.DestinationKey == "" {
				printError("Copy operation requires object_key and destination_key")
				errorCount++
				continue
			}
			_, err := client.CopyObject(context.Background(), bucketName, op.ObjectKey, bucketName, op.DestinationKey)
			if err != nil {
				printError("Copy failed: %v", err)
				errorCount++
				if !continueOnError {
					return err
				}
				continue
			}
			printSuccess("Copied: %s -> %s", op.ObjectKey, op.DestinationKey)
			successCount++

		default:
			printError("Unknown operation: %s", op.Action)
			errorCount++
			if !continueOnError {
				return fmt.Errorf("invalid operation: %s", op.Action)
			}
		}
	}

	printInfo("Batch operations complete: %d successful, %d failed", successCount, errorCount)
	return nil
}

// Helper types and functions

type BatchSpec struct {
	Operations []BatchOperation `json:"operations"`
}

type BatchOperation struct {
	Action         string `json:"action"`
	LocalPath      string `json:"local_path,omitempty"`
	ObjectKey      string `json:"object_key,omitempty"`
	DestinationKey string `json:"destination_key,omitempty"`
}

func parseBatchSpec(filename string, spec *BatchSpec) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read batch spec file: %w", err)
	}

	return json.Unmarshal(data, spec)
}

func etagDisplay(etag string) string {
	if len(etag) > 16 {
		return etag[:16] + "..."
	}
	return etag
}

// getAPIClient creates an R2 client using the pkg/r2go2 library.
func getAPIClient() (r2go2.R2Client, error) {
	return r2go2.NewClient(
		r2go2.WithAccountID(AccountID),
		r2go2.WithAPIToken(APIToken),
	)
}