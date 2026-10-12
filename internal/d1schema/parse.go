// Package d1schema implements FEAT-051 (bounded slice): schema-drift parity
// between local migration files and a remote D1 database.
//
// This file parses a migrations directory into a table -> column-names map.
// Only the schema-drift class is covered here; cost and linting classes are
// out of scope.
package d1schema

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Schema is the parsed shape of a database schema: table name to the ordered
// list of column names.
type Schema struct {
	Tables map[string][]string
}

// createTableRe locates CREATE TABLE [IF NOT EXISTS] <name> ( — the paren
// body is then matched by a depth scan (createTableBody) so nested parens in
// column defaults and checks are handled correctly.
var createTableRe = regexp.MustCompile(`(?is)\bCREATE\s+(?:TEMP(?:ORARY)?\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?("[^"]+"|` + "`[^`]+`" + `|\[[^\]]+\]|[\w$]+(?:\s*\.\s*[\w$]+)?)\s*\(`)

// alterAddColumnRe matches ALTER TABLE <name> ADD [COLUMN] <name>.
var alterAddColumnRe = regexp.MustCompile(`(?is)\bALTER\s+TABLE\s+("[^"]+"|` + "`[^`]+`" + `|\[[^\]]+\]|[\w$]+(?:\s*\.\s*[\w$]+)?)\s+ADD\s+(?:COLUMN\s+)?("[^"]+"|` + "`[^`]+`" + `|\[[^\]]+\]|[\w$]+)`)

// tableConstraintTokens are the leading keywords of a table-level constraint
// (not a column definition) inside a CREATE TABLE paren body.
var tableConstraintTokens = map[string]bool{
	"PRIMARY":    true,
	"FOREIGN":    true,
	"UNIQUE":     true,
	"CHECK":      true,
	"CONSTRAINT": true,
}

// ParseMigrationsDir reads every *.sql file in dir (sorted by filename, the
// usual NNNN_name.sql ordering) and extracts the schema: CREATE TABLE
// columns plus columns added by ALTER TABLE ... ADD COLUMN.
func ParseMigrationsDir(dir string) (Schema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Schema{}, fmt.Errorf("d1schema: reading migrations dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)

	schema := Schema{Tables: make(map[string][]string)}
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return Schema{}, fmt.Errorf("d1schema: reading migration %q: %w", path, err)
		}
		applySQL(&schema, string(content))
	}
	return schema, nil
}

// applySQL folds one migration file's SQL into the schema. Matches from both
// statement shapes are collected and applied in document order so an ALTER
// that follows a CREATE appends to the CREATE's columns, as execution would.
func applySQL(schema *Schema, sql string) {
	type edit struct {
		pos int
		fn  func()
	}
	var edits []edit

	for _, m := range alterAddColumnRe.FindAllStringSubmatchIndex(sql, -1) {
		table, column := normalizeIdent(sql[m[2]:m[3]]), normalizeIdent(sql[m[4]:m[5]])
		edits = append(edits, edit{m[0], func() {
			schema.Tables[table] = append(schema.Tables[table], column)
		}})
	}
	for _, loc := range createTableRe.FindAllStringSubmatchIndex(sql, -1) {
		table := normalizeIdent(sql[loc[2]:loc[3]])
		// loc[1] is the end of the whole match, i.e. just past the paren
		// that opens the column list.
		body := createTableBody(sql, loc[1])
		edits = append(edits, edit{loc[0], func() {
			schema.Tables[table] = append(schema.Tables[table], parseColumnNames(body)...)
		}})
	}

	sort.Slice(edits, func(i, j int) bool { return edits[i].pos < edits[j].pos })
	for _, e := range edits {
		e.fn()
	}
}

// createTableBody returns the text between the paren that opens a CREATE
// TABLE column list and its matching close paren, tracking nesting depth and
// skipping string literals and comments so defaults like 'a,b)' or
// (datetime('now')) do not cut the body short. openIdx points just after the
// opening paren.
func createTableBody(sql string, openIdx int) string {
	depth := 1
	inLineComment, inBlockComment := false, false
	var quote byte
	for i := openIdx; i < len(sql); i++ {
		c := sql[i]
		switch {
		case inLineComment:
			if c == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			if c == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				inBlockComment = false
				i++
			}
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case (c == '-' && i+1 < len(sql) && sql[i+1] == '-'):
			inLineComment = true
			i++
		case (c == '/' && i+1 < len(sql) && sql[i+1] == '*'):
			inBlockComment = true
			i++
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				return sql[openIdx:i]
			}
		}
	}
	return sql[openIdx:]
}

// parseColumnNames extracts the column names from a CREATE TABLE paren body:
// the body is split on top-level commas and the first identifier of each
// definition is taken; table-level constraints (PRIMARY KEY, FOREIGN KEY,
// UNIQUE, CHECK, CONSTRAINT ...) are skipped.
func parseColumnNames(body string) []string {
	var columns []string
	for _, def := range splitTopLevel(body) {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}
		first := firstIdent(def)
		if first == "" || tableConstraintTokens[strings.ToUpper(first)] {
			continue
		}
		columns = append(columns, normalizeIdent(first))
	}
	return columns
}

// splitTopLevel splits s on commas that are not nested inside parens or
// string literals.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// firstIdent returns the first identifier token of a column definition,
// including any surrounding quotes so normalizeIdent can strip them.
func firstIdent(def string) string {
	switch {
	case strings.HasPrefix(def, `"`):
		if end := strings.Index(def[1:], `"`); end >= 0 {
			return def[:end+2]
		}
	case strings.HasPrefix(def, "`"):
		if end := strings.Index(def[1:], "`"); end >= 0 {
			return def[:end+2]
		}
	case strings.HasPrefix(def, "["):
		if end := strings.Index(def, "]"); end >= 0 {
			return def[:end+1]
		}
	default:
		if end := strings.IndexAny(def, " \t\r\n("); end >= 0 {
			return def[:end]
		}
		return def
	}
	return ""
}

// normalizeIdent unquotes an SQL identifier and drops any schema qualifier.
func normalizeIdent(ident string) string {
	ident = strings.TrimSpace(ident)
	switch {
	case len(ident) >= 2 && strings.HasPrefix(ident, `"`) && strings.HasSuffix(ident, `"`):
		ident = ident[1 : len(ident)-1]
	case len(ident) >= 2 && strings.HasPrefix(ident, "`") && strings.HasSuffix(ident, "`"):
		ident = ident[1 : len(ident)-1]
	case len(ident) >= 2 && strings.HasPrefix(ident, "[") && strings.HasSuffix(ident, "]"):
		ident = ident[1 : len(ident)-1]
	}
	if idx := strings.LastIndex(ident, "."); idx >= 0 {
		ident = ident[idx+1:]
	}
	return ident
}
