package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// watchSyncStubClient stubs the R2Client methods the watch sync helpers call.
// The embedded interface covers every other method with a nil panic, which is
// fine: only Upload and DeleteObject are reachable from syncUpload/syncDelete.
type watchSyncStubClient struct {
	cosmoflare.R2Client
	uploads   []string // recorded "bucket/key" per successful upload
	deletes   []string // recorded "bucket/key" per successful delete
	lastSize  int64
	uploadErr error
	deleteErr error
}

func (c *watchSyncStubClient) Upload(_ context.Context, bucket, key string, _ io.Reader, size int64, _ ...cosmoflare.UploadOption) (*cosmoflare.UploadResult, error) {
	if c.uploadErr != nil {
		return nil, c.uploadErr
	}
	c.uploads = append(c.uploads, bucket+"/"+key)
	c.lastSize = size
	return &cosmoflare.UploadResult{}, nil
}

func (c *watchSyncStubClient) DeleteObject(_ context.Context, bucket, key string) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	c.deletes = append(c.deletes, bucket+"/"+key)
	return nil
}

// watchSyncTempFile creates a file of the given size and returns its path.
func watchSyncTempFile(t *testing.T, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(path, bytesOfSize(size), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func bytesOfSize(n int) []byte {
	return make([]byte, n)
}

// TestSyncUpload_DryRunEmitsPreview verifies dry-run uploads never touch the
// client and print a preview line instead.
func TestSyncUpload_DryRunEmitsPreview(t *testing.T) {
	watchRunGlobals(t)
	DryRun = true
	JSONOutput = false

	client := &watchSyncStubClient{}
	change := cosmoflare.FileChange{Path: "a.txt", R2Key: "a.txt", Type: cosmoflare.ChangeAdded, Size: 7, FullPath: "/definitely/not/opened.txt"}

	out := capturePrint(t, func() {
		syncUpload(context.Background(), client, "bkt", change)
	})

	if !strings.Contains(out, "[+] uploaded a.txt") || !strings.Contains(out, "(dry-run)") {
		t.Errorf("dry-run preview line wrong: %q", out)
	}
	if len(client.uploads) != 0 {
		t.Errorf("dry-run must not upload, got %v", client.uploads)
	}
}

// TestSyncUpload_UploadedVersusUpdated verifies the action label and symbol
// depend on the change type, and that a real upload reaches the client with
// the correct bucket, key, and size.
func TestSyncUpload_UploadedVersusUpdated(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	JSONOutput = false

	cases := []struct {
		name   string
		change cosmoflare.ChangeType
		symbol string
		action string
	}{
		{"added file", cosmoflare.ChangeAdded, "[+]", "uploaded"},
		{"modified file", cosmoflare.ChangeModified, "[~]", "updated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := watchSyncTempFile(t, 12)
			client := &watchSyncStubClient{}
			change := cosmoflare.FileChange{Path: "dir/a.txt", R2Key: "prefix/dir/a.txt", Type: tc.change, Size: 12, FullPath: path}

			out := capturePrint(t, func() {
				syncUpload(context.Background(), client, "my-bucket", change)
			})

			if !strings.Contains(out, tc.symbol+" "+tc.action+" dir/a.txt") {
				t.Errorf("output %q missing %q", out, tc.symbol+" "+tc.action)
			}
			if len(client.uploads) != 1 || client.uploads[0] != "my-bucket/prefix/dir/a.txt" {
				t.Errorf("client recorded uploads %v", client.uploads)
			}
			if client.lastSize != 12 {
				t.Errorf("uploaded size = %d, want 12 (stat size, not change size)", client.lastSize)
			}
		})
	}
}

// TestSyncUpload_OpenFailureEmitsError verifies an unopenable local file is
// reported as a watch error without calling the client.
func TestSyncUpload_OpenFailureEmitsError(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	JSONOutput = false

	client := &watchSyncStubClient{}
	change := cosmoflare.FileChange{Path: "gone.txt", R2Key: "gone.txt", Type: cosmoflare.ChangeAdded, FullPath: filepath.Join(t.TempDir(), "does-not-exist.bin")}

	out := capturePrint(t, func() {
		syncUpload(context.Background(), client, "bkt", change)
	})

	if !strings.Contains(out, "failed to open gone.txt") {
		t.Errorf("expected open-failure message, got %q", out)
	}
	if len(client.uploads) != 0 {
		t.Errorf("failed upload path must not call client, got %v", client.uploads)
	}
}

// TestSyncUpload_UploadFailureEmitsError verifies a client upload error is
// surfaced as a watch error line.
func TestSyncUpload_UploadFailureEmitsError(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	JSONOutput = false

	path := watchSyncTempFile(t, 3)
	client := &watchSyncStubClient{uploadErr: errors.New("quota exceeded")}
	change := cosmoflare.FileChange{Path: "a.txt", R2Key: "a.txt", Type: cosmoflare.ChangeAdded, FullPath: path}

	out := capturePrint(t, func() {
		syncUpload(context.Background(), client, "bkt", change)
	})

	if !strings.Contains(out, "failed to upload a.txt") || !strings.Contains(out, "quota exceeded") {
		t.Errorf("expected upload-failure message, got %q", out)
	}
}

// TestSyncDelete_Modes verifies delete preview (dry-run), success line, and
// client-failure reporting.
func TestSyncDelete_Modes(t *testing.T) {
	t.Run("dry run previews without deleting", func(t *testing.T) {
		watchRunGlobals(t)
		DryRun = true
		JSONOutput = false

		client := &watchSyncStubClient{}
		change := cosmoflare.FileChange{Path: "old.txt", R2Key: "prefix/old.txt", Type: cosmoflare.ChangeDeleted}

		out := capturePrint(t, func() {
			syncDelete(context.Background(), client, "bkt", change)
		})

		// Dry-run renders via emitWatchEvent: symbol, action, path, suffix.
		if !strings.Contains(out, "[-] deleted old.txt (dry-run)") {
			t.Errorf("dry-run delete line wrong: %q", out)
		}
		if len(client.deletes) != 0 {
			t.Errorf("dry-run must not delete, got %v", client.deletes)
		}
	})

	t.Run("success deletes the r2 key", func(t *testing.T) {
		watchRunGlobals(t)
		DryRun = false
		JSONOutput = false

		client := &watchSyncStubClient{}
		change := cosmoflare.FileChange{Path: "old.txt", R2Key: "prefix/old.txt", Type: cosmoflare.ChangeDeleted}

		out := capturePrint(t, func() {
			syncDelete(context.Background(), client, "my-bucket", change)
		})

		// The non-dry-run path prints the two-space "deleted  " line directly.
		if !strings.Contains(out, "[-] deleted  old.txt") {
			t.Errorf("delete line wrong: %q", out)
		}
		if len(client.deletes) != 1 || client.deletes[0] != "my-bucket/prefix/old.txt" {
			t.Errorf("client recorded deletes %v", client.deletes)
		}
	})

	t.Run("client failure emits error", func(t *testing.T) {
		watchRunGlobals(t)
		DryRun = false
		JSONOutput = false

		client := &watchSyncStubClient{deleteErr: errors.New("key locked")}
		change := cosmoflare.FileChange{Path: "old.txt", R2Key: "prefix/old.txt", Type: cosmoflare.ChangeDeleted}

		out := capturePrint(t, func() {
			syncDelete(context.Background(), client, "bkt", change)
		})

		if !strings.Contains(out, "failed to delete") || !strings.Contains(out, "key locked") {
			t.Errorf("expected delete-failure message, got %q", out)
		}
	})
}

// TestEmitWatchEvent_JSONEmitsNDJSON verifies --json mode renders each event
// as a single parseable NDJSON line carrying the event fields.
func TestEmitWatchEvent_JSONEmitsNDJSON(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	JSONOutput = true

	change := cosmoflare.FileChange{Path: "p/a.txt", R2Key: "prefix/p/a.txt", Type: cosmoflare.ChangeModified, Size: 99}

	out := capturePrint(t, func() { emitWatchEvent("updated", change, true) })

	var evt watchEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &evt); err != nil {
		t.Fatalf("event line is not JSON: %v; raw %q", err, out)
	}
	if evt.Type != "updated" || evt.Path != "p/a.txt" || evt.R2Key != "prefix/p/a.txt" || evt.Size != 99 || !evt.DryRun {
		t.Errorf("parsed event fields wrong: %+v", evt)
	}
	if evt.Time == "" {
		t.Error("event time is empty")
	}
}

