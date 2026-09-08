---
title: 'Quota / Plan-Limit View — cosmoflare limits — Implementation Plan'
created: "2026-09-09T00:51:09+04:00"
status: DRAFT
tags: [plan, limits, control-plane, road-090]
brainstorm_ref: docs/brainstorming/2026-09-09-quota-limit-view.md
roadmap: ROAD-090
deliverables:
  - id: P-01
    title: "Static limit tables + limitFor pure function with table tests"
  - id: P-02
    title: "LimitsService types, constructor, consumer interfaces"
  - id: P-03
    title: "Workers plan resolution — subscriptions API + config/flag fallback"
  - id: P-04
    title: "DNS usage endpoint + ZonePlan.LegacyID extension"
  - id: P-05
    title: "Snapshot assembly with partial-failure semantics"
  - id: P-06
    title: "ProjectConfig workers_plan field"
  - id: P-07
    title: "cmd/limits.go CLI — table, JSON, flags, exit codes"
  - id: P-08
    title: "Alert-evaluator feed — EvalMetrics fields, conditions, serve wiring"
  - id: P-09
    title: "Documentation — docs/USAGE.md limits section"
---

# Quota / Plan-Limit View — `cosmoflare limits` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `cosmoflare limits` — a plan-limit proximity view joining live usage counts against documented Cloudflare limit tables, with a live DNS quota API, partial-failure semantics, and an alert-evaluator feed.

**Architecture:** New `LimitsService` in `pkg/cosmoflare/limits.go` (BucketDomainService pattern: option-constructor, raw REST, injectable HTTP client). It composes existing services through narrow consumer interfaces, fetches each source independently (ROAD-090 producer doctrine: per-source errors, never a blank snapshot), and joins against embedded static tables via the pure function `limitFor`. `cmd/limits.go` renders; `internal/webhook` gains limit metrics for the alert loop.

**Tech Stack:** Go 1.26, cobra (cmd), net/http + httptest (tests), existing cloudflare-go v0.116 wrapper services.

**Design spec:** `docs/brainstorming/2026-09-09-quota-limit-view.md` (limit table values, decision log, semantics).

**Test commands** (repo root):
```bash
go test ./pkg/cosmoflare/ -run 'TestLimitFor|TestDNSRecordsStaticLimit|TestNewLimitsService|TestResolveWorkersPlan|TestDNSUsage|TestSnapshot' -v  # library
go test ./cmd/ -run 'TestLimits|TestSortRows' -v  # CLI
go test ./internal/webhook/ -run 'TestConditionValueLimit|TestCollectLimit' -v
go build ./... && go vet ./...
```

Note: no library test name contains the substring `TestLimits` — the library tests are `TestLimitFor`, `TestSnapshot`, etc., so `-run TestLimits` would match zero tests and report a false green.

---

### Task 1: Static limit tables + `limitFor` pure function

**Files:**
- Create: `pkg/cosmoflare/limits.go`
- Create: `pkg/cosmoflare/limits_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/cosmoflare/limits_test.go`:

```go
package cosmoflare

import "testing"

func TestLimitFor(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		plan     string
		want     uint64
		wantOK   bool
	}{
		{"workers scripts free", "workers.scripts", "free", 100, true},
		{"workers scripts paid", "workers.scripts", "paid", 500, true},
		{"workers daily requests free", "workers.daily_requests", "free", 100000, true},
		{"workers daily requests paid is unlimited", "workers.daily_requests", "paid", 0, true},
		{"r2 buckets plan-independent", "r2.buckets", "", 1000000, true},
		{"r2 custom domains plan-independent", "r2.custom_domains_per_bucket", "paid", 100, true},
		{"workers resource with unknown plan", "workers.scripts", "unknown", 0, false},
		{"unknown resource", "nope", "free", 0, false},
		{"workers resource with empty plan", "workers.scripts", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := limitFor(tt.resource, tt.plan)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("limitFor(%q, %q) = (%d, %v), want (%d, %v)",
					tt.resource, tt.plan, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDNSRecordsStaticLimit(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		created string // ISO date; empty = zero time
		want    uint64
		wantOK  bool
	}{
		{"pro", "pro", "", 3500, true},
		{"business", "business", "", 3500, true},
		{"free before cutoff", "free", "2024-01-15", 1000, true},
		{"free on cutoff", "free", "2024-09-01", 200, true},
		{"free after cutoff", "free", "2025-06-01", 200, true},
		{"enterprise has no per-zone limit", "enterprise", "", 0, false},
		{"unknown plan", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := time.Time{}
			if tt.created != "" {
				parsed, err := time.Parse("2006-01-02", tt.created)
				if err != nil {
					t.Fatalf("bad fixture date %q: %v", tt.created, err)
				}
				created = parsed
			}
			got, ok := dnsRecordsStaticLimit(tt.plan, created)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("dnsRecordsStaticLimit(%q, %v) = (%d, %v), want (%d, %v)",
					tt.plan, created, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
```

Add `"time"` to the test imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run 'TestLimitFor|TestDNSRecordsStaticLimit' -v`
Expected: FAIL — `undefined: limitFor`, `undefined: dnsRecordsStaticLimit`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/limits.go`:

```go
package cosmoflare

import "time"

// workerPlanLimits maps plan-dependent resources to their per-tier limits.
// Limit 0 means "unlimited". Sources (verified 2026-09-09):
//   - https://developers.cloudflare.com/workers/platform/limits/
var workerPlanLimits = map[string]map[string]uint64{
	"workers.scripts":         {"free": 100, "paid": 500},
	"workers.daily_requests":  {"free": 100000, "paid": 0},
}

// staticLimits maps plan-independent account resources to documented limits.
// Source (verified 2026-09-09):
//   - https://developers.cloudflare.com/r2/platform/limits/
var staticLimits = map[string]uint64{
	"r2.buckets":                     1000000,
	"r2.custom_domains_per_bucket":   100,
}

// dnsFreeZoneCutoff splits Free-zone DNS record quotas: zones created on or
// after this UTC date get 200 records, older ones keep 1,000.
// Source: https://developers.cloudflare.com/dns/manage-dns-records/
var dnsFreeZoneCutoff = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// limitFor returns the documented limit for a resource given the Workers plan
// tier ("free" or "paid"). ok=false when the resource is unknown or the plan
// is required but unresolved. A returned limit of 0 with ok=true means
// "unlimited".
func limitFor(resource, plan string) (limit uint64, ok bool) {
	if tiers, exists := workerPlanLimits[resource]; exists {
		limit, ok = tiers[plan]
		return limit, ok
	}
	limit, ok = staticLimits[resource]
	return limit, ok
}

// dnsRecordsStaticLimit returns the per-zone DNS record quota by zone plan
// tier. Used only as the fallback when the live DNS usage API is unavailable
// to the token. Enterprise has no per-zone limit (account-level quota), so
// ok=false there.
func dnsRecordsStaticLimit(zonePlan string, createdOn time.Time) (limit uint64, ok bool) {
	switch zonePlan {
	case "pro", "business":
		return 3500, true
	case "free":
		if createdOn.Before(dnsFreeZoneCutoff) {
			return 1000, true
		}
		return 200, true
	default: // enterprise or unknown
		return 0, false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestLimitFor|TestDNSRecordsStaticLimit' -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/limits.go pkg/cosmoflare/limits_test.go
git commit -m "feat(limits): static limit tables and limitFor join function"
```

