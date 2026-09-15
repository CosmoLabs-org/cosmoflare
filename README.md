# Cosmoflare

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/Cloudflare-Platform-F38020?style=flat-square&logo=cloudflare" alt="Cloudflare">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/github/v/release/CosmoLabs-org/cosmoflare?style=flat-square" alt="GitHub Release">
</p>

**Open-source CLI and Go library for the full Cloudflare developer platform.**

Built by [CosmoLabs](https://cosmolabs.org).

---

## What is Cosmoflare?

Cosmoflare lets you control the entire Cloudflare platform from the terminal. It covers R2 storage, Workers, KV, DNS, Zones, SSL/TLS, Cache, and more -- with every service accessible through a single binary.

It is designed agent-first: every command supports `--json` output, has detailed `--help`, uses predictable exit codes, and returns actionable error messages. Claude Code, AI agents, and human developers all use the same interface.

The Go library (`pkg/cosmoflare/`) is the stable API surface. The CLI and a Tauri desktop app wrap it; a React Native mobile app will follow. The `r2go2` binary remains as a backward-compatible alias.

## Services

| Service | Status | Commands |
|---------|--------|----------|
| **R2 Storage** | Implemented | `bucket`, `object`, `sync`, `watch` |
| **Workers** | Implemented | `worker` |
| **KV** | Implemented | `kv` |
| **DNS Records** | Implemented | `dns` |
| **Zones** | Implemented | `zone` |
| **SSL/TLS** | Implemented | `ssl` |
| **Cache** | Implemented | `cache` |
| **Page/Redirect Rules** | Implemented | `pagerules`, `redirects` |
| **WAF/Firewall** | Implemented | `waf`, `firewall` |
| **Email Routing** | Implemented | `email` |
| **CORS** | Implemented | `cors` |
| **D1 Database** | Implemented | `d1` |
| **Pages** | Implemented | `pages` |
| **Queues** | Implemented | `queue` |
| **Images** | Implemented | `images` |
| **Hyperdrive** | Implemented | `hyperdrive` |
| **Vectorize** | Implemented | `vectorize` |
| **Workers AI / AI Gateway** | Implemented | `ai` |
| **Stream** | Implemented | `stream` |
| **Healthchecks** | Implemented | library only (`pkg/cosmoflare/healthcheck.go`) |
| **Diagnostics** | Implemented | `doctor` |
| **Domains** | Implemented | `domains` |

Workflow commands (not tied to one service): `dev`, `init`, `diff`, `apply`, `cost`, `export`/`import`, `templates`, `validate`, `terraform`, `wrangler`, `mcp`, `audit`, `alerts`, `account`, `plugin`, `status`, `metrics`, `dashboard`.

## Quick Start

### Install

```bash
go install github.com/CosmoLabs-org/cosmoflare@latest
```

Or build from source:

```bash
git clone https://github.com/CosmoLabs-org/cosmoflare.git
cd cosmoflare
make build
```

### Configure

```bash
export CLOUDFLARE_API_TOKEN="your-token"
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
```

Or use the interactive setup wizard:

```bash
cosmoflare config init
```

### Use

```bash
# Zones and DNS
cosmoflare zone list --json
cosmoflare dns list ZONE_ID
cosmoflare dns create ZONE_ID --type A --name app --content 1.2.3.4

# SSL/TLS
cosmoflare ssl status ZONE_ID
cosmoflare ssl settings ZONE_ID

# Cache
cosmoflare cache purge ZONE_ID --all --force
cosmoflare cache settings ZONE_ID

# R2 Storage
cosmoflare bucket list
cosmoflare object put my-bucket ./file.txt --key="uploads/file.txt"
cosmoflare object get my-bucket uploads/file.txt --output=local.txt
cosmoflare object presign my-bucket uploads/file.txt --expires=1h

# Workers
cosmoflare worker list
cosmoflare worker deploy my-worker --script=worker.js

# KV
cosmoflare kv namespace list
cosmoflare kv put NAMESPACE_ID my-key --value="my-value"
cosmoflare kv get NAMESPACE_ID my-key
```

## Agent-First Design

Every command is built to be used programmatically:

- **`--json` on every command** -- structured output for parsing, no decorative formatting to strip
- **Rich `--help`** -- agents read help text to discover usage without documentation
- **Predictable exit codes** -- 0 for success, non-zero for failure, consistent across all commands
- **Actionable errors** -- messages include what failed, why, and how to fix it
- **MCP server built in** -- `cosmoflare mcp` exposes the full CLI surface (140+ auto-generated tools) to AI agents, read-only by default with mutations gated behind an explicit allow switch

```bash
# Machine-readable output
cosmoflare bucket list --json | jq '.buckets[].name'

# Scripting
result=$(cosmoflare object put my-bucket ./file.txt --json)
echo "$result" | jq -e '.success'
```

## Go Library

The public library can be imported by any Go project:

```go
import cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"

// DNS management
dnsSvc, _ := cosmoflare.NewDNSServiceFromCreds(zoneID, apiToken)
records, _ := dnsSvc.List(ctx)

// Zone management
zoneSvc, _ := cosmoflare.NewZoneServiceFromCreds(accountID, apiToken)
zones, _ := zoneSvc.List(ctx)

// SSL/TLS
sslSvc, _ := cosmoflare.NewSSLServiceFromCreds(zoneID, apiToken)
status, _ := sslSvc.GetSSL(ctx)

// Cache
cacheSvc, _ := cosmoflare.NewCacheServiceFromCreds(zoneID, apiToken)
result, _ := cacheSvc.PurgeAll(ctx)

// R2 Storage
client, _ := cosmoflare.NewClient(
    cosmoflare.WithAccountID(accountID),
    cosmoflare.WithAPIToken(apiToken),
)
buckets, _ := client.ListBuckets(ctx)
```

## Shell Completion

```bash
# Bash
cosmoflare completion bash > /etc/bash_completion.d/cosmoflare

# Zsh
cosmoflare completion zsh > "${fpath[1]}/_cosmoflare"

# Fish
cosmoflare completion fish > ~/.config/fish/completions/cosmoflare.fish
```

## Development

```bash
# Build
go build -o build/cosmoflare .

# Unit tests (no network required)
go test ./pkg/... ./internal/...

# Integration tests (no network required)
go test ./tests/integration/... -v -count=1

# Static analysis
go vet ./...
```

### Releasing

Releases are cut locally — there is no CI, by policy. One command prepares and
stages everything, then prints the publish command for you to run:

```bash
make release TAG=v0.X.Y
```

It runs the full preparation (clean, test, vulncheck, cross-build, archives),
sanity-gates every archive (>1MB — smaller means a failed platform build),
stages the tar.gz/zip archives in `dist/upload/` with sha256 checksums, and
prints the `gh release create` command with the release notes file. Publishing
itself stays manual and operator-gated.

### Project Structure

```
cmd/                  CLI commands (Cobra, 40+ files — one per service group)
pkg/cosmoflare/       Public Go library (stable API, one *Service per service)
internal/             Private packages (config, server, cli, interactive, tui, utils)
tests/                Integration tests
docs/                 Documentation, roadmap, planning
desktop/              Tauri desktop app (Rust sidecar + React UI)
```

## Roadmap

Every major Cloudflare service is covered today (see the table above). In progress:

- Desktop dashboard (Tauri + React) -- GUI tier wrapping the same Go library
- Guardrail enforcement for agent-driven mutations (sync, MCP tools)
- Mobile tier (React Native) -- on-the-go monitoring and quick actions

Full roadmap: [docs/roadmap/](docs/roadmap/)

## Product Vision

Cosmoflare is the open-source CLI tier of a 3-tier product. The CLI and Go library are free (MIT). A Tauri desktop app (GUI dashboard, real-time notifications) and a React Native mobile app (push alerts, quick actions) are the paid tiers. The Go library powers all three -- same API logic, no separate implementations.

Full vision: [docs/PRODUCT-VISION.md](docs/PRODUCT-VISION.md)

## License

MIT -- see [LICENSE](LICENSE) for details.

Built by [CosmoLabs](https://cosmolabs.org).
