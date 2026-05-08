---
schema_version: 1
created: 2026-05-08
project: CosmoDev-R2Go2
phase: "Phase 1 — Library Extraction & Real API Wiring"
status: ready
---

# Phase 1: Library Extraction & Real API Wiring

## Context

R2Go2 is an open-source Go library + CLI for the full Cloudflare developer platform (R2, Workers, KV, D1, Pages, Queues). Phase 0 bug fixes (BUG-007 through BUG-011) are done and committed to master. The build passes with 145 tests. The project currently has a polished UX shell with stubbed APIs in `internal/api/` and a complete, working S3+Cloudflare implementation disabled in `internal/api_disabled/client.go`.

This session starts **Phase 1**: extracting the public library into `pkg/r2go2/`, wiring the real S3 API, and connecting CLI commands to the new library.

## Session Goals

### Goal 1: Create `pkg/r2go2/` public library structure

Create the importable library package. Each file gets a corresponding `_test.go`.

| File | Purpose |
|------|---------|
| `client.go` | `R2Client` interface + `NewClient` factory with functional options |
| `options.go` | `WithProfile`, `WithBucket`, `WithCacheControl`, `WithHTTPClient`, etc. |
| `types.go` | `Bucket`, `Object`, `UploadResult`, `DownloadResult`, `ListResult` |
| `errors.go` | `R2Error`, `R2NotFoundError`, `R2AuthError`, `R2QuotaError`, `R2AccessDeniedError` |
| `config.go` | `ProjectConfig` (`.r2go2.yaml` loader) + `MachineConfig` (`~/.r2go2/` loader) |
| `storage.go` | Bucket CRUD (Create, List, Get, Delete) + Object CRUD (List, Get, Head, Delete) |
| `upload.go` | Upload with progress tracking + auto cache header injection |
| `download.go` | Download operations |
| `cache.go` | 5-tier cache policy engine (immutable, long-static, static, dynamic, no-cache) |
| `guardrails.go` | Upload validation, access scoping, quota checks |
| `audit.go` | JSONL audit logging to `.r2go2/audit.log` |

Stub files for future phases (empty, with doc comments):
| `worker.go` | [Phase 3] Workers service |
| `kv.go` | [Phase 3] KV namespace/key-value operations |
| `d1.go` | [Phase 4] D1 database operations |
| `pages.go` | [Phase 4] Pages project operations |
| `queue.go` | [Phase 5] Queues operations |

### Goal 2: Wire real S3 API

Migrate the working implementation from `internal/api_disabled/client.go` into the new library:
- Cloudflare API client for bucket management (Create, List, Delete)
- AWS S3 SDK v2 for object operations (List, Get, Put, Delete, Head)
- Profile-based and env-based auth with proper credential chain
- Functional options pattern for client configuration

### Goal 3: Wire CLI commands to new library

Update each `cmd/*.go` to import and use `pkg/r2go2` instead of `internal/api` stubs:
- `cmd/root.go` — Initialize `R2Client` from profile + project config
- `cmd/bucket.go` — Use `R2Client.ListBuckets()`, `CreateBucket()`, `DeleteBucket()`
- `cmd/list.go` — Use `R2Client.ListObjects()` with `--json` output
- `cmd/object.go` — Use `R2Client.GetObject()`, `HeadObject()`, `DeleteObject()`
- `cmd/create.go` — Use `R2Client.CreateBucket()`
- `cmd/delete.go` — Use `R2Client.DeleteBucket()` / `DeleteObject()`
- `cmd/copy.go` — Use `R2Client.S3()` for `CopyObject`

## Execution Order

```
1. P1-1  pkg/r2go2/ skeleton (client, options, types, errors) + tests
2. P1-3  config.go (ProjectConfig + MachineConfig loaders) + tests
3. P1-2  S3 implementation migrated from internal/api_disabled/ into storage.go, upload.go, download.go
4. P1-4  cache.go (5-tier cache engine) + tests
5. P1-5  guardrails.go (validation, access scoping, quota) + tests
6. P1-6  audit.go (JSONL audit logging) + tests
7. P1-7  Wire all CLI commands to pkg/r2go2/
8. P1-8  convenience.go (package-level one-liner functions)
9. P1-9  Full test suite for pkg/r2go2/
```

Steps 1-6 can be parallelized after step 1 completes. Steps 7-9 are sequential.

## Success Criteria

- [ ] `go build -o r2go2 .` compiles clean
- [ ] `go vet ./...` passes
- [ ] `go test ./pkg/r2go2/...` passes (unit tests with mocks)
- [ ] `r2go2 bucket list --json` returns real R2 buckets
- [ ] `r2go2 upload logo.png` auto-applies cache headers from `.r2go2.yaml`
- [ ] Upload validation rejects files that violate project rules
- [ ] Access scoping prevents operations on non-declared buckets
- [ ] Audit log entries written for every upload/delete
- [ ] External project can `import "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"`
- [ ] No CCS dependencies in `pkg/r2go2/` (library-first design)

## Key Constraints

- **Library-first**: `pkg/r2go2/` must have zero CCS dependencies. It must be importable by any Go project.
- **Agent-friendly**: Every CLI command supports `--json` output. Error messages include what failed, why, and how to fix.
- **Existing tests**: Do not break the 145 passing tests. Run `go test ./...` after each major change.
- **No CCS dependencies**: `pkg/r2go2/` imports only standard library + aws-sdk-go-v2 + cloudflare-go.
- **Module path**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`

## Roadmap Items Filed

ROAD-025 through ROAD-033 are filed for this and future phases. Check `docs/roadmap/index.yaml` for details.

## Bugs Fixed This Session

BUG-007 through BUG-011 were fixed and committed in the previous session. See `docs/issues/` for details.
