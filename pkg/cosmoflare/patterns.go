// Project cache scanning (FEAT-067 wave 3): a regex-based, corpus-calibrated
// scanner that reads a Worker project's TypeScript source and builds a
// source-derived caching inventory - kvCache'd routes, cache-delete sites,
// whole-table aggregates in uncached handlers, and the cache middleware's
// own behavior. Read-only, no API calls: later tasks grade the inventory
// against real D1 write cadence and emit P1-P5 findings.
package cosmoflare

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CachedRoute is one kvCache attachment found in project source.
type CachedRoute struct {
	File         string `json:"file"` // relative to scan root, '/'-separated
	Line         int    `json:"line"`
	Route        string `json:"route"` // "GET /api/meta/cities" form; path as written
	KeyFn        bool   `json:"key_fn"`
	QueryAware   bool   `json:"query_aware"`   // kvCache options text contains ".query("
	HandlerQuery bool   `json:"handler_query"` // route window contains ".query(" outside the kvCache options
	TTL          int    `json:"ttl"`           // seconds; 0 = not specified (middleware default)
	KeyPrefix    string `json:"key_prefix"`    // static head before '${' or ""; "" = dynamic/function-derived
	HasAuth      bool   `json:"has_auth"`
}

// InvalidationCall is one cache-delete site with a string-ish argument.
type InvalidationCall struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	KeyPattern string `json:"key_pattern"` // static head before '${' or ")"
}

// AggregateQuery is a whole-table aggregate in an uncached route handler.
type AggregateQuery struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Route string `json:"route"`
	SQL   string `json:"sql"`
}

// MiddlewareBehavior reports what the file defining kvCache itself does.
type MiddlewareBehavior struct {
	File              string `json:"file"`
	EmitsCacheControl bool   `json:"emits_cache_control"`
	OnlyStatus200     bool   `json:"only_status_200"`
}

// ProjectCacheInventory is the full source-derived caching picture.
type ProjectCacheInventory struct {
	Root           string               `json:"root"`
	Routes         []CachedRoute        `json:"routes"`
	Invalidation   []InvalidationCall   `json:"invalidation"`
	Aggregates     []AggregateQuery     `json:"aggregates"`
	Middleware     []MiddlewareBehavior `json:"middleware"`
	TotalGETRoutes int                  `json:"total_get_routes"`
	FilesScanned   int                  `json:"files_scanned"`
}

// ScanProjectCache walks root and builds the inventory. Read-only.
func ScanProjectCache(root string) (*ProjectCacheInventory, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("source directory %s does not exist", root)
	}
	inv := &ProjectCacheInventory{Root: root}

	scanErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry: skip, never fail the scan
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case "node_modules", ".git", "dist", "build":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".ts") {
			return nil
		}
		if strings.HasSuffix(name, "_test.ts") || strings.HasSuffix(name, ".spec.ts") || strings.HasSuffix(name, ".test.ts") {
			return nil
		}
		// Path filters use '/'-separated subpaths on every platform.
		relSlash := filepath.ToSlash(relPath(root, path))
		for _, banned := range []string{"node_modules/", ".git/", "dist/", "build/"} {
			if strings.Contains(relSlash, banned) {
				return nil
			}
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return nil // unreadable file: skip
		}
		inv.FilesScanned++
		scanFile(inv, relSlash, string(src))
		return nil
	})
	if scanErr != nil {
		return nil, scanErr
	}
	return inv, nil
}

// scanFile runs the per-file passes over one scanned source file.
func scanFile(inv *ProjectCacheInventory, file string, text string) {
	lines := strings.Split(text, "\n")
	scanFileRoutes(inv, file, lines)
	scanFileInvalidation(inv, file, lines)
	scanFileMiddleware(inv, file, text)
}

// relPath is filepath.Rel that never fails: a prefix-less path falls back to
// the path as given.
func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err == nil {
		return rel
	}
	return path
}

