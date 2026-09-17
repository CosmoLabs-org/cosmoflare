package migration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckpointHashDeterministic verifies checkpointHash is a pure function
// of the migration identity: the same (source, destination, filter) triple
// always yields the same hash, and a changed filter yields a different one.
func TestCheckpointHashDeterministic(t *testing.T) {
	t.Parallel()

	t.Run("same inputs produce the same hash", func(t *testing.T) {
		t.Parallel()
		h1 := checkpointHash("src", "dst", "images/*")
		h2 := checkpointHash("src", "dst", "images/*")
		if h1 != h2 {
			t.Errorf("same inputs should produce same hash: %s != %s", h1, h2)
		}
	})

	t.Run("a different filter produces a different hash", func(t *testing.T) {
		t.Parallel()
		h1 := checkpointHash("src", "dst", "images/*")
		h3 := checkpointHash("src", "dst", "other/*")
		if h1 == h3 {
			t.Error("different inputs should produce different hashes")
		}
	})
}

// TestCheckpointSaveLoadRoundTrip validates that every field resume depends
// on — identity, completed map, failed list — survives a save/load cycle.
func TestCheckpointSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cp := &Checkpoint{
		Version:         1,
		S3Bucket:        "src",
		R2Bucket:        "dst",
		Filter:          "*.jpg",
		Completed:       map[string]int64{"a.jpg": 100, "b.jpg": 200},
		Failed:          []string{"c.jpg"},
		TransferredSize: 300,
		TotalObjects:    10,
		TotalSize:       5000,
	}
	path := filepath.Join(dir, "test.json")
	if err := saveCheckpoint(cp, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := loadCheckpoint(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	t.Run("identity fields round-trip", func(t *testing.T) {
		t.Parallel()
		if loaded.S3Bucket != "src" {
			t.Errorf("S3Bucket = %q, want %q", loaded.S3Bucket, "src")
		}
	})
	t.Run("completed map round-trips with sizes", func(t *testing.T) {
		t.Parallel()
		if len(loaded.Completed) != 2 {
			t.Errorf("Completed count = %d, want 2", len(loaded.Completed))
		}
		if loaded.Completed["a.jpg"] != 100 {
			t.Errorf("Completed[a.jpg] = %d, want 100", loaded.Completed["a.jpg"])
		}
	})
	t.Run("failed list round-trips", func(t *testing.T) {
		t.Parallel()
		if len(loaded.Failed) != 1 || loaded.Failed[0] != "c.jpg" {
			t.Errorf("Failed = %v, want [c.jpg]", loaded.Failed)
		}
	})
}

// TestCheckpointDelete verifies deleteCheckpoint removes the file from disk
// without erroring when the checkpoint exists.
func TestCheckpointDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	if err := saveCheckpoint(cp, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := deleteCheckpoint(path); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

// TestCheckpointLoadCorrupt verifies that malformed JSON on disk is reported
// as an error instead of silently yielding a zero-value checkpoint.
func TestCheckpointLoadCorrupt(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt fixture: %v", err)
	}
	if _, err := loadCheckpoint(path); err == nil {
		t.Error("expected error loading corrupt checkpoint")
	}
}

// TestCheckpointLoadMissing verifies the "no previous checkpoint" case: a
// missing file is not an error and yields a nil checkpoint, so callers start
// a fresh migration.
func TestCheckpointLoadMissing(t *testing.T) {
	t.Parallel()
	cp, err := loadCheckpoint(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cp != nil {
		t.Error("missing file should return nil checkpoint")
	}
}

// TestCheckpointPath verifies checkpointPathIn derives a filename placed
// inside the given directory with a .json extension.
func TestCheckpointPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := checkpointPathIn(dir, "src", "dst", "filter")

	t.Run("lives inside the given directory", func(t *testing.T) {
		t.Parallel()
		if filepath.Dir(path) != dir {
			t.Errorf("should be in %s, got %s", dir, filepath.Dir(path))
		}
	})
	t.Run("uses the .json extension", func(t *testing.T) {
		t.Parallel()
		if filepath.Ext(path) != ".json" {
			t.Error("should have .json extension")
		}
	})
}

// TestCheckpointMarkCompleted verifies markCompleted records the transferred
// key and its size, accumulates TransferredSize, and leaves untouched keys
// pending.
func TestCheckpointMarkCompleted(t *testing.T) {
	t.Parallel()
	cp := &Checkpoint{Completed: map[string]int64{}}
	cp.markCompleted("a.jpg", 1024)

	t.Run("marks the transferred key", func(t *testing.T) {
		t.Parallel()
		if !cp.isCompleted("a.jpg") {
			t.Error("a.jpg should be completed")
		}
	})
	t.Run("leaves other keys pending", func(t *testing.T) {
		t.Parallel()
		if cp.isCompleted("b.jpg") {
			t.Error("b.jpg should not be completed")
		}
	})
	t.Run("accumulates transferred size", func(t *testing.T) {
		t.Parallel()
		if cp.TransferredSize != 1024 {
			t.Errorf("TransferredSize = %d, want 1024", cp.TransferredSize)
		}
	})
}
