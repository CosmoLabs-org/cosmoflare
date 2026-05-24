# Project Brief: CosmoDev-R2Go2

**Purpose**: Condensed project context for future Claude Code sessions. Read this before working on the codebase.

## Identity

- **Name**: CosmoDev-R2Go2 (Cosmoflare)
- **Version**: 0.9.0
- **Language**: Go 1.26
- **Module**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Binary**: `r2go2` (alias for `cosmoflare`)
- **License**: MIT
- **What it does**: Go CLI and library for managing the full Cloudflare developer platform (R2, Workers, KV, DNS, Zones, SSL, Cache, Healthchecks, Domains, Doctor)

## Tech Stack

| Layer | Technology |
|-------|-----------|
| CLI framework | Cobra |
| Config management | Viper |
| TUI dashboard | Bubble Tea (charmbracelet) |
| S3 client | AWS SDK v2 |
| HTTP client | net/http (stdlib) |
| Testing | stdlib testing + testify |
| Build | go build (goreleaser for releases) |
| CI | GitHub Actions (currently broken) |

## Architecture (3 layers)

1. **`pkg/r2go2/`** (30 files) -- Public library. Importable. 12 service structs. Functional options. Typed errors. This is the stable API.
2. **`cmd/`** (33 files) -- Cobra CLI. Thin shell over the library. Supports `--json` on all commands.
3. **`internal/`** (38 files) -- Config (Viper profiles), TUI (Bubble Tea), interactive wizard, utilities.

## Key Patterns

- **Functional options**: `r2go2.NewClient(r2go2.WithAccountID(...))` -- idiomatic, extensible
- **Typed errors**: `R2NotFoundError`, `R2ValidationError`, `R2AuthError` -- programmatic matching, no string parsing
- **Service struct**: Every service follows `NewXxxService(accountID, apiToken) -> CRUD methods`
- **Input validation**: Every public method validates before API call
- **Dual protocol**: S3-compatible for R2, Cloudflare REST for everything else

## Known Critical Issues (as of audit 2026-05-19)

1. **upload.go:253** -- Multipart sort bug silently corrupts large files
2. **download.go:53** -- Range header always applied (default triggers condition)
3. **internal/config** -- File permissions race on creation (credentials exposed briefly)
4. **cmd/bucket.go** -- `bucket update` is a no-op
5. **CI pipeline** -- Never exercised, would fail on `go vet` errors
6. **install.sh** -- Wrong binary name
7. **printError** -- Silently drops errors in `--json` mode

## Known Debt

- 3,929 lines dead code in `internal/migration/` and `cmd_disabled/`
- TUI uses simulated data, not live API
- No unified service client (services are siloed)
- Global mutable CLI state (package-level vars)
- O(n) list-and-scan for bucket lookups
- `os.Exit()` calls in library code
- No CONTRIBUTING.md
- README shows 6 implemented services as "Planned"
- 332-line config package has zero tests

## Entry Points

| Context | Entry |
|---------|-------|
| CLI | `main.go` -> `cmd.Execute()` -> `cmd/root.go` |
| Library (R2) | `r2go2.NewClient(opts...)` |
| Library (services) | `r2go2.NewXxxService(accountID, apiToken)` |
| TUI | `cmd/dashboard.go` -> `internal/tui/model.go` |
| Tests | `go test ./pkg/... ./internal/...` |
| Config | `internal/config/config.go` -> `.cosmoflare.yaml` |

## Build & Test

```bash
go build -o build/r2go2 .             # Build binary
go test ./pkg/... ./internal/...       # Unit tests (no network)
go vet ./...                            # Static analysis
go test ./tests/integration/... -v      # Integration tests
```

## Metrics Snapshot (v0.9.0)

| Metric | Value |
|--------|-------|
| Source files | 114 |
| Test files | 107 |
| Source LOC | 36,338 |
| Test LOC | 55,009 |
| Test:Source ratio | 1.53x |
| Audit score | 70.8/100 (Mature) |
| Binary size | ~33.6 MB (unstripped) |
| Startup time | ~12ms |

## Audit Reference

Full audit: `docs/audit/2026-05-19-r2go2/README.md`
