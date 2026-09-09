---
title: 'CF API Knowledge Layer — endpoint registry, plan caps, field preflight, error decoding'
created: "2026-09-09T18:23:41+04:00"
status: planned
tags: [brainstorm, knowledge, ratelimit, errors, feat-012]
issue: FEAT-012
deliverables:
  - id: BR-01
    title: "Knowledge types + loader + embedded JSON pack format + ratelimit seed pack"
  - id: BR-02
    title: "KnowledgeTransport — scoped route-check + response error decode"
  - id: BR-03
    title: "ValidatePayload — plan caps + field invariants (pure functions)"
  - id: BR-04
    title: "RateLimitService reference consumer — list + create via the correct rulesets flow"
  - id: BR-05
    title: "CLI — decode, knowledge list, ratelimit commands"
  - id: BR-06
    title: "docs/USAGE.md — knowledge layer, decode, ratelimit commands"
---

# CF API Knowledge Layer — endpoint registry, plan caps, field preflight, error decoding

## Problem

FEAT-012 (origin: MyCarGuide feedback FB-6, 2026-09-09): applying ONE Cloudflare
rate-limit rule took six failed API calls across four failure classes, none of
which the CF API explains at the point of failure:

1. **10405** on a nonexistent route (`POST /zones/{id}/rulesets/phases/http_ratelimit/entrypoint/rules`)
   reads as an auth-scope problem — the operator edited token permissions for
   20 minutes while the real problem was the route.
2. A fresh Free zone has **no phase entrypoint**; GET returns a result
   indistinguishable from "0 rules" until a follow-up call fails **1000 not_found**.
3. Payload validation is server-side and sequential — **20155** reveals only after
   auth+route succeed that rate-limit characteristics MUST include `cf.colo.id`
   ("ratelimiting counting is processed at colocation level only").
4. **Free-plan caps** (1 rule/zone, 10s window, 10s mitigation, IP-only counting)
   live in a marketing availability table, enforced client-side nowhere.

Every CosmoLabs project touching Cloudflare hits this maze. A cosmoflare
knowledge layer makes it one call.

## Decisions (Q&A with user, 2026-09-09)

| # | Question | Decision |
|---|----------|----------|
| 1 | v1 breadth — full endpoint registry (~2500 endpoints) vs seed data? | **Framework + ratelimit pack** — generic primitives, ONE seeded data pack from the MyCarGuide evidence. Other products arrive as data when researched (Qwen pass pending). |
| 2 | Enforcement point — transport, validators, or both? | **Hybrid** — error decoding + route-check in an `http.RoundTripper` (automatic for every service); plan-caps + field preflight as pure validators in the service layer (they are payload-shaped). |
| 3 | Data storage form? | **Embedded JSON packs** via `go:embed` under `pkg/cosmoflare/knowledge/packs/` — one SSOT file per product, single binary, research updates land as data. |

## Design

### Scoped route-check semantic (the v1 footgun avoided)

The route-check applies ONLY within a pack's declared `scopes` (path templates
like `/zones/{zone_id}/rulesets*`):

- Route **outside every scope** → passes through untouched. Knowledge is absent,
  not wrong. One ratelimit pack must not "reject" the other ~2,490 endpoints.
- Route **inside a scope**, method+route matches a registered endpoint → proceed.
- Route **inside a scope**, no match → **block pre-send** with a client-side
  `KnowledgeError`: "endpoint does not exist — CF would return 10405 (its
  auth-scheme wording is misleading)".

Doctrine: **knowledge is advisory when absent, authoritative when present.**

### Components (`pkg/cosmoflare/knowledge/` + consumers)

