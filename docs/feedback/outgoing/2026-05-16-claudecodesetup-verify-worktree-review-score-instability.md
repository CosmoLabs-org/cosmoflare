---
id: FB-966
title: verify-worktree review score instability across rounds — issues change instead of confirming fixes
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-16T07:30:38.330231-03:00"
updated: "2026-05-16T07:30:38.330231-03:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 2027
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

# FB-966: verify-worktree review score instability across rounds — issues change instead of confirming fixes

## Problem Description

verify-worktree produces different issue lists across consecutive review rounds on the SAME diff. Instead of confirming that previous issues were fixed, each round finds entirely new nits. This causes convergence-cap to be the only viable path for agent worktrees, defeating the purpose of the fix-and-re-review loop.

## Current vs Expected

Current: Round 1 finds Issues A,B (score 6). Fix A,B, commit. Round 2 finds Issues C,D (score 6). Fix C,D. Round 3 finds Issues E,F (score 7). Eventually converges at cap after 8-15 rounds.

Expected: Round 1 finds Issues A,B (score 6). Fix A,B. Round 2 confirms A,B fixed, may find one new issue C (score 8). Fix C. Round 3: approved.

Concrete example from this session — email service worktree (agent-a2d6b0debf0c3b045):
- Round 1: "Update --enabled defaults to true" + "UpdateRule uses concrete types" + "Verified omitempty no-op" + "TestEmptyAccountID misleading" (score 6)
- Rounds 2-3: Fixed ALL four. New issues appeared: "priority sentinel -1 not interpreted" + "empty matchers sent on update" (score 6)
- Rounds 4-5: Fixed those with fetch-before-update. New issues appeared about different concerns (score 6→8, finally converged at cap round 15)

## Why It Matters

Every agent merge in this session required 7-15 verification rounds. Six worktrees × 10 rounds average = ~60 LLM review calls for work that could have been reviewed in ~12 rounds (2 per worktree). This is the dominant cost of the session's merge pipeline. The "fix loop" becomes a "nit treadmill" where the reviewer generates new observations faster than the developer can converge.

## Priority Justification

High — affects every session that uses agent worktrees. This session dispatched 8 agents and spent more time on verify-worktree loops than on actual implementation.

## Reproduction Steps

1. Dispatch any agent that creates 200+ lines of new Go code
2. Run `ccs verify-worktree <name> --approve`
3. Fix all reported Issues
4. Re-run `ccs verify-worktree <name> --approve`
5. Observe: new, different Issues appear instead of confirming fixes

## Affected Files

tools/ccsession/cmd/verify_worktree.go — the LLM prompt that generates the review may not include prior-round context
tools/ccsession/internal/review/ — review caching and round tracking

## Suggested Implementation

Option A: Include prior-round Issues in the review prompt so the LLM can confirm fixes rather than finding new ones. "Previous round flagged: [list]. Verify these are fixed. Only report NEW issues not in the prior list."

Option B: Lock the issue list after round 1. Subsequent rounds ONLY check whether those specific issues are resolved. New findings go to Suggestions, not Issues.

Option C: Score the DELTA between rounds. If all prior Issues are fixed and new ones are lower-severity, auto-approve.

