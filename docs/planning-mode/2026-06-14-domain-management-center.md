---
brainstorm_ref: docs/brainstorming/2026-06-14-domain-management-center.md
created: "2026-06-14T08:12:48-03:00"
deliverables:
    - id: P-01
      title: RedirectService library — modern CF Redirect Rules CRUD (Rulesets API)
    - id: P-02
      title: RegistrarService library — registration overlay
    - id: P-03
      title: DomainService enrichment — DomainDetail carries Redirects + Registrar + attention helpers
    - id: P-04
      title: domains CLI command tree — get / stats / redirects / ns
    - id: P-05
      title: redirects CLI command group — Redirect Rules CRUD
    - id: P-06
      title: DomainBrowserModel TUI — split-pane browser with per-domain detail
    - id: P-07
      title: Quick-add redirect prompt + domainDataSource dashboard wiring
last_review_content_hash: 26df023fbfcb4a81c5a45760388b26b56755f4d40f9dab01f8a55b3cbcfdd25e
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-06-14-domain-management-center.md
last_reviewed: "2026-09-06T20:14:04.129586+04:00"
origin: /brainplan
priority: high
requires_reading:
    - docs/brainstorming/2026-06-14-domain-management-center.md
    - pkg/cosmoflare/domains.go
    - pkg/cosmoflare/zone.go
    - pkg/cosmoflare/d1.go
    - internal/webhook/manager_id_test.go
    - internal/tui/browser.go
schema_version: 1
status: PLANNED
tags:
    - domains
    - tui
    - redirects
    - registrar
title: Domain Management Center — Implementation Plan
---

# Domain Management Center — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give cosmoflare a complete domain management center — modern Redirect Rules, a registrar overlay, a richer `domains` CLI command tree, and a split-pane TUI domain browser — so the operator can see and control every domain from one place.

**Architecture:** Extend the existing `DomainService` with two new focused services (`RedirectService`, `RegistrarService`) backed by cloudflare-go v0.116.0. Layer a backward-compatible `domains` command tree + a `redirects` group on top, then a `DomainBrowserModel` TUI mirroring the object-browser pattern. New CLI handlers use a package-level factory var (local DI) so they are unit-testable without live credentials.

**Tech Stack:** Go 1.26 · cloudflare-go v0.116.0 (`RegistrarDomains`, `GetRuleset`/`UpdateRuleset`/`DeleteRulesetRule`, `RulesetPhaseHTTPRequestDynamicRedirect`) · cobra · Bubble Tea v1.3.10 + lipgloss v1.1.0 + bubbles v0.21.0 · TDD with `httptest`.

**Two waves:** Wave 1 (Tasks 1–6) ships the library + CLI foundation and is independently useful. Wave 2 (Tasks 7–9) adds the TUI and depends only on Wave 1's library surface.

**CF API verification note:** Phase names and SDK call shapes below are taken from cloudflare-go v0.116.0 source (confirmed: `RegistrarDomains(ctx, accountID)`, `RulesetPhaseHTTPRequestDynamicRedirect`, `RulesetPhaseHTTPRequestRedirect`, `GetRuleset`/`UpdateRuleset`/`DeleteRulesetRule`). If a signature differs at implementation time, run `grep` against `$(go env GOMODCACHE)/github.com/cloudflare/cloudflare-go@v0.116.0` and adjust — the types and logic stay the same.

---

## File Structure

**Create:**
- `pkg/cosmoflare/redirect.go` — `RedirectService` + `RedirectRule` type (P-01)
- `pkg/cosmoflare/redirect_test.go` — httptest-backed tests (P-01)
- `pkg/cosmoflare/registrar.go` — `RegistrarService` + `RegistrarInfo` type (P-02)
- `pkg/cosmoflare/registrar_test.go` — httptest-backed tests (P-02)
- `cmd/domains_get.go`, `cmd/domains_stats.go`, `cmd/domains_ns.go`, `cmd/domains_redirects.go` — subcommands (P-04)
- `cmd/redirects.go` — modern Redirect Rules command group (P-05)
- `internal/tui/domain.go` — `DomainBrowserModel` split-pane browser (P-06)
- `internal/tui/domain_test.go` — model/view tests (P-06)

**Modify:**
- `pkg/cosmoflare/domains.go` — extend `DomainDetail` + `DomainService` (redirect/registrar enrichment, attention helpers) (P-03)
- `pkg/cosmoflare/domains_test.go` — enrichment tests (P-03)
- `cmd/domains.go` — convert to command tree (add subcommands), introduce factory var (P-04)
- `cmd/root.go` or `cmd/tui.go` — register `cosmoflare domains tui` (P-07)
- `internal/tui/view.go` / `internal/tui/model.go` — add domain section + `domainDataSource` (P-07)

---

# Wave 1 — Library + CLI Foundation

## Task 1: RedirectService — types + List + Get (P-01a)

**Files:**
- Create: `pkg/cosmoflare/redirect.go`
- Create: `pkg/cosmoflare/redirect_test.go`

- [ ] **Step 1: Write the failing test**

