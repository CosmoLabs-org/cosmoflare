package cosmoflare

import (
	"context"
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
	return nil
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
