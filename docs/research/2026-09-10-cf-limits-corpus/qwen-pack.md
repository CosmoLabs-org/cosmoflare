# RESEARCH PACK — Cloudflare Plan Limits Corpus (QWEN)

transmission-cleared: yes (2026-09-10 — no private workspace marker, gate N/A)

You are helping build a limits-awareness layer for an open-source tool. This
pack is fully self-contained. Your edge here: **technical depth** — the edge
cases, the enforceability questions, the numbers that only matter in
practice.

## Project identity

- **Project**: cosmoflare — open-source Go CLI + library managing the full
  Cloudflare developer platform (R2, Workers, KV, D1, DNS, and ~20 more
  services). MIT.
- **Goal**: a machine-readable catalog of EVERY documented Cloudflare limit,
  free tier and paid tiers, with provenance (source URL + as-of date), kept
  refreshable as Cloudflare changes limits and billing.
- **Phase 2 of this project is enforcement**: before uploading a file, the
  CLI checks the file size against the R2 object-size limit locally; before
  creating resource N+1, it checks the count quota. Your enforceability
  analysis directly shapes that.

## Mission

For every service in the checklist below, collect every DOCUMENTED limit:
countable quotas, size constraints, and rate limits — with special attention
to the EDGE CASES that break naive implementations.

## Rules of engagement (non-negotiable)

1. **Cite every number** with the URL it came from and the date you read it.
2. **Null when unpublished.** No published number (common for Enterprise) →
   `null`. Never guess, never extrapolate.
3. **Prefer current developers.cloudflare.com pages.** Verify expected
   locations; correct any that moved.
4. **Flag conflicts** in a dedicated CONFLICTS section.
5. **Memory is not a source.** For a number you recall but cannot verify:
   set the tier value to `null`, set `source_url` and `verified_on` to
   `null`, and tag `unverified:memory` plus your recall in `notes`. Never
   invent a URL to fill `source_url`.

## Service checklist (~21 groups)

| # | Service | Edge cases to chase | Expected docs location (verify) |
|---|---------|--------------------|--------------------------------|
| 1 | Workers | script size: raw vs gzipped; CPU soft vs hard limits; free-plan daily request cap enforcement time zone; subrequest limits per fetch vs total; env var byte limits | developers.cloudflare.com/workers/platform/limits/ |
| 2 | R2 | single PUT max vs multipart; multipart part count × part size ceiling; max parts; upload methods (S3 API, presigned); object metadata size | developers.cloudflare.com/r2/platform/limits/ |
| 3 | KV | value size with/without metadata; writes/min enforcement window; eventual-consistency read limits | developers.cloudflare.com/kv/platform/limits/ (or /reference/limits/) |
| 4 | D1 | DB size (GB) vs rows-read/written billing quotas; 5-minute query window; import/export size; time travel retention | developers.cloudflare.com/d1/platform/limits/ |
| 5 | Queues | message size with/without headers; max batch size; consumer concurrency; retry/ack grace; max delivery attempts | developers.cloudflare.com/queues/platform/limits/ |
| 6 | Pages | max files per deployment; single file size cap; build minutes; _headers/_redirects rules count; functions limits | developers.cloudflare.com/pages/platform/limits/ |
| 7 | Images | upload size; variants cap; transformations/month; flexible variants; animated (GIF) specifics | developers.cloudflare.com/images/platform/limits/ (or /reference/pricing/) |
| 8 | Stream | max video size + duration; direct uploads concurrent; recorded minutes vs stored minutes; signed URL limits | developers.cloudflare.com/stream/platform/limits/ + /stream/pricing/ |
| 9 | Vectorize | dims per vector (Pinecone-comparable), metadata bytes, queries/sec per index, max indexes, max vector id length | developers.cloudflare.com/vectorize/platform/limits/ + /vectorize/billing/ |
| 10 | Workers AI + AI Gateway | neurons/plan, tokens/min, context window per model (catalog-level), gateway request size | developers.cloudflare.com/workers-ai/platform/limits/ + /ai-gateway/ |
| 11 | Hyperdrive | databases/account, concurrent connections, connection retention | developers.cloudflare.com/hyperdrive/platform/limits/ |
| 12 | DNS | records/zone by plan; free-zone 2024-09 cutoff; API rate limits for DNS endpoints specifically | developers.cloudflare.com/dns/manage-dns-records/ |
| 13 | Zones | zones/account per plan tier | developers.cloudflare.com/fundamentals/ |
| 14 | SSL/TLS | custom certs/zone, custom hostnames/zone, SNI/total certs | developers.cloudflare.com/ssl/ |
| 15 | Cache | purge-by-URL batch max (30? verify), purge API rate, cache tag count/entry, TTL min/max | developers.cloudflare.com/cache/ |
| 16 | Rules | page rules/zone by plan; modern rules equivalents (redirects, transforms, config rules counts) | developers.cloudflare.com/rules/ |
| 17 | WAF | custom rules/zone by plan, rate limiting rules, expression size/complexity | developers.cloudflare.com/waf/ |
| 18 | Email Routing | rules/zone, destinations, message size, attachments | developers.cloudflare.com/email-routing/ |
| 19 | Healthchecks | healthchecks/account, checks/min, regions | developers.cloudflare.com/health-checks/ |
| 20 | Domains/Registrar | domains/account, transfer locks, TLD availability | developers.cloudflare.com/registrar/ |
| 21 | Cloudflare REST API | global rate (1,200/5min? verify), per-endpoint overrides (DNS records, Workers scripts, R2), 429 semantics + Retry-After | developers.cloudflare.com/api/ + /fundamentals/api/ |

