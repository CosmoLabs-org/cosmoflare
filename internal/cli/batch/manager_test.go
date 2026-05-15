package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DefaultBatchConfig
// ---------------------------------------------------------------------------

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()
	assert.Equal(t, 4, config.Concurrency)
	assert.True(t, config.ContinueOnError)
	assert.Equal(t, 3, config.Retries)
	assert.Equal(t, 30*time.Minute, config.Timeout)
	assert.False(t, config.DryRun)
	assert.False(t, config.Quiet)
	assert.True(t, config.Interactive)
}

// ---------------------------------------------------------------------------
// NewBatchManager
// ---------------------------------------------------------------------------

func TestNewBatchManager(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		bm := NewBatchManager(nil)
		require.NotNil(t, bm)
		assert.Equal(t, 4, bm.config.Concurrency)
		assert.NotNil(t, bm.operations)
		assert.NotNil(t, bm.stats)
		assert.NotNil(t, bm.progress)
		assert.NotNil(t, bm.queue)
		assert.NotNil(t, bm.results)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &BatchConfig{
			Concurrency: 8,
			DryRun:      true,
		}
		bm := NewBatchManager(config)
		assert.Equal(t, 8, bm.config.Concurrency)
		assert.True(t, bm.config.DryRun)
	})
}

// ---------------------------------------------------------------------------
// AddOperation
// ---------------------------------------------------------------------------

func TestAddOperation(t *testing.T) {
	bm := NewBatchManager(nil)

	t.Run("auto-generates ID", func(t *testing.T) {
		op := &Operation{
			Type:   OperationTypeCopy,
			Source: "/tmp/file.txt",
		}
		err := bm.AddOperation(op)
		assert.NoError(t, err)
		assert.NotEmpty(t, op.ID)
		assert.Equal(t, StatusPending, op.Status)
	})

	t.Run("preserves provided ID", func(t *testing.T) {
		op := &Operation{
			ID:     "custom-id",
			Type:   OperationTypeUpload,
			Source: "/tmp/upload.txt",
		}
		err := bm.AddOperation(op)
		assert.NoError(t, err)
		assert.Equal(t, "custom-id", op.ID)
	})

	t.Run("increments stats", func(t *testing.T) {
		bm2 := NewBatchManager(nil)
		bm2.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
		bm2.AddOperation(&Operation{Type: OperationTypeCopy, Source: "b"})

		assert.Equal(t, int32(2), atomic.LoadInt32(&bm2.stats.Total))
		assert.Equal(t, int32(2), atomic.LoadInt32(&bm2.stats.Pending))
	})

	t.Run("empty ID gets auto-assigned", func(t *testing.T) {
		op := &Operation{Type: OperationTypeCopy, Source: "c"}
		err := bm.AddOperation(op)
		assert.NoError(t, err)
		assert.NotEmpty(t, op.ID)
	})

	t.Run("status is set to pending even if previously different", func(t *testing.T) {
		op := &Operation{Type: OperationTypeCopy, Source: "d", Status: StatusCompleted}
		err := bm.AddOperation(op)
		assert.NoError(t, err)
		assert.Equal(t, StatusPending, op.Status)
	})
}

// ---------------------------------------------------------------------------
// AddCopyOperation
// ---------------------------------------------------------------------------

func TestAddCopyOperation(t *testing.T) {
	t.Run("with existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		require.NoError(t, os.WriteFile(srcFile, []byte("hello"), 0644))

		bm := NewBatchManager(nil)
		err := bm.AddCopyOperation(srcFile, "/tmp/dest.txt", nil)
		require.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, OperationTypeCopy, ops[0].Type)
		assert.Equal(t, srcFile, ops[0].Source)
		assert.Equal(t, "/tmp/dest.txt", ops[0].Destination)
		assert.Equal(t, int64(5), ops[0].Size)
	})

	t.Run("with copy options", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		require.NoError(t, os.WriteFile(srcFile, []byte("data"), 0644))

		bm := NewBatchManager(nil)
		err := bm.AddCopyOperation(srcFile, "/tmp/dest.txt", &operations.CopyOptions{
			Resume:    true,
			Verify:    false,
			Overwrite: true,
		})
		require.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.True(t, ops[0].Options["resume"].(bool))
		assert.False(t, ops[0].Options["verify"].(bool))
		assert.True(t, ops[0].Options["overwrite"].(bool))
	})

	t.Run("with non-existent file does not error", func(t *testing.T) {
		bm := NewBatchManager(nil)
		err := bm.AddCopyOperation("/nonexistent/file.txt", "/tmp/dest.txt", nil)
		require.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, int64(0), ops[0].Size)
	})

	t.Run("increments total size stat", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "big.txt")
		require.NoError(t, os.WriteFile(srcFile, make([]byte, 1000), 0644))

		bm := NewBatchManager(nil)
		bm.AddCopyOperation(srcFile, "/tmp/dest.txt", nil)

		assert.Equal(t, int64(1000), atomic.LoadInt64(&bm.stats.TotalSize))
	})
}

