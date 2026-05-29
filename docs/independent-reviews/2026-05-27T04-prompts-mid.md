---
reviewed_files:
  - docs/prompts/2026-05-14-batch2-batch3-quick-wins.md
  - docs/prompts/2026-05-14-session-continuation.md
  - docs/prompts/2026-05-08-phase1-library-extraction.md
reviewed_at: "2026-05-29T12:00:00-03:00"
mode: scan
findings_total: 19
findings_critical: 2
findings_major: 5
findings_minor: 12
fixes_applied: 3
dimensions_checked: 10
reviewer: opus
---

# Independent Review: 3 Continuation Prompts (Mid-Cycle Scan)

## Summary

Three continuation prompts were reviewed against a 10-dimension critique checklist. Two of
the three had incorrect status fields: one had 8/8 goals completed but remained PENDING,
and another was 21 days old with all work long shipped but still marked "ready." Both were
fixed inline. The third prompt (session-continuation) is a valid open menu prompt but
contains stale data throughout -- version numbers, test counts, bug references, and roadmap
percentages are all significantly out of date relative to the current project state
(v0.11.0, 4,267 tests). No prompt references any `requires_reading` files, which means
none participate in the three-tier documentation chain enforced by ADR-005.

---

## File 1: `docs/prompts/2026-05-14-batch2-batch3-quick-wins.md`

### Fixes Applied

| # | Dimension | Severity | Finding | Action |
|---|-----------|----------|---------|--------|
| F1-1 | (10) Frontmatter hygiene | Critical | `status: PENDING` with `goals_completed: 8` / `goals_total: 8` -- contradictory. All 8 goals marked `[x]` in body. | Fixed: set status to COMPLETED via `ccs prompts complete`. |

### Observations (no fix needed)

| # | Dimension | Severity | Finding |
|---|-----------|----------|---------|
| F1-2 | (4) Contradiction scan | Major | Body says "v0.4.0" and references version bump to "v0.4.1" (G-06). Project is now at v0.11.0. The version claims are historically accurate for when the prompt was written, but someone re-running this prompt would get confused. Mitigated by COMPLETED status. |
| F1-3 | (4) Contradiction scan | Minor | Body header says `goals_total: 8` in frontmatter but body lists 6 numbered goals (G-01 through G-06) plus 2 carry-over tasks. The frontmatter counts carry-overs as goals, but the body separates them under "Carry-Over Tasks." Inconsistent taxonomy. |
| F1-4 | (6) Feasibility check | Minor | G-06 acceptance criteria says "Run `go test ./pkg/... ./cmd/`" but the project has tests in `tests/unit/`, `tests/integration/`, and other paths. This would miss a significant portion of the test suite. Moot since completed. |
| F1-5 | (5) Assumption surfacing | Minor | G-02 references `golang.org/x/term` for terminal detection but does not verify it is in `go.mod`. Assumes dependency availability. |
| F1-6 | (9) Duplication & bloat | Minor | GLM Dispatch Rules section is duplicated verbatim in the session-continuation prompt (file 2). |
| F1-7 | (10) Frontmatter hygiene | Minor | `completed: "2026-05-29"` uses date-only format, violating the ISO8601+TZ constitutional rule. This was written by `ccs prompts complete` itself -- indicates a ccs tooling bug, not a prompt authoring error. |
| F1-8 | (1) Structural gaps | Minor | `requires_reading: []` -- prompt does not participate in the three-tier chain (ADR-005). No linked brainstorm or plan document. |

---

## File 2: `docs/prompts/2026-05-14-session-continuation.md`

### Fixes Applied

None. Status PENDING with 0/0 goals is correct for a menu-style prompt that presents
options for the next session to choose from. However, several major issues exist.

### Observations

| # | Dimension | Severity | Finding |
|---|-----------|----------|---------|
| F2-1 | (4) Contradiction scan | Major | Header says "v0.6.0, build 104" but the project is now at v0.11.0, build 423. Every version reference in this file is stale. |
| F2-2 | (4) Contradiction scan | Major | Claims "49% roadmap completion (17/35 items done)" and "57+ tests." The project now has 4,267 test functions. The roadmap has expanded well beyond 35 items. All numerical claims are outdated by 15 days of active development. |
| F2-3 | (2) Impossible operations | Major | BUG-001 "speed calculation" references `pkg/r2go2/enhanced_client.go` with `time.Since(time.Now())`. This file does not appear in the current `pkg/r2go2/` listing (57 files). Either the bug was fixed and the file was renamed/removed, or the path is wrong. Either way, following this prompt's bug fix instructions would fail. |
| F2-4 | (3) Missing error paths | Minor | Option D (ROAD-013 Multipart Uploads) says "Consider splitting across sessions" but provides no guidance on how to split, what the boundary is, or how to hand off. |
| F2-5 | (5) Assumption surfacing | Minor | "Predecessor: Batch 2-3 Quick Wins (COMPLETED)" is stated in the body but the frontmatter had no `related_prompts` linking to the predecessor. The relationship exists only in prose, not in machine-readable form. |
| F2-6 | (7) Scope creep detection | Minor | Lists 10 roadmap items, 3 category coverage gaps, 3 pre-existing bugs, 4 open issues, and 4 session goal options (A-D) with sub-goals. This is a sprawling scope menu with no prioritization constraint -- a session could easily overcommit. |
| F2-7 | (8) Interface mismatches | Minor | Suggests `cmd/config.go` for Option B (ROAD-009 Config Profiles) but `cmd/config.go` already exists at 16.9K. The prompt frames it as new creation rather than modification. |
| F2-8 | (1) Structural gaps | Minor | `requires_reading: []` -- no three-tier chain participation. |
| F2-9 | (9) Duplication & bloat | Minor | "GLM Dispatch Rules" section is nearly identical to file 1, plus an extra R2Go2-specific note. Could be a shared include. |
| F2-10 | (10) Frontmatter hygiene | Minor | `branch: session-continuation` -- this branch does not exist in git. The prompt was never executed on a named branch. Cosmetic but misleading. |

