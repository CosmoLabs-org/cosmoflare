---
created: ""
goals_completed: 0
goals_total: 48
origin: migrated by ccs prompts migrate
priority: medium
status: PENDING
title: 'Session 008 Prompt: Enhanced CLI Features - Real File Operations'
---

# Session 008 Prompt: Enhanced CLI Features - Real File Operations

**Objective**: Implement advanced CLI features for R2Go2 with professional progress bars, batch operations, and user-friendly error handling to complement the professional TUI interface.

## 🎯 **Session Goals**

### **Primary Objective**
Build production-ready CLI commands that go beyond basic file operations, featuring:
- Real-time progress bars for long-running operations
- Batch operation support with cancellation capability
- Professional error handling and recovery
- Transfer speed monitoring and statistics
- Queue management for concurrent operations

### **Current CLI Limitations**
- Basic file operations without progress feedback
- No batch processing capabilities
- Limited error information for troubleshooting
- Single operation processing only
- No transfer monitoring or statistics

## 🚀 **Enhanced CLI Feature Roadmap**

### **File Operations Enhancement**
```bash
# Current State:
r2go2 copy source.txt dest.txt
# → No feedback until completion

# Target State:
r2go2 copy --progress source.txt dest.txt
# 📁 Copying file...
# ████████████████████████████████████ 75%
# Size: 1.2GB | Speed: 45MB/s | ETA: 15s
```

### **Batch Operations**
```bash
# New Features:
r2go2 copy --batch files.txt destination/
r2go2 sync --progress --concurrent local/ r2:bucket/
r2go2 upload --queue files/ --parallel 4
r2go2 download --progress --resume large-files/
```

### **Advanced Monitoring**
```bash
# Real-time Statistics:
r2go2 sync --monitor
┌─ Operations ── Files ── Speed ── Progress ── ETA ──────────────┐
│ Uploading     │ 12    │ 45MB │ 67%     │  30s │
│ Downloading  │ 8     │ 22MB │ 34%     │  2m  │
│ Syncing      │ 5     │ 0B/s  │ 100%    │ Done │
└─────────────┴──────────────────────────────────────┘
```

## 📋 **Implementation Strategy**

### **Phase 1: Core File Operations Enhancement**
Enhance basic copy/move operations with progress indicators:

```go
// Enhanced file copy with progress
type CopyOperation struct {
    Source      string
    Destination string
    Size        int64
    Progress    float64
    Speed       float64
    Start       time.Time
    Canceled    bool
    Error       error
}
```

### **Phase 2: Progress Bar System**
Create reusable progress visualization components:

```go
// Progress bar types
type ProgressBar interface {
    Update(current, total int64)
    SetMessage(message string)
    Complete()
    Error(err error)
}

type ProgressBarType int
const (
    ProgressBarTypeLinear ProgressBarType = iota
    ProgressBarTypeCircular
    ProgressBarTypePercentage
)
```

### **Phase 3: Batch Operation Manager**
Implement queue-based batch processing:

```go
// Batch operation manager
type BatchManager struct {
    Queue        []Operation
    Concurrency   int
    Active       map[string]*Operation
    Completed    int
    Failed       int
    Cancelled    int
    Stats        *BatchStats
}
```

### **Phase 4: Advanced CLI Commands**
Create new CLI commands with enhanced features:

```bash
# Enhanced Commands to Implement:
r2go2 copy --progress --resume source.txt dest.txt
r2go2 move --overwrite --confirm old/ new/
r2go2 sync --dry-run --parallel local/ remote/
r2go2 upload --metadata --tags "prod,v1.0" files/
r2go2 download --check-integrity --verify file.txt
r2go2 batch --config batch.json operations.json
```

## 🔧 **Technical Implementation**

### **File Operation Enhancement**

#### **Enhanced Copy Operation**
```go
type CopyOptions struct {
    Resume      bool
    Verify      bool
    Overwrite   bool
    Preserve    bool
    ProgressBar bool
    Quiet       bool
    ChunkSize    int64
    Concurrency  int
}

func CopyFile(src, dst string, opts CopyOptions) error {
    // File size detection
    fileSize, err := os.Stat(src)
    // Progress tracking setup
    // Enhanced copy logic with resume capability
}
```

