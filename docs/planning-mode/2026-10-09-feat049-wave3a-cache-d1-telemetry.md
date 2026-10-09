---
title: FEAT-049 Wave 3a — Zone cache + D1 rows-read telemetry alerts
created: "2026-10-09T00:10:00+04:00"
issue: FEAT-049
roadmap: ROAD-101
brainstorm_ref: docs/brainstorming/2026-10-08-feat049-wave3a-cache-d1-telemetry.md
status: PLANNED
deliverables:
    - P-01: ZoneCache batched cacheStatus query + MissPct/UncachedPct math (pkg/cosmoflare/analytics_cache.go)
    - P-02: D1RowsRead account-wide per-database query (pkg/cosmoflare/analytics_d1rows.go)
    - P-03: AlertRule.Exclude + zone/d1 rule services + --exclude CLI flag
    - P-04: Evaluator — zone/d1 scopes, shared FireState, per-scope cooldown, 2x escalation, telemetry-gap pages, human-readable row counts
    - P-05: Watch + check wiring — name caches, telemetry collection, help text
    - P-06: D1 monthly rows-read catalog row + additive CollectUsage dimension
    - P-07: Opt-in live smoke test (build tag live)
    - P-08: USAGE.md + README documentation (parallel doc agents)
---

# FEAT-049 Wave 3a Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `alerts watch` pages when a zone's cache misses or never caches, and when a D1 database scans too many rows — naming the zone or database — without flooding the pager, and pages once when that telemetry itself goes dark.

**Architecture:** Two new batched analytics queries (zones by `cacheStatus`, D1 by `databaseId`) feed two new fan-out scopes (`zone`, `d1`) in the existing condition registry and evaluator. Cooldown state moves into a `FireState` the watch keeps across cycles; only `zone`/`d1` scopes get a 1h cooldown (with re-page on doubling), every other scope keeps re-paging each cycle. Telemetry for the new scopes is additive: a failure records a gap that pages once an hour instead of stopping the cycle.

**Tech Stack:** Go 1.26, cobra, net/http/httptest mocks, Cloudflare GraphQL Analytics API.

**Spec:** `docs/brainstorming/2026-10-08-feat049-wave3a-cache-d1-telemetry.md` (decisions D1–D16, code constraints 1–9).

## Decisions for the operator

Big items, in one place. **Decided** = settled in this session (who chose in brackets). **Open** = needs an operator call; the plan's default applies until then.

| # | Item | Status | What the plan does | If you choose otherwise |
|---|------|--------|--------------------|-------------------------|
| O1 | Split FEAT-049: telemetry now, probes (header sweep, repeat-probe, auth-route cache check, unused bindings) later under FEAT-050 | Decided (operator) | Wave 3a = telemetry only | — |
| O2 | `cacheStatus = none` (undocumented) | Decided (operator) | Excluded from both ratios, still in JSON | Counting it as uncached makes cosmolabs.org read ~84% uncached today |
| O3 | Re-paging policy | Decided (operator) | zone/d1: 1h per-scope cooldown, kept **in memory** across watch cycles; script/do/account: unchanged (re-page every cycle). Note: a watch restart re-pages every current zone/d1 breach once; a recovery and re-breach inside the hour stays quiet | — |
| O4 | Escalation inside the cooldown | Decided (operator) | Re-page if the value reaches 2× the last paged value. Note: only d1 can really escalate — a % above 50 cannot double. Applies only where the watch sets a scope cooldown, so `cosmoflare serve` is unchanged | — |
| O5 | Telemetry-gap page | Decided (operator) | One page per dataset per hour, only when an enabled rule depends on the failed dataset | — |
| O6 | Per-zone / per-database exclude list | Decided (operator) | `--exclude name1,name2` on rules; matches zone or database names, case-insensitive | — |
| O7 | Opt-in live smoke test | Decided (operator) | `go test -tags live`; never runs in the default suite | — |
| O8 | Nothing pushed, released or published | Decided (operator) | All merges stay on local master; v0.33.0 held | — |
| O9 | **Pager severity** — every page is `info` today (`cmd/alerts_watch.go:306`); gap pages and escalations are also `info` | **Open** | Unchanged; escalation and gap pages say so in the message text | A severity field on rules is a separate small feature (affects every rule) |
| O10 | **Starter thresholds** — no rules exist until you create them | Decided (live dry run 2026-10-09) | Docs suggest: `zone-uncached-requests` 10000/24h, `zone-cache-miss-pct` 50, `d1-rows-read` 1e9 (≈ 30B/month, above the 25B Paid allowance). Dry run on the live account: 7 pages (2 uncached-volume, 4 miss, 1 D1) | A `--preset` that creates these rules is a separate small feature |
| O11 | **Rolling 24h vs Free-plan UTC day** — D1 Free limits reset at 00:00 UTC; the watch window is rolling 24h | **Open (deferred)** | Documented limitation; your account is Paid | A per-UTC-day D1 condition later |
| O12 | **New rule service names** `zone` and `d1` | Decided (Claude, low risk) | Added to rule validation so rules can say `--service d1` | Rules could reuse `dns`/`workers` instead — misleading in history |
| O13 | **API budget** — the watch adds ⌈zones/10⌉ + 1 GraphQL queries per cycle, plus 2 REST list calls per 15 min, only when a zone/d1 rule is enabled | Decided (Claude) | 42 active zones (all Free, counted 2026-10-09) → 6 extra queries/cycle; ~70 per 5 min total at 60s, under the 300 limit | — |
| O14 | **mycarguide-db** (2.95B rows/day) | Open (MyCarGuide) | Feedback delivered to MyCarGuide; no change here | — |
| O15 | **KV hit rate** | Decided (Claude, evidence) | Out — `result` field semantics undocumented | Revisit when documented |
| O16 | **D1 in the usage-pct conditions** — the new `d1.rows_read_monthly` dimension feeds `usage-pct`/`usage-projected-pct` (account scope, re-page every cycle). At ~88B/month vs 25B, D1 sits near 350% projected: a usage-projected rule would page every 60s about D1 and hide the other dimensions in the max | Decided (operator, 2026-10-09) | D1 shows in `cosmoflare usage` but `usageMax` skips `d1.*` dimensions, so usage-pct/usage-projected-pct ignore D1; `d1-rows-read` (hourly cooldown) is the D1 alert (T4) | Including D1 would page every cycle until mycarguide-db is fixed |
| O17 | **zone/d1 rules under `cosmoflare serve`** — serve collects only Workers/R2/DO today, so zone/d1 rules would never fire there and nothing would say so | Decided (operator, 2026-10-09) | Watch and check only; `cosmoflare serve` logs one warning at start when a zone/d1 rule is enabled (T5); documented in USAGE.md | Wiring serve is a few lines later |
| O18 | **Chronic-breach volume** — zone ratios over 24h are structural, not incidents: churches.app (~71% uncached) pages every hour, 24×/day, until fixed or excluded; 42 zones can burst on first start | Decided (operator, after dry run 2026-10-09) | Dry run at starter thresholds: zone-uncached-pct ≥ 80 paged 23 of 42 zones (Workers responses are cacheStatus=dynamic by design, 97-100% per site), 27 pages total. Replaced by `zone-uncached-requests` (absolute dynamic+bypass eyeball requests, no floor): 10000/24h pages 2 zones; whole starter set 7 pages. Percent page text prints one decimal (merged cf8d0b7) | — |

