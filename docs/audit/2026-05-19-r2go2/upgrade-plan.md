# Upgrade Plan

**Source**: Synthesized from all 5 agent reports
**Baseline**: v0.9.0 (score 70.8/100)
**Target**: v1.0.0 (score 85+/100, production-ready)

---

## Phase 0: Critical Fixes (1-2 days)

**Goal**: Eliminate data corruption risk, security hole, and broken delivery.

### 0.1 Fix Multipart Sort Bug (CRIT-1)

- **File**: `pkg/r2go2/upload.go:253`
- **Action**: Fix the sort comparator to sort parts by part number numerically
- **Test**: Add test that uploads a multi-part file (>5MB synthetic) and verifies byte-for-byte integrity
- **Source**: agent-2-core-logic.md

### 0.2 Fix Config Permissions Race (CRIT-2)

- **File**: `internal/config/config.go`
- **Action**: Use `os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)` to create config with correct permissions from the start. Alternatively, write to temp file and `os.Rename` into place.
- **Test**: Verify newly created config file has mode `0600` before any content is written
- **Source**: agent-2-core-logic.md

### 0.3 Fix install.sh Binary Name (HIGH-3)

- **File**: `install.sh`
- **Action**: Change uppercase binary name reference to lowercase `r2go2`
- **Test**: Run `bash -n install.sh` for syntax, manual test on clean system
- **Source**: agent-4-infrastructure.md

### 0.4 Fix CI Pipeline (CRIT-3)

- **Files**: `.github/workflows/*.yml`, any files flagged by `go vet`
- **Action**: Fix all `go vet` errors, push a commit, verify CI passes
- **Test**: CI pipeline runs green on the fix commit
- **Source**: agent-4-infrastructure.md

### 0.5 Fix Download Range Bug (HIGH-1)

- **File**: `pkg/r2go2/download.go:53`
- **Action**: Add `rangeSet bool` field to download options. Only apply `Range` header when `rangeSet` is true.
- **Test**: Verify download without explicit range does NOT send `Range` header
- **Source**: agent-1-code-quality.md

### 0.6 Fix S3 Error Misclassification (CRIT-2)

- **Files**: `pkg/r2go2/storage.go:158`, `pkg/r2go2/download.go:59`
- **Action**: Inspect underlying S3 error codes. Only wrap as `R2NotFoundError` for `NoSuchKey`/HTTP 404. Map 403 to `R2AuthError`, 429 to `R2RateLimitError`, 500 to `R2Error`.
- **Test**: Add tests asserting correct error types for simulated 403, 404, 429, 500 responses
- **Source**: agent-2-core-logic.md

### 0.7 Fix Double defer Close (CRIT-6)

- **File**: `cmd/object.go:389,391`
- **Action**: Remove duplicate `defer obj.Content.Close()` at line 391
- **Test**: Code review verification (single-line fix)
- **Source**: agent-3-api-design.md

### 0.8 Fix Bucket Update No-Op (CRIT-4)

- **File**: `cmd/bucket.go:353-380`
- **Action**: Either implement the actual update API call or remove the command and return a clear "not implemented" error
- **Test**: Verify the command either updates or returns an error
- **Source**: agent-3-api-design.md

**Phase 0 exit criteria**: All 8 fixes merged, CI green, no critical bugs remaining.

---

## Phase 1: Foundation (1 week)

**Goal**: Reliable releases, correct error behavior, contributor readiness.

### 1.1 Adopt goreleaser Properly

- Fix asset path mismatch in `.goreleaser.yml` (MED-2)
- Add `-ldflags="-s -w"` to strip binaries (MED-8, reduces ~33.6MB to ~15-20MB)
- Verify release artifacts are complete and correctly named
- **Source**: agent-4-infrastructure.md

### 1.2 Fix Error Classification

- Audit all error wrapping sites in `pkg/r2go2/`
- Ensure network errors are not wrapped as `R2ValidationError`
- Consider adding `R2NetworkError` type for transient failures
- Add tests for error type assertions on simulated network failures
- **Source**: agent-2-core-logic.md

### 1.3 Fix printError in JSON Mode

- Change `printError` to output `{"error": "message", "code": "type"}` when `--json` is active
- Ensure non-zero exit code accompanies JSON error output
- **Source**: agent-3-api-design.md

### 1.4 Fix Bucket Update No-Op

- Either implement the update logic or remove the command
- If removing, add a clear error message pointing to available operations
- **Source**: agent-3-api-design.md

### 1.5 Add CONTRIBUTING.md

