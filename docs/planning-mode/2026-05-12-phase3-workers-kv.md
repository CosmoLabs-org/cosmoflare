---
deliverables:
  - id: P-01
    title: "WorkerService interface and implementation (pkg/r2go2/worker.go)"
  - id: P-02
    title: "Worker unit tests (pkg/r2go2/worker_test.go)"
  - id: P-03
    title: "Worker CLI commands (cmd/worker.go) registered in root"
  - id: P-04
    title: "KVService interface and implementation (pkg/r2go2/kv.go)"
  - id: P-05
    title: "KV unit tests (pkg/r2go2/kv_test.go)"
  - id: P-06
    title: "KV CLI commands (cmd/kv.go) registered in root"
  - id: P-07
    title: "Update USAGE.md and CLAUDE.md for Workers/KV"
  - id: P-08
    title: "Version bump to v0.4.0"
---

# Phase 3: Workers and KV Service Implementation

**Date**: 2026-05-12
**Predecessor**: Phase 2 (library extraction, bug resolution, hardening)
**Target**: v0.4.0

## Goal

Implement the Workers (compute) and KV (key-value) services in `pkg/r2go2/`, expanding R2Go2 beyond R2 storage into the full Cloudflare developer platform.

## Current State

- `pkg/r2go2/worker.go` -- empty struct `WorkerService{}`
- `pkg/r2go2/kv.go` -- empty struct `KVService{}`
- Library architecture established: `R2Client` interface, functional options, error hierarchy
- S3-compatible client wired via `aws-sdk-go-v2`
- Cloudflare API client wired via `cloudflare-go`

## Workers Service

### Public API Surface

```go
type WorkerService interface {
    Deploy(ctx context.Context, name string, script io.Reader, opts ...WorkerOption) (*Worker, error)
    List(ctx context.Context) ([]*Worker, error)
    Get(ctx context.Context, name string) (*Worker, error)
    Delete(ctx context.Context, name string) error
    Logs(ctx context.Context, name string, opts ...LogOption) ([]*LogEntry, error)
    UpdateSettings(ctx context.Context, name string, settings WorkerSettings) error
}
```

### Types

```go
type Worker struct {
    Name        string            `json:"name"`
    Modified    time.Time         `json:"modified"`
    Size        int64             `json:"size"`
    Runtime     string            `json:"runtime"`
    Bindings    []WorkerBinding   `json:"bindings,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

type WorkerBinding struct {
    Name string `json:"name"`
    Type string `json:"type"` // "kv", "r2", "d1", "queue", "var"
    ID   string `json:"id"`
}

type WorkerSettings struct {
    CompatibilityDate string `json:"compatibility_date"`
    UsageModel        string `json:"usage_model"` // "bundled" or "unbound"
}

