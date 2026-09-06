---
brainstorm_ref: docs/brainstorming/2026-06-08-road007-s3-migration-resume.md
created: 2026-06-08T03:30:00-03:00
deliverables:
    - P-01: Checkpoint persistence — save/load/delete with atomic writes
    - P-02: Worker pool — concurrent S3→R2 streaming with retry
    - P-03: Orchestrator — wire checkpoint + workers into Execute()
    - P-04: ETag verification mode
    - P-05: Documentation and close-out
last_review_content_hash: e2de010d7627b7981474bc63a1e64d199459ea3958e8080b8b9ab993f8ab6cf6
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-06-08-road007-s3-migration-resume.md
last_reviewed: "2026-09-06T20:14:04.094279+04:00"
origin: ROAD-007
status: COMPLETED
title: 'ROAD-007: S3 to R2 Migration with Resume Support'
updated: 2026-09-06T00:00:00-03:00
---

# ROAD-007: S3 to R2 Migration with Resume Support

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the simulated migration loop with real S3→R2 streaming transfers, checkpoint-based resume, a concurrent worker pool, and graceful SIGINT handling.

**Architecture:** Three files in `internal/migration/`: `checkpoint.go` (load/save/delete state from `~/.cosmoflare/migrations/`), `worker.go` (goroutine pool streaming S3 GetObject to R2 Upload with retry), and `s3.go` (orchestrator wiring checkpoint + workers + SIGINT + progress bar). The checkpoint uses SHA256 hash of bucket+filter for deterministic file names.

**Tech Stack:** Go, AWS SDK v2 (`s3`), `pkg/cosmoflare` (R2Client), `cheggaaa/pb/v3` (progress bar), `sync` (Mutex, WaitGroup, atomic)

