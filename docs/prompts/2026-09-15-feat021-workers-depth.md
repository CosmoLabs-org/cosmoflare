---
brainstorm_ref: docs/brainstorming/2026-09-15-feat021-workers-depth.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
    - BR-07
    - BR-08
    - BR-09
    - BR-10
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
    - P-09
    - P-10
created: "2026-09-15T23:55:40+04:00"
id: P-2026-09-15-feat021-workers-depth
plan_ref: docs/planning-mode/2026-09-15-feat021-workers-depth.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-09-15-feat021-workers-depth.md
    - docs/planning-mode/2026-09-15-feat021-workers-depth.md
schema_version: 1
status: PENDING
title: FEAT-021 Workers Command Depth — Full Implementation
---
# FEAT-021 Workers Command Depth — Full Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-15-feat021-workers-depth.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-09-15-feat021-workers-depth.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

_Describe the session context._

## Goals

### [ ] G-01 worker_secrets.go: Put/Delete/List/Bulk + tests + 4 commands
Covers P-01.

### [ ] G-02 worker_routes.go: zone-scoped CRUD + tests + 4 commands
Covers P-02.

### [ ] G-03 worker_versions.go: raw-REST versions client + tests + 6 commands
Covers P-03.

### [ ] G-04 worker_deployments.go: list/view/rollback + tests + 3 commands
Covers P-04.

### [ ] G-05 worker_domains.go: attach/detach/list + tests + 3 commands
Covers P-05.

### [ ] G-06 worker_subdomain.go: get/set + tests + 2 commands
Covers P-06.

### [ ] G-07 worker_cron.go: CRUD via Update + tests + 4 commands
Covers P-07.

### [ ] G-08 bindings list + tail commands (library reuse)
Covers P-08.

### [ ] G-09 worker types: .d.ts generator (pure) + command + tests
Covers P-09.

### [ ] G-10 Integration: cmd/worker.go wiring, --help sweep, acceptance gates
Covers P-10.

## Related

- Brainstorm: `docs/brainstorming/2026-09-15-feat021-workers-depth.md`
- Plan: `docs/planning-mode/2026-09-15-feat021-workers-depth.md`

# FEAT-021 Workers Command Depth — Full Implementation

## Context

The Workers daily loop (versions, deployments, rollback, secrets, routes,
cron, domains, subdomain, bindings, tail, types) is the audit's top
competitive gap — 29 new commands close it. Transport split: cloudflare-go
v0.116 covers secrets/routes/domains/subdomain/cron; versions+deployments
need a raw-REST helper on WorkerService with injectable base URL.

Design spec: `docs/brainstorming/2026-09-15-feat021-workers-depth.md`
Implementation plan: `docs/planning-mode/2026-09-15-feat021-workers-depth.md`
GLM manifest: `docs/prompts/2026-09-15-feat021-workers-depth-glm-tasks.yaml`

## Goals

### [ ] 1. Wave 1 — secrets + routes + versions + deployments/rollback (P-01..P-04)
Acceptance: manifest tasks 1-4 merged through the S334 gate; go test ./... green; funlen 0 for new files
### [ ] 2. Wave 2 — domains + subdomain + cron + bindings + tail (P-05..P-08)
Acceptance: same shape as wave 1, per plan tasks 5-8; tail streams via existing TailLogs
### [ ] 3. Wave 3 — worker types .d.ts generator (P-09)
Acceptance: pure GenerateWorkerTypes(settings) string; tests cover every binding kind; worker types NAME command
### [ ] 4. Integration + acceptance (P-10, Opus-side)
Acceptance: all register funcs wired in cmd/worker.go; every command answers --help and --json arg-validation offline; golangci funlen 0; go test ./... green; USAGE.md Workers section updated; coverage of new cmd files >= 40%

## Execution Strategy

GLM batch dispatch via the manifest (wave-size 4). Wave-1 task 4
(deployments) depends on task 3's rawRequest landing first — dispatch it
after task 3 merges, or accept the manifest's ordering note. Waves 2-3:
extend the manifest with the same per-task shape (plan tasks 5-9). The
cmd/worker.go wiring, USAGE.md sweep, and acceptance gates stay Opus-side.

    agents:
      - task: "secrets put/delete/list/bulk"
        model: glm-turbo
        files: [pkg/cosmoflare/worker_secrets.go, cmd/worker_secrets.go]
        ready: true
      - task: "routes CRUD (zone-scoped)"
        model: glm-turbo
        files: [pkg/cosmoflare/worker_routes.go, cmd/worker_routes.go]
        ready: true
      - task: "versions raw-REST client"
        model: glm-turbo
        files: [pkg/cosmoflare/worker.go, pkg/cosmoflare/worker_versions.go, cmd/worker_versions.go]
        ready: true
      - task: "deployments + rollback"
        model: glm-turbo
        files: [pkg/cosmoflare/worker_deployments.go, cmd/worker_deployments.go]
        ready: false   # after versions merges (needs rawRequest)

## File Scope

- pkg/cosmoflare/worker_{secrets,routes,versions,deployments}.go + tests (wave 1)
- pkg/cosmoflare/worker.go (FromCreds token capture, versions task only)
- cmd/worker_{secrets,routes,versions,deployments}.go + run tests (wave 1)
- Waves 2-3: worker_{domains,subdomain,cron,bindings,tail,types} pairs
- cmd/worker.go wiring + docs/USAGE.md (Opus, wave 3)
