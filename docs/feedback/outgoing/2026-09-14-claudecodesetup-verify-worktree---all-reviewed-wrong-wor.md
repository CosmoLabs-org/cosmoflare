---
ulid: 01M2GG5NCC6C0RRT5X9HA69AA0
title: verify-worktree --all reviewed wrong worktree set (TauriApp) during glm-agent batch
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-14T21:41:52.652214+04:00"
updated: "2026-09-14T21:41:52.652214+04:00"
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

# FB-pA69AA0: verify-worktree --all reviewed wrong worktree set (TauriApp) during glm-agent batch

What happened: during a 55-agent species-2 batch in cosmoflare (2026-09-14), 'ccs verify-worktree --all --approve --score 9 --issues 0' reviewed exactly one worktree — a stale TauriApp worktree (+0/-0, no changes) — instead of the five pending _glm-agent-* worktrees; each had to be verified individually by exact name. Why it matters: --all is the documented batch path (rules/quality-gate.md) but silently no-ops the review for agent worktrees, inviting merges of unreviewed agent work. Fix direction: make --all enumerate every worktree with unmerged commits (or add an --agents scope); skip cleanly when the reviewed set is empty rather than blessing an unrelated stale worktree. Evidence: session transcript 2026-09-14 cosmoflare; 'Reviewed 1 worktrees' output naming TauriApp while 5 _glm-agent-* worktrees awaited review.

