---
ulid: 01M23YHFGSC3EBJ69PY42T4YZM
title: Skill launchers redirect in loops without loading content (fix-bug/bug-fix, triage-feedback/triage)
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T00:42:52.313054+04:00"
updated: "2026-09-10T00:42:52.313054+04:00"
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

# FB-p2T4YZM: Skill launchers redirect in loops without loading content (fix-bug/bug-fix, triage-feedback/triage)

What happened twice today (2026-09-09, cosmoflare session): (1) invoking the 'fix-bug' skill returned only 'Invoke the bug-fix skill using the Skill tool with the user's arguments'; invoking 'bug-fix' returned the SAME redirect again — an infinite launcher chain, no content ever loaded. (2) 'triage-feedback' returned 'Invoke the triage skill' — no 'triage' skill is registered ('Unknown skill: triage'). Both are the BUG-498 double-invocation guard's sibling failure: launcher chains that deliver no body. Why it matters: the operator cannot distinguish a registry gap from a transient failure, and each retry burns a turn; the documented fallback (execute the skill's description directly) requires guessing the workflow. Proposed solution: launcher commands should verify the target skill exists and is content-bearing before emitting the redirect (a two-hop cap with a clear 'launcher target missing from registry' error naming the missing skill and the correct fallback). Repro: /fix-bug BUG-039 and /triage-feedback in a cosmoflare session on the 2026-09-09 skill set.

