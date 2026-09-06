---
branch: r2go2-phase0-phase1-implementation
completed: "2026-05-29T00:00:00-03:00"
created: "2026-05-07T12:00:00-03:00"
goals_completed: 15
goals_total: 15
origin: migrated by ccs prompts migrate
priority: medium
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: R2Go2 Phase 0 + Phase 1 Implementation Plan
deliverables:
  - P-01: Phase 0 critical bug fixes and Phase 1 library extraction with real Cloudflare API integration
---

# R2Go2 Phase 0 + Phase 1 Implementation Plan

**Date**: 2026-05-07
**Status**: APPROVED
**From brainstorm**: `docs/brainstorming/2026-03-28-r2go2-product-vision-and-integration.md`

## Deliverables

- P0-1: Speed calc fix in `internal/api/enhanced_client.go`
- P0-2: PersistentPreRun fix in `cmd/root.go`
- P0-3: printErrorAndExit fix in `cmd/list.go`
- P0-4: Keyboard off-by-one fix in `internal/tui/components/navigation/menu_selection.go`
- P0-5: MIT LICENSE file
- P0-6: install.sh binary name fix
- P1-1: `pkg/r2go2/` skeleton (client, options, types, errors)
- P1-2: S3 implementation migrated from `internal/api_disabled/client.go`
- P1-3: `.r2go2.yaml` config loader in `pkg/r2go2/config.go`
- P1-4: 5-tier cache engine in `pkg/r2go2/cache.go`
- P1-5: Guardrails (validation, access scoping, quota) in `pkg/r2go2/guardrails.go`
- P1-6: Audit logging in `pkg/r2go2/audit.go`
- P1-7: All CLI commands wired to `pkg/r2go2/`
- P1-8: High-level convenience API in `pkg/r2go2/convenience.go`
- P1-9: Test suite for all `pkg/r2go2/` packages

## Goal

Transform R2Go2 from a polished UX shell with stubbed APIs into a fully functional open-source Go library + CLI for the full Cloudflare developer platform (R2, Workers, KV, D1, Pages, Queues). Near-term focus (Phase 0-1): R2 storage with real API wiring. Future phases expand to the remaining services.

## Phase 0: Critical Bug Fixes

Quick one-liner fixes that unblock everything else. All files exist, changes are surgical.

### P0-1: Speed calculation Inf/NaN (BUG-001)
- **File**: `internal/api/enhanced_client.go:198,277`
- **Fix**: Guard division by zero in speed calculation. Check elapsed time > 0 before dividing bytes/seconds.

### P0-2: PersistentPreRun blocks setup (BUG-002)
- **File**: `cmd/root.go:63`
- **Fix**: Skip API validation for non-API commands (setup, config init, auth login, completion, help). Commands that don't need an R2 connection shouldn't fail if credentials are missing.

### P0-3: printErrorAndExit doesn't exit
- **File**: `cmd/list.go:130`
- **Fix**: Add `os.Exit(1)` after printing the error message.

### P0-4: Keyboard off-by-one (BUG-004)
- **File**: `internal/tui/components/navigation/menu_selection.go:170`
- **Fix**: Correct the index calculation for menu item selection.

### P0-5: Missing LICENSE file (BUG-010)
- **File**: Create `LICENSE` (MIT)
- **Content**: Standard MIT license with Copyright CosmoLabs

### P0-6: install.sh binary name mismatch
- **File**: Check `scripts/install.sh` or equivalent
- **Fix**: Ensure the install script references `r2go2` not `r2-go2` or other variants.

**Verification**: `go build -o r2go2 . && go vet ./...`

---

## Phase 1: Real API + Library Extraction

### P1-1: Create `pkg/r2go2/` public library skeleton

Create the public package structure. This is the importable library.

**Files to create**:
```
pkg/r2go2/
  client.go       — R2Client interface + NewClient factory
  options.go      — Functional options (WithProfile, WithBucket, WithCacheControl, etc.)
  types.go        — Bucket, Object, UploadResult, DownloadResult, ListResult
  errors.go       — R2Error, R2NotFoundError, R2AuthError, R2QuotaError, R2AccessDeniedError
  config.go       — ProjectConfig (.r2go2.yaml loader) + MachineConfig (~/.r2go2/ loader)
  cache.go        — CachePolicy engine (5-tier rule-based)
  storage.go      — Storage operations: Bucket CRUD + Object CRUD
  upload.go       — Upload with progress + auto cache headers
  download.go     — Download operations
  guardrails.go   — Upload validation, access scoping, quota checks
  audit.go        — Audit logging
  worker.go       — [Phase 3] Workers service (deploy, list, logs, tail)
  kv.go           — [Phase 3] KV namespace/key-value operations
  d1.go           — [Phase 4] D1 database operations (create, query, migrations)
  pages.go        — [Phase 4] Pages project operations (deploy, list, aliases)
  queue.go        — [Phase 5] Queues operations (create, send, consume)
```

