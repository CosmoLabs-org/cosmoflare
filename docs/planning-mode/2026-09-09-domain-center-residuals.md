---
title: 'Domain Center Residuals — Implementation Plan'
created: "2026-09-09T02:11:44+04:00"
status: DRAFT
tags: [plan, domains, redirects, pagerules, doctor, road-087]
brainstorm_ref: docs/brainstorming/2026-09-09-domain-center-residuals.md
roadmap: ROAD-087
deliverables:
  - id: P-01
    title: "RedirectRule.Source + legacyForwardingRules pure helper with table tests"
  - id: P-02
    title: "PageRuleLister + DomainService.WithPageRules + GetDetail merge with tests"
  - id: P-03
    title: "RedirectProber — bounded probes, loop detection, httptest coverage"
  - id: P-04
    title: "DomainStatus.RedirectIssue + classifyRedirectIssue + attention criterion"
  - id: P-05
    title: "domains stats --check-redirects + factory wiring + tests"
  - id: P-06
    title: "DiagnosticReport.RedirectTargets + cmd/doctor.go section + tests"
  - id: P-07
    title: "TUI detail-pane redirect-issue badge + view test"
  - id: P-08
    title: "docs/USAGE.md — merge completeness + --check-redirects + doctor section"
---

# Domain Center Residuals — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the two residual Domain Center requirements — legacy Page-Rule `forwarding_url` merge into redirect visibility, and the opt-in redirect-target attention check (≥400 / loop).

**Architecture:** Library-first, mirroring the proven enrichment seams. `RedirectRule` gains an additive `Source`; a pure helper maps legacy Page Rules; `DomainService.WithPageRules(factory)` merges them in `GetDetail`. A stdlib `RedirectProber` (own file, DoctorService pattern) classifies destinations into `DomainStatus.RedirectIssue`, which the existing attention machinery, `domains stats --check-redirects`, `doctor`, and a conditional TUI badge consume.

**Tech Stack:** Go 1.26, cobra, net/http + httptest, existing cloudflare-go services.

**Design spec:** `docs/brainstorming/2026-09-09-domain-center-residuals.md`

**Test commands** (repo root):
```bash
go test ./pkg/cosmoflare/ -run 'TestLegacyForwarding|TestRedirectProber|TestClassifyRedirectIssue|TestDomainNeedsAttention|TestGetDetail' -v  # library
go test ./cmd/ -run 'TestDomainsStats|TestDoctor' -v      # CLI
go test ./internal/tui/ -run TestDomainBrowser -v          # TUI
go build ./... && go vet ./...
```

Note: `-run TestLimits`-style substring traps — use the exact alternations above.

---

### Task 1: `RedirectRule.Source` + `legacyForwardingRules`

**Files:**
- Modify: `pkg/cosmoflare/redirect.go`
- Modify: `pkg/cosmoflare/redirect_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/redirect_test.go`:

```go
func TestLegacyForwardingRules(t *testing.T) {
	rules := []*PageRule{
		{
			ID: "pr1", Status: "active", Priority: 1,
			Targets: []PageRuleTarget{{Target: "url", Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/old/*"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/new", "status_code": float64(302)}}},
		},
		{
			ID: "pr2", Status: "disabled", Priority: 2,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/gone"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/"}},
		},
		{
			ID: "pr3", Status: "active", Priority: 3,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/cache"}}},
			Actions: []PageRuleAction{{ID: "cache_level", Value: map[string]interface{}{"value": "bypass"}}}, // not forwarding → dropped
		},
		{
			ID: "pr4", Status: "active", Priority: 4,
			Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Operator: "matches", Value: "*example.com/broken"}}},
			Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"status_code": float64(301)}}}, // no url → dropped
		},
	}

	got := legacyForwardingRules("zone1", rules)
	if len(got) != 2 {
		t.Fatalf("legacyForwardingRules returned %d rules, want 2: %+v", len(got), got)
	}
	first := got[0]
	if first.ZoneID != "zone1" || first.When != "*example.com/old/*" || first.Destination != "https://example.com/new" {
		t.Fatalf("first mapping wrong: %+v", first)
	}
	if first.StatusCode != 302 || first.Enabled || first.Source != "pagerules" {
		t.Fatalf("first flags wrong: %+v", first)
	}
	second := got[1]
	if second.When != "*example.com/gone*" || second.Destination != "https://example.com/" || second.StatusCode != 301 {
		t.Fatalf("second mapping wrong (status default 301 expected): %+v", second)
	}
	if second.Enabled {
		t.Fatal("disabled page rule must map to Enabled=false")
	}
}

func TestLegacyForwardingRulesEmpty(t *testing.T) {
	if got := legacyForwardingRules("z", nil); len(got) != 0 {
		t.Fatalf("nil input must yield empty, got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestLegacyForwarding -v`
