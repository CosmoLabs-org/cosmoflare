package cosmoflare

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- NewAuditLogger ---

func TestNewAuditLogger_EmptyPath(t *testing.T) {
	logger, err := NewAuditLogger("")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if logger != nil {
		t.Fatal("expected nil logger for empty path")
	}
}

func TestNewAuditLogger_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "audit.log")

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

func TestNewAuditLogger_BadPath(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(blocker, "nested", "audit.log")

	logger, err := NewAuditLogger(badPath)
	if err == nil {
		if logger != nil {
			logger.Close()
		}
		t.Fatal("expected error for bad path")
	}
}

// --- Log ---

func TestAuditLogger_Log_WritesEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := AuditEntry{
		Timestamp: ts,
		Operation: "bucket.create",
		Service:   "r2",
		Resource:  "my-bucket",
		Action:    "create",
		User:      "testuser",
		Details:   map[string]string{"region": "wnam"},
		Success:   true,
	}
	if err := logger.Log(entry); err != nil {
		t.Fatal(err)
	}
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Operation != "bucket.create" {
		t.Errorf("Operation = %q, want %q", e.Operation, "bucket.create")
	}
	if e.Service != "r2" {
		t.Errorf("Service = %q, want %q", e.Service, "r2")
	}
	if e.Resource != "my-bucket" {
		t.Errorf("Resource = %q, want %q", e.Resource, "my-bucket")
	}
	if e.Action != "create" {
		t.Errorf("Action = %q, want %q", e.Action, "create")
	}
	if !e.Success {
		t.Error("expected Success=true")
	}
	if e.Details["region"] != "wnam" {
		t.Errorf("Details[region] = %q, want %q", e.Details["region"], "wnam")
	}
}

func TestAuditLogger_Log_SetsTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC()
	entry := AuditEntry{
		Operation: "test.op",
		Service:   "test",
		Resource:  "res",
		Action:    "create",
		Success:   true,
	}
	if err := logger.Log(entry); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	ts := entries[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestAuditLogger_Log_SetsUser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	entry := AuditEntry{
		Operation: "test.op",
		Service:   "test",
		Resource:  "res",
		Action:    "create",
		Success:   true,
	}
	if err := logger.Log(entry); err != nil {
		t.Fatal(err)
	}
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].User == "" {
		t.Error("expected User to be auto-populated")
	}
}

func TestAuditLogger_Log_NilLogger(t *testing.T) {
	var logger *AuditLogger
	err := logger.Log(AuditEntry{Operation: "test"})
	if err != nil {
		t.Fatalf("expected nil error from nil logger, got %v", err)
	}
}

// --- LogMutation ---

func TestAuditLogger_LogMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	logger.LogMutation("dns.create", "dns", "example.com", "create",
		map[string]string{"type": "A", "content": "1.2.3.4"}, true)
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Operation != "dns.create" {
		t.Errorf("Operation = %q, want %q", entries[0].Operation, "dns.create")
	}
	if entries[0].Details["type"] != "A" {
		t.Errorf("Details[type] = %q, want %q", entries[0].Details["type"], "A")
	}
}

// --- ReadAuditLog ---

func TestReadAuditLog_NonExistentFile(t *testing.T) {
	entries, err := ReadAuditLog("/nonexistent/path/audit.log")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %d", len(entries))
	}
}

func TestReadAuditLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	os.WriteFile(path, []byte(""), 0600)

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadAuditLog_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("op1", "svc1", "res1", "create", nil, true)
	logger.LogMutation("op2", "svc2", "res2", "delete", nil, false)
	logger.LogMutation("op3", "svc3", "res3", "update", nil, true)
	logger.Close()

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestReadAuditLog_MalformedEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	// Write a valid entry followed by garbage
	os.WriteFile(path, []byte(`{"operation":"test","success":true}`+"\n{bad json\n"), 0600)

	entries, err := ReadAuditLog(path)
	if err == nil {
		t.Fatal("expected error for malformed entry")
	}
	// Should still return the valid entries parsed so far
	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry, got %d", len(entries))
	}
}

// --- ReadAuditLogWithLimit ---

func TestReadAuditLogWithLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	for i := 0; i < 10; i++ {
		logger.LogMutation("op", "svc", "res", "create", nil, true)
	}
	logger.Close()

	entries, err := ReadAuditLogWithLimit(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestReadAuditLogWithLimit_ZeroReturnsAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	for i := 0; i < 5; i++ {
		logger.LogMutation("op", "svc", "res", "create", nil, true)
	}
	logger.Close()

	entries, err := ReadAuditLogWithLimit(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}
}

// --- ReadAuditLogSince ---

func TestReadAuditLogSince(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)

	old := AuditEntry{
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Operation: "old.op",
		Service:   "svc",
		Action:    "create",
		Success:   true,
	}
	recent := AuditEntry{
		Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Operation: "new.op",
		Service:   "svc",
		Action:    "create",
		Success:   true,
	}
	logger.Log(old)
	logger.Log(recent)
	logger.Close()

	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entries, err := ReadAuditLogSince(path, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Operation != "new.op" {
		t.Errorf("Operation = %q, want %q", entries[0].Operation, "new.op")
	}
}

// --- SearchAuditLog ---

func TestSearchAuditLog_ByQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("bucket.create", "r2", "my-bucket", "create", nil, true)
	logger.LogMutation("dns.create", "dns", "example.com", "create", nil, true)
	logger.LogMutation("worker.deploy", "workers", "my-worker", "deploy", nil, true)
	logger.Close()

	results, err := SearchAuditLog(path, "dns", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Service != "dns" {
		t.Errorf("Service = %q, want %q", results[0].Service, "dns")
	}
}

