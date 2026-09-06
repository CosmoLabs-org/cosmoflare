---
ulid: 01M1W1YE48AH7HTVSQDR2EHQ5J
title: glm-agent exec auto-commits worktree-local state files (intel status, fingerprint cache) into agent commits
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-06T23:08:27.14419+04:00"
updated: "2026-09-06T23:08:27.14419+04:00"
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

# FB-p2EHQ5J: glm-agent exec auto-commits worktree-local state files (intel status, fingerprint cache) into agent commits

PROBLEM: all 4 doc-review agents dispatched via 'ccs glm-agent exec' on 2026-09-06 committed three worktree-local state files into their feature commits: .gorchestra/fingerprint-cache.json (project_name rewritten to the WORKTREE name), .version-registry.json (build counter bumped), GOrchestra/intel/status.json (project_path pointing at the worktree). CURRENT VS EXPECTED: Current — every agent commit carries the contamination; the operator must strip per worktree before merge (I used 'git show master:<f> > <f>' + commit --amend on each; the hook correctly blocks 'git checkout master -- <f>'). The merge pipeline's BUG-597 strip caught the remainder on 3 of 4 merges — partial coverage, and the merged tree still risks a status.json whose project_path names a dead worktree if the strip misses. Expected — these files are never committed by agent auto-commit in the first place. WHY IT MATTERS: 4/4 agents contaminated; without manual pre-stripping, master receives intel state pointing at ephemeral worktree paths — silently wrong telemetry. The auto-commit exclusion list should match the drift-exclusion list. SUGGESTED IMPLEMENTATION: kill.go recoverBranchContent already excludes drift via the FB-1492 add-pathspec excludes (':!.claude', ':!.version-registry.json', ...). The glm-agent exec pre-dispatch auto-commit (FB-262) appears to lack the same excludes for '.gorchestra/fingerprint-cache.json' and 'GOrchestra/intel/status.json' — extend its exclusion pathspec to match, and include '.gorchestra/'. AFFECTED FILES: tools/ccsession/cmd/glm_agent_exec.go (auto-commit path, FB-262; exact func unverified — grep for autoCommit), kill.go:685 for the existing exclusion pattern to copy. SESSION CONTEXT: cosmoflare session-end 2026-09-06, merging 4 GLM doc-review agents (0048-0051); observed identical 3-file contamination in every worktree diff.

