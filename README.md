# Cosmoflare

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/Cloudflare-Platform-F38020?style=flat-square&logo=cloudflare" alt="Cloudflare">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/github/actions/workflow/status/CosmoLabs-org/CosmoDev-R2Go2/ci.yml?style=flat-square&label=CI" alt="CI">
</p>

**Open-source CLI and Go library for the full Cloudflare developer platform.**

Built by [CosmoLabs](https://cosmolabs.org).

---

## What is Cosmoflare?

Cosmoflare lets you control the entire Cloudflare platform from the terminal. It covers R2 storage, Workers, KV, DNS, Zones, SSL/TLS, Cache, and more -- with every service accessible through a single binary.

It is designed agent-first: every command supports `--json` output, has detailed `--help`, uses predictable exit codes, and returns actionable error messages. Claude Code, AI agents, and human developers all use the same interface.

The Go library (`pkg/r2go2/`) is the stable API surface. The CLI wraps it. A future React Native mobile app will wrap the same library. The `r2go2` binary remains as a backward-compatible alias.

## Services

| Service | Status | Commands |
|---------|--------|----------|
| **R2 Storage** | Implemented | `bucket`, `object` |
| **Workers** | Implemented | `worker` |
| **KV** | Implemented | `kv` |
| **DNS Records** | Implemented | `dns` |
| **Zones** | Implemented | `zone` |
| **SSL/TLS** | Implemented | `ssl` |
| **Cache** | Implemented | `cache` |
| **Page Rules** | Planned | -- |
| **WAF/Firewall** | Planned | -- |
| **D1 Database** | Planned | -- |
| **Pages** | Planned | -- |
| **Email Routing** | Planned | -- |
| **Images** | Planned | -- |
| **Stream** | Planned | -- |
| **Workers AI** | Planned | -- |
| + more | Planned | [see roadmap](docs/roadmap/) |

## Quick Start

### Install

```bash
go install github.com/CosmoLabs-org/CosmoDev-R2Go2@latest
```

Or build from source:

```bash
git clone https://github.com/CosmoLabs-org/CosmoDev-R2Go2.git
cd CosmoDev-R2Go2
make build
```

### Configure

```bash
export CLOUDFLARE_API_TOKEN="your-token"
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
```

Or use the interactive setup wizard:

```bash
r2go2 config init
```

### Use

```bash
# Zones and DNS
r2go2 zone list --json
r2go2 dns list ZONE_ID
r2go2 dns create ZONE_ID --type A --name app --content 1.2.3.4

# SSL/TLS
r2go2 ssl status ZONE_ID
r2go2 ssl settings ZONE_ID

# Cache
r2go2 cache purge ZONE_ID --all --force
r2go2 cache settings ZONE_ID

# R2 Storage
r2go2 bucket list
r2go2 object put my-bucket ./file.txt --key="uploads/file.txt"
r2go2 object get my-bucket uploads/file.txt --output=local.txt
r2go2 object presign my-bucket uploads/file.txt --expires=1h

# Workers
r2go2 worker list
r2go2 worker deploy --name my-worker --script worker.js

# KV
r2go2 kv namespace list
r2go2 kv put NAMESPACE_ID my-key "my-value"
r2go2 kv get NAMESPACE_ID my-key
```

## Agent-First Design

Every command is built to be used programmatically:

- **`--json` on every command** -- structured output for parsing, no decorative formatting to strip
- **Rich `--help`** -- agents read help text to discover usage without documentation
- **Predictable exit codes** -- 0 for success, non-zero for failure, consistent across all commands
- **Actionable errors** -- messages include what failed, why, and how to fix it

```bash
# Machine-readable output
r2go2 bucket list --json | jq '.buckets[].name'

# Scripting
result=$(r2go2 object put my-bucket ./file.txt --json)
echo "$result" | jq -e '.success'
```

## Go Library

The public library can be imported by any Go project:

```go
import r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"

// DNS management
dnsSvc, _ := r2go2.NewDNSServiceFromCreds(zoneID, apiToken)
records, _ := dnsSvc.List(ctx)

// Zone management
zoneSvc, _ := r2go2.NewZoneServiceFromCreds(accountID, apiToken)
zones, _ := zoneSvc.List(ctx)

// SSL/TLS
sslSvc, _ := r2go2.NewSSLServiceFromCreds(zoneID, apiToken)
status, _ := sslSvc.GetSSL(ctx)

// Cache
cacheSvc, _ := r2go2.NewCacheServiceFromCreds(zoneID, apiToken)
result, _ := cacheSvc.PurgeAll(ctx)

// R2 Storage
client, _ := r2go2.NewClient(
    r2go2.WithAccountID(accountID),
    r2go2.WithAPIToken(apiToken),
)
buckets, _ := client.ListBuckets(ctx)
```

## Shell Completion

```bash
# Bash
r2go2 completion bash > /etc/bash_completion.d/r2go2

# Zsh
r2go2 completion zsh > "${fpath[1]}/_r2go2"

# Fish
r2go2 completion fish > ~/.config/fish/completions/r2go2.fish
```

## Development

```bash
# Build
go build -o build/r2go2 .

# Unit tests (no network required)
go test ./pkg/... ./internal/...

# Integration tests (no network required)
go test ./tests/integration/... -v -count=1

# Static analysis
go vet ./...
```

### Project Structure

```
cmd/                  CLI commands (Cobra)
  bucket.go           R2 bucket operations
  object.go           R2 object operations
  worker.go           Workers management
  kv.go               KV store operations
  dns.go              DNS record management
  zone.go             Zone management
  ssl.go              SSL/TLS certificates and settings
  cache.go            Cache purge and settings
  config.go           Profile management
  root.go             Root command and global flags
pkg/r2go2/            Public Go library (stable API)
internal/             Private packages (config, tui, utils)
tests/                Integration tests
docs/                 Documentation, roadmap, planning
```

## Roadmap

Cosmoflare aims to cover every service the Cloudflare API exposes. Upcoming:

- Page/Redirect Rules, WAF/Firewall, Email Routing
- D1 (SQL at edge), Pages (static hosting), Queues
- Images, Stream (video), Hyperdrive, Vectorize, Workers AI
- MCP server for native AI tool integration
- `cosmoflare status` dashboard, `cosmoflare diff`, config-as-code

Full roadmap: [docs/roadmap/](docs/roadmap/)

## Product Vision

Cosmoflare is the open-source CLI tier of a 3-tier product. The CLI and Go library are free (MIT). A React Native mobile app with push notifications and one-tap Cloudflare management is the planned paid product. The Go library powers both -- same API logic, no separate implementations.

Full vision: [docs/PRODUCT-VISION.md](docs/PRODUCT-VISION.md)

## License

MIT -- see [LICENSE](LICENSE) for details.

Built by [CosmoLabs](https://cosmolabs.org).
