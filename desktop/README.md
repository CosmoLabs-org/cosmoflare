# Cosmoflare Desktop

The desktop tier of Cosmoflare — a cross-platform (macOS / Windows / Linux) GUI
dashboard built with **Tauri v2**. It is the paid Desktop product in Cosmoflare's
3-tier model (CLI → Desktop → Mobile); the open-source CLI and this desktop app
both wrap the **same** `pkg/cosmoflare` Go core, so there is no per-tier
reimplementation of Cloudflare logic.

**v1 scope:** a read-only multi-account dashboard (Zones / R2 / Workers / KV)
with two-tier health and real-time notifications. The infrastructure graph and
write/CRUD operations are deferred to v2.

## Architecture

```
┌─────────────────── Tauri app (one process tree) ───────────────────┐
│  Rust shell (lifecycle owner)          React + Vite webview (UI)    │
│  ├─ spawns cosmoflare sidecar  ──┐     ├─ REST calls ──┐            │
│  └─ health-checks + kills on exit│     └─ SSE subscribe │            │
└──────────────────────────────────┼──────────────────────┼──────────┘
                                   ▼                      ▼
                   127.0.0.1:<ephemeral>  ◄── cosmoflare serve (Go daemon)
                                                     │ reuses pkg/cosmoflare services
                                                     ▼
                                             Cloudflare API
```

The app adds **zero new Cloudflare logic**. Three cleanly separable layers, each
testable in isolation against the fixed REST/SSE contract:

| Layer | Lives in | Role |
|-------|----------|------|
| **Go daemon** (`cosmoflare serve`) | `cmd/serve.go`, `internal/server/` | Thin HTTP+SSE transport over the existing per-service constructors |
| **Rust lifecycle shell** | `desktop/src-tauri/src/` | Spawns the sidecar, parses the handshake, health-polls, kills on exit, watchdog |
| **React webview** | `desktop/src/` | Dashboard + notifications; talks to the daemon over REST (reads) + SSE (live) |

The Go binary is bundled as a Tauri `externalBin` **sidecar**, cross-compiled per
target-triple so the same design runs on every OS with no platform-specific app
code.

## Prerequisites

- **Go 1.26+** (the daemon / sidecar)
- **Rust** (stable; `rustup`) — the Tauri shell
- **bun** (or npm/node) — the frontend

## Development

All commands are exposed as Make targets from the repo root:

```bash
make desktop-dev       # build the host sidecar, then `tauri dev` (hot-reload)
make desktop-build     # frontend build + all-triple sidecars + Tauri bundle
make desktop-test      # JS (vitest) + Rust (cargo test)
make desktop-sidecar   # cross-compile the Go daemon for all 4 triples
```

Or, from `desktop/` directly:

```bash
bun install            # first time
bun run tauri dev      # iterative dev (window launches, daemon spawns)
bun run test           # vitest
cargo test --manifest-path src-tauri/Cargo.toml   # Rust unit + integration
```

`tauri dev` rebuilds the **host** sidecar each launch so it matches the current
Go code. The daemon prints a one-line JSON handshake to stdout
(`{"addr":"127.0.0.1:<port>","token":"<random>"}`); the Rust shell parses it,
health-polls `/healthz`, and exposes the endpoint to the webview via the
`daemon_endpoint` Tauri command.

## Project structure

```
desktop/
├── src/                       # React + Vite + TypeScript webview
│   ├── api/                   # client.ts (REST), sse.ts (SSE hook)
│   ├── components/            # Header (account switcher + health dots)
│   ├── views/                 # Dashboard, Notifications
│   └── __tests__/             # vitest + @testing-library
├── src-tauri/                 # Rust (Tauri v2)
│   ├── src/{main,lib,daemon}.rs   # lifecycle: spawn / handshake / health / kill
│   ├── capabilities/          # shell permission scope (sidecar)
│   ├── tauri.conf.json        # externalBin sidecar + build hooks
│   └── binaries/              # built sidecars (gitignored)
└── scripts/
    └── build-sidecar.sh       # per-triple Go cross-compile (CGO_ENABLED=0)
```

## The daemon contract

`cosmoflare serve` exposes a token-gated localhost API (see `docs/USAGE.md` →
"Desktop Daemon" for the full reference):

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` | Two-tier health: `{systems_online, cloudflare_online, version}` |
| `GET` | `/accounts` | Local config profiles (not a Cloudflare call) |
| `GET` | `/zones`, `/r2/buckets`, `/workers`, `/kv` | Service lists; `?profile=<name>` re-scopes per request (read-only) |
| `GET` | `/events` | SSE: multiplexed `metrics` / `notifications` / `status` |

Auth is `Authorization: Bearer <token>` everywhere; the SSE endpoint also accepts
`?token=` because the browser's `EventSource` cannot set headers.

**Credentials:** the daemon resolves creds lazily per request from the selected
config profile, starts with no valid creds (reports `cloudflare_online=false` →
first-run setup screen), and **never probes the OS keychain**
(`COSMOFLARE_NO_KEYCHAIN=1`).

## Testing

- **Go daemon** — `httptest`-backed handler tests (`internal/server/`), full
  suite via `go test ./...`.
- **Rust** — `parse_handshake` / `backoff` / `wait_until_ready` unit tests, plus
  integration tests that spawn the **real** Go sidecar and exercise the full
  lifecycle (handshake → health → auth).
- **Frontend** — vitest + @testing-library/react; components tested with a
  stubbed client, the SSE hook with a mocked `EventSource`.

## Further reading

- **Design rationale** — `docs/brainstorming/2026-06-20-cosmoflare-desktop.md`
- **Implementation plan** — `docs/planning-mode/2026-06-20-cosmoflare-desktop.md`
- **CLI usage (incl. `serve`)** — `docs/USAGE.md`
- **Product vision (3-tier)** — `docs/PRODUCT-VISION.md`

Roadmap: **ROAD-063** (this app), **ROAD-079** (the `serve` daemon), **ROAD-081**
(code-signing / notarization / auto-update — v2).