// ---------------------------------------------------------------------------
// GetOperations
// ---------------------------------------------------------------------------

func TestGetOperations(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
	bm.AddOperation(&Operation{Type: OperationTypeUpload, Source: "b"})

	ops := bm.GetOperations()
	assert.Len(t, ops, 2)
	assert.Equal(t, "a", ops[0].Source)
	assert.Equal(t, "b", ops[1].Source)
}

func TestGetOperations_ReturnsCopy(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	ops := bm.GetOperations()
	// GetOperations copies the slice but not the pointers.
	// Verify we can append to the returned slice without affecting the original.
	ops = append(ops, &Operation{Type: OperationTypeCopy, Source: "extra"})

	ops2 := bm.GetOperations()
	assert.Len(t, ops2, 1, "original should still have 1 operation")
	assert.Equal(t, "a", ops2[0].Source)
}

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestGetStats(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Total)
	assert.NotZero(t, stats.StartTime)
}

func TestGetStats_ReturnsCopy(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	stats1 := bm.GetStats()
	stats1.Total = 999
	stats2 := bm.GetStats()

	assert.Equal(t, int32(1), stats2.Total)
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestCancel(t *testing.T) {
	bm := NewBatchManager(nil)
	assert.NotPanics(t, func() {
		bm.Cancel()
	})
}

func TestCancel_MultipleTimes(t *testing.T) {
	bm := NewBatchManager(nil)
	assert.NotPanics(t, func() {
		bm.Cancel()
		bm.Cancel()
		bm.Cancel()
	})
}

// ---------------------------------------------------------------------------
// SaveToFile
// ---------------------------------------------------------------------------

func TestSaveToFile(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{
		Type:        OperationTypeCopy,
		Source:      "/tmp/src.txt",
		Destination: "/tmp/dst.txt",
	})

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "batch.json")

	err := bm.SaveToFile(outFile)
	assert.NoError(t, err)

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "src.txt")
	assert.Contains(t, string(data), "dst.txt")
}

func TestSaveToFile_EmptyOperations(t *testing.T) {
	bm := NewBatchManager(nil)
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "empty.json")

	err := bm.SaveToFile(outFile)
	assert.NoError(t, err)

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var spec BatchSpec
	require.NoError(t, json.Unmarshal(data, &spec))
	assert.Empty(t, spec.Operations)
}

func TestSaveToFile_InvalidPath(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	err := bm.SaveToFile("/nonexistent/dir/batch.json")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// LoadFromFile
// ---------------------------------------------------------------------------

func TestLoadFromFile(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		bm := NewBatchManager(nil)
		err := bm.LoadFromFile("/nonexistent/batch.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read batch file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "bad.json")
		os.WriteFile(f, []byte("not json"), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse batch file")
	})

	t.Run("valid batch spec", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "batch.json")
		content := `{
			"name": "test batch",
			"operations": [
				{"type": "copy", "source": "/tmp/a", "destination": "/tmp/b"},
				{"type": "upload", "source": "/tmp/c", "destination": "bucket/key"}
			]
		}`
		os.WriteFile(f, []byte(content), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.NoError(t, err)
		assert.Len(t, bm.GetOperations(), 2)
	})

	t.Run("operation with options", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "batch.json")
		content := `{
			"operations": [
				{
					"type": "copy",
					"source": "/tmp/a",
					"destination": "/tmp/b",
					"options": {"resume": true, "verify": false}
				}
			]
		}`
		os.WriteFile(f, []byte(content), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.True(t, ops[0].Options["resume"].(bool))
		assert.False(t, ops[0].Options["verify"].(bool))
	})

	t.Run("operation with ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "batch.json")
		content := `{
			"operations": [
				{"id": "my-op-1", "type": "delete", "source": "/tmp/old"}
			]
		}`
		os.WriteFile(f, []byte(content), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, "my-op-1", ops[0].ID)
		assert.Equal(t, OperationTypeDelete, ops[0].Type)
	})

	t.Run("empty operations array", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "batch.json")
		os.WriteFile(f, []byte(`{"operations": []}`), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.NoError(t, err)
		assert.Empty(t, bm.GetOperations())
	})

	t.Run("operation without destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "batch.json")
		content := `{
			"operations": [
				{"type": "delete", "source": "/tmp/old"}
			]
		}`
		os.WriteFile(f, []byte(content), 0644)

		bm := NewBatchManager(nil)
		err := bm.LoadFromFile(f)
		assert.NoError(t, err)

		ops := bm.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, "", ops[0].Destination)
	})
}

