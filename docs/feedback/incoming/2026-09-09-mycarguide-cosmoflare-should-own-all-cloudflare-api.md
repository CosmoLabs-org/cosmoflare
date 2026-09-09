---
ulid: 01M23094G3N4HZB8HH294RVZ4Y
id: FB-6
title: CosmoFlare should own all Cloudflare API operations — endpoint registry, plan caps, and error decoding so failures are diagnosed before/at call time
type: feature
status: pending
priority: high
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-09-09T15:54:01.602952+04:00"
updated: "2026-09-09T15:54:01.602952+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 17
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

# FB-p4RVZ4Y: CosmoFlare should own all Cloudflare API operations — endpoint registry, plan caps, and error decoding so failures are diagnosed before/at call time

Summary: Applying one rate-limit rule to mycar.guide (2026-09-09) took six failed API calls across three distinct failure classes, none of which the Cloudflare API explains at the point of failure: (1) POST to /zones/{id}/rulesets/phases/http_ratelimit/entrypoint/rules — an endpoint that does not exist — returns 10405 'Method not allowed for this authentication scheme', which reads as an auth-scope problem and sent us editing token permissions for 20 minutes when the real problem was the route; (2) the same phase has no entrypoint ruleset on a fresh Free zone, and GET returns an empty result that is indistinguishable from 'exists with 0 rules' until a follow-up call fails 1000 not_found; (3) payload validation is server-side and sequential — 20155 arrives only after auth+route succeed, revealing that characteristics MUST include cf.colo.id ('ratelimiting counting is processed at colocation level only'), a requirement no client could know in advance; (4) Free-plan parameter caps (1 rule/zone, 10s window, 10s mitigation, IP-only counting) are documented in a marketing availability table, not enforced client-side anywhere.

Motivation: Every CosmoLabs project touching Cloudflare (MyCarGuide today, every future Pages/Workers deploy) hits this same maze. Today's session burned ~1 hour of senior-operator time on one rule. A CosmoFlare client layer would have made it one call.

Proposed solution: CosmoFlare ships a Cloudflare API client (TS module + CLI) that encodes the tribal knowledge: (a) endpoint registry — only documented routes exist, client rejects unknown method+route combos with 'this endpoint does not exist' instead of letting CF return 10405; (b) plan-cap validator — knows Free/Pro/Business/Enterprise parameter limits per product (ratelimit: Free=10s/10s/1 rule/IP-only) and validates payloads before sending; (c) mandatory-field preflight — cf.colo.id in characteristics, similar invariants elsewhere; (d) error-decode table — 10405='route not valid for token auth (usually: endpoint does not exist)', 1000 not_found on phase entrypoint='create it with PUT first', 20155='missing cf.colo.id', 10000='missing token scope for this endpoint'; (e) permission catalog from GET /user/tokens/permission_groups (machine-readable SSOT; names are cosmetic and drift) plus the documented-correct Zone WAF Edit finding from the companion feedback sent today. Seed data: this session's transcript of six calls and four error classes against zone e586a1c56dc1020690f9eeea7e46a3bd.

Repro/evidence: MyCarGuide zone e586a1c56dc1020690f9eeea7e46a3bd, 2026-09-09: POST entrypoint/rules -> 10405 (endpoint does not exist); GET entrypoint -> empty; POST rulesets/{id}/rules -> 1000 not_found (no entrypoint); PUT entrypoint without cf.colo.id -> 20155; PUT with ['cf.colo.id','ip.src'] -> success. Companion feedback: CF API token permission names drifted (sent earlier today).

