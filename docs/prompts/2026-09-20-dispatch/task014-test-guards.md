# TASK-014 — Shared test-global guard: consolidate per-file snapshot helpers

Repo: /Users/gabstudio/PROJECTS/cosmoflare, package cmd.

## Problem

~23 `cmd/*_run_test.go` files each hand-roll a `xxxRunSnapshot(t)` / `buckXxxGlobals(t)` helper doing the same save/restore/clear of the core run globals: `JSONOutput`, `DryRun`, `AccountID`, `APIToken` (some also `t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")` / `t.Setenv("CLOUDFLARE_API_TOKEN", "")`). ~150 duplicated lines; every new test file re-invents it.

## Fix

1. Add ONE shared helper in `cmd/root_helpers_test.go` (beside the existing `setOutputMode`):

```go
// runGlobalsSnapshot zeroes the shared run globals (output mode, dry-run,
// credentials) for the duration of the test and clears the credential env
// vars, restoring everything on cleanup. Per-file helpers keep only their
// own domain variables on top of this.
func runGlobalsSnapshot(t *testing.T) { ... }
```

2. Rewrite each per-file helper to call `runGlobalsSnapshot(t)` first, then handle ONLY its domain-specific variables (e.g. `d1Force`, `streamExpires`, `domainsFilter`). Where a helper does nothing beyond the core four + env, replace its call sites with `runGlobalsSnapshot(t)` directly and delete the helper.
3. DO NOT touch: `cmd/manifest_tree_test.go` (no snapshot helper), `cmd/dryrun_wire_test.go` (inline save/restore is its test subject), `cmd/part5_run_test.go`'s `part5Buffer`/`part5NewCommand` (different contract).
4. Preserve behavior exactly: same variables zeroed, same restore semantics. No test logic changes.

## Files

All `cmd/*_run_test.go` + `cmd/root_helpers_test.go`. Nothing outside `cmd/` and nothing non-test.

## Verify

`go test ./cmd/ -count=1 -timeout 300s` green (full package — no -run filter). `go vet ./cmd/` clean.

## Commit

`refactor(test): shared runGlobalsSnapshot replaces 23 per-file snapshot helpers (TASK-014)` — conventional, no AI attribution.
