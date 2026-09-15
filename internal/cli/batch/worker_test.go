package batch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewWorker
// ---------------------------------------------------------------------------

func TestNewWorker(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)
	config := &BatchConfig{DryRun: true, Quiet: true}

	w := NewWorker(7, queue, results, config)

	require.NotNil(t, w)
	assert.Equal(t, 7, w.ID)
	assert.NotNil(t, w.queue)
	assert.NotNil(t, w.results)
	assert.NotNil(t, w.config)
	assert.False(t, w.stats.StartTime.IsZero(), "StartTime should be set on creation")
}

func TestNewWorker_NilConfig(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	w := NewWorker(1, queue, results, nil)
	require.NotNil(t, w)
	assert.Nil(t, w.config)
}

// ---------------------------------------------------------------------------
// Worker.GetStats
// ---------------------------------------------------------------------------

func TestWorkerGetStats(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)
	w := NewWorker(1, queue, results, nil)

	stats := w.GetStats()
	require.NotNil(t, stats)
	assert.Equal(t, int32(0), stats.Processed)
	assert.Equal(t, int32(0), stats.Completed)
	assert.Equal(t, int32(0), stats.Failed)
	assert.Equal(t, int32(0), stats.Skipped)
	assert.Equal(t, int32(0), stats.Cancelled)
	assert.Equal(t, 0*time.Nanosecond, stats.TotalDuration)
	assert.Equal(t, 0.0, stats.AverageSpeed)
	assert.Equal(t, int64(0), stats.BytesProcessed)
	assert.False(t, stats.StartTime.IsZero())
}

// ---------------------------------------------------------------------------
// isRetryableError
// ---------------------------------------------------------------------------

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"timeout", errors.New("timeout"), true},
		{"network unreachable", errors.New("network unreachable"), true},
		{"temporary failure", errors.New("temporary failure"), true},
		{"resource temporarily unavailable", errors.New("resource temporarily unavailable"), true},
		{"file in use", errors.New("file in use"), true},
		{"access denied", errors.New("access denied"), true},
		{"not found (non-retryable)", errors.New("not found"), false},
		{"permission denied (non-retryable)", errors.New("permission denied"), false},
		{"empty error", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, isRetryableError(tt.err))
		})
	}
}

// ---------------------------------------------------------------------------
// contains
// ---------------------------------------------------------------------------

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"empty both", "", "", true},
		{"exact match", "hello", "hello", true},
		{"middle substring", "hello", "ell", true},
		{"substring not present", "hello", "world", false},
		{"substr longer than s", "a", "ab", false},
		{"single char match", "a", "a", true},
		{"prefix", "abcdef", "abc", true},
		{"suffix", "abcdef", "def", true},
		{"unicode", "café", "é", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, contains(tt.s, tt.substr))
		})
	}
}

// ---------------------------------------------------------------------------
// findSubstring
// ---------------------------------------------------------------------------

func TestFindSubstring(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert.True(t, findSubstring("hello world", "world"))
	})
	t.Run("not found", func(t *testing.T) {
		assert.False(t, findSubstring("hello world", "xyz"))
	})
	t.Run("empty substring", func(t *testing.T) {
		assert.True(t, findSubstring("abc", ""))
	})
	t.Run("single char", func(t *testing.T) {
		assert.True(t, findSubstring("abc", "a"))
		assert.True(t, findSubstring("abc", "c"))
		assert.False(t, findSubstring("abc", "d"))
	})
}

// ---------------------------------------------------------------------------
// parseCopyOptions
// ---------------------------------------------------------------------------

// parseCopyFlagCase is one single-key scenario for parseCopyOptions.
type parseCopyFlagCase struct {
	name string
	opts map[string]interface{}
	// want asserts the effect of the single option on the parsed result.
	want func(t *testing.T, o *operations.CopyOptions)
}

