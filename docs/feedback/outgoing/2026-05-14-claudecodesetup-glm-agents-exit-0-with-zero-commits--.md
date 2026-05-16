---
id: FB-958
title: GLM agents exit 0 with zero commits — no failure signal
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-14T23:59:45.741427-03:00"
updated: "2026-05-14T23:59:45.741427-03:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 2027
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

# FB-958: GLM agents exit 0 with zero commits — no failure signal

## Problem
Dispatched 4 GLM agents via ccs glm-agent exec for R2Go2 features (presigned URLs, pipe support, bucket compare, analytics). All returned exit code 0 but produced zero commits. One was blocked by validation (score 52/100) with a clear error. The other 3 silently completed with no indication of failure.

## Current vs Expected
Command: ccs glm-agent exec --files ... "task description"
Output:
  Idle timeout: 2m0s
  Max timeout:  30m0s
  Max turns:    25
  Poll with: ccs glm-agent status --json

Expected: If an agent exits with 0 commits, the wrapper should surface this as a failure or at minimum report "0 commits produced" in the output. Currently the output is identical whether the agent succeeds with 3 commits or produces nothing.

## Why It Matters
In parallel dispatch workflows (GOrchestra), the orchestrator relies on exit code to decide whether to merge. Exit 0 + 0 commits = silent failure that gets missed. This happened on 3/4 agents this session, and I only caught it by checking worktree diff counts.

## Priority
High — GOrchestra and parallel dispatch depend on accurate success/failure signals. Without it, the merge gate can't function.

## Reproduction
1. ccs glm-agent exec --files cmd/object.go "Modify runObjectGet to support stdout piping"
2. Agent opens worktree, reads files, but doesn't commit
3. Wrapper exits 0
4. Output shows only config info

## Affected Files
The agent wrapper script that handles exit code and output formatting. Likely in tools/ccsession/ internal GLM agent execution code.

## Suggested Implementation
After agent completion, check git commit count in worktree (like ccs verify-worktree already does). If 0 commits and agent didn't report an error, emit a warning or set exit code to non-zero. Add a summary line like 'Commits: 0 (agent may have failed silently)' to stdout.

