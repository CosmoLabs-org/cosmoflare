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

The Ops app today has zero WAF visibility (no collector, no view, no pricing rows) while the CLI already manages WAF custom rules, rule lists and rate-limiting rules (cmd/waf.go, waf_lists.go, waf_ratelimit.go). WAF rules are quota-limited by plan (rule COUNTS), not usage-billed, so the right UX is a WafView pager section: inventory of custom rules + rate-limiting rules + rule lists across zones, each row with enabled/action/expression, plus quota gauges in the existing gauge style (rules used vs plan allowance) that run the red flowing glow when at or near the limit — same real-numbers-only rule as everywhere (counts from the API, never modeled). Alert rules get an over-quota condition. Research agents 0398/0399/0400 (2026-10-11) are verifying per-plan limits, API endpoints and the current CLI surface.
