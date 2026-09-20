package cmd

import (
	"strings"
	"testing"
)

// d1LocalSnapshot snapshots the local-query globals and restores them on
// cleanup, keeping tests independent of flag defaults and ambient profile.
func d1LocalSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	origSQL, origParams := d1SQL, d1Params
	origProfile := ActiveProfile
	t.Cleanup(func() {
		d1SQL, d1Params = origSQL, origParams
		ActiveProfile = origProfile
	})
	d1SQL, d1Params = "", nil
	ActiveProfile = nil
}

// TestRunD1QueryLocal_DryRun verifies the dry-run path reports the query it
// would run without touching the filesystem or requiring credentials.
func TestRunD1QueryLocal_DryRun(t *testing.T) {
	d1LocalSnapshot(t)
	DryRun = true
	d1SQL = "SELECT 1"

	out := capturePrint(t, func() {
		if err := runD1QueryLocal("db"); err != nil {
			t.Errorf("dry-run should exit cleanly, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would execute query locally on database 'db'") {
		t.Errorf("expected dry-run notice, got %q", out)
	}
}

// TestRunD1QueryLocal_ExecutesScript verifies a real local round trip in a
// scratch directory: DDL, DML, and a read statement in one script, with the
// read result reported through the standard result path.
func TestRunD1QueryLocal_ExecutesScript(t *testing.T) {
	d1LocalSnapshot(t)
	t.Chdir(t.TempDir())
	d1SQL = "CREATE TABLE t (x TEXT); INSERT INTO t VALUES ('hello'); SELECT x FROM t;"

	out := capturePrint(t, func() {
		if err := runD1QueryLocal("db"); err != nil {
			t.Errorf("local script should succeed, got %v", err)
		}
	})

	if !strings.Contains(out, "hello") {
		t.Errorf("expected selected value in output, got %q", out)
	}
}

// TestRunD1QueryLocal_InvalidStatement verifies a failing statement surfaces
// as an execution error carrying the offending SQL.
func TestRunD1QueryLocal_InvalidStatement(t *testing.T) {
	d1LocalSnapshot(t)
	t.Chdir(t.TempDir())
	d1SQL = "SELECT * FROM missing_table"

	err := runD1QueryLocal("db")
	if err == nil {
		t.Fatal("invalid SQL should return an error")
	}
	if !strings.Contains(err.Error(), "failed to execute local query") {
		t.Errorf("error = %q, want it to contain the execution failure", err.Error())
	}
}

// TestSplitSQLStatements covers the script splitter: top-level semicolons
// split, quoted semicolons do not, comments never split, and empties are
// dropped.
func TestSplitSQLStatements(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"two statements", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"trailing without semicolon", "SELECT 1; SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"semicolon in single quotes", "SELECT ';' AS x", []string{"SELECT ';' AS x"}},
		{"semicolon in double quotes", `SELECT "a;b" AS x`, []string{`SELECT "a;b" AS x`}},
		{"semicolon in backticks", "SELECT `a;b` AS x", []string{"SELECT `a;b` AS x"}},
		{"semicolon in line comment does not split", "-- c; omment\nSELECT 1;", []string{"-- c; omment\nSELECT 1"}},
		{"block comment ignored", "/* a ; b */ SELECT 1", []string{"/* a ; b */ SELECT 1"}},
		{"unterminated block comment", "SELECT 1 /* trailing", []string{"SELECT 1 /* trailing"}},
		{"empty script", "", nil},
		{"only separators", "  ;  ; ", nil},
		{"insert with quoted value", "INSERT INTO t VALUES ('a;b');", []string{"INSERT INTO t VALUES ('a;b')"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitSQLStatements(tc.query)
			if len(got) != len(tc.want) {
				t.Fatalf("splitSQLStatements(%q) = %v (len %d), want len %d", tc.query, got, len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("statement %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
