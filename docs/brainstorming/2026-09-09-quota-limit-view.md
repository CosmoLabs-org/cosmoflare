---
title: 'Quota / Plan-Limit View — `cosmoflare limits`'
created: "2026-09-09T00:51:09+04:00"
status: planned
tags: [brainstorm, limits, control-plane, road-090]
roadmap: ROAD-090
origin_prompt: docs/prompts/2026-09-08-post-v0.22.0-control-plane.md
deliverables:
  - id: BR-01
    title: "pkg/cosmoflare/limits.go — LimitsService with Snapshot, limitFor, plan resolution"
  - id: BR-02
    title: "pkg/cosmoflare/limits_test.go — httptest + table tests per Testing Strategy"
  - id: BR-03
    title: "cmd/limits.go + cmd/limits_test.go — CLI with table/JSON output and per-source warnings"
  - id: BR-04
    title: "Alert-bridge limits provider + tests — metric-to-row mapping into existing evaluator"
  - id: BR-05
    title: ".cosmoflare.yaml workers_plan config field + --plan flag wiring"
---

# Quota / Plan-Limit View — `cosmoflare limits`

## Problem

An operator running R2-centric infrastructure has no local answer to "how close am I to my plan limits?" The account dashboard shows it per-product behind OAuth and clicks. The continuation prompt `2026-09-08-post-v0.22.0-control-plane.md` Goal 3 names this feature; it feeds ROAD-090 (local Cloudflare control plane) and the existing alert evaluator.

## Research Findings (doc-verified, 2026-09-09)

Cloudflare exposes limits in three disjoint ways. The feature joins all three:

1. **Live usage APIs** — resource counts we already wrap (workers scripts, R2 buckets, zones).
2. **Static per-plan limit tables** — documented constants, not API-queryable per account:
   - Workers Free/Paid: https://developers.cloudflare.com/workers/platform/limits/
   - R2 account/bucket/object limits: https://developers.cloudflare.com/r2/platform/limits/
3. **Live quota APIs** — rare endpoints reporting usage AND quota together:
   - DNS records per zone: https://developers.cloudflare.com/dns/manage-dns-records/ (see "DNS records quota"; documented in the API portal under dns → usage, zone and account variants; REST path pinned in the plan as `GET /zones/{zone_id}/dns/usage`)
   - (Out of scope v1: custom hostnames `GET /zones/{id}/custom_hostnames/quota`)

### Static limit tables (verbatim values, the spec for the join)

**Workers** (Free / Paid):

| Resource | Free | Paid |
|---|---|---|
| Workers scripts per account | 100 | 500 |
| Requests per day (account-wide) | 100,000 | no limit |
| CPU time per invocation | 10 ms | 30 s default, 5 min max |
| Cron Triggers per account | 5 | 250 |

**R2** (plan-independent):

| Resource | Limit |
|---|---|
| Buckets per account | 1,000,000 |
| Custom domains per bucket | 100 |
| Storage / objects per bucket | unlimited |
| REST API rate | 1,200 req / 5 min (informational) |

**DNS records per zone** (context for the live quota API; the API value is authoritative):
Free zones created before 2024-09-01: 1,000; Free zones created on/after: 200; Pro: 3,500; Business: 3,500; Enterprise: account-level quota (1,000,000 default).

**GraphQL Analytics settings node** (deferred from v1, documented for the follow-up):
`viewer { accounts(filter: {accountTag: $accountTag}) { settings { <dataset> { enabled availableFields maxDuration maxNumberOfFields maxPageSize notOlderThan } } } }` — returns per-dataset availability, field whitelist, window width, lookback depth, page size. Plan-gated; must be read at runtime, never hardcoded. Source: https://developers.cloudflare.com/analytics/graphql-api/features/discovery/settings/.

## Decisions (Q&A with user, 2026-09-09)

| # | Question | Decision |
|---|---|---|
| 1 | v1 limit families | **Core 3: Workers + R2 + Zones** (GraphQL meta rows and API rate-limit rows deferred) |
| 2 | Workers plan-tier resolution | **Auto-detect via subscriptions API, fall back to `.cosmoflare.yaml` `workers_plan`, then `--plan` flag**; never a hard failure |
| 3 | Alerting integration | **Command + alert-evaluator feed** (no daemon SSE / Desktop work in this feature) |

## Architecture — Approach A: new `LimitsService`

`pkg/cosmoflare/limits.go` — one service, `BucketDomainService` pattern: option-constructor, raw REST, injectable `*http.Client` (httptest-friendly). Rejected alternatives: piecemeal `Usage()` methods on existing services (join logic stuck in cmd layer, not reusable by the alert bridge or Desktop — violates library-first); GraphQL settings node as primary source (wrong data domain: analytics dataset limits, not resource quotas).

### Row inventory (v1)

| Row resource | Used (live source) | Limit (source) |
|---|---|---|
| `workers.scripts` | `WorkerService.List()` count | 100 / 500 (static join by plan) |
| `workers.daily_requests` | `AnalyticsService.Workers()` over today UTC | 100,000 free / unlimited paid (static) |
| `r2.buckets` | R2 `ListBuckets()` count | 1,000,000 (static, plan-independent) |
| `r2.custom_domains_per_bucket` | only with `--bucket X` (one API call) | 100 (static) |
| `dns.records` (per zone) | **live quota API** `GET /zones/{id}/dns/usage` | API-reported (authoritative, no static join) |
| `zones.count` | zone list count | informational (no documented account cap) |

