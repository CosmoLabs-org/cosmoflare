---
branch: master
created: "2026-10-08T23:13:05+04:00"
date: "2026-10-08T23:13:05+04:00"
goals_completed: 0
goals_total: 8
priority: medium
related_prompts: []
requires_reading:
  - docs/brainstorming/2026-10-08-feat047-usage-waste-alerting.md
  - docs/planning-mode/2026-10-08-feat048-usage-pacing.md
schema_version: 1
status: PENDING
tags: []
title: Session 2032 Continuation
type: prompt
supersedes: "docs/prompts/2026-10-08-session-2031-continuation.md"
---

# Session 2032 Continuation

## File Scope
```yaml
files_modified:
  - pkg/cosmoflare/alerts.go
  - pkg/cosmoflare/analytics.go
  - pkg/cosmoflare/usage.go
  - pkg/cosmoflare/limitsdata/catalog.json
  - internal/webhook/evaluator.go
  - cmd/usage.go
  - cmd/alerts_watch.go
  - internal/cmdmanifest/data.go
  - docs/USAGE.md
  - README.md
  - docs/issues/FEAT-047.yaml
  - docs/issues/FEAT-048.yaml
  - docs/roadmap/items/ROAD-101.yaml
files_created:
  - pkg/cosmoflare/usage_test.go
  - cmd/usage_test.go
  - docs/launch/cosmoflare-og.png
  - docs/launch/cosmoflare-social-preview.png
```

## Context

The usage-monitoring program went from operator idea to two shipped waves in
one day: FEAT-047 (stuck-work alerts that name the offending Worker script
or DO namespace, merged `fb99392`) and FEAT-048 (`cosmoflare usage` monthly
pacing + projected-overage paging, merged `56a29ed`). Both waves ran the
full arc — brainstorm, plan, worktree, TDD, parallel doc agents, quality
gate — and every external fact was verified against live sources (the DO
dataset name, pricing values, tier semantics). Master holds a large unpushed
backlog and a stacked changelog: BR-03 + FEAT-047 + FEAT-048 await v0.33.0.
Two high-priority feedback items from Churches-app arrived at close: a
caching/D1-rows-read/edge-headers audit feature (wave-3 territory plus new
dimensions) and a CHARTER declaring Cosmoflare the CosmoLabs meta-project
for Cloudflare auditing across all projects. The operator flagged the FB
inbox as the first stop.

## GLM Dispatch Rules

When goals involve dispatching subagents:

1. **ALWAYS** use `ccs glm-agent exec` for GLM agents (routes through queue with retry logic)
2. **NEVER** use Agent tool with `model:sonnet` or `model:haiku` for GLM work (bypasses queue, risks 429 rate limits)
3. Agent tool with `model:opus` is fine for Opus subagents
4. For parallel work: use `/glm-sprint` or `ccs glm-agent exec-batch`

## What Got Done

- FEAT-047 wave 1: registry Scope dimension, per-script fan-out in the
  evaluator (alerts name the offender), DurableObjects analytics dataset
  (schema-verified live), six stuck-work conditions. 8 commits, `fb99392`.
- FEAT-048 wave 2: seven verified monthly catalog rows (workers 10M req /
  30M CPU-ms, DO 1M req / 400k GB-s, R2 10 GB-mo / 1M Class A / 10M Class B),
  UsageService pacing engine, `cosmoflare usage` command, usage-pct /
  usage-projected-pct conditions on a 15-min watch cache. 7 commits, `56a29ed`.
- Launch artifacts refreshed to v0.32.0 + social cards (session morning,
  `3868649`); fresh-eyes review caught 3 false public claims.
- Repo debt cleared: `go build ./...` fixed (12 doc.go files, `0641bb7`),
  full-suite timeout fixed (cloudflare-go retry sleeps, `5d5d5df`).
- Docs by parallel agents throughout (operator standing request): USAGE.md
  conditions tables + usage section, README mentions — diffs hand-verified.
- Session-2031 closed cleanly; this working block (the two waves) landed
  after that close and still needs its session-end bookkeeping.

## Goals

### [ ] 1. Triage the FB inbox — two high-priority Churches-app items
**Model:** `sonnet` | **Files:** `docs/feedback/incoming/` (2 digest files), `docs/issues/`, `docs/roadmap/`
Operator flagged this FIRST. Read both files fully: (a)
`2026-10-08-churches-app-cf-efficiency-audit-dimension--caching-d.md` —
caching/D1-rows-read/edge-headers audit feature; (b)
`2026-10-08-churches-app-charter-cosmoflare-is-the-cosmolabs-met.md` — the
meta-project charter. Convert each via `ccs feedback convert <id>` (or
`--section N`) into FEAT/ROAD items with full context; the caching item
feeds goal 3's design, the charter likely becomes a roadmap anchor that
re-weights the program. Do not close either as duplicate without reading.
**Acceptance:** both items leave `pending` (converted or deliberately
deferred with a note); wave-3 design references the caching feedback's
requirements.