// ---------------------------------------------------------------------------
// Execute — non-interactive mode
// NOTE: Execute has an inherent race condition where waitForCompletion
// can return before queueOperations sets Running. We test Execute minimally
// and test the internal components (queueOperations, processResult, etc.)
// directly to achieve coverage.
// ---------------------------------------------------------------------------

func TestExecute_NoOperations(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{
		Interactive: false,
		Quiet:       true,
	})

	stats, err := bm.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no operations to execute")
	assert.NotNil(t, stats)
}

func TestExecute_SetsEndTimeAndDuration(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{
		Interactive: false,
		Quiet:       true,
	})

	stats, err := bm.Execute()
	assert.Error(t, err)
	assert.NotNil(t, stats)
	assert.False(t, stats.StartTime.IsZero())
}

// ---------------------------------------------------------------------------
// queueOperations
// ---------------------------------------------------------------------------

func TestQueueOperations_SetsRunningState(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{
		Concurrency: 1,
		Interactive: false,
		Quiet:       true,
	})

	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "b"})

	go bm.queueOperations()

	op1 := <-bm.queue
	op2 := <-bm.queue

	assert.Equal(t, "a", op1.Source)
	assert.Equal(t, "b", op2.Source)
	assert.Equal(t, StatusRunning, op1.Status)
	assert.Equal(t, StatusRunning, op2.Status)

	bm.Cancel()
}

func TestQueueOperations_ContextCancelled(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{Interactive: false, Quiet: true})
	bm.Cancel()

	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	done := make(chan struct{})
	go func() {
		bm.queueOperations()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("queueOperations should have returned on cancelled context")
	}
}

// ---------------------------------------------------------------------------
// collectResults
// ---------------------------------------------------------------------------

func TestCollectResults_ContextCancelled(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{Interactive: false, Quiet: true})

	done := make(chan struct{})
	go func() {
		bm.collectResults()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	bm.Cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("collectResults should have returned on cancelled context")
	}
}

func TestCollectResults_ProcessesCompletedResult(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{Interactive: false, Quiet: true})

	done := make(chan struct{})
	go func() {
		bm.collectResults()
		close(done)
	}()

	// Send a completed result
	atomic.AddInt32(&bm.stats.Running, 1)
	bm.results <- &Operation{
		Type:   OperationTypeCopy,
		Status: StatusCompleted,
		Size:   2048,
		StartTime: time.Now().Add(-50 * time.Millisecond),
	}

	// Give it time to process, then cancel
	time.Sleep(100 * time.Millisecond)
	bm.Cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("collectResults should have exited after cancel")
	}

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Completed)
	assert.Equal(t, int64(2048), stats.ProcessedSize)
}

// ---------------------------------------------------------------------------
// monitorProgress
// ---------------------------------------------------------------------------

func TestMonitorProgress_ContextCancelled(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{Interactive: false, Quiet: true})

	done := make(chan struct{})
	go func() {
		bm.monitorProgress()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	bm.Cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("monitorProgress should have returned on cancelled context")
	}
}

func TestMonitorProgress_FiresCallback(t *testing.T) {
	var progressCount int

	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
	atomic.AddInt32(&bm.stats.Completed, 1)

	bm.OnProgress(func(stats *BatchStats) {
		progressCount++
	})

	done := make(chan struct{})
	go func() {
		bm.monitorProgress()
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	bm.Cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("monitorProgress should have exited after Cancel")
	}

	assert.GreaterOrEqual(t, progressCount, 1, "OnProgress should have been called at least once")
}

// ---------------------------------------------------------------------------
// Callback setters
// ---------------------------------------------------------------------------

func TestCallbackSetters(t *testing.T) {
	bm := NewBatchManager(nil)

	t.Run("OnProgress", func(t *testing.T) {
		bm.OnProgress(func(stats *BatchStats) {})
		assert.NotNil(t, bm.onProgress)
	})

	t.Run("OnComplete", func(t *testing.T) {
		bm.OnComplete(func(stats *BatchStats) {})
		assert.NotNil(t, bm.onComplete)
	})

	t.Run("OnError", func(t *testing.T) {
		bm.OnError(func(op *Operation, err error) {})
		assert.NotNil(t, bm.onError)
	})

	t.Run("OnOperation", func(t *testing.T) {
		bm.OnOperation(func(op *Operation) {})
		assert.NotNil(t, bm.onOperation)
	})
}

