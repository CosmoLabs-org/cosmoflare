package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/batch"
	"github.com/CosmoLabs-org/cosmoflare/internal/cli/operations"
)

// copyRunGlobals snapshots the copy flag variables and resets them to their
// registered defaults (progress bars disabled, quiet on) so tests cannot leak
// state or spam the terminal.
func copyRunGlobals(t *testing.T) {
	t.Helper()
	oldRecursive, oldResume, oldVerify := copyRecursive, copyResume, copyVerify
	oldOverwrite, oldPreserve, oldProgress := copyOverwrite, copyPreserve, copyProgress
	oldQuiet, oldBatch, oldParallel := copyQuiet, copyBatch, copyParallel
	oldChunk, oldRetries, oldTimeout := copyChunkSize, copyRetries, copyTimeout
	oldInteractive, oldDryRun, oldNoClobber := copyInteractive, copyDryRun, copyNoClobber
	oldStats, oldFormat := copyStats, copyFormat
	t.Cleanup(func() {
		copyRecursive, copyResume, copyVerify = oldRecursive, oldResume, oldVerify
		copyOverwrite, copyPreserve, copyProgress = oldOverwrite, oldPreserve, oldProgress
		copyQuiet, copyBatch, copyParallel = oldQuiet, oldBatch, oldParallel
		copyChunkSize, copyRetries, copyTimeout = oldChunk, oldRetries, oldTimeout
		copyInteractive, copyDryRun, copyNoClobber = oldInteractive, oldDryRun, oldNoClobber
		copyStats, copyFormat = oldStats, oldFormat
	})
	copyRecursive, copyResume, copyVerify = false, false, true
	copyOverwrite, copyPreserve, copyProgress = false, true, false
	copyQuiet, copyBatch, copyParallel = true, false, 4
	copyChunkSize, copyRetries, copyTimeout = "8MB", 3, "30m"
	copyInteractive, copyDryRun, copyNoClobber = false, false, false
	copyStats, copyFormat = false, "table"
}

