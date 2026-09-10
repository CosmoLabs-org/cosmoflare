---
ulid: 01M25WBEJSTNYZPE4Q741JH7MC
title: 'verify-agent false-fails real work: improvement-mode check-4 and metadata check-2'
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T18:43:06.457485+04:00"
updated: "2026-09-10T18:43:06.457485+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 49
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

# FB-p1JH7MC: verify-agent false-fails real work: improvement-mode check-4 and metadata check-2

PROBLEM: ccs coverage verify-agent rejected two categories of legitimate agent work this session, forcing manual override via verify-worktree session review.

CURRENT VS EXPECTED: (1) Improvement-mode agents (internal/config 0098, internal/tui 0097, pkg/cosmoflare 0106) failed check 4 'coverage is 0.0% (no positive coverage detected)' — improvement mode does not ADD coverage by design; the check measures generation-mode output. Expected: --improvement relaxes check 4 the way it relaxes the deletion check. (2) Generation agents (main 0101, installer_tui 0102) failed check 2 'non-test file(s) modified: .gorchestra/fingerprint-cache.json, GOrchestra/intel/status.json' — those CCS metadata files ride along in EVERY agent commit; ccs merge itself strips them (BUG-597 chore commit). Expected: check 2 whitelists the known agent-metadata set.

WHY IT MATTERS: Every false failure forces the operator to bypass the deterministic gate and fall back to judgment — the gate's whole value is being trusted.

PRIORITY: High — the /cover skill's Phase 6 step 1 is this command; every improvement-mode run in any repo hits (1).

REPRO: ccs coverage verify-agent <worktree-of-improvement-agent> --improvement — observe check 4 FAIL 0.0% while the full test suite passes in the same run.

AFFECTED FILES: Unverified — the verify-agent check implementations live in the coverage command path (cmd/coverage*.go or internal/coverage/ in tools/ccsession; needs find-func to confirm exact site).

SUGGESTED IMPLEMENTATION: In --improvement mode skip check 4 (or compare before/after coverage equality instead of positive delta). In check 2, ignore .gorchestra/, GOrchestra/intel/, .version-registry.json — the same set the merge gate strips as BUG-597 artifacts.

SESSION CONTEXT: 2026-09-10 cosmoflare /cover run — 8 agents gated; 3 improvement agents all failed check 4 identically, 2 generation agents failed check 2 on the metadata pair. All were correct work; all merges succeeded after manual session review.

