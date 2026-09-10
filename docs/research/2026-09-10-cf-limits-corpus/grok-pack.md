# RESEARCH PACK — Cloudflare Plan Limits Corpus (GROK)

transmission-cleared: yes (2026-09-10 — no private workspace marker, gate N/A)

You are helping build a limits-awareness layer for an open-source tool. This
pack is fully self-contained. You have live web access — that is your edge
here: fetch CURRENT Cloudflare documentation and spot RECENT changes.

## Project identity

- **Project**: cosmoflare — open-source Go CLI + library managing the full
  Cloudflare developer platform (R2, Workers, KV, D1, DNS, and ~20 more
  services). MIT.
- **Goal**: a machine-readable catalog of EVERY documented Cloudflare limit,
  free tier and paid tiers, with provenance (source URL + as-of date), kept
  refreshable as Cloudflare changes limits and billing.
- **Your output feeds**: an embedded JSON catalog. Every number you return
  must be traceable to a URL you actually consulted.

## Mission

For every service in the checklist below, collect every DOCUMENTED limit:
countable quotas (e.g. number of Workers scripts), size constraints (e.g.
max R2 object size, max D1 database size, max Worker script size gzipped),
and rate limits (e.g. requests/day, storage ops/month, API calls/min).

## Rules of engagement (non-negotiable)

1. **Cite every number** with the URL it came from and the date you read it.
2. **Null when unpublished.** If Cloudflare publishes no number (common for
   Enterprise), return `null` — never guess, never extrapolate. A page saying
   "contact sales" or "custom" counts as unpublished: `null`, with the exact
   wording quoted in `notes`.
3. **Prefer current developers.cloudflare.com pages.** Expected locations
   are listed per service — verify them; correct any that moved.
4. **Flag conflicts** between pages (e.g. limits page vs pricing page
   disagree) in a dedicated CONFLICTS section.
5. **Memory is not a source.** If you cannot verify a number on a live page,
   put `unverified:memory` in `notes`, set `source_url: null` and
   `verified_on: null` — and never invent a URL to fill `source_url`.

## Service checklist (~22 groups)

| # | Service | Expected docs location (verify) |
|---|---------|--------------------------------|
| 1 | Workers (script count, script size, CPU time, memory, subrequests, env vars, startup time) | developers.cloudflare.com/workers/platform/limits/ |
| 2 | R2 (object size, single PUT vs multipart, storage, Class A/B ops, buckets, custom domains) | developers.cloudflare.com/r2/platform/limits/ |
| 3 | KV (namespaces, keys per namespace, key length, value size, writes/min) | developers.cloudflare.com/kv/platform/limits/ (or /reference/limits/) |
| 4 | D1 (databases, total storage per DB, rows read/written, queries, timeTravel) | developers.cloudflare.com/d1/platform/limits/ |
| 5 | Queues (queues per account, message size, throughput, consumers per queue, backlog) | developers.cloudflare.com/queues/platform/limits/ |
| 6 | Pages (projects, builds, deployments, concurrent builds, file count/size limits) | developers.cloudflare.com/pages/platform/limits/ |
| 7 | Images (stored images, variants, transformations/month, upload size, animated images) | developers.cloudflare.com/images/platform/limits/ (or /reference/pricing/) |
| 8 | Stream (storage minutes, direct upload limits, max video size/duration, requests) | developers.cloudflare.com/stream/platform/limits/ |
| 9 | Vectorize (indexes, dimensions per vector, vectors per index, queries/sec, metadata size) | developers.cloudflare.com/vectorize/platform/limits/ |
| 10 | Workers AI (neurons, requests, tokens; AI Gateway limits) | developers.cloudflare.com/workers-ai/platform/limits/ + /ai-gateway/ |
| 11 | Hyperdrive (databases per account, connection limits, pool size) | developers.cloudflare.com/hyperdrive/platform/limits/ |
| 12 | DNS records per zone (priors to verify, not sourced: free 1,000 pre-Sep-2024 zones / 200 newer; pro/business 3,500; enterprise account-level) | developers.cloudflare.com/dns/manage-dns-records/ |
| 13 | Zones (zones per account by plan) | developers.cloudflare.com/fundamentals/get-started/concepts/ |
| 14 | SSL/TLS (custom certificates, custom hostnames, SNI certs per zone) | developers.cloudflare.com/ssl/ |
| 15 | Cache (purge limits, cache tags, purge-by-url counts, TTL bounds) | developers.cloudflare.com/cache/ |
| 16 | Page Rules / Redirect Rules (rules per zone by plan) | developers.cloudflare.com/rules/ |
| 17 | WAF / Firewall (custom rules per zone, rate limiting rules, expressions) | developers.cloudflare.com/waf/ |
| 18 | Email Routing (rules per zone, destination addresses) | developers.cloudflare.com/email-routing/ |
| 19 | Healthchecks (healthchecks per account) | developers.cloudflare.com/health-checks/ |
| 20 | Domains / Registrar (domains per account) | developers.cloudflare.com/registrar/ + /cloudflare-for-platforms/ |
| 21 | Cloudflare REST API itself (requests/minute global + per-endpoint; prior to verify: 1,200 req/5min) | developers.cloudflare.com/api/ + /fundamentals/api/ |
| 22 | CORS response headers (count/size limits if any are documented — often none; if handled by Rules, say so) | developers.cloudflare.com/rules/transform/ |

