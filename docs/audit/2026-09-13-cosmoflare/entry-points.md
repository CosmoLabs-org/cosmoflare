# Entry Points & Execution Paths — cosmoflare v0.26.0

Synthesized from audit agents (2026-09-13). Every claim cites its source agent report.

## Primary Entry Points

**Binary: `cosmoflare`** (backward-compat alias `r2go2`) — `main.go` → `cmd/root.go` cobra root. 40+ command files in `cmd/` (see agent-1-code-quality.md). Notable command groups and their entry files:

| Entry | File | Notes |
|-------|------|-------|
| `cosmoflare sync` | cmd/sync.go, pkg/cosmoflare/sync.go | Flagship workflow; plan/execute engine with direction-blind delete bug (sync.go:425, see agent-2-core-logic.md) |
| `cosmoflare apply` | pkg/cosmoflare/apply.go:423 | Declarative reconcile; deletes-by-omission hazard (see agent-2-core-logic.md) |
| `cosmoflare serve` | cmd/serve.go | Daemon: REST+SSE, token auth, limits snapshot at serve.go:238 (see agent-15-work-completion.md) |
| `cosmoflare dev` | pkg/cosmoflare/dev.go:161 | **Stub — returns 502 for every request** while USAGE.md documents a working proxy (see agent-7-api-design.md) |
| `cosmoflare auth` | cmd/auth.go | login/logout/status/rotate; rotate is placeholder that always errors (auth.go:479), revoke-old ordering bug (auth.go:227-239) (see agent-9-infrastructure.md) |
| `cosmoflare mcp` | pkg/cosmoflare/mcp.go:112 | MCP tool server, stdio-only; fail-closed mutation gating (unique vs official cloudflare/mcp) (see agent-4-competitive.md) |
| `cosmoflare doctor` | cmd/ | Domain health diagnostics incl. permission manifest doctor (FEAT-019 landed, cmd/auth_permissions.go) (see agent-15-work-completion.md) |
| TUI dashboard | internal/tui/ | bubbletea; `handleKeyMsg` hotspot (update.go) (see agent-1-code-quality.md) |
| Installer TUI | cmd/installer_tui/main.go | Raw `cmd.Start()` at line 1280 violates spawn policy ROAD-525 (see agent-1, agent-2, agent-9 — 3 agents) |

## Request Lifecycle (daemon path)

1. Desktop launches → sidecar spawns CLI daemon (`cosmoflare serve`) on localhost with generated token
2. Desktop `api/client.ts` calls REST endpoints with bearer token (unchecked `any` casts on responses, see agent-7-api-design.md)
3. Daemon `authMiddleware` (internal/server/server.go:148) — **non-constant-time token compare**, `?token=` query fallback, no 405 enforcement (see agent-9-infrastructure.md, agent-7-api-design.md)
4. Pollers gather per-source data (limits every 30s); SSE stream (`api/sse.ts`) pushes to desktop
5. Desktop renders: Dashboard/Notifications views; live regions per-view (a11y gap, see agent-18-accessibility.md)

## Request Lifecycle (sync path — the data-loss surface)

1. `cmd/sync.go` parses flags → `SyncPlanInput{Direction, Prefix, Include, Exclude, Delete...}`
2. `planUp/planDown` diff local vs remote listings → `SyncPlan{Operations[]}`
3. **`--include` flag never read** (sync.go:113) — silent no-op (see agent-2-core-logic.md)
4. Delete planning skips exclude checks (sync.go:275) — `.env`-pattern files deletable (see agent-2-core-logic.md)
5. `Execute` loop: sequential ops; delete ops always `DeleteRemoteObject` with unprefixed key (sync.go:425) regardless of direction (see agent-2-core-logic.md)
6. Uploads → multipart (upload.go, multipart.go); abort uses canceled ctx → orphaned billed parts (upload.go:181) (see agent-2-core-logic.md)
7. Downloads stream to final path directly — interrupted run truncates the pre-existing local file (download.go:94) (see agent-2-core-logic.md)

## Background Processes

- **serve daemon** — the only long-running process; spawned as desktop sidecar (version skew: sidecar config 0.16.0 vs CLI 0.26.0, missing linux/arm64 triple, see agent-5-distribution.md)
- **watch** (`cosmoflare watch`) — file-watcher triggering sync; shares the sync engine defects above
- No cron/scheduled jobs; no workers; MCP server runs per-invocation over stdio

## Release Path (local-only by policy)

`make release-prepare` → goreleaser local build → manual publish. **Broken rung**: tags v0.23–v0.26 pushed but no `gh release create` ever ran — published release is v0.22.0 while `go install @latest` builds v0.26.0 (see agent-5-distribution.md, agent-4-competitive.md). Divergent duplicate path: r2go2-branded `make dist` (tars docs/ into archives — the AWS-key leak vector, Makefile:97, see agent-5-distribution.md).
