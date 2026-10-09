---
ulid: 01M4F65HZ9TMGF9XK64RW1Z6AD
from: cosmoflare
type: feature
severity: medium
title: 'cosmolearning.org: 36k uncached requests/day on public listing pages + RSS — cache at the edge'
created: "2026-10-09T05:59:12.617378+04:00"
---

# cosmolearning.org: 36k uncached requests/day on public listing pages + RSS — cache at the edge

Evidence from cosmoflare live analytics, 2026-10-09 (read-only, last 24h, eyeball traffic).

## What is happening

cosmolearning.org served 36,321 of 53,052 eyeball requests uncached (`cacheStatus=dynamic`) — the highest uncached volume of all 42 CosmoLabs zones. Every one is a billed Worker invocation and, behind it, D1 queries. The top uncached paths are public, identical-for-everyone pages:

| Path | Uncached requests/24h |
|---|---|
| `GET /documentaries` | 1,211 |
| `GET /courses` | 989 |
| `GET /` | 934 |
| `GET /rss/latest-courses-added/english-language/` | 288 |
| `GET /history/documentaries` | 218 |
| `GET /rss/latest-videos-added/environment/` | 180 |
| `GET /rss/latest-videos-added/english-language/` | 179 |
| `GET /mathematics/courses`, `/beauty/courses`, `/communication/courses` | 140-177 each |

The D1 side is healthy: `cosmolearning` read ~3.3M rows/day across ~70k queries (about 46 rows/query). The top query (`SELECT id, lectureNumber, name FROM videoLectures WHERE courseID = ? AND online = 1 ORDER BY lectureNumber ASC, id ASC`, 10,023 runs) reads ~303 rows to return ~151 — fine, and an index on `videoLectures(courseID, online, lectureNumber)` would bring it to ~151 if you want the last bit.

## Fix

Cache the public listing pages and RSS feeds at the edge. These responses are the same for every visitor and change only when content is added:

- Emit `Cache-Control: public, s-maxage=3600` (or longer for RSS) on listing pages, subject pages and RSS feeds, and serve them through the Workers Cache API or a KV cache with path-based keys.
- Never cache anything user-specific (comments forms, account, admin).
- Reference implementation of the CosmoLabs CF efficiency standard: Churches-app commit 155d1d3e (IMP-017) — KV cache middleware with query-aware keys, `Cache-Control` emitted on hits and misses, opt-in routes only so auth routes stay uncacheable by construction.

## Verify

Repeat-probe: `for i in 1 2; do curl -sD - -o /dev/null https://cosmolearning.org/courses | grep -iE 'cf-cache-status|cache-control'; done` — the second response should show `cf-cache-status: HIT`. cosmoflare's `zone-uncached-requests` alert (threshold 10000/24h) currently fires for this zone and will clear once the volume drops.

**Priority:** medium — cost and latency, no correctness impact.