func TestSearchAuditLog_ByActionFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("bucket.create", "r2", "b1", "create", nil, true)
	logger.LogMutation("bucket.delete", "r2", "b2", "delete", nil, true)
	logger.LogMutation("dns.create", "dns", "d1", "create", nil, true)
	logger.Close()

	results, err := SearchAuditLog(path, "r2", "delete")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Resource != "b2" {
		t.Errorf("Resource = %q, want %q", results[0].Resource, "b2")
	}
}

func TestSearchAuditLog_MatchesDetails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("dns.create", "dns", "example.com", "create",
		map[string]string{"type": "CNAME", "target": "cdn.example.com"}, true)
	logger.LogMutation("dns.create", "dns", "other.com", "create",
		map[string]string{"type": "A", "target": "1.2.3.4"}, true)
	logger.Close()

	results, err := SearchAuditLog(path, "CNAME", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Resource != "example.com" {
		t.Errorf("Resource = %q, want %q", results[0].Resource, "example.com")
	}
}

func TestSearchAuditLog_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("bucket.create", "R2", "MyBucket", "create", nil, true)
	logger.Close()

	results, err := SearchAuditLog(path, "mybucket", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

// --- ExportAuditLogJSON ---

func TestExportAuditLogJSON(t *testing.T) {
	entries := []AuditEntry{
		{
			Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			Operation: "bucket.create",
			Service:   "r2",
			Resource:  "my-bucket",
			Action:    "create",
			User:      "testuser",
			Success:   true,
		},
	}

	var buf bytes.Buffer
	if err := ExportAuditLogJSON(entries, &buf); err != nil {
		t.Fatal(err)
	}

	var parsed []AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse exported JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(parsed))
	}
	if parsed[0].Operation != "bucket.create" {
		t.Errorf("Operation = %q, want %q", parsed[0].Operation, "bucket.create")
	}
}

// --- ExportAuditLogCSV ---

func TestExportAuditLogCSV(t *testing.T) {
	entries := []AuditEntry{
		{
			Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			Operation: "dns.delete",
			Service:   "dns",
			Resource:  "example.com",
			Action:    "delete",
			User:      "testuser",
			Details:   map[string]string{"record_id": "abc123"},
			Success:   false,
		},
	}

	var buf bytes.Buffer
	if err := ExportAuditLogCSV(entries, &buf); err != nil {
		t.Fatal(err)
	}

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// Header + 1 data row
	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header+data), got %d", len(records))
	}
	header := records[0]
	if header[0] != "timestamp" || header[1] != "operation" {
		t.Errorf("unexpected header: %v", header)
	}
	row := records[1]
	if row[1] != "dns.delete" {
		t.Errorf("operation = %q, want %q", row[1], "dns.delete")
	}
	if row[6] != "false" {
		t.Errorf("success = %q, want %q", row[6], "false")
	}
	if !strings.Contains(row[7], "record_id=abc123") {
		t.Errorf("details = %q, should contain %q", row[7], "record_id=abc123")
	}
}

// --- ClearAuditLog ---

func TestClearAuditLog_All(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.LogMutation("op1", "svc", "res", "create", nil, true)
	logger.LogMutation("op2", "svc", "res", "delete", nil, true)
	logger.Close()

	removed, err := ClearAuditLog(path, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}

	entries, _ := ReadAuditLog(path)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestClearAuditLog_Before(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	old := AuditEntry{
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Operation: "old.op",
		Service:   "svc",
		Action:    "create",
		Success:   true,
	}
	recent := AuditEntry{
		Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Operation: "new.op",
		Service:   "svc",
		Action:    "create",
		Success:   true,
	}
	logger.Log(old)
	logger.Log(recent)
	logger.Close()

	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	removed, err := ClearAuditLog(path, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	entries, _ := ReadAuditLog(path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Operation != "new.op" {
		t.Errorf("Operation = %q, want %q", entries[0].Operation, "new.op")
	}
}

func TestClearAuditLog_BeforeNoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	logger.Log(AuditEntry{
		Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Operation: "recent",
		Service:   "svc",
		Action:    "create",
		Success:   true,
	})
	logger.Close()

	cutoff := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	removed, err := ClearAuditLog(path, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

// --- DefaultAuditLogPath ---

func TestDefaultAuditLogPath(t *testing.T) {
	path := DefaultAuditLogPath()
	if path == "" {
		t.Skip("could not determine home dir")
	}
	if !strings.HasSuffix(path, filepath.Join(".cosmoflare", "audit.log")) {
		t.Errorf("path = %q, expected suffix .cosmoflare/audit.log", path)
	}
}

// --- Path ---

func TestAuditLogger_Path(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	logger, _ := NewAuditLogger(path)
	defer logger.Close()

	if logger.Path() != path {
		t.Errorf("Path() = %q, want %q", logger.Path(), path)
	}
}

func TestAuditLogger_Path_Nil(t *testing.T) {
	var logger *AuditLogger
	if logger.Path() != "" {
		t.Errorf("Path() on nil logger = %q, want empty", logger.Path())
	}
}

// --- Close ---

func TestAuditLogger_Close_Nil(t *testing.T) {
	var logger *AuditLogger
	if err := logger.Close(); err != nil {
		t.Fatalf("expected nil error from nil logger close, got %v", err)
	}
}

// --- Concurrent writes ---

func TestAuditLogger_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger, _ := NewAuditLogger(path)
	defer logger.Close()

	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			logger.LogMutation("concurrent.op", "svc", "res", "create", nil, true)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}

	entries, err := ReadAuditLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 20 {
		t.Errorf("expected 20 entries, got %d", len(entries))
	}
}
