---
ulid: 01M4M5XZPKRWTMGNS7Q2KN1A2F
title: 'Ops WAF section: rule inventory with plan-quota tracking and over-limit warnings'
created: "2026-10-11T04:31:16.691323+04:00"
status: seed
source: human
origin:
    session: 75
---

# Ops WAF section: rule inventory with plan-quota tracking and over-limit warnings

# Ops WAF section: rule inventory with plan-quota tracking and over-limit warnings

The Ops app today has zero WAF visibility (no collector, no view, no pricing rows) while the CLI already manages WAF custom rules, rule lists and rate-limiting rules (cmd/waf.go, waf_lists.go, waf_ratelimit.go). WAF rules are quota-limited by plan (rule COUNTS), not usage-billed, so the right UX is a WafView pager section: inventory of custom rules + rate-limiting rules + rule lists across zones, each row with enabled/action/expression, plus quota gauges in the existing gauge style (rules used vs plan allowance) that run the red flowing glow when at or near the limit — same real-numbers-only rule as everywhere (counts from the API, never modeled). Alert rules get an over-quota condition. Research agents 0398/0399/0400 (2026-10-11) are verifying per-plan limits, API endpoints and the current CLI surface.

---

RESEARCH VERIFIED 2026-10-11 (agent 0398, live Cloudflare docs; agent 0400 repo scan): Per-zone custom rules: Free 5 / Pro 20 / Business 100 / Enterprise 1000. Zone-level custom rulesets: 1/2/5/10. Rate-limiting rules: Free 1 / Pro 2 / Business 5 / Enterprise 100 (contract caveat). Custom lists: Free 1 / Pro+Business 10 / Enterprise 1000 (items 10k/10k/10k/500k). API: GET /zones/{id}/rulesets/phases/http_request_firewall_custom/entrypoint and .../http_ratelimit/entrypoint — NO quota endpoint exists; clients count the rules arrays themselves (matches the real-numbers-only rule: counts from API, limits hardcoded with verifiedOn like pricing.ts). Plan per zone already arrives via the domains collector (z.plan.name). CLI surface confirmed complete: cmd/waf.go, waf_lists.go, waf_ratelimit.go, firewall.go (legacy, deprecated upstream — migrated into custom rules). Ops gap confirmed: zero waf/firewall/rulesets usage in apps/ops today.
