---
ulid: 01M239MY0M7XFJJEY1PMW2BY25
title: Memory description stale — exact edit requested (FB-3 forward from cosmoflare)
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-09T18:37:45.364226+04:00"
updated: "2026-09-09T18:37:45.364226+04:00"
suggested_conversion: bug
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

# FB-pW2BY25: Memory description stale — exact edit requested (FB-3 forward from cosmoflare)

What happened: cosmoflare's auto-memory description line is stale and already caused damage — the 2026-08-31 audit hydration read 'Wave A landed, Waves B-D remaining' and created ROAD-087 for already-shipped work (caught 2026-09-09 before re-planning; ROAD-087 re-scoped and closed same day). The file is a symlink target from ~/.claude/projects/-Users-gabstudio-PROJECTS-cosmoflare/memory -> ClaudeCodeSetup/memory/projects/cosmoflare, so a cross-project guard correctly blocks the edit from cosmoflare sessions; the fix must happen in this repo.

Exact edit requested in memory/projects/cosmoflare/cosmoflare-feat006-domain-center.md — replace the frontmatter description line with:
description: "FEAT-006 Domain Management Center — fully shipped incl. FEAT-010/ROAD-087 residuals (2026-09-09); holds SDK gotchas + CLI seams"

Why: every future recall of that memory re-propagates the error; one stale line already spawned a phantom roadmap item.

Evidence: cosmoflare FB-3 (docs/feedback/incoming/2026-09-09-cosmoflare-feat-006-memory-description-stale--wa.md) — note FB-3's own proposed replacement text is now ALSO stale (it lists residuals as open; those shipped 2026-09-09 via FEAT-010). Use the text above, not FB-3's.

