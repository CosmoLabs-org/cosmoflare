# Session 2034 — Efficiency Review

Date: 2026-10-10 · Deep tier · 28+ commits, 10 GLM dispatches, 2 deploys.

## Token waste found

1. **`ccs glm-agent review` dumps raw JSONL transcripts** — thinking-token spam
   dominates; the useful payload is `.glm-agent/report.md` + the final text
   message. One review call cost several thousand tokens of noise before I
   switched to reading report.md directly.
2. **`ccs test-in-worktree` ran the Go root suite** for a pager-only worktree
   (~90 s + output). For JS packages inside this repo it is the wrong runner;
   subshell `(cd <wt>/<pkg> && ...)` is correct.
3. **Image reads render ~2 per session window** — 16 screenshots taken, most
   uploaded to CDN but never rendered. DOM audits and the vision MCP (when not
   rate-limited) substituted. Worth knowing before planning screenshot-gated
   reviews.
4. **4 of 10 agent deaths** (0365, 0368, 0371, 0373) burned dispatch cycles;
   salvage recovered real work twice, one brief was implemented directly after
   two deaths.

## Friction, not waste

- BUG-726 ancestry false-positive after every rebase merge (4×) — cheap to
  dismiss once the pattern is known, but it cries wolf.
- rm→Trash hook blocks actual disk-space recovery (Trash shares the volume);
  `ccs cleanup --full` is the working path.

## What worked

- Bounded briefs + `--skip-validate` for research; parallel scout fleet on the
  analysis clone; scratchpad live-data harness (reused twice this session);
  per-goal prompt ticks with autotick + add-commit backfill closing the I5 gate
  fully COVERED.