// scanFileRoutes finds route definitions and their windows: kvCache
// attachments are attributed to their route, aggregates are collected from
// uncached windows, and /-anchored GET defs feed TotalGETRoutes.
func scanFileRoutes(inv *ProjectCacheInventory, file string, lines []string) {
	// Route windows close at a bare `));`, `)`, or `})` line - the corpus
	// closes multi-line route defs with `})`, and an unterminated window
	// swallows the next route's kvCache options (HandlerQuery false
	// positives).
	closeRe := regexp.MustCompile(`^\s*(?:\)\)?;?|\}\);?)$`)
	// Route path literal: '/'-anchored so Hono context getters (c.get('user'),
	// formData.get('file')) never count as route defs. \s spans newlines, so
	// the regex matches multi-line route defs too - the corpus mounts
	// churches.get( with the path literal on the next line.
	routeRe := regexp.MustCompile(`\.(get|post|put|patch|delete)\(\s*['"\x60]/([^'"\x60]*)['"\x60]`)

	// Auth markers need a word boundary: a lowercase substring 'auth' test
	// false-positives on 'authors'/'oauth' table names and phantom-flags a
	// CRITICAL p2 that also suppresses P1. (?i) case-folds, and the [^a-z]
	// classes fold with it, so 'AUTH' inside an uppercase identifier is not
	// a marker either.
	authRe := regexp.MustCompile(`(?i)(authmiddleware|requireauth|authguard|useauth|getauth|[^a-z]auth[^a-z]|^auth)`)

	for i := 0; i < len(lines); i++ {
		// The .get( call open must sit on this line (comment banners above
		// multi-line route defs quote route-looking text and must not match).
		if !strings.Contains(lines[i], ".get(") && !strings.Contains(lines[i], ".post(") &&
			!strings.Contains(lines[i], ".put(") && !strings.Contains(lines[i], ".patch(") &&
			!strings.Contains(lines[i], ".delete(") {
			continue
		}
		// Def text: this line plus the next two - a path literal landing more
		// than 3 lines after its .get( open is not a route def.
		joinEnd := i + 3
		if joinEnd > len(lines) {
			joinEnd = len(lines)
		}
		m := routeRe.FindStringSubmatch(strings.Join(lines[i:joinEnd], "\n"))
		if m == nil {
			continue
		}
		method := strings.ToUpper(m[1])
		routePath := "/" + m[2]

		// Route window: this line up to the first of — the matching close
		// paren line, the next route def, or 40 lines.
		winEnd := i + 40
		if winEnd > len(lines) {
			winEnd = len(lines)
		}
		window := []string{}
		for j := i; j < winEnd; j++ {
			// The next route's def line must NOT be absorbed into this
			// window: a one-line def whose close never matches closeRe
			// would otherwise claim the next route's kvCache options as a
			// phantom cached route.
			if j > i && routeRe.MatchString(lines[j]) {
				break
			}
			window = append(window, lines[j])
			if j > i && closeRe.MatchString(lines[j]) {
				break
			}
		}
		windowText := strings.Join(window, "\n")

		if method == "GET" {
			inv.TotalGETRoutes++
		}
		routeName := method + " " + routePath
		kvIdx := strings.Index(windowText, "kvCache(")
		if kvIdx >= 0 {
			route := CachedRoute{
				File:  file,
				Line:  i + 1,
				Route: routeName,
			}
			optsText, optsEnd := kvCacheOptions(windowText, kvIdx)
			route.KeyFn = strings.Contains(optsText, "keyFn:")
			route.QueryAware = strings.Contains(optsText, ".query(")
			if ttl := regexp.MustCompile(`ttl:\s*(\d+)`).FindStringSubmatch(optsText); ttl != nil {
				route.TTL, _ = strconv.Atoi(ttl[1])
			}
			route.KeyPrefix = keyPrefix(optsText)
			route.HasAuth = authRe.MatchString(windowText)
			// HandlerQuery: .query( outside the kvCache options span.
			outside := windowText
			if optsEnd >= 0 && optsEnd <= len(windowText) {
				outside = windowText[:kvIdx] + windowText[optsEnd:]
			}
			route.HandlerQuery = strings.Contains(outside, ".query(")
			inv.Routes = append(inv.Routes, route)
			continue
		}

		// Uncached window: whole-table aggregate SQL literals.
		for _, lit := range sqlLiterals(windowText) {
			if isAggregateSQL(lit) {
				inv.Aggregates = append(inv.Aggregates, AggregateQuery{
					File:  file,
					Line:  i + 1 + strings.Count(windowText[:strings.Index(windowText, lit)], "\n"),
					Route: routeName,
					SQL:   lit,
				})
			}
		}
	}
}

