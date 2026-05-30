package cosmoflare

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewAuditLogger_EmptyPath verifies that an empty path returns nil, nil
// (audit logging disabled).
func TestNewAuditLogger_EmptyPath(t *testing.T) {
	logger, err := NewAuditLogger("")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if logger != nil {
		t.Fatal("expected nil logger for empty path")
	}
}

// TestNewAuditLogger_CreatesFileAndDir verifies that a non-empty path causes
// the directory and file to be created.
func TestNewAuditLogger_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "audit.jsonl")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	defer logger.Close()

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected audit file to exist at %s: %v", path, statErr)
	}
}

// TestNewAuditLogger_BadPath verifies that a path whose parent directory
// cannot be created returns an error (using a file as a parent).
func TestNewAuditLogger_BadPath(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file, then try to use it as a directory component.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(blocker, "nested", "audit.jsonl")

	logger, err := NewAuditLogger(badPath)
	if err == nil {
		if logger != nil {
			logger.Close()
		}
		t.Fatal("expected error for bad path, got nil")
	}
	if logger != nil {
		t.Fatal("expected nil logger on error")
	}
}

// TestLog_NilReceiver ensures Log on a nil *AuditLogger returns nil without panicking.
func TestLog_NilReceiver(t *testing.T) {
	var l *AuditLogger
	if err := l.Log(AuditEntry{Action: "test"}); err != nil {
		t.Fatalf("expected nil from nil receiver, got %v", err)
	}
}

// TestLog_SetsTimestampIfZero verifies that a zero Timestamp is replaced with
// the current UTC time before writing.
func TestLog_SetsTimestampIfZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	before := time.Now().UTC().Add(-time.Second)

	entry := AuditEntry{Action: "read", Result: "success"}
	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	after := time.Now().UTC().Add(time.Second)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	ts := entries[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v is outside expected range [%v, %v]", ts, before, after)
	}
}

// TestLog_PreservesExplicitTimestamp ensures that a non-zero Timestamp is not
// overwritten.
func TestLog_PreservesExplicitTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	fixed := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	entry := AuditEntry{Timestamp: fixed, Action: "list", Result: "success"}
	if err := logger.Log(entry); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].Timestamp.Equal(fixed) {
		t.Errorf("expected timestamp %v, got %v", fixed, entries[0].Timestamp)
	}
}

// TestLog_WritesJSONL verifies that each Log call produces a valid JSON object
// on its own line.
func TestLog_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	entries := []AuditEntry{
		{Action: "upload", Bucket: "b1", Key: "k1", Size: 100, Result: "success"},
		{Action: "delete", Bucket: "b2", Key: "k2", Result: "failure", Error: "not found"},
	}
	for _, e := range entries {
		if err := logger.Log(e); err != nil {
			t.Fatalf("Log error: %v", err)
		}
	}
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}
	for i, line := range lines {
		var got AuditEntry
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		if got.Action != entries[i].Action {
			t.Errorf("line %d: expected action %q, got %q", i, entries[i].Action, got.Action)
		}
	}
}

// TestLogUpload_NoError verifies LogUpload writes the correct fields when
// no error is provided.
func TestLogUpload_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.LogUpload("my-bucket", "path/to/file.txt", 1024, "success", nil)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Action != "upload" {
		t.Errorf("expected action 'upload', got %q", e.Action)
	}
	if e.Bucket != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got %q", e.Bucket)
	}
	if e.Key != "path/to/file.txt" {
		t.Errorf("expected key 'path/to/file.txt', got %q", e.Key)
	}
	if e.Size != 1024 {
		t.Errorf("expected size 1024, got %d", e.Size)
	}
	if e.Result != "success" {
		t.Errorf("expected result 'success', got %q", e.Result)
	}
	if e.Error != "" {
		t.Errorf("expected empty error field, got %q", e.Error)
	}
}

// TestLogUpload_WithError verifies LogUpload records the error message when an
// error is supplied.
func TestLogUpload_WithError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	uploadErr := errors.New("upload failed: quota exceeded")
	logger.LogUpload("bucket", "key", 512, "failure", uploadErr)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Error != uploadErr.Error() {
		t.Errorf("expected error %q, got %q", uploadErr.Error(), entries[0].Error)
	}
	if entries[0].Result != "failure" {
		t.Errorf("expected result 'failure', got %q", entries[0].Result)
	}
}

// TestLogUpload_NilLogger ensures LogUpload on a nil receiver does not panic.
func TestLogUpload_NilLogger(t *testing.T) {
	var l *AuditLogger
	// Should not panic.
	l.LogUpload("b", "k", 0, "success", nil)
}

// TestLogDelete_NoError verifies LogDelete writes the correct fields when no
// error is provided.
func TestLogDelete_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.LogDelete("del-bucket", "del/key.bin", "success", nil)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Action != "delete" {
		t.Errorf("expected action 'delete', got %q", e.Action)
	}
	if e.Bucket != "del-bucket" {
		t.Errorf("expected bucket 'del-bucket', got %q", e.Bucket)
	}
	if e.Key != "del/key.bin" {
		t.Errorf("expected key 'del/key.bin', got %q", e.Key)
	}
	if e.Result != "success" {
		t.Errorf("expected result 'success', got %q", e.Result)
	}
	if e.Error != "" {
		t.Errorf("expected empty error, got %q", e.Error)
	}
}

