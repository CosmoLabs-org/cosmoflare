---
ulid: 01M4HDD75JXETW8RPKKJNZS2WN
title: 'CF usage review gaps: KV analytics, D1 pagination, cache-coverage audit, cache-rule detection, planner guardrails'
type: feature
status: pending
priority: high
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T02:44:21.747532+04:00"
updated: "2026-10-10T02:44:21.747532+04:00"
suggested_conversion: ""
converted_to: null
related_issues: []
brainstorm_ref: null
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
---

# FB-pNZS2WN: CF usage review gaps: KV analytics, D1 pagination, cache-coverage audit, cache-rule detection, planner guardrails

# CF usage review gaps: KV analytics, D1 pagination, cache-coverage audit, cache-rule detection, planner guardrails

**What happened:** MyCarGuide's CF usage review (2026-10-09/10) validated cosmoflare's D1 firehole catch end-to-end — and hit six wrapper gaps doing the rest of the review by hand.

**Credit first:** FB-pP1TQPX (BUG-011) was exactly right. EXPLAIN-verified, migration-ready. After applying migration 0013 + PRAGMA optimize remotely, the complaints query measured **399,085 → 311 rows/call (1,281×)** live on the production DB. That catch saved ~$65/mo in D1 overage alone.

**Gaps we hit ourselves (all measured 2026-10-09/10):**

1. **No KV analytics.** `cosmoflare analytics` is R2-listing-based. We hit the GraphQL API directly (`kvOperationsAdaptiveGroups`) to get 24.1K reads / 2.5K writes per day vs 10M/1M monthly allowances (7%). A KV metrics + cost-verdict command would have answered "is KV overdone?" (it wasn't) in one call.
2. **D1 analytics pagination trap.** A single day of D1 query batches exceeds the 10,000-group GraphQL cap (verified). One unpaged query undercounted the same window **17×** (458M vs 8.39B rows). Any D1 usage feature must paginate (hourly windows worked for us).
3. **No caching-coverage view.** Baseline was 12.3K requests/day at **0.23% edge cache hit** — found only by hand-querying `httpRequests1dGroups`. A per-route-family cache-eligibility report (what's cached, what's dynamic, hit rate trend) surfaces the biggest CF lever directly.
4. **The HTML caching trap.** Cloudflare does not cache HTML by extension default: origin `s-maxage` headers alone produce `cf-cache-status: DYNAMIC` — verified on a live deploy after BUG-011 fix 3 turned out to be a silent no-op. Detect "origin emits s-maxage on HTML but no zone Cache Rule exists" — that one check would have caught a fix that looked shipped and did nothing.
5. **D1 guardrails:** prompt/run `PRAGMA optimize` after any index creation (D1 runs no ANALYZE — planner goes blind; the specs query burned 165K rows/call on indexed tables from a bad plan until statistics landed); document unary `+` hint patterns; keep the FTS5 export block.
6. **Alert classes beyond d1-rows-read:** KV writes/day, Workers invocations, cache-hit-rate regression (e.g. alert when zone hit rate falls below threshold for 24h).

**Why it matters:** every CosmoLabs project on Cloudflare pays the same currency — D1 rows read — and gets the same multiplier — the edge cache. MyCarGuide is the evidence case: 3.06B rows/day peak (3.6× the 25B/mo allowance) to indexed + colo/CDN-cached in one session, with before/after JSON snapshots (`docs/architecture/cf-usage/`) as the protocol.

**Proposed:** KV metrics/cost verdicts, paginated D1 usage, per-route cache-coverage audit incl. the cache-rule detection, D1 planner guardrails, and the wider alert classes — plus a one-command spend-vs-allowance page (D1 rows, KV ops, R2, invocations vs plan) feeding a monthly review habit.

**Priority:** high — direct cost lever with the evidence attached.

