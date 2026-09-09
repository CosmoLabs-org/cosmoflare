---
ulid: 01M230JHX7ZX7M7WKKF10Y9DSQ
id: FB-7
title: Zone rate-limiting rules do not appear to count cached static-asset requests (Pages custom domain) — CosmoFlare should encode which traffic classes WAF rate limiting actually sees
type: improvement
status: pending
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-09-09T15:59:10.247572+04:00"
updated: "2026-09-09T15:59:10.247572+04:00"
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

# FB-p0Y9DSQ: Zone rate-limiting rules do not appear to count cached static-asset requests (Pages custom domain) — CosmoFlare should encode which traffic classes WAF rate limiting actually sees

Summary: After successfully creating a zone rate-limiting rule via the Rulesets API (Free plan, http_ratelimit phase, expression path eq /catalog.json, 10 req/10s per IP+colo, block 10s, verified enabled=true via API), three separate burst tests — 45+ requests inside 10-second windows, including cache-busted variants with unique query strings — never produced a single 429. The target is a Cloudflare Pages static asset served on the zone's custom domain (mycar.guide/catalog.json).

Motivation: The rule exists but provides no observable protection for its exact intended target (a large static bundle fetched by every /compare visitor). Operators will believe they are protected when they are not. This is a traffic-class visibility question no Cloudflare surface answers at creation time — the API accepts the rule, reports enabled, and silently does not count the traffic.

Proposed solution: CosmoFlare's CF knowledge base (companion to the API-client feedback FB-p4RVZ4Y and the permission-naming feedback sent earlier today) should document and, where possible, detect the traffic classes WAF rate limiting counts vs skips: cached static assets, Pages asset serving on custom domains, cache HIT vs MISS, origin-passing requests. When CosmoFlare applies a rate-limit rule it should run an active trip probe (burst N requests, expect 429) and report 'rule live but traffic class not counted' instead of trusting enabled=true.

Repro/evidence: MyCarGuide zone e586a1c56dc1020690f9eeea7e46a3bd, rule d65876b408bd4a439d0bc64b4a1a8c37 (2026-09-09): GET entrypoint shows enabled=true; bursts of 15 plain + 15 unique-query-string fetches x3 rounds all returned 200, zero 429s.