## Global Constraints

- Every external fact comes from live introspection or a cited docs page (design record "Live schema facts"); values from memory are forbidden.
- Registry stays the single definition site (FEAT-015); `TestConditionRegistryCoverage`-style coverage test must stay green.
- One batched query per dataset per cycle; zones in batches of ≤ 10 (`zoneTag_in`, documented hard limit).
- `--json` + rich help on every changed command; deterministic exit codes.
- Conventional commits; no AI attribution trailers; no push, no tag, no release (O8).
- Go: `go build ./... && go vet ./...` exit 0; affected packages green.

## Execution Model

| Wave | Tasks | Executor | Base |
|------|-------|----------|------|
| 1 (parallel) | T1, T2, T3 | GLM (`ccs glm-agent exec`, one worktree each) | master |
| 2 | T4 + T5 (one worktree, coupled) | Opus (main session) | master after wave 1 merges |
| 2 (parallel with T4) | T6, T7 | GLM | master after wave 1 merges |
| 3 (parallel) | T8a USAGE.md, T8b README | GLM | master after wave 2 merges |

Every GLM worktree passes the S334 gate (diff read + package tests re-run by Opus in the worktree) before `ccs merge`. Merge order: T1 → T2 → T3 → T4/T5 → T6 → T7 → T8. Integration point: T5 (watch wiring) consumes T1–T4.

---

### Task 1: ZoneCache batched query + ratio math (GLM)

**Files:**
- Create: `pkg/cosmoflare/analytics_cache.go`
- Create: `pkg/cosmoflare/analytics_cache_test.go`

**Interfaces:**
- Consumes: `AnalyticsService.query`, `validationError`, `validateAnalyticsWindow`, `AnalyticsWindow` (all in `pkg/cosmoflare/analytics.go`).
- Produces: `type ZoneRef struct{ ID, Name string }`, `type ZoneCacheSummary struct{ ZoneID, Zone string; ByStatus map[string]uint64 }`, `func (z ZoneCacheSummary) MissPct(floor uint64) (float64, bool)`, `func (z ZoneCacheSummary) UncachedPct(floor uint64) (float64, bool)`, `func (s *AnalyticsService) ZoneCache(ctx context.Context, zones []ZoneRef, w AnalyticsWindow) ([]ZoneCacheSummary, error)`.

- [ ] **Step 1: Write the failing tests** — `pkg/cosmoflare/analytics_cache_test.go`:

```go
package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// churchesLive is the live churches.app breakdown captured 2026-10-08 (24h, eyeball).
func churchesLive() ZoneCacheSummary {
	return ZoneCacheSummary{ZoneID: "z1", Zone: "churches.app", ByStatus: map[string]uint64{
		"miss": 83, "bypass": 21, "dynamic": 939, "revalidated": 236, "expired": 4, "none": 197, "hit": 61,
	}}
}

func TestZoneCacheSummaryMissPct(t *testing.T) {
	t.Parallel()
	got, ok := churchesLive().MissPct(100)
	// eligible = hit 61 + miss 83 + expired 4 + revalidated 236 = 384; missed = 87
	if !ok || math.Abs(got-100*87.0/384.0) > 1e-9 {
		t.Fatalf("MissPct = %v, %v; want %v, true", got, ok, 100*87.0/384.0)
	}
	if _, ok := churchesLive().MissPct(385); ok {
		t.Error("MissPct must not be judgeable when eligible (384) < floor (385)")
	}
	if _, ok := (ZoneCacheSummary{ByStatus: map[string]uint64{"dynamic": 5000}}).MissPct(0); ok {
		t.Error("MissPct must not be judgeable with zero eligible requests")
	}
}

func TestZoneCacheSummaryUncachedPct(t *testing.T) {
	t.Parallel()
	got, ok := churchesLive().UncachedPct(100)
	// uncached = dynamic 939 + bypass 21 = 960; known = 384 + 960 = 1344; none excluded
	if !ok || math.Abs(got-100*960.0/1344.0) > 1e-9 {
		t.Fatalf("UncachedPct = %v, %v; want %v, true", got, ok, 100*960.0/1344.0)
	}
	if _, ok := churchesLive().UncachedPct(1345); ok {
		t.Error("UncachedPct must not be judgeable when known (1344) < floor (1345)")
	}
	only := ZoneCacheSummary{ByStatus: map[string]uint64{"none": 9000, "ignored": 5}}
	if _, ok := only.UncachedPct(0); ok {
		t.Error("none/ignored-only traffic must not be judgeable")
	}
}

// TestAnalyticsZoneCacheBatches: 12 zones → 2 requests (10 + 2), zoneTag_in
// variable, eyeball filter, cacheStatus grouping, results in input order,
// zones without rows returned with an empty map.
func TestAnalyticsZoneCacheBatches(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(data))
		mu.Unlock()
		var req struct {
			Variables struct {
				ZoneTags []string `json:"zoneTags"`
			} `json:"variables"`
		}
		_ = json.Unmarshal(data, &req)
		var zones []string
		for _, tag := range req.Variables.ZoneTags {
			if tag == "zone-03" { // one zone with no traffic rows at all
				continue
			}
			zones = append(zones, fmt.Sprintf(`{"zoneTag":%q,"httpRequestsAdaptiveGroups":[`+
				`{"count":7,"dimensions":{"cacheStatus":"hit"}},`+
				`{"count":3,"dimensions":{"cacheStatus":"miss"}},`+
				`{"count":2,"dimensions":{"cacheStatus":"hit"}}]}`, tag))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"viewer":{"zones":[%s]}}}`, strings.Join(zones, ","))
	}))
	defer srv.Close()

	refs := make([]ZoneRef, 12)
	for i := range refs {
		refs[i] = ZoneRef{ID: fmt.Sprintf("zone-%02d", i+1), Name: fmt.Sprintf("site%02d.example", i+1)}
	}
	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.ZoneCache(context.Background(), refs, analyticsWindow())
	if err != nil {
		t.Fatalf("ZoneCache: %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want 2 (batches of 10)", len(bodies))
	}
	for _, b := range bodies {
		for _, want := range []string{"zoneTag_in", "cacheStatus", `requestSource: \"eyeball\"`, "zoneTag"} {
			if !strings.Contains(b, want) {
				t.Errorf("request body missing %s: %s", want, b)
			}
		}
	}
	if len(got) != 12 {
		t.Fatalf("summaries = %d, want 12", len(got))
	}
	if got[0].Zone != "site01.example" || got[0].ZoneID != "zone-01" {
		t.Errorf("got[0] = %+v, want input order with names", got[0])
	}
	if got[0].ByStatus["hit"] != 9 || got[0].ByStatus["miss"] != 3 {
		t.Errorf("got[0].ByStatus = %v, want hit 9 (7+2 summed) miss 3", got[0].ByStatus)
	}
	if got[2].ByStatus == nil || len(got[2].ByStatus) != 0 {
		t.Errorf("zone without rows: ByStatus = %v, want empty non-nil map", got[2].ByStatus)
	}
}

func TestAnalyticsZoneCacheValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewAnalyticsService("acct", "", WithAnalyticsBaseURL("http://unused")).ZoneCache(context.Background(), []ZoneRef{{ID: "z"}}, analyticsWindow()); err == nil {
		t.Error("missing token must error")
	}
	got, err := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL("http://unused")).ZoneCache(context.Background(), nil, analyticsWindow())
	if err != nil || len(got) != 0 {
		t.Errorf("no zones = %v, %v; want empty, nil (no request)", got, err)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./pkg/cosmoflare/ -run 'ZoneCache' -v` → compile failure (`ZoneCacheSummary` undefined).

- [ ] **Step 3: Implement** — `pkg/cosmoflare/analytics_cache.go`:

```go
package cosmoflare

import (
	"context"
	"time"
)

// ZoneRef identifies a zone for batched zone analytics (FEAT-049).
type ZoneRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZoneCacheSummary counts one zone's eyeball requests per cacheStatus over a
// window (FEAT-049 wave 3a). Every status the API returns is kept, including
// statuses no ratio uses (none, ignored, …), so JSON output stays complete.
type ZoneCacheSummary struct {
	ZoneID   string            `json:"zone_id"`
	Zone     string            `json:"zone"`
	ByStatus map[string]uint64 `json:"by_status"`
}

// Cache status sets — design record D3, D4, D5, D13.
var (
	cacheEligibleStatuses = []string{"hit", "miss", "expired", "revalidated", "updating", "stale"}
	cacheMissStatuses     = []string{"miss", "expired"}
	cacheUncachedStatuses = []string{"dynamic", "bypass"}
)

// zoneCacheBatchSize is the documented cap on zones per zone-scoped query.
const zoneCacheBatchSize = 10

func (z ZoneCacheSummary) total(statuses []string) uint64 {
	var n uint64
	for _, st := range statuses {
		n += z.ByStatus[st]
	}
	return n
}

// MissPct returns (miss+expired) ÷ cache-eligible requests × 100. ok is false
// when the eligible count is zero or below floor (D6).
func (z ZoneCacheSummary) MissPct(floor uint64) (float64, bool) {
	eligible := z.total(cacheEligibleStatuses)
	if eligible == 0 || eligible < floor {
		return 0, false
	}
	return 100 * float64(z.total(cacheMissStatuses)) / float64(eligible), true
}

// UncachedPct returns (dynamic+bypass) ÷ known-status requests × 100, where
// known = eligible + dynamic + bypass. ok is false when known is zero or
// below floor (D4, D6).
func (z ZoneCacheSummary) UncachedPct(floor uint64) (float64, bool) {
	uncached := z.total(cacheUncachedStatuses)
	known := z.total(cacheEligibleStatuses) + uncached
	if known == 0 || known < floor {
		return 0, false
	}
	return 100 * float64(uncached) / float64(known), true
}

// ZoneCache returns per-zone eyeball request counts grouped by cacheStatus
// over the window. Zones are queried in batches of 10 with zoneTag_in — one
// request per batch, never one per zone. A zone with no traffic comes back
// with an empty ByStatus map. Results follow the input order.
func (s *AnalyticsService) ZoneCache(ctx context.Context, zones []ZoneRef, w AnalyticsWindow) ([]ZoneCacheSummary, error) {
	const op = "AnalyticsZoneCache"
	if s.apiToken == "" {
		return nil, validationError(op, "API token is required")
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($zoneTags: [String!], $start: Time!, $end: Time!) {
  viewer {
    zones(filter: {zoneTag_in: $zoneTags}) {
      zoneTag
      httpRequestsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end, requestSource: "eyeball"}) {
        count
        dimensions { cacheStatus }
      }
    }
  }
}`
	out := make([]ZoneCacheSummary, 0, len(zones))
	for i := 0; i < len(zones); i += zoneCacheBatchSize {
		end := i + zoneCacheBatchSize
		if end > len(zones) {
			end = len(zones)
		}
		batch := zones[i:end]
		tags := make([]string, len(batch))
		for j, z := range batch {
			tags[j] = z.ID
		}
		var resp struct {
			Viewer struct {
				Zones []struct {
					ZoneTag string `json:"zoneTag"`
					Groups  []struct {
						Count      uint64 `json:"count"`
						Dimensions struct {
							CacheStatus string `json:"cacheStatus"`
						} `json:"dimensions"`
					} `json:"httpRequestsAdaptiveGroups"`
				} `json:"zones"`
			} `json:"viewer"`
		}
		vars := map[string]any{
			"zoneTags": tags,
			"start":    w.Start.Format(time.RFC3339),
			"end":      w.End.Format(time.RFC3339),
		}
		if err := s.query(ctx, op, gql, vars, &resp); err != nil {
			return nil, err
		}
		byID := make(map[string]map[string]uint64, len(resp.Viewer.Zones))
		for _, z := range resp.Viewer.Zones {
			m := byID[z.ZoneTag]
			if m == nil {
				m = make(map[string]uint64)
				byID[z.ZoneTag] = m
			}
			for _, g := range z.Groups {
				m[g.Dimensions.CacheStatus] += g.Count
			}
		}
		for _, z := range batch {
			m := byID[z.ID]
			if m == nil {
				m = make(map[string]uint64)
			}
			out = append(out, ZoneCacheSummary{ZoneID: z.ID, Zone: z.Name, ByStatus: m})
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS** — `go test ./pkg/cosmoflare/ -run 'ZoneCache' -v`, then `go vet ./pkg/cosmoflare/` and `go test ./pkg/cosmoflare/` (whole package).
- [ ] **Step 5: Commit** — `feat(analytics): batched zone cacheStatus query + miss/uncached ratios (FEAT-049)`

### Task 2: D1RowsRead account-wide query (GLM)

**Files:**
- Create: `pkg/cosmoflare/analytics_d1rows.go`
- Create: `pkg/cosmoflare/analytics_d1rows_test.go`

**Interfaces:**
- Consumes: `AnalyticsService.query`, `validateAccount`, `validateAnalyticsWindow`, `windowVars` (`pkg/cosmoflare/analytics.go`).
- Produces: `type D1RowsReadSummary struct{ DatabaseID, Name string; RowsRead, RowsWritten, ReadQueries uint64 }`, `func (s *AnalyticsService) D1RowsRead(ctx context.Context, w AnalyticsWindow) ([]D1RowsReadSummary, error)` — sorted by RowsRead descending, then DatabaseID; Name left empty (callers fill it from the D1 list).

- [ ] **Step 1: Write the failing tests** — `pkg/cosmoflare/analytics_d1rows_test.go`:

```go
package cosmoflare

import (
	"context"
	"strings"
	"testing"
)

func TestAnalyticsD1RowsRead(t *testing.T) {
	t.Parallel()
	var cap analyticsCapture
	resp := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
		`{"sum":{"rowsRead":1000,"rowsWritten":5,"readQueries":10},"dimensions":{"databaseId":"db-small"}},` +
		`{"sum":{"rowsRead":2945546702,"rowsWritten":7,"readQueries":73151},"dimensions":{"databaseId":"db-big"}},` +
		`{"sum":{"rowsRead":500,"rowsWritten":1,"readQueries":2},"dimensions":{"databaseId":"db-small"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &cap)
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.D1RowsRead(context.Background(), analyticsWindow())
	if err != nil {
		t.Fatalf("D1RowsRead: %v", err)
	}
	for _, want := range []string{"d1AnalyticsAdaptiveGroups", "datetime_geq", "rowsRead", "databaseId", "accountTag"} {
		if !strings.Contains(cap.Body, want) {
			t.Errorf("body missing %s: %s", want, cap.Body)
		}
	}
	if strings.Contains(cap.Body, "databaseId:") {
		t.Errorf("query must not filter by databaseId (account-wide fan-out): %s", cap.Body)
	}
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2 (db-small rows summed)", len(got))
	}
	if got[0].DatabaseID != "db-big" || got[0].RowsRead != 2945546702 || got[0].ReadQueries != 73151 {
		t.Errorf("got[0] = %+v, want db-big first (sorted by rows read desc)", got[0])
	}
	if got[1].DatabaseID != "db-small" || got[1].RowsRead != 1500 || got[1].RowsWritten != 6 || got[1].ReadQueries != 12 {
		t.Errorf("got[1] = %+v, want db-small summed 1500/6/12", got[1])
	}
	if got[0].Name != "" {
		t.Errorf("Name = %q, want empty (caller maps names)", got[0].Name)
	}
}

func TestAnalyticsD1RowsReadValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewAnalyticsService("", "tok").D1RowsRead(context.Background(), analyticsWindow()); err == nil {
		t.Error("missing account ID must error")
	}
	w := analyticsWindow()
	w.Start, w.End = w.End, w.Start
	if _, err := NewAnalyticsService("acct", "tok").D1RowsRead(context.Background(), w); err == nil {
		t.Error("inverted window must error")
	}
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./pkg/cosmoflare/ -run 'D1RowsRead' -v` → compile failure.

- [ ] **Step 3: Implement** — `pkg/cosmoflare/analytics_d1rows.go`:

```go
package cosmoflare

import (
	"context"
	"sort"
	"time"
)

// D1RowsReadSummary is one database's D1 query load over a window (FEAT-049).
// D1 bills rows read (scanned), so RowsRead is the waste signal.
type D1RowsReadSummary struct {
	DatabaseID  string `json:"database_id"`
	Name        string `json:"name,omitempty"` // filled by callers from the D1 list; empty when unknown
	RowsRead    uint64 `json:"rows_read"`
	RowsWritten uint64 `json:"rows_written"`
	ReadQueries uint64 `json:"read_queries"`
}

// D1RowsRead returns per-database rows read/written over the window in one
// account-wide query grouped by databaseId (no per-database N+1). Rows are
// sorted by RowsRead descending, then DatabaseID.
func (s *AnalyticsService) D1RowsRead(ctx context.Context, w AnalyticsWindow) ([]D1RowsReadSummary, error) {
	const op = "AnalyticsD1RowsRead"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      d1AnalyticsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        sum { rowsRead rowsWritten readQueries }
        dimensions { databaseId }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				Groups []struct {
					Sum struct {
						RowsRead    uint64 `json:"rowsRead"`
						RowsWritten uint64 `json:"rowsWritten"`
						ReadQueries uint64 `json:"readQueries"`
					} `json:"sum"`
					Dimensions struct {
						DatabaseID string `json:"databaseId"`
					} `json:"dimensions"`
				} `json:"d1AnalyticsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	byID := make(map[string]*D1RowsReadSummary)
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.Groups {
			row := byID[g.Dimensions.DatabaseID]
			if row == nil {
				row = &D1RowsReadSummary{DatabaseID: g.Dimensions.DatabaseID}
				byID[g.Dimensions.DatabaseID] = row
			}
			row.RowsRead += g.Sum.RowsRead
			row.RowsWritten += g.Sum.RowsWritten
			row.ReadQueries += g.Sum.ReadQueries
		}
	}
	rows := make([]D1RowsReadSummary, 0, len(byID))
	for _, r := range byID {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].RowsRead != rows[j].RowsRead {
			return rows[i].RowsRead > rows[j].RowsRead
		}
		return rows[i].DatabaseID < rows[j].DatabaseID
	})
	return rows, nil
}
```

- [ ] **Step 4: Run, expect PASS** — `go test ./pkg/cosmoflare/ -run 'D1RowsRead' -v`, then `go vet ./pkg/cosmoflare/` and `go test ./pkg/cosmoflare/`.
- [ ] **Step 5: Commit** — `feat(analytics): account-wide D1 rows-read per database (FEAT-049)`

### Task 3: Rule exclude list + zone/d1 services (GLM)

**Files:**
- Modify: `pkg/cosmoflare/alerts.go` (AlertRule, AlertRuleUpdate, validAlertServices, two error messages, Update)
- Modify: `cmd/alerts.go` (create/update flags, rule literal, update patch)
- Test: `pkg/cosmoflare/alerts_test.go` (append), `cmd/alerts_test.go` (append)

**Interfaces:**
- Produces: `AlertRule.Exclude []string` (yaml/json `exclude,omitempty`), `AlertRuleUpdate.Exclude *[]string`, `func (r *AlertRule) Excludes(scopeID string) bool` (case-insensitive exact match; nil-safe), services `zone` and `d1` valid.

- [ ] **Step 1: Failing tests** — append to `pkg/cosmoflare/alerts_test.go`:

```go
func TestAlertRuleExcludes(t *testing.T) {
	t.Parallel()
	r := &AlertRule{Exclude: []string{"api.churches.app", " Auth-DB"}}
	if !r.Excludes("api.churches.app") || !r.Excludes("auth-db") {
		t.Error("Excludes must match listed names case-insensitively, ignoring surrounding spaces (pflag keeps the space in --exclude \"a, b\")")
	}
	if r.Excludes("churches.app") || r.Excludes("") {
		t.Error("Excludes must be exact (no substring, no empty match)")
	}
	var nilRule *AlertRule
	if nilRule.Excludes("x") {
		t.Error("nil rule excludes nothing")
	}
}