// TestEmitWatchEvent_TextFormatting verifies plain-mode formatting: sized
// changes include a byte count, zero-size ones do not.
func TestEmitWatchEvent_TextFormatting(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	JSONOutput = false

	t.Run("with size", func(t *testing.T) {
		change := cosmoflare.FileChange{Path: "a.txt", Type: cosmoflare.ChangeAdded, Size: 10}
		out := capturePrint(t, func() { emitWatchEvent("uploaded", change, false) })
		if !strings.Contains(out, "[+] uploaded a.txt (10 bytes)") {
			t.Errorf("sized line wrong: %q", out)
		}
	})

	t.Run("zero size omits byte count", func(t *testing.T) {
		change := cosmoflare.FileChange{Path: "a.txt", Type: cosmoflare.ChangeAdded}
		out := capturePrint(t, func() { emitWatchEvent("uploaded", change, false) })
		if !strings.Contains(out, "[+] uploaded a.txt\n") {
			t.Errorf("zero-size line wrong: %q", out)
		}
	})
}

// TestEmitWatchError_JSONAndText verifies error emission in both output
// modes: an NDJSON error event in --json mode, a plain error line otherwise.
func TestEmitWatchError_JSONAndText(t *testing.T) {
	watchRunGlobals(t)

	t.Run("json emits error event", func(t *testing.T) {
		JSONOutput = true
		out := capturePrint(t, func() { emitWatchError("scan error: boom", "a.txt") })

		var evt watchEvent
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &evt); err != nil {
			t.Fatalf("error line is not JSON: %v; raw %q", err, out)
		}
		if evt.Type != "error" || evt.Error != "scan error: boom" || evt.Path != "a.txt" {
			t.Errorf("parsed error event wrong: %+v", evt)
		}
	})

	t.Run("text prints error line", func(t *testing.T) {
		JSONOutput = false
		out := capturePrint(t, func() { emitWatchError("scan error: boom", "a.txt") })
		if !strings.Contains(out, "scan error: boom") {
			t.Errorf("text error line wrong: %q", out)
		}
	})
}

// TestWatchPrintHeader_TextMode verifies the startup banner names the
// directory, bucket, scan count, and the dry-run warning when enabled.
func TestWatchPrintHeader_TextMode(t *testing.T) {
	watchRunGlobals(t)
	watchPrefix = "assets/"
	watchInterval = 5 * time.Second
	watchDelete = true
	DryRun = true
	JSONOutput = false

	out := capturePrint(t, func() { watchPrintHeader("/tmp/project", "my-bucket", 3) })

	for _, want := range []string{
		`Watching /tmp/project`,
		`bucket "my-bucket"`,
		`prefix="assets/"`,
		`interval=5s`,
		`delete=true`,
		"Initial scan: 3 file(s)",
		"DRY RUN: no files will be uploaded or deleted",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q: %q", want, out)
		}
	}
}