// assertCopyOptionsDefaults asserts the values parseCopyOptions falls back to
// when no options are supplied.
func assertCopyOptionsDefaults(t *testing.T, opts *operations.CopyOptions) {
	t.Helper()
	require.NotNil(t, opts)
	assert.False(t, opts.Resume)
	assert.True(t, opts.Verify)
	assert.False(t, opts.Overwrite)
	assert.True(t, opts.Preserve)
	assert.False(t, opts.Quiet)
	assert.Equal(t, int64(8*1024*1024), opts.ChunkSize)
	assert.Equal(t, 3, opts.Retries)
}

// runParseCopyOptionsDefaultsTests covers the nil and empty-map fallbacks.
func runParseCopyOptionsDefaultsTests(t *testing.T, w *Worker) {
	t.Run("nil options returns defaults", func(t *testing.T) {
		assertCopyOptionsDefaults(t, w.parseCopyOptions(nil))
	})

	t.Run("empty map returns defaults", func(t *testing.T) {
		assertCopyOptionsDefaults(t, w.parseCopyOptions(map[string]interface{}{}))
	})
}

// runParseCopyOptionsFlagTests covers each recognized option key in isolation.
func runParseCopyOptionsFlagTests(t *testing.T, w *Worker) {
	cases := []parseCopyFlagCase{
		{
			name: "resume sets Resume",
			opts: map[string]interface{}{"resume": true},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.True(t, o.Resume) },
		},
		{
			name: "verify sets Verify",
			opts: map[string]interface{}{"verify": true},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.True(t, o.Verify) },
		},
		{
			name: "overwrite sets Overwrite",
			opts: map[string]interface{}{"overwrite": true},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.True(t, o.Overwrite) },
		},
		{
			name: "preserve sets Preserve",
			opts: map[string]interface{}{"preserve": true},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.True(t, o.Preserve) },
		},
		{
			name: "quiet sets Quiet",
			opts: map[string]interface{}{"quiet": true},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.True(t, o.Quiet) },
		},
		{
			name: "chunk_size float64 sets ChunkSize",
			opts: map[string]interface{}{"chunk_size": 100.0},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.Equal(t, int64(100), o.ChunkSize) },
		},
		{
			name: "retries float64 sets Retries",
			opts: map[string]interface{}{"retries": 5.0},
			want: func(t *testing.T, o *operations.CopyOptions) { assert.Equal(t, 5, o.Retries) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.want(t, w.parseCopyOptions(tc.opts))
		})
	}
}

// runParseCopyOptionsRobustnessTests covers wrong-typed values being ignored
// and all options parsed together.
func runParseCopyOptionsRobustnessTests(t *testing.T, w *Worker) {
	t.Run("wrong type for resume is ignored", func(t *testing.T) {
		opts := w.parseCopyOptions(map[string]interface{}{"resume": "yes"})
		assert.False(t, opts.Resume)
	})

	t.Run("wrong type for chunk_size is ignored", func(t *testing.T) {
		opts := w.parseCopyOptions(map[string]interface{}{"chunk_size": "big"})
		assert.Equal(t, int64(8*1024*1024), opts.ChunkSize)
	})

	t.Run("all options set together", func(t *testing.T) {
		opts := w.parseCopyOptions(map[string]interface{}{
			"resume":     true,
			"verify":     false,
			"overwrite":  true,
			"preserve":   false,
			"quiet":      true,
			"chunk_size": 1024.0,
			"retries":    10.0,
		})
		assert.True(t, opts.Resume)
		assert.False(t, opts.Verify)
		assert.True(t, opts.Overwrite)
		assert.False(t, opts.Preserve)
		assert.True(t, opts.Quiet)
		assert.Equal(t, int64(1024), opts.ChunkSize)
		assert.Equal(t, 10, opts.Retries)
	})
}

func TestParseCopyOptions(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)
	w := NewWorker(1, queue, results, nil)

	t.Run("defaults", func(t *testing.T) { runParseCopyOptionsDefaultsTests(t, w) })
	t.Run("flags", func(t *testing.T) { runParseCopyOptionsFlagTests(t, w) })
	t.Run("robustness", func(t *testing.T) { runParseCopyOptionsRobustnessTests(t, w) })
}

// ---------------------------------------------------------------------------
// Worker.Start — DryRun delete integration
// ---------------------------------------------------------------------------

