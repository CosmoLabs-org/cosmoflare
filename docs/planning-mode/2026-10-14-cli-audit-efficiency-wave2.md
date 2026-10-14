---
title: 'CLI audit + efficiency — wave 2: measurement completeness (FEAT-069)'
status: PLANNED
branch: master
created: "2026-10-14T17:05:00+04:00"
brainstorm_ref: docs/brainstorming/2026-10-14-cli-audit-efficiency.md
plan_ref: docs/planning-mode/2026-10-14-cli-audit-efficiency.md
prompt_ref: docs/prompts/2026-10-14-cli-pivot-audit-efficiency.md
related_issues:
    - FEAT-069
tags: [efficiency, audit, cli, wave-2, cache]
deliverables:
    - id: P-06
      title: "ZoneCacheCoverage: per-zone hit-rate finding in cosmoflare efficiency (httpRequests1dGroups, cachedRequests)"
    - id: P-07
      title: "cosmoflare analytics kv — expose KVOperations totals per namespace"
    - id: P-08
      title: "HTML caching-trap detector — s-maxage on HTML with no zone Cache Rule → finding"
    - id: P-09
      title: "KV ops dimension in the usage view (spend-vs-allowance row)"
    - id: P-10
      title: "Alert classes: kv-ops/day threshold + zone-cache-hit regression"
implemented_commits:
    - {sha: '8e1f72e9c5aa', covers: [P-07]}
    - {sha: '07289797e543', covers: [P-06]}
    - {sha: 'a010281e48ad', covers: [P-08]}
    - {sha: '56ffe135988a', covers: [P-09]}
---

# Cosmoflare efficiency — Wave 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans, task-by-task.

**Goal:** Close FEAT-069's measurement gaps: cache coverage visibility (the biggest cost lever), KV analytics exposure, the HTML caching-trap detector, and the KV spend-vs-allowance row.

**Architecture:** Wave 1's `EfficiencyService` grows a zone-aware check backed by a new `ZoneCacheCoverage` analytics method; the analytics command exposes the wave-1 KV dataset; a new caching-trap check live-probes zone apexes. Same house rules: real numbers only, `--json`, TDD, one batched query per dataset.

**Spec:** `docs/brainstorming/2026-10-14-cli-audit-efficiency.md` (wave 2 section).

## Verified facts (live-probed 2026-10-14 — trust these, do not re-guess)

1. `httpRequests1dGroups` on `viewer.zones(filter: {zoneTag})` returns `dimensions { date } sum { requests cachedRequests bytes }` — verified live (probe zone: 3.5k req/day, 14 cached).
2. `kvOperationsAdaptiveGroups` exposes ONLY `dimensions { namespaceId } sum { requests }` — no reads/writes split (wave-1 discovery, commit 006aa48).
3. `d1_usage.go`'s per-database per-day query groups by date (≤92 rows) — the 10k-group trap does NOT apply to it; the trap applies to account-wide multi-group queries only.
4. `KVOperationsSummary{NamespaceID, Requests}` exists on `AnalyticsService` (wave 1, merged).
5. `WorkerService`, `ZoneService` (cmd `getZoneService()`), `AnalyticsService.Workers/R2Storage/KVOperations` all exist and are mock-tested via `cloudflare.NewWithAPIToken(..., cloudflare.BaseURL(srv.URL))` + `WithAnalyticsBaseURL(srv.URL)`.

## Global Constraints

(same as wave 1 — real numbers, --json envelope, agent-readable findings, one batched query per dataset, TDD, conventional commits, no touching `cosmoflare audit`)

## Reference files

`cmd/efficiency.go` (seam pattern, wave 1) · `pkg/cosmoflare/efficiency.go` (finding model) · `pkg/cosmoflare/analytics.go` (dataset pattern; put ZoneCacheCoverage next to ZoneHTTP) · `cmd/analytics.go` (R2 analytics command shape) · `pkg/cosmoflare/usage.go` (dimension collection) · `docs/planning-mode/2026-10-14-cli-audit-efficiency.md` (wave-1 plan conventions)

---

## Task W2-A: Zone cache coverage — the hit-rate finding (deliverable P-06; P-10 resolution below)

**P-10 resolution (2026-10-14, verified in-tree):** the alert classes already exist — `zone-cache-miss-pct` and `zone-uncached-requests` (FEAT-049 wave 3a, zone-scoped) cover the cache-hit regression class, and `kv-writes` (per-namespace, fed by `analytics_kvwrites.go`) covers KV volume alerting. No new registry entries needed; this plan was written without checking the registry (plan gap, caught during execution).

**Model:** sonnet · **Files:** modify `pkg/cosmoflare/analytics.go`, `pkg/cosmoflare/analytics_test.go`, `pkg/cosmoflare/efficiency.go`, `pkg/cosmoflare/efficiency_test.go`, `cmd/efficiency.go`, `cmd/efficiency_test.go`

