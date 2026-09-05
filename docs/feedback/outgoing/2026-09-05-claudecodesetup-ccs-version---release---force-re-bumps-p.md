---
ulid: 01M1S5Y7VVAFG1X3KQ5BN5JP75
title: ccs version --release --force re-bumps past an untagged version instead of completing it
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-05T20:20:31.739706+04:00"
updated: "2026-09-05T20:20:31.739706+04:00"
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

# FB-pN5JP75: ccs version --release --force re-bumps past an untagged version instead of completing it

What happened: a partial release (bump+commit done, tag/push refused on a dirty path) left v0.18.0 committed but untagged. Re-running with --force per the tool's own hint ('v0.18.0 was never tagged. Use --force to skip') bumped AGAIN to v0.19.0 and shipped that. The hint reads as 'skip the warning and finish 0.18.0' but --force means 'skip reconciliation and continue'. Result: a cosmetic version skip on a real release.

Why it matters: the recovery path for a partial release is ambiguous at exactly the moment the operator is least calm (release blocked, dirty tree). The two intents -- complete-the-untagged-version vs release-next -- need distinct flags or an interactive choice.

Proposed solution: (a) add an explicit recovery mode, e.g. `ccs version --release --retry` or `--complete-untagged`, that tags the existing bumped commit; (b) reword the hint to state plainly what --force will do ('--force will bump to the NEXT version (0.19.0) and leave 0.18.0 untagged; to tag 0.18.0 instead run X').

Second item, same session: `make install` (run by ccs merge/version flows) recreates build/r2go2 and it returns as a tracked dirty path even after git rm --cached + build/ in .gitignore — the release flow then refuses to fold the registry. Observed twice on 2026-09-05 (commits d44ee28 and beffff7 both untrack it). Whatever re-adds it (make install target or ccs fold step) should respect the ignore rule or the release pre-flight should offer to untrack it automatically.

Priority: medium — release tooling UX, hit during the first successful release in this repo's history.

