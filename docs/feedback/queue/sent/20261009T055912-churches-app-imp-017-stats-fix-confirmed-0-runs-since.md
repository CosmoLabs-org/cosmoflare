---
ulid: 01M4F65J0CFY51JW87CEH589XF
from: cosmoflare
type: feature
severity: low
title: 'IMP-017 stats fix confirmed (0 runs since deploy) — remaining: uncached geojson, scanner probes, mappings index'
created: "2026-10-09T05:59:12.652027+04:00"
---

# IMP-017 stats fix confirmed (0 runs since deploy) — remaining: uncached geojson, scanner probes, mappings index

Follow-up from cosmoflare on the 2026-10-08 CF efficiency audit (IMP-017, commit 155d1d3e). Evidence: live D1 per-query analytics and zone analytics, read-only, 2026-10-09.

## Confirmed: the stats fix works

`/api/meta/stats`'s 18-subquery aggregate (~939k rows read per run) ran 35 times in the 24h before the deploy and **0 times** in the 9h after it (2026-10-08 16:40Z → 2026-10-09 01:54Z). The materialized snapshot is doing its job. Next check after the 03:30 UTC cron: the aggregate should appear exactly once per day.

## Remaining items (small)

1. **Static GeoJSON served uncached.** `GET /geojson/50m/FR.json` (44/24h), `/geojson/50m/AR.json` (20/24h) and siblings return `cacheStatus=dynamic`. These are static files — serve them with `Cache-Control: public, max-age=86400, immutable` (or as Pages static assets) so the edge holds them.
2. **Scanner probes reach the Worker.** `/txets.php`, `/wordpress/`, `/wp/` and similar return `dynamic` — each is a billed Worker invocation. A WAF custom rule that blocks common scanner paths (`.php`, `wp-`, `.env`, `.git`) stops them at the edge (Free plan: 5 custom rules per zone, no regex — use `contains`; https://developers.cloudflare.com/waf/custom-rules/, updated Aug 25, 2026).
3. **Low-volume maintenance queries scan `external_church_mappings`.** `SELECT COUNT(*) … WHERE church_id IN (…)` and the matching `DELETE` read ~36.5k rows per run (8 runs/24h). If `external_church_mappings(church_id)` has no index, add one; low priority at this volume.

Note on the miss rate: churches.app shows ~52% cache misses on cache-eligible requests, mostly `/marquee/*.webp` images with ~20 requests each. At this traffic level that is cold edge caches across many locations, not a bug — it will improve as traffic grows.

**Priority:** low/medium — the main fix landed; these are polish.
