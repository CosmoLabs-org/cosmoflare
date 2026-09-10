package operations

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyTestWriteFile creates a file at path with the given content and
// permissions, failing the test if it cannot be created.
func copyTestWriteFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

// ---------------------------------------------------------------------------
// DefaultCopyOptions
// ---------------------------------------------------------------------------

// TestDefaultCopyOptions verifies that DefaultCopyOptions returns a fully
// populated options struct with the documented default values (safe defaults:
// verification and attribute preservation on, overwrite off, 8MB chunks,
// 4-way concurrency, 30 minute timeout, 3 retries).
func TestDefaultCopyOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultCopyOptions()
	require.NotNil(t, opts)

	assert.False(t, opts.Resume, "resume should default to false")
	assert.True(t, opts.Verify, "verify should default to true")
	assert.False(t, opts.Overwrite, "overwrite should default to false")
	assert.True(t, opts.Preserve, "preserve should default to true")
	assert.True(t, opts.ProgressBar, "progress bar should default to true")
	assert.False(t, opts.Quiet, "quiet should default to false")
	assert.Equal(t, int64(8*1024*1024), opts.ChunkSize, "chunk size should default to 8MB")
	assert.Equal(t, 4, opts.Concurrency, "concurrency should default to 4")
	assert.Equal(t, 30*time.Minute, opts.Timeout, "timeout should default to 30 minutes")
	assert.Equal(t, 3, opts.Retries, "retries should default to 3")
	assert.False(t, opts.DryRun, "dry run should default to false")
}

// ---------------------------------------------------------------------------
// NewCopyOperation
// ---------------------------------------------------------------------------

// TestNewCopyOperation verifies that NewCopyOperation wires the source and
// destination paths, applies default options when none are supplied, keeps a
// caller-provided options pointer, and returns a cancellable context.
func TestNewCopyOperation(t *testing.T) {
	t.Parallel()

	t.Run("nil options fall back to defaults", func(t *testing.T) {
		t.Parallel()

		op := NewCopyOperation("/src/file.txt", "/dst/file.txt", nil)
		require.NotNil(t, op)
		assert.Equal(t, "/src/file.txt", op.Source)
		assert.Equal(t, "/dst/file.txt", op.Destination)
		require.NotNil(t, op.Options)
		assert.Equal(t, DefaultCopyOptions(), op.Options)
	})

	t.Run("custom options are preserved by pointer", func(t *testing.T) {
		t.Parallel()

		custom := &CopyOptions{
			Verify:    false,
			Quiet:     true,
			Retries:   7,
			ChunkSize: 1024,
		}
		op := NewCopyOperation("a", "b", custom)
		require.NotNil(t, op)
		assert.Same(t, custom, op.Options, "options pointer must be reused, not copied")
		assert.Equal(t, 7, op.Options.Retries)
		assert.False(t, op.Options.Verify)
	})

	t.Run("returns cancellable context", func(t *testing.T) {
		t.Parallel()

		op := NewCopyOperation("a", "b", nil)
		require.NotNil(t, op.Context)
		require.NotNil(t, op.CancelFunc)
		assert.NoError(t, op.Context.Err(), "fresh context must not be done")

		op.Cancel()
		assert.ErrorIs(t, op.Context.Err(), context.Canceled)
	})

	t.Run("empty paths do not panic", func(t *testing.T) {
		t.Parallel()

		op := NewCopyOperation("", "", &CopyOptions{Quiet: true})
		require.NotNil(t, op)
		assert.Empty(t, op.Source)
		assert.Empty(t, op.Destination)
	})
}

// TestNewCopyOperationExecuteErrors verifies the error paths reachable
// through an operation created by NewCopyOperation: a missing source file
// fails the stat, and an existing destination with overwrite disabled is
// rejected.
func TestNewCopyOperationExecuteErrors(t *testing.T) {
	t.Parallel()

	newOpts := func() *CopyOptions {
		return &CopyOptions{
			Verify:      false,
			Preserve:    false,
			ProgressBar: false,
			Quiet:       true,
			Retries:     0,
		}
	}

	t.Run("missing source file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		op := NewCopyOperation(
			filepath.Join(dir, "does-not-exist.bin"),
			filepath.Join(dir, "out.bin"),
			newOpts(),
		)

		result, err := op.Execute()
		require.Error(t, err)
		require.NotNil(t, result)
		assert.Contains(t, err.Error(), "failed to stat source file")
		assert.False(t, result.Success)
		assert.Equal(t, result.Error, err, "result must carry the same error")
		assert.Zero(t, result.Size)
	})

	t.Run("destination exists and overwrite disabled", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		copyTestWriteFile(t, src, "source content")
		copyTestWriteFile(t, dst, "existing content")

		op := NewCopyOperation(src, dst, newOpts())
		result, err := op.Execute()
		require.Error(t, err)
		require.NotNil(t, result)
		assert.Contains(t, err.Error(), "destination file exists")
		assert.False(t, result.Success)

		// The pre-existing destination must be left untouched.
		got, readErr := os.ReadFile(dst)
		require.NoError(t, readErr)
		assert.Equal(t, "existing content", string(got))
	})
}

