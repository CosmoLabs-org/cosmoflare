package cmd

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize local directories with R2 buckets",
	Long: `Rsync-like synchronization between local directories and Cloudflare R2 buckets.

Compares local filesystem state against R2 object listing and transfers only
changed files. Supports upload (local -> R2) and download (R2 -> local).

Subcommands:
  up      Upload local directory to R2 bucket
  down    Download R2 bucket to local directory

Comparison modes:
  Default: compares file size and modification time
  --checksum: compares MD5 checksums (slower but accurate)

Examples:
  cosmoflare sync up ./dist my-bucket
  cosmoflare sync up ./dist my-bucket/assets --delete
  cosmoflare sync down my-bucket ./backup
  cosmoflare sync down my-bucket/prefix ./local --exclude "*.tmp"
  cosmoflare sync up . my-bucket --checksum --progress --dry-run`,
}

var (
	syncFlagDelete   bool
	syncFlagExclude  []string
	syncFlagInclude  []string
	syncFlagChecksum bool
	syncFlagProgress bool
)

var syncUpCmd = &cobra.Command{
	Use:   "up <local-dir> <bucket>[/prefix]",
	Short: "Upload local directory to R2 bucket",
	Long: `Upload a local directory to an R2 bucket, transferring only new or changed files.

The local directory is scanned recursively. Each file is compared against the
corresponding R2 object. Only files that are new, modified (by size/mtime or
checksum), or deleted (with --delete) are transferred.

Arguments:
  <local-dir>        Local directory to upload from (e.g., ".", "./dist", "/path/to/dir")
  <bucket>[/prefix]  R2 bucket name with optional key prefix (e.g., "my-bucket", "my-bucket/assets")

Examples:
  cosmoflare sync up ./dist my-bucket
  cosmoflare sync up ./dist my-bucket/static/v2
  cosmoflare sync up . my-bucket --delete --exclude "*.log" --exclude ".git/*"
  cosmoflare sync up ./build my-bucket --checksum --progress
  cosmoflare sync up ./public my-bucket/site --dry-run --json`,
	RunE: runSyncUp,
}

var syncDownCmd = &cobra.Command{
	Use:   "down <bucket>[/prefix] <local-dir>",
	Short: "Download R2 bucket to local directory",
	Long: `Download objects from an R2 bucket to a local directory, transferring only
new or changed files.

Remote objects are listed under the given prefix. Each object is compared
against the corresponding local file. Only objects that are new, modified,
or deleted locally (with --delete) are transferred.

Arguments:
  <bucket>[/prefix]  R2 bucket name with optional key prefix (e.g., "my-bucket", "my-bucket/assets")
  <local-dir>        Local directory to download to (e.g., "./backup", "/path/to/dir")

Examples:
  cosmoflare sync down my-bucket ./backup
  cosmoflare sync down my-bucket/logs ./local-logs
  cosmoflare sync down my-bucket ./mirror --delete
  cosmoflare sync down my-bucket/data ./data --exclude "*.tmp" --progress
  cosmoflare sync down my-bucket ./restore --checksum --dry-run --json`,
	RunE: runSyncDown,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.AddCommand(syncUpCmd)
	syncCmd.AddCommand(syncDownCmd)

	// Shared flags for sync up
	syncUpCmd.Flags().BoolVar(&syncFlagDelete, "delete", false, "Delete destination files not present at source")
	syncUpCmd.Flags().StringArrayVar(&syncFlagExclude, "exclude", nil, "Exclude files matching glob pattern (can be repeated)")
	syncUpCmd.Flags().StringArrayVar(&syncFlagInclude, "include", nil, "Include only files matching glob pattern (can be repeated)")
	syncUpCmd.Flags().BoolVar(&syncFlagChecksum, "checksum", false, "Compare files by MD5 checksum instead of size/mtime")
	syncUpCmd.Flags().BoolVar(&syncFlagProgress, "progress", false, "Show progress for each file operation")

	// Shared flags for sync down
	syncDownCmd.Flags().BoolVar(&syncFlagDelete, "delete", false, "Delete local files not present in bucket")
	syncDownCmd.Flags().StringArrayVar(&syncFlagExclude, "exclude", nil, "Exclude files matching glob pattern (can be repeated)")
	syncDownCmd.Flags().StringArrayVar(&syncFlagInclude, "include", nil, "Include only files matching glob pattern (can be repeated)")
	syncDownCmd.Flags().BoolVar(&syncFlagChecksum, "checksum", false, "Compare files by MD5 checksum instead of size/mtime")
	syncDownCmd.Flags().BoolVar(&syncFlagProgress, "progress", false, "Show progress for each file operation")
}

// parseBucketPrefix splits "bucket/prefix/path" into bucket and prefix.
// The prefix always ends with "/" if non-empty.
func parseBucketPrefix(input string) (bucket, prefix string) {
	parts := strings.SplitN(input, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
	}
	return
}

