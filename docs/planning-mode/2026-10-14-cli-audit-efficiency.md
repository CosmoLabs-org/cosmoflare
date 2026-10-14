---
title: 'CLI audit + efficiency — wave 1: cosmoflare efficiency + cost waste line'
status: PLANNED
branch: master
created: "2026-10-14T15:05:00+04:00"
brainstorm_ref: docs/brainstorming/2026-10-14-cli-audit-efficiency.md
prompt_ref: docs/prompts/2026-10-14-cli-pivot-audit-efficiency.md
related_issues:
    - FEAT-069
    - FEAT-047
    - FEAT-048
tags: [efficiency, audit, cli, wave-1]
last_reviewed: "2026-10-14T15:35:00+04:00"
last_review_ref: docs/independent-reviews/2026-10-14T15-35-00-cli-audit-efficiency-trio.md
last_review_findings: 5
deliverables:
    - id: P-01
      title: "AnalyticsService.KVOperations — kvOperationsAdaptiveGroups dataset, batched, tested"
    - id: P-02
      title: "EfficiencyService + Finding/WasteSummary model in pkg/cosmoflare/efficiency.go, all three checks tested"
    - id: P-03
      title: "cmd/efficiency.go command — usage.go seam pattern, doctor-grade help, JSON envelope, 4 named tests green"
    - id: P-04
      title: "cmd/cost.go idle-waste line reusing efficiency collectors — TestCostIncludesIdleWaste green"
    - id: P-05
      title: "Live smoke of efficiency + cost against the real account"
---

# Cosmoflare `efficiency` — Wave 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `cosmoflare efficiency` — a findings command that reports idle Cloudflare resources from real API numbers — and wire an idle-waste line into `cosmoflare cost`.

**Architecture:** A new `EfficiencyService` in `pkg/cosmoflare/efficiency.go` composes three existing services (WorkerService, KVService, AnalyticsService) plus one new GraphQL dataset (`kvOperationsAdaptiveGroups` added to `AnalyticsService`). It emits an `EfficiencyReport` of `EfficiencyFinding` values (what/why/evidence/fix). The cmd layer (`cmd/efficiency.go`) mirrors `cmd/usage.go`'s seam pattern (`efficiencyCollectFn`, writer vars). Cost wiring reuses the same collectors through a best-effort seam — no second collector.

**Tech Stack:** Go 1.26, cobra, cloudflare-go, GraphQL Analytics API (graphql.cloudflare.com via existing AnalyticsService), stdlib testing + httptest.

**Spec:** `docs/brainstorming/2026-10-14-cli-audit-efficiency.md` (wave 1 section + decisions D1-D6).

## Global Constraints

- Real API numbers only — never modeled. Any dollar figure must be a rate × a real measured number (house rule, KV 86× incident).
- `--json` envelope on the command: `{"status":"success","data":...}` (usage.go precedent).
- Agent-readable findings: every finding carries evidence (counts, window) and an actionable fix.
- One batched GraphQL query per dataset per run — never per-resource N+1 (FEAT-047 rule).
- TDD: write the failing test first, run it red, implement, run green, commit (conventional commits, no AI attribution).
- Do not rename or restructure `cosmoflare audit` (decision D1).
- Tests must not require live credentials: httptest servers + WithAnalyticsBaseURL / cloudflare.BaseURL mocking only.

## Reference files (read before implementing)

| File | Why |
|------|-----|
| `cmd/usage.go` | The seam pattern to mirror exactly (collectFn + writer vars + JSON envelope) |
| `cmd/doctor.go` | Diagnostic command shape: rich Long help, findings printing, fix suggestions |
| `pkg/cosmoflare/analytics.go` | AnalyticsService: `query()` helper, `Workers()`, `R2Storage()`, `windowVars()`, `AnalyticsWindow`; the decode-struct pattern for GraphQL responses |
| `pkg/cosmoflare/kv.go` | `KVService.ListNamespaces`, `KVNamespace{ID,Title}` |
| `pkg/cosmoflare/worker.go` | `WorkerService.List`, `NewWorkerServiceFromCreds` |
| `pkg/cosmoflare/kv_test.go` | httptest mock pattern: `cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))` |
| `cmd/cost.go`, `cmd/cost_test.go` | Cost command structure being extended in Task 3 |

---

## Task 1: Library — `pkg/cosmoflare/efficiency.go` (EfficiencyService + findings model)

**Model for dispatch:** sonnet (bounded-with-discovery: matches existing service patterns across 4 files)

**Files:**
- Create: `pkg/cosmoflare/efficiency.go`
- Create: `pkg/cosmoflare/efficiency_test.go`
- Modify: `pkg/cosmoflare/analytics.go` (add `KVOperations` method + types, following the existing `Workers()` method pattern)
- Modify: `pkg/cosmoflare/analytics_test.go` (add KVOperations tests)

