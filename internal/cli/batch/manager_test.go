package batch

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()
	assert.Equal(t, 4, config.Concurrency)
	assert.True(t, config.ContinueOnError)
	assert.Equal(t, 3, config.Retries)
	assert.Equal(t, 30*time.Minute, config.Timeout)
	assert.False(t, config.DryRun)
}

func TestNewBatchManager(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		bm := NewBatchManager(nil)
		require.NotNil(t, bm)
		assert.Equal(t, 4, bm.config.Concurrency)
		assert.NotNil(t, bm.operations)
		assert.NotNil(t, bm.stats)
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
}

func TestGetOperations(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})
	bm.AddOperation(&Operation{Type: OperationTypeUpload, Source: "b"})

	ops := bm.GetOperations()
	assert.Len(t, ops, 2)
	assert.Equal(t, "a", ops[0].Source)
	assert.Equal(t, "b", ops[1].Source)
}

func TestGetStats(t *testing.T) {
	bm := NewBatchManager(nil)
	bm.AddOperation(&Operation{Type: OperationTypeCopy, Source: "a"})

	stats := bm.GetStats()
	assert.Equal(t, int32(1), stats.Total)
	assert.NotZero(t, stats.StartTime)
}

func TestCancel(t *testing.T) {
	bm := NewBatchManager(nil)
	// Should not panic
	assert.NotPanics(t, func() {
		bm.Cancel()
	})
}

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

	// Verify file exists and has content
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "src.txt")
}

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
}

func TestOperationTypes(t *testing.T) {
	assert.Equal(t, OperationType("copy"), OperationTypeCopy)
	assert.Equal(t, OperationType("upload"), OperationTypeUpload)
	assert.Equal(t, OperationType("download"), OperationTypeDownload)
	assert.Equal(t, OperationType("delete"), OperationTypeDelete)
	assert.Equal(t, OperationType("sync"), OperationTypeSync)
}

func TestOperationStatus(t *testing.T) {
	assert.Equal(t, OperationStatus("pending"), StatusPending)
	assert.Equal(t, OperationStatus("running"), StatusRunning)
	assert.Equal(t, OperationStatus("completed"), StatusCompleted)
	assert.Equal(t, OperationStatus("failed"), StatusFailed)
	assert.Equal(t, OperationStatus("skipped"), StatusSkipped)
	assert.Equal(t, OperationStatus("cancelled"), StatusCancelled)
}
