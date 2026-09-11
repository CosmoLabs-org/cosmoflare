package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// d1ImportWithStdin replaces os.Stdin with a pipe pre-loaded with input and
// restores it when the test ends, mirroring configRunWithStdin
// (cmd/config_run_test.go) for commands that read a confirmation response.
func d1ImportWithStdin(t *testing.T, input string) {
	t.Helper()
	saved := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("writing stdin fixture failed: %v", err)
	}
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = saved })
}

// d1ImportWriteTestFile writes a small valid SQL dump to dir and returns its
// path.
func d1ImportWriteTestFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "dump.sql")
	sql := "CREATE TABLE t (id INTEGER);\nINSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n"
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatalf("failed to write test SQL file: %v", err)
	}
	return path
}

// d1ImportTestEnv sets DryRun/JSONOutput/AccountID/APIToken/flag vars for a
// test and restores them on cleanup.
func d1ImportTestEnv(t *testing.T) {
	t.Helper()
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origFile := d1ImportFile
	origLocal := d1ImportLocal
	origForce := d1ImportForce
	origBatchSize := d1ImportBatchSize
	t.Cleanup(func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		d1ImportFile = origFile
		d1ImportLocal = origLocal
		d1ImportForce = origForce
		d1ImportBatchSize = origBatchSize
	})

	AccountID = "test-account"
	APIToken = "test-token"
	d1ImportForce = false
	d1ImportLocal = false
	d1ImportBatchSize = 40960
}

// --- Registration ---

func TestD1ImportCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range d1Cmd.Commands() {
		if sub.Name() == "import" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("d1ImportCmd not registered under d1Cmd")
	}
}

func TestD1ImportCmd_Metadata(t *testing.T) {
	if d1ImportCmd.RunE == nil {
		t.Error("d1ImportCmd has nil RunE")
	}
	if d1ImportCmd.Long == "" {
		t.Error("d1ImportCmd.Long is empty")
	}
	if !strings.Contains(d1ImportCmd.Long, "Examples:") {
		t.Error("d1ImportCmd.Long should contain an Examples section")
	}
	if d1ImportCmd.Args == nil {
		t.Error("d1ImportCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestD1ImportCmd_Flags(t *testing.T) {
	expected := []string{"file", "local", "remote", "force", "batch-size"}
	for _, name := range expected {
		if d1ImportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on d1ImportCmd", name)
		}
	}
}

// --- Validation ---

func TestD1Import_MissingFile(t *testing.T) {
	d1ImportTestEnv(t)
	d1ImportFile = ""

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})
	if err == nil {
		t.Fatal("expected error when --file flag is empty")
	}
	if !strings.Contains(err.Error(), "file") {
		t.Errorf("error = %q, want it to mention 'file'", err.Error())
	}
}

func TestD1Import_UnreadableFile(t *testing.T) {
	d1ImportTestEnv(t)
	d1ImportFile = filepath.Join(t.TempDir(), "does-not-exist.sql")

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})
	if err == nil {
		t.Fatal("expected error when --file does not exist")
	}
}

// --- Dry run ---

func TestD1Import_DryRun(t *testing.T) {
	d1ImportTestEnv(t)
	DryRun = true
	JSONOutput = false
	d1ImportFile = d1ImportWriteTestFile(t, t.TempDir())

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if err != nil {
		t.Fatalf("runD1Import(DryRun) returned error: %v", err)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("dry-run output should contain 'DRY RUN', got: %q", out)
	}
	if !strings.Contains(out, "3 statement(s)") {
		t.Errorf("dry-run output should report 3 statements, got: %q", out)
	}
	if !strings.Contains(out, "1 batch(es)") {
		t.Errorf("dry-run output should report 1 batch, got: %q", out)
	}
}

func TestD1Import_DryRunJSON(t *testing.T) {
	d1ImportTestEnv(t)
	DryRun = true
	JSONOutput = true
	d1ImportFile = d1ImportWriteTestFile(t, t.TempDir())

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if err != nil {
		t.Fatalf("runD1Import(DryRun+JSON) returned error: %v", err)
	}
	if !strings.Contains(out, `"total_statements": 3`) {
		t.Errorf("JSON dry-run output should contain total_statements: 3, got: %q", out)
	}
	if !strings.Contains(out, `"success": true`) {
		t.Errorf("JSON dry-run output should report success: true, got: %q", out)
	}
}

// --- Confirmation prompt ---

func TestD1Import_ConfirmationAbortsOnWrongName(t *testing.T) {
	d1ImportTestEnv(t)
	DryRun = false
	JSONOutput = false
	d1ImportFile = d1ImportWriteTestFile(t, t.TempDir())

	d1ImportWithStdin(t, "wrong-database-id\n")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if err != nil {
		t.Fatalf("runD1Import(wrong confirmation) returned error: %v, want nil (cancelled)", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Errorf("expected cancellation message, got: %q", out)
	}
}

func TestD1Import_ForceSkipsConfirmation(t *testing.T) {
	// --force plus --dry-run exercises the flag registration and the
	// dry-run short-circuit (which itself skips the confirmation prompt);
	// a real network-executing happy path is covered at the library level
	// (pkg/cosmoflare/d1_import_test.go) since the cmd package's
	// getD1Service has no fake-transport injection point.
	d1ImportTestEnv(t)
	d1ImportForce = true
	DryRun = true
	JSONOutput = false
	d1ImportFile = d1ImportWriteTestFile(t, t.TempDir())

	err := runD1Import(d1ImportCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})
	if err != nil {
		t.Errorf("runD1Import(--force, DryRun) returned error: %v", err)
	}
}

// --- Resume hint ---

func TestD1ImportResumeOffset_NoCacheFile(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(old) })

	if _, ok := d1ImportResumeOffset("db-1", "dump.sql"); ok {
		t.Error("expected ok=false when no progress cache file exists")
	}
}

func TestD1ImportResumeOffset_EntryWithErrors(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(old) })

	filePath := "dump.sql"
	if err := os.WriteFile(filePath, []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatalf("failed to write dump.sql: %v", err)
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	cacheJSON := `{"db-1|` + strings.ReplaceAll(abs, `\`, `\\`) + `":{"last_applied_offset":4096,"had_errors":true,"updated_at":"2026-01-01T00:00:00Z"}}`
	if err := os.WriteFile(d1ImportProgressCacheFile, []byte(cacheJSON), 0o644); err != nil {
		t.Fatalf("failed to write progress cache: %v", err)
	}

	offset, ok := d1ImportResumeOffset("db-1", filePath)
	if !ok {
		t.Fatal("expected ok=true for a cache entry with had_errors=true")
	}
	if offset != 4096 {
		t.Errorf("offset = %d, want 4096", offset)
	}
}

// --- formatCommaInt helper ---

func TestFormatCommaInt(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{12, "12"},
		{123, "123"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	}
	for _, tc := range cases {
		got := formatCommaInt(tc.input)
		if got != tc.want {
			t.Errorf("formatCommaInt(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