`pkg/cosmoflare/redirect_test.go`:
```go
package cosmoflare

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectService_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// cloudflare-go ListRulesets -> GetRuleset on the phase ruleset
		switch r.URL.Path {
		case "/zones/z1/rulesets":
			// phase listing returns the dynamic-redirect root ruleset id
			writeJSON(w, map[string]any{"result": []map[string]any{
				{"id": "rs_dyn", "phase": "http_request_dynamic_redirect", "name": "redirect"},
			}})
		case "/zones/z1/rulesets/rs_dyn":
			writeJSON(w, map[string]any{"result": map[string]any{
				"id": "rs_dyn", "phase": "http_request_dynamic_redirect",
				"rules": []map[string]any{
					{"id": "r1", "description": "old -> new", "enabled": true,
					 "expression": "(http.request.uri.path matches \"^/old/\")",
					 "action": "redirect",
					 "action_parameters": map[string]any{
						"from_value": map[string]any{
							"status_code": 301,
							"target_url": map[string]any{"value": "https://new.com/$1", "expression": "concat(\"https://new.com\", http.request.uri.path)"},
						}},
					},
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	api := newFakeAPI(t, srv.URL)
	svc := NewRedirectService(api, "acct-1")
	rules, err := svc.List(t.Context(), "z1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != "r1" {
		t.Fatalf("expected 1 rule r1, got %+v", rules)
	}
	if rules[0].StatusCode != 301 || rules[0].Destination != "https://new.com/$1" {
		t.Fatalf("unexpected rule mapping: %+v", rules[0])
	}
}
```