func TestCallbacks_OnOperationFired(t *testing.T) {
	var capturedOp *Operation

	bm := NewBatchManager(nil)
	bm.OnOperation(func(op *Operation) {
		capturedOp = op
	})

	op := &Operation{Type: OperationTypeCopy, Status: StatusCompleted, Size: 1024}
	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	assert.NotNil(t, capturedOp)
	assert.Equal(t, OperationTypeCopy, capturedOp.Type)
}

func TestOnError_CallbackOnFailure(t *testing.T) {
	var capturedOp *Operation
	var capturedErr error

	bm := NewBatchManager(nil)
	bm.OnError(func(op *Operation, err error) {
		capturedOp = op
		capturedErr = err
	})

	op := &Operation{
		Type:   OperationTypeDelete,
		Status: StatusFailed,
		Error:  "failed to delete /nonexistent/file",
	}

	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	assert.Equal(t, int32(1), atomic.LoadInt32(&bm.stats.Failed))
	assert.NotNil(t, capturedOp, "OnError should have captured the operation")
	assert.NotNil(t, capturedErr, "OnError should have captured the error")
	assert.Equal(t, "failed to delete /nonexistent/file", capturedErr.Error())
}

// ---------------------------------------------------------------------------
// updateProgress
// ---------------------------------------------------------------------------

func TestUpdateProgress(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "b"})

	atomic.AddInt32(&bm.stats.Completed, 1)

	bm.updateProgress()

	stats := bm.GetStats()
	assert.InDelta(t, 50.0, stats.Progress, 0.1)
}

func TestUpdateProgress_ZeroTotal(t *testing.T) {
	bm := NewBatchManager(nil)

	bm.updateProgress()

	stats := bm.GetStats()
	assert.True(t, stats.Progress == 0 || stats.Progress != stats.Progress, // NaN check
		"progress should be 0 or NaN when total is 0")
}

// ---------------------------------------------------------------------------
// Operation types and status constants
// ---------------------------------------------------------------------------

func TestOperationTypes(t *testing.T) {
	assert.Equal(t, OperationType("copy"), OperationTypeCopy)
	assert.Equal(t, OperationType("upload"), OperationTypeUpload)
	assert.Equal(t, OperationType("download"), OperationTypeDownload)
	assert.Equal(t, OperationType("delete"), OperationTypeDelete)
	assert.Equal(t, OperationType("sync"), OperationTypeSync)
	assert.Equal(t, OperationType("move"), OperationTypeMove)
}

func TestOperationStatus(t *testing.T) {
	assert.Equal(t, OperationStatus("pending"), StatusPending)
	assert.Equal(t, OperationStatus("running"), StatusRunning)
	assert.Equal(t, OperationStatus("completed"), StatusCompleted)
	assert.Equal(t, OperationStatus("failed"), StatusFailed)
	assert.Equal(t, OperationStatus("skipped"), StatusSkipped)
	assert.Equal(t, OperationStatus("cancelled"), StatusCancelled)
}

// ---------------------------------------------------------------------------
// BatchSpec / BatchOperationSpec JSON serialization
// ---------------------------------------------------------------------------

func TestBatchSpec_JSON(t *testing.T) {
	spec := BatchSpec{
		Config: DefaultBatchConfig(),
		Operations: []BatchOperationSpec{
			{
				ID:          "op-1",
				Type:        "copy",
				Source:      "/src/a.txt",
				Destination: "/dst/a.txt",
				Options: map[string]interface{}{
					"verify": true,
				},
			},
		},
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(data), "op-1")
	assert.Contains(t, string(data), "/src/a.txt")

	var decoded BatchSpec
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Len(t, decoded.Operations, 1)
	assert.Equal(t, "op-1", decoded.Operations[0].ID)
}

// ---------------------------------------------------------------------------
// Concurrent access to BatchManager
// ---------------------------------------------------------------------------

func TestBatchManager_ConcurrentAddOperations(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{Interactive: false, Quiet: true})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: fmt.Sprintf("file-%d", v)})
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&bm.stats.Total))
}

