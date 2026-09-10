---
title: Cloudflare Limits Awareness Layer — Security + Budget Foundation
created: 2026-09-10T19:28:56+04:00
status: design approved in session — pending corpus acquisition
deliverables:
  - id: BR-01
    title: "catalog.json (schema v1) — full service checklist, every entry with source_url + verified_on"
  - id: BR-02
    title: "Research corpus — 3 packs + 3 results files + conflicts register in docs/research/2026-09-10-cf-limits-corpus/"
  - id: BR-03
    title: "LimitsService v2 — reads embedded catalog, deletes hardcoded maps, per-service plan resolution, stale-entry warning"
  - id: BR-04
    title: "Tracking expansion — Snapshot rows for every trackable resource across all covered services"
  - id: BR-05
    title: "cosmoflare limits catalog --json export (catalog as data API for desktop/mobile/MCP)"
  - id: BR-06
    title: "Phase 2/3 spec sections (guards D4, alerts/budget D5) accepted as contract for next builds"
---

# Cloudflare Limits Awareness Layer — Security + Budget Foundation

**Created:** 2026-09-10T19:28:56+04:00
**Status:** design approved in session — pending corpus acquisition
**Research pack:** `docs/research/2026-09-10-cf-limits-corpus/`
**Ingestion prompt:** `docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md`

## Why

Cosmoflare's differentiator is full-platform awareness. The one thing no
Cloudflare tool does well: know every documented plan limit (free and paid)
across all services, measure how close the account sits to each one, and
refuse operations that would cross a limit. That turns cosmoflare from a
wrapper into a **security and budget layer**: quota drift becomes visible
before it becomes an outage or a surprise bill.

Cloudflare changes limits and billing continuously. Therefore the limit
values are **data, not code** — a versioned catalog with provenance that we
refresh on a cadence.

## Existing foundation (v0.25.0)

- `pkg/cosmoflare/limits.go` — `LimitsService.Snapshot()` joins live usage
  counts against documented limits. Covers 6 resources: `workers.scripts`,
  `workers.daily_requests`, `r2.buckets`, `r2.custom_domains_per_bucket`,
  `zones.count`, `dns.records`.
- `cmd/limits.go` — `cosmoflare limits [--bucket] [--plan] [--json]`, rows
  sorted by percent-of-limit.
- Already-solved hard parts: partial-failure tolerance (`SourceError`),
  `LimitSource` honesty (`static-docs | live-api | unknown`), Workers plan
  resolution chain (subscriptions API → flag → config → unknown), live DNS
  quota API with static fallback.

## Decision history (session 2026-09-10)

| # | Question | Decision |
|---|----------|----------|
| 1 | V1 scope | **Phased: track → guard → alert.** V1 = full catalog + tracking. Guards (phase 2) and alerts/budget (phase 3) are specified in this doc, built after the catalog proves accurate. |
| 2 | Catalog coverage | **Full platform sweep.** All covered services, every documented limit: countable quotas, size constraints, rate limits. Constraint-only limits (max upload size) ship as reference data even without a live counter. |
| 3 | Catalog storage | **Embedded JSON catalog** via `go:embed`. Not Go maps (limits change → data edit, not code release), not generated code (generation step per update), not external file (install drift + trust). |

## D1 — Catalog schema

`pkg/cosmoflare/limitsdata/catalog.json`, embedded, `schema_version: 1`:

```json
{
  "schema_version": 1,
  "catalog_version": "2026-09-10",
  "entries": [
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
      "notes": "null = no published number; enterprise account-specific"
    }
  ]
}
```

Rules:

- `id` = `service.resource`, stable forever — it appears in CLI output, JSON,
  alerts, and the desktop/mobile tiers.
- `kind` = `quota` (countable resource count), `size` (bytes/dimensions of a
  single thing), `rate` (operations per time window).
- `tiers` keys are per-service (`free`/`paid` for Workers, R2, D1;
  `free`/`pro`/`business`/`enterprise` for zones). Value `null` means "no
  published number" — the tool reports unknown, never fabricates.
- `trackable` = a live usage counter exists (or is derivable) → the resource
  gets a Snapshot row. `false` → reference data for guards and docs only.
- `verified_on` = full ISO8601 timestamp with timezone (IMP-032) of the
  moment the value was checked against `source_url`. Three research sources
  verify the same limit within one day — date-only loses that ordering.
- Optional fields from the qwen research track: `enforceability`
  (`local | api-list | api-counter | opaque`, cheapest-wins precedence) and
  `soft` (`true` soft/throttled, `false` hard/rejected, `null` unknown).
  They drive phase-2 guard behavior (D4).

