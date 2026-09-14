---
ulid: 01M2GG5NWAB4Q2GPSQ22GB7BY1
title: merge auto-rebase invalidates recorded review SHA; consider content-keyed reviews
type: idea
status: pending
priority: low
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-14T21:41:53.162608+04:00"
updated: "2026-09-14T21:41:53.162608+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 55
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

# FB-pGB7BY1: merge auto-rebase invalidates recorded review SHA; consider content-keyed reviews

What happened: merging _glm-agent-0133-email (cosmoflare 2026-09-14), the first ccs merge auto-rebased the branch (tip moved after a hand-finish commit), then the gate refused with 'review stale — re-run verify-worktree'; re-verifying with the identical verdict unblocked it. Why it matters: in serialized merge batches this forces a verify/merge stutter whenever a branch gained a commit after review, though the reviewed diff content was unchanged by the rebase. Fix direction: key the recorded review to the branch diff/content hash rather than the tip SHA so a pure rebase keeps the verdict valid. Evidence: merge log 'Review: stale (was 63b58c3, HEAD is 244d52e)' in session transcript 2026-09-14 cosmoflare.