func TestAlertServiceZoneD1ServicesAndExclude(t *testing.T) {
	t.Parallel()
	svc := newTestAlertService(t) // existing helper, alerts_test.go:11
	for _, service := range []string{"zone", "d1"} {
		_, err := svc.Create(&AlertRule{Name: "r-" + service, Service: service, Condition: "error-rate", Threshold: 1, Action: "log", Target: "x", Exclude: []string{"a.example"}})
		if err != nil {
			t.Fatalf("Create service %q: %v", service, err)
		}
	}
	got, err := svc.Get("r-d1")
	if err != nil || len(got.Exclude) != 1 || got.Exclude[0] != "a.example" {
		t.Fatalf("Get r-d1 = %+v, %v; want Exclude [a.example] persisted", got, err)
	}
	ex := []string{"b.example", "c.example"}
	upd, err := svc.Update("r-d1", &AlertRuleUpdate{Exclude: &ex})
	if err != nil || len(upd.Exclude) != 2 {
		t.Fatalf("Update Exclude = %+v, %v", upd, err)
	}
	empty := []string{}
	upd, err = svc.Update("r-d1", &AlertRuleUpdate{Exclude: &empty})
	if err != nil || len(upd.Exclude) != 0 {
		t.Fatalf("Update to empty Exclude = %+v, %v; want cleared", upd, err)
	}
}
```

In `cmd/alerts_test.go`, `setupAlertTestEnv` resets every flag variable (lines 28-36): add `alertExclude = nil` there, beside `alertSince = ""`.

Append to `cmd/alerts_test.go` one test that runs `alerts create d1-hot --service d1 --condition error-rate --threshold 1 --action log --target x --exclude a.example,b.example --json` through the existing cmd test harness in that file and asserts the stored rule has `Exclude == [a.example b.example]`; then `alerts update d1-hot --exclude c.example` and asserts `[c.example]`. Follow the exact helper pattern (temp alert service via `getAlertServiceFn`, flag reset) the existing create/update tests in `cmd/alerts_test.go` use.

- [ ] **Step 2: Run, expect FAIL** — `go test ./pkg/cosmoflare/ -run 'Excludes|ZoneD1Services' -v` and `go test ./cmd/ -run 'Alerts' -v`.

- [ ] **Step 3: Implement**
  - `AlertRule`: add after `Enabled`:
    ```go
    	Exclude   []string  `json:"exclude,omitempty" yaml:"exclude,omitempty"` // scope instances (zone or database names) zone/d1 rules skip (FEAT-049)
    ```
  - `AlertRuleUpdate`: add `Exclude *[]string \`json:"exclude,omitempty"\``; in `AlertService.Update` next to the other field patches: `if update.Exclude != nil { found.Exclude = append([]string(nil), (*update.Exclude)...) }`.
  - Method (place after `AlertRule`):
    ```go
    // Excludes reports whether scopeID is on the rule's exclude list
    // (case-insensitive exact match, surrounding spaces ignored). A nil rule
    // excludes nothing.
    func (r *AlertRule) Excludes(scopeID string) bool {
    	scopeID = strings.TrimSpace(scopeID)
    	if r == nil || scopeID == "" {
    		return false
    	}
    	for _, e := range r.Exclude {
    		if strings.EqualFold(strings.TrimSpace(e), scopeID) {
    			return true
    		}
    	}
    	return false
    }
    ```
  - `validAlertServices`: add `"zone": true, "d1": true`. Both error strings `must be one of: r2, workers, kv, dns` → `must be one of: r2, workers, kv, dns, zone, d1`.
  - `cmd/alerts.go`: new var `alertExclude []string`; on both `alertsCreateCmd` and `alertsUpdateCmd`: `Flags().StringSliceVar(&alertExclude, "exclude", nil, "Zone or database names a zone/d1 rule skips (comma-separated)")`; service flag help `(r2, workers, kv, dns)` → `(r2, workers, kv, dns, zone, d1)` on both; in `runAlertsCreate` set `Exclude: alertExclude` on the rule literal; in `runAlertsUpdate` add `if cmd.Flags().Changed("exclude") { ex := alertExclude; update.Exclude = &ex }` beside the other `Changed` checks (`--exclude ""` therefore clears the list — say so in the update command's flag help: `Zone or database names a zone/d1 rule skips (comma-separated; --exclude "" clears)`); in the `alerts get` text output (`cmd/alerts.go` around lines 385-397) print an `Exclude:` line with `strings.Join(rule.Exclude, ", ")` when the list is non-empty. Add one help example line to the create command's Long text: `cosmoflare alerts create d1-scan --service d1 --condition d1-rows-read --threshold 1e9 --action log --target - --exclude analytics-db`.
- [ ] **Step 4: PASS** — the two runs above, then `go test ./pkg/cosmoflare/ ./cmd/` and `go vet ./...`.
- [ ] **Step 5: Commit** — `feat(alerts): per-rule exclude list + zone/d1 rule services (FEAT-049)`

Note: the help example names `d1-rows-read`, which T4 registers; the example is text only and does not run.

### Task 4: Evaluator — zone/d1 scopes, FireState, cooldown policy, escalation, gaps (Opus)

**Files:**
- Modify: `pkg/cosmoflare/alerts.go` (3 registry rows; `Scope` comment adds `zone`, `d1`)
- Modify: `internal/webhook/evaluator.go`
- Create: `internal/webhook/telemetry.go` (CollectTelemetryMetrics), `internal/webhook/telemetry_test.go`
- Test: `internal/webhook/evaluator_test.go` (append; extend the coverage fixture)

**Interfaces:**
- Consumes: T1 `ZoneCacheSummary.MissPct/UncachedPct`, `ZoneRef`, `ZoneCache`; T2 `D1RowsReadSummary`, `D1RowsRead`; T3 `AlertRule.Excludes`.
- Produces:
  - Registry rows `zone-cache-miss-pct` (zone, %), `zone-uncached-pct` (zone, %), `d1-rows-read` (d1, rows).
  - `EvalMetrics.Zones []cosmoflare.ZoneCacheSummary`, `EvalMetrics.D1 []cosmoflare.D1RowsReadSummary`, `EvalMetrics.Gaps map[string]string` (scope → error text).
  - `const ZoneCacheFloor = 100`.
  - `type FireState struct` (unexported maps: lastFired, lastValue, fires — in memory only), `func NewFireState() *FireState`, `func (e *Evaluator) UseState(st *FireState)`, `func (e *Evaluator) SetCooldownPolicy(def time.Duration, perScope map[string]time.Duration)` — `def` replaces the evaluator default for scopes without an entry (0 = no cooldown); the watch passes `def = 0` so only zone/d1 opt in and any future scope keeps re-paging every cycle.
  - Escalation: ONLY for scopes with an explicit `perScope` entry > 0 (so `cosmoflare serve`, which sets no policy, keeps its exact 15-min behavior). Inside such a cooldown a fire still happens when the last paged value > 0 and the value ≥ 2× it; message starts `escalated (≥2× last page): `. Effectively d1-only: a % value above 50 cannot double (O4).
  - Gap pages: for each `scope → err` in `m.Gaps` where at least one enabled rule has a condition of that scope: fire alert ID `telemetry-gap/<scope>`, Name `telemetry-gap`, 1h cooldown via the same FireState (`telemetryGapCooldown = time.Hour`), message `telemetry gap: <scope> analytics unavailable, <scope> rules cannot fire: <err>`.
  - Row-count formatting: unit `rows` prints `2.9B` / `52.3M` / `3.3k` style in messages (`humanCount`); other units keep `%g`.
  - `type TelemetryRefs struct { Zones []cosmoflare.ZoneRef; ZonesErr error; DBNames map[string]string; DBNamesErr error }` and `func CollectTelemetryMetrics(ctx context.Context, analytics *cosmoflare.AnalyticsService, wantZones, wantD1 bool, refs TelemetryRefs, w cosmoflare.AnalyticsWindow, m *EvalMetrics)`. Explicit want-flags, never nil sentinels. Every failure lands in `m.Gaps`, never silently skipped: `wantZones && refs.ZonesErr != nil` → `Gaps["zone"]` (zone list failed); ZoneCache error → `Gaps["zone"]`; D1RowsRead error → `Gaps["d1"]`; `refs.DBNamesErr != nil` does NOT skip D1 — rows keep the database ID as Name (D8 fallback). Errors are never returned. A ZoneCache failure in any batch drops the whole zone dataset for that cycle (one gap page covers it).

- [ ] **Step 1: Failing tests** (evaluator_test.go):
  - coverage fixture gains `Zones: []cosmoflare.ZoneCacheSummary{churches-like row}` and `D1: []cosmoflare.D1RowsReadSummary{{DatabaseID: "db1", Name: "mycarguide-db", RowsRead: 2945546702}}` so the registry coverage test fails until branches exist.
  - `TestEvaluateZoneUncachedNamesZone`: one zone at 71.4% uncached, threshold 60 → fires, Alert.ID `rule/churches.app`, message names the zone.
  - `TestEvaluateZoneFloorSkips`: zone with 50 known requests → no fire at threshold 1.
  - `TestEvaluateExcludeSkipsScope`: rule with `Exclude: ["churches.app"]` → no fire for that zone, fires for another.
  - `TestEvaluateD1HumanReadable`: d1 rule threshold 1e9 fires naming `mycarguide-db`, message contains `2.9B`.
  - `TestEvaluateScopeCooldownPersists`: shared FireState, `SetCooldownPolicy(0, {zone: 1h, d1: 1h})`, two fresh evaluators sharing the state, Evaluate 1 min apart (SetClock): zone rule pages once, script rule pages twice.
  - `TestUsageMaxSkipsD1` (O16): dims `workers.requests_monthly` Pct 40 and `d1.rows_read_monthly` Pct 350 → `usageMax` = 40; `usage-pct` threshold 100 does not fire.
  - `TestEvaluateNoPolicyNoEscalation`: evaluator without a policy (serve's path), 15-min default cooldown, value doubles inside it → no second page (serve unchanged).
  - `TestCollectTelemetryRefsFailureIsAGap`: `wantZones` with `refs.ZonesErr` set → `Gaps["zone"]` recorded, no zone request; `refs.DBNamesErr` set → D1 rows still collected, Name = database ID.
  - `TestEvaluateEscalationDoubling`: d1 value 1.0e9 then 1.5e9 (no page) then 2.1e9 (page, message `escalated`) inside the hour.
  - `TestEvaluateTelemetryGapPagesOnce`: `Gaps{"d1": "boom"}` with an enabled d1 rule → one `telemetry-gap/d1` page; second Evaluate within 1h → none; with no d1 rule enabled → none.
  - telemetry_test.go: `CollectTelemetryMetrics` against a mock server returning a GraphQL error for D1 and valid zone rows → `m.Zones` populated, `m.Gaps["d1"]` set, no panic; nil zones → no zone request.
- [ ] **Step 2: FAIL run** — `go test ./internal/webhook/ -v`.
- [ ] **Step 3: Implement** as specified in Interfaces. `conditionValues` gains `case "zone"` and `case "d1"` branches (skip via `continue` when not judgeable, constraint 2); the exclude check lives in `Evaluate` (`if rule.Excludes(sv.ScopeID) { continue }`), so `conditionValues` stays rule-agnostic. The Evaluate cooldown uses `scopeCooldown(desc.Scope)`; state reads/writes go through `e.state`.
- [ ] **Step 4: PASS** — `go test ./internal/webhook/ ./pkg/cosmoflare/ ./cmd/`, `go vet ./...`.
- [ ] **Step 5: Commit** — `feat(alerts): zone cache + D1 rows-read conditions with per-scope cooldown, escalation, gap pages (FEAT-049)`

### Task 5: Watch + check wiring (Opus, same worktree as T4)

**Files:**
- Modify: `cmd/alerts_watch.go`, `cmd/alerts.go` (runAlertsCheck)
- Test: `cmd/alerts_watch_test.go` (append)

**Interfaces:**
- Consumes: T4 `CollectTelemetryMetrics`, `NewFireState`, `UseState`, `SetCooldownPolicy`; `ZoneService.List`, `D1Service.List`.
- Produces: package vars `watchFireState = webhook.NewFireState()` (in memory: a watch restart pages every current zone/d1 breach once), `watchScopeCooldowns = map[string]time.Duration{"zone": time.Hour, "d1": time.Hour}` passed as `SetCooldownPolicy(0, watchScopeCooldowns)`; `telemetryRefsFn func(ctx) webhook.TelemetryRefs` seam with a 15-min keep-last-good cache per list (`cachedTelemetryRefs`), active zones only (`Status == "active"`); `collectWatchMetricsFn` signature becomes `func(ctx context.Context, rules []*cosmoflare.AlertRule) (webhook.EvalMetrics, error)` (update the 3 existing overrides in `cmd/alerts_watch_test.go`, lines ~63, 71, 213); `rulesNeedScope(rules, scope) bool` over enabled rules.

- [ ] **Step 1: Failing tests** — a `resetWatchState(t)` helper (fresh `watchFireState`, cleared refs cache, restored seams via `t.Cleanup`) used by every new watch test. Two-cycle watch test with canned metrics (zone row over threshold) via `collectWatchMetricsFn`: the pager receives one zone page across two `evaluateOnce` calls, while a script rule over threshold pages twice; refs fetcher not called when no zone/d1 rule is enabled; refs fetch error with a zone rule enabled → one `telemetry-gap/zone` page; help text mentions zone cache and D1.
- [ ] **Step 2: FAIL.** **Step 3:** wire `evaluateOnce` (`eval.UseState(watchFireState)`, `eval.SetCooldownPolicy(0, watchScopeCooldowns)`, pass `rules` to `collectWatchMetricsFn`), `collectWatchMetricsFn` (call `CollectTelemetryMetrics` with cached refs and want-flags from `rulesNeedScope`), `runAlertsCheck` (same collection, uncached refs, one-shot evaluator unchanged), Long help: datasets list + "zone and d1 rules page at most once an hour per zone/database unless the value doubles; a recovery and re-breach inside the hour stays quiet; a telemetry gap pages once an hour; cooldown memory is in-process, so a restart re-pages current breaches once". `cosmoflare serve` (O17): at alert-cycle start, if any enabled rule has a zone/d1 condition, log once: `[alerts] zone/d1 rules are evaluated by 'alerts watch' and 'alerts check' only; serve skips them`; test it via the serve alert-cycle test seam.
- [ ] **Step 4: PASS** — `go test ./cmd/ ./internal/webhook/`, `go build ./... && go vet ./...`.
- [ ] **Step 5: Commit** — `feat(watch): zone/D1 telemetry collection, persistent zone/d1 cooldown (FEAT-049)`

### Task 6: D1 monthly rows-read in the usage view (GLM)

**Files:**
- Modify: `pkg/cosmoflare/limitsdata/catalog.json` (append one row next to the other monthly rows)
- Modify: `pkg/cosmoflare/usage.go` (CollectUsage)
- Test: `pkg/cosmoflare/limitsdata/limitsdata_test.go`, `pkg/cosmoflare/usage_test.go` (append)

**Interfaces:**
- Consumes: T2 `AnalyticsService.D1RowsRead`.
- Produces: catalog row `d1.rows_read_monthly`; `CollectUsage` adds that dimension when D1 analytics succeed and omits it (no error) when they fail.

- [ ] **Step 1: Failing tests**
  - `pkg/cosmoflare/limitsdata/limitsdata_test.go`:
    - In `TestMonthlyUsageRowsPresent` (line 117) add `"d1.rows_read_monthly": 25_000_000_000,` to the `expected` map.
    - Append:
      ```go
      func TestD1RowsReadMonthlyFreeTierNull(t *testing.T) {
      	e, ok := Lookup("d1.rows_read_monthly")
      	if !ok {
      		t.Fatal("catalog row d1.rows_read_monthly missing")
      	}
      	if e.Tiers.Free != nil {
      		t.Errorf("free tier = %v, want null (Free is a daily 5M allowance, not monthly)", *e.Tiers.Free)
      	}
      }
      ```
      (Use the field names `TestMonthlyUsageRowsPresent` uses to read tiers; if `Tiers.Free` is spelled differently there, copy that spelling.)
    - In `TestLoadEntryCount` (line ~13) change `75` → `76` in both places and the comment to `// 74 embedded corpus entries + 2 appended local constants.` and the message to `(74 catalog + 2 local)`.
  - `pkg/cosmoflare/usage_test.go` — in `usageMockServer` add a D1 response and route it BEFORE the `default:` case (the default returns R2 storage for any unmatched body, which would make a missing D1 route pass silently with zero rows):
    ```go
    	d1Resp := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
    		`{"sum":{"rowsRead":1000000000,"rowsWritten":0,"readQueries":10},"dimensions":{"databaseId":"db-a"}},` +
    		`{"sum":{"rowsRead":500000000,"rowsWritten":0,"readQueries":5},"dimensions":{"databaseId":"db-b"}}` +
    		`]}]}}}`
    ```
    ```go
    		case strings.Contains(q, "d1AnalyticsAdaptiveGroups"):
    			w.Write([]byte(d1Resp))
    ```
    Append `TestCollectUsageD1Dimension` asserting the snapshot has dimension `d1.rows_read_monthly` with `Used == 1.5e9` and `Limit == 2.5e10`, and `TestCollectUsageD1FailureIsAdditive` using a copy of the mock whose D1 branch writes `{"errors":[{"message":"boom"}]}`: `CollectUsage` returns nil error, `workers.requests_monthly` is present, `d1.rows_read_monthly` is absent. If any existing usage test asserts the dimension count, update it for the extra D1 dimension.
