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

func TestCheckpointMarkCompleted(t *testing.T) {
	cp := &Checkpoint{Completed: map[string]int64{}}
	cp.markCompleted("a.jpg", 1024)
	if !cp.isCompleted("a.jpg") {
		t.Error("a.jpg should be completed")
	}
	if cp.isCompleted("b.jpg") {
		t.Error("b.jpg should not be completed")
	}
	if cp.TransferredSize != 1024 {
		t.Errorf("TransferredSize = %d, want 1024", cp.TransferredSize)
	}
}