**Interfaces:**
- Consumes: `WorkerService.List(ctx) ([]*Worker, error)`; `KVService.ListNamespaces(ctx) ([]*KVNamespace, error)`; `AnalyticsService.Workers(ctx, w) ([]WorkersSummary, error)`; `AnalyticsService.R2Storage(ctx, w) ([]R2BucketStorage, error)`; `AnalyticsService.query(...)`, `AnalyticsService.windowVars(w)`; `AnalyticsWindow{Start,End}`; `Worker.Name`, `KVNamespace{ID,Title}`, `WorkersSummary{Script,Requests}`, `R2BucketStorage{Bucket,ObjectCount}`.
- Produces (Task 2 and 3 depend on these exact names):
  - `type EfficiencyFinding struct { Resource, Check, Severity, Evidence, Fix string }`
  - `type EfficiencyReport struct { WindowStart, WindowEnd time.Time; Findings []EfficiencyFinding; WorkersChecked, KVNamespacesChecked, BucketsChecked int }`
  - `type EfficiencyWasteSummary struct { IdleWorkers, DeadKVNamespaces, EmptyR2Buckets int; EstimatedMonthlyUSD float64; Basis string }`
  - `func NewEfficiencyService(workers *WorkerService, kv *KVService, analytics *AnalyticsService) *EfficiencyService`
  - `func NewEfficiencyServiceFromCreds(accountID, apiToken string) (*EfficiencyService, error)`
  - `func (s *EfficiencyService) Report(ctx context.Context, w AnalyticsWindow) (*EfficiencyReport, error)`
  - `func SummarizeWaste(r *EfficiencyReport) *EfficiencyWasteSummary`
  - `type KVOperationsSummary struct { NamespaceID string; Reads, Writes uint64 }`
  - `func (s *AnalyticsService) KVOperations(ctx context.Context, w AnalyticsWindow) ([]KVOperationsSummary, error)`

Design decisions locked in (do not deviate):
- **Idle = absent or zero.** A worker/KV namespace/bucket that never appears in the analytics response counts as zero usage. Never treat absence as an error.
- **Empty R2 detection uses `R2Storage` analytics (batched), not per-bucket object listing.** Buckets are enumerated from a `BucketLister` interface so the cmd layer can pass the existing R2 client without this package importing S3 machinery:
  ```go
  // efficiency.go
  type BucketLister interface { ListBuckets(ctx context.Context) ([]*Bucket, error) }
  func NewEfficiencyServiceWithBuckets(workers *WorkerService, kv *KVService, analytics *AnalyticsService, buckets BucketLister) *EfficiencyService
  ```
  (`NewEfficiencyService` passes nil buckets → empty-bucket checks are skipped and `BucketsChecked` stays 0; the cmd layer always passes the real client.)
- **Severity:** idle worker = `warning`, dead KV namespace = `warning`, empty R2 bucket = `info`.
- **EstimatedMonthlyUSD in wave 1 is always a real rate × real measured number.** Idle workers and empty buckets meter at $0 (requests/storage are the meter and both are zero) — the summary reports 0.00 with a `Basis` string that says storage-byte attribution for dead KV namespaces lands with wave-2 KV storage analytics. No guessed dollar figures, ever.

- [ ] **Step 1: Write the failing tests for the KVOperations dataset** (in `pkg/cosmoflare/analytics_test.go`; mirror the existing analytics tests' httptest + `WithAnalyticsBaseURL(server.URL)` setup):

```go
func TestAnalyticsKVOperations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil { // sanity — request must be a GraphQL POST
			t.Errorf("parse form: %v", err)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if !strings.Contains(body["query"].(string), "kvOperationsAdaptiveGroups") {
			t.Errorf("query misses kvOperationsAdaptiveGroups: %v", body["query"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"viewer":{"accounts":[{"kvOperationsAdaptiveGroups":[`+
			`{"dims":{"namespaceID":"ns-live"},"sum":{"reads":24100,"writes":2500}},`+
			`{"dims":{"namespaceID":"ns-dead"},"sum":{"reads":0,"writes":0}}]}]}}}`)
	}))
	defer srv.Close()
	svc := NewAnalyticsService("acct-1", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := svc.KVOperations(context.Background(), AnalyticsWindow{Start: time.Now().Add(-168 * time.Hour), End: time.Now()})
	if err != nil {
		t.Fatalf("KVOperations: %v", err)
	}
	if len(got) != 2 || got[0].NamespaceID != "ns-live" || got[0].Reads != 24100 || got[1].Writes != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestAnalyticsKVOperationsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{"kvOperationsAdaptiveGroups":[]}]}}}`)
	}))
	defer srv.Close()
	svc := NewAnalyticsService("acct-1", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := svc.KVOperations(context.Background(), AnalyticsWindow{Start: time.Now().Add(-time.Hour), End: time.Now()})
	if err != nil {
		t.Fatalf("KVOperations: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 rows, got %+v", got)
	}
}
```