// TestLogDelete_WithError verifies LogDelete records the error message.
func TestLogDelete_WithError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	delErr := errors.New("object not found")
	logger.LogDelete("bucket", "missing.txt", "failure", delErr)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Error != delErr.Error() {
		t.Errorf("expected error %q, got %q", delErr.Error(), entries[0].Error)
	}
}

// TestLogDelete_NilLogger ensures LogDelete on a nil receiver does not panic.
func TestLogDelete_NilLogger(t *testing.T) {
	var l *AuditLogger
	l.LogDelete("b", "k", "success", nil)
}

// TestClose_NilReceiver verifies that Close on a nil *AuditLogger returns nil.
func TestClose_NilReceiver(t *testing.T) {
	var l *AuditLogger
	if err := l.Close(); err != nil {
		t.Fatalf("expected nil from nil receiver, got %v", err)
	}
}

// TestClose_NilFile verifies that Close on a logger with a nil file field
// returns nil without panicking.
func TestClose_NilFile(t *testing.T) {
	l := &AuditLogger{file: nil}
	if err := l.Close(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// TestClose_ClosesFile verifies that a normal Close succeeds and subsequent
// writes fail (file is actually closed).
func TestClose_ClosesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	// Writing after close should fail.
	writeErr := logger.Log(AuditEntry{Action: "post-close"})
	if writeErr == nil {
		t.Error("expected error writing to closed file, got nil")
	}
}

// TestReadAuditLog_ReadsEntries verifies that ReadAuditLog correctly reads
// back all entries written by an AuditLogger.
func TestReadAuditLog_ReadsEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []AuditEntry{
		{Action: "upload", Bucket: "b1", Key: "k1", Size: 42, Result: "success"},
		{Action: "delete", Bucket: "b2", Key: "k2", Result: "failure", Error: "gone"},
		{Action: "list", Result: "success"},
	}
	for _, e := range want {
		if err := logger.Log(e); err != nil {
			t.Fatal(err)
		}
	}
	logger.Close()

	got, err := ReadAuditLog(path)
	if err != nil {
		t.Fatalf("ReadAuditLog error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d entries, got %d", len(want), len(got))
	}
	for i, g := range got {
		w := want[i]
		if g.Action != w.Action {
			t.Errorf("[%d] action: want %q, got %q", i, w.Action, g.Action)
		}
		if g.Bucket != w.Bucket {
			t.Errorf("[%d] bucket: want %q, got %q", i, w.Bucket, g.Bucket)
		}
		if g.Key != w.Key {
			t.Errorf("[%d] key: want %q, got %q", i, w.Key, g.Key)
		}
		if g.Size != w.Size {
			t.Errorf("[%d] size: want %d, got %d", i, w.Size, g.Size)
		}
		if g.Result != w.Result {
			t.Errorf("[%d] result: want %q, got %q", i, w.Result, g.Result)
		}
		if g.Error != w.Error {
			t.Errorf("[%d] error: want %q, got %q", i, w.Error, g.Error)
		}
	}
}

// TestReadAuditLog_EmptyFile verifies that an empty file returns an empty
// slice without error.
func TestReadAuditLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// TestReadAuditLog_MissingFile verifies that a missing file returns an error.
func TestReadAuditLog_MissingFile(t *testing.T) {
	_, err := ReadAuditLog("/nonexistent/path/audit.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestReadAuditLog_MalformedEntry verifies that a JSONL file containing a
// malformed line returns an error (and any entries decoded before the bad line).
func TestReadAuditLog_MalformedEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.jsonl")

	good := AuditEntry{Action: "upload", Result: "success", Timestamp: time.Now().UTC()}
	goodBytes, _ := json.Marshal(good)

	content := string(goodBytes) + "\n" + "this is not json\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAuditLog(path)
	if err == nil {
		t.Fatal("expected error for malformed entry, got nil")
	}
	if !strings.Contains(err.Error(), "malformed audit entry") {
		t.Errorf("expected 'malformed audit entry' in error, got: %v", err)
	}
	// Entries decoded before the error should be returned.
	if len(entries) != 1 {
		t.Errorf("expected 1 partial entry before error, got %d", len(entries))
	}
}

// TestReadAuditLog_HandlesEOFCleanly verifies that a well-formed file with a
// trailing newline does not return an EOF error.
func TestReadAuditLog_HandlesEOFCleanly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.Log(AuditEntry{Action: "upload", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatalf("unexpected EOF error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

// TestAuditLogger_AppendMode verifies that re-opening the same path appends
// rather than truncates.
func TestAuditLogger_AppendMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	l1, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	l1.Log(AuditEntry{Action: "first", Result: "success"})
	l1.Close()

	l2, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	l2.Log(AuditEntry{Action: "second", Result: "success"})
	l2.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after append, got %d", len(entries))
	}
	if entries[0].Action != "first" || entries[1].Action != "second" {
		t.Errorf("unexpected entry order: %v", entries)
	}
}