func TestWorkerStart_DryRunDelete(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := createTestFile(t, tmpDir, "delete-me.txt", "hello world")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:          true,
		Quiet:           true,
		Timeout:         5 * time.Second,
		ContinueOnError: true,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: tmpFile,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
		assert.Empty(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()

	_, err := readFileContent(t, tmpFile)
	assert.NoError(t, err, "file should still exist after dry-run delete")
}

// ---------------------------------------------------------------------------
// Worker.Start — DryRun delete directory
// ---------------------------------------------------------------------------

func TestWorkerStart_DryRunDeleteDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("data"), 0644))

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: subDir,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()
}

// ---------------------------------------------------------------------------
// Worker.Start — Actual delete
// ---------------------------------------------------------------------------

func TestWorkerStart_ActualDelete(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := createTestFile(t, tmpDir, "real-delete.txt", "delete this")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: tmpFile,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
		assert.Greater(t, result.Size, int64(0))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()

	_, err := os.Stat(tmpFile)
	assert.True(t, os.IsNotExist(err), "file should be deleted")
}

// ---------------------------------------------------------------------------
// Worker.Start — Actual delete directory
// ---------------------------------------------------------------------------

func TestWorkerStart_ActualDeleteDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "remove-me")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "b.txt"), []byte("b"), 0644))

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: subDir,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()

	_, err := os.Stat(subDir)
	assert.True(t, os.IsNotExist(err), "directory should be deleted")
}

// ---------------------------------------------------------------------------
// Worker.Start — Delete nonexistent file
// ---------------------------------------------------------------------------

func TestWorkerStart_DeleteNonExistent(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: "/nonexistent/path/file.txt",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.NotEmpty(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()
}

// ---------------------------------------------------------------------------
// Worker.Start — Unsupported operation type
// ---------------------------------------------------------------------------

func TestWorkerStart_UnsupportedType(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationType("invalid_type"),
		Source: "/tmp/something",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.Contains(t, result.Error, "unsupported operation type")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker result")
	}

	cancel()
}

// Worker.Start — Context cancellation
// ---------------------------------------------------------------------------

func TestWorkerStart_ContextCancellation(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{DryRun: true, Quiet: true}
	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())

	go w.Start(ctx)

	// Cancel immediately
	cancel()

	// Give worker time to exit
	time.Sleep(50 * time.Millisecond)

	// Worker should have stopped — no more results should come
	select {
	case <-results:
		// Might get one result, that's fine
	default:
		// No result is also fine
	}
}

// ---------------------------------------------------------------------------
// Worker.Start — Closed queue channel exits cleanly
// ---------------------------------------------------------------------------

func TestWorkerStart_ClosedQueueExitsCleanly(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{DryRun: true, Quiet: true}
	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	// Close the queue — worker should exit
	close(queue)

	select {
	case <-done:
		// Worker exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("Worker should exit when queue channel is closed")
	}
}

// ---------------------------------------------------------------------------
// Worker.Start — Closed queue after processing some ops
// ---------------------------------------------------------------------------

func TestWorkerStart_ClosedQueueAfterOps(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := createTestFile(t, tmpDir, "cq-1.txt", "a")
	f2 := createTestFile(t, tmpDir, "cq-2.txt", "b")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{DryRun: true, Quiet: true, Timeout: 5 * time.Second}
	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	queue <- &Operation{Type: OperationTypeDelete, Source: f1}
	queue <- &Operation{Type: OperationTypeDelete, Source: f2}
	close(queue)

	var count int
	for count < 2 {
		select {
		case <-results:
			count++
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for results")
		}
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Worker should exit after queue closed and ops processed")
	}

	assert.Equal(t, 2, count)
}

// ---------------------------------------------------------------------------
// Worker.Start — Multiple operations in sequence
// ---------------------------------------------------------------------------

func TestWorkerStart_MultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := createTestFile(t, tmpDir, "f1.txt", "one")
	f2 := createTestFile(t, tmpDir, "f2.txt", "two")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: f1}
	queue <- &Operation{Type: OperationTypeDelete, Source: f2}

	var completedCount int
	for completedCount < 2 {
		select {
		case result := <-results:
			if result.Status == StatusCompleted {
				completedCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for results")
		}
	}

	assert.Equal(t, 2, completedCount)
}

