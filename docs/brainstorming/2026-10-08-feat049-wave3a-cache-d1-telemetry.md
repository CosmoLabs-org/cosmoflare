---
title: FEAT-049 Wave 3a — Zone cache + D1 rows-read telemetry alerts — Brainstorm
created: "2026-10-08T23:45:00+04:00"
issue: FEAT-049
roadmap: ROAD-101
status: settled
last_reviewed: "2026-10-08T23:59:00+04:00"
last_review_ref: "none (independent-review report returned inline to the caller)"
last_review_findings: 14
deliverables:
    - BR-01: Decision record — split FEAT-049 into telemetry (3a, watch conditions) and probes (FEAT-050 rule set)
    - BR-02: Live schema facts (introspected 2026-10-08) for cacheStatus, zoneTag_in, D1 datetime grouping
    - BR-03: Condition definitions — zone-cache-miss-pct, zone-uncached-pct, d1-rows-read (formulas, floor, scope)
    - BR-04: Scope boundary — KV hit rate and probe sweep out; D1 monthly allowance row in
---

# FEAT-049 Wave 3a — Zone cache + D1 rows-read telemetry

**Date**: 2026-10-08 · **Session**: 2033 · **Status**: design approved by the operator
**Tracker**: FEAT-049 (wave 3a) · ROAD-101 (program) · ROAD-102 (CF governance charter) · FEAT-050 (probe rule sets)

## Why