// copyRunWriteFile creates a file with the given content under dir.
func copyRunWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestRunCopy_ValidationErrors verifies every fast-fail branch of runCopy:
// argument count, timeout parsing, source access, directory guard, and the
// batch-mode file-extension check.
func TestRunCopy_ValidationErrors(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T) []string
		wantErr string
	}{
		{"no args", func(*testing.T) []string { return nil },
			"source and destination are required"},
		{"one arg", func(*testing.T) []string { return []string{"src.txt"} },
			"source and destination are required"},
		{"invalid timeout", func(*testing.T) []string {
			copyTimeout = "not-a-duration"
			return []string{"src", "dst"}
		}, "invalid timeout format"},
		{"missing source", func(*testing.T) []string {
			return []string{filepath.Join(os.TempDir(), "copy-run-no-such-src"), "dst"}
		}, "failed to access source"},
		{"directory without recursive", func(t *testing.T) []string {
			return []string{t.TempDir(), filepath.Join(t.TempDir(), "out")}
		}, "source is a directory, use --recursive"},
		{"batch file bad extension", func(t *testing.T) []string {
			src := copyRunWriteFile(t, t.TempDir(), "list.csv", "a,b\n")
			copyBatch = true
			return []string{src, t.TempDir()}
		}, "batch mode requires a .txt or .json file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copyRunGlobals(t)
			args := tc.setup(t)
			err := runCopy(copyCmd, args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

// TestRunCopy_SingleFile verifies a real local single-file copy through
// runCopy, including on-disk content and the --stats result display.
func TestRunCopy_SingleFile(t *testing.T) {
	copyRunGlobals(t)
	src := copyRunWriteFile(t, t.TempDir(), "src.txt", "copy-run-payload")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	if err := runCopy(copyCmd, []string{src, dst}); err != nil {
		t.Fatalf("runCopy returned error: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "copy-run-payload" {
		t.Errorf("destination content = %q, want %q", got, "copy-run-payload")
	}
}

// TestRunCopy_SingleFileStats exercises the copyStats display branch of
// copyFile after a successful local copy.
func TestRunCopy_SingleFileStats(t *testing.T) {
	copyRunGlobals(t)
	copyStats = true
	src := copyRunWriteFile(t, t.TempDir(), "src.txt", "stats-payload")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	if err := runCopy(copyCmd, []string{src, dst}); err != nil {
		t.Fatalf("runCopy with --stats returned error: %v", err)
	}
}

// TestRunCopy_DirectoryRecursive copies a two-file directory tree through the
// batch path of copyDirectory.
func TestRunCopy_DirectoryRecursive(t *testing.T) {
	copyRunGlobals(t)
	copyRecursive = true
	srcDir := t.TempDir()
	copyRunWriteFile(t, srcDir, "a.txt", "alpha")
	copyRunWriteFile(t, srcDir, "b.txt", "beta")
	dstDir := t.TempDir()

	if err := runCopy(copyCmd, []string{srcDir, dstDir}); err != nil {
		t.Fatalf("runCopy directory returned error: %v", err)
	}
	for name, want := range map[string]string{"a.txt": "alpha", "b.txt": "beta"} {
		got, err := os.ReadFile(filepath.Join(dstDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s content = %q, want %q", name, got, want)
		}
	}
}

// TestHandleBatchCopy_InvalidExtension verifies the batch guard rejects file
// lists that are neither .txt nor .json.
func TestHandleBatchCopy_InvalidExtension(t *testing.T) {
	copyRunGlobals(t)
	opts := &operations.CopyOptions{
		Timeout:     time.Minute,
		ChunkSize:   8 * 1024 * 1024,
		Concurrency: 4,
		Retries:     1,
	}
	err := handleBatchCopy("list.yaml", t.TempDir(), opts)
	if err == nil || !strings.Contains(err.Error(), "batch mode requires a .txt or .json file") {
		t.Fatalf("error = %v, want extension guard error", err)
	}
}

// TestParseTextBatchFile covers the text-batch parser: unreadable files fail
// fast, and comment/blank/arrow lines are accepted without error.
func TestParseTextBatchFile(t *testing.T) {
	dir := t.TempDir()
	listFile := copyRunWriteFile(t, dir, "list.txt", "# comment\n\n")
	fallbackSrc := copyRunWriteFile(t, dir, "fallback.txt", "fallback")
	arrowSrc := copyRunWriteFile(t, dir, "arrow.txt", "arrow")
	arrowDst := filepath.Join(dir, "arrow-dst.txt")

	if err := os.WriteFile(listFile, []byte("# comment\n\n"+fallbackSrc+"\n"+arrowSrc+" -> "+arrowDst+"\n"), 0o600); err != nil {
		t.Fatalf("rewrite list: %v", err)
	}

	t.Run("missing file", func(t *testing.T) {
		bm := batch.NewBatchManager(&batch.BatchConfig{})
		err := parseTextBatchFile(filepath.Join(dir, "no-such-list.txt"), dir, bm, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to read batch file") {
			t.Fatalf("error = %v, want read error", err)
		}
	})

	t.Run("parses entries", func(t *testing.T) {
		bm := batch.NewBatchManager(&batch.BatchConfig{})
		if err := parseTextBatchFile(listFile, dir, bm, nil); err != nil {
			t.Fatalf("parseTextBatchFile returned error: %v", err)
		}
	})
}

// TestRunCopy_BatchTextFile executes a real .txt batch through handleBatchCopy
// and verifies each arrow-mapped entry lands at its destination.
func TestRunCopy_BatchTextFile(t *testing.T) {
	copyRunGlobals(t)
	copyBatch = true
	dir := t.TempDir()
	firstSrc := copyRunWriteFile(t, dir, "one.txt", "one")
	secondSrc := copyRunWriteFile(t, dir, "two.txt", "two")
	firstDst := filepath.Join(dir, "one-dst.txt")
	secondDst := filepath.Join(dir, "two-dst.txt")
	list := copyRunWriteFile(t, dir, "list.txt",
		firstSrc+" -> "+firstDst+"\n"+secondSrc+" -> "+secondDst+"\n")

	if err := runCopy(copyCmd, []string{list, dir}); err != nil {
		t.Fatalf("runCopy batch returned error: %v", err)
	}
	for _, pair := range [][2]string{{firstDst, "one"}, {secondDst, "two"}} {
		got, err := os.ReadFile(pair[0])
		if err != nil {
			t.Fatalf("read %s: %v", pair[0], err)
		}
		if string(got) != pair[1] {
			t.Errorf("%s content = %q, want %q", pair[0], got, pair[1])
		}
	}
}