**Brainstorm ref:** `docs/brainstorming/2026-06-08-road007-s3-migration-resume.md`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/migration/checkpoint.go` | **New.** `Checkpoint` struct, load/save/delete, hash path, periodic flusher |
| `internal/migration/checkpoint_test.go` | **New.** Round-trip, hash determinism, corruption recovery |
| `internal/migration/worker.go` | **New.** `transferWorker`, `transferResult`, retry with backoff, size-based upload routing |
| `internal/migration/worker_test.go` | **New.** Retry logic, result reporting |
| `internal/migration/s3.go` | **Modified.** Replace simulated loop with checkpoint+worker orchestration, SIGINT, progress |

---

### Task 1: Checkpoint Persistence

**Files:**
- Create: `internal/migration/checkpoint.go`
- Create: `internal/migration/checkpoint_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/migration/checkpoint_test.go
package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointHashDeterministic(t *testing.T) {
	h1 := checkpointHash("src", "dst", "images/*")
	h2 := checkpointHash("src", "dst", "images/*")
	if h1 != h2 {
		t.Errorf("same inputs should produce same hash: %s != %s", h1, h2)
	}
	h3 := checkpointHash("src", "dst", "other/*")
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestCheckpointSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cp := &Checkpoint{
		Version:   1,
		S3Bucket:  "src",
		R2Bucket:  "dst",
		Filter:    "*.jpg",
		Completed: map[string]int64{"a.jpg": 100, "b.jpg": 200},
		Failed:    []string{"c.jpg"},
		TransferredSize: 300,
		TotalObjects: 10,
		TotalSize: 5000,
	}

	path := filepath.Join(dir, "test.json")
	if err := saveCheckpoint(cp, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := loadCheckpoint(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.S3Bucket != "src" {
		t.Errorf("S3Bucket = %q, want %q", loaded.S3Bucket, "src")
	}
	if len(loaded.Completed) != 2 {
		t.Errorf("Completed count = %d, want 2", len(loaded.Completed))
	}
	if loaded.Completed["a.jpg"] != 100 {
		t.Errorf("Completed[a.jpg] = %d, want 100", loaded.Completed["a.jpg"])
	}
	if len(loaded.Failed) != 1 || loaded.Failed[0] != "c.jpg" {
		t.Errorf("Failed = %v, want [c.jpg]", loaded.Failed)
	}
}

func TestCheckpointDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	saveCheckpoint(cp, path)

	if err := deleteCheckpoint(path); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestCheckpointLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	os.WriteFile(path, []byte("{invalid json"), 0644)

	_, err := loadCheckpoint(path)
	if err == nil {
		t.Error("expected error loading corrupt checkpoint")
	}
}

func TestCheckpointLoadMissing(t *testing.T) {
	cp, err := loadCheckpoint("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cp != nil {
		t.Error("missing file should return nil checkpoint")
	}
}

func TestCheckpointPath(t *testing.T) {
	dir := t.TempDir()
	path := checkpointPathIn(dir, "src", "dst", "filter")
	if filepath.Dir(path) != dir {
		t.Errorf("should be in %s, got %s", dir, filepath.Dir(path))
	}
	if filepath.Ext(path) != ".json" {
		t.Error("should have .json extension")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/migration/ -run TestCheckpoint -v`
Expected: FAIL — types and functions not defined

- [ ] **Step 3: Implement checkpoint.go**

```go
// internal/migration/checkpoint.go
package migration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Checkpoint struct {
	Version         int              `json:"version"`
	S3Bucket        string           `json:"s3_bucket"`
	R2Bucket        string           `json:"r2_bucket"`
	Filter          string           `json:"filter"`
	StartedAt       time.Time        `json:"started_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	TotalObjects    int64            `json:"total_objects"`
	TotalSize       int64            `json:"total_size"`
	Completed       map[string]int64 `json:"completed"`
	Failed          []string         `json:"failed"`
	TransferredSize int64            `json:"transferred_size"`

	mu   sync.Mutex `json:"-"`
	path string     `json:"-"`
}

func checkpointHash(s3Bucket, r2Bucket, filter string) string {
	h := sha256.Sum256([]byte(s3Bucket + ":" + r2Bucket + ":" + filter))
	return fmt.Sprintf("%x", h[:8])
}

func checkpointDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cosmoflare", "migrations")
}

func checkpointPathIn(dir, s3Bucket, r2Bucket, filter string) string {
	return filepath.Join(dir, checkpointHash(s3Bucket, r2Bucket, filter)+".json")
}

func (m *S3Migration) checkpointPath() string {
	return checkpointPathIn(checkpointDir(), m.S3Bucket, m.R2Bucket, m.Filter)
}

func loadCheckpoint(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("corrupt checkpoint: %w", err)
	}
	cp.path = path
	return &cp, nil
}

func saveCheckpoint(cp *Checkpoint, path string) error {
	cp.UpdatedAt = time.Now()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint dir: %w", err)
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize checkpoint: %w", err)
	}

	return nil
}

func deleteCheckpoint(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (cp *Checkpoint) markCompleted(key string, size int64) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.Completed[key] = size
	cp.TransferredSize += size
}

func (cp *Checkpoint) markFailed(key string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.Failed = append(cp.Failed, key)
}

func (cp *Checkpoint) isCompleted(key string) bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	_, ok := cp.Completed[key]
	return ok
}

func (cp *Checkpoint) flush() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return saveCheckpoint(cp, cp.path)
}

const (
	flushInterval = 30 * time.Second
	flushCount    = 10
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/migration/ -run TestCheckpoint -v`
Expected: All 6 PASS

- [ ] **Step 5: Commit**

```bash
ccs commit --direct -m "feat(migration): add checkpoint persistence with atomic writes (ROAD-007 P-01)"
```

---

### Task 2: Worker Pool with Retry

**Files:**
- Create: `internal/migration/worker.go`
- Create: `internal/migration/worker_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/migration/worker_test.go
package migration

import (
	"testing"
	"time"
)

func TestTransferResultSuccess(t *testing.T) {
	r := transferResult{Key: "a.jpg", Size: 1024, Err: nil}
	if r.Err != nil {
		t.Error("expected no error")
	}
	if r.Key != "a.jpg" {
		t.Errorf("Key = %q, want %q", r.Key, "a.jpg")
	}
}

func TestRetryBackoff(t *testing.T) {
	delays := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for i, expected := range delays {
		got := retryDelay(i)
		if got != expected {
			t.Errorf("retryDelay(%d) = %v, want %v", i, got, expected)
		}
	}
}

func TestTransferTimeout(t *testing.T) {
	small := transferTimeout(1024)
	if small < 5*time.Minute {
		t.Errorf("small object timeout should be at least 5m, got %v", small)
	}

	large := transferTimeout(10 * 1024 * 1024 * 1024)
	if large < 10*time.Minute {
		t.Errorf("10GB object should have timeout > 10m, got %v", large)
	}
}

func TestShouldUseMultipart(t *testing.T) {
	if shouldUseMultipart(50 * 1024 * 1024) {
		t.Error("50MB should not use multipart")
	}
	if !shouldUseMultipart(200 * 1024 * 1024) {
		t.Error("200MB should use multipart")
	}
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/migration/ -run "TestTransfer|TestRetry|TestShould" -v`
Expected: FAIL

- [ ] **Step 3: Implement worker.go**

```go
// internal/migration/worker.go
package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

const (
	maxRetries       = 3
	multipartMinSize = 100 * 1024 * 1024 // 100MB
)

type transferResult struct {
	Key  string
	Size int64
	Err  error
}

func retryDelay(attempt int) time.Duration {
	base := time.Second
	for i := 0; i < attempt; i++ {
		base *= 4
	}
	return base
}

func transferTimeout(size int64) time.Duration {
	minTimeout := 5 * time.Minute
	sizeBasedTimeout := time.Duration(size/(1024*1024)) * time.Second // 1MB/s
	if sizeBasedTimeout > minTimeout {
		return sizeBasedTimeout
	}
	return minTimeout
}

func shouldUseMultipart(size int64) bool {
	return size >= multipartMinSize
}

func transferWorker(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	work <-chan *S3Object,
	results chan<- transferResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for obj := range work {
		result := transferObject(s3Client, r2Client, s3Bucket, r2Bucket, obj)
		results <- result
	}
}

func transferObject(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	obj *S3Object,
) transferResult {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay(attempt - 1))
		}

		err := doTransfer(s3Client, r2Client, s3Bucket, r2Bucket, obj)
		if err == nil {
			return transferResult{Key: obj.Key, Size: obj.Size}
		}
		lastErr = err
	}

	return transferResult{Key: obj.Key, Err: fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)}
}

func doTransfer(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	obj *S3Object,
) error {
	timeout := transferTimeout(obj.Size)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	getOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(obj.Key),
	})
	if err != nil {
		return fmt.Errorf("S3 GetObject failed for %s: %w", obj.Key, err)
	}
	defer getOutput.Body.Close()

	size := obj.Size
	if getOutput.ContentLength != nil {
		size = *getOutput.ContentLength
	}

	if shouldUseMultipart(size) {
		_, err = r2Client.MultipartUpload(ctx, r2Bucket, obj.Key, getOutput.Body, size)
	} else {
		_, err = r2Client.Upload(ctx, r2Bucket, obj.Key, getOutput.Body, size)
	}
	if err != nil {
		return fmt.Errorf("R2 upload failed for %s: %w", obj.Key, err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/migration/ -run "TestTransfer|TestRetry|TestShould" -v`
Expected: All 4 PASS

- [ ] **Step 5: Run build**

Run: `go build -o /dev/null .`
Expected: Success

- [ ] **Step 6: Commit**

```bash
ccs commit --direct -m "feat(migration): add worker pool with retry and multipart routing (ROAD-007 P-02)"
```

---

### Task 3: Orchestrator — Wire into Execute()

**Files:**
- Modify: `internal/migration/s3.go` (replace lines 77-157)

- [ ] **Step 1: Replace the Execute() method**

Replace the entire `Execute()` function body (lines 77-157 of `s3.go`) with the real orchestrator. Keep the function signature `func (m *S3Migration) Execute(r2Client cosmoflare.R2Client) (*MigrationResult, error)`.

Add imports at the top of s3.go: `"os/signal"`, `"sync"`, `"sync/atomic"`, `"syscall"`.

```go
func (m *S3Migration) Execute(r2Client cosmoflare.R2Client) (*MigrationResult, error) {
	startTime := time.Now()
	if m.Concurrency <= 0 {
		m.Concurrency = 10
	}

	printInfo("🚀 Starting S3 to R2 migration")
	printInfo("Source: s3://%s", m.S3Bucket)
	printInfo("Destination: %s", m.R2Bucket)
	printInfo("Concurrency: %d", m.Concurrency)

	s3Client, err := m.createS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	objects, err := m.listS3Objects(s3Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	if len(objects) == 0 {
		printInfo("No objects to migrate")
		return &MigrationResult{Duration: time.Since(startTime)}, nil
	}

	totalSize := m.getTotalSize(objects)
	printInfo("Found %d objects (%s)", len(objects), FormatBytes(totalSize))

	if m.DryRun {
		return m.performDryRun(objects)
	}

	// Load or create checkpoint
	cpPath := m.checkpointPath()
	var cp *Checkpoint

	if m.Resume {
		cp, err = loadCheckpoint(cpPath)
		if err != nil {
			printWarning("Corrupt checkpoint, starting fresh: %v", err)
			cp = nil
		}
	} else {
		existing, _ := loadCheckpoint(cpPath)
		if existing != nil {
			printWarning("Stale checkpoint found. Use --resume to continue or delete %s", cpPath)
		}
	}

	if cp == nil {
		cp = &Checkpoint{
			Version:      1,
			S3Bucket:     m.S3Bucket,
			R2Bucket:     m.R2Bucket,
			Filter:       m.Filter,
			StartedAt:    startTime,
			TotalObjects: int64(len(objects)),
			TotalSize:    totalSize,
			Completed:    make(map[string]int64),
			path:         cpPath,
		}
	} else {
		cp.path = cpPath
		printInfo("Resuming: %d/%d already transferred (%s)",
			len(cp.Completed), cp.TotalObjects, FormatBytes(cp.TransferredSize))
	}

	// Filter remaining objects
	var remaining []*S3Object
	for _, obj := range objects {
		if !cp.isCompleted(obj.Key) {
			remaining = append(remaining, obj)
		}
	}

	if len(remaining) == 0 {
		printInfo("All objects already transferred")
		deleteCheckpoint(cpPath)
		return &MigrationResult{
			TotalObjects:    int64(len(objects)),
			SuccessCount:    int64(len(objects)),
			TransferredSize: totalSize,
			Duration:        time.Since(startTime),
		}, nil
	}

	printInfo("%d objects remaining", len(remaining))

	// Setup SIGINT handling
	var stopping atomic.Bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create channels
	work := make(chan *S3Object, m.Concurrency)
	results := make(chan transferResult, m.Concurrency)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.Concurrency; i++ {
		wg.Add(1)
		go transferWorker(s3Client, r2Client, m.S3Bucket, m.R2Bucket, work, results, &wg)
	}

	// Close results channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Feed objects to workers
	go func() {
		for _, obj := range remaining {
			if stopping.Load() {
				break
			}
			work <- obj
		}
		close(work)
	}()

	// Handle SIGINT in background
	go func() {
		<-sigCh
		stopping.Store(true)
		signal.Stop(sigCh)
	}()

	// Progress bar
	bar := pb.StartNew(len(remaining))
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{rtime . }} {{etime . }}`)

	// Periodic checkpoint flush
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	sinceLastFlush := 0

	// Collect results
	result := &MigrationResult{
		TotalObjects: int64(len(objects)),
		SkippedCount: int64(len(objects) - len(remaining)),
	}

	for r := range results {
		if r.Err != nil {
			cp.markFailed(r.Key)
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", r.Key, r.Err))
		} else {
			cp.markCompleted(r.Key, r.Size)
			result.SuccessCount++
			result.TransferredSize += r.Size
		}
		bar.Increment()
		sinceLastFlush++

		// Periodic flush
		select {
		case <-flushTicker.C:
			cp.flush()
			sinceLastFlush = 0
		default:
			if sinceLastFlush >= flushCount {
				cp.flush()
				sinceLastFlush = 0
			}
		}
	}

	bar.Finish()
	result.Duration = time.Since(startTime)

	// Final checkpoint handling
	if result.ErrorCount == 0 && !stopping.Load() {
		deleteCheckpoint(cpPath)
		printSuccess("✅ Migration completed successfully!")
	} else if stopping.Load() {
		cp.flush()
		printWarning("Interrupted: %d/%d transferred. Run with --resume to continue.",
			result.SuccessCount+result.SkippedCount, result.TotalObjects)
		printMigrationSummary(result)
		return result, fmt.Errorf("migration interrupted")
	} else {
		cp.flush()
		printWarning("%d objects failed. Run with --resume to retry.", result.ErrorCount)
	}

	// Verify if requested
	if m.Verify && !stopping.Load() && result.ErrorCount == 0 {
		m.verifyTransfers(s3Client, r2Client, objects, result)
	}

	printMigrationSummary(result)
	return result, nil
}
```

- [ ] **Step 2: Run build**

Run: `go build -o /dev/null .`
Expected: Success (may need to fix import paths)

- [ ] **Step 3: Run existing tests**

Run: `go test ./internal/migration/ -timeout 30s`
Expected: Pass (no existing s3_test.go, only checkpoint and worker tests)

- [ ] **Step 4: Commit**

```bash
ccs commit --direct -m "feat(migration): replace simulated loop with real checkpoint+worker orchestrator (ROAD-007 P-03)"
```

---

### Task 4: ETag Verification Mode

**Files:**
- Modify: `internal/migration/s3.go` (add verifyTransfers method)

- [ ] **Step 1: Add verification method**

Add to `s3.go` after the `Execute()` function:

```go
func (m *S3Migration) verifyTransfers(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	objects []*S3Object,
	result *MigrationResult,
) {
	printInfo("🔍 Verifying transferred objects...")

	mismatchCount := 0
	bar := pb.StartNew(len(objects))
	bar.SetTemplateString(`Verifying: {{counters . }} {{bar . }} {{percent . }}`)

	for _, obj := range objects {
		bar.Increment()

		r2Head, err := r2Client.HeadObject(context.Background(), m.R2Bucket, obj.Key)
		if err != nil {
			printWarning("⚠ Verify failed for %s: %v", obj.Key, err)
			mismatchCount++
			continue
		}

		s3ETag := strings.Trim(obj.ETag, "\"")
		r2ETag := strings.Trim(r2Head.ETag, "\"")

		if s3ETag != r2ETag {
			printWarning("⚠ ETag mismatch: %s (S3: %s, R2: %s)", obj.Key, s3ETag, r2ETag)
			mismatchCount++
		}
	}

	bar.Finish()

	if mismatchCount == 0 {
		printSuccess("✅ All %d objects verified", len(objects))
	} else {
		printWarning("⚠ %d verification mismatches found", mismatchCount)
	}
}
```

- [ ] **Step 2: Run build**

Run: `go build -o /dev/null .`
Expected: Success

- [ ] **Step 3: Run all migration tests**

Run: `go test ./internal/migration/ -v -timeout 30s`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
ccs commit --direct -m "feat(migration): add ETag verification mode (ROAD-007 P-04)"
```

---

### Task 5: Documentation and Close-Out

**Files:**
- Modify: `docs/USAGE.md`

- [ ] **Step 1: Find and update the migrate section in USAGE.md**

Search for the migration section. If it doesn't exist, add it after the Dashboard section. Add/update:

```markdown
### Resume interrupted migration
```bash
cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --resume
```

Migration state is saved to `~/.cosmoflare/migrations/`. If interrupted (Ctrl+C, network failure), re-run with `--resume` to continue from where it left off. Completed objects are skipped automatically.

### Verify migration integrity
```bash
cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --verify
```

After transfer, compares S3 ETags with R2 ETags. Reports any mismatches.

### Concurrent transfers
```bash
cosmoflare migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --concurrency=20
```

Default concurrency: 10 parallel transfers. Objects over 100MB automatically use multipart upload.
```

- [ ] **Step 2: Add changelog entry**

Run: `ccs changelog add "Add real S3-to-R2 migration with resume support, concurrent transfers, and ETag verification (ROAD-007)" --type added`

- [ ] **Step 3: Update roadmap**

Run: `ccs roadmap update ROAD-007 --status completed`

- [ ] **Step 4: Commit**

```bash
ccs commit --direct -m "docs: update USAGE.md and close ROAD-007"
```
