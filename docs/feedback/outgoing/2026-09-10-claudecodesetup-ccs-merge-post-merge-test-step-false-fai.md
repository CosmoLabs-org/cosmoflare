---
ulid: 01M25WDF1H3E2DGW9A7361D6QT
title: ccs merge post-merge test step false-FAILs on slow packages under concurrent load
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T18:44:12.465025+04:00"
updated: "2026-09-10T18:44:12.465025+04:00"
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

# FB-p61D6QT: ccs merge post-merge test step false-FAILs on slow packages under concurrent load

PROBLEM: The merge command's post-merge full-suite run failed 3x this session on pkg/cosmoflare (a ~80s package), always at ~45-46s elapsed, while the SAME tests passed pre-merge in the worktree (31-34/34 packages) and passed solo on master immediately after (3/3 solo re-runs green).

CURRENT VS EXPECTED: ccs merge <wt> output shows 'FAIL github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare 45.916s' + 'FAIL' after 'Successfully merged', every time another test process ran concurrently (my own gate runs or the CCS go-test lock queue). Expected: either the post-merge step passes when the code is good, or it names the flaky test. Notably one merge printed its own hint 're-run solo: go -C . test ./pkg/cosmoflare/' — the tooling knows this pattern.

WHY IT MATTERS: A false FAIL on the merge step trains operators to ignore merge-gate test output — the exact trust erosion S334 warns about. It also forces a manual solo re-run after every affected merge.

PRIORITY: Medium — deterministic trigger (concurrent load + slow package), 3 occurrences in one session.

REPRO: Start a go test of pkg A in one process; run ccs merge on a worktree touching a ~80s package; observe the post-merge FAIL at ~45s with no failing test named.

AFFECTED FILES: Unverified — the post-merge test invocation in cmd/merge*.go (or internal/merge/) and whatever timeout it applies; the ~45s ceiling suggests a fixed deadline shorter than the package runtime under load.

SUGGESTED IMPLEMENTATION: Name the failing test in the output (a bare 'FAIL <pkg> 45.9s' with no --- FAIL lines is a timeout kill, not a test failure — distinguish them), and either raise the post-merge deadline proportional to the package's cached solo runtime or serialize on the existing go-test lock the pre-merge step uses.

SESSION CONTEXT: 2026-09-10 cosmoflare — 16 agent merges; the three false FAILs (0093, 0103, 0106 worktrees) all hit pkg/cosmoflare while /cover gate runs or my worktree verification tests held CPU; all three solo re-runs passed in 73-84s.