### [ ] 2. Release v0.33.0 (operator-gated cut)
**Model:** `glm-turbo` | **Files:** `.version-registry.json`, `docs/changelog/`
Pre-steps: `ccs sync` (BUG-978 counter clamp needs it before push), then
verify the changelog queue holds BR-03 + FEAT-047 + FEAT-048 staged. The
cut itself (`ccs version --bump minor`, tag, local build + publish per the
no-CI local-release rule) needs the operator's explicit go — surface the
readiness state and ask. **Acceptance:** either v0.33.0 tagged and
published with the three entries, or an explicit operator deferral
recorded in the session notes.

### [ ] 3. Wave 3 — cache-miss warning (promote from ROAD-101)
**Model:** `sonnet` for design, then `glm-turbo` per-task | **Files:** `pkg/cosmoflare/analytics.go`, `pkg/cosmoflare/alerts.go`, `internal/webhook/evaluator.go`
Promote via `ccs issues create feature` from ROAD-101's wave-3 line. Design
AFTER goal 1's caching feedback lands (it defines the real requirements —
per-zone cache hit-ratio, maybe edge-header signals). Zone cache ratio
comes from `httpRequestsAdaptiveGroups` (ZoneHTTP already queries it —
extend rather than add). Same arc as waves 1-2: brainstorm → plan →
worktree → TDD → parallel doc agents → gate → merge.
**Acceptance:** cache conditions registered and evaluated per zone, alert
names the zone, suites green, merged through verify-worktree.

### [ ] 4. Session-end bookkeeping for the wave-1/2 block
**Model:** `glm-turbo` | **Files:** `docs/sessions/`, `docs/prompts/`
The two waves landed after Session-2031's close. Run `/session-end`
(full mode) when this block wraps: summary, this prompt's supersede chain,
feedback flush, roadmap notes for FEAT-047/048 closures.
**Acceptance:** session marker matches, tree clean, `ccs workcheck`
conversation ledger has no unadjudicated MISSED rows.

### [ ] 5. cosmolabs.org deploy: TASK-010 + shader hero + pager PWA (carried)
**Acceptance:** /cosmoflare serves OG+JSON-LD using the committed
cosmoflare-og.png; /pager installs; TASK-010 closed with live URL.
Operator-driven session at ~/PROJECTS/cosmolabs.org following
docs/launch/task010-cosmolabs-org-handoff.md. FEAT-046 hero prototype was
paused mid-inspection (clone at ~/.analysis-clones/shader-effects-inc-shaders).

### [ ] 6. Launch posts live (carried)
**Acceptance:** Show HN + r/Cloudflare URLs recorded in a ROAD-096 note.
Drafts are v0.32.0-accurate and review-clean; posting notes inside.

### [ ] 7. Real-phone E2E pager smoke (carried)
**Acceptance:** `alerts watch --test-fire` buzzes a paired phone; note on
FEAT-045.

### [ ] 8. Operator research pastes (carried)
**Acceptance:** grok limits results + grok/gemini perms results land;
ingestion goals close (limits corpus is 5/6, perms is 2/4 — see pending
prompts list).

## Carry-Over Tasks
- [ ] `ccs kill feat048-wave2` — merge landed, stale LSP PIDs blocked the
  auto-kill at close (nothing at risk)
- [ ] CF_API_TOKEN rotation — 57 days overdue per `ccs credentials status`
- [ ] Cloudflare MCP servers re-auth (stale OAuth cache flagged at session
  start)
- [ ] `ccs memory scan` — 1 unenrolled ClaudeCodeSetup-worktree memory file
  noted at session start (out-of-repo, low priority)

## Carry-Overs
1. **cf-limits-corpus research ingestion** (5/6 goals — needs grok-results.md)
   → `docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md`
2. **cf-perms next-waves research ingestion** (2/4 — needs grok + gemini results)
   → `docs/prompts/2026-09-17-cf-perms-next-waves-tiers-research-ingestion.md`
3. **FEAT-045 pager plan** (0/35 checkboxes — feature SHIPPED and closed;
   verify-and-archive candidate, never archive on age alone)
   → `docs/planning-mode/2026-10-07-feat045-cosmoflare-pager.md`

## Where We're Headed

The Churches-app charter, if adopted, reframes Cosmoflare from "a CLI with
alerts" to the operational spine of every CosmoLabs Cloudflare deployment:
auditing, security review, and resource efficiency as standing services.
Waves 1-2 built the detection substrate (per-script telemetry, pacing
arithmetic, verified allowance catalog); wave 3 adds the caching dimension;
model-B baselines (idea filed) turn the same rows into anomaly detection
later. The near unlock is proving the loop on a real account — goal 5's
deploy plus a live `usage` read — so the meta-project story has a working
artifact behind it.

## Priority Order
1. Goal 1 — FB inbox triage (operator-flagged first; gates goal 3's design)
2. Goal 4 — session-end bookkeeping (cheap, keeps the chain honest)
3. Goal 2 — release readiness surfaced to the operator
4. Goal 3 — wave 3 after triage informs it
5. Goals 5-8 — operator-gated, ride along