- [ ] **Step 2: Run to verify red**

Run: `go test ./pkg/cosmoflare/ -run TestAnalyticsKVOperations -v`
Expected: FAIL — `svc.KVOperations undefined`.

- [ ] **Step 3: Implement `KVOperations` in `analytics.go`** — copy the `Workers()` method's structure (query constant, response structs, `s.query(ctx, "KVOperations", gql, s.windowVars(w), &resp)`, account validation, row flattening):

```go
// KVOperationsSummary is KV traffic for one namespace over a window.
type KVOperationsSummary struct {
	NamespaceID string `json:"namespace_id"`
	Reads       uint64 `json:"reads"`
	Writes      uint64 `json:"writes"`
}

// KVOperations returns per-namespace KV reads/writes for the window, one
// batched query. Namespaces with no traffic may be absent — absence means
// zero, callers must not treat it as an error.
func (s *AnalyticsService) KVOperations(ctx context.Context, w AnalyticsWindow) ([]KVOperationsSummary, error) {
	if err := s.validateAccount("KVOperations"); err != nil {
		return nil, err
	}
	const gql = `query KVOperations($accountTag: string!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      kvOperationsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        dims { namespaceID }
        sum { reads writes }
      }
    }
  }
}`
	var resp struct {
		Viewer struct {
			Accounts []struct {
				Groups []struct {
					Dim struct {
						NamespaceID string `json:"namespaceID"`
					} `json:"dims"`
					Sum struct {
						Reads  uint64 `json:"reads"`
						Writes uint64 `json:"writes"`
					} `json:"sum"`
				} `json:"kvOperationsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, "KVOperations", gql, s.windowVars(w), &resp); err != nil {
		return nil, err
	}
	out := make([]KVOperationsSummary, 0)
	for _, a := range resp.Viewer.Accounts {
		for _, g := range a.Groups {
			out = append(out, KVOperationsSummary{NamespaceID: g.Dim.NamespaceID, Reads: g.Sum.Reads, Writes: g.Sum.Writes})
		}
	}
	return out, nil
}
```

Note: the GraphQL field shape (`dims.namespaceID`, `sum.reads`, `sum.writes`) matches the pattern of every other dataset in this file. If the live API rejects a field name, the response error will name it — adjust the query string only, never the Go contract. The `limit: 10000` cap does NOT hit the D1 pagination trap here: grouping by `namespaceID` alone collapses time, so the row count equals the number of namespaces with activity in the window — not rows × days. An account near 10k active namespaces would truncate silently; that is far outside any realistic CosmoLabs account and wave 2 (FEAT-069 pagination machinery) covers it if it ever matters.

- [ ] **Step 4: Run to verify green**

Run: `go test ./pkg/cosmoflare/ -run TestAnalyticsKVOperations -v`
Expected: PASS ×2.

- [ ] **Step 5: Write the failing EfficiencyService tests** (`pkg/cosmoflare/efficiency_test.go`). Build the three backing services against httptest servers exactly like `kv_test.go`'s `kvMockSetup`, plus an analytics server with `WithAnalyticsBaseURL`. One handler serves all fixtures by path/body:

```go
package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// effMock spins the REST mock (workers list + KV namespaces) and the GraphQL
// mock (workers sums, KV sums, R2 storage) and wires an EfficiencyService.
type effMock struct {
	svc    *EfficiencyService
	rest   *httptest.Server
	gql    *httptest.Server
}

func effWrite(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
}

func newEffMock(t *testing.T, workersList []any, kvList []any, gqlBody string) *effMock {
	t.Helper()
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/accounts/acct-eff/workers/scripts":
			effWrite(w, workersList)
		case r.URL.Path == "/accounts/acct-eff/storage/kv/namespaces":
			effWrite(w, kvList)
		default:
			t.Errorf("unexpected REST call %s %s", r.Method, r.URL.Path)
			effWrite(w, []any{})
		}
	}))
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gqlBody)
	}))
	cf, _ := cloudflare.NewWithAPIToken("tok", cloudflare.BaseURL(rest.URL))
	workers, _ := NewWorkerService(cf, "acct-eff")
	kv, _ := NewKVService(cf, "acct-eff")
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	return &effMock{
		svc:  NewEfficiencyService(workers, kv, analytics),
		rest: rest, gql: gql,
	}
}

