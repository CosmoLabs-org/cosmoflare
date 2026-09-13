# Architecture Map — cosmoflare v0.26.0

Synthesized from 13 audit agents (2026-09-13). Every claim cites its source agent report.

## System Layers

Four-layer library-first architecture, verified by Code Quality and API Design agents:

1. **CLI layer** — `cmd/` (40+ cobra command files). Workflow commands (dev, init, diff, apply, sync, watch, cost, export/import, terraform, mcp) + per-service groups. Layer is thick: 618 inline JSON-output branches, two competing error-flow idioms (see agent-1-code-quality.md).
2. **Public library** — `pkg/cosmoflare/` (148 files): one `*Service` struct per Cloudflare service (R2, Workers, KV, DNS, SSL, D1, Pages, Queues, Images, Hyperdrive, Vectorize, AI, Stream, Healthchecks, Domains, Redirect, Registrar + more), shared `types.go`/`errors.go`/`options.go`/`config.go`. Importable as `github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare` (see agent-1-code-quality.md).
3. **Internal packages** — `internal/`: `server/` (serve daemon: REST + SSE + token auth), `tui/` (bubbletea UI), `cli/`, `config/`, `interactive/`, `utils/`, `webhook/`. Daemon exposes `ServeSource` methods returning untyped `any` payloads — the weakest boundary in the system (see agent-7-api-design.md).
4. **Desktop tier** — `desktop/` (Tauri + React/TS, `src/api/client.ts` + `sse.ts` consuming the daemon; `styles.css` token-driven design system). Sidecar ships the CLI binary; version skew 0.16.0 vs CLI 0.26.0 (see agent-5-distribution.md, agent-17-design-taste.md).

## Module Boundaries

- **cmd ↔ pkg/cosmoflare**: clean dependency direction (CLI wraps library; no library imports of cmd). Shared `getAPIClient` factory lives inside a domain file `cmd/object.go:1102` — misplaced (see agent-1-code-quality.md).
- **daemon ↔ desktop**: REST + SSE over localhost with bearer-token auth. Contract is untyped end-to-end: `ServeSource` returns `any` (internal/server/rest.go:24), TS client casts unchecked (see agent-7-api-design.md). No 405 enforcement — handlers accept any verb (internal/server/rest.go:48).
- **three HTTP transports coexist**: R2/S3 SDK client, `restClient` (Retry-After-aware), and the cloudflare-go path via 27 `NewXServiceFromCreds` constructors — each with its own timeout/retry policy; 27 constructors fall back to `http.DefaultClient` (no timeout) while the S3 path shares a 30s timeout that is wrong for large transfers (see agent-7-api-design.md, agent-2-core-logic.md).

## Data Flow

**Sync pipeline** (flagship, riskiest): `cmd/sync.go` → `SyncService` (pkg/cosmoflare/sync.go) → `planUp/planDown` → `Execute` loop (strictly sequential, sync.go:408) → R2 client (storage.go, upload.go, multipart.go). Delete handling is direction-blind: `Execute` always calls `DeleteRemoteObject` with an unprefixed key (sync.go:425) — the audit's most dangerous defect (see agent-2-core-logic.md).

**Serve/daemon flow**: `cmd/serve.go` → `internal/server` → per-source pollers (limits snapshot wired at serve.go:238, 30s cadence) → SSE fan-out to desktop. Desktop renders via always-live SSE connection (see agent-15-work-completion.md re FEAT-014 premise, agent-18-accessibility.md re live regions).

**Config flow**: `.cosmoflare.yaml` (project) + machine config (viper, `pkg/cosmoflare/config.go`) + OS keychain with plaintext fallback (internal/config/config.go:381). Machine config tokens written 0644-then-chmod — TOCTOU window (config.go:169) (see agent-2-core-logic.md, agent-7-api-design.md).

## Dependency Graph

- No circular deps found; layering is genuinely clean (see agent-1-code-quality.md).
- `retry.Do` exported helper (pkg/cosmoflare/retry.go:68) is unwired dead code — `restClient` implements its own superior retry (see agent-7-api-design.md).
- `internal/tui/update.go` `handleKeyMsg` (238 lines) is the coupling hotspot — god function with per-key-path branching (see agent-1-code-quality.md).
- Desktop depends on daemon contract with zero codegen — drift caught only at runtime (see agent-7-api-design.md).

## Agent Cross-References

| Finding | Source Agent | File:Line |
|---------|-------------|-----------|
| Direction-blind sync deletes | agent-2-core-logic.md | pkg/cosmoflare/sync.go:425 |
| Untyped daemon wire contract | agent-7-api-design.md | internal/server/rest.go:24 |
| 27 timeout-less constructors | agent-7-api-design.md | pkg/cosmoflare/kv.go:93 |
| 618 JSON-output branches | agent-1-code-quality.md | cmd/object.go:483 |
| Token 0644→chmod TOCTOU | agent-2-core-logic.md, agent-7-api-design.md | pkg/cosmoflare/config.go:164-169 |
| Desktop/CLI version skew | agent-5-distribution.md | desktop/src-tauri/tauri.conf.json:4 |
| Limits snapshot wired (FEAT-014 premise outdated) | agent-15-work-completion.md | cmd/serve.go:238 |