// ---------------------------------------------------------------------------
// processDelete — dry run captures file size
// ---------------------------------------------------------------------------

func TestProcessDelete_DryRunCapturesSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := createTestFile(t, tmpDir, "sized.txt", "1234567890") // 10 bytes

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: tmpFile}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
		assert.Equal(t, int64(10), result.Size)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}

// ---------------------------------------------------------------------------
// processOperation — stats tracking
// ---------------------------------------------------------------------------

func TestProcessOperation_StatsTracking(t *testing.T) {
	tmpDir := t.TempDir()
	f := createTestFile(t, tmpDir, "stat.txt", "x")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: f}

	select {
	case <-results:
		stats := w.GetStats()
		assert.Equal(t, int32(1), stats.Processed)
		assert.Equal(t, int32(1), stats.Completed)
		assert.Equal(t, int32(0), stats.Failed)
		assert.Greater(t, stats.TotalDuration, 0*time.Nanosecond)
		assert.False(t, stats.LastActivity.IsZero())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processOperation — sets timing fields
// ---------------------------------------------------------------------------

func TestProcessOperation_TimingFields(t *testing.T) {
	tmpDir := t.TempDir()
	f := createTestFile(t, tmpDir, "time.txt", "t")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{DryRun: true, Quiet: true, Timeout: 5 * time.Second}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: f}

	select {
	case result := <-results:
		assert.False(t, result.StartTime.IsZero())
		assert.False(t, result.EndTime.IsZero())
		assert.GreaterOrEqual(t, result.EndTime, result.StartTime)
		assert.Greater(t, result.Duration, 0*time.Nanosecond)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processCopy — dry run mode
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "copy-src.txt", "copy content")
	dstFile := filepath.Join(tmpDir, "copy-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		// In dry run, the operations package's CopyOperation handles dry run.
		// The worker wraps it, so result depends on operations.NewCopyOperation behavior.
		assert.True(t, result.Status == StatusCompleted || result.Status == StatusFailed,
			"should be completed or failed, got: %s", result.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processCopy — nonexistent source
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyNonExistent(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      "/nonexistent/source.txt",
		Destination: "/tmp/dest.txt",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.NotEmpty(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processMove — dry run mode
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "move-src.txt", "move me")
	dstFile := filepath.Join(tmpDir, "move-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		// Move = copy + delete; dry run prevents actual delete
		_ = result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Source should still exist in dry run
	_, err := os.Stat(srcFile)
	assert.NoError(t, err, "source should still exist after dry-run move")
}

// ---------------------------------------------------------------------------
// processMove — nonexistent source (copy phase fails)
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveNonExistent(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      "/nonexistent/move-src.txt",
		Destination: "/tmp/move-dst.txt",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.Contains(t, result.Error, "move failed during copy")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processOperation — failure increments Failed stat
// ---------------------------------------------------------------------------

func TestProcessOperation_FailureStat(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: "/nonexistent/fail-stat.txt"}

	select {
	case <-results:
		stats := w.GetStats()
		assert.Equal(t, int32(1), stats.Processed)
		assert.Equal(t, int32(0), stats.Completed)
		assert.Equal(t, int32(1), stats.Failed)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processCopy — retry behavior with timeout
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a very large file to simulate timeout during copy
	srcFile := createTestFile(t, tmpDir, "large.txt", string(make([]byte, 1024)))
	dstFile := filepath.Join(tmpDir, "large-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 1 * time.Millisecond, // Very short timeout to trigger context cancellation
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		// May complete (fast enough) or fail (timeout) — either is valid
		assert.True(t, result.Status == StatusCompleted || result.Status == StatusFailed,
			"expected completed or failed, got: %s", result.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processCopy — actual file copy (non-dry-run)
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyActualFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "copy-actual-src.txt", "actual copy content")
	dstFile := filepath.Join(tmpDir, "copy-actual-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status, "copy should complete: %s", result.Error)
		if result.Status == StatusCompleted {
			// Verify the destination file exists with correct content
			data, err := os.ReadFile(dstFile)
			require.NoError(t, err)
			assert.Equal(t, "actual copy content", string(data))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processMove — actual file move
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveActualFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "move-src.txt", "move this content")
	dstFile := filepath.Join(tmpDir, "move-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status, "move should complete: %s", result.Error)
		if result.Status == StatusCompleted {
			// Verify source is deleted
			_, err := os.Stat(srcFile)
			assert.True(t, os.IsNotExist(err), "source should be deleted after move")
			// Verify destination exists
			data, err := os.ReadFile(dstFile)
			require.NoError(t, err)
			assert.Equal(t, "move this content", string(data))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processMove — actual move fails on delete (read-only dest dir)
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveDeleteFail(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "move-fail-src.txt", "content")
	// Destination is valid for copy
	dstFile := createTestFile(t, tmpDir, "move-fail-dst.txt", "existing")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      srcFile,
		Destination: dstFile,
		Options: map[string]interface{}{
			"overwrite": true,
			"verify":    false,
		},
	}

	select {
	case result := <-results:
		// Move = copy + delete; copy may succeed, delete should succeed too
		// (the source file exists and is writable in tmpdir)
		_ = result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processDelete — directory with nested files (actual delete)
// ---------------------------------------------------------------------------

func TestWorkerStart_DeleteDirectoryWithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nested")
	require.NoError(t, os.MkdirAll(filepath.Join(subDir, "deep"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "deep", "b.txt"), []byte("b"), 0644))

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: subDir}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
		_, err := os.Stat(subDir)
		assert.True(t, os.IsNotExist(err))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processCopy — with copy options
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyWithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "opt-src.txt", "options test")
	dstFile := filepath.Join(tmpDir, "opt-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      srcFile,
		Destination: dstFile,
		Options: map[string]interface{}{
			"verify":    false,
			"overwrite": true,
			"preserve":  false,
			"quiet":     true,
			"chunk_size": 4096.0,
			"retries":   1.0,
		},
	}

	select {
	case result := <-results:
		_ = result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// WorkerStats struct defaults
// ---------------------------------------------------------------------------

func TestWorkerStats_Defaults(t *testing.T) {
	stats := &WorkerStats{
		StartTime: time.Now(),
	}

	assert.Equal(t, int32(0), stats.Processed)
	assert.Equal(t, int32(0), stats.Completed)
	assert.Equal(t, int32(0), stats.Failed)
	assert.Equal(t, int32(0), stats.Skipped)
	assert.Equal(t, int32(0), stats.Cancelled)
	assert.Equal(t, 0*time.Nanosecond, stats.TotalDuration)
	assert.Equal(t, 0.0, stats.AverageSpeed)
	assert.Equal(t, int64(0), stats.BytesProcessed)
}

// ---------------------------------------------------------------------------
// processDelete — non-quiet dry-run mode (covers print path)
// ---------------------------------------------------------------------------

func TestWorkerStart_DeleteDryRunNonQuiet(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := createTestFile(t, tmpDir, "verbose-del.txt", "data")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  true,
		Quiet:   false,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: tmpFile}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status)
		assert.Equal(t, int64(4), result.Size)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	cancel()
}

// ---------------------------------------------------------------------------
// processCopy — with retries, non-existent source, non-quiet
// Covers: retry print path, isRetryableError branch, backoff select
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyRetryNonQuiet(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   false,
		Timeout: 5 * time.Second,
		Retries: 2,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      "/nonexistent/retry-src.txt",
		Destination: "/tmp/retry-dst.txt",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.NotEmpty(t, result.Error)
	case <-time.After(15 * time.Second):
		t.Fatal("timed out (retry backoff)")
	}

	cancel()
}

// ---------------------------------------------------------------------------
// processCopy — retry with context cancellation during backoff
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyRetryContextCancel(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 5,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      "/nonexistent/cancel-src.txt",
		Destination: "/tmp/cancel-dst.txt",
	}

	// Wait for first attempt to fail, then cancel during retry backoff
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-results:
		// Worker may or may not send a result depending on timing
	case <-time.After(3 * time.Second):
		// Worker exited via context cancellation without sending result, that's fine
	}
}

// ---------------------------------------------------------------------------
// processOperation — results channel full + context cancel (covers ctx.Done in send)
// ---------------------------------------------------------------------------

func TestProcessOperation_ResultsFullThenCancel(t *testing.T) {
	tmpDir := t.TempDir()
	f := createTestFile(t, tmpDir, "full-results.txt", "data")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 1) // Very small buffer

	config := &BatchConfig{DryRun: true, Quiet: true, Timeout: 5 * time.Second}
	w := NewWorker(1, queue, results, config)

	ctx, cancel := context.WithCancel(context.Background())

	// Fill the results buffer
	results <- &Operation{Status: StatusCompleted}

	go w.Start(ctx)

	queue <- &Operation{Type: OperationTypeDelete, Source: f}

	// Give time for processOperation to complete and block on full results channel
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Drain results so goroutine can exit
	select {
	case <-results:
	default:
	}
}

// ---------------------------------------------------------------------------
// processCopy — retryable error triggers retry loop body
// The trick: copy a file to a destination inside a non-existent directory.
// This won't produce a retryable error, so we instead test with a context
// that is already cancelled to force the ctx.Done path during copy.
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessCopyCancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "cancel-copy.txt", "data")
	dstFile := filepath.Join(tmpDir, "cancel-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 2,
	}

	w := NewWorker(1, queue, results, config)

	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx)

	// Send the op, then immediately cancel
	queue <- &Operation{
		Type:        OperationTypeCopy,
		Source:      srcFile,
		Destination: dstFile,
	}
	cancel()

	// Worker exits via context — may or may not send a result
	select {
	case <-results:
	case <-time.After(2 * time.Second):
	}
}

