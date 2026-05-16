---
id: FB-963
title: ccs merge cascading uncommitted-file blocks require multiple commit-retry cycles
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-16T03:25:56.662223-03:00"
updated: "2026-05-16T03:25:56.662223-03:00"
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

# FB-963: ccs merge cascading uncommitted-file blocks require multiple commit-retry cycles

## Problem
When running ccs merge on a worktree (UnitTesting), it repeatedly blocked on uncommitted files. Each attempt auto-committed some metadata but then discovered MORE uncommitted files (from its own auto-commit creating new session-id/session-start changes). Required 4 separate attempts before the merge could proceed.

## Current vs Expected
Current: ccs merge UnitTesting → 'refusing to merge: 12 uncommitted files' → manual commit → ccs merge UnitTesting → 'refusing to merge: 4 uncommitted files' → manual commit → ccs merge UnitTesting → 'refusing to merge: 2 uncommitted files' → commit → finally proceeds.
Expected: ccs merge should handle its own prerequisite commits in a single pass, or its auto-commit should commit ALL uncommitted files (not just some).

## Why It Matters
Each failed merge attempt costs ~5 tool calls and token context. In this session it consumed ~15 tool calls across 4 attempts. For sessions with multiple worktree merges this compounds.

## Priority Justification
Medium — workaround exists (pre-commit everything manually) but the auto-commit feature creates a false sense of handling it.

## Reproduction Steps
1. Have a session with uncommitted metadata (.claude/session-id, .glm-agent-counter, build/r2go2, etc.)
2. Run: ccs merge <worktree-name>
3. Observe: blocks on uncommitted files
4. Commit those files manually
5. Re-run: blocks on NEW uncommitted files (.claude/session-start changed by the commit)
6. Repeat until clean

## Affected Files
tools/ccsession/cmd/merge.go — the uncommitted-file check and auto-commit logic

## Suggested Implementation
Option A: ccs merge's auto-commit should loop until git status is clean (max 3 iterations) before checking the merge precondition.
Option B: The uncommitted-file check should exclude .claude/ session metadata files that are expected to change during normal operation.
Option C: Stage everything with git add -A before the check (risky — could catch secrets).

