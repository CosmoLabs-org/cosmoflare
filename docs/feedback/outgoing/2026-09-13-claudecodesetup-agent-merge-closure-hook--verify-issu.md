---
ulid: 01M2DDZK8W2QAER90N4DGW5KXB
title: Agent-merge closure hook — verify issue closure when _glm-agent-* branches merge
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-13T17:05:53.436208+04:00"
updated: "2026-09-13T17:05:53.436208+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 53
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

# FB-pGW5KXB: Agent-merge closure hook — verify issue closure when _glm-agent-* branches merge

Agent-merge closure hook: when a _glm-agent-* or _claude-* worktree branch carrying an issue ID merges, ccs merge should verify the referenced issue's scope and prompt for closure (or auto-annotate the issue with the merge SHA). Evidence from cosmoflare 2026-09-13 audit (agent-15-work-completion): FEAT-025 merged via 562a602 (+513 lines incl. tests) and FEAT-019 via 4f512ef (+1,399 lines) both sat OPEN after their agent merges landed — the close-the-loop failure class. A branch-name issue-ID -> closure-prompt hook in ccs merge kills this class. Full analysis: docs/audit/2026-09-13-cosmoflare/agent-15-work-completion.md