#### **Progress Monitoring**
```go
type ProgressMonitor struct {
    StartTime   time.Time
    LastUpdate  time.Time
    TotalBytes  int64
    CurrentBytes int64
    Speed       float64
    Bar         ProgressBar
    UpdateChan  chan ProgressUpdate
}
```

### **Batch Processing Framework**

#### **Operation Queue System**
```go
type Operation interface {
    Execute() error
    Cancel() error
    Progress() ProgressInfo
    Estimate() time.Duration
}

type QueueManager struct {
    Operations   []Operation
    Workers      []*Worker
    Queue        chan Operation
    Results      chan Result
    Stats        *QueueStats
}
```

#### **Concurrent Processing**
```go
func (qm *QueueManager) ProcessConcurrent(ops []Operation) {
    // Distribute operations among workers
    // Monitor progress and statistics
    // Handle errors and retries
    // Support cancellation
}
```

### **User Experience Enhancements**

#### **Interactive Confirmations**
```bash
# Confirmation Dialogs:
r2go2 rm --recursive directory/
? This will delete directory/ and all its contents. Continue? [y/N]
```

#### **Error Recovery**
```bash
# Smart Error Handling:
r2go2 upload large-file.txt bucket/
✅ Connection established
✅ Authentication successful
❌ Upload failed: File too large (100MB limit)
💡 Suggestion: Use --chunk-size 50MB or check account limits
Retry? [y/N/q]
```

#### **Speed Optimization**
```bash
# Performance Features:
r2go2 copy --optimize --parallel source/ dest/
🚀 Detected optimal chunk size: 16MB
🔄 Using 8 concurrent connections
⚡ Estimated time: 45s (vs 2m)
```

## 📁 **Enhanced CLI File Structure**

### **New CLI Commands Structure**
```
cmd/
├── copy.go          # Enhanced copy with progress
├── move.go          # Move operations with confirmation
├── sync.go          # Sync operations with monitoring
├── upload.go        # Upload with metadata and progress
├── download.go      # Download with integrity checking
├── batch.go         # Batch operation manager
├── queue.go         # Operation queue management
└── progress.go       # Progress monitoring system
```

### **Enhanced Core Components**
```
internal/cli/
├── operations/
│   ├── copy.go               # Enhanced copy operations
│   ├── move.go               # Move with verification
│   ├── sync.go               # Sync with reconciliation
│   ├── upload.go             # Upload with multipart handling
│   ├── download.go           # Download with resumption
│   └── delete.go             # Safe deletion with confirm
├── progress/
│   ├── bar.go                 # Progress bar implementations
│   ├── spinner.go             # Loading animations
│   ├── speed.go               # Speed calculation
│   ├── eta.go                 # ETA estimation
│   └── display.go             # Progress visualization
├── batch/
│   ├── manager.go             # Batch operation manager
│   ├── worker.go              # Worker threads
│   ├── queue.go               # Operation queue
│   └── stats.go               # Statistics tracking
└── ux/
    ├── confirmation.go        # Interactive confirmations
    ├── feedback.go            # User feedback systems
    ├── retry.go               # Smart retry logic
    └── cancellation.go        # Operation cancellation
```

## ⚡ **Performance Optimization Strategies**

### **Intelligent Chunking**
```go
// Adaptive chunk size based on file size and network conditions
func CalculateOptimalChunkSize(fileSize int64, networkSpeed float64) int64 {
    if fileSize < 1*1024*1024 {          // < 1MB
        return 64 * 1024
    } else if fileSize < 10*1024*1024 {       // < 10MB
        return 256 * 1024
    } else if fileSize < 100*1024*1024 {     // < 100MB
        return 512 * 1024
    } else {
        return 8 * 1024 * 1024           // > 100MB
    }
}
```

