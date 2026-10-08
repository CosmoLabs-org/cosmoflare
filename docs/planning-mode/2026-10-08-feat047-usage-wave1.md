---
title: FEAT-047 Wave 1 — Per-Script Stuck-Work Thresholds
created: "2026-10-08T20:05:00+04:00"
issue: FEAT-047
roadmap: ROAD-101
brainstorm_ref: docs/brainstorming/2026-10-08-feat047-usage-waste-alerting.md
status: PLANNED
deliverables:
    - P-01: Registry Scope field + worker-* condition descriptors (pkg/cosmoflare/alerts.go)
    - P-02: EvalMetrics.Scripts population — per-script rows reach the evaluator (internal/webhook/evaluator.go)
    - P-03: Scoped evaluation — per-script threshold firing with offender named in ID and message (internal/webhook/evaluator.go)
    - P-04: DurableObjects analytics — batched GraphQL dataset + DurableObjectSummary (pkg/cosmoflare/analytics.go)
    - P-05: do-* descriptors + DO wiring through scoped evaluation
    - P-06: USAGE.md + help coverage for the new conditions
---

# FEAT-047 Wave 1 — Per-Script Stuck-Work Thresholds

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `alerts watch` fires threshold alerts that NAME the stuck or overworking Worker script or Durable Object ("worker api-proxy CPU p99 812ms ≥ 500ms threshold").

**Architecture:** The data already flows — `AnalyticsService.Workers()` returns per-script rows (`WorkersSummary{Script, Requests, Errors, Subrequests, CPUP50, CPUP99}`) and `CollectEvalMetrics` flattens them. Wave 1 stops discarding the rows, adds a `Scope` dimension to the condition registry (account vs script vs do-object), and generalizes the evaluator from "one value per rule" to "one value per rule × scope-instance". Durable Objects get a new batched GraphQL query. No cadence changes (60s loop exists), no evaluator latch state (window smoothing already comes from the analytics window; cooldown semantics unchanged).

**Tech Stack:** Go 1.26, cobra, existing testify test patterns, Cloudflare GraphQL Analytics API.

**Spec:** docs/brainstorming/2026-10-08-feat047-usage-waste-alerting.md

## Global Constraints

- Registry is the single definition site (FEAT-015): every condition appears in `alertConditionRegistry` with `Scope`, `Unit`, `Help`, `FedBy`, `DataKey`, `Service`; `conditionValues` must implement every registered name (existing coverage test enforces this — keep it green).
- Alert identity: per-script/per-DO fires use `ID = rule.Name + "/" + scopeID` but `Name = rule.Name` — cmd/alerts_watch.go:233 looks rules up via `byName[p.Alert.Name]`; breaking that breaks severity/service mapping.
- One batched GraphQL query per dataset per cycle — never per-script N+1 (brainstorm D5).
- Existing flat conditions (`error-rate`, `latency`, …) behave identically — their tests must pass unchanged.
- Every command keeps `--json` and rich help (repo convention).
- Conventional commits, no AI attribution.

## Deviation from brainstorm (recorded)

The brainstorm sketched "N consecutive windows" latch state in the evaluator.
Dropped for wave 1 (YAGNI): the analytics window (last N minutes of data)
already smooths, the one-shot-per-cycle evaluator discards state by design
(cmd/alerts_watch.go:253-256), and cooldown semantics handle re-fire. If
latching proves necessary it becomes a wave-2 refinement with usage pacing.

---

### Task 1: Registry Scope field + worker-* descriptors

**Files:**
- Modify: `pkg/cosmoflare/alerts.go:98-116` (descriptor struct + registry slice)
- Test: `pkg/cosmoflare/alerts_test.go` (append)

**Interfaces:**
- Produces: `AlertConditionDescriptor.Scope string` — values `"account"` (default/empty), `"script"`, `"do"`. Produces four registered conditions: `worker-cpu` (unit `ms`, scope `script`), `worker-errors` (unit `errors`, scope `script`), `worker-requests` (unit `requests`, scope `script`), `worker-subrequests` (unit `subrequests`, scope `script`).

- [ ] **Step 1: Write the failing test**