## Catalog entry schema (per limit)

One illustrative example — its values are UNVERIFIED; do not copy them:

```json
{
  "id": "queues.message_size_max",
  "service": "queues",
  "kind": "size",
  "unit": "bytes",
  "name": "Maximum message size",
  "scope": "queue",
  "tiers": { "free": null, "paid": 131072, "enterprise": null },
  "trackable": false,
  "enforceability": "local",
  "soft": false,
  "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
  "verified_on": "2026-09-10T09:15:00Z",
  "notes": "values illustrative — verify"
}
```

- `kind`: `quota` | `size` | `rate`. `unit`: machine-parseable (e.g.
  `count`, `bytes`, `mb`, `ms`, `s`, `requests_per_day`, `ops_per_month`,
  `per_minute`, `per_second`, `rows`, `minutes`, `dimensions`; add new
  units in the same style when a limit needs one).
- `scope`: `account | zone | bucket | object | namespace | database |
  script | queue | index | project`.
- `tiers`: keys as the service actually tiers; `null` = unpublished.
- `trackable`: true if a live counter exists or is derivable via API.
- `enforceability`: `local | api-list | api-counter | opaque` (defined in
  your specialization, item 2). If more than one applies, pick the
  cheapest: `local` > `api-list` > `api-counter` > `opaque`.
- `soft`: `true` = soft (burst allowed, throttled after); `false` = hard
  (request rejected); `null` = enforcement behavior undocumented. The
  `soft/hard` table column shows `soft` / `hard` / `?` respectively.
- `verified_on`: full ISO8601 with timezone (e.g. `2026-09-10T09:15:00Z`);
  never date-only.

## YOUR SPECIALIZATION — edge cases + enforceability

1. **Edge-case register.** Every limit with a subtlety that breaks naive
   checks — measured-before/after-compression, with/without-headers,
   soft (exceeded → throttled) vs hard (exceeded → rejected), time-window
   semantics (calendar day UTC vs rolling 5-min vs calendar month).
2. **Enforceability column.** For each entry: `local` (checkable offline —
   file size, config counts), `api-list` (derivable by listing — script
   count), `api-counter` (needs an analytics/metrics endpoint — daily
   requests, ops/month), or `opaque` (no practical client-side check).
   This drives which limits our phase-2 guards can actually enforce.
3. **Soft-limit semantics.** Explicitly mark limits that are soft (burst
   allowed, throttled after) vs hard (request rejected). Different guard
   behavior for each.
4. **429 semantics** for API rate limits: Retry-After behavior, per-endpoint
   vs global buckets — our CLI must back off correctly.

## Return envelope (exact format)

Return ONE markdown document:

1. For each service (1-21): `## <n>. <Service>` then a table with columns:
   `id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes`
   — include only tier columns that exist (tier names as the service
   actually names them); `null` = unpublished; `—` = no such tier.
2. `## EDGE CASES` — the register from your specialization (1), including
   the 429 / Retry-After semantics from specialization (4).
3. `## CONFLICTS` — page-vs-page disagreements.
4. `## CATALOG JSON` — one fenced ```json block: an array of entries in
   the schema above, one per table row — all schema fields, including
   `enforceability`, `soft`, and `trackable`; take `service` from the
   section heading. Valid JSON; consistent with your tables.

Completeness beats elegance. A row with `null` and a URL is a good row.
