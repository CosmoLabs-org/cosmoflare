---
title: 'Rate-limit traffic classes — Implementation Plan'
created: "2026-09-10T16:27:12+04:00"
status: DRAFT
tags: [plan, knowledge, ratelimit, probe, feat-013]
brainstorm_ref: docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md
issue: FEAT-013
deliverables:
  - id: P-01
    title: "TrafficClass schema + SkippedTrafficClasses + evidence-linked seed matrix"
  - id: P-02
    title: "RateLimitProber — ExpressionPath, classifyProbe, bounded burst"
  - id: P-03
    title: "CLI — ratelimit probe command, --probe on create, pack-driven advisory"
  - id: P-04
    title: "docs/USAGE.md — traffic-class matrix and probe documentation"
---

# Rate-limit traffic classes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Encode which traffic classes WAF rate limiting counts vs skips as knowledge-pack data, and add an opt-in, capped live trip probe that reports `tripped | not-counted | inconclusive` instead of trusting `enabled=true`.

**Architecture:** The knowledge pack gains a `traffic_classes` block (new `TrafficClass` type + `SkippedTrafficClasses` lookup — advisory when absent). A new `RateLimitProber` (RedirectProber pattern: stdlib-only, options-based, no CF credentials) fires a bounded GET burst (half cache-busted) and classifies via the pure `classifyProbe`. The CLI exposes `ratelimit probe` and `--probe` on create, always prints a pack-driven advisory after create.

**Tech Stack:** Go 1.26, net/http + httptest, cobra.

**Spec:** `docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md`

**Verified codebase facts (2026-09-10, do not re-derive):**
- Pack loader/types live in `pkg/cosmoflare/knowledge/knowledge.go` (FEAT-012, merged); packs embed from `knowledge/packs/ratelimit.json`.
- `RedirectProber` at `pkg/cosmoflare/redirectprobe.go` is the options-pattern precedent (`WithProbeHTTPClient/Timeout/Concurrency`).
- `RateLimitService` (list/create) at `pkg/cosmoflare/ratelimit.go`; CLI at `cmd/ratelimit.go` with `ratelimitServiceAndZone(ctx, target)` already defined there (uses `resolveZoneID` from `cmd/bucket_domain.go:145`).
- `ZoneService.Get(ctx, zoneID)` returns a zone with `.Name`.
- Commands register via `func init() { rootCmd.AddCommand(x) }`.
- `JSONOutput`, `printJSON`, `printErrorJSON` exist in package `cmd`.

**Test commands** (repo root):
```bash
go test ./pkg/cosmoflare/knowledge/ -v
go test ./pkg/cosmoflare/ -run 'TestRateLimitProbe|TestClassifyProbe|TestExpressionPath' -v
go test ./cmd/ -run 'TestRateLimitProbe|TestProbeExitCode|TestProbeAdvisory' -v
go build ./... && go vet ./...
```

---

### Task 1: TrafficClass schema + SkippedTrafficClasses + seed matrix

**Files:**
- Modify: `pkg/cosmoflare/knowledge/knowledge.go`
- Modify: `pkg/cosmoflare/knowledge/knowledge_test.go` (append)
- Modify: `pkg/cosmoflare/knowledge/packs/ratelimit.json`

