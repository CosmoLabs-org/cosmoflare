package r2go2

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- FileWatcher construction ---

func TestNewFileWatcher_ValidDir(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWatcher(dir, nil)
	if err != nil {
		t.Fatalf("NewFileWatcher(%q) returned error: %v", dir, err)
	}
	if fw == nil {
		t.Fatal("NewFileWatcher returned nil")
	}
	if fw.Dir() != dir {
		t.Errorf("Dir() = %q, want %q", fw.Dir(), dir)
	}
}

func TestNewFileWatcher_NonexistentDir(t *testing.T) {
	_, err := NewFileWatcher("/nonexistent/path/xyz", nil)
	if err == nil {
		t.Fatal("NewFileWatcher with nonexistent dir should return error")
	}
}

func TestNewFileWatcher_FileNotDir(t *testing.T) {
	f, err := os.CreateTemp("", "watcher-test-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	_, err = NewFileWatcher(f.Name(), nil)
	if err == nil {
		t.Fatal("NewFileWatcher with a file (not dir) should return error")
	}
}

// --- Snapshot / change detection ---

func TestFileWatcher_Snapshot_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	fw, _ := NewFileWatcher(dir, nil)

	snap, err := fw.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	if len(snap) != 0 {
		t.Errorf("Snapshot() on empty dir should return 0 entries, got %d", len(snap))
	}
}

func TestFileWatcher_Snapshot_WithFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0644)

	fw, _ := NewFileWatcher(dir, nil)
	snap, err := fw.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	if len(snap) != 2 {
		t.Errorf("Snapshot() should return 2 entries, got %d", len(snap))
	}
	// Check that keys are relative paths
	if _, ok := snap["a.txt"]; !ok {
		t.Error("Snapshot() missing key 'a.txt'")
	}
	if _, ok := snap["b.txt"]; !ok {
		t.Error("Snapshot() missing key 'b.txt'")
	}
}

func TestFileWatcher_Snapshot_Subdirectories(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "c.txt"), []byte("deep"), 0644)

	fw, _ := NewFileWatcher(dir, nil)
	snap, err := fw.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	key := filepath.Join("sub", "c.txt")
	if _, ok := snap[key]; !ok {
		t.Errorf("Snapshot() missing key %q for nested file", key)
	}
}

func TestFileWatcher_Snapshot_ExcludePatterns(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("yes"), 0644)
	os.WriteFile(filepath.Join(dir, "skip.log"), []byte("no"), 0644)
	os.WriteFile(filepath.Join(dir, "also.log"), []byte("no"), 0644)

	fw, _ := NewFileWatcher(dir, &WatcherOptions{
		Exclude: []string{"*.log"},
	})
	snap, err := fw.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	if len(snap) != 1 {
		t.Errorf("Snapshot() with exclude *.log should return 1 entry, got %d", len(snap))
	}
	if _, ok := snap["keep.txt"]; !ok {
		t.Error("Snapshot() should include keep.txt")
	}
}

func TestFileWatcher_Snapshot_ExcludeDotDirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("yes"), 0644)
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0644)

	fw, _ := NewFileWatcher(dir, nil)
	snap, err := fw.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	// .git/HEAD should be excluded (hidden dirs skipped by default)
	for k := range snap {
		if filepath.Base(k) == "HEAD" {
			t.Errorf("Snapshot() should skip .git directory, but found key %q", k)
		}
	}
}

// --- Diff computation ---

func TestFileWatcher_Diff_NewFile(t *testing.T) {
	dir := t.TempDir()
	fw, _ := NewFileWatcher(dir, nil)

	// Take initial snapshot (empty)
	old, _ := fw.Snapshot()

	// Add a file
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new content"), 0644)

	// Take new snapshot
	cur, _ := fw.Snapshot()

	changes := fw.Diff(old, cur)
	if len(changes) != 1 {
		t.Fatalf("Diff should find 1 change, got %d", len(changes))
	}
	if changes[0].Path != "new.txt" {
		t.Errorf("change path = %q, want 'new.txt'", changes[0].Path)
	}
	if changes[0].Type != ChangeAdded {
		t.Errorf("change type = %v, want ChangeAdded", changes[0].Type)
	}
}

