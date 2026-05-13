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
  - docs/prompts/2026-05-08-phase2-integration-and-hardening.md
  - .version-registry.json
deliverables:
  - id: P-01
    title: "Integration tests against real R2 API (pkg/r2go2/integration_test.go)"
  - id: P-02
    title: "Multipart upload implementation for large files (pkg/r2go2/upload.go)"
  - id: P-03
    title: "Progress bars and transfer stats (internal/utils/progress.go)"
  - id: P-04
    title: "--json output for all commands (cmd/bucket.go, cmd/object.go)"
  - id: P-05
    title: "Update USAGE.md documentation"
  - id: P-06
    title: "Roadmap review and expansion (docs/roadmap/)"
  - id: P-07
    title: "Phase 3 implementation plan (Workers/KV services)"
schema_version: 1
---

# Phase 2 (continued): Integration Tests, Hardening, and Roadmap Expansion

**Project**: CosmoDev-R2Go2 (v0.3.1, build 82)
**Branch**: master
**Date**: 2026-05-12
**Predecessor**: Phase 2 - Integration and Hardening (2026-05-08, partially complete)

## What Got Done

### v0.3.0 -- Library Extraction (2026-05-08)

Two commits extracted the public library and wired the CLI:
- `d5352b3` feat(r2go2): extract public library into pkg/r2go2/
- `a15ece8` refactor(cmd): wire CLI commands to pkg/r2go2 library

Result: 17 files in `pkg/r2go2/` (client, types, storage, upload, download, errors, options, config, guardrails, cache, audit), all CLI commands backed by library.

### v0.3.1 -- Bug Resolution (2026-05-12)

All 11 tracked bugs resolved across 4 commits:
- `5f54356` fix(cmd): resolve 6 CLI bugs across bucket, object, and root commands
- `d5c3cbe` fix(build): update Go version to 1.26, remove mock API client, add LICENSE
- `43a5a25` chore: resolve remaining bugs (all 11 marked resolved)
- `08a4639` chore(release): v0.3.1

Bugs fixed:
| ID | Bug | Fix |
|----|-----|-----|
| BUG-001 | Speed calculation uses time.Since wrong | Corrected to proper duration math |
| BUG-002 | PersistentPreRun blocks non-API commands | Guard moved to API-only paths |
| BUG-003 | CI workflows use non-existent Go version | Updated to Go 1.26 |
| BUG-004 | TUI menu keyboard shortcuts off-by-one | Fixed index offset |
| BUG-005 | YAML parsing not implemented in config | Replaced with Viper-backed config |
| BUG-006 | Object search regex/glob use fake patterns | Wired to real S3 ListObjects filtering |
| BUG-007 | API client returns mock data instead of real calls | Removed mock client entirely |
| BUG-008 | Disabled packages fail to build (cloudflare.CustomSSL) | Packages renamed to .disabled |
| BUG-009 | Compiled binaries tracked in git | Removed, added to .gitignore |
| BUG-010 | Missing LICENSE file despite MIT in README | Added MIT LICENSE |
| BUG-011 | Makefile binary name case mismatch | Fixed to r2go2 |

**Build status**: Clean. `go test ./pkg/... ./cmd/...` passes. `go vet` clean. All 11 bugs resolved.

## Goals

### G-01: Integration Tests Against Real R2 API (carry-over)

Currently all tests are unit tests. Need integration tests that hit a real R2 bucket to prove the S3 wiring works end-to-end.

**What to do:**
- Create `pkg/r2go2/integration_test.go` with `//go:build integration` tag
- Test full lifecycle: CreateBucket, Upload, ListObjects, GetObject, HeadObject, CopyObject, DeleteObject, DeleteBucket
- Test error paths: GetObject on missing key, DeleteObject on missing key
- Test pagination: upload 10+ objects, verify ListObjects with maxKeys and continuation tokens
- Test content types and metadata round-trip
- Add Makefile target: `make test-integration`
- Document required env vars in test file header

**Files:** `pkg/r2go2/integration_test.go`, `Makefile`

### G-02: Multipart Upload for Large Files (carry-over)

Current `Upload()` uses `s3.PutObject` -- fine for small files but R2 has a 5GB single-upload limit. For files > 100MB, multipart provides better throughput and resumability.

**What to do:**
- Add `MultipartUpload()` to the `R2Client` interface in `client.go`
- Implement using `s3manager` from `aws-sdk-go-v2/feature/s3/manager` (or manual multipart)
- Add `WithPartSize(n int64)`, `WithConcurrency(n int)`, `WithProgressCallback(fn)` options
- Auto-threshold: if size > 100MB, automatically use multipart
- Add unit tests for the multipart path
- Update `UploadResult` to include `Parts` count

**Files:** `pkg/r2go2/client.go`, `pkg/r2go2/upload.go`, `pkg/r2go2/multipart_test.go`

### G-03: Progress Bars and Transfer Stats (carry-over)

CLI has `--progress` flags declared but no implementation. Uploads/downloads complete silently.

**What to do:**
- Create `internal/utils/progress.go` with terminal progress bar (bytes, speed, ETA)
- Support upload and download with `TransferProgress` struct
- Format: `[=====>          ] 45% | 23.4 MB/s | ETA 12s`
- Respect `--json` mode (suppress progress bar) and `--verbose` mode (per-chunk details)
- Wire into `runObjectPut()` and `runObjectGet()` in `cmd/object.go`

