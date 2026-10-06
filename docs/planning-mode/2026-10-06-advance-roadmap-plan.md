---
title: "Roadmap Acceleration Plan — 2026-10-06"
created: "2026-10-06T21:05:00+04:00"
status: PENDING
branch: master
schema_version: 1
deliverables:
  - id: P-01
    title: "Dispatch-batch analysis for all open issues (result: batch empty — all items gated)"
  - id: P-02
    title: "Roadmap maintenance actions applied/recommended (ROAD-100 completed; ROAD-099 verify; ROAD-085 review)"
  - id: P-03
    title: "Next-wave plan: post-FEAT-044 candidates and their gates"
---

# Roadmap Acceleration Plan — 2026-10-06

## State

Project: cosmoflare (v0.31.0) | Open: 0 bugs, 2 features, 1 task
Roadmap: 78% (88/111 after ROAD-100 completion)
Capacity: 1 agent slot in use (GLM 0314, FEAT-044 in flight); pool otherwise idle

## Dispatch Plan

**Result: the parallel dispatch batch is EMPTY.** Every open item is gated:

| # | ID | Title | Score | Model | Class | Gate |
|---|----|-------|-------|-------|-------|------|
| 1 | FEAT-044 | knowledge.Transport chokepoint wiring | 48 | glm-5.3 | — | **IN FLIGHT** — agent 0314 running now; S334 gate after |
| 2 | FEAT-029 | Auth modernization (token retrieval + device-flow) | 29 | opus-only | cross-cutting | Security-sensitive (credential redaction) + operator decision needed on OAuth client registration before the device-flow half |
| 3 | TASK-010 | cosmolabs.org product page | — | — | — | Operator session in ~/PROJECTS/cosmolabs.org (cross-project, hook-blocked here) |

Skipped (need plan first): none — no complex unplanned candidates remain.

Sequential queue: FEAT-044 (in flight) → merge → then FEAT-029's `auth token`
half CAN be specced for Opus (not GLM — credential-handling), pending the
operator's go.

## Roadmap Gaps (untracked work)

| Score | ID | Category | Title | Action |
|-------|----|----------|-------|--------|
| 48 | FEAT-044 | workflow | knowledge.Transport wiring | Already in flight; link a roadmap item post-merge via `ccs roadmap link-issue` |
| 29 | FEAT-029 | infra | Auth modernization | Roadmap item after the OAuth registration decision |

## Roadmap Health Actions

- ✅ APPLIED: ROAD-100 → completed (FEAT-041 shipped in v0.31.0; health check confirmed orphaned)
- ⚠ VERIFY: ROAD-099 "Production-evidence reliability loop" — all 3 linked issues closed (FEAT-018/011/014) but reads as an ongoing campaign. Verify sole scope before completing; do NOT blanket `--fix`.
- 🔍 REVIEW: ROAD-085 secrets hygiene — 3 recent commits match keywords; review and update notes/status.
- 18 THIN items (BASE-*) — cosmetic; batch-link or archive in a maintenance pass.

## Next Wave (post-FEAT-044)

1. Merge FEAT-044 through the S334 gate → close issue → changelog.
2. Operator: decide OAuth client registration → unblocks FEAT-029 spec (Opus-tier).
3. Operator: TASK-010 session in cosmolabs.org (handoff doc ready).
4. Optional maintenance wave (GLM flash, parallel): BASE-* thin-item linking + ROAD-085 notes — bounded, no code.