func TestFileWatcher_Diff_ModifiedFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mod.txt"), []byte("original"), 0644)
	fw, _ := NewFileWatcher(dir, nil)

	old, _ := fw.Snapshot()

	// Modify the file (change size to force detection)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "mod.txt"), []byte("modified content, longer now"), 0644)

	cur, _ := fw.Snapshot()

	changes := fw.Diff(old, cur)
	if len(changes) != 1 {
		t.Fatalf("Diff should find 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeModified {
		t.Errorf("change type = %v, want ChangeModified", changes[0].Type)
	}
}

func TestFileWatcher_Diff_DeletedFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "gone.txt"), []byte("bye"), 0644)
	fw, _ := NewFileWatcher(dir, nil)

	old, _ := fw.Snapshot()

	os.Remove(filepath.Join(dir, "gone.txt"))

	cur, _ := fw.Snapshot()

	changes := fw.Diff(old, cur)
	if len(changes) != 1 {
		t.Fatalf("Diff should find 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeDeleted {
		t.Errorf("change type = %v, want ChangeDeleted", changes[0].Type)
	}
}

func TestFileWatcher_Diff_NoChanges(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "stable.txt"), []byte("same"), 0644)
	fw, _ := NewFileWatcher(dir, nil)

	snap1, _ := fw.Snapshot()
	snap2, _ := fw.Snapshot()

	changes := fw.Diff(snap1, snap2)
	if len(changes) != 0 {
		t.Errorf("Diff with no changes should return 0, got %d", len(changes))
	}
}

func TestFileWatcher_Diff_MixedChanges(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0644)
	os.WriteFile(filepath.Join(dir, "modify.txt"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(dir, "remove.txt"), []byte("gone"), 0644)

	fw, _ := NewFileWatcher(dir, nil)
	old, _ := fw.Snapshot()

	// Modify one, delete one, add one
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "modify.txt"), []byte("new content here"), 0644)
	os.Remove(filepath.Join(dir, "remove.txt"))
	os.WriteFile(filepath.Join(dir, "added.txt"), []byte("hi"), 0644)

	cur, _ := fw.Snapshot()
	changes := fw.Diff(old, cur)

	if len(changes) != 3 {
		t.Fatalf("Diff should find 3 changes, got %d", len(changes))
	}

	types := map[ChangeType]int{}
	for _, c := range changes {
		types[c.Type]++
	}
	if types[ChangeAdded] != 1 {
		t.Errorf("expected 1 added, got %d", types[ChangeAdded])
	}
	if types[ChangeModified] != 1 {
		t.Errorf("expected 1 modified, got %d", types[ChangeModified])
	}
	if types[ChangeDeleted] != 1 {
		t.Errorf("expected 1 deleted, got %d", types[ChangeDeleted])
	}
}

// --- Prefix handling ---

func TestFileWatcher_PrefixedKey(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	fw, _ := NewFileWatcher(dir, &WatcherOptions{
		Prefix: "uploads/",
	})

	snap, _ := fw.Snapshot()
	old := make(FileSnapshot)

	changes := fw.Diff(old, snap)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].R2Key != "uploads/file.txt" {
		t.Errorf("R2Key = %q, want %q", changes[0].R2Key, "uploads/file.txt")
	}
}

// --- WatcherOptions defaults ---

func TestWatcherOptions_DefaultInterval(t *testing.T) {
	dir := t.TempDir()
	fw, _ := NewFileWatcher(dir, nil)
	if fw.Interval() != time.Second {
		t.Errorf("default interval = %v, want 1s", fw.Interval())
	}
}

func TestWatcherOptions_CustomInterval(t *testing.T) {
	dir := t.TempDir()
	fw, _ := NewFileWatcher(dir, &WatcherOptions{
		Interval: 500 * time.Millisecond,
	})
	if fw.Interval() != 500*time.Millisecond {
		t.Errorf("interval = %v, want 500ms", fw.Interval())
	}
}
