---
created: "2026-06-21T08:10:00-03:00"
updated: "2026-06-21T08:10:00-03:00"
status: PENDING
priority: high
branch: master
title: "Next session — desktop v1.1 follow-ups + ROAD-080 event bus"
origin: "/session-end"
tags: [continuation, desktop, followups, bugs, ROAD-080, event-bus]
requires_reading:
  - docs/prompts/2026-06-21-inprocess-event-bus.md
  - docs/issues/BUG-022.yaml
  - docs/issues/BUG-023.yaml
  - docs/issues/BUG-024.yaml
schema_version: 1
---

# Next session — desktop v1.1 follow-ups + ROAD-080 event bus

## Context

Cosmoflare Desktop v1 shipped and was closed out last session (FEAT-007,
ROAD-063, ROAD-079 closed; the whole Tauri app — Go serve daemon, Rust
lifecycle, React dashboard/notifications — merged and the 5 critical/high review
findings fixed). This session's explicit goal (per the user): **address the open
bugs and follow-ups caught during that review, right away.** Three bugs were
filed plus two v1.1 ideas, and ROAD-080 (the in-process event bus) was
brainplanned and is ready to build.

All Go changes are tested (go test ./internal/server/ ./cmd/ + -race), React is
green (vitest 15/15, tsc, vite build), Rust compiles (cargo check). The desktop
toolchain is present: go 1.26, cargo, bun, node.

## Goals

### [x] G-01 BUG-023 — Rust handshake-read loop can exceed HANDSHAKE_TIMEOUT
**Model:** sonnet — bounded fix in one file. In `desktop/src-tauri/src/daemon.rs`
`spawn_and_handshake`, the `while Instant::now() < deadline` loop only re-checks
the deadline between `rx.recv().await` calls; `recv()` has no timeout, so a
slow/stalling sidecar makes the wait unbounded. Wrap the recv in
`tokio::time::timeout(deadline - Instant::now(), rx.recv())` (tokio `time`
feature already enabled); on elapsed, kill the child and return HandshakeTimeout.
Verify with `cargo check --manifest-path desktop/src-tauri/Cargo.toml` (build the
host sidecar first: `bash desktop/scripts/build-sidecar.sh --host`).

### [x] G-02 BUG-024 — scope the Tauri shell capability to the sidecar
**Model:** sonnet — needs the exact Tauri v2 shell-sidecar permission syntax.
`desktop/src-tauri/capabilities/default.json` grants broad `shell:default`;
replace it with a permission scoped to only the `cosmoflare` sidecar
(allow-execute / sidecar permission with the specific name+args), granting only
what the daemon spawn needs. Re-verify the sidecar still spawns. Security-relevant
for the paid build.

### [x] G-03 BUG-022 — serve graceful Shutdown can hang with an SSE client
**Model:** sonnet — touches `cmd/serve.go` + `internal/server/sse.go`.
`httpServer.Shutdown(context.Background())` waits indefinitely for the long-lived
SSE handler. Pass a bounded context (e.g. 5s) to Shutdown AND/OR have the SSE
handlers select on a server-wide done-channel closed at shutdown so streams exit
promptly. Add/extend a test in `internal/server`. Verify with
`GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ ./cmd/`.

### [ ] G-04 ROAD-080 / FEAT-008 — in-process event bus (real-time notifications)
**Model:** Opus-orchestrated (the build itself dispatches GLM/sonnet per sub-goal).
Run `/run-continuation docs/prompts/2026-06-21-inprocess-event-bus.md`. That
three-tier prompt has its own G-01..G-04 (events.Bus → webhook.Manager publish →
Server.SubscribeBus → cmd/serve.go wiring), each with dispatch models and TDD
code. Dependency order is G-01 first, then G-02∥G-03, then G-04. Opus reviews every
diff + re-runs tests before merge (S334).

### [ ] G-05 IDEA-037 — live metrics producer for the SSE metrics channel (v1.1)
**Model:** needs design (/brainplan). The dashboard's `metrics` SSE channel has no
producer, so cards go stale until an account switch. Either a daemon metrics poll
loop that `Publish("metrics", deltas)` every N s, or a `refetchInterval` on the
dashboard's useQuery calls. Brainplan before building.

### [ ] G-06 IDEA-038 — Rust watchdog mid-session restart (v1.1)
**Model:** needs design (/brainplan). The Rust daemon watchdog only covers initial
spawn (3× backoff), not a daemon that dies after going ready. Keep the rx-drain
task alive after readiness and re-spawn on `CommandEvent::Terminated`. Couples
with the now-fixed React SSE token re-resolution (sse.ts) for full recovery.

## Recommended order

1. **Quick bug fixes first** — G-01, G-02, G-03 (bounded, one or two files each).
2. **Then the main feature** — G-04 (ROAD-080 event bus) via /run-continuation.
3. **Then v1.1 polish** — G-05, G-06 (each needs its own brainplan first).

## Carry-Over / Notes

- Smoke test `Docker build` step failed last session — likely environmental
  (Docker daemon not running); verify before treating as a real bug.
- `ccs workcheck` flagged possible `docs/USAGE.md` gaps for the new `desktop/`
  dirs — check if USAGE needs a desktop section.
- Leftover dir `~/PROJECTS/cosmoflare-worktrees/TauriApp/` (220M, de-registered
  from git, work fully merged) can be trashed via Finder when convenient.

## Related

- Open issues: BUG-022, BUG-023, BUG-024, FEAT-008
- Roadmap: ROAD-080 (event bus), IDEA-037, IDEA-038
- Build prompt: `docs/prompts/2026-06-21-inprocess-event-bus.md`
