# Enhanced CLI Features Documentation

**Professional-grade file operations with real-time monitoring and advanced capabilities**

## 📋 Table of Contents

1. [Overview](#-overview)
2. [Progress Monitoring System](#-progress-monitoring-system)
3. [Enhanced Copy Operations](#-enhanced-copy-operations)
4. [Batch Operation Manager](#-batch-operation-manager)
5. [Error Handling & Recovery](#-error-handling--recovery)
6. [User Experience Enhancements](#-user-experience-enhancements)
7. [Performance Optimization](#-performance-optimization)
8. [Integration & Automation](#-integration--automation)
9. [Technical Architecture](#-technical-architecture)

---

## 🎯 Overview

The enhanced CLI features transform R2Go2 from a basic command-line tool into a **professional-grade utility** that rivals commercial file transfer applications. These features are designed for enterprise use cases, large-scale operations, and scenarios where reliability, performance, and user experience are critical.

### Key Features

- **Real-time Progress Monitoring**: Live feedback with speed, ETA, and customizable displays
- **Batch Operations**: Process multiple files/directories concurrently with intelligent queuing
- **Smart Error Recovery**: Automatic retry logic with exponential backoff and contextual suggestions
- **Interactive Confirmations**: Safe operations with detailed previews and user control
- **Performance Optimization**: Adaptive chunking, parallel processing, and memory efficiency
- **Professional Output**: Enterprise-grade formatting, statistics, and logging

---

## 📊 Progress Monitoring System

### Real-time Progress Tracking

The progress monitoring system provides comprehensive real-time feedback during file operations:

```bash
# Example output during file copy
📁 Copying: large-dataset.tar.gz -> backup/large-dataset.tar.gz
📊 Size: 15.2GB
🔄 Resume mode enabled
✅ Verification enabled

⚡ Progress: 67.3% | 10.2GB/15.2GB | 89.4 MB/s | ETA: 1m 45s
```

### Progress Bar Types

The system supports multiple progress display formats:

- **Linear Progress Bar**: Classic horizontal progress bar with percentage
- **Circular Progress**: Rotating spinner for indeterminate operations
- **Percentage Display**: Simple numeric percentage with time remaining
- **Multi-progress**: Concurrent tracking of multiple operations

### Speed & ETA Calculation

- **Real-time Speed Monitoring**: Updates every 100ms (configurable)
- **Intelligent ETA**: Based on current speed with smoothing algorithms
- **Adaptive Refresh**: Higher refresh rates for fast operations, lower for slow ones
- **Bandwidth Optimization**: Minimal overhead on transfer performance

### Customization Options

```bash
# Custom refresh rate
r2go2 copy --progress --refresh-rate 50ms large-file.txt backup/

# Quiet progress (minimal output)
r2go2 copy --progress --quiet batch-operation/ dest/

# Verbose progress with detailed stats
r2go2 copy --progress --stats --verbose source/ dest/
```

---

## 🔄 Enhanced Copy Operations

### Basic Copy with Enhancement

```bash
# Standard copy with progress monitoring
r2go2 copy source.txt destination.txt

# Copy with comprehensive statistics
r2go2 copy source.txt destination.txt --stats

# Output example:
# 📁 Copying: source.txt -> destination.txt
# 📊 Size: 1.2GB
# ⚡ Progress: 100.0% | 1.2GB/1.2GB | 67.2 MB/s | ETA: Done
#
# 📊 Copy Results:
#   Source:      source.txt
#   Destination: destination.txt
#   Size:        1.2GB
#   Duration:    18s
#   Speed:       67.2 MB/s
#   Verified:    true
#   Resumed:     No
```

### Advanced Copy Features

#### Resume Capability

```bash
# Resume interrupted transfers
r2go2 copy --resume large-dataset.tar.gz backup/

# System automatically detects partial transfers:
# ✅ Partial file detected: 8.5GB/15.2GB (55.9%)
# 🔄 Resuming from byte 8,500,000,000
# ⚡ Progress: 100.0% | 15.2GB/15.2GB | 89.4 MB/s | ETA: Done
```

#### Integrity Verification

```bash
# Copy with checksum verification
r2go2 copy --verify --preserve critical-data.tar.gz backup/

# Output includes verification:
# ✅ Copy completed successfully
# 🔍 Verifying file integrity...
# ✅ Verification passed: SHA-256 checksums match
```

#### Performance Optimization

```bash
# High-performance parallel copy
r2go2 copy --parallel 8 --chunk-size 16MB source/ dest/

# Adaptive chunking based on file size
r2go2 copy --adaptive-chunking source/ dest/

# Memory-efficient streaming for large files
r2go2 copy --stream-buffer 64MB huge-dataset.tar.gz backup/
```

### Copy Options Reference

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--progress, -P` | Show progress bars | `true` | `--progress` |
| `--stats, -s` | Show detailed statistics | `false` | `--stats` |
| `--parallel, -j` | Number of parallel operations | `4` | `--parallel 8` |
| `--chunk-size` | Chunk size for large files | `8MB` | `--chunk-size 16MB` |
| `--resume` | Resume interrupted transfers | `false` | `--resume` |
| `--verify, -V` | Verify file integrity | `true` | `--verify` |
| `--overwrite, -o` | Overwrite existing files | `false` | `--overwrite` |
| `--preserve, -p` | Preserve file attributes | `true` | `--preserve` |
| `--recursive, -r` | Copy directories recursively | `false` | `--recursive` |
| `--quiet, -q` | Suppress output except errors | `false` | `--quiet` |
| `--timeout` | Operation timeout | `30m` | `--timeout 1h` |
| `--retries` | Number of retry attempts | `3` | `--retries 5` |
| `--dry-run` | Preview operations without executing | `false` | `--dry-run` |

---

## 📦 Batch Operation Manager

### Batch Copy from File List

```bash
# Create file list
cat > files.txt << EOF
# Simple format: source -> destination
./dist/app.js -> ./backup/app.js
./dist/styles.css -> ./backup/styles.css
./dist/images/logo.png -> ./backup/images/logo.png

# Simple format: source files to common destination
./config/production.json
./config/database.json
EOF

# Execute batch copy
r2go2 copy --batch files.txt backup/ --parallel 4

# Output:
# 📦 Processing batch copy operations...
# ⚡ Progress: 75.0% | 18/24 | 67.2 MB/s | ETA: 45s
#
# 📊 Batch Copy Results:
#   Total:        24
#   Completed:    18
#   Failed:       0
#   Skipped:      6
#   Duration:     3m 15s
#   Total Size:   1.8GB
#   Avg Speed:    67.8 MB/s
```

### JSON Batch Specifications

```json
{
  "config": {
    "concurrency": 8,
    "continue_on_error": true,
    "retries": 3,
    "timeout": "30m",
    "dry_run": false,
    "quiet": false,
    "interactive": true
  },
  "operations": [
    {
      "id": "copy_1",
      "type": "copy",
      "source": "./dist/app.js",
      "destination": "./backup/app.js",
      "options": {
        "verify": true,
        "preserve": true,
        "chunk_size": 16777216
      }
    },
    {
      "id": "copy_2",
      "type": "copy",
      "source": "./dist/styles.css",
      "destination": "./backup/styles.css",
      "options": {
        "overwrite": true,
        "resume": false
      }
    },
    {
      "id": "copy_3",
      "type": "delete",
      "source": "./temp/old-cache/",
      "options": {
        "recursive": true
      }
    }
  ]
}
```

### Advanced Batch Features

#### Interactive Confirmation

```bash
# Interactive batch processing
r2go2 copy --batch files.txt backup/ --interactive

# Output:
# 📦 Perform batch copy operation on 45 items?
#
# Operations:
#   • ./dist/app.js -> ./backup/app.js
#   • ./dist/styles.css -> ./backup/styles.css
#   • ./dist/images/logo.png -> ./backup/images/logo.png
#   • ... and 42 more operations
#
# ⚠️ This will affect multiple files/folders
# [Y]es/[N]o/[A]ll/[C]ancel: y
```

#### Progress Monitoring

```bash
# Real-time batch progress
r2go2 copy --batch large-batch.json dest/ --parallel 12 --stats

# Live output updates every 100ms:
# ⚡ Progress: 45.2% | 234/517 | 89.4 MB/s | ETA: 3m 15s
#    Active: [copy_124] [copy_125] [copy_126] [copy_127] [copy_128]
#    Queue: 283 remaining
#    Errors: 2 (network timeout)
#    Speed: 89.4 MB/s | Peak: 127.3 MB/s
```

#### Error Handling

```bash
# Continue on errors with detailed reporting
r2go2 copy --batch files.txt backup/ --continue-on-error --retries 5

# Error reporting:
# ❌ Operation failed: copy_89
#    Error: Network timeout during copy
#    Source: ./dist/large-video.mp4
#    Destination: ./backup/large-video.mp4
#    💡 Suggestion: Check network connection and retry
#    🔄 Retrying... (attempt 2/5)
```

---

## 🛡️ Error Handling & Recovery

### Smart Error Classification

The system automatically classifies errors and provides contextual suggestions:

```bash
# Network error example
❌ Copy failed: connection refused
   Type: Network Error
   💡 Suggestion: Check network connection and try again
   🔄 Retrying with exponential backoff... (attempt 1/3)

# Permission error example
❌ Copy failed: permission denied
   Type: Permission Error
   💡 Suggestion: Check file permissions and user privileges
   ❌ This error is not retryable

# Storage error example
❌ Copy failed: no space left on device
   Type: Storage Error
   💡 Suggestion: Free up disk space and try again
   ❌ This error is not retryable
```

### Retry Strategies

```bash
# Network retry strategy (optimized for network issues)
r2go2 copy --retry-strategy network --retries 5 source.txt dest/

# File system retry strategy (minimal retries for local issues)
r2go2 copy --retry-strategy filesystem --retries 2 source.txt dest/

# Custom retry configuration
r2go2 copy --retry-base-delay 2s --retry-max-delay 60s --retry-backoff 2.5 source.txt dest/
```

### Interactive Recovery

```bash
# Interactive error handling
r2go2 copy --interactive --retries 3 unstable-source.txt dest/

# On error:
# ❌ Operation failed (attempt 2):
#    Error: Network timeout during copy
#    Type: Network Error
#    💡 Suggestion: Check network connection and retry
#
# 🔄 Retry operation? [R]etry/[C]ancel: r
```

---

## 👥 User Experience Enhancements

### Interactive Confirmations

#### Destructive Operation Warnings

```bash
# Delete operation with confirmation
r2go2 copy --delete-source --interactive source/ dest/

# Confirmation dialog:
# ⚠️  This will delete source/ and all its contents. Continue? [y/N]
```

#### Batch Operation Preview

```bash
# Batch operation preview
r2go2 copy --batch files.txt dest/ --dry-run --interactive

# Preview:
# 📋 Batch Operations Preview (45 items):
#
# Operations:
#   • Copy: ./dist/app.js -> dest/app.js (2.1MB)
#   • Copy: ./dist/styles.css -> dest/styles.css (156KB)
#   • Copy: ./dist/images/logo.png -> dest/logo.png (89KB)
#   • ... and 42 more operations
#
# Total Size: 1.8GB
# Estimated Time: 3m 30s
#
# ⚠️ This will affect 45 files
# [Y]es/[N]o/[A]ll/[C]ancel:
```

#### Progress Feedback

```bash
# Rich progress information
r2go2 copy --progress --stats --verbose large-dataset/ backup/

# Detailed progress:
# 📁 Copying: large-dataset/ -> backup/
# 📊 Total Size: 45.2GB | Files: 1,847
# ⚡ Progress: 67.3% | 1,242/1,847 files | 89.4 MB/s | ETA: 3m 15s
#
# 📊 Current Operation:
#    File: large-dataset/videos/presentation.mp4
#    Size: 2.1GB / 2.1GB (100%)
#    Speed: 127.3 MB/s
#    Time: 45s
#
# 📊 Batch Statistics:
#    Completed: 1,242 files (28.7GB)
#    Remaining: 605 files (16.5GB)
#    Errors: 0
#    Avg Speed: 89.4 MB/s
#    Peak Speed: 127.3 MB/s
```

### Professional Output Formatting

#### Color-coded Messages

- ✅ **Success**: Green checkmarks for successful operations
- ❌ **Errors**: Red indicators for failures with detailed messages
- ⚠️ **Warnings**: Yellow alerts for potential issues
- 💡 **Suggestions**: Blue tips for problem resolution
- ⚡ **Progress**: Cyan indicators for active operations

#### Structured Output

```bash
# Machine-readable JSON output
r2go2 copy --json --stats source.txt dest.txt

{
  "success": true,
  "operation": "copy",
  "source": "source.txt",
  "destination": "dest.txt",
  "results": {
    "size": 1289748480,
    "duration": "18.234s",
    "speed": 67.2,
    "verified": true,
    "resumed": false,
    "chunks": 77
  },
  "timestamp": "2025-01-24T15:30:45Z"
}
```

---

## ⚡ Performance Optimization

### Adaptive Chunking

The system automatically optimizes chunk sizes based on file characteristics:

```bash
# Automatic optimization (default)
r2go2 copy source/ dest/ --adaptive-chunking

# Manual chunk size specification
r2go2 copy --chunk-size 32MB huge-dataset.tar.gz backup/
```

**Automatic Chunk Size Logic:**
- **< 1MB files**: 64KB chunks
- **1-10MB files**: 256KB chunks
- **10-100MB files**: 512KB chunks
- **> 100MB files**: 8MB chunks

### Parallel Processing

```bash
# Optimal parallelism based on operation count
r2go2 copy --parallel auto source/ dest/  # Automatically determines optimal count

# Manual parallelism control
r2go2 copy --parallel 16 source/ dest/  # 16 concurrent operations

# CPU-aware optimization
r2go2 copy --parallel-cpu-aware source/ dest/  # Based on CPU cores
```

### Memory Efficiency

```bash
# Memory-efficient streaming for large files
r2go2 copy --stream-buffer 64MB huge-file.tar.gz backup/

# Low memory mode (slower but minimal RAM usage)
r2go2 copy --low-memory source/ dest/
```

### Bandwidth Optimization

```bash
# Bandwidth limiting for network operations
r2go2 copy --bandwidth-limit 100MB/s source.txt dest/

# Adaptive bandwidth based on network conditions
r2go2 copy --adaptive-bandwidth source/ dest/
```

---

## 🔧 Integration & Automation

### CI/CD Integration

```yaml
# GitHub Actions workflow with enhanced copy
- name: Deploy to R2 with Progress
  env:
    CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
    CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
  run: |
    ./r2go2 copy --batch deployment-manifest.json r2://my-bucket/ \
      --parallel 8 \
      --verify \
      --stats \
      --json > deployment-results.json

    # Post to Slack with results
    echo "Deployment completed: $(cat deployment-results.json | jq -r '.results.duration')" | \
      curl -X POST -H 'Content-type: application/json' \
      --data '{"text":"'"$(cat)"'"}' \
      ${{ secrets.SLACK_WEBHOOK }}
```

### Scripting & Automation

```bash
#!/bin/bash
# Enhanced backup script with progress monitoring

SOURCE_DIR="/data/important"
BACKUP_DIR="/backup/$(date +%Y-%m-%d)"
R2_BUCKET="daily-backups"

echo "🚀 Starting enhanced backup process..."

# Create local backup with progress
r2go2 copy --recursive \
  --parallel 8 \
  --verify \
  --stats \
  "$SOURCE_DIR" \
  "$BACKUP_DIR"

# Compress and upload to R2
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"

r2go2 copy --resume \
  --chunk-size 32MB \
  --progress \
  "$BACKUP_DIR.tar.gz" \
  "r2://$R2_BUCKET/backups/$(date +%Y-%m-%d).tar.gz"

# Clean up local backup
rm -rf "$BACKUP_DIR" "$BACKUP_DIR.tar.gz"

echo "✅ Enhanced backup completed successfully!"
```

### Monitoring & Alerting

```bash
# Real-time monitoring with alerts
r2go2 copy --batch large-dataset.json r2://archive/ \
  --parallel 12 \
  --monitor \
  --alert-webhook https://hooks.slack.com/... \
  --alert-threshold errors=5,speed-drop=50%
```

---

## 🏗️ Technical Architecture

### Component Architecture

```
Enhanced CLI Features
├── Progress Monitoring System
│   ├── Real-time Progress Bars
│   ├── Speed & ETA Calculation
│   ├── Multi-progress Tracking
│   └── Customizable Display Formats
├── Batch Operation Manager
│   ├── Queue Management
│   ├── Worker Pool Coordination
│   ├── Statistics Collection
│   └── Error Handling & Recovery
├── Enhanced File Operations
│   ├── Smart Copy with Resume
│   ├── Integrity Verification
│   ├── Adaptive Chunking
│   └── Performance Optimization
└── User Experience Components
    ├── Interactive Confirmations
    ├── Error Classification
    ├── Professional Output
    └── Smart Retry Logic
```

### Performance Characteristics

- **Progress Update Rate**: 100ms (configurable)
- **Memory Usage**: < 50MB for large operations
- **CPU Usage**: < 20% during parallel operations
- **Network Efficiency**: Optimized chunking and connection pooling
- **Error Recovery**: < 1s retry overhead with exponential backoff

### Thread Safety & Concurrency

- **Atomic Operations**: Thread-safe counters and statistics
- **Goroutine Coordination**: Proper synchronization for worker pools
- **Context Management**: Clean cancellation and timeout handling
- **Resource Cleanup**: Automatic resource release on completion

### Extensibility

- **Plugin Architecture**: Easy addition of new operation types
- **Progress Interface**: Standardized progress tracking for all operations
- **Error Handling Framework**: Consistent error classification and recovery
- **Configuration System**: Flexible option management and validation

---

## 📚 Further Reading

- [Basic Usage Guide](USAGE.md) - Comprehensive command reference
- [API Documentation](docs/api/) - Programmatic interface details
- [Performance Tuning](docs/performance.md) - Optimization techniques
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
- [Best Practices](docs/best-practices.md) - Production deployment guidelines

---

**Version**: 1.0.0 | **Last Updated**: 2025-01-24 | **License**: MIT

*Enhanced CLI features built with ❤️ by CosmoLabs for enterprise-grade file operations*