| Unit | File | Purpose |
|------|------|---------|
| Types + loader | `knowledge/knowledge.go` | `Pack{Product, Scopes, Endpoints, Errors, PlanCaps, Invariants}`; `Endpoint{Method, PathTemplate}` with `{param}` templates; `ErrorDecode{Code, Context, Cause, Fix}` (context disambiguates: 1000-on-entrypoint ≠ 1000-elsewhere); `Load()` via `sync.Once`; `MatchRoute(method, path)`. Malformed pack JSON fails loud at Load(). |
| Seed pack | `knowledge/packs/ratelimit.json` | MyCarGuide evidence: rulesets endpoints (the nonexistent `entrypoint/rules` registered as absent), decodes for 10405/1000/20155/10000, Free caps (rules_per_zone=1, window_seconds≤10, timeout_seconds≤10, characteristics=IP-only), `cf.colo.id` invariant. |
| Transport | `knowledge/transport.go` | `KnowledgeTransport{base http.RoundTripper}` — request-side scope/route check; response-side CF error-body decode → `KnowledgeError{Code, Cause, Fix}` wrapping the original body. |
| Validators | `knowledge/validate.go` | `ValidatePayload(product, plan, payload) []Violation` — pure. Plan sourced from `Zone.Plan` (existing ZoneService data path). |
| Reference consumer | `pkg/cosmoflare/ratelimit.go` | Minimal `RateLimitService` (list + create) doing the correct flow: ensure entrypoint (PUT) when missing, preflight `cf.colo.id`, then create. Proves the framework end-to-end. |
| CLI | `cmd/decode.go`, `cmd/knowledge.go`, `cmd/ratelimit.go` | `cosmoflare decode 10405 --context rulesets` (offline table, `--json`); `cosmoflare knowledge list` (loaded packs, registry transparency); `cosmoflare ratelimit list/create`. |

### Data flow

`ratelimit create` → `ValidatePayload` (caps by plan, `cf.colo.id`) →
violation ⇒ stop with an agent-readable error, zero API calls ⇒ else
cloudflare-go call through `KnowledgeTransport` → route-check → CF →
error response ⇒ decoded `KnowledgeError` with cause + fix.

## Error Handling

| Failure | Behavior |
|---------|----------|
| Pack JSON malformed | Fail loud at `Load()` — corrupted knowledge never silently passes |
| Route outside all scopes | Pass through untouched |
| Route in scope, no endpoint match | Block pre-send, client-side `KnowledgeError` citing the registry |
| CF error, code has decode entry | Wrap: original body + `Cause` + `Fix` |
| CF error, no decode entry | Pass original error through — never fabricate a verdict |
| Plan unknown (zone fetch failed) | Skip plan-cap checks, still run invariants, say so |
| Validator violation | Stop pre-send; name the field, the cap, and the plan |

## Wire-in

`cloudflare.NewWithAPIToken` accepts an `http.Client` option — services built in
`New<Svc>FromCreds` get a client whose `Transport` is `KnowledgeTransport`.
The plan verifies the exact option against cloudflare-go v0.116.0; if injection
is unavailable at that seam, fallback: exported `DecodeCFError(err)` called in
service error paths (same data, one explicit call per site).

## Testing Strategy (TDD)

- **Pure**: `MatchRoute` (templates vs concrete paths), `DecodeError`
  (code+context precedence), `ValidatePayload` (Free caps, `cf.colo.id`,
  plan-unknown skip) — table tests, no network.
- **Transport**: httptest — in-scope unknown route blocked pre-send; CF-shaped
  4xx body decoded; unregistered code passes through.
- **RateLimitService**: httptest rulesets flow — entrypoint auto-create on
  1000-not_found; preflight catches missing `cf.colo.id` before any request
  leaves.
- **CLI**: flag + JSON-shape tests per house pattern.

## Non-Goals (deferred)

- **FEAT-011** (permission catalog): separate issue; v1 reserves the pack schema
  field, ships no permission data.
- **FEAT-013** (traffic-class matrix + trip probe): later extension of
  `ratelimit create`; schema field reserved.
- **~2,490 uncovered endpoints**: out of scope. They arrive as packs via the
  pending Qwen coverage research (`docs/research/delegation/2026-09-08-cf-gap-qwen-pack.md`,
  results not yet arrived) — a separate ingestion plan.
- **No new CF product services** beyond the minimal ratelimit reference consumer.
- **Docs site / marketing**: none.

## Deliverables

- BR-01: knowledge types + loader + JSON pack format + ratelimit seed pack
- BR-02: KnowledgeTransport (scoped route-check + error decode)
- BR-03: ValidatePayload (plan caps + invariants, pure)
- BR-04: RateLimitService (list + create, correct rulesets flow)
- BR-05: CLI (decode / knowledge / ratelimit)
- BR-06: docs/USAGE.md updates
