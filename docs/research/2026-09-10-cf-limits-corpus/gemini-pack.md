# RESEARCH PACK — Cloudflare Plan Limits Corpus (GEMINI)

transmission-cleared: yes (2026-09-10 — no private workspace marker, gate N/A)

You are helping build a limits-awareness layer for an open-source tool. This
pack is fully self-contained. Your edge here: **long-context exhaustive
synthesis** — read everything relevant, miss nothing, structure it into
complete comparison tables.

## Project identity

- **Project**: cosmoflare — open-source Go CLI + library managing the full
  Cloudflare developer platform (R2, Workers, KV, D1, DNS, and ~20 more
  services). MIT.
- **Goal**: a machine-readable catalog of EVERY documented Cloudflare limit,
  free tier and paid tiers, with provenance (source URL + as-of date), kept
  refreshable as Cloudflare changes limits and billing.
- **Your output feeds**: an embedded JSON catalog. Every number you return
  must be traceable to a URL — the only exception is rule 5's
  `unverified:memory` rows, which carry no URL by definition.

## Mission

For every service in the checklist below, collect every DOCUMENTED limit:
countable quotas (e.g. number of Workers scripts), size constraints (e.g.
max R2 object size, max D1 database size, max Worker script size gzipped),
and rate limits (e.g. requests/day, storage ops/month, API calls/min).

## Rules of engagement (non-negotiable)

1. **Cite every number** with the URL it came from and the date you read it.
2. **Null when unpublished.** If Cloudflare publishes no number (common for
   Enterprise), return `null` — never guess, never extrapolate.
3. **Prefer current developers.cloudflare.com pages.** Expected locations
   are listed per service — verify them; correct any that moved. Also check
   each service's pricing page where limits and included quotas interleave.
4. **Flag conflicts** between pages in a dedicated CONFLICTS section.
5. **Memory is not a source.** If you recall a number but cannot verify it
   on a live page, do NOT return it as data: set the tier value `null`, put
   `unverified:memory` plus your recall in `notes`, and set `source_url`
   and `verified_on` to `null`.

## Service checklist (~21 groups)

| # | Service | Collect at minimum | Expected docs location (verify) |
|---|---------|--------------------|--------------------------------|
| 1 | Workers | script count/plan, script size (raw vs gzipped!), CPU time per request (soft/hard), memory, simultaneous subrequests, env var count/size, startup time, duration limits | developers.cloudflare.com/workers/platform/limits/ |
| 2 | R2 | object size, single-request upload max, multipart part size/count, storage/plan, Class A and Class B ops (free included + caps), buckets, custom domains/bucket | developers.cloudflare.com/r2/platform/limits/ + /r2/pricing/ |
| 3 | KV | namespaces/account, keys/namespace, key length, value size, writes/min/namespace, list operations | developers.cloudflare.com/kv/platform/limits/ (or /reference/limits/) + /kv/pricing/ |
| 4 | D1 | databases/account, max DB size, rows read/written (5-min window), queries, time travel window, concurrent queries | developers.cloudflare.com/d1/platform/limits/ + /d1/pricing/ |
| 5 | Queues | queues/account, message size, throughput (msgs/s), consumer concurrency, max consumers/queue, ack grace, backlog retention | developers.cloudflare.com/queues/platform/limits/ |
| 6 | Pages | projects, builds/month, concurrent builds, deployments/project, file count + max file size, headers/redirects config limits | developers.cloudflare.com/pages/platform/limits/ |
| 7 | Images | stored images cap, variants, transformations/month, upload size, animated limits, delivery | developers.cloudflare.com/images/platform/limits/ (or /reference/pricing/) |
| 8 | Stream | minutes stored (free/paid caps), direct upload count, max video size + duration, copies, watermark rules | developers.cloudflare.com/stream/platform/limits/ + /stream/pricing/ |
| 9 | Vectorize | indexes/account, vectors/index, dimensions/vector, vector + metadata size, queries/index/s, distance metrics | developers.cloudflare.com/vectorize/platform/limits/ + /vectorize/billing/ |
| 10 | Workers AI + AI Gateway | neurons included/plan, requests, tokens, models-per-tenant, AI Gateway requests | developers.cloudflare.com/workers-ai/platform/limits/ + /ai-gateway/ |
| 11 | Hyperdrive | databases/account, connection pooling, query caching | developers.cloudflare.com/hyperdrive/platform/limits/ |
| 12 | DNS | records/zone by plan (free zones created before/after 2024-09 split; pro; business; enterprise account-level) | developers.cloudflare.com/dns/manage-dns-records/ |
| 13 | Zones | zones/account by plan tier | developers.cloudflare.com/fundamentals/ |
| 14 | SSL/TLS | custom certs, custom hostnames, SNI certs/zone | developers.cloudflare.com/ssl/ |
| 15 | Cache | purge by URL batch size, purge rate, cache-tag limits, TTL bounds | developers.cloudflare.com/cache/ |
| 16 | Rules | page rules/zone (with Legacy vs modern rules note), redirect rules, transform rules counts | developers.cloudflare.com/rules/ |
| 17 | WAF | custom rules/zone by plan, rate-limiting rules, expression complexity | developers.cloudflare.com/waf/ |
| 18 | Email Routing | rules/zone, destination addresses, message size | developers.cloudflare.com/email-routing/ |
| 19 | Healthchecks | healthchecks/account, checks/min | developers.cloudflare.com/health-checks/ |
| 20 | Domains/Registrar | domains/account, supported TLD constraints | developers.cloudflare.com/registrar/ |
| 21 | Cloudflare REST API | global requests/5-min, per-endpoint overrides (esp. DNS, Workers, R2) | developers.cloudflare.com/api/ + /fundamentals/api/ |

