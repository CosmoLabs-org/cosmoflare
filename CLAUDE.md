# CosmoDev-R2Go2 (Cosmoflare)

Open-source Go library and CLI tool for managing the full Cloudflare developer platform. R2Go2 is the R2 storage component within the larger **Cosmoflare** ecosystem.

## Product Vision

Cosmoflare is a **3-tier product** by CosmoLabs:

| Tier | Product | Model | Purpose |
|------|---------|-------|---------|
| **CLI** | `cosmoflare` (alias: `r2go2`) | Free, open-source (MIT) | Developer tool, agent-friendly, community adoption |
| **Desktop** | Tauri app (macOS/Windows/Linux) | Paid | GUI dashboard, real-time notifications, infrastructure graph |
| **Mobile** | React Native (iOS/Android) | Paid (subscription) | On-the-go monitoring, push alerts, quick actions |

**Design principle**: Library-first, CLI on top, GUI apps wrap the same core. No separate API implementations per tier.

### Full Cloudflare Platform Coverage
Everything the Cloudflare API allows us to interact with:

| Service | Status | Phase |
|---------|--------|-------|
| **R2** (storage) | Implemented | Phase 1 |
| **Workers** (compute) | Implemented | Phase 3 |
| **KV** (key-value) | Implemented | Phase 3 |
| **DNS Records** | Implemented | Phase 4 |
| **Zones** | Implemented | Phase 4 |
| **SSL/TLS** | Implemented | Phase 4 |
| **Cache** | Implemented | Phase 4 |
| **Page/Redirect Rules** | ROAD-039 | Phase 4 |
| **WAF/Firewall** | ROAD-040 | Phase 5 |
| **Email Routing** | ROAD-041 | Phase 5 |
| **D1** (SQL database) | Stub → ROAD-042 | Phase 5 |
| **Pages** (static hosting) | Stub → ROAD-043 | Phase 5 |
| **Queues** (message queues) | Stub | Phase 5 |
| **Images** | ROAD-044 | Phase 6 |
| **Hyperdrive** | ROAD-046 | Phase 6 |
| **Vectorize** | ROAD-048 | Phase 6 |
| **Workers AI / AI Gateway** | ROAD-049 | Phase 6 |
| **Stream** (video) | ROAD-045 | Phase 7 |
| All managed through a single `.cosmoflare.yaml` project config |

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

- `cmd/` - CLI commands (cobra): `bucket.go`, `object.go`, `worker.go`, `kv.go`, `dns.go`, `zone.go`, `ssl.go`, `cache.go`
- `pkg/r2go2/` - Public library (importable by any Go project):
  - **R2 Storage**: `client.go`, `storage.go`, `upload.go`, `download.go`
  - **Workers**: `worker.go` — `WorkerService` (Deploy, List, Get, Delete, Logs, UpdateSettings)
  - **KV**: `kv.go` — `KVService` (CreateNamespace, ListNamespaces, GetNamespace, DeleteNamespace, Put, Get, Delete, ListKeys)
  - **Shared**: `types.go`, `errors.go`, `options.go`, `config.go`
  - **DNS**: `dns.go` — `DNSService` (Create, List, Get, Update, Delete) — zone-scoped
  - **Zones**: `zone.go` — `ZoneService` (Create, List, Get, Delete, GetSettings) — account-scoped
  - **SSL/TLS**: `ssl.go` — `SSLService` (GetSSL, UpdateSSL, GetVerification, GetSettings, UpdateSettings) — zone-scoped
  - **Cache**: `cloudflare_cache.go` — `CacheService` (PurgeAll, PurgeByURLs/Tags/Hosts, GetSettings, UpdateSettings) — zone-scoped
  - **Future stubs**: `d1.go`, `pages.go`, `queue.go`
- `internal/` - Internal packages (api, cli, config, interactive, tui, utils)
- `docs/` - Documentation, sessions, planning, issues, roadmap
- `docs/PRODUCT-VISION.md` - Full Cosmoflare product vision, 3-tier model, roadmap references
- `docs/USAGE.md` - Agent-reference CLI usage guide
- `docs/roadmap/` - Roadmap items (ROAD-001 through ROAD-065+)
- `docs/audit/` - Comprehensive codebase audits
- `.claude/CLAUDE.md` - Claude Code project instructions (build/test, architecture, known gaps)

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
- Full Cloudflare platform coverage: see roadmap for all 20+ services
- Each service gets its own subcommand group and library package
- All services configured via single `.cosmoflare.yaml` (legacy: `.r2go2.yaml`) project config
- `r2go2` binary remains as backward-compatible alias for `cosmoflare`
