---
title: 'www→non-www automation — detect, alert, fix (three layers)'
status: PLANNED
branch: master
created: "2026-10-14T21:10:00+04:00"
related_roadmap:
    - ROAD-p9X66J9
tags: [www-redirect, dns, pagerules, efficiency, alerts, automation]
deliverables:
    - id: P-01
      title: "Library: WWWRedirectService — per-zone audit (rule + DNS record) and idempotent fix"
    - id: P-02
      title: "Detect: www-redirect-missing finding in cosmoflare efficiency (zone loop)"
    - id: P-03
      title: "Fix: cosmoflare domains www-redirect --all [--fix] fleet command + USAGE.md"
    - id: P-04
      title: "Alert: www-redirect-missing ops condition riding the rules-quota rotation (zero extra fetches)"
    - id: P-05
      title: "init/apply integration so new cosmoflare projects get the redirect by construction"
---

# www→non-www automation — three layers

**Goal:** Make the fleet www→non-www standard self-enforcing: every audit
detects gaps, the ops watch pages on drift, one command (and project setup)
fixes them. No human re-checking.

**Spec context:** fleet standard verified 2026-10-14 (44/45 zones 301;
cosmofy.org pending nameserver activation). House policy:
`ClaudeCodeSetup/docs/standards/www-redirect-policy.md`.

## Verified facts (live, 2026-10-14 — do not re-derive)

1. **Cloudflare has no account-level redirects** — the API rejects the
   `http_request_dynamic_redirect` phase at account scope ("not allowed at
   account level"). No "apply to new zones" setting exists on any plan.
2. The mechanism is necessarily **per-zone**: a proxied `www` CNAME → apex
   (or any resolving www record; the edge rule fires before origin) plus
   one redirect rule. Page Rules work on the current token; Dynamic
   Redirect Rules need the Zone Rulesets permission (the token NOW HAS it,
   flipped 2026-10-14).
3. Fleet state after today: 44/45 active zones redirect (verified by live
   curl each); cosmofy.org is zone-status "pending" — no rule exists there
   yet; it must be fixed once active (or pre-created now).
4. Fix recipe (proven twice today, cosmolabs.org + 5 zones): create proxied
   CNAME `www` → apex when NXDOMAIN, create Page Rule
   `www.<domain>/*` → 301 → `https://<domain>/$1`. Leave working records of
   other shapes (origin A / pages.dev / parking CNAME) untouched.
5. `kvOperationsAdaptiveGroups` has an `actionType` dimension (read/write/
   list) — the earlier "no split" claim was a probe error, corrected.

## Recon (in-tree, next session starts from here)

| Piece | Where | Shape |
|---|---|---|
| Page rules service | `pkg/cosmoflare/pagerules.go` | `PageRuleService` per-zone (`NewPageRuleService(api, zoneID)`), `List/Create` exist; `PageRule{Targets, Actions, Status, Priority}` |
| DNS records | `pkg/cosmoflare/dns.go` | `DNSService.List(ctx, opts...)` — filter to `www.<zone>` |
| Domains command group | `cmd/domains.go` | exists; add `www-redirect` subcommand there |
| Efficiency zone loop | `pkg/cosmoflare/efficiency.go` | zone lister + coverage loop live; findings pattern established (caching.go is the newest example: pure assess + service + optional wiring) |
| Ops rotation | `apps/ops/src/rules_quota.ts` | `collectPageRules` ALREADY extracts each rule's target + forwarding action per zone every rotation cycle (~30 min fleet) — layer 4 (alert) derives wwwRedirect-present per zone from THIS, zero extra fetches |
| Alert registry | `apps/ops/src/scheduled.ts` | `CONDITIONS` registry + datasets; zone-scoped precedents: `zone-cache-miss-pct`, `zone-uncached-requests` |
| Fix script (reference) | `/tmp/fix-www-redirects.py` (session-only; logic copied below into P-01) | zone lookup by name → ensure record → ensure rule, idempotent |

## Layer designs

### P-01 Library — `pkg/cosmoflare/wwwredirect.go` (+ `_test.go`)

```go
type WWWRedirectStatus struct {
    Zone        string
    RuleExists  bool // any page rule (or redirect rule) forwarding www.<zone>/* → apex
    WWWResolves bool // any DNS record for www.<zone> (any shape — edge fires before origin)
}
type WWWRedirectService struct{ ... } // PageRules + DNS per zone, from creds
func (s *WWWRedirectService) Audit(ctx, zone) (*WWWRedirectStatus, error)
func (s *WWWRedirectService) Fix(ctx, zone) (*FixResult, error) // creates CNAME only when NXDOMAIN; creates rule only when absent; reports CreatedCNAME/CreatedRule booleans
func AssessWWWRedirect(st *WWWRedirectStatus) *EfficiencyFinding // nil when compliant; Check "www-redirect-missing", Severity "warning"
```
Tests: REST mocks (cloudflare-go `BaseURL`), variants: compliant / rule-only
(dead www, the cosmolabs bug) / record-only / neither / fix-idempotence
(second Fix creates nothing).

### P-02 Detect — efficiency finding

Wire `WWWRedirectService` into `EfficiencyService` (optional field +
constructor param following the caching/WithCaching pattern); run inside
the zone loop; 2 REST calls per zone (pagerules + dns_records?name=www).
cmd/efficiency.go wires it when creds exist. Tests extend the zone-loop
fixtures. Evidence string: "no redirect rule for www.<zone>" / "www does
not resolve" / both.

### P-03 Fix — `cosmoflare domains www-redirect [--all] [--fix] [--json]`

Audit mode: fleet table ZONE | RULE | WWW RECORD | VERIFIED (live curl
optional `--probe`). `--fix`: runs Fix per non-compliant zone, prints what
it created. Seam pattern (usage.go/efficiency.go precedents) for cmd tests.
USAGE.md section beside the efficiency one.

### P-04 Alert — ops condition, zero extra fetches

Extend `collectPageRules` (rules_quota.ts) to set `wwwRedirect: boolean`
per zone in the summary store (rule target starts with "www." AND action
forwarding_url). New condition `www-redirect-missing` evaluates from the
summary store — the rotation already fetches this data every cycle; the
alert adds NO calls. Alert names the zones. Registry + evaluator follow
the zone-scoped precedents. Degrades quietly when the store is empty.

### P-05 Setup integration

`init`/`apply` add the CNAME + rule when a project config declares
`www_redirect: true` (default true). Reads from the same Fix code path.

## Sequencing (next session)

1. P-01 → P-02 → P-03 (Go, sequential — shared files)
2. P-04 (TS, parallel-safe after P-01's semantics are fixed)
3. P-05 last (touches apply; needs P-01 stable)
4. Live verification: fleet audit → intentionally delete one test rule →
   efficiency + alert both fire → --fix restores it.

## Out of scope

- Dynamic Redirect Rule migration (separate slice; needs expression
  `starts_with(http.host, "www.")` + regex target — covers www subdomains,
  frees Page Rule slots; do AFTER this automation exists to enforce it)
- cosmofy.org activation watch (one-off fix when the zone activates)
