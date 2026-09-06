---
ulid: 01M1W1W5VB7HZF4MNVMS6E0QGF
title: Continuation prompt goals are invisible to machine verification (0/0 parsed)
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: cosmoflare
to_target: self
created: "2026-09-06T23:07:13.131812+04:00"
updated: "2026-09-06T23:07:13.131812+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 38
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

# FB-p6E0QGF: Continuation prompt goals are invisible to machine verification (0/0 parsed)

WHAT HAPPENED: docs/prompts/2026-09-05-next-session.md uses '### N. [GOAL] Title' headings. ccs workcheck and prompts autotick both report 0/0 goals — the numbered format has no machine-parseable markers, so goal completion cannot tick automatically and workcheck prints 'no parseable goals; drift detection N/A'. WHY IT MATTERS: the I2 invariant (one TaskCreate per goal) works because the human/agent parses the prompt, but the verification layer (I5) is blind. Set-goals 5/6 had to be manual. PROPOSED SOLUTION: continuation prompts should emit '### [ ] G-NN Title' markers (same as /continuation-prompt template in ClaudeCodeSetup) — or ccs prompts should also parse '### N. [GOAL]' as a fallback. Evidence: ccs workcheck output 2026-09-06 showing 'Progress: 0/0 goals' while the prompt declares goals_total: 6 and 5/6 were actually completed.

