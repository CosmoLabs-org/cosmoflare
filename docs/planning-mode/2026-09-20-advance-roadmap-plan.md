---
title: Roadmap Acceleration Plan — 2026-09-20
created: 2026-09-20
type: plan
mode: plan-only
deliverables:
  - id: P-01
    title: "Dispatch batch of 5 green candidates with model + rationale"
    check: "Table lists FEAT-020-w3, TASK-013, FEAT-029, FEAT-035, FEAT-036 with scores, models, conflict analysis"
  - id: P-02
    title: "Sequential queue for cross-cutting TASK-011/012 with dependency reasoning"
    check: "Queue section names the shared-file ordering constraint"
  - id: P-03
    title: "Skip/block classification for FEAT-011/018/037 and TASK-010"
    check: "Each skipped item names its blocking reason"
  - id: P-04
    title: "Roadmap gaps + health findings with recommended ccs actions"
    check: "Gaps table + orphaned/drifted items with fix commands"
---

# Roadmap Acceleration Plan

Project: cosmoflare | Open: 0 bugs, 9 features, 4 tasks | Roadmap: 75% (83/110)
Capacity: 5 agent slots (max 5 − 0 active GLM agents; 4 read-only review agents running, no worktree usage)

## Dispatch Batch (green — no shared files)

| # | ID | Title | Score | Model | Class | Rationale |
|---|----|-------|-------|-------|-------|-----------|
| 1 | FEAT-020 | Registry wave 3 — email/waf/ssl/cache/hyperdrive/alerts groups | 78 | glm-5.3 | cross-file, single agent | Mechanical data entry vs manifest_test invariants; wave-2 precedent is the spec; permissions from landed qwen-results.md; ALL entries in data.go → one agent only, never parallel |
| 2 | TASK-013 | AlertService disabled-rule API gap | 72 | glm-5.3 | indep | Small, exact files (pkg alerts + cmd/alerts.go), fix direction filed, filed this week |
| 3 | FEAT-029 | Auth modernization: `auth token` + device-flow login | 68 | glm-5.3 (Opus spec first) | indep | cmd/auth* files unique; security-sensitive → spec must pin redaction conventions; wrangler precedent documented in issue |
| 4 | FEAT-035 | Cloudflare Tunnels management CLI | 66 | glm-5.3 (Opus spec first) | indep | New files (pkg tunnel.go + cmd/tunnel.go); endpoints in coverage-catalog-draft.json; high differentiation |
| 5 | FEAT-036 | Account management + audit logs | 64 | glm-5.3 (Opus spec first) | indep | New files; security-layer cornerstone; feeds audit trail |

## Conflict Analysis

- FEAT-020 wave 3 writes `internal/cmdmanifest/data.go` exclusively — no other candidate touches it. Single agent mandatory (S313: one prompt, one file set).
- TASK-013 touches `cmd/alerts.go`; no other batch member does. Green.
- FEAT-029/035/036 each introduce disjoint file sets. Green.

## Sequential Queue (cross-cutting — after batch settles)

1. TASK-011 (cobra Args-level prefix mechanism) — touches many cmd/*.go including `cmd/alerts.go`; MUST follow TASK-013 merge to avoid conflict.
2. TASK-012 (unify three HTTP-status seams) — touches internal/ + pkg/ shared classification; run alone after TASK-011.

## Skipped (need /brainplan first)

- FEAT-018 (MyCarGuide production friction) — complex, multi-service, no planning doc.
- FEAT-037 (zone long-tail batch) — complex breadth feature; issue itself proposes wave order → plan it.

## Blocked (external dependency)

- FEAT-011 (permission names drift) — waits on G-03 research ingestion (Grok/Gemini packs unopened). Entry point: docs/prompts/2026-09-17-cf-perms-next-waves-tiers-research-ingestion.md.
- TASK-010 (product page) — operator-side, ~/PROJECTS/cosmoflare → cosmolabs.org session.

## Roadmap Gaps (untracked work)

| Score | ID | Category | Title | Action |
|-------|----|----------|-------|--------|
| 35 | FEAT-041 | workflow | Notification severity + light theme (desktop) | `ccs roadmap gaps --apply --top 1` |

No gap scores ≥70 — no high-priority flag this round.

## Health Findings (pre-dispatch hygiene)

- 4 orphaned items (BASE-015, ROAD-091, ROAD-095 +1) — all linked issues closed → run `ccs roadmap health --fix`.
- 4 drifted items (ROAD-085/091/095/098) — recent commits match keywords; verify done-ness before dispatching anything they cover.

## Scoring Notes

Weights per skill: feedback 30% / friction 25% / recency 20% / plan-presence 15% / affected-files 10%. FEAT-020 wave 3 scores highest on friction + de-facto plan (wave-1/2 precedent). FEAT-041 carries the top gaps-engine score (35) but is desktop-tier polish; listed under gaps, not batch.
