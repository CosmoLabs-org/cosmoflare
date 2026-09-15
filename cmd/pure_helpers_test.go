package cmd

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/batch"
	"github.com/CosmoLabs-org/cosmoflare/internal/cli/operations"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// capturePrint swaps os.Stdout for a pipe, runs fn, and returns what it
// printed. Restores the original in all cases.
func capturePrint(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	os.Stdout = old
	_ = w.Close()
	return <-done
}

func TestExtractColumns(t *testing.T) {
	t.Run("empty rows yield nil", func(t *testing.T) {
		if got := extractColumns(nil); got != nil {
			t.Fatalf("extractColumns(nil) = %v, want nil", got)
		}
	})
	t.Run("returns every column of first row", func(t *testing.T) {
		rows := []map[string]any{
			{"id": 1, "name": "a", "size": 3.0},
			{"id": 2},
		}
		got := extractColumns(rows)
		sort.Strings(got)
		want := []string{"id", "name", "size"}
		if len(got) != len(want) {
			t.Fatalf("extractColumns = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("extractColumns = %v, want %v", got, want)
			}
		}
	})
}

func TestBucketSubCommands(t *testing.T) {
	want := []string{"create", "list", "get", "update", "delete", "exists", "import"}
	got := bucketSubCommands()
	if len(got) != len(want) {
		t.Fatalf("bucketSubCommands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("bucketSubCommands = %v, want %v", got, want)
		}
	}
}

func TestPrintQueryMeta(t *testing.T) {
	result := &cosmoflare.D1QueryResult{
		Meta: cosmoflare.D1QueryMeta{RowsRead: 10, RowsWritten: 2, Changes: 2, Duration: 1.5},
	}
	out := capturePrint(t, func() { printQueryMeta(result) })
	for _, want := range []string{"Rows read: 10", "Rows written: 2", "Changes: 2", "1.50ms"} {
		if !strings.Contains(out, want) {
			t.Fatalf("printQueryMeta output %q missing %q", out, want)
		}
	}
}

func TestRenderCompareResult(t *testing.T) {
	result := CompareResult{
		OnlyInSource:  []string{"only-src.txt"},
		OnlyInDest:    []string{"only-dst.bin"},
		DifferentSize: []CompareDiff{{Key: "diff.txt", SourceSize: 1024, DestSize: 2048}},
		Same:          []string{"same.txt"},
	}
	srcMap := map[string]int64{"only-src.txt": 512}
	dstMap := map[string]int64{"only-dst.bin": 8}

	out := capturePrint(t, func() { renderCompareResult(result, srcMap, dstMap) })
	for _, want := range []string{"ONLY IN SOURCE", "only-src.txt", "ONLY IN DEST", "only-dst.bin", "DIFFERENT SIZE", "diff.txt", "1 same"} {
		if !strings.Contains(out, want) {
			t.Fatalf("renderCompareResult output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintBatchProgressAndResults(t *testing.T) {
	stats := &batch.BatchStats{
		Total: 4, Completed: 3, Failed: 1, Skipped: 0,
		Progress: 75, AverageSpeed: 12.5, ETA: 2 * time.Second,
		TotalDuration: 8 * time.Second, TotalSize: 1 << 20,
	}

	out := capturePrint(t, func() { printBatchProgress(stats) })
	if !strings.Contains(out, "75.0%") || !strings.Contains(out, "3/4") {
		t.Fatalf("printBatchProgress output unexpected: %q", out)
	}

	oldJSON := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = oldJSON })

	out = capturePrint(t, func() { printBatchResults(stats) })
	for _, want := range []string{"Batch Copy Results", "Total:        4", "Completed:    3", "Failed:       1", "Duration:     8s"} {
		if !strings.Contains(out, want) {
			t.Fatalf("printBatchResults output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintEnhancedCopyResult(t *testing.T) {
	result := &operations.CopyResult{
		Source: "src-bucket/a.txt", Destination: "dst-bucket/a.txt",
		Size: 2048, Duration: 500 * time.Millisecond, Speed: 4.0,
		Verified: true, Resumed: true, Success: true,
	}
	out := capturePrint(t, func() { printEnhancedCopyResult(result) })
	for _, want := range []string{"src-bucket/a.txt", "dst-bucket/a.txt", "true", "Yes"} {
		if !strings.Contains(out, want) {
			t.Fatalf("printEnhancedCopyResult output missing %q:\n%s", want, out)
		}
	}
}

func TestDemoSuccess(t *testing.T) {
	out := capturePrint(t, demoSuccess)
	for _, want := range []string{"Operation completed successfully!", "All files processed!", "Deployment ready!"} {
		if !strings.Contains(out, want) {
			t.Fatalf("demoSuccess output missing %q:\n%s", want, out)
		}
	}
}
