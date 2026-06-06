---
schema_version: 1
status: COMPLETED
type: continuation
created: 2026-05-30T09:30:00-03:00
priority: high
requires_reading:
  - docs/USAGE.md
  - CLAUDE.md
---

# Next Session — Cosmoflare Post-Rename Continuation

## Session State

- **Project**: CosmoDev-R2Go2 (Cosmoflare)
- **Version**: v0.11.0 (build 475)
- **Branch**: master
- **Binary**: `cosmoflare` (r2go2 is the backward-compatible alias)
- **Package**: `pkg/cosmoflare/` — import as `cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"`

## What Was Done Last Session

- Renamed binary and package from r2go2 → cosmoflare
- Implemented 14 new features across all Cloudflare service areas
- Added 170+ tests (cmd registration/flag tests across all subcommands)
- Fixed upgrade/migration issues from the rename
- Project now has comprehensive Cloudflare platform coverage across R2, Workers, KV, DNS, Zones, SSL, Cache, Healthchecks, Diagnostics, Domains

## Remaining Work (Prioritized)

### 1. Release v0.12.0 (HIGH)

14 new features + the binary/package rename warrant a minor version bump. Run:

```
/release minor
```

This is the first action to take. The rename and feature set are release-worthy.

---

### 2. ROAD-017 — Integration Test Suite (MEDIUM, priority 80)

All current cmd tests are registration/flag tests — they verify commands exist and flags are wired, but do not execute command logic. Need real integration tests using mock HTTP servers.

**Approach:**
- Use `net/http/httptest` to spin up mock Cloudflare API servers
- Test actual command execution paths (not just cobra registration)
- Cover happy path + error cases for each service
- File: `cmd/*_integration_test.go` or `cmd/integration_test.go`
- TDD mandatory: write tests first, watch fail, then implement

---

### 3. ROAD-053 — cosmoflare import/export (MEDIUM, priority 78)

Full account configuration as YAML for disaster recovery. Export all services to a single file, import to restore.

**Scope:**
- `cosmoflare export > account.yaml` — dumps all R2, Workers, KV, DNS, Zone settings
- `cosmoflare import account.yaml` — restores from a previously exported YAML
- Useful for: environment cloning, disaster recovery, config-as-code workflows
- New CLI commands in `cmd/` + new library methods in `pkg/cosmoflare/`
- TDD mandatory

---

### 4. ROAD-013 — Multipart Uploads and Resumable Transfers (LARGE, priority 77)

Large file uploads with automatic multipart chunking, resume on failure, and progress tracking.

**Scope:**
- Auto-detect file size; use multipart when size > threshold (e.g. 100MB)
- Chunk uploads in parallel where possible
- Resume: store upload state locally, detect partial uploads, continue from last chunk
- Progress bar in TTY mode, silent in `--json` mode
- Relevant files: `pkg/cosmoflare/upload.go`, `cmd/object.go`
- TDD mandatory

---

### 5. Pre-existing Test Failures (MEDIUM)

Two known pre-existing failures to fix:

1. **`internal/migration/s3.go`** — build errors due to undefined functions. The migration package has stubs that were never completed.
2. **`internal/interactive`** — runtime test failures. Likely from interactive prompts that can't run headlessly in CI.

Investigate with `go build ./...` and `go test ./...` to get the full error list, then fix.

---

### 6. GitHub Repo Rename (WHEN READY — P-02 of rename plan)

After v0.12.0 is released and the codebase is stable:

1. Rename GitHub repo from `CosmoDev-R2Go2` → `cosmoflare` (via GitHub settings)
2. Update `go.mod` module path: `github.com/CosmoLabs-org/cosmoflare`
3. Update all internal imports across the codebase (`sed -i` or global replace)
4. Tag and push

This is a breaking change for any consumers of the library. Coordinate with the v0.12.0 release notes.

---

## Key Constraints

- **Binary name**: `cosmoflare` (root command in `cmd/root.go`); `r2go2` is the alias — both must work
- **Package path**: `pkg/cosmoflare/` — do NOT use `pkg/r2go2/`
- **Import alias**: `cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"`
- **TDD mandatory**: for ALL new features — write tests first, watch them fail, then implement
- **Agent-first UX**: every command needs `--help` with examples, `--json` output, actionable error messages
- **Bun** for any JS tooling; `go build -o cosmoflare .` for the binary
- No `Co-Authored-By` in commits; author is `GΛB <Gab@CosmoLabs.org>`

## Suggested Session Start

1. Run `go build -o cosmoflare . && go test ./...` to confirm current state
2. Run `/release minor` to cut v0.12.0
3. Pick the next item from the priority list above
4. Use `/triage` if unsure what to tackle next
