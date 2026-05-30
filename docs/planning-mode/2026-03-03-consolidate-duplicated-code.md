---
branch: consolidate-duplicated-code
completed: "2026-05-29"
created: "2026-03-03T12:00:00-03:00"
goals_completed: 0
goals_total: 0
origin: migrated by ccs prompts migrate
priority: medium
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: 'ROAD-001: Consolidate Duplicated Code'
deliverables:
  - P-01: Consolidate duplicated formatBytes and maskAccountID functions into internal/utils with security fix
---

# ROAD-001: Consolidate Duplicated Code

## Context

`formatBytes` is duplicated 6 times across 5 files (plus 1 test-only copy). `maskAccountID` is duplicated 3 times across 3 files, with one variant having a **security issue** (returns unmasked IDs for non-32-char inputs). This creates maintenance burden and inconsistency. ROAD-001 (p80, planned) and TASK-001 track this.

## Plan

### Step 1: Create `internal/utils/format.go`

The empty `internal/utils/` directory already exists. Create `format.go` with two exported functions:

```go
package utils

func FormatBytes(bytes int64) string { ... }  // canonical from cmd/bucket.go:681
func MaskAccountID(accountID string) string { ... }  // canonical from cmd/root.go:135
```

### Step 2: Create `internal/utils/format_test.go`

Test both functions with edge cases:
- `FormatBytes`: 0, 1, 1023, 1024, 1048576, large values, negative
- `MaskAccountID`: empty, short (≤8), exactly 32 chars, other lengths

### Step 3: Replace `formatBytes` (6 locations)

| File | Line | Change |
|------|------|--------|
| `cmd/bucket.go:681` | ~681 | Delete function, add `utils` import, update call sites |
| `internal/api/enhanced_client.go:425` | ~425 | Delete function, add `utils` import, update call sites |
| `internal/cli/visual/animations.go:455` | ~455 | Delete method, replace `te.formatBytes(n)` → `utils.FormatBytes(n)` |
| `internal/cli/visual/r2-progress.go:281` | ~281 | Delete method, replace `rup.formatBytes(n)` → `utils.FormatBytes(n)` |
| `internal/cli/visual/r2-progress.go:485` | ~485 | Delete method, replace `m.formatBytes(n)` → `utils.FormatBytes(n)` |
| `internal/tui/view.go:601` | ~601 | Delete function, add `utils` import, update call sites |

**Leave alone**: `tests/tui/performance/rendering/performance_test.go:569` (self-contained test helper)

### Step 4: Replace `maskAccountID` (3 locations)

| File | Line | Change |
|------|------|--------|
| `cmd/root.go:135` | ~135 | Delete function, add `utils` import, update call sites |
| `internal/config/config.go:313` | ~313 | Delete function, add `utils` import, update call sites |
| `internal/interactive/setup.go:348` | ~348 | Delete **different** implementation, replace with `utils.MaskAccountID()` |

### Step 5: Update existing test

- `internal/api/enhanced_client_test.go` — update `formatBytes()` calls to `utils.FormatBytes()`

### Step 6: Verify

1. `go build ./...` — compiles clean
2. `go vet ./...` — no issues
3. `go test ./internal/utils/...` — new tests pass
4. `go test ./...` — all existing tests still pass

## Files Modified

**New:**
- `internal/utils/format.go`
- `internal/utils/format_test.go`

**Modified (delete duplicate + update imports):**
- `cmd/bucket.go`
- `cmd/root.go`
- `internal/api/enhanced_client.go`
- `internal/api/enhanced_client_test.go`
- `internal/cli/visual/animations.go`
- `internal/cli/visual/r2-progress.go`
- `internal/config/config.go`
- `internal/interactive/setup.go`
- `internal/tui/view.go`
