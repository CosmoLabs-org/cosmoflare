---
created: "2026-06-21T01:10:08-03:00"
updated: "2026-06-21T01:10:08-03:00"
status: PENDING
priority: high
branch: master
title: "Master session — review TauriApp Wave 1, then Waves 2-4 or other features"
tags: [continuation, master-session, tauri, handoff]
requires_reading:
  - docs/prompts/2026-06-20-cosmoflare-desktop.md
  - docs/sessions/Session-017-domain-management-center-and-desktop-brainplan.md
schema_version: 1
---

# Master session — next

## ⚠️ FIRST, READ THIS: Wave 1 is being built elsewhere — review it, don't rebuild it

A **glm-tree worktree `TauriApp`** (separate tab, GLM engine) is building **Wave 1**
of the Cosmoflare Desktop app (the Go `cosmoflare serve` daemon — G-01/02/03 in
`docs/prompts/2026-06-20-cosmoflare-desktop.md`). **Do NOT `/run-continuation` the
desktop prompt and start building the daemon yourself** — that would duplicate the
glm-tree's work. This master session's job is to *review, merge, and continue*.

## Priority order

1. **Review + merge the glm-tree's Wave 1 output (S334 gate).**
   - `ccs worktree list` and inspect the `TauriApp` worktree commits.
   - For each daemon piece: read the diff, **re-run `GOWORK=off go test ./internal/server/` yourself** (don't trust prose), confirm `go build ./...`.
   - If clean: `ccs merge TauriApp` (or `git merge --no-ff`), then `ccs kill TauriApp`.
   - If stalled/partial: `ccs glm-agent salvage` / cherry-pick the good parts and finish Wave 1 inline.
   - Verify on master: `GOWORK=off go test ./internal/server/ && go build ./...`, and the keychain-silence check.

2. **Then choose:**
   - **(a) Continue the Tauri build — Waves 2-4 (Opus-led):** Tauri v2 scaffold (P-04), Rust daemon lifecycle (P-05), React UI (P-06/07/08), packaging (P-09). Per `docs/planning-mode/2026-06-20-cosmoflare-desktop.md`. These are NOT GLM-suitable (interactive tooling + cross-platform judgment).
   - **(b) Other roadmap work:** the 6 items added this session — ROAD-079 (serve daemon, overlaps Wave 1), ROAD-080 (webhook event bus — unblocks real notifications), ROAD-075 (desktop v2 graph+CRUD), ROAD-081 (desktop distribution/signing), ROAD-082 (paid licensing & auth), ROAD-083 (cmd/ DI rollout) — plus the 12 active roadmap items. Run `/triage` to score.

## State at handoff (v0.16.0)
- GLM pool **fixed** (stale ccsdaemon restarted; `glm-5.2[1m]`→`glm-5.2` normalizes). G-00 done.
- Working tree clean; FEAT-006 shipped + closed; no open bugs/features/feedback.
- If GLM dispatch ever fails again with `unknown pool`, the daemon is stale → `ccs daemon restart`.
- `exec-batch` gotcha: no real `--dry-run`; default `--wave-size 2` parallelizes — use `--wave-size 1` for sequential same-package tasks (filed to ClaudeCodeSetup).

## Suggested opener
> "Check the TauriApp worktree — review the Wave 1 daemon commits (S334: diffs + re-run `go test ./internal/server/`), merge if clean, then tell me whether to do Tauri Waves 2-4 or pick roadmap work."
