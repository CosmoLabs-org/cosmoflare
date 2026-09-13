package cosmoflare

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- StorageBackend mock ---

type mockStorageBackend struct {
	objects   []ObjectInfo
	uploaded  []string
	downloaded []string
	deleted   []string
	failList  bool
}

func (m *mockStorageBackend) ListRemoteObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	if m.failList {
		return nil, newError("ListRemoteObjects", "mock failure", nil)
	}
	var filtered []ObjectInfo
	for _, o := range m.objects {
		if prefix == "" || len(o.Key) >= len(prefix) && o.Key[:len(prefix)] == prefix {
			filtered = append(filtered, o)
		}
	}
	return filtered, nil
}

func (m *mockStorageBackend) UploadFile(ctx context.Context, bucket, key, localPath string) error {
	m.uploaded = append(m.uploaded, key)
	return nil
}

func (m *mockStorageBackend) DownloadFile(ctx context.Context, bucket, key, localPath string) error {
	m.downloaded = append(m.downloaded, key)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(localPath, []byte("data"), 0o644)
}

func (m *mockStorageBackend) DeleteRemoteObject(ctx context.Context, bucket, key string) error {
	m.deleted = append(m.deleted, key)
	return nil
}

// --- SyncService construction ---

func TestNewSyncService(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)
	if svc == nil {
		t.Fatal("NewSyncService returned nil")
	}
	if svc.backend != backend {
		t.Error("backend not stored correctly")
	}
}

// --- SyncDirection ---

func TestSyncDirection_String(t *testing.T) {
	cases := []struct {
		dir  SyncDirection
		want string
	}{
		{SyncUp, "up"},
		{SyncDown, "down"},
	}
	for _, tc := range cases {
		if tc.dir.String() != tc.want {
			t.Errorf("SyncDirection(%d).String() = %q, want %q", tc.dir, tc.dir.String(), tc.want)
		}
	}
}

// --- SyncOp ---

func TestSyncOpAction_String(t *testing.T) {
	cases := []struct {
		action SyncOpAction
		want   string
	}{
		{SyncOpUpload, "upload"},
		{SyncOpDownload, "download"},
		{SyncOpDelete, "delete"},
		{SyncOpSkip, "skip"},
	}
	for _, tc := range cases {
		if tc.action.String() != tc.want {
			t.Errorf("SyncOpAction(%d).String() = %q, want %q", tc.action, tc.action.String(), tc.want)
		}
	}
}

// --- Plan: upload direction ---

func TestPlanUp_NewFilesUploaded(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 100, ModTime: time.Now()},
		{RelPath: "dir/file2.txt", Size: 200, ModTime: time.Now()},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncUp,
		Bucket:    "test-bucket",
		Prefix:    "",
		LocalDir:  "/tmp/test",
		LocalFiles: localFiles,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Plan returned nil")
	}

	uploadCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpUpload {
			uploadCount++
		}
	}
	if uploadCount != 2 {
		t.Errorf("expected 2 uploads, got %d", uploadCount)
	}
}

func TestPlanUp_SkipUnchanged(t *testing.T) {
	now := time.Now()
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100, LastModified: now, ETag: "abc"},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 100, ModTime: now.Add(-time.Hour)},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	skipCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpSkip {
			skipCount++
		}
	}
	if skipCount != 1 {
		t.Errorf("expected 1 skip, got %d", skipCount)
	}
}

func TestPlanUp_UploadModified(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour)
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100, LastModified: old, ETag: "abc"},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 200, ModTime: time.Now()},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	uploadCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpUpload {
			uploadCount++
		}
	}
	if uploadCount != 1 {
		t.Errorf("expected 1 upload (modified), got %d", uploadCount)
	}
}

func TestPlanUp_DeleteExtra(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100, LastModified: time.Now(), ETag: "abc"},
			{Key: "orphan.txt", Size: 50, LastModified: time.Now(), ETag: "xyz"},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 100, ModTime: time.Now().Add(-time.Hour)},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Delete:     true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	deleteCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpDelete {
			deleteCount++
		}
	}
	if deleteCount != 1 {
		t.Errorf("expected 1 delete (orphan), got %d", deleteCount)
	}
}

