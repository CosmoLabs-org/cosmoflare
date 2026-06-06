/*
Package batch provides worker implementation for batch operations

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package batch

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/operations"
)

// Worker represents a worker that processes operations
type Worker struct {
	ID       int
	queue    chan *Operation
	results  chan *Operation
	config   *BatchConfig
	stats    *WorkerStats
}

// WorkerStats contains statistics for a worker
type WorkerStats struct {
	Processed     int32         `json:"processed"`
	Completed     int32         `json:"completed"`
	Failed        int32         `json:"failed"`
	Skipped       int32         `json:"skipped"`
	Cancelled     int32         `json:"cancelled"`
	TotalDuration time.Duration `json:"total_duration"`
	AverageSpeed  float64       `json:"average_speed_mbps"`
	BytesProcessed int64        `json:"bytes_processed"`
	StartTime     time.Time     `json:"start_time"`
	LastActivity  time.Time     `json:"last_activity"`
}

// NewWorker creates a new worker
func NewWorker(id int, queue chan *Operation, results chan *Operation, config *BatchConfig) *Worker {
	return &Worker{
		ID:      id,
		queue:   queue,
		results: results,
		config:  config,
		stats: &WorkerStats{
			StartTime: time.Now(),
		},
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case op, ok := <-w.queue:
			if !ok {
				return
			}
			w.processOperation(ctx, op)
		case <-ctx.Done():
			return
		}
	}
}

// processOperation processes a single operation
func (w *Worker) processOperation(ctx context.Context, op *Operation) {
	startTime := time.Now()
	w.stats.LastActivity = startTime

	atomic.AddInt32(&w.stats.Processed, 1)

	// Set operation to running
	op.Status = StatusRunning
	op.StartTime = startTime

	// Process based on operation type
	var err error
	var processedBytes int64

	switch op.Type {
	case OperationTypeCopy:
		processedBytes, err = w.processCopy(ctx, op)
	case OperationTypeMove:
		processedBytes, err = w.processMove(ctx, op)
	case OperationTypeDelete:
		processedBytes, err = w.processDelete(ctx, op)
	default:
		err = fmt.Errorf("unsupported operation type: %s", op.Type)
	}

	// Update operation status
	duration := time.Since(startTime)
	op.EndTime = time.Now()
	op.Duration = duration

	if err != nil {
		op.Status = StatusFailed
		op.Error = err.Error()
		atomic.AddInt32(&w.stats.Failed, 1)
	} else {
		op.Status = StatusCompleted
		op.Progress = 100
		atomic.AddInt32(&w.stats.Completed, 1)
	}

	op.Size = processedBytes
	w.stats.BytesProcessed += processedBytes
	w.stats.TotalDuration += duration

	if w.stats.Processed > 0 {
		w.stats.AverageSpeed = float64(w.stats.BytesProcessed) / w.stats.TotalDuration.Seconds() / (1024 * 1024)
	}

	// Send result
	select {
	case w.results <- op:
	case <-ctx.Done():
		return
	}
}

// processCopy processes a copy operation
func (w *Worker) processCopy(ctx context.Context, op *Operation) (int64, error) {
	// Parse copy options
	copyOpts := w.parseCopyOptions(op.Options)

	// Create copy operation
	copyOp := operations.NewCopyOperation(op.Source, op.Destination, copyOpts)
	copyOp.Context = ctx

	// Execute copy with timeout
	if w.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.config.Timeout)
		defer cancel()
		copyOp.Context = ctx
		copyOp.CancelFunc = cancel
	}

	// Execute with retries
	var result *operations.CopyResult
	var err error

	for attempt := 0; attempt <= w.config.Retries; attempt++ {
		if attempt > 0 {
			if !w.config.Quiet {
				fmt.Printf("Worker %d: Retry %d/%d for %s\n", w.ID, attempt, w.config.Retries, op.Source)
			}

			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, err = copyOp.Execute()
		if err == nil {
			break
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
	}

	if err != nil {
		return 0, fmt.Errorf("copy operation failed: %w", err)
	}

	return result.Size, nil
}

// processMove processes a move operation
func (w *Worker) processMove(ctx context.Context, op *Operation) (int64, error) {
	// Move is copy + delete
	// First copy the file
	copiedBytes, err := w.processCopy(ctx, op)
	if err != nil {
		return 0, fmt.Errorf("move failed during copy: %w", err)
	}

	// Then delete the source file
	if !w.config.DryRun {
		if err := os.Remove(op.Source); err != nil {
			return copiedBytes, fmt.Errorf("move failed during delete: %w", err)
		}
	}

	return copiedBytes, nil
}

// processDelete processes a delete operation
func (w *Worker) processDelete(ctx context.Context, op *Operation) (int64, error) {
	// Get file size for stats
	var size int64
	if info, err := os.Stat(op.Source); err == nil {
		size = info.Size()
	}

	if w.config.DryRun {
		if !w.config.Quiet {
			fmt.Printf("Worker %d: DRY RUN - Would delete %s\n", w.ID, op.Source)
		}
		return size, nil
	}

	// Check if it's a directory
	info, err := os.Stat(op.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to stat %s: %w", op.Source, err)
	}

	if info.IsDir() {
		err = os.RemoveAll(op.Source)
	} else {
		err = os.Remove(op.Source)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to delete %s: %w", op.Source, err)
	}

	return size, nil
}

// parseCopyOptions parses copy options from operation options
func (w *Worker) parseCopyOptions(opts map[string]interface{}) *operations.CopyOptions {
	copyOpts := operations.DefaultCopyOptions()

	if opts == nil {
		return copyOpts
	}

	// Parse individual options
	if resume, ok := opts["resume"].(bool); ok {
		copyOpts.Resume = resume
	}

	if verify, ok := opts["verify"].(bool); ok {
		copyOpts.Verify = verify
	}

	if overwrite, ok := opts["overwrite"].(bool); ok {
		copyOpts.Overwrite = overwrite
	}

	if preserve, ok := opts["preserve"].(bool); ok {
		copyOpts.Preserve = preserve
	}

	if quiet, ok := opts["quiet"].(bool); ok {
		copyOpts.Quiet = quiet
	}

	if chunkSize, ok := opts["chunk_size"].(float64); ok {
		copyOpts.ChunkSize = int64(chunkSize)
	}

	if retries, ok := opts["retries"].(float64); ok {
		copyOpts.Retries = int(retries)
	}

	return copyOpts
}

// GetStats returns the worker's statistics
func (w *Worker) GetStats() *WorkerStats {
	return w.stats
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
		"file in use",
		"access denied",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
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