Expected: FAIL — `undefined: legacyForwardingRules` (and no `Source` field).

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/redirect.go`, add `Source` to `RedirectRule` (after `Enabled`):

```go
type RedirectRule struct {
	ID            string `json:"id"`
	ZoneID        string `json:"zone_id"`
	When          string `json:"when"`        // expression / URL pattern
	Destination   string `json:"destination"` // target URL (may use $1..$n captures)
	StatusCode    int    `json:"status_code"` // 301, 302, 307, 308
	PreserveQuery bool   `json:"preserve_query"`
	Enabled       bool   `json:"enabled"`
	Source        string `json:"source,omitempty"` // "" = modern Rulesets rule; "pagerules" = legacy forwarding_url
}
```

And append the helper:

```go
// legacyForwardingRules maps a zone's legacy Page Rules into RedirectRule
// entries. Only actions with ID "forwarding_url" produce entries; their
// Value decodes as {url, status_code} (map form after JSON round-trip).
// Rules without a forwarding action, or whose Value lacks a url, are
// dropped — not errored. status_code defaults to 301 when absent.
func legacyForwardingRules(zoneID string, rules []*PageRule) []RedirectRule {
	out := make([]RedirectRule, 0, len(rules))
	for _, pr := range rules {
		if pr == nil {
			continue
		}
		for _, action := range pr.Actions {
			if action.ID != "forwarding_url" {
				continue
			}
			url, ok := forwardingURL(action.Value)
			if !ok {
				continue
			}
			entry := RedirectRule{
				ID:         pr.ID,
				ZoneID:     zoneID,
				Destination: url,
				StatusCode: forwardingStatusCode(action.Value, 301),
				Enabled:    pr.Status == "active",
				Source:     "pagerules",
			}
			if len(pr.Targets) > 0 {
				entry.When = pr.Targets[0].Constraint.Value
			}
			out = append(out, entry)
		}
	}
	return out
}

// forwardingURL extracts the target url from a forwarding_url action value.
func forwardingURL(value interface{}) (string, bool) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return "", false
	}
	url, ok := m["url"].(string)
	return url, ok && url != ""
}