func TestPlanUp_NoDeleteWithoutFlag(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "orphan.txt", Size: 50, LastModified: time.Now(), ETag: "xyz"},
		},
	}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: nil,
		Delete:     false,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	for _, op := range plan.Operations {
		if op.Action == SyncOpDelete {
			t.Error("unexpected delete operation when --delete is false")
		}
	}
}

// --- Plan: download direction ---

func TestPlanDown_NewFilesDownloaded(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "remote1.txt", Size: 100, LastModified: time.Now(), ETag: "abc"},
			{Key: "remote2.txt", Size: 200, LastModified: time.Now(), ETag: "def"},
		},
	}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncDown,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: nil,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	downloadCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpDownload {
			downloadCount++
		}
	}
	if downloadCount != 2 {
		t.Errorf("expected 2 downloads, got %d", downloadCount)
	}
}

func TestPlanDown_DeleteExtraLocal(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "stale-local.txt", Size: 100, ModTime: time.Now()},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncDown,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Delete:     true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	deleteCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpDelete {
			deleteCount++
		}
	}
	if deleteCount != 1 {
		t.Errorf("expected 1 delete (local stale), got %d", deleteCount)
	}
}

// --- Plan: prefix handling ---

func TestPlanUp_WithPrefix(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file.txt", Size: 100, ModTime: time.Now()},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		Prefix:     "subdir/",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	if len(plan.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(plan.Operations))
	}
	op := plan.Operations[0]
	if op.Key != "subdir/file.txt" {
		t.Errorf("expected key %q, got %q", "subdir/file.txt", op.Key)
	}
}

// --- Plan: exclude patterns ---

func TestPlanUp_Exclude(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file.txt", Size: 100, ModTime: time.Now()},
		{RelPath: "file.log", Size: 50, ModTime: time.Now()},
		{RelPath: "dir/other.log", Size: 30, ModTime: time.Now()},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Exclude:    []string{"*.log"},
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	uploadCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpUpload {
			uploadCount++
		}
	}
	if uploadCount != 1 {
		t.Errorf("expected 1 upload (excluding .log files), got %d", uploadCount)
	}
}

// --- Plan: validation ---

func TestPlan_EmptyBucket(t *testing.T) {
	svc := NewSyncService(&mockStorageBackend{})
	_, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncUp,
		Bucket:    "",
		LocalDir:  "/tmp/test",
	})
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
}

func TestPlan_EmptyLocalDir(t *testing.T) {
	svc := NewSyncService(&mockStorageBackend{})
	_, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncUp,
		Bucket:    "test-bucket",
		LocalDir:  "",
	})
	if err == nil {
		t.Fatal("expected error for empty local dir")
	}
}

// --- Plan: summary stats ---

func TestPlanSummary_Counts(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "existing.txt", Size: 100, LastModified: time.Now(), ETag: "abc"},
			{Key: "orphan.txt", Size: 50, LastModified: time.Now(), ETag: "xyz"},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "new.txt", Size: 300, ModTime: time.Now()},
		{RelPath: "existing.txt", Size: 100, ModTime: time.Now().Add(-time.Hour)},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Delete:     true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	if plan.Summary.Uploads != 1 {
		t.Errorf("expected 1 upload, got %d", plan.Summary.Uploads)
	}
	if plan.Summary.Skips != 1 {
		t.Errorf("expected 1 skip, got %d", plan.Summary.Skips)
	}
	if plan.Summary.Deletes != 1 {
		t.Errorf("expected 1 delete, got %d", plan.Summary.Deletes)
	}
}

// --- Execute ---

func TestExecute_UploadOps(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)

	plan := &SyncPlan{
		Bucket:    "test-bucket",
		Direction: SyncUp,
		Operations: []SyncOp{
			{Action: SyncOpUpload, Key: "file1.txt", LocalPath: "/tmp/test/file1.txt", Size: 100},
			{Action: SyncOpUpload, Key: "file2.txt", LocalPath: "/tmp/test/file2.txt", Size: 200},
		},
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(backend.uploaded) != 2 {
		t.Errorf("expected 2 uploads, got %d", len(backend.uploaded))
	}
	if result.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", result.Succeeded)
	}
}

