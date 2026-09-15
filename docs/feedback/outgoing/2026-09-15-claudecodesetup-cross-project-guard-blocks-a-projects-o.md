---
ulid: 01M2H763RP23M0TJ9VF6MM52XZ
title: cross-project-guard blocks a project's OWN memory dir (~/.claude/projects/<proj>/memory/)
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-15T04:24:04.630275+04:00"
updated: "2026-09-15T04:24:04.630275+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 55
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

# FB-pMM52XZ: cross-project-guard blocks a project's OWN memory dir (~/.claude/projects/<proj>/memory/)

What happened: in a cosmoflare session (2026-09-14/15), Write to /Users/gabstudio/.claude/projects/-Users-gabstudio-PROJECTS-cosmoflare/memory/cosmoflare-species2-sweep-execution.md was blocked by the cosmohooks cross-project-guard with the message 'this file belongs to memory... Edits from another project bypass the target's hooks'. That directory is the CURRENT project's own auto-memory (path encodes -Users-gabstudio-PROJECTS-cosmoflare), not another project's checkout. The same guard also blocked an earlier feedback-driven memory write (docs/feedback/incoming FB ulid 01M2EKRE9PSHYTQ8FCXR26FGWZ in cosmoflare requests exactly this file creation and could not be executed for the same reason). Why it matters: the memory system's contract ('write to it directly with the Write tool') is unenforceable in any project session — memory writes accumulate as forever-pending feedback items instead. Fix direction: allowlist each project's own memory dir in the guard (match the cwd-derived project slug under ~/.claude/projects/), or provide a ccs memory write path. Evidence: session transcript cosmoflare 2026-09-14; blocked Write + the pending FB item.

