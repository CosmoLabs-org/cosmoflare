# LESSON-003: Case-Sensitive Pattern Matching After ToLower

**Category**: Go
**Session**: 014 (2026-03-02)
**Occurrences**: 1

## Lesson

When lowercasing input with `strings.ToLower()` for comparison, the patterns being matched against must also be lowercase. Mixing cases guarantees zero matches.

## Context

Discovered while fixing input validation tests. The validation function lowercased user input before checking against a list of dangerous patterns, but some patterns in the list contained uppercase characters (e.g., `DROP TABLE`), so they never matched.

## Pattern

```go
// WRONG — lowercased input will never match uppercase pattern
input := strings.ToLower(userInput)
if strings.Contains(input, "DROP TABLE") { // never true
    return ErrDangerous
}

// RIGHT — patterns also lowercase
input := strings.ToLower(userInput)
if strings.Contains(input, "drop table") { // matches correctly
    return ErrDangerous
}
```

## Key Insight

When normalizing case for comparison, normalize both sides. This seems obvious but is easy to miss when patterns are defined in a separate list or constant.