func TestExecute_DownloadOps(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)

	plan := &SyncPlan{
		Bucket:    "test-bucket",
		Direction: SyncDown,
		Operations: []SyncOp{
			{Action: SyncOpDownload, Key: "remote.txt", LocalPath: "/tmp/test/remote.txt", Size: 100},
		},
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(backend.downloaded) != 1 {
		t.Errorf("expected 1 download, got %d", len(backend.downloaded))
	}
	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}
}

func TestExecute_DeleteOps(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)

	plan := &SyncPlan{
		Bucket:    "test-bucket",
		Direction: SyncUp,
		Operations: []SyncOp{
			{Action: SyncOpDelete, Key: "orphan.txt", Size: 50},
		},
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(backend.deleted) != 1 {
		t.Errorf("expected 1 delete, got %d", len(backend.deleted))
	}
	if result.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", result.Succeeded)
	}
}

// BUG-040: a down-sync with --delete must remove stale LOCAL files and must
// never issue a remote delete (the old code called DeleteRemoteObject with an
// unprefixed key, which could delete an unrelated bucket-root object).
func TestExecuteSyncDown_DeleteRemovesLocalFileNeverRemote(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "stale-local.txt")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "shared.txt", Size: 10, LastModified: time.Now()},
		},
	}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncDown,
		Bucket:    "test-bucket",
		LocalDir:  dir,
		LocalFiles: []LocalFileInfo{
			{RelPath: "shared.txt", Size: 10, ModTime: time.Now()},
			{RelPath: "stale-local.txt", Size: 5, ModTime: time.Now()},
		},
		Delete: true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Failed != 0 {
		t.Fatalf("unexpected failures: %v", result.Errors)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("down-sync --delete must remove the local file, still present: %s", stale)
	}
	if len(backend.deleted) != 0 {
		t.Errorf("down-sync must never delete remote objects; DeleteRemoteObject called with %v", backend.deleted)
	}
}

// Regression guard: up-sync --delete still deletes remote objects.
func TestExecuteSyncUp_DeleteStillRemovesRemote(t *testing.T) {
	dir := t.TempDir()
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "orphan-remote.txt", Size: 50, LastModified: time.Now()},
		},
	}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   dir,
		LocalFiles: []LocalFileInfo{},
		Delete:     true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Failed != 0 {
		t.Fatalf("unexpected failures: %v", result.Errors)
	}

	if len(backend.deleted) != 1 || backend.deleted[0] != "orphan-remote.txt" {
		t.Errorf("up-sync --delete must delete the remote orphan; deleted = %v", backend.deleted)
	}
}

// BUG-041: --delete must never target paths matching exclude patterns.
func TestPlanUp_DeleteNeverTargetsExcludedRemoteObjects(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: ".env", Size: 30, LastModified: time.Now()},
			{Key: "keep.txt", Size: 10, LastModified: time.Now()},
			{Key: ".git/config", Size: 10, LastModified: time.Now()},
		},
	}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   t.TempDir(),
		LocalFiles: []LocalFileInfo{},
		Delete:     true,
		Exclude:    []string{".env", ".git/"},
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	for _, op := range plan.Operations {
		if op.Action != SyncOpDelete {
			continue
		}
		rel := strings.TrimPrefix(op.Key, plan.Prefix)
		if rel == ".env" || rel == ".git/config" {
			t.Errorf("delete op planned for excluded path %q (BUG-041)", op.Key)
		}
	}
	if plan.Summary.Deletes != 1 {
		t.Errorf("expected exactly 1 delete (keep.txt), got %d", plan.Summary.Deletes)
	}
}

