package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func d1ImportWriteTestFile(t *testing.T, dir string, sql string) string {
	t.Helper()
	path := filepath.Join(dir, "dump.sql")
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatalf("failed to write test SQL file: %v", err)
	}
	return path
}

func d1QuerySuccessResponse() map[string]interface{} {
	success := true
	changedDB := true
	return map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result": []map[string]interface{}{
			{
				"results": []map[string]interface{}{},
				"success": &success,
				"meta": map[string]interface{}{
					"changed_db":   &changedDB,
					"changes":      1,
					"duration":     0.1,
					"last_row_id":  0,
					"rows_read":    0,
					"rows_written": 1,
					"size_after":   1024,
				},
			},
		},
	}
}

func d1QueryErrorResponse(status int, message string, code int) (int, map[string]interface{}) {
	errItem := map[string]interface{}{"message": message}
	if code != 0 {
		errItem["code"] = code
	}
	return status, map[string]interface{}{
		"success": false,
		"errors":  []interface{}{errItem},
	}
}

func TestImportValidation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Import(context.Background(), "", "dump.sql", ImportOptions{})
	if err == nil {
		t.Error("expected error when databaseID is empty")
	}

	_, err = svc.Import(context.Background(), "db-123", "", ImportOptions{})
	if err == nil {
		t.Error("expected error when file path is empty")
	}

	_, err = svc.Import(context.Background(), "db-123", "/nonexistent/path/dump.sql", ImportOptions{})
	if err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestImportDryRun(t *testing.T) {
	dir := t.TempDir()
	path := d1ImportWriteTestFile(t, dir, "CREATE TABLE t (id INTEGER);\nINSERT INTO t VALUES (1);\n")

	var calls int
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls++
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	result, err := svc.Import(context.Background(), "db-123", path, ImportOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Errorf("expected no queries executed on dry run, got %d", calls)
	}
	if result.TotalStatements != 2 {
		t.Errorf("expected 2 statements, got %d", result.TotalStatements)
	}
	if !result.Success {
		t.Errorf("expected Success=true for dry run, got false: errors=%v", result.Errors)
	}
}

