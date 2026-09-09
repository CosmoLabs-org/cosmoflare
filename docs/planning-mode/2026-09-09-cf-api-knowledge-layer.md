---
title: 'CF API Knowledge Layer — Implementation Plan'
created: "2026-09-09T18:47:31+04:00"
status: DRAFT
tags: [plan, knowledge, ratelimit, errors, feat-012]
brainstorm_ref: docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md
issue: FEAT-012
deliverables:
  - id: P-01
    title: "Knowledge package — types, embedded loader, matchPath, CheckRoute with tests"
  - id: P-02
    title: "ratelimit.json seed pack + pack-validation test"
  - id: P-03
    title: "knowledge.Transport (route block) + DecodeCFError + newError hook with tests"
  - id: P-04
    title: "ValidatePayload — cap interpreter + invariants (pure) with tests"
  - id: P-05
    title: "RateLimitService — list + create via correct rulesets flow with tests"
  - id: P-06
    title: "CLI — decode, knowledge list, ratelimit list/create with tests"
  - id: P-07
    title: "docs/USAGE.md — knowledge layer, decode, ratelimit commands"
---

# CF API Knowledge Layer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Encode Cloudflare API tribal knowledge (endpoint registry, plan caps, field invariants, error decodes) as embedded data packs, enforced by a route-checking transport, pure validators, and a central error decoder — proven by a minimal RateLimitService.

**Architecture:** New package `pkg/cosmoflare/knowledge/` holds typed packs loaded from embedded JSON (`go:embed`). Request-side enforcement is `knowledge.Transport` (an `http.RoundTripper` that blocks in-scope unregistered routes); response-side decoding is `DecodeCFError(err, context)` hooked into the repo's central `newError` so every existing service inherits global-code decoding. Payload rules (caps + invariants) are interpreted data-driven by `ValidatePayload`. The reference consumer is `RateLimitService` (rulesets `http_ratelimit` phase flow).

**Tech Stack:** Go 1.26, cloudflare-go v0.116.0, net/http + httptest, cobra.

**Design spec:** `docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md`

**Verified codebase facts (2026-09-09, do not re-derive):**
- `cloudflare.NewWithAPIToken(token, cloudflare.HTTPClient(httpClient))` — the client-injection seam, already used at `pkg/cosmoflare/client.go:117`.
- `cloudflare.ZoneIdentifier(zoneID)` builds the rulesets `*ResourceContainer` (see `redirect.go:74`).
- `GetEntrypointRuleset(ctx, rc, phase)` and `UpdateEntrypointRuleset(ctx, rc, UpdateEntrypointRulesetParams{Phase, Description, Rules})` exist in v0.116.0; **`CreateRulesetRule`/`ListRulesetRules` do NOT** — per-rule create goes through the entrypoint PUT.
- `cloudflare.Error` is a VALUE type: `{StatusCode int, Errors []ResponseInfo, ErrorCodes []int, ErrorMessages []string, ...}`; there is NO `cloudflare.ErrNotFound` sentinel in v0.116.0. **Chain shape (verified in module source):** `makeRequestWithAuthTypeAndHeadersComplete` returns 4xx/5xx as typed wrappers around a POINTER — `&NotFoundError{cloudflareError: *Error}` for 404, `&RequestError{...}` for other 4xx, `&ServiceError{...}` for 5xx — each exposing `Unwrap()`. Therefore `errors.As` MUST target `*cloudflare.Error` (`var cfErr *cloudflare.Error; errors.As(err, &cfErr)`); a value target (`var cfErr cloudflare.Error`) never matches a real API error chain.
- Rate-limit rules live on `RulesetRule.RateLimit` (`*RulesetRuleRateLimit{Characteristics []string, RequestsPerPeriod, Period, MitigationTimeout int}`), with `Action: "block"`.
- `Zone.Plan.LegacyID` is `"free" | "pro" | "business" | "enterprise"` (zone.go:26).
- Repo error seam: `newError(op, msg, err) *R2Error` (errors.go:52) — every service wraps through it.
- Commands register via `func init() { rootCmd.AddCommand(x) }` + `x.AddCommand(y)` (cmd/domains.go:82-97).

**Test commands** (repo root):
```bash
go test ./pkg/cosmoflare/knowledge/ -v
go test ./pkg/cosmoflare/ -run 'TestRateLimit' -v
go test ./cmd/ -run 'TestDecode|TestKnowledge|TestRateLimitCmd' -v
go build ./... && go vet ./...
```

---

### Task 1: knowledge package — types, loader, matchPath, CheckRoute

**Files:**
- Create: `pkg/cosmoflare/knowledge/knowledge.go`
- Create: `pkg/cosmoflare/knowledge/knowledge_test.go`
- Create: `pkg/cosmoflare/knowledge/packs/ratelimit.json` (placeholder `{}` — Task 2 overwrites with the real pack)

- [ ] **Step 1: Write the failing test**

Create `pkg/cosmoflare/knowledge/knowledge_test.go`:

```go
package knowledge

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		path string
		want bool
	}{
		{"exact", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets", true},
		{"param mismatch length", "/zones/{zone_id}/rulesets", "/zones/z1/rulesets/extra", false},
		{"two params", "/zones/{z}/rulesets/{r}/rules", "/zones/z1/rulesets/r1/rules", true},
		{"literal segment differs", "/zones/{z}/dns", "/accounts/a1/dns", false},
		{"scope star consumes rest", "/zones/{zone_id}/rulesets*", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint", true},
		{"star requires prefix", "/zones/{zone_id}/rulesets*", "/zones/z1/other", false},
		{"empty param segment", "/zones/{z}/rulesets", "/zones//rulesets", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPath(tt.tmpl, tt.path); got != tt.want {
				t.Fatalf("matchPath(%q, %q) = %v, want %v", tt.tmpl, tt.path, got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/client/v4/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/zones/z1/rulesets", "/zones/z1/rulesets"},
		{"/client/v4/client/v4/zones", "/client/v4/zones"}, // strip one prefix only
	}
	for _, tt := range tests {
		if got := NormalizePath(tt.in); got != tt.want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/knowledge/ -v`