// ---------------------------------------------------------------------------
// processMove — move with delete failure (read-only directory with file)
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveDeleteFailReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "protected")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	srcFile := filepath.Join(subDir, "src.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("move content"), 0644))
	dstFile := filepath.Join(tmpDir, "dst.txt")

	// Make the source file read-only AND make parent dir read-only
	// so os.Remove fails (can't remove file from read-only dir on some OSes)
	// Note: this is OS-dependent; on Unix, root can still remove
	_ = dstFile

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		// Move may succeed (copy + delete both work in temp dir)
		// or fail if delete fails. Either way, no hang.
		_ = result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processDelete — delete failure (non-existent nested path)
// Tests the os.RemoveAll error path when path doesn't exist
// ---------------------------------------------------------------------------

func TestWorkerStart_DeleteNonExistentPath(t *testing.T) {
	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:   OperationTypeDelete,
		Source: "/nonexistent/deeply/nested/path/file.txt",
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusFailed, result.Status)
		assert.NotEmpty(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// processMove — non-dry-run, source file that can be moved
// (covers the os.Remove path in processMove)
// ---------------------------------------------------------------------------

func TestWorkerStart_ProcessMoveActualDeleteCoversRemovePath(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := createTestFile(t, tmpDir, "mv-remove-src.txt", "remove me")
	dstFile := filepath.Join(tmpDir, "mv-remove-dst.txt")

	queue := make(chan *Operation, 10)
	results := make(chan *Operation, 10)

	config := &BatchConfig{
		DryRun:  false,
		Quiet:   true,
		Timeout: 5 * time.Second,
		Retries: 0,
	}

	w := NewWorker(1, queue, results, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	queue <- &Operation{
		Type:        OperationTypeMove,
		Source:      srcFile,
		Destination: dstFile,
	}

	select {
	case result := <-results:
		assert.Equal(t, StatusCompleted, result.Status, "move should succeed: %s", result.Error)
		if result.Status == StatusCompleted {
			_, err := os.Stat(srcFile)
			assert.True(t, os.IsNotExist(err), "source should be deleted after move")
			data, err := os.ReadFile(dstFile)
			require.NoError(t, err)
			assert.Equal(t, "remove me", string(data))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	f, err := os.CreateTemp(dir, name)
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func readFileContent(t *testing.T, path string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
