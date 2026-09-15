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
	sp := &sqlSplitter{runes: []rune(sql)}

	for i := 0; i < len(sp.runes); i++ {
		r := sp.runes[i]
		var next rune
		if i+1 < len(sp.runes) {
			next = sp.runes[i+1]
		}

		if sp.state == sqlSplitNormal {
			sp.handleStatementBoundary(r, next)
			continue
		}
		if sp.handleNested(r, next) {
			i++
		}
	}

	sp.flush()
	return sp.statements
}

// sqlSplitter carries the scanner state shared by the SplitSQLStatements
// phase handlers.
type sqlSplitter struct {
	statements []string
	current    strings.Builder
	state      sqlSplitState
	runes      []rune
}

// flush appends the trimmed current statement to the result, if non-empty,
// and resets the buffer.
func (s *sqlSplitter) flush() {
	stmt := strings.TrimSpace(s.current.String())
	if stmt != "" {
		s.statements = append(s.statements, stmt)
	}
	s.current.Reset()
}

// handleNested processes r while the lexer is inside a quoted literal or a
// comment. It returns true when the rune pair (r, next) was consumed as an
// escaped quote or a comment terminator, so the caller should skip next.
func (s *sqlSplitter) handleNested(r, next rune) bool {
	s.current.WriteRune(r)
	switch s.state {
	case sqlSplitLineComment:
		if r == '\n' {
			s.state = sqlSplitNormal
		}
		return false
	case sqlSplitBlockComment:
		return s.handleBlockCommentEnd(next, r)
	default:
		return s.handleQuoteClose(r, next)
	}
}

// handleBlockCommentEnd closes a block comment when r/next form the */
// terminator. r has already been written by the caller.
func (s *sqlSplitter) handleBlockCommentEnd(next, r rune) bool {
	if r == '*' && next == '/' {
		s.current.WriteRune(next)
		s.state = sqlSplitNormal
		return true
	}
	return false
}

// handleQuoteClose closes the active quoted literal when r is the matching
// (non-doubled) quote character. r has already been written by the caller.
func (s *sqlSplitter) handleQuoteClose(r, next rune) bool {
	q := s.quoteChar()
	if r != q {
		return false
	}
	if next == q {
		s.current.WriteRune(next)
		return true
	}
	s.state = sqlSplitNormal
	return false
}

// quoteChar returns the opening quote character of the active quoted
// literal state.
func (s *sqlSplitter) quoteChar() rune {
	switch s.state {
	case sqlSplitSingleQuote:
		return '\''
	case sqlSplitDoubleQuote:
		return '"'
	default:
		return '`'
	}
}

// handleStatementBoundary processes r while in the normal state: opening
// quotes and comments, statement separators, and ordinary content.
func (s *sqlSplitter) handleStatementBoundary(r, next rune) {
	switch {
	case r == '\'', r == '"', r == '`':
		s.enterQuote(r)
		s.current.WriteRune(r)
	case r == '-' && next == '-':
		s.state = sqlSplitLineComment
		s.current.WriteRune(r)
	case r == '/' && next == '*':
		s.state = sqlSplitBlockComment
		s.current.WriteRune(r)
	case r == ';':
		s.flush()
	default:
		s.current.WriteRune(r)
	}
}

// enterQuote transitions into the quoted-literal state matching r.
func (s *sqlSplitter) enterQuote(r rune) {
	switch r {
	case '\'':
		s.state = sqlSplitSingleQuote
	case '"':
		s.state = sqlSplitDoubleQuote
	default:
		s.state = sqlSplitBacktick
	}
}
