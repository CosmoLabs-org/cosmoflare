---
brainstorm_ref: docs/brainstorming/2026-10-14-cli-audit-efficiency.md
branch: master
completed: "2026-10-15T07:48:12+04:00"
created: "2026-10-15T10:05:00+04:00"
deliverables:
    - id: P-11
      title: ProjectCacheScanner in pkg/cosmoflare/patterns.go — source-derived cached-route/invalidation/aggregate inventory
    - id: P-12
      title: TTL verdict engine in pkg/cosmoflare/ttl.go — cadence × invalidation coverage grades with the asymmetry rule
    - id: P-13
      title: cosmoflare efficiency caching — P1-P5 pattern findings from source, --json, severity 'critical' for P2
    - id: P-14
      title: cosmoflare efficiency ttl — verdicts wired to real D1 write cadence + live calibration on Churches-app
goals_completed: 8
goals_total: 8
implemented_commits:
    - covers:
        - P-11
      sha: 2ec88aa84a19
    - covers:
        - P-12
      sha: 43831a7e5b51
    - covers:
        - P-13
      sha: 2ba53e436a38
    - covers:
        - P-14
      sha: 80371fe3e537
plan_ref: docs/planning-mode/2026-10-14-cli-audit-efficiency-wave2.md
related_issues:
    - FEAT-067
    - FEAT-068
    - FEAT-070
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags:
    - efficiency
    - remediation
    - cli
    - wave-3
    - caching
    - ttl
title: 'CLI audit + efficiency — wave 3: remediation (FEAT-067 patterns + FEAT-068 TTL strategy)'
---

# Cosmoflare efficiency — Wave 3 Implementation Plan (remediation)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans, task-by-task.

**Goal:** From findings to fixes — detect the Churches-app caching patterns (P1-P5) in any project's Worker source and grade every cached route's TTL against measured data-change cadence and invalidation coverage.

**Architecture:** Waves 1-2 were pure CF-API dataset checks. Wave 3 adds a second input axis: the project's TypeScript Worker source. A new `ProjectCacheScanner` (regex-based, corpus-calibrated) builds a `ProjectCacheInventory`; two new subcommands turn it into findings — `efficiency caching` (pattern violations, each citing its P-number) and `efficiency ttl` (verdicts combining the inventory with wave-2's real `D1UsageService.DailyUsage` rows-written). Patch emission (`--fix`) and P6-P9 are deliberately out of scope (stretch per brainstorm).

**Tech Stack:** Go 1.26, stdlib `regexp` + `filepath.WalkDir`, cobra (existing), no new dependencies.

**Spec:** `docs/brainstorming/2026-10-14-cli-audit-efficiency.md` (wave-3 section, lines 130-154) + FEAT-067/068/070 issue bodies.

## Global Constraints

(same as waves 1-2: real numbers only — never modeled; `--json` envelope on every new surface; agent-readable findings (what/why/how-to-fix); no N+1 API calls; TDD red→green per named test; conventional commits; never touch `cosmoflare audit`; no push — repo-local commits only)

## Verified facts (2026-10-15, in-tree + corpus — trust these, do not re-guess)

1. `D1UsageService.DailyUsage(ctx, databaseID, days) ([]D1DailyUsage, error)` exists (`pkg/cosmoflare/d1_usage.go:76`) — `D1DailyUsage.RowsWritten int64` per UTC day is the cadence signal. Database-wide, not per-table (D1 analytics has no table dimension — document this honestly in help text).
2. `D1Service.List(ctx) ([]*D1Database, error)` exists (`pkg/cosmoflare/d1.go:106`) — single-database accounts can auto-resolve `--database`.
3. The calibration corpus is local: `~/PROJECTS/Churches-app/workers/src/`. Verified in-tree shapes (fixtures copy these verbatim):
   - `kvCache({ keyFn: () => 'api:meta:stats', ttl: 600 }),` — static key
   - `kvCache({ keyFn: (c) => \`api:meta:cities:${c.req.query('country_code') || 'all'}\`, ttl: 21600 }),` — query-aware template key (options text contains `${` — naive `\{[^{}]*\}` matching FAILS on this line)
   - `kvCache({ keyFn: (c) => (c.req.query('country_code') ? \`api:meta:states:${c.req.query('country_code')}\` : null), ttl: 21600 }),` — null escape hatch
   - Invalidation: `cache.delete(churchCacheKey(slug))` and `cache.delete(\`page:/church/${slug}\`)` inside `invalidateChurchCache` (`middleware/cache.ts:86-94`)
   - Middleware behavior: `middleware/cache.ts` emits `Cache-Control: public, max-age=300` on HIT and MISS (lines 59, 73) and guards `c.res.status === 200` (line 67)
   - Route files mount Hono sub-routers: route defs look like `xxxRoutes.get('/stats', kvCache({...}), async (c) => {` — the kvCache attachment sits in the middleware-args window after the path literal
