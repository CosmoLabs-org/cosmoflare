# CosmoDev-R2Go2

Production-ready CLI tool for managing Cloudflare R2 buckets.

## Project

- **Language**: Go 1.25.3
- **Module**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **Version**: See `.version-registry.json`
- **Binary**: `r2go2`

## Structure

- `cmd/` - CLI commands (cobra)
- `internal/` - Internal packages (api, auth, cli, config, storage, tui, types, utils)
- `docs/` - Documentation, sessions, planning, issues, roadmap

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
