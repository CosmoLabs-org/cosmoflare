/*
Package cmd provides enhanced copy command for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/batch"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/operations"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/progress"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/ux"
	visual "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/visual"
)

// Enhanced copy command with progress monitoring and batch support
var copyCmd = &cobra.Command{
	Use:   "copy [source] [destination]",
	Short: "Copy files with enhanced progress monitoring",
	Long: `Copy files and directories with real-time progress monitoring,
resume capability, integrity verification, and batch processing support.

Features:
  • Real-time progress bars with speed and ETA
  • Resume interrupted transfers
  • File integrity verification
  • Batch operation support
  • Smart retry logic with exponential backoff
  • Concurrent processing for multiple files
  • Interactive confirmations for destructive operations

Progress Monitoring:
  • Real-time progress bars with customizable refresh rates
  • Transfer speed monitoring in MB/s
  • ETA calculation based on current speed
  • Multiple progress display formats (linear, circular, percentage)

Batch Operations:
  • Process multiple files/directories concurrently
  • Queue management with configurable concurrency
  • Detailed statistics and error reporting
  • Operation cancellation support
  • Progress monitoring for entire batch

Examples:
  # Single file copy with progress
  cosmoflare copy source.txt dest.txt

  # Copy directory recursively
  cosmoflare copy --recursive source/ dest/

  # Copy with custom options
  cosmoflare copy --progress --verify --resume source.txt dest.txt

  # Batch copy from file list
  cosmoflare copy --batch files.txt destination/

  # High-performance copy with optimal settings
  cosmoflare copy --parallel 8 --chunk-size 16MB source/ dest/

  # Dry run to see what would be copied
  cosmoflare copy --dry-run --verbose source/ dest/`,

	RunE: runCopy,
}

var (
	copyRecursive    bool
	copyResume       bool
	copyVerify       bool
	copyOverwrite    bool
	copyPreserve     bool
	copyProgress     bool
	copyQuiet        bool
	copyBatch        bool
	copyParallel     int
	copyChunkSize    string
	copyRetries      int
	copyTimeout      string
	copyInteractive  bool
	copyDryRun       bool
	copyNoClobber    bool
	copyStats        bool
	copyFormat       string
)

func init() {
	rootCmd.AddCommand(copyCmd)

	// Basic options
	copyCmd.Flags().BoolVarP(&copyRecursive, "recursive", "r", false, "Copy directories recursively")
	copyCmd.Flags().BoolVar(&copyResume, "resume", false, "Resume interrupted transfers")
	copyCmd.Flags().BoolVarP(&copyVerify, "verify", "V", true, "Verify file integrity after copy")
	copyCmd.Flags().BoolVarP(&copyOverwrite, "overwrite", "o", false, "Overwrite existing files")
	copyCmd.Flags().BoolVarP(&copyPreserve, "preserve", "p", true, "Preserve file attributes")

	// Progress options
	copyCmd.Flags().BoolVarP(&copyProgress, "progress", "P", true, "Show progress bars")
	copyCmd.Flags().BoolVarP(&copyQuiet, "quiet", "q", false, "Suppress output except errors")
	copyCmd.Flags().BoolVarP(&copyStats, "stats", "s", false, "Show detailed statistics")

	// Performance options
	copyCmd.Flags().IntVarP(&copyParallel, "parallel", "j", 4, "Number of parallel operations")
	copyCmd.Flags().StringVar(&copyChunkSize, "chunk-size", "8MB", "Chunk size for large files (e.g., 8MB, 1GB)")

	// Batch options
	copyCmd.Flags().BoolVarP(&copyBatch, "batch", "b", false, "Batch operation mode")
	copyCmd.Flags().StringVar(&copyFormat, "format", "table", "Output format (table, json, csv)")

	// Advanced options
	copyCmd.Flags().BoolVar(&copyInteractive, "interactive", true, "Interactive confirmations")
	copyCmd.Flags().BoolVar(&copyDryRun, "dry-run", false, "Show what would be copied without executing")
	copyCmd.Flags().BoolVar(&copyNoClobber, "no-clobber", false, "Do not overwrite existing files")
	copyCmd.Flags().IntVar(&copyRetries, "retries", 3, "Number of retry attempts")
	copyCmd.Flags().StringVar(&copyTimeout, "timeout", "30m", "Operation timeout (e.g., 30s, 5m, 1h)")
}

func runCopy(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if len(args) < 2 {
		return fmt.Errorf("source and destination are required")
	}

	source := args[0]
	destination := args[1]

	// Parse timeout
	timeout, err := time.ParseDuration(copyTimeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	// Parse chunk size
	chunkSize, err := parseChunkSize(copyChunkSize)
	if err != nil {
		return fmt.Errorf("invalid chunk size: %w", err)
	}

	// Create copy options
	copyOpts := &operations.CopyOptions{
		Resume:      copyResume,
		Verify:      copyVerify,
		Overwrite:   copyOverwrite || copyDryRun,
		Preserve:    copyPreserve,
		ProgressBar: copyProgress && !copyQuiet,
		Quiet:       copyQuiet,
		ChunkSize:   chunkSize,
		Concurrency: copyParallel,
		Timeout:     timeout,
		Retries:     copyRetries,
		DryRun:      copyDryRun,
	}

	// Check if source is a directory
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to access source: %w", err)
	}

	if sourceInfo.IsDir() {
		return copyDirectory(source, destination, copyOpts)
	}

	// Handle batch mode
	if copyBatch {
		return handleBatchCopy(source, destination, copyOpts)
	}

	// Single file copy
	return copyFile(source, destination, copyOpts)
}

// copyFile copies a single file with enhanced features
func copyFile(source, destination string, opts *operations.CopyOptions) error {
	// Get source file info
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Check destination exists
	destinationInfo, err := os.Stat(destination)
	if err == nil && !opts.Overwrite && !copyNoClobber {
		if !opts.Quiet {
			fmt.Printf("Destination file exists: %s\n", destination)
		}

		if copyInteractive {
			confirm := ux.ConfirmOverwrite(destination, sourceInfo.ModTime().After(destinationInfo.ModTime()))
			if !confirm {
				fmt.Println("Copy cancelled by user")
				return nil
			}
		}
	}

	// Show copy info
	if !opts.Quiet {
		fmt.Printf("📁 Copying: %s -> %s\n", source, destination)
		fmt.Printf("📊 Size: %s\n", utils.FormatBytes(sourceInfo.Size()))
		if opts.Resume {
			fmt.Printf("🔄 Resume mode enabled\n")
		}
		if opts.Verify {
			fmt.Printf("✅ Verification enabled\n")
		}
	}

	// Create and execute copy operation
	copyOp := operations.NewCopyOperation(source, destination, opts)

	// Execute copy
	result, err := copyOp.Execute()
	if err != nil {
		return fmt.Errorf("copy operation failed: %w", err)
	}

	// Show results with enhanced visual display
	if copyStats && !opts.Quiet {
		printEnhancedCopyResult(result)
	} else if !opts.Quiet {
		visual.ShowSuccess(fmt.Sprintf("Successfully copied %s", filepath.Base(source)))
	}

	return nil
}

// copyDirectory copies a directory recursively
func copyDirectory(source, destination string, opts *operations.CopyOptions) error {
	if !copyRecursive {
		return fmt.Errorf("source is a directory, use --recursive to copy directories")
	}

	// Create batch manager for directory copy
	batchConfig := &batch.BatchConfig{
		Concurrency:     opts.Concurrency,
		ContinueOnError: true,
		Retries:         opts.Retries,
		Timeout:         opts.Timeout,
		DryRun:          opts.DryRun,
		Quiet:           opts.Quiet,
		Interactive:     copyInteractive,
	}

	batchManager := batch.NewBatchManager(batchConfig)

	// Walk directory and add copy operations
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// Calculate destination path
		destPath := filepath.Join(destination, relPath)

		// Add copy operation to batch
		return batchManager.AddCopyOperation(path, destPath, opts)
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Execute batch copy
	if !opts.Quiet {
		fmt.Printf("📁 Copying directory: %s -> %s\n", source, destination)
	}

	// Set up progress callback
	batchManager.OnProgress(func(stats *batch.BatchStats) {
		if copyProgress && !opts.Quiet {
			printBatchProgress(stats)
		}
	})

	// Execute batch
	stats, err := batchManager.Execute()
	if err != nil {
		return fmt.Errorf("batch copy failed: %w", err)
	}

	// Show final results
	if copyStats && !opts.Quiet {
		printBatchResults(stats)
	} else if !opts.Quiet {
		fmt.Printf("✅ Successfully copied %d files\n", stats.Completed)
	}

	return nil
}

// handleBatchCopy handles batch copy from file list
func handleBatchCopy(source, destination string, opts *operations.CopyOptions) error {
	// source should be a file containing list of files to copy
	if !strings.HasSuffix(source, ".txt") && !strings.HasSuffix(source, ".json") {
		return fmt.Errorf("batch mode requires a .txt or .json file with file list")
	}

	// Create batch manager
	batchConfig := &batch.BatchConfig{
		Concurrency:     opts.Concurrency,
		ContinueOnError: true,
		Retries:         opts.Retries,
		Timeout:         opts.Timeout,
		DryRun:          opts.DryRun,
		Quiet:           opts.Quiet,
		Interactive:     copyInteractive,
	}

	batchManager := batch.NewBatchManager(batchConfig)

	// Load operations from file
	if strings.HasSuffix(source, ".json") {
		err := batchManager.LoadFromFile(source)
		if err != nil {
			return fmt.Errorf("failed to load batch file: %w", err)
		}
	} else {
		// Parse text file with list of files
		err := parseTextBatchFile(source, destination, batchManager, opts)
		if err != nil {
			return fmt.Errorf("failed to parse batch file: %w", err)
		}
	}

	// Set up progress callback
	batchManager.OnProgress(func(stats *batch.BatchStats) {
		if copyProgress && !opts.Quiet {
			printBatchProgress(stats)
		}
	})

	// Execute batch
	if !opts.Quiet {
		fmt.Printf("📦 Processing batch copy operations...\n")
	}

	stats, err := batchManager.Execute()
	if err != nil {
		return fmt.Errorf("batch copy failed: %w", err)
	}

	// Show final results
	if copyStats && !opts.Quiet {
		printBatchResults(stats)
	} else if !opts.Quiet {
		fmt.Printf("✅ Batch copy completed: %d successful, %d failed\n", stats.Completed, stats.Failed)
	}

	return nil
}

// parseTextBatchFile parses a text file containing list of files to copy
func parseTextBatchFile(filename, destination string, batchManager *batch.BatchManager, opts *operations.CopyOptions) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read batch file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple format: source_file
		// Or: source_file -> destination_file
		parts := strings.Split(line, "->")
		source := strings.TrimSpace(parts[0])
		dest := destination

		if len(parts) == 2 {
			dest = strings.TrimSpace(parts[1])
		}

		// Add to batch
		err := batchManager.AddCopyOperation(source, dest, opts)
		if err != nil {
			return fmt.Errorf("failed to add copy operation: %w", err)
		}
	}

	return nil
}

// Helper functions

func parseChunkSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(sizeStr)
	if strings.HasSuffix(sizeStr, "GB") {
		size := parseFloat64(strings.TrimSuffix(sizeStr, "GB"))
		return int64(size * 1024 * 1024 * 1024), nil
	} else if strings.HasSuffix(sizeStr, "MB") {
		size := parseFloat64(strings.TrimSuffix(sizeStr, "MB"))
		return int64(size * 1024 * 1024), nil
	} else if strings.HasSuffix(sizeStr, "KB") {
		size := parseFloat64(strings.TrimSuffix(sizeStr, "KB"))
		return int64(size * 1024), nil
	} else {
		size := parseFloat64(sizeStr)
		return int64(size), nil
	}
}

func parseFloat64(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func printEnhancedCopyResult(result *operations.CopyResult) {
	// Create animated success display
	visual.ShowSuccess("Copy Operation Completed!")

	// Create professional result display
	details := map[string]interface{}{
		"Source":      result.Source,
		"Destination": result.Destination,
		"Size":        utils.FormatBytes(result.Size),
		"Duration":    result.Duration.String(),
		"Speed":       fmt.Sprintf("%.2f MB/s", result.Speed),
		"Verified":    fmt.Sprintf("%t", result.Verified),
	}

	if result.Resumed {
		details["Resumed"] = "Yes"
	}

	visual.ShowResult("📊 Copy Operation Details", details)

	// Show animated progress summary
	fmt.Println()
	visual.ShowProgress(result.Size, result.Size, "Transfer Summary")
}

func printBatchProgress(stats *batch.BatchStats) {
	// Clear line and show progress
	fmt.Printf("\r⚡ Progress: %.1f%% | %d/%d | %.1f MB/s | ETA: %v",
		stats.Progress,
		stats.Completed,
		stats.Total,
		stats.AverageSpeed,
		progress.FormatDuration(stats.ETA))
}

func printBatchResults(stats *batch.BatchStats) {
	fmt.Printf("\n📊 Batch Copy Results:\n")
	fmt.Printf("  Total:        %d\n", stats.Total)
	fmt.Printf("  Completed:    %d\n", stats.Completed)
	fmt.Printf("  Failed:       %d\n", stats.Failed)
	fmt.Printf("  Skipped:      %d\n", stats.Skipped)
	fmt.Printf("  Duration:     %v\n", stats.TotalDuration)
	fmt.Printf("  Total Size:   %s\n", utils.FormatBytes(stats.TotalSize))
	fmt.Printf("  Avg Speed:    %.2f MB/s\n", stats.AverageSpeed)

	if JSONOutput {
		printJSON(stats)
	}
}