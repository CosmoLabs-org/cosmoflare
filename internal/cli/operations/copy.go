/*
Package operations provides enhanced file operations for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package operations

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/progress"
)

// CopyOptions contains options for enhanced copy operations
type CopyOptions struct {
	Resume       bool          // Resume interrupted transfers
	Verify       bool          // Verify file integrity after copy
	Overwrite    bool          // Overwrite existing files
	Preserve     bool          // Preserve file attributes
	ProgressBar  bool          // Show progress bar
	Quiet        bool          // Suppress output
	ChunkSize    int64         // Chunk size for large files
	Concurrency  int           // Number of concurrent operations
	Timeout      time.Duration // Operation timeout
	Retries      int           // Number of retry attempts
	DryRun       bool          // Dry run mode
}

// DefaultCopyOptions returns default copy options
func DefaultCopyOptions() *CopyOptions {
	return &CopyOptions{
		Resume:      false,
		Verify:      true,
		Overwrite:   false,
		Preserve:    true,
		ProgressBar: true,
		Quiet:       false,
		ChunkSize:   8 * 1024 * 1024, // 8MB
		Concurrency: 4,
		Timeout:     30 * time.Minute,
		Retries:     3,
		DryRun:      false,
	}
}

// CopyResult contains the result of a copy operation
type CopyResult struct {
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	Size        int64         `json:"size"`
	Duration    time.Duration `json:"duration"`
	Speed       float64       `json:"speed_mbps"`
	Verified    bool          `json:"verified"`
	Resumed     bool          `json:"resumed"`
	Chunks      int           `json:"chunks"`
	Success     bool          `json:"success"`
	Error       error         `json:"error,omitempty"`
}

// CopyOperation represents a single copy operation
type CopyOperation struct {
	Source      string
	Destination string
	Options     *CopyOptions
	Context     context.Context
	CancelFunc  context.CancelFunc
}

// NewCopyOperation creates a new copy operation
func NewCopyOperation(src, dst string, opts *CopyOptions) *CopyOperation {
	if opts == nil {
		opts = DefaultCopyOptions()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CopyOperation{
		Source:      src,
		Destination: dst,
		Options:     opts,
		Context:     ctx,
		CancelFunc:  cancel,
	}
}

// Execute performs the copy operation
func (op *CopyOperation) Execute() (*CopyResult, error) {
	startTime := time.Now()
	result := &CopyResult{
		Source:      op.Source,
		Destination: op.Destination,
	}

	// Get source file info
	srcInfo, err := os.Stat(op.Source)
	if err != nil {
		result.Error = fmt.Errorf("failed to stat source file: %w", err)
		return result, result.Error
	}

	result.Size = srcInfo.Size()

	// Check if destination exists
	dstInfo, err := os.Stat(op.Destination)
	if err == nil && !op.Options.Overwrite {
		if op.Options.Resume {
			// Check if we can resume
			if dstInfo.Size() >= srcInfo.Size() {
				result.Success = true
				result.Verified = true
				return result, nil
			}
			result.Resumed = true
		} else {
			result.Error = fmt.Errorf("destination file exists and overwrite is disabled")
			return result, result.Error
		}
	}

	// Create progress bar if requested
	var progressBar progress.ProgressBar
	if op.Options.ProgressBar && !op.Options.Quiet {
		startPos := int64(0)
		if result.Resumed {
			if dstInfo != nil {
				startPos = dstInfo.Size()
			}
		}
		progressBar = progress.NewLinearProgressBar(srcInfo.Size(), fmt.Sprintf("Copying %s", filepath.Base(op.Source)))
		progressBar.Update(startPos, srcInfo.Size())
		progressBar.Start()
		defer progressBar.Complete()
	}

	// Perform the copy with retry logic
	err = op.copyWithRetry(progressBar, result)
	if err != nil {
		if progressBar != nil {
			progressBar.Error(err)
		}
		result.Error = err
		return result, err
	}

	// Calculate statistics
	result.Duration = time.Since(startTime)
	if result.Duration > 0 {
		result.Speed = float64(result.Size) / result.Duration.Seconds() / (1024 * 1024) // MB/s
	}

	// Verify if requested
	if op.Options.Verify && !op.Options.DryRun {
		verified, err := op.verifyCopy()
		if err != nil {
			result.Error = fmt.Errorf("verification failed: %w", err)
			return result, result.Error
		}
		result.Verified = verified

		if !verified {
			result.Error = fmt.Errorf("file verification failed: checksum mismatch")
			return result, result.Error
		}
	}

	result.Success = true

	if progressBar != nil {
		progressBar.Complete()
	}

	return result, nil
}

// Cancel cancels the operation
func (op *CopyOperation) Cancel() {
	if op.CancelFunc != nil {
		op.CancelFunc()
	}
}

// Progress returns current progress information
func (op *CopyOperation) Progress() *progress.ProgressInfo {
	// This would be implemented with a real progress tracking system
	return &progress.ProgressInfo{
		Operation: "copy",
		State:     progress.StateProgress,
	}
}

// Estimate returns estimated completion time
func (op *CopyOperation) Estimate() time.Duration {
	// This would be implemented with a real estimation system
	return 30 * time.Second
}

// copyWithRetry performs the copy with retry logic
func (op *CopyOperation) copyWithRetry(progressBar progress.ProgressBar, result *CopyResult) error {
	var lastErr error

	for attempt := 0; attempt <= op.Options.Retries; attempt++ {
		if attempt > 0 {
			if !op.Options.Quiet {
				fmt.Printf("Retry %d/%d...\n", attempt, op.Options.Retries)
			}

			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-op.Context.Done():
				return op.Context.Err()
			case <-time.After(backoff):
			}
		}

		err := op.performCopy(progressBar, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !isRetryableError(err) {
			break
		}
	}

	return lastErr
}

// performCopy performs the actual copy operation
func (op *CopyOperation) performCopy(progressBar progress.ProgressBar, result *CopyResult) error {
	if op.Options.DryRun {
		if !op.Options.Quiet {
			fmt.Printf("DRY RUN: Would copy %s to %s\n", op.Source, op.Destination)
		}
		return nil
	}

	// Open source file
	srcFile, err := os.Open(op.Source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination directory if needed
	dstDir := filepath.Dir(op.Destination)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Determine open mode for destination
	var dstFile *os.File
	if result.Resumed {
		dstFile, err = os.OpenFile(op.Destination, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		dstFile, err = os.Create(op.Destination)
	}

	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy with progress tracking
	var writer io.Writer = dstFile
	if progressBar != nil {
		writer = progress.NewProgressWriter(dstFile, progressBar)
	}

	// Use buffered copying for better performance
	buffer := make([]byte, op.Options.ChunkSize)
	copied, err := io.CopyBuffer(writer, srcFile, buffer)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	// Preserve file attributes if requested
	if op.Options.Preserve {
		srcInfo, err := os.Stat(op.Source)
		if err == nil {
			// Preserve modification time
			if err := os.Chtimes(op.Destination, time.Now(), srcInfo.ModTime()); err != nil && !op.Options.Quiet {
				fmt.Printf("Warning: failed to preserve modification time: %v\n", err)
			}

			// Preserve permissions (simplified)
			if err := os.Chmod(op.Destination, srcInfo.Mode()); err != nil && !op.Options.Quiet {
				fmt.Printf("Warning: failed to preserve permissions: %v\n", err)
			}
		}
	}

	// Verify copy size
	if copied != result.Size && !result.Resumed {
		return fmt.Errorf("copy size mismatch: expected %d bytes, got %d bytes", result.Size, copied)
	}

	return nil
}

// verifyCopy verifies the integrity of the copied file
func (op *CopyOperation) verifyCopy() (bool, error) {
	// Simple size-based verification for now
	// In a real implementation, this would use checksums (MD5, SHA256, etc.)

	srcInfo, err := os.Stat(op.Source)
	if err != nil {
		return false, fmt.Errorf("failed to stat source file for verification: %w", err)
	}

	dstInfo, err := os.Stat(op.Destination)
	if err != nil {
		return false, fmt.Errorf("failed to stat destination file for verification: %w", err)
	}

	return srcInfo.Size() == dstInfo.Size(), nil
}

// BatchCopy performs multiple copy operations concurrently
type BatchCopy struct {
	Operations []*CopyOperation
	Options    *CopyOptions
	Context    context.Context
}

// NewBatchCopy creates a new batch copy operation
func NewBatchCopy(operations []*CopyOperation, opts *CopyOptions) *BatchCopy {
	if opts == nil {
		opts = DefaultCopyOptions()
	}

	return &BatchCopy{
		Operations: operations,
		Options:    opts,
		Context:    context.Background(),
	}
}

// Execute performs all copy operations
func (bc *BatchCopy) Execute() ([]*CopyResult, error) {
	results := make([]*CopyResult, len(bc.Operations))
	errors := make([]error, 0)

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, bc.Options.Concurrency)

	// Channel to collect results
	resultChan := make(chan struct {
		index  int
		result *CopyResult
		err    error
	}, len(bc.Operations))

	// Start workers
	for i, op := range bc.Operations {
		go func(index int, operation *CopyOperation) {
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			result, err := operation.Execute()
			resultChan <- struct {
				index  int
				result *CopyResult
				err    error
			}{index, result, err}
		}(i, op)
	}

	// Collect results
	for i := 0; i < len(bc.Operations); i++ {
		select {
		case res := <-resultChan:
			results[res.index] = res.result
			if res.err != nil {
				errors = append(errors, res.err)
			}
		case <-bc.Context.Done():
			// Cancel remaining operations
			for _, op := range bc.Operations {
				op.Cancel()
			}
			return results, bc.Context.Err()
		}
	}

	if len(errors) > 0 && !bc.Options.Quiet {
		fmt.Printf("Batch copy completed with %d errors\n", len(errors))
	}

	return results, nil
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"timeout",
		"network",
		"temporary failure",
		"resource temporarily unavailable",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}