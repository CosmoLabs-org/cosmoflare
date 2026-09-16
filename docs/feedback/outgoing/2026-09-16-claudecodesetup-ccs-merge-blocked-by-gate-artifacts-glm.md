---
ulid: 01M2KK9ZASYB03R0NVYS2Q4EMM
title: ccs merge blocked by gate artifacts; glm idle-timeout kills test runs.
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-16T02:34:22.937555+04:00"
updated: "2026-09-16T02:34:22.937555+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 56
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

# FB-p2Q4EMM: ccs merge blocked by gate artifacts; glm idle-timeout kills test runs.

(1) Three merges refused with 'MERGE REFUSED — 1 blocking file(s)' because .verify and .review.json sit uncommitted in the agent worktree; committing them in-worktree also invalidated one review state, forcing a re-verify. Proposal: verify-worktree/merge should auto-stage-or-ignore its own gate artifacts the way BUG-597 strips agent-injected files. (2) The default 2m idle-timeout killed 7 GLM agents mid go-test (package suites run 2-5m silent); every one had complete work under auto-salvage, so the kills were pure overhead. Proposal: raise the idle-timeout default when the task's verify step contains 'go test', or emit keepalives during long test runs. Both cost ~20 minutes of orchestration per wave; workarounds exist, so medium priority.

