---
ulid: 01M1Y4HVS2W0H1VT071FES37JK
title: ccs merge post-merge test timeout too short for cosmoflare pkg/cosmoflare — false FAIL on every merge
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-07T18:32:29.730144+04:00"
updated: "2026-09-07T18:32:29.730144+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 43
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

# FB-pES37JK: ccs merge post-merge test timeout too short for cosmoflare pkg/cosmoflare — false FAIL on every merge

What happened: merging _glm-agent-0060-task-sync-correctness into cosmoflare master, the merge gate itself passed (30/30 packages) but the post-merge test pass FAILED pkg/cosmoflare at 46.066s with a timeout panic. The same package passes solo in 74-85s (verified three times this session: worktree solo 84.3s, master post-merge solo 74.9s, earlier worktree run panicked at a 60s alarm). The post-merge runner's budget (~45s) is below the suite's solo runtime, so the final report shows FAIL on a healthy merge and the tool prints 're-run solo' as a hint. Why it matters: every cosmoflare merge will report a false test failure, training operators to ignore the post-merge FAIL line — exactly the alarm-fatigue S334 exists to prevent. Proposed fix: raise the post-merge go test budget to 3-5 minutes, or run only the packages touched by the merged branch post-merge. Repro: any ccs merge of a branch touching pkg/cosmoflare on cosmoflare (merge c544409 this session; output shows FAIL github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare 46.066s then a clean solo re-run). Secondary, same merge: 'Could not stage archive artifacts: exit status 1' — archive dir GOrchestra/sessions/_glm-agent-0060.../ was created but staging failed; print the underlying git error instead of a bare exit status.

