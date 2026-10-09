---
ulid: 01M4H2PHH81T4FG6F96BHT5JDH
id: FB-19
title: 'TTL strategy engine: recommend cache lifetimes from data-change cadence + invalidation coverage (owner-requested)'
type: feature
status: pending
priority: high
complexity: ""
from_project: Churches-app
from_path: /Users/gabstudio/PROJECTS/Churches-app
to_project: cosmoflare
to_target: project
created: "2026-10-09T23:37:03.784062+04:00"
updated: "2026-10-09T23:37:03.784062+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 47
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

# FB-pHT5JDH: TTL strategy engine: recommend cache lifetimes from data-change cadence + invalidation coverage (owner-requested)

What happened: owner reviewed the Churches-app caching layer (18 GET routes, KV TTLs 10min-1h, Cache-Control 300s, daily materialization) and asked: is the cache time enough, given the information rarely changes? The owner explicitly notes NOT deep-knowing Cloudflare caching semantics — which is exactly the knowledge cosmoflare should carry for every CosmoLabs project. This feedback requests a TTL-strategy capability.

The domain knowledge to encode (validated on Churches-app):
1. TTL is a function of CHANGE CADENCE x INVALIDATION COVERAGE, not a universal constant. Churches-app data changes via weekly migrations + rare user submissions, so 1h KV TTL is safe-conservative; a news site would need seconds.
2. Layer asymmetry is the key insight most projects miss: KV/response caches CAN be invalidated programmatically (we invalidate church keys on update), but the browser/edge Cache-Control layer CANNOT be targeted from a migration — so its max-age must stay short (300s) even when KV TTLs are hours. Long edge TTLs without a purge path = uncontrollable staleness.
3. Materialized snapshots follow the cron cadence (daily for stats that only change when data changes), with a short response TTL to cover deploy gaps.
4. Stale-until-invalidated is the upgrade path: rare-change data can take 6-24h KV TTLs IF writes also invalidate the affected keys — Churches-app invalidates per-church keys but not the meta-family keys (country counts went stale up to 1h after a +1,020-church migration today).

What cosmoflare should do (the ask):
1. `cosmoflare audit caching` gains a TTL dimension: for each cached route, estimate data-change cadence (from D1 write analytics: rows written/day to the tables the handler reads) + invalidation coverage (does the codebase hold invalidation calls touching this key namespace?) and verdict the TTL as over/under/appropriate.
2. Emit recommendations: KV TTL vs Cache-Control max-age split (the asymmetry rule above), when to move to stale-while-invalidate, when a materialization cron beats a TTL.
3. Flag missing invalidation coverage: keys whose underlying tables received writes in the audit window but whose namespace has no invalidation call in source.
4. Explain in plain language per finding (the owner-facing value: cosmoflare becomes the CF caching knowledge layer so project owners do not need to be).

Reference implementation for calibration: Churches-app workers/src/middleware/cache.ts + jobs/stats-materialize.ts + docs/optimization/2026-10-08-cf-efficiency-audit.md (commits 155d1d3e, 73c700ac).

Priority: high — direct owner request; pairs with the patterns + auto-remediation feedback (FB-p4FVJYM) in this inbox.

