package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// d1SchemaDiffSnapshot zeroes the schema-diff command globals on top of the
// shared run globals and stubs the remote fetch seam.
func d1SchemaDiffSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	savedDatabase, savedMigrations := d1SchemaDiffDatabase, d1SchemaDiffMigrations
	savedFetch := d1SchemaDiffFetchRemote
	t.Cleanup(func() {
		d1SchemaDiffDatabase, d1SchemaDiffMigrations = savedDatabase, savedMigrations
		d1SchemaDiffFetchRemote = savedFetch
	})
}

// d1SchemaDiffMakeMigrations writes a small real-shaped migrations dir.
func d1SchemaDiffMakeMigrations(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		"0001_init.sql": "CREATE TABLE IF NOT EXISTS users (\n" +
			"    id INTEGER PRIMARY KEY,\n" +
			"    email TEXT NOT NULL UNIQUE\n" +
			");\n" +
			"CREATE TABLE IF NOT EXISTS posts (\n" +
			"    id INTEGER PRIMARY KEY,\n" +
			"    user_id INTEGER NOT NULL\n" +
			");\n",
		"0002_add_plan.sql": "ALTER TABLE users ADD COLUMN plan TEXT NOT NULL DEFAULT 'free';\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestSchemaDiffCommand_Mapping(t *testing.T) {
	d1SchemaDiffSnapshot(t)
	dir := d1SchemaDiffMakeMigrations(t)
	d1SchemaDiffDatabase = "prod-db"
	d1SchemaDiffMigrations = dir

	// Remote drift: users lost the plan column (ADDED-LOCAL-ONLY drift),
	// posts match, audit_logs exists only remotely.
	d1SchemaDiffFetchRemote = func(ctx context.Context, svc *cosmoflare.D1Service, database string) (map[string][]string, error) {
		if database != "prod-db" {
			t.Errorf("fetch got database %q, want prod-db", database)
		}
		return map[string][]string{
			"users":      {"id", "email"},
			"posts":      {"id", "user_id"},
			"audit_logs": {"id"},
		}, nil
	}

	// Human mode: statuses printed, exit code is always zero.
	out := capturePrint(t, func() {
		if err := runD1SchemaDiff(nil, nil); err != nil {
			t.Errorf("schema-diff must exit zero, got %v", err)
		}
	})
	for _, want := range []string{"ADDED-LOCAL-ONLY", "REMOTE-ONLY", "audit_logs", "users", "posts"} {
		if !strings.Contains(out, want) {
			t.Errorf("human output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "plan") {
		t.Errorf("human output should name drifted column plan:\n%s", out)
	}

	// JSON mode: envelope carries table/status/columns.
	JSONOutput = true
	out = capturePrint(t, func() {
		if err := runD1SchemaDiff(nil, nil); err != nil {
			t.Errorf("json schema-diff must exit zero, got %v", err)
		}
	})
	for _, want := range []string{`"table"`, `"status"`, `"ADDED-LOCAL-ONLY"`, `"REMOTE-ONLY"`, `"local_only"`, `"audit_logs"`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON envelope missing %s:\n%s", want, out)
		}
	}
}

func TestSchemaDiffCommand_MissingMigrationsDir(t *testing.T) {
	d1SchemaDiffSnapshot(t)
	d1SchemaDiffDatabase = "prod-db"
	d1SchemaDiffMigrations = filepath.Join(t.TempDir(), "nope")

	err := runD1SchemaDiff(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "migrations") {
		t.Fatalf("expected migrations-dir error, got %v", err)
	}
}

func TestSchemaDiffCommand_MissingFlags(t *testing.T) {
	d1SchemaDiffSnapshot(t)
	d1SchemaDiffDatabase = ""
	d1SchemaDiffMigrations = t.TempDir()
	if err := runD1SchemaDiff(nil, nil); err == nil || !strings.Contains(err.Error(), "--database") {
		t.Fatalf("expected --database error, got %v", err)
	}

	d1SchemaDiffDatabase = "prod-db"
	d1SchemaDiffMigrations = ""
	if err := runD1SchemaDiff(nil, nil); err == nil || !strings.Contains(err.Error(), "--migrations") {
		t.Fatalf("expected --migrations error, got %v", err)
	}
}

func TestSchemaDiffCommand_RemoteErrorIsInformational(t *testing.T) {
	d1SchemaDiffSnapshot(t)
	d1SchemaDiffDatabase = "prod-db"
	d1SchemaDiffMigrations = d1SchemaDiffMakeMigrations(t)
	d1SchemaDiffFetchRemote = func(ctx context.Context, svc *cosmoflare.D1Service, database string) (map[string][]string, error) {
		return nil, context.DeadlineExceeded
	}

	out := capturePrint(t, func() {
		if err := runD1SchemaDiff(nil, nil); err != nil {
			t.Errorf("remote failure must stay informational (exit 0), got %v", err)
		}
	})
	if !strings.Contains(out, "error") {
		t.Errorf("human output should surface the remote error:\n%s", out)
	}
}
