# R2Go2 — Claude Code Project Instructions

## Build & Test

```bash
go build -o build/r2go2 .          # Build binary
go test ./pkg/... ./internal/...   # Unit tests (no network)
go vet ./...                        # Static analysis
```

Integration tests (also no network, but slower):
```bash
go test ./tests/integration/... -v -count=1
```

Real R2 integration tests (require credentials, build-tagged):
```bash
go test ./tests/integration/api/real/ -tags=integration
```

## Architecture

- **`pkg/r2go2/`** — Public library. Import as `r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"`. This is the stable API surface.
  - **R2 Storage**: `client.go`, `storage.go`, `upload.go`, `download.go`
  - **Workers**: `worker.go` — WorkerService (Deploy, List, Get, Delete, Logs, UpdateSettings)
  - **KV**: `kv.go` — KVService (namespaces + key-value CRUD)
  - **DNS**: `dns.go` — DNSService (Create, List, Get, Update, Delete) — zone-scoped
  - **Zones**: `zone.go` — ZoneService (Create, List, Get, Delete, GetSettings) — account-scoped
  - **SSL/TLS**: `ssl.go` — SSLService (GetSSL, UpdateSSL, GetVerification, GetSettings, UpdateSettings) — zone-scoped
  - **Cache**: `cloudflare_cache.go` — CacheService (PurgeAll, PurgeByURLs/Tags/Hosts, GetSettings, UpdateSettings) — zone-scoped
- **`cmd/`** — Cobra CLI commands. `root.go` has the root command; each service gets its own file (`bucket.go`, `object.go`, `worker.go`, `kv.go`, `dns.go`, `zone.go`, `ssl.go`, `cache.go`).
- **`internal/`** — Private packages. `config/` (viper-based profiles), `tui/` (Bubble Tea dashboard), `interactive/` (setup wizard), `utils/`.
- **`internal/migration/`** and `cmd_disabled/` — Disabled code. Don't modify unless re-enabling.

## Key Patterns

- Client options use functional options: `r2go2.WithAccountID(...)`, `r2go2.WithAPIToken(...)`.
- All commands support `--json` for machine-readable output.
- Profile struct in `internal/config/config.go` needs `json`, `yaml`, AND `mapstructure` tags (viper uses mapstructure).
- S3 client is concrete `*s3.Client` — no interface injection yet (TASK-002).
- Presigned URLs use AWS SDK v2 presigner with `WithCredentials` (access key + secret key).
- TUI uses Bubble Tea `tea.Model` with `Update`/`View` pattern in `internal/tui/`.

## Testing Conventions

- Tests alongside source: `foo_test.go` next to `foo.go`.
- TUI tests use `newTestModel()` helper in `internal/tui/model_test.go`.
- Integration tests in `tests/integration/` are isolated by package (no network required).
- Upload tests can't use httptest mock because AWS SDK uses virtual-hosted-style paths.

## Known Gaps

- `internal/migration/s3.go:84` — undefined `printInfo` (pre-existing vet error).
- `tests/integration/api/real/` — module import issues (build-tagged, safe to ignore).
- Upload `multipart_test.go` tests multipart logic but no real S3 round-trip.

## Commands Quick Reference

| Command | Description |
|---------|-------------|
| `r2go2 bucket create/list/get/delete` | Bucket management |
| `r2go2 object put/get/delete/copy/head` | Object operations |
| `r2go2 object presign` | Pre-signed URLs |
| `r2go2 object batch` | Batch operations |
| `r2go2 worker deploy/list/get/delete/logs` | Worker management |
| `r2go2 kv namespace create/list/delete` | KV namespaces |
| `r2go2 kv put/get/delete/list` | KV key-value ops |
| `r2go2 dns create/list/get/update/delete` | DNS record management (zone-scoped) |
| `r2go2 zone create/list/get/settings/delete` | Zone management (account-scoped) |
| `r2go2 ssl status/settings/update/verify` | SSL/TLS management (zone-scoped) |
| `r2go2 cache purge/settings` | Cache purge and settings (zone-scoped) |
| `r2go2 config init/set/list/show/switch/export` | Profile management |
| `r2go2 analytics` | Usage statistics |
| `r2go2 compare` | Bucket comparison |