- [ ] **Step 2: FAIL.**
- [ ] **Step 3: Implement** — catalog row (exact JSON):

```json
  {
   "id": "d1.rows_read_monthly",
   "service": "d1",
   "kind": "quota",
   "unit": "rows_per_month",
   "name": "Monthly D1 rows-read allowance",
   "scope": "account",
   "tiers": {
    "free": null,
    "paid": 25000000000,
    "enterprise": null
   },
   "trackable": true,
   "enforceability": "opaque",
   "soft": true,
   "source_url": "https://developers.cloudflare.com/d1/platform/pricing/",
   "verified_on": "2026-10-08T23:50:00Z",
   "notes": "Paid: 'First 25 billion / month included + $0.001 / million rows' (page last updated Apr 21, 2026). Free: '5 million / day' — daily, no monthly aggregate, so free stays null."
  }
```

  In `CollectUsage`, after the R2 operations call:

```go
	// D1 is additive (design D15): a D1 analytics failure must not drop the
	// Workers/DO/R2 pacing rows, so the dimension is simply omitted.
	var d1Rows uint64
	d1Summaries, d1Err := analytics.D1RowsRead(ctx, cycleWin)
	for _, r := range d1Summaries {
		d1Rows += r.RowsRead
	}
```

  and after the `used` map literal:

