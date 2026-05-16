---
completed: "2026-05-16"
created: "2026-05-13"
deliverables:
    - id: P-01
      title: WorkerService interface and implementation (pkg/r2go2/worker.go)
    - id: P-02
      title: Worker unit tests (pkg/r2go2/worker_test.go)
    - id: P-03
      title: Worker CLI commands (cmd/worker.go) registered in root
    - id: P-04
      title: KVService interface and implementation (pkg/r2go2/kv.go)
    - id: P-05
      title: KV unit tests (pkg/r2go2/kv_test.go)
    - id: P-06
      title: KV CLI commands (cmd/kv.go) registered in root
    - id: P-07
      title: Update USAGE.md and CLAUDE.md for Workers/KV
    - id: P-08
      title: Version bump to v0.4.0
goals_completed: 6
goals_total: 6
related_prompts: []
requires_reading: null
schema_version: 1
status: COMPLETED
tags: []
title: 'Phase 3: Workers and KV Service Implementation'
---

# Phase 3: Workers and KV Service Implementation

**Project**: CosmoDev-R2Go2 (v0.3.1, build 86)
**Branch**: master
**Date**: 2026-05-13
**Predecessor**: Phase 2 -- Roadmap Expansion (2026-05-12, complete)
**Target version**: v0.4.0

## What Got Done

### Phase 2 (2026-05-08 to 2026-05-12) -- Complete

All 7 goals from the continuation prompt were implemented:

- **Multipart upload**: `MultipartUpload()` on `R2Client` interface, auto-threshold at 100MB, `WithPartSize` and `WithConcurrency` options. Uses `aws-sdk-go-v2/feature/s3/manager` uploader.
- **JSON output**: All 10 remaining commands now support `--json` with consistent `{"success": true, "data": ...}` envelope.
- **Progress bars**: `internal/utils/progress.go` with terminal-safe bars (`[=====>          ] 45% | 23.4 MB/s | ETA 12s`). Wired into `runObjectPut()` and `runObjectGet()`. Suppressed in `--json` mode.
- **Integration tests**: `pkg/r2go2/integration_test.go` with `//go:build integration` tag, 6 tests covering bucket lifecycle, object CRUD, pagination, and content types.
- **Roadmap review**: 8 items marked complete, overall completion 26% -> 28%. Quick wins identified.
- **USAGE.md**: Updated with all current commands, flags, and JSON examples.
- **Phase 3 plan**: Drafted at `docs/planning-mode/2026-05-12-phase3-workers-kv.md` with full API surface definitions for Workers and KV.

**Build status**: Clean. `go test ./pkg/... ./cmd/...` passes. 56 tests total.

## Goals

### G-01: WorkerService Library Implementation

Replace the empty stub in `pkg/r2go2/worker.go` with a full `WorkerService` implementation.

