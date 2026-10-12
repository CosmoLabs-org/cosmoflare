package d1schema

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeMigrationFiles creates dir with the given filename -> content files.
func writeMigrationFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestParseMigrationsDir(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFiles(t, dir, map[string]string{
		"0001_init.sql": `-- initial schema
CREATE TABLE IF NOT EXISTS churches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    founded_on TEXT,
    CHECK (founded_on >= '1800-01-01'),
    PRIMARY KEY (id)
);

CREATE TABLE "mass schedules" (
    id TEXT PRIMARY KEY,
    church_id INTEGER NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL,
    times_json TEXT NOT NULL DEFAULT '[{"time":"09:00"}]'
);`,
		"0002_add_users.sql": `CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    display_name TEXT,
    CONSTRAINT uq_email UNIQUE (email),
    FOREIGN KEY (church_id) REFERENCES churches(id)
);

ALTER TABLE churches ADD COLUMN city TEXT;
ALTER TABLE "churches" ADD COLUMN region TEXT NOT NULL DEFAULT '';`,
		"notes.txt": "CREATE TABLE not_sql (id INTEGER);",
	})

	schema, err := ParseMigrationsDir(dir)
	if err != nil {
		t.Fatalf("ParseMigrationsDir: %v", err)
	}

	want := map[string][]string{
		"churches":       {"id", "name", "slug", "founded_on", "city", "region"},
		"mass schedules": {"id", "church_id", "day_of_week", "times_json"},
		"users":          {"id", "email", "display_name"},
	}
	if !reflect.DeepEqual(schema.Tables, want) {
		t.Errorf("Tables = %#v\nwant %#v", schema.Tables, want)
	}
}

func TestParseMigrationsDir_QuotedAndConstraintShapes(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFiles(t, dir, map[string]string{
		"0001.sql": "create table if not exists `feast days` (\n" +
			"  `id` integer primary key,\n" +
			"  [church id] integer,\n" +
			"  note text default 'a, b, (c)'\n" +
			");\n" +
			"ALTER TABLE `feast days` ADD COLUMN church_id INTEGER;",
	})

	schema, err := ParseMigrationsDir(dir)
	if err != nil {
		t.Fatalf("ParseMigrationsDir: %v", err)
	}

	want := map[string][]string{
		"feast days": {"id", "church id", "note", "church_id"},
	}
	if !reflect.DeepEqual(schema.Tables, want) {
		t.Errorf("Tables = %#v\nwant %#v", schema.Tables, want)
	}
}

func TestParseMigrationsDir_MissingDir(t *testing.T) {
	if _, err := ParseMigrationsDir(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}

func TestParseMigrationsDir_EmptyDir(t *testing.T) {
	schema, err := ParseMigrationsDir(t.TempDir())
	if err != nil {
		t.Fatalf("ParseMigrationsDir: %v", err)
	}
	if len(schema.Tables) != 0 {
		t.Errorf("Tables = %#v, want empty", schema.Tables)
	}
}

func TestParseMigrationsDir_NoCreateTableAsSelect(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFiles(t, dir, map[string]string{
		"0001.sql": "CREATE TABLE big_copy AS SELECT * FROM source;",
	})
	schema, err := ParseMigrationsDir(dir)
	if err != nil {
		t.Fatalf("ParseMigrationsDir: %v", err)
	}
	if len(schema.Tables) != 0 {
		t.Errorf("CREATE TABLE AS SELECT should not produce columns, got %#v", schema.Tables)
	}
}
