# Contributing to Cosmoflare

Cosmoflare is a Go library and CLI for the full Cloudflare developer platform
(R2, Workers, KV, DNS, D1, Pages, Queues, and 20+ more services). Library
first: the CLI and the desktop app are thin layers over
`pkg/cosmoflare/` — new capabilities land in the library, then get a command.

## Prerequisites

- Go 1.26+ (`go.mod` is the toolchain source of truth)
- A Cloudflare account + API token for anything touching live APIs
  (unit tests never need credentials; integration tests are opt-in via
  build tags)

## Build & Test

```bash
go build -o build/cosmoflare .     # Build the CLI
go test ./cmd/ ./pkg/cosmoflare/ ./internal/...   # Unit tests
go vet ./...                       # Vet
```

The release pipeline (`make release-prepare`) additionally runs the
govulncheck security gate, multi-platform builds, archives, and checksums.

## Code Conventions

- **Library-first**: service logic lives in `pkg/cosmoflare/` (one
  `*Service` per Cloudflare service); `cmd/` only parses flags and renders
  output.
- **Tests live alongside source files** (`*_test.go`). New behavior ships
  with tests — bug fixes ship with a regression test that fails before the
  fix.
- **Every command supports `--json`** and carries detailed `--help` with
  examples. Cosmoflare is designed to be driven by AI agents as much as by
  humans: machine-readable output and actionable error messages (what
  failed, why, how to fix) are product features.
- **Error style**: wrap failures in the service layer's error type with the
  failing operation context — never return bare SDK errors to the CLI layer.
- Follow the existing code patterns in the package you touch.

## Commit Messages

Conventional commits (`type(scope): subject`), types:
`feat|fix|test|docs|chore|refactor|perf|ci|build|security`. The subject is
imperative and specific: `fix(sync): down-sync deletes remove local files`,
not `fixed bug`.

## Environment Variables

| Variable | Purpose | Used by |
|----------|---------|---------|
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account ID (credential fallback) | client, all services |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token (credential fallback) | client, all services |
| `CLOUDFLARE_EMAIL` | Account email for interactive auth prompts | `auth login` |
| `COSMOFLARE_NO_KEYCHAIN` | Set to `1` to skip OS keychain storage (store config file only) | config layer |

Credential precedence: **explicit options/flags > environment variables >
named profile** in the machine config (`~/.cosmoflare/config.yaml`).

## Reporting Bugs

Open an issue with: the command you ran, the exact error output, your OS,
and the CLI version. `--json` output is the most useful thing to paste.
