/*
Package cmd provides migration and bulk operations for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"

	"github.com/CosmoLabs-org/cosmoflare/internal/migration"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migration and bulk operations",
	Long: `Migration and bulk operation tools for Cloudflare R2.

Commands:
  from-s3    Migrate from AWS S3 to R2
  sync       Sync buckets
  backup     Backup buckets to local storage
  restore    Restore from backup
  batch      Batch delete operations

Migration features:
- Resume interrupted transfers
- Parallel processing
- Checksum verification
- Progress tracking
- Bandwidth throttling

Example:
  cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket
  cosmoflare migrate backup my-bucket --to-local=/backup/
  cosmoflare migrate sync source-bucket dest-bucket`,
}

var (
	migrateSource       string
	migrateDest         string
	migrateFilter       string
	migrateConcurrency  int
	migrateDryRun       bool
	migrateResume       bool
	migrateVerify       bool
	migrateDeleteExtras bool
	migrateCompress     bool
	migrateIncremental  bool
	migrateManifest     string
)

// migrateFromS3Cmd represents the migrate from-s3 command
var migrateFromS3Cmd = &cobra.Command{
	Use:   "from-s3 [s3-bucket] to-r2 [r2-bucket]",
	Short: "Migrate from AWS S3 to R2",
	Long: `Migrate data from AWS S3 to Cloudflare R2.

Features:
- Parallel uploads for speed
- Checksum verification
- Resume capability
- Progress tracking
- Filter by prefix or pattern

Prerequisites:
- AWS credentials configured
- Target R2 bucket created
- Sufficient permissions

Examples:
  cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket
  cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --filter="images/*"
  cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --concurrency=20`,
	RunE: runMigrateFromS3,
}

// migrateSyncCmd represents the migrate sync command
var migrateSyncCmd = &cobra.Command{
	Use:   "sync [source-bucket] [dest-bucket]",
	Short: "Sync between buckets",
	Long: `Synchronize data between buckets.

Sync modes:
- One-way: Source → Destination
- Two-way: Bidirectional sync
- Mirror: Destination mirrors source

Options:
- --delete-extras: Delete files not in source
- --filter: Sync specific paths
- --verify: Verify checksums after sync
- --dry-run: Show what would be synced

Examples:
  cosmoflare migrate sync source-bucket dest-bucket
  cosmoflare migrate sync source-bucket dest-bucket --delete-extras
  cosmoflare migrate sync s3://my-bucket r2://my-bucket`,
	RunE: runMigrateSync,
}

// migrateBackupCmd represents the migrate backup command
var migrateBackupCmd = &cobra.Command{
	Use:   "backup [bucket-name]",
	Short: "Backup bucket to local storage",
	Long: `Create a complete backup of an R2 bucket to local storage.

Backup options:
- Include metadata and versions
- Compression for space savings
- Incremental backups
- Verification after backup

Examples:
  cosmoflare migrate backup my-bucket --to-local=/backup/
  cosmoflare migrate backup my-bucket --to-local=/backup/ --compress
  cosmoflare migrate backup my-bucket --to-local=/backup/ --include-versions`,
	RunE: runMigrateBackup,
}

// migrateRestoreCmd represents the migrate restore command
var migrateRestoreCmd = &cobra.Command{
	Use:   "restore [backup-file] to [bucket-name]",
	Short: "Restore from backup",
	Long: `Restore bucket data from a backup file.

Restore options:
- Verify data integrity
- Preserve timestamps
- Resume interrupted restores
- Selective path restore

Examples:
  cosmoflare migrate restore backup.tar.gz to my-bucket
  cosmoflare migrate restore backup.tar.gz to my-bucket --verify
  cosmoflare migrate restore backup.tar.gz to my-bucket --path="images/*"`,
	RunE: runMigrateRestore,
}

// migrateBatchCmd represents the migrate batch command
var migrateBatchCmd = &cobra.Command{
	Use:   "batch [bucket-name] [manifest-file]",
	Short: "Batch operations",
	Long: `Perform batch operations from a manifest file.

Manifest format (JSON):
{
  "operations": [
    {
      "action": "delete",
      "key": "old-file.txt"
    },
    {
      "action": "copy",
      "source": "old-path/file.txt",
      "dest": "new-path/file.txt"
    }
  ]
}

Examples:
  cosmoflare migrate batch my-bucket operations.json
  cosmoflare migrate batch my-bucket operations.json --continue`,
	RunE: runMigrateBatch,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Add subcommands
	migrateCmd.AddCommand(migrateFromS3Cmd)
	migrateCmd.AddCommand(migrateSyncCmd)
	migrateCmd.AddCommand(migrateBackupCmd)
	migrateCmd.AddCommand(migrateRestoreCmd)
	migrateCmd.AddCommand(migrateBatchCmd)

	// Flags for from-s3
	migrateFromS3Cmd.Flags().StringVar(&migrateFilter, "filter", "", "Filter by prefix or pattern")
	migrateFromS3Cmd.Flags().IntVar(&migrateConcurrency, "concurrency", 10, "Number of parallel uploads")
	migrateFromS3Cmd.Flags().BoolVar(&migrateResume, "resume", false, "Resume interrupted migration")
	migrateFromS3Cmd.Flags().BoolVar(&migrateVerify, "verify", true, "Verify checksums after upload")
	migrateFromS3Cmd.Flags().StringVar(&migrateManifest, "manifest", "", "Migration manifest file path")
	migrateFromS3Cmd.Flags().String("aws-region", "us-east-1", "AWS S3 region")
	migrateFromS3Cmd.Flags().String("aws-profile", "default", "AWS profile name")
	migrateFromS3Cmd.Flags().BoolVar(&migrateDeleteExtras, "delete-source", false, "Delete source objects after successful migration")
	migrateFromS3Cmd.Flags().BoolVar(&migrateCompress, "compress", false, "Compress data during transfer")

	// Flags for sync
	migrateSyncCmd.Flags().BoolVar(&migrateDeleteExtras, "delete-extras", false, "Delete files not in source")
	migrateSyncCmd.Flags().StringVar(&migrateFilter, "filter", "", "Sync specific paths")
	migrateSyncCmd.Flags().BoolVar(&migrateVerify, "verify", true, "Verify checksums after sync")
	migrateSyncCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be synced")

	// Flags for backup
	migrateBackupCmd.Flags().String("to-local", "", "Local backup directory (required)")
	migrateBackupCmd.Flags().BoolVar(&migrateCompress, "compress", false, "Compress backup files")
	migrateBackupCmd.Flags().Bool("include-versions", false, "Include object versions")
	migrateBackupCmd.Flags().BoolVar(&migrateIncremental, "incremental", false, "Incremental backup")

	// Flags for restore
	migrateRestoreCmd.Flags().String("path", "", "Restore specific path only")
	migrateRestoreCmd.Flags().BoolVar(&migrateVerify, "verify", true, "Verify data integrity")
	migrateRestoreCmd.Flags().BoolVar(&migrateResume, "resume", false, "Resume interrupted restore")

	// Flags for batch
	migrateBatchCmd.Flags().Bool("continue", false, "Continue on errors")
	migrateBatchCmd.Flags().IntVar(&migrateConcurrency, "concurrency", 5, "Number of parallel operations")
}

func runMigrateFromS3(cmd *cobra.Command, args []string) error {
	// Parse arguments (from-s3 s3-bucket to-r2 r2-bucket)
	var s3Bucket, r2Bucket string
	var err error

	s3Bucket, r2Bucket, err = parseMigrateFromS3Args(args)
	if err != nil {
		return err
	}

	filter, _ := cmd.Flags().GetString("filter")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	resume, _ := cmd.Flags().GetBool("resume")
	verify, _ := cmd.Flags().GetBool("verify")
	manifest, _ := cmd.Flags().GetString("manifest")
	awsRegion, _ := cmd.Flags().GetString("aws-region")
	awsProfile, _ := cmd.Flags().GetString("aws-profile")
	deleteSource, _ := cmd.Flags().GetBool("delete-source")
	compress, _ := cmd.Flags().GetBool("compress")

	printInfo("🔄 Migrating from S3 to R2")
	printInfo("Source: s3://%s", s3Bucket)
	printInfo("Destination: %s", r2Bucket)

	if filter != "" {
		printInfo("Filter: %s", filter)
	}
	printInfo("Concurrency: %d", concurrency)
	printInfo("AWS Region: %s", awsRegion)
	printInfo("AWS Profile: %s", awsProfile)

	if manifest != "" {
		printInfo("Manifest: %s", manifest)
	}
	if deleteSource {
		printWarning("Will delete source objects after successful migration")
	}

	// Create R2 client
	r2Client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create R2 client: %w", err)
	}

	// Create migration configuration
	migrationConfig := &migration.S3Migration{
		S3Bucket:     s3Bucket,
		R2Bucket:     r2Bucket,
		Filter:       filter,
		Concurrency:  concurrency,
		Resume:       resume,
		Verify:       verify,
		AWSRegion:    awsRegion,
		AWSProfile:   awsProfile,
		ManifestFile: manifest,
		DryRun:       DryRun,
		DeleteSource: deleteSource,
		Compression:  compress,
	}

	// Execute migration
	result, err := migrationConfig.Execute(r2Client)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print result summary
	printSuccess("🎉 Migration completed successfully!")
	printInfo("Processed %d objects (%s)", result.SuccessCount, migration.FormatBytes(result.TransferredSize))
	printInfo("Duration: %s", result.Duration.String())

	if result.ErrorCount > 0 {
		printWarning("⚠️ %d objects failed to migrate", result.ErrorCount)
		printInfo("Check manifest file for details: %s", result.ManifestPath)
	}

	return nil
}

func runMigrateSync(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("sync is not yet implemented — see ROAD-007")
}

func runMigrateBackup(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("backup is not yet implemented — see ROAD-007")
}

func runMigrateRestore(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("restore is not yet implemented — see ROAD-007")
}

func runMigrateBatch(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("batch operations are not yet implemented — see ROAD-007")
}

// Helper types and functions

func parseMigrateFromS3Args(args []string) (string, string, error) {
	if len(args) < 4 {
		return "", "", fmt.Errorf("usage: from-s3 [s3-bucket] to-r2 [r2-bucket]")
	}

	if args[1] != "to-r2" {
		return "", "", fmt.Errorf("expected 'to-r2' after S3 bucket name")
	}

	return args[0], args[3], nil
}

func parseRestoreArgs(args []string) (string, string, error) {
	if len(args) < 4 {
		return "", "", fmt.Errorf("usage: restore [backup-file] to [bucket-name]")
	}

	if args[1] != "to" {
		return "", "", fmt.Errorf("expected 'to' after backup file")
	}

	return args[0], args[3], nil
}

// Helper functions for migration