func effWindow() AnalyticsWindow {
	return AnalyticsWindow{Start: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)}
}
```

Then the behavior tests (each asserts the finding contents — resource, evidence, fix):

```go
func TestEfficiencyIdleWorkers(t *testing.T) {
	// api-proxy has traffic; ghost-worker does not appear in analytics → idle.
	m := newEffMock(t,
		[]any{map[string]any{"id": "api-proxy"}, map[string]any{"id": "ghost-worker"}},
		[]any{},
		`{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[`+
			`{"scriptName":"api-proxy","sum":{"requests":5000,"errors":0,"subrequests":0}}]}]}}}`,
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.WorkersChecked != 2 {
		t.Fatalf("WorkersChecked = %d, want 2", rep.WorkersChecked)
	}
	var idle []EfficiencyFinding
	for _, f := range rep.Findings {
		if f.Check == "idle-worker" {
			idle = append(idle, f)
		}
	}
	if len(idle) != 1 || idle[0].Resource != "worker:ghost-worker" {
		t.Fatalf("idle findings = %+v", idle)
	}
	if idle[0].Severity != "warning" || idle[0].Evidence == "" || idle[0].Fix == "" {
		t.Fatalf("finding lacks severity/evidence/fix: %+v", idle[0])
	}
}

func TestEfficiencyDeadKVNamespaces(t *testing.T) {
	// ns-live has reads; ns-dead returns zeros; ns-absent never appears.
	m := newEffMock(t, []any{},
		[]any{map[string]any{"id": "ns-live", "title": "live"}, map[string]any{"id": "ns-dead", "title": "dead"}, map[string]any{"id": "ns-absent", "title": "absent"}},
		`{"data":{"viewer":{"accounts":[{"kvOperationsAdaptiveGroups":[`+
			`{"dims":{"namespaceID":"ns-live"},"sum":{"reads":100,"writes":5}},`+
			`{"dims":{"namespaceID":"ns-dead"},"sum":{"reads":0,"writes":0}}]}]}}}`,
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.KVNamespacesChecked != 3 {
		t.Fatalf("KVNamespacesChecked = %d, want 3", rep.KVNamespacesChecked)
	}
	dead := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Check == "dead-kv-namespace" {
			dead[f.Resource] = true
		}
	}
	// zero-rows AND absent rows are both dead
	if len(dead) != 2 || !dead["kv:ns-dead"] || !dead["kv:ns-absent"] {
		t.Fatalf("dead findings = %+v", dead)
	}
}

func TestEfficiencyEmptyR2Buckets(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("tok")
	workers, _ := NewWorkerService(cf, "acct-eff")
	kv, _ := NewKVService(cf, "acct-eff")
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[`+
			`{"dims":{"bucketName":"has-things"},"sum":{"objectCount":42,"payloadSize":1000,"metadataSize":10,"uploadCount":0}},`+
			`{"dims":{"bucketName":"void"},"sum":{"objectCount":0,"payloadSize":0,"metadataSize":0,"uploadCount":0}}]}]}}}`)
	}))
	defer gql.Close()
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	buckets := stubBucketLister{names: []string{"has-things", "void", "never-seen"}}
	svc := NewEfficiencyServiceWithBuckets(workers, kv, analytics, buckets)
	rep, err := svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.BucketsChecked != 3 {
		t.Fatalf("BucketsChecked = %d, want 3", rep.BucketsChecked)
	}
	empty := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Check == "empty-r2-bucket" {
			empty[f.Resource] = true
		}
	}
	if len(empty) != 2 || !empty["r2:void"] || !empty["r2:never-seen"] {
		t.Fatalf("empty findings = %+v", empty)
	}
}

type stubBucketLister struct{ names []string }

func (s stubBucketLister) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	out := make([]*Bucket, 0, len(s.names))
	for _, n := range s.names {
		out = append(out, &Bucket{Name: n})
	}
	return out, nil
}

func TestEfficiencyCleanAccount(t *testing.T) {
	// everything busy → zero findings, no error
	m := newEffMock(t,
		[]any{map[string]any{"id": "busy"}},
		[]any{map[string]any{"id": "ns-live", "title": "live"}},
		`{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[`+
			`{"scriptName":"busy","sum":{"requests":10,"errors":0,"subrequests":0}}],`+
			`"kvOperationsAdaptiveGroups":[{"dims":{"namespaceID":"ns-live"},"sum":{"reads":1,"writes":1}}]}]}}}`,
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("want 0 findings, got %+v", rep.Findings)
	}
}

func TestSummarizeWaste(t *testing.T) {
	rep := &EfficiencyReport{
		Findings: []EfficiencyFinding{
			{Resource: "worker:a", Check: "idle-worker", Severity: "warning"},
			{Resource: "worker:b", Check: "idle-worker", Severity: "warning"},
			{Resource: "kv:x", Check: "dead-kv-namespace", Severity: "warning"},
			{Resource: "r2:y", Check: "empty-r2-bucket", Severity: "info"},
		},
	}
	s := SummarizeWaste(rep)
	if s.IdleWorkers != 2 || s.DeadKVNamespaces != 1 || s.EmptyR2Buckets != 1 {
		t.Fatalf("summary = %+v", s)
	}
	if s.EstimatedMonthlyUSD != 0 {
		t.Fatalf("EstimatedMonthlyUSD = %v, want 0 (wave 1 meters only real usage)", s.EstimatedMonthlyUSD)
	}
	if s.Basis == "" {
		t.Fatal("Basis must explain the $0 attribution")
	}
}
```