**Interfaces:**
- Produces: `type TrafficClass struct { Class string; Counted bool; Note string; Source string }` (JSON: class/counted/note/source); `Pack.TrafficClasses []TrafficClass` (json `traffic_classes,omitempty`); `func SkippedTrafficClasses(product string) []TrafficClass`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/knowledge/knowledge_test.go`:

```go
// TestSkippedTrafficClasses verifies the ratelimit pack declares its
// not-counted traffic classes with evidence sources, and that unknown
// products return nothing (advisory when absent).
func TestSkippedTrafficClasses(t *testing.T) {
	skipped := SkippedTrafficClasses("ratelimit")
	if len(skipped) != 2 {
		t.Fatalf("ratelimit pack must declare exactly 2 skipped classes, got %d: %+v", len(skipped), skipped)
	}
	for _, tc := range skipped {
		if tc.Counted {
			t.Fatalf("skipped classes must have Counted=false: %+v", tc)
		}
		if tc.Source == "" {
			t.Fatalf("every matrix entry must carry a source: %+v", tc)
		}
	}
	if got := SkippedTrafficClasses("nosuch"); len(got) != 0 {
		t.Fatalf("unknown product must return empty, got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/knowledge/ -run TestSkippedTrafficClasses -v`
Expected: FAIL — `undefined: SkippedTrafficClasses`.

- [ ] **Step 3: Write the implementation**

In `pkg/cosmoflare/knowledge/knowledge.go`, add after the `Invariant` type:

```go
// TrafficClass documents whether WAF rate limiting counts one traffic class.
// Source carries the evidence link (CF docs or field evidence) so drift is
// auditable.
type TrafficClass struct {
	Class   string `json:"class"`
	Counted bool   `json:"counted"`
	Note    string `json:"note,omitempty"`
	Source  string `json:"source,omitempty"`
}
```

Add the field to `Pack` (after `Invariants`):

```go
	TrafficClasses []TrafficClass `json:"traffic_classes,omitempty"`
```

Add the lookup (after `LookupDecode`):

```go
// SkippedTrafficClasses returns the pack's documented not-counted classes
// for one product. Advisory when absent: unknown products and absent blocks
// return nil.
func SkippedTrafficClasses(product string) []TrafficClass {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var out []TrafficClass
	for _, p := range loaded {
		if p.Product != product {
			continue
		}
		for _, tc := range p.TrafficClasses {
			if !tc.Counted {
				out = append(out, tc)
			}
		}
	}
	return out
}
```

In `pkg/cosmoflare/knowledge/packs/ratelimit.json`, add a `traffic_classes` array (after `invariants`):

```json
  "traffic_classes": [
    {"class": "cache-hit-static-asset", "counted": false, "note": "requests served from cache never reach the rate-limiting counter", "source": "https://developers.cloudflare.com/waf/rate-limiting-rules/ + FB-7 evidence rule d65876b4 (2026-09-09)"},
    {"class": "pages-custom-domain-asset", "counted": false, "note": "Pages asset serving on custom domains observed uncounted", "source": "FB-7 evidence zone e586a1c5 (2026-09-09)"},
    {"class": "origin-miss", "counted": true, "note": "requests that reach origin are counted", "source": "https://developers.cloudflare.com/waf/rate-limiting-rules/"},
    {"class": "cache-bypass", "counted": true, "note": "cache-bypassed requests are counted", "source": "https://developers.cloudflare.com/waf/rate-limiting-rules/"}
  ]
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/knowledge/ -v`
Expected: PASS (all, including the pre-existing suite).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/knowledge/
git commit -m "feat(knowledge): traffic-class matrix with evidence sources"
```

---

### Task 2: RateLimitProber — ExpressionPath, classifyProbe, bounded burst

**Files:**
- Create: `pkg/cosmoflare/ratelimitprobe.go`
- Create: `pkg/cosmoflare/ratelimitprobe_test.go`

**Interfaces:**
- Consumes: `knowledge.SkippedTrafficClasses(product string) []TrafficClass` (Task 1).
- Produces: `func ExpressionPath(expr string) string`; `const VerdictTripped/VerdictNotCounted/VerdictInconclusive`; `type RateLimitProbeResult struct { URL string; Requests int; Statuses map[int]int; Tripped bool; Verdict string; Explanation string }`; `type RateLimitProber struct`; `func NewRateLimitProber(opts ...RateLimitProbeOption) *RateLimitProber`; options `WithProbeHTTPClient(*http.Client)`, `WithProbeTimeout(time.Duration)`, `WithProbeConcurrency(int)`; `func (p *RateLimitProber) Probe(ctx context.Context, rawURL string, requests int) RateLimitProbeResult`; `const MaxProbeRequests = 60`. Unexported pure helpers `classifyProbe(statuses map[int]int, sent, requests int) (verdict, explanation string)` — tests call it directly.

- [ ] **Step 1: Write the failing tests**

Create `pkg/cosmoflare/ratelimitprobe_test.go`:

```go
package cosmoflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestExpressionPath verifies literal path extraction from the two
// documented expression forms, and non-path expressions yield "".
func TestExpressionPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{`path eq "/catalog.json"`, "/catalog.json"},
		{`http.request.uri.path eq "/a/b"`, "/a/b"},
		{`(http.host eq "x.com" and path eq "/x")`, ""}, // composite: not a literal prefix
		{`http.request.method eq "GET"`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		if got := ExpressionPath(tt.in); got != tt.want {
			t.Errorf("ExpressionPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestClassifyProbe drives the pure verdict engine directly — the test must
// not re-derive the verdict rules inline (tautological-test guard).
func TestClassifyProbe(t *testing.T) {
	t.Run("any 429 trips", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{200: 19, 429: 1}, 20, 20)
		if v != VerdictTripped {
			t.Fatalf("verdict = %q, want tripped", v)
		}
	})
	t.Run("all ok means not counted with matrix explanation", func(t *testing.T) {
		v, expl := classifyProbe(map[int]int{200: 20}, 20, 20)
		if v != VerdictNotCounted {
			t.Fatalf("verdict = %q, want not-counted", v)
		}
		if !strings.Contains(expl, "cache-hit-static-asset") {
			t.Fatalf("explanation must cite the skipped classes: %q", expl)
		}
	})
	t.Run("transport errors dominate means inconclusive", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{-1: 15, 200: 5}, 5, 20)
		if v != VerdictInconclusive {
			t.Fatalf("verdict = %q, want inconclusive", v)
		}
	})
	t.Run("short send means inconclusive", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{200: 10}, 10, 20)
		if v != VerdictInconclusive {
			t.Fatalf("verdict = %q, want inconclusive", v)
		}
	})
	t.Run("403 also trips", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{403: 3, 200: 7}, 10, 10)
		if v != VerdictTripped {
			t.Fatalf("verdict = %q, want tripped", v)
		}
	})
}