### **Connection Pooling**
```go
// Reusable connection pool for API calls
type ConnectionPool struct {
    MaxIdle     int
    MaxActive   int
    IdleTimeout  time.Duration
    Connections  chan *Connection
    semaphore   *semaphore.Weighted
}
```

### **Parallel Processing**
```go
// Automatic concurrency based on operation count
func CalculateConcurrency(opCount int) int {
    if opCount < 5 {
        return 1
    } else if opCount < 20 {
        return opCount // 1:1 ratio for small batches
    } else {
        return min(20, opCount/2) // Max 20 parallel
    }
}
```

## 📊 **Monitoring and Statistics**

### **Real-time Statistics Dashboard**
```go
type Statistics struct {
    Operations      int           `json:"ops_completed"`
    TotalSize        int64         `json:"total_bytes"`
    CurrentSpeed     float64       `json:"speed_mbps"`
    AverageSpeed     float64       `json:"avg_speed_mbps"`
    PeakSpeed        float64       `json:"peak_speed_mbps"`
    Duration         time.Duration `json:"duration"`
    Errors            int           `json:"errors"`
    Retries           int           `json:"retries"`
    StartTime        time.Time     `json:"start_time"`
}
```

### **Progress Reporting**
```go
// JSON progress output for machine readability
type ProgressReport struct {
    Operation    string    `json:"operation"`
    Progress    float64   `json:"progress"`
    Speed        float64   `json:"speed_mbps"`
    Eta          string    `json:"eta"`
    CurrentSize  int64     `json:"current_bytes"`
    TotalSize    int64     `json:"total_bytes"`
    Rate         string    `json:"transfer_rate"`
}
```

## 🛡️ **Error Handling and Recovery**

### **Smart Error Categories**
```go
type ErrorType int

const (
    ErrorTypeNetwork ErrorType = iota
    ErrorTypePermission
    ErrorTypeStorage
    ErrorTypeAuthentication
    ErrorTypeQuota
    ErrorTypeValidation
    ErrorTypeUserCancel
)
```

### **Recovery Strategies**
```go
// Automatic retry logic with exponential backoff
type RetryStrategy struct {
    MaxRetries    int
    BaseDelay     time.Duration
    MaxDelay     time.Duration
    BackoffFactor float64
}

func (r *RetryStrategy) ShouldRetry(err error, attempt int) bool {
    if attempt >= r.MaxRetries {
        return false
    }
    if isRetryableError(err) {
        delay := time.Duration(float64(r.BaseDelay) * math.Pow(r.BackoffFactor, float64(attempt)))
        if delay > r.MaxDelay {
            delay = r.MaxDelay
        }
        // Implement retry with delay
        return true
    }
    return false
}
```

## 🎯 **Success Criteria**

### **Enhanced Features**
- [ ] Real-time progress bars for all file operations
- [ ] Batch operation support with cancellation
- [ ] Queue management for multiple operations
- [ ] Parallel processing for improved performance
- [ ] Smart retry logic with exponential backoff
- [ ] Transfer speed monitoring and optimization

### **Professional Quality**
- [ ] Comprehensive error handling and recovery
- [ ] Interactive confirmations for dangerous operations
- [ ] Progress estimation with ETA calculations
- [ ] Statistical reporting and monitoring
- [ ] Machine-readable JSON output integration

### **Performance Standards**
- [ ] Progress updates every 100ms (10Hz refresh rate)
- [ ] Sub-1ms response time for user input
- [ ] Memory usage under 50MB for large operations
- [ ] CPU usage under 20% during parallel operations
- [] Network optimization with connection pooling

## 🔧 **Implementation Dependencies**

### **Core Dependencies**
```go
require (
    github.com/charmbracelet/bubbletea v0.26.0
    github.com/charmbracelet/lipgloss v0.11.0
    github.com/spf13/viper    // Configuration management
    github.com/sirupsen/logrus // Structured logging
    github.com/briandowns/spin  // Progress indicators
    github.com/pkg/errors   // Error handling
)
```

