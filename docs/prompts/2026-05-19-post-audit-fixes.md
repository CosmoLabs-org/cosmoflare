---
audit_score: 70.8
completed: "2026-05-29T12:00:00-03:00" # backfilled
created: "2026-05-19T00:00:00-05:00"
goals_completed: 7
goals_total: 7
priority: high
related_prompts: []
requires_reading:
    - docs/sessions/Session-2026-05-19-superwork-sprint.md
    - docs/audit/2026-03-28-r2go2/scorecard.json
schema_version: 1
session_source: Session-2026-05-19-superwork-sprint
status: COMPLETED
tags: []
title: 'Continuation: Post-Audit Phase 0 Fixes'
type: audit-followup
---

# Continuation: Post-Audit Phase 0 Fixes

## Context

The 360° audit (v0.10.0, scored 70.8/100) identified 7 critical/high issues in Phase 0. This session
fixes all of them. All fixes are bounded code changes with no new feature scope.

## Goal

Fix all 7 Phase 0 audit items, run `go test ./...`, confirm `go build` passes, commit, and close
any corresponding issues.

## Tasks (in priority order)

### 1. Multipart upload part sorting — `pkg/r2go2/upload.go:253-261`

Parts uploaded to S3 multipart must be sorted ascending by `PartNumber` before the
`CompleteMultipartUpload` call. The current code appends parts in response order, which is
non-deterministic.

**Fix**: Sort the `completedParts` slice by `PartNumber` before calling
`CompleteMultipartUpload`. Standard library `sort.Slice` is sufficient.

**Risk**: Data corruption on large uploads if parts arrive out of order.

---

### 2. S3 error misclassification — `pkg/r2go2/storage.go:158`

When S3 returns `NoSuchKey`, the code returns a generic error instead of the sentinel
`ErrNotFound`. Callers doing `errors.Is(err, r2go2.ErrNotFound)` get false negatives.

**Fix**: Unwrap the S3 error and check for `NoSuchKey` (use `errors.As` with `*types.NoSuchKey`
from the AWS SDK), then return `ErrNotFound` in that case.

---

### 3. S3 error misclassification — `pkg/r2go2/download.go:59`

Same issue as above but in the download path.

**Fix**: Same pattern — unwrap and map `NoSuchKey` to `ErrNotFound`.

---

### 4. Config file permission race — `internal/config/config.go:114-120`

Config is written with `os.WriteFile` directly, which is not atomic. A crash mid-write leaves a
truncated config. The permission check before write is also a TOCTOU race.

**Fix**: Write to a temp file in the same directory, set permissions on the temp file, then
`os.Rename` to the final path. Rename is atomic on POSIX systems.

---

### 5. install.sh BINARY_NAME case — `install.sh`

`BINARY_NAME` is set to `R2Go2` (mixed case). On case-sensitive filesystems (Linux, most CI),
the installed binary won't be found as `r2go2`.

**Fix**: Change `BINARY_NAME=R2Go2` to `BINARY_NAME=r2go2`.

---

### 6. release.yml asset path — `.github/workflows/release.yml`

The release workflow uses `matrix.platform` in the artifact asset path. `matrix.platform` values
contain `/` (e.g., `linux/amd64`), which breaks the filename.

**Fix**: Replace `matrix.platform` in the asset path with `matrix.os` or introduce a separate
`matrix.artifact_name` variable that uses `-` instead of `/` (e.g., `linux-amd64`).

---

### 7. bucket update no-op — `cmd/bucket.go:353-380`

The `r2go2 bucket update` command reads flags but never calls any API. Users get a success message
with no changes applied.

**Fix**: Identify what the bucket update should do (likely updating bucket CORS, lifecycle rules,
or public access settings via the Cloudflare API). If the API endpoint doesn't support the
operation, remove the command and return a clear "not supported" error rather than silently
succeeding.

---

## Execution Notes

- Work in isolated worktrees per fix group (items 2+3 can share a worktree; others separate).
- Each merge must pass: `go test ./pkg/... ./internal/...` in the worktree, `ccs verify-worktree --approve`, `ccs merge`.
- After all fixes: run `go test ./...` from master to confirm 997+ tests still pass.
- Close audit tracking issues if any exist; update `docs/audit/` with fix notes.

## Acceptance Criteria

- [x] `go build -o build/r2go2 .` passes
- [x] `go test ./pkg/... ./internal/...` passes (997+ tests)
- [x] `go vet ./...` clean
- [x] All 7 items above addressed (fixed or explicitly deferred with documented reason)
- [x] Commits use conventional format: `fix(upload): sort multipart parts before completion`
