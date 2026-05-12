---
requires_reading:
  - CLAUDE.md
  - pkg/r2go2/client.go
  - pkg/r2go2/types.go
  - pkg/r2go2/storage.go
  - pkg/r2go2/upload.go
  - pkg/r2go2/download.go
  - pkg/r2go2/errors.go
  - pkg/r2go2/options.go
  - pkg/r2go2/config.go
  - pkg/r2go2/guardrails.go
  - pkg/r2go2/cache.go
  - pkg/r2go2/audit.go
  - cmd/bucket.go
  - cmd/object.go
  - cmd/root.go
  - .version-registry.json
deliverables:
  - id: P-01
    title: "Integration tests against real R2 API (pkg/r2go2/integration_test.go)"
  - id: P-02
    title: "Multipart upload implementation for large files (pkg/r2go2/upload.go)"
  - id: P-03
    title: "Progress bars and transfer stats (internal/utils/progress.go)"
  - id: P-04
    title: "Fix internal/domain_disabled/ build failure"
  - id: P-05
    title: "--json output for all commands (cmd/bucket.go, cmd/object.go)"
  - id: P-06
    title: "Update USAGE.md documentation"
schema_version: 1
---

# Phase 2: Integration Tests, Multipart Upload, and CLI Hardening

**Project**: CosmoDev-R2Go2 (v0.3.0, build 75)
**Branch**: master
**Date**: 2026-05-08
**Predecessor**: Phase 1 - Library Extraction (completed)

## Context

Phase 1 extracted the public library into `pkg/r2go2/` (17 files) and wired all CLI commands to use it. Two commits landed:
- `d5352b3` feat(r2go2): extract public library into pkg/r2go2/
- `a15ece8` refactor(cmd): wire CLI commands to pkg/r2go2 library

Build is clean. Tests pass (12 tests in 3 packages for `pkg/` and `cmd/`). Note: `internal/domain_disabled/` has broken references (`cloudflare.CustomSSL` undefined) -- skip it with `go test ./pkg/... ./cmd/...` until fixed or removed.

The library has real S3/Cloudflare API wiring via `aws-sdk-go-v2` and `cloudflare-go`. Phase 2 focuses on proving it works against a live R2 API, hardening edge cases, and preparing the codebase for Phase 3 (Workers/KV).

## Goals

### G-01: Integration Tests Against Real R2 API

Currently all tests are unit tests (error types, client options, validation). We need integration tests that hit a real R2 bucket to prove the S3 wiring works end-to-end.

**What to do:**
- Create `pkg/r2go2/integration_test.go` with `//go:build integration` tag
- Test the full lifecycle: CreateBucket, Upload, ListObjects, GetObject, HeadObject, CopyObject, DeleteObject, DeleteBucket
- Test error paths: GetObject on missing key, DeleteObject on missing key, CreateBucket with duplicate name (cleanup after)
- Test pagination: upload 10+ objects, verify ListObjects with maxKeys and continuation tokens
- Test content types: upload with explicit Content-Type, verify via HeadObject
- Test metadata: upload with custom metadata, verify via HeadObject
- Add a Makefile target: `make test-integration` that runs `go test -tags=integration ./pkg/r2go2/`
- Document in the test file header: required env vars (`CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN`), bucket naming convention (prefix `r2go2-integration-`), cleanup expectations

**Files:** `pkg/r2go2/integration_test.go`, `Makefile` (add target)

### G-02: Implement Multipart Upload for Large Files

The current `Upload()` in `pkg/r2go2/upload.go` uses `s3.PutObject` which is fine for small files but R2 (and S3) has a 5GB single-upload limit. For files > 5GB, multipart upload is required. Even for files > 100MB, multipart provides better throughput and resumability.

**What to do:**
- Add `MultipartUpload(ctx, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error)` to the `R2Client` interface in `client.go`
- Implement using `s3manager` from `aws-sdk-go-v2/feature/s3/manager` (or manual multipart with `CreateMultipartUpload`, `UploadPart`, `CompleteMultipartUpload`)
- Add `WithPartSize(n int64) UploadOption` for controlling part size (default 8MB)
- Add `WithConcurrency(n int) UploadOption` for parallel part uploads (default 3)
- Add progress reporting: `WithProgressCallback(fn func(uploaded, total int64))` option
- Auto-threshold: if size > 100MB, automatically use multipart even when `Upload()` is called
- Add unit tests for the multipart path (mock the S3 client)
- Update `UploadResult` to include `Parts` count when multipart was used

**Files:** `pkg/r2go2/client.go` (interface), `pkg/r2go2/upload.go` (implementation), `pkg/r2go2/multipart_test.go`

### G-03: Progress Bars and Transfer Stats

The CLI has `--progress` flags declared (`objectProgress` in `cmd/object.go`) but no progress bar implementation. Downloads and uploads silently complete with no feedback beyond "Uploaded successfully".

