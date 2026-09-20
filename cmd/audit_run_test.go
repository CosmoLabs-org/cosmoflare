package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// auditRunEnv isolates the audit runners against a temp HOME and snapshots
// the audit flag variables plus output-mode globals.
func auditRunEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	runGlobalsSnapshot(t)

	oldLimit, oldSince := auditLimit, auditSince
	oldType, oldForce, oldBefore := auditType, auditForce, auditBefore
	t.Cleanup(func() {
		auditLimit, auditSince = oldLimit, oldSince
		auditType, auditForce, auditBefore = oldType, oldForce, oldBefore
	})
	auditLimit, auditSince = 50, ""
	auditType, auditForce, auditBefore = "", false, ""
	return os.Getenv("HOME")
}

// auditRunPath returns the audit log path inside the isolated HOME.
func auditRunPath(t *testing.T) string {
	t.Helper()
	home := os.Getenv("HOME")
	if home == "" {
		t.Fatal("HOME not set; call auditRunEnv first")
	}
	return filepath.Join(home, ".cosmoflare", "audit.log")
}

// auditRunSeed writes NDJSON entries to the isolated audit log and returns
// the log path.
func auditRunSeed(t *testing.T, entries ...cosmoflare.AuditEntry) string {
	t.Helper()
	path := auditRunPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir audit dir: %v", err)
	}
	var b strings.Builder
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal entry: %v", err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		t.Fatalf("write audit log: %v", err)
	}
	return path
}

// auditRunEntry builds a seeded entry at the given RFC3339 time.
func auditRunEntry(ts, resource string) cosmoflare.AuditEntry {
	when, _ := time.Parse(time.RFC3339, ts)
	return cosmoflare.AuditEntry{
		Timestamp: when,
		Operation: "cosmoflare",
		Service:   "dns",
		Resource:  resource,
		Action:    "create",
		User:      "tester",
		Success:   true,
	}
}

func TestRunAuditLog_EmptyLog(t *testing.T) {
	auditRunEnv(t)

	if err := runAuditLog(auditLogCmd, nil); err != nil {
		t.Fatalf("empty log should succeed: %v", err)
	}
}

func TestRunAuditLog_LimitAndJSON(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		json  bool
	}{
		{"limit-1", 1, false},
		{"limit-all", 50, false},
		{"limit-json", 2, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auditRunEnv(t)
			auditRunSeed(t,
				auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"),
				auditRunEntry("2026-02-01T00:00:00Z", "b.example.com"),
			)
			auditLimit = tc.limit
			JSONOutput = tc.json

			if err := runAuditLog(auditLogCmd, nil); err != nil {
				t.Fatalf("runAuditLog should succeed: %v", err)
			}
		})
	}
}

func TestRunAuditLog_InvalidSince(t *testing.T) {
	auditRunEnv(t)
	auditRunSeed(t, auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"))
	auditSince = "not-a-date"

	err := runAuditLog(auditLogCmd, nil)
	if err == nil || !strings.Contains(err.Error(), `invalid --since value "not-a-date"`) {
		t.Fatalf("expected --since validation error, got %v", err)
	}
}

func TestRunAuditLog_ValidSince(t *testing.T) {
	auditRunEnv(t)
	auditRunSeed(t,
		auditRunEntry("2020-01-01T00:00:00Z", "old.example.com"),
		auditRunEntry("2026-06-01T00:00:00Z", "new.example.com"),
	)
	auditSince = "2026-01-01"

	if err := runAuditLog(auditLogCmd, nil); err != nil {
		t.Fatalf("runAuditLog with --since should succeed: %v", err)
	}
}

func TestRunAuditSearch_Table(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		typeFlag  string
		wantError string
	}{
		{"match", "example", "", ""},
		{"no-match", "nomatch", "", ""},
		{"type-filter-match", "example", "create", ""},
		{"type-filter-none", "example", "delete", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auditRunEnv(t)
			auditRunSeed(t, auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"))
			auditType = tc.typeFlag

			err := runAuditSearch(auditSearchCmd, []string{tc.query})
			if tc.wantError == "" && err != nil {
				t.Fatalf("runAuditSearch should succeed: %v", err)
			}
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestRunAuditExport_EmptyLog(t *testing.T) {
	auditRunEnv(t)

	if err := runAuditExport(auditExportCmd, nil); err != nil {
		t.Fatalf("exporting an empty log should succeed: %v", err)
	}
}

func TestRunAuditExport_Files(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		want     string
	}{
		{"json", "audit-backup.json", `"resource": "a.example.com"`},
		{"csv", "audit-report.csv", "timestamp,operation,service"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auditRunEnv(t)
			auditRunSeed(t, auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"))

			target := filepath.Join(t.TempDir(), tc.filename)
			if err := runAuditExport(auditExportCmd, []string{target}); err != nil {
				t.Fatalf("export should succeed: %v", err)
			}
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("reading export file: %v", err)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("export %s: output missing %q\n%s", tc.filename, tc.want, data)
			}
		})
	}
}

func TestRunAuditExport_CreateFails(t *testing.T) {
	auditRunEnv(t)
	auditRunSeed(t, auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"))

	// A directory path cannot be created as a file by os.Create.
	err := runAuditExport(auditExportCmd, []string{t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "failed to create export file") {
		t.Fatalf("expected create failure, got %v", err)
	}
}

func TestRunAuditClear_Guards(t *testing.T) {
	cases := []struct {
		name      string
		force     bool
		before    string
		wantError string
	}{
		{"no-force", false, "", "requires --force"},
		{"invalid-before", true, "bogus-date", `invalid --before value "bogus-date"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auditRunEnv(t)
			auditRunSeed(t, auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"))
			auditForce = tc.force
			auditBefore = tc.before

			err := runAuditClear(auditClearCmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestRunAuditClear_ForceAll(t *testing.T) {
	auditRunEnv(t)
	path := auditRunSeed(t,
		auditRunEntry("2026-01-01T00:00:00Z", "a.example.com"),
		auditRunEntry("2026-02-01T00:00:00Z", "b.example.com"),
	)
	auditForce = true

	if err := runAuditClear(auditClearCmd, nil); err != nil {
		t.Fatalf("clear --force should succeed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading cleared log: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("log should be empty after clear, got %d bytes", len(data))
	}
}

func TestRunAuditClear_BeforePartial(t *testing.T) {
	auditRunEnv(t)
	path := auditRunSeed(t,
		auditRunEntry("2020-01-01T00:00:00Z", "old.example.com"),
		auditRunEntry("2026-06-01T00:00:00Z", "new.example.com"),
	)
	auditForce = true
	auditBefore = "2021-01-01"

	if err := runAuditClear(auditClearCmd, nil); err != nil {
		t.Fatalf("clear --before should succeed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading partially cleared log: %v", err)
	}
	if strings.Count(string(data), "\n") != 1 || !strings.Contains(string(data), "new.example.com") {
		t.Fatalf("expected only the new entry to remain, got:\n%s", data)
	}
}
