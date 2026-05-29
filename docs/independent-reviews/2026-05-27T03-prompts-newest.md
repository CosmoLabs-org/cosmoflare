---
reviewed_files:
  - docs/prompts/2026-05-27-next-session.md
  - docs/prompts/2026-05-19-post-audit-fixes.md
  - docs/prompts/2026-05-16-cosmoflare-phase4-rebrand.md
reviewed_at: "2026-05-29T12:00:00-03:00"
mode: scan
findings_total: 18
findings_critical: 3
findings_major: 5
findings_minor: 10
fixes_applied: 7
dimensions_checked: 10
reviewer: opus
---

# Independent Review: 3 Newest Pending Prompts

## Summary

Three continuation prompts were reviewed against a 10-dimension checklist. Two of the three
(post-audit-fixes and phase4-rebrand) were still marked PENDING despite all their work being
completed and merged weeks ago. The third (next-session) contained a broken `requires_reading`
reference and stale version numbers. Seven fixes were applied inline (3 critical, 4 major).
Ten minor observations are documented below for awareness.

The dominant failure pattern across all three files is **staleness** -- prompts written at the
end of one session that are never updated when subsequent sessions complete the work. This
creates a misleading project state where `ccs prompts` reports PENDING work that is actually
done.

---

## File 1: `docs/prompts/2026-05-27-next-session.md`

**Status before review**: PENDING (still valid -- work not yet started)

### Critical Findings (fixed)

**C-01: Broken `requires_reading` reference** (Dim 8: Interface mismatches)
- `requires_reading` listed `.claude/CLAUDE.md` which does not exist. The project CLAUDE.md
  lives at `./CLAUDE.md` (root). If this prompt were executed via `ccs prompts load-context`,
  it would fail with a hard error (missing file exits non-zero per ADR-005).
- **Fix applied**: Removed `.claude/CLAUDE.md` from `requires_reading`.

**C-02: Stale version number** (Dim 4: Contradiction scan)
- Context section stated `v0.10.0 (build 417)` but the actual version is `v0.11.0 (build 423)`
  per `.version-registry.json`. The v0.11.0 release commit (`2bde4a2`) predates this prompt.
- **Fix applied**: Updated to `v0.11.0 (build 423)`.

### Major Findings (fixed)

**M-01: Wrong next-version guidance** (Dim 4: Contradiction scan)
- Version Note section said "Next release should be v0.11.0" but v0.11.0 is already released.
  A session following this prompt would either skip the release (thinking it was already done)
  or attempt to re-release the same version.
- **Fix applied**: Updated to "Current version is v0.11.0. Next release should be v0.12.0."

### Minor Findings (noted)

**m-01: Missing `title` field in frontmatter** (Dim 10: Frontmatter hygiene)
- No `title:` field. Other prompts in this project include it. Not blocking but inconsistent.

**m-02: Missing `goals_completed`/`goals_total` in frontmatter** (Dim 10: Frontmatter hygiene)
- The prompt has 5 numbered tasks but no machine-readable goal tracking in frontmatter.
  `ccs prompts` cannot report progress without these fields.

**m-03: ROAD-058/051/057 are `captured` not `planned`** (Dim 6: Feasibility check)
- The roadmap items referenced (ROAD-058, ROAD-051, ROAD-057) are still in `captured` status.
  Convention is to move them to `planned` before execution. Not a prompt defect per se, but
  the session should update roadmap status early.

**m-04: CCS upgrade commands may not exist** (Dim 2: Impossible operations)
- Task 5 lists `ccs upgrade --subsystem doc-structure` and five other subsystems. These
  commands are project-specific and may fail if the CCS version has changed since the prompt
  was written. The prompt provides no fallback if a subsystem fails.

---

## File 2: `docs/prompts/2026-05-19-post-audit-fixes.md`

**Status before review**: PENDING (should have been COMPLETED)

### Critical Findings (fixed)

