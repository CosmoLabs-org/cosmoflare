---
ulid: 01M1AF1A55Q5J95M6N8JYQGBP1
id: IDEA-047
title: Add plan_ref back-links to 7 orphaned brainstorms
created: "2026-08-31T03:10:53.093494+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-14-doc-integrity
    file: docs/audit/2026-08-31-cosmoflare/agent-14-doc-integrity.md
tags:
    - audit
    - doc-integrity
---

# Add plan_ref back-links to 7 orphaned brainstorms

7 plan->brainstorm chains are one-way: plans carry brainstorm_ref but the brainstorms lack plan_ref back-links, consuming the full 25-point broken-chain penalty in the doc-integrity score (47/100). One-line edit each: road020-dashboard-tui, road002-tui-object-browser, road007-s3-migration-resume, domain-management-center, cosmoflare-desktop, inprocess-event-bus, live-metrics-producer (all in docs/brainstorming/). Cheap 25-point score lift.
