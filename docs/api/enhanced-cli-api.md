# Enhanced CLI API Documentation

**Complete API reference for the enhanced CLI components**

## 📋 Table of Contents

1. [Progress Monitoring API](#-progress-monitoring-api)
2. [Operations API](#-operations-api)
3. [Batch Manager API](#-batch-manager-api)
4. [User Experience API](#-user-experience-api)
5. [Error Handling API](#-error-handling-api)
6. [Integration Examples](#-integration-examples)

---

## 📊 Progress Monitoring API

### ProgressBar Interface

```go
type ProgressBar interface {
    Update(current, total int64)
    SetMessage(message string)
    SetSpeed(bytesPerSecond float64)
    Complete()
    Error(err error)
    Cancel()
    GetInfo() *ProgressInfo
    Start()
}
```

### ProgressInfo Structure

```go
type ProgressInfo struct {
    Operation     string        `json:"operation"`
    Current       int64         `json:"current"`
    Total         int64         `json:"total"`
    Percentage    float64       `json:"percentage"`
    Speed         float64       `json:"speed_mbps"`
    ETA           time.Duration `json:"eta"`
    StartTime     time.Time     `json:"start_time"`
    LastUpdate    time.Time     `json:"last_update"`
    State         ProgressState `json:"state"`
    Message       string        `json:"message"`
    Error         error         `json:"error,omitempty"`
}
```

### Creating Progress Bars

```go
// Linear progress bar
bar := progress.NewLinearProgressBar(totalSize, "Copying file")

// Circular progress bar
spinner := progress.NewCircularProgressBar("Processing...")

// Custom configuration
config := &progress.ProgressBarConfig{
    Type:        progress.ProgressTypeLinear,
    Template:    `{{counters . }} {{bar . }} {{percent . }} {{speed . }}`,
    ShowSpeed:   true,
    ShowETA:     true,
    ColorOutput: true,
    RefreshRate: 100 * time.Millisecond,
    Width:       80,
}

bar := progress.NewProgressBar(config)
```

### Progress Readers & Writers

```go
// Wrap an io.Reader with progress tracking
file, _ := os.Open("large-file.txt")
progressBar := progress.NewLinearProgressBar(fileSize, "Reading")
progressReader := progress.NewProgressReader(file, progressBar)

// Use like a regular reader
data, err := io.ReadAll(progressReader)

// Wrap an io.Writer with progress tracking
destFile, _ := os.Create("destination.txt")
progressWriter := progress.NewProgressWriter(destFile, progressBar)

// Use like a regular writer
_, err = progressWriter.Write(data)
```

### Multi-Progress Management

```go
// Create multi-progress manager
multiProgress := progress.NewMultiProgress()

// Add progress bars
multiProgress.Add("upload", uploadBar)
multiProgress.Add("download", downloadBar)
multiProgress.Add("processing", processingBar)

// Update individual progress bars
multiProgress.Update("upload", currentBytes, totalBytes)
multiProgress.Complete("upload")

// Get all progress information
allInfo := multiProgress.GetAllInfo()
```

---

## 🔄 Operations API

### CopyOptions Structure

```go
type CopyOptions struct {
    Resume       bool          // Resume interrupted transfers
    Verify       bool          // Verify file integrity after copy
    Overwrite    bool          // Overwrite existing files
    Preserve     bool          // Preserve file attributes
    ProgressBar  bool          // Show progress bar
    Quiet       bool          // Suppress output
    ChunkSize    int64         // Chunk size for large files
    Concurrency  int           // Number of concurrent operations
    Timeout      time.Duration // Operation timeout
    Retries      int           // Number of retry attempts
    DryRun       bool          // Dry run mode
}
```

### CopyOperation Structure

```go
type CopyOperation struct {
    Source      string
    Destination string
    Options     *CopyOptions
    Context     context.Context
    CancelFunc  context.CancelFunc
}
```

### Creating Copy Operations

```go
// Create with default options
copyOp := operations.NewCopyOperation("source.txt", "dest.txt", nil)

// Create with custom options
options := &operations.CopyOptions{
    Resume:      true,
    Verify:      true,
    ChunkSize:   16 * 1024 * 1024, // 16MB
    Concurrency: 8,
    ProgressBar: true,
}

copyOp := operations.NewCopyOperation("large-file.txt", "backup/large-file.txt", options)
```

### Executing Operations

```go
// Execute a copy operation
result, err := copyOp.Execute()
if err != nil {
    log.Printf("Copy failed: %v", err)
    return
}

fmt.Printf("Copy completed: %s in %v at %.2f MB/s\n",
    formatBytes(result.Size),
    result.Duration,
    result.Speed)
```

### CopyResult Structure

```go
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
```

### Batch Copy Operations

```go
// Create batch copy operations
operations := []*operations.CopyOperation{
    operations.NewCopyOperation("file1.txt", "backup/file1.txt", options),
    operations.NewCopyOperation("file2.txt", "backup/file2.txt", options),
    operations.NewCopyOperation("file3.txt", "backup/file3.txt", options),
}

// Execute batch
batchCopy := operations.NewBatchCopy(operations, options)
results, err := batchCopy.Execute()

// Process results
for i, result := range results {
    if result.Success {
        fmt.Printf("✅ %s copied successfully\n", result.Source)
    } else {
        fmt.Printf("❌ %s failed: %v\n", result.Source, result.Error)
    }
}
```

---

## 📦 Batch Manager API

### BatchConfig Structure

```go
type BatchConfig struct {
    Concurrency     int           `json:"concurrency"`
    ContinueOnError bool          `json:"continue_on_error"`
    Retries         int           `json:"retries"`
    Timeout         time.Duration `json:"timeout"`
    DryRun          bool          `json:"dry_run"`
    Quiet           bool          `json:"quiet"`
    Interactive     bool          `json:"interactive"`
}
```

### BatchManager Structure

```go
type BatchManager struct {
    // Internal fields
    config      *BatchConfig
    operations  []*Operation
    stats       *BatchStats
    progress    *progress.MultiProgress
    context     context.Context
    // ... other internal fields
}
```

### Creating Batch Manager

```go
// Create with default configuration
batchManager := batch.NewBatchManager(nil)

// Create with custom configuration
config := &batch.BatchConfig{
    Concurrency:     8,
    ContinueOnError: true,
    Retries:         3,
    Timeout:         30 * time.Minute,
    DryRun:          false,
    Interactive:     true,
}

batchManager := batch.NewBatchManager(config)
```

### Adding Operations

```go
// Add copy operation
err := batchManager.AddCopyOperation("source.txt", "dest.txt", copyOptions)

// Add custom operation
operation := &batch.Operation{
    ID:   "custom_op_1",
    Type: batch.OperationTypeCopy,
    Source: "source.txt",
    Destination: "dest.txt",
    Status: batch.StatusPending,
}

err := batchManager.AddOperation(operation)
```

### Setting Callbacks

```go
// Progress callback
batchManager.OnProgress(func(stats *batch.BatchStats) {
    fmt.Printf("Progress: %.1f%% | %d/%d operations\n",
        stats.Progress, stats.Completed, stats.Total)
})

// Completion callback
batchManager.OnComplete(func(stats *batch.BatchStats) {
    fmt.Printf("Batch completed: %d successful, %d failed\n",
        stats.Completed, stats.Failed)
})

// Error callback
batchManager.OnError(func(op *batch.Operation, err error) {
    fmt.Printf("Operation %s failed: %v\n", op.ID, err)
})

// Individual operation callback
batchManager.OnOperation(func(op *batch.Operation) {
    fmt.Printf("Operation %s completed with status: %s\n", op.ID, op.Status)
})
```

### Executing Batch Operations

```go
// Execute all operations
stats, err := batchManager.Execute()
if err != nil {
    log.Printf("Batch execution failed: %v", err)
}

// Get final statistics
fmt.Printf("Batch completed in %v\n", stats.TotalDuration)
fmt.Printf("Success rate: %.1f%%\n", float64(stats.Completed)/float64(stats.Total)*100)
```

### Batch Operations from Files

```go
// Load from JSON specification
err := batchManager.LoadFromFile("batch-operations.json")

// Save current operations to file
err := batchManager.SaveToFile("current-operations.json")
```

### BatchStats Structure

```go
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
```

---

## 👥 User Experience API

### Confirmation Types

```go
type ConfirmationType int

const (
    ConfirmationTypeYesNo ConfirmationType = iota
    ConfirmationTypeYesNoAll
    ConfirmationTypeYesNoCancel
    ConfirmationTypeContinue
    ConfirmationTypeRetry
    ConfirmationTypeOverwrite
)
```

### ConfirmationOptions Structure

```go
type ConfirmationOptions struct {
    Type        ConfirmationType
    Default     bool
    Timeout     int // Timeout in seconds, 0 for no timeout
    ShowDetails bool
    Message     string
    Details     string
    Warning     string
}
```

### Confirmation Functions

```go
// Simple yes/no confirmation
confirmed := ux.Confirm("Do you want to continue?")

// Confirmation with details
confirmed := ux.ConfirmWithDetails(
    "Delete this directory?",
    "This will remove all files in /tmp/data and cannot be undone"
)

// Destructive operation confirmation
confirmed := ux.ConfirmDestructive("delete", []string{
    "/tmp/important-file.txt",
    "/tmp/another-file.txt",
})

// File overwrite confirmation
confirmed := ux.ConfirmOverwrite("existing-file.txt", isNewer)

// Batch confirmation
result := ux.BatchConfirmation(files, "copy")
if result.Answer {
    // Proceed with batch operation
}
```

### ConfirmationResult Structure

```go
type ConfirmationResult struct {
    Answer      bool
    All         bool
    Cancelled   bool
    Timeout     bool
    Confidence  string // "high", "medium", "low"
    UserInput   string
}
```

### Interactive Confirmation

```go
// Create custom confirmation options
options := &ux.ConfirmationOptions{
    Type:        ux.ConfirmationTypeYesNoCancel,
    Default:     false,
    ShowDetails: true,
    Message:     "Delete old backup files?",
    Details:     "Found 15 old backup files consuming 2.3GB",
    Warning:     "This action cannot be undone!",
}

// Prompt for confirmation
result := ux.PromptConfirmation(options)

switch {
case result.Cancelled:
    fmt.Println("Operation cancelled by user")
case result.Answer:
    fmt.Println("User confirmed operation")
default:
    fmt.Println("User declined operation")
}
```

---

## 🛡️ Error Handling API

### Error Types

```go
type ErrorType int

const (
    ErrorTypeNetwork ErrorType = iota
    ErrorTypePermission
    ErrorTypeStorage
    ErrorTypeAuthentication
    ErrorTypeQuota
    ErrorTypeValidation
    ErrorTypeTimeout
    ErrorTypeUserCancel
    ErrorTypeFileSystem
    ErrorTypeUnknown
)
```

### Error Classification

```go
// Classify an error
err := os.Open("nonexistent.txt")
errorInfo := ux.ClassifyError(err)

fmt.Printf("Error Type: %s\n", ux.ErrorTypeToString(errorInfo.Type))
fmt.Printf("Retryable: %t\n", errorInfo.Retryable)
fmt.Printf("Suggestion: %s\n", errorInfo.Suggestion)
```

### ErrorInfo Structure

```go
type ErrorInfo struct {
    Type         ErrorType
    Message      string
    Original     error
    Retryable    bool
    Suggestion   string
    ErrorCode    string
    Context      map[string]interface{}
    Attempts     int
    LastAttempt  time.Time
    NextAttempt  time.Time
}
```

### Retry Strategies

```go
// Default retry strategy
strategy := ux.DefaultRetryStrategy()

// Network-optimized retry strategy
strategy := ux.NetworkRetryStrategy()

// File system retry strategy
strategy := ux.FileSystemRetryStrategy()

// Custom retry strategy
strategy := &ux.RetryStrategy{
    MaxRetries:    5,
    BaseDelay:     2 * time.Second,
    MaxDelay:      60 * time.Second,
    BackoffFactor: 2.0,
    Jitter:        true,
    Strategy:      "exponential",
}
```

### Retry Functions

```go
// Retry with custom strategy
result := ux.RetryWithStrategy(strategy, func() error {
    return performOperation()
})

if result.Success {
    fmt.Printf("Operation succeeded after %d attempts\n", result.Attempts)
} else {
    fmt.Printf("Operation failed after %d attempts: %v\n", result.Attempts, result.LastError)
}

// Simple retry with backoff
err := ux.RetryWithBackoff(3, func() error {
    return performNetworkOperation()
})
```

### Interactive Retry

```go
// Interactive retry with user confirmation
err := ux.InteractiveRetry(func() error {
    return performUnstableOperation()
}, "Retry the failed operation?")

// Smart retry with context-aware strategies
smartRetry := ux.NewSmartRetryContext()

// Set strategy for specific error type
smartRetry.SetStrategy(ux.ErrorTypeNetwork, ux.NetworkRetryStrategy())

// Execute with smart retry logic
err = smartRetry.Execute(func() error {
    return performComplexOperation()
})
```

### RetryResult Structure

```go
type RetryResult struct {
    Success       bool
    Attempts      int
    TotalTime     time.Duration
    StartTime     time.Time
    LastError     error
    ErrorInfo     *ErrorInfo
    Retryed       bool
    BackoffDelays []time.Duration
}
```

---

## 🔧 Integration Examples

### Complete Enhanced Copy Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/operations"
    "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/progress"
    "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/ux"
)

func main() {
    // Create enhanced copy options
    options := &operations.CopyOptions{
        Resume:      true,
        Verify:      true,
        Overwrite:   false,
        Preserve:    true,
        ProgressBar: true,
        ChunkSize:   8 * 1024 * 1024, // 8MB
        Concurrency: 4,
        Timeout:     30 * time.Minute,
        Retries:     3,
    }

    // Create copy operation
    copyOp := operations.NewCopyOperation("source.txt", "destination.txt", options)

    // Create progress bar
    progressBar := progress.NewLinearProgressBar(0, "Copying file")
    progressBar.Start()

    // Execute with retry logic
    strategy := ux.NetworkRetryStrategy()
    result := ux.RetryWithStrategy(strategy, func() error {
        _, err := copyOp.Execute()
        return err
    })

    if result.Success {
        progressBar.Complete()
        fmt.Printf("✅ Copy completed successfully!\n")
        fmt.Printf("   Attempts: %d\n", result.Attempts)
        fmt.Printf("   Duration: %v\n", result.TotalTime)
    } else {
        progressBar.Error(result.LastError)
        fmt.Printf("❌ Copy failed: %v\n", result.LastError)
        ux.PrintRetryProgress(result)
    }
}
```

### Batch Processing Example

```go
func processBatchOfFiles() {
    // Create batch configuration
    config := &batch.BatchConfig{
        Concurrency:     8,
        ContinueOnError: true,
        Retries:         3,
        Timeout:         30 * time.Minute,
        Interactive:     true,
    }

    // Create batch manager
    batchManager := batch.NewBatchManager(config)

    // Set up progress callback
    batchManager.OnProgress(func(stats *batch.BatchStats) {
        fmt.Printf("\r⚡ Progress: %.1f%% | %d/%d | %.1f MB/s | ETA: %v",
            stats.Progress,
            stats.Completed,
            stats.Total,
            stats.AverageSpeed,
            progress.FormatDuration(stats.ETA),
        )
    })

    // Add operations
    files := []string{"file1.txt", "file2.txt", "file3.txt"}
    for _, file := range files {
        err := batchManager.AddCopyOperation(file, "backup/"+file, nil)
        if err != nil {
            log.Printf("Failed to add operation for %s: %v", file, err)
        }
    }

    // Execute batch
    fmt.Println("📦 Processing batch operations...")
    stats, err := batchManager.Execute()
    if err != nil {
        log.Printf("Batch execution failed: %v", err)
    }

    // Print results
    fmt.Printf("\n📊 Batch Results:\n")
    fmt.Printf("   Total: %d\n", stats.Total)
    fmt.Printf("   Completed: %d\n", stats.Completed)
    fmt.Printf("   Failed: %d\n", stats.Failed)
    fmt.Printf("   Duration: %v\n", stats.TotalDuration)
    fmt.Printf("   Avg Speed: %.1f MB/s\n", stats.AverageSpeed)
}
```

### Progress Monitoring Example

```go
func monitorProgress() {
    // Create multi-progress manager
    multiProgress := progress.NewMultiProgress()

    // Create progress bars for different operations
    uploadBar := progress.NewLinearProgressBar(0, "Uploading files")
    downloadBar := progress.NewLinearProgressBar(0, "Downloading files")
    processBar := progress.NewCircularProgressBar("Processing data")

    // Add to multi-progress manager
    multiProgress.Add("upload", uploadBar)
    multiProgress.Add("download", downloadBar)
    multiProgress.Add("process", processBar)

    // Start all progress bars
    uploadBar.Start()
    downloadBar.Start()

    // Simulate progress updates
    go func() {
        for i := 0; i <= 100; i += 10 {
            multiProgress.Update("upload", int64(i), 100)
            time.Sleep(100 * time.Millisecond)
        }
        multiProgress.Complete("upload")
    }()

    go func() {
        for i := 0; i <= 100; i += 5 {
            multiProgress.Update("download", int64(i), 100)
            time.Sleep(50 * time.Millisecond)
        }
        multiProgress.Complete("download")
    }()

    // Wait for completion
    time.Sleep(2 * time.Second)

    // Get final progress information
    allInfo := multiProgress.GetAllInfo()
    for id, info := range allInfo {
        fmt.Printf("%s: %s\n", id, progress.FormatDuration(info.ETA))
    }
}
```

### Error Handling Example

```go
func handleErrors() {
    // Operation that might fail
    operation := func() error {
        // Simulate a network error
        return fmt.Errorf("connection timeout")
    }

    // Classify the error
    err := operation()
    errorInfo := ux.ClassifyError(err)

    fmt.Printf("Error Type: %s\n", ux.ErrorTypeToString(errorInfo.Type))
    fmt.Printf("Retryable: %t\n", errorInfo.Retryable)
    fmt.Printf("Suggestion: %s\n", errorInfo.Suggestion)

    // Retry with smart strategy
    if errorInfo.Retryable {
        strategy := ux.NetworkRetryStrategy()
        result := ux.RetryWithStrategy(strategy, operation)

        if result.Success {
            fmt.Printf("✅ Operation succeeded after %d attempts\n", result.Attempts)
        } else {
            fmt.Printf("❌ Operation failed after %d attempts\n", result.Attempts)
            fmt.Printf("Last error: %v\n", result.LastError)
        }
    }
}
```

---

**Version**: 1.0.0 | **Last Updated**: 2025-01-24 | **License**: MIT

*API documentation for enhanced CLI features built with ❤️ by CosmoLabs*