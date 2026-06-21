---
brainstorm_ref: docs/brainstorming/2026-06-21-inprocess-event-bus.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
created: "2026-06-21"
id: P-2026-06-21-inprocess-event-bus
plan_ref: docs/planning-mode/2026-06-21-inprocess-event-bus.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-06-21-inprocess-event-bus.md
    - docs/planning-mode/2026-06-21-inprocess-event-bus.md
schema_version: 1
status: PENDING
title: In-process event bus — full implementation (ROAD-080)
---
# In-process event bus — full implementation (ROAD-080)

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-06-21-inprocess-event-bus.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-06-21-inprocess-event-bus.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

`internal/webhook` is outbound-only, so alerts never reach the Cosmoflare
Desktop app's live notifications panel (the BR-07 gap, ROAD-080). This adds a
standalone account-agnostic `internal/events` bus (non-blocking, drop-on-full)
that `webhook.Manager.TriggerAlert` publishes to and the serve daemon subscribes
to, forwarding alerts onto the SSE `notifications` channel. Scope is core-bus-only;
live metrics (IDEA-037) and the outbound sender are out of scope.

## Goals

### [ ] G-01 internal/events package: Bus (Publish/Subscribe, drop-on-full, concurrent-safe) + tests
**Model:** glm-turbo — self-contained new package; plan has the exact `Event`/`Bus` code + 5 tests. Covers P-01.

### [ ] G-02 webhook.Manager: optional nil-safe bus, TriggerAlert publishes to it before outbound + test
**Model:** sonnet — must read `internal/webhook/manager.go` to match the stored account-id field + `Alert` literal before inserting the publish. Covers P-02.

### [ ] G-03 internal/server: Server.SubscribeBus forwards bus events to the SSE topic + test
**Model:** sonnet — reuses the `internal/server/sse_test.go` httptest SSE-client + `Subscribers()` wait pattern. Covers P-03.

### [ ] G-04 cmd/serve.go: construct the bus, wire daemon subscription + Manager producer
**Model:** glm-turbo — exact snippet + placement given (after `srv := server.New(...)`); build + suite verification. Covers P-04.

## Execution Strategy

Dependency order: **G-01 first** (everything imports `internal/events`), then
**G-02 ∥ G-03** (independent of each other), then **G-04** (depends on both).
Suitable for a small GLM dispatch or a single inline TDD pass. **Opus reviews
every diff + re-runs `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./...` before
merge (S334 gate).** Architecture is settled in the brainstorm — implement, don't redesign.

```yaml
agents:
  - task: "G-01 events.Bus core + tests"
    model: glm-turbo
    files: [internal/events/bus.go, internal/events/bus_test.go]
    ready: true
  - task: "G-02 webhook.Manager bus publish"
    model: sonnet
    files: [internal/webhook/manager.go, internal/webhook/manager_bus_test.go]
    blocked_by: G-01
  - task: "G-03 Server.SubscribeBus → SSE"
    model: sonnet
    files: [internal/server/events.go, internal/server/events_test.go]
    blocked_by: G-01
  - task: "G-04 serve.go wiring"
    model: glm-turbo
    files: [cmd/serve.go]
    blocked_by: [G-02, G-03]
```

## File Scope

- **NEW:** `internal/events/bus.go`, `internal/events/bus_test.go`, `internal/server/events.go`, `internal/server/events_test.go`, `internal/webhook/manager_bus_test.go`
- **MODIFY:** `internal/webhook/manager.go`, `cmd/serve.go`

## Related

- Brainstorm: `docs/brainstorming/2026-06-21-inprocess-event-bus.md`
- Plan: `docs/planning-mode/2026-06-21-inprocess-event-bus.md`
