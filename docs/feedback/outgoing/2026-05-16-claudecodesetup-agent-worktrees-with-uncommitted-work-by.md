---
id: FB-964
title: Agent worktrees with uncommitted work bypass ccs merge quality gate
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-16T03:26:16.43772-03:00"
updated: "2026-05-16T03:26:16.43772-03:00"
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

# FB-964: Agent worktrees with uncommitted work bypass ccs merge quality gate

## Problem
When dispatching parallel Opus agents via Agent tool with isolation: worktree, agents write files but don't always commit. This means ccs merge has no commits to merge, and the only option is manually copying files from the worktree to the main repo — completely bypassing the quality gate (verify-worktree, review score, diff review).

## Current vs Expected
Current: Agent writes pkg/r2go2/dns.go + cmd/dns.go in worktree → no commit → git log master..HEAD shows nothing → ccs merge has nothing to merge → must cp files manually.
Expected: Either (a) agents should always commit their work, or (b) ccs merge should detect uncommitted changes in worktrees and offer to commit+merge them.

## Why It Matters
In this session, 3 of 4 Phase 4 agents and 3 of 4 Phase 5 agents produced correct code but no commits. All files were manually copied, skipping the entire quality gate. The quality gate exists for a reason (S334 retrospective) — this is an architectural bypass.

## Priority Justification
High — this affects every session using parallel Agent-tool worktrees. The workaround (manual copy + build verify) works but defeats the merge gate's purpose.

## Reproduction Steps
1. Dispatch Agent with isolation: worktree for a code generation task
2. Agent writes files, confirms build passes
3. Agent returns — check: git -C <worktree> log --oneline master..HEAD → empty
4. ccs merge <worktree> → nothing to merge
5. Must cp files manually

## Affected Files
The Agent tool's worktree isolation mode — it creates the worktree but doesn't enforce commits before returning.

## Suggested Implementation
Option A: Add a post-agent hook that auto-commits all new/modified files in the worktree before returning to the parent session.
Option B: Add ccs merge --uncommitted flag that stages + commits worktree changes before merging.
Option C: Document that Agent prompts should explicitly instruct 'commit your changes before finishing' — but this is fragile.