**Key interfaces** (`client.go`):
```go
type R2Client interface {
    // Bucket operations
    CreateBucket(ctx context.Context, name string, opts ...Option) (*Bucket, error)
    ListBuckets(ctx context.Context, opts ...Option) ([]*Bucket, error)
    GetBucket(ctx context.Context, name string) (*Bucket, error)
    DeleteBucket(ctx context.Context, name string) error

    // Object operations
    ListObjects(ctx context.Context, bucket, prefix string, opts ...Option) (*ListResult, error)
    GetObject(ctx context.Context, bucket, key string) (*DownloadResult, error)
    HeadObject(ctx context.Context, bucket, key string) (*Object, error)
    DeleteObject(ctx context.Context, bucket, key string) error

    // Upload/Download
    Upload(ctx context.Context, bucket, key string, reader io.Reader, opts ...Option) (*UploadResult, error)
    Download(ctx context.Context, bucket, key string, w io.Writer, opts ...Option) error

    // Raw S3 access
    S3() *s3.Client
}
```

### P1-2: Extract real S3 implementation from `internal/api_disabled/`

The disabled API at `internal/api_disabled/client.go` has a complete S3 + Cloudflare implementation:
- Cloudflare API client for bucket management (Create, List, Delete)
- S3 SDK client for object operations (List, Get, Put, Delete, Head)
- Profile-based + env-based auth
- Proper credential chain (profile keys > CF API token > env vars)

**Migration plan**:
1. Copy the working S3 init logic (`initS3Client`) into `pkg/r2go2/client.go`
2. Copy bucket CRUD + object methods into `pkg/r2go2/storage.go`
4. Refactor to use `R2Client` interface instead of concrete struct
5. Add functional options pattern for configuration
6. Keep `internal/api/` for backward compat during migration, deprecation notice in comments

### P1-3: Implement project config (`.r2go2.yaml`) loader

**File**: `pkg/r2go2/config.go`

Parse `.r2go2.yaml` walking up from CWD to repo root (like `.gitignore` discovery).

**Config struct**:
```go
type ProjectConfig struct {
    Profile   string         `yaml:"profile"`
    Bucket    string         `yaml:"bucket"`
    Cache     CacheConfig    `yaml:"cache"`
    Upload    UploadConfig   `yaml:"upload"`
    Access    AccessConfig   `yaml:"access"`
    Quotas    QuotaConfig    `yaml:"quotas"`
    Lifecycle []LifecycleRule `yaml:"lifecycle"`
    Workers   []WorkerConfig `yaml:"workers"`
}
```

### P1-4: Implement 5-tier cache policy engine

**File**: `pkg/r2go2/cache.go`

```go
type CacheTier string
const (
    CacheImmutable  CacheTier = "immutable"   // max-age=31536000, immutable
    CacheLongStatic CacheTier = "long-static" // max-age=2592000
    CacheStatic     CacheTier = "static"      // max-age=86400 (default)
    CacheDynamic    CacheTier = "dynamic"     // max-age=3600
    CacheNoCache    CacheTier = "no-cache"    // no-store
)

func ResolveCachePolicy(key string, cfg *CacheConfig) string
// Walks rules in order, returns first match Cache-Control header value
// Falls back to default tier if no rules match
```

### P1-5: Implement guardrails (upload validation + access scoping)

**File**: `pkg/r2go2/guardrails.go`

```go
func ValidateUpload(key string, size int64, contentType string, cfg *UploadConfig) error
func CheckAccessScope(bucket string, cfg *AccessConfig) error
func CheckQuota(ctx context.Context, bucket string, additionalBytes int64, cfg *QuotaConfig) error
```

These run before every upload. `Upload()` calls `ValidateUpload` + `CheckAccessScope` + `CheckQuota` + resolves cache policy, then calls S3 `PutObject`.

### P1-6: Implement audit logging

**File**: `pkg/r2go2/audit.go`

Local JSONL file at `.r2go2/audit.log` (gitignored). Every upload/delete/copy logged with timestamp, action, bucket, key, size, cache-control, source.

### P1-7: Wire CLI commands to `pkg/r2go2/`

Update each `cmd/*.go` to use the public library instead of `internal/api`:

| Command | Changes |
|---------|---------|
| `cmd/bucket.go` | Import `pkg/r2go2`, use `R2Client.ListBuckets()`, `CreateBucket()`, `DeleteBucket()` |
| `cmd/list.go` | Use `R2Client.ListObjects()` with `--json` output |
| `cmd/object.go` | Use `R2Client.GetObject()`, `HeadObject()`, `DeleteObject()` |
| `cmd/create.go` | Use `R2Client.CreateBucket()` |
| `cmd/delete.go` | Use `R2Client.DeleteBucket()` or `DeleteObject()` |
| `cmd/copy.go` | Use `R2Client.S3()` for `CopyObject` |
| `cmd/auth.go` | Validate credentials via `R2Client.TestConnection()` |
| `cmd/config.go` | Load/display machine config + project config |
| `cmd/root.go` | Initialize `R2Client` from profile + project config |

