# LESSON-001: Go HTTP Headers Must Be Set Before WriteHeader()

**Category**: Go
**Session**: 014 (2026-03-02)
**Occurrences**: 1

## Lesson

In Go's `net/http`, `w.Header().Set()` calls after `w.WriteHeader()` are silently ignored. Headers must be set **before** the status code is written.

## Context

Discovered while fixing test failures in the security test suite. Tests were checking response headers that were being set after `WriteHeader()`, causing them to be missing from the response.

## Pattern

```go
// WRONG — headers silently lost
w.WriteHeader(http.StatusOK)
w.Header().Set("Content-Type", "application/json")

// RIGHT — headers set before status
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
```

## Key Insight

Go does not warn or error when you set headers after `WriteHeader()`. This makes the bug invisible unless you specifically check for the expected headers in tests.
