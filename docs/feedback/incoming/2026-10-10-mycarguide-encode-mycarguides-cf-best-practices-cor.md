---
ulid: 01M4HHTVMZW101AQCCEJT3ZHNM
id: FB-23
title: Encode MyCarGuide's CF best-practices corpus as cosmoflare audit classes (reading list attached)
type: feature
status: pending
priority: high
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T04:01:33.900871+04:00"
updated: "2026-10-10T04:01:33.900871+04:00"
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

# FB-pT3ZHNM: Encode MyCarGuide's CF best-practices corpus as cosmoflare audit classes (reading list attached)

# Encode MyCarGuide's CF best-practices corpus as cosmoflare audit classes (reading list attached)

**What:** MyCarGuide distilled its CF efficiency discipline (learned fixing BUG-011, a 3.06B rows/day D1 firehose) into reviewable docs. Each rule in them is mechanically checkable — encode them as cosmoflare audit classes so every CosmoLabs project gets checked, not just the one that burned first. Companion evidence FB (already delivered): `01M4HDD75JXETW8RPKKJNZS2WN` — measured gaps; this one points at the practices corpus.

**Where to read (MyCarGuide repo):**
- `docs/architecture/ADR-002-d1-cost-model.md` — the six standing rules: (1) index the exact filter path; (2) `PRAGMA optimize` after every index creation; (3) CDN-cache every query-less shell; (4) measure before/after every release; (5) write discipline — never write per-request, bulk data in one idempotent batch, state expected rows_written/day; (6) storage discipline — every byte has an owner and a bound (R2 for long text, D1 keeps scalars; TTL on every KV key; deterministic R2 keys; >2× growth between snapshots = review trigger).
- `docs/architecture/caching-strategy.md` — the per-route-family matrix: content-change-frequency → mechanism (prerender / CDN s-maxage+SWR / immutable JSON / version-keyed KV ladder / dynamic), the two delivery layers (colo Cache API vs zone Cache Rule), the **HTML-needs-Cache-Rule trap** (origin s-maxage alone = cf-cache-status DYNAMIC, verified live), and the invalidation model (SWR convergence, purge operating model).
- `docs/architecture/d1-indexing-strategy.md` — query-pattern → index map discipline, EXPLAIN QUERY PLAN verification before/after, unary `+` planner-hint pattern, index inventory kept honest.
- `docs/architecture/cf-usage-spend.md` — spend-vs-allowance one-pager format (D1 rows read/written, KV ops AND storage, R2, requests vs plan), KV storage measurement + reduction levers, monthly review habit.
- `infra/scripts/cf-usage-snapshot.mjs` — the measurement instrument: hourly-paginated D1 analytics (single day >10k GraphQL groups), KV ops by action, zone cache-hit rate — the direct blueprint for a `cosmoflare usage snapshot` command.

**Why:** cosmoflare is the fleet's audit layer. Every rule above failed silently in production until measured by hand; each maps to a check cosmoflare can run across all projects: query-vs-index mismatch (EXPLAIN), s-maxage-without-cache-rule, KV keys without TTL, per-request write paths, storage growth trends, allowance burn rates.

**Proposed:** a `cosmoflare audit` reporting per-project compliance per rule + `cosmoflare usage snapshot/compare` built from the script pattern; alert classes already listed in the companion FB.

**Priority:** high — the audit program turns one project's incident into every project's guardrail.

