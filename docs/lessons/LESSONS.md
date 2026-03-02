# CosmoDev-R2Go2 Lessons Index

Master index of persistent learnings. Grouped by category, updated as lessons accumulate.

> **Update Pattern**: When a lesson recurs, update the count and add context rather than creating duplicates.
> Example: `⚠️ Updated 3x (latest: 2026-01-15)` with note about where else it applied.

---

## Categories

## Go

- [HTTP Headers Before WriteHeader](LESSON-001-go-http-header-before-writeheader.md)
  > `w.Header().Set()` after `w.WriteHeader()` is silently ignored — set headers first

- [json.Marshal HTML Escaping](LESSON-002-go-json-marshal-html-escaping.md)
  > `json.Marshal` converts `<`/`>` to `\u003c`/`\u003e` — check decoded values for injection detection

- [Case-Sensitive Patterns After ToLower](LESSON-003-case-sensitive-pattern-matching-after-tolower.md)
  > When lowercasing input, patterns must also be lowercase or they never match

---

## How This Works

1. **Capture**: `/session-summary` prompts for lessons at session end
2. **Categorize**: Lessons grouped by topic (api, swiftui, git, etc.)
3. **Update**: Recurring lessons get count updates, not duplicates
4. **Reference**: Claude reads this index to apply past learnings

## Statistics

- **Total Lessons**: 3
- **Most Updated**: -
- **Last Added**: 2026-03-02 (Session 014)
