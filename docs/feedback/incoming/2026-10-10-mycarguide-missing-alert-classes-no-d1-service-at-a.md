---
ulid: 01M4HK9Q8M18GWHZ2FS1ZS896R
id: FB-29
title: 'Missing alert classes: no D1 service at all in alerts (d1-rows-read regression, KV writes/day, cache-hit-rate)'
type: feature
status: pending
priority: high
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T04:27:55.00704+04:00"
updated: "2026-10-10T04:27:55.00704+04:00"
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

# FB-pZS896R: Missing alert classes: no D1 service at all in alerts (d1-rows-read regression, KV writes/day, cache-hit-rate)

# Missing alert classes: no D1 service at all in alerts (d1-rows-read regression, KV writes/day, cache-hit-rate)

**What:** MyCarGuide's BUG-011 documentation claimed a cosmoflare `d1-rows-read` alert guarded regression. Live check 2026-10-10 (`cosmoflare alerts list --json` → empty) plus `alerts create --help` shows the alert system supports services r2/workers/kv/dns only — **there is no D1 service and no rows-read condition at all**, so the claimed alert could never have existed. Docs on our side have been corrected.

**Why:** D1 rows-read is the single most expensive resource class in the fleet (MyCarGuide: 3.06B rows/day peak = 3.6x the 25B/mo allowance). An audit platform without an alert class for its hottest meter is missing the one guard that pays for itself. MyCarGuide shipped an interim tripwire inside its snapshot script (`--d1-warn-millions`, default 500M/day, warns on the last full day) — that logic belongs in cosmoflare.

**Proposed alert classes:** d1-rows-read-per-day (threshold, per database), d1-rows-written-per-day, kv-writes-per-day, workers-invocations-per-day, and zone-cache-hit-rate-floor (fire when rolling hit rate drops below X% — catches "a caching layer silently stopped working", exactly the failure that made BUG-011 fix 3 a no-op until live-verified). Condition payloads should carry the measured value, the threshold, and the window.

**Evidence:** companion FBs FB-p0RWGMB (storage metric 86x off), FB-pR22EFG (analytics CLI), corpus FB 01M4HHTVMZW101AQCCEJT3ZHNM; tripwire implementation at MyCarGuide `infra/scripts/cf-usage-snapshot.mjs`.

**Priority:** high — the fleet's costliest resource has no guard.