```go
func TestAlertConditionScopes(t *testing.T) {
	for _, name := range []string{"worker-cpu", "worker-errors", "worker-requests", "worker-subrequests"} {
		desc, ok := LookupAlertCondition(name)
		if !ok {
			t.Fatalf("condition %s not registered", name)
		}
		if desc.Scope != "script" {
			t.Errorf("%s: Scope=%q, want script", name, desc.Scope)
		}
		if desc.DataKey == "" || desc.Unit == "" || desc.Help == "" || desc.FedBy == "" || desc.Service == "" {
			t.Errorf("%s: descriptor has empty fields: %+v", name, desc)
		}
	}
	// account conditions keep zero-value scope compatibility
	if d, _ := LookupAlertCondition("error-rate"); d.Scope != "account" && d.Scope != "" {
		t.Errorf("error-rate scope drifted: %q", d.Scope)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestAlertConditionScopes -v`
Expected: FAIL — `condition worker-cpu not registered`.

- [ ] **Step 3: Implement**

In `alerts.go`, extend the descriptor struct (add after `Service`):

```go
Scope string // "account" (default), "script", "do" — instances the evaluator fans out over
```

Append to `alertConditionRegistry`:

```go
{Name: "worker-cpu", Scope: "script", Unit: "ms", Help: "per-script Workers CPU p99 over the window", FedBy: "Workers analytics per-script rows", DataKey: "Scripts[].CPUP99", Service: "workers"},
{Name: "worker-errors", Scope: "script", Unit: "errors", Help: "per-script Workers error count over the window", FedBy: "Workers analytics per-script rows", DataKey: "Scripts[].Errors", Service: "workers"},
{Name: "worker-requests", Scope: "script", Unit: "requests", Help: "per-script Workers request volume over the window", FedBy: "Workers analytics per-script rows", DataKey: "Scripts[].Requests", Service: "workers"},
{Name: "worker-subrequests", Scope: "script", Unit: "subrequests", Help: "per-script subrequest count — the stuck-loop fan-out signature", FedBy: "Workers analytics per-script rows", DataKey: "Scripts[].Subrequests", Service: "workers"},
```

