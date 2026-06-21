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
goals_completed: 1
requires_reading:
    - docs/brainstorming/2026-06-20-cosmoflare-desktop.md
    - docs/planning-mode/2026-06-20-cosmoflare-desktop.md
schema_version: 1
status: PENDING
title: Cosmoflare Desktop (Tauri) — v1 Implementation
---
# Cosmoflare Desktop (Tauri) — v1 Implementation

## ✅ BLOCKER RESOLVED — GLM Pool Fixed (2026-06-20)

G-00 is done. Root cause: the `ccsdaemon` hosting the Conductor had been running since Jun 16 with stale in-memory code predating the BUG-582 `[1m]` normalization (`conductor.go:107` `glmconfig.ResolveQueueModel`). The on-disk binary already had the fix; **restarting the daemon** (`ccs daemon restart`) reloaded it. Verified: `glm-5.2[1m]` now resolves to the `glm-5.2` pool and dispatch succeeds. **GLM dispatch is unblocked — start at G-01.**

---

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

## GLM Dispatch Rules

When goals involve dispatching subagents:

1. **ALWAYS** use `ccs glm-agent exec` / `exec-batch` for GLM agents (routes through queue with retry logic)
2. **NEVER** use Agent tool with `model:sonnet` or `model:haiku` for GLM work (bypasses queue, risks 429)
3. Agent tool with `model:opus` is fine for Opus subagents
4. For Wave 1 parallel work: `ccs glm-agent exec-batch docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml --wave-size 1`

## Goals

### [x] G-00 Fix GLM pool config: `unknown pool: glm-5.2[1m]` — ✅ RESOLVED 2026-06-20
**Done:** stale `ccsdaemon` (running since Jun 16) served the Conductor with pre-BUG-582 code; `ccs daemon restart` reloaded the binary that normalizes `glm-5.2[1m]`→`glm-5.2`. Verified by a clean dispatch.

The error `unknown pool: glm-5.2[1m]` comes from `conductor.go:107` (`AcquireContext`). The normalization that should strip `[1m]` lives in `tools/ccsession/internal/glmqueue/daemon.go:325` (`resolveModel`), which is called at line 421 before the conductor acquire. If the running binary predates that wiring, the daemon passes the raw `glm-5.2[1m]` string directly to the conductor, which has no such pool.

**Diagnosis + fix sequence (in order — stop at whichever step resolves it):**

1. **Rebuild ccs binary** in ClaudeCodeSetup:
   ```bash
   ccs build
   # or: go -C /Users/gabstudio/PROJECTS/ClaudeCodeSetup/tools/ccsession build -o ~/.local/bin/ccs .
   ```
2. **Restart the GLM queue daemon** (so it picks up the new binary):
   ```bash
   ccs glm-queue restart   # or: ccs daemon restart
   ```
3. **Smoke test**: `ccs glm-agent exec "echo hello" --model glm-5.2 --max-turns 1 --skip-validate` — should not error with `unknown pool`.
4. **If still failing**: search for any caller that passes the raw model string directly to `conductor.Acquire` without going through `resolveModel`:
   ```bash
   grep -n "Acquire\|conductor" /Users/gabstudio/PROJECTS/ClaudeCodeSetup/tools/ccsession/internal/glmqueue/daemon.go
   grep -rn "\.Acquire(" /Users/gabstudio/PROJECTS/ClaudeCodeSetup/tools/ccsession/ --include="*.go"
   ```
   If a second code path bypasses `resolveModel`, add the same `[` stripping logic there (mirror line 329-331 of `daemon.go`).
5. **Acceptance**: `ccs glm-agent exec-batch docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml --wave-size 1 --dry-run` (or equivalent) exits 0 without `unknown pool` error.

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

## Where We're Headed

The Cosmoflare Desktop is the bridge that turns the CLI into a product people pay for. Once v1 ships (read-only dashboard + real-time alerts), the desktop tier enables the subscription model that funds ongoing open-source CLI work. The GLM pool fix is the gate: Wave 1 (Go daemon) can't run without it, and without Wave 1 none of the Tauri/React waves can start. Fix the pool, run Wave 1, then Opus leads Waves 2-4. FEAT-007 / ROAD-063 tracks this milestone.

## Priority Order

1. **G-00 — GLM pool fix** (blocker for all GLM dispatch — do this first, takes minutes)
2. **G-01 → G-03 — Wave 1 Go daemon** (GLM-dispatched sequentially after G-00; Opus reviews each diff before merge)
3. **G-04/G-05 + G-06/G-07/G-08 — Waves 2 & 3** (run in parallel once Wave 1 lands; Opus-led)
4. **G-09 — Wave 4 packaging** (last; depends on all prior waves)

## Related

- Brainstorm: `docs/brainstorming/2026-06-20-cosmoflare-desktop.md`
- Plan: `docs/planning-mode/2026-06-20-cosmoflare-desktop.md`
- Issue: FEAT-007 | Roadmap: ROAD-063
