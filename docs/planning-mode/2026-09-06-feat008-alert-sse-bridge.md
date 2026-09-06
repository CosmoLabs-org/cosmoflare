---
completed: "2026-09-06T19:48:24+04:00"
created: "2026-09-06T00:00:00+02:00"
deliverables:
    - id: P-01
      title: webhook.Manager notifier hook with tests (TriggerAlert invokes notifier, nil-safe)
    - id: P-02
      title: serve daemon wiring Manager → sseHub notifications channel with SSE-level test
    - id: P-03
      title: FEAT-008 issue closed with re-scope note; changelog entry
goals_completed: 0
goals_total: 0
implemented_commits:
    - covers:
        - P-01
        - P-02
      sha: bc18c4c13f7e
issue: FEAT-008
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: FEAT-008 Re-Scope — TriggerAlert → sseHub Notifications Bridge
type: implementation-plan
---

# FEAT-008 Re-Scope — TriggerAlert → sseHub Notifications Bridge

## Goal

Bridge `webhook.Manager.TriggerAlert` to the serve daemon's existing SSE hub so
that every alert trigger fans out to connected desktop clients on the
`notifications` channel. This replaces the 71-day-stalled standalone event-bus
plan (docs/brainstorming/2026-06-21-inprocess-event-bus.md) — the sseHub IS the
bus now.

## Why Thin (Assumptions)

1. The sseHub (`internal/server/sse.go`) already multiplexes channels
   (metrics / notifications / status) to every `/events` client and never
   blocks publishers. No new bus is needed — only a producer.
2. `webhook.Manager` is currently an island: nothing outside its own tests
   constructs it. The bridge is its first production wiring.
3. Alert rules (`pkg/cosmoflare/alerts.go`) reference conditions (error-rate,
   latency, storage-limit, failure-count) that no metrics producer computes
   today. An evaluator would require NEW producers — out of scope. This plan
   ships the plumbing; any future producer calls `TriggerAlert` and desktop
   clients see the event immediately.

## Design

Callback, not import: `internal/server` must not import `internal/webhook`
(wrong direction — the server does not own alerts). The Manager grows one
optional hook; the daemon owns both objects and wires them.

```
TriggerAlert(alert, value, msg, data)
  ├─ updates alert state (unchanged)
  ├─ sends to configured webhooks (unchanged)
  └─ NEW: invokes m.notifier(payload) if set        ← webhook.Manager
                │
                └─ srv.Publish("notifications", payload)   ← cmd/serve.go wiring
                       └─ sseHub → every /events client
```

SSE frame the desktop receives:

```
event: notifications
data: {"event":"alert_triggered","alert":{...},"value":...,"threshold":...,"message":"...","timestamp":"..."}
```

## Steps (TDD)

1. **Red** — `internal/webhook/manager_test.go`: TriggerAlert invokes the
   notifier with the full NotificationPayload when set; does not panic when
   unset; fires once per call.
2. **Green** — `internal/webhook/manager.go`: add `SetNotifier(func(*NotificationPayload))`,
   call it at the end of TriggerAlert.
3. **Red** — `cmd/serve.go` wiring test: constructing the daemon's alert
   manager and triggering an alert publishes a `notifications` frame to a
   subscribed SSE client (httptest against Server, same pattern as
   `internal/server` tests).
4. **Green** — `cmd/serve.go`: construct `webhook.NewManager`, `SetNotifier`
   → `srv.Publish("notifications", payload)`.
5. Full package tests: `go test ./internal/webhook/ ./cmd/ ./internal/server/`.

## Non-Goals

- Alert rule evaluation loops (needs metric producers that do not exist).
- Persistent notification history (SSE is a live channel, not a queue).
- Changes to webhook delivery semantics.

## Deliverables

- P-01: webhook.Manager notifier hook with tests (steps 1-2)
- P-02: serve daemon wiring Manager → sseHub notifications with SSE-level test (steps 3-4)
- P-03: FEAT-008 issue closed with re-scope note; changelog entry