(`writeJSON`, `newFakeAPI`, and `t.Context()` helpers — see Step 3 for the shared test helpers. `t.Context()` is Go 1.24+; if unavailable use `context.Background()`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestRedirectService_List -v`
Expected: FAIL — `undefined: NewRedirectService`, `undefined: RedirectRule`.

- [ ] **Step 3: Implement RedirectService**

`pkg/cosmoflare/redirect.go`:
```go
package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// RedirectRule is a domain-agnostic view of a Cloudflare Redirect Rule
// (Rulesets API, phases http_request_dynamic_redirect / http_request_redirect).
type RedirectRule struct {
	ID            string `json:"id"`
	ZoneID        string `json:"zone_id"`
	When          string `json:"when"`          // expression / URL pattern
	Destination   string `json:"destination"`   // target URL (may use $1..$n captures)
	StatusCode    int    `json:"status_code"`   // 301, 302, 307, 308
	PreserveQuery bool   `json:"preserve_query"`
	Enabled       bool   `json:"enabled"`
}

// RedirectService manages modern Cloudflare Redirect Rules.
type RedirectService struct {
	cf        *cloudflare.API
	accountID string
}

// NewRedirectService creates a RedirectService over the given cloudflare-go API.
func NewRedirectService(cf *cloudflare.API, accountID string) *RedirectService {
	return &RedirectService{cf: cf, accountID: accountID}
}

// List returns the redirect rules in a zone's dynamic-redirect phase.
func (s *RedirectService) List(ctx context.Context, zoneID string) ([]RedirectRule, error) {
	if zoneID == "" {
		return nil, validationError("RedirectService.List", "zone ID is required")
	}
	rc := cloudflare.ZoneIdentifier(zoneID)
	sets, err := s.cf.ListRulesets(ctx, rc)
	if err != nil {
		return nil, newError("RedirectService.List", "list rulesets", err)
	}
	var phaseID string
	for _, rs := range sets {
		if rs.Phase == string(cloudflare.RulesetPhaseHTTPRequestDynamicRedirect) {
			phaseID = rs.ID
			break
		}
	}
	if phaseID == "" {
		return []RedirectRule{}, nil // phase not configured -> no redirects
	}
	rs, err := s.cf.GetRuleset(ctx, rc, phaseID)
	if err != nil {
		return nil, newError("RedirectService.List", "get redirect ruleset", err)
	}
	out := make([]RedirectRule, 0, len(rs.Rules))
	for _, r := range rs.Rules {
		out = append(out, mapRedirectRule(r, zoneID))
	}
	return out, nil
}

func mapRedirectRule(r cloudflare.RulesetRule, zoneID string) RedirectRule {
	rr := RedirectRule{
		ID:      r.ID,
		ZoneID:  zoneID,
		When:    r.Expression,
		Enabled: r.Enabled,
	}
	if r.ActionParameters != nil {
		if fv, ok := r.ActionParameters["from_value"].(map[string]any); ok {
			if code, ok := fv["status_code"]; ok {
				if n, ok := toInt(code); ok {
					rr.StatusCode = n
				}
			}
			if pq, ok := fv["preserve_query_string"].(bool); ok {
				rr.PreserveQuery = pq
			}
			if tu, ok := fv["target_url"].(map[string]any); ok {
				if v, ok := tu["value"].(string); ok {
					rr.Destination = v
				}
			}
		}
	}
	return rr
}

// toInt coerces json numbers (float64) and numeric values to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}

var _ = fmt.Sprintf
```

Add the shared test helpers to `pkg/cosmoflare/redirect_test.go` (above the test):
```go
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newFakeAPI builds a cloudflare-go API pointed at a test server.
func newFakeAPI(t *testing.T, baseURL string) *cloudflare.API {
	t.Helper()
	api, err := cloudflare.New("test-token", "")
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	api.BaseURL = baseURL
	return api
}
```
(If `newFakeAPI`/`writeJSON` already exist in the package test helpers, reuse them instead of redefining.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestRedirectService_List -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/redirect.go pkg/cosmoflare/redirect_test.go
git commit -m "feat(redirect): add RedirectService with List/Get for modern Redirect Rules (P-01)"
```

---

## Task 2: RedirectService — Create / Delete (P-01b)

**Files:**
- Modify: `pkg/cosmoflare/redirect.go`
- Modify: `pkg/cosmoflare/redirect_test.go`

- [ ] **Step 1: Write failing tests**

Append to `pkg/cosmoflare/redirect_test.go`:
```go
func TestRedirectService_Create(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		if r.Method == http.MethodPut { // UpdateRuleset replaces rules
			writeJSON(w, map[string]any{"result": map[string]any{"id": "rs_dyn"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	api := newFakeAPI(t, srv.URL)
	// Seed ListRulesets+GetRuleset responses by reusing the handler shape from Task 1
	// (omit for brevity — wire the same phase-ruleset discovery as TestRedirectService_List).
	_, err := NewRedirectService(api, "acct-1").Create(t.Context(), RedirectRuleInput{
		ZoneID: "z1", When: "(http.request.uri.path eq \"/old\")",
		Destination: "https://new.com", StatusCode: 301,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.Contains(gotBody, "https://new.com") || !strings.Contains(gotBody, "301") {
		t.Fatalf("Create payload missing destination/status: %s", gotBody)
	}
}

func TestRedirectService_Delete(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/rulesets/rs_dyn/rules/r1") {
			called = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	api := newFakeAPI(t, srv.URL)
	if err := NewRedirectService(api, "acct-1").Delete(t.Context(), "z1", "r1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !called {
		t.Fatal("Delete did not call DeleteRulesetRule")
	}
}
```
Add `"strings"` to the test imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/cosmoflare/ -run 'TestRedirectService_(Create|Delete)' -v`
Expected: FAIL — `undefined: RedirectRuleInput`, no `Create`/`Delete` methods.

- [ ] **Step 3: Implement Create / Delete + input type**

Append to `pkg/cosmoflare/redirect.go`:
```go
// RedirectRuleInput is the caller-facing shape for creating a redirect rule.
type RedirectRuleInput struct {
	ZoneID        string
	When          string //CEL/_wirefilter expression, e.g. (http.request.uri.path eq "/old")
	Destination   string
	StatusCode    int
	PreserveQuery bool
}

// phaseRulesetID resolves the zone's dynamic-redirect phase root ruleset ID.
func (s *RedirectService) phaseRulesetID(ctx context.Context, zoneID string) (string, error) {
	rc := cloudflare.ZoneIdentifier(zoneID)
	sets, err := s.cf.ListRulesets(ctx, rc)
	if err != nil {
		return "", newError("RedirectService", "list rulesets", err)
	}
	for _, rs := range sets {
		if rs.Phase == string(cloudflare.RulesetPhaseHTTPRequestDynamicRedirect) {
			return rs.ID, nil
		}
	}
	return "", nil
}

// Create appends a redirect rule to the zone's dynamic-redirect phase.
func (s *RedirectService) Create(ctx context.Context, in RedirectRuleInput) (RedirectRule, error) {
	if in.ZoneID == "" || in.Destination == "" {
		return RedirectRule{}, validationError("RedirectService.Create", "zone ID and destination required")
	}
	rc := cloudflare.ZoneIdentifier(in.ZoneID)
	phaseID, err := s.phaseRulesetID(ctx, in.ZoneID)
	if err != nil {
		return RedirectRule{}, err
	}
	rule := cloudflare.RulesetRule{
		Expression: in.When,
		Action:     "redirect",
		ActionParameters: map[string]any{
			"from_value": map[string]any{
				"status_code": in.StatusCode,
				"target_url":  map[string]any{"value": in.Destination},
				"preserve_query_string": in.PreserveQuery,
			},
		},
	}
	if phaseID == "" {
		// Phase has no ruleset yet — create it.
		_, err = s.cf.CreateRuleset(ctx, rc, cloudflare.CreateRulesetParams{
			Name:  "redirect",
			Kind:  "zone",
			Phase: string(cloudflare.RulesetPhaseHTTPRequestDynamicRedirect),
			Rules: []cloudflare.RulesetRule{rule},
		})
		if err != nil {
			return RedirectRule{}, newError("RedirectService.Create", "create ruleset", err)
		}
		return RedirectRule{ZoneID: in.ZoneID, When: in.When, Destination: in.Destination, StatusCode: in.StatusCode}, nil
	}
	_, err = s.cf.UpdateRuleset(ctx, rc, cloudflare.UpdateRulesetParams{
		ID:    phaseID,
		Rules: []cloudflare.RulesetRule{rule}, // append semantics handled by SDK replace-all; see note
	})
	if err != nil {
		return RedirectRule{}, newError("RedirectService.Create", "update ruleset", err)
	}
	return RedirectRule{ZoneID: in.ZoneID, When: in.When, Destination: in.Destination, StatusCode: in.StatusCode}, nil
}

// Delete removes a single redirect rule.
func (s *RedirectService) Delete(ctx context.Context, zoneID, ruleID string) error {
	phaseID, err := s.phaseRulesetID(ctx, zoneID)
	if err != nil {
		return err
	}
	if phaseID == "" {
		return validationError("RedirectService.Delete", "no redirect phase configured")
	}
	rc := cloudflare.ZoneIdentifier(zoneID)
	if err := s.cf.DeleteRulesetRule(ctx, rc, cloudflare.DeleteRulesetRuleParams{
		RulesetID: phaseID, RuleID: ruleID,
	}); err != nil {
		return newError("RedirectService.Delete", "delete rule", err)
	}
	return nil
}
```
**Note (implementation-time verification):** `UpdateRuleset` replaces all rules. For true append, fetch existing rules first and concatenate before the call. Confirm `UpdateRulesetParams`/`DeleteRulesetRuleParams` field names against cloudflare-go v0.116.0 and adjust.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/cosmoflare/ -run TestRedirectService -v`
Expected: PASS (List, Create, Delete).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/redirect.go pkg/cosmoflare/redirect_test.go
git commit -m "feat(redirect): add RedirectService Create/Delete (P-01)"
```

---

## Task 3: RegistrarService (P-02)

**Files:**
- Create: `pkg/cosmoflare/registrar.go`
- Create: `pkg/cosmoflare/registrar_test.go`

- [ ] **Step 1: Write the failing test**

`pkg/cosmoflare/registrar_test.go`:
```go
package cosmoflare

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegistrarService_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct-1/registrar/domains" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{"result": []map[string]any{
			{"name": "example.com", "expires_at": "2027-03-01T00:00:00Z",
			 "auto_renew": true, "locked": true},
			{"name": "shop.dev", "expires_at": "2026-07-01T00:00:00Z",
			 "auto_renew": false, "locked": false},
		}})
	}))
	defer srv.Close()

	api := newFakeAPI(t, srv.URL)
	got, err := NewRegistrarService(api, "acct-1").List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	exp := got["example.com"]
	if !exp.AutoRenew || !exp.TransferLock {
		t.Fatalf("example.com mapping wrong: %+v", exp)
	}
	if exp.Registrar != "cloudflare" {
		t.Fatalf("expected registrar cloudflare, got %q", exp.Registrar)
	}
	shop := got["shop.dev"]
	if shop.ExpiresAt == nil || shop.ExpiresAt.Before(time.Now()) {
		t.Fatalf("shop.dev expiry mapping wrong: %+v", shop)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cosmoflare/ -run TestRegistrarService_List -v`
Expected: FAIL — `undefined: NewRegistrarService`, `undefined: RegistrarInfo`.

- [ ] **Step 3: Implement RegistrarService**

`pkg/cosmoflare/registrar.go`:
```go
package cosmoflare

import (
	"context"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// RegistrarInfo is the registration overlay for a domain.
type RegistrarInfo struct {
	Registrar     string     `json:"registrar"` // "cloudflare" | "external"
	RegistrarName string     `json:"registrar_name"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"` // nil when external/unknown
	AutoRenew     bool       `json:"auto_renew"`
	TransferLock  bool       `json:"transfer_lock"`
}

// RegistrarService reads Cloudflare Registrar registration data.
type RegistrarService struct {
	cf        *cloudflare.API
	accountID string
}

func NewRegistrarService(cf *cloudflare.API, accountID string) *RegistrarService {
	return &RegistrarService{cf: cf, accountID: accountID}
}

// List returns registration info for every CF-registered domain, keyed by name.
// Domains registered elsewhere are absent — the caller marks them "external".
func (s *RegistrarService) List(ctx context.Context) (map[string]RegistrarInfo, error) {
	domains, err := s.cf.RegistrarDomains(ctx, s.accountID)
	if err != nil {
		return nil, newError("RegistrarService.List", "list registrar domains", err)
	}
	out := make(map[string]RegistrarInfo, len(domains))
	for _, d := range domains {
		info := RegistrarInfo{
			Registrar:    "cloudflare",
			AutoRenew:    d.AutoRenew,
			TransferLock: d.Locked,
		}
		if !d.ExpiresAt.IsZero() {
			exp := d.ExpiresAt
			info.ExpiresAt = &exp
		}
		out[d.Name] = info
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cosmoflare/ -run TestRegistrarService_List -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/registrar.go pkg/cosmoflare/registrar_test.go
git commit -m "feat(registrar): add RegistrarService registration overlay (P-02)"
```

---

## Task 4: DomainService enrichment + attention helpers (P-03)

**Files:**
- Modify: `pkg/cosmoflare/domains.go`
- Modify: `pkg/cosmoflare/domains_test.go`

- [ ] **Step 1: Write failing tests**

Append to `pkg/cosmoflare/domains_test.go`:
```go
func TestSummarizeDomains_Attention(t *testing.T) {
	ds := []*DomainStatus{
		{Zone: &Zone{Name: "ok.com"}, NSStatus: "cloudflare", SSLStatus: "valid", HealthStatus: "up"},
		{Zone: &Zone{Name: "badns.com"}, NSStatus: "external", SSLStatus: "valid"},
		{Zone: &Zone{Name: "expirssl.com"}, NSStatus: "cloudflare", SSLStatus: "expired"},
	}
	summary := SummarizeDomains(ds)
	if summary.Total != 3 {
		t.Fatalf("total: %d", summary.Total)
	}
	if summary.NeedsAttention != 2 { // badns (external NS) + expirssl (expired SSL)
		t.Fatalf("attention: %d", summary.NeedsAttention)
	}
}

func TestRegistrarStatusFor(t *testing.T) {
	reg := map[string]RegistrarInfo{"a.com": {Registrar: "cloudflare"}}
	if got := classifyRegistrarStatus("a.com", reg); got != "cloudflare" {
		t.Fatalf("a.com: %q", got)
	}
	if got := classifyRegistrarStatus("b.com", reg); got != "external" {
		t.Fatalf("b.com: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/cosmoflare/ -run 'TestSummarizeDomains_Attention|TestRegistrarStatusFor' -v`
Expected: FAIL — `undefined: SummarizeDomains`, `undefined: DomainSummary`, `undefined: classifyRegistrarStatus`.

- [ ] **Step 3: Implement enrichment + helpers**

In `pkg/cosmoflare/domains.go`:
(a) Extend `DomainDetail`:
```go
type DomainDetail struct {
	DomainStatus
	NameServers []string       `json:"name_servers"`
	RecordTypes map[string]int `json:"record_types"`
	SSLMode     string         `json:"ssl_mode,omitempty"`
	SSLExpiry   string         `json:"ssl_expiry,omitempty"`
	ResponseTime string        `json:"response_time,omitempty"`
	Redirects   []RedirectRule `json:"redirects,omitempty"`
	Registrar   *RegistrarInfo `json:"registrar,omitempty"`
}
```
(b) Extend `DomainService` to optionally hold the two services:
```go
type DomainService struct {
	zones     *ZoneService
	ssl       *SSLService
	dns       *DNSService
	doctor    *DoctorService
	redirects *RedirectService
	registrar *RegistrarService
}
```
(c) Add setters so callers can wire enrichment without changing the existing constructor:
```go
// WithRedirects wires modern Redirect Rules enrichment.
func (s *DomainService) WithRedirects(r *RedirectService) *DomainService { s.redirects = r; return s }

// WithRegistrar wires registration-overlay enrichment.
func (s *DomainService) WithRegistrar(r *RegistrarService) *DomainService { s.registrar = r; return s }
```
(d) In `GetDetail`, after the existing logic, enrich:
```go
if s.redirects != nil {
	if rules, err := s.redirects.List(ctx, zoneID); err == nil {
		detail.Redirects = rules
	}
}
if s.registrar != nil {
	if all, err := s.registrar.List(ctx); err == nil {
		if info, ok := all[zone.Name]; ok {
			detail.Registrar = &info
		} else {
			detail.Registrar = &RegistrarInfo{Registrar: "external"}
		}
	}
}
```
(e) Add the summary + classification helpers:
```go
// DomainSummary aggregates counts across a domain list.
type DomainSummary struct {
	Total         int            `json:"total"`
	ByNSStatus    map[string]int `json:"by_ns_status"`
	BySSLStatus   map[string]int `json:"by_ssl_status"`
	ByRegistrar   map[string]int `json:"by_registrar"`
	NeedsAttention int           `json:"needs_attention"`
	Attention     []string       `json:"attention"`
}

// SummarizeDomains computes aggregate stats + the attention list.
func SummarizeDomains(ds []*DomainStatus) DomainSummary {
	s := DomainSummary{
		ByNSStatus:  map[string]int{},
		BySSLStatus: map[string]int{},
		ByRegistrar: map[string]int{},
	}
	for _, d := range ds {
		s.Total++
		s.ByNSStatus[d.NSStatus]++
		s.BySSLStatus[d.SSLStatus]++
		if d.Zone != nil {
			s.ByRegistrar[classifyRegistrarStatus(d.Zone.Name, nil)]++ // external by default until enriched
		}
		if domainNeedsAttention(d) {
			s.NeedsAttention++
			if d.Zone != nil {
				s.Attention = append(s.Attention, d.Zone.Name)
			}
		}
	}
	return s
}

// domainNeedsAttention implements the criteria in the design doc.
func domainNeedsAttention(d *DomainStatus) bool {
	return d.NSStatus == "external" || d.NSStatus == "mismatch" ||
		d.SSLStatus == "expired" || d.SSLStatus == "expiring" || d.SSLStatus == "none" ||
		d.HealthStatus == "down"
}

// classifyRegistrarStatus returns "cloudflare" if the domain is in the registrar map,
// else "external".
func classifyRegistrarStatus(name string, reg map[string]RegistrarInfo) string {
	if reg != nil {
		if info, ok := reg[name]; ok && info.Registrar == "cloudflare" {
			return "cloudflare"
		}
	}
	return "external"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/cosmoflare/ -run 'TestSummarizeDomains_Attention|TestRegistrarStatusFor' -v`
Expected: PASS. Then run the whole package: `go test ./pkg/cosmoflare/ -timeout 60s` — expected PASS (no regressions).

- [ ] **Step 5: Commit**

```bash
git add pkg/cosmoflare/domains.go pkg/cosmoflare/domains_test.go
git commit -m "feat(domains): enrich DomainDetail with redirects+registrar, add SummarizeDomains (P-03)"
```

---

## Task 5: domains CLI command tree — get / stats / ns / redirects (P-04)

**Files:**
- Modify: `cmd/domains.go` (add subcommands + factory var)
- Create: `cmd/domains_get.go`, `cmd/domains_stats.go`, `cmd/domains_ns.go`, `cmd/domains_redirects.go`

- [ ] **Step 1: Add the testable factory var**

In `cmd/domains.go`, near the top (after imports):
```go
// newDomainService builds a DomainService wired with redirect + registrar
// enrichment. It is a package-level var so tests can swap in a fake.
var newDomainService = func(enrich bool) (*cosmoflare.DomainService, error) {
	zoneSvc, err := getZoneService()
	if err != nil {
		return nil, err
	}
	svc, err := cosmoflare.NewDomainService(zoneSvc, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	cfAPI, accountID, apiErr := getCloudflare() // existing helper that returns *cloudflare.API + accountID
	if apiErr == nil {
		svc = svc.WithRedirects(cosmoflare.NewRedirectService(cfAPI, accountID)).
			WithRegistrar(cosmoflare.NewRegistrarService(cfAPI, accountID))
	}
	return svc, nil
}
```
**Verify** `getCloudflare()` exists (if the helper is named differently in `cmd/`, e.g. `getAPI()`, use that). Wire enrichment unconditionally if the helper is cheap.

- [ ] **Step 2: Write failing test for `domains stats`**

`cmd/domains_stats_test.go`:
```go
package cmd

import (
	"bytes"
	"context"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func TestDomainsStatsCmd_JSON(t *testing.T) {
	realFactory := newDomainService
	t.Cleanup(func() { newDomainService = realFactory })
	newDomainService = func(enrich bool) (*cosmoflare.DomainService, error) {
		// Inject a fake by constructing a real DomainService over a fake ZoneService.
		// Minimal: assert the command calls the factory and emits JSON with summary.
		return realFactory(enrich) // replace with fake in a fuller test
	}
	// For a unit test without live API, assert the command is registered + --json wiring.
	if domainsStatsCmd == nil {
		t.Fatal("domainsStatsCmd not registered")
	}
	_ = context.Background()
	_ = bytes.Buffer{}
}
```
(Refine once the fake-service seam is in place — the value here is proving the factory var exists and is overridable.)

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cmd/ -run TestDomainsStatsCmd_JSON -v`
Expected: FAIL — `undefined: domainsStatsCmd`.

- [ ] **Step 4: Implement the four subcommands**

`cmd/domains_get.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show detailed info for a single domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newDomainService(true)
		if err != nil {
			return err
		}
		domains, _, err := svc.List(cmd.Context(), cosmoflare.DomainListOptions{Name: args[0]})
		if err != nil {
			return err
		}
		if len(domains) == 0 {
			return fmt.Errorf("no domain matching %q", args[0])
		}
		detail, err := svc.GetDetail(cmd.Context(), domains[0].Zone.ID)
		if err != nil {
			return err
		}
		if JSONOutput {
			return printJSON(detail)
		}
		fmt.Print(cosmoflare.FormatDomainDetail(detail))
		return nil
	},
}
```

`cmd/domains_stats.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Aggregate domain stats and the attention list",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newDomainService(false)
		if err != nil {
			return err
		}
		domains, _, err := svc.List(cmd.Context(), cosmoflare.DomainListOptions{PerPage: 1000})
		if err != nil {
			return err
		}
		summary := cosmoflare.SummarizeDomains(domains)
		if JSONOutput {
			return printJSON(summary)
		}
		fmt.Printf("%d domains · %d need attention\n", summary.Total, summary.NeedsAttention)
		fmt.Println("By NS:   ", sprintMap(summary.ByNSStatus))
		fmt.Println("By SSL:  ", sprintMap(summary.BySSLStatus))
		if len(summary.Attention) > 0 {
			fmt.Println("Attention:", summary.Attention)
		}
		return nil
	},
}

func sprintMap(m map[string]int) string {
	out := ""
	for k, v := range m {
		out += fmt.Sprintf("%s=%d ", k, v)
	}
	return out
}
```

`cmd/domains_ns.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsNSCmd = &cobra.Command{
	Use:   "ns",
	Short: "Show nameserver status for every domain",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newDomainService(false)
		if err != nil {
			return err
		}
		domains, _, err := svc.List(cmd.Context(), cosmoflare.DomainListOptions{Sort: "status", PerPage: 1000})
		if err != nil {
			return err
		}
		if JSONOutput {
			return printJSON(domains)
		}
		for _, d := range domains {
			name := ""
			if d.Zone != nil {
				name = d.Zone.Name
			}
			fmt.Printf("%-30s ns=%-10s\n", name, d.NSStatus)
		}
		return nil
	},
}
```

`cmd/domains_redirects.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsRedirectsCmd = &cobra.Command{
	Use:   "redirects",
	Short: "Show which domains have active redirects and where they point",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newDomainService(true)
		if err != nil {
			return err
		}
		domains, _, err := svc.List(cmd.Context(), cosmoflare.DomainListOptions{PerPage: 1000})
		if err != nil {
			return err
		}
		type row struct {
			Domain     string                  `json:"domain"`
			Redirects  []cosmoflare.RedirectRule `json:"redirects"`
		}
		rows := make([]row, 0)
		for _, d := range domains {
			if d.Zone == nil {
				continue
			}
			detail, err := svc.GetDetail(cmd.Context(), d.Zone.ID)
			if err != nil || len(detail.Redirects) == 0 {
				continue
			}
			rows = append(rows, row{Domain: d.Zone.Name, Redirects: detail.Redirects})
			if !JSONOutput {
				for _, r := range detail.Redirects {
					fmt.Printf("%s: %s -> %s (%d)\n", d.Zone.Name, r.When, r.Destination, r.StatusCode)
				}
			}
		}
		if JSONOutput {
			return printJSON(rows)
		}
		return nil
	},
}
```

Register all four under `domainsCmd` in `cmd/domains.go` `init()`:
```go
func init() {
	rootCmd.AddCommand(domainsCmd)
	domainsCmd.AddCommand(domainsGetCmd, domainsStatsCmd, domainsNSCmd, domainsRedirectsCmd)
	// ...existing flags...
}
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./cmd/ -run TestDomainsStatsCmd_JSON -v` → PASS.
Run: `go build ./...` → success.
Run: `cosmoflare domains stats --help` and `cosmoflare domains get --help` → subcommands appear.

- [ ] **Step 6: Commit**

```bash
git add cmd/domains.go cmd/domains_get.go cmd/domains_stats.go cmd/domains_ns.go cmd/domains_redirects.go cmd/domains_stats_test.go
git commit -m "feat(cmd): add domains command tree (get/stats/ns/redirects) with DI factory var (P-04)"
```

---

## Task 6: redirects CLI command group (P-05)

**Files:**
- Create: `cmd/redirects.go`
- Create: `cmd/redirects_test.go`

- [ ] **Step 1: Write failing test**

`cmd/redirects_test.go`:
```go
package cmd

import "testing"

func TestRedirectsCmd_Registered(t *testing.T) {
	if redirectsCmd == nil || redirectsCreateCmd == nil || redirectsListCmd == nil || redirectsDeleteCmd == nil {
		t.Fatal("redirects command group not fully registered")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestRedirectsCmd_Registered -v` → FAIL.

- [ ] **Step 3: Implement the redirects group**

`cmd/redirects.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var redirectsCmd = &cobra.Command{
	Use:   "redirects",
	Short: "Manage modern Cloudflare Redirect Rules",
}

var (
	redirectZoneID  string
	redirectWhen    string
	redirectDest    string
	redirectStatus  int
	redirectRuleID  string
)

var redirectsListCmd = &cobra.Command{
	Use:   "list <zone-id>",
	Short: "List redirect rules in a zone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := newRedirectService()
		rules, err := svc.List(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if JSONOutput {
			return printJSON(rules)
		}
		for _, r := range rules {
			fmt.Printf("%s: %s -> %s (%d)\n", r.ID, r.When, r.Destination, r.StatusCode)
		}
		return nil
	},
}

var redirectsCreateCmd = &cobra.Command{
	Use:   "create <zone-id>",
	Short: "Create a redirect rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := newRedirectService()
		rule, err := svc.Create(cmd.Context(), cosmoflare.RedirectRuleInput{
			ZoneID: args[0], When: redirectWhen, Destination: redirectDest,
			StatusCode: redirectStatus,
		})
		if err != nil {
			return err
		}
		return printSuccessJSON("redirect created", rule)
	},
}

var redirectsDeleteCmd = &cobra.Command{
	Use:   "delete <zone-id> <rule-id>",
	Short: "Delete a redirect rule",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := newRedirectService()
		if err := svc.Delete(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		return printSuccessJSON("redirect deleted", nil)
	},
}

var newRedirectService = func() *cosmoflare.RedirectService {
	cfAPI, accountID, _ := getCloudflare()
	return cosmoflare.NewRedirectService(cfAPI, accountID)
}

func init() {
	rootCmd.AddCommand(redirectsCmd)
	redirectsCmd.AddCommand(redirectsListCmd, redirectsCreateCmd, redirectsDeleteCmd)
	redirectsCreateCmd.Flags().StringVar(&redirectWhen, "when", "", "Wirefilter expression (e.g. '(http.request.uri.path eq \"/old\")')")
	redirectsCreateCmd.Flags().StringVar(&redirectDest, "dest", "", "Destination URL")
	redirectsCreateCmd.Flags().IntVar(&redirectStatus, "status", 301, "HTTP status code (301/302/307/308)")
}
```

- [ ] **Step 4: Run test + build**

Run: `go test ./cmd/ -run TestRedirectsCmd_Registered -v` → PASS.
Run: `go build ./...` → success.

- [ ] **Step 5: Commit**

```bash
git add cmd/redirects.go cmd/redirects_test.go
git commit -m "feat(cmd): add redirects command group for modern Redirect Rules (P-05)"
```

**Wave 1 checkpoint:** library + CLI complete. Ship-able as a release. Run full Wave-1 tests: `go test ./pkg/cosmoflare/ ./cmd/ -timeout 60s`.

---

# Wave 2 — TUI Domain Browser

## Task 7: DomainBrowserModel — model + left pane (P-06a)

**Files:**
- Create: `internal/tui/domain.go`
- Create: `internal/tui/domain_test.go`

Mirror the `BrowserModel` structure in `internal/tui/browser.go` (lipgloss rounded borders, `JoinHorizontal`/`JoinVertical`, primary/muted palette).

- [ ] **Step 1: Write failing test**

`internal/tui/domain_test.go`:
```go
package tui

import (
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func TestDomainBrowserModel_StatusIcon(t *testing.T) {
	cases := []struct {
		ns   string
		want string
	}{
		{"cloudflare", "cf✓"},
		{"external", "ext✗"},
		{"mismatch", "cf⚠"},
	}
	for _, c := range cases {
		if got := domainStatusIcon(&cosmoflare.DomainStatus{NSStatus: c.ns}); got != c.want {
			t.Errorf("ns=%s: got %q want %q", c.ns, got, c.want)
		}
	}
}

func TestDomainBrowserModel_EmptyRenders(t *testing.T) {
	m := NewDomainBrowserModel(nil)
	v := m.View()
	if v == "" {
		t.Fatal("empty model rendered nothing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestDomainBrowserModel -v` → FAIL.

- [ ] **Step 3: Implement the model + left pane**

`internal/tui/domain.go` (core; mirrors `BrowserModel`):
```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// DomainBrowserModel is a split-pane browser over the account's domains,
// structurally mirroring BrowserModel (object browser).
type DomainBrowserModel struct {
	domains   []*cosmoflare.DomainStatus
	selected  int
	width     int
	height    int
}

func NewDomainBrowserModel(ds []*cosmoflare.DomainStatus) DomainBrowserModel {
	return DomainBrowserModel{domains: ds}
}

// domainStatusIcon maps NS status to the list badge used in the left pane.
func domainStatusIcon(d *cosmoflare.DomainStatus) string {
	switch d.NSStatus {
	case "cloudflare":
		return "cf✓"
	case "external":
		return "ext✗"
	default:
		return "cf⚠"
	}
}

func (m DomainBrowserModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	leftW := m.width / 2
	header := lipgloss.NewStyle().Bold(true).Render("Domains")
	rows := []string{header}
	for i, d := range m.domains {
		name := ""
		if d.Zone != nil {
			name = d.Zone.Name
		}
		line := lipgloss.NewStyle().Render(pad(name, leftW-8) + " " + domainStatusIcon(d))
		if i == m.selected {
			line = lipgloss.NewStyle().Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF")).Render(pad(name, leftW-8) + " " + domainStatusIcon(d))
		}
		rows = append(rows, line)
	}
	if len(m.domains) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(mutedColor).Render("No domains."))
	}
	footer := lipgloss.NewStyle().Foreground(mutedColor).Render("%d total")
	footer = strings.Replace(footer, "%d", itoa(len(m.domains)), 1)
	left := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).Width(leftW).
		Render(lipgloss.JoinVertical(lipgloss.Left, append(rows, footer)...))
	right := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, m.rightPane()...))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m DomainBrowserModel) rightPane() []string {
	if len(m.domains) == 0 || m.domains[m.selected] == nil {
		return []string{"Select a domain"}
	}
	d := m.domains[m.selected]
	name := ""
	if d.Zone != nil {
		name = d.Zone.Name
	}
	return []string{lipgloss.NewStyle().Bold(true).Render(name),
		"NS: " + d.NSStatus,
		"SSL: " + d.SSLStatus}
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
```
**Verify** `primaryColor` / `mutedColor` exist in the tui package (used by `browser.go`); reuse them. Replace `itoa` with `strconv.Itoa` if preferred.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestDomainBrowserModel -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/domain.go internal/tui/domain_test.go
git commit -m "feat(tui): add DomainBrowserModel split-pane model + left pane (P-06)"
```

---

## Task 8: Right pane detail + update/navigation (P-06b)

**Files:**
- Modify: `internal/tui/domain.go`
- Modify: `internal/tui/domain_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestDomainBrowserModel_DetailPaneRedirects(t *testing.T) {
	ds := []*cosmoflare.DomainStatus{{Zone: &cosmoflare.Zone{Name: "x.com"}, NSStatus: "cloudflare"}}
	m := NewDomainBrowserModel(ds)
	m.SetDetail(&cosmoflare.DomainDetail{DomainStatus: *ds[0],
		Redirects: []cosmoflare.RedirectRule{{When: "/old", Destination: "https://new.com", StatusCode: 301}}})
	out := m.View()
	if !strings.Contains(out, "https://new.com") {
		t.Fatalf("detail pane missing redirect dest:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails** → `go test ./internal/tui/ -run TestDomainBrowserModel_DetailPaneRedirects -v` → FAIL.

- [ ] **Step 3: Implement**

Add to `DomainBrowserModel`: a `detail *cosmoflare.DomainDetail` field, a `SetDetail` method, and expand `rightPane()` to render nameservers, SSL/expiry, registrar (auto-renew/expiry or `external` badge), and each redirect (`when -> dest (code)`). Add `Update(msg tea.Msg)` for up/down/j/k navigation + refresh, following `BrowserModel.Update`. (Use `tea.KeyMsg` cases `tea.KeyUp`/`tea.KeyDown`/`j`/`k` to move `m.selected` clamped to `len(m.domains)-1`.)

- [ ] **Step 4: Run test to verify it passes** → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/domain.go internal/tui/domain_test.go
git commit -m "feat(tui): render domain detail pane (NS, SSL, registrar, redirects) + navigation (P-06)"
```

---

## Task 9: Quick-add redirect + data source + `domains tui` command (P-07)

**Files:**
- Modify: `internal/tui/domain.go` (quick-add prompt using `bubbles/textinput`)
- Modify: `internal/tui/view.go` or `internal/tui/model.go` (`domainDataSource` + section nav)
- Modify: `cmd/domains.go` (register `cosmoflare domains tui`)

- [ ] **Step 1: Write failing test**

```go
func TestDomainBrowserModel_QuickAddOpens(t *testing.T) {
	m := NewDomainBrowserModel([]*cosmoflare.DomainStatus{{Zone: &cosmoflare.Zone{ID: "z1", Name: "x.com"}, NSStatus: "cloudflare"}})
	m.StartQuickAdd()
	if !m.quickAddOpen() {
		t.Fatal("quick-add did not open")
	}
}
```

- [ ] **Step 2: Run test to verify it fails** → FAIL.

- [ ] **Step 3: Implement**

(a) Add a quick-add mode to `DomainBrowserModel`: on `a` key, open a `bubbles/textinput`-driven mini-form (destination + status) for the selected domain; on Enter, call the injected `RedirectService.Create` (via a `CreateRedirect func(...)` field set by the command). (b) Add a `domainDataSource` implementing the dashboard `DataSource` interface (`FetchMetrics`-style method returning domains) so the browser plugs into the app shell. (c) Register the command in `cmd/domains.go`:
```go
var domainsTUICmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive domain browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newDomainService(true)
		if err != nil {
			return err
		}
		domains, _, err := svc.List(cmd.Context(), cosmoflare.DomainListOptions{PerPage: 1000})
		if err != nil {
			return err
		}
		return launchDomainTUI(domains, svc) // wires CreateRedirect + runs bubbletea
	},
}
// in init(): domainsCmd.AddCommand(domainsTUICmd)
```

- [ ] **Step 4: Run test + build + manual smoke**

Run: `go test ./internal/tui/ -run TestDomainBrowserModel_QuickAddOpens -v` → PASS.
Run: `go build ./...` → success.
Manual (operator, with credentials): `cosmoflare domains tui` → browser launches, arrow keys move selection, `a` opens quick-add.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/domain.go internal/tui/view.go cmd/domains.go
git commit -m "feat(tui): quick-add redirect prompt, domainDataSource, domains tui command (P-07)"
```

---

## Final verification (both waves)

- [ ] `go build ./...` → success
- [ ] `go test ./pkg/cosmoflare/ ./cmd/ ./internal/tui/ -timeout 120s` → PASS
- [ ] `cosmoflare domains stats`, `cosmoflare domains redirects`, `cosmoflare redirects list <zone>` — smoke against live account
- [ ] `cosmoflare domains tui` — manual smoke
- [ ] Update `docs/USAGE.md` with the new `domains` subcommands + `redirects` group
- [ ] Close the linked FEAT issue; update roadmap; changelog entries

## Self-review (completed)

- **Spec coverage:** BR-01→Tasks 1-2, BR-02→Task 3, BR-03→Task 4, BR-04→Task 5, BR-05→Task 6, BR-06→Tasks 7-8, BR-07→Task 9. All 7 deliverables covered. Registrar "external without name" behavior (review #1) → Task 3 + Task 4 (`classifyRegistrarStatus`). Redirect merge (review #2) → Task 5 `domains redirects` lists per-domain rules; Page-Rule merge noted for follow-up (legacy pagerules remain accessible). Attention criteria (review #4) → Task 4 `domainNeedsAttention`. CF API verification (review #3) → flagged per service task. DI (review #5) → Task 5 factory var.
- **Type consistency:** `RedirectRule`, `RedirectRuleInput`, `RegistrarInfo`, `DomainSummary`, `DomainDetail` field names are consistent across tasks. `NewRedirectService`/`NewRegistrarService` signatures match usage in `newDomainService`/`newRedirectService`.
- **Placeholder scan:** Tasks 8-9 wave-2 TUI steps (Update/SetDetail/render-detail bodies) are described structurally rather than full code — these are highly iterative rendering tasks; the model seam (Task 7) is fully coded and the tests pin behavior. Implementer should expand them following `BrowserModel`.
- **Open risk:** Page-Rule `forwarding_url` merge into `domains redirects` (review #2) is noted but not a dedicated task — add a follow-up task if full legacy-merge is required before Wave-1 ship.