// BUG-041 (down direction): excluded local files must survive --delete.
func TestPlanDown_DeleteNeverTargetsExcludedLocalFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	backend := &mockStorageBackend{objects: []ObjectInfo{}}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncDown,
		Bucket:    "test-bucket",
		LocalDir:  dir,
		LocalFiles: []LocalFileInfo{
			{RelPath: ".env", Size: 9, ModTime: time.Now()},
		},
		Delete:  true,
		Exclude: []string{".env"},
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	for _, op := range plan.Operations {
		if op.Action == SyncOpDelete {
			t.Errorf("delete op planned for excluded local file %q (BUG-041)", op.LocalPath)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err != nil {
		t.Fatalf(".env must be protected, stat error: %v", err)
	}
}

func TestExecute_SkipsSkipOps(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)

	plan := &SyncPlan{
		Bucket:    "test-bucket",
		Direction: SyncUp,
		Operations: []SyncOp{
			{Action: SyncOpSkip, Key: "unchanged.txt", Size: 100},
		},
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(backend.uploaded) != 0 {
		t.Errorf("expected 0 uploads for skip ops, got %d", len(backend.uploaded))
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
}

func TestExecute_ProgressCallback(t *testing.T) {
	backend := &mockStorageBackend{}
	svc := NewSyncService(backend)

	var events []SyncProgress
	svc.OnProgress = func(p SyncProgress) {
		events = append(events, p)
	}

	plan := &SyncPlan{
		Bucket:    "test-bucket",
		Direction: SyncUp,
		Operations: []SyncOp{
			{Action: SyncOpUpload, Key: "file1.txt", LocalPath: "/tmp/test/file1.txt", Size: 100},
			{Action: SyncOpUpload, Key: "file2.txt", LocalPath: "/tmp/test/file2.txt", Size: 200},
		},
	}

	_, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 progress events, got %d", len(events))
	}
}

// --- Checksum mode ---

func TestPlanUp_ChecksumMode(t *testing.T) {
	now := time.Now()
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100, LastModified: now, ETag: "\"abc123\""},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 100, ModTime: now.Add(time.Hour), Checksum: "abc123"},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Checksum:   true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	// Same checksum despite different mtime => skip
	skipCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpSkip {
			skipCount++
		}
	}
	if skipCount != 1 {
		t.Errorf("expected 1 skip (same checksum), got %d", skipCount)
	}
}

func TestPlanUp_ChecksumModeDifferent(t *testing.T) {
	now := time.Now()
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100, LastModified: now, ETag: "\"abc123\""},
		},
	}
	svc := NewSyncService(backend)

	localFiles := []LocalFileInfo{
		{RelPath: "file1.txt", Size: 100, ModTime: now, Checksum: "different_hash"},
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction:  SyncUp,
		Bucket:     "test-bucket",
		LocalDir:   "/tmp/test",
		LocalFiles: localFiles,
		Checksum:   true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	uploadCount := 0
	for _, op := range plan.Operations {
		if op.Action == SyncOpUpload {
			uploadCount++
		}
	}
	if uploadCount != 1 {
		t.Errorf("expected 1 upload (different checksum), got %d", uploadCount)
	}
}

// --- Multipart ETag fallback + download mtime restore ---

func TestIsUnchangedMultipartETagFallback(t *testing.T) {
	ts := time.Now().Add(-time.Hour)
	svc := NewSyncService(&mockStorageBackend{})

	local := LocalFileInfo{Size: 10, ModTime: ts}
	remote := ObjectInfo{
		Size:         10,
		LastModified: ts,
		ETag:         "\"d41d8cd98f00b204e9800998ecf8427e-4\"",
	}

	if !svc.isUnchanged(local, remote, true) {
		t.Error("expected isUnchanged=true for multipart ETag with equal size/mtime")
	}

	remote.Size = 11
	if svc.isUnchanged(local, remote, true) {
		t.Error("expected isUnchanged=false for multipart ETag with size mismatch")
	}
}

