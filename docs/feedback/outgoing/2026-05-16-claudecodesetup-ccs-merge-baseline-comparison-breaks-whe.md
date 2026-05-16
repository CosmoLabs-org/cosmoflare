---
id: FB-967
title: ccs merge baseline comparison breaks when master has cross-dependency build errors from uncommitted peer worktrees
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-16T07:30:58.858653-03:00"
updated: "2026-05-16T07:30:58.858653-03:00"
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

# FB-967: ccs merge baseline comparison breaks when master has cross-dependency build errors from uncommitted peer worktrees

## Problem Description

When two worktrees create interdependent files (A depends on B), and A's file is committed to master before B's worktree is merged, master's build breaks. ccs merge then classifies B's pre-existing errors as "worktree-introduced" because the baseline changed.

## Current vs Expected

Current: Commit domains.go (depends on DoctorService) to master. Try to merge doctor worktree (provides DoctorService). Baseline comparison runs against broken master, sees migration errors as "new=6, inherited=0" even though they are pre-existing. Merge refused even with --acknowledge-baseline.

Expected: Baseline comparison should diff error SIGNATURES (file:message), not just counts. Errors in internal/migration/s3.go are clearly pre-existing regardless of what else changed on master.

Workaround used: git rm domains.go from master, merge doctor, re-add domains.go. This is fragile and error-prone.

## Why It Matters

Any wave-based parallel development where files in the same package depend on each other hits this. Common pattern: library + CLI developed in separate worktrees, library committed first, CLI merge fails baseline.

## Priority Justification

Medium — workaround exists but wastes 5-10 minutes per occurrence and risks losing work.

## Reproduction Steps

1. Create worktree A that adds pkg/r2go2/domains.go (depends on DoctorService)
2. Create worktree B that adds pkg/r2go2/doctor.go (provides DoctorService)
3. Merge worktree B's file to master first (or commit domains.go directly)
4. Try `ccs merge worktreeA --acknowledge-baseline` — refuses with "worktree introduces N new errors"

## Affected Files

tools/ccsession/cmd/merge.go — baseline comparison logic
tools/ccsession/internal/build/ — error signature diffing

## Suggested Implementation

The baseline comparison should match on (file, error_message) tuples, not raw counts. If error file+message exists in BOTH master and worktree builds, it's inherited regardless of how the error count shifted. The current approach breaks when master's error set changes between the worktree branch point and merge time.

