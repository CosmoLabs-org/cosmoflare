# Entry Points

**Source**: Synthesized from agent-1-code-quality.md, agent-3-api-design.md

## CLI Entry

```
main.go
  -> cmd.Execute()
     -> cmd/root.go: rootCmd (cobra.Command)
        -> PersistentPreRunE: load config, resolve profile
        -> Subcommands: bucket, object, worker, kv, dns, zone, ssl, cache, ...
```

`main.go` is a single-line entry: calls `cmd.Execute()`. All CLI logic lives in `cmd/`.

`cmd/root.go` defines:
- Root command with persistent flags (`--json`, `--profile`, `--account-id`, `--api-token`)
- `PersistentPreRunE` hook that loads configuration and resolves the active profile
- Helper functions: `printError`, `printJSON`, `printSuccess`
- Global flag variables (anti-pattern, see patterns.md)

## Command Tree Entry Points

Each command file registers subcommands in its `init()` function:

| File | Registers | Parent |
|------|-----------|--------|
| `cmd/root.go` | `rootCmd` | (top-level) |
| `cmd/bucket.go` | `bucketCmd` (create, list, get, delete, update) | rootCmd |
| `cmd/object.go` | `objectCmd` (put, get, delete, copy, head, presign, batch) | rootCmd |
| `cmd/worker.go` | `workerCmd` (deploy, list, get, delete, logs, settings) | rootCmd |
| `cmd/kv.go` | `kvCmd` (namespace, put, get, delete, list) | rootCmd |
| `cmd/dns.go` | `dnsCmd` (create, list, get, update, delete) | rootCmd |
| `cmd/zone.go` | `zoneCmd` (create, list, get, settings, delete) | rootCmd |
| `cmd/ssl.go` | `sslCmd` (status, settings, update, verify) | rootCmd |
| `cmd/cache.go` | `cacheCmd` (purge, settings) | rootCmd |
| `cmd/domains.go` | `domainsCmd` | rootCmd |
| `cmd/doctor.go` | `doctorCmd` | rootCmd |
| `cmd/config.go` | `configCmd` (init, set, list, show, switch, export) | rootCmd |
| `cmd/dashboard.go` | `dashboardCmd` | rootCmd |
| `cmd/analytics.go` | `analyticsCmd` | rootCmd |
| `cmd/compare.go` | `compareCmd` | rootCmd |
| `cmd/completion.go` | `completionCmd` | rootCmd |

### Legacy/Alias Commands

These provide backward-compatible entry points (see agent-3-api-design.md):

| File | Command | Aliases For |
|------|---------|-------------|
| `cmd/create.go` | `r2go2 create` | `r2go2 bucket create` |
| `cmd/delete.go` | `r2go2 delete` | `r2go2 bucket delete` / `r2go2 object delete` |
| `cmd/copy.go` | `r2go2 copy` | `r2go2 object copy` |

## TUI Dashboard Entry

```
cmd/dashboard.go
  -> internal/tui/dashboard.go: NewDashboard()
     -> internal/tui/model.go: NewModel() (tea.Model)
        -> tea.NewProgram(model).Run()
           -> Update() loop handles input
           -> View() renders frames
           -> components/navigation/ handles menu
```

The dashboard is a Bubble Tea application. `NewModel()` creates the initial model state. The `Update`/`View` cycle renders the TUI. Currently uses simulated data (see agent-1-code-quality.md, Weakness #5).

## Library Entry Points

For external Go consumers importing `pkg/r2go2/`:

### R2 Storage Client

```go
import r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"

client, err := r2go2.NewClient(
    r2go2.WithAccountID("..."),
    r2go2.WithAPIToken("..."),
)
// client.ListBuckets(), client.PutObject(), etc.
```

### Individual Services

```go
// DNS
dns := r2go2.NewDNSService(accountID, apiToken)
// or
dns := r2go2.NewDNSServiceFromCreds(creds)

// Workers
workers := r2go2.NewWorkerService(accountID, apiToken)

// KV
kv := r2go2.NewKVService(accountID, apiToken)

// Zones
zones := r2go2.NewZoneService(accountID, apiToken)

// SSL
ssl := r2go2.NewSSLService(accountID, apiToken)

// Cache
cache := r2go2.NewCacheService(accountID, apiToken)

// Healthchecks
hc := r2go2.NewHealthcheckService(accountID, apiToken)

// Doctor (diagnostic probes)
doc := r2go2.NewDoctorService()

// Domains
dom := r2go2.NewDomainService(accountID, apiToken)
```

Each service is independently constructible. There is no unified client that provides access to all services (noted as an architectural weakness in agent-1-code-quality.md).

## Configuration Entry

```
internal/config/config.go
  -> LoadConfig() -> reads .cosmoflare.yaml (or .r2go2.yaml legacy)
  -> GetProfile(name) -> returns Profile struct
  -> Profile { AccountID, APIToken, AccessKeyID, SecretAccessKey, Endpoint }
```

The config system uses Viper for file parsing. Profiles are named sections within the YAML config file. The active profile is determined by `--profile` flag or `default_profile` in config.

## Test Entry Points

```bash
go test ./pkg/...              # Library tests
go test ./internal/...         # Internal package tests
go test ./cmd/...              # CLI tests
go test ./tests/integration/...  # Integration tests (no network)
go test ./tests/integration/api/real/ -tags=integration  # Real API tests
```

Test files live alongside source files. `newTestModel()` helper in `internal/tui/model_test.go` provides a pre-configured model for TUI tests.