---

### Task 2: Snapshot types, consumer interfaces, LimitsService constructor

**Files:**
- Modify: `pkg/cosmoflare/limits.go`
- Modify: `pkg/cosmoflare/limits_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/limits_test.go`:

```go
func TestNewLimitsServiceDefaults(t *testing.T) {
	s := NewLimitsService("acct", "tok")
	if s.accountID != "acct" || s.apiToken != "tok" {
		t.Fatal("credentials not stored")
	}
	if s.httpClient == nil {
		t.Fatal("default HTTP client missing")
	}
	if s.baseURL != "https://api.cloudflare.com/client/v4" {
		t.Fatalf("baseURL = %q", s.baseURL)
	}
}

func TestNewLimitsServiceValidation(t *testing.T) {
	if _, err := NewLimitsService("", "tok").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty account ID must fail validation")
	} else if !strings.Contains(err.Error(), "account ID is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewLimitsService("acct", "").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty token must fail validation")
	} else if !strings.Contains(err.Error(), "API token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Add `"context"` and `"strings"` to imports. (Snapshot arrives in Step 3 with only validation — fetchers are nil, so validation fires first by construction: validate before any fetch.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run 'TestNewLimitsService' -v`
Expected: FAIL — `undefined: NewLimitsService`.

- [ ] **Step 3: Write minimal implementation**

Append to `pkg/cosmoflare/limits.go` (merge imports: add `"context"`, `"net/http"`):

```go
// LimitRow is one usage-vs-limit observation.
type LimitRow struct {
	Resource    string  `json:"resource"`              // "workers.scripts", "dns.records", ...
	Scope       string  `json:"scope,omitempty"`       // "" account-wide, or bucket/zone name
	Used        uint64  `json:"used"`
	Limit       uint64  `json:"limit"`                 // 0 = unlimited
	Percent     float64 `json:"percent,omitempty"`     // 0 when Limit == 0 or unknown
	PlanTier    string  `json:"plan_tier,omitempty"`   // "free" | "paid" | "unknown" | "" when not plan-dependent
	LimitSource string  `json:"limit_source"`          // "static-docs" | "live-api" | "unknown"
}

// SourceError records one failed producer without failing the snapshot.
type SourceError struct {
	Source string `json:"source"` // "workers.list", "subscriptions", "dns.usage:<zone>", ...
	Err    string `json:"err"`
}

// LimitsSnapshot is the partial-failure result of one limits collection pass.
type LimitsSnapshot struct {
	Rows        []LimitRow    `json:"rows"`
	Sources     []SourceError `json:"sources,omitempty"`
	WorkersPlan string        `json:"workers_plan"`  // "free" | "paid" | "unknown"
	PlanSource  string        `json:"plan_source"`   // "auto" | "config" | "flag" | "unknown"
}

// Consumer interfaces: LimitsService composes existing services through the
// narrow surface it needs. Any service satisfying the signature works,
// including test fakes.
type WorkersLister interface {
	List(ctx context.Context) ([]*Worker, error)
}

type BucketLister interface {
	ListBuckets(ctx context.Context) ([]*Bucket, error)
}

type ZoneLister interface {
	List(ctx context.Context) ([]*Zone, error)
}

type BucketDomainLister interface {
	List(ctx context.Context, bucket string) ([]BucketDomain, error)
}

type WorkersAnalytics interface {
	Workers(ctx context.Context, w AnalyticsWindow) ([]WorkersSummary, error)
}

// LimitsService joins live usage counts against documented plan limits.
type LimitsService struct {
	accountID  string
	apiToken   string
	httpClient *http.Client
	baseURL    string
	workers    WorkersLister
	r2         BucketLister
	zones      ZoneLister
	domains    BucketDomainLister
	analytics  WorkersAnalytics
	configPlan string
	flagPlan   string
}

// LimitsOption configures the LimitsService.
type LimitsOption func(*LimitsService)

// WithLimitsHTTPClient sets a custom HTTP client.
func WithLimitsHTTPClient(c *http.Client) LimitsOption {
	return func(s *LimitsService) { s.httpClient = c }
}

// WithLimitsBaseURL overrides the REST API base URL.
func WithLimitsBaseURL(u string) LimitsOption {
	return func(s *LimitsService) { s.baseURL = u }
}

// WithLimitsWorkers sets the Workers script lister.
func WithLimitsWorkers(w WorkersLister) LimitsOption {
	return func(s *LimitsService) { s.workers = w }
}

// WithLimitsR2 sets the R2 bucket lister.
func WithLimitsR2(r BucketLister) LimitsOption {
	return func(s *LimitsService) { s.r2 = r }
}

// WithLimitsZones sets the zone lister.
func WithLimitsZones(z ZoneLister) LimitsOption {
	return func(s *LimitsService) { s.zones = z }
}

// WithLimitsDomains sets the per-bucket custom domain lister.
func WithLimitsDomains(d BucketDomainLister) LimitsOption {
	return func(s *LimitsService) { s.domains = d }
}

// WithLimitsAnalytics sets the Workers analytics source (daily requests).
func WithLimitsAnalytics(a WorkersAnalytics) LimitsOption {
	return func(s *LimitsService) { s.analytics = a }
}

// WithLimitsConfigPlan supplies the workers_plan value from project config.
func WithLimitsConfigPlan(plan string) LimitsOption {
	return func(s *LimitsService) { s.configPlan = plan }
}

// WithLimitsFlagPlan supplies the workers_plan value from the --plan flag.
// The flag outranks config.
func WithLimitsFlagPlan(plan string) LimitsOption {
	return func(s *LimitsService) { s.flagPlan = plan }
}

// NewLimitsService creates a limits collector for one account.
func NewLimitsService(accountID, apiToken string, opts ...LimitsOption) *LimitsService {
	s := &LimitsService{
		accountID:  accountID,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// validate ensures account-scoped collection has credentials.
func (s *LimitsService) validate(op string) error {
	if s.accountID == "" {
		return validationError(op, "account ID is required")
	}
	if s.apiToken == "" {
		return validationError(op, "API token is required")
	}
	return nil
}
```

And a minimal `Snapshot` stub so the validation test compiles (the full assembly is Task 5):

