package cosmoflare

import "strings"

// sqlSplitState enumerates the lexer states SplitSQLStatements tracks while
// scanning a SQL script for top-level statement separators.
type sqlSplitState int

const (
	sqlSplitNormal sqlSplitState = iota
	sqlSplitSingleQuote
	sqlSplitDoubleQuote
	sqlSplitBacktick
	sqlSplitLineComment
	sqlSplitBlockComment
)

// SplitSQLStatements splits a SQL script into individual statements on
// semicolons, tracking single-quoted strings, double-quoted and
// backtick-quoted identifiers, and -- line / block comments so that
// semicolons inside any of those are not treated as statement separators.
// Doubled quote characters ('', "", ``) are treated as an escaped literal
// quote rather than a closing quote, matching SQL's standard escaping.
//
// Each returned statement is trimmed of leading/trailing whitespace; empty
// statements (blank lines, stray semicolons, comment-only segments) are
// omitted from the result.
func SplitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder

	flush := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
		current.Reset()
	}

	state := sqlSplitNormal
	runes := []rune(sql)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		var next rune
		if i+1 < n {
			next = runes[i+1]
		}

		switch state {
		case sqlSplitSingleQuote:
			current.WriteRune(r)
			if r == '\'' {
				if next == '\'' {
					current.WriteRune(next)
					i++
				} else {
					state = sqlSplitNormal
				}
			}
			continue
		case sqlSplitDoubleQuote:
			current.WriteRune(r)
			if r == '"' {
				if next == '"' {
					current.WriteRune(next)
					i++
				} else {
					state = sqlSplitNormal
				}
			}
			continue
		case sqlSplitBacktick:
			current.WriteRune(r)
			if r == '`' {
				if next == '`' {
					current.WriteRune(next)
					i++
				} else {
					state = sqlSplitNormal
				}
			}
			continue
		case sqlSplitLineComment:
			current.WriteRune(r)
			if r == '\n' {
				state = sqlSplitNormal
			}
			continue
		case sqlSplitBlockComment:
			current.WriteRune(r)
			if r == '*' && next == '/' {
				current.WriteRune(next)
				i++
				state = sqlSplitNormal
			}
			continue
		}

		switch {
		case r == '\'':
			state = sqlSplitSingleQuote
			current.WriteRune(r)
		case r == '"':
			state = sqlSplitDoubleQuote
			current.WriteRune(r)
		case r == '`':
			state = sqlSplitBacktick
			current.WriteRune(r)
		case r == '-' && next == '-':
			state = sqlSplitLineComment
			current.WriteRune(r)
		case r == '/' && next == '*':
			state = sqlSplitBlockComment
			current.WriteRune(r)
		case r == ';':
			flush()
		default:
			current.WriteRune(r)
		}
	}

	flush()
	return statements
}
