# Entry Points & Execution Paths — cosmoflare v0.17.0

> Synthesized from 13 agent reports (2026-08-31 audit).

## Primary Entry Points

**1. CLI binary** (`main.go` → `cmd/root.go`, 366 cobra commands across 68 files) — agent-4 counted; agent-7 traced lifecycle:
- Global: `--json` envelope (`OutputResponse`, root.go:215-221), `--dry-run`, `--profile`
- Command groups: r2/bucket/object, worker, kv, dns, zone, ssl, cache, pagerules, waf/firewall, email, cors, d1, pages, queue, images, hyperdrive, vectorize, ai, stream, healthcheck, doctor, domains, redirects
- Workflow: `dev`, `init`, `diff`, `apply`, `sync`, `watch`, `cost`, `export`/`import`, `templates`, `validate`, `terraform`, `mcp`, `wrangler`, `audit`, `alerts`, `account`, `plugin`, `dashboard`, `serve`, `metrics`, `status`
- Disabled (dead surface): `analytics.go.disabled`, `cicd.go.disabled`, `domain.go.disabled`, `migrate.go.disabled` + `cmd_disabled/{policy,restore,upload,webhook}.go` (agent-4)

**2. Daemon** — `cosmoflare serve` (cmd/serve.go) → `internal/server`:
- Binds `127.0.0.1:0`, generates 128-bit bearer token (serve.go:76-83,152), emits handshake JSON `{addr, token}` on stdout, logs to stderr (agent-7)
- `--metrics-interval` flag (serve.go:62) wires MetricsProducer (serve.go:128) (agent-15)
- REST: `/healthz`, `/profiles`, `/zones`, `/metrics`, `/sources`, `/events` — all effectively GET, no method enforcement (agent-7)
- SSE: `/events` multiplexing `metrics` + `notifications` channels (sse.go:64-66,90-124)

**3. MCP server** — `cosmoflare mcp` → `pkg/cosmoflare/mcp.go`: JSON-RPC 2.0 over stdio, protocol `2024-11-05` pinned, 8 tools registered (bucket_list, worker_list, worker_deploy, dns_list, kv_list, zone_list, cache_purge, doctor) at `mcp.go:286-335` (agent-4).

**4. Go library** — `import "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"`: `NewClient(ctx, opts...)` with 14 functional options (6 of which are currently no-ops — agent-2), per-service `New*Service` constructors (agent-1).

**5. Desktop app** — `desktop/` Tauri 2: Rust sidecar spawns the daemon, React 18 shell consumes REST+SSE (`desktop/src/api/client.ts`, `api/sse.ts`), React Query cache bridges (agent-15, agent-7).

**6. Installer TUI** — `cmd/installer_tui/main.go` (1299 lines, single-model bubbletea app; agent-1).

## Request Lifecycle

**Daemon REST request** (agent-7):
```
GET /zones?profile=X
→ auth middleware: Bearer token or ?token= (== compare ⚠, empty-token edge ⚠; server.go:147-152)
→ ?profile= scoping against source registry
→ withCloudflare wrapper: source → service call
→ any error → 502 {"error": msg} (rest.go:94 ⚠)
→ success → writeJSON application/json (server.go:169)
```

**SSE subscription** (agent-7, agent-2):
```
GET /events (token-gated)
→ register subscriber in sseHub (buffer 32)
→ metrics channel: MetricsProducer tick → delta check (DeepEqual) → publish-if-changed
→ notifications channel: SetCloudflareOnline transitions ONLY (server.go:73-87)
→ no heartbeat (proxies drop idle conns ⚠), no replay on reconnect ⚠
```

**Upload** (agent-2): `cmd object put` → option assembly ×3 paths (stdin/resume/normal, object.go:483-713) → `Upload` (<100MB) or `MultipartUpload` → per-part `io.ReadFull` → Complete. Abort path reuses canceled ctx ⚠ (upload.go:175).

**Sync** (agent-2): `cmd sync up|down` → `scanLocalDir` (no default excludes ⚠) → `planUp/planDown` (mtime+size or checksum heuristic) → list remote (single page ⚠) → execute ops with optional `--delete`.

## Background Processes

- **MetricsProducer** (`internal/server/metrics.go`): interval poll of zones/workers/R2/KV; subscriber-gated; delta-suppressed; returns on first source error ⚠ (metrics.go:78-80) — the direct cause of "desktop dashboard is not live" when any source lacks permission (agent-2; matches open issue docs/ideas/2026-06-21).
- **FileWatcher** (`pkg/cosmoflare/watcher.go` + `cmd/watch.go`): snapshot → diff → upload; `--delete` propagates phantom deletions on scan errors ⚠ (agent-2).
- **Rust daemon watchdog** (desktop): covers initial spawn only — mid-session crash only logged, not acted on (IDEA-038 seed; agent-12).
- **CI**: 3 workflows (ci.yml, test.yml redundant ⚠, release.yml 18/18 failed) (agent-5, agent-9).

## Agent Cross-References

| Entry point | Source Agent | Evidence |
|-------------|-------------|----------|
| 366 cobra commands; 8 MCP tools | agent-4-competitive.md | pkg/cosmoflare/mcp.go:286 |
| Daemon handshake + auth model | agent-7-api-design.md | cmd/serve.go:104-114 |
| SSE channel inventory | agent-7-api-design.md | internal/server/sse.go:64 |
| MetricsProducer gating/delta | agent-15-work-completion.md | internal/server/metrics.go:41 |
| Notifications channel single source | agent-15-work-completion.md | internal/server/server.go:73-87 |
| Upload triple path | agent-1-code-quality.md | cmd/object.go:483-713 |
