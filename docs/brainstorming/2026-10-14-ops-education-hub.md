---
title: 'Ops education hub — Cloudflare learning section + rule-quota usage display'
source: Operator request (2026-10-14, session 78)
related_roadmap:
    - ROAD-p9X66J9
status: proposed
created: "2026-10-14T19:40:00+04:00"
deliverables:
    - id: BR-01
      title: "Rule-quota collector — per-zone used/free per rule family from live API (degrade-first: Page Rules now, rulesets families when token permission lands)"
    - id: BR-02
      title: "Zone drill-in Rules section — used/free pips + rule names, red glow at cap"
    - id: BR-03
      title: "Overview account strip — fleet rule-slot usage at a glance"
    - id: BR-04
      title: "Education hub — curated concept cards (~15-25) with live-data links from findings"
    - id: BR-05
      title: "Verified catalog rows for the five rule-family allowances (source_url + verified_on)"
---

# Ops education hub

## Context

The Ops pager reached its UI milestone (liquid-glass nav, entity profiles, Guides
section). The operator's next direction (2026-10-14):

> "these rules and our allowance for them — display that in the ops. Where could
> we make it clear and show how much we're using of our free allowance and also
> mention which rule we're using?"
> "the ops should be educational and have a whole section explaining different
> things and different features from Cloudflare. That should be our next
> feature to build."

Triggering context: the www→non-www redirect standard work surfaced that rule
quotas (Page Rules 3/zone, Redirect Rules 10/zone, WAF custom rules 5/zone) are
invisible until you hit them, and Cloudflare concepts (edge cache, rulesets,
redirect vs page rules) are knowledge the operator should not need to hold.

## Two linked capabilities

### A. Rule-quota usage display (the "real numbers" half)

Per zone, from the live API: which rules exist (name + one-line purpose) and
how many of the plan's free slots each rule family consumes.

| Rule family | Free allowance/zone | API source |
|---|---|---|
| Page Rules | 3 | GET /zones/{id}/pagerules |
| Redirect Rules (dynamic + bulk) | 10 | Rulesets API — needs the Zone Rulesets token permission |
| WAF Custom Rules | 5 | Rulesets API (same permission) |
| Cache Rules | 10 | Rulesets API (same permission) |
| Configuration Rules | 10 | Rulesets API (same permission) |

House rules: real API numbers only (never modeled); the allowances come from
the verified limits catalog (`pkg/cosmoflare/limitsdata`) with source_url +
verified_on, same discipline as FEAT-048.

Candidate placements:
1. **Zone drill-in (zone profile) — a "Rules" section**: list per family with
   `used / free` pips and rule names; a family at its cap shows the ops
   red-flowing-glow treatment (house style: severity-gated glows).
2. **Overview account strip**: one line "rule slots: 41 page rules / 45 zones,
   2 zones at cap" — the fleet-at-a-glance.
3. **Worker profile / other profiles**: only where rule families actually
   attach (zones are the home; profiles stay clean).

### B. The education section (the "explain" half)

Builds on the existing GuidesView. Candidate shape: a Cloudflare Concepts
library — short, plain-language cards ("What is an edge cache hit?",
"Page Rule vs Redirect Rule", "Why s-maxage on HTML does nothing") each linked
FROM live data (e.g. the low-cache-hit finding links to the edge-cache card;
a rule at cap links to the rule-family card). Education anchored to the
account's real state, not a generic wiki.

## Design decisions (operator, 2026-10-14)

1. **Education scope: curated cards + live links.** ~15-25 authored concept
   cards; findings/rules link to them (low-cache-hit → edge-cache card; a rule
   at cap → the rule-family card). Education anchored to the account's real
   state.
2. **Quota display: zone drill-in + Overview strip.** Rules section in the
   zone profile (used/free pips per family, rule names, red glow at cap) plus
   one fleet line on Overview.
3. **Rulesets permission: flip it now.** The operator adds Zone Rulesets Edit
   to the token; the collector still degrades gracefully (Page Rules only)
   if a probe finds the permission missing — degrade-first code, full data
   once the token edit lands.

## Constraints (house)

- Real numbers only; allowances from the verified catalog
- Ops UI taste: no left borders, no tile outlines, impeccable detect = 0,
  "CosmoLabs Ops · Cosmoflare" masthead
- Paid Workers plan: every new fetch path carries an enforced budget
- Rulesets data needs the Zone Rulesets token permission (pending operator
  action) — the feature degrades to Page-Rules-only until then
