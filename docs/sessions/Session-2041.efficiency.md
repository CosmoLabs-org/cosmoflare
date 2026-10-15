# Session 2041 — Efficiency Review (2026-10-16)

## What worked

- Per-task GLM manifests over one batch file: exec-batch's dry-run exposed the dependency-deferral machinery immediately; five sequential dispatches with gated merges cost ~30 orchestrator minutes total.
- Salvage-first on dead agents: zero re-dispatches, zero wasted agent turns — every death's committed work became the reviewed diff.
- Browser-geometry assertions (getBBox in Playwright) replaced three rounds of vision-model argument over the same 4px question — the measurement closed it permanently.
- The workcheck conversation ledger + G-ID ticking caught a 5/5-evidenced prompt sitting at 0/5 ticked.

## Waste observed

- Vision-model screenshot passes flip-flopped on sub-8px details (dot "visible" → "absent" → "visible" across identical DOM) — four analyze_image rounds where one getBBox check would have settled point 3 immediately. ~3 tool rounds of noise; lesson recorded in LESSONS and the fit spec now measures instead.
- Char-count width thresholds (6.8 / 5.44 / ≥6) were authored, reviewed, and replaced within one session — absolute thresholds against size-varying geometry. Should have gone straight to the chars × advance vs innerD rule the altitude review proposed.
- The first push of the release failed on a stale non-fast-forward that a fetch would have cleared — fetch-before-push next time.

## CCS opportunities

None filed this session (cap discipline): both deferred improvements (cost collector parallelization, vanilla gauge builder deletion) are captured as continuation goals, which is the right home — they need code, not tracker rows.
