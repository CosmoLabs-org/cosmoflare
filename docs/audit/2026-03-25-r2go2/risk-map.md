# R2Go2 Risk Map

## Critical Risk: Non-Functional Core

**Severity**: CRITICAL | **Probability**: Certain | **Impact**: Product is unusable

The entire `internal/api/client.go` consists of placeholder implementations. Every method (`ListBuckets`, `CreateBucket`, `DeleteBucket`, `GetBucket`, `ListObjects`, `GetObject`, `DeleteObject`, `HeadObject`, `BucketExists`) returns mock/empty data. A user installing R2Go2 would find that no commands actually work against Cloudflare R2.

**Affected area**: 9 methods across 225 lines in `internal/api/client.go`
**Mitigation**: ROAD-000 (priority 95) tracks this. Wire up the existing `s3.Client` field on the `Client` struct.

## High Risk: Speed Calculation Bug

**Severity**: HIGH | **Probability**: Every upload | **Impact**: Incorrect metrics, potential NaN/Inf in JSON output

`enhanced_client.go:198` and `:277` both compute speed as:
```go
speed := float64(fileSize) / time.Since(time.Now()).Seconds() / (1024 * 1024)
```
`time.Since(time.Now())` returns ~0 nanoseconds. Division by ~0 seconds produces +Inf or NaN.

**Fix**: Store `startTime := time.Now()` at the beginning of the upload function and use it in the calculation.

## High Risk: PersistentPreRun Blocks Non-API Commands

**Severity**: HIGH | **Probability**: Every first-run | **Impact**: Cannot run `r2go2 --help`, `r2go2 completion`, `r2go2 setup` without credentials

The root command's `PersistentPreRun` calls `validateEnvironment()` which requires `CLOUDFLARE_API_TOKEN`. This runs before ALL subcommands, including those that don't need API access. First-time users cannot even see help output.

**Fix**: Check `cmd.Name()` or use `PreRun` on individual commands instead of `PersistentPreRun`.

## Medium Risk: Plaintext Credential Storage

**Severity**: MEDIUM | **Probability**: Conditional | **Impact**: Credential exposure if filesystem compromised

Config file `~/.r2go2/config.yaml` stores API tokens and secret keys in plaintext. While file permissions are set to 0600, this provides no protection against:
- Backup tools that don't preserve permissions
- Disk imaging/cloning
- Process memory dumps
- Other users with root access

**Mitigation options**: macOS Keychain integration, Linux secret-service, encrypted config with master password.

## Medium Risk: Broken Disabled Packages

**Severity**: MEDIUM | **Probability**: Every build/test cycle | **Impact**: CI noise, contributor confusion

6 packages fail to build:
- `cmd_disabled/` — two conflicting package declarations
- `internal/analytics_disabled/` — undefined references
- `internal/api_disabled/` — undefined references
- `internal/domain_disabled/` — undefined references
- `internal/migration/` — undefined `printInfo`, `printWarning`, wrong `config.WithSharedCredentialsFiles` signature
- `internal/migration_disabled/` — build failure

These create noise in `go test ./...` output and confuse contributors.

## Medium Risk: Large Binaries in Repository

**Severity**: MEDIUM | **Probability**: Every clone | **Impact**: Slow clones, bloated repo

- `simple-setup` (4.8MB) — tracked in git
- `test-setup` (4.8MB) — tracked in git
- `r2go2-enhanced` (15.9MB) — untracked but present
- `CosmoDev-R2Go2` (16.2MB) — untracked but present

The tracked binaries inflate every `git clone` permanently. The untracked ones should be gitignored.

## Low Risk: CI Configuration Issues

- `test-suite.yml` references Go 1.26 which does not exist
- `build.yml` runs `go test ./...` without excluding broken packages
- `release.yml` uses deprecated `actions/create-release@v1`
- No golangci-lint in CI pipeline

## Fragility Map (High Complexity + Low Test Coverage)

| File/Package | Complexity | Test Coverage | Risk |
|-------------|-----------|---------------|------|
| `internal/api/client.go` | Low (stubs) | Tested against stubs | HIGH — false confidence |
| `internal/api/enhanced_client.go` | Medium | Integration tests exist | MEDIUM — speed bug |
| `internal/interactive/` | High (14 files) | Medium coverage | MEDIUM — many UI flows |
| `internal/tui/model.go` | Medium | Good (78.8%) | LOW |
| `cmd/*.go` | Medium (17 files) | **No cmd-level tests** | HIGH |
| `internal/config/config.go` | Low | **No test files** | MEDIUM |
