---
title: "FEAT-021 Workers Command Depth — Implementation Plan"
created: 2026-09-15
updated: 2026-09-16T00:02:19+04:00
status: IN_PROGRESS
branch: master
brainstorm_ref: docs/brainstorming/2026-09-15-feat021-workers-depth.md
tags: [plan, workers, cli, feat-021]
deliverables:
  - id: P-01
    title: "worker_secrets.go: Put/Delete/List/Bulk + tests + 4 commands"
  - id: P-02
    title: "worker_routes.go: zone-scoped CRUD + tests + 4 commands"
  - id: P-03
    title: "worker_versions.go: raw-REST versions client + tests + 6 commands"
  - id: P-04
    title: "worker_deployments.go: list/view/rollback + tests + 3 commands"
  - id: P-05
    title: "worker_domains.go: attach/detach/list + tests + 3 commands"
  - id: P-06
    title: "worker_subdomain.go: get/set + tests + 2 commands"
  - id: P-07
    title: "worker_cron.go: CRUD via Update + tests + 4 commands"
  - id: P-08
    title: "bindings list + tail commands (library reuse)"
  - id: P-09
    title: "worker types: .d.ts generator (pure) + command + tests"
  - id: P-10
    title: "Integration: cmd/worker.go wiring, --help sweep, acceptance gates"
---

# FEAT-021 Implementation Plan

Goal: full corpus workers-block coverage — the `worker` tree reaches 35
subcommands (6 existing + 29 new), all with `--json`, library-first,
behavior gated by tests. Design: see
`docs/brainstorming/2026-09-15-feat021-workers-depth.md` (transport split,
registration-func isolation, rollback semantics).

Every task follows the same shape (repeat per group):

1. Library file `pkg/cosmoflare/worker_<group>.go`: methods on
   `*WorkerService`, typed errors via `validationError`/`newError`, struct
   types with JSON tags, options where needed.
2. Contract tests `pkg/cosmoflare/worker_<group>_test.go`: httptest mock
   server (pattern: `worker_test.go`), happy path + validation + API error.
3. Command file `cmd/worker_<group>.go`: `registerWorker<Group>Cmds(parent
   *cobra.Command)` + run funcs using Presenter helpers; `--json` everywhere;
   detailed `--help` with examples.
4. Command tests `cmd/worker_<group>_run_test.go`: offline validation paths
   (exemplar `cmd/cache_run_test.go`).

## Phase 1 — Wave 1 (dispatched this session)

### Task 1 (P-01) — Secrets
Files: `pkg/cosmoflare/worker_secrets.go`, `..._test.go`,
`cmd/worker_secrets.go`, `cmd/worker_secrets_run_test.go`.
- `SecretPut(ctx, name, key, value)`, `SecretDelete(ctx, name, key)`,
  `SecretList(ctx, name) ([]SecretInfo, error)` (names only),
  `SecretsBulk(ctx, name, map[string]string) ([]SecretBulkResult, error)`
  (one entry per key: name + error).
- cloudflare-go: `SetWorkersSecret`/`DeleteWorkersSecret`/
  `ListWorkersSecrets`. Bulk loops Put, collects per-key errors.
- Commands: `worker secret put KEY [--value | stdin]`, `secret delete KEY`,
  `secret list [name]`, `secret bulk FILE` (JSON object).
- NEVER print secret values (redaction rule).
- Acceptance: tests green; `go build ./...`; funlen 0 for new files.

### Task 2 (P-02) — Routes (zone-scoped)
Files: `pkg/cosmoflare/worker_routes.go` + tests, `cmd/worker_routes.go` +
run tests.
- `RouteList(ctx, zoneID)`, `RouteCreate(ctx, zoneID, pattern, script)`,
  `RouteUpdate(ctx, zoneID, routeID, pattern, script)`,
  `RouteDelete(ctx, zoneID, routeID)` — via cloudflare-go routes funcs with
  `cloudflare.ZoneIdentifier(zoneID)`.
- Commands: `worker route list ZONE_ID`, `route create ZONE_ID
  --pattern --script`, `route update ZONE_ID ROUTE_ID --pattern --script`,
  `route delete ZONE_ID ROUTE_ID --force`.

### Task 3 (P-03) — Versions (raw REST)
Files: `pkg/cosmoflare/worker_versions.go` + tests,
`cmd/worker_versions.go` + run tests.
- Struct fields are the one `worker.go` edit (Go cannot declare struct
  fields in a second file): add unexported `apiToken string` /
  `apiBaseURL string` to the `WorkerService` struct declaration in
  `worker.go`; all raw-REST *methods* stay in the NEW file only.
  Constructors, same task: `NewWorkerServiceFromCreds` stores the token and
  defaults `apiBaseURL` to `https://api.cloudflare.com/client/v4`;
  `NewWorkerService` leaves both zero (raw REST unavailable → typed error
  "raw API requires credentials-based construction"). Existing
  `NewWorkerService` callers/tests are unaffected (fields are additive and
  unexported).
