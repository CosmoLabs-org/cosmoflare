---
title: Cache-miss alert rides a tiny base - percentage alerts page without volume floor or absolute numbers
severity: medium
status: fixed
date_reported: "2026-10-12T04:40:00+04:00"
date_fixed: "2026-10-12T05:05:00+04:00"
session: 2038
id: BUG-pEPB03A
introduced_in: Ops cron alert engine starter rules (FEAT-052, 2026-10-09)
commits: []
affected_files:
    - apps/ops/src/rules.ts
    - apps/ops/src/scheduled.ts
    - apps/ops/src/summary.ts
    - apps/ops/src/scheduled.test.ts
    - apps/ops/src/rules.test.ts
    - apps/ops/src/summary.test.ts
    - pager/src/alerts.ts
    - pager/src/app/views/AlertsView.tsx
    - pager/src/app/views/AlertsView.test.tsx
related_issues:
    - FEAT-074
---

# Cache-miss alert rides a tiny base — percentage alerts page without volume floor or absolute numbers

## Problem

The `zone-cache-miss-pct` starter rule (threshold 50%) judged any zone above
`ZONE_CACHE_FLOOR = 100` eligible requests per 24h. A percentage on a base
that small is statistical noise, but the rule paged on it — and the page
carried a bare percentage with no absolute base, so the operator could not
judge it without opening the dashboard.

## Evidence (live data, 2026-10-12)

Read from the deployed worker's KV (`alert-feed`, `summary`):

- `illuminate.am` alerted at **50.4% miss on 139 eligible requests** —
  46 misses + 16 expired out of 1,944 total requests (70% dynamic, 372
  bypassed cache entirely).
- That is **62 origin requests per day** — no cost problem, no caching
  problem, and nothing code or Cloudflare settings should change: the zone
  is dynamic by design.
- The 24h rolling window slides the ratio across the 50% line, so the rule
  re-fired nearly every cooldown. Under the old 1h cooldown this produced
  the flood of "cache miss rate" notifications the operator reported.

## Fix

Two changes, both in the alerting path (the summary view still measures
every zone):

1. **Alerting floor** — `MISS_PCT_ELIGIBLE_FLOOR = 1000` eligible
   requests/day in `rules.ts` `conditionValues`. Below it, a zone's miss
   ratio is measured but never pages. illuminate.am (139 eligible) stops
   alerting without any operator action.
2. **Ratios travel with their base** — `ZoneObs`/`ZoneRow` now carry the
   absolute `misses` count; `Fire`/`AlertFeedEntry`/the push body carry a
   `context` line ("700 misses of 1100 eligible requests"). The pager's
   feed detail renders it. A percentage can no longer arrive alone.

## Also caught while fixing (same change set)

- **`%` formatting wart in the push body**: `fireBody` rendered
  "63.6 % in 24h" (unit appended after a space). Now `%` conditions render
  "63.6% in 24h — threshold 50.0%".
- **Raw float in the pager feed**: the wire carries full precision
  (`50.40650406504065`) and `FeedRow`/`FeedDetail` printed it verbatim —
  invisible while only integer counts fired, ugly the moment a `%` entry
  landed. Added `feedNum()` (`%` → one decimal, counts plain).
- **Test-fixture drift**: `AlertFeedEntry.context` is required; the pager
  test factory omitted it and `tsc` caught it. (Process note: an earlier
  verification piped `tsc | tail` and read `$?` from `tail` — the failure
  was silently treated as success. Exit codes are now read from `tsc`
  directly.)

## Verification

- `apps/ops`: 180 tests green (new: floor suppresses a 50.4%-on-139 fire;
  above-floor fire carries context; push body includes the base), tsc clean.
- `pager`: 278 tests green (context line renders in the detail), tsc clean,
  build green, `impeccable detect` 0.
- illuminate.am needs no exclusion: the floor removes it from alerting
  automatically.
