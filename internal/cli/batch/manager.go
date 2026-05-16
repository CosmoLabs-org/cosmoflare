/*
Package batch provides batch operation management for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/operations"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/progress"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/ux"
)

// OperationType defines different types of batch operations
type OperationType string

const (
	OperationTypeCopy    OperationType = "copy"
	OperationTypeMove    OperationType = "move"
	OperationTypeDelete  OperationType = "delete"
	OperationTypeUpload  OperationType = "upload"
	OperationTypeDownload OperationType = "download"
	OperationTypeSync    OperationType = "sync"
)

// OperationStatus defines the status of an operation
type OperationStatus string

const (
	StatusPending   OperationStatus = "pending"
	StatusRunning   OperationStatus = "running"
	StatusCompleted OperationStatus = "completed"
	StatusFailed    OperationStatus = "failed"
	StatusSkipped   OperationStatus = "skipped"
	StatusCancelled OperationStatus = "cancelled"
)

// Operation represents a single operation in a batch
type Operation struct {
	ID          string                 `json:"id"`
	Type        OperationType          `json:"type"`
	Status      OperationStatus        `json:"status"`
	Source      string                 `json:"source"`
	Destination string                 `json:"destination,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Progress    float64                `json:"progress"`
	Error       string                 `json:"error,omitempty"`
	StartTime   time.Time              `json:"start_time,omitempty"`
	EndTime     time.Time              `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Size        int64                  `json:"size,omitempty"`
	Speed       float64                `json:"speed_mbps,omitempty"`
}

// BatchConfig contains configuration for batch operations
type BatchConfig struct {
	Concurrency     int           `json:"concurrency"`
	ContinueOnError bool          `json:"continue_on_error"`
	Retries         int           `json:"retries"`
	Timeout         time.Duration `json:"timeout"`
	DryRun          bool          `json:"dry_run"`
	Quiet           bool          `json:"quiet"`
	Interactive     bool          `json:"interactive"`
}

// BatchStats contains statistics about batch operations
type BatchStats struct {
	Total          int32         `json:"total"`
	Pending        int32         `json:"pending"`
	Running        int32         `json:"running"`
	Completed      int32         `json:"completed"`
	Failed         int32         `json:"failed"`
	Skipped        int32         `json:"skipped"`
	Cancelled      int32         `json:"cancelled"`
	TotalSize      int64         `json:"total_size"`
	ProcessedSize  int64         `json:"processed_size"`
	TotalDuration  time.Duration `json:"total_duration"`
	AverageSpeed   float64       `json:"average_speed_mbps"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time,omitempty"`
	Progress       float64       `json:"progress"`
	ETA            time.Duration `json:"eta"`
}

// BatchManager manages and executes batch operations
type BatchManager struct {
	config      *BatchConfig
	operations  []*Operation
	stats       *BatchStats
	progress    *progress.MultiProgress
	context     context.Context
	cancel      context.CancelFunc
	workers     []*Worker
	queue       chan *Operation
	results     chan *Operation
	wg          sync.WaitGroup
	wgTracking  bool
	mu          sync.RWMutex
	onProgress  func(*BatchStats)
	onComplete  func(*BatchStats)
	onError     func(*Operation, error)
	onOperation func(*Operation)
}

// NewBatchManager creates a new batch manager
func NewBatchManager(config *BatchConfig) *BatchManager {
	if config == nil {
		config = DefaultBatchConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BatchManager{
		config:     config,
		operations: make([]*Operation, 0),
		stats: &BatchStats{
			StartTime: time.Now(),
		},
		progress: progress.NewMultiProgress(),
		context:  ctx,
		cancel:   cancel,
		queue:    make(chan *Operation, 100),
		results:  make(chan *Operation, 100),
	}
}

// DefaultBatchConfig returns default batch configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		Concurrency:     4,
		ContinueOnError: true,
		Retries:         3,
		Timeout:         30 * time.Minute,
		DryRun:          false,
		Quiet:           false,
		Interactive:     true,
	}
}

// AddOperation adds an operation to the batch
func (bm *BatchManager) AddOperation(op *Operation) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if op.ID == "" {
		op.ID = fmt.Sprintf("op_%d_%d", time.Now().Unix(), len(bm.operations))
	}

	op.Status = StatusPending
	bm.operations = append(bm.operations, op)
	atomic.AddInt32(&bm.stats.Total, 1)
	atomic.AddInt32(&bm.stats.Pending, 1)

	return nil
}

// AddCopyOperation adds a copy operation to the batch
func (bm *BatchManager) AddCopyOperation(src, dst string, opts *operations.CopyOptions) error {
	op := &Operation{
		ID:       fmt.Sprintf("copy_%d_%s", time.Now().Unix(), filepath.Base(src)),
		Type:     OperationTypeCopy,
		Status:   StatusPending,
		Source:   src,
		Destination: dst,
		Options: make(map[string]interface{}),
	}

	if opts != nil {
		op.Options["resume"] = opts.Resume
		op.Options["verify"] = opts.Verify
		op.Options["overwrite"] = opts.Overwrite
		op.Options["preserve"] = opts.Preserve
		op.Options["chunk_size"] = opts.ChunkSize
		op.Options["retries"] = opts.Retries
	}

	// Get file size for stats
	if info, err := os.Stat(src); err == nil {
		op.Size = info.Size()
		atomic.AddInt64(&bm.stats.TotalSize, op.Size)
	}

	return bm.AddOperation(op)
}

// LoadFromFile loads operations from a JSON file
func (bm *BatchManager) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read batch file: %w", err)
	}

	var batchSpec BatchSpec
	if err := json.Unmarshal(data, &batchSpec); err != nil {
		return fmt.Errorf("failed to parse batch file: %w", err)
	}

	for _, opSpec := range batchSpec.Operations {
		op := &Operation{
			ID:     opSpec.ID,
			Type:   OperationType(opSpec.Type),
			Status: StatusPending,
			Source: opSpec.Source,
		}

		if opSpec.Destination != "" {
			op.Destination = opSpec.Destination
		}

		if opSpec.Options != nil {
			op.Options = opSpec.Options
		}

		if err := bm.AddOperation(op); err != nil {
			return fmt.Errorf("failed to add operation: %w", err)
		}
	}

	return nil
}

// SaveToFile saves current operations to a JSON file
func (bm *BatchManager) SaveToFile(filename string) error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	batchSpec := BatchSpec{
		Config:      bm.config,
		Operations:  make([]BatchOperationSpec, len(bm.operations)),
	}

	for i, op := range bm.operations {
		batchSpec.Operations[i] = BatchOperationSpec{
			ID:          op.ID,
			Type:        string(op.Type),
			Source:      op.Source,
			Destination: op.Destination,
			Options:     op.Options,
		}
	}

	data, err := json.MarshalIndent(batchSpec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal batch spec: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write batch file: %w", err)
	}

	return nil
}

// Execute runs all operations in the batch
func (bm *BatchManager) Execute() (*BatchStats, error) {
	bm.mu.Lock()
	if len(bm.operations) == 0 {
		bm.mu.Unlock()
		return bm.stats, fmt.Errorf("no operations to execute")
	}
	bm.mu.Unlock()

	// Interactive confirmation
	if bm.config.Interactive && !bm.config.Quiet {
		operationNames := make([]string, len(bm.operations))
		for i, op := range bm.operations {
			operationNames[i] = fmt.Sprintf("%s %s", op.Type, op.Source)
		}

		confirm := ux.BatchConfirmation(operationNames, "batch")
		if !confirm.Answer {
			return bm.stats, fmt.Errorf("operation cancelled by user")
		}
	}

	// Create workers
	bm.workers = make([]*Worker, bm.config.Concurrency)
	for i := 0; i < bm.config.Concurrency; i++ {
		worker := NewWorker(i, bm.queue, bm.results, bm.config)
		bm.workers[i] = worker
	}

	// Set up WaitGroup for tracking operation completion
	bm.wg.Add(len(bm.operations))
	bm.wgTracking = true

	// Start workers
	for _, worker := range bm.workers {
		go worker.Start(bm.context)
	}

	// Start progress monitor
	go bm.monitorProgress()

	// Queue operations
	go bm.queueOperations()

	// Collect results
	go bm.collectResults()

	// Wait for completion
	bm.waitForCompletion()

	bm.stats.EndTime = time.Now()
	bm.stats.TotalDuration = bm.stats.EndTime.Sub(bm.stats.StartTime)

	if bm.stats.Completed > 0 {
		bm.stats.AverageSpeed = float64(bm.stats.ProcessedSize) / bm.stats.TotalDuration.Seconds() / (1024 * 1024)
	}

	return bm.stats, nil
}

// Cancel cancels all operations
func (bm *BatchManager) Cancel() {
	if bm.cancel != nil {
		bm.cancel()
	}
}

// GetStats returns current statistics
func (bm *BatchManager) GetStats() *BatchStats {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.updateProgress()

	stats := *bm.stats
	return &stats
}

// GetOperations returns all operations
func (bm *BatchManager) GetOperations() []*Operation {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	operations := make([]*Operation, len(bm.operations))
	copy(operations, bm.operations)
	return operations
}

// queueOperations queues operations for execution
func (bm *BatchManager) queueOperations() {
	bm.mu.RLock()
	operations := make([]*Operation, len(bm.operations))
	copy(operations, bm.operations)
	bm.mu.RUnlock()

	for _, op := range operations {
		op.Status = StatusRunning
		op.StartTime = time.Now()

		select {
		case bm.queue <- op:
			atomic.AddInt32(&bm.stats.Pending, -1)
			atomic.AddInt32(&bm.stats.Running, 1)
		case <-bm.context.Done():
			return
		}
	}
}

// collectResults collects operation results
func (bm *BatchManager) collectResults() {
	for {
		select {
		case op := <-bm.results:
			bm.processResult(op)
		case <-bm.context.Done():
			return
		}
	}
}

// processResult processes the result of an operation
func (bm *BatchManager) processResult(op *Operation) {
	atomic.AddInt32(&bm.stats.Running, -1)

	op.EndTime = time.Now()
	if !op.StartTime.IsZero() {
		op.Duration = op.EndTime.Sub(op.StartTime)
	}

	switch op.Status {
	case StatusCompleted:
		atomic.AddInt32(&bm.stats.Completed, 1)
		atomic.AddInt64(&bm.stats.ProcessedSize, op.Size)
	case StatusFailed:
		atomic.AddInt32(&bm.stats.Failed, 1)
		if bm.onError != nil {
			bm.onError(op, fmt.Errorf("%s", op.Error))
		}
	case StatusSkipped:
		atomic.AddInt32(&bm.stats.Skipped, 1)
	case StatusCancelled:
		atomic.AddInt32(&bm.stats.Cancelled, 1)
	}

	if bm.onOperation != nil {
		bm.onOperation(op)
	}

	if bm.wgTracking {
		bm.wg.Done()
	}
}

// monitorProgress monitors overall progress
func (bm *BatchManager) monitorProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.updateProgress()
			if bm.onProgress != nil {
				bm.onProgress(bm.stats)
			}
		case <-bm.context.Done():
			return
		}
	}
}

// updateProgress updates progress statistics
// Must be called with bm.mu held (any lock mode).
func (bm *BatchManager) updateProgress() {
	total := atomic.LoadInt32(&bm.stats.Total)
	completed := atomic.LoadInt32(&bm.stats.Completed)
	failed := atomic.LoadInt32(&bm.stats.Failed)
	skipped := atomic.LoadInt32(&bm.stats.Skipped)
	cancelled := atomic.LoadInt32(&bm.stats.Cancelled)

	if total > 0 {
		progress := float64(completed+failed+skipped+cancelled) / float64(total) * 100
		bm.stats.Progress = progress

		// Calculate ETA
		if progress > 0 {
			elapsed := time.Since(bm.stats.StartTime)
			estimatedTotal := time.Duration(float64(elapsed) * 100.0 / progress)
			bm.stats.ETA = estimatedTotal - elapsed
		}
	}
}

// waitForCompletion waits for all operations to complete via WaitGroup
func (bm *BatchManager) waitForCompletion() {
	done := make(chan struct{})
	go func() {
		bm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-bm.context.Done():
	}
}

// Callback setters
func (bm *BatchManager) OnProgress(fn func(*BatchStats))      { bm.onProgress = fn }
func (bm *BatchManager) OnComplete(fn func(*BatchStats))      { bm.onComplete = fn }
func (bm *BatchManager) OnError(fn func(*Operation, error))   { bm.onError = fn }
func (bm *BatchManager) OnOperation(fn func(*Operation))      { bm.onOperation = fn }

// BatchSpec represents a batch specification file
type BatchSpec struct {
	Config     *BatchConfig           `json:"config,omitempty"`
	Operations []BatchOperationSpec   `json:"operations"`
}

// BatchOperationSpec represents an operation in the specification
type BatchOperationSpec struct {
	ID          string                 `json:"id,omitempty"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Destination string                 `json:"destination,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}