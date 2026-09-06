# Session 40 — Efficiency Analysis (narrative supplement)

**Date**: 2026-09-07
**Deterministic checks**: clean (no manual JSON edits, no git-add bypass)
**Delegation**: 5 agents dispatched (3 goals + 2 session-end docs); 1 merged (score 9), 2 verified no-op (work pre-existing on master), 2 in flight

## Token Waste Identified

| Task | Tokens | Resolution |
|------|--------|------------|
| Release publish flow executed manually twice (v0.20.0, v0.21.0) | ~6,000 | IDEA-050: `make release TAG=` target codifying the recipe |
| Locating agent worktree paths (3 lookups per wave) | ~800 | FB filed: glm-agent status --json expose worktree/branch |
| Copyright scan + sweep (grep/perl, 3 calls) | ~1,200 | FB-p97SDF1: `ccs copyright` scan/fix |
| Force-push lease dance after rewrite (2 failed attempts) | ~600 | FB filed: hook guidance with explicit lease-value form |

**Total potential savings**: ~8,600 tokens next occurrence

## Pattern Observations

- Two of three dispatched goal-agents found their work already done: the continuation prompt was written BEFORE a parallel wave landed goals 3+5. Fix pattern: check `git log master --grep` for the goal's keywords before dispatch (cheap pre-flight), or make session-end annotate prompts with post-creation commits per goal.
- Changelog near-duplicate landed despite dedupe (known FB-pGVA5Z3 gap) — manual substring removal needed again.

## GLM Opportunities (missed)

None material: the copyright sweep was cheaper inline (2 calls) than a dispatch; release publishing is main-session operator work by design (gh auth + goreleaser).

## Priority Actions

1. **High**: IDEA-050 (make release) — twice-in-one-session friction, fully mechanical
2. **Medium**: pre-dispatch dedup check (prompt goals vs recent master commits)
3. **Low**: session-end docs agents could reuse a shared prompt template file
