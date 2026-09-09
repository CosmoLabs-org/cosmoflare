---
ulid: 01M22Z7TP60KRR36ZKDBP2TAVY
id: FB-5
title: CF API token permission names drifted — Zone WAF Edit governs rate limiting; legacy Rate Limiting API is off
type: feature
status: pending
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-09-09T15:35:50.2144+04:00"
updated: "2026-09-09T15:35:50.2144+04:00"
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

# FB-pP2TAVY: CF API token permission names drifted — Zone WAF Edit governs rate limiting; legacy Rate Limiting API is off

Summary: On 2026-09-09, creating a zone rate-limiting rule via the Rulesets API (POST /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint) failed with error 10405 'Method not allowed for this authentication scheme', and the legacy endpoint /zones/{zone_id}/rate_limits failed with 10000 'Authentication error' — both using an active Bearer token. Root cause: the token lacked the Zone WAF: Edit permission. Research confirmed modern rate-limiting rules are Ruleset Engine constructs (http_ratelimit phase) governed by the token permission Zone → Zone WAF → Edit; the classic Rate Limiting API is deprecated and turned off. Two guidance passes (including an AI assistant) mis-directed the operator to search for a 'Rate Limiting' permission that no longer governs these operations.

Motivation: Cloudflare's token-permission names have drifted from product names ('Zone WAF' now covers rate limiting; a 'Rate Limiting'-named scope covers a dead API). Any CosmoFlare docs, scope wizards, or automation that recommend which permissions to attach for WAF/rate-limit work will reproduce this confusion. Error 10405 reads as an auth-SCHEME problem but is actually a missing-SCOPE signal — that mapping is undocumented folklore today.

Proposed solution: Maintain a current CF API-token permission catalog inside CosmoFlare: permission name → endpoints/phases it unlocks, sourced from developers.cloudflare.com. Seed entries: (1) Zone → Zone WAF → Edit unlocks rulesets phases http_ratelimit + http_request_firewall_custom writes; (2) classic /zones/{id}/rate_limits is OFF — do not recommend; (3) error-signature table: 10405 on rulesets writes = missing Zone WAF Edit, 10000 on classic endpoints = wrong/missing scope. GAB will run a Qwen deep-research pass to enumerate the exact current names of ALL Cloudflare API token permissions; the results arrive separately and should be structured into this catalog as the canonical data set.

Repro/evidence: MyCarGuide zone e586a1c56dc1020690f9eeea7e46a3bd, API v4, 2026-09-09: POST to rulesets phase entrypoint without Zone WAF Edit → 10405; same call after adding the permission → success (rule applied).

