---
ulid: 01M239NDKJ1K9AKFNV8M4GYQQ4
title: Numbered goal headings parse as 0/0 in ccs prompts machinery (FB-2 forward from cosmoflare)
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-09T18:38:01.330862+04:00"
updated: "2026-09-09T18:38:01.330862+04:00"
suggested_conversion: feature
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

# FB-p4GYQQ4: Numbered goal headings parse as 0/0 in ccs prompts machinery (FB-2 forward from cosmoflare)

What happened: continuation prompts using '### N. [GOAL] Title' headings parse as zero goals — ccs workcheck prints 'no parseable goals; drift detection N/A' and prompts autotick cannot attribute completion. Verified again 2026-09-09: 'ccs prompts verify docs/prompts/2026-09-05-next-session.md --mechanical-only' reports 'coverage: 100% verdicts: 0' (no machine structure found).

Counter-evidence from the same day: prompts using the '### [ ] G-NN Title' marker format parse and verify perfectly — an 8-goal prompt went 8/8 CONFIRMED_COVERED with position-aware set-goal ticking and commit attribution (cosmoflare session 2026-09-09, docs/prompts/2026-09-09-domain-center-residuals.md).

Why it matters: the I5 verification layer is blind for any prompt authored in the numbered style; goal completion silently drops out of machine checks.

Proposed solution: either (a) make the marker format the only documented prompt style and lint against numbered headings at prompts validate, or (b) teach the parser '### N. [GOAL] Title' as a fallback format. (a) is cheaper and matches the /continuation-prompt template already.

Evidence/repro: ccs prompts verify on any numbered-format prompt shows verdicts: 0 while the body declares goals_total.

