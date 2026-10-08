---
title: FEAT-048 Wave 2 — Usage Pacing + `cosmoflare usage` Monthly View
created: "2026-10-08T21:20:00+04:00"
roadmap: ROAD-101
brainstorm_ref: docs/brainstorming/2026-10-08-feat047-usage-waste-alerting.md
status: PLANNED
deliverables:
    - P-01: Verified monthly usage rows in the limits catalog (limitsdata)
    - P-02: UsageService — cycle-to-date collection + pacing math (pkg/cosmoflare/usage.go)
    - P-03: `cosmoflare usage` command — table + --json + help (cmd/usage.go)
    - P-04: usage-pct + usage-projected-pct alert conditions wired into the watch
    - P-05: USAGE.md + README documentation (parallel doc agents)
---

# FEAT-048 Wave 2 — Usage Pacing + `cosmoflare usage`

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `cosmoflare usage` shows every monthly usage dimension against its plan limit with cycle pacing, and the watch daemon pages when any dimension is projected to overrun the cycle.

**Architecture:** All telemetry already flows from wave 1 — `Workers()`, `DurableObjects()`, `R2Storage()`, `R2Operations()` over an arbitrary window. Pacing is arithmetic: cycle-to-date usage (analytics over the cycle window) ÷ limit (limits catalog tier) with linear projection to cycle end. The billing-cycle anchor defaults to the calendar month; an explicit flag covers anniversary billing. A new `usage-pct`/`usage-projected-pct` condition pair (account scope, max-across-dimensions like `dns-record-quota`) rides the existing registry.

**Tech Stack:** Go 1.26, cobra, existing testify patterns, Cloudflare GraphQL Analytics API.

**Spec:** docs/brainstorming/2026-10-08-feat047-usage-waste-alerting.md (D3 dual cadence lands here; D6 usage view)

## Global Constraints

- Catalog discipline: every new row carries `source_url` + `verified_on` (today) + real tier values — **verify against live Cloudflare docs before adding** (delegate URL reads to subagents; values from memory are forbidden).
- Registry stays the single definition site (FEAT-015); coverage test must stay green.
- One batched GraphQL query per dataset per collection — never per-dimension N+1.
- `--json` + rich help on the new command; deterministic exit codes.
- Conventional commits, no AI attribution.

---

### Task 1: Verified monthly usage rows (limitsdata)

**Files:**
- Modify: `pkg/cosmoflare/limitsdata/catalog.json`
- Test: `pkg/cosmoflare/limitsdata/limitsdata_test.go` (append)

**Interfaces:**
- Produces catalog rows (ids): `workers.requests_monthly`, `workers.cpu_ms_monthly`, `do.requests_monthly`, `do.duration_ms_monthly`, `r2.storage_monthly_bytes`, `r2.class_a_monthly`, `r2.class_b_monthly` — each `{id, service, kind: "quota", unit, name, scope: "account", tiers: {free, paid, enterprise}, trackable: true, enforceability: "opaque", soft: true, source_url, verified_on}`.

- [ ] **Step 1: Verify values (spike — subagent, throwaway notes)**

Dispatch one web-reading subagent per docs page (rule: never WebFetch in main session):
- https://developers.cloudflare.com/workers/platform/pricing/ (requests/mo + CPU-ms/mo per plan)
- https://developers.cloudflare.com/durable-objects/platform/pricing/ (requests + duration/mo)
- https://developers.cloudflare.com/r2/pricing/ (storage TB-mo, Class A/B ops-mo)

Record free/paid values verbatim with the URL. Any dimension the docs do not state as a monthly allowance → do not add that row; note it in the commit body.

- [ ] **Step 2: Failing test**

```go
func TestMonthlyUsageRowsPresent(t *testing.T) {
	for _, id := range []string{
		"workers.requests_monthly", "workers.cpu_ms_monthly",
		"do.requests_monthly", "r2.storage_monthly_bytes",
		"r2.class_a_monthly", "r2.class_b_monthly",
	} {
		if _, ok := limitFor(id, "paid"); !ok {
			t.Errorf("catalog row %s missing or paid tier null", id)
		}
	}
}
```

Run: `go test ./pkg/cosmoflare/limitsdata/ -run TestMonthlyUsageRowsPresent -v` → FAIL.

- [ ] **Step 3: Add rows** with verified values (omit `do.duration_ms_monthly` if unverifiable).
- [ ] **Step 4:** Test passes + existing catalog tests green.
- [ ] **Step 5: Commit** — `feat(limits): verified monthly usage rows for pacing (FEAT-048)`

### Task 2: UsageService — collection + pacing math

**Files:**
- Create: `pkg/cosmoflare/usage.go`, `pkg/cosmoflare/usage_test.go`
- Modify: nothing else

**Interfaces:**
- Consumes: `AnalyticsService.{Workers, DurableObjects, R2Storage, R2Operations}`, `limitFor(resource, plan)` from limitsdata, `AnalyticsWindow`.
- Produces:

```go
type UsageDimension struct {
	ID            string  `json:"id"`             // catalog row id
	Name          string  `json:"name"`
	Unit          string  `json:"unit"`
	Used          float64 `json:"used"`
	Limit         float64 `json:"limit"`          // 0 = no tier value (unknown plan)
	Pct           float64 `json:"pct"`            // used/limit*100, 0 when limit unknown
	ProjectedPct  float64 `json:"projected_pct"`  // linear projection to cycle end
}

type UsageSnapshot struct {
	CycleStart time.Time        `json:"cycle_start"`
	CycleEnd   time.Time        `json:"cycle_end"`
	DaysElapsed, DaysTotal float64
	Dimensions []UsageDimension `json:"dimensions"`
}

// CycleWindow anchors the billing cycle. Zero Anchor = calendar month (UTC)
// containing now. Anniversary billing passes an explicit Anchor day-of-month
// via the CLI flag.
func CycleWindow(anchor int, now time.Time) (start, end time.Time)

// CollectUsage assembles the snapshot: analytics over [start, now) for
// cumulative dims; R2 storage is a gauge read at now. plan routes tier
// resolution ("free"|"paid"|"enterprise"; empty = paid default).
func CollectUsage(ctx context.Context, analytics *AnalyticsService, plan string, anchor int, now time.Time) (*UsageSnapshot, error)
```

Dimension mapping: `workers.requests_monthly` ← Σ `WorkersSummary.Requests`; `workers.cpu_ms_monthly` ← Σ `Requests×CPUP99` is wrong — CPU-ms aggregate needs `workersInvocationsAdaptive` sum field; check the dataset's `sum { cpuTime }` availability in the existing query and add the field if present (one-line query extension + struct field). `do.requests_monthly` ← Σ `DurableObjectSummary.Requests`; `r2.storage_monthly_bytes` ← Σ `R2BucketStorage.PayloadSize`; `r2.class_a/b_monthly` ← Σ `R2OperationCount` grouped by action class (Class A: PutObject/CopyObject/ListObjects/…; B: HeadObject/GetObject — map from the action strings `R2Operations` already returns).

- [ ] **Step 1: Failing tests** — mock analytics server (wave-1 pattern): two scripts + one DO + two buckets + op counts; anchor 1 (calendar), fixed `now`; assert Pct/ProjectedPct arithmetic exactly (e.g. used 40% of limit at day 10 of 30 → projected 120%).
- [ ] **Step 2:** FAIL run. **Step 3:** Implement. **Step 4:** PASS + package green.
- [ ] **Step 5: Commit** — `feat(usage): UsageService — cycle collection + pacing projection (FEAT-048)`

### Task 3: `cosmoflare usage` command

**Files:**
- Create: `cmd/usage.go`, `cmd/usage_test.go`
- Modify: `cmd/cmdmanifest` registration site if one exists (wave-1 lesson: `68baf3b` — register subcommands)

**Interfaces:**
- Produces: `cosmoflare usage [--plan free|paid|enterprise] [--anchor-day N] [--json]` — success envelope `{status: "success", data: UsageSnapshot}`, human table: dimension rows with used/limit, pct bar, projected pct, and a status column (ok / projected-overage when ProjectedPct > 100).

- [ ] **Step 1: Failing tests** — cobra execution against a mock analytics base URL (existing cmd test patterns): JSON mode parses the envelope; table mode names dimensions; unknown-plan dim with limit 0 renders "limit unknown", never NaN.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. **Step 4:** PASS + `go build`.
- [ ] **Step 5: Commit** — `feat(cmd): cosmoflare usage — monthly pacing view (FEAT-048)`

### Task 4: usage-pct + usage-projected-pct conditions

**Files:**
- Modify: `pkg/cosmoflare/alerts.go` (registry), `internal/webhook/evaluator.go` (EvalMetrics.Usage + conditionValue case + collect), `cmd/alerts_watch.go` (usage collection cadence)

**Interfaces:**
- Produces: conditions `usage-pct` (account, `%`, max across dimensions with known limits — the dns-record-quota pattern), `usage-projected-pct` (account, `%`, same max over ProjectedPct). `EvalMetrics.Usage []UsageDimension` populated at most once per 15 min inside the watch (timestamp cache — the D3 slow loop): the collect helper takes a `lastUsage time.Time` state pointer.

- [ ] **Step 1: Failing tests** — registry scope test + coverage fixture row + evaluator test: threshold 80 with one dimension at Pct 62/ProjectedPct 120 → `usage-pct` no-fire, `usage-projected-pct` fires naming the dimension.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. **Step 4:** whole-package PASS.
- [ ] **Step 5: Commit** — `feat(alerts): usage-pct + usage-projected-pct pacing conditions (FEAT-048)`

### Task 5: Documentation (parallel doc agents — standing operator request)

- [ ] Dispatch two agents (USAGE.md conditions rows + `usage` command section; README bullet), verify diffs, commit.

### Task 6: Gate

- [ ] `go build ./... && go vet ./...` exit 0; three affected suites green via `ccs worktree exec`.
- [ ] `ccs verify-worktree <name> --approve --score N --issues 0 && ccs merge <name>`; ancestry check; `ccs kill`.

## Self-Review

- **Spec coverage:** D3 dual cadence lands (Task 4 timestamp cache); D6 usage view (Task 3); ROAD-101 wave-2 text covered. Baselines stay out (model B idea).
- **Placeholder risk:** catalog VALUES are deliberately not in the plan — Task 1's spike forbids memory values; that is a verified-inputs gate, not a TBD.
- **Type consistency:** `UsageDimension`/`UsageSnapshot`/`CycleWindow`/`CollectUsage` signatures used consistently; conditions reuse the scoped-evaluator account path from wave 1.
