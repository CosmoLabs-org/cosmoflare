---
title: AI Gateway monitoring in Ops
created: 2026-10-10T19:13:00+04:00
status: captured
last_reviewed: 2026-10-10T19:15:37+04:00
last_review_findings: 5
deliverables:
    - BR-01: "Library fix — AIGatewayLog struct gains tokens_in/tokens_out, metadata, model_type, custom_cost, request/response content types (schema per live API docs, read 2026-10-10)"
    - BR-02: "Worker collector collectAIGateways + /api/ai endpoint — GraphQL aiGatewayRequestsAdaptiveGroups aggregated by gateway × model × provider, with a one-time live probe for undocumented token/cost fields and graceful degradation to counts-plus-note"
    - BR-03: "Pager AI section — per-project (gateway) cards with per-model ranked rows (requests, tokens in/out, cache share, cost labeled gateway estimate)"
    - BR-04: "Attribution bootstrap — one gateway per project (naming convention + gateway-to-project map), gateways created via the existing library CreateGateway, pipeline verified on real first requests"
    - BR-05: "Pricing awareness — verified per-model price rows (sourced + dated, pricing.ts discipline) feeding custom_cost overrides once operator-supplied Grok research lands"
---

# AI Gateway monitoring in Ops

## Why

The operator plans to route model traffic (DeepSeek, Qwen, and other providers)
through Cloudflare AI Gateway and needs to see, in Cosmoflare Ops, which
projects are requesting which models and how much they consume. Today the
library covers gateway CRUD and raw log fetch (`pkg/cosmoflare/ai.go`) and
Ops has no AI surface at all — no analytics, no attribution, no view.

Operator intent (2026-10-10, session 2036): multi-provider monitoring, not
Workers-AI-only — the gateway's purpose is access to many models; per-project
consumption; aware of latest pricing.

## Decisions (Q&A with operator)

| Question | Decision | Why |
|----------|----------|-----|
| v1 traffic scope | Everything the gateway proxies | Model/provider are data dimensions — any proxied model appears as its own row with zero per-provider code |
| Surface | Own "AI" section in Ops first | Attribution mapping lives there; existing views unchanged; Projects rollup is a later seam |
| Attribution | One gateway per project | `gateway` is a first-class GraphQL dimension; operator-controlled naming; works for every provider today. `cf-aig-metadata` header (5 entries, log-filterable) noted as a later request-level option |
| Aggregation | GraphQL-first with live field probe | Matches every Ops view's cached-endpoint shape; undocumented ≠ absent — probe once, degrade honestly |

## Research findings (live docs, read 2026-10-10)

- **Analytics**: GraphQL dataset `aiGatewayRequestsAdaptiveGroups`
  (`https://api.cloudflare.com/client/v4/graphql`). Documented dimensions:
  `model`, `provider`, `gateway`, `datetimeMinute`; documented metric
  `count`. Tokens/cost are NOT documented as GraphQL fields.
  https://developers.cloudflare.com/ai-gateway/observability/analytics/
- **REST logs**: `GET /accounts/{account_id}/ai-gateway/gateways/{gateway_id}/logs`.
  Fields: `id, cached, created_at, duration, model, path, provider, success,
  tokens_in, tokens_out, cost, custom_cost, metadata, model_type,
  request_content_type, request_type, response_content_type, status_code,
  step`. Pagination `per_page` max 50; filters incl. `metadata.key/value`.
  https://developers.cloudflare.com/api/resources/ai_gateway/subresources/logs/methods/list/
- **Cost is an estimate** — docs verbatim: "an estimation based on the number
  of tokens sent and received in requests"; provider dashboards are
  authoritative; `custom_cost` overrides "will override the default or public
  model costs".
  https://developers.cloudflare.com/ai-gateway/observability/costs/
- **Metadata tagging**: `cf-aig-metadata` header, up to five entries per
  request, string/number/boolean values; appears in logs, filterable.
  https://developers.cloudflare.com/ai-gateway/observability/custom-metadata/
- **Retention/limits**: legacy logs persist 10M/gateway (Paid); gateways
  created on/after 2026-09-24 follow Workers Logs pricing/retention; GraphQL
  rate limits undocumented — treat empirically.
  https://developers.cloudflare.com/ai-gateway/observability/logging/legacy-logs/

## Design (approved 2026-10-10)

- **Collector** (`apps/ops/src/ai_gateway.ts`): `collectAIGateways(accountId,
  token, now)` — list gateways (REST), then one GraphQL query grouped by
  gateway × model × provider over 24h and MTD. A field probe requests the
  undocumented token/cost fields; its result is memoized per isolate AND
  persisted through the KV L2 with a 24 h TTL — so the probe runs ~once a
  day, not per request or per cold start, bounding the undocumented GraphQL
  rate limits. Absent fields degrade the payload to counts + an explicit
  gap note (never invented numbers). Cached at summary cadence (5 min
  fresh / 1 h stale) through the existing `cached()` wrapper.
- **API**: `GET /api/ai` → `{ gateways: [{ id, name, project, models: [{
  model, provider, requests, tokensIn?, tokensOut?, cacheShare, costUsd? }] }],
  costBasis: "gateway-estimate", errors, cache }`.
- **View** (pager `ai.ts`): AI section in the nav (icon: spark); per-project
  card per gateway; per-model ranked rows (requests, tokens, cache share,
  cost). Every dollar figure renders with the muted "gateway estimate" label —
  the real-numbers rule applies: API-sourced numbers only, and the API itself
  calls cost an estimate, so the label is permanent until `custom_cost` rows
  exist.
- **Attribution map**: gateway name → project, operator-controlled naming
  convention (e.g. gateway `mycarguide` → project MyCarGuide); unmapped
  gateways render under their raw name with an "unmapped" marker, never
  dropped.
- **Library fix** (rides along): `AIGatewayLog` struct brought to the real
  schema (tokens_in/tokens_out split, metadata, model_type, custom_cost,
  request/response content types — full BR-01 scope).

## Deliberately excluded (YAGNI, later phases)

- Recent-requests drawer from the logs API (50/page makes it a sample).
- Alert rules on AI spend/tokens (extends the rules engine after the data
  lands).
- `cf-aig-metadata` request tagging (single-projects-operator first).
- Projects-view rollup of AI cost (second-order seam).

## Open items

- Operator-supplied Grok research: current per-token pricing (input/output,
  sources + dates) for the planned model list → feeds BR-05 price rows.
- Live probe result: do token/cost fields exist on the GraphQL dataset?
  (Determines whether BR-02 ships tokens/cost or counts-first.)
- Gateway naming convention confirmation at bootstrap time.
- Phase 4 must design the `custom_cost` WRITE path before dispatch — the
  library only reads it. Likely the per-request `cf-aig-custom-cost`
  mechanism from the costs doc; verify whether any account/gateway-level
  pricing override exists before choosing.

## Roadmap mapping

Parent ROAD item ROAD-p1D6G3P "AI Gateway monitoring (Ops)" with phases:
1. ROAD-pK05936 — Library log-schema fix + collector + /api/ai (BR-01, BR-02)
2. ROAD-pH8BK60 — AI section view (BR-03)
3. ROAD-p713JYR — Attribution bootstrap on live traffic (BR-04)
4. ROAD-pS5NYJN — Verified pricing + custom_cost overrides (BR-05, awaits Grok research)
5. ROAD-pGXTT2M — (later) AI alert rules + recent-requests drawer