**Files:** `internal/utils/progress.go`, `internal/utils/progress_test.go`, `cmd/object.go`

### G-04: --json Output Verification for All Commands (carry-over)

Several commands have `--json` declared but do not produce JSON output.

**Commands missing JSON:** bucket create/delete/exists/update/import, object put/get/delete/copy/batch
**Commands with JSON:** bucket list/get, object ls/head/search

**What to do:**
- Add JSON output using existing `printJSON()` and `OutputResponse` pattern
- Wrap results in `{"success": true, "data": ..., "dry_run": false}`
- Error cases use `printErrorJSON()` for consistent structure

**Files:** `cmd/bucket.go`, `cmd/object.go`

### G-05: Update USAGE.md Documentation (carry-over)

Phase 1 changed the command architecture. Documentation needs updating.

**What to do:**
- Update `docs/USAGE.md` to reflect current flag set, JSON output support, library-backed behavior
- Include `--json` output examples for each command
- Verify per-command USAGE.md files in `cmd/` subdirectories are current

**Files:** `docs/USAGE.md`, `cmd/*/USAGE.md`

### G-06: Roadmap Review and Expansion (new)

The roadmap has 33 items but only 2% are complete (ROAD-025, library extraction). Many items were captured in early brainstorming and may need re-scoping or re-prioritization given the current codebase state.

**What to do:**
- Review all 33 roadmap items against current codebase state
- Mark ROAD-025 (Library extraction) and any other completed items as done
- Evaluate ROAD-000 (Wire up real S3 API client) -- partially done via library extraction, may need scope update
- Check for gaps: are there features the codebase supports that the roadmap doesn't cover?
- Identify quick wins (small items that can be knocked out in parallel)
- Identify items that should be combined or split
- Ensure roadmap reflects the 3-tier product vision (library, CLI, CCS)
- Add any new items surfaced by the Phase 2 work (e.g., retry logic, error recovery patterns)
- Use `ccs roadmap update` for any changes

**Files:** `docs/roadmap/` (via ccs commands)

### G-07: Phase 3 Implementation Plan (new, if time permits)

Based on roadmap review, draft a Phase 3 plan for Workers and KV services. The stubs exist:
- `pkg/r2go2/worker.go` (251B -- empty struct)
- `pkg/r2go2/kv.go` (228B -- empty struct)

**What to do:**
- Define the public API surface for Workers: `DeployWorker()`, `ListWorkers()`, `GetWorker()`, `DeleteWorker()`, `WorkerLogs()`
- Define the public API surface for KV: `CreateNamespace()`, `ListNamespaces()`, `Put()`, `Get()`, `Delete()`
- Identify which Cloudflare API endpoints to use
- Draft as `docs/planning-mode/2026-05-12-phase3-workers-kv.md`

**Files:** `docs/planning-mode/2026-05-12-phase3-workers-kv.md`

## Carry-Over Tasks (Phase 2 incomplete)

These were defined in the 2026-05-08 Phase 2 prompt and remain incomplete:

| Goal | Status | Notes |
|------|--------|-------|
| G-01: Integration tests | Not started | Needs R2 credentials, full lifecycle test |
| G-02: Multipart upload | Not started | Most complex item, allocate most context |
| G-03: Progress bars | Not started | Polish, can defer |
| G-04: JSON output | Not started | Systematic, good for parallel agent dispatch |
| G-05: USAGE.md update | Not started | Documentation, lowest priority |

Note: G-04 from the original prompt (fix internal/domain_disabled/ build failure) was resolved in v0.3.1 via BUG-008 -- packages renamed to .disabled.

## Session Priorities

1. **G-06** (roadmap review) -- sets direction for the entire session, informs which Phase 2 items to prioritize
2. **G-01** (integration tests) -- proves the library actually works against live R2
3. **G-02** (multipart upload) -- most complex, do while context is fresh
4. **G-04** (JSON output) -- systematic, good for parallel agent dispatch
5. **G-03** (progress bars) -- polish, can be deferred if context runs low
6. **G-05** (USAGE.md) -- documentation, lowest priority
7. **G-07** (Phase 3 plan) -- only if time permits after Phase 2 progress

## Technical Notes

- **Module path**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Go version**: 1.26
- **Key dependencies**: `aws-sdk-go-v2`, `cloudflare-go`, `spf13/cobra`, `spf13/viper`
- **Test command**: `go test ./pkg/... ./cmd/...`
- **Build command**: `go build -o r2go2 .`
- **R2Client interface** in `pkg/r2go2/client.go` is the public contract -- any new methods must be added there
- **Error types** in `pkg/r2go2/errors.go`: `R2Error` base, `R2NotFoundError`, `R2AuthError`, `R2QuotaError`, `R2AccessDeniedError`, `R2ValidationError`
- **Config system**: `pkg/r2go2/config.go` loads from `.r2go2.yaml` (project) and `~/.r2go2/config.yaml` (user profiles)
- **Guardrails**: `pkg/r2go2/guardrails.go` validates uploads against project rules
- **Cache tiers**: `pkg/r2go2/cache.go` classifies files by extension for Cache-Control headers
- **Audit logging**: `pkg/r2go2/audit.go` writes JSONL audit entries
- **Disabled packages**: `internal/*_disabled/` -- renamed from original names to fix BUG-008, excluded from build
- **All 11 bugs resolved**: v0.3.1 is a clean release with no known open bugs
