package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// d1PushSnapshot snapshots the push-sql globals (flags, executor seam,
// backoff sleep) and restores them on cleanup, keeping tests independent.
func d1PushSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	origDB, origConc := d1PushDatabase, d1PushConcurrency
	origBytes, origFrom := d1PushBatchBytes, d1PushFromBatch
	origQuery, origSleep := d1PushQueryBatch, d1PushSleep
	t.Cleanup(func() {
		d1PushDatabase, d1PushConcurrency = origDB, origConc
		d1PushBatchBytes, d1PushFromBatch = origBytes, origFrom
		d1PushQueryBatch, d1PushSleep = origQuery, origSleep
	})
	d1PushDatabase, d1PushConcurrency = "", d1PushDefaultConcurrency
	d1PushBatchBytes, d1PushFromBatch = d1PushDefaultBatchBytes, 1
	d1PushSleep = func(time.Duration) {}
	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		return nil
	}
}

// d1PushWriteFile writes content to a scratch SQL file and returns its path.
func d1PushWriteFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "seed.sql")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	return path
}

// TestPackSQLBatches verifies byte-capped packing: statements accumulate
// until the cap, and an oversized statement becomes a lone batch.
func TestPackSQLBatches(t *testing.T) {
	stmts := []string{
		"INSERT INTO t VALUES (1)",
		"INSERT INTO t VALUES (2)",
		strings.Repeat("X", 500),
	}

	t.Run("packs to the cap", func(t *testing.T) {
		got := packSQLBatches(stmts[:2], 50)
		if len(got) != 2 {
			t.Fatalf("two statements whose joined size (52 bytes) exceeds a 50-byte cap must not share a batch, got %d", len(got))
		}
	})

	t.Run("fills up to the cap", func(t *testing.T) {
		got := packSQLBatches(stmts[:2], 100)
		if len(got) != 1 || len(got[0]) != 2 {
			t.Fatalf("both statements fit under 100 bytes, got %v", got)
		}
	})

	t.Run("oversized statement is a lone batch", func(t *testing.T) {
		got := packSQLBatches(stmts, 60)
		if len(got) != 2 {
			t.Fatalf("expected 2 batches, got %v", got)
		}
		if len(got[1]) != 1 || got[1][0] != stmts[2] {
			t.Fatalf("oversized statement should be its own batch, got %v", got[1])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if got := packSQLBatches(nil, 100); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}

// TestD1PushIsTransientAndTooBig verifies the error classifiers used to
// route failures between retry, split, and fail-fast.
func TestD1PushIsTransientAndTooBig(t *testing.T) {
	transient := []string{
		"D1_ERROR: internal error code: 7500",
		"POST https://api: 502 Bad Gateway",
		"Service Unavailable",
		"context deadline exceeded (Client.Timeout exceeded)",
		"read tcp: connection reset by peer",
	}
	for _, msg := range transient {
		if !d1PushIsTransient(errors.New(msg)) {
			t.Errorf("d1PushIsTransient(%q) = false, want true", msg)
		}
	}
	if d1PushIsTransient(errors.New("no such table: users")) {
		t.Error("plain SQL errors must not be transient")
	}

	tooBig := []string{"D1_ERROR: SQLITE_TOOBIG", "query is too big to run", "request body too large"}
	for _, msg := range tooBig {
		if !d1PushIsTooBig(errors.New(msg)) {
			t.Errorf("d1PushIsTooBig(%q) = false, want true", msg)
		}
		if d1PushIsTransient(errors.New(msg)) {
			t.Errorf("TOOBIG error %q must not be classified transient", msg)
		}
	}
	if d1PushIsTooBig(errors.New("no such table: users")) {
		t.Error("plain SQL errors must not be TOOBIG")
	}
	if !d1PushIsTooBig(fmt.Errorf("push: %w", errors.New("SQLITE_TOOBIG"))) {
		t.Error("TOOBIG must be detected through the wrap chain")
	}
}

// TestD1PushExecBatch_RetriesTransient verifies a 7500 error followed by
// success costs exactly one retry and the batch eventually applies.
func TestD1PushExecBatch_RetriesTransient(t *testing.T) {
	d1PushSnapshot(t)

	var calls int
	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		calls++
		if calls == 1 {
			return errors.New("D1_ERROR: internal error code: 7500")
		}
		return nil
	}

	batch := []string{"INSERT OR IGNORE INTO t VALUES (1)", "INSERT OR IGNORE INTO t VALUES (2)"}
	if err := d1PushExecBatch(context.Background(), nil, "db", batch, 0); err != nil {
		t.Fatalf("batch should succeed after retry, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (1 fail + 1 retry), got %d", calls)
	}
}

// TestD1PushExecBatch_SplitsTooBig verifies a TOOBIG-looking failure splits
// the batch in half and both halves execute.
func TestD1PushExecBatch_SplitsTooBig(t *testing.T) {
	d1PushSnapshot(t)

	var mu sync.Mutex
	var sizes []int
	d1PushQueryBatch = func(_ context.Context, _ *cosmoflare.D1Service, _, sql string) error {
		mu.Lock()
		defer mu.Unlock()
		n := strings.Count(sql, ";\n") + 1
		sizes = append(sizes, n)
		if n > 2 {
			return errors.New("D1_ERROR: SQLITE_TOOBIG")
		}
		return nil
	}

	batch := []string{"S1", "S2", "S3", "S4"}
	if err := d1PushExecBatch(context.Background(), nil, "db", batch, 0); err != nil {
		t.Fatalf("split recovery should succeed, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sizes) != 3 { // whole batch fails, then two halves succeed
		t.Fatalf("expected 3 executions (4-stmt batch + 2 halves), got %v", sizes)
	}
	for _, n := range sizes[1:] {
		if n != 2 {
			t.Errorf("halves must carry 2 statements each, got sizes %v", sizes)
		}
	}
}

// TestD1PushExecBatch_NonTransientFailsFast verifies plain SQL errors are
// returned immediately without retry or split.
func TestD1PushExecBatch_NonTransientFailsFast(t *testing.T) {
	d1PushSnapshot(t)

	var calls int
	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		calls++
		return errors.New("no such table: users")
	}

	err := d1PushExecBatch(context.Background(), nil, "db", []string{"INSERT INTO users VALUES (1)"}, 0)
	if err == nil || !strings.Contains(err.Error(), "no such table") {
		t.Fatalf("expected immediate SQL error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("non-transient errors must not be retried, got %d calls", calls)
	}
}

// TestRunD1PushSQL_Resume verifies --from-batch skips earlier batches: only
// the remaining batches hit the query seam.
func TestRunD1PushSQL_Resume(t *testing.T) {
	d1PushSnapshot(t)
	AccountID, APIToken = "acct", "tok"
	d1PushDatabase = "db-id"
	d1PushConcurrency = 1
	d1PushBatchBytes = 10 // one statement per batch
	d1PushFromBatch = 2

	var mu sync.Mutex
	var sent []string
	d1PushQueryBatch = func(_ context.Context, _ *cosmoflare.D1Service, _, sql string) error {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, sql)
		return nil
	}

	file := d1PushWriteFile(t, "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2); INSERT INTO t VALUES (3)")

	out := capturePrint(t, func() {
		if err := runD1PushSQL(nil, []string{file}); err != nil {
			t.Errorf("push should succeed, got %v", err)
		}
	})

	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 2 {
		t.Fatalf("resume from batch 2 of 3 should execute 2 batches, sent %v", sent)
	}
	if !strings.Contains(out, "2 batch(es)") && !strings.Contains(out, "3 batch(es)") {
		t.Errorf("summary should report batch counts, got %q", out)
	}
}

// TestRunD1PushSQL_FailurePrintsResumeHint verifies a failing batch reports
// the failed batch index and a --from-batch resume hint.
func TestRunD1PushSQL_FailurePrintsResumeHint(t *testing.T) {
	d1PushSnapshot(t)
	AccountID, APIToken = "acct", "tok"
	d1PushDatabase = "db-id"
	d1PushConcurrency = 1
	d1PushBatchBytes = 10

	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		return errors.New("no such table: users")
	}

	file := d1PushWriteFile(t, "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2)")

	var runErr error
	out := capturePrint(t, func() {
		runErr = runD1PushSQL(nil, []string{file})
	})

	if runErr == nil {
		t.Fatal("failing push must return an error")
	}
	if !strings.Contains(runErr.Error(), "resume with --from-batch 1") {
		t.Errorf("error should carry resume hint, got %q", runErr.Error())
	}
	if !strings.Contains(out, "resume with --from-batch 1") {
		t.Errorf("output should print resume hint, got %q", out)
	}
	if !strings.Contains(out, "batch 1/2 failed") {
		t.Errorf("output should report the failed batch, got %q", out)
	}
}

// TestRunD1PushSQL_JSONEnvelope verifies --json prints one final envelope
// describing the push.
func TestRunD1PushSQL_JSONEnvelope(t *testing.T) {
	d1PushSnapshot(t)
	JSONOutput = true
	AccountID, APIToken = "acct", "tok"
	d1PushDatabase = "db-id"
	d1PushConcurrency = 1
	d1PushBatchBytes = 10

	var mu sync.Mutex
	var batches int
	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		mu.Lock()
		defer mu.Unlock()
		batches++
		return nil
	}

	file := d1PushWriteFile(t, "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2)")

	out := capturePrint(t, func() {
		if err := runD1PushSQL(nil, []string{file}); err != nil {
			t.Errorf("push should succeed, got %v", err)
		}
	})

	mu.Lock()
	defer mu.Unlock()
	if batches != 2 {
		t.Fatalf("expected 2 batch executions, got %d", batches)
	}

	var envelope struct {
		Success         bool   `json:"success"`
		DatabaseID      string `json:"database_id"`
		TotalStatements int    `json:"total_statements"`
		TotalBatches    int    `json:"total_batches"`
		BatchesExecuted int    `json:"batches_executed"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("output is not one JSON envelope: %v\n%s", err, out)
	}
	if !envelope.Success || envelope.DatabaseID != "db-id" || envelope.TotalStatements != 2 || envelope.TotalBatches != 2 || envelope.BatchesExecuted != 2 {
		t.Errorf("envelope mismatch: %+v", envelope)
	}
}

// TestRunD1PushSQL_Guards verifies the missing-file and missing-credentials
// errors fire before any query is issued.
func TestRunD1PushSQL_Guards(t *testing.T) {
	d1PushSnapshot(t)
	d1PushDatabase = "db-id"

	t.Run("missing file", func(t *testing.T) {
		err := runD1PushSQL(nil, []string{filepath.Join(t.TempDir(), "nope.sql")})
		if err == nil || !strings.Contains(err.Error(), "cannot read file") {
			t.Errorf("expected missing-file error, got %v", err)
		}
	})

	t.Run("missing --database", func(t *testing.T) {
		d1PushDatabase = ""
		file := d1PushWriteFile(t, "SELECT 1")
		err := runD1PushSQL(nil, []string{file})
		if err == nil || !strings.Contains(err.Error(), "--database") {
			t.Errorf("expected --database requirement error, got %v", err)
		}
		d1PushDatabase = "db-id"
	})

	t.Run("missing credentials", func(t *testing.T) {
		file := d1PushWriteFile(t, "SELECT 1")
		AccountID, APIToken = "", ""
		err := runD1PushSQL(nil, []string{file})
		if err == nil || !strings.Contains(err.Error(), "failed to create D1 service") {
			t.Errorf("expected service error, got %v", err)
		}
	})
}

// TestRunD1PushSQL_ProgressLines verifies human mode prints per-batch
// progress lines to stderr.
func TestRunD1PushSQL_ProgressLines(t *testing.T) {
	d1PushSnapshot(t)
	AccountID, APIToken = "acct", "tok"
	d1PushDatabase = "db-id"
	d1PushConcurrency = 1
	d1PushBatchBytes = 10

	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		return nil
	}

	file := d1PushWriteFile(t, "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2)")

	var sb strings.Builder
	done := make(chan struct{})
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				close(done)
				return
			}
		}
	}()
	_ = capturePrint(t, func() {
		if err := runD1PushSQL(nil, []string{file}); err != nil {
			t.Errorf("push should succeed, got %v", err)
		}
	})
	os.Stderr = oldStderr
	_ = w.Close()
	<-done

	progress := sb.String()
	if !strings.Contains(progress, "batch 1/2 ok (1 statements)") || !strings.Contains(progress, "batch 2/2 ok (1 statements)") {
		t.Errorf("expected progress lines on stderr, got %q", progress)
	}
}

// TestRunD1PushSQL_DryRun verifies dry-run reports the plan without any
// query execution.
func TestRunD1PushSQL_DryRun(t *testing.T) {
	d1PushSnapshot(t)
	DryRun = true
	AccountID, APIToken = "acct", "tok"
	d1PushDatabase = "db-id"
	d1PushBatchBytes = 10

	called := false
	d1PushQueryBatch = func(context.Context, *cosmoflare.D1Service, string, string) error {
		called = true
		return nil
	}

	file := d1PushWriteFile(t, "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2)")

	out := capturePrint(t, func() {
		if err := runD1PushSQL(nil, []string{file}); err != nil {
			t.Errorf("dry-run should exit cleanly, got %v", err)
		}
	})

	if called {
		t.Error("dry-run must not execute any query")
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("expected dry-run notice, got %q", out)
	}
}
