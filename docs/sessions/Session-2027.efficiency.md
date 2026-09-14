# Session 2027 — Efficiency Review

## What worked

- Audit-skill verbatim protocol (agents wrote reports to disk directly) — zero transcription loss, orchestrator context stayed lean through 13 agents
- Species-1 regex sweep: 196 sites converted mechanically in one pass vs ~40 hand edits — the two-species classification paid for itself
- `--force-with-lease` + delete-then-push patterns after the first hook block: zero repeated force-push friction
- Per-change red-green discipline meant session-end needed no re-verification runs

## Waste observed

- `ccs session-end-docs --apply` slot-shape contract: 3 failed attempts (array → object-of-strings → object-of-arrays) before hand-writing artifacts. Feedback sent to ClaudeCodeSetup (schema/example in scaffold).
- `ccs handoff --smart` resurrected a completed prompt's goals — had to manually mark stale + rewrite. Root cause: status-blind prompt selection. Noted in LESSONS.
- Smoke 2m timeout on the full-test step despite the 300s budget commit (6f3ea6e) — pre-flagged environmental class, still noisy at session-end.
- filter-repo's `already_ran` prompt needed a manual marker delete; the error message surfaces only the prompt text (EOF in non-interactive shells).

## Numbers

- ~20h, 20+ commits, 12 bugs fixed, 2 releases, 1 history purge, repo public
- cmd if-JSONOutput: 618 → 410
- Repo pack: 21.8 → 4.2 MiB
- Known-efficiency items resolved this session: 1 (index rebuild OK:99)
