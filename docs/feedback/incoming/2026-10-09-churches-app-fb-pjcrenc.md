---
ulid: 01M4ENH22QRRJQHHZ4AYJCRENC
title: 'Detection matrix: the D1/Cloudflare problem classes systematic vetting catches (evidence-backed from Churches-app 2026-10-08)'
type: feature
status: pending
priority: high
complexity: ""
from_project: Churches-app
from_path: /Users/gabstudio/PROJECTS/Churches-app
to_project: cosmoflare
to_target: project
created: "2026-10-09T01:08:23.767206+04:00"
updated: "2026-10-09T01:08:23.767206+04:00"
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

# FB-pJCRENC: Detection matrix: the D1/Cloudflare problem classes systematic vetting catches (evidence-backed from Churches-app 2026-10-08)

What happened: following the charter + audit-caching specs already in this inbox, here is the evidence-backed problem catalog those checks would have caught. Every item below is a REAL incident from Churches-app on 2026-08..10, most discovered only by manual audit or by hitting production errors. The question this feedback answers: what classes of problems does proper vetting/auditing/optimization in cosmoflare fix for all projects?

D1 COST/CORRECTNESS CLASSES:
1. Rows-read blowups — /api/meta/stats scanned ~1.08M D1 rows per cold call (18 subqueries over 60k churches + 300k mass_schedules + 40k practical_info; D1 bills rows read). No traffic = no pain, so it survived months; at 100 calls/day it projects to ~$3,200/month. Detection: static handler analysis (uncached aggregates scanning whole tables) + rows-read analytics trend per database.
2. Schema drift local-vs-prod — prod D1 was missing masstimes_review, poi_figures, poi_approach_guidance tables and feast_days.church_id column; local migrations never applied remotely. Also 1,020 churches existed in prod but not locally (reverse drift, a lost wave). Detection: table/column parity diff + per-table count diff between a local snapshot and remote D1 (both are one API call each).
3. FK semantics divergence — local SQLite runs FKs OFF, prod D1 enforces them: a migration that ran clean locally failed remotely (UPDATE children before parent rename). Detection: migration linter that flags parent/child write ordering + rejects BEGIN TRANSACTION blocks (D1 rejects explicit transactions via remote execute — hit twice in one day).
4. Unindexed hot paths — LIKE '%...%' and GROUP BY over full tables on every request. Detection: EXPLAIN QUERY PLAN on captured handler queries; flag SCAN without INDEX.

CACHING/EFFICIENCY CLASSES (the audit-caching spec covers these; evidence: 8+ uncached public GETs, zero Cache-Control anywhere, KV CACHE binding nearly idle):
5. Uncached public GETs with table-scan handlers (class 1 + 5 combined).
6. Cache-key collisions — path-only cache keys on filtered routes (cities?country_code=IT vs PL would share one entry). Detection: keyFn lint — keys must include every query param the handler reads.
7. The INVERSE failure — over-caching: auth/user-scoped responses must NEVER carry Cache-Control or KV entries; a cross-user cache leak is a security incident, not a cost one. Detection: flag any cache middleware on routes with auth middleware or per-user output.
8. Worker invocation waste + Pages-vs-Workers fit — static-prerenderable content served through Workers; every request billed when the CDN could hold it.

SECURITY/VETTING CLASSES:
9. Rate-limit coverage — expensive uncached endpoints without rateLimit are DoS-amplifiers (stats had none). Detection: route table cross-check (public GET + expensive handler + no rate limit).
10. Token/secret scope review + account identifiers in configs.

OPERATIONAL CLASSES:
11. Cron coverage — expensive aggregates without materialization jobs.
12. FTS/trigger drift — FTS doc counts diverging from table counts (verified manually: 60,305/60,305 both sides after reconcile).

Why it matters: every one of these was found by hand AFTER the code shipped or during a live incident. All twelve are mechanically detectable from (a) wrangler configs, (b) route source, (c) D1 information_schema/query plans, (d) CF GraphQL analytics. That is the difference between cosmoflare as a convenience CLI and cosmoflare as the governance layer the charter designates.

Proposed solution: structure the audit engine as this detection matrix — each class = one check with severity (cost vs correctness vs security), a detection method, and a fix pointer to the standard (Churches-app commit 155d1d3e + docs/optimization/2026-10-08-cf-efficiency-audit.md are the reference). Classes 2 and 3 (drift + FK linter) are NEW capabilities beyond the audit-caching spec and have the freshest incident evidence — recommend prioritizing them with class 1.

Priority: high — complements the charter and audit-caching specs; this is the input catalog for both.