**What to do:**
- Create `internal/utils/progress.go` with a terminal progress bar (bytes transferred, speed, ETA)
- Support both upload and download with a `TransferProgress` struct tracking: bytes done, total bytes, start time, current speed
- Format: `[=====>          ] 45% | 23.4 MB/s | ETA 12s`
- Respect `--json` mode: when JSONOutput is true, suppress progress bar entirely
- Respect `--verbose` mode: show per-chunk transfer details
- Wire into `runObjectPut()` and `runObjectGet()` in `cmd/object.go`
- Add tests for the progress formatting functions

**Files:** `internal/utils/progress.go`, `internal/utils/progress_test.go`, `cmd/object.go` (wire in)

### G-04: Fix internal/domain_disabled/ Build Failure

`internal/domain_disabled/manager.go` references `cloudflare.CustomSSL` and `cloudflare.CustomSSLOptions` which do not exist in the current `cloudflare-go` module. This causes `go test ./...` to fail.

**What to do:**
- Either delete `internal/domain_disabled/` (it is unused stub code) or add a build tag to exclude it
- Verify `go test ./...` passes cleanly after fix
- Check `go vet ./...` also passes

**Files:** `internal/domain_disabled/` (remove or tag)

### G-05: --json Output Verification for All Commands

Several commands have `--json` declared globally but do not produce JSON output. Specifically:

- `bucket create` -- prints human text only, no JSON
- `bucket delete` -- prints human text only, no JSON
- `bucket exists` -- prints human text only, no JSON
- `bucket update` -- prints human text only, no JSON
- `bucket import` -- prints human text only, no JSON
- `object put` -- prints human text only, no JSON
- `object get` -- prints human text only, no JSON
- `object delete` -- prints human text only, no JSON
- `object copy` -- prints human text only, no JSON
- `object batch` -- prints human text only, no JSON

Commands that DO support JSON: `bucket list`, `bucket get`, `object ls`, `object head`, `object search`.

**What to do:**
- For each command above, add JSON output using the existing `printJSON()` and `OutputResponse` pattern
- JSON output should wrap results in `{"success": true, "data": ..., "dry_run": false}`
- Error cases should use `printErrorJSON()` for consistent structure
- Add tests that verify JSON output is valid when `--json` flag is set

**Files:** `cmd/bucket.go`, `cmd/object.go`, optionally `cmd/` test files

### G-06: Update USAGE.md for Modified Commands

Phase 1 changed the command architecture (library extraction). The USAGE.md files for `cmd/bucket/` and `cmd/object/` may be stale.

**What to do:**
- Check if `docs/USAGE.md` or per-command USAGE.md files exist
- Update to reflect current flag set, JSON output support, and library-backed behavior
- Include `--json` output examples for each command

**Files:** `docs/USAGE.md` (or equivalent)

## Carry-Over Tasks (Phase 3+ Prep)

These are NOT for this session but should be noted for future prompts:

- **Phase 3**: Workers service (`pkg/r2go2/worker.go` is a stub with empty struct). Needs `DeployWorker()`, `ListWorkers()`, `GetWorker()`, `DeleteWorker()`, `WorkerLogs()`. Same for KV (`pkg/r2go2/kv.go`).
- **Phase 4**: D1 service (`pkg/r2go2/d1.go`) and Pages service (`pkg/r2go2/pages.go`) stubs.
- **Phase 5**: Queues service (`pkg/r2go2/queue.go`) stub.
- **CCS integration**: `ccs r2` subcommand (future, requires library to be stable first).
- **TUI improvements**: `internal/tui/` and `internal/interactive/` packages need attention.

## Technical Notes

- **Module path**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Go version**: 1.25.3
- **Key dependencies**: `aws-sdk-go-v2`, `cloudflare-go`, `spf13/cobra`, `spf13/viper`
- **Test command**: `go test ./pkg/... ./cmd/...` (avoid `internal/domain_disabled/`)
- **Build command**: `go build -o r2go2 .`
- **R2Client interface** in `pkg/r2go2/client.go` is the public contract -- any new methods must be added there
- **Error types** in `pkg/r2go2/errors.go` use a hierarchy: `R2Error` base, `R2NotFoundError`, `R2AuthError`, `R2QuotaError`, `R2AccessDeniedError`, `R2ValidationError`
- **Config system**: `pkg/r2go2/config.go` loads from `.r2go2.yaml` (project) and `~/.r2go2/config.yaml` (user profiles)
- **Guardrails**: `pkg/r2go2/guardrails.go` validates uploads against project rules (allowed buckets, max file size, blocked keys)
- **Cache tiers**: `pkg/r2go2/cache.go` classifies files by extension for automatic Cache-Control headers
- **Audit logging**: `pkg/r2go2/audit.go` writes JSONL audit entries for upload/delete operations

## Session Priorities

1. **G-04** (fix build) -- quick, unblocks `go test ./...`
2. **G-01** (integration tests) -- proves the library actually works
3. **G-02** (multipart upload) -- most complex, do while context is fresh
4. **G-05** (JSON output) -- systematic, good for parallel agent dispatch
5. **G-03** (progress bars) -- polish, can be deferred if context runs low
6. **G-06** (USAGE.md) -- documentation, lowest priority
