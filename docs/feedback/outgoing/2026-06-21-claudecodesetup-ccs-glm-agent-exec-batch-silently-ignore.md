---
id: FB-1232
title: ccs glm-agent exec-batch silently ignores --dry-run AND defaults to wave-size 2 (dispatched real agents on sequential same-package tasks)
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-06-21T01:09:48.514809-03:00"
updated: "2026-06-21T01:09:48.514809-03:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 17
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

# FB-1232: ccs glm-agent exec-batch silently ignores --dry-run AND defaults to wave-size 2 (dispatched real agents on sequential same-package tasks)

What happened: I ran 'ccs glm-agent exec-batch <manifest> --dry-run' expecting a no-op validation. The --dry-run flag was silently ignored ('Unknown flag --dry-run ignored') and it dispatched 3 REAL GLM agents. Worse, it defaulted to --wave-size 2 (parallel) even though the manifest header documented the tasks as SEQUENTIAL same-package (all touch internal/server/server.go) — so the parallel agents would have produced conflicting/broken output (P-02/P-03 modify a server.go that P-01 hadn't created yet on their shared master base). I had to kill+discard all 3. Why it matters: 'exec-batch --dry-run' reads as safe; silently dispatching real agents is a footgun, and ignoring the manifest's sequential intent (no per-manifest default wave-size, or a 'sequential: true' key) causes broken merges. Proposed: (1) support --dry-run on exec-batch (validate+plan, no dispatch) OR reject unknown flags instead of ignoring; (2) let the manifest declare wave-size / sequential per-file-conflict detection so same-file tasks never parallelize. Priority: high — this is an easy way to accidentally launch a fleet.