**Interfaces:**
- Produces:
  - `type ZoneCacheSummary struct { ZoneID, Zone string; Requests, CachedRequests uint64; HitRatePct float64 }`
  - `func (s *AnalyticsService) ZoneCacheCoverage(ctx, zoneID string, w AnalyticsWindow) (*ZoneCacheSummary, error)` — ONE batched `httpRequests1dGroups` query (verified shape above), sums requests+cachedRequests over the window, HitRatePct = cached/requests*100 (0 when requests==0).
  - `EfficiencyService` gains an optional zone lister: `type ZoneLister interface { List(ctx) ([]Zone, error) }` (Zone type from zone.go — read it for exact fields) + `NewEfficiencyServiceWithZones(workers, kv, analytics, buckets, zones ZoneLister)`; keep the existing constructors compiling (add the param to WithBuckets via a variadic or a small options struct — follow Go idioms already in the repo, do not break wave-1 callers).
  - `Report` gains: for each zone, `ZoneCacheCoverage`; finding `Check: "low-cache-hit"` Severity `"info"` when `Requests >= 1000 && HitRatePct < 5` — Evidence `"X requests, Y% edge-cache hit over <window>"`, Fix `"origin Cache-Control headers on cacheable responses, or a zone Cache Rule (HTML needs a Cache Rule — s-maxage alone does nothing)"`. ZonesChecked count on the report.
  - `cmd/efficiency.go`: wire zones via `getZoneService()`; the check runs only when the zone service resolves (same graceful pattern as the R2 client).
- Alert class P-10 (cache part): extend the alert condition registry (read `pkg/cosmoflare/alerts.go` + `internal/webhook/evaluator.go` patterns from FEAT-047) with `zone-cache-hit-low` (threshold on HitRatePct, sustained N windows) evaluated from a batched per-zone coverage query on the alerts slow loop. If the registry wiring exceeds ~100 lines of change, STOP after the library + efficiency check and note the alert as follow-up — do not half-wire an evaluator.

TDD per wave-1 conventions: `TestAnalyticsZoneCacheCoverage` (mock fixture with the verified response shape, incl. empty zone), `TestEfficiencyLowCacheHit` (zone above/below thresholds, no finding when requests < 1000), `TestEfficiencyCleanAccount` extended (busy zone → no finding), cmd tests for the wired section. Gate: `go test ./pkg/cosmoflare/ ./cmd/` + `go vet ./...` + live smoke `cosmoflare efficiency --json` (zone findings expected — the probe zone runs ~0.4%).

## Task W2-B: `cosmoflare analytics kv` (deliverable P-07)

**Model:** glm-turbo · **Files:** modify `cmd/analytics.go`, `cmd/analytics_test.go` — PARALLEL-SAFE with W2-A (no shared files).

- Add a `kv` subcommand to the analytics command (read its current structure — R2 today): `cosmoflare analytics kv [--window days]` lists per-namespace `requests` totals over the window from `AnalyticsService.KVOperations`, sorted desc, with `--json` envelope matching the command's existing JSON style. Flag validation mirrors wave-1's `--window` (1-90). Tests: `TestAnalyticsKVCmd_*` (registered, flag default, JSON envelope from a stubbed service seam — copy the command's existing test seams).

## Task W2-C: HTML caching-trap detector (deliverable P-08) — AFTER W2-A merges (shares efficiency.go)

**Model:** sonnet · **Files:** create `pkg/cosmoflare/caching.go`, `pkg/cosmoflare/caching_test.go`; modify `pkg/cosmoflare/efficiency.go` (+ its test) only to register the check.

- `func DetectHTMLCacheTrap(ctx, zone Zone, probe func(url string) (*ProbeResult, error)) (*TrapFinding, error)`: GET `https://<zone.Name>/` (injectable probe for tests); trap = response is `text/html` AND (`cache-control` carries `s-maxage`/`max-age` ≥ 60 public) AND `cf-cache-status` is `DYNAMIC` (or absent). On trap, cross-check zone Cache Rules via `GET zones/:id/cache/rules` (cloudflare-go or raw REST — read zone.go for the client pattern; if the endpoint 404s on this token, treat as "no rule visible" and say so in the evidence). Finding: `Check: "html-cache-trap"` Severity `"warning"`, Evidence quoting the three observed signals, Fix: "origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule for the HTML paths or emit Cache-Control on the response at the edge".
- Wire into `EfficiencyService.Report` under the zone loop from W2-A (one probe per zone, bounded; skip zones that failed the coverage check's request floor).
- Tests: mocked probe responses (trap / clean / non-HTML / error) — no live fetches in unit tests; live smoke covers reality.

## Task W2-D: KV ops in the usage view (deliverable P-09) — AFTER W2-A merges (shares analytics.go usage collection if it lives there; verify file first)

**Model:** glm-turbo · **Files:** modify `pkg/cosmoflare/usage.go`, `pkg/cosmoflare/usage_test.go` (and the limits catalog ONLY if a KV row must be added there — read how d1.rows_read_monthly got its Name/Unit).

- Add `kv.requests_monthly` dimension: sum `KVOperations` requests across namespaces over the cycle; limit from the plan catalog (free 100k/day reads + 1k/day writes are DAILY — model as monthly totals with the catalog's convention; if the catalog has no KV row, follow FEAT-048's catalog-verification process note in the dimension's source_url and use the documented free/paid totals, quoting the source URL in the test fixture).

## Sequencing

1. W2-A and W2-B in parallel (disjoint files).
2. After W2-A merges: W2-C, then W2-D (each sequential — shared efficiency.go/usage.go).
3. P-10's KV-ops alert class: fold into W2-D if the registry pattern is clear from the code; otherwise note as follow-up.
4. Cost real usage: NOT this wave's scope (deliberate — brainstorm decision D-cost).

## Acceptance

- `cosmoflare efficiency --json` includes zone cache findings on the live account (expect at least one low-hit zone).
- `cosmoflare analytics kv --json` returns per-namespace totals.
- `cosmoflare usage` shows the KV row.
- Full `go test ./pkg/cosmoflare/ ./cmd/` green; `go vet ./...` clean; one live smoke per new surface.
