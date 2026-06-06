# Cosmoflare

Open-source Go library and CLI for managing the full Cloudflare developer platform. The `pkg/cosmoflare/` library provides importable Go packages for every Cloudflare service — R2 storage, Workers, KV, DNS, and 20+ more.

## Product Vision

Cosmoflare is a **3-tier product** by CosmoLabs:

| Tier | Product | Model | Purpose |
|------|---------|-------|---------|
| **CLI** | `cosmoflare` | Free, open-source (MIT) | Developer tool, agent-friendly, community adoption |
| **Desktop** | Tauri app (macOS/Windows/Linux) | Paid | GUI dashboard, real-time notifications, infrastructure graph |
| **Mobile** | React Native (iOS/Android) | Paid (subscription) | On-the-go monitoring, push alerts, quick actions |

**Design principle**: Library-first, CLI on top, GUI apps wrap the same core. No separate API implementations per tier.

### Full Cloudflare Platform Coverage

| Service | Status | Library |
|---------|--------|---------|
| **R2** (storage) | Implemented | `client.go`, `storage.go`, `upload.go`, `download.go`, `multipart.go` |
| **Workers** (compute) | Implemented | `worker.go` |
| **KV** (key-value) | Implemented | `kv.go` |
| **DNS Records** | Implemented | `dns.go` |
| **Zones** | Implemented | `zone.go` |
| **SSL/TLS** | Implemented | `ssl.go` |
| **Cache** | Implemented | `cloudflare_cache.go` |
| **Page/Redirect Rules** | Implemented | `pagerules.go` |
| **WAF/Firewall** | Implemented | `waf.go`, `firewall.go` |
| **Email Routing** | Implemented | `email.go` |
| **CORS** | Implemented | `cors.go` |
| **D1** (SQL database) | Implemented | `d1.go` |
| **Pages** (static hosting) | Implemented | `pages.go` |
| **Queues** (message queues) | Implemented | `queue.go` |
| **Images** | Implemented | `images.go` |
| **Hyperdrive** | Implemented | `hyperdrive.go` |
| **Vectorize** | Implemented | `vectorize.go` |
| **Workers AI / AI Gateway** | Implemented | `ai.go` |
| **Stream** (video) | Implemented | `stream.go` |
| **Healthchecks** | Implemented | `healthcheck.go` |
| **Diagnostics** | Implemented | `doctor.go` |
| **Domains** | Implemented | `domains.go` |

### Workflow Commands (not service-specific)

| Command | Purpose |
|---------|---------|
| `cosmoflare dev` | Local dev server proxy with hot-reload |
| `cosmoflare init` | Project scaffolding with framework detection |
| `cosmoflare diff` | Compare local config vs live Cloudflare state |
| `cosmoflare apply` | Declarative config reconciliation |
| `cosmoflare sync` | rsync-like directory synchronization with R2 |
| `cosmoflare watch` | Auto-sync local directory to R2 on file changes |
| `cosmoflare cost` | Monthly cost estimation |
| `cosmoflare export/import` | Full account config backup/restore |
| `cosmoflare templates` | Project scaffolding from 5 built-in templates |
| `cosmoflare validate` | Config validation against CF API constraints |
| `cosmoflare terraform` | Generate Terraform .tf files from live state |
| `cosmoflare mcp` | MCP tool server for AI agent integration |
| `cosmoflare wrangler` | Import wrangler.toml compatibility |
| `cosmoflare audit` | CLI mutation audit logging |
| `cosmoflare alerts` | Alert rules for error rates, limits, failures |
| `cosmoflare account` | Multi-account switching |
| `cosmoflare plugin` | Community extension system |
| `cosmoflare doctor` | Domain health diagnostics |

### Agent-First UX
The CLI must be as usable by an AI agent as by a human:
- Rich `--help` on every command (agents read help to learn usage)
- Comprehensive `USAGE.md` as agent reference documentation
- `--json` output on all commands (agents parse JSON, not tables)
- Clear error messages with actionable fix suggestions
- Predictable, consistent command structure
- Deterministic exit codes for scripting
- MCP server mode for direct AI agent integration

## Project

- **Language**: Go 1.26
- **Module**: `github.com/CosmoLabs-org/cosmoflare`
- **Version**: See `.version-registry.json`
- **Binary**: `cosmoflare` (backward-compat alias: `r2go2`)
- **License**: MIT (open-source)

## Structure

- `cmd/` - CLI commands (cobra): 40+ command files
- `pkg/cosmoflare/` - Public library (importable by any Go project):
  - Each Cloudflare service has its own file with a `*Service` struct
  - Shared: `types.go`, `errors.go`, `options.go`, `config.go`
  - Import as: `cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"`
- `internal/` - Internal packages (cli, config, interactive, tui, utils, webhook)
- `docs/` - Documentation, sessions, planning, issues, roadmap
- `docs/PRODUCT-VISION.md` - Full Cosmoflare product vision, 3-tier model
- `docs/USAGE.md` - Agent-reference CLI usage guide (1500+ lines)
- `docs/roadmap/` - Roadmap items (ROAD-001 through ROAD-072)

## Development

```bash
go build -o build/cosmoflare .   # Build
go test ./cmd/ ./pkg/cosmoflare/ ./internal/... -timeout 60s  # Test
go vet ./...                     # Vet
```

## Conventions

- Follow existing code patterns
- Tests live alongside source files (`*_test.go`)
- Use `internal/` for non-exported packages
- Every command must support `--json` output
- Every command must have detailed `--help` with examples
- Agent-readable error messages (include what failed, why, and how to fix)
- Each service gets its own subcommand group and library package
- All services configured via single `.cosmoflare.yaml` project config
- `r2go2` binary remains as backward-compatible alias