Adjust the GraphQL fixture nesting to match however `Workers()` actually decodes its response (read `analytics.go` — the existing tests in `analytics_test.go` show the exact response shape the mock must emit; copy that shape verbatim rather than inventing one). Same for the workers-list REST path — confirm against `worker_test.go` or the cloudflare-go script-list endpoint path.

- [ ] **Step 6: Run to verify red**

Run: `go test ./pkg/cosmoflare/ -run 'TestEfficiency|TestSummarizeWaste' -v`
Expected: FAIL — `undefined: NewEfficiencyService`.

- [ ] **Step 7: Implement `efficiency.go`** — the full type set from the Interfaces block. Shape of `Report`:

```go
func (s *EfficiencyService) Report(ctx context.Context, w AnalyticsWindow) (*EfficiencyReport, error) {
	rep := &EfficiencyReport{WindowStart: w.Start, WindowEnd: w.End}

	// Workers: listed scripts × per-script request sums. Absent from the
	// analytics rows or Requests==0 → idle.
	if s.workers != nil {
		scripts, err := s.workers.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list workers: %w", err)
		}
		sums, err := s.analytics.Workers(ctx, w)
		if err != nil {
			return nil, fmt.Errorf("workers analytics: %w", err)
		}
		byScript := map[string]uint64{}
		for _, sm := range sums {
			byScript[sm.Script] = sm.Requests
		}
		rep.WorkersChecked = len(scripts)
		for _, sc := range scripts {
			if byScript[sc.Name] == 0 {
				rep.Findings = append(rep.Findings, EfficiencyFinding{
					Resource: "worker:" + sc.Name,
					Check:    "idle-worker",
					Severity: "warning",
					Evidence: fmt.Sprintf("0 requests between %s and %s", w.Start.Format("2006-01-02"), w.End.Format("2006-01-02")),
					Fix:      fmt.Sprintf("delete it (cosmoflare worker delete %s) or remove its routes if retired", sc.Name),
				})
			}
		}
	}

	// KV: same absent-or-zero rule over reads+writes.
	// R2: buckets (when a BucketLister is wired) absent from R2Storage rows
	// or with ObjectCount==0 → empty.
	// (implement both following the worker block; KV fix text should point at
	//  `cosmoflare kv namespace delete <id>` and note keys survive until deleted)
	return rep, nil
}
```

`SummarizeWaste` counts findings by Check and sets `Basis` to a fixed sentence: `"Idle request-priced resources meter $0 directly; per-namespace KV storage attribution lands with wave-2 KV storage analytics."` Constructors validate creds the way `NewCostService` does (`NewEfficiencyServiceFromCreds` errors on empty accountID/token).

- [ ] **Step 8: Run to verify green, then the full package**

Run: `go test ./pkg/cosmoflare/ -run 'TestEfficiency|TestSummarizeWaste|TestAnalyticsKVOperations' -v && go vet ./pkg/cosmoflare/`
Expected: all PASS, vet clean.

- [ ] **Step 9: Commit**

```bash
git add pkg/cosmoflare/efficiency.go pkg/cosmoflare/efficiency_test.go pkg/cosmoflare/analytics.go pkg/cosmoflare/analytics_test.go
git commit -m "feat(efficiency): EfficiencyService — idle workers, dead KV namespaces, empty R2 buckets from real analytics"
```

---

## Task 2: Command — `cmd/efficiency.go`

**Model for dispatch:** sonnet (same agent as Task 1 — same worktree, sequential)

**Files:**
- Create: `cmd/efficiency.go`
- Create: `cmd/efficiency_test.go`