```go
	if d1Err == nil {
		used["d1.rows_read_monthly"] = float64(d1Rows)
	}
```
- [ ] **Step 4: PASS** — `go test ./pkg/cosmoflare/... ./cmd/`, `go vet ./...`.
- [ ] **Step 5: Commit** — `feat(usage): D1 monthly rows-read pacing dimension, verified catalog row (FEAT-049)`

### Task 7: Opt-in live smoke test (GLM)

**Files:**
- Create: `pkg/cosmoflare/analytics_live_test.go`
- Modify: `Makefile` (add `test-live` target) if a Makefile exists at the repo root

**Interfaces:**
- Consumes: `ZoneHTTP`, `ZoneCache`, `D1RowsRead`, `Workers`, `NewZoneServiceFromCreds(...).List`.

- [ ] **Step 1: Write the test** (this task is the test):

```go
//go:build live

package cosmoflare

import (
	"context"
	"os"
	"testing"
	"time"
)

// liveEnv reads the repo's credential names, falling back to the CF_* names
// that `ccs credentials source` exports.
func liveEnv(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

// TestLiveAnalyticsQueries runs every analytics query cosmoflare's alerts
// depend on against the real Cloudflare GraphQL API, with the SAME windows
// production uses: 24h for the watch, the billing cycle for usage. Mocked
// tests cannot catch schema drift or window limits (BUG-055 shipped green).
// Opt-in only:
//
//	CLOUDFLARE_ACCOUNT_ID=… CLOUDFLARE_API_TOKEN=… go test -tags live ./pkg/cosmoflare/ -run Live -v
func TestLiveAnalyticsQueries(t *testing.T) {
	acct := liveEnv("CLOUDFLARE_ACCOUNT_ID", "CF_ACCOUNT_ID")
	tok := liveEnv("CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")
	if acct == "" || tok == "" {
		t.Skip("CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN (or CF_*) are required for live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	now := time.Now().UTC()
	w := AnalyticsWindow{Start: now.Add(-24 * time.Hour), End: now} // watch window
	cycleStart, _ := CycleWindow(0, now)
	cycle := AnalyticsWindow{Start: cycleStart, End: now} // usage window
	a := NewAnalyticsService(acct, tok)

	if _, err := a.Workers(ctx, w); err != nil {
		t.Errorf("Workers (24h): %v", err)
	}
	if _, err := a.D1RowsRead(ctx, w); err != nil {
		t.Errorf("D1RowsRead (24h): %v", err)
	}
	if _, err := a.D1RowsRead(ctx, cycle); err != nil {
		t.Errorf("D1RowsRead (billing cycle %s → now): %v", cycleStart.Format(time.RFC3339), err)
	}
	if _, err := CollectUsage(ctx, a, "paid", 0, now); err != nil {
		t.Errorf("CollectUsage: %v", err)
	}

	zs, err := NewZoneServiceFromCreds(acct, tok)
	if err != nil {
		t.Fatalf("zone service: %v", err)
	}
	zones, err := zs.List(ctx)
	if err != nil {
		t.Fatalf("zone list: %v", err)
	}
	if len(zones) == 0 {
		t.Skip("account has no zones; zone queries not exercised")
	}
	// One zone per distinct plan first (plan-specific field limits are the
	// risk), then fill to a full batch of 10.
	refs := make([]ZoneRef, 0, 10)
	seenPlan := map[string]bool{}
	picked := map[string]bool{}
	for _, z := range zones {
		if !seenPlan[z.Plan.Name] {
			seenPlan[z.Plan.Name] = true
			picked[z.ID] = true
			refs = append(refs, ZoneRef{ID: z.ID, Name: z.Name})
		}
	}
	for _, z := range zones {
		if len(refs) >= 10 {
			break
		}
		if !picked[z.ID] {
			refs = append(refs, ZoneRef{ID: z.ID, Name: z.Name})
		}
	}
	if len(refs) > 10 {
		refs = refs[:10]
	}
	t.Logf("live zones: %d across %d plans", len(refs), len(seenPlan))
	if _, err := a.ZoneHTTP(ctx, refs[0].ID, w); err != nil {
		t.Errorf("ZoneHTTP: %v", err)
	}
	got, err := a.ZoneCache(ctx, refs, w)
	if err != nil {
		t.Errorf("ZoneCache: %v", err)
	} else if len(got) != len(refs) {
		t.Errorf("ZoneCache returned %d zones, want %d", len(got), len(refs))
	}
}
```