**What to do:**
- Define `WorkerService` interface with: `Deploy`, `List`, `Get`, `Delete`, `Logs`, `UpdateSettings`
- Define types: `Worker`, `WorkerBinding`, `WorkerSettings`, `LogEntry`
- Implement using `cloudflare-go` client methods: `PutWorkerScript`, `ListWorkerScripts`, `GetWorkerScript`, `DeleteWorkerScript`
- Worker struct holds the `*cloudflare.API` client and `accountID` (same pattern as `client` in `client.go`)
- Constructor: `NewWorkerService(api *cloudflare.API, accountID string) (*WorkerService, error)`
- Functional options: `WithWorkerCompatibilityDate`, `WithWorkerBindings`, `WithWorkerTags`, `WithLogLimit`, `WithLogSince`
- Reuse existing error hierarchy from `errors.go` (`R2Error` base -- keeping the name per the plan's ADR)
- For `Logs`, start with REST-based analytics approach (WebSocket tail is complex, can be enhanced later)

**Files:** `pkg/r2go2/worker.go`

**Reference:** The `client` struct in `pkg/r2go2/client.go` shows the pattern -- it holds `cf *cloudflare.API` and `accountID string`. WorkerService should follow the same pattern.

### G-02: Worker Unit Tests

**What to do:**
- Create `pkg/r2go2/worker_test.go`
- Test type constructors and functional options
- Test that `NewWorkerService` validates inputs (nil client, empty accountID)
- Test option application (compatibility date, bindings, tags)
- Mock the Cloudflare API for service-level tests (see `client_test.go` for patterns)
- Target: 8-12 test cases

**Files:** `pkg/r2go2/worker_test.go`

### G-03: Worker CLI Commands

**What to do:**
- Create `cmd/worker.go` with cobra subcommand group
- Commands: `deploy`, `list`, `get`, `delete`, `logs`, `settings`
- All commands must support `--json` output (follow `cmd/bucket.go` patterns)
- `deploy`: `r2go2 worker deploy <name> --script=worker.js [--compatibility-date=2024-01-01] [--bindings=kv:my-kv]`
- `list`: `r2go2 worker list [--json]`
- `get`: `r2go2 worker get <name> [--json]`
- `delete`: `r2go2 worker delete <name> [--force]`
- `logs`: `r2go2 worker logs <name> [--limit=100] [--json]`
- `settings`: `r2go2 worker settings <name> --compatibility-date=2024-01-01 [--usage-model=bundled]`
- Register `workerCmd` in `cmd/root.go`'s `init()` function
- Add "worker" to the skipValidation list in root's PersistentPreRun (logs may not need API validation, but deploy/list/get/delete do)

**Files:** `cmd/worker.go`, `cmd/root.go`

### G-04: KVService Library Implementation

Replace the empty stub in `pkg/r2go2/kv.go` with a full `KVService` implementation.

**What to do:**
- Define `KVService` interface with: `CreateNamespace`, `ListNamespaces`, `GetNamespace`, `DeleteNamespace`, `Put`, `Get`, `Delete`, `ListKeys`
- Define types: `KVNamespace`, `KVKey`
- Implement namespace management using `cloudflare-go`: `CreateWorkersKVNamespace`, `ListWorkersKVNamespaces`, `GetWorkersKVNamespace`, `DeleteWorkersKVNamespace`
- Implement key-value operations using Cloudflare API endpoints (not S3 -- KV uses its own REST API)
- Constructor: `NewKVService(api *cloudflare.API, accountID string) (*KVService, error)`
- Functional options: `WithKVTTL`, `WithKVMetadata`, `WithKVPrefix`, `WithKVLimit`
- Support TTL via `expiration_ttl` parameter
- `ListKeys` should support prefix filtering and pagination

**Files:** `pkg/r2go2/kv.go`

### G-05: KV Unit Tests

**What to do:**
- Create `pkg/r2go2/kv_test.go`
- Test type constructors and functional options
- Test that `NewKVService` validates inputs
- Test option application (TTL, metadata, prefix, limit)
- Target: 8-12 test cases

**Files:** `pkg/r2go2/kv_test.go`

### G-06: KV CLI Commands

**What to do:**
- Create `cmd/kv.go` with cobra subcommand group
- Namespace commands: `kv namespace create <title>`, `kv namespace list`, `kv namespace delete <id>`
- Key-value commands: `kv put <namespace-id> <key> [--value="data"] [--file=data.json] [--ttl=3600]`, `kv get <namespace-id> <key> [--json]`, `kv delete <namespace-id> <key>`, `kv list <namespace-id> [--prefix=cache/] [--limit=100] [--json]`
- All commands must support `--json` output
- Register `kvCmd` in `cmd/root.go`'s `init()` function

**Files:** `cmd/kv.go`, `cmd/root.go`

### G-07: Update Documentation

**What to do:**
- Update `docs/USAGE.md` with worker and kv command examples
- Update `CLAUDE.md` structure section to mention `pkg/r2go2/worker.go` and `pkg/r2go2/kv.go` as active services
- Verify all `--help` text is useful for agent consumption

**Files:** `docs/USAGE.md`, `CLAUDE.md`

### G-08: Version Bump

**What to do:**
- Update `.version-registry.json` to v0.4.0
- Run `go test ./pkg/... ./cmd/...` and `go vet ./...` to confirm clean build
- Commit as `chore(release): v0.4.0 -- Workers and KV services`

**Files:** `.version-registry.json`

## Implementation Order

Follow TDD -- write tests first for each service, then implementation, then CLI.

1. **G-01 + G-02**: WorkerService library + tests (TDD: tests first, watch them fail, then implement)
2. **G-03**: Worker CLI commands (register in root.go)
3. **G-04 + G-05**: KVService library + tests (same TDD approach)
4. **G-06**: KV CLI commands (register in root.go)
5. **G-07**: Documentation updates
6. **G-08**: Version bump and final verification

## Session Priorities

1. **G-01 + G-02** (WorkerService) -- most complex, do while context is fresh
2. **G-04 + G-05** (KVService) -- second most complex
3. **G-03** (Worker CLI) -- straightforward once library exists
4. **G-06** (KV CLI) -- straightforward once library exists
5. **G-07** (docs) -- can be done quickly at end
6. **G-08** (version) -- final step

## Technical Notes

- **Module path**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Go version**: 1.26
- **Key dependencies**: `aws-sdk-go-v2`, `cloudflare-go`, `spf13/cobra`, `spf13/viper`
- **Test command**: `go test ./pkg/... ./cmd/...`
- **Build command**: `go build -o r2go2 .`
- **Pattern to follow**: `pkg/r2go2/client.go` shows how to hold `*cloudflare.API` + `accountID` and make API calls
- **Error types**: Reuse `R2Error` hierarchy from `errors.go` (keeping the R2 prefix per plan ADR -- renaming to CFError would be a breaking change)
- **Options pattern**: Follow `pkg/r2go2/options.go` for functional option style
- **CLI pattern**: Follow `cmd/bucket.go` for cobra command structure, `--json` output, and `printJSON()` usage
- **Registration**: Add `workerCmd` and `kvCmd` to root's subcommands in `cmd/root.go` init()
- **Skip validation**: Add "worker" and "kv" appropriately to the `skipValidation` list in root's PersistentPreRun
- **cloudflare-go Workers API**: `PutWorkerScript`, `ListWorkerScripts`, `GetWorkerScript`, `DeleteWorkerScript`, `PutWorkerSettings`
- **cloudflare-go KV API**: `CreateWorkersKVNamespace`, `ListWorkersKVNamespaces`, `GetWorkersKVNamespace`, `DeleteWorkersKVNamespace`, `PutWorkersKV`, `GetWorkersKV`, `DeleteWorkersKV`, `ListWorkersKVKeys`
- **Existing stubs**: `worker.go` (9 lines, empty struct), `kv.go` (9 lines, empty struct) -- replace entirely
- **Other Phase 3 stubs exist but are NOT in scope**: `d1.go`, `pages.go`, `queue.go` -- leave them as-is

## Future Quick Wins (if time remains after G-01 through G-08)

These are small, independent items from the roadmap that could be picked up:

| Item | Effort | Description |
|------|--------|-------------|
| ROAD-014 | Small | Retry logic with exponential backoff for API calls |
| ROAD-012 | Small | Shell completions (bash, zsh, fish) |
| ROAD-004 | Small | Pre-signed URL generation for downloads |
| ROAD-009 | Medium | Config profiles system (multiple Cloudflare accounts) |

Do NOT start these until G-01 through G-08 are complete.
