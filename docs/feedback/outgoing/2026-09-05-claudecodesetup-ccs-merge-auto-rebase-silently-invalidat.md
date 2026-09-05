---
ulid: 01M1S5ZNWHHH7T54C1M1Q6FHAG
title: ccs merge auto-rebase silently invalidates the BUG-726 ancestry check the quality-gate SOP mandates
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-05T20:21:18.865653+04:00"
updated: "2026-09-05T20:21:18.865653+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 36
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

# FB-pQ6FHAG: ccs merge auto-rebase silently invalidates the BUG-726 ancestry check the quality-gate SOP mandates

What happened: quality-gate.md instructs orchestrators to verify merges with `git merge-base --is-ancestor <branch-tip> master`. Twice today (BUG-029, BUG-030 merges) the check failed after a successful ccs merge because ccs merge auto-rebased the branch before merging — the original branch-tip SHA no longer exists in history (rebased to a new SHA), and after merge+archive the branch ref itself is deleted. The merge had landed correctly; only the verification protocol was checkable-proof-less.

Why it matters: the SOP's anti-hallucination gate depends on a SHA that ccs merge itself destroys. Orchestrators either false-alarm (my case: extra diagnostic steps) or learn to skip the check — both bad.

Proposed solution: have `ccs merge` print the POST-rebase commit SHA it actually merged (or write it to the archived session.json as merged_sha), so the documented verification becomes `git merge-base --is-ancestor <merged_sha> master` and always works. Alternatively document 'verify by content grep, not SHA' in the SOP.

Priority: medium — protocol coherence, hit twice in one session.

