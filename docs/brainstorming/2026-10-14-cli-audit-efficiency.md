---
title: 'CLI pivot — audit + efficiency program: sequencing FEAT-047/048/067/068/069/070'
source: Operator pivot directive (2026-10-14, session 78) — "make cosmoflare an indispensable tool for each project to integrate with Cloudflare, audit its usage, and ensure we are being resource efficient"
related_issues:
    - FEAT-047
    - FEAT-048
    - FEAT-067
    - FEAT-068
    - FEAT-069
    - FEAT-070
related_prompts:
    - docs/prompts/2026-10-14-cli-pivot-audit-efficiency.md
related_research:
    - docs/research/2026-10-10-cf-agent-cli-gap-analysis.md
status: proposed
created: "2026-10-14T14:40:00+04:00"
plan_ref: docs/planning-mode/2026-10-14-cli-audit-efficiency.md
last_reviewed: "2026-10-14T15:35:00+04:00"
last_review_ref: docs/independent-reviews/2026-10-14T15-35-00-cli-audit-efficiency-trio.md
last_review_findings: 4
deliverables:
    - id: BR-01
      title: "Program doc: three-wave sequencing of FEAT-067/068/069/070 over the shipped 047/048 foundation"
    - id: BR-02
      title: "Wave-1 spec: cosmoflare efficiency command (idle workers, dead KV, empty R2) with exact datasets and finding model"
    - id: BR-03
      title: "Wave-1 cost wiring spec: idle-waste line reusing efficiency collectors"
    - id: BR-04
      title: "Wave-2 scope: KV analytics, D1 pagination, cache coverage, HTML trap detector, spend-vs-allowance"
    - id: BR-05
      title: "Wave-3 scope: patterns P1-P9 checks, patch emission, TTL strategy verdicts"
    - id: BR-06
      title: "Decision record D1-D6 with rejected alternatives"
---

# CLI pivot — audit + efficiency program

## Context

The Ops pager app reached its milestone (deployed, build DQHsMwHV). The operator
pivoted the default focus to the CLI core (`cmd/` + `pkg/cosmoflare/`):

> "shift gears from the Ops app to the infrastructure of cosmoflare — the CLI
> tools and workflows. Make cosmoflare an indispensable tool for each project
> to integrate with Cloudflare, audit its usage, and ensure we are being
> resource efficient."

The backlog already matches this vision. Two features shipped as the
foundation; four are open. This doc sequences all six into one program.

## Strategic frame

Every CosmoLabs project touches Cloudflare. Cosmoflare should be the tool each
project installs first:

1. **Integrate** — `init`, `templates`, `apply` (shipped long ago).
2. **Audit** — `efficiency`, `usage`, `alerts` — see the waste, live, from real
   API numbers.
3. **Remediate** — pattern detection, suggested patches, TTL strategy
   (FEAT-067/068).

Library-first applies: collectors and findings live in `pkg/cosmoflare/`, the
CLI wraps them, the Ops app stays a read-only window on the same data. House
rules bind throughout: real API numbers only (never modeled — the KV 86×
incident rule), `--json` on every command, agent-readable errors, TDD for new
cmd code, conventional commits.

## Foundation (already shipped — no work in this program)

| Feature | What landed | Evidence |
|---------|-------------|----------|
| FEAT-047 usage-waste alerting | Per-script Workers/DO threshold conditions (sustained CPU, error loops, invocation spikes) in the alerts watch, batched GraphQL, offender named in the alert | acceptance criteria all `met: true` |
| FEAT-048 usage pacing | `cosmoflare usage` view — every monthly dimension vs plan limit with linear pacing, `usage-pct`/`usage-projected-pct` alert conditions, 15-min usage cache in watch | acceptance criteria all `met: true`; plan at docs/planning-mode/2026-10-08-feat048-usage-pacing.md |

These two are the "alert on waste" half. What is missing is the "see the waste
on demand" half — a findings command — and the "fix the waste" half.

## The program: three waves over the four open features

### Wave 1 — `cosmoflare efficiency`: the audit flagship (FEAT-069 first slice)

**Goal:** one command answers "what are we paying for that does nothing?"

`cosmoflare efficiency [--json]` reports idle resources with real API evidence:

- **Idle workers** — 0 requests over the last 7 days (`workersInvocationsAdaptive`
  per-script sum; dataset already exists in `pkg/cosmoflare/analytics.go`).
- **Dead KV namespaces** — 0 reads AND 0 writes over 7 days
  (`kvOperationsAdaptiveGroups` — NEW dataset; FEAT-069 gap 1).
- **Empty R2 buckets** — 0 objects (R2 bucket metadata / object count via the
  R2 API).

Every finding carries: what (resource + check), evidence (counts, dates,
window), why it matters, and how to fix (an actionable cosmoflare or dashboard
step). Pattern: `cmd/doctor.go`'s diagnostic shape (severity indicators,
`--json`, `--fix` suggestions, agent-readable).

Collectors land in `pkg/cosmoflare/efficiency.go` as an `EfficiencyService`
with a shared `Finding` model. This model is the substrate waves 2-3 build on.

Also in wave 1: `cosmoflare cost` gains a waste line (idle-resource dollars,
reusing the same collectors — no second implementation).

### Wave 2 — measurement completeness + spend-vs-allowance (FEAT-069 rest)

**Goal:** trust every number; see spend against allowance per service.

The MyCarGuide evidence case (FEAT-069) drives this wave:

