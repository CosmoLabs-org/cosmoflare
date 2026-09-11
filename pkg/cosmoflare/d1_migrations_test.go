package cosmoflare

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readReqBody reads and returns the request body as a string for
// inspection in mock D1 API handlers.
func readReqBody(r *http.Request) string {
	data, _ := io.ReadAll(r.Body)
	return string(data)
}

func TestMigrationsCreate(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")

	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	path, err := svc.MigrationsCreate("create_users", migrationsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := filepath.Base(path)
	if base != "0001_create_users.sql" {
		t.Errorf("expected file name 0001_create_users.sql, got %s", base)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestMigrationsCreate_NextSequence(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	for _, name := range []string{"0001_first.sql", "0003_third.sql", "not_a_migration.txt"} {
		if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte("-- test"), 0o644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	path, err := svc.MigrationsCreate("fourth", migrationsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := filepath.Base(path)
	if base != "0004_fourth.sql" {
		t.Errorf("expected file name 0004_fourth.sql, got %s", base)
	}
}

func TestMigrationsList_Empty(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")

	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected API call for missing migrations directory")
	})
	defer server.Close()

	migrations, err := svc.MigrationsList(context.Background(), "db-123", migrationsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 0 {
		t.Errorf("expected 0 migrations, got %d", len(migrations))
	}
}

func TestMigrationsApply_DryRun(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "0001_create_users.sql"), []byte("CREATE TABLE users (id INTEGER);"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	executedMigrationSQL := false
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		body := readReqBody(r)
		if strings.Contains(body, "CREATE TABLE users") {
			executedMigrationSQL = true
		}
		d1WriteJSON(w, map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"results": []map[string]interface{}{}, "success": true, "meta": map[string]interface{}{}},
			},
		})
	})
	defer server.Close()

	results, err := svc.MigrationsApply(context.Background(), "db-123", migrationsDir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "would_apply" {
		t.Errorf("expected status would_apply, got %s", results[0].Status)
	}
	if executedMigrationSQL {
		t.Error("expected migration SQL not to be executed in dry-run mode")
	}
}

func TestMigrationsApply_Idempotent(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "0001_create_users.sql"), []byte("CREATE TABLE users (id INTEGER);"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	applied := false
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		body := readReqBody(r)
		switch {
		case strings.Contains(body, "SELECT name, applied_at"):
			rows := []map[string]interface{}{}
			if applied {
				rows = append(rows, map[string]interface{}{"name": "0001_create_users.sql", "applied_at": "2026-01-01T00:00:00Z"})
			}
			d1WriteJSON(w, map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{"results": rows, "success": true, "meta": map[string]interface{}{}},
				},
			})
		case strings.Contains(body, "INSERT INTO d1_migrations"):
			applied = true
			d1WriteJSON(w, map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{"results": []map[string]interface{}{}, "success": true, "meta": map[string]interface{}{}},
				},
			})
		default:
			d1WriteJSON(w, map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{"results": []map[string]interface{}{}, "success": true, "meta": map[string]interface{}{}},
				},
			})
		}
	})
	defer server.Close()

	first, err := svc.MigrationsApply(context.Background(), "db-123", migrationsDir, false)
	if err != nil {
		t.Fatalf("unexpected error on first apply: %v", err)
	}
	if len(first) != 1 || first[0].Status != "applied" {
		t.Fatalf("expected first apply to succeed with status applied, got %+v", first)
	}

	second, err := svc.MigrationsApply(context.Background(), "db-123", migrationsDir, false)
	if err != nil {
		t.Fatalf("unexpected error on second apply: %v", err)
	}
	if len(second) != 1 || second[0].Status != "skipped" {
		t.Fatalf("expected second apply to be skipped, got %+v", second)
	}
}

func TestDestructiveSQLDetection(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{
		{"DROP TABLE users;", true},
		{"drop table users;", true},
		{"DROP INDEX idx_users;", true},
		{"DROP DATABASE mydb;", true},
		{"TRUNCATE TABLE users;", true},
		{"DELETE FROM users;", true},
		{"DELETE FROM users WHERE id = 1;", false},
		{"CREATE TABLE users (id INTEGER);", false},
		{"SELECT * FROM users;", false},
		{"INSERT INTO users (id) VALUES (1);", false},
	}
	for _, c := range cases {
		got := ContainsDestructiveSQL(c.sql)
		if got != c.want {
			t.Errorf("ContainsDestructiveSQL(%q) = %v, want %v", c.sql, got, c.want)
		}
	}
}
