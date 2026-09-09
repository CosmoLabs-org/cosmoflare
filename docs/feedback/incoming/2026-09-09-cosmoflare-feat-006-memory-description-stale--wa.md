---
ulid: 01M21H38E9H654ECGXGHDHG006
id: FB-3
title: FEAT-006 memory description stale — Waves B-D shipped, ROAD-087 hydrated from stale line
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: cosmoflare
to_target: self
created: "2026-09-09T02:09:25.961666+04:00"
updated: "2026-09-09T02:09:25.961666+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 45
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

# FB-pDHG006: FEAT-006 memory description stale — Waves B-D shipped, ROAD-087 hydrated from stale line

What happened: the auto-memory file ~/.claude/projects/-Users-gabstudio-PROJECTS-cosmoflare/memory/cosmoflare-feat006-domain-center.md carries a stale frontmatter description ('Wave A (services) landed, Waves B-D remaining') while its body (updated 2026-06-20) correctly records all waves A-D merged and FEAT-006 CLOSED (c86a436). The 2026-08-31 audit hydration read the stale description and created ROAD-087 ('Domain Center completion — Waves B-D') for already-shipped work; the 2026-09-09 session caught it before re-planning. A cross-project guard blocks editing the memory file from the cosmoflare session, hence this item.

Why it matters: one stale line spawned a phantom roadmap item that nearly consumed a brainplan session. Every future recall of that memory re-propagates the error.

Proposed fix — in that file replace the description line with:
description: "FEAT-006 Domain Management Center — ALL waves (A-D) shipped and closed 2026-06-20; ROAD-087 was hydrated from a stale version of this note. Residual gaps: legacy pagerules merge, redirect-loop attention check."
And append a 'Residual gaps' section to the body: (1) legacy pagerules forwarding_url merge into domains redirects never shipped (design Layer-1 requirement, plan open risk); (2) redirect-target >=400/loop attention criterion (design Phase-2 best-effort) never implemented. Both verified absent in code on 2026-09-09 (grep over redirect.go, domains.go, cmd/domains_redirects.go).

Evidence: docs/sessions/Session-017-domain-management-center-and-desktop-brainplan.md (all four waves completed, FEAT-006 closed); code listing pkg/cosmoflare/{redirect,registrar}.go, cmd/domains_{get,stats,ns,redirects,tui}.go, internal/tui/domain.go; ROAD-087 created 2026-08-31 audit-hydrated.