The catalog becomes the **single source of truth** (constitutional rule).
`limitFor()` in `limits.go` becomes a catalog lookup; the hardcoded maps are
deleted. `cosmoflare limits catalog --json` exports it for other tiers.

## D2 — Per-service plan resolution

Extend the Workers-only chain to a per-service map. Order per service:
subscriptions API → `--plan <service>=<tier>` flag → `.cosmoflare.yaml`
`plans: { workers: paid, r2: free }` → unknown. Workers, R2, D1 resolve from
one subscriptions call; zones carry their plan on the zone object (already
used by `dnsRecordsStaticLimit`).

## D3 — Freshness

- Every entry carries `verified_on` + `source_url`.
- `cosmoflare limits` prints a warning when any row's entry is >90 days
  stale; `--json` carries `verified_on` per row.
- Refresh workflow: re-run the research pack (same prompt, new date) → paste
  results → `/run-continuation` ingestion → PR updates `catalog.json` +
  corpus snapshot. Data-only diff, no code review surface beyond schema.

## D4 — Phase 2: guards (pre-flight enforcement)

- Local checks first: `cosmoflare object put/copy/sync` stat the file and
  compare against `r2.object_size_max` (and multipart thresholds) before any
  network call. Zero API cost.
- Count checks: `apply`/`create` compare desired counts against a cached
  snapshot (`.cosmoflare-limits-cache.json`, TTL 24h) for quota kinds.
- Refusal is a hard error with a distinct exit code (proposal: exit 3) and an
  actionable message naming the catalog id, the limit, and the override flag.
- `--no-verify` escape hatch; refusal events feed the audit log.

## D5 — Phase 3: alerts + budget

- `alerts.go` gains a `limit` condition type: `when: r2.storage > 80%` —
  evaluated against catalog + snapshot, next to the existing error-rate and
  failure rules.
- `cost.go` reads catalog included-quota metadata (e.g. free-tier included
  Class A/B operations, D1 included rows read) for budget projection: "at
  current rate you cross the paid boundary on <date>".
- Desktop/mobile tiers render the same snapshot (library-first: they wrap
  `LimitsService`, no separate implementation).

## D6 — Research corpus (this is what feeds the catalog)

`docs/research/2026-09-10-cf-limits-corpus/` contains three self-contained
packs (Grok: live web + recent-change detection; Gemini: long-context
exhaustive tables; Qwen: technical edge cases + enforceability). Full-text
results are pasted back as `*-results.md` — that is the provenance corpus.
The ingestion prompt cross-checks the three sources, flags conflicts, and
drafts `catalog.json` from the consensus.

Service checklist for the sweep (cosmoflare's covered services + the API
itself): Workers, R2, KV, D1, Queues, Pages, Images, Stream, Vectorize,
Workers AI / AI Gateway, Hyperdrive, DNS records, Zones, SSL/TLS, Cache,
Page Rules / Redirect Rules, WAF / Firewall, Email Routing, Healthchecks,
Domains / Registrar, CORS response headers, Cloudflare API rate limits.

## Non-goals (v1)

- No behavior change to any mutating command (guards are phase 2).
- No Enterprise NDA values — unpublished stays `null`.
- No automatic scraping/refresh — the cadence is a human-run research pass.
- No cost-table maintenance (pricing is out of scope; included quotas that
  affect limits ARE in scope).

## Deliverables

- **BR-01** — `catalog.json` (schema v1) covering the full service checklist,
  every entry with `source_url` + `verified_on`, derived from the research
  corpus. Blocking for everything else.
- **BR-02** — Corpus: 3 packs + 3 results files in
  `docs/research/2026-09-10-cf-limits-corpus/`, plus a conflicts register
  from ingestion.
- **BR-03** — `LimitsService` v2: reads catalog via `go:embed`, deletes
  hardcoded maps, per-service plan resolution (D2), stale-entry warning (D3).
- **BR-04** — Tracking expansion: Snapshot rows for every `trackable: true`
  resource across Workers, R2, KV, D1, Queues, Pages, Images, Stream,
  Vectorize, Hyperdrive, zones/DNS (existing), Healthchecks.
- **BR-05** — `cosmoflare limits catalog --json` export (catalog as data API
  for desktop/mobile/MCP).
- **BR-06** — Phase 2/3 spec sections (D4, D5) reviewed and accepted as the
  contract for the next two builds.
