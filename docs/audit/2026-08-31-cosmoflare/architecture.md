# Architecture Map — cosmoflare v0.17.0

> Synthesized from 13 agent reports (2026-08-31 audit). Every claim cites its source agent.

## System Layers

Five layers, cleanly separated on the library side; drift concentrates at the transport edges:

| Layer | Location | Role | Verdict |
|-------|----------|------|---------|
| **CLI** | `cmd/` (68 files, 366 cobra commands) | Human + agent entry point, `--json` envelope | Structurally sound, boilerplate-heavy (see agent-1) |
| **Public library** | `pkg/cosmoflare/` (54 files) | Importable Go API: 23 service clients, typed errors, sync engine | The strongest layer — disciplined template per service (agent-1, agent-4) |
| **Internal packages** | `internal/` (config, server, webhook, interactive, tui, cli, utils) | Daemon, keychain, TUIs, retry/batch ops | Mixed: server/webhook near 90-99% test coverage, `cli/operations` at zero (agent-1) |
| **Desktop tier** | `desktop/` (Tauri 2 + React 18 + Rust sidecar) | Paid-tier GUI dashboard over the daemon | Pre-foundation: unstyled shell, one-platform sidecar (agent-17, agent-5) |
| **Orchestration artifacts** | `GOrchestra/` (agent session recovery patches) | Tooling output, not product code | 234MB tracked — repo liability (agent-1, agent-9) |

The library-first principle holds: `internal/server/server.go:1-7` explicitly documents itself as "a thin transport over the existing per-service constructors... owns no Cloudflare logic of its own," and the code matches (agent-1). Each Cloudflare service follows an identical two-file template (struct + `NewXService` with nil guards + typed errors, `kv.go:66-95`), so adding a service is mechanical (agent-1).

## Module Boundaries

**Well-bounded:**
- `pkg/cosmoflare` → `cloudflare-go` v0.116 + AWS SDK (R2 S3 API). `R2Client` interface with 18 methods at `client.go:19` (agent-1).
- `internal/server` → `pkg/cosmoflare` constructors only; REST+SSE transport with bearer auth (agent-7).
- `pkg/cosmoflare/retry.go` — exponential backoff with jitter, context-aware, status-code-aware for both cloudflare-go and AWS error shapes. "Genuinely good resilience code" (agent-7).

**Leaky / drifting:**
- **Config boundary split-brain**: `pkg/cosmoflare/config.go:67` loads `.r2go2.yaml` + `~/.r2go2/config.yaml`, while `pkg/cosmoflare/diff.go:131` loads `.cosmoflare.yaml`. Two names, one tool, contradicts CLAUDE.md's "single .cosmoflare.yaml" contract (agent-1, agent-7).
- **Guardrails boundary exists but unwired**: `GuardrailChecker` (guardrails.go:29-65) has zero production callers — parsed, tested, never enforced (agent-2).
- **MCP surface is a 2% sample**: 8 registered tools vs 366 CLI commands (`mcp.go:286-335`) — the agent channel is not derived from the command tree (agent-4, agent-7).

## Data Flow

**CLI happy path** (traced by agent-2, agent-7):
```
cobra cmd (cmd/*.go)
  → PersistentPreRun token validation (cmd/root.go:99-110)   ⚠ silently exits in --json mode
  → service constructor New*Service (pkg/cosmoflare/*)
  → cloudflare-go / AWS SDK (with retry.go backoff)
  → OutputResponse envelope (--json) or human printer
```

**Daemon path** (agent-7, agent-2):
```
cosmoflare serve (cmd/serve.go)
  → random 128-bit token, 127.0.0.1:0 bind, handshake JSON on stdout (addr+token)
  → internal/server: REST (6 GET endpoints, all errors → 502 ⚠)
  → MetricsProducer poll (internal/server/metrics.go) — ⚠ one failing source silences all
  → sseHub (buffer 32, drop-on-full) → /events SSE
  → desktop React Query bridge (desktop/src/api/sse.ts)
```

**Upload path** (agent-2) — the corruption chain:
```
Upload/MultipartUpload (upload.go)
  → io.ReadFull loop  ⚠ short read accepted as success (upload.go:207)
  → part upload → CompleteMultipartUpload
  → object stored truncated with NO error returned
```

**Sync path** (agent-2): `scanLocalDir` → `planUp/planDown` (sync.go:224-356) → `ListRemoteObjects` ⚠ single-page 1000 (cmd/sync.go:428) → apply ops. `--include` accepted, never read (cmd/sync.go:104).

## Dependency Graph

- `cmd/*` → `pkg/cosmoflare` + `internal/{config,server,interactive,tui,cli}` (no cycles observed; agent-1)
- `pkg/cosmoflare` → `github.com/cloudflare/cloudflare-go v0.116`, AWS SDK v2, no internal deps (clean public boundary; agent-4)
- `internal/server` → `pkg/cosmoflare` (one-way, documented; agent-1)
- `desktop/` → `cosmoflare serve` daemon via HTTP/SSE only (loose coupling, good; agent-7) — but the Rust sidecar shells out to the CLI binary, committed for macOS-arm64 only (agent-5)
- Dead weight: 4 `//go:build disabled` packages (~57KB), 4 `.go.disabled` cmd files, 6 empty internal dirs (agent-1)

## Agent Cross-References

| Finding | Source Agent | File:Line |
|---------|-------------|-----------|
| 3-layer discipline holds; god files in cmd/ | agent-1-code-quality.md | cmd/object.go:483, cmd/installer_tui/main.go:1280 |
| Dual config filenames split contract | agent-1-code-quality.md, agent-7-api-design.md | pkg/cosmoflare/config.go:67 |
| Multipart short-read silent corruption | agent-2-core-logic.md | pkg/cosmoflare/upload.go:207 |
| Daemon = thin transport, all errors→502 | agent-7-api-design.md | internal/server/rest.go:94 |
| Metrics: one failing source silences all | agent-2-core-logic.md, agent-1-code-quality.md | internal/server/metrics.go:78 |
| SSE transport shipped; event bus absent | agent-15-work-completion.md | internal/server/sse.go:43, internal/webhook/manager.go:182 |
| Desktop consumes daemon via SSE→React Query | agent-7-api-design.md, agent-15-work-completion.md | desktop/src/api/sse.ts:10 |
| Desktop design system scaffolded, never written | agent-17-design-taste.md, agent-18-accessibility.md | desktop/src/styles.css:1-15 |