- `rawRequest(ctx, method, path string, body any, out any) error`:
  Bearer auth, 30s timeout, CF error-envelope unwrap, typed errors.
- `VersionUpload(ctx, name, script io.Reader, opts ...WorkerOption)`,
  `VersionList(ctx, name)`, `VersionGet(ctx, name, id)`,
  `VersionDeploy(ctx, name, id)` (PUT deployments pointer),
  `VersionDelete(ctx, name, id)`, `VersionRollback(ctx, name, id)` =
  deploy with rollback reporting.
- Endpoints: `accounts/{id}/workers/scripts/{name}/versions`,
  `.../versions/{vid}`, `.../deployments` (pointer PUT).
- `VersionUpload` sends `multipart/form-data` (metadata JSON part + script
  part), NOT JSON — the real API rejects a JSON body; contract mocks must
  assert the request content type.
- Tests inject `httptest.NewServer` as apiBaseURL.
- Commands: `worker versions upload|list|view|deploy|delete|rollback`.

### Task 4 (P-04) — Deployments + rollback
Files: `pkg/cosmoflare/worker_deployments.go` + tests,
`cmd/worker_deployments.go` + run tests.
- DEPENDS ON Task 3 (must land after it): all three methods go over Task
  3's `rawRequest`/`apiToken`/`apiBaseURL` — cloudflare-go v0.116.0 ships
  no workers-deployments API (verified; only Pages deployments exist). If
  dispatched before Task 3 merges, those symbols are undefined: STOP and
  re-run after Task 3 — never re-implement `rawRequest` in this task.
- `DeploymentList(ctx, name)`, `DeploymentGet(ctx, name, id)`,
  `Rollback(ctx, name, deploymentID)` (resolves the deployment's version and
  re-points, raw REST).
- Commands: `worker deployments list [name]`, `deployments view NAME ID`,
  `worker rollback NAME [DEPLOYMENT_ID]` (omit ID = previous deployment).

## Phase 2 — Wave 2 (next session, same shape)

### Task 5 (P-05) — Custom domains: List/Attach/Detach via cloudflare-go;
commands `worker domain list|attach|detach`.
### Task 6 (P-06) — Subdomain: Get/Create(Set) via cloudflare-go;
commands `worker subdomain get|set`.
### Task 7 (P-07) — Cron: List/Update (create/delete = Update with
filtered sets); commands `worker cron list|create|update|delete NAME`.
### Task 8 (P-08) — Bindings list (settings bindings) + `worker tail NAME`
streaming over existing `TailLogs`.

## Phase 3 — Wave 3

### Task 9 (P-09) — `worker types NAME [--out FILE]`:
Files: `pkg/cosmoflare/worker_types.go` + `worker_types_test.go`,
`cmd/worker_types.go` + `cmd/worker_types_run_test.go`.
pure `GenerateWorkerTypes(settings WorkerSettings) string` producing
`worker-configuration.d.ts` (Env interface; KVNamespace/R2Bucket/
D1Database/Queue/Fetcher/secret_string mapping). Tests assert generated
text for every binding kind — name them `TestGenerateWorkerTypes*`
(`worker_test.go:105` already defines `TestWorkerTypes`; a duplicate name
breaks compilation). Command prints or writes.
### Task 10 (P-10) — Integration (Opus): wire all register funcs into
`cmd/worker.go`; `--help` example sweep; acceptance gates:
- every command runs with `--json` (smoke via `--help` + arg validation),
- `golangci-lint run ./...` funlen 0,
- `go test ./...` green,
- coverage of new cmd files ≥ 40% offline paths,
- USAGE.md §Workers updated with the 29 new commands.

## Verification (per task, non-negotiable)

Name all new tests `TestWorker<Group>*` (e.g. `TestWorkerSecretsPut`) so
the patterns below select only the new suites — bare stems like 'Domain'
or 'Deployment' over-match existing tests (`TestBucketDomain*`,
`TestPages*Deployment*`, `TestRootCmd_VersionTemplate`, ...).

```
go test ./pkg/cosmoflare/ -run 'TestWorker<Group>' -count=1
go test ./cmd/ -run 'TestWorker<Group>' -count=1
golangci-lint run ./pkg/cosmoflare/ ./cmd/   # zero funlen hits
go build ./...
```

Funlen gate (TASK-009, `.golangci.yml`): every function ≤ 80 lines and
≤ 50 statements. Never disable or raise it to make a task pass.

## File Scope (planned)

pkg/cosmoflare/worker_{secrets,routes,versions,deployments,domains,subdomain,cron,types}.go
(+ _test.go each), pkg/cosmoflare/worker.go (apiToken/apiBaseURL struct
fields + FromCreds token capture; Task 3 only),
cmd/worker_{secrets,routes,versions,deployments,domains,subdomain,cron,bindings,tail,types}.go
(+ _run_test.go each), cmd/worker.go (registration wiring only, Opus),
docs/USAGE.md (Workers section).