// scanLocalDir walks a directory and returns LocalFileInfo for each regular file.
func scanLocalDir(dir string, checksum bool) ([]cosmoflare.LocalFileInfo, error) {
	var files []cosmoflare.LocalFileInfo

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}
		// Normalize to forward slashes for R2 key compatibility
		relPath = filepath.ToSlash(relPath)

		lfi := cosmoflare.LocalFileInfo{
			RelPath: relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		if checksum {
			hash, err := md5File(path)
			if err != nil {
				return fmt.Errorf("failed to compute checksum for %s: %w", path, err)
			}
			lfi.Checksum = hash
		}

		files = append(files, lfi)
		return nil
	})

	return files, err
}

// md5File computes the MD5 hex digest of a file.
func md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func runSyncUp(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("local directory is required\n\nUsage: cosmoflare sync up <local-dir> <bucket>[/prefix]")
	}
	if len(args) < 2 {
		return fmt.Errorf("bucket is required\n\nUsage: cosmoflare sync up <local-dir> <bucket>[/prefix]")
	}

	localDir := args[0]
	bucket, prefix := parseBucketPrefix(args[1])

	// Validate local directory exists
	dirInfo, err := os.Stat(localDir)
	if err != nil {
		return fmt.Errorf("cannot access local directory %q: %w", localDir, err)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("%q is not a directory", localDir)
	}

	// Scan local files
	localFiles, err := scanLocalDir(localDir, syncFlagChecksum)
	if err != nil {
		return fmt.Errorf("failed to scan local directory: %w", err)
	}

	// Create R2-backed storage backend
	backend, err := newR2StorageBackend(AccountID, APIToken)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	svc := cosmoflare.NewSyncService(backend)

	// Generate plan
	plan, err := svc.Plan(context.Background(), cosmoflare.SyncPlanInput{
		Direction:  cosmoflare.SyncUp,
		Bucket:     bucket,
		Prefix:     prefix,
		LocalDir:   localDir,
		LocalFiles: localFiles,
		Delete:     syncFlagDelete,
		Exclude:    syncFlagExclude,
		Include:    syncFlagInclude,
		Checksum:   syncFlagChecksum,
	})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to generate sync plan: %v", err))
		}
		return fmt.Errorf("failed to generate sync plan: %w", err)
	}

	// Dry-run: show plan and exit
	if DryRun {
		return printSyncPlan(plan)
	}

	// Set up progress reporting
	if syncFlagProgress {
		svc.OnProgress = func(p cosmoflare.SyncProgress) {
			if !JSONOutput {
				fmt.Printf("[%d/%d] %s %s (%d bytes)\n", p.Current, p.Total, p.Action, p.Key, p.Size)
			}
		}
	}

	// Execute
	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("sync execution failed: %v", err))
		}
		return fmt.Errorf("sync execution failed: %w", err)
	}

	return printSyncResult(plan, result)
}

func runSyncDown(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bucket is required\n\nUsage: cosmoflare sync down <bucket>[/prefix] <local-dir>")
	}
	if len(args) < 2 {
		return fmt.Errorf("local directory is required\n\nUsage: cosmoflare sync down <bucket>[/prefix] <local-dir>")
	}

	bucket, prefix := parseBucketPrefix(args[0])
	localDir := args[1]

	// Scan local files if directory exists
	var localFiles []cosmoflare.LocalFileInfo
	if info, err := os.Stat(localDir); err == nil && info.IsDir() {
		localFiles, err = scanLocalDir(localDir, syncFlagChecksum)
		if err != nil {
			return fmt.Errorf("failed to scan local directory: %w", err)
		}
	}

	// Create R2-backed storage backend
	backend, err := newR2StorageBackend(AccountID, APIToken)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	svc := cosmoflare.NewSyncService(backend)

	// Generate plan
	plan, err := svc.Plan(context.Background(), cosmoflare.SyncPlanInput{
		Direction:  cosmoflare.SyncDown,
		Bucket:     bucket,
		Prefix:     prefix,
		LocalDir:   localDir,
		LocalFiles: localFiles,
		Delete:     syncFlagDelete,
		Exclude:    syncFlagExclude,
		Include:    syncFlagInclude,
		Checksum:   syncFlagChecksum,
	})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to generate sync plan: %v", err))
		}
		return fmt.Errorf("failed to generate sync plan: %w", err)
	}

	// Dry-run: show plan and exit
	if DryRun {
		return printSyncPlan(plan)
	}

	// Set up progress reporting
	if syncFlagProgress {
		svc.OnProgress = func(p cosmoflare.SyncProgress) {
			if !JSONOutput {
				fmt.Printf("[%d/%d] %s %s (%d bytes)\n", p.Current, p.Total, p.Action, p.Key, p.Size)
			}
		}
	}

	// Execute
	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("sync execution failed: %v", err))
		}
		return fmt.Errorf("sync execution failed: %w", err)
	}

	return printSyncResult(plan, result)
}