// forwardingStatusCode extracts the status code, falling back to def.
func forwardingStatusCode(value interface{}, def int) int {
	m, ok := value.(map[string]interface{})
	if !ok {
		return def
	}
	if code, ok := m["status_code"].(float64); ok && code > 0 {
		return int(code)
	}
	return def
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestLegacyForwarding|TestRedirect' -v`
Expected: PASS — new tests plus the existing redirect suite (Source is additive).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/redirect.go pkg/cosmoflare/redirect_test.go
git commit -m "feat(redirects): map legacy forwarding_url page rules into RedirectRule entries"
```

---

### Task 2: `WithPageRules` + GetDetail merge

**Files:**
- Modify: `pkg/cosmoflare/domains.go` (struct field ~line 56, `WithPageRules` after `WithRegistrar` ~line 87, merge block in `GetDetail` after the redirects block ~line 203)
- Modify: `pkg/cosmoflare/domains_test.go`

**Prerequisite:** Task 1 landed (`legacyForwardingRules`, `RedirectRule.Source`).

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/domains_test.go`, following that file's existing service-fixture style (httptest-backed `ZoneService` etc. — mirror `TestGetDetail*` if present, else build the minimal zone-only service the file already uses):

```go
func TestGetDetailMergesLegacyPageRules(t *testing.T) {
	// Build a DomainService the same way existing GetDetail tests do
	// (zones-only; ssl/dns/doctor nil), then wire both redirect sources.
	svc := newDetailTestService(t) // existing helper in this file; zones serve one zone "z1"/"example.com"

	svc = svc.
		WithRedirects(newFakeRedirectService([]RedirectRule{
			{ID: "r1", ZoneID: "z1", When: "starts_with(\"/a\")", Destination: "https://example.com/b", Enabled: true},
		})).
		WithPageRules(func(zoneID string) PageRuleLister {
			return fakePageRuleLister{rules: []*PageRule{{
				ID: "pr1", Status: "active",
				Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Value: "*example.com/old/*"}}},
				Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/new", "status_code": float64(302)}}},
			}}}
		})

	detail, err := svc.GetDetail(context.Background(), "z1")
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Redirects) != 2 {
		t.Fatalf("merged redirects = %d, want 2 (modern + legacy): %+v", len(detail.Redirects), detail.Redirects)
	}
	if detail.Redirects[0].Source != "" || detail.Redirects[0].ID != "r1" {
		t.Fatalf("modern rule must come first with empty Source: %+v", detail.Redirects[0])
	}
	if detail.Redirects[1].Source != "pagerules" || detail.Redirects[1].Destination != "https://example.com/new" {
		t.Fatalf("legacy rule wrong: %+v", detail.Redirects[1])
	}
}

func TestGetDetailPageRuleFactoryFailureSkipsLegacy(t *testing.T) {
	svc := newDetailTestService(t).
		WithRedirects(newFakeRedirectService([]RedirectRule{{ID: "r1", ZoneID: "z1"}})).
		WithPageRules(func(zoneID string) PageRuleLister { return nil }) // nil lister → skip

	detail, err := svc.GetDetail(context.Background(), "z1")
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Redirects) != 1 {
		t.Fatalf("nil lister must skip legacy, got %d: %+v", len(detail.Redirects), detail.Redirects)
	}
}
```

If `newDetailTestService` / `newFakeRedirectService` / `fakePageRuleLister` do not exist yet, define them in this test file following the file's current fakes: `newDetailTestService` wraps whatever existing helper builds a zones-backed `DomainService`; `newFakeRedirectService` is a `*RedirectService` constructed with `cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))` serving one ruleset (copy the pattern from `redirect_test.go`); `fakePageRuleLister` is:

```go
type fakePageRuleLister struct {
	rules []*PageRule
	err   error
}

func (f fakePageRuleLister) List(ctx context.Context) ([]*PageRule, error) {
	return f.rules, f.err
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestGetDetailMergesLegacy -v`
Expected: FAIL — `svc.WithPageRules undefined`.

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/domains.go`:

Add the consumer interface near the other service types:

```go
// PageRuleLister is the consumer-side surface for legacy Page Rules.
type PageRuleLister interface {
	List(ctx context.Context) ([]*PageRule, error)
}
```

Add the struct field (after `registrar *RegistrarService`):

```go
	pageRules func(zoneID string) PageRuleLister
```

Add the wiring method (after `WithRegistrar`):

```go
// WithPageRules wires legacy forwarding-rule enrichment into GetDetail.
// The factory is invoked per zone (PageRuleService is zone-scoped at
// construction); a nil factory, nil lister, or list failure silently skips
// legacy rows — modern rules still show (partial-failure doctrine).
func (s *DomainService) WithPageRules(factory func(zoneID string) PageRuleLister) *DomainService {
	s.pageRules = factory
	return s
}
```

In `GetDetail`, immediately after the modern-redirects block (the `if s.redirects != nil { ... }` ending ~line 203), add:

```go
	// Optional legacy enrichment — forwarding_url Page Rules merged after
	// modern rules so redirect visibility is complete regardless of which
	// system created the rule. Failures skip legacy rows silently.
	if s.pageRules != nil {
		if lister := s.pageRules(zoneID); lister != nil {
			if legacy, err := lister.List(ctx); err == nil {
				detail.Redirects = append(detail.Redirects, legacyForwardingRules(zoneID, legacy)...)
			}
		}
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestGetDetail|TestDomain' -v`
Expected: PASS — new merge tests plus the existing domains suite.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/domains.go pkg/cosmoflare/domains_test.go
git commit -m "feat(domains): GetDetail merges legacy forwarding_url page rules via WithPageRules"
```

---

### Task 3: `RedirectProber`

**Files:**
- Create: `pkg/cosmoflare/redirectprobe.go`
- Create: `pkg/cosmoflare/redirectprobe_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/cosmoflare/redirectprobe_test.go`:

```go
package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestProber() *RedirectProber {
	return NewRedirectProber(
		WithProbeConcurrency(4),
		WithProbeTimeout(2*time.Second),
		WithProbeHopCap(5),
	)
}

func TestRedirectProberOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "fine")
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusOK || res.Loop || res.Err != "" || res.Skipped {
		t.Fatalf("Probe = %+v, want 200 clean", res)
	}
}

func TestRedirectProber4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusNotFound {
		t.Fatalf("Status = %d, want 404", res.Status)
	}
}

func TestRedirectProberLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/again", http.StatusFound) // /again → /again → loop
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL+"/again")
	if !res.Loop {
		t.Fatalf("Probe = %+v, want Loop=true", res)
	}
}

