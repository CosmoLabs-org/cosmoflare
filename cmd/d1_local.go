package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"

	// CGO-free SQLite driver; the release cross-compiles with CGO_ENABLED=0.
	_ "modernc.org/sqlite"
)

// d1StateDir is the root, relative to the current working directory, under
// which local (--local) D1 databases keep their SQLite state files.
const d1StateDir = ".cosmoflare/state/d1"

// runD1QueryLocal executes --sql against a local SQLite file instead of the
// remote D1 API. The database positional argument is the local database key
// (any user-supplied string), not a server UUID.
func runD1QueryLocal(database string) error {
	if DryRun {
		return outPayload("DRY RUN: Would execute local query", func() any {
			return map[string]string{"database": database, "mode": "local", "sql": d1SQL}
		}, func() {
			printInfo("DRY RUN: Would execute query locally on database '%s'", database)
		})
	}

	results, err := execLocalD1(database, d1SQL, d1Params)
	if err != nil {
		return outErr("failed to execute local query", err)
	}
	return outResult(results, func() { printD1Results(results) })
}

// execLocalD1 opens (creating if needed) the state file for database and
// executes each statement of query, mirroring the remote path's positional
// text binding of ?1, ?2, ... parameters.
func execLocalD1(database, query string, params []string) ([]*cosmoflare.D1QueryResult, error) {
	path, err := localD1Path(database)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare state file: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	args := make([]any, len(params))
	for i, p := range params {
		args[i] = p
	}

	statements := splitSQLStatements(query)
	results := make([]*cosmoflare.D1QueryResult, 0, len(statements))
	for _, stmt := range statements {
		result, err := execLocalStmt(db, stmt, args)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", stmt, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// execLocalStmt runs a single statement: read statements produce columns and
// rows, everything else reports an affected-rows count. Parameters are bound
// positionally as text (?1, ?2, ...), mirroring the remote path; statements
// that take no parameters simply ignore the bound values.
func execLocalStmt(db *sql.DB, stmt string, args []any) (*cosmoflare.D1QueryResult, error) {
	start := time.Now()

	prepared, err := db.Prepare(stmt)
	if err != nil {
		return nil, err
	}
	defer prepared.Close()

	if prepared.NumInput() == 0 {
		args = nil
	} else if prepared.NumInput() != len(args) {
		return nil, fmt.Errorf("statement expects %d parameter(s), got %d", prepared.NumInput(), len(args))
	}

	if isReadStatement(stmt) {
		return queryLocalStmt(prepared, args, start)
	}

	res, err := prepared.Exec(args...)
	if err != nil {
		return nil, err
	}
	changes, _ := res.RowsAffected()
	return &cosmoflare.D1QueryResult{
		Success: true,
		Meta: cosmoflare.D1QueryMeta{
			ChangedDB:   true,
			Changes:     int(changes),
			RowsWritten: int(changes),
			Duration:    sinceMillis(start),
		},
	}, nil
}

// queryLocalStmt runs a row-returning statement and materializes the rows as
// map[string]any, matching the remote path's result shape ([]byte values are
// stringified so --json output looks the same).
func queryLocalStmt(db *sql.DB, stmt string, args []any, start time.Time) (*cosmoflare.D1QueryResult, error) {
	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := &cosmoflare.D1QueryResult{
		Columns: columns,
		Rows:    []map[string]any{},
		Success: true,
	}
	for rows.Next() {
		vals := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			if b, ok := vals[i].([]byte); ok {
				vals[i] = string(b)
			}
			row[col] = vals[i]
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Meta.RowsRead = len(result.Rows)
	result.Meta.Duration = sinceMillis(start)
	return result, nil
}

// localD1Path returns .cosmoflare/state/d1/<env>/<database>.sqlite relative
// to the cwd, creating the parent directory (0755) as needed. <env> is the
// active profile name, or "default" when no profile is active.
func localD1Path(database string) (string, error) {
	env := "default"
	if ActiveProfile != nil && ActiveProfile.Name != "" {
		env = ActiveProfile.Name
	}
	path := filepath.Join(d1StateDir, sanitizeD1Key(env), sanitizeD1Key(database)+".sqlite")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// sanitizeD1Key makes a user-supplied database key path-safe: slashes,
// backslashes, and ".." sequences are replaced with "_".
func sanitizeD1Key(key string) string {
	s := strings.ReplaceAll(key, "/", "_")
	s = strings.ReplaceAll(s, `\`, "_")
	return strings.ReplaceAll(s, "..", "_")
}

// isReadStatement reports whether stmt is expected to return rows.
func isReadStatement(stmt string) bool {
	first := strings.Fields(strings.TrimSpace(stmt))
	if len(first) == 0 {
		return false
	}
	switch strings.ToUpper(first[0]) {
	case "SELECT", "WITH", "VALUES", "PRAGMA", "EXPLAIN", "TABLE":
		return true
	}
	return false
}

// splitSQLStatements splits a SQL script into individual statements on
// top-level semicolons, ignoring semicolons inside string literals, quoted
// identifiers, and comments.
func splitSQLStatements(query string) []string {
	var stmts []string
	start, quote := 0, byte(0)
	for i := 0; i < len(query); i++ {
		c := query[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			for i < len(query) && query[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			if end := strings.Index(query[i+2:], "*/"); end >= 0 {
				i += 2 + end + 1
			} else {
				i = len(query)
			}
		case c == ';':
			stmts = append(stmts, strings.TrimSpace(query[start:i]))
			start = i + 1
		}
	}
	if tail := strings.TrimSpace(query[start:]); tail != "" {
		stmts = append(stmts, tail)
	}
	filtered := stmts[:0]
	for _, s := range stmts {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// sinceMillis returns the elapsed duration in milliseconds.
func sinceMillis(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}