```go
// Snapshot collects every configured source and joins usage against limits.
// bucket selects the optional per-bucket rows; empty skips them.
func (s *LimitsService) Snapshot(ctx context.Context, bucket string) (*LimitsSnapshot, error) {
	const op = "LimitsSnapshot"
	if err := s.validate(op); err != nil {
		return nil, err
	}
	return &LimitsSnapshot{WorkersPlan: "unknown", PlanSource: "unknown"}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestNewLimitsService|TestLimitFor' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/limits.go pkg/cosmoflare/limits_test.go
git commit -m "feat(limits): snapshot types, consumer interfaces, service constructor"
```

---

### Task 3: Workers plan resolution — subscriptions API + fallback chain

**Files:**
- Modify: `pkg/cosmoflare/limits.go`
- Modify: `pkg/cosmoflare/limits_test.go`

Resolution order: subscriptions API → `--plan` flag → `workers_plan` config → unknown. This inverts brainstorm design decision 2, which listed config before flag — as ordered there, a config file would permanently outrank the flag and `--plan` could never override anything. The code and tests below implement flag-over-config deliberately. A subscriptions 403/404 (scoped token without Billing Read) is a fallback, never an error surfaced to the user.

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/limits_test.go`:

```go
func testLimitServer(t *testing.T, handler http.HandlerFunc) (*LimitsService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewLimitsService("acct", "tok", WithLimitsBaseURL(srv.URL)), srv
}

func TestResolveWorkersPlanAuto(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct/subscriptions" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "cdn"}, "rate_plan": {"id": "cdn_pro"}},
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_paid", "public_name": "Workers Paid"}}
		]}`)
	})
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != "paid" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (paid, auto)", plan, source)
	}
}

func TestResolveWorkersPlanAutoFree(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_free"}}
		]}`)
	})
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != "free" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (free, auto)", plan, source)
	}
}

func TestResolveWorkersPlanFallbackChain(t *testing.T) {
	// Subscriptions endpoint 403s (scoped token) → config → flag ordering.
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success": false}`)
	})

	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "unknown" || source != "unknown" {
		t.Fatalf("no fallbacks: got (%q, %q, %v), want (unknown, unknown, nil)", plan, source, err)
	}

	s.configPlan = "paid"
	plan, source, err = s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "paid" || source != "config" {
		t.Fatalf("config fallback: got (%q, %q, %v), want (paid, config, nil)", plan, source, err)
	}

	s.flagPlan = "free"
	plan, source, err = s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "free" || source != "flag" {
		t.Fatalf("flag outranks config: got (%q, %q, %v), want (free, flag, nil)", plan, source, err)
	}
}

func TestResolveWorkersPlanInvalidValues(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	s.configPlan = "enterprise" // not a Workers tier — must be rejected, not trusted
	plan, source, err := s.resolveWorkersPlan(context.Background())
	if err != nil || plan != "unknown" || source != "unknown" {
		t.Fatalf("invalid config plan: got (%q, %q, %v), want (unknown, unknown, nil)", plan, source, err)
	}
}
```

Add `"fmt"`, `"net/http"`, `"net/http/httptest"` to the test imports (`"strings"` is needed only by the library, not the tests).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestResolveWorkersPlan -v`
Expected: FAIL — `s.resolveWorkersPlan undefined`.

- [ ] **Step 3: Write minimal implementation**

Append to `pkg/cosmoflare/limits.go` (merge imports: add `"encoding/json"`, `"io"`, `"strings"`):

```go
// subscription wraps the parts of a /subscriptions result entry we join on.
type subscription struct {
	Product struct {
		Name string `json:"name"`
	} `json:"product"`
	RatePlan struct {
		ID         string `json:"id"`
		PublicName string `json:"public_name"`
	} `json:"rate_plan"`
}

// fetchSubscriptions lists account subscriptions. It requires Billing Read;
// callers treat any failure as "auto-detection unavailable".
func (s *LimitsService) fetchSubscriptions(ctx context.Context) ([]subscription, error) {
	const op = "LimitsSubscriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/accounts/"+s.accountID+"/subscriptions", nil)
	if err != nil {
		return nil, newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, newError(op, "request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newError(op, "failed to read response body", err)
	}
	var env struct {
		Success bool           `json:"success"`
		Errors  []struct{ Message string `json:"message"` } `json:"errors"`
		Result  []subscription `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, newError(op, fmt.Sprintf("unexpected response (HTTP %d)", resp.StatusCode), err)
	}
	if !env.Success {
		msg := "subscriptions unavailable"
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		return nil, newError(op, msg, nil)
	}
	return env.Result, nil
}

// normalizePlanTier validates a tier string from any source. Only "free" and
// "paid" are Workers plan tiers; anything else is "unknown".
func normalizePlanTier(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "free":
		return "free"
	case "paid":
		return "paid"
	default:
		return "unknown"
	}
}

// resolveWorkersPlan resolves the Workers plan tier. Order: subscriptions
// API → config → flag → unknown. A subscriptions failure is silent — the
// source string records which path decided.
func (s *LimitsService) resolveWorkersPlan(ctx context.Context) (plan, source string, err error) {
	subs, err := s.fetchSubscriptions(ctx)
	if err == nil {
		for _, sub := range subs {
			id := sub.RatePlan.ID
			if strings.HasPrefix(id, "workers") {
				if strings.Contains(id, "paid") || strings.Contains(strings.ToLower(sub.RatePlan.PublicName), "paid") {
					return "paid", "auto", nil
				}
				return "free", "auto", nil
			}
		}
		// Subscriptions readable but no workers entry: fall through to config.
	}
	if tier := normalizePlanTier(s.flagPlan); tier != "unknown" {
		return tier, "flag", nil
	}
	if tier := normalizePlanTier(s.configPlan); tier != "unknown" {
		return tier, "config", nil
	}
	return "unknown", "unknown", nil
}
```

Note: flag outranks config (a one-shot CLI override beats a stale file). `fmt` joins the imports from Task 2's context addition.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestResolveWorkersPlan -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/limits.go pkg/cosmoflare/limits_test.go
git commit -m "feat(limits): workers plan resolution with subscriptions API and fallback chain"
```

---

### Task 4: DNS usage endpoint + ZonePlan.LegacyID extension

**Files:**
- Modify: `pkg/cosmoflare/zone.go:26-30` (ZonePlan struct) and `pkg/cosmoflare/zone.go:188` (cfZoneToZone mapping)
- Modify: `pkg/cosmoflare/limits.go`
- Modify: `pkg/cosmoflare/limits_test.go`