## Catalog entry schema (per limit)

The example below shows FIELD SHAPE only. Its values are illustrative
placeholders, NOT real limits — do not copy them; look up every value.

```json
{
  "id": "workers.script_size_max",
  "service": "workers",
  "kind": "size",
  "name": "Maximum script size after gzip",
  "unit": "bytes",
  "scope": "script",
  "tiers": { "free": 1048576, "paid": 10485760 },
  "trackable": true,
  "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
  "verified_on": "2026-09-10T14:32:05Z",
  "notes": "values illustrative — verify"
}
```

- `kind`: `quota` (countable resource), `size` (one thing's magnitude),
  `rate` (ops per time window).
- `tiers`: keys as Cloudflare actually tiers the service — `free`/`paid` for
  Workers, R2, D1; `free`/`pro`/`business`/`enterprise` for zone-scoped;
  `null` = unpublished.
- `unit`: `count`, `bytes`, `mb`, `ms`, `s`, `requests_per_day`,
  `ops_per_month`, `per_minute`, `per_5_minutes`, `per_second`, `rows`,
  `minutes`, `dimensions` — machine-parseable, no prose.
- `scope`: `account | zone | bucket | object | namespace | database |
  script | queue | index | project`.
- `trackable`: true if a live usage counter exists via API or is derivable
  by listing resources (script sizes: yes, deployment API returns sizes;
  max object size: no).
- `verified_on`: full ISO8601 with timezone, e.g. `2026-09-10T14:32:05Z` —
  the moment you verified the value, not a bare date.

## YOUR SPECIALIZATION — exhaustive synthesis + cross-page consistency

1. **Sweep wide.** For each service, check BOTH the limits page AND the
   pricing page — included quotas (free-tier Class A ops, D1 rows included,
   Stream minutes included) live on pricing pages and ARE limits for our
   purposes. Emit them as catalog entries too (kind `rate` or `quota` with
   notes "included quota").
2. **Cross-page consistency table.** Where limits page and pricing page
   describe the same quota, verify they agree; disagree → CONFLICTS.
3. **Zone-plan axes.** For zone-scoped services (DNS, rules, WAF, SSL),
   give the full free/pro/business/enterprise axis, not just free/paid.
4. **Raw vs compressed.** Where a size limit is defined post-compression
   (Workers script size), say so in `notes` and use the compressed number.

## Return envelope (exact format)

Return ONE markdown document:

1. For each service (1-21): `## <n>. <Service>` then a table with columns:
   `id | service | name | kind | unit | scope | free | paid | pro | business | enterprise | trackable | source_url | verified_on | notes`
   — include only the tier columns that exist for that service; `null` for
   unpublished; `—` for "no such tier". Every non-tier column is required
   on every row: table columns and JSON fields must map 1:1 (`service` =
   lowercase slug, e.g. `workers_ai` vs `ai_gateway` for service 10).
2. `## INCLUDED QUOTAS` — pricing-page included quotas you elevated to
   entries (they also appear in section 1 tables + JSON).
3. `## CONFLICTS` — cross-page disagreements with both URLs.
4. `## CATALOG JSON` — one fenced ```json block: an array of entries in the
   schema above, one per table row. This block gets parsed — valid JSON,
   complete, consistent with your tables.

Completeness beats elegance. A row with `null` and a URL is a good row.
