package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// streamRunSnapshot snapshots the credential, output and stream flag variables
// so tests cannot leak state into each other.
func streamRunSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	savedURL, savedMeta := streamURL, streamMetadata
	savedExpires := streamExpires
	t.Cleanup(func() {
		streamURL, streamMetadata = savedURL, savedMeta
		streamExpires = savedExpires
	})
}

// TestRunStreamList_ServiceCreationFailsOffline verifies the list runner
// aborts with a wrapped service error before any network call when
// credentials are empty.
func TestRunStreamList_ServiceCreationFailsOffline(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runStreamList(streamListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create Stream service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunStreamLiveList_ServiceCreationFailsOffline verifies the live-input
// list runner aborts with a wrapped service error when credentials are empty.
func TestRunStreamLiveList_ServiceCreationFailsOffline(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runStreamLiveList(streamLiveListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create Stream service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunStreamUploadByURL_DryRun verifies the URL upload path short-circuits
// in dry-run mode without credentials and without contacting the API.
func TestRunStreamUploadByURL_DryRun(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = true
	streamURL = "https://example.com/video.mp4"

	if err := runStreamUploadByURL(streamUploadCmd); err != nil {
		t.Fatalf("dry-run URL upload returned error: %v", err)
	}
}

// TestRunStreamUploadByURL_ServiceCreationFailsOffline verifies the URL upload
// path aborts with a wrapped service error when credentials are empty.
func TestRunStreamUploadByURL_ServiceCreationFailsOffline(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false
	streamURL = "https://example.com/video.mp4"

	err := runStreamUploadByURL(streamUploadCmd)
	if err == nil || !strings.Contains(err.Error(), "failed to create Stream service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunStreamUpload_MissingFileFailsBeforeService verifies the file upload
// path checks the local file before building the service, so a bad path errors
// even with empty credentials.
func TestRunStreamUpload_MissingFileFailsBeforeService(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false
	streamURL = ""

	err := runStreamUpload(streamUploadCmd, []string{filepath.Join(os.TempDir(), "definitely-missing-video.mp4")})
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("expected file-not-found error, got %v", err)
	}
}

// TestRunStreamUpload_InvalidMetadata verifies the metadata flag is rejected
// as malformed JSON after the file check but before the upload starts.
func TestRunStreamUpload_InvalidMetadata(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "acct", "token"
	DryRun = false
	streamURL = ""
	streamMetadata = "{not valid json"

	file := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(file, []byte("fake"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := runStreamUpload(streamUploadCmd, []string{file})
	if err == nil || !strings.Contains(err.Error(), "invalid metadata JSON") {
		t.Fatalf("expected invalid-metadata error, got %v", err)
	}
}

// TestRunStreamToken_InvalidExpires verifies the token runner rejects a
// malformed --expires value before building a service. The "7d" form must
// parse as seven days.
func TestRunStreamToken_InvalidExpires(t *testing.T) {
	streamRunSnapshot(t)
	AccountID, APIToken = "", ""
	DryRun = false
	streamExpires = "not-a-duration"

	err := runStreamToken(streamTokenCmd, []string{"vid-1"})
	if err == nil || !strings.Contains(err.Error(), "invalid --expires value") {
		t.Fatalf("expected invalid-expires error, got %v", err)
	}
}