func TestRedirectProberHopCap(t *testing.T) {
	hops := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, fmt.Sprintf("/hop-%d", hops), http.StatusFound) // never repeats a URL, never terminates
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL+"/start")
	if res.Loop || res.Err == "" {
		t.Fatalf("Probe = %+v, want non-loop error at hop cap (Err set, Loop false)", res)
	}
}

func TestRedirectProberSkipsCaptures(t *testing.T) {
	res := newTestProber().Probe(context.Background(), "https://example.com/$1/x")
	if !res.Skipped {
		t.Fatalf("Probe = %+v, want Skipped=true for $1 capture", res)
	}
}

func TestRedirectProberUnreachable(t *testing.T) {
	res := newTestProber().Probe(context.Background(), "http://127.0.0.1:1/nope")
	if res.Err == "" || res.Status != 0 {
		t.Fatalf("Probe = %+v, want transport error with Status 0", res)
	}
}

func TestRedirectProberProbeAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "x", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	results := newTestProber().ProbeAll(context.Background(), []string{srv.URL, srv.URL + "/bad", srv.URL})
	if len(results) != 2 {
		t.Fatalf("ProbeAll must dedupe, got %d entries: %+v", len(results), results)
	}
	if results[srv.URL].Status != 200 || results[srv.URL+"/bad"].Status != 500 {
		t.Fatalf("ProbeAll results wrong: %+v", results)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestRedirectProber -v`
Expected: FAIL — `undefined: NewRedirectProber`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/redirectprobe.go`:

```go
package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RedirectProbeResult is one redirect-destination probe outcome.
type RedirectProbeResult struct {
	Destination string `json:"destination"`
	Status      int    `json:"status"`            // final HTTP status; 0 on transport error or loop
	Loop        bool   `json:"loop,omitempty"`    // a hop was revisited
	Err         string `json:"err,omitempty"`     // transport error / hop-cap message
	Skipped     bool   `json:"skipped,omitempty"` // unprobeable ($n capture reference)
}

// RedirectProber probes redirect destinations with bounded redirects and
// visited-set loop detection. Stdlib only — no Cloudflare credentials
// (DoctorService pattern).
type RedirectProber struct {
	httpClient   *http.Client
	timeout      time.Duration // per-target budget
	hopCap       int           // max redirects followed per target
	concurrency  int           // parallel probes in ProbeAll
}

// RedirectProbeOption configures the prober.
type RedirectProbeOption func(*RedirectProber)

// WithProbeHTTPClient sets a custom HTTP client (tests).
func WithProbeHTTPClient(c *http.Client) RedirectProbeOption {
	return func(p *RedirectProber) { p.httpClient = c }
}

// WithProbeTimeout sets the per-target budget (default 10s).
func WithProbeTimeout(d time.Duration) RedirectProbeOption {
	return func(p *RedirectProber) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithProbeHopCap sets the max redirects followed (default 10).
func WithProbeHopCap(n int) RedirectProbeOption {
	return func(p *RedirectProber) {
		if n > 0 {
			p.hopCap = n
		}
	}
}

// WithProbeConcurrency sets ProbeAll parallelism (default 8).
func WithProbeConcurrency(n int) RedirectProbeOption {
	return func(p *RedirectProber) {
		if n > 0 {
			p.concurrency = n
		}
	}
}

// NewRedirectProber builds a prober with the documented defaults.
func NewRedirectProber(opts ...RedirectProbeOption) *RedirectProber {
	p := &RedirectProber{
		httpClient:  &http.Client{},
		timeout:     10 * time.Second,
		hopCap:      10,
		concurrency: 8,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Probe tests one destination: GET, then follow redirects up to hopCap with
// a visited-set; a revisit marks a loop. Destinations containing "$"
// capture references are unprobeable and reported as Skipped.
func (p *RedirectProber) Probe(ctx context.Context, destination string) RedirectProbeResult {
	if strings.Contains(destination, "$") {
		return RedirectProbeResult{Destination: destination, Skipped: true}
	}

	client := *p.httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // manual hop control below
	}

	deadline := time.Now().Add(p.timeout)
	visited := map[string]bool{}
	url := destination
	for hop := 0; hop <= p.hopCap; hop++ {
		if visited[url] {
			return RedirectProbeResult{Destination: destination, Loop: true}
		}
		visited[url] = true

		if time.Now().After(deadline) {
			return RedirectProbeResult{Destination: destination, Err: "timeout"}
		}
		reqCtx, cancel := context.WithDeadline(ctx, deadline)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return RedirectProbeResult{Destination: destination, Err: err.Error()}
		}
		resp, err := client.Do(req)
		cancel()
		if err != nil {
			return RedirectProbeResult{Destination: destination, Err: err.Error()}
		}
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return RedirectProbeResult{Destination: destination, Status: resp.StatusCode}
			}
			next, err := resolveReference(url, loc)
			if err != nil {
				return RedirectProbeResult{Destination: destination, Err: err.Error()}
			}
			url = next
			continue
		}
		return RedirectProbeResult{Destination: destination, Status: resp.StatusCode}
	}
	return RedirectProbeResult{Destination: destination, Err: fmt.Sprintf("exceeded %d redirect hops", p.hopCap)}
}

// ProbeAll fans out over deduplicated destinations with a concurrency
// semaphore. Never issues more than p.concurrency requests at once.
func (p *RedirectProber) ProbeAll(ctx context.Context, destinations []string) map[string]RedirectProbeResult {
	seen := map[string]bool{}
	uniq := make([]string, 0, len(destinations))
	for _, d := range destinations {
		if d != "" && !seen[d] {
			seen[d] = true
			uniq = append(uniq, d)
		}
	}

	results := make(map[string]RedirectProbeResult, len(uniq))
	sem := make(chan struct{}, p.concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, d := range uniq {
		wg.Add(1)
		go func(dest string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := p.Probe(ctx, dest)
			mu.Lock()
			results[dest] = res
			mu.Unlock()
		}(d)
	}
	wg.Wait()
	return results
}

// resolveReference resolves a Location header against the requesting URL.
func resolveReference(base, location string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(ref).String(), nil
}
```

Add `"net/url"` to the imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestRedirectProber -v`
Expected: PASS (7 tests). The loop test works because `/again` redirects to `/again` — the visited-set catches the revisit on hop 2.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/redirectprobe.go pkg/cosmoflare/redirectprobe_test.go
git commit -m "feat(domains): RedirectProber — bounded destination probes with loop detection"
```

---

### Task 4: `RedirectIssue` + classification + attention criterion

**Files:**
- Modify: `pkg/cosmoflare/domains.go` (`DomainStatus` struct ~line 220 area, `domainNeedsAttention` ~line 290)
- Modify: `pkg/cosmoflare/domains_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/domains_test.go`:

```go
func TestClassifyRedirectIssue(t *testing.T) {
	tests := []struct {
		name    string
		results []RedirectProbeResult
		want    string
	}{
		{"empty", nil, ""},
		{"all clean", []RedirectProbeResult{{Status: 200}}, ""},
		{"skipped only", []RedirectProbeResult{{Skipped: true}}, ""},
		{"4xx", []RedirectProbeResult{{Status: 200}, {Status: 404}}, "http-4xx"},
		{"5xx", []RedirectProbeResult{{Status: 503}}, "http-5xx"},
		{"loop", []RedirectProbeResult{{Status: 404}, {Loop: true}}, "loop"},
		{"unreachable", []RedirectProbeResult{{Err: "timeout"}}, "unreachable"},
		{"loop beats 5xx", []RedirectProbeResult{{Status: 500}, {Loop: true}}, "loop"},
		{"5xx beats 4xx", []RedirectProbeResult{{Status: 404}, {Status: 500}}, "http-5xx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRedirectIssue(tt.results); got != tt.want {
				t.Fatalf("classifyRedirectIssue = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDomainNeedsAttentionRedirectIssue(t *testing.T) {
	d := &DomainStatus{NSStatus: "cloudflare", SSLStatus: "valid", HealthStatus: "up"}
	if domainNeedsAttention(d) {
		t.Fatal("clean domain must not need attention")
	}
	d.RedirectIssue = "loop"
	if !domainNeedsAttention(d) {
		t.Fatal("RedirectIssue must trigger attention")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run 'TestClassifyRedirectIssue|TestDomainNeedsAttentionRedirect' -v`
Expected: FAIL — no `RedirectIssue` field, no `classifyRedirectIssue`.

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/domains.go`, add the field to `DomainStatus` (after `NSStatus`):

```go
	NSStatus     string `json:"ns_status"`      // "cloudflare", "external", "mismatch"
	RedirectIssue string `json:"redirect_issue,omitempty"` // "" | "loop" | "http-4xx" | "http-5xx" | "unreachable"
```

Extend `domainNeedsAttention`:

```go
func domainNeedsAttention(d *DomainStatus) bool {
	return d.NSStatus == "external" || d.NSStatus == "mismatch" ||
		d.SSLStatus == "expired" || d.SSLStatus == "expiring" || d.SSLStatus == "none" ||
		d.HealthStatus == "down" ||
		d.RedirectIssue != ""
}
```

Append the pure classifier:

```go
// classifyRedirectIssue reduces a domain's probe results to the worst
// redirect issue: loop > http-5xx > http-4xx > unreachable > none.
// Skipped destinations contribute nothing.
func classifyRedirectIssue(results []RedirectProbeResult) string {
	worst := ""
	rank := map[string]int{"": 0, "unreachable": 1, "http-4xx": 2, "http-5xx": 3, "loop": 4}
	for _, r := range results {
		issue := ""
		switch {
		case r.Skipped:
			continue
		case r.Loop:
			issue = "loop"
		case r.Status >= 500:
			issue = "http-5xx"
		case r.Status >= 400:
			issue = "http-4xx"
		case r.Err != "":
			issue = "unreachable"
		}
		if rank[issue] > rank[worst] {
			worst = issue
		}
	}
	return worst
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run 'TestClassify|TestDomainNeedsAttention|TestSummarize|TestDomain' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/domains.go pkg/cosmoflare/domains_test.go
git commit -m "feat(domains): RedirectIssue classification feeds the needs-attention criteria"
```

---

### Task 5: `domains stats --check-redirects`

**Files:**
- Modify: `cmd/domains.go:54-72` (factory gains legacy wiring)
- Modify: `cmd/domains_stats.go`
- Modify: `cmd/domains_stats_test.go`

**Prerequisite:** Tasks 1-4 landed.

- [ ] **Step 1: Write the failing test**

Append to `cmd/domains_stats_test.go` following its existing factory-swap pattern:

```go
func TestDomainsStatsCheckRedirectsFlagRegistered(t *testing.T) {
	if domainsStatsCmd.Flags().Lookup("check-redirects") == nil {
		t.Fatal("--check-redirects flag not registered on domains stats")
	}
}
```

(If the file's existing tests exercise `runDomainsStats` end-to-end with a fake service, extend one to assert `RedirectIssue` appears in the JSON summary when the flag is set and the fake service sets the field. Match the file's real pattern; the flag registration test above is the floor.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestDomainsStatsCheckRedirects -v`
Expected: FAIL — flag missing.

- [ ] **Step 3: Write minimal implementation**

In `cmd/domains.go`, extend the enrich branch of the factory (after the registrar wiring):

```go
		if enrich {
			if rs, err := cosmoflare.NewRedirectServiceFromCreds(AccountID, APIToken); err == nil {
				svc = svc.WithRedirects(rs)
			}
			if rg, err := cosmoflare.NewRegistrarServiceFromCreds(AccountID, APIToken); err == nil {
				svc = svc.WithRegistrar(rg)
			}
			svc = svc.WithPageRules(func(zoneID string) cosmoflare.PageRuleLister {
				prs, err := cosmoflare.NewPageRuleServiceFromCreds(zoneID, APIToken)
				if err != nil {
					return nil
				}
				return prs
			})
		}
```

In `cmd/domains_stats.go`, add the flag and probe pass:

```go
var domainsStatsCheckRedirects bool
```

register in the command definition (next to `Use`/`Short` wiring — the file has no Flags yet, add in the same style as other cmd files):

```go
	domainsStatsCmd.Flags().BoolVar(&domainsStatsCheckRedirects, "check-redirects", false,
		"Probe redirect destinations (opt-in: live HTTP probes, 8 concurrent, 10s each)")
```

In `runDomainsStats`, after `domains` is listed and BEFORE `SummarizeDomains`:

```go
	if domainsStatsCheckRedirects {
		if err := checkDomainRedirects(ctx, svc, domains); err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("redirect check failed: %v", err))
			}
			return fmt.Errorf("redirect check failed: %w", err)
		}
	}
```

And the helper at file bottom:

```go
// checkDomainRedirects probes every domain's redirect destinations and sets
// RedirectIssue on each DomainStatus. Destinations are deduplicated across
// domains so each URL is probed once per run.
func checkDomainRedirects(ctx context.Context, svc *cosmoflare.DomainService, domains []*cosmoflare.DomainStatus) error {
	byZone := make(map[string]*cosmoflare.DomainStatus, len(domains))
	var zoneIDs []string
	for _, d := range domains {
		if d != nil && d.Zone != nil {
			byZone[d.Zone.ID] = d
			zoneIDs = append(zoneIDs, d.Zone.ID)
		}
	}

	perDomain := make(map[string][]cosmoflare.RedirectProbeResult, len(zoneIDs))
	var dests []string
	for _, zoneID := range zoneIDs {
		detail, err := svc.GetDetail(ctx, zoneID)
		if err != nil {
			continue // partial-failure: domains we cannot detail keep no verdict
		}
		for _, r := range detail.Redirects {
			if r.Destination != "" {
				dests = append(dests, r.Destination)
				perDomain[zoneID] = append(perDomain[zoneID], cosmoflare.RedirectProbeResult{Destination: r.Destination})
			}
		}
	}

	results := cosmoflare.NewRedirectProber().ProbeAll(ctx, dests)
	for zoneID, rs := range perDomain {
		withStatus := make([]cosmoflare.RedirectProbeResult, 0, len(rs))
		for _, r := range rs {
			if res, ok := results[r.Destination]; ok {
				withStatus = append(withStatus, res)
			}
		}
		if d := byZone[zoneID]; d != nil {
			d.RedirectIssue = cosmoflare.ClassifyRedirectIssues(withStatus)
		}
	}
	return nil
}
```

`classifyRedirectIssue` from Task 4 is unexported — export a wrapper in `pkg/cosmoflare/domains.go` for cmd consumption:

```go
// ClassifyRedirectIssues is the exported form of classifyRedirectIssue for
// cmd/TUI consumers.
func ClassifyRedirectIssues(results []RedirectProbeResult) string {
	return classifyRedirectIssue(results)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run 'TestDomainsStats' -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/domains.go cmd/domains_stats.go cmd/domains_stats_test.go
git commit -m "feat(cmd): domains stats --check-redirects probes destinations into the attention list"
```

---

### Task 6: `doctor` redirect-target section

**Files:**
- Modify: `pkg/cosmoflare/doctor.go:424-433` (DiagnosticReport field only — the service stays stdlib/credless)
- Modify: `cmd/doctor.go` (fetch destinations, probe, attach)
- Modify: `cmd/doctor_test.go` (extend)

- [ ] **Step 1: Write the failing test**

Append to `cmd/doctor_test.go` following its existing style — assert the report JSON carries `redirect_targets` when destinations were attached:

```go
func TestDoctorReportCarriesRedirectTargets(t *testing.T) {
	report := &cosmoflare.DiagnosticReport{Domain: "example.com"}
	report.RedirectTargets = []cosmoflare.RedirectProbeResult{
		{Destination: "https://example.com/new", Status: 200},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "redirect_targets") {
		t.Fatal("DiagnosticReport JSON must carry redirect_targets when set")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestDoctorReportCarriesRedirectTargets -v`
Expected: FAIL — no `RedirectTargets` field.

- [ ] **Step 3: Write minimal implementation**

In `pkg/cosmoflare/doctor.go`, extend `DiagnosticReport` (additive, omitempty):

```go
type DiagnosticReport struct {
	Domain         string                  `json:"domain"`
	Timestamp      time.Time               `json:"timestamp"`
	DNS            *DNSPropagationResult   `json:"dns"`
	SSL            *SSLProbeResult         `json:"ssl"`
	HTTP           *HTTPProbeResult        `json:"http"`
	Nameservers    *NSProbeResult          `json:"nameservers,omitempty"`
	RedirectTargets []RedirectProbeResult  `json:"redirect_targets,omitempty"`
	Issues         []DiagnosticIssue       `json:"issues"`
	Score          string                  `json:"score"`
}
```

In `cmd/doctor.go`, inside `runDoctorSingle` (cmd/doctor.go:165 — the target is already resolved to `domain`, with zone-ID inputs translated to names) after the report is produced and before rendering/JSON, attach redirect targets when the domain has redirects:

```go
	// Redirect-target section (opt-in by having redirects): fetch the
	// domain's destinations via DomainService (CF API), probe them with the
	// stdlib RedirectProber, attach to the report. DoctorService itself
	// stays credential-free — the cmd layer supplies the data.
	if zoneSvc, err := getZoneService(); err == nil {
		if domSvc, err := cosmoflare.NewDomainService(zoneSvc, nil, nil, nil); err == nil {
			if rs, err := cosmoflare.NewRedirectServiceFromCreds(AccountID, APIToken); err == nil {
				domSvc = domSvc.WithRedirects(rs)
				if zones, err := zoneSvc.List(context.Background()); err == nil {
					for _, z := range zones {
						if z.Name != domain {
							continue
						}
						if detail, err := domSvc.GetDetail(context.Background(), z.ID); err == nil && len(detail.Redirects) > 0 {
							dests := make([]string, 0, len(detail.Redirects))
							for _, r := range detail.Redirects {
								if r.Destination != "" {
									dests = append(dests, r.Destination)
								}
							}
							results := cosmoflare.NewRedirectProber().ProbeAll(context.Background(), dests)
							for _, d := range dests {
								report.RedirectTargets = append(report.RedirectTargets, results[d])
							}
							if issue := cosmoflare.ClassifyRedirectIssues(report.RedirectTargets); issue != "" {
								report.Issues = append(report.Issues, cosmoflare.DiagnosticIssue{
									Probe: "redirect-target", Severity: "warn",
									Message: "redirect target problem: " + issue,
									Fix:     "inspect the redirect destination — it loops, errors, or is unreachable",
								})
							}
						}
						break
					}
				}
			}
		}
	}
```

The match key is the `domain` variable already resolved at the top of `runDoctorSingle` (zone-ID inputs are translated to names there — reuse `domain`, not the raw target). Degrade silently on any error — the doctor run must not fail because redirects were unavailable.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestDoctor -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/doctor.go cmd/doctor.go cmd/doctor_test.go
git commit -m "feat(doctor): redirect-target probe section — cmd supplies destinations, report carries results"
```

---

### Task 7: TUI redirect-issue badge

**Files:**
- Modify: `internal/tui/domain.go` (detail-pane render)
- Modify: `internal/tui/domain_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/domain_test.go` following its existing view-test style:

```go
func TestDomainBrowserDetailShowsRedirectIssueBadge(t *testing.T) {
	m := NewDomainBrowserModel([]*cosmoflare.DomainStatus{
		{Zone: &cosmoflare.Zone{ID: "z1", Name: "example.com"}, RedirectIssue: "loop"},
	})
	view := m.View()
	if !strings.Contains(view, "redirect: loop") {
		t.Fatal("detail pane must render the redirect issue badge when set")
	}

	m2 := NewDomainBrowserModel([]*cosmoflare.DomainStatus{
		{Zone: &cosmoflare.Zone{ID: "z1", Name: "example.com"}},
	})
	if strings.Contains(m2.View(), "redirect:") {
		t.Fatal("badge must be absent when RedirectIssue is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestDomainBrowserDetailShowsRedirectIssueBadge -v`
Expected: FAIL — badge not rendered.

- [ ] **Step 3: Write minimal implementation**

In `internal/tui/domain.go`, find the detail-pane rendering (the function that builds the right pane from the selected `DomainStatus`). Add, in the style of the existing detail lines:

```go
	if d.RedirectIssue != "" {
		rows = append(rows, warningStyle.Render("redirect: "+d.RedirectIssue))
	}
```

Adapt `rows`/`warningStyle` to the actual local variable and the palette the file uses for warnings (mirror an existing attention/error style; if none exists, use the muted/error color helper the file already imports from the shared palette). The TUI runs no probes itself — the badge renders only when the fetched `DomainStatus` carries an issue (documented behavior).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestDomainBrowser -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/domain.go internal/tui/domain_test.go
git commit -m "feat(tui): domain detail pane renders redirect-issue badge when present"
```

---

### Task 8: Documentation

**Files:**
- Modify: `docs/USAGE.md`

- [ ] **Step 1: Update USAGE.md**

1. In the `cosmoflare domains redirects` section: note that output MERGES modern Redirect Rules with legacy Page-Rule `forwarding_url` entries, and that legacy rows carry `"source": "pagerules"` in JSON.
2. In the `cosmoflare domains stats` section: document `--check-redirects` (opt-in live probes; defaults 8 concurrent / 10s per target; sets `redirect_issue` on domains; extends the attention list).
3. In the `cosmoflare doctor` section: document the `redirect_targets` report field and the redirect-target issue severity.
4. In the `cosmoflare domains tui` section (if present): note the conditional redirect-issue badge.

- [ ] **Step 2: Verify docs integrity**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add docs/USAGE.md
git commit -m "docs(domains): legacy redirect merge, --check-redirects, doctor redirect targets"
```

---

## Verification (whole plan)

```bash
go build ./... && go vet ./...
go test ./pkg/cosmoflare/ ./cmd/ ./internal/tui/ -count=1 -timeout 180s
```

Expected: all PASS. No probe touches the live network in tests (httptest only); no fabricated verdicts (skips are skips).

## Stop Conditions

- If `PageRuleAction.Value` for `forwarding_url` arrives as a shape other than `map[string]interface{}` in live JSON (e.g. typed struct via cloudflare-go), STOP the Task 1 green step, record the observed shape in this plan, and adjust `forwardingURL`/`forwardingStatusCode` — do not error on unknown shapes, drop them.
- If `cmd/doctor.go` has no positional domain argument (different invocation shape than assumed), adapt the Task 6 lookup to the real argument source before wiring — do not add a new flag.
