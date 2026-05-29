---
schema_version: 1
status: PENDING
type: continuation
created: 2026-05-27T08:10:00-03:00
priority: high
requires_reading:
  - docs/USAGE.md
  - CLAUDE.md
---

# Next Session: ROAD-058 cosmoflare dev + Docs Gaps + CCS Upgrades

## Context

Current state: **v0.11.0** (build 423), master branch. The post-audit Phase 0 fixes are done
(multipart sort, NoSuchKey classification, config race, etc. — all merged). The CLI now covers
R2, Workers, KV, DNS, Zones, SSL, Cache, Pages (stub), D1 (stub), Queue (stub), Email Routing,
Firewall, WAF, CORS, Pagerules, Domains, Doctor — a full Cloudflare platform footprint.

No open issues in the tracker. Six CCS upgrade subsystems remain unresolved (see below).

## Primary Goal

Implement **ROAD-058: `cosmoflare dev`** — a local dev server that proxies all Cloudflare services
with hot-reload support. This is the highest-priority workflow feature (priority 82).

## Tasks (ordered by priority)

### 1. ROAD-058 — `cosmoflare dev` local dev server (priority 82) [LARGE]

A `cmd/dev.go` command that spins up a local HTTP proxy for Cloudflare services, enabling
offline-friendly development with hot-reload.

**Scope:**
- `r2go2 dev` (alias: `cosmoflare dev`) — start local dev server
- Flags: `--port` (default 8787, matches Wrangler), `--watch` (hot-reload config changes),
  `--services` (comma-separated: r2,kv,workers,d1 — default: all), `--profile` (reuse existing
  profile credentials)
- Proxies requests to real Cloudflare APIs with a local URL rewrite layer
- Config: reads from `.cosmoflare.yaml` in cwd
- `--json` output for machine-readable startup events
- Clean shutdown on SIGINT/SIGTERM

**Files to create/modify:**
- `cmd/dev.go` — main Cobra command
- `pkg/r2go2/dev.go` — `DevServer` struct with `Start(ctx)`, `Stop()` methods
- `cmd/dev_test.go` + `pkg/r2go2/dev_test.go` — unit tests (no network required)
- `docs/USAGE.md` — add `cosmoflare dev` section

**Pattern to follow:** `cmd/doctor.go` for the Cobra structure; `pkg/r2go2/doctor.go` for the
service struct pattern (stdlib-only where possible).

**TDD order:** Write failing tests first (`DevServer.Start` returns nil error, config parsing,
graceful shutdown via context cancel), then implement.

---

### 2. ROAD-051 — `cosmoflare logs --follow` real-time log tailing (priority 82) [MEDIUM]

**Scope:**
- Extend `cmd/worker.go` `logs` subcommand with `--follow` / `-f` flag
- Poll Cloudflare Workers Logs API on interval (default 2s, `--interval` flag)
- Filter by `--level` (error|warn|info|debug), `--since` (duration, e.g. `15m`)
- `--json` streams newline-delimited JSON events
- Clean exit on SIGINT

**Files to modify:**
- `cmd/worker.go` — add `--follow`, `--interval`, `--level`, `--since` flags
- `pkg/r2go2/worker.go` — add `TailLogs(ctx, scriptName, opts)` method with polling loop
- `cmd/worker_test.go` — extend existing test suite

---

### 3. ROAD-057 — `cosmoflare init` interactive project scaffolding (priority 80) [MEDIUM]

**Scope:**
- `r2go2 init` — detect framework (Wrangler, plain Go, Node), scaffold `.cosmoflare.yaml`,
  create `wrangler.toml` stub if applicable
- Interactive prompts (use existing `internal/interactive/` patterns)
- `--yes` / `-y` flag for non-interactive CI mode with sensible defaults
- `--template` flag: `worker|pages|r2|full` (default: auto-detect)

**Files to create/modify:**
- `cmd/init.go` — Cobra command (check: `cmd/create.go` exists at 1.6K, may be related — review first)
- `internal/interactive/init.go` — prompt flow
- `cmd/init_test.go`

---

### 4. Documentation gaps — Pages, Queue, Bucket commands (MEDIUM)

Three command groups lack proper USAGE.md / README entries flagged by `ccs workcheck`:

**Pages (`cmd/pages.go`):**
- Add `## cosmoflare pages` section to `docs/USAGE.md` covering:
  `pages list`, `pages get`, `pages deploy`, `pages delete`, `pages env`
- Create `READMEs/commands/pages.md` (follow `READMEs/commands/` structure if it exists, else
  mirror the dns or cache entry in `docs/USAGE.md`)

**Queue (`cmd/queue.go`):**
- Add `## cosmoflare queue` section to `docs/USAGE.md` covering:
  `queue list`, `queue create`, `queue delete`, `queue send`, `queue consumer`

**Bucket commands:**
- Audit `cmd/bucket.go` (14.6K) against `docs/USAGE.md` for any undocumented flags/subcommands
  added since the last USAGE.md update (CORS, lifecycle, metrics flags likely missing)

---

### 5. CCS upgrade remaining subsystems (LOW — batch at session-end)

Six subsystems from the last `ccs upgrade` pass still need attention. Run at end of session
after main feature work:

```bash
ccs upgrade --subsystem doc-structure
ccs upgrade --subsystem yaml-frontmatter
ccs upgrade --subsystem prompt-statuses
ccs upgrade --subsystem release-notes
ccs upgrade --subsystem session-summaries
ccs upgrade --subsystem portless
```

Check each for failures before moving to the next. Commit after each subsystem if changes are
made.

---

## Session Entry Checklist

```bash
# 1. Confirm clean state
git status
go build -o build/r2go2 .
go test ./pkg/... ./internal/... -count=1 -timeout 60s

# 2. Review roadmap context
ccs roadmap list --brief | head -10

# 3. Start with TDD for ROAD-058
# Write cmd/dev_test.go and pkg/r2go2/dev_test.go first
```

## Key Constraints

- **Agent-first UX**: every new command needs `--help` examples, `--json` output, actionable
  error messages
- **No network in unit tests**: mock or stub all Cloudflare API calls
- **Bun not applicable** (Go project) — `go build`, `go test`, never npm/pnpm
- **Binary name**: `r2go2` (build target `build/r2go2`); `cosmoflare` is the product alias
- **Profile pattern**: credentials from `internal/config/config.go` via `--profile` flag,
  consistent with all other commands
- **TDD mandatory**: invoke `superpowers:test-driven-development` before writing implementation
  code for ROAD-058

## Version Note

Current version is **v0.11.0**. Next release after this session should be **v0.12.0** (new `dev` command = minor bump).
Use `/release` at session-end after all tests pass.