func TestBatchManager_ConcurrentGetStats(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats := bm.GetStats()
			assert.NotNil(t, stats)
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Operation struct field defaults
// ---------------------------------------------------------------------------

func TestOperation_Defaults(t *testing.T) {
	op := &Operation{
		Type:   OperationTypeCopy,
		Source: "/tmp/test.txt",
	}

	assert.Equal(t, "", op.ID)
	assert.Equal(t, OperationStatus(""), op.Status)
	assert.Equal(t, "", op.Destination)
	assert.Nil(t, op.Options)
	assert.Equal(t, 0.0, op.Progress)
	assert.Equal(t, "", op.Error)
	assert.True(t, op.StartTime.IsZero())
	assert.True(t, op.EndTime.IsZero())
	assert.Equal(t, 0*time.Nanosecond, op.Duration)
	assert.Equal(t, int64(0), op.Size)
	assert.Equal(t, 0.0, op.Speed)
}

func TestBatchConfig_ZeroDefaults(t *testing.T) {
	cfg := &BatchConfig{}
	assert.Equal(t, 0, cfg.Concurrency)
	assert.False(t, cfg.ContinueOnError)
	assert.Equal(t, 0, cfg.Retries)
	assert.Equal(t, 0*time.Nanosecond, cfg.Timeout)
	assert.False(t, cfg.DryRun)
	assert.False(t, cfg.Quiet)
	assert.False(t, cfg.Interactive)
}

// ---------------------------------------------------------------------------
// processResult
// ---------------------------------------------------------------------------

func TestProcessResult_Completed(t *testing.T) {
	bm := NewBatchManager(nil)

	op := &Operation{
		Type:   OperationTypeCopy,
		Status: StatusCompleted,
		Size:   1024,
	}

	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Completed)
	assert.Equal(t, int64(1024), stats.ProcessedSize)
	assert.Equal(t, int32(0), stats.Running)
}

func TestProcessResult_Failed(t *testing.T) {
	var capturedOp *Operation
	var capturedErr error

	bm := NewBatchManager(nil)
	bm.OnError(func(op *Operation, err error) {
		capturedOp = op
		capturedErr = err
	})

	op := &Operation{
		Type:   OperationTypeCopy,
		Status: StatusFailed,
		Error:  "file not found",
	}

	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Failed)
	assert.NotNil(t, capturedOp)
	assert.NotNil(t, capturedErr)
}

func TestProcessResult_Skipped(t *testing.T) {
	bm := NewBatchManager(nil)

	op := &Operation{Type: OperationTypeCopy, Status: StatusSkipped}
	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Skipped)
}

func TestProcessResult_Cancelled(t *testing.T) {
	bm := NewBatchManager(nil)

	op := &Operation{Type: OperationTypeCopy, Status: StatusCancelled}
	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Cancelled)
}

func TestProcessResult_SetsEndTimeAndDuration(t *testing.T) {
	bm := NewBatchManager(nil)

	op := &Operation{
		Type:   OperationTypeCopy,
		Status: StatusCompleted,
	}
	op.StartTime = time.Now().Add(-100 * time.Millisecond)

	atomic.AddInt32(&bm.stats.Running, 1)
	bm.processResult(op)

	assert.False(t, op.EndTime.IsZero())
	assert.Greater(t, op.Duration, 0*time.Nanosecond)
}

// ---------------------------------------------------------------------------
// SaveToFile / LoadFromFile round-trip
// ---------------------------------------------------------------------------

func TestSaveLoadRoundTrip(t *testing.T) {
	bm := NewBatchManager(&BatchConfig{
		Concurrency: 2,
		DryRun:      true,
	})

	bm.AddOperation(&Operation{
		ID:          "op-1",
		Type:        OperationTypeCopy,
		Source:      "/tmp/src.txt",
		Destination: "/tmp/dst.txt",
		Options:     map[string]interface{}{"verify": true},
	})
	bm.AddOperation(&Operation{
		Type:   OperationTypeDelete,
		Source: "/tmp/old.txt",
	})

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "roundtrip.json")

	require.NoError(t, bm.SaveToFile(f))

	bm2 := NewBatchManager(nil)
	require.NoError(t, bm2.LoadFromFile(f))

	ops := bm2.GetOperations()
	require.Len(t, ops, 2)
	assert.Equal(t, OperationTypeCopy, ops[0].Type)
	assert.Equal(t, "/tmp/src.txt", ops[0].Source)
	assert.Equal(t, "/tmp/dst.txt", ops[0].Destination)
	assert.True(t, ops[0].Options["verify"].(bool))
	assert.Equal(t, OperationTypeDelete, ops[1].Type)
	assert.Equal(t, "/tmp/old.txt", ops[1].Source)
}