The live endpoint is `GET /zones/{zone_id}/dns/usage` (docs portal: `dns/subresources/usage/subresources/zone`). Field names are pinned defensively: the parser accepts `quota` or `max_records` for the limit and `used` or `records_used` for usage. Any HTTP failure falls back to the static per-plan table via `dnsRecordsStaticLimit` (Task 1), which needs the zone plan tier — hence the `LegacyID` extension (cloudflare-go already carries it; we currently drop it).

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/limits_test.go`:

```go
func TestDNSUsageLive(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/z1/dns/usage" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": {"used": 180, "quota": 200}}`)
	})
	used, limit, source, err := s.dnsUsage(context.Background(), "z1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 180 || limit != 200 || source != "live-api" {
		t.Fatalf("dnsUsage = (%d, %d, %q), want (180, 200, live-api)", used, limit, source)
	}
}

func TestDNSUsageAlternateFieldNames(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": {"records_used": 10, "max_records": 3500}}`)
	})
	used, limit, source, err := s.dnsUsage(context.Background(), "z1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 10 || limit != 3500 || source != "live-api" {
		t.Fatalf("dnsUsage = (%d, %d, %q), want (10, 3500, live-api)", used, limit, source)
	}
}

func TestDNSUsageFallbackToStatic(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success": false}`)
	})
	// Zone created 2025-03-01 on free plan → 200 by cutoff rule.
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	used, limit, source, err := s.dnsUsageFallback(context.Background(), "z1", "free", created)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No live data → used falls back to a record count of 0 only when caller
	// passes it; here we assert the static limit and source.
	if limit != 200 || source != "static-docs" {
		t.Fatalf("dnsUsageFallback = (%d, %q, %v), want (200, static-docs, nil)", limit, source, err)
	}
	_ = used
}

func TestZonePlanLegacyID(t *testing.T) {
	z := Zone{Plan: ZonePlan{ID: "x", LegacyID: "pro", Name: "Pro"}}
	if z.Plan.LegacyID != "pro" {
		t.Fatalf("LegacyID = %q, want pro", z.Plan.LegacyID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run 'TestDNSUsage|TestZonePlanLegacyID' -v`
Expected: FAIL — `s.dnsUsage undefined`, `unknown field LegacyID`.

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/zone.go`, extend the struct and mapping:

```go
// ZonePlan contains the plan information for a zone.
type ZonePlan struct {
	ID       string `json:"id"`
	LegacyID string `json:"legacy_id,omitempty"` // "free" | "pro" | "business" | "enterprise"
	Name     string `json:"name"`
}
```

and in `cfZoneToZone` (around line 188) change the Plan mapping to:

```go
		Plan:        ZonePlan{ID: z.Plan.ID, LegacyID: z.Plan.LegacyID, Name: z.Plan.Name},
```

Append to `pkg/cosmoflare/limits.go`:

```go
// dnsUsage fetches one zone's DNS record usage and quota from the live API.
// Field names are parsed defensively ({used,records_used} × {quota,max_records}).
// Any failure returns the static fallback by zone plan tier.
func (s *LimitsService) dnsUsage(ctx context.Context, zoneID string) (used, limit uint64, source string, err error) {
	const op = "LimitsDNSUsage"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/zones/"+zoneID+"/dns/usage", nil)
	if err != nil {
		return 0, 0, "", newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, 0, "", newError(op, "request failed", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, "", newError(op, "failed to read response body", err)
	}
	var env struct {
		Success bool `json:"success"`
		Result  struct {
			Used         *uint64 `json:"used"`
			RecordsUsed  *uint64 `json:"records_used"`
			Quota        *uint64 `json:"quota"`
			MaxRecords   *uint64 `json:"max_records"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil || !env.Success {
		return 0, 0, "", newError(op, fmt.Sprintf("dns usage unavailable (HTTP %d)", resp.StatusCode), err)
	}
	r := env.Result
	switch {
	case r.Used != nil && (r.Quota != nil || r.MaxRecords != nil):
		used = *r.Used
	case r.RecordsUsed != nil && (r.Quota != nil || r.MaxRecords != nil):
		used = *r.RecordsUsed
	default:
		return 0, 0, "", newError(op, "dns usage response missing fields", nil)
	}
	if r.Quota != nil {
		limit = *r.Quota
	} else {
		limit = *r.MaxRecords
	}
	return used, limit, "live-api", nil
}

// dnsUsageFallback resolves the static per-plan limit when the live endpoint
// is unavailable. used comes from the caller (a record count) or 0.
func (s *LimitsService) dnsUsageFallback(ctx context.Context, zoneID, zonePlan string, createdOn time.Time) (used, limit uint64, source string, err error) {
	limit, ok := dnsRecordsStaticLimit(zonePlan, createdOn)
	if !ok {
		return 0, 0, "unknown", nil // enterprise: account-level quota, not per-zone
	}
	return 0, limit, "static-docs", nil
}
```

(The snapshot assembly in Task 5 passes a live record count into the fallback path when the API fails but DNS listing succeeds; `dnsUsageFallback` itself owns only the limit side, keeping it pure for tests.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestDNSUsage|TestZonePlanLegacyID' -v && go test ./pkg/cosmoflare/ -run TestZone -v`
Expected: PASS — new tests plus the existing zone suite (LegacyID addition is additive).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/limits.go pkg/cosmoflare/limits_test.go pkg/cosmoflare/zone.go
git commit -m "feat(limits): live DNS usage endpoint with static fallback, zone plan legacy_id"
```

---

### Task 5: Snapshot assembly with partial-failure semantics

**Files:**
- Modify: `pkg/cosmoflare/limits.go` (replace the Task 2 Snapshot stub)
- Modify: `pkg/cosmoflare/limits_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/limits_test.go`:

```go
type fakeWorkersLister struct{ n int; err error }
func (f fakeWorkersLister) List(ctx context.Context) ([]*Worker, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*Worker, f.n), nil
}

type fakeBucketLister struct{ n int; err error }
func (f fakeBucketLister) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*Bucket, f.n), nil
}

type fakeZoneLister struct{ zones []*Zone; err error }
func (f fakeZoneLister) List(ctx context.Context) ([]*Zone, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.zones, nil
}

type fakeDomainLister struct{ n int; err error }
func (f fakeDomainLister) List(ctx context.Context, bucket string) ([]BucketDomain, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]BucketDomain, f.n), nil
}

type fakeWorkersAnalytics struct{ sum []WorkersSummary; err error }
func (f fakeWorkersAnalytics) Workers(ctx context.Context, w AnalyticsWindow) ([]WorkersSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sum, nil
}

func snapshotFixture(t *testing.T, opts ...LimitsOption) *LimitsService {
	t.Helper()
	// Subscriptions: 403 → unknown plan path (tests must not depend on it).
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	s.configPlan = "paid"
	for _, o := range opts {
		o(s)
	}
	return s
}

