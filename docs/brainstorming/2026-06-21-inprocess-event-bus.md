---
created: "2026-06-21T05:55:00-03:00"
updated: "2026-06-21T05:55:00-03:00"
status: PENDING
priority: high
branch: master
title: "In-process event bus — real-time notification sourcing (ROAD-080)"
roadmap_ref: ROAD-080
plan_ref: docs/planning-mode/2026-06-21-inprocess-event-bus.md
related_ideas:
  - IDEA-037   # SSE metrics channel has no producer (out of scope here)
  - IDEA-038   # watchdog startup-only (out of scope here)
tags: [brainstorm, infra, events, notifications, desktop, ROAD-080]
deliverables:
  - id: BR-01
    title: "internal/events package: Bus with Publish/Subscribe, non-blocking drop-on-full, concurrent-safe"
  - id: BR-02
    title: "events.Event type carrying topic + account context + kind/message/data/time"
  - id: BR-03
    title: "webhook.Manager gains optional nil-safe bus; TriggerAlert publishes to it (before outbound)"
  - id: BR-04
    title: "serve daemon creates the bus, subscribes, and forwards events to the SSE notifications channel"
  - id: BR-05
    title: "Tests: bus unit (incl. -race + drop-on-full), Manager-with-bus emits event, daemon bus→SSE forwarding"
---

# In-process event bus — real-time notification sourcing (ROAD-080)

## Problem

`internal/webhook/manager.go` is **outbound-only**: `TriggerAlert` / `SendWebhook`
POST to registered Slack/Discord/HTTP endpoints, but nothing in-process can
subscribe to those events. The Cosmoflare Desktop app (shipped this session)
renders a real-time notifications panel fed by the serve daemon's SSE
`notifications` channel — but the only producer today is the daemon's own
`cloudflare_online` transition emitter (commit `3cd093a`). Alerts (error-rate /
limit / failure rules) never reach the desktop live; they only go outbound.

This was flagged as the **BR-07 gap** in the desktop brainplan's independent
review and captured as **ROAD-080**.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Scope** | Core bus only | Ship the BR-07 gap (alerts → live notifications). Live metrics (IDEA-037) and the deferred outbound sender stay as separate follow-ups — keeps this bounded. |
| **Delivery** | Non-blocking, drop-on-full (buf 32) | Mirrors `internal/server/sseHub`. A slow subscriber must never stall webhook/alert producers. Best-effort live stream, not a durable queue. |
| **Placement** | Standalone `internal/events` package | `webhook.NewManager(cf, accountID)` is bound to a **single CF account**; the serve daemon is multi-account (per-request profile). An account-agnostic bus sidesteps that binding, decouples production from consumption, and lets outbound webhooks + the in-process daemon both be consumers. Events carry account context in the payload. |

## Design

### Components

1. **`internal/events/bus.go`** (new, account-agnostic)

   ```go
   type Event struct {
       Topic   string         // routing key, e.g. "notifications"
       Account string         // CF account/profile this event pertains to ("" = global)
       Kind    string         // "alert" | "webhook" | ... (UI/category hint)
       Message string         // human-readable summary
       Data    map[string]any // structured payload
       Time    time.Time
   }

   type Bus struct {
       mu   sync.Mutex
       subs map[string]map[chan Event]struct{} // topic -> set of subscriber channels
   }

   func New() *Bus
   func (b *Bus) Publish(e Event)                              // fan-out by e.Topic; non-blocking, drop-on-full
   func (b *Bus) Subscribe(topic string) (<-chan Event, func()) // buffered chan + unsubscribe func
   ```

   - `Publish` locks, iterates subscribers of `e.Topic`, does a `select { case ch <- e: default: }` per channel (drop-on-full, buf 32).
   - `Subscribe` registers a fresh buffered channel and returns it plus an idempotent unsubscribe closure that removes + closes it under lock.
   - Concurrent-safe (single mutex). No global singleton — the bus is constructed and injected.

2. **`internal/webhook/manager.go`** (modify)

   - Add an optional `bus *events.Bus` field, set via a constructor option or `SetBus(*events.Bus)`. **nil-safe** — when unset, behavior is exactly today's.
   - In `TriggerAlert`, after the alert `message`/`data` are known and **before** the outbound `sendNotification` loop, also `bus.Publish(events.Event{Topic: "notifications", Kind: "alert", Account: ..., Message: ..., Data: ...})`. The existing outbound POST path is untouched. (Only the alert path publishes for v1; generic `SendWebhook` events stay outbound-only — see Out of scope.)

3. **`cmd/serve.go` + `internal/server`** (modify/wire)

   - The daemon constructs an `events.Bus`, subscribes to `"notifications"`, and runs a goroutine that drains the subscription channel and calls `srv.Publish("notifications", event)` — feeding the existing SSE hub.
   - On shutdown: call the unsubscribe func and stop the goroutine (tie to the existing `ctx` cancellation in `runServe`).
   - The same bus instance is handed to whatever constructs the webhook `Manager` so alerts flow through it.

### Data flow

```
TriggerAlert(alert, value, msg)
   ├─ existing: SendWebhook → outbound POST (Slack/Discord/HTTP)   [unchanged]
   └─ new:      bus.Publish(Event{Topic:"notifications", Kind:"alert", ...})
                     │
                     ▼
              events.Bus  ──fan-out──▶  daemon subscription goroutine
                                              │
                                              ▼
                                   srv.Publish("notifications", event)
                                              │
                                              ▼
                                     SSE hub ▶ desktop notifications panel (live)
```

### Error handling & semantics

- **Non-blocking:** drop-on-full per subscriber; producers never block. Buffer 32 (matches `sseHub`).
- **nil-bus safe:** the webhook Manager works with no bus attached.
- **Concurrency:** `Publish`/`Subscribe`/unsubscribe are mutex-guarded; verified under `-race`.
- **Lifecycle:** unsubscribe is idempotent and closes the channel; the daemon's forwarding goroutine exits on channel close or ctx cancel.

### Testing

- `internal/events/bus_test.go`: subscribe→publish→receive; topic isolation (subscriber to topic A doesn't get topic B); drop-on-full (fill buffer, publisher doesn't block, excess dropped); unsubscribe stops delivery; `-race` concurrent publishers + subscribers.
- `internal/webhook`: `TriggerAlert` with a bus attached publishes exactly one event observed on a subscription; with nil bus it does not panic.
- `internal/server` (or `cmd`): publishing to the bus results in a `notifications` SSE frame delivered to a connected `/events` client (end-to-end forwarding).

## File scope

- **NEW:** `internal/events/bus.go`, `internal/events/bus_test.go`
- **MODIFY:** `internal/webhook/manager.go` (+ `manager_*_test.go`), `internal/server/` (subscribe→SSE wiring, likely `server.go` or a small `events.go`), `cmd/serve.go` (construct bus, wire Manager + daemon subscription)

## Out of scope (separate follow-ups)

- **IDEA-037** — live metrics producer for the SSE `metrics` channel (dashboard cards go stale until account switch). Separate: a daemon poll loop or a `refetchInterval`.
- **IDEA-038** — Rust watchdog mid-session restart (currently startup-only).
- Outbound webhook sender driven from daemon-observed events (v2 notification platform).

## Related

- Roadmap: ROAD-080 (priority 70, infra, medium)
- Origin: BR-07 gap from `docs/brainstorming/2026-06-20-cosmoflare-desktop.md` independent review
- Consumer already shipped: `internal/server/sse.go`, desktop `src/views/Notifications.tsx`
