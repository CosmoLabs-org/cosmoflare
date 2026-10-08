---
title: FEAT-047 Usage-Waste Alerting — Brainstorm
created: "2026-10-08T19:45:00+04:00"
issue: FEAT-047
roadmap: ROAD-101
status: settled
deliverables:
    - BR-01: Decision record — thresholds now, baselines later, waves 1-3, dual cadence, per-script naming, batched GraphQL
    - BR-02: API-cost analysis — free analytics loop, cents-level limits loop
    - BR-03: Q&A trail with operator answers
    - BR-04: Scope boundary for wave 1 (in/out)
---

# FEAT-047 — Usage-Waste Alerting: Brainstorm & Decision Record

**Date**: 2026-10-08 · **Session**: 67 · **Status**: design settled, wave 1 planned
**Tracker**: FEAT-047 (wave 1) · ROAD-101 (program) · IDEA model-B (baselines, deferred)

## Why

Operator direction, verbatim intent: cosmoflare must help operators of
ambitious Cloudflare accounts catch usage waste *as it happens* — "a durable
object or some worker that is overworking and getting stuck and wasting our
usage", "high usage especially during development or lack of caching",
"monitor in real time and give us alerts and show in a dashboard for monthly
usage". Prevention, not post-mortem: catch the stuck loop in minutes, not on
the monthly invoice.

## Existing spine (what this builds on)

- `cosmoflare alerts watch` (FEAT-045): the daemon — evaluates rules on an
  interval, dispatches Web Push to the pager PWA.
- `alertConditionRegistry` (pkg/cosmoflare/alerts.go): declarative conditions
  fed by `EvalMetrics` (today: account-flat Workers requests/errors/CPU p99,
  R2 storage, limits-snapshot quotas).
- `internal/webhook/evaluator.go`: `CollectEvalMetrics` (analytics) +
  `CollectLimitMetrics` (limits snapshot) → `Evaluate`.
- Limits catalog (BR-03, unreleased v0.33): 66-entry embedded catalog with
  per-service plan resolution.

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Thresholds now (model A), baselines later (model B)** | Stuck workers show instantly under thresholds; usage pacing is arithmetic, not anomaly detection. Baselines need time-series storage — idea filed, built on wave-1 infra. |
| D2 | **Three waves under the existing watch daemon** | 1) stuck-work thresholds → 2) usage pacing + `cosmoflare usage` view → 3) cache-miss warning. Each independently shippable and paged. |
| D3 | **Dual cadence inside `alerts watch`** | Fast loop 30–60s (GraphQL analytics — free, rate-limited not billed); slow loop 5–15min (limits snapshot REST lists — Class B ops, cacheable, cents/month). |
| D4 | **Per-script granularity, offender named** | The alert must say "worker api-proxy CPU p99 812ms for 3 windows", not "account CPU high". Requires widening `EvalMetrics` to per-script rows. |
| D5 | **One batched GraphQL query per cycle** | `workersInvocationsAdaptive` (per-script) + `durableObjectsAnalytics` as multi-dataset queries — never per-script N+1. |
| D6 | **CLI `cosmoflare usage` for the monthly view** | Wave 2. Always-on dashboard graphs remain the desktop tier's job (product vision). |

## API-cost analysis (operator concern: "would use up a lot of API calls")

| Call | Cost at 60s cycles |
|------|--------------------|
| GraphQL Analytics (Workers/DO/cache/R2 metrics) | Free — rate-limited, not billed |
| Limits snapshot REST lists (buckets, scripts, DNS) | R2 Class B — few thousand/day ≈ fractions of a cent; slow-loop cadence + caching |

## Q&A trail

- **Q: detection model?** A: A now, B later (operator).
- **Q: first-cut scope?** A: sequenced waves 1→2→3 (operator accepted filing
  per-wave; waves 2-3 promote from ROAD-101 as wave 1 lands).
- **Q: daemon or cron?** A: the daemon already exists — `alerts watch`.
  Cost concern resolved by D3/D5.
- **Q: where do baselines live?** A: deferred; wave-1 per-script rows +
  evaluator window state become the time-series seed.

## Out of scope (wave 1)

Model-B baselines; usage pacing + `usage` command (wave 2); cache-miss
(wave 3); desktop dashboard graphs.
