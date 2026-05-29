---
branch: batch2-batch3-quick-wins
completed: "2026-05-29"
created: "2026-05-14T12:00:00-03:00"
goals_completed: 8
goals_total: 8
origin: migrated by ccs prompts migrate
priority: medium
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: 'Batch 2-3: Quick Wins from Roadmap'
---

# Batch 2-3: Quick Wins from Roadmap

**Project**: CosmoDev-R2Go2 (v0.4.0)
**Branch**: master
**Date**: 2026-05-14
**Predecessor**: Phase 3 Workers/KV (complete), Batch 1 retry/completions (dispatched)

## File Scope
```yaml
files_modified:
  - cmd/object.go
  - cmd/bucket.go
  - docs/USAGE.md
files_created:
  - pkg/r2go2/presign.go
  - pkg/r2go2/presign_test.go
  - cmd/compare.go
  - cmd/analytics.go
```

## Context

R2Go2 just shipped Phase 3 (Workers + KV services), expanding beyond R2 storage into the full Cloudflare platform. The project has strong momentum — 57 tests passing, clean build, solid library architecture. Now we're knocking out quick wins from the roadmap to round out the R2 feature set before tackling larger items (sync, TUI refactor). The Batch 1 agents (ROAD-014 retry logic, ROAD-012 shell completions) are in flight and may already be merged by the time this prompt runs.

## GLM Dispatch Rules

When goals involve dispatching subagents:

1. **ALWAYS** use `ccs glm-agent exec` for GLM agents (routes through queue with retry logic)
2. **NEVER** use Agent tool with `model:sonnet` or `model:haiku` for GLM work (bypasses queue, risks 429 rate limits)
3. Agent tool with `model:opus` is fine for Opus subagents
4. For parallel work: use `/glm-sprint` or `ccs glm-agent exec-batch`

## What Got Done

### Phase 3 (2026-05-13 to 2026-05-14)

Shipped Workers and KV services across 4 commits:
- `551a07d` feat(r2go2): add Workers and KV services (Phase 3) — WorkerService (Deploy/List/Get/Delete/Logs/UpdateSettings), KVService (namespace CRUD + key-value ops), CLI with `r2go2 worker` (6 subcommands) and `r2go2 kv` (7 subcommands), 25 new tests
- `349fb20` chore: session metadata
- `739091c` docs: transcripts
- `c34509c` chore: remove dead demo files (TASK-003 resolved)

ROAD-026 (Workers) and ROAD-030 (KV) marked completed in roadmap. 23 roadmap items remain, most still `captured`.

### Batch 1 (in flight)

- **ROAD-014**: Retry logic — Sonnet agent dispatched, creating `pkg/r2go2/retry.go`
- **ROAD-012**: Shell completions — Sonnet agent dispatched, enhancing `cmd/completion.go`
- **TASK-003**: Dead demo files — completed and committed

## Goals

### [x] G-01: Pre-signed URL Generation

**Model:** `glm-turbo` | **Files:** `pkg/r2go2/presign.go`, `pkg/r2go2/presign_test.go`, `cmd/object.go`

Add pre-signed URL generation for temporary download access without exposing credentials.

**Library (pkg/r2go2/presign.go):**
- Add `PresignGetObject(ctx context.Context, bucket, key string, expiresIn time.Duration) (string, error)` to the `R2Client` interface in `client.go`
- Implement using `s3.NewPresignClient(c.s3)` from `aws-sdk-go-v2/service/s3`
- Return the presigned URL string
- Add `PresignOptions` struct with `WithPresignExpires(d time.Duration)` option
- Error wrapping via `newError("PresignGetObject", ...)`

**Tests (pkg/r2go2/presign_test.go):**
- Test `PresignOptions` application
- Test validation: empty bucket, empty key, zero/negative expires
- Target: 5-6 tests

**CLI (cmd/object.go):**
- Add `presign` subcommand to `objectCmd`: `r2go2 object presign <bucket> <key> --expires=1h`
- Default expiry: 1 hour
- `--json` output: `{"url": "https://...", "expires_in": "1h", "key": "..."}`
- Register in init()

**Acceptance:** `go test ./pkg/r2go2/ -run TestPresign -v` passes, `go build .` succeeds

### [x] G-02: Pipe and Stdin Support for Unix Workflows

**Model:** `glm-turbo` | **Files:** `cmd/object.go`

Support reading from stdin when key is `-` and writing to stdout when output is `-`.

