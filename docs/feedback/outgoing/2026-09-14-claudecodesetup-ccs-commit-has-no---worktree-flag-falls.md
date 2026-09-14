---
ulid: 01M2GG5NQNPPVMW2SYDRKX2564
title: ccs commit has no --worktree flag; falls through to git-commit(1) help noise
type: bug
status: pending
priority: low
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-14T21:41:53.013334+04:00"
updated: "2026-09-14T21:41:53.013334+04:00"
suggested_conversion: bug
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

# FB-pKX2564: ccs commit has no --worktree flag; falls through to git-commit(1) help noise

What happened: hand-finishing a conversion inside an agent worktree (cosmoflare 2026-09-14), 'ccs commit --direct --worktree <path> -m ...' passed the unknown flag through to git and printed git-commit(1) usage; nothing committed; 'ccs commit --help' shows only git's help so the correct invocation is undiscoverable. Workaround: plain 'git -C <wt> commit -m' (hooks pass on clean conventional messages). Why it matters: rules say never raw git commit, but the sanctioned wrapper cannot target worktrees — exactly where hand-finish commits happen after agent stalls. Fix direction: support 'ccs commit --direct --worktree <path>' (bind -C), or document the git -C workaround in rules/quality-gate.md. Evidence: session transcript 2026-09-14 cosmoflare, agent 0133 worktree commit attempts.