---

## File 3: `docs/prompts/2026-05-08-phase1-library-extraction.md`

### Fixes Applied

| # | Dimension | Severity | Finding | Action |
|---|-----------|----------|---------|--------|
| F3-1 | (10) Frontmatter hygiene | Critical | `status: ready` -- non-standard status value (valid values: PENDING, STARTED, COMPLETED). All Phase 1 work shipped weeks ago (v0.11.0). Missing `goals_completed`, `goals_total`, `title`, `tags`, `related_prompts`, `requires_reading` fields. | Fixed: set goals 10/10, status COMPLETED via `ccs prompts set-goals` and `ccs prompts set-status`. CCS auto-populated missing fields. |

### Observations

| # | Dimension | Severity | Finding |
|---|-----------|----------|---------|
| F3-2 | (4) Contradiction scan | Major | Body says "145 tests" passing. Project now has 4,267 test functions. The claim was accurate at authoring time but is stale by 30x. |
| F3-3 | (2) Impossible operations | Minor | Goal 2 says "Migrate the working implementation from `internal/api_disabled/client.go`." If that file no longer exists (likely removed after migration), anyone attempting to re-execute this prompt would fail at this step. Mitigated by COMPLETED status. |
| F3-4 | (8) Interface mismatches | Minor | Success criteria includes `r2go2 upload logo.png` but the actual CLI command is `r2go2 object put`. The prompt uses a simplified alias that does not exist in the current command structure. |
| F3-5 | (5) Assumption surfacing | Minor | Lists D1, Pages, Queues as "stub files" for future phases. All three are now fully implemented (d1.go: 5.7K, pages.go: 7.3K, queue.go: 9.1K). The stubs assumption is superseded. |
| F3-6 | (1) Structural gaps | Minor | No `File Scope` YAML block listing files_modified/files_created (present in file 1 and file 2). Inconsistent prompt structure across the set. |
| F3-7 | (10) Frontmatter hygiene | Minor | `completed: "2026-05-29"` written by ccs uses date-only format -- same ccs tooling bug as file 1. |
| F3-8 | (6) Feasibility check | Minor | Execution Order says "Steps 1-6 can be parallelized after step 1 completes" but steps have data dependencies (config.go is needed by storage.go for credential loading). The parallelization claim is optimistic. |

---

## Cross-File Findings

| # | Dimension | Severity | Finding |
|---|-----------|----------|---------|
| X-1 | (9) Duplication | Minor | GLM Dispatch Rules section appears in files 1 and 2 with ~90% overlap. Should be a shared reference or omitted from completed prompts. |
| X-2 | (10) Frontmatter | Minor | `completed` timestamps from `ccs prompts complete` use date-only format (`"2026-05-29"`) on files 1 and 3. This violates the ISO8601+TZ constitutional rule in `rules/determinism.md`. The bug is in the `ccs` binary, not in the prompts themselves. Consider filing as a ccs bug. |
| X-3 | (1) Structural gaps | Minor | None of the three prompts populate `requires_reading`. The three-tier documentation chain (ADR-005) is not exercised by any of these prompts. They exist as standalone artifacts with no machine-verifiable links to brainstorms or plans. |

---

## Dimension Coverage Matrix

| Dimension | File 1 | File 2 | File 3 |
|-----------|--------|--------|--------|
| (1) Structural gaps | F1-8 | F2-8 | F3-6 |
| (2) Impossible operations | -- | F2-3 | F3-3 |
| (3) Missing error paths | -- | F2-4 | -- |
| (4) Contradiction scan | F1-2, F1-3 | F2-1, F2-2 | F3-2 |
| (5) Assumption surfacing | F1-5 | F2-5 | F3-5 |
| (6) Feasibility check | F1-4 | -- | F3-8 |
| (7) Scope creep detection | -- | F2-6 | -- |
| (8) Interface mismatches | -- | F2-7 | F3-4 |
| (9) Duplication & bloat | F1-6 | F2-9 | -- |
| (10) Frontmatter hygiene | F1-1, F1-7 | F2-10 | F3-1, F3-7 |

All 10 dimensions produced at least one finding.

---

## Recommendations

1. **Mark file 2 as SUPERSEDED** -- its data is too stale to be useful. A session picking it up would work from wrong version numbers, nonexistent bug paths, and outdated roadmap percentages. If the menu-style format is still wanted, create a fresh prompt with current project state.
2. **File a ccs bug** for `completed` timestamps using date-only format. The `ccs prompts complete` command should write full ISO8601+TZ.
3. **Establish a prompt expiry policy** -- prompts older than 14 days with status PENDING should be flagged by `ccs prompts lint` or `/triage` for staleness review.