// TestNewCopyOperationExecuteSuccess verifies the happy path reachable
// through NewCopyOperation: a normal copy, and a dry run that must not touch
// the destination.
func TestNewCopyOperationExecuteSuccess(t *testing.T) {
	t.Parallel()

	t.Run("copies file content and verifies", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "sub", "dst.txt")
		copyTestWriteFile(t, src, "hello, copy world")

		op := NewCopyOperation(src, dst, &CopyOptions{
			Verify:      true,
			Preserve:    false,
			ProgressBar: false,
			Quiet:       true,
			Retries:     0,
			ChunkSize:   4, // force multiple buffered chunks
		})

		result, err := op.Execute()
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		assert.True(t, result.Verified)
		assert.Equal(t, int64(len("hello, copy world")), result.Size)
		assert.False(t, result.Resumed)

		got, readErr := os.ReadFile(dst)
		require.NoError(t, readErr)
		assert.Equal(t, "hello, copy world", string(got))
	})

	t.Run("dry run does not create destination", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		copyTestWriteFile(t, src, "data")

		op := NewCopyOperation(src, dst, &CopyOptions{
			DryRun:      true,
			ProgressBar: false,
			Quiet:       true,
			Retries:     0,
		})

		result, err := op.Execute()
		require.NoError(t, err)
		assert.True(t, result.Success)

		_, statErr := os.Stat(dst)
		assert.True(t, os.IsNotExist(statErr), "dry run must not create the destination")
	})
}

// ---------------------------------------------------------------------------
// NewBatchCopy
// ---------------------------------------------------------------------------

// TestNewBatchCopy verifies that NewBatchCopy stores the operations slice,
// falls back to default options when none are supplied, keeps a caller-owned
// options pointer, and runs on a background context.
func TestNewBatchCopy(t *testing.T) {
	t.Parallel()

	t.Run("nil options fall back to defaults", func(t *testing.T) {
		t.Parallel()

		bc := NewBatchCopy(nil, nil)
		require.NotNil(t, bc)
		assert.Nil(t, bc.Operations)
		require.NotNil(t, bc.Options)
		assert.Equal(t, DefaultCopyOptions(), bc.Options)
	})

	t.Run("custom options are preserved by pointer", func(t *testing.T) {
		t.Parallel()

		custom := &CopyOptions{Concurrency: 12, Quiet: true}
		bc := NewBatchCopy(nil, custom)
		assert.Same(t, custom, bc.Options)
		assert.Equal(t, 12, bc.Options.Concurrency)
	})

	t.Run("operations slice is preserved", func(t *testing.T) {
		t.Parallel()

		ops := []*CopyOperation{
			NewCopyOperation("a", "b", nil),
			NewCopyOperation("c", "d", nil),
		}
		bc := NewBatchCopy(ops, nil)
		assert.Len(t, bc.Operations, 2)
		assert.Equal(t, "a", bc.Operations[0].Source)
		assert.Equal(t, "d", bc.Operations[1].Destination)
	})

	t.Run("context is background", func(t *testing.T) {
		t.Parallel()

		bc := NewBatchCopy(nil, nil)
		require.NotNil(t, bc.Context)
		assert.Equal(t, context.Background(), bc.Context)
		assert.NoError(t, bc.Context.Err())
	})
}

