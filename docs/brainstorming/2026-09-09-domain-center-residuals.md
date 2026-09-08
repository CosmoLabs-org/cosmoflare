---
title: 'Domain Center Residuals — Legacy Pagerules Merge + Redirect-Target Attention Check'
created: "2026-09-09T02:11:44+04:00"
status: planned
tags: [brainstorm, domains, redirects, pagerules, doctor, road-087]
roadmap: ROAD-087
parent_design: docs/brainstorming/2026-06-14-domain-management-center.md
deliverables:
  - id: BR-01
    title: "RedirectRule.Source field + legacyForwardingRules pure helper with tests"
  - id: BR-02
    title: "DomainService.WithPageRules factory + GetDetail legacy merge with tests"
  - id: BR-03
    title: "RedirectProber library — bounded HTTP probes with loop detection, httptest-covered"
  - id: BR-04
    title: "DomainStatus.RedirectIssue + domainNeedsAttention criterion + tests"
  - id: BR-05
    title: "domains stats --check-redirects flag + doctor redirect-target section + tests"
  - id: BR-06
    title: "TUI detail-pane issue badge (conditional render) + tests"
  - id: BR-07
    title: "docs/USAGE.md updates for both features"
---

# Domain Center Residuals — Legacy Pagerules Merge + Redirect-Target Attention Check

## Problem

FEAT-006 shipped the Domain Management Center in full — both waves (library+CLI, then TUI) of the 2026-06-14 design's four layers (verified 2026-09-09; ROAD-087 was hydrated from a stale memory line and re-scoped to this residual scope). Two requirements from the 2026-06-14 design never landed:

1. **Legacy pagerules merge** — the design's Layer 1 requires redirect visibility to MERGE modern Redirect Rules with legacy Page-Rule `forwarding_url` entries so "which domains redirect where" is complete regardless of which system created the rule. Today `domains redirects`, `GetDetail`, and the TUI right pane show modern rules only. CF is deprecating Page Rules, but existing accounts still carry legacy forwarding rules — they are invisible in every Domain Center surface.
2. **Redirect-target attention check** — "needs attention" criterion #4 (a redirect target that resolves ≥400, or a redirect loop) was Phase-2 best-effort in the design and was never implemented. `domainNeedsAttention` (pkg/cosmoflare/domains.go:290) covers NS/SSL/health only.

## Decisions (Q&A with user, 2026-09-09)

| # | Question | Decision |
|---|----------|----------|
| 1 | Where does the legacy merge live? | **Library GetDetail** — one merge site; `domains redirects`, TUI, and future consumers inherit complete visibility. Additive `Source` field on `RedirectRule`. |
| 2 | When do redirect-target probes run? | **Opt-in flag, bounded probes** — 8 concurrent, 10s per target, 10-hop loop detection. Never on the default listing path. |
| 3 | Where do results surface? | **CLI + doctor + TUI badge** — `domains stats --check-redirects`, a `doctor` section, and a conditional TUI detail badge (TUI runs no probes in v1; the badge activates when data is present). |

## Design — Part 1: Legacy Pagerules Merge

### RedirectRule.Source

`RedirectRule` (pkg/cosmoflare/redirect.go) gains one additive field:

```go
Source string `json:"source,omitempty"` // "" = modern Rulesets rule; "pagerules" = legacy forwarding_url entry
```

Modern rules keep the empty default (JSON unchanged for existing consumers). Legacy entries carry `"pagerules"`.

### Pure helper — legacyForwardingRules

```go
// legacyForwardingRules maps a zone's legacy Page Rules into RedirectRule
// entries. Only actions with ID "forwarding_url" produce entries; their
// Value decodes as {url, status_code}. Rules without a forwarding action
// are dropped. The output feeds the GetDetail merge.
func legacyForwardingRules(zoneID string, rules []*PageRule) []RedirectRule
```