**Interfaces:**
- Consumes: everything Task 1 produced; cmd globals `AccountID`, `APIToken`; R2 client construction (see `cmd/status.go:234 getR2ClientForStatus` — reuse that helper or its pattern); `rootCmd.AddCommand`.
- Produces: registered `efficiency` command; package vars `efficiencyJSON bool`, `efficiencyJSONOut`, `efficiencyTextOut io.Writer`, `efficiencyCollectFn` (the seam Task 3's pattern mirrors).

- [ ] **Step 1: Write the failing command tests** (`cmd/efficiency_test.go`) — stub the seam like `usage_test.go` does:

```go
package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func stubEfficiency(t *testing.T, rep *cosmoflare.EfficiencyReport) {
	t.Helper()
	prev := efficiencyCollectFn
	efficiencyCollectFn = func(windowDays int, now time.Time) (*cosmoflare.EfficiencyReport, error) {
		return rep, nil
	}
	t.Cleanup(func() { efficiencyCollectFn = prev })
}

func effSampleReport() *cosmoflare.EfficiencyReport {
	return &cosmoflare.EfficiencyReport{
		WindowStart: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		WorkersChecked: 5, KVNamespacesChecked: 3, BucketsChecked: 4,
		Findings: []cosmoflare.EfficiencyFinding{
			{Resource: "worker:ghost", Check: "idle-worker", Severity: "warning",
				Evidence: "0 requests between 2026-10-07 and 2026-10-14",
				Fix:      "delete it (cosmoflare worker delete ghost)"},
			{Resource: "kv:ns-old", Check: "dead-kv-namespace", Severity: "warning",
				Evidence: "0 reads, 0 writes between 2026-10-07 and 2026-10-14",
				Fix:      "cosmoflare kv namespace delete ns-old"},
			{Resource: "r2:void", Check: "empty-r2-bucket", Severity: "info",
				Evidence: "0 objects at 2026-10-14", Fix: "cosmoflare bucket delete void"},
		},
	}
}

func TestEfficiencyIdleWorkers(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "worker:ghost") || !strings.Contains(out.String(), "idle-worker") {
		t.Fatalf("table lacks idle worker finding:\n%s", out.String())
	}
}

func TestEfficiencyEmptyKV(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "kv:ns-old") || !strings.Contains(out.String(), "dead-kv-namespace") {
		t.Fatalf("table lacks dead KV finding:\n%s", out.String())
	}
}

func TestEfficiencyJSONOutput(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyJSONOut = &out
	prevJSON := efficiencyJSON
	efficiencyJSON = true
	t.Cleanup(func() { efficiencyJSONOut = os.Stdout; efficiencyJSON = prevJSON })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	var env struct {
		Status string                        `json:"status"`
		Data   cosmoflare.EfficiencyReport   `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("envelope not JSON: %v\n%s", err, out.String())
	}
	if env.Status != "success" || len(env.Data.Findings) != 3 {
		t.Fatalf("envelope = %+v", env)
	}
}

func TestEfficiencyEmptyAccount(t *testing.T) {
	stubEfficiency(t, &cosmoflare.EfficiencyReport{
		WindowStart: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		WorkersChecked: 2, KVNamespacesChecked: 1, BucketsChecked: 1,
	})
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "No waste found") {
		t.Fatalf("empty account must print a clean bill:\n%s", out.String())
	}
}

func TestEfficiencyMissingCredentials(t *testing.T) {
	prev := efficiencyCollectFn
	efficiencyCollectFn = nil // force reset then restore production seam is awkward; instead:
	_ = prev
}
```

Drop `TestEfficiencyMissingCredentials` (not in the spec's test list) and instead make the production `efficiencyCollectFn` return the credential error string from `usageCollectFn`'s precedent — assert it via a stub that returns that error if you want, but the four named tests above are the required set. Also mirror `usage_test.go`'s exact cleanup idiom (`os.Stdout` restore) — check how usage_test.go restores writers and copy it.

- [ ] **Step 2: Run to verify red**

Run: `go test ./cmd/ -run TestEfficiency -v`
Expected: FAIL — `undefined: efficiencyCollectFn` / `runEfficiency`.

- [ ] **Step 3: Implement `cmd/efficiency.go`** — mirror `cmd/usage.go` line-for-line in structure:

```go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	efficiencyWindowDays int
	efficiencyJSON       bool
)

var (
	efficiencyJSONOut io.Writer = os.Stdout
	efficiencyTextOut io.Writer = os.Stdout
)

var efficiencyCollectFn = func(windowDays int, now time.Time) (*cosmoflare.EfficiencyReport, error) {
	if AccountID == "" || APIToken == "" {
		return nil, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	workers, err := cosmoflare.NewWorkerServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}
	kv, err := cosmoflare.NewKVServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV service: %w", err)
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	w := cosmoflare.AnalyticsWindow{Start: now.Add(-time.Duration(windowDays) * 24 * time.Hour).UTC(), End: now.UTC()}
	r2, r2Err := getR2ClientForStatus()
	if r2Err != nil {
		// No R2 client → skip empty-bucket checks; workers+KV still audited.
		return cosmoflare.NewEfficiencyService(workers, kv, analytics).Report(context.Background(), w)
	}
	return cosmoflare.NewEfficiencyServiceWithBuckets(workers, kv, analytics, r2).Report(context.Background(), w)
}