func TestSnapshotHappyPath(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{n: 80}),
		WithLimitsR2(fakeBucketLister{n: 7}),
		WithLimitsZones(fakeZoneLister{zones: []*Zone{
			{ID: "z1", Name: "one.example", Plan: ZonePlan{LegacyID: "pro"}, CreatedOn: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}}),
		WithLimitsAnalytics(fakeWorkersAnalytics{sum: []WorkersSummary{{Requests: 42000}}}),
	)
	// DNS usage live endpoint serves 180/200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": {"used": 180, "quota": 200}}`)
	}))
	t.Cleanup(srv.Close)
	s.baseURL = srv.URL

	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.WorkersPlan != "paid" || snap.PlanSource != "config" {
		t.Fatalf("plan = (%q, %q), want (paid, config)", snap.WorkersPlan, snap.PlanSource)
	}

	byResource := map[string]LimitRow{}
	for _, row := range snap.Rows {
		byResource[row.Resource+"\x00"+row.Scope] = row
	}

	row := byResource["workers.scripts\x00"]
	if row.Used != 80 || row.Limit != 500 || row.Percent != 16 {
		t.Fatalf("workers.scripts = %+v, want used=80 limit=500 pct=16", row)
	}
	row = byResource["workers.daily_requests\x00"]
	if row.Used != 42000 || row.Limit != 0 || row.Percent != 0 {
		t.Fatalf("workers.daily_requests = %+v, want used=42000 limit=0(unlimited) pct=0", row)
	}
	row = byResource["r2.buckets\x00"]
	if row.Used != 7 || row.Limit != 1000000 {
		t.Fatalf("r2.buckets = %+v", row)
	}
	row = byResource["zones.count\x00"]
	if row.Used != 1 || row.Limit != 0 {
		t.Fatalf("zones.count = %+v, want informational used=1 limit=0", row)
	}
	row = byResource["dns.records\x00one.example"]
	if row.Used != 180 || row.Limit != 200 || row.Percent != 90 || row.LimitSource != "live-api" {
		t.Fatalf("dns.records = %+v, want 180/200=90%% live-api", row)
	}
	if len(snap.Sources) != 0 {
		t.Fatalf("unexpected source errors: %+v", snap.Sources)
	}
}

func TestSnapshotPartialFailure(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{err: errors.New("workers down")}),
		WithLimitsR2(fakeBucketLister{n: 3}),
		WithLimitsZones(fakeZoneLister{err: errors.New("zones down")}),
		WithLimitsAnalytics(fakeWorkersAnalytics{err: errors.New("analytics down")}),
	)
	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("partial failure must not fail the snapshot: %v", err)
	}
	if len(snap.Rows) == 0 {
		t.Fatal("healthy sources must still produce rows")
	}
	found := map[string]bool{}
	for _, se := range snap.Sources {
		found[se.Source] = true
	}
	if !found["workers.list"] || !found["zones.list"] || !found["analytics.workers"] {
		t.Fatalf("source errors = %+v, want workers.list, zones.list, analytics.workers", snap.Sources)
	}
}

func TestSnapshotAllSourcesFail(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{err: errors.New("x")}),
		WithLimitsR2(fakeBucketLister{err: errors.New("x")}),
		WithLimitsZones(fakeZoneLister{err: errors.New("x")}),
		WithLimitsAnalytics(fakeWorkersAnalytics{err: errors.New("x")}),
	)
	if _, err := s.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("all sources failing must return an error")
	}
}

func TestSnapshotBucketScope(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsR2(fakeBucketLister{n: 1}),
		WithLimitsDomains(fakeDomainLister{n: 99}),
	)
	snap, err := s.Snapshot(context.Background(), "my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range snap.Rows {
		if row.Resource == "r2.custom_domains_per_bucket" {
			if row.Scope != "my-bucket" || row.Used != 99 || row.Limit != 100 || row.Percent != 99 {
				t.Fatalf("bucket row = %+v, want my-bucket 99/100=99%%", row)
			}
			return
		}
	}
	t.Fatal("r2.custom_domains_per_bucket row missing with --bucket set")
}

func TestSnapshotDNSStaticFallback(t *testing.T) {
	// Live DNS endpoint fails (testLimitServer returns 403 for every path);
	// zone is free + created 2025 → static 200.
	s := snapshotFixture(t,
		WithLimitsZones(fakeZoneLister{zones: []*Zone{
			{ID: "z1", Name: "new.example", Plan: ZonePlan{LegacyID: "free"}, CreatedOn: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)},
		}}),
	)
	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range snap.Rows {
		if row.Resource == "dns.records" && row.Scope == "new.example" {
			if row.Limit != 200 || row.LimitSource != "static-docs" {
				t.Fatalf("dns fallback row = %+v, want limit=200 static-docs", row)
			}
			return
		}
	}
	t.Fatal("dns.records fallback row missing")
}
```

Add `"errors"` to the test imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestSnapshot -v`
Expected: FAIL — the stub returns an empty snapshot; rows missing.

- [ ] **Step 3: Write minimal implementation**

Replace the Task 2 `Snapshot` stub in `pkg/cosmoflare/limits.go`:

```go
// percentOf computes used/limit as a percentage rounded to one decimal.
// Returns 0 for unlimited or unknown limits.
func percentOf(used, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return math.Round(float64(used)/float64(limit)*1000) / 10
}

// Snapshot collects every configured source and joins usage against limits.
// bucket selects optional per-bucket rows; empty skips them. Every source is
// independent: a failure records a SourceError and the snapshot continues.
// Only zero rows + at least one error is a hard failure.
func (s *LimitsService) Snapshot(ctx context.Context, bucket string) (*LimitsSnapshot, error) {
	const op = "LimitsSnapshot"
	if err := s.validate(op); err != nil {
		return nil, err
	}

	snap := &LimitsSnapshot{}
	plan, source, _ := s.resolveWorkersPlan(ctx)
	snap.WorkersPlan, snap.PlanSource = plan, source

	// workers.scripts
	if s.workers != nil {
		scripts, err := s.workers.List(ctx)
		if err != nil {
			snap.recordSourceError("workers.list", err)
		} else if limit, ok := limitFor("workers.scripts", plan); ok {
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "workers.scripts", Used: uint64(len(scripts)), Limit: limit,
				Percent: percentOf(uint64(len(scripts)), limit), PlanTier: plan, LimitSource: "static-docs",
			})
		} else {
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "workers.scripts", Used: uint64(len(scripts)),
				PlanTier: plan, LimitSource: "unknown",
			})
		}
	}

	// workers.daily_requests (today UTC)
	if s.analytics != nil {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		summaries, err := s.analytics.Workers(ctx, AnalyticsWindow{Start: start, End: now})
		if err != nil {
			snap.recordSourceError("analytics.workers", err)
		} else {
			var reqs uint64
			for _, sum := range summaries {
				reqs += sum.Requests
			}
			row := LimitRow{Resource: "workers.daily_requests", Used: reqs, PlanTier: plan}
			if limit, ok := limitFor("workers.daily_requests", plan); ok {
				row.Limit, row.Percent, row.LimitSource = limit, percentOf(reqs, limit), "static-docs"
			} else {
				row.LimitSource = "unknown"
			}
			snap.Rows = append(snap.Rows, row)
		}
	}

	// r2.buckets
	if s.r2 != nil {
		buckets, err := s.r2.ListBuckets(ctx)
		if err != nil {
			snap.recordSourceError("r2.list_buckets", err)
		} else {
			limit, _ := limitFor("r2.buckets", "")
			n := uint64(len(buckets))
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "r2.buckets", Used: n, Limit: limit,
				Percent: percentOf(n, limit), LimitSource: "static-docs",
			})
		}
	}

	// r2.custom_domains_per_bucket (only with --bucket)
	if bucket != "" && s.domains != nil {
		domains, err := s.domains.List(ctx, bucket)
		if err != nil {
			snap.recordSourceError("r2.bucket_domains:"+bucket, err)
		} else {
			limit, _ := limitFor("r2.custom_domains_per_bucket", "")
			n := uint64(len(domains))
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "r2.custom_domains_per_bucket", Scope: bucket, Used: n, Limit: limit,
				Percent: percentOf(n, limit), LimitSource: "static-docs",
			})
		}
	}

	// zones.count + dns.records per zone
	if s.zones != nil {
		zones, err := s.zones.List(ctx)
		if err != nil {
			snap.recordSourceError("zones.list", err)
		} else {
			n := uint64(len(zones))
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "zones.count", Used: n, Limit: 0, LimitSource: "unknown", // no documented account cap
			})
			for _, z := range zones {
				used, limit, src, err := s.dnsUsage(ctx, z.ID)
				if err != nil {
					_, limit, src, _ = s.dnsUsageFallback(ctx, z.ID, z.Plan.LegacyID, z.CreatedOn)
					used = 0 // live usage unavailable; percent stays 0 — honest, not fabricated
				}
				if src == "unknown" {
					continue // enterprise: account-level quota, nothing per-zone to report
				}
				snap.Rows = append(snap.Rows, LimitRow{
					Resource: "dns.records", Scope: z.Name, Used: used, Limit: limit,
					Percent: percentOf(used, limit), LimitSource: src,
				})
			}
		}
	}

	if len(snap.Rows) == 0 && len(snap.Sources) > 0 {
		return nil, newError(op, "all limit sources failed", nil)
	}
	return snap, nil
}

// recordSourceError appends one producer failure.
func (s *LimitsSnapshot) recordSourceError(source string, err error) {
	s.Sources = append(s.Sources, SourceError{Source: source, Err: err.Error()})
}
```