// scanFileInvalidation records every .delete( call site whose argument is a
// string literal/template or function call. False positives (array deletes)
// are acceptable - coverage detection must err toward covered.
func scanFileInvalidation(inv *ProjectCacheInventory, file string, lines []string) {
	for i, line := range lines {
		for _, idx := range allIndexes(line, ".delete(") {
			arg := line[idx+len(".delete("):]
			if strings.ContainsAny(arg, "'\"`") {
				inv.Invalidation = append(inv.Invalidation, InvalidationCall{
					File:       file,
					Line:       i + 1,
					KeyPattern: literalHead(arg),
				})
				continue
			}
			// No literal: function-call argument -> bare call name up to "(".
			if p := strings.IndexByte(arg, '('); p >= 0 {
				inv.Invalidation = append(inv.Invalidation, InvalidationCall{
					File:       file,
					Line:       i + 1,
					KeyPattern: strings.TrimSpace(arg[:p]),
				})
			}
		}
	}
}

// scanFileMiddleware records the file defining kvCache itself and its two
// behavioral booleans.
func scanFileMiddleware(inv *ProjectCacheInventory, file string, text string) {
	if !strings.Contains(text, "export const kvCache") && !strings.Contains(text, "function kvCache(") {
		return
	}
	inv.Middleware = append(inv.Middleware, MiddlewareBehavior{
		File:              file,
		EmitsCacheControl: strings.Contains(text, "Cache-Control"),
		OnlyStatus200:     strings.Contains(text, "=== 200"),
	})
}

// kvCacheOptions extracts the kvCache options text starting at kvCache( up
// to the line-terminating attachment close, capped at 10 lines. The options
// end is the FIRST line (within the cap) matching the attachment-terminator
// shape `})`/`}),`/`});` at END of line — not the first `})` anywhere:
// block-bodied keyFns contain inner closures (.map(x => { ... }),
// .replace(..., () => ...)) whose inline `})` would otherwise truncate the
// options early and drop a ttl declared after them. Returns the options text
// and the index just past the terminator in windowText coordinates (-1-safe:
// an unterminated options block consumes the whole tail).
func kvCacheOptions(windowText string, kvIdx int) (string, int) {
	tail := windowText[kvIdx:]
	// Cap: the corpus's longest block keyFn (mass /city-window, ttl declared
	// before the keyFn) spans ~15 lines - 20 keeps every corpus block whole
	// while still bounding runaway matches.
	const optCap = 20
	lines := strings.SplitN(tail, "\n", optCap+1)
	if len(lines) == optCap+1 {
		tail = strings.Join(lines[:optCap], "\n")
		lines = lines[:optCap]
	}
	// `\s` never spans lines here (the text is already split), so `$` is the
	// line end: only a line-terminating `}),`/`})`/`});` closes the options.
	attachCloseRe := regexp.MustCompile(`\}\s*\)\s*[,);]?\s*$`)
	offset := 0
	for _, line := range lines {
		if loc := attachCloseRe.FindStringIndex(line); loc != nil {
			return tail[:offset+loc[0]], kvIdx + offset + loc[1]
		}
		offset += len(line) + 1 // +1 restores the newline SplitN consumed
	}
	return tail, len(windowText)
}