Mapping: `PageRuleTarget.Constraint.Value` (URL pattern, e.g. `*example.com/old/*`) → `When`; forwarding `url` → `Destination`; `status_code` → `StatusCode` (default 301 when absent); `PageRule.Status == "active"` → `Enabled`; `Source = "pagerules"`; `PreserveQuery` stays false (legacy forwarding does not preserve query by default; the Value shape has no such flag).

`PageRuleAction.Value` is `interface{}` — decode defensively: on JSON round-trip it is `map[string]any`; accept `url` (string) and `status_code` (float64 → int). An action whose Value lacks `url` is dropped, not errored.

### DomainService.WithPageRules

`PageRuleService` is zone-scoped at construction (`NewPageRuleService(api, zoneID)`), so the enrichment seam is a factory, not an instance:

```go
// PageRuleLister is the consumer-side surface for legacy Page Rules.
type PageRuleLister interface {
	List(ctx context.Context) ([]*PageRule, error)
}

// WithPageRules wires legacy forwarding-rule enrichment into GetDetail.
// The factory is invoked per zone; a nil or failing factory silently skips
// legacy rows (modern rules still show — partial-failure doctrine).
func (s *DomainService) WithPageRules(factory func(zoneID string) PageRuleLister) *DomainService
```

`GetDetail` appends `legacyForwardingRules(...)` output after the modern rules (stable order: modern first, then legacy by PageRule priority). A factory error or nil lister → skip legacy, never fail the detail.

Cmd wiring: `newDomainService` (cmd/domains.go) gains the `WithPageRules` wiring in its `enrich` branch (per-zone `NewPageRuleServiceFromCreds`). Without it, `domains redirects`, `domains get`, and the TUI construct a `DomainService` with no legacy enrichment and inherit nothing from the merge.

## Design — Part 2: Redirect-Target Attention Check

### RedirectProber (pkg/cosmoflare/redirectprobe.go)

Stdlib-only HTTP prober, DoctorService pattern (no CF credentials, timeout-bounded client):

```go
// RedirectProbeResult is one destination probe outcome.
type RedirectProbeResult struct {
	Destination string `json:"destination"`
	Status      int    `json:"status"`              // final HTTP status; 0 on transport error/loop
	Loop        bool   `json:"loop,omitempty"`      // revisited a hop
	Err         string `json:"err,omitempty"`       // transport error, if any
	Skipped     bool   `json:"skipped,omitempty"`   // unprobeable (contains $1..$n captures)
}

// RedirectProber probes redirect destinations with bounded redirects and
// loop detection.
type RedirectProber struct { /* http.Client, per-target budget, hop cap, concurrency */ }

// Probe tests one destination: GET, follow redirects up to hopCap with a
// visited-set; a revisit sets Loop. Status >= 400 on the final hop is the
// issue signal. Destinations containing "$" capture references are skipped.
func (p *RedirectProber) Probe(ctx context.Context, destination string) RedirectProbeResult

// ProbeAll fans out over deduplicated destinations with a concurrency
// semaphore (default 8) and a per-target timeout (default 10s).
func (p *RedirectProber) ProbeAll(ctx context.Context, destinations []string) map[string]RedirectProbeResult
```

Defaults: 8 concurrent, 10s per target, 10-hop cap. Loop detection = visited-set of URLs (scheme+host+path); a cycle sets `Loop` and stops following. `StatusCode` capture references (`$1..$n` in `Destination`) make a URL unprobeable — `Skipped: true`, never a fabricated verdict.

### DomainStatus.RedirectIssue

```go
// DomainStatus gains:
RedirectIssue string `json:"redirect_issue,omitempty"` // "" | "loop" | "http-4xx" | "http-5xx" | "unreachable"
```

`domainNeedsAttention` gains `|| d.RedirectIssue != ""`. Issue classification from probe results: `Loop` → `"loop"`; final status 400-499 → `"http-4xx"`; 500-599 → `"http-5xx"`; transport error/timeout → `"unreachable"`. Skipped destinations contribute nothing.