- **KV analytics** in `cosmoflare analytics` (the wave-1 dataset, exposed).
- **D1 usage with hourly pagination** — a single day exceeds the 10,000-group
  GraphQL cap; one unpaged query undercounted 17× (458M vs 8.39B rows). Any D1
  usage feature must paginate hourly windows.
- **Cache-coverage view** — per-zone edge hit rate (`httpRequests1dGroups`),
  cached vs dynamic route families.
- **The HTML caching trap detector** — origin emits `s-maxage` on HTML but no
  zone Cache Rule exists → `cf-cache-status: DYNAMIC` silently. One check that
  catches a fix that "shipped" and did nothing.
- **Spend-vs-allowance page** — D1 rows, KV ops, R2, invocations vs plan in
  one command (extend `cosmoflare usage`; this is FEAT-070's
  `cf-usage-spend.md` one-pager format).
- **New alert classes** — KV writes/day, cache-hit-rate regression (extends the
  FEAT-047 registry).
- **Cost real usage** — `cosmoflare cost` currently estimates from zero-value
  placeholder usage structs; wire real usage fetching here (bigger change,
  deliberately not wave 1).

### Wave 3 — remediation: patterns + TTL strategy (FEAT-067 + FEAT-068)

**Goal:** from findings to fixes.

- **Pattern detection** (FEAT-067): encode the Churches-app P1-P9 patterns as
  checks — query-aware cache keys, opt-in middleware, materialize-by-cron,
  Cache-Control emission, media gating — each finding cites the pattern number
  and proposes the fix.
- **Patch emission** (FEAT-067 stretch): `efficiency --fix` emits ready-to-apply
  patches (the `kvCache({ keyFn })` line derived from handler source).
- **TTL strategy engine** (FEAT-068): verdict each cached route's TTL as
  over/under/appropriate from data-change cadence (D1 write analytics) ×
  invalidation coverage (source analysis); explain the KV-TTL vs
  Cache-Control-max-age asymmetry per finding.

Wave 3 is last because every verdict needs wave 1's findings model and wave 2's
trustworthy per-resource numbers.

### FEAT-070 — the corpus, not a wave

FEAT-070 (MyCarGuide best-practices corpus: ADR-002 six standing rules,
caching-strategy matrix, d1-indexing discipline) is the rule source each wave
encodes check classes from. It lands incrementally: wave 2 takes the
spend/snapshot rules, wave 3 takes the caching/indexing rules. No separate
delivery.

## Key decisions

| # | Decision | Rationale | Alternatives rejected |
|---|----------|-----------|----------------------|
| D1 | `efficiency` is the findings umbrella command | The name `audit` is TAKEN — `cosmoflare audit` is mutation audit logging (account_audit.go, audit_wire.go). Growing a findings family under `efficiency` (`efficiency caching`, `efficiency ttl` later) avoids breaking an existing command | Restructuring `audit` into an audit-family front door with mutation logging moved under it — bigger blast radius, no user ask |
| D2 | Findings model lives in `pkg/cosmoflare/efficiency.go` | Library-first; Ops app and MCP server consume the same findings later. doctor.go's DiagnosticIssue stays domain-specific (network probes) | Reusing DiagnosticIssue — conflates probe results with waste findings |
| D3 | Wave 1 checks are exactly three | Idle workers, dead KV, empty R2 — each is a single dataset query, each is unambiguous waste. More checks (D1, cache) need wave-2 pagination machinery first | Loading wave 1 with D1/cache checks — the 10k-group pagination trap makes D1 nontrivial; keep the flagship shippable |
| D4 | Real numbers, GraphQL analytics, batched | House rule (KV 86× incident). Analytics API is free (rate-limited, not billed). One query per dataset per run, never per-resource N+1 | REST-only (the `cf` CLI cannot pull Workers/D1 analytics at all — our differentiator per ROAD-104 gap analysis) |
| D5 | Cost wiring reuses efficiency collectors | G-03 explicitly: no second collector. The waste line attributes projected monthly dollars to idle resources found by the same service | Independent cost-side waste scan — duplicated API calls, drift between the two answers |
| D6 | Waves 2→3 order | Remediation verdicts (TTL over/under, pattern violations) require trustworthy per-resource numbers first — the 17× undercount proves unpaged D1 numbers lie | Remediation-first — would emit wrong verdicts on undercounted data |

## Operator-gated decisions (not blocking wave 1)

1. **Naming umbrella** — D1 recommends `efficiency`. If the operator prefers an
   `audit` restructure, say so before wave 2 lands subcommands.
2. **v0.33.0 scope** — which waves ship in the release (release itself is
   operator-go-only per the keep-local rule).
3. **Cost real-usage** — confirm wave-2 placement (D-decision above assumes it).

## What each wave's first slice looks like (ship-ability check)

- Wave 1: `cosmoflare efficiency` — user runs one command, gets waste findings
  with evidence. Shippable alone.
- Wave 2: `cosmoflare usage` gains spend-vs-allowance; `cosmoflare analytics`
  gains KV + paginated D1. Each is a command upgrade, shippable independently.
- Wave 3: `efficiency caching` + `efficiency ttl` subcommands; each finding
  carries a fix. Shippable per-subcommand.

## Out of scope

- FEAT-047 model B (baselines/anomaly detection) — deferred, separate idea.
- Desktop dashboard graphs (desktop tier, later).
- Ops app UI for efficiency findings (the app stays read-only window; wire
  after the CLI ships).
- OAuth login flow, fuzzy search, agent-context help (ROAD-104 follow-ups —
  separate track from this program).
