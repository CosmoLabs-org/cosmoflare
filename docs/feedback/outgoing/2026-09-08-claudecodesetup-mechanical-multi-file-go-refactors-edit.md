---
ulid: 01M20MS4J1BC0ZENCQD6Y0E1AX
title: 'Mechanical multi-file Go refactors: Edit-tool Read-first gate forces fragile python3 heredocs'
type: improvement
status: pending
priority: low
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-08T17:54:34.177214+04:00"
updated: "2026-09-08T17:54:34.177214+04:00"
suggested_conversion: feature
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

# FB-pY0E1AX: Mechanical multi-file Go refactors: Edit-tool Read-first gate forces fragile python3 heredocs

Problem: consolidating a duplicated helper across 3-4 Go files this session required either N times (Read+Edit) tool cycles or python3 heredoc scripts with exact-match asserts. Current vs expected: the python path took 3 iterations — 2 failed on text mismatches (caught by asserts, no corruption) before succeeding; expected is a sanctioned batch-edit path with the same safety. Why it matters: every cross-file mechanical refactor (the simplify phase produces these routinely) pays this friction; the assert-guarded python pattern works but is undocumented, so the next session may use unguarded sed and corrupt a file. Suggested implementation: document the assert-guarded python heredoc as the sanctioned pattern in the simplify/session-end guidance, or add a ccs edit --batch command accepting per-file exact-match replacement pairs with the same assert semantics. Session context: 2026-09-08 cosmoflare session-end simplify phase — restClient transport consolidation across bucket_domain/lifecycle/notifications (3 python iterations, all asserts held, build+tests green after).