In the JSON, `service` is a short lowercase slug — `workers`, `r2`, `kv`,
`d1`, `queues`, `pages`, `images`, `stream`, `vectorize`, `workers_ai`,
`ai_gateway`, `hyperdrive`, `dns`, `zones`, `ssl`, `cache`, `rules`, `waf`,
`email_routing`, `healthchecks`, `domains`, `api`, `cors` — and each `id` is
`<slug>.<limit_name>` (e.g. `r2.object_size_max`).

## Catalog entry schema (per limit)

```json
{
  "id": "r2.object_size_max",
  "service": "r2",
  "kind": "size",
  "name": "Maximum object size",
  "unit": "bytes",
  "scope": "object",
  "tiers": { "free": 5000000000000, "paid": 5000000000000, "enterprise": null },
  "trackable": false,
  "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
  "verified_on": "2026-09-10T14:32:05Z",
  "notes": ""
}
```

- `kind`: `quota` (countable resource), `size` (one thing's magnitude),
  `rate` (ops per time window).
- `tiers`: keys per service — `free`/`paid` for Workers, R2, D1; KV, Queues,
  Pages, Images etc. as Cloudflare actually tiers them; `null` = unpublished.
- `unit`: `count`, `bytes`, `ms`, `s`, `requests_per_day`, `ops_per_month`,
  `per_minute`, `mb`, `rows` — whatever is true; keep it machine-parseable.
- `scope`: `account | zone | bucket | object | namespace | database | script
  | queue | index | project`.
- `trackable`: true if a live usage counter exists via API (or is derivable
  by listing resources). Max sizes are usually `false`.
- `verified_on`: full ISO8601 with timezone, e.g. `2026-09-10T14:32:05Z` —
  the moment you verified the value, not a bare date.

## YOUR SPECIALIZATION — live web + recent changes

Beyond the checklist: actively hunt for **limit and billing changes**.

1. Search the Cloudflare blog + changelog (blog.cloudflare.com,
   developers.cloudflare.com/changelog) for limit changes in the last 18
   months: "limits", "quota", "Workers free plan", "R2 pricing", "D1 limits".
2. Search X for Cloudflare limit complaints/announcements — users often
   discover undocumented limit changes before docs update.
3. Return a **RECENT CHANGES** section: every limit you found evidence of
   changing recently — old value → new value, effective date, source URL.
4. Where the docs page looks stale vs. an announcement, say so explicitly
   in CONFLICTS.

## Return envelope (exact format)

Return ONE markdown document:

1. For each service (1-22): `## <n>. <Service>` then a table with columns:
   `id | name | kind | unit | scope | free | paid | enterprise | trackable | source_url | verified_on | notes`
   — omit tier columns that don't apply to that service; if a service tiers
   differently (e.g. free/pro/business/enterprise), use those actual tier
   names as columns and the same keys in the JSON `tiers` object. `null` for
   unpublished; `—` for "no such tier exists" (omit that key from `tiers`).
2. `## RECENT CHANGES` — your specialization output.
3. `## CONFLICTS` — page-vs-page, docs-vs-announcement disagreements.
4. `## CATALOG JSON` — one fenced ```json block: an array of entries in the
   schema above, one per table row. This block is what gets parsed — make it
   valid, complete, and consistent with your tables.

Completeness beats elegance. A row with `null` and a URL is a good row. A row
marked `unverified:memory` with `source_url: null` is acceptable — a row
with an invented URL is a failure.