4. The known live stale case (FEAT-068 origin story): `api:meta:*` keys carry ttl 21600 with NO invalidation call touching that prefix — scanning the corpus MUST produce exactly this finding. Runtime semantics: invalidations touch `api:/api/churches/*` (via `churchCacheKey`) and `page:/church/*`; in the SCANNER's inventory the `cache.delete(churchCacheKey(slug))` call yields the bare name `churchCacheKey` (zero colon-segments — covers nothing), and `cache.delete(\`page:/church/${slug}\`)` yields literal head `page:/church/`; the `api:/api/churches/` prefix lives inside the `churchCacheKey` function body, which rule 6 does not dereference — outcome unchanged: no invalidation covers `api:meta:*`. Shared-prefix rule: count shared leading colon-segments; `api:/api/churches/x` vs `api:meta:countries` share depth 1 (`api`) → NOT covered; depth ≥ 2 → covered.
5. `EfficiencyFinding{Resource, Check, Severity, Evidence, Fix}` (`pkg/cosmoflare/efficiency.go:11`) — Severity today is `"warning" | "info"`. P2 (auth+cache) needs `"critical"` added to the allowed set (comment update + cmd glyph `‼`).
6. `cmd/efficiency.go` is a single cobra command with `efficiencyCollectFn` seam + `efficiencyJSONOut/efficiencyTextOut` writer seams — new subcommands follow this pattern, each in its OWN file with its own `init()` calling `efficiencyCmd.AddCommand(...)` (var init precedes package init regardless of file order — parallel-safe).
7. `go test ./pkg/cosmoflare/ ./cmd/ -timeout 60s` and `go vet ./...` are the gates; live smoke needs `AccountID`/`APIToken` configured.

## Reference files

`pkg/cosmoflare/efficiency.go` (findings model, nil-able check pattern) · `pkg/cosmoflare/caching.go` (wave-2 detector style: pure verdict fn + service wrapper) · `pkg/cosmoflare/d1_usage.go` (cadence input) · `cmd/efficiency.go` (command seams) · `cmd/analytics.go` (subcommand + `--json` envelope style) · `~/PROJECTS/Churches-app/workers/src/` (calibration corpus, read-only)

---

## Task W3-A: ProjectCacheScanner (deliverable P-11)

**Model:** sonnet · **Files:** create `pkg/cosmoflare/patterns.go`, `pkg/cosmoflare/patterns_test.go`, fixture tree `pkg/cosmoflare/testdata/project/` (see Step 1)

**Interfaces (exact — later tasks compile against these):**

```go
// CachedRoute is one kvCache attachment found in project source.
type CachedRoute struct {
    File       string `json:"file"`        // relative to scan root, '/'-separated
    Line       int    `json:"line"`
    Route      string `json:"route"`       // "GET /api/meta/cities" form; path as written
    KeyFn        bool `json:"key_fn"`
    QueryAware   bool `json:"query_aware"`    // kvCache options text contains ".query("
    HandlerQuery bool `json:"handler_query"`  // route window contains ".query(" outside the kvCache options
    TTL        int    `json:"ttl"`         // seconds; 0 = not specified (middleware default)
    KeyPrefix  string `json:"key_prefix"`  // static head before '${' or ""; "" = dynamic/function-derived
    HasAuth    bool   `json:"has_auth"`
}

// InvalidationCall is one cache-delete site with a string-ish argument.
type InvalidationCall struct {
    File       string `json:"file"`
    Line       int    `json:"line"`
    KeyPattern string `json:"key_pattern"` // static head before '${' or ")"
}

// AggregateQuery is a whole-table aggregate in an uncached route handler.
type AggregateQuery struct {
    File  string `json:"file"`
    Line  int    `json:"line"`
    Route string `json:"route"`
    SQL   string `json:"sql"`
}

// MiddlewareBehavior reports what the file defining kvCache itself does.
type MiddlewareBehavior struct {
    File              string `json:"file"`
    EmitsCacheControl bool   `json:"emits_cache_control"`
    OnlyStatus200     bool   `json:"only_status_200"`
}

// ProjectCacheInventory is the full source-derived caching picture.
type ProjectCacheInventory struct {
    Root           string               `json:"root"`
    Routes         []CachedRoute        `json:"routes"`
    Invalidation   []InvalidationCall   `json:"invalidation"`
    Aggregates     []AggregateQuery     `json:"aggregates"`
    Middleware     []MiddlewareBehavior `json:"middleware"`
    TotalGETRoutes int                  `json:"total_get_routes"`
    FilesScanned   int                  `json:"files_scanned"`
}

// ScanProjectCache walks root and builds the inventory. Read-only.
func ScanProjectCache(root string) (*ProjectCacheInventory, error)
```

