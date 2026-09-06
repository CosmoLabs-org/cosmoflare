---
ulid: 01M1WEXMX01PQR7AGCMXVRR9AH
title: 'glm-agent status --json: expose worktree path and branch fields'
type: feature
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-07T02:55:12.80005+04:00"
updated: "2026-09-07T02:55:12.80005+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 40
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

# FB-pVRR9AH: glm-agent status --json: expose worktree path and branch fields

Problem: 'ccs glm-agent status --json' returns id/name/status/summary/exit_code/elapsed but NOT the worktree directory or branch name. After a 3-agent wave finished in cosmoflare (2026-09-07), finding where to run the S334 re-verification tests took three extra lookups: field-guessing jq, then 'git branch --list _glm-agent*', then 'ls ../cosmoflare-worktrees/'.

Current vs expected: expected the status JSON to carry the two fields the quality gate needs — branch (e.g. _glm-agent-0056-task-daemon-error) and worktree path (../cosmoflare-worktrees/_glm-agent-0056-task-daemon-error). Got: neither; the agent worktree dir is deterministic but had to be re-derived by hand each time.

Why it matters: rules/quality-gate.md S334 mandates re-running tests in the worktree before merge; glm-agent status is the single natural source for that lookup and it omits exactly those fields. Every orchestrated agent wave pays this friction.

Suggested implementation: add Branch string + WorktreePath string to the agent status struct in the glm-agent status command, populated from the session metadata that already knows the spawn dir.

Session context: cosmoflare session 2026-09-07 dispatched three sonnet agents (guardrails, error contract, desktop CSS); the completion monitor fired and the orchestrator needed git-level archaeology to locate worktrees for verification.

