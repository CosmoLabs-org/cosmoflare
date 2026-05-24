# Continuation Prompt: Post-Audit Critical Fixes

## Context

A comprehensive audit of CosmoDev-R2Go2 v0.9.0 was completed on 2026-05-19, scoring 70.8/100 (Mature). 10 critical/high bugs were found across 5 agent reports. This prompt covers Phase 0 of the upgrade plan -- the critical fixes that must land before any other work.

## Required Reading

- `docs/audit/2026-05-19-r2go2/README.md` -- audit executive summary
- `docs/audit/2026-05-19-r2go2/risk-map.md` -- full risk assessment
- `docs/audit/2026-05-19-r2go2/upgrade-plan.md` -- phased plan (execute Phase 0)

## Goals

### Goal 1: Fix Multipart Upload Sort Bug (CRITICAL)

**File**: `pkg/r2go2/upload.go:253`

The multipart upload reassembles parts in incorrect order, silently corrupting files larger than the multipart threshold. Fix the sort comparator to use numeric part number ordering. Add a test that uploads a multi-part synthetic file and verifies byte-for-byte integrity.

### Goal 2: Fix Config Permissions Race (CRITICAL)

**File**: `internal/config/config.go`

Config file is created with default permissions (0644) before `chmod` to 0600. Use `os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)` at creation time. Add a test verifying the file mode is correct immediately after creation.

### Goal 3: Fix Download Range Bug (HIGH)

**File**: `pkg/r2go2/download.go:53`

`rangeStart >= 0` always evaluates to true because the default is 0. Add a `rangeSet bool` field or use -1 as sentinel. Only apply the Range header when the caller explicitly set range parameters. Add a test verifying no Range header is sent for default downloads.

### Goal 4: Fix install.sh Binary Name (HIGH)

**File**: `install.sh`

The script references an uppercase binary name. Change to lowercase `r2go2` to match actual build output.

### Goal 5: Fix CI Pipeline (CRITICAL)

**Files**: `.github/workflows/*.yml` and any files flagged by `go vet`

Run `go vet ./...` locally, fix all errors, push a commit, verify CI runs green.

### Goal 6: Fix S3 Error Misclassification (CRITICAL)

**Files**: `pkg/r2go2/storage.go:158`, `pkg/r2go2/download.go:59`

All S3 errors are wrapped as `R2NotFoundError` regardless of actual error code. Inspect the underlying S3 error: map `NoSuchKey`/404 to `R2NotFoundError`, 403 to `R2AuthError`, 429 to `R2RateLimitError`, 500 to `R2Error`. Add tests asserting correct error type for each scenario.

### Goal 7: Fix Double defer Close (CRITICAL)

**File**: `cmd/object.go:389,391`

Remove the duplicate `defer obj.Content.Close()` at line 391. Keep only the first defer at line 389.

### Goal 8: Fix Bucket Update No-Op (CRITICAL)

**File**: `cmd/bucket.go:353-380`

The command claims success but makes no API call. Either implement the actual update or return a clear "not implemented" error.

## Acceptance Criteria

- [ ] Multipart upload test passes with correct byte ordering
- [ ] Config file created with 0600 permissions from the start
- [ ] Download without explicit range does not send Range header
- [ ] install.sh references correct binary name
- [ ] `go vet ./...` exits 0
- [ ] CI pipeline runs and passes
- [ ] S3 403 errors return R2AuthError, not R2NotFoundError
- [ ] Only one defer Close() in object.go download path
- [ ] Bucket update either works or returns error

## Estimated Effort

2-3 days for all 8 fixes. Each fix is independent and can be parallelized across agents.
