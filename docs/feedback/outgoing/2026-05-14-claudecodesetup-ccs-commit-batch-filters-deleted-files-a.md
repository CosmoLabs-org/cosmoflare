---
id: FB-952
title: ccs commit-batch filters deleted files as irrelevant
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-14T05:29:17.302387-03:00"
updated: "2026-05-14T05:29:17.302387-03:00"
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

# FB-952: ccs commit-batch filters deleted files as irrelevant

## Problem description

ccs commit-batch silently drops deleted files (files being removed from the repo) from the commit plan. When a commit contains only deleted files, the commit is skipped entirely.

## Current vs expected

**What happened:**
Tried to commit removal of 4 dead demo files (cmd_disabled/policy.go, cmd_disabled/restore.go, cmd_disabled/upload.go, cmd_disabled/webhook.go). commit-batch filtered them out as irrelevant. Had to bundle the deletions with an unrelated file (docs/issues/TASK-003.yaml) to force inclusion.

**Expected:** Deleted files should be committed just like modified/added files. File deletion is a deliberate, meaningful change.

## Why it matters

Dead file cleanup is a common session task. commit-batch silently losing the commit means the user thinks files are removed, but they persist. Discovering this mid-session wastes time diagnosing why git status still shows the files.

## Priority justification

Medium — it's a silent data loss that causes confusion, but there's a workaround (bundle with another file).

## Reproduction steps

1. Delete some tracked files: rm file1.go file2.go
2. Run: echo '[{\"files\": [\"file1.go\", \"file2.go\"], \"message\": \"chore: remove dead files\"}]' | ccs commit-batch --json
3. Observe: files are skipped, nothing committed

## Affected files

The filtering logic in ccs commit-analyze or commit-batch — wherever the relevance filter is applied.

## Suggested implementation

Include file deletion (git status 'D') as a recognized change type. Deletion is not irrelevant — it's one of the most intentional operations in version control.