FB-11 (Churches-app, spec recovered from `Churches-app/docs/audits/2026-10-08-cf-efficiency-audit.md`)
asked for a caching efficiency dimension: header sweep, repeat-probe, D1 rows-read,
KV hit rate, unused bindings. Two different kinds of work sit inside that list:
continuous telemetry (fits the `alerts watch` fast loop) and active HTTP probes of
operator endpoints (a pass/fail audit, which is FEAT-050's verdict command).

Live evidence on the operator's account, 2026-10-08, last 24h:

- `churches.app` (Free plan): 939 of 1,541 eyeball requests were `dynamic` (61%) —
  the "API emits no Cache-Control" pattern the Churches-app audit found by hand.
- `mycarguide-db`: 2,945,546,702 rows read from 73,151 read queries (~40k rows/query).
  At this rate ≈ 88B rows/month against the Paid allowance of 25B/month
  (≈ $63/month overage at $0.001/M). Reported to MyCarGuide as feedback.

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Split FEAT-049**: wave 3a = telemetry conditions; probes (header sweep, repeat-probe, auth-route cache check, unused bindings) become FEAT-050's first rule set | Telemetry is free GraphQL on the existing watch loop; probes need an endpoint list and produce a verdict, not a page (operator choice) |
| D2 | **Inverted ratios** — `zone-cache-miss-pct` and `zone-uncached-pct`, not "hit %" | The evaluator fires only at value ≥ threshold; inversion keeps one comparator |
| D3 | **Cache-eligible denominator** for miss %: hit, miss, expired, revalidated, updating, stale. Miss numerator: miss + expired | `dynamic`/`bypass` never enter the cache, so they would drown the miss signal |
| D4 | **Uncached %** = (dynamic + bypass) ÷ known-status requests (D3 eligible set + dynamic + bypass; `none` and the other excluded statuses never count, see D5 and Live schema facts) | The Churches-app failure mode: traffic that never reaches the cache |
| D5 | **`none` excluded** from every numerator and denominator; still reported in JSON | Undocumented in GraphQL; 26k requests on cosmolabs.org would distort both ratios (operator choice) |
| D6 | **100-request floor** per zone per window for both zone ratios, applied to each ratio's own denominator (eligible count for miss %, known-status count for uncached %) | A zone with 3 requests and 2 misses must not page at 67%. A floor on total requests would not stop this: a mostly-dynamic zone can have 1,000 requests and 3 eligible ones |
| D7 | **Batched queries** — `zoneTag_in` with ≤ 10 zones per query; one account-wide D1 query grouped by `databaseId` | Decision D5 of the FEAT-047 brainstorm (`2026-10-08-feat047-usage-waste-alerting.md`, no N+1); 10-zone cap is a documented hard limit |
| D8 | **D1 alerts name the database** — names from the D1 list call, cached on the slow loop | "mycarguide-db read 2.9B rows", never a UUID |
| D9 | **KV hit rate out** | `kvOperationsAdaptiveGroups` has `result`/`responseStatusCode` in the schema, but no doc defines `result`; no verified semantics |
| D10 | **D1 monthly rows-read catalog row in** (`d1.rows_read_monthly`, paid 25B, verified 2026-10-08) feeding the FEAT-048 usage view | One verified row makes `cosmoflare usage` show D1 pacing; free tier is per-day (5M/day), so free stays null for the monthly row |

### Decisions settled after independent review (2026-10-08)

| # | Decision | Rationale |
|---|----------|-----------|
| D11 | **Persistent per-scope cooldown for zone and d1 scopes only** — default 1h; script/do/account rules keep the re-page-every-cycle policy (`cmd/alerts_watch.go:292`) | A rolling 24h D1 sum stays tripped for ~24h; re-paging every 60s is noise. BUG-054 (`a545c29`) already keys cooldown per scope instance (operator choice) |
| D12 | **Eyeball traffic only** — zone ratios filter `requestSource: "eyeball"` | Matches `ZoneHTTP` and every live number in this record |
| D13 | **`ignored`, `stream_hit`, `deferred`, `unknown` excluded** from both ratios | Undocumented in GraphQL or not cache-tier traffic; JSON still reports them |
| D14 | **Human-readable d1 page values** ("2.9B rows"), not `%g` | `%g` prints 2.945546702e+09 on a phone |
| D15 | **Zone and D1 telemetry are additive** — in the watch (DO pattern) and in `CollectUsage` (a D1 failure keeps Workers/R2 pacing rows) | One missing token permission must not stop every alert |
| D16 | **`alerts watch` help lists the new datasets** | Help is the agent-facing contract |

### Operator decision log

The full list of big decisions (decided and open) lives in one table: the
"Decisions for the operator" section of
`docs/planning-mode/2026-10-09-feat049-wave3a-cache-d1-telemetry.md` (O1–O15).
Open at planning time: O9 pager severity, O10 starter thresholds, O11
rolling-24h vs Free-plan UTC day, O14 mycarguide-db fix (MyCarGuide's call).
Mitigations approved 2026-10-09 after the risk review: telemetry-gap page,
per-rule exclude list, 2× escalation inside the cooldown, opt-in live smoke
test.

## Live schema facts (BR-02, introspected 2026-10-08 with the operator token, read-only)

- `ZoneHttpRequestsAdaptiveGroupsDimensions` has `cacheStatus`, `datetimeHour`, `datetimeFiveMinutes`, `edgeResponseStatus`, `requestSource`.
- `count` is a field of the group, **not** of `ZoneHttpRequestsAdaptiveGroupsSum` (this was BUG-055 in `ZoneHTTP`, fixed `fb17689`).
- `cacheStatus` values observed on Free zones: `hit`, `miss`, `expired`, `revalidated`, `bypass`, `dynamic`, `none`. Logpush documents also `updating`, `stale`, `ignored`, `stream_hit`, `deferred`, `unknown` — treat `updating`/`stale` as eligible, everything else not listed in D3/D4 as excluded.
- `zones(filter: {zoneTag_in: [...]})` works in one request (verified with two zones); docs cap a zone-scoped query at 10 zones.
- `AccountD1AnalyticsAdaptiveGroupsDimensions`: `databaseId`, `date`, `datetime`, `datetimeHour`, `datetimeFiveMinutes`, …; `Sum`: `readQueries`, `rowsRead`, `rowsWritten`. Account query with `datetime_geq/leq` and no `databaseId` filter returns one row per database (verified).
- D1 pricing (https://developers.cloudflare.com/d1/platform/pricing/, "Last updated Apr 21, 2026"): Free 5M rows read/day; Paid first 25B/month + $0.001/M rows.

## Conditions (BR-03)

| Condition | Scope | Unit | Value | Not judgeable when |
|-----------|-------|------|-------|--------------------|
| `zone-cache-miss-pct` | zone | % | (miss+expired) ÷ (hit+miss+expired+revalidated+updating+stale) × 100 | eligible requests < 100 (D6) |
| `zone-uncached-pct` | zone | % | (dynamic+bypass) ÷ (hit+miss+expired+revalidated+updating+stale+dynamic+bypass) × 100 | known-status requests < 100 (D6) |
| `d1-rows-read` | d1 | rows | Σ rowsRead over the watch window (rolling 24h, `cmd/alerts_watch.go:53`) per database | the database has no analytics row in the window (no queries). A returned row with rowsRead = 0 is a value |

Scope IDs: zone name for zone scope; database name (fallback: id) for d1 scope.

## Code constraints (verified 2026-10-08 by independent review)

These facts come from the current tree. The plan must cover each one.

1. **New scopes need new evaluator branches.** `conditionValues` (`internal/webhook/evaluator.go:171-219`) fans out only for `script` and `do`. Any other scope falls to the account path, then to `conditionValue`'s `default`, which returns ok=false. A `zone` or `d1` condition without its own `case` never fires and gives no error.
2. **"Not judgeable" already has a mechanism.** Fan-out branches skip an instance with `continue` and append no `scopedValue` (`evaluator.go:188-191`, `207-209`). The Evaluate loop only sees appended values (`evaluator.go:259`). The zone floor and the d1 no-row case use this pattern. No sentinel value (such as 0) is needed.
3. **`EvalMetrics` has no zone or D1 rows** (`evaluator.go:21-37`). Add per-zone cache rows and per-database D1 rows. Update the `Scope` field comment in `pkg/cosmoflare/alerts.go:102`.
4. **Collection failure policy is not stated in this record.** `CollectEvalMetrics` fails the whole cycle on a Workers or R2 error, but treats DO data as additive (`evaluator.go:328-335`). Zone and D1 telemetry must follow the DO pattern, or one missing permission (zone analytics, D1 read) stops every alert.
5. **`ZoneHTTP` cannot be reused as is.** It queries one `zoneTag`, selects no `zoneTag` or `cacheStatus`, and filters `requestSource: "eyeball"` (`pkg/cosmoflare/analytics.go:169-178`). Wave 3a needs a new batched query that returns `zoneTag` per zone and groups by `cacheStatus`. It also needs a zone ID → name map from the zone list call.
6. **The "slow loop" is only the usage-snapshot cache.** `cmd/alerts_watch.go:66-95` caches one `UsageSnapshot` for 15 min. No general slow loop exists. The D8 database-name cache (and a zone-name cache) need their own TTL cache with the same keep-last-good rule.
7. **D1 names are available.** `D1Service.List` returns `UUID` and `Name` (`pkg/cosmoflare/d1.go:25-32`, `106-118`). Map `databaseId` → `Name`; use the UUID when a row has no list match (for example, a deleted database).
8. **The existing D1 query uses another filter.** `D1UsageService.DailyUsage` queries `d1AnalyticsAdaptiveGroups` with `date_geq`/`date_lt` and a `databaseId` filter (`pkg/cosmoflare/d1_usage.go:89-98`). Wave 3a needs a new account-wide query with `datetime_geq`/`datetime_leq` and no `databaseId` filter (Live schema facts).
9. **D10 needs collector wiring, not only a catalog row.** `CollectUsage` builds its `used` map from a fixed list of IDs (`pkg/cosmoflare/usage.go:145-153`). A `d1.rows_read_monthly` catalog row shows nothing until `CollectUsage` also queries D1 rows read over the cycle window. `CollectUsage` returns an error on any dataset failure (`usage.go:99-116`), so a D1 query failure would also remove the Workers and R2 pacing rows.

## Out of scope (BR-04)

KV hit rate (D9); header sweep, repeat-probe, auth-route cache check, unused bindings (FEAT-050);
model-B baselines; dashboard graphs; D1 cost projection in alerts (the usage view covers pacing).
