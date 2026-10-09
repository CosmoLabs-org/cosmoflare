# Efficiency Review — Session 2032 (2026-10-08 → 2026-10-09)

## What cost the most

| Item | Cost | Avoidable? |
|------|------|-----------|
| `ccs merge` refusing fresh reviews (stale tracked `.review.json`) | 3 failed merges, 1 orphaned worktree, 1 branch recovery from the object store, ~10 tool calls | Yes — root cause fixed (9d94423); tool feedback queued |
| `zone-uncached-pct` built, tested, documented, then replaced after the live dry run | One full implement-review-merge cycle plus one discarded docs wave (2 GLM agents) | Partly — a live dry run at design time (before T4) would have caught it |
| Discarded doc agents 0331/0332 | 2 GLM runs | Yes — same cause as above |
| GLM dispatch blocked at the 40 GiB disk floor | 1 operator round-trip | No — real constraint; Go build cache cleared |
| Doc-lint false positives (installed v0.32.0 binary; snippet executed as a command) | 3 tool calls to diagnose | Tool issue |

## What worked

- Live GraphQL introspection before design found BUG-055 and the real field names; no schema guesswork reached code.
- Fable plan review caught 3 compile/test failures in GLM task code before dispatch — zero GLM reruns for those.
- Byte-for-byte plan-vs-diff checks (`diff <(gofmt plan) file`) made GLM gate reviews fast for T1, T2, T6, T7.
- Gating worktree cleanup on an ancestry check after the one bad chain prevented any repeat.

## Next time

- Dry-run new alert conditions against the live account during design, before implementation.
- Never chain `ccs kill` after `ccs merge` with `;`.
- Name test packages explicitly (`./cmd/ ./internal/webhook/ ./pkg/cosmoflare/...`) — `./internal/...` includes the keychain package (operator rule: no keychain probing).