**Pattern for each command**:
```go
// cmd/bucket.go — ListBuckets
func runBucketList(cmd *cobra.Command, args []string) error {
    client, err := r2go2.NewClient(r2go2.FromConfig())
    if err != nil {
        return err
    }
    buckets, err := client.ListBuckets(cmd.Context())
    if err != nil {
        return err
    }
    if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
        return json.NewEncoder(os.Stdout).Encode(buckets)
    }
    // Table output
    printBucketTable(buckets)
    return nil
}
```

### P1-8: High-level convenience API

**File**: `pkg/r2go2/convenience.go`

```go
// Package-level functions that auto-load config from .r2go2.yaml
func Upload(path string, opts ...Option) (*UploadResult, error)
func Download(key string, dest string, opts ...Option) error
func List(opts ...Option) (*ListResult, error)
```

These are the "agent-friendly" one-liners. They auto-discover `.r2go2.yaml`, load the profile, apply cache rules, validate upload, and execute.

### P1-9: Tests

Every `pkg/r2go2/*.go` file gets a corresponding `_test.go`:

| File | Tests |
|------|-------|
| `config_test.go` | Parse `.r2go2.yaml`, walk directory tree, merge with machine config |
| `cache_test.go` | Each tier produces correct header, rule matching order, default fallback |
| `guardrails_test.go` | Upload validation (size, type, path), access scoping, quota checks |
| `client_test.go` | NewClient with options, profile loading, env loading |
| `bucket_test.go` | Mock S3/CF responses for bucket + object CRUD |
| `upload_test.go` | Cache header injection, progress tracking, multipart threshold |
| `audit_test.go` | Log format, append, search |
| `errors_test.go` | Error types, wrapping, Is/As |

**Mock strategy**: Interface-based mocking. `R2Client` interface allows `MockClient` for unit tests. Integration tests use `//go:build integration` tag against real R2.

---

## Execution Order

```
Phase 0 (1 day, sequential)
  P0-1 → P0-2 → P0-3 → P0-4 → P0-5 → P0-6 → build + vet

Phase 1 (1-2 weeks, can parallelize after P1-1)
  P1-1 (pkg skeleton) ──┬── P1-2 (S3 migration)
                         ├── P1-3 (config loader)
                         ├── P1-4 (cache engine)
                         ├── P1-5 (guardrails)
                         └── P1-6 (audit logging)
  Then sequential:
  P1-7 (wire CLI) → P1-8 (convenience API) → P1-9 (tests)

Phase 3-5 (future, after Phase 1 ships)
  Phase 3 (Workers + KV) → Phase 4 (D1 + Pages) → Phase 5 (Queues)
```

## Phase 3-5: Full Cloudflare Platform Expansion

Phase 0 and Phase 1 deliver R2 storage. Future phases expand R2Go2 to cover the entire Cloudflare developer platform.

### Phase 3: Workers + KV
**Workers** (`pkg/r2go2/worker.go`): Deploy scripts, list workers, tail logs, manage secrets, versioning/rollback
**KV** (`pkg/r2go2/kv.go`): Namespace CRUD, key-value get/put/list/delete, bulk write, worker bindings
**CLI**: `r2go2 worker deploy|list|logs|secrets`, `r2go2 kv namespace|put|get|list|delete`

### Phase 4: D1 + Pages
**D1** (`pkg/r2go2/d1.go`): Database CRUD, SQL query execution, migration management, export/import
**Pages** (`pkg/r2go2/pages.go`): Project deployment, list deployments, alias management, rollback
**CLI**: `r2go2 d1 create|query|migrate|export`, `r2go2 pages deploy|list|aliases`

### Phase 5: Queues
**Queues** (`pkg/r2go2/queue.go`): Queue CRUD, message send/batch/receive/ack, dead-letter management, consumer config
**CLI**: `r2go2 queue create|send|receive|list|delete`

### Cross-cutting concerns for Phase 3-5
- All services follow the `R2Client` interface pattern (service methods return service-specific interfaces)
- `.r2go2.yaml` gains service-specific config sections (`workers:`, `kv:`, `d1:`, `pages:`, `queues:`)
- Every command supports `--json` output (agent-friendly)
- Guardrails extend to each service (D1 query limits, Worker script size, etc.)
- Audit logging covers all service operations

## Success Criteria

- [x] `go build -o r2go2 .` compiles clean
- [x] `go vet ./...` passes
- [x] `go test ./pkg/r2go2/...` passes (unit tests with mocks)
- [x] `r2go2 bucket list --json` returns real R2 buckets
- [x] `r2go2 upload logo.png` auto-applies cache headers from `.r2go2.yaml`
- [x] Upload validation rejects files that violate project rules
- [x] Access scoping prevents operations on non-declared buckets
- [x] Audit log entries written for every upload/delete
- [x] External project can `import "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"`
- [x] Library architecture supports adding Workers, KV, D1, Pages, Queues without breaking changes
