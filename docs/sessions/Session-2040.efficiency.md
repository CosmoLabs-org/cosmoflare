# Session 2040 — efficiency notes (2026-10-15)

## Token wins
- All 8 implementation waves dispatched to GLM worktree agents (S334-gated in-session) — main-session context held plans + diffs only.
- `ccs glm-agent wait` background pattern avoided poll-waits; combined waits for parallel agents.
- Fresh-eyes reviewers (2x Opus forks) caught 17 defects total before merge/execution — cheaper than post-merge fix cycles, and both paid for themselves immediately.

## Waste observed
- Two agent deaths on provider faults (429 rate-limit, 529 overload) burned ~2 dispatch cycles each before recovery; salvage + --base-branch continuation recovered 100% of work. No fix available in-repo — provider-side.
- `ccs merge`'s post-merge archive step fails every time in cosmoflare (GOrchestra/ gitignored) and prints a scary repair banner + BUG-726 false positive; diagnosed 4 times this session (~4 tool calls each). Fix candidate for ClaudeCodeSetup: merge should no-op the archive when the target path is gitignored.
- `bun test` vs `vitest run` runner mismatch cost one false-red investigation (202 fake failures). Noted in continuation prompt lessons.
- The doc-reviewer's wrong corpus count (47 vs 28) would have sent a fix agent after correct code; direct re-measurement caught it. Reviewer numbers are claims, not facts.

## Efficiency-review standing items
The 4 unresolved efficiency-review items (IMP-001/003/004, docker-smoke-classification) remain open from prior sessions — unchanged by this session.