var efficiencyCmd = &cobra.Command{
	Use:   "efficiency",
	Short: "Find idle Cloudflare resources wasting your account",
	Long: `Find idle Cloudflare resources wasting your account.

Checks the whole account against real Cloudflare Analytics numbers and
reports resources that do nothing:

  idle workers          0 requests over the window (per-script GraphQL sums)
  dead KV namespaces    0 reads AND 0 writes over the window
  empty R2 buckets      0 objects at window end

Every finding carries its evidence (counts and the window it was measured
over) and an actionable fix. Numbers are always measured from the API —
never estimated or modeled.

Flags:
  --window   Look-back window in days (default 7, max 90)
  --json     Machine-readable envelope

Examples:
  cosmoflare efficiency                # audit the account for waste
  cosmoflare efficiency --window 30    # a month of patience before calling something idle
  cosmoflare efficiency --json | jq '.data.findings[] | select(.check=="idle-worker")'`,
	RunE: runEfficiency,
}

func init() {
	rootCmd.AddCommand(efficiencyCmd)
	efficiencyCmd.Flags().IntVar(&efficiencyWindowDays, "window", 7, "Look-back window in days (1-90)")
	efficiencyCmd.Flags().BoolVar(&efficiencyJSON, "json", false, "Output JSON envelope")
}