Add `"math"` to the imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestSnapshot|TestLimits|TestNewLimitsService|TestResolveWorkersPlan|TestDNSUsage' -v`
Expected: PASS — full limits suite green.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/limits.go pkg/cosmoflare/limits_test.go
git commit -m "feat(limits): snapshot assembly with partial-failure semantics"
```

---

### Task 6: ProjectConfig workers_plan field

**Files:**
- Modify: `pkg/cosmoflare/config.go:12-23` (ProjectConfig struct)
- Modify: `pkg/cosmoflare/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/config_test.go` (follow that file's existing temp-dir fixture style):

```go
func TestProjectConfigWorkersPlan(t *testing.T) {
	dir := t.TempDir()
	yamlBody := "workers_plan: paid\n"
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte(yamlBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.WorkersPlan != "paid" {
		t.Fatalf("WorkersPlan = %q, want paid", cfg.WorkersPlan)
	}
}
```

Match the import set already used in `config_test.go` (`os`, `path/filepath` are almost certainly present; add only what the compiler asks for).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestProjectConfigWorkersPlan -v`
Expected: FAIL — `cfg.WorkersPlan undefined`.

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/config.go`, add one field to `ProjectConfig`:

```go
	Environment    string            `mapstructure:"env" json:"env,omitempty"`
	WorkersPlan    string            `mapstructure:"workers_plan" json:"workers_plan,omitempty"` // "free" | "paid" — fallback when the subscriptions API is unreadable
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestProjectConfig -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/config.go pkg/cosmoflare/config_test.go
git commit -m "feat(config): workers_plan project config field for limit tier fallback"
```

---

### Task 7: `cmd/limits.go` CLI

**Files:**
- Create: `cmd/limits.go`
- Create: `cmd/limits_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/limits_test.go` following the patterns in `cmd/analytics_test.go` and `cmd/status_test.go` (command registration and metadata assertions):

```go
package cmd

import (
	"testing"
)

// The CLI wiring test asserts flag registration and help surface. Service
// behavior is covered by pkg/cosmoflare tests; rendering helpers are pure and
// tested directly below.
func TestLimitsCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "limits" {
			found = true
		}
	}
	if !found {
		t.Fatal("limits command not registered on root")
	}
}

func TestSortRowsByPercent(t *testing.T) {
	rows := []cosmoflare.LimitRow{
		{Resource: "r2.buckets", Used: 1, Limit: 1000000, Percent: 0.1},
		{Resource: "dns.records", Scope: "a.io", Used: 90, Limit: 100, Percent: 90},
		{Resource: "workers.scripts", Used: 10, Limit: 100, Percent: 10},
	}
	got := sortRowsByPercent(rows)
	if got[0].Resource != "dns.records" || got[1].Resource != "workers.scripts" || got[2].Resource != "r2.buckets" {
		t.Fatalf("sort order wrong: %+v", got)
	}
}
```

Add the cosmoflare import (`cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"`). JSON rendering goes through `printSuccessJSON`, which existing cmd tests already cover; do not duplicate it here.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run 'TestLimits|TestSortRows' -v`
Expected: FAIL — `limits` not registered, `sortRowsByPercent undefined`.

- [ ] **Step 3: Write minimal implementation**

Create `cmd/limits.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	limitsBucket string
	limitsPlan   string
)

var limitsCmd = &cobra.Command{
	Use:   "limits",
	Short: "Show plan-limit proximity for the account",
	Long: `Show how close the account is to its Cloudflare plan limits.

Rows cover Workers (scripts, daily requests on the free plan), R2 (buckets,
per-bucket custom domains with --bucket), zones, and per-zone DNS record
quotas. Usage comes from live API counts; limit values come from documented
plan tables (workers limits, r2 limits) or the live DNS quota API.

Workers plan tier resolution order:
  1. Subscriptions API (auto — needs Billing Read)
  2. --plan flag (free|paid)
  3. workers_plan in .cosmoflare.yaml (config)
  4. unknown — plan-dependent rows show usage without a percent

Every source is independent: a failing source prints a warning after the
table and the rest of the snapshot is unaffected. Exit code is non-zero only
when every source fails.

