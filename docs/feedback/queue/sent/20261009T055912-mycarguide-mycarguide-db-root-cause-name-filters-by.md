---
ulid: 01M4F65HY55FJQRA0E30P1TQPX
from: cosmoflare
type: bug
severity: high
title: 'mycarguide-db root cause: name filters bypass indexes (SCAN complaints 399k rows/call) + no planner stats — verified fixes, /compare uncached'
created: "2026-10-09T05:59:12.581278+04:00"
---

# mycarguide-db root cause: name filters bypass indexes (SCAN complaints 399k rows/call) + no planner stats — verified fixes, /compare uncached

Follow-up to the 2026-10-08 report (ULID 01M4EGHYQTWCFE12BYQFS5173B, "mycarguide-db scans ~2.95B D1 rows/day"). This one has the root cause, verified fixes, and the page that drives it. All evidence is read-only (D1 per-query analytics, `EXPLAIN QUERY PLAN`, `sqlite_master`), gathered 2026-10-09 from cosmoflare.

## What is happening

Two queries cause 98% of the database's rows read (24h, `d1QueriesAdaptiveGroups`):

| Query | Runs/day | Rows read/day | Rows read per run | Rows returned per run |
|---|---|---|---|---|
| complaints by make/model/year (`SELECT c.component, c.description FROM complaints c JOIN models m … JOIN makes mk … WHERE mk.name = ? AND m.name = ? AND c.year = ?`) | 4,968 | 1.98B | 398,730 | ~17 |
| specs for a model (`SELECT v.id AS vehicle_id … FROM vehicles v JOIN trims t … JOIN generations g … JOIN specs s … WHERE g.model_id = ? AND g.superseded_by IS NULL AND s.key IN (?,?,?,?) ORDER BY …`) | 4,834 | 0.80B | 165,367 | ~19 |
| recalls by make/model/year (same shape as complaints, on `recalls`) | 4,679 | 45M | 9,705 | <1 |

D1 bills rows read (scanned), not rows returned (https://developers.cloudflare.com/d1/platform/pricing/). At ~2.83B/day the database uses ~85B rows/month against the account-wide Workers Paid allowance of 25B — about $60/month overage from this one database, and it consumes the allowance every other CosmoLabs project shares.

## Root cause (verified on the live database)

The indexes in `packages/db` DO exist remotely (`complaints_model_year`, `recalls_model_year`, `specs_vehicle_key`, `generations_model`, …) — this is not schema drift. The planner just cannot reach them:

1. **complaints / recalls:** the filters are on `makes.name` and `models.name`, which have no index. `EXPLAIN QUERY PLAN` shows `SCAN c` — a full scan of all 390,117 complaints rows on every call (`SCAN r` for recalls, 9,532 rows).
2. **specs:** the database has no planner statistics (no `sqlite_stat1` table). The planner guesses and starts from `specs_key` — every specs row with those 4 keys out of 1,412,930 — then filters by model at the end.

## Fixes (each plan verified with EXPLAIN QUERY PLAN on the live DB)

**1. Rewrite the complaints and recalls queries to resolve `model_id` first** — verified plan: `SEARCH c USING INDEX complaints_model_year (model_id=? AND year=?)`:

```sql
SELECT c.component, c.description
FROM complaints c
WHERE c.model_id = (
  SELECT m.id FROM models m JOIN makes mk ON m.make_id = mk.id
  WHERE mk.name = ? AND m.name = ?
)
AND c.year = ?;
```

Same shape for recalls (keep `ORDER BY r.report_date DESC`). The subquery still scans `models` (2,740 rows), so also add:

```sql
CREATE INDEX IF NOT EXISTS makes_name ON makes(name);
CREATE INDEX IF NOT EXISTS models_make_name ON models(make_id, name);
```

Then re-check with `EXPLAIN QUERY PLAN` — expect `SEARCH mk USING INDEX makes_name` and `SEARCH m USING INDEX models_make_name`. (Not verified here: creating indexes is a write, so it was not done from cosmoflare.)

**2. Give the planner statistics:** run `PRAGMA optimize;` against the remote database once after the migration. Cloudflare D1 docs: "After creating an index, run the `PRAGMA optimize` command to improve your database performance." (https://developers.cloudflare.com/d1/best-practices/use-indexes/, updated Aug 10, 2026). Note: D1 does not support `PRAGMA optimize(-1)`. Then re-check the specs query plan; it should start from `generations_model`. If it still starts from `specs_key`, use this verified fallback — a unary `+` stops the planner from choosing `specs_key`:

```sql
… AND +s.key IN (?,?,?,?) …
```

Verified plan with the `+`: `SEARCH g USING INDEX generations_model` → `trims_generation` → `vehicles_trim` → `SEARCH s USING COVERING INDEX specs_vehicle_key`.

**3. Cache the page that runs them.** On mycar.guide, `GET /compare` is served uncached (`cacheStatus=dynamic`) 5,018 times/day — the same volume as the query runs above. The catalog only changes on seed loads, so cache `/compare` responses (KV with a query-aware key, or `Cache-Control: public, s-maxage=…` on the route). Reference implementation of the CosmoLabs CF efficiency standard: Churches-app commit 155d1d3e (IMP-017) — KV cache middleware with query-aware keys, `Cache-Control` emitted on hits and misses, auth routes never cached.

**4. Block scanner probes at the edge.** mycar.guide's Worker also serves `/.env.sample`, `/.env.aws`, `/settings.json`, `/.claude/settings.json`, `/phpinfo`, `/.aws/config` probes — each one a billed Worker invocation. A WAF custom rule blocking these paths stops them before the Worker runs (Free plan: 5 custom rules per zone, no regex — use `contains`; https://developers.cloudflare.com/waf/custom-rules/, updated Aug 25, 2026).

## Expected result and how to verify

Rows read per complaints call should drop from ~399k to well under 1k, and the whole database from ~2.8B/day to tens of millions/day (estimate — confirm after deploy). Verify with:

```bash
wrangler d1 insights mycarguide-db --sort-type sum --sort-by reads --timePeriod 1d
```

cosmoflare now has a `d1-rows-read` alert (`cosmoflare alerts create d1-scans --service d1 --condition d1-rows-read --threshold 1e9 --action log --target -`) that pages when any database passes 1B rows read in 24h.

**Priority:** high — ongoing cost and shared-allowance burn; fixes 1-3 are small and verified.