**C-03: Status was PENDING but all work is done** (Dim 4: Contradiction scan)
- All 7 audit fix tasks were completed and merged in commits `8fc340c` ("sort multipart parts
  and classify NoSuchKey as ErrNotFound") and `258e9d5` ("audit phase 0 -- install.sh case,
  release paths, atomic config write, bucket update no-op"). The `install.sh` BINARY_NAME is
  now lowercase `r2go2`. The prompt was never marked complete.
- **Fix applied**: Changed status to COMPLETED via `ccs prompts complete`. Set goals to 7/7.

### Major Findings (fixed)

**M-02: Missing `goals_completed`/`goals_total` fields** (Dim 10: Frontmatter hygiene)
- These fields were absent, preventing `ccs prompts` from reporting completion percentage.
- **Fix applied**: Added `goals_completed: 7`, `goals_total: 7` via `ccs prompts set-goals`.

### Minor Findings (noted)

**m-05: Inconsistent timezone in `created`** (Dim 10: Frontmatter hygiene)
- `created: 2026-05-19T00:00:00-05:00` uses UTC-5, while the project consistently uses
  UTC-3 (`-03:00`). This suggests the timestamp was authored from a different machine or
  timezone was guessed. Not functionally broken but inconsistent.

**m-06: Acceptance criteria checkboxes are plain `- [ ]` markdown** (Dim 1: Structural gaps)
- The acceptance criteria at the bottom use `- [ ]` checkboxes that were never checked off.
  Since the work is done, these create a visual contradiction with the COMPLETED status.
  However, editing them is low priority since the prompt is now archived.

**m-07: Worktree isolation advice is overly prescriptive** (Dim 9: Duplication & bloat)
- Execution Notes suggest separate worktrees per fix group. For 7 small fixes (most are
  1-3 line changes), this creates unnecessary overhead. The actual session merged them in
  two commits, which was the right call.

**m-08: Test count claim "997+ tests"** (Dim 5: Assumption surfacing)
- The prompt claims 997+ tests should pass. This number was accurate at time of writing but
  may have drifted. Not wrong, but a fixed number in a prompt creates a false expectation
  if tests are added or removed.

---

## File 3: `docs/prompts/2026-05-16-cosmoflare-phase4-rebrand.md`

**Status before review**: PENDING (should have been COMPLETED -- 8/8 goals done)

### Major Findings (fixed)

**M-03: Status was PENDING despite 8/8 goals completed** (Dim 4: Contradiction scan)
- Frontmatter had `goals_completed: 8` and `goals_total: 8` but `status: PENDING`. All 8
  deliverables are implemented: DNS (`cmd/dns.go`, `pkg/r2go2/dns.go`), Zones (`cmd/zone.go`,
  `pkg/r2go2/zone.go`), SSL (`cmd/ssl.go`, `pkg/r2go2/ssl.go`), Cache (`cmd/cache.go`,
  `pkg/r2go2/cloudflare_cache.go`), tests (3 of 4 test files exist), shell completion
  (`cmd/completion.go`), and ROAD-035/036/037/038/055 are all `completed` in roadmap.
- **Fix applied**: Changed status to COMPLETED via `ccs prompts complete`.

**M-04: Date-only timestamps violating ISO8601 constitutional rule** (Dim 10: Frontmatter hygiene)
- `completed: "2026-05-29"` and `started: "2026-05-16"` used date-only format, violating
  the IMP-032 constitutional rule (86,400 seconds of lost precision).
- **Fix applied**: `ccs timestamps backfill --apply` upgraded to full ISO8601+TZ.

**M-05: Body goal checkboxes were all `[ ]` despite 8/8 complete** (Dim 4: Contradiction scan)
- All 8 `### [ ] P-0X` headings showed unchecked boxes while frontmatter said 8/8 complete.
- **Fix applied**: `ccs prompts complete` updated all checkboxes to `[x]`.

### Minor Findings (noted)

**m-09: P-04 references `pkg/r2go2/cache.go` but actual file is `cloudflare_cache.go`**
(Dim 8: Interface mismatches)
- The prompt says `Files: pkg/r2go2/cache.go, cmd/cache.go` but the actual library file is
  `pkg/r2go2/cloudflare_cache.go`. This is a naming deviation from the prompt's plan. Not
  harmful now (work is done) but would have confused an agent following the prompt literally.

**m-10: P-05 references `pkg/r2go2/cache_test.go` but actual files differ** (Dim 8: Interface mismatches)
- The prompt expected `cache_test.go` but the actual test files are
  `cloudflare_cache_test.go` and `cache_tier_test.go` (plus `cmd/cache_test.go`). Also,
  `pkg/r2go2/cache_test.go` does not exist -- only 3 of the 4 expected test files were
  created (`dns_test.go`, `zone_test.go`, `ssl_test.go`). The cache service tests live
  under different names.

---

## Dimension Coverage Matrix

| # | Dimension | File 1 | File 2 | File 3 |
|---|-----------|--------|--------|--------|
| 1 | Structural gaps | Clean | m-06 | Clean |
| 2 | Impossible operations | m-04 | Clean | Clean |
| 3 | Missing error paths | Clean | Clean | Clean |
| 4 | Contradiction scan | C-02, M-01 | C-03 | M-03, M-05 |
| 5 | Assumption surfacing | Clean | m-08 | Clean |
| 6 | Feasibility check | m-03 | Clean | Clean |
| 7 | Scope creep detection | Clean | Clean | Clean |
| 8 | Interface mismatches | C-01 | Clean | m-09, m-10 |
| 9 | Duplication & bloat | Clean | m-07 | Clean |
| 10 | Frontmatter hygiene | m-01, m-02 | M-02, m-05 | M-04 |

## Fixes Applied Summary

| ID | File | Fix | Method |
|----|------|-----|--------|
| C-01 | next-session | Removed broken `.claude/CLAUDE.md` from requires_reading | Edit |
| C-02 | next-session | Version `v0.10.0 (417)` -> `v0.11.0 (423)` | Edit |
| C-03 | post-audit-fixes | Status PENDING -> COMPLETED, goals 7/7 | `ccs prompts complete` + `set-goals` |
| M-01 | next-session | Next release `v0.11.0` -> `v0.12.0` | Edit |
| M-03 | phase4-rebrand | Status PENDING -> COMPLETED | `ccs prompts complete` |
| M-04 | phase4-rebrand | Date-only timestamps -> ISO8601+TZ | `ccs timestamps backfill --apply` |
| M-05 | phase4-rebrand | All `[ ]` checkboxes -> `[x]` | `ccs prompts complete` (automatic) |