// TestNewBatchCopyExecute verifies the Execute path of a batch created by
// NewBatchCopy: results keep per-operation order, a failing operation is
// reported through its result without failing the whole batch, and empty
// batches return no results.
func TestNewBatchCopyExecute(t *testing.T) {
	t.Parallel()

	// Derive from DefaultCopyOptions: a hand-rolled CopyOptions with a zero
	// Concurrency makes BatchCopy's semaphore unbuffered and Execute
	// deadlocks (workers block on acquire forever).
	batchOpts := DefaultCopyOptions()
	batchOpts.Verify = false
	batchOpts.Preserve = false
	batchOpts.ProgressBar = false
	batchOpts.Quiet = true
	batchOpts.Retries = 0

	t.Run("mixed success and failure keeps order", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		goodSrc := filepath.Join(dir, "good.txt")
		copyTestWriteFile(t, goodSrc, "good data")

		ops := []*CopyOperation{
			NewCopyOperation(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "out1.txt"), batchOpts),
			NewCopyOperation(goodSrc, filepath.Join(dir, "out2.txt"), batchOpts),
		}
		bc := NewBatchCopy(ops, batchOpts)

		results, err := bc.Execute()
		require.NoError(t, err, "batch Execute only fails on context cancellation")
		require.Len(t, results, 2)

		require.NotNil(t, results[0])
		assert.False(t, results[0].Success, "missing source must fail")
		assert.Error(t, results[0].Error)

		require.NotNil(t, results[1])
		assert.True(t, results[1].Success, "valid source must succeed")
		assert.NoError(t, results[1].Error)

		got, readErr := os.ReadFile(filepath.Join(dir, "out2.txt"))
		require.NoError(t, readErr)
		assert.Equal(t, "good data", string(got))
	})

	t.Run("empty batch returns no results", func(t *testing.T) {
		t.Parallel()

		bc := NewBatchCopy([]*CopyOperation{}, batchOpts)
		results, err := bc.Execute()
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// ---------------------------------------------------------------------------
// isRetryableError
// ---------------------------------------------------------------------------

// TestIsRetryableError verifies that isRetryableError recognises every
// documented transient failure keyword (including when embedded in a larger
// message), rejects nil and non-transient errors, and is case sensitive
// because it relies on exact substring matching.
func TestIsRetryableError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error is not retryable", nil, false},
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"timeout", errors.New("i/o timeout while reading"), true},
		{"network", errors.New("network is unreachable"), true},
		{"temporary failure", errors.New("temporary failure in name resolution"), true},
		{"resource temporarily unavailable", errors.New("read: resource temporarily unavailable"), true},
		{"keyword embedded in wrapped message", errors.New("copy failed: network error occurred"), true},
		{"plain non-retryable error", errors.New("permission denied"), false},
		{"permission style message", errors.New("failed to stat source file: no such file or directory"), false},
		{"empty message error", errors.New(""), false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isRetryableError(tc.err))
		})
	}
}

// ---------------------------------------------------------------------------
// contains
// ---------------------------------------------------------------------------

// TestContains verifies the custom contains helper: exact matches, prefix,
// suffix and interior occurrences all return true, no-match and
// longer-than-haystack cases return false, and empty substrings match
// anything. Matching is byte-wise exact, i.e. case sensitive.
func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact equal strings", "network", "network", true},
		{"substr is prefix", "connection refused by host", "connection refused", true},
		{"substr is suffix", "read timeout", "timeout", true},
		{"substr in middle", "failed: network unreachable", "network", true},
		{"single character match", "a", "a", true},
		{"empty substr matches non-empty string", "anything", "", true},
		{"empty substr matches empty string", "", "", true},
		{"no match", "connection refused", "denied", false},
		{"substr longer than string", "net", "network error", false},
		{"different lengths no match", "abcd", "abce", false},
		{"case sensitive mismatch", "Network Error", "network", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, contains(tc.s, tc.substr))
		})
	}
}

// ---------------------------------------------------------------------------
// findSubstring
// ---------------------------------------------------------------------------

// TestFindSubstring verifies the naive substring search helper: matches at
// the start, middle and end of the haystack, empty-needle behaviour,
// no-match cases and inputs where the needle is longer than the haystack.
func TestFindSubstring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"match at start", "hello world", "hello", true},
		{"match in middle", "hello world", "lo w", true},
		{"match at end", "hello world", "world", true},
		{"full string equals needle", "timeout", "timeout", true},
		{"overlapping needle", "aaa", "aa", true},
		{"empty needle matches any string", "abc", "", true},
		{"empty needle matches empty string", "", "", true},
		{"empty haystack with non-empty needle", "", "a", false},
		{"needle longer than haystack", "ab", "abc", false},
		{"no occurrence", "connection refused", "denied", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, findSubstring(tc.s, tc.substr))
		})
	}
}