Examples:
  cosmoflare limits
  cosmoflare limits --bucket=my-bucket
  cosmoflare limits --plan free --json`,
	RunE: runLimits,
}

func init() {
	rootCmd.AddCommand(limitsCmd)
	limitsCmd.Flags().StringVar(&limitsBucket, "bucket", "", "Include per-bucket rows for this bucket (custom domains)")
	limitsCmd.Flags().StringVar(&limitsPlan, "plan", "", "Workers plan tier override: free|paid")
}

// sortRowsByPercent orders rows by proximity to limit, descending. Rows
// without a percent sink to the end, keeping insertion order among equals.
func sortRowsByPercent(rows []cosmoflare.LimitRow) []cosmoflare.LimitRow {
	out := make([]cosmoflare.LimitRow, len(rows))
	copy(out, rows)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Percent > out[j].Percent })
	return out
}

func runLimits(cmd *cobra.Command, args []string) error {
	printInfo("Collecting plan limits...")

	cfg, cfgErr := cosmoflare.LoadProjectConfig(".")
	configPlan := ""
	if cfgErr == nil {
		configPlan = cfg.WorkersPlan
	}

	client, err := getAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	workersSvc, err := cosmoflare.NewWorkerServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return fmt.Errorf("failed to create workers service: %w", err)
	}
	zonesSvc, err := cosmoflare.NewZoneServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}
	domainsSvc := cosmoflare.NewBucketDomainService(AccountID, APIToken)
	analyticsSvc := cosmoflare.NewAnalyticsService(AccountID, APIToken)

	svc := cosmoflare.NewLimitsService(AccountID, APIToken,
		cosmoflare.WithLimitsR2(client),
		cosmoflare.WithLimitsWorkers(workersSvc),
		cosmoflare.WithLimitsZones(zonesSvc),
		cosmoflare.WithLimitsDomains(domainsSvc),
		cosmoflare.WithLimitsAnalytics(analyticsSvc),
		cosmoflare.WithLimitsConfigPlan(configPlan),
		cosmoflare.WithLimitsFlagPlan(limitsPlan),
	)

	snap, err := svc.Snapshot(context.Background(), limitsBucket)
	if err != nil {
		return fmt.Errorf("failed to collect limits: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Plan limits", snap)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RESOURCE\tSCOPE\tUSED\tLIMIT\tUSED%\tSOURCE")
	for _, row := range sortRowsByPercent(snap.Rows) {
		limit := "unlimited"
		if row.Limit > 0 {
			limit = fmt.Sprintf("%d", row.Limit)
		}
		pct := "—"
		if row.Percent > 0 {
			pct = fmt.Sprintf("%.1f%%", row.Percent)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			row.Resource, row.Scope, row.Used, limit, pct, row.LimitSource)
	}
	w.Flush()

	fmt.Printf("\nWorkers plan: %s (resolved via %s)\n", snap.WorkersPlan, snap.PlanSource)
	for _, se := range snap.Sources {
		printInfo("Source %s failed: %s", se.Source, se.Err)
	}
	if cfgErr != nil {
		printInfo("No project config found (%v) — workers_plan fallback unavailable", cfgErr)
	}
	return nil
}
```

Note: `NewWorkerServiceFromCreds` returns `(*WorkerService, error)` (worker.go:113); `NewZoneServiceFromCreds` likewise (zone.go:59); `NewBucketDomainService(accountID, apiToken)` returns the service directly (bucket_domain.go:53). If `getAPIClient()`'s R2Client already satisfies `BucketLister` (it declares `ListBuckets(ctx) ([]*Bucket, error)` at client.go:22), the interface accepts it with no adapter.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run 'TestLimits|TestSortRows' -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/limits.go cmd/limits_test.go
git commit -m "feat(cmd): cosmoflare limits — plan-limit proximity table and JSON"
```

---

### Task 8: Alert-evaluator feed

**Files:**
- Modify: `internal/webhook/evaluator.go` (EvalMetrics, conditionValue, metricData, new CollectLimitMetrics)
- Modify: `internal/webhook/evaluator_test.go`
- Modify: `cmd/serve.go:213-241` (runServeAlertEvalCycle)

- [ ] **Step 1: Write the failing test**

Append to `internal/webhook/evaluator_test.go` (follow its existing fixture style):

```go
func TestConditionValueLimitMetrics(t *testing.T) {
	m := EvalMetrics{WorkersScriptCount: 90, R2BucketCount: 5, DNSRecordQuotaPct: 95}

	if v, unit, ok := conditionValue("workers-script-count", m); !ok || v != 90 || unit != "scripts" {
		t.Fatalf("workers-script-count = (%v, %q, %v)", v, unit, ok)
	}
	if v, unit, ok := conditionValue("r2-bucket-count", m); !ok || v != 5 || unit != "buckets" {
		t.Fatalf("r2-bucket-count = (%v, %q, %v)", v, unit, ok)
	}
	if v, unit, ok := conditionValue("dns-record-quota", m); !ok || v != 95 || unit != "%" {
		t.Fatalf("dns-record-quota = (%v, %q, %v)", v, unit, ok)
	}

	// Zero DNS percent means "no quota rows" — the condition must not fire.
	if _, _, ok := conditionValue("dns-record-quota", EvalMetrics{}); ok {
		t.Fatal("dns-record-quota must be !ok with no quota rows")
	}
}

func TestCollectLimitMetrics(t *testing.T) {
	// All sources failing must error; partial success must fill fields.
	// Fixture: reuse the package's existing fake/httptest patterns. A limits
	// service with no listers configured returns a snapshot with only
	// plan-unknown rows when DNS is absent — use a real httptest server for
	// subscriptions 403 + fake listers via exported options.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	limits := cosmoflare.NewLimitsService("acct", "tok",
		cosmoflare.WithLimitsBaseURL(srv.URL),
		cosmoflare.WithLimitsWorkers(fakeWorkers{n: 42}),
		cosmoflare.WithLimitsR2(fakeBuckets{n: 3}),
	)

	var m EvalMetrics
	if err := CollectLimitMetrics(context.Background(), limits, &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.WorkersScriptCount != 42 || m.R2BucketCount != 3 {
		t.Fatalf("metrics = %+v, want 42 scripts / 3 buckets", m)
	}

	// All sources failing must error — zero rows + all sources errored is the
	// snapshot's hard-failure condition.
	failing := cosmoflare.NewLimitsService("acct", "tok",
		cosmoflare.WithLimitsBaseURL(srv.URL),
		cosmoflare.WithLimitsWorkers(fakeWorkers{err: errors.New("x")}),
		cosmoflare.WithLimitsR2(fakeBuckets{err: errors.New("x")}),
		cosmoflare.WithLimitsZones(fakeZones{err: errors.New("x")}),
		cosmoflare.WithLimitsAnalytics(fakeAnalytics{err: errors.New("x")}),
	)
	if err := CollectLimitMetrics(context.Background(), failing, &m); err == nil {
		t.Fatal("all limit sources failing must return an error")
	}
}
```

Define the tiny fakes in the same test file (or reuse existing ones if the file already has compatible fakes):

```go
type fakeWorkers struct {
	n   int
	err error
}

func (f fakeWorkers) List(ctx context.Context) ([]*cosmoflare.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*cosmoflare.Worker, f.n), nil
}

type fakeBuckets struct {
	n   int
	err error
}

func (f fakeBuckets) ListBuckets(ctx context.Context) ([]*cosmoflare.Bucket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*cosmoflare.Bucket, f.n), nil
}

