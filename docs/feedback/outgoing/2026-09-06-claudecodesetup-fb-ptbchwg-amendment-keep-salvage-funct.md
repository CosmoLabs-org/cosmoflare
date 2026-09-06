---
ulid: 01M1W0M2BNAWYFFE34H2S6PF06
title: 'FB-pTBCHWG amendment: keep salvage functionality — restore-from-archive command required before age-prune ships'
type: improvement
status: pending
priority: high
complexity: medium
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-06T22:45:18.837061+04:00"
updated: "2026-09-06T22:45:18.837061+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 38
suggested_workflow:
  - brainstorming
  - implementation
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
---

# FB-pS6PF06: FB-pTBCHWG amendment: keep salvage functionality — restore-from-archive command required before age-prune ships

# SUMMARY (what happened)

Amendment to FB-pTBCHWG (same file: docs/feedback/incoming/2026-09-06-cosmoflare-gorchestra-recovery-patches-never-pruned.md, not yet ingested). The operator reviewed the proposal and added a binding design constraint: the fix must KEEP the salvage functionality. Prune the accumulation, not the insurance. The original Layer 1 proposed a 30-day blind age-prune; that alone would silently destroy the only copy of unmerged work a human might still want.

# MOTIVATION (why it matters)

A recovery.patch is the ONLY surviving copy of a killed worktree's unmerged commits — kill deletes both the worktree and the branch. Deleting by age alone converts "insurance nobody can claim" into "insurance nobody was told was expiring". The reason patches accumulate unclaimed is not that the data is worthless; it is that NO command can restore from the archive (`ccs glm-agent salvage` reads live sessions only). Fix the claim path and the retention window becomes a real lifecycle instead of a landfill.

# SOLUTION (amend Layer 1 of FB-pTBCHWG with this block)

Append after the Layer 1 bullet list in the SOLUTION section:

> **DESIGN CONSTRAINT (operator requirement, 2026-09-06): keep the salvage functionality — prune the accumulation, not the insurance.**
> 1. Make the archive readable, not write-only. Extend `ccs glm-agent salvage` (or add `ccs sessions restore <name>`) to re-apply an archived recovery.patch into a fresh worktree. With a working restore, the 30-day prune becomes expiry of UNCLAIMED insurance rather than data loss.
> 2. Surface the countdown. `ccs sessions` listing shows archived-with-patch sessions with patch age / remaining retention; `ccs discard <name>` for explicit early destruction.
> 3. Never weaken the no-patch-when-merged invariant (kill.go:725) — the patch stays exclusively the unmerged-work insurance.
> 4. Prune policy = age OR claim. Default 30d expiry; restoring a patch resets/removes its expiry. Per-project config for longer insurance windows.

Also amend the first Layer 1 bullet: record patch creation time AND mark the session status salvageable-with-unmerged-work (distinct from the BUG-076 zombie "working" status).

# REPRO / EVIDENCE

Original FB-pTBCHWG file, lines 65-68 (Layer 1 block being amended). Cross-project edit was hook-blocked (cosmohooks cross-project-guard), so this follow-up carries the amendment per the sanctioned path.

# PRIORITY

high — same as parent FB-pTBCHWG; this constraint changes the implementation shape (restore command required before prune-by-age ships).

## Suggested Workflow

1. brainstorming
2. implementation