// TestRateLimitProbeBurst exercises the live prober against httptest
// servers: 429-server trips, 200-server reports not-counted, dead server is
// inconclusive. Also asserts the cache-buster is present on odd requests.
func TestRateLimitProbeBurst(t *testing.T) {
	t.Run("always 429", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithProbeTimeout(5 * time.Second)).Probe(context.Background(), srv.URL, 6)
		if !res.Tripped || res.Verdict != VerdictTripped {
			t.Fatalf("want tripped, got %+v", res)
		}
	})
	t.Run("always 200 with cache busters", func(t *testing.T) {
		var total, busted int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&total, 1)
			if strings.Contains(r.URL.RawQuery, "cfprobe=") {
				atomic.AddInt64(&busted, 1)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithProbeTimeout(5 * time.Second)).Probe(context.Background(), srv.URL+"/catalog.json", 6)
		if res.Verdict != VerdictNotCounted {
			t.Fatalf("want not-counted, got %+v", res)
		}
		if busted != 3 || total != 6 {
			t.Fatalf("want 6 requests / 3 cache-busted, got %d/%d", total, busted)
		}
	})
	t.Run("dead server is inconclusive", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()
		res := NewRateLimitProber(WithProbeTimeout(2 * time.Second)).Probe(context.Background(), url, 4)
		if res.Verdict != VerdictInconclusive {
			t.Fatalf("want inconclusive, got %+v", res)
		}
	})
	t.Run("requests clamp to MaxProbeRequests", func(t *testing.T) {
		var n int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithProbeTimeout(5 * time.Second)).Probe(context.Background(), srv.URL, 500)
		if res.Requests != MaxProbeRequests {
			t.Fatalf("requests must clamp to %d, got %d", MaxProbeRequests, res.Requests)
		}
	})
}

