---
id: FB-181
title: GLM agent dispatch fails when glmqueue daemon is not running
type: idea
status: pending
priority: high
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-03-10T04:44:15.944177+01:00"
updated: "2026-03-10T04:44:15.944177+01:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
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

# FB-181: GLM agent dispatch fails when glmqueue daemon is not running

## What happened
Running `ccs glm-agent exec` from an R2Go2 session failed with:
`Error: queue acquire: connect to daemon: dial unix ~/.glmqueue/glmqueue.sock: no such file or directory`

The GLM agent successfully created the worktree and tmux session, but the claude CLI inside crashed immediately because the glmqueue daemon wasn't running. The fast-fail detection (3s grace period) caught it and reported "agent crashed on startup."

## Why it matters
This is a silent failure that wastes time debugging. Users must manually run `ccs glmqueue daemon &` before using GLM agents, which is undocumented and breaks the expected workflow. The SessionStart hook should handle this.

## Proposed solution
1. **Auto-start glmqueue daemon**: The SessionStart hook or `ccs glm-agent exec` itself should check if the daemon socket exists and start it if missing (backgrounded)
2. **Better error message**: When the agent crashes due to missing glmqueue, surface the actual error ("glmqueue daemon not running") instead of generic "agent crashed on startup"
3. **Health check**: Add glmqueue daemon status to `ccs diagnose` output

## Code location
- Crash detection: `tools/ccsession/internal/glmagent/agent.go:322-333`
- Queue client connect: `tools/ccsession/internal/glmqueue/client.go`
- The `detectFailReason` function should parse the output.log for "glmqueue" errors

## Priority justification
High — completely blocks GLM agent workflow, which is a core CCS feature. Required manual debugging to find root cause.