`dns.records` costs one API call per zone. Acceptable for typical zone counts (≤ 25). Accounts with more zones get every row anyway — the calls are read-only and cheap — but the plan documents this cost and the daemon bridge reuses one snapshot per evaluation cycle rather than re-fetching per rule.

Deferred: `workers.cron_triggers` (schedules endpoint is per-script → N+1 calls), GraphQL settings-node section, API rate-limit rows (REST 1200/5min, GraphQL 300/5min — informational only).

### Data model

```go
type LimitRow struct {
    Resource    string  // "workers.scripts", "dns.records", ...
    Scope       string  // "" account-wide, or bucket/zone name
    Used        uint64
    Limit       uint64  // 0 = unlimited
    Percent     float64 // 0 when Limit == 0 or unknown
    PlanTier    string  // "free" | "paid" | "unknown" | "" when not plan-dependent
    LimitSource string  // "static-docs" | "live-api" | "unknown"
}

type SourceError struct {
    Source string // "workers.list", "subscriptions", "dns.usage:<zone>", ...
    Err    string
}

type LimitsSnapshot struct {
    Rows        []LimitRow
    Sources     []SourceError // per-source failures; snapshot still returns
    WorkersPlan string        // "free" | "paid" | "unknown"
    PlanSource  string        // "auto" | "config" | "flag" | "unknown"
}
```

### Plan-tier resolution order

1. `GET /accounts/{id}/subscriptions` (needs Billing Read; scoped tokens often lack it)
2. `--plan free|paid` flag — an explicit one-shot override outranks the file (standard CLI precedence)
3. `workers_plan: free|paid` field in `.cosmoflare.yaml`
4. `unknown` → Workers plan-dependent rows show used-only, no percent

R2 and DNS rows never depend on plan resolution.

### Partial-failure semantics (ROAD-090 producer doctrine)

Every source runs independently. One failed source → one `SourceError` row; the rest of the snapshot returns. Exit code 0 unless ALL sources fail. `--json` carries `sources[]` inline so agents see exactly what is missing. No source failure ever blanks the whole command.

### CLI surface

`cosmoflare limits [--bucket X] [--plan free|paid] [--json]`

- Table output: sorted by Percent descending; unlimited rows print "—"; per-source errors print after the table as warnings.
- `--json`: full `LimitsSnapshot`.
- `--help` documents every row, its source, and the plan-resolution order (agent-first).

### Alert-evaluator feed

The serve alert bridge (`cmd/serve.go`, existing `newServeAlertBridge`) gains a limits provider: rule conditions `workers-script-count`, `r2-bucket-count`, `dns-record-quota` map to snapshot rows — dash-case to match the existing condition vocabulary (`error-rate`, `storage-limit`). `dns-record-quota` carries the maximum percent across per-zone rows (rows carry `Scope` per zone; `AlertRule` has no zone field today). Threshold stays a plain value fed to the existing `AlertService.Evaluate(name, currentValue)` — no new rule type, but wiring the new metrics requires new `webhook.EvalMetrics` fields and condition cases (plan P-08).

### Static join as pure function

`limitFor(resource string, plan string) (uint64, bool)` — table-tested pure function. Unexported map with doc-source comments. This is the testability extraction point: tests call `limitFor` directly, never re-derive the tables.

## Error Handling

- Per-source: wrap with `newError(op, ...)` following the library convention; source tag names the failing producer.
- `--bucket` with unknown bucket: fail that row only (SourceError), not the command.
- Subscriptions 403 (missing Billing Read): silent fallback to config/flag, `PlanSource` records the path taken.

## Testing Strategy

- `pkg/cosmoflare/limits_test.go`: httptest servers per source (pattern: `bucket_domain_test.go`); table tests for `limitFor`; snapshot assembly with mixed healthy/failed sources asserting partial semantics; plan-resolution order tests (subscriptions 403 → config → flag).
- `cmd/limits_test.go`: table vs JSON rendering, exit codes (all-fail vs partial).
- Alert bridge: fake snapshot provider asserting metric→row mapping and percent computation.

## Open Risk

DNS usage endpoint exact response field names are not shown in the docs excerpt (convention suggests `{used, quota}`-style, cf. custom_hostnames quota `{allocated, used, exceeded, hard_cap}`). Pin them with ONE live curl during implementation (quality-gate step). Fallback if the endpoint is unavailable to the token: per-zone record count via existing DNS list + static per-plan table, `LimitSource: "static-docs"`.

The static limit tables are documented constants captured 2026-09-09 — Cloudflare can retune them. The implementation session spot-checks the four v1 joined values (100/500 scripts, 100k daily requests, 1M buckets, 100 domains/bucket) against the live docs pages once, during the quality gate.

## Deliverables

- BR-01: `pkg/cosmoflare/limits.go` — LimitsService with Snapshot, limitFor, plan resolution
- BR-02: `pkg/cosmoflare/limits_test.go` — httptest + table tests per Testing Strategy
- BR-03: `cmd/limits.go` + `cmd/limits_test.go` — CLI with table/JSON output and per-source warnings
- BR-04: Alert-bridge limits provider + tests — metric→row mapping into existing evaluator
- BR-05: `.cosmoflare.yaml` `workers_plan` config field + `--plan` flag wiring
