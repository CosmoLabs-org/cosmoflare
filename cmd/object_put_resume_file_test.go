package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// objectPutCredsGuard clears the credential globals and the ambient
// CLOUDFLARE_* env vars for the duration of a test, so API client
// construction fails deterministically instead of using real credentials.
// It also normalises DryRun/JSONOutput and restores everything afterwards.
func objectPutCredsGuard(t *testing.T) {
	t.Helper()
	origAccount, origToken := AccountID, APIToken
	origDry, origJSON := DryRun, JSONOutput
	AccountID, APIToken = "", ""
	DryRun, JSONOutput = false, false
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Cleanup(func() {
		AccountID, APIToken = origAccount, origToken
		DryRun, JSONOutput = origDry, origJSON
	})
}

// objectPutSaveState persists a resumable multipart-upload state for the
// given bucket/key and registers cleanup of the temp state file.
func objectPutSaveState(t *testing.T, bucket, key string) {
	t.Helper()
	state := &cosmoflare.MultipartUploadState{
		UploadID:       "upload-test-1",
		Bucket:         bucket,
		Key:            key,
		TotalSize:      1024,
		PartSize:       512,
		TotalParts:     2,
		CompletedParts: []cosmoflare.CompletedPartInfo{{PartNumber: 1, ETag: "etag-1", Size: 512}},
		StartedAt:      time.Now().UTC(),
	}
	if err := cosmoflare.SaveUploadState(state); err != nil {
		t.Fatalf("SaveUploadState failed: %v", err)
	}
	t.Cleanup(func() {
		_ = cosmoflare.RemoveUploadState(bucket, key)
	})
}

// TestObjectPutResume_NoStateFound verifies that resuming without a saved
// upload state returns the "no interrupted upload found" error instead of
// starting a new upload.
func TestObjectPutResume_NoStateFound(t *testing.T) {
	objectPutCredsGuard(t)

	bucket, key := "cf-test-resume-nostate", "video.mp4"
	t.Cleanup(func() { _ = cosmoflare.RemoveUploadState(bucket, key) })

	err := objectPutResume(&objectPutParams{bucketName: bucket, key: key, localPath: "unused.bin"})
	if err == nil {
		t.Fatal("objectPutResume without saved state should return an error")
	}
	if !strings.Contains(err.Error(), "no interrupted upload found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no interrupted upload found")
	}
	if !strings.Contains(err.Error(), bucket+"/"+key) {
		t.Errorf("error = %q, want it to name bucket/key %q", err.Error(), bucket+"/"+key)
	}
}

// TestObjectPutResume_EmptyKeyFailsLoad verifies an empty key surfaces the
// upload-state load error rather than panicking.
func TestObjectPutResume_EmptyKeyFailsLoad(t *testing.T) {
	objectPutCredsGuard(t)

	err := objectPutResume(&objectPutParams{bucketName: "some-bucket", key: "", localPath: "f.bin"})
	if err == nil {
		t.Fatal("objectPutResume with empty key should return an error")
	}
	if !strings.Contains(err.Error(), "failed to load upload state") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to load upload state")
	}
}

// TestObjectPutResume_MissingLocalFile verifies that a saved state with a
// nonexistent local file fails when opening the file, after the state loads.
func TestObjectPutResume_MissingLocalFile(t *testing.T) {
	objectPutCredsGuard(t)

	bucket, key := "cf-test-resume-nofile", "video.mp4"
	objectPutSaveState(t, bucket, key)

	err := objectPutResume(&objectPutParams{
		bucketName: bucket,
		key:        key,
		localPath:  filepath.Join(t.TempDir(), "does-not-exist.bin"),
	})
	if err == nil {
		t.Fatal("objectPutResume with missing local file should return an error")
	}
	if !strings.Contains(err.Error(), "failed to open local file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to open local file")
	}
}

// TestObjectPutResume_MissingAPICreds verifies the resume path reaches API
// client construction (state loaded, file opened) and surfaces the client
// error when credentials are absent — without issuing network requests.
func TestObjectPutResume_MissingAPICreds(t *testing.T) {
	objectPutCredsGuard(t)

	bucket, key := "cf-test-resume-nocreds", "video.mp4"
	objectPutSaveState(t, bucket, key)

	local := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(local, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}

	err := objectPutResume(&objectPutParams{bucketName: bucket, key: key, localPath: local})
	if err == nil {
		t.Fatal("objectPutResume without API credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create API client") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create API client")
	}
}

// TestObjectPutFile_MissingLocalFile verifies a nonexistent local path fails
// with the file-open error before any client or metadata handling.
func TestObjectPutFile_MissingLocalFile(t *testing.T) {
	objectPutCredsGuard(t)

	err := objectPutFile(&objectPutParams{
		bucketName: "my-bucket",
		key:        "k.bin",
		localPath:  filepath.Join(t.TempDir(), "missing.bin"),
	}, 8*1024*1024)
	if err == nil {
		t.Fatal("objectPutFile with missing local file should return an error")
	}
	if !strings.Contains(err.Error(), "failed to open local file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to open local file")
	}
}

// TestObjectPutFile_InvalidMetadata verifies malformed key=value metadata
// fails with the metadata parse error.
func TestObjectPutFile_InvalidMetadata(t *testing.T) {
	objectPutCredsGuard(t)

	local := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(local, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := objectPutFile(&objectPutParams{
		bucketName: "my-bucket",
		key:        "f.txt",
		localPath:  local,
		metadata:   []string{"not-a-pair"},
	}, 8*1024*1024)
	if err == nil {
		t.Fatal("objectPutFile with invalid metadata should return an error")
	}
	if !strings.Contains(err.Error(), "failed to parse metadata") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to parse metadata")
	}
}

// TestObjectPutFile_DryRun verifies dry-run mode describes the upload and
// returns nil without constructing an API client.
func TestObjectPutFile_DryRun(t *testing.T) {
	objectPutCredsGuard(t)
	DryRun = true

	local := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(local, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := objectPutFile(&objectPutParams{
		bucketName:   "my-bucket",
		key:          "f.txt",
		localPath:    local,
		contentType:  "text/plain",
		cacheControl: "max-age=60",
		metadata:     []string{"author=test"},
	}, 8*1024*1024)
	if err != nil {
		t.Errorf("objectPutFile(DryRun) returned error: %v", err)
	}
}

// TestObjectPutFile_MissingAPICreds verifies the non-dry-run path surfaces
// the API client construction error when credentials are absent.
func TestObjectPutFile_MissingAPICreds(t *testing.T) {
	objectPutCredsGuard(t)

	local := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(local, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := objectPutFile(&objectPutParams{
		bucketName: "my-bucket",
		key:        "f.txt",
		localPath:  local,
	}, 8*1024*1024)
	if err == nil {
		t.Fatal("objectPutFile without credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create API client") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create API client")
	}
}
