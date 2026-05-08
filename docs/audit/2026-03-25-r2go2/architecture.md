# R2Go2 Architecture

## Layers

```
main.go
  -> cmd/root.go (Cobra root, global flags, PersistentPreRun)
       -> cmd/*.go (17 subcommands: auth, bucket, object, config, etc.)
            -> internal/api/ (R2 API client layer — currently stubs)
            -> internal/config/ (Profile/credential management via Viper)
            -> internal/interactive/ (Terminal input, validation, themes)
            -> internal/tui/ (Bubbletea dashboard: model.go, update.go, view.go)
            -> internal/cli/ (UX layer: progress, batch, visual, retry)
            -> internal/utils/ (Formatting utilities)
```

## Package Responsibilities

| Package | Role | Status |
|---------|------|--------|
| `cmd/` | CLI commands via Cobra | Active, 17 commands |
| `internal/api/` | R2 API operations (Client + EnhancedClient) | **Stub implementations** |
| `internal/config/` | Config file management (~/.r2go2/config.yaml) | Working |
| `internal/tui/` | Bubbletea interactive dashboard | Working |
| `internal/interactive/` | Setup wizards, input handling, themes | Working |
| `internal/cli/visual/` | Progress bars, animations, spinners | Working |
| `internal/cli/ux/` | Retry logic, error classification | Working |
| `internal/cli/batch/` | Batch operation manager | Working |
| `internal/cli/progress/` | Upload progress tracking | Working |
| `internal/utils/` | Format helpers (bytes, mask IDs) | Working |
| `internal/webhook/` | Webhook manager | Scaffolded only |
| `internal/migration/` | S3 migration from other providers | **Broken build** |
| `*_disabled/` | Disabled features (analytics, domain, old API) | **Broken, should be removed** |

## Data Flow

```
User Input
  -> Cobra Command (cmd/*.go)
     -> PersistentPreRun: validateEnvironment() + load AccountID
        [BUG: blocks non-API commands]
     -> Command RunE function
        -> NewClient/NewClientFromProfile
           -> ConfigManager.GetProfile() or GetCurrent()
           -> Client.CreateBucket() / ListBuckets() / etc.
              [ALL STUBS - return mock data]
        -> EnhancedClient.UploadFile()
           -> singlePartUpload() or multipartUpload()
              -> s3.PutObject / CreateMultipartUpload
              [Real S3 SDK calls, but base client init is incomplete]
```

## Dependency Graph

```
cmd/ depends on: internal/api, internal/config, internal/interactive, internal/tui, internal/utils
internal/api depends on: internal/config, internal/cli/visual, internal/utils, aws-sdk-go-v2
internal/tui depends on: internal/api, bubbletea, lipgloss
internal/interactive depends on: internal/config, internal/utils, bubbletea, lipgloss
internal/cli/* depends on: internal/utils
internal/config depends on: internal/utils, viper
```

## Key Dependencies

- **Cobra v1.10.1** — CLI framework
- **Bubbletea v1.3.10 + Lipgloss v1.1** — TUI framework
- **AWS SDK Go v2** — S3-compatible API (R2 uses S3 protocol)
- **Viper v1.21** — Configuration management
- **Cloudflare-go v0.116** — Cloudflare API (imported but not heavily used)
