---
ulid: 01M4E6CMG3SP9FYEXFX6PN1FX5
title: 'CHARTER: Cosmoflare is the CosmoLabs meta project for Cloudflare workflows — auditing, security, resource efficiency for ALL projects'
type: feature
status: duplicate
priority: high
complexity: ""
from_project: Churches-app
from_path: /Users/gabstudio/PROJECTS/Churches-app
to_project: cosmoflare
to_target: project
created: "2026-10-08T20:43:50.147246+04:00"
updated: "2026-10-08T20:43:50.147246+04:00"
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
duplicate_of: /Users/gabstudio/PROJECTS/cosmoflare/docs/feedback/incoming/2026-10-08-churches-app-charter-cosmoflare-is-the-cosmolabs-met.md
---

# FB-pPN1FX5: CHARTER: Cosmoflare is the CosmoLabs meta project for Cloudflare workflows — auditing, security, resource efficiency for ALL projects

What happened: Owner directive 2026-10-08 (Churches-app session, CF efficiency audit): Cosmoflare is designated the meta project for Cloudflare workflows across all CosmoLabs projects. It becomes the layer that (1) audits CF infrastructure for efficiency and security, (2) ensures resources are not wasted unnecessarily, and (3) owns the CF best-practice standards every project follows.

Why it matters: Every CosmoLabs CF-backed project (Churches-app, Noble.Coffee, BaristaBase, AXO, BeeKey, and future ones) faces the same risks: uncached endpoints, D1 rows-read cost blowups, missing edge headers, unused bindings, no pre-launch efficiency gate. Scattered per-project habits do not scale; the audit spec already in this inbox (CF efficiency audit dimension, delivered 2026-10-08) is the first concrete standard under this charter.

Proposed solution: treat this as a charter/roadmap anchor, not a single feature:
1. Roadmap pillar: CF governance for CosmoLabs — audit caching (spec already delivered), plus future dimensions: security audit (WAF/firewall posture, token scope), cost audit (rows-read/write trends, KV op waste, Pages vs Workers fit), compliance sweep (CORS, cache-control on user-scoped routes must NOT be cached)
2. Every new CF best-practice standard lands as a cosmoflare audit rule first — Churches-app commit 155d1d3e is the reference implementation of rule set 1; CosmoKit cf-cache module and ClaudeCodeSetup SOP specs were delivered in parallel
3. Success metric: any project can run `cosmoflare audit` before a launch and get a pass/fail efficiency + security verdict

