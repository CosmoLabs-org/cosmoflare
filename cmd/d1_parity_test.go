package cmd

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// d1ParitySnapshot snapshots the parity command's globals on top of the
// shared run globals.
func d1ParitySnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	savedPath, savedTables, savedNoFail := d1ParityLocalPath, d1ParityTables, d1ParityNoFail
	t.Cleanup(func() {
		d1ParityLocalPath, d1ParityTables, d1ParityNoFail = savedPath, savedTables, savedNoFail
	})
}

// d1ParityMakeLocalDB creates a SQLite file at path with the given tables and
// row counts, returning nothing (failures abort the test).
func d1ParityMakeLocalDB(t *testing.T, path string, tables map[string]int) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	for table, n := range tables {
		if _, err := db.Exec(fmt.Sprintf("CREATE TABLE %q (id INTEGER PRIMARY KEY)", table)); err != nil {
			t.Fatalf("create %s: %v", table, err)
		}
		for i := 0; i < n; i++ {
			if _, err := db.Exec(fmt.Sprintf("INSERT INTO %q (id) VALUES (?1)", table), i+1); err != nil {
				t.Fatalf("insert into %s: %v", table, err)
			}
		}
	}
}

func TestParseD1ParityTables(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"users,orders", []string{"users", "orders"}},
		{" users , orders ,items ", []string{"users", "orders", "items"}},
		{"", nil},
		{" , ,", nil},
		{"users", []string{"users"}},
	}
	for _, tc := range cases {
		got := parseD1ParityTables(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("parseD1ParityTables(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseD1ParityTables(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestBuildD1ParityRow(t *testing.T) {
	t.Run("exact equality matches", func(t *testing.T) {
		row := buildD1ParityRow("users", 5, 5, nil, nil)
		if !row.Match || row.Local == nil || *row.Local != 5 || row.Remote == nil || *row.Remote != 5 || row.Error != "" {
			t.Errorf("got %+v, want match with counts 5/5 and no error", row)
		}
	})

	t.Run("count difference mismatches", func(t *testing.T) {
		row := buildD1ParityRow("users", 5, 6, nil, nil)
		if row.Match {
			t.Errorf("got %+v, want match=false for 5 vs 6", row)
		}
	})

	t.Run("local error is a row not a crash", func(t *testing.T) {
		row := buildD1ParityRow("users", 0, 7, fmt.Errorf("no such table"), nil)
		if row.Match || row.Local != nil || row.Error == "" || !strings.Contains(row.Error, "local") {
			t.Errorf("got %+v, want no-match row with local error", row)
		}
	})

	t.Run("remote error is a row not a crash", func(t *testing.T) {
		row := buildD1ParityRow("users", 7, 0, nil, fmt.Errorf("bad token"))
		if row.Match || row.Remote != nil || !strings.Contains(row.Error, "remote") {
			t.Errorf("got %+v, want no-match row with remote error", row)
		}
	})

	t.Run("both sides error", func(t *testing.T) {
		row := buildD1ParityRow("users", 0, 0, fmt.Errorf("l"), fmt.Errorf("r"))
		if row.Match || !strings.Contains(row.Error, "local") || !strings.Contains(row.Error, "remote") {
			t.Errorf("got %+v, want combined error", row)
		}
	})
}

func TestD1ParityMismatchCount(t *testing.T) {
	rows := []d1ParityRow{
		{Table: "a", Match: true},
		{Table: "b", Match: false},
		{Table: "c", Match: false},
	}
	if n := d1ParityMismatchCount(rows); n != 2 {
		t.Errorf("d1ParityMismatchCount = %d, want 2", n)
	}
}

func TestD1ParityLocalCount(t *testing.T) {
	dir := chdirTemp(t)
	path := filepath.Join(dir, "local.sqlite")
	d1ParityMakeLocalDB(t, path, map[string]int{"users": 3, "orders": 0})

	if n, err := d1ParityLocalCount(path, "users"); err != nil || n != 3 {
		t.Errorf("users count = %d, %v; want 3, nil", n, err)
	}
	if n, err := d1ParityLocalCount(path, "orders"); err != nil || n != 0 {
		t.Errorf("orders count = %d, %v; want 0, nil", n, err)
	}
	if _, err := d1ParityLocalCount(path, "missing"); err == nil {
		t.Errorf("missing table should error, got nil")
	}
}

func TestRunD1Parity_MissingLocalFile(t *testing.T) {
	d1ParitySnapshot(t)
	d1ParityLocalPath = "/nonexistent/dir/nope.sqlite"
	d1ParityTables = "users"

	err := runD1Parity(nil, []string{"db-uuid"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing-file error, got %v", err)
	}
}

func TestRunD1Parity_MissingTablesFlag(t *testing.T) {
	d1ParitySnapshot(t)
	d1ParityLocalPath = "unused"
	d1ParityTables = " , "

	err := runD1Parity(nil, []string{"db-uuid"})
	if err == nil || !strings.Contains(err.Error(), "--tables") {
		t.Fatalf("expected --tables error, got %v", err)
	}
}

// With no credentials the remote side is unreachable: rows are still emitted
// with a remote error, and the exit is non-zero (mismatch) unless --no-fail.
func TestRunD1Parity_RemoteUnreachable(t *testing.T) {
	d1ParitySnapshot(t)
	dir := chdirTemp(t)
	path := filepath.Join(dir, "local.sqlite")
	d1ParityMakeLocalDB(t, path, map[string]int{"users": 2})
	d1ParityLocalPath = path
	d1ParityTables = "users"

	out := capturePrint(t, func() {
		err := runD1Parity(nil, []string{"db-uuid"})
		if err == nil || !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("expected mismatch exit error, got %v", err)
		}
	})
	if !strings.Contains(out, "users") || !strings.Contains(out, "NO") {
		t.Errorf("human output missing mismatched row: %q", out)
	}
	if !strings.Contains(out, "remote") {
		t.Errorf("human output missing remote error hint: %q", out)
	}

	d1ParityNoFail = true
	err := runD1Parity(nil, []string{"db-uuid"})
	if err != nil {
		t.Errorf("--no-fail should exit zero, got %v", err)
	}
}

func TestRunD1Parity_JSONEnvelope(t *testing.T) {
	d1ParitySnapshot(t)
	dir := chdirTemp(t)
	path := filepath.Join(dir, "local.sqlite")
	d1ParityMakeLocalDB(t, path, map[string]int{"users": 2})
	d1ParityLocalPath = path
	d1ParityTables = "users"
	JSONOutput = true
	d1ParityNoFail = true

	out := capturePrint(t, func() {
		if err := runD1Parity(nil, []string{"db-uuid"}); err != nil {
			t.Errorf("json run with --no-fail should not fail: %v", err)
		}
	})
	for _, want := range []string{`"table"`, `"local"`, `"match"`, `"error"`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON envelope missing %s: %q", want, out)
		}
	}
}
