---
ulid: 01M1WEY6YCMA12XWNT8H4TZJ48
title: 'force-push hook: suggest explicit lease-value form for post-rewrite pushes'
type: feature
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-07T02:55:31.276834+04:00"
updated: "2026-09-07T02:55:31.276834+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 40
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

# FB-p4TZJ48: force-push hook: suggest explicit lease-value form for post-rewrite pushes

Problem: the git-push block hook rejects 'git push --force' and suggests 'git push --force-with-lease'. After a git-filter-repo history rewrite (which strips remotes), the suggested bare --force-with-lease fails with 'stale info' because remote-tracking refs are gone — the working form is '--force-with-lease=refs/heads/<branch>:<expected-sha>' with the expected SHA from 'git ls-remote'.

Current vs expected: expected the hook guidance to cover the rewritten-history case. Got: one failed push attempt before deriving the explicit lease form. Repro context: cosmoflare 2026-09-07 history purge — 'git push --force-with-lease=refs/heads/master:<mirror-sha> origin master' failed 'stale info' (remote had moved past the mirror), then succeeded with the ls-remote value 4299870.

Why it matters: history rewrites are a sanctioned workflow in this ecosystem (recovery-patch lifecycle work); every one of them walks the operator into the same dead end. The lease-with-value form is strictly safer than --force anyway (pins the exact expected remote state) — the hook could recommend it as the primary form.

Suggested implementation: extend the hook message in tools/cosmohooks (push-guard) with: 'After a history rewrite or when tracking refs are stale, use --force-with-lease=refs/heads/<branch>:<expected-sha> (expected from git ls-remote origin <ref>).'

Session context: cosmoflare session 2026-09-07 purged 630MB from git history; force-push of rewritten master/TauriApp/tags hit the hook block, then the stale-info failure, before the explicit lease form worked on all three.

