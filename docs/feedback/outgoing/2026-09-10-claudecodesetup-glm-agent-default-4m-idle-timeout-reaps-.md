---
ulid: 01M25WCA3JDY6A4WAD3P8G9F7X
title: glm-agent default 4m idle-timeout reaps finished agents — salvage saves work but wastes cycles
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T18:43:34.64239+04:00"
updated: "2026-09-10T18:43:34.64239+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 49
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

# FB-p8G9F7X: glm-agent default 4m idle-timeout reaps finished agents — salvage saves work but wastes cycles

PROBLEM: The default --idle-timeout 4m killed 8+ agents this session AFTER they finished their file work but before they wrote their final summary. The timeout salvager auto-committed (recovering 6 of 8), but 4 agents (0091, 0094/0107, 0096, 0099/0110) were reaped with ZERO work, two of them twice.

CURRENT VS EXPECTED: ccs glm-agent status shows 'timeout (5m3s) idle timeout: no file activity detected' while commits: 1 contains the complete task. Expected: an agent that has committed its full task output is 'done', not idle-timed-out. WORKAROUND: dispatch with --idle-timeout 6m (100% success rate after adoption) plus post-hoc salvage inspection of every timed-out worktree.

WHY IT MATTERS: Every reap costs a gate cycle (inspect salvage commit, amend message, verify). The zero-work deaths burned 4 dispatch cycles on glm-5.3 API stalls the reaper cannot distinguish from agent completion.

PRIORITY: Medium-high — the API instability wave made it constant today; on stable days it bites any agent whose summary write takes >4m.

REPRO: ccs glm-agent exec --prompt <p> --files <f> (defaults) on any task whose verify step (go test of a slow package) runs >4m without file writes; observe reaper kill at ~5m elapsed.

AFFECTED FILES: Unverified — the idle monitor in the glm-agent runner (tools/ccsession internal/agent/ or cmd/glm_agent.go; needs find-func 'idle').

SUGGESTED IMPLEMENTATION: Treat a fresh commit whose files match the task's --files set as a liveness signal (reset or disable the idle timer), or raise the default to 8-10m — today's evidence shows 6m was sufficient for every successful run.

SESSION CONTEXT: 2026-09-10 cosmoflare — 15+ GLM dispatches (FEAT-012/013 waves, /cover batch + retry, session-end agents); the reaper hit agents in both tmux waves and session-end doc agents.