func runEfficiency(cmd *cobra.Command, args []string) error {
	if efficiencyWindowDays < 1 || efficiencyWindowDays > 90 {
		return fmt.Errorf("--window must be 1-90 days, got %d", efficiencyWindowDays)
	}
	rep, err := efficiencyCollectFn(efficiencyWindowDays, time.Now())
	if err != nil {
		return err
	}
	if efficiencyJSON {
		enc := json.NewEncoder(efficiencyJSONOut)
		return enc.Encode(map[string]interface{}{"status": "success", "data": rep})
	}
	return writeEfficiencyTable(rep)
}
```

`writeEfficiencyTable`: header line with the window, one line per finding (`⚠/ℹ  RESOURCE  CHECK  EVIDENCE`), a footer with checked counts, and `No waste found` + checked counts when `len(Findings)==0`. If `getR2ClientForStatus` doesn't exist with that exact name, find the existing R2-client helper in `cmd/` (grep `NewClient(` in cmd/) and use it — the seam comment in the Long help stays accurate either way.

- [ ] **Step 4: Run to verify green + vet + full cmd suite**

Run: `go test ./cmd/ -run TestEfficiency -v && go vet ./... && go build ./...`
Expected: 4 named tests PASS, vet clean, build clean.

- [ ] **Step 5: Live smoke (real numbers rule)**

With the operator's configured credentials on this machine:

```bash
go build -o build/cosmoflare . && ./build/cosmoflare efficiency --json | head -40
./build/cosmoflare efficiency --window 7
```

Expected: JSON envelope with real per-resource counts. If the GraphQL endpoint rejects a `kvOperationsAdaptiveGroups` field name, the error names the field — fix the query string in `analytics.go`, re-run tests, re-smoke. Do NOT ship without one green live run.

- [ ] **Step 6: Commit**

```bash
git add cmd/efficiency.go cmd/efficiency_test.go
git commit -m "feat(cmd): cosmoflare efficiency — idle-resource findings from real analytics"
```

---

## Task 3: Cost wiring — idle-waste line in `cmd/cost.go`

**Model for dispatch:** glm-turbo (exact code below; zero design freedom)

**Files:**
- Modify: `cmd/cost.go`
- Modify: `cmd/cost_test.go`

**Interfaces:**
- Consumes: `cosmoflare.EfficiencyWasteSummary`, `cosmoflare.SummarizeWaste`, `cosmoflare.EfficiencyReport` (Task 1); `efficiencyCollectFn` (Task 2 — read-only reuse).
- Produces: `costWasteFn` seam; JSON gains a sibling `waste` key (backward compatible); table/CSV gain a waste line/row.

Sequencing: MUST run after Task 1+2 are merged (reuses both).

- [ ] **Step 1: Write the failing test** (append to `cmd/cost_test.go`):

```go
func TestCostIncludesIdleWaste(t *testing.T) {
	prev := costWasteFn
	costWasteFn = func() (*cosmoflare.EfficiencyWasteSummary, error) {
		return &cosmoflare.EfficiencyWasteSummary{
			IdleWorkers: 2, DeadKVNamespaces: 1, EmptyR2Buckets: 3,
			EstimatedMonthlyUSD: 0, Basis: "test basis",
		}, nil
	}
	t.Cleanup(func() { costWasteFn = prev })

	prevJSON, prevFmt := JSONOutput, costFormat
	JSONOutput, costFormat = true, "json"
	t.Cleanup(func() { JSONOutput, costFormat = prevJSON, prevFmt })

	var buf bytes.Buffer
	printJSONOut = &buf // add this writer seam beside printJSON, mirroring usageJSONOut
	t.Cleanup(func() { printJSONOut = os.Stdout })

	if err := runCost(nil, nil); err != nil {
		t.Fatalf("runCost: %v", err)
	}
	out := buf.String()
	for _, want := range []string{`"waste"`, `"idle_workers":2`, `"dead_kv_namespaces":1`, `"empty_r2_buckets":3`} {
		if !strings.Contains(out, want) {
			t.Fatalf("cost JSON missing %s:\n%s", want, out)
		}
	}
}
```

Note: `printJSON(data)` in `cmd/root.go` writes to os.Stdout directly. Add `var printJSONOut io.Writer = os.Stdout` in `cmd/cost.go` and have `runCost`'s JSON path encode to `printJSONOut` via `json.NewEncoder(printJSONOut).Encode(...)` instead of calling `printJSON` — only in runCost; other commands keep `printJSON`. (Adjust the test import block: bytes, os, strings already appear in cost_test.go or add them.)

- [ ] **Step 2: Run to verify red**

Run: `go test ./cmd/ -run TestCostIncludesIdleWaste -v`
Expected: FAIL — `undefined: costWasteFn`.

- [ ] **Step 3: Implement in `cmd/cost.go`:**

```go
// costWasteFn is the idle-waste seam: production collects an efficiency
// report ONLY when credentials exist and treats failure as "unavailable"
// (cost must never break because analytics did). Tests stub it.
var costWasteFn = func() (*cosmoflare.EfficiencyWasteSummary, error) {
	if AccountID == "" || APIToken == "" {
		return nil, nil // not configured — cost stays a pure estimator
	}
	rep, err := efficiencyCollectFn(7, time.Now())
	if err != nil {
		return nil, err
	}
	return cosmoflare.SummarizeWaste(rep), nil
}

type costReport struct {
	*cosmoflare.TotalCostEstimate
	Waste *cosmoflare.EfficiencyWasteSummary `json:"waste,omitempty"`
}
```

In `runCost` (and `runCostDetail`): after `total.Period = costPeriod`, call `costWasteFn()`; on error set `Waste = nil` (silently — best-effort, doctor enrichment precedent). JSON path encodes `costReport{TotalCostEstimate: total, Waste: waste}` to `printJSONOut`. Table path: after the TOTAL row and before the disclaimer, if `waste != nil` and any count > 0, print:

```
Waste: 2 idle workers, 1 dead KV namespace, 3 empty R2 buckets — $0.00/mo direct (run 'cosmoflare efficiency')
```

CSV path (`printCostCSV` only): append `waste` rows when waste != nil. Subcommands r2/workers/kv stay untouched.

- [ ] **Step 4: Run to verify green + full cost suite + vet**

Run: `go test ./cmd/ -run TestCost -v && go vet ./...`
Expected: all TestCost* PASS (including the existing 60), vet clean.

- [ ] **Step 5: Live smoke**

```bash
go build -o build/cosmoflare . && ./build/cosmoflare cost --json | jq '.waste' && ./build/cosmoflare cost
```
Expected: `waste` key present with real counts (or null if creds unset — both valid).

- [ ] **Step 6: Commit**

```bash
git add cmd/cost.go cmd/cost_test.go
git commit -m "feat(cost): idle-waste line from efficiency collectors — counts + honest $0 basis"
```

---

## Acceptance (maps to G-02/G-03 in the continuation prompt)

- `go test ./cmd/ -run TestEfficiency` green — 4 named tests exist and pass.
- `go test ./cmd/ -run TestCost` green — TestCostIncludesIdleWaste included.
- `go vet ./...` clean; `go build ./...` clean.
- One green live run of `cosmoflare efficiency --json` (real API numbers, no mocks).
- No changes outside: `cmd/efficiency.go`, `cmd/efficiency_test.go`, `pkg/cosmoflare/efficiency.go`, `pkg/cosmoflare/efficiency_test.go`, `pkg/cosmoflare/analytics.go`, `pkg/cosmoflare/analytics_test.go`, `cmd/cost.go`, `cmd/cost_test.go`, `cmd/root.go` (only if the printJSONOut seam lands there — prefer cost.go).
