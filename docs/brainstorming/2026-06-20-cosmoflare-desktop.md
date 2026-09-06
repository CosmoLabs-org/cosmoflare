---
created: "2026-06-20T17:44:43-03:00"
deliverables:
    - id: BR-01
      title: cosmoflare serve — local HTTP+SSE daemon over existing services
    - id: BR-02
      title: Rust daemon lifecycle (spawn/handshake/health/kill/watchdog) + sidecar bundling
    - id: BR-03
      title: 'Two-tier health model: Systems online + Cloudflare online (/healthz + SSE status)'
    - id: BR-04
      title: Credentials via config/.cosmoflare.yaml + first-run setup — never the OS keychain
    - id: BR-05
      title: React + Vite + TS app shell (header w/ account switcher + 2 health dots, sidebar, main pane)
    - id: BR-06
      title: Multi-account read-only dashboard (zones / R2 / Workers / KV live cards)
    - id: BR-07
      title: Real-time notifications panel (SSE from the alerts/webhook system)
    - id: BR-08
      title: Cross-platform packaging (Tauri v2 installers + per-target Go sidecar binary)
last_review_content_hash: 9cd91801345e238f0cf1017cde7a9df64f58f7d61965185cad59bfc0047d2f1b
last_review_findings: 0
last_review_ref: docs/brainstorming/2026-06-20-cosmoflare-desktop.md
last_reviewed: "2026-09-06T20:17:20.214377+04:00"
origin: /brainplan
plan_ref: docs/planning-mode/2026-06-20-cosmoflare-desktop.md
priority: high
roadmap_ref: ROAD-063
status: APPROVED
tags:
    - desktop
    - tauri
    - daemon
    - react
    - notifications
    - dashboard
title: Cosmoflare Desktop (Tauri) — v1 Design
updated: "2026-06-20T17:44:43-03:00"
---

# Cosmoflare Desktop (Tauri) — v1 Design

## Context

ROAD-063 calls for a cross-platform desktop GUI for Cosmoflare (the paid Desktop
tier of the 3-tier product) with real-time notifications and an infrastructure
graph. This document designs **v1**: a daemon + read-only multi-account dashboard
+ real-time notifications. The infrastructure graph and write/CRUD operations are
explicitly deferred to v2+.

The guiding product principle is **library-first**: the CLI, desktop, and mobile
tiers all wrap the same `pkg/cosmoflare` core — there are no per-tier
reimplementations of Cloudflare logic. The desktop app therefore introduces a
thin transport (`cosmoflare serve`) over the existing service packages, nothing
more.

## Decisions (from the brainstorm Q&A)

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | **Go ↔ Tauri bridge = local daemon** (`cosmoflare serve`, HTTP + SSE on localhost) | Real-time notifications and the (future) live graph need a persistent push channel; a daemon gives streaming and keeps the Go core as the single source of truth. Extends the existing `cosmoflare mcp` server pattern. Chosen over a sidecar-CLI (polling only, per-action process spawn) and CGo/FFI (fragile cross-platform builds). |
| 2 | **Frontend = React + Vite + TypeScript** | Matches the repo's existing Vercel React skill set and allows component/logic sharing with the planned React Native mobile tier (ROAD-064); largest ecosystem for charts/graphs. |
| 3 | **v1 scope = daemon + dashboard + notifications** | Smallest slice that proves the architecture end-to-end. Graph + CRUD deferred to v2. |
| 4 | **Tauri (Rust) owns the daemon lifecycle** | The Rust shell spawns the daemon on launch, health-checks it, and kills it on exit. The Go binary is a per-OS bundled sidecar — so the same design works on macOS, Windows, and Linux with zero platform-specific app logic. |

## Architecture

```
┌─────────────────── Tauri app (one process tree) ───────────────────┐
│  Rust shell (lifecycle owner)          React + Vite webview (UI)    │
│  ├─ spawns cosmoflare sidecar  ──┐     ├─ REST calls ──┐            │
│  └─ health-checks + kills on exit│     └─ SSE subscribe │            │
└──────────────────────────────────┼──────────────────────┼──────────┘
                                    ▼                      ▼
                    127.0.0.1:<ephemeral>  ◄── cosmoflare serve (Go daemon)
                                                      │ reuses existing services
                                                      ▼  (Zone/Worker/KV services, R2 client, config profiles)
                                              Cloudflare API
```

The desktop app adds **zero new Cloudflare logic**. `cosmoflare serve` is an HTTP/SSE
layer that delegates to the existing per-service constructors (`NewZoneServiceFromCreds`,
`NewWorkerServiceFromCreds`, `NewKVServiceFromCreds`, `NewClient` for R2) and the
config profile list — exposed through a small daemon-local `ServeSource` interface.
(Note: `internal/tui.DataSource` is R2-storage-specific and is NOT reused here.)

## Components & interfaces

### BR-01 — `cosmoflare serve` daemon (`cmd/serve.go`, `internal/server/`)
A new CLI command starts an HTTP server bound to `127.0.0.1:<port>` (port `0` →
OS picks an ephemeral port). On startup it prints a one-line JSON handshake to
stdout — `{"addr":"127.0.0.1:54123","token":"<random>"}` — which the Rust shell
parses. All endpoints require `Authorization: Bearer <token>`. The server reuses
the existing services; it owns no Cloudflare logic of its own.

**v1 endpoints (read-only):**
- `GET /healthz` → `{systems_online: true, cloudflare_online: bool, version}`
- `GET /accounts`, `GET /zones`, `GET /r2/buckets`, `GET /workers`, `GET /kv`
  — each delegates to the existing service / `DataSource`, returns the existing
  `--json` shapes.