// TestRateLimitProberDefaults verifies the option defaults construct a
// usable prober (non-nil client, sane timeout/concurrency).
func TestRateLimitProberDefaults(t *testing.T) {
	p := NewRateLimitProber()
	if p == nil || p.httpClient == nil || p.concurrency < 1 || p.timeout <= 0 {
		t.Fatalf("bad defaults: %+v", p)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/cosmoflare/ -run 'TestExpressionPath|TestClassifyProbe|TestRateLimitProbe' -v`
Expected: FAIL — `undefined: ExpressionPath`, `undefined: NewRateLimitProber`.

- [ ] **Step 3: Write the implementation**

Create `pkg/cosmoflare/ratelimitprobe.go`:

```go
package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// Probe verdicts.
const (
	VerdictTripped      = "tripped"
	VerdictNotCounted   = "not-counted"
	VerdictInconclusive = "inconclusive"
)

// MaxProbeRequests is the hard ceiling on one probe burst, regardless of
// caller input (issue constraint: capped, opt-in live traffic).
const MaxProbeRequests = 60

// cfKnowledgeProduct is the knowledge pack the prober explains verdicts with.
const cfKnowledgeProduct = "ratelimit"

// RateLimitProbeResult is one trip-probe outcome. Statuses maps HTTP status
// to count; -1 accumulates transport errors.
type RateLimitProbeResult struct {
	URL         string      `json:"url"`
	Requests    int         `json:"requests"`
	Statuses    map[int]int `json:"statuses"`
	Tripped     bool        `json:"tripped"`
	Verdict     string      `json:"verdict"`
	Explanation string      `json:"explanation,omitempty"`
}

// classifyProbe derives a verdict from a probe's status histogram. Pure —
// tests call it directly; do not re-derive verdict rules in test bodies.
func classifyProbe(statuses map[int]int, sent, requests int) (verdict, explanation string) {
	blocked := statuses[http.StatusTooManyRequests] + statuses[http.StatusForbidden]
	if blocked > 0 {
		return VerdictTripped, fmt.Sprintf("%d of %d requests were blocked (429/403) — the rule sees this traffic class", blocked, sent)
	}
	errs := statuses[-1]
	if sent+errs < requests || errs > sent {
		return VerdictInconclusive, fmt.Sprintf("%d of %d requests errored, %d completed — cannot judge; re-run when the target is reachable", errs, requests, sent)
	}
	var skipped []string
	for _, tc := range knowledge.SkippedTrafficClasses(cfKnowledgeProduct) {
		skipped = append(skipped, tc.Class)
	}
	return VerdictNotCounted, fmt.Sprintf(
		"all %d requests passed unblocked — the rule is live but this traffic class is likely not counted (documented skipped classes: %s); verify with a cache-busted URL or an origin-only path",
		sent, strings.Join(skipped, ", "))
}

// ExpressionPath extracts the literal path from a `path eq "/x"` or
// `http.request.uri.path eq "/x"` expression. Empty when the expression is
// not a bare literal path equality.
func ExpressionPath(expr string) string {
	expr = strings.TrimSpace(expr)
	for _, prefix := range []string{"path eq ", "http.request.uri.path eq "} {
		rest, ok := strings.CutPrefix(expr, prefix)
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
			return rest[1 : len(rest)-1]
		}
	}
	return ""
}

// RateLimitProber fires a bounded burst of GETs at a live URL and reports
// whether a zone rate-limiting rule trips. Stdlib only, no Cloudflare
// credentials (RedirectProber pattern). Explicit invocation only — never on
// default paths.
type RateLimitProber struct {
	httpClient  *http.Client
	timeout     time.Duration
	concurrency int
}

// RateLimitProbeOption configures a RateLimitProber.
type RateLimitProbeOption func(*RateLimitProber)

// WithProbeHTTPClient overrides the probe HTTP client.
func WithProbeHTTPClient(c *http.Client) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if c != nil {
			p.httpClient = c
		}
	}
}

// WithProbeTimeout sets the per-request budget.
func WithProbeTimeout(d time.Duration) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithProbeConcurrency caps parallel requests inside one burst.
func WithProbeConcurrency(n int) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if n > 0 {
			p.concurrency = n
		}
	}
}

// NewRateLimitProber builds a prober with defaults: 10s per request, 5
// concurrent requests.
func NewRateLimitProber(opts ...RateLimitProbeOption) *RateLimitProber {
	p := &RateLimitProber{
		httpClient:  &http.Client{},
		timeout:     10 * time.Second,
		concurrency: 5,
	}
	for _, opt := range opts {
		opt(p)
	}
	p.httpClient.Timeout = p.timeout
	return p
}

