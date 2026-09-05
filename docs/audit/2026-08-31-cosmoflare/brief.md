# Cosmoflare Project Brief

> For future sessions: load this instead of re-auditing. Source: 2026-08-31 audit (13 agents). Fresh until ~v0.19.0 or major architecture change.

## What It Is

Go CLI + importable library (`pkg/cosmoflare/`) + Tauri desktop tier for the full Cloudflare platform. 23 services, 366 cobra commands. MIT, v0.17.0. 3-tier product vision: CLI (free OSS) → Desktop (paid) → Mobile (subscription). Module: `github.com/CosmoLabs-org/cosmoflare`.

## Tech Stack (actual)

Go 1.26 / cobra / viper / cloudflare-go v0.116 / AWS SDK v2 (R2 via S3 API) / bubbletea+lipgloss TUIs / Tauri 2 + React 18 (NOTE: profile mandates React 19 + Tailwind 4 + shadcn + Zustand — desktop diverges) / Vitest. Build: Makefile + 3 GitHub workflows.

## Architecture (from architecture.md)

```
cmd/ (366 commands, --json envelope)
  ├─ pkg/cosmoflare/ — public library, 23 *Service clients, typed R2Error hierarchy
  │    └─ cloudflare-go + AWS SDK, retry.go (backoff+jitter, dual-SDK)
  ├─ internal/server — serve daemon: REST (6 GET endpoints) + SSE (/events), bearer auth
  ├─ internal/{config,keychain,webhook,interactive,cli,utils} — keychain UNWIRED to profiles
  └─ desktop/ — Tauri shell, Rust sidecar spawns daemon, SSE → React Query
```

Disciplined at the library layer (identical service template, behavioral httptest tests). Weak at transport edges (all daemon errors → 502; `--json` silent-exits on config errors).

## The Five Things That Matter Most (2026-08-31)

1. **Storage engine has silent-corruption paths** — multipart short-read accepted as success (`upload.go:207`); sync unpaginated (`cmd/sync.go:428` — `--delete` can destroy local files); watcher deletes on scan errors. Fix before trusting sync/watch with real data.
2. **The r2go2→cosmoflare rename never finished** — dead module path in ldflags (binaries report `dev`), installers 404, `r2go2:` in public library errors, `.r2go2.yaml` config. One sweep fixes release tooling.
3. **Zero distribution** — repo private, 0 GitHub Releases (18/18 failed release runs since March; root cause: `internal/interactive` races fail the `-race` gate), no brew/scoop, README undersells 13 shipped services as "Planned".
4. **Competitive pivot needed** — Cloudflare shipped `cf` CLI + 2,500-endpoint official MCP (April 2026). Surviving differentiators: agent trust tier (guardrails+audit — currently dead code), pure-Go single binary, neutrality, ops bundle (cost/alerts/doctor).
5. **Desktop tier is one focused week from credible** — semantic skeleton is right, data layer works; missing: the `cf-*` stylesheet (30+ classes referenced, 15-line CSS), 6 ARIA fixes, 3-OS sidecar builds (currently macOS-arm64-only committed).

## Known Patterns (from patterns.md)

- Extend the **httptest mock pattern** (`cmd/kv_test.go`) for new command tests — proven, behavioral.
- 489 `if JSONOutput` branches — introduce `reportError`/`reportResult` helpers before touching output contracts.
- New services: copy `kv.go` template (struct + `NewXService` + typed errors).
- Never auto-trigger macOS keychain probing (project memory — caused reset-dialog flood 2026-06-20); keychain must stay opt-in.

## Where NOT to Touch Without Tests (from risk-map.md)

`pkg/cosmoflare/upload.go` + `multipart.go` (corruption invariants being added), `cmd/sync.go` (deletion logic), `internal/interactive` (races), `internal/server/metrics.go` (one source error currently silences all — known issue FEAT-008 adjacent), `Makefile` ldflags (SSOT for version injection being introduced).

## Open Work Snapshot

- **FEAT-008** (only open issue, 71 days): event bus — code-verified absent; re-scope to "bridge `TriggerAlert` → existing sseHub" (the hub shipped after the plan was written)
- 19 high-severity audit findings (see action-plan.md 🔴 Now tier)
- 7 doc chains need `plan_ref` back-links; 24 design docs unreviewed
- Roadmap: dedupe ROAD-074/081/082/083, regenerate index.yaml, audit 9 BASE items

## Conventions

Conventional commits (100/100 last 50) · tests alongside sources · every command: `--json` + rich `--help` · single `.cosmoflare.yaml` project config (aspirationally — see rename debt) · three-tier doc chain (ADR-005) via ccs tools · ISO8601+TZ timestamps constitutional.

## Load Order for Deep Dives

risk-map.md → action-plan.md → surprises.md → the specific agent-*.md for any dimension.