Set `Scope: "account"` explicitly on the seven existing entries (clearer than relying on zero value).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestAlertCondition' -v`
Expected: PASS (new test + existing registry tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/alerts.go pkg/cosmoflare/alerts_test.go
git commit -m "feat(alerts): registry Scope dimension + worker-* stuck-work conditions (FEAT-047)"
```

---

### Task 2: EvalMetrics carries per-script rows

**Files:**
- Modify: `internal/webhook/evaluator.go:20-31` (struct) and `CollectEvalMetrics` (line 188+)
- Test: `internal/webhook/evaluator_test.go` (append)

**Interfaces:**
- Consumes: `cosmoflare.WorkersSummary` (pkg/cosmoflare/analytics.go:44).
- Produces: `EvalMetrics.Scripts []cosmoflare.WorkersSummary` — populated by `CollectEvalMetrics`; flat fields (`WorkersRequests` etc.) keep their existing meaning (account sums) so account conditions are untouched.

- [ ] **Step 1: Write the failing test**

```go
func TestCollectEvalMetricsKeepsScriptRows(t *testing.T) {
	// Mock analytics server returning two script rows — reuse the existing
	// mock pattern from evaluator_test.go's CollectEvalMetrics tests
	// (httptest server + WithAnalyticsBaseURL).
	m, err := CollectEvalMetrics(ctx, analytics, window)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(m.Scripts) != 2 {
		t.Fatalf("Scripts len = %d, want 2 (rows must not be discarded)", len(m.Scripts))
	}
	// flat fields still carry the account sums
	if m.WorkersRequests == 0 {
		t.Error("WorkersRequests account sum lost")
	}
}
```

(Fill the mock from the existing `TestCollectEvalMetrics*` fixtures in the same file — same handler shape, two `scriptName` dimensions.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/webhook/ -run TestCollectEvalMetricsKeepsScriptRows -v`
Expected: FAIL — `Scripts len = 0`.

- [ ] **Step 3: Implement**

Add to `EvalMetrics`:

```go
Scripts []cosmoflare.WorkersSummary // per-script rows (stuck-work conditions); flat fields stay account sums
```

In `CollectEvalMetrics`, the `scripts` slice already exists (line 189: `scripts, err := analytics.Workers(...)`) — assign it: `m.Scripts = scripts`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/webhook/ -run 'TestCollectEvalMetrics' -v`
Expected: PASS — new + existing collection tests.

- [ ] **Step 5: Commit**

```bash
git add internal/webhook/evaluator.go internal/webhook/evaluator_test.go
git commit -m "feat(alerts): EvalMetrics carries per-script Workers rows (FEAT-047)"
```

---

### Task 3: Scoped evaluation — per-script firing with offender named

**Files:**
- Modify: `internal/webhook/evaluator.go:74-182` (conditionValue → conditionValues, Evaluate loop)
- Test: `internal/webhook/evaluator_test.go` (append)

**Interfaces:**
- Consumes: Task 1's `Descriptor.Scope`, Task 2's `EvalMetrics.Scripts`.
- Produces:

```go
type scopedValue struct {
	ScopeID string // "" for account scope; script/DO name otherwise
	Value   float64
	Unit    string
}

func conditionValues(condition string, m EvalMetrics) []scopedValue
```

`Evaluate` fires once per (rule, scopedValue) pair; `conditionValue(condition, m)` remains as the account-scope convenience wrapper (existing tests use it).

- [ ] **Step 1: Write the failing test**

```go
func TestEvaluateWorkerCPUNamesOffender(t *testing.T) {
	svc := newTestRuleService(t) // existing helper: creates AlertService with a rule
	if err := svc.Create("hot-cpu", cosmoflare.AlertRule{
		Condition: "worker-cpu", Threshold: 500, Enabled: true,
	}); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	var got []webhook.NotificationPayload // capture via Manager.SetNotifier, existing pattern
	m := EvalMetrics{Scripts: []cosmoflare.WorkersSummary{
		{Script: "api-proxy", CPUP99: 812, Requests: 100},
		{Script: " calm-worker", CPUP99: 40, Requests: 100},
	}}
	fired := eval.Evaluate(m)
	if len(fired) != 1 || fired[0] != "hot-cpu" {
		t.Fatalf("fired = %v, want [hot-cpu] once", fired)
	}
	p := got[0]
	if p.Alert.ID != "hot-cpu/api-proxy" {
		t.Errorf("Alert.ID = %q, want hot-cpu/api-proxy (per-script identity)", p.Alert.ID)
	}
	if p.Alert.Name != "hot-cpu" {
		t.Errorf("Alert.Name = %q, want hot-cpu (rule-name lookup for severity mapping)", p.Alert.Name)
	}
	if !strings.Contains(p.Message, "api-proxy") || !strings.Contains(p.Message, "812") {
		t.Errorf("message must name script and value: %q", p.Message)
	}
}

func TestEvaluateAccountConditionsUnchanged(t *testing.T) {
	// error-rate with the same metrics as the pre-existing test — assert
	// identical fire/no-fire behavior (regression guard for the refactor).
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/webhook/ -run 'TestEvaluateWorker|TestEvaluateAccount' -v`
Expected: worker test FAIL (no per-script path); account test PASS.

- [ ] **Step 3: Implement**

Add `scopedValue` + `conditionValues`:

```go
func conditionValues(condition string, m EvalMetrics) []scopedValue {
	desc, ok := cosmoflare.LookupAlertCondition(condition)
	if !ok {
		return nil
	}
	switch desc.Scope {
	case "script":
		out := make([]scopedValue, 0, len(m.Scripts))
		for _, s := range m.Scripts {
			var v float64
			switch condition {
			case "worker-cpu":
				v = s.CPUP99
			case "worker-errors":
				v = float64(s.Errors)
			case "worker-requests":
				v = float64(s.Requests)
			case "worker-subrequests":
				v = float64(s.Subrequests)
			default:
				return nil // registered-but-unimplemented: coverage test guards
			}
			out = append(out, scopedValue{ScopeID: s.Script, Value: v, Unit: desc.Unit})
		}
		return out
	default: // account
		v, unit, ok := conditionValue(condition, m)
		if !ok {
			return nil
		}
		return []scopedValue{{Value: v, Unit: unit}}
	}
}
```

Rewrite the `Evaluate` inner loop: replace `value, unit, ok := conditionValue(...)` with `for _, sv := range conditionValues(rule.Condition, m) { ... }` — fire check `sv.Value >= rule.Threshold`, cooldown keyed `rule.Name` (unchanged semantics), `alert.ID = rule.Name` when `sv.ScopeID == ""` else `rule.Name+"/"+sv.ScopeID`, `alert.Name = rule.Name` always, message:

```go
message := fmt.Sprintf("%s %s: observed %g meets threshold %g (%s)",
	rule.Condition, rule.Name, sv.Value, rule.Threshold, sv.Unit)
if sv.ScopeID != "" {
	message = fmt.Sprintf("%s %s [%s]: observed %g meets threshold %g (%s)",
		rule.Condition, sv.ScopeID, rule.Name, sv.Value, rule.Threshold, sv.Unit)
}
```

Skip zero-signal script rows (Requests==0 && CPUP99==0 && Errors==0 && Subrequests==0) so idle scripts never fire volume conditions.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/webhook/ -v`
Expected: PASS — whole package, including the registry-coverage test.

- [ ] **Step 5: Commit**

```bash
git add internal/webhook/evaluator.go internal/webhook/evaluator_test.go
git commit -m "feat(alerts): scoped evaluation — stuck-work alerts name the offender (FEAT-047)"
```

---

### Task 4: DurableObjects analytics dataset

**Files:**
- Modify: `pkg/cosmoflare/analytics.go` (new method + type, after `Workers()`)
- Test: `pkg/cosmoflare/analytics_test.go` (append)

**Interfaces:**
- Produces:

```go
type DurableObjectSummary struct {
	Script   string  `json:"script"`   // scriptName dimension (DO namespace lives under the script)
	Requests uint64  `json:"requests"` // sum of requests
	CPUTime  float64 `json:"cpu_ms"`   // sum of cpuTime (ms)
	Errors   uint64  `json:"errors"`   // sum of errors when present in the dataset
}

func (s *AnalyticsService) DurableObjects(ctx context.Context, w AnalyticsWindow) ([]DurableObjectSummary, error)
```

- [ ] **Step 1: Verify the GraphQL dataset shape (spike, throwaway)**

The DO analytics dataset name/fields must come from the live schema, not memory. Run against the real API once (operator token in env):

```bash
curl -s https://api.cloudflare.com/client/v4/graphql \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"{ viewer { accounts(filter: {accountTag: \"'$CLOUDFLARE_ACCOUNT_ID'\"}) { durableObjectsAnalytics(limit: 1, filter: {datetime_geq: \"2026-10-01T00:00:00Z\"}) { sum { requests cpuTime } dimensions { scriptName } } } } }"}'
```

If `durableObjectsAnalytics` errors, introspect: `{"query":"{ __type(name: \"AccountAnalyticsGroup\") { possibleTypes { name } } }"}` and locate the DO dataset (`*Adaptive` suffix preferred). Record the verified field names; adjust Step 3's query + struct tags to match. If the dataset is account-inaccessible, STOP and report — do not guess fields.

- [ ] **Step 2: Write the failing test** (mock server, existing pattern)

```go
func TestDurableObjects(t *testing.T) {
	svc, server := analyticsMockSetup(t, `{"data":{"viewer":{"accounts":[{"durableObjectsAnalytics":[
		{"sum":{"requests":4200,"cpuTime":910.5},"dimensions":{"scriptName":"lobby-do"}}]}]}}}}`)
	defer server.Close()
	rows, err := svc.DurableObjects(ctx, win)
	if err != nil {
		t.Fatalf("DurableObjects: %v", err)
	}
	if len(rows) != 1 || rows[0].Script != "lobby-do" || rows[0].Requests != 4200 || rows[0].CPUTime != 910.5 {
		t.Fatalf("rows = %+v", rows)
	}
}
```

Run: `go test ./pkg/cosmoflare/ -run TestDurableObjects -v` → Expected: FAIL (method undefined).

- [ ] **Step 3: Implement** — mirror `Workers()` (analytics.go:330): same envelope, `validateAccount` + `validateAnalyticsWindow(op, w, 92*24*time.Hour)`, aggregate rows keyed by `scriptName`, one batched query.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestDurableObjects|TestWorkers' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/analytics.go pkg/cosmoflare/analytics_test.go
git commit -m "feat(analytics): DurableObjects dataset — batched DO telemetry (FEAT-047)"
```

---

### Task 5: do-* conditions wired end to end

**Files:**
- Modify: `pkg/cosmoflare/alerts.go` (registry), `internal/webhook/evaluator.go` (EvalMetrics.DurableObjects + conditionValues case)
- Test: both `_test.go` files

**Interfaces:**
- Consumes: Task 4's `DurableObjectSummary` + `DurableObjects()`.
- Produces: conditions `do-cpu` (unit `ms`, scope `do`, DataKey `DurableObjects[].CPUTime`), `do-requests` (unit `requests`, scope `do`). `EvalMetrics.DurableObjects []cosmoflare.DurableObjectSummary`; `CollectEvalMetrics` gains the DO call — a DO dataset error is non-fatal (log + empty rows) so DO-less accounts never break the watch cycle.

- [ ] **Step 1: Failing tests** — registry scope test extended with `do-cpu`/`do-requests` (Task 1 pattern); evaluator test: rule `do-cpu` threshold 500, `DurableObjects: []cosmoflare.DurableObjectSummary{{Script: "lobby-do", CPUTime: 910.5}}` → fires, `ID = "rule/lobby-do"`, message names `lobby-do`.

- [ ] **Step 2: Verify fail** — `go test ./pkg/cosmoflare/ ./internal/webhook/ -run 'Do|ConditionScopes' -v` → FAIL on the new names.

- [ ] **Step 3: Implement** — registry entries; `EvalMetrics.DurableObjects` field; `CollectEvalMetrics` DO call with non-fatal error handling; `conditionValues` gains `case "do":` iterating `m.DurableObjects` (`do-cpu` → `CPUTime`, `do-requests` → `Requests`).

- [ ] **Step 4: Verify pass** — `go test ./internal/webhook/ ./pkg/cosmoflare/ -v` → PASS, coverage test green.

- [ ] **Step 5: Commit** — `feat(alerts): do-cpu/do-requests conditions end to end (FEAT-047)`

---

### Task 6: Docs and help coverage

**Files:**
- Modify: `docs/USAGE.md` (alerts watch section — condition table), `cmd/alerts.go` help text if it enumerates conditions.

**Interfaces:** none (docs).

- [ ] **Step 1:** Add the six conditions to the USAGE.md alerts table (name, unit, scope, what it catches, sensible default thresholds: worker-cpu 500ms, worker-subrequests 10000/window, do-cpu 500ms).
- [ ] **Step 2:** Check `cosmoflare alerts rules create --help` output lists conditions from the registry dynamically — if hardcoded, point it at `AlertConditions()` (one-line fix + test).
- [ ] **Step 3:** `go build ./... && go vet ./...` → exit 0.
- [ ] **Step 4: Commit** — `docs(usage): stuck-work alert conditions (FEAT-047)`

---

### Task 7: Full verification gate

- [ ] `go test ./pkg/cosmoflare/ ./internal/webhook/ ./cmd/ -timeout 120s` → PASS (suite now fits the budget — see 5d5d5df).
- [ ] `go build ./... && go vet ./...` → exit 0.
- [ ] Manual smoke: `go run . alerts rules create stuck-cpu --condition worker-cpu --threshold 500` then `alerts check --json` against a real account shows the rule listed and evaluated (operator-gated — mark deferred if no live account this session).
- [ ] Update FEAT-047 acceptance criteria checkboxes; changelog entry via `ccs changelog add FEAT-047 --type added`.

---

## Self-Review

- **Spec coverage:** D1 thresholds ✓ (Tasks 1-3, 5), D4 per-script naming ✓ (Task 3), D5 batched GraphQL ✓ (Tasks 2, 4 — one query per dataset), D3 dual cadence — deferred to wave 2 with rationale (60s loop exists; deviation recorded). DO telemetry ✓ (Tasks 4-5). Pager delivery ✓ (existing pipeline, Task 3 preserves Name-lookup seam). `--json`/help ✓ (Task 6).
- **Placeholder scan:** the two "fill from existing pattern" test bodies (Tasks 2, 3) reference concrete existing fixtures by name — the pattern is in-file; code shown is the assertion core. No TBDs.
- **Type consistency:** `scopedValue`, `conditionValues`, `DurableObjectSummary`, `EvalMetrics.Scripts/.DurableObjects` used identically across tasks; `AlertConditionDescriptor.Scope` values (`account|script|do`) consistent.