func TestImportSuccessMultiBatch(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "INSERT INTO t VALUES (%d);\n", i)
	}
	path := d1ImportWriteTestFile(t, dir, sb.String())
	restore := d1ChdirForTest(t, dir)
	defer restore()

	var calls int
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls++
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	var progressCalls int
	result, err := svc.Import(context.Background(), "db-123", path, ImportOptions{
		BatchSize: 50, // force multiple small batches
		Progress: func(batchIndex, totalBatches, stmtCount int, bytesProcessed int64) {
			progressCalls++
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalStatements != 20 {
		t.Errorf("expected 20 statements, got %d", result.TotalStatements)
	}
	if result.TotalBatches < 2 {
		t.Errorf("expected multiple batches with small BatchSize, got %d", result.TotalBatches)
	}
	if result.BatchesApplied != result.TotalBatches {
		t.Errorf("expected all %d batches applied, got %d", result.TotalBatches, result.BatchesApplied)
	}
	if calls != result.TotalBatches {
		t.Errorf("expected %d query calls, got %d", result.TotalBatches, calls)
	}
	if progressCalls != result.TotalBatches {
		t.Errorf("expected %d progress callbacks, got %d", result.TotalBatches, progressCalls)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false: errors=%v", result.Errors)
	}
}

func TestImportTOOBIGSplitRetry(t *testing.T) {
	dir := t.TempDir()
	path := d1ImportWriteTestFile(t, dir, "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n")
	restore := d1ChdirForTest(t, dir)
	defer restore()

	var calls []string
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		var req struct {
			SQL string `json:"sql"`
		}
		_ = json.Unmarshal(body, &req)
		calls = append(calls, req.SQL)

		if len(calls) == 1 {
			// First call is the full 2-statement batch: too big, forcing a split.
			status, resp := d1QueryErrorResponse(http.StatusBadRequest, "query is TOOBIG to execute", 0)
			w.WriteHeader(status)
			d1WriteJSON(w, resp)
			return
		}
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	svc.sleepFn = noSleep(&[]time.Duration{})

	result, err := svc.Import(context.Background(), "db-123", path, ImportOptions{BatchSize: 1_000_000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls (1 failed + 2 split halves), got %d: %v", len(calls), calls)
	}
	if result.BatchesApplied != 1 {
		t.Errorf("expected the 1 outer batch applied (via internal split-retry), got %d", result.BatchesApplied)
	}
	if !result.Success {
		t.Errorf("expected Success=true after successful split-retry, got false: errors=%v", result.Errors)
	}
}

func TestImport7500BackoffThenSplit(t *testing.T) {
	dir := t.TempDir()
	path := d1ImportWriteTestFile(t, dir, "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n")

	var callCount int
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 4 {
			// Original batch fails with 7500 on the initial attempt plus 3
			// backoff retries (4 total), then falls back to split.
			status, resp := d1QueryErrorResponse(http.StatusBadRequest, "query too big", 7500)
			w.WriteHeader(status)
			d1WriteJSON(w, resp)
			return
		}
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	var sleeps []time.Duration
	svc.sleepFn = noSleep(&sleeps)

	result, err := svc.Import(context.Background(), "db-123", path, ImportOptions{BatchSize: 1_000_000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sleeps) == 0 {
		t.Error("expected backoff sleeps before falling back to split retry on error 7500")
	}
	if result.BatchesApplied != 1 {
		t.Errorf("expected the 1 outer batch applied (via backoff+split), got %d", result.BatchesApplied)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false: errors=%v", result.Errors)
	}
}

func TestImportOtherErrorRecordsAndContinues(t *testing.T) {
	dir := t.TempDir()
	path := d1ImportWriteTestFile(t, dir, "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n")

	var calls int
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			status, resp := d1QueryErrorResponse(http.StatusBadRequest, "syntax error near VALUES", 0)
			w.WriteHeader(status)
			d1WriteJSON(w, resp)
			return
		}
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	// Force one batch per statement so the failure is isolated to statement 1.
	result, err := svc.Import(context.Background(), "db-123", path, ImportOptions{BatchSize: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 recorded error, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.BatchesApplied != 1 {
		t.Errorf("expected 1 of 2 batches applied, got %d", result.BatchesApplied)
	}
	if result.Success {
		t.Error("expected Success=false when a batch error was recorded")
	}
}

func TestImportResumeHintOnRerunAfterErrors(t *testing.T) {
	dir := t.TempDir()
	path := d1ImportWriteTestFile(t, dir, "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n")
	cacheFile := filepath.Join(dir, d1ImportProgressCacheFile)

	callFails := true
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		if callFails {
			status, resp := d1QueryErrorResponse(http.StatusBadRequest, "syntax error", 0)
			w.WriteHeader(status)
			d1WriteJSON(w, resp)
			return
		}
		d1WriteJSON(w, d1QuerySuccessResponse())
	})
	defer server.Close()

	restore := d1ChdirForTest(t, dir)
	defer restore()

	// First run: batch 1 fails, batch 2 succeeds. Progress cache records it.
	result1, err := svc.Import(context.Background(), "db-resume", path, ImportOptions{BatchSize: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1.Success {
		t.Fatal("expected first run to have errors")
	}
	if _, err := os.Stat(cacheFile); err != nil {
		t.Fatalf("expected progress cache file to be written: %v", err)
	}

	// Second run against the same file+database: expect a resume hint
	// referencing the previous run's state.
	callFails = false
	result2, err := svc.Import(context.Background(), "db-resume", path, ImportOptions{BatchSize: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ResumeHint == "" {
		t.Error("expected a resume hint on re-run after a prior run left errors")
	}
}

// d1ChdirForTest changes the working directory for the duration of a test
// (the D1 import progress cache is written relative to cwd, like the
// time-travel quota cache) and returns a func to restore it.
func d1ChdirForTest(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	return func() {
		_ = os.Chdir(orig)
	}
}