// Probe sends `requests` GETs to rawURL — even indices plain, odd indices
// with a unique `cfprobe` query (cache-buster, so cached vs origin classes
// are distinguishable) — and classifies the outcome.
func (p *RateLimitProber) Probe(ctx context.Context, rawURL string, requests int) RateLimitProbeResult {
	if requests < 1 {
		requests = 1
	}
	if requests > MaxProbeRequests {
		requests = MaxProbeRequests
	}

	var mu sync.Mutex
	statuses := map[int]int{}
	sent := 0

	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			u := rawURL
			if i%2 == 1 {
				sep := "?"
				if strings.Contains(u, "?") {
					sep = "&"
				}
				u = fmt.Sprintf("%s%scfprobe=%d-%d", u, sep, i, time.Now().UnixNano())
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				mu.Lock()
				statuses[-1]++
				mu.Unlock()
				return
			}
			req.Header.Set("User-Agent", "Cosmoflare-RateLimitProbe/1.0")
			resp, err := p.httpClient.Do(req)
			if err != nil {
				mu.Lock()
				statuses[-1]++
				mu.Unlock()
				return
			}
			resp.Body.Close()
			mu.Lock()
			statuses[resp.StatusCode]++
			sent++
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	verdict, explanation := classifyProbe(statuses, sent, requests)
	return RateLimitProbeResult{
		URL:         rawURL,
		Requests:    requests,
		Statuses:    statuses,
		Tripped:     verdict == VerdictTripped,
		Verdict:     verdict,
		Explanation: explanation,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/cosmoflare/ -run 'TestExpressionPath|TestClassifyProbe|TestRateLimitProbe' -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/ratelimitprobe.go pkg/cosmoflare/ratelimitprobe_test.go
git commit -m "feat(ratelimit): bounded trip prober with classify verdict engine"
```

---

### Task 3: CLI — probe command, --probe on create, advisory

**Files:**
- Modify: `cmd/ratelimit.go`
- Create: `cmd/ratelimit_probe_test.go`

**Interfaces:**
- Consumes: `cosmoflare.NewRateLimitProber`, `cosmoflare.RateLimitProbeResult`, `cosmoflare.ExpressionPath`, `cosmoflare.MaxProbeRequests`, `cosmoflare.VerdictTripped/NotCounted/Inconclusive` (Task 2); `knowledge.SkippedTrafficClasses` (Task 1); existing `ratelimitServiceAndZone(ctx, target)` in `cmd/ratelimit.go`.
- Produces: `rateLimitProbeCmd` (registered under `rateLimitCmd`); flags `--path`, `--requests`, `--probe` (on create); helpers `probeExitCode(verdict string) int`, `probeAdvisory() string`, `probeBurstDefault(ctx, svc, zoneID, path) (int, error)`, `zoneHostname(ctx, target, zoneID) (string, error)`.

- [ ] **Step 1: Write the failing tests**

Create `cmd/ratelimit_probe_test.go`:

```go
package cmd

import (
	"context"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestRateLimitProbeCmdRegistered verifies the probe subcommand exists.
func TestRateLimitProbeCmdRegistered(t *testing.T) {
	for _, c := range rateLimitCmd.Commands() {
		if c.Name() == "probe" {
			return
		}
	}
	t.Fatal("ratelimit probe subcommand missing")
}

// TestProbeExitCode verifies the scriptable exit-code mapping.
func TestProbeExitCode(t *testing.T) {
	tests := []struct {
		verdict string
		want    int
	}{
		{cosmoflare.VerdictTripped, 0},
		{cosmoflare.VerdictNotCounted, 2},
		{cosmoflare.VerdictInconclusive, 3},
		{"", 3},
	}
	for _, tt := range tests {
		if got := probeExitCode(tt.verdict); got != tt.want {
			t.Errorf("probeExitCode(%q) = %d, want %d", tt.verdict, got, tt.want)
		}
	}
}

// TestProbeAdvisory verifies the create advisory is pack-driven: it names
// the skipped classes from the embedded ratelimit pack.
func TestProbeAdvisory(t *testing.T) {
	got := probeAdvisory()
	if !strings.Contains(got, "cache-hit-static-asset") || !strings.Contains(got, "ratelimit probe") {
		t.Fatalf("advisory must list skipped classes and suggest the probe: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run 'TestRateLimitProbe|TestProbeExitCode|TestProbeAdvisory' -v`
Expected: FAIL — `undefined: probeExitCode`, probe subcommand missing.

- [ ] **Step 3: Write the implementation**

In `cmd/ratelimit.go` add imports (`os`, `knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"`) and append:

```go
// probeExitCode maps a probe verdict to a scriptable exit code:
// 0 tripped, 2 not-counted, 3 inconclusive.
func probeExitCode(verdict string) int {
	switch verdict {
	case cosmoflare.VerdictTripped:
		return 0
	case cosmoflare.VerdictNotCounted:
		return 2
	default:
		return 3
	}
}

// probeAdvisory renders the post-create advisory from pack data — never
// hardcoded. Empty when the pack declares no skipped classes.
func probeAdvisory() string {
	skipped := knowledge.SkippedTrafficClasses("ratelimit")
	if len(skipped) == 0 {
		return ""
	}
	classes := make([]string, 0, len(skipped))
	for _, tc := range skipped {
		classes = append(classes, tc.Class)
	}
	return "note: WAF rate limiting does not count: " + strings.Join(classes, ", ") +
		" — run `cosmoflare ratelimit probe` to verify this rule sees its traffic"
}

// zoneHostname resolves the probe target host: the argument itself when it
// looks like a hostname, else the zone's registered name.
func zoneHostname(ctx context.Context, target, zoneID string) (string, error) {
	if strings.Contains(target, ".") {
		return target, nil
	}
	zones, err := cosmoflare.NewZoneServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return "", err
	}
	zone, err := zones.Get(ctx, zoneID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch zone name for probe target: %w", err)
	}
	return zone.Name, nil
}

// probeBurstDefault derives the burst size from the live rule matching the
// path: 2x its requests_per_period (at least 2).
func probeBurstDefault(ctx context.Context, svc *cosmoflare.RateLimitService, zoneID, path string) (int, error) {
	rules, err := svc.List(ctx, zoneID)
	if err != nil {
		return 0, err
	}
	for _, r := range rules {
		if strings.Contains(r.Expression, path) {
			n := r.RequestsPerPeriod * 2
			if n < 2 {
				n = 2
			}
			return n, nil
		}
	}
	return 0, fmt.Errorf("no rate-limiting rule matches path %q on this zone — create one first or pass --requests", path)
}

var (
	probePath     string
	probeRequests int
	probeOnCreate bool
)

var rateLimitProbeCmd = &cobra.Command{
	Use:   "probe <zone-id-or-name>",
	Short: "Burst-probe whether a rate-limiting rule actually sees the zone's traffic",
	Long: `Send a bounded burst of live GETs (opt-in, cap 60, half cache-busted) at the
zone and report whether the rule trips.

Verdicts and exit codes:
  tripped        exit 0 — the rule sees this traffic
  not-counted    exit 2 — rule live but the traffic class is likely skipped
  inconclusive   exit 3 — network errors dominated

This command sends live traffic to the target origin. It never runs unless
explicitly invoked.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if probePath == "" {
			return fmt.Errorf("--path is required (the URL path the rule matches)")
		}
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		burst := probeRequests
		if burst == 0 {
			burst, err = probeBurstDefault(cmd.Context(), svc, zoneID, probePath)
			if err != nil {
				return err
			}
		}
		if burst > cosmoflare.MaxProbeRequests {
			fmt.Fprintf(os.Stderr, "note: --requests clamped to %d\n", cosmoflare.MaxProbeRequests)
			burst = cosmoflare.MaxProbeRequests
		}
		host, err := zoneHostname(cmd.Context(), args[0], zoneID)
		if err != nil {
			return err
		}
		res := cosmoflare.NewRateLimitProber().Probe(cmd.Context(), "https://"+host+probePath, burst)
		if JSONOutput {
			printJSON(res)
		} else {
			fmt.Printf("probe   %s\nsent    %d requests\nverdict %s\n        %s\n",
				res.URL, res.Requests, res.Verdict, res.Explanation)
		}
		os.Exit(probeExitCode(res.Verdict))
		return nil
	},
}

func init() {
	rateLimitCmd.AddCommand(rateLimitProbeCmd)
	rateLimitProbeCmd.Flags().StringVar(&probePath, "path", "", "URL path the rule matches (required)")
	rateLimitProbeCmd.Flags().IntVar(&probeRequests, "requests", 0, "burst size (default: 2x the matching rule's requests per period, capped at 60)")
	rateLimitCreateCmd.Flags().BoolVar(&probeOnCreate, "probe", false, "run a live trip probe against the created rule immediately after creation")
}
```

Then wire the advisory + optional probe into `rateLimitCreateCmd`'s RunE success path (after the existing `fmt.Printf("created ...")` line, before `return nil`):

```go
		if adv := probeAdvisory(); adv != "" {
			fmt.Println(adv)
		}
		if probeOnCreate {
			path := cosmoflare.ExpressionPath(rule.Expression)
			if path == "" {
				fmt.Println("probe skipped: expression has no literal path — run `cosmoflare ratelimit probe <zone> --path <p>`")
				return nil
			}
			host, err := zoneHostname(cmd.Context(), args[0], zoneID)
			if err != nil {
				fmt.Printf("probe skipped: %v\n", err)
				return nil
			}
			burst := rule.RequestsPerPeriod * 2
			if burst < 2 {
				burst = 2
			}
			if burst > cosmoflare.MaxProbeRequests {
				burst = cosmoflare.MaxProbeRequests
			}
			res := cosmoflare.NewRateLimitProber().Probe(cmd.Context(), "https://"+host+path, burst)
			if JSONOutput {
				return printJSON(res)
			}
			fmt.Printf("probe   %s\nsent    %d requests\nverdict %s\n        %s\n",
				res.URL, res.Requests, res.Verdict, res.Explanation)
			os.Exit(probeExitCode(res.Verdict))
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run 'TestRateLimitProbe|TestProbeExitCode|TestProbeAdvisory' -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/ratelimit.go cmd/ratelimit_probe_test.go
git commit -m "feat(cmd): ratelimit probe command + create --probe + advisory"
```

---

### Task 4: Documentation

**Files:**
- Modify: `docs/USAGE.md`

- [ ] **Step 1: Update USAGE.md**

Extend the `## Rate Limiting Command` section (added by FEAT-012) with:

1. A **Traffic-class matrix** subsection: the four v1 classes with counted/skipped and their evidence sources, and the advisory-when-absent rule.
2. A `cosmoflare ratelimit probe` subsection: what it sends (burst, cap 60, half cache-busted), the three verdicts with their exit codes (0/2/3), the opt-in safety statement, and `--json` examples.
3. A note on `ratelimit create --probe` and the always-on pack-driven advisory.

Match the surrounding heading/bullet style.

- [ ] **Step 2: Verify**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add docs/USAGE.md
git commit -m "docs(ratelimit): traffic-class matrix and trip probe"
```

---

## Verification (whole plan)

```bash
go build ./... && go vet ./...
go test ./pkg/cosmoflare/knowledge/ ./pkg/cosmoflare/ ./cmd/ -count=1 -timeout 180s
```

Expected: all PASS. httptest only; no live network in tests; probe is
opt-in everywhere; advisory renders from pack data only.

## Stop Conditions

- If `ZoneServiceFromCreds`'s `Get` signature differs from `(ctx, zoneID)` at implementation time, adapt `zoneHostname` to the real signature — the contract (name-or-hostname → hostname) is the requirement.
- If cobra's exit handling makes `os.Exit` inside RunE double-print under `--json`, print-then-exit is the contract — silence cobra's error path (`cmd.SilenceUsage`/`SilenceErrors`) rather than dropping the exit codes.

## Accepted limitations (deliberate)

- Matrix is docs-derived and can drift — every entry carries a `source` link; the probe is the authoritative live verifier.
- `probeBurstDefault` matches rules by substring containment of the path in the expression — good enough for `path eq` rules; composite expressions should pass `--requests` explicitly.
- No probe history persistence — results print (and `--json`) only.