// printSyncPlan displays the sync plan (used for --dry-run).
func printSyncPlan(plan *cosmoflare.SyncPlan) error {
	if JSONOutput {
		return printSuccessJSON("Sync plan (dry run)", plan)
	}

	printWarning("DRY RUN: No actual changes will be made")
	fmt.Printf("\nSync %s: %s -> %s%s\n\n", plan.Direction, plan.LocalDir, plan.Bucket, prefixDisplay(plan.Prefix))

	for _, op := range plan.Operations {
		symbol := "  "
		switch op.Action {
		case cosmoflare.SyncOpUpload:
			symbol = "+ "
		case cosmoflare.SyncOpDownload:
			symbol = "< "
		case cosmoflare.SyncOpDelete:
			symbol = "- "
		case cosmoflare.SyncOpSkip:
			symbol = "= "
		}
		fmt.Printf("  %s%s (%s)\n", symbol, op.Key, op.Reason)
	}

	fmt.Printf("\nSummary: %d upload(s), %d download(s), %d delete(s), %d skip(s)\n",
		plan.Summary.Uploads, plan.Summary.Downloads, plan.Summary.Deletes, plan.Summary.Skips)
	return nil
}

// printSyncResult displays the outcome of a sync execution.
func printSyncResult(plan *cosmoflare.SyncPlan, result *cosmoflare.SyncResult) error {
	if JSONOutput {
		return printSuccessJSON("Sync completed", map[string]interface{}{
			"direction": plan.Direction.String(),
			"bucket":    plan.Bucket,
			"prefix":    plan.Prefix,
			"succeeded": result.Succeeded,
			"failed":    result.Failed,
			"skipped":   result.Skipped,
			"errors":    result.Errors,
		})
	}

	if result.Failed > 0 {
		printWarning("Sync completed with errors")
	} else {
		printSuccess("Sync completed successfully!")
	}

	printInfo("Direction: %s", plan.Direction)
	printInfo("Bucket: %s%s", plan.Bucket, prefixDisplay(plan.Prefix))
	printInfo("Succeeded: %d, Failed: %d, Skipped: %d", result.Succeeded, result.Failed, result.Skipped)

	if len(result.Errors) > 0 {
		printError("Errors:")
		for _, e := range result.Errors {
			printError("  - %s", e)
		}
	}

	return nil
}

func prefixDisplay(prefix string) string {
	if prefix == "" {
		return ""
	}
	return "/" + strings.TrimSuffix(prefix, "/")
}

// r2StorageBackend implements StorageBackend using a cosmoflare R2Client.
type r2StorageBackend struct {
	client cosmoflare.R2Client
}

func newR2StorageBackend(accountID, apiToken string) (*r2StorageBackend, error) {
	client, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(accountID),
		cosmoflare.WithAPIToken(apiToken),
	)
	if err != nil {
		return nil, err
	}
	return &r2StorageBackend{client: client}, nil
}

// listAllR2Objects lists every object under prefix by following the
// continuation token returned by each ListObjects page. A single call
// returns at most 1000 items, so stopping at the first page silently
// drops every object beyond the first 1000.
func listAllR2Objects(ctx context.Context, client cosmoflare.R2Client, bucket, prefix string) ([]cosmoflare.ObjectInfo, error) {
	var allObjects []cosmoflare.ObjectInfo
	continuationToken := ""
	for {
		result, err := client.ListObjects(ctx, bucket, prefix, "", 1000, continuationToken)
		if err != nil {
			return nil, err
		}
		for _, obj := range result.Items {
			allObjects = append(allObjects, cosmoflare.ObjectInfo{
				Key:          obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
				ETag:         obj.ETag,
			})
		}
		if result.NextToken == "" {
			return allObjects, nil
		}
		continuationToken = result.NextToken
	}
}

func (b *r2StorageBackend) ListRemoteObjects(ctx context.Context, bucket, prefix string) ([]cosmoflare.ObjectInfo, error) {
	return listAllR2Objects(ctx, b.client, bucket, prefix)
}

func (b *r2StorageBackend) UploadFile(ctx context.Context, bucket, key, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", localPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", localPath, err)
	}

	_, err = b.client.Upload(ctx, bucket, key, f, info.Size())
	return err
}

func (b *r2StorageBackend) DownloadFile(ctx context.Context, bucket, key, localPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	result, err := b.client.Download(ctx, bucket, key)
	if err != nil {
		return err
	}
	defer result.Content.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", localPath, err)
	}
	defer f.Close()

	_, err = io.Copy(f, result.Content)
	return err
}

func (b *r2StorageBackend) DeleteRemoteObject(ctx context.Context, bucket, key string) error {
	return b.client.DeleteObject(ctx, bucket, key)
}