**What to do in cmd/object.go:**
- In `runObjectPut`: if the file argument is `-`, read from `os.Stdin` instead of opening a file
- In `runObjectGet`: if `--output` is `-` or not specified and stdout is not a terminal (`!term.IsTerminal(int(os.Stdout.Fd()))`), write to `os.Stdout` directly (skip progress bar)
- Add helper: `func isTerminal(f *os.File) bool` using `golang.org/x/term` or `os.File.Stat()` check
- Update `--help` text for both commands to mention `-` convention

**Acceptance:** `echo "hello" | ./r2go2 object put bucket key -` reads stdin, `./r2go2 object get bucket key` pipes to stdout when not a terminal

### [x] G-03: Bucket Comparison Tool

**Model:** `glm-turbo` | **Files:** `cmd/compare.go`

Create a `r2go2 compare` command that shows differences between two buckets.

**What to do:**
- Create `cmd/compare.go` with cobra command
- Command: `r2go2 compare <source-bucket> <dest-bucket> [--prefix=] [--json]`
- Logic: ListObjects on both buckets, diff by key
- Output categories: `only_in_source`, `only_in_dest`, `different_size`, `same`
- Tabular output by default, `--json` for structured output
- Register `compareCmd` in `cmd/root.go` init()
- Follow `cmd/bucket.go` patterns for output formatting

**Acceptance:** `go build .` succeeds, `./r2go2 compare --help` shows usage, `--json` flag present

### [x] G-04: Analytics Command

**Model:** `glm-turbo` | **Files:** `cmd/analytics.go`

Re-enable analytics command showing R2 usage statistics.

**What to do:**
- Create `cmd/analytics.go` with cobra command
- Command: `r2go2 analytics [--bucket=] [--period=7d] [--json]`
- Logic: ListBuckets, aggregate size + object count, show per-bucket breakdown
- If `--bucket` specified: show that bucket's details with ListObjects for object count
- Period flag is advisory (real Cloudflare Analytics API is Phase 5+)
- Table output: bucket name, size, objects, percentage of total
- JSON output: structured with `--json`
- Register in `cmd/root.go` init(), add "analytics" to skipValidation list

**Acceptance:** `go build .` succeeds, `./r2go2 analytics --help` shows usage

### [x] G-05: Update Documentation

**Model:** `glm-turbo` | **Files:** `docs/USAGE.md`

Update USAGE.md with the new commands added in G-01 through G-04.

**What to do:**
- Add section for `object presign` with examples
- Add pipe/stdin examples under `object put` and `object get`
- Add `compare` command section with examples
- Add `analytics` command section
- Add library example for PresignGetObject

### [x] G-06: Version Bump and Verification

**Model:** `glm-turbo` | **Files:** `.version-registry.json`

- Update `.version-registry.json` to v0.4.1
- Run `go test ./pkg/... ./cmd/` — all tests pass
- Run `go vet ./pkg/r2go2/ ./cmd/` — clean
- Commit as `chore(release): v0.4.1 -- presigned URLs, pipe support, compare, analytics`

## Carry-Over Tasks

- [x] ROAD-014: Retry logic with exponential backoff — Sonnet agent dispatched in previous session, check if `pkg/r2go2/retry.go` exists and tests pass. If agent failed, implement directly. (was: in_progress)
- [x] ROAD-012: Shell completions — Sonnet agent dispatched in previous session, check if `cmd/completion.go` was enhanced. If agent failed, skip (existing completion works). (was: in_progress)

## Where We're Headed

After these quick wins, R2Go2 will have a solid R2 feature set with presigned URLs, pipe support, bucket comparison, and analytics. The next major milestone would be either:
- **ROAD-009**: Config profiles (highest priority at 85) for multi-account support
- **ROAD-003**: rsync-like sync command for directory synchronization
- **Phase 4**: D1 and Pages service stubs (ROAD-031, ROAD-032)

The project is at ~35% roadmap completion. These quick wins push it toward 45% and make the CLI genuinely useful for production workflows.

## Priority Order

1. **G-01** (Pre-signed URLs) — highest value, most requested feature
2. **G-02** (Pipe/stdin) — unlocks Unix composition, small change
3. **G-03** (Bucket compare) — migration validation use case
4. **G-04** (Analytics) — quick to implement, good UX
5. **G-05** (Docs) — update after all features land
6. **G-06** (Version) — final step