The classification is a pure helper (`classifyRedirectIssue(results []RedirectProbeResult) string` — worst issue wins: loop > 5xx > 4xx > unreachable > none) so tests never touch the network.

### Surfacing

- **`cosmoflare domains stats --check-redirects`** — after listing, collect every domain's `GetDetail` destinations, dedupe, `ProbeAll`, classify per domain, set `RedirectIssue`; the existing attention flow then picks it up (`NeedsAttention` count + `Attention` list via the new criterion). The flag path constructs the service with `newDomainService(true)` — today `stats` calls it with `false`, which wires no redirect overlay, so `GetDetail` would return zero destinations to probe. `--json` carries the signal via a new `ByRedirectIssue map[string]int` breakdown on `DomainSummary` (mirroring `ByNSStatus`/`BySSLStatus`) — the summary struct carries no per-domain fields today, so the raw `RedirectIssue` field cannot appear in the stats JSON without this extension. Without the flag, behavior is byte-identical to today.
- **`cosmoflare doctor`** — gains a redirect-target section that probes CALLER-SUPPLIED destinations. The `doctor` library stays stdlib-only and credential-free (its documented contract); the `cmd/doctor.go` handler fetches the domain's redirect destinations via `DomainService.GetDetail` and hands the destination list to the prober, whose results land in the existing `DiagnosticReport` structure.
- **TUI** — the detail pane renders an issue badge line (`redirect: loop → https://…`) when `DomainStatus.RedirectIssue` is non-empty. The TUI itself runs no probes in v1; the field is empty in TUI-launched fetches (documented, not hidden).

## Error Handling

- Legacy list failure → silent skip (modern rows unaffected).
- Probe transport errors → `"unreachable"`, never a crash; context cancellation propagates.
- No credentials at TUI launch → existing placeholder pattern, unchanged.
- Every new failure path produces actionable, agent-readable errors per house style.

## Testing Strategy (TDD)

- **Pure helpers first**: `legacyForwardingRules` (table tests: forwarding rule mapping, non-forwarding dropped, missing url dropped, status_code default), `classifyRedirectIssue` (precedence table), capture-reference detection.
- **RedirectProber**: httptest servers — a self-redirect loop handler, a 404, a 200, a slow handler (timeout), a capture-URL (skipped). Concurrency respected (no live network).
- **GetDetail merge**: fake `PageRuleLister` factory (legacy + modern mixed; failing factory).
- **stats/doctor cmd**: factory-var injection per the existing `newDomainService` seam; flag on/off behavior identical when off.
- **TUI**: view test asserting the badge renders when set and absent when empty.

## Non-Goals (deferred)

- Probing the redirect SOURCE pattern (synthesizing concrete URLs from CF expressions / pagerule globs) — fuzzy, out of scope.
- Probe result persistence or cross-run caching — in-process only for v1.
- WHOIS registrar naming, domain transfers, buying domains (unchanged from the parent design).

**Other parent-design gaps consciously out of scope here** (surfaced by review 2026-09-09, so the residual inventory is honest): attention criterion #3 (registration expiry <30 days) also never landed — it needs `DomainStatus` to carry registrar data on the LIST path, a different enrichment surface than this design's; and the TUI footer "needs attention" count plus the DNS record-type breakdown in the detail pane were descoped when the 2026-06-14 plan coded the reduced criterion set. Both are candidates for a future TUI-depth pass, not this one.

## Deliverables

- BR-01: `RedirectRule.Source` + `legacyForwardingRules` + tests
- BR-02: `DomainService.WithPageRules` + GetDetail merge + tests
- BR-03: `RedirectProber` + `ProbeAll` + httptest coverage
- BR-04: `DomainStatus.RedirectIssue` + `classifyRedirectIssue` + attention criterion + tests
- BR-05: `domains stats --check-redirects` + doctor section + tests
- BR-06: TUI conditional badge + view test
- BR-07: docs/USAGE.md — redirects completeness note + new flag/section
