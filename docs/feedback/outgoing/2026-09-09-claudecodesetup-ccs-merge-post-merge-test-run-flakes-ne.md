---
ulid: 01M22YD238RW7ZVVGYQYHHHGE2
title: ccs merge post-merge test run flakes (net/http transport), solo re-run always passes
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-09T15:21:13.064329+04:00"
updated: "2026-09-09T15:21:13.064329+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 47
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

# FB-pHHHGE2: ccs merge post-merge test run flakes (net/http transport), solo re-run always passes

What happened: 4/4 agent merges this session (cosmoflare; agents 0079/0080/0081/0082/0083/0084) printed 'FAIL github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare 45s' with net/http transport.go dialConn goroutine dumps from the merge's own post-merge verification, then a 're-run solo' hint. Every solo re-run passed (75-82s) and every merge landed correctly (ancestry verified). Identical signature each time: FAIL at ~45s while the same package takes 75-80s solo and passes — the suite is being interrupted mid-run, not asserting a failure.

Why it matters: the gate reports FAIL on 100% of healthy merges in this repo; orchestrators must re-run solo after every merge (doubling test time per merge), and a naive consumer would treat the merge as failed despite exit 0.

Proposed solution: align the merge-path test invocation's timeout with the repo's real suite duration (or retry once before reporting FAIL) — the ~45s FAIL vs 75-80s solo pass suggests the runner interrupts the binary. Also surface flakes distinctly from real gate failures; exit 0 + FAIL output reads as success and failure at once.

Repro: in cosmoflare, run ccs merge on any agent worktree touching pkg/cosmoflare; observe the FAIL block, then run 'go test ./pkg/cosmoflare/ -count=1' solo — passes.