### **Optional Dependencies**
```go
require (
    github.com/fatih/color      // Terminal colors
    github.com/jedib0/retry     // Retry with backoff
    github.com/cenkalti/backoff // Exponential backoff
    github.com/gorilla/websocket // Real-time monitoring
)
```

## 🚀 **Session Outcomes**

### **User Experience**
- **Visual Feedback**: Users see real-time progress instead of silent operations
- **Confidence Building**: Clear statistics and success indicators
- **Error Recovery**: Smart retry logic prevents failed operations
- **Efficiency**: Batch processing reduces manual intervention
- **Professionalism**: Enterprise-grade operation monitoring

### **Technical Excellence**
- **Modern Architecture**: Clean separation of concerns and reusable components
- **Performance Optimized**: Intelligent chunking and parallel processing
- **Error Resilient**: Comprehensive error handling and recovery strategies
- **Monitoring Ready**: Real-time statistics and progress reporting
- **Integration Ready**: JSON output for CI/CD pipeline integration

### **Business Value**
- **Time Savings**: Batch operations reduce manual operation time
- **Reliability**: Smart error handling increases success rates
- **Scalability**: Queue management handles enterprise workloads
- **Monitoring**: Real-time tracking provides operational insights
- **Professional**: Enhanced CLI matches industry standards

## 📋 **Implementation Plan**

### **Phase 1: Core Operations Enhancement (Week 1)**
- [ ] Enhanced copy/move operations with progress bars
- [ ] Real-time speed calculation and ETA estimation
- [ ] Basic error handling and retry logic
- [ ] User confirmation dialogs for destructive operations

### **Phase 2: Progress System (Week 1)**
- [ ] Reusable progress bar components
- [ ] Spinner and loading animations
- [ ] Progress estimation algorithms
- [] Visual feedback for all operation states

### **Phase 3: Batch Processing (Week 2)**
- [ ] Operation queue management system
- [ ] Worker thread pool implementation
- [ ] Concurrent operation coordination
- [ ] Statistics collection and reporting

### **Phase 4: Advanced Features (Week 2)**
- [ ] Cancellation support for long-running operations
- [ ] Resume capability for interrupted transfers
- [ ] Speed optimization and automatic tuning
- [ ] Monitoring dashboard integration

### **Phase 5: Integration and Polish (Week 3)**
- [ ] Integration with professional TUI interface
- [ ] JSON output for machine readability
- [ ] Configuration file support
- [ ] Testing and optimization
- [ ] Documentation and examples

## 🔧 **Development Workflow**

### **Code Standards**
- **Error Handling**: Use pkg/errors for typed error handling
- **Logging**: Structured logging with different levels
- **Testing**: Comprehensive unit tests and integration tests
- **Documentation**: Code comments and examples
- **Performance**: Benchmarking and optimization

### **Testing Strategy**
- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test complete workflows
- **Performance Tests**: Validate speed and resource usage
- **User Tests**: Validate user experience
- **Edge Cases**: Test error conditions and recovery

### **Code Review Process**
- **Peer Review**: All code must be peer-reviewed
- **Style Checking**: Automated style and lint checks
- **Security Review**: Security vulnerability assessment
- **Performance Review**: Resource usage optimization
- **Documentation Review**: Accuracy and completeness

## 🎯 **Session Success Indicators**

### **Feature Completeness**
- [ ] All basic operations enhanced with progress
- [ ] Batch operation support implemented
- [ ] Progress monitoring system complete
- [ ] Error handling and recovery robust
- [ ] Performance optimization achieved

### **Quality Standards**
- [ ] Code quality meets project standards
- [ ] Test coverage exceeds 90%
- [ ] Performance meets benchmarks
- [ ] Error rates below 1%
- [ ] User experience significantly improved

### **Integration Success**
- [ ] Seamless integration with existing CLI
- [] Compatible with professional TUI interface
- [ ] JSON output for automation
- [] Configuration file support
- [ ] Documentation complete

---

**Session Ready for Implementation**: ✅ **COMPLETE PROMPT**
**Next Actions**: Begin with Phase 1 implementation of enhanced file operations with progress monitoring