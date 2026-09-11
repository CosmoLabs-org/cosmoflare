package cosmoflare

import (
	"reflect"
	"testing"
)

func TestSplitSQLStatementsBasic(t *testing.T) {
	sql := "CREATE TABLE t (id INTEGER);\nINSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);"
	got := SplitSQLStatements(sql)
	want := []string{
		"CREATE TABLE t (id INTEGER)",
		"INSERT INTO t VALUES (1)",
		"INSERT INTO t VALUES (2)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestSplitSQLStatementsQuoteAware(t *testing.T) {
	sql := `INSERT INTO t (a, b) VALUES ('semi;colon', "another;one");
SELECT 1;`
	got := SplitSQLStatements(sql)
	want := []string{
		`INSERT INTO t (a, b) VALUES ('semi;colon', "another;one")`,
		"SELECT 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestSplitSQLStatementsCommentAware(t *testing.T) {
	sql := "SELECT 1; -- trailing line comment ; with semicolon\nSELECT /* block ; comment */ 2;"
	got := SplitSQLStatements(sql)
	if len(got) != 2 {
		t.Fatalf("expected 2 statements, got %d: %#v", len(got), got)
	}
	if got[0] != "SELECT 1" {
		t.Errorf("statement 0 = %q, want %q", got[0], "SELECT 1")
	}
	if got[1] != "-- trailing line comment ; with semicolon\nSELECT /* block ; comment */ 2" {
		t.Errorf("statement 1 = %q", got[1])
	}
}

func TestSplitSQLStatementsTrimAndSkipEmpty(t *testing.T) {
	sql := "  ; \n\n SELECT 1 ;\n\n  ;  \nSELECT 2;   "
	got := SplitSQLStatements(sql)
	want := []string{"SELECT 1", "SELECT 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestSplitSQLStatementsBacktickIdentifier(t *testing.T) {
	sql := "SELECT * FROM `my;table` WHERE `col``with backtick` = 1;\nSELECT 2;"
	got := SplitSQLStatements(sql)
	want := []string{
		"SELECT * FROM `my;table` WHERE `col``with backtick` = 1",
		"SELECT 2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestSplitSQLStatementsEmptyInput(t *testing.T) {
	got := SplitSQLStatements("")
	if len(got) != 0 {
		t.Errorf("expected 0 statements for empty input, got %#v", got)
	}

	got = SplitSQLStatements("   \n\t  ")
	if len(got) != 0 {
		t.Errorf("expected 0 statements for whitespace-only input, got %#v", got)
	}
}