// keyPrefix finds the keyFn's value literal, head truncated at `${` or the
// closing quote. Block bodies: the FIRST return whose remainder starts with
// a literal — null/identifier early-returns are skipped (coverage must not
// read as known when the key template never appears). Expression arrows:
// the first literal after `=>`, with the block-body guard below.
// Function-call keys yield "".
func keyPrefix(optsText string) string {
	kvRe := regexp.MustCompile(`keyFn\s*:`)
	loc := kvRe.FindStringIndex(optsText)
	if loc == nil {
		return ""
	}
	value := optsText[loc[1]:]

	// Block bodies: iterate ALL return statements — `return\s` (whitespace
	// after the keyword, so a trailing '// returns all...' comment never
	// matches) — and take the FIRST whose remainder starts with a quote or
	// backtick. A LastIndex('return') shape breaks when the key template
	// comes first and `return null` is the fallback (yielding an empty
	// prefix = silent coverage-unknown).
	retRe := regexp.MustCompile(`return\s`)
	for _, ret := range retRe.FindAllStringIndex(value, -1) {
		rem := strings.TrimLeft(value[ret[1]:], " \t")
		if rem != "" && (rem[0] == '\'' || rem[0] == '"' || rem[0] == '`') {
			return literalHead(rem)
		}
	}
	// Expression arrow: the first TEMPLATE literal after `=>` — the corpus's
	// ternary shape (`(c) => (c.req.query('cc') ? \`api:...\` : null)`) puts a
	// query-argument string literal before the key template, and the query arg
	// is not the key prefix. Reached only when the body is not a block — a
	// block body without a literal return derives its key from code.
	if idx := strings.Index(value, "=>"); idx >= 0 {
		arrow := value[idx+2:]
		if strings.HasPrefix(strings.TrimLeft(arrow, " \t\n"), "{") {
			return ""
		}
		if bt := strings.IndexByte(arrow, '`'); bt >= 0 {
			// Template literal present: it is the key (the corpus's ternary
			// shape puts query-argument strings before the template).
			return literalHead(arrow[bt:])
		}
		// No template: a function-call key (churchCacheKey(...)) has its first
		// literal inside the argument list — the "(" right before it gives it
		// away, and the head must be "" (function-derived, coverage unknown).
		lit := strings.IndexAny(arrow, `'"`)
		if lit > 0 && strings.HasSuffix(strings.TrimSpace(arrow[:lit]), "(") {
			return ""
		}
		return literalHead(arrow)
	}
	return literalHead(value)
}

// literalHead extracts the first string literal's head: the text after the
// opening quote up to `${` or the closing quote. Returns "" when no literal
// appears (e.g. the value is a function call like churchCacheKey(...)).
func literalHead(s string) string {
	for i := 0; i < len(s) && i < 200; i++ {
		switch s[i] {
		case '\'', '"', '`':
			var close int
			for j := i + 1; j < len(s); j++ {
				if s[j] == '\\' {
					j++
					continue
				}
				if s[j] == s[i] {
					close = j
					break
				}
			}
			if close == 0 {
				// Unterminated literal: take the rest as the head.
				if dol := strings.Index(s[i+1:], "${"); dol >= 0 {
					return s[i+1 : i+1+dol]
				}
				return s[i+1:]
			}
			if dol := strings.Index(s[i+1:close], "${"); dol >= 0 {
				return s[i+1 : i+1+dol]
			}
			return s[i+1 : close]
		}
	}
	return ""
}

// allIndexes returns the start indexes of substr in s.
func allIndexes(s string, substr string) []int {
	var out []int
	for i := 0; ; {
		j := strings.Index(s[i:], substr)
		if j < 0 {
			break
		}
		out = append(out, i+j)
		i += j + len(substr)
	}
	return out
}

// sqlLiterals returns the string literals (backtick, single, double quoted)
// in the window text, quote-stripped.
func sqlLiterals(windowText string) []string {
	litRe := regexp.MustCompile("`[^`]*`|'[^']*'|\"[^\"]*\"")
	var out []string
	for _, m := range litRe.FindAllString(windowText, -1) {
		if len(m) >= 2 {
			out = append(out, m[1:len(m)-1])
		}
	}
	return out
}

// isAggregateSQL reports a whole-table aggregate: SELECT with COUNT( or
// GROUP BY and no WHERE clause (all case-insensitive, so `${where}`
// interpolation counts as having a WHERE).
func isAggregateSQL(sql string) bool {
	up := strings.ToUpper(sql)
	if !strings.Contains(up, "SELECT") {
		return false
	}
	if !strings.Contains(up, "COUNT(") && !strings.Contains(up, "GROUP BY") {
		return false
	}
	return !strings.Contains(up, "WHERE")
}
