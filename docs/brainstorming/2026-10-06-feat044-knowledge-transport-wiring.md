---
title: "FEAT-044: Wire knowledge.Transport into the shared CF client factory"
created: "2026-10-06T20:40:00+04:00"
status: COMPLETED
issue: FEAT-044
schema_version: 1
deliverables:
  - id: BR-01
    title: "knowledge.Transport installed at controlPlaneClient() — every newCloudflareAPI caller inherits the endpoint registry"
  - id: BR-02
    title: "NewRateLimitServiceFromCreds goes through newCloudflareAPI; manual Transport wiring removed"
  - id: BR-03
    title: "NewClient default httpClient wrapped; WithHTTPClient overrides wrapped via knowledge.Transport{Base: existing}"
  - id: BR-04
    title: "restClient httpClient wrapped — same CF v4 surface, same registry"
  - id: BR-05
    title: "Tests: full-path block through NewClient, chokepoint guard pins no direct NewWithAPIToken regressions"
---

# FEAT-044: Wire knowledge.Transport into the shared CF client factory

Status: COMPLETED (fast path — this doc is the spec; approved in-session 2026-10-06).
Issue: FEAT-044 (promoted from agent seed 2026-09-10, HOLD lifted when the
permissions pack — knowledge pack #2 — landed via FEAT-011).

## Why

`knowledge.Transport` (pkg/cosmoflare/knowledge/transport.go) enforces the
endpoint registry on the request side: routes inside a pack scope that match
no registered endpoint never leave the process — Cloudflare would answer
10405 with wording that misreads as an auth-scope problem. It is currently
installed only in `NewRateLimitServiceFromCreds`; every other control-plane
path gets no registry protection.

The original seed (2026-09-10) counted ~24 unwired `cloudflare.NewWithAPIToken`
sites. FEAT-039's unified transport policy (pkg/cosmoflare/transport.go)
consolidated all of them into the `newCloudflareAPI` / `controlPlaneClient`
chokepoint — structurally enforced by `TestTransportChokepoint_Guard`. The
remaining work is installing the Transport at that chokepoint and folding in
two stragglers.

## Design (approved)

| File | Change |
|------|--------|
| `pkg/cosmoflare/transport.go` | `controlPlaneClient()` returns `&http.Client{Timeout: DefaultControlPlaneTimeout, Transport: &knowledge.Transport{}}` — every `newCloudflareAPI` caller inherits the registry |
| `pkg/cosmoflare/ratelimit.go` | `NewRateLimitServiceFromCreds` switches from manual `cloudflare.NewWithAPIToken` + `knowledge.Transport` wiring to `newCloudflareAPI(apiToken)` — chokepoint compliance, redundant wiring removed |
| `pkg/cosmoflare/client.go` | `NewClient`: default httpClient (when caller passed none) gets `Transport: &knowledge.Transport{}` |
| `pkg/cosmoflare/rest_client.go` | restClient's constructed httpClient wrapped the same way — it hits the same CF v4 API surface (limits.go, bucket_notifications.go, bucket_policy.go), so the same registry applies |

**WithHTTPClient override semantics (decision: wrap always).** When a
library caller passes `WithHTTPClient`, wrap that client's Transport:
`knowledge.Transport{Base: existing.Transport}`. The caller keeps their
tuning (timeouts, custom dialers); the registry still applies. A caller who
truly wants to bypass knowledge can pass a client whose Transport is already
a bare RoundTripper — knowledge stays composable, not mandatory-by-trap.
Update the transport.go policy comment to say so.

**R2 data plane untouched.** The S3 SDK client (BUG-042 timeout policy)
never sees knowledge.Transport — scopes are CF REST routes; transfers stay
on their own policy.

## Safety verification (done in-session, pre-approval)

- Redirect service (`redirect.go`) lists zone rulesets and edits the
  `http_request_dynamic_redirect` phase entrypoint via the SDK — every call
  matches the ratelimit pack's generic parameterized registrations
  (`GET /zones/{zone_id}/rulesets`, `GET|PUT .../phases/{phase}/entrypoint`,
  rules CRUD). No false blocks.
- WAF managed-rulesets commands (`cmd/waf_lists.go`) are account-scoped
  (`/accounts/{id}/rulesets*`) — outside both packs' zone scopes →
  pass-through.
- `account verify` hits `GET /user/tokens/verify` — the permissions pack
  scopes `/user/tokens/permission_groups*` only → verify is out of scope,
  pass-through.
- Import direction: parent package `cosmoflare` imports child
  `cosmoflare/knowledge`; the child never imports the parent — no cycle.

## Tests

1. **Full-path block**: construct a client via `NewClient` (test creds), fire
   a request to an in-scope unregistered route (e.g.
   `DELETE /zones/{id}/rulesets/phases/{phase}/entrypoint`), assert the
   knowledge error surfaces (message contains "not a registered endpoint").
   Server-side: use an `httptest.Server` as Base to prove the request never
   left the process.
2. **Chokepoint guard extension**: the existing
   `TestTransportChokepoint_Guard` pattern pins that no new direct
   `cloudflare.NewWithAPIToken(` sites appear outside transport.go; extend
   to cover client.go's sanctioned site with a comment, and assert
   ratelimit.go no longer constructs one.
3. **Wrap-always pin**: `NewClient(WithHTTPClient(custom))` still enforces
   the registry (blocked-route test through the custom client), and the
   custom Transport remains in the Base chain (a marker RoundTripper sees
   pass-through traffic).
4. Existing suites stay green: `go test ./pkg/cosmoflare/... ./cmd/ ...`.

## Out of scope

- Any new pack content, registry data, or scope changes (knowledge packs
  are data; this change is wiring only).
- The desktop/mobile tiers (they wrap the same library — they inherit).