type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Event     string    `json:"event"`
}
```

### Cloudflare API Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Deploy/Update | PUT | `/accounts/{account_id}/workers/scripts/{name}` |
| List | GET | `/accounts/{account_id}/workers/scripts` |
| Get | GET | `/accounts/{account_id}/workers/scripts/{name}` |
| Delete | DELETE | `/accounts/{account_id}/workers/scripts/{name}` |
| Settings | PATCH | `/accounts/{account_id}/workers/scripts/{name}/settings` |
| Logs (tail) | WebSocket | `/accounts/{account_id}/workers/scripts/{name}/tails` |

Implementation notes:
- `cloudflare-go` has `workers` package with script management methods
- Deploy uses `PutWorkerScript` with `cloudflare.WorkerScriptParams`
- Bindings are configured via `PutWorkerSettings` (routes, KV namespaces, R2 buckets)
- Logs use WebSocket-based tail API or REST-based analytics

### Options

```go
func WithWorkerCompatibilityDate(date string) WorkerOption
func WithWorkerBindings(bindings []WorkerBinding) WorkerOption
func WithWorkerTags(tags []string) WorkerOption
func WithLogLimit(n int) LogOption
func WithLogSince(t time.Time) LogOption
```

### CLI Commands

```
r2go2 worker deploy <name> --script=worker.js
r2go2 worker list
r2go2 worker get <name>
r2go2 worker delete <name>
r2go2 worker logs <name> --follow
r2go2 worker settings <name> --compatibility-date=2024-01-01
```

## KV Service

### Public API Surface

```go
type KVService interface {
    CreateNamespace(ctx context.Context, title string) (*KVNamespace, error)
    ListNamespaces(ctx context.Context) ([]*KVNamespace, error)
    GetNamespace(ctx context.Context, id string) (*KVNamespace, error)
    DeleteNamespace(ctx context.Context, id string) error

    Put(ctx context.Context, namespaceID, key string, value io.Reader, opts ...KVOption) error
    Get(ctx context.Context, namespaceID, key string) ([]byte, error)
    Delete(ctx context.Context, namespaceID, key string) error
    ListKeys(ctx context.Context, namespaceID string, opts ...KVListOption) (*ListResult[*KVKey], error)
}
```

### Types

```go
type KVNamespace struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type KVKey struct {
    Key       string `json:"key"`
    Expiration int64 `json:"expiration,omitempty"`
}
```

### Cloudflare API Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create NS | POST | `/accounts/{account_id}/storage/kv/namespaces` |
| List NS | GET | `/accounts/{account_id}/storage/kv/namespaces` |
| Get NS | GET | `/accounts/{account_id}/storage/kv/namespaces/{id}` |
| Delete NS | DELETE | `/accounts/{account_id}/storage/kv/namespaces/{id}` |
| Put key | PUT | `/accounts/{account_id}/storage/kv/namespaces/{id}/values/{key}` |
| Get key | GET | `/accounts/{account_id}/storage/kv/namespaces/{id}/values/{key}` |
| Delete key | DELETE | `/accounts/{account_id}/storage/kv/namespaces/{id}/values/{key}` |
| List keys | GET | `/accounts/{account_id}/storage/kv/namespaces/{id}/keys` |

Implementation notes:
- `cloudflare-go` has KV management via `CreateWorkersKVNamespace`, `ListWorkersKVNamespaces`, etc.
- Value reads/writes use the Cloudflare API (not S3-compatible)
- Supports TTL via `expiration_ttl` parameter
- Supports metadata on keys

### Options

```go
func WithKVTTL(seconds int64) KVOption
func WithKVMetadata(m map[string]string) KVOption
func WithKVPrefix(prefix string) KVListOption
func WithKVLimit(n int) KVListOption
```

### CLI Commands

```
r2go2 kv namespace create <title>
r2go2 kv namespace list
r2go2 kv namespace delete <id>
r2go2 kv put <namespace-id> <key> --value="data" --file=data.json
r2go2 kv get <namespace-id> <key>
r2go2 kv delete <namespace-id> <key>
r2go2 kv list <namespace-id> --prefix=cache/
```

## Architecture Decisions

1. **Separate service interfaces** -- Workers and KV are distinct services with their own clients, not part of `R2Client`. Each service gets its own constructor (`NewWorkerService`, `NewKVService`) sharing the same `cloudflare-go` API client and account ID.

2. **Shared credentials** -- All services share `CLOUDFLARE_ACCOUNT_ID` and `CLOUDFLARE_API_TOKEN`. Services can be created independently or from a shared config.

3. **CLI subcommand groups** -- `r2go2 worker` and `r2go2 kv` as top-level command groups, mirroring the Cloudflare dashboard structure.

4. **Error hierarchy reuse** -- Workers and KV errors use the same `R2Error` hierarchy (renaming to `CFError` would be a breaking change, so we keep the prefix).

## Implementation Order

1. `pkg/r2go2/worker.go` -- WorkerService interface + implementation
2. `pkg/r2go2/worker_test.go` -- Unit tests for worker types/options
3. `cmd/worker.go` -- CLI commands
4. `pkg/r2go2/kv.go` -- KVService interface + implementation
5. `pkg/r2go2/kv_test.go` -- Unit tests for KV types/options
6. `cmd/kv.go` -- CLI commands
7. Update `CLAUDE.md` and `docs/USAGE.md`

## Files to Create/Modify

| File | Action |
|------|--------|
| `pkg/r2go2/worker.go` | Replace stub with full implementation |
| `pkg/r2go2/worker_test.go` | New |
| `pkg/r2go2/kv.go` | Replace stub with full implementation |
| `pkg/r2go2/kv_test.go` | New |
| `cmd/worker.go` | New |
| `cmd/kv.go` | New |
| `cmd/root.go` | Register worker and kv commands |
| `CLAUDE.md` | Update structure section |