- `GET /events` (SSE) → multiplexed channels: `metrics` (dashboard deltas),
  `notifications` (from alerts/webhook), `status` (health transitions).

### BR-02 — Daemon lifecycle (Rust, `desktop/src-tauri/`)
On app launch the Rust backend spawns the bundled `cosmoflare` binary as a Tauri
**sidecar** (`externalBin`): `cosmoflare serve --addr 127.0.0.1:0 --token <random>`,
with `COSMOFLARE_NO_KEYCHAIN=1` set in the child's environment (the env var is the
only keychain gate — there is no `--no-keychain` flag). It reads the stdout handshake for the port, polls `/healthz` until
ready (timeout + backoff), then exposes the URL+token to the webview via a Tauri
command. On window-close / app-quit it terminates the child. A watchdog restarts
the daemon on crash (3× exponential backoff, then surface an error screen).
Single-instance enforced.

### BR-03 — Two-tier health model
Two independent signals, both rendered as indicators in the app header:
- **Systems online** — daemon process up and `/healthz` reachable.
- **Cloudflare online** — the daemon's most recent Cloudflare API call succeeded
  (credentials valid + API reachable).

Pushed live over the SSE `status` channel and also available via `/healthz`
polling as a fallback. `cloudflare_online == false` with a healthy daemon routes
the user to the credentials setup screen (BR-04).

### BR-04 — Credentials (hard constraint)
The daemon reads Cloudflare credentials from the existing config file
(`.cosmoflare.yaml`) / the project credentials flow. It runs with
`COSMOFLARE_NO_KEYCHAIN=1` and **must never probe the macOS keychain** — doing so
spammed reset dialogs in a prior session (see the keychain fix in
`internal/keychain/keychain.go`). First run with no credentials → the app shows a
setup screen that writes the config file.

### BR-05 — React app shell (`desktop/src/`)
App shell: header (account switcher + the two health dots) → sidebar (services) →
main content pane. Server state via React Query against the daemon's REST API; a
small SSE client hook feeds live updates. The REST/SSE JSON shapes ARE the API
contract (reused from the CLI's `--json`).

### BR-06 — Multi-account read-only dashboard
Live cards for zones, R2 buckets, Workers, and KV, mirroring the terminal
dashboard's layout. (The daemon serves these via its own `ServeSource` interface
over the per-service constructors — see the plan; `internal/tui.DataSource` is
R2-only and is NOT the source.) **Account model (read-only):** an "account" is a
local config profile (`ConfigManager.ListProfiles()`); the header switcher selects
one and the frontend passes `?profile=<name>` on every REST call, so the daemon
re-resolves creds per request. No write-side profile "switch" — that keeps v1
strictly read-only. Single profile → switcher is a static label.

### BR-07 — Real-time notifications panel
A panel fed by the SSE `notifications` channel, with a scrollable history and
unread badge. **Source (v1):** the existing `internal/webhook` system is
*outbound-only* (it POSTs to Slack/Discord/HTTP; no subscribe/event-bus to tap),
so v1 notifications come from **daemon-internal events** the daemon already
observes — `cloudflare_online` transitions, per-poll errors/recoveries, and any
alert-rule evaluations the daemon runs against its polled metrics. Sourcing from
the webhook sender is deferred to v2 (needs an in-process pub/sub seam added to
`internal/webhook`).

### BR-08 — Cross-platform packaging
Tauri v2 produces per-OS installers: `.dmg`/`.app` (macOS), `.msi`/`.exe`
(Windows), `.deb`/`.AppImage` (Linux). The Go daemon is cross-compiled per
target-triple and shipped as a Tauri `externalBin` sidecar named
`cosmoflare-<target-triple>`. Repo layout: the Tauri project lives in **`desktop/`**
at the repo root; the Go `serve` command in `cmd/serve.go` + `internal/server/`.

## Data flow
1. App launch → Rust spawns daemon → handshake → `/healthz` ready → webview gets URL+token.
2. Webview opens SSE `/events`; subscribes to `metrics`, `notifications`, `status`.
3. Dashboard cards hydrate from REST, then update from `metrics` SSE deltas.
4. Header health dots update from the `status` SSE channel.
5. App quit → Rust kills the daemon.

## Error handling
- Daemon spawn failure → error screen + manual retry.
- Handshake / health timeout → backoff retry; surface after N attempts.
- CF auth failure → `cloudflare_online=false` + setup prompt (not a hard crash).
- Daemon crash → Rust watchdog restarts (3× backoff), then error screen.
- SSE disconnect → client auto-reconnects with backoff.

## Testing
- **Go daemon** — `httptest`-backed handler tests (auth required, JSON shapes,
  SSE framing); reuse existing service tests (in-memory, no keychain).
- **Rust** — lifecycle unit tests (handshake parse, health-poll state machine,
  child termination).
- **Frontend** — Vitest component tests with mocked REST + SSE; health-indicator
  and notifications-panel behavior.

## Scope boundary (YAGNI)
**In v1:** daemon + Rust lifecycle + two-tier health + read-only multi-account
dashboard + real-time notifications + cross-platform packaging scaffold.

**Deferred to v2+:** infrastructure graph visualization, write/CRUD operations,
auto-update, code-signing / notarization, licensing & paid-tier auth.

## Execution note
Three cleanly separable work units with a well-defined seam (the REST/SSE
contract): the Go daemon (BR-01/03/04), the Rust lifecycle shell (BR-02/08), and
the React frontend (BR-05/06/07). Suitable for parallel GLM dispatch once the API
contract (BR-01) is fixed first.