**Scanning rules (pin these; the corpus lines in Verified facts 3 are the ground truth):**

1. Walk with `filepath.WalkDir`; scan `*.ts`; skip paths containing `node_modules/`, `.git/`, `dist/`, `build/`, and names matching `*_test.ts`/`*.spec.ts`. Count scanned files.
2. **kvCache attachment:** find `kvCache(`; the options text runs from there to the next `})` (join lines, cap 10 lines — handles the template-literal `${` bodies and the multi-line block-bodied keyFns in churches/dioceses/mass). From options extract: `KeyFn` (`keyFn:` present), `QueryAware` (options contains `.query(`), `TTL` (`ttl:\s*(\d+)`), `KeyPrefix` (first string/template literal in the keyFn VALUE — for expression arrows the first literal after `=>`, for block bodies the first literal after `return`, head truncated at `${` or closing quote; `""` when that expression is a function call like `churchCacheKey(...)`). Do NOT take the first quoted literal in the raw options text: block-bodied keyFns (churches `/search`, dioceses `/search`, mass `/city-window`) contain incidental argument literals (`' '` inside `.replace(...)`, `'7'` defaults) before the real key template — for `() => 'api:meta:stats'`-style static keys the literal follows `=>` directly. `HandlerQuery` (rule 3's route window contains `.query(`) is set after the window is known.
3. **Route attribution:** scan upward from the kvCache line (max 5 lines) for `\.(get|post|put|patch|delete)\(\s*['"\`]/([^'"\`]*)['"\`]` — the path literal MUST start with `/` (Hono route paths always do; this excludes Hono context getters `c.get('user')`, `formData.get('file')`, and map gets — the corpus contains ~23 such quote-arg `.get(` calls that a bare quote match would miscount as routes). Route = `METHOD path`, path as written at the sub-router (mount prefixes from `index.ts` `app.route(...)` are NOT resolved). Route *window* = from that line to the matching route close (next line matching `^\s*\)\);?$` or 40 lines, whichever first).
4. **HasAuth:** route window text matches `(?i)auth` as a substring (catches `authMiddleware`, `requireAuth`, plain `auth` — `\bauth\b` would NOT match `authMiddleware`/`requireAuth`, there is no word boundary inside those identifiers). Collateral matches (`author`, `authorization`) are possible on arbitrary projects; the corpus contains zero case-insensitive `auth` substrings in the four kvCache route files, so the calibration gate below catches any surprise.
5. **TotalGETRoutes:** count every method==`get` route-def match across all scanned files (cached or not), using rule 3's `/`-anchored regex (corpus ground truth: 47 GET routes; a bare quote match yields 70 — if the scanner reports a much larger number than expected, the anchor is missing).
6. **Invalidation:** every `.delete(` whose argument (rest of logical line) contains a `'`, `"`, or `` ` `` literal → `KeyPattern` = literal head up to `${` or closing quote. Function-call args with no literal (e.g. `churchCacheKey(slug)`) → `KeyPattern` = the call text up to `(`, e.g. `churchCacheKey`. False positives (array deletes) are acceptable — coverage detection must err toward covered.
7. **Aggregates:** for each route window WITHOUT a kvCache attachment, find quoted/backtick literals containing `SELECT` (case-insensitive) AND (`COUNT(` or `GROUP BY`) AND NOT `WHERE` — the WHERE test is also case-insensitive, so `${where}` interpolation (churches.ts:133) counts as having a WHERE → one AggregateQuery with the SQL literal.
8. **Middleware:** any file containing `export const kvCache` or `function kvCache(` → MiddlewareBehavior; `EmitsCacheControl` = body contains `Cache-Control`; `OnlyStatus200` = body contains `=== 200`.
9. Errors: nonexistent root → error `"source directory %s does not exist"`. Unreadable file → skip (never fail the scan).

**TDD (red→green each):**

- [x] **Step 1 — fixtures:** create `pkg/cosmoflare/testdata/project/` mirroring the corpus shapes: `routes/meta.ts` (static-key route ttl 600, query-aware template route ttl 21600, null-escape route), `routes/user.ts` (route with `authMiddleware` + kvCache → P2 shape), `routes/stats.ts` (uncached route with backtick SQL `` `SELECT COUNT(*) FROM churches GROUP BY country` ``), `middleware/cache.ts` (kvCache definition emitting Cache-Control + 200-guard), plus `routes/old.spec.ts` (must be skipped). Copy the Verified-facts-3 lines verbatim where possible.
- [x] **Step 2 — failing tests:** `TestScanProjectCacheInventory` (routes count, per-route fields incl. QueryAware on the cities shape, KeyPrefix `api:meta:stats`, HasAuth on user.ts), `TestScanProjectCacheSkipsTestFiles` (spec file absent), `TestScanProjectCacheInvalidation` (both delete sites found — the `page:/church/${slug}` template yields literal head `page:/church/`, and the `churchCacheKey(slug)` function-call arg yields `churchCacheKey`), `TestScanProjectCacheAggregates` (stats route found, WHERE'd SQL not), `TestScanProjectCacheMiddlewareBehavior` (both booleans true), `TestScanProjectCacheMissingRoot` (error mentions path).
- [x] **Step 3:** run `go test ./pkg/cosmoflare/ -run TestScanProjectCache -v` — all FAIL (undefined).
- [x] **Step 4 — implement** `patterns.go` per the rules above. Re-run — all PASS. Add `go vet ./...`.
- [x] **Step 5 — commit:** `feat(efficiency): project cache scanner — source-derived route/invalidation/aggregate inventory (P-11, FEAT-067)`

## Task W3-B: TTL verdict engine (deliverable P-12)

**Model:** sonnet · **Files:** create `pkg/cosmoflare/ttl.go`, `pkg/cosmoflare/ttl_test.go` · AFTER W3-A merges (consumes its types)

**Interfaces:**

```go
// TTLVerdict grades one cached route's lifetime (FEAT-068).
type TTLVerdict struct {
    Route        string `json:"route"`
    KeyPrefix    string `json:"key_prefix"`
    TTLSeconds   int    `json:"ttl_seconds"`
    WritesPerDay int64  `json:"writes_per_day"` // measured, database-wide
    Cadence      string `json:"cadence"`        // "rare" | "daily" | "hot"
    Covered      bool   `json:"invalidation_covered"`
    CoverageKnown bool  `json:"coverage_known"` // false when KeyPrefix == ""
    Verdict      string `json:"verdict"`        // "appropriate" | "raise" | "stale-risk" | "no-invalidation"
    Reason       string `json:"reason"`
}

// Cadence thresholds (named so tests pin them).
const (
    TTLCadenceRareBelow  = 1     // writes/day < 1 → rare
    TTLCadenceHotAbove   = 1000  // writes/day > 1000 → hot
    TTLStaleRiskAbove    = 3600  // uncovered TTL past 1h with non-rare cadence → stale-risk
    TTLRaiseBelow        = 21600 // covered + rare + TTL under 6h → raise
)

// InvalidationCovered reports whether any invalidation literal shares ≥ 2
// leading colon-segments with the route's key prefix (Verified fact 4).
func InvalidationCovered(routePrefix string, invalidations []InvalidationCall) bool

// GradeTTLs grades every inventory route. writesPerDay is the measured
// database-wide D1 rows-written average (caller computes from DailyUsage).
func GradeTTLs(inv *ProjectCacheInventory, writesPerDay int64) []TTLVerdict
```

**Verdict table (encode exactly; every Reason carries the asymmetry sentence):**

| condition | verdict |
|---|---|
| `KeyPrefix == ""` | `appropriate` — Reason notes coverage not statically determinable (function-derived key) |
| `!Covered && ttl > TTLStaleRiskAbove && cadence != "rare"` | `stale-risk` — "nothing purges this prefix; entries can serve data up to TTL old after a write" |
| `!Covered && cadence != "rare" && ttl <= TTLStaleRiskAbove` | `no-invalidation` — "safe today only because the TTL is short; adding invalidation is the upgrade path to longer TTLs" |
| `Covered && cadence == "rare" && ttl < TTLRaiseBelow` | `raise` — "rare-change data with invalidation covered can safely take 6-24h" |
| else | `appropriate` |

Every Reason ends with the fixed asymmetry string (export it as `TTLAsymmetryNote = "KV TTLs can be purged by code, but the browser/edge Cache-Control layer cannot be targeted from a migration — keep Cache-Control max-age short (≤ 300s) even when KV TTLs are hours."`).

**TDD:**

- [x] **Step 1 — failing tests:** `TestInvalidationCovered` (depth-1 shared `api` NOT covered; `api:meta:countries` vs invalidation `api:meta:` head covered; empty prefix → false), `TestGradeTTLsStaleRisk` (meta-countries shape: prefix `api:meta:countries`, ttl 21600, uncovered, writesPerDay 500 → `stale-risk` — the live corpus case), `TestGradeTTLsNoInvalidation` (ttl 600, uncovered, daily → `no-invalidation`), `TestGradeTTLsRaise` (covered, writesPerDay 0 — sub-daily cadence, the int64 seam cannot carry 0.2 → `raise`), `TestGradeTTLsAppropriate` (covered, daily, ttl 3600), `TestGradeTTLsUnknownCoverage` (KeyPrefix "" → appropriate + coverage_known false), `TestGradeTTLsAsymmetryNote` (every Reason contains `TTLAsymmetryNote`).
- [x] **Step 2:** run — FAIL (undefined). **Step 3 — implement** `ttl.go`; re-run — PASS; `go vet ./...`.
- [x] **Step 4 — commit:** `feat(efficiency): TTL verdict engine — cadence × invalidation coverage grades (P-12, FEAT-068)`

## Task W3-C: `cosmoflare efficiency caching` (deliverable P-13)

**Model:** sonnet · **Files:** create `cmd/efficiency_caching.go`, `cmd/efficiency_caching_test.go`; modify `pkg/cosmoflare/efficiency.go:14` (Severity comment: `"critical" | "warning" | "info"`) and `cmd/efficiency.go:125-127` (table glyph: `‼` for critical) · AFTER W3-A merges (parallel-safe with W3-B: disjoint files)

**Behavior:** `efficiency caching [--source DIR] [--json]` (default source `.`). No API credentials needed — pure source analysis; say so in Long help. Output: findings table + summary line `X of Y GET routes cached`.

**Finding mapping (exact Check names + severities):**

| condition | Check | Severity | Fix text (cite pattern + reference) |
|---|---|---|---|
| `!KeyFn && HandlerQuery` | `p1-path-only-cache-key` | warning | "path-only KV key collapses query variants onto one entry — add a keyFn including every query param the handler reads (P1; cf. Churches-app meta.ts cities keyFn)" |
| `KeyFn && !QueryAware && HandlerQuery` | `p1-key-misses-query` | warning | "handler reads query params the key does not include (P1)" |
| `HasAuth` | `p2-auth-cache-leak` | **critical** | "cached route carries auth — cross-user leak by construction; remove the cache middleware from this route (P2)" |
| AggregateQuery found | `p3-unmaterialized-aggregate` | warning | "whole-table aggregate per request — materialize by cron into KV and serve the snapshot (P3; 40-line reference: Churches-app jobs/stats-materialize.ts)" |
| middleware exists && `!EmitsCacheControl` | `p4-no-cache-control` | warning | "cache HIT still costs an invocation + KV read — emit `Cache-Control: public, max-age=300` on cached responses (P4)" |
| middleware exists && `!OnlyStatus200` | `p5-caches-non-200` | warning | "guard the put on status 200 — errors and 404s must bypass (P5)" |

Resource field: `route:GET /cities` — path as written at the sub-router (the scanner does not resolve `index.ts` mount prefixes; `/api/meta/cities` in the mounted app is `GET /cities` in `routes/meta.ts`) + `file:line` in Evidence so the user can locate it. P4/P5 fire once (middleware-level), Resource `middleware:<scanned file>` from the discovered `MiddlewareBehavior.File` (corpus: `middleware:middleware/cache.ts` — do not hardcode `cache.ts`, other projects name it differently).

**TDD:** `TestEfficiencyCachingCmdRegistered`, `TestEfficiencyCachingFindings` (stub `ScanProjectCache` via package seam `efficiencyCachingScanFn = cosmoflare.ScanProjectCache` — same seam pattern as `efficiencyCollectFn`; fixture inventory hitting every row of the table above, assert Check+Severity+Evidence), `TestEfficiencyCachingJSONEnvelope` (status/data envelope), `TestEfficiencyCachingMissingSource` (error, exit non-zero), `TestEfficiencyCachingSummary` ("3 of 5 GET routes cached"). Red → implement → green → vet.

**Commit:** `feat(efficiency): efficiency caching — P1-P5 pattern findings from Worker source (P-13, FEAT-067)`

## Task W3-D: `cosmoflare efficiency ttl` + calibration (deliverable P-14)

**Model:** glm-turbo · **Files:** create `cmd/efficiency_ttl.go`, `cmd/efficiency_ttl_test.go` · AFTER W3-B and W3-C merge (consumes engine; parent already has subcommand precedent)

**Behavior:** `efficiency ttl [--source DIR] [--database ID] [--window N] [--json]` (source default `.`; window default 7, 1-90 validated like `efficiency`). Requires AccountID+APIToken (real numbers rule). `--database` omitted: resolve via `D1Service.List` — exactly one database → use it; more → error listing candidates by name/ID. Cadence = `sum(DailyUsage.RowsWritten)/days` over the window. Output: one line per cached route — route, TTL, cadence, coverage, verdict, reason — plus findings for `stale-risk` (warning) and `raise`/`no-invalidation` (info) as `EfficiencyFinding`s with Check `ttl-stale-risk` / `ttl-raise` / `ttl-no-invalidation`, Fix text quoting the verdict's upgrade path. JSON envelope: `{status, data: {verdicts, findings, database, writes_per_day, window_days}}`.

**Seams:** `efficiencyTTLScanFn = cosmoflare.ScanProjectCache`, `efficiencyTTLCadenceFn func(databaseID string, days int) (int64, error)` (production: D1UsageService.DailyUsage sums; tests stub).

**TDD:** `TestEfficiencyTTLCmdRegistered`, `TestEfficiencyTTLVerdictsJSON` (stubbed inventory + cadence → assert verdict fields + finding severities), `TestEfficiencyTTLWindowValidation` (0 and 91 rejected), `TestEfficiencyTTLDatabaseAutoResolve` (stub list → single DB picked; two DBs → error names both), `TestEfficiencyTTLRequiresCreds`. Red → implement → green → vet.

**Live calibration (acceptance — run against the real corpus, read the output yourself):**

```bash
go build -o build/cosmoflare . && ./build/cosmoflare efficiency caching --source ~/PROJECTS/Churches-app/workers/src
./build/cosmoflare efficiency ttl --source ~/PROJECTS/Churches-app/workers/src --window 7
./build/cosmoflare efficiency --json | jq '.data.findings | length'   # wave-1/2 regression check
```

Expected: caching reports `19 of 28 GET routes cached` (cached: meta 8 + dioceses 6 + churches 4 + mass 1 — all multi-line defs, `.get(` at line end with the path on the next line; uncached: 9 single-line defs across index/media/churches/user/admin/preview-codes/mass — 28 is the directly measured corpus truth 2026-10-15: 19 multiline + 9 single-line; the pre-calibration draft's "47" was an overcount), zero P1/P2/P3/P4/P5 findings (the corpus is clean — it IS the pattern source; any P-finding on the corpus is a scanner false positive to fix before merge; P3 is zero because every `GROUP BY`/`COUNT(` literal sits inside a kvCache'd route or carries a WHERE/`${where}` — the WHERE-less whole-table aggregates live in `jobs/stats-materialize.ts`, which is a cron job, not a route window). ttl reports `stale-risk` (or `no-invalidation`) on `api:meta:*` prefixes — the known live case (Verified fact 4) — and `appropriate`+coverage_known=false on function-keyed church routes. Full `go test ./pkg/cosmoflare/ ./cmd/` green; `go vet ./...` clean.

**Commit:** `feat(efficiency): efficiency ttl — verdicts from real D1 write cadence (P-14, FEAT-068/070)`

## Sequencing

1. W3-A first (foundation types).
2. W3-B ∥ W3-C in parallel (disjoint files, both consume only W3-A).
3. W3-D last (consumes B's engine, C's subcommand precedent).
4. Out of scope, do not start: `--fix` patch emission, P6-P9 checks, FEAT-070 indexing rules (EXPLAIN discipline), Ops-app wiring of these findings.

## Acceptance

- `cosmoflare efficiency caching --source <any Worker project>` returns P-findings with pattern citations; clean on the Churches-app corpus.
- `cosmoflare efficiency ttl` verdicts every cached route with measured writes/day; reproduces the known `api:meta:*` stale case.
- Full suite + vet green; wave-1/2 `efficiency` output unchanged.
