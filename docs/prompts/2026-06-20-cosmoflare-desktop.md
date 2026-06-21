---
brainstorm_ref: docs/brainstorming/2026-06-20-cosmoflare-desktop.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
    - BR-07
    - BR-08
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
    - P-09
created: "2026-06-20"
id: P-2026-06-20-cosmoflare-desktop
plan_ref: docs/planning-mode/2026-06-20-cosmoflare-desktop.md
glm_tasks_ref: docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml
priority: high
goals_total: 10
goals_completed: 0
requires_reading:
    - docs/brainstorming/2026-06-20-cosmoflare-desktop.md
    - docs/planning-mode/2026-06-20-cosmoflare-desktop.md
schema_version: 1
status: PENDING
title: Cosmoflare Desktop (Tauri) — v1 Implementation
---
# Cosmoflare Desktop (Tauri) — v1 Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-06-20-cosmoflare-desktop.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-06-20-cosmoflare-desktop.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

v1 of the Cosmoflare desktop app (ROAD-063): a Tauri shell that spawns and supervises a local `cosmoflare serve` daemon (HTTP+SSE over the existing Go services) and renders a read-only multi-account dashboard with real-time notifications. The Rust shell owns the daemon lifecycle so the same design runs on macOS/Windows/Linux. Credentials come from the config flow — **never the OS keychain** (`COSMOFLARE_NO_KEYCHAIN=1`). Graph + write/CRUD are deferred to v2.

Design: `docs/brainstorming/2026-06-20-cosmoflare-desktop.md`
Plan: `docs/planning-mode/2026-06-20-cosmoflare-desktop.md`
GLM manifest: `docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml`

## Execution Strategy

Build via a GLM session or glm-tree following the plan's wave structure. **Dependency order matters:**

1. **Wave 1 — Go daemon (P-01 → P-02 → P-03), SEQUENTIAL.** All three tasks touch `internal/server/` + `cmd/serve.go`, so they must run one after another (not the same batch). This wave fixes the REST/SSE API contract.
2. **Wave 2 — Rust lifecycle (P-04 → P-05)** and **Wave 3 — React UI (P-06 → P-07 → P-08)** run **in parallel** once Wave 1 lands — they share only the (now-fixed) API contract, no files. Within each wave, tasks are sequential.
3. **Wave 4 — packaging (P-09)** last.

```yaml
dispatch:
  - wave: 1   # sequential — same package
    tasks: [P-01, P-02, P-03]
    engine: glm   # ccs glm-agent exec-batch (wave-size 1)
  - wave: 2   # parallel with wave 3
    tasks: [P-04, P-05]
  - wave: 3
    tasks: [P-06, P-07, P-08]
  - wave: 4
    tasks: [P-09]
```

Manifest: `ccs glm-agent exec-batch docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml --wave-size 1` for Wave 1, then larger wave-size for Waves 2/3. **Opus reviews every agent diff + re-runs tests before merge** (S334 gate). Architecture is settled in the design doc — agents implement, they do not redesign.

## File Scope

- **Go daemon:** `cmd/serve.go`, `internal/server/{server,rest,sse}.go` (+ `_test.go`)
- **Tauri/Rust:** `desktop/src-tauri/` (`tauri.conf.json`, `Cargo.toml`, `src/{main,daemon}.rs`)
- **React:** `desktop/src/` (`App.tsx`, `api/{client,sse}.ts`, `components/`, `views/`, `__tests__/`)
- **Packaging:** `desktop/scripts/build-sidecar.sh`, `tauri.conf.json` build hooks

## Goals

### [ ] G-01 internal/server + `cosmoflare serve`: HTTP server, token auth, stdout handshake, /healthz
Covers P-01.

### [ ] G-02 REST read endpoints (/accounts /zones /r2/buckets /workers /kv) over existing services
Covers P-02.

### [ ] G-03 SSE /events (metrics/notifications/status) + two-tier health (systems/cloudflare online)
Covers P-03.

### [ ] G-04 Tauri v2 scaffold in desktop/ + sidecar (externalBin) + build config
Covers P-04.

### [ ] G-05 Rust daemon lifecycle: spawn, handshake parse, health-poll state machine, kill, watchdog
Covers P-05.

### [ ] G-06 React+Vite+TS app shell + REST/SSE client hooks + two health indicators
Covers P-06.

### [ ] G-07 Multi-account read-only dashboard (zones/R2/Workers/KV cards)
Covers P-07.

### [ ] G-08 Real-time notifications panel (SSE notifications channel)
Covers P-08.

### [ ] G-09 Cross-platform packaging: per-triple Go sidecar + Tauri bundles
Covers P-09.

## Related

- Brainstorm: `docs/brainstorming/2026-06-20-cosmoflare-desktop.md`
- Plan: `docs/planning-mode/2026-06-20-cosmoflare-desktop.md`