type fakeZones struct{ err error }

func (f fakeZones) List(ctx context.Context) ([]*cosmoflare.Zone, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}

type fakeAnalytics struct{ err error }

func (f fakeAnalytics) Workers(ctx context.Context, w cosmoflare.AnalyticsWindow) ([]cosmoflare.WorkersSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}
```

Add `"fmt"` and `"errors"` to `evaluator_test.go` imports — the handler closures use `fmt.Fprint` and the all-fail case uses `errors.New`, and the file currently imports `context`, `net/http`, `net/http/httptest`, and `cosmoflare` but neither.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/webhook/ -run 'TestConditionValueLimit|TestCollectLimit' -v`
Expected: FAIL — unknown fields and function.

- [ ] **Step 3: Write minimal implementation**

In `internal/webhook/evaluator.go`:

Add fields to `EvalMetrics` (keep the existing five untouched):

```go
	WorkersScriptCount uint64  // live script count (LimitsService)
	R2BucketCount      uint64  // live bucket count (LimitsService)
	DNSRecordQuotaPct  float64 // max percent across per-zone dns.records rows (0 = no rows)
```

Add cases to `conditionValue` (before `default:`):

```go
	case "workers-script-count":
		return float64(m.WorkersScriptCount), "scripts", true
	case "r2-bucket-count":
		return float64(m.R2BucketCount), "buckets", true
	case "dns-record-quota":
		if m.DNSRecordQuotaPct == 0 {
			return 0, "%", false // no quota rows → nothing to judge
		}
		return m.DNSRecordQuotaPct, "%", true
```

Extend `metricData` with:

```go
		"workers_script_count": m.WorkersScriptCount,
		"r2_bucket_count":      m.R2BucketCount,
		"dns_record_quota_pct": m.DNSRecordQuotaPct,
```

Add the collector after `CollectEvalMetrics`:

```go
// CollectLimitMetrics augments m with limit-proximity metrics from the
// LimitsService. Only a total failure (zero rows collected) is an error —
// partial snapshots leave the untouched fields at zero and the evaluator
// skips conditions it cannot judge. DNSRecordQuotaPct is the MAXIMUM percent
// across per-zone rows so one hot zone fires the rule.
func CollectLimitMetrics(ctx context.Context, limits *cosmoflare.LimitsService, m *EvalMetrics) error {
	snap, err := limits.Snapshot(ctx, "")
	if err != nil {
		return fmt.Errorf("limits snapshot: %w", err)
	}
	for _, row := range snap.Rows {
		switch row.Resource {
		case "workers.scripts":
			m.WorkersScriptCount = row.Used
		case "r2.buckets":
			m.R2BucketCount = row.Used
		case "dns.records":
			if row.Percent > m.DNSRecordQuotaPct {
				m.DNSRecordQuotaPct = row.Percent
			}
		}
	}
	return nil
}
```

Wire the serve cycle in `cmd/serve.go` `runServeAlertEvalCycle` — after `CollectEvalMetrics` succeeds and BEFORE the `eval.Evaluate(metrics)` call, insert:

```go
	limits := cosmoflare.NewLimitsService(p.AccountID, p.APIToken)
	if err := webhook.CollectLimitMetrics(ctx, limits, &metrics); err != nil {
		// Limit collection failing must NOT skip the analytics-based rules.
		log.Printf("[alerts] collect limits: %v", err)
	}
```

(A limits failure logs and continues — the analytics metrics already collected still evaluate. The reverse order would let a limits outage blank every rule.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/webhook/ -v && go test ./cmd/ -run TestServeAlert -v && go build ./...`
Expected: PASS — new tests plus the existing alert-bridge suite.

- [ ] **Step 5: Commit**

```bash
git add internal/webhook/evaluator.go internal/webhook/evaluator_test.go cmd/serve.go
git commit -m "feat(alerts): limit-proximity metrics feed the evaluator (workers, buckets, dns quota)"
```

---

### Task 9: Documentation

**Files:**
- Modify: `docs/USAGE.md` (new `cosmoflare limits` section)

cosmoflare has no `READMEs/` tree — `docs/USAGE.md` is the single command reference, so this task extends it only and does not create a new documentation convention.

- [ ] **Step 1: Add the USAGE.md section**

Find the existing command sections in `docs/USAGE.md` (agent-reference format: synopsis, flags, examples, JSON shape). Add a `## cosmoflare limits` section with: purpose line, the row table from the design spec, flag reference (`--bucket`, `--plan`, `--json`), plan-resolution order, partial-failure semantics, and one full `--json` example output matching `LimitsSnapshot`:

```json
{
  "rows": [
    {"resource": "dns.records", "scope": "example.com", "used": 180, "limit": 200, "percent": 90.0, "limit_source": "live-api"},
    {"resource": "workers.scripts", "used": 80, "limit": 500, "percent": 16.0, "plan_tier": "paid", "limit_source": "static-docs"}
  ],
  "workers_plan": "paid",
  "plan_source": "auto"
}
```

Include the alert-condition names this command feeds (`workers-script-count`, `r2-bucket-count`, `dns-record-quota` with threshold semantics) so agents wiring alert rules find them from the usage doc.

- [ ] **Step 2: Verify docs integrity**

Run: `go build ./... && go vet ./...`
Expected: clean (docs-only change; build guards accidental edits).

- [ ] **Step 3: Commit**

```bash
git add docs/USAGE.md
git commit -m "docs(limits): usage guide for cosmoflare limits"
```

---

## Verification (whole plan)

```bash
go build ./... && go vet ./...
go test ./pkg/cosmoflare/ ./cmd/ ./internal/webhook/ -timeout 120s
```

Expected: all PASS. No test may fabricate data on missing sources (per-source errors, `ok=false` conditions).

## Stop Conditions

- ~~If the live DNS usage endpoint's field names differ from both accepted shapes~~ **RESOLVED 2026-09-09 (live pin)**: the real endpoint is `GET /zones/{zone_id}/dns_records/usage` (segment `dns_records`, not `dns` — the originally planned path returned HTTP 400 "No route for that URI"), and the response is `result: {record_usage: number, record_quota: number|null}` with `record_quota` null when an account-level quota applies. Parser fixed to the pinned shape (commit `fix(limits): pin DNS usage endpoint path and fields to the live API`); speculative alternate field names removed; null-quota test added. Account variant exists at `/accounts/{account_id}/dns_records/usage` (adds `internal_record_usage`/`internal_record_quota` when internal DNS is enabled) — unused in v1.
- If cloudflare-go v0.116's `Zone.Plan.LegacyID` is empty for all zones in a live smoke test, the static DNS fallback silently reports `unknown` limits — acceptable (row shows usage-only), do not guess tiers from plan display names. **Live smoke 2026-09-09**: static fallback emitted 200/1000 limits correctly per zone, so LegacyID resolves.
