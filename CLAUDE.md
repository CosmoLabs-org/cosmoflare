# CosmoDev-R2Go2

Open-source Go library and CLI tool for managing Cloudflare R2 buckets.

## Product Vision

R2Go2 is a **3-tier product**:
1. **Go library** — clean public API, importable by any Go project (OpenCode, Codex, etc.)
2. **Standalone CLI** — open-source tool anyone can install, **agent-friendly** (LLMs like Claude/GLM use it from terminal)
3. **CCS subcommand** (future) — `ccs r2` integration, R2Go2 as core requirement like GoRalph

**Design principle**: Library-first, CLI on top, CCS integration last. No CCS dependencies in the core.

### Full Cloudflare Platform
R2Go2 covers the entire Cloudflare developer platform:
- **R2** (storage) — Phase 1: buckets, objects, uploads, downloads
- **Workers** (compute) — Phase 3: deploy, list, logs, bindings
- **KV** (key-value) — Phase 3: namespaces, get/put/delete
- **D1** (SQL database) — Phase 4: create, query, migrate
- **Pages** (static hosting) — Phase 4: deploy, list, custom domains
- **Queues** (message queues) — Phase 5: create, send, consume
- All managed through a single `.r2go2.yaml` project config

### Agent-First UX
The CLI must be as usable by an AI agent as by a human:
- Rich `--help` on every command (agents read help to learn usage)
- Comprehensive `USAGE.md` as agent reference documentation
- `--json` output on all commands (agents parse JSON, not tables)
- Clear error messages with actionable fix suggestions
- Predictable, consistent command structure
- Deterministic exit codes for scripting

## Project

- **Language**: Go 1.26
- **Module**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Version**: See `.version-registry.json`
- **Binary**: `r2go2`
- **License**: MIT (open-source)

## Structure

- `cmd/` - CLI commands (cobra): `bucket.go`, `object.go`, `worker.go`, `kv.go`
- `pkg/r2go2/` - Public library (importable by any Go project):
  - **R2 Storage**: `client.go`, `storage.go`, `upload.go`, `download.go`
  - **Workers**: `worker.go` — `WorkerService` (Deploy, List, Get, Delete, Logs, UpdateSettings)
  - **KV**: `kv.go` — `KVService` (CreateNamespace, ListNamespaces, GetNamespace, DeleteNamespace, Put, Get, Delete, ListKeys)
  - **Shared**: `types.go`, `errors.go`, `options.go`, `config.go`
  - **Future stubs**: `d1.go`, `pages.go`, `queue.go`
- `internal/` - Internal packages (api, cli, config, interactive, tui, utils)
- `docs/` - Documentation, sessions, planning, issues, roadmap
- `docs/audit/` - Comprehensive codebase audits

## Development

```bash
go build -o r2go2 .        # Build
go test ./...               # Test all
go vet ./...                # Vet
```

## Conventions

- Follow existing code patterns
- Tests live alongside source files (`*_test.go`)
- Use `internal/` for non-exported packages
- Every command must support `--json` output
- Every command must have detailed `--help` with examples
- Agent-readable error messages (include what failed, why, and how to fix)
- Full Cloudflare platform coverage: R2, Workers, KV, D1, Pages, Queues
- Each service gets its own subcommand group and library package
- All services configured via single `.r2go2.yaml` project config