- Include: setup instructions, coding standards, PR process, testing expectations
- Reference the existing CLAUDE.md for architecture context
- **Source**: agent-5-docs-roadmap.md

### 1.6 Update README Service Table

- Mark DNS, Zones, SSL, Cache, Healthchecks, Doctor as "Implemented"
- Fix ROAD-035 title mismatch
- **Source**: agent-5-docs-roadmap.md

### 1.7 Add Config Tests

- Write unit tests for `internal/config/` covering:
  - Profile CRUD (create, read, update, delete)
  - Credential serialization/deserialization
  - Missing file handling
  - Corrupt YAML handling
  - Legacy `.r2go2.yaml` migration
- **Source**: agent-1-code-quality.md

**Phase 1 exit criteria**: goreleaser produces correct artifacts, error classification is audited, README is accurate, config has tests.

---

## Phase 2: Quality (2-3 weeks)

**Goal**: Improve library ergonomics, complete CLI coverage, reduce debt.

### 2.1 Unified Service Client

- Add accessor methods on `R2Client`: `client.DNS()`, `client.KV()`, `client.Workers()`, etc.
- Lazy initialization with shared configuration
- Maintain backward compatibility with standalone `NewXxxService()` constructors
- **Source**: agent-1-code-quality.md

### 2.2 Pagination on All List Commands

- Implement cursor-based pagination for: `kv list`, `dns list`, and any other commands missing it
- Add `--limit` and `--cursor` flags
- Auto-paginate by default, show page info
- **Source**: agent-3-api-design.md

### 2.3 DNS Update Missing Flags

- Add CLI flags for all updateable DNS record fields
- Ensure parity between library API and CLI capabilities
- **Source**: agent-3-api-design.md

### 2.4 Delete Dead Code

- Remove `internal/migration/` (~2,500 lines)
- Remove `cmd_disabled/` (~1,400 lines)
- Tag commit before deletion for archival reference
- **Source**: agent-1-code-quality.md

### 2.5 Extract CLI Global State

- Move global flag variables into a `CommandContext` struct
- Pass through Cobra command tree
- Enables parallel testing and embedding
- **Source**: agent-1-code-quality.md

### 2.6 Fix O(n) Lookups

- Replace `GetBucket` list-and-scan with direct API lookup
- Same for `BucketExists`
- **Source**: agent-1-code-quality.md

### 2.7 Remove Legacy Command Aliases

- Deprecate `r2go2 create`, `r2go2 delete`, `r2go2 copy` top-level aliases
- Add deprecation warnings pointing to the proper subcommand paths
- **Source**: agent-3-api-design.md

### 2.8 Remove os.Exit from Library Code

- Replace `os.Exit()` calls in `pkg/r2go2/` with returned errors
- Library code must never terminate the process
- **Source**: agent-1-code-quality.md

**Phase 2 exit criteria**: Unified client available, pagination complete, dead code removed, CLI state extracted.

---

## Phase 3: Growth (ongoing)

**Goal**: Expand platform coverage, build community, prepare for 1.0.

### 3.1 Wire TUI to Live Data

- Connect dashboard to real API calls with caching
- Show "no data" / "configure credentials" state when not authenticated
- **Source**: agent-1-code-quality.md

### 3.2 Implement Remaining Services

- Page Rules (ROAD-039)
- WAF/Firewall (ROAD-040)
- Email Routing (ROAD-041)
- D1 Database (ROAD-042) -- currently stub
- Pages (ROAD-043) -- currently stub
- **Source**: agent-5-docs-roadmap.md

### 3.3 Homebrew Tap

- Create `CosmoLabs-org/homebrew-tap` repository
- Add formula for `cosmoflare` / `r2go2`
- **Source**: agent-4-infrastructure.md

### 3.4 Community Building

- GitHub issue templates (bug, feature, question)
- GitHub Discussions enabled
- First release blog post
- **Source**: agent-5-docs-roadmap.md

### 3.5 Workers Dev Server

- Local development server for Workers testing
- Hot reload on script changes
- **Source**: agent-5-docs-roadmap.md

**Phase 3 exit criteria**: v1.0.0 release with all Phase 0-2 items complete, 5+ implemented services beyond R2, Homebrew installation available.

---

## Score Projections

| Phase | Projected Score | Grade |
|-------|----------------|-------|
| Current (v0.9.0) | 70.8 | C+ |
| After Phase 0 | 76 | C+ |
| After Phase 1 | 82 | B |
| After Phase 2 | 88 | B+ |
| After Phase 3 | 92+ | A |