- [ ] **Step 2: Verify it compiles out of the default suite** — `go test ./pkg/cosmoflare/ -run Live -v` → "no tests to run"; `go vet -tags live ./pkg/cosmoflare/` exit 0.
- [ ] **Step 3: Makefile** — if `Makefile` exists, add:

```make
test-live: ## Live Cloudflare API smoke test (needs CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_API_TOKEN)
	go test -tags live ./pkg/cosmoflare/ -run Live -v -count=1
```

- [ ] **Step 4: Commit** — `test(analytics): opt-in live smoke test for every alert query (FEAT-049)`
- Opus runs it once live at the gate (credentials via `ccs credentials source`).

### Task 8: Documentation (GLM, two agents in parallel)

- [ ] **T8a `docs/USAGE.md`:** conditions table gains the three rows (name, scope, unit, help text from the registry verbatim); `alerts create/update` sections gain `--exclude` and services `zone`, `d1`; `alerts watch` section gains: datasets list, zone/d1 1h cooldown + 2× escalation, telemetry-gap pages, the 100-request floor, starter thresholds from O10, the rolling-24h vs Free UTC-day note (O11), and `make test-live`.
- [ ] **T8b `README.md`:** one alerting bullet: "zone cache-miss / uncached and D1 rows-read alerts that name the zone or database".
- [ ] Opus verifies both diffs against the registry and this plan; one commit each: `docs(usage): wave 3a cache + D1 conditions (FEAT-049)`, `docs(readme): cache + D1 alerting mention (FEAT-049)`.

