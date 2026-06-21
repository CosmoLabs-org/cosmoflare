---
id: FB-1228
title: auto-mode classifier blocks ccs kill --force even after explicit user authorization
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-06-20T21:05:09.512952-03:00"
updated: "2026-06-20T21:05:09.512952-03:00"
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

# FB-1228: auto-mode classifier blocks ccs kill --force even after explicit user authorization

What happened: user explicitly answered 'Yes, remove them' to a worktree-removal question, but the auto-mode classifier denied 'ccs kill --force', citing the global rule 'NEVER reach for --force to get past a kill safety check'. It later allowed --force after a stronger restated authorization. Why it matters: the BUG-538 rule's intent is to prevent destroying UNSAVED work / others' live worktrees — but it also blocks legitimate, user-authorized removal of merged worktrees the session created (the only reason --force was needed was a non-agent-prefix name). Proposed: let explicit, in-context user authorization of a SPECIFIC named worktree satisfy the gate (the rule already carves out 'without explicit user say-so'); the classifier should weight a direct user 'yes, remove <name>' as that say-so. Priority: medium.