Expected: FAIL — package has no non-test source files.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/knowledge/knowledge.go`:

```go
// Package knowledge encodes Cloudflare API tribal knowledge — endpoint
// registries, plan caps, field invariants, and error decodes — as embedded
// JSON data packs. Knowledge is advisory when absent, authoritative when
// present: routes outside every pack scope pass through untouched; routes
// inside a scope that match no registered endpoint are blocked before send.
package knowledge

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed packs/*.json
var packFS embed.FS

// Endpoint registers one API route. A PathTemplate segment written as
// "{name}" matches any single non-empty path segment; a trailing "*" glued
// to a segment (or a standalone "*" segment) prefix-matches the rest.
type Endpoint struct {
	Method       string `json:"method"`
	PathTemplate string `json:"path"`
	Status       string `json:"status,omitempty"` // "" = exists; "absent" = known-nonexistent; "deprecated-off" = API disabled
	Note         string `json:"note,omitempty"`
}

// ErrorDecode maps one CF error code (optionally refined by context) to a
// human/agent-actionable cause and fix.
type ErrorDecode struct {
	Code    int    `json:"code"`
	Context string `json:"context,omitempty"` // "" = applies globally
	Cause   string `json:"cause"`
	Fix     string `json:"fix"`
}

// PlanCap is one plan's payload-parameter caps. Cap keys are op-prefixed:
// "max:<field>" (numeric ceiling), "allow_set:<field>" (permitted string set).
type PlanCap struct {
	Plan string         `json:"plan"`
	Caps map[string]int `json:"caps"`
	Sets map[string][]string `json:"sets,omitempty"`
}

// Invariant is a mandatory-field rule checked pre-send.
type Invariant struct {
	Field string `json:"field"`
	Op    string `json:"op"`  // "must_include"
	Value string `json:"value"`
}

// Pack is one product's knowledge.
type Pack struct {
	Product    string        `json:"product"`
	Scopes     []string      `json:"scopes"`
	Endpoints  []Endpoint    `json:"endpoints"`
	Errors     []ErrorDecode `json:"errors"`
	PlanCaps   []PlanCap     `json:"plan_caps"`
	Invariants []Invariant   `json:"invariants"`
}

var (
	loadOnce sync.Once
	packs    []*Pack
	loadErr  error
)

// Load parses every embedded pack exactly once. Malformed pack JSON fails
// loud — corrupted knowledge must never silently pass.
func Load() ([]*Pack, error) {
	loadOnce.Do(func() {
		entries, err := packFS.ReadDir("packs")
		if err != nil {
			loadErr = fmt.Errorf("knowledge: read packs dir: %w", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := packFS.ReadFile("packs/" + e.Name())
			if err != nil {
				loadErr = fmt.Errorf("knowledge: read pack %s: %w", e.Name(), err)
				return
			}
			var p Pack
			if err := json.Unmarshal(data, &p); err != nil {
				loadErr = fmt.Errorf("knowledge: parse pack %s: %w", e.Name(), err)
				return
			}
			packs = append(packs, &p)
		}
	})
	return packs, loadErr
}

// NormalizePath strips one leading "/client/v4" (present when the default
// cloudflare-go BaseURL is used; absent under httptest BaseURLs).
func NormalizePath(path string) string {
	return strings.TrimPrefix(path, "/client/v4")
}

// matchPath reports whether a concrete path matches a template.
func matchPath(tmpl, path string) bool {
	if tmpl == "" || path == "" {
		return false
	}
	t := strings.Split(strings.TrimPrefix(tmpl, "/"), "/")
	p := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, seg := range t {
		if i >= len(p) {
			return false
		}
		if strings.HasSuffix(seg, "*") {
			return strings.HasPrefix(p[i], strings.TrimSuffix(seg, "*"))
		}
		if strings.HasPrefix(seg, "{") {
			if p[i] == "" {
				return false
			}
			continue
		}
		if seg != p[i] {
			return false
		}
	}
	return len(t) == len(p)
}

// RouteVerdict is the outcome of a route check.
type RouteVerdict struct {
	InScope  bool
	Blocked  bool
	Pack     string
	Endpoint *Endpoint
}

// CheckRoute classifies a method+path against every loaded pack.
func CheckRoute(method, path string) RouteVerdict {
	loaded, err := Load()
	if err != nil {
		return RouteVerdict{} // loader errors surface at Load() call sites
	}
	for _, p := range loaded {
		scoped := false
		for _, s := range p.Scopes {
			if matchPath(s, path) {
				scoped = true
				break
			}
		}
		if !scoped {
			continue
		}
		for i := range p.Endpoints {
			e := p.Endpoints[i]
			if e.Method == method && matchPath(e.PathTemplate, path) {
				if e.Status == "" {
					return RouteVerdict{InScope: true, Pack: p.Product, Endpoint: &e}
				}
				// Registered as known-absent or API-disabled → blocked too
				// (the Endpoint is kept so callers can surface the note).
				return RouteVerdict{InScope: true, Blocked: true, Pack: p.Product, Endpoint: &e}
			}
		}
		// In scope, no endpoint match → blocked.
		return RouteVerdict{InScope: true, Blocked: true, Pack: p.Product}
	}
	return RouteVerdict{}
}

// LookupDecode finds the best decode for a code: exact (code+context) first,
// then the code's context-free entry. Nil when nothing is registered.
func LookupDecode(code int, context string) *ErrorDecode {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var generic *ErrorDecode
	for _, p := range loaded {
		for _, d := range p.Errors {
			if d.Code != code {
				continue
			}
			if d.Context == context && context != "" {
				return &d
			}
			if d.Context == "" && generic == nil {
				generic = &d
			}
		}
	}
	return generic
}
```

Also create `pkg/cosmoflare/knowledge/packs/ratelimit.json` containing exactly
`{}` (a valid, empty JSON object). `go:embed` globs that match zero files are
COMPILE errors — `pattern packs/*.json: no matching files found` (verified
empirically against Go 1.26; a lone `.gitkeep` does not help because embed
excludes dotfiles, and directory-embed of a dotfile-only dir fails with
"cannot embed directory packs: contains no embeddable files"). The placeholder
must exist from Task 1 or Step 4 cannot build. With `{}` the pack parses with
an empty product, so Task 2's failing-test step still fails with exactly
"ratelimit pack not loaded". Task 2 overwrites this file with the real pack.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/knowledge/ -v`
Expected: PASS — TestMatchPath, TestNormalizePath.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/knowledge/
git commit -m "feat(knowledge): pack types, embedded loader, route matching"
```

---

### Task 2: ratelimit seed pack

**Files:**
- Modify: `pkg/cosmoflare/knowledge/packs/ratelimit.json` (replace the `{}` placeholder created in Task 1)
- Modify: `pkg/cosmoflare/knowledge/knowledge_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `pkg/cosmoflare/knowledge/knowledge_test.go`:

```go
func TestRatelimitPackLoadedAndValid(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pack *Pack
	for _, p := range loaded {
		if p.Product == "ratelimit" {
			pack = p
		}
	}
	if pack == nil {
		t.Fatal("ratelimit pack not loaded")
	}
	if len(pack.Scopes) == 0 || len(pack.Endpoints) == 0 {
		t.Fatalf("ratelimit pack must declare scopes and endpoints: %+v", pack)
	}

	// Scope check: rulesets paths are in scope, everything else is not.
	if !CheckRoute("GET", "/zones/z1/rulesets").InScope {
		t.Fatal("rulesets path must be in scope")
	}
	if CheckRoute("GET", "/zones/z1/dns_records").InScope {
		t.Fatal("dns_records path must be out of scope")
	}

	// The known-nonexistent route blocks pre-send.
	v := CheckRoute("POST", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint/rules")
	if !v.Blocked {
		t.Fatal("POST entrypoint/rules must be blocked (endpoint does not exist)")
	}

	// The documented routes pass.
	if v2 := CheckRoute("PUT", "/zones/z1/rulesets/phases/http_ratelimit/entrypoint"); v2.Blocked || v2.Endpoint == nil {
		t.Fatalf("PUT entrypoint must be registered: %+v", v2)
	}

	// All four evidence codes decode.
	for _, code := range []int{10405, 20155, 10000} {
		if LookupDecode(code, "") == nil {
			t.Fatalf("code %d must have a global decode", code)
		}
	}
	if LookupDecode(1000, "phase-entrypoint") == nil {
		t.Fatal("code 1000 must decode under context phase-entrypoint")
	}

	// Free caps present.
	found := false
	for _, c := range pack.PlanCaps {
		if c.Plan == "free" {
			found = true
			if c.Caps["max:period_seconds"] != 10 || c.Caps["max:mitigation_timeout_seconds"] != 10 || c.Caps["max:rules_count"] != 1 {
				t.Fatalf("free caps wrong: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("free plan caps missing")
	}

	// The cf.colo.id invariant is present.
	hasInvariant := false
	for _, inv := range pack.Invariants {
		if inv.Field == "characteristics" && inv.Value == "cf.colo.id" {
			hasInvariant = true
		}
	}
	if !hasInvariant {
		t.Fatal("cf.colo.id invariant missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/knowledge/ -run TestRatelimitPackLoadedAndValid -v`
Expected: FAIL — `ratelimit pack not loaded`.

- [ ] **Step 3: Write the pack**

Replace the contents of `pkg/cosmoflare/knowledge/packs/ratelimit.json` (the Task 1 placeholder) with:

```json
{
  "product": "ratelimit",
  "scopes": ["/zones/{zone_id}/rulesets*", "/zones/{zone_id}/rate_limits*"],
  "endpoints": [
    {"method": "GET", "path": "/zones/{zone_id}/rulesets"},
    {"method": "GET", "path": "/zones/{zone_id}/rulesets/phases/{phase}/entrypoint"},
    {"method": "PUT", "path": "/zones/{zone_id}/rulesets/phases/{phase}/entrypoint"},
    {"method": "GET", "path": "/zones/{zone_id}/rulesets/{ruleset_id}/rules"},
    {"method": "POST", "path": "/zones/{zone_id}/rulesets/{ruleset_id}/rules"},
    {"method": "DELETE", "path": "/zones/{zone_id}/rulesets/{ruleset_id}/rules/{rule_id}"},
    {"method": "POST", "path": "/zones/{zone_id}/rulesets/phases/{phase}/entrypoint/rules", "status": "absent", "note": "returns 10405 'method not allowed for this authentication scheme' — the route does not exist"},
    {"method": "GET", "path": "/zones/{zone_id}/rate_limits", "status": "deprecated-off", "note": "classic Rate Limiting API is deprecated and disabled"},
    {"method": "POST", "path": "/zones/{zone_id}/rate_limits", "status": "deprecated-off", "note": "classic Rate Limiting API is deprecated and disabled"}
  ],
  "errors": [
    {"code": 10405, "cause": "route not valid for this authentication scheme — usually: the endpoint does not exist", "fix": "check the route against the registered endpoints; e.g. POST .../entrypoint/rules does not exist — use PUT on the phase entrypoint"},
    {"code": 1000, "context": "phase-entrypoint", "cause": "phase entrypoint ruleset missing on this zone", "fix": "create it with PUT /zones/{id}/rulesets/phases/{phase}/entrypoint"},
    {"code": 20155, "cause": "missing cf.colo.id in rate-limit characteristics", "fix": "add 'cf.colo.id' to characteristics — ratelimit counting is processed at colocation level only"},
    {"code": 10000, "cause": "token missing the required scope for this endpoint", "fix": "for rate-limit rule writes the token needs Zone > Zone WAF > Edit (the permission NAMED 'Rate Limiting' covers the dead classic API)"}
  ],
 "plan_caps": [
    {
      "plan": "free",
      "caps": {
        "max:rules_count": 1,
        "max:period_seconds": 10,
        "max:mitigation_timeout_seconds": 10
      },
      "sets": {
        "allow_set:characteristics": ["ip.src", "cf.colo.id"]
      }
    }
  ],
  "invariants": [
    {"field": "characteristics", "op": "must_include", "value": "cf.colo.id"}
  ]
}
```

Evidence: MyCarGuide zone e586a1c56dc1020690f9eeea7e46a3bd, 2026-09-09 (FB-6). Sources: developers.cloudflare.com/waf/rate-limiting-rules/parameters + the availability table.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/knowledge/ -v`
Expected: PASS (all).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/knowledge/
git commit -m "feat(knowledge): ratelimit seed pack from MyCarGuide evidence"
```

---

### Task 3: Transport + DecodeCFError + newError hook

**Files:**
- Create: `pkg/cosmoflare/knowledge/transport.go`
- Create: `pkg/cosmoflare/knowledge/transport_test.go`
- Create: `pkg/cosmoflare/knowledge/decode.go`
- Modify: `pkg/cosmoflare/errors.go` (newError)

- [ ] **Step 1: Write the failing tests**

Create `pkg/cosmoflare/knowledge/transport_test.go`:

```go
package knowledge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func TestTransportBlocksUnregisteredInScopeRoute(t *testing.T) {
	var sawRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &Transport{Base: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		srv.URL+"/zones/z1/rulesets/phases/http_ratelimit/entrypoint/rules", nil)
	_, err := tr.RoundTrip(req)

	if err == nil {
		t.Fatal("unregistered in-scope route must be blocked")
	}
	if sawRequest {
		t.Fatal("blocked route must not reach the server")
	}
	if !strings.Contains(err.Error(), "10405") {
		t.Fatalf("block message must cite CF's 10405 wording: %v", err)
	}
}

func TestTransportPassesRegisteredRouteAndOutOfScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &Transport{Base: http.DefaultTransport}
	for _, path := range []string{
		"/zones/z1/rulesets",                     // in scope, registered
		"/client/v4/zones/z1/rulesets",           // in scope after prefix strip, registered
		"/zones/z1/dns_records",                  // out of scope entirely
	} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatalf("GET %s must pass: %v", path, err)
		}
		resp.Body.Close()
	}
}

func TestDecodeCFError(t *testing.T) {
	// Build the error the way the SDK really returns 4xx: a typed wrapper
	// (RequestError) around *cloudflare.Error, then service-layer wrapping.
	// A bare value-type cloudflare.Error would not match errors.As against
	// real chains (see the header facts).
	wrapped := fmt_wrap(cloudflare.NewRequestError(&cloudflare.Error{
		StatusCode: 400, ErrorCodes: []int{20155},
	}))

	out := DecodeCFError(wrapped, "")
	var ke *KnowledgeError
	if !errors.As(out, &ke) {
		t.Fatalf("DecodeCFError must return *KnowledgeError, got %T", out)
	}
	if ke.Code != 20155 || !strings.Contains(ke.Fix, "cf.colo.id") {
		t.Fatalf("decode wrong: %+v", ke)
	}
	if !strings.Contains(out.Error(), "colocation") {
		t.Fatalf("error string must carry the cause: %v", out)
	}

	unknown := DecodeCFError(errors.New("plain"), "")
	if unknown.Error() != "plain" {
		t.Fatalf("unknown error must pass through unchanged, got %v", unknown)
	}
}

// fmt_wrap mimics a wrapped cloudflare error the way service layers do.
func fmt_wrap(err error) error {
	return fmt.Errorf("wrapped: %w", err)
}
```

Create `pkg/cosmoflare/knowledge/decode.go` AFTER writing the test only if implementing (Step 3).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/cosmoflare/knowledge/ -run 'TestTransport|TestDecodeCFError' -v`
Expected: FAIL — `undefined: Transport`, `undefined: KnowledgeError`, `undefined: DecodeCFError`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/knowledge/transport.go`:

```go
package knowledge

import (
	"fmt"
	"net/http"
)

// Transport is an http.RoundTripper enforcing the endpoint registry on the
// request side. Routes inside a pack scope that match no registered
// endpoint never leave the process: CF would answer 10405 with wording
// that misreads as an auth-scope problem.
type Transport struct {
	Base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (tr *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	verdict := CheckRoute(req.Method, NormalizePath(req.URL.Path))
	if verdict.Blocked {
		hint := ""
		if absent := absentEndpointHint(verdict.Pack, req.Method); absent != "" {
			hint = " " + absent
		}
		return nil, fmt.Errorf(
			"cosmoflare knowledge: %s %s is not a registered endpoint (pack %q).%s Cloudflare would return 10405 \"method not allowed for this authentication scheme\" — misleading: the route does not exist",
			req.Method, NormalizePath(req.URL.Path), verdict.Pack, hint)
	}
	base := tr.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// absentEndpointHint surfaces a pack note when the blocked method has a
// known-absent endpoint registered with one.
func absentEndpointHint(product, method string) string {
	loaded, err := Load()
	if err != nil {
		return ""
	}
	for _, p := range loaded {
		if p.Product != product {
			continue
		}
		for _, e := range p.Endpoints {
			if e.Method == method && e.Status != "" && e.Note != "" {
				return fmt.Sprintf("Note: %s.", e.Note)
			}
		}
	}
	return ""
}
```

Create `pkg/cosmoflare/knowledge/decode.go`:

```go
package knowledge

import (
	"errors"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// KnowledgeError decorates a Cloudflare API error with the knowledge
// layer's cause and fix.
type KnowledgeError struct {
	Code    int
	Message string
	Cause   string
	Fix     string
	Err     error
}

func (e *KnowledgeError) Error() string {
	return fmt.Sprintf("cloudflare error %d (%s): %s — fix: %s", e.Code, e.Message, e.Cause, e.Fix)
}

func (e *KnowledgeError) Unwrap() error { return e.Err }

// DecodeCFError wraps err in a *KnowledgeError when a decode exists for its
// code (context refines: "phase-entrypoint" for entrypoint 1000s). Unknown
// errors return unchanged.
func DecodeCFError(err error, context string) error {
	if err == nil {
		return nil
	}
	var cfErr *cloudflare.Error
	if !errors.As(err, &cfErr) {
		return err
	}
	for _, code := range cfErr.ErrorCodes {
		if d := LookupDecode(code, context); d != nil {
			msg := ""
			if len(cfErr.ErrorMessages) > 0 {
				msg = cfErr.ErrorMessages[0]
			}
			return &KnowledgeError{Code: code, Message: msg, Cause: d.Cause, Fix: d.Fix, Err: err}
		}
	}
	return err
}
```

Then wire the global hook. In `pkg/cosmoflare/errors.go`, extend `newError`:

```go
func newError(op, msg string, err error) *R2Error {
	e := &R2Error{Op: op, Message: msg, Err: err}
	// Knowledge layer: decorate globally-decoded CF errors for every
	// service without call-site changes (context-free entries only).
	if ke, ok := knowledge.DecodeCFError(err, "").(*knowledge.KnowledgeError); ok {
		e.Message = msg + " — " + ke.Cause + " (fix: " + ke.Fix + ")"
	}
	return e
}
```

Add the import to errors.go: `knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/cosmoflare/knowledge/ -v && go build ./...`
Expected: PASS + clean build (errors.go import compiles).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/knowledge/ pkg/cosmoflare/errors.go
git commit -m "feat(knowledge): route-blocking transport + central CF error decoding"
```

---

### Task 4: ValidatePayload — caps + invariants interpreter

**Files:**
- Create: `pkg/cosmoflare/knowledge/validate.go`
- Create: `pkg/cosmoflare/knowledge/validate_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/cosmoflare/knowledge/validate_test.go`:

```go
package knowledge

import "testing"

func TestValidatePayloadRatelimit(t *testing.T) {
	base := map[string]any{
		"rules_count":                1,
		"period_seconds":             10,
		"mitigation_timeout_seconds": 10,
		"characteristics":            []string{"cf.colo.id", "ip.src"},
	}

	t.Run("clean on free", func(t *testing.T) {
		if v := ValidatePayload("ratelimit", "free", base); len(v) != 0 {
			t.Fatalf("clean free payload must pass, got %+v", v)
		}
	})

	t.Run("missing cf.colo.id violates invariant", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 1, "period_seconds": 10, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"ip.src"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one characteristics violation, got %+v", v)
		}
	})

	t.Run("free caps enforced", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 2, "period_seconds": 60, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"cf.colo.id", "ip.src"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 2 {
			t.Fatalf("want rules_count + period violations, got %+v", v)
		}
	})

	t.Run("characteristics allow_set enforced", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 1, "period_seconds": 10, "mitigation_timeout_seconds": 10,
			"characteristics": []string{"cf.colo.id", "http.request.headers.x-tiny"},
		}
		v := ValidatePayload("ratelimit", "free", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("want one allow_set violation, got %+v", v)
		}
	})

	t.Run("unknown plan skips caps not invariants", func(t *testing.T) {
		p := map[string]any{
			"rules_count": 99, "period_seconds": 3600, "mitigation_timeout_seconds": 600,
			"characteristics": []string{"ip.src"},
		}
		v := ValidatePayload("ratelimit", "", p)
		if len(v) != 1 || v[0].Field != "characteristics" {
			t.Fatalf("unknown plan: only the invariant fires, got %+v", v)
		}
	})

	t.Run("unknown product passes", func(t *testing.T) {
		if v := ValidatePayload("nosuch", "free", base); len(v) != 0 {
			t.Fatalf("unknown product must pass, got %+v", v)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/knowledge/ -run TestValidatePayload -v`
Expected: FAIL — `undefined: ValidatePayload`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/knowledge/validate.go`:

```go
package knowledge

import "fmt"

// Violation is one preflight failure: the field, the rule, and the fix.
type Violation struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Fix   string `json:"fix"`
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s — %s", v.Field, v.Rule, v.Fix)
}

// ValidatePayload checks a product payload against its pack's plan caps and
// invariants. plan "" (unknown) skips caps but still runs invariants.
// Unknown products return nil (knowledge is advisory when absent).
func ValidatePayload(product, plan string, payload map[string]any) []Violation {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var pack *Pack
	for _, p := range loaded {
		if p.Product == product {
			pack = p
		}
	}
	if pack == nil {
		return nil
	}

	var out []Violation

	if plan != "" {
		for _, cap := range pack.PlanCaps {
			if cap.Plan != plan {
				continue
			}
			for key, limit := range cap.Caps {
				field, ok := strings_CutPrefix(key, "max:")
				if !ok {
					continue
				}
				if val, ok := payload[field].(int); ok && val > limit {
					out = append(out, Violation{
						Field: field,
						Rule:  fmt.Sprintf("plan %q caps %s at %d (got %d)", plan, field, limit, val),
						Fix:   fmt.Sprintf("reduce %s to <= %d or upgrade the plan", field, limit),
					})
				}
			}
			for key, allowed := range cap.Sets {
				field, ok := strings_CutPrefix(key, "allow_set:")
				if !ok {
					continue
				}
				vals, ok := payload[field].([]string)
				if !ok {
					continue
				}
				for _, v := range vals {
					if !contains(allowed, v) {
						out = append(out, Violation{
							Field: field,
							Rule:  fmt.Sprintf("plan %q allows only %v (got %q)", plan, allowed, v),
							Fix:   "remove the characteristic or upgrade the plan",
						})
					}
				}
			}
		}
	}

	for _, inv := range pack.Invariants {
		if inv.Op != "must_include" {
			continue
		}
		vals, ok := payload[inv.Field].([]string)
		if !ok || !contains(vals, inv.Value) {
			out = append(out, Violation{
				Field: inv.Field,
				Rule:  fmt.Sprintf("must include %q", inv.Value),
				Fix:   fmt.Sprintf("add %q to %s", inv.Value, inv.Field),
			})
		}
	}
	return out
}

// strings_CutPrefix avoids importing strings for one helper.
func strings_CutPrefix(s, prefix string) (string, bool) {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/knowledge/ -v`
Expected: PASS (all).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/knowledge/
git commit -m "feat(knowledge): data-driven payload validation with plan caps"
```

---

### Task 5: RateLimitService — the reference consumer

**Files:**
- Create: `pkg/cosmoflare/ratelimit.go`
- Create: `pkg/cosmoflare/ratelimit_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/cosmoflare/ratelimit_test.go`:

```go
package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// ratelimitTestServer serves the full rulesets flow. Phase is the
// http_ratelimit entrypoint ruleset "rl-rs" with `existing` rules.
func ratelimitTestServer(t *testing.T, existing []cloudflare.RulesetRule) (*httptest.Server, *cloudflare.API) {
	t.Helper()
	mux := http.NewServeMux()
	encode := func(w http.ResponseWriter, result any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
	}
	mux.HandleFunc("/zones/z1", func(w http.ResponseWriter, r *http.Request) {
		encode(w, map[string]any{"id": "z1", "name": "example.com", "plan": map[string]any{"legacy_id": "free"}})
	})
	mux.HandleFunc("/zones/z1/rulesets/phases/http_ratelimit/entrypoint", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if existing == nil {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"errors":  []map[string]any{{"code": 1000, "message": "not_found"}},
				})
				return
			}
			encode(w, cloudflare.Ruleset{ID: "rl-rs", Phase: "http_ratelimit", Rules: existing})
		case http.MethodPut:
			var body struct {
				Rules []cloudflare.RulesetRule `json:"rules"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode PUT body: %v", err)
			}
			for i := range body.Rules {
				if body.Rules[i].ID == "" {
					body.Rules[i].ID = "generated-id"
				}
			}
			encode(w, cloudflare.Ruleset{ID: "rl-rs", Phase: "http_ratelimit", Rules: body.Rules})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return srv, cf
}

func newRatelimitService(t *testing.T, cf *cloudflare.API) *RateLimitService {
	t.Helper()
	zones, err := NewZoneService(cf, "acct-test")
	if err != nil {
		t.Fatalf("NewZoneService: %v", err)
	}
	svc, err := NewRateLimitService(cf, zones)
	if err != nil {
		t.Fatalf("NewRateLimitService: %v", err)
	}
	return svc
}

func TestRateLimitListEmptyOnMissingEntrypoint(t *testing.T) {
	_, cf := ratelimitTestServer(t, nil)
	got, err := newRatelimitService(t, cf).List(context.Background(), "z1")
	if err != nil {
		t.Fatalf("List on fresh zone must return empty, not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 rules, got %+v", got)
	}
}

func TestRateLimitListReturnsRules(t *testing.T) {
	enabled := true
	_, cf := ratelimitTestServer(t, []cloudflare.RulesetRule{{
		ID: "r1", Action: "block", Expression: `path eq "/a"`, Enabled: &enabled,
		RateLimit: &cloudflare.RulesetRuleRateLimit{
			Characteristics: []string{"cf.colo.id", "ip.src"},
			RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		},
	}})
	got, err := newRatelimitService(t, cf).List(context.Background(), "z1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "r1" || got[0].RequestsPerPeriod != 10 {
		t.Fatalf("mapping wrong: %+v", got)
	}
	if !containsStr(got[0].Characteristics, "cf.colo.id") {
		t.Fatalf("characteristics must carry through: %+v", got[0])
	}
}

func TestRateLimitCreatePreflightBlocksMissingColo(t *testing.T) {
	_, cf := ratelimitTestServer(t, nil)
	_, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/a"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"ip.src"}, // missing cf.colo.id
	})
	if err == nil {
		t.Fatal("preflight must block missing cf.colo.id")
	}
	if !strings.Contains(err.Error(), "cf.colo.id") {
		t.Fatalf("error must name the field: %v", err)
	}
}

func TestRateLimitCreatePreflightBlocksFreeCap(t *testing.T) {
	enabled := true
	_, cf := ratelimitTestServer(t, []cloudflare.RulesetRule{{
		ID: "existing", Action: "block", Expression: `path eq "/old"`, Enabled: &enabled,
	}}) // zone already has 1 rule; free cap is 1
	_, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/new"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"cf.colo.id", "ip.src"},
	})
	if err == nil {
		t.Fatal("free plan allows 1 rule — second create must block")
	}
	if !strings.Contains(err.Error(), "rules_count") {
		t.Fatalf("error must name the cap: %v", err)
	}
}

func TestRateLimitCreatePutsEntrypoint(t *testing.T) {
	_, cf := ratelimitTestServer(t, nil)
	rule, err := newRatelimitService(t, cf).Create(context.Background(), RateLimitCreateInput{
		ZoneID: "z1", Expression: `path eq "/catalog.json"`,
		RequestsPerPeriod: 10, Period: 10, MitigationTimeout: 10,
		Characteristics: []string{"cf.colo.id", "ip.src"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID == "" || rule.Period != 10 {
		t.Fatalf("created rule wrong: %+v", rule)
	}
}

func containsStr(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestRateLimit -v`
Expected: FAIL — `undefined: RateLimitService`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/cosmoflare/ratelimit.go`:

```go
package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// RateLimitRule is one http_ratelimit phase rule in our shape.
type RateLimitRule struct {
	ID                 string   `json:"id"`
	Expression         string   `json:"expression"`
	Description        string   `json:"description,omitempty"`
	Enabled            bool     `json:"enabled"`
	Characteristics     []string `json:"characteristics,omitempty"`
	RequestsPerPeriod  int      `json:"requests_per_period"`
	Period             int      `json:"period_seconds"`
	MitigationTimeout  int      `json:"mitigation_timeout_seconds"`
}

// RateLimitCreateInput captures what a caller supplies to create one rule.
type RateLimitCreateInput struct {
	ZoneID             string
	Expression         string
	Description        string
	RequestsPerPeriod  int
	Period             int
	MitigationTimeout  int
	Characteristics    []string
}

// RateLimitService manages zone rate-limiting rules via the Rulesets
// http_ratelimit phase. It is the reference consumer of the knowledge
// layer: preflight validation before any call, decoded errors after.
type RateLimitService struct {
	cf    *cloudflare.API
	zones *ZoneService
}

// NewRateLimitService builds the service over an existing client.
func NewRateLimitService(cf *cloudflare.API, zones *ZoneService) (*RateLimitService, error) {
	if cf == nil {
		return nil, validationError("NewRateLimitService", "cloudflare client is required")
	}
	if zones == nil {
		return nil, validationError("NewRateLimitService", "zone service is required")
	}
	return &RateLimitService{cf: cf, zones: zones}, nil
}

// NewRateLimitServiceFromCreds builds the service with the knowledge
// transport wired into its client.
func NewRateLimitServiceFromCreds(accountID, apiToken string) (*RateLimitService, error) {
	if accountID == "" {
		return nil, validationError("NewRateLimitService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewRateLimitService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken,
		cloudflare.HTTPClient(&http.Client{Transport: &knowledge.Transport{}}))
	if err != nil {
		return nil, newError("NewRateLimitService", "failed to create cloudflare client", err)
	}
	zones, err := NewZoneServiceFromCreds(accountID, apiToken)
	if err != nil {
		return nil, err
	}
	return NewRateLimitService(cf, zones)
}

// cfRatelimitPhase is the Rulesets phase owning zone rate limiting.
const cfRatelimitPhase = "http_ratelimit"

// List returns the zone's rate-limiting rules. A missing phase entrypoint
// (fresh zone) is an empty list, not an error — this is the disambiguation
// CF's API does not offer.
func (s *RateLimitService) List(ctx context.Context, zoneID string) ([]RateLimitRule, error) {
	if zoneID == "" {
		return nil, validationError("RateLimitService.List", "zone ID is required")
	}
	rs, err := s.cf.GetEntrypointRuleset(ctx, cloudflare.ZoneIdentifier(zoneID), cfRatelimitPhase)
	if err != nil {
		if isNotFoundCF(err) {
			return []RateLimitRule{}, nil
		}
		return nil, newError("RateLimitService.List",
			fmt.Sprintf("failed to list rate-limit rules for zone %q", zoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}
	return fromCFRules(rs.Rules), nil
}

// Create appends one rate-limiting rule via the entrypoint PUT (cloudflare-go
// v0.116.0 exposes no per-rule create). Preflight validates caps + invariants
// before any API call; a fresh zone's missing entrypoint is created by the
// same PUT.
func (s *RateLimitService) Create(ctx context.Context, in RateLimitCreateInput) (*RateLimitRule, error) {
	if in.ZoneID == "" {
		return nil, validationError("RateLimitService.Create", "zone ID is required")
	}
	if in.Expression == "" {
		return nil, validationError("RateLimitService.Create", "expression is required")
	}

	plan := ""
	if z, err := s.zones.Get(ctx, in.ZoneID); err == nil && z != nil {
		plan = z.Plan.LegacyID
	}

	existing, err := s.List(ctx, in.ZoneID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"rules_count":                len(existing) + 1,
		"period_seconds":             in.Period,
		"mitigation_timeout_seconds": in.MitigationTimeout,
		"characteristics":            in.Characteristics,
	}
	if violations := knowledge.ValidatePayload("ratelimit", plan, payload); len(violations) > 0 {
		msgs := make([]string, 0, len(violations))
		for _, v := range violations {
			msgs = append(msgs, v.String())
		}
		return nil, validationError("RateLimitService.Create",
			fmt.Sprintf("payload rejected before send (plan %q): %s", plan, strings.Join(msgs, "; ")))
	}

	rc := cloudflare.ZoneIdentifier(in.ZoneID)
	rs, err := s.cf.GetEntrypointRuleset(ctx, rc, cfRatelimitPhase)
	rules := make([]cloudflare.RulesetRule, 0, len(existing)+1)
	if err == nil {
		rules = append(rules, rs.Rules...)
	} else if !isNotFoundCF(err) {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("failed to read entrypoint for zone %q", in.ZoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}

	enabled := true
	rules = append(rules, cloudflare.RulesetRule{
		Action:       "block",
		Expression:   in.Expression,
		Description:  in.Description,
		Enabled:      &enabled,
		RateLimit: &cloudflare.RulesetRuleRateLimit{
			Characteristics:    in.Characteristics,
			RequestsPerPeriod: in.RequestsPerPeriod,
			Period:            in.Period,
			MitigationTimeout: in.MitigationTimeout,
		},
	})

	updated, err := s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
		Phase: cfRatelimitPhase,
		Rules: rules,
	})
	if err != nil {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("failed to apply rate-limit rule to zone %q", in.ZoneID),
			knowledge.DecodeCFError(err, "phase-entrypoint"))
	}

	// The new rule is the one we appended.
	for _, r := range fromCFRules(updated.Rules) {
		if r.Expression == in.Expression && r.RequestsPerPeriod == in.RequestsPerPeriod {
			return &r, nil
		}
	}
	rules2 := fromCFRules(updated.Rules)
	if len(rules2) == 0 {
		return nil, newError("RateLimitService.Create",
			fmt.Sprintf("zone %q entrypoint returned zero rules after update", in.ZoneID), nil)
	}
	return &rules2[len(rules2)-1], nil
}

// fromCFRules maps SDK rules into our shape.
func fromCFRules(in []cloudflare.RulesetRule) []RateLimitRule {
	out := make([]RateLimitRule, 0, len(in))
	for _, r := range in {
		rr := RateLimitRule{
			ID:          r.ID,
			Expression:  r.Expression,
			Description: r.Description,
			Enabled:     r.Enabled == nil || *r.Enabled,
		}
		if r.RateLimit != nil {
			rr.Characteristics = r.RateLimit.Characteristics
			rr.RequestsPerPeriod = r.RateLimit.RequestsPerPeriod
			rr.Period = r.RateLimit.Period
			rr.MitigationTimeout = r.RateLimit.MitigationTimeout
		}
		out = append(out, rr)
	}
	return out
}

// isNotFoundCF reports CF not-found shapes: HTTP 404 or code 1000
// (cloudflare-go v0.116.0 has no ErrNotFound sentinel).
func isNotFoundCF(err error) bool {
	if err == nil {
		return false
	}
	var cfErr *cloudflare.Error
	if errors.As(err, &cfErr) {
		if cfErr.StatusCode == http.StatusNotFound {
			return true
		}
		for _, c := range cfErr.ErrorCodes {
			if c == 1000 {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestRateLimit -v && go build ./...`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/ratelimit.go pkg/cosmoflare/ratelimit_test.go
git commit -m "feat(ratelimit): RateLimitService — preflight + entrypoint PUT flow"
```

---

### Task 6: CLI — decode, knowledge list, ratelimit

**Files:**
- Create: `cmd/decode.go`
- Create: `cmd/knowledge_cmd.go`
- Create: `cmd/ratelimit.go`
- Create: `cmd/decode_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/decode_test.go`:

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeCommandPrintsCauseAndFix(t *testing.T) {
	var buf bytes.Buffer
	decodeCmd.SetOut(&buf)
	decodeCmd.SetArgs([]string{"10405"})
	if err := decodeCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "endpoint does not exist") || !strings.Contains(out, "fix") {
		t.Fatalf("decode output must carry cause and fix: %q", out)
	}
}

func TestDecodeCommandUnknownCodeFails(t *testing.T) {
	decodeCmd.SetArgs([]string{"99999"})
	if err := decodeCmd.Execute(); err == nil {
		t.Fatal("unknown code must error")
	}
}

func TestKnowledgeListCommandRegistered(t *testing.T) {
	if knowledgeCmd == nil || knowledgeCmd.Name() != "knowledge" {
		t.Fatal("knowledge command not registered")
	}
}

func TestRateLimitCommandsRegistered(t *testing.T) {
	for _, name := range []string{"list", "create"} {
		found := false
		for _, c := range rateLimitCmd.Commands() {
			if c.Name() == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("ratelimit subcommand %q missing", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run 'TestDecode|TestKnowledge|TestRateLimitCmd' -v`
Expected: FAIL — `undefined: decodeCmd` etc.

- [ ] **Step 3: Write minimal implementation**

Create `cmd/decode.go`:

```go
package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

var decodeContext string

var decodeCmd = &cobra.Command{
	Use:   "decode <code>",
	Short: "Decode a Cloudflare API error code into its cause and fix",
	Long: `Decode a Cloudflare API error code using cosmoflare's knowledge packs.

Prints the cause and the fix for codes the knowledge layer understands.
Unknown codes exit non-zero with a clear message — no fabricated verdicts.

Examples:
  cosmoflare decode 10405
  cosmoflare decode 1000 --context phase-entrypoint
  cosmoflare decode 20155 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid error code %q: expected an integer", args[0])
		}
		d := knowledge.LookupDecode(code, decodeContext)
		if d == nil {
			return fmt.Errorf("no knowledge entry for code %d (context %q) — the knowledge layer never fabricates a verdict", code, decodeContext)
		}
		if JSONOutput {
			return printJSON(d)
		}
		cmd.Printf("code:    %d\ncontext: %s\ncause:   %s\nfix:     %s\n", d.Code, d.Context, d.Cause, d.Fix)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(decodeCmd)
	decodeCmd.Flags().StringVar(&decodeContext, "context", "", "refine the decode by context (e.g. phase-entrypoint)")
}
```

Create `cmd/knowledge_cmd.go` (NOT `knowledge.go` — see the note below):

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Inspect the loaded Cloudflare knowledge packs",
	Long: `List the knowledge packs compiled into this binary.

Each pack carries an endpoint registry, error decodes, plan caps, and field
invariants for one Cloudflare product. Routes inside a pack's scope that are
not registered are blocked before send.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		packs, err := knowledge.Load()
		if err != nil {
			return fmt.Errorf("knowledge packs failed to load: %w", err)
		}
		if JSONOutput {
			return printJSON(packs)
		}
		for _, p := range packs {
			fmt.Printf("%s: %d endpoints, %d error decodes, %d plan caps, %d invariants (scopes: %v)\n",
				p.Product, len(p.Endpoints), len(p.Errors), len(p.PlanCaps), len(p.Invariants), p.Scopes)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(knowledgeCmd)
}
```

**Naming note:** the file is `cmd/knowledge_cmd.go` purely for readability —
the `knowledge` import alias and a file named `knowledge.go` do not collide
in Go, but distinct names scan better. Do not rename.

Create `cmd/ratelimit.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var rateLimitCmd = &cobra.Command{
	Use:   "ratelimit",
	Short: "Manage zone rate-limiting rules (Rulesets http_ratelimit phase)",
	Long: `Manage zone rate-limiting rules via the Rulesets http_ratelimit phase.

Create runs client-side preflight first: plan caps (Free: 1 rule/zone, 10s
window, 10s mitigation, IP-only counting) and the mandatory cf.colo.id
characteristic — violations stop before any API call.`,
}

var rateLimitListCmd = &cobra.Command{
	Use:   "list <zone-id-or-name>",
	Short: "List a zone's rate-limiting rules",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		rules, err := svc.List(cmd.Context(), zoneID)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(err.Error())
			}
			return err
		}
		if JSONOutput {
			return printSuccessJSON("rate-limiting rules listed", rules)
		}
		if len(rules) == 0 {
			fmt.Println("no rate-limiting rules (fresh zones have no entrypoint — this counts as zero)")
			return nil
		}
		for _, r := range rules {
			fmt.Printf("%s  %s  %d req/%ds block %ds  characteristics=%s\n",
				r.ID, r.Expression, r.RequestsPerPeriod, r.Period, r.MitigationTimeout,
				strings.Join(r.Characteristics, ","))
		}
		return nil
	},
}

var (
	rateLimitRequests  int
	rateLimitPeriod    int
	rateLimitTimeout   int
	rateLimitChars     string
	rateLimitDesc      string
)

var rateLimitCreateCmd = &cobra.Command{
	Use:   "create <zone-id-or-name>",
	Short: "Create a rate-limiting rule (preflight-validated)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		var chars []string
		for _, c := range strings.Split(rateLimitChars, ",") {
			if c = strings.TrimSpace(c); c != "" {
				chars = append(chars, c)
			}
		}
		rule, err := svc.Create(cmd.Context(), cosmoflare.RateLimitCreateInput{
			ZoneID:             zoneID,
			Expression:         ratelimitExpression,
			Description:        rateLimitDesc,
			RequestsPerPeriod:  rateLimitRequests,
			Period:             rateLimitPeriod,
			MitigationTimeout:  rateLimitTimeout,
			Characteristics:    chars,
		})
		if err != nil {
			if JSONOutput {
				return printErrorJSON(err.Error())
			}
			return err
		}
		if JSONOutput {
			return printSuccessJSON("rate-limiting rule created", rule)
		}
		fmt.Printf("created %s  %s  %d req/%ds block %ds\n",
			rule.ID, rule.Expression, rule.RequestsPerPeriod, rule.Period, rule.MitigationTimeout)
		return nil
	},
}

var ratelimitExpression string

// ratelimitServiceAndZone resolves the zone argument and builds the service.
// REUSE the existing resolveZoneID(ctx, domain) from cmd/bucket_domain.go:145
// (same `cmd` package — no import, no new helper). It delegates to
// ZoneService.ResolveIDForDomain and accepts a zone ID or name.
func ratelimitServiceAndZone(ctx context.Context, target string) (*cosmoflare.RateLimitService, string, error) {
	svc, err := cosmoflare.NewRateLimitServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create rate-limit service: %w", err)
	}
	zoneID, err := resolveZoneID(ctx, target)
	if err != nil {
		return nil, "", err
	}
	return svc, zoneID, nil
}

func init() {
	rootCmd.AddCommand(rateLimitCmd)
	rateLimitCmd.AddCommand(rateLimitListCmd)
	rateLimitCmd.AddCommand(rateLimitCreateCmd)

	rateLimitCreateCmd.Flags().StringVar(&ratelimitExpression, "expression", "",
		"traffic expression, e.g. 'path eq \"/catalog.json\"' (required)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitRequests, "requests", 10, "requests per period")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitPeriod, "period", 10, "period seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitTimeout, "timeout", 10, "mitigation timeout seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitChars, "characteristics", "cf.colo.id,ip.src",
		"comma-separated counting characteristics (must include cf.colo.id)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitDesc, "description", "", "rule description")
	_ = rateLimitCreateCmd.MarkFlagRequired("expression")
}
```

**Verified 2026-09-09:** `resolveZoneID(ctx, domain)` already exists at
`cmd/bucket_domain.go:145` and handles both zone IDs and names (it delegates
to `ZoneService.ResolveIDForDomain`). Reuse it — do not add a resolver. The
`context` import in `cmd/ratelimit.go` exists for the
`ratelimitServiceAndZone` signature.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run 'TestDecode|TestKnowledge|TestRateLimitCmd' -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "feat(cmd): decode, knowledge list, ratelimit list/create"
```

---

### Task 7: Documentation

**Files:**
- Modify: `docs/USAGE.md`

- [ ] **Step 1: Update USAGE.md**

1. Add a `## Knowledge Layer` section (after the Domains Command section): what packs are, the scoped route-check semantic (advisory when absent, authoritative when present), the four decoded codes with their causes, and the `cosmoflare decode` + `cosmoflare knowledge list` commands with `--json` examples.
2. Add a `## Rate Limiting Command` section: `ratelimit list` / `ratelimit create` with the preflight rules (Free caps, cf.colo.id), the fresh-zone empty-list semantic, and `--json` examples.
3. Cross-reference from the `## Doctor Command` section: CF error decoding is centralized — `cosmoflare decode <code>` explains any wrapped error code.

Match the surrounding heading/bullet style.

- [ ] **Step 2: Verify docs integrity**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add docs/USAGE.md
git commit -m "docs(knowledge): knowledge layer, decode, ratelimit commands"
```

---

## Verification (whole plan)

```bash
go build ./... && go vet ./...
go test ./pkg/cosmoflare/knowledge/ ./pkg/cosmoflare/ ./cmd/ -count=1 -timeout 180s
```

Expected: all PASS. No live network (httptest only); no fabricated verdicts
(unknown codes and unknown products pass through).

## Stop Conditions

- If `cloudflare.Error` value-type assertion fails at runtime against real
  wrapped errors (errors.As returning false where tests say true), STOP Task 3's
  green step, record the observed error chain shape in this plan, and adjust
  `DecodeCFError` — do not weaken the pass-through contract.
- If `ZoneService.List` in Task 6's `resolveZoneID` uses a different options
  type or pagination shape than assumed, adapt to the real signature — the
  helper's behavior (name-or-ID → ID) is the contract.

## Accepted limitations (review 2026-09-09, deliberate)

- **Transport fails OPEN on a pack-loader error** — if `Load()` fails (corrupt
  pack), `CheckRoute` returns a zero verdict and traffic passes. Deliberate:
  failing closed would block all API traffic on one bad pack file, which is
  worse than missing knowledge. The loader error still surfaces loudly at the
  first direct `Load()` call site (`cosmoflare knowledge list`, validators).
  This narrows the brainstorm's "corrupted knowledge never silently passes"
  to: never silently passes *where it is consulted directly*.
- **decode/knowledge commands load packs from the compiled binary** — no
  runtime pack installation in v1 (single-binary promise).
- **Global newError hook decorates only context-free decode entries** —
  contextual decodes (1000-on-entrypoint) surface only via services that pass
  context (RateLimitService does). Existing services adopt contextual decoding
  incrementally.
- **Transport wraps only the RateLimitServiceFromCreds client in v1** —
  retrofitting every FromCreds constructor is follow-up scope (one-line each,
  mechanical, safe to batch later).