### Task 9: Gate + close

- [ ] Per worktree: `git -C <wt> diff master..HEAD` read; `ccs worktree exec <wt> -- go test ./pkg/cosmoflare/... ./internal/... ./cmd/` exit 0; `ccs verify-worktree <name> --approve --score N --issues 0 && ccs merge <name>`; `git merge-base --is-ancestor <tip> master`.
- [ ] After all merges: `go build ./... && go vet ./...`; full affected suites on master; `make test-live` (or the go test line) once with live credentials.
- [ ] O18 dry run: with temporary rules at the O10 starter thresholds in a scratch rules file (never the operator's `.cosmoflare-alerts.yaml`), run `cosmoflare alerts check --json` live and report how many zones/databases would page. No pager dispatch.
- [ ] `ccs changelog add --type added FEAT-049`; `ccs issues update FEAT-049 --status implemented`; ROAD-101 note: wave 3a done.
- [ ] No push, no tag (O8).

## Self-Review

- **Spec coverage:** D1 (split) → scope of this plan; D2–D6, D12–D13 → T1 + T4; D7 → T1/T2 batching; D8 → T4/T5 names; D9 → excluded; D10 → T6; D11 → T4/T5; D14 → T4 humanCount; D15 → T4 gaps + T6 additive; D16 → T5 help. Code constraints 1–9 → T4 (1, 2, 3, 4), T1 (5), T5 (6), T4/T5 (7), T2 (8), T6 (9). Mitigations: gap page, exclude, escalation, live test → T4, T3, T4, T7.
- **Placeholders:** T4/T5 are Opus tasks described by exact interfaces and named tests; their code is written in the worktree under TDD, not pre-written here. All GLM tasks carry full code.
- **Type consistency:** `ZoneRef`, `ZoneCacheSummary`, `MissPct/UncachedPct(floor uint64)`, `D1RowsReadSummary`, `D1RowsRead`, `Excludes`, `FireState`, `UseState`, `SetCooldownPolicy`, `CollectTelemetryMetrics` are spelled identically in every task.