func TestExecutePlanDownloadRestoresModTime(t *testing.T) {
	remoteTime := time.Now().Add(-2 * time.Minute).Truncate(time.Second)
	dir := t.TempDir()
	localPath := filepath.Join(dir, "k")

	svc := NewSyncService(&mockStorageBackend{})
	plan := &SyncPlan{
		Direction: SyncDown,
		Bucket:    "b",
		LocalDir:  dir,
		Operations: []SyncOp{
			{
				Action:        SyncOpDownload,
				Key:           "k",
				LocalPath:     localPath,
				RemoteModTime: remoteTime,
			},
		},
	}

	result, err := svc.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Failed != 0 {
		t.Fatalf("expected 0 failed ops, got %d (%v)", result.Failed, result.Errors)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
	if diff := info.ModTime().Sub(remoteTime); diff > time.Second || diff < -time.Second {
		t.Errorf("expected ModTime ≈ %v, got %v (diff %v)", remoteTime, info.ModTime(), diff)
	}
}

func TestIsUnchangedPlainETagChecksum(t *testing.T) {
	svc := NewSyncService(&mockStorageBackend{})

	local := LocalFileInfo{Checksum: "d41d8cd98f00b204e9800998ecf8427e"}
	remote := ObjectInfo{ETag: "\"d41d8cd98f00b204e9800998ecf8427e\""}

	if !svc.isUnchanged(local, remote, true) {
		t.Error("expected isUnchanged=true for matching plain ETag checksum")
	}

	local.Checksum = "0123456789abcdef0123456789abcdef"
	if svc.isUnchanged(local, remote, true) {
		t.Error("expected isUnchanged=false for mismatched plain ETag checksum")
	}
}

// --- BUG-051: --include must filter the sync universe (it was a no-op) ---

func TestPlanUp_IncludeFiltersUploads(t *testing.T) {
	backend := &mockStorageBackend{objects: []ObjectInfo{}}
	svc := NewSyncService(backend)

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncUp,
		Bucket:    "test-bucket",
		LocalDir:  t.TempDir(),
		LocalFiles: []LocalFileInfo{
			{RelPath: "data/a.csv", Size: 10, ModTime: time.Now()},
			{RelPath: "data/b.csv", Size: 10, ModTime: time.Now()},
			{RelPath: "other/x.txt", Size: 10, ModTime: time.Now()},
		},
		Include: []string{"data/"},
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	uploaded := map[string]bool{}
	for _, op := range plan.Operations {
		if op.Action == SyncOpUpload {
			uploaded[strings.TrimPrefix(op.Key, plan.Prefix)] = true
		}
	}
	if !uploaded["data/a.csv"] || !uploaded["data/b.csv"] {
		t.Errorf("included files must upload, got %v", uploaded)
	}
	if uploaded["other/x.txt"] {
		t.Error("file outside the include filter must not upload (BUG-051: --include was a silent no-op)")
	}
}

func TestPlanDown_IncludeFiltersDownloadsAndDeletes(t *testing.T) {
	backend := &mockStorageBackend{
		objects: []ObjectInfo{
			{Key: "keep/1.csv", Size: 10, LastModified: time.Now()},
			{Key: "skip/2.txt", Size: 10, LastModified: time.Now()},
		},
	}
	svc := NewSyncService(backend)

	// Local has a stale file OUTSIDE the include universe: down-sync --delete
	// must not touch it (it is not part of the managed set).
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "skip"), 0o755); err != nil {
		t.Fatalf("mkdir skip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip", "stale.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	plan, err := svc.Plan(context.Background(), SyncPlanInput{
		Direction: SyncDown,
		Bucket:    "test-bucket",
		LocalDir:  dir,
		LocalFiles: []LocalFileInfo{
			{RelPath: "skip/stale.txt", Size: 1, ModTime: time.Now()},
		},
		Include: []string{"keep/"},
		Delete:  true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	for _, op := range plan.Operations {
		if op.Action == SyncOpDownload && op.Key == "skip/2.txt" {
			t.Error("remote object outside the include filter must not download (BUG-051)")
		}
		if op.Action == SyncOpDelete {
			t.Errorf("delete planned for %q — paths outside the include universe must never be delete-eligible (BUG-051)", op.Key)
		}
	}
	if plan.Summary.Deletes != 0 {
		t.Errorf("expected 0 deletes, got %d", plan.Summary.Deletes)
	}
}
