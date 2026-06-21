---
created: "2026-06-21T06:05:00-03:00"
updated: "2026-06-21T06:05:00-03:00"
status: PENDING
priority: high
branch: master
title: "In-process event bus — implementation plan (ROAD-080)"
roadmap_ref: ROAD-080
brainstorm_ref: docs/brainstorming/2026-06-21-inprocess-event-bus.md
tags: [plan, infra, events, notifications, ROAD-080]
deliverables:
  - id: P-01
    title: "internal/events package: Bus (Publish/Subscribe, drop-on-full, concurrent-safe) + tests"
  - id: P-02
    title: "webhook.Manager: optional nil-safe bus, TriggerAlert publishes to it before outbound + test"
  - id: P-03
    title: "internal/server: Server.SubscribeBus forwards bus events to the SSE topic + test"
  - id: P-04
    title: "cmd/serve.go: construct the bus, wire daemon subscription + Manager producer"
---

# In-Process Event Bus Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a standalone in-process pub/sub bus so webhook/alert events reach the serve daemon's SSE `notifications` channel in real time.

**Architecture:** A new account-agnostic `internal/events` package provides a `Bus` with non-blocking, drop-on-full fan-out (mirroring `internal/server/sseHub`). `webhook.Manager` publishes to the bus (nil-safe) in addition to its existing outbound POSTs; the serve daemon subscribes and forwards events into its SSE hub.

**Tech Stack:** Go 1.26, stdlib `sync` + `time`, existing `internal/server` SSE hub, `encoding/json` (struct tags drive the SSE frame shape the desktop consumes).

**Test runner:** `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./<pkg>/ -v` (keychain env is mandatory project hygiene; never assume a global go.work).

Design spec: `docs/brainstorming/2026-06-21-inprocess-event-bus.md`

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/events/bus.go` (new) | `Event` type + `Bus` pub/sub (Publish/Subscribe), the whole package |
| `internal/events/bus_test.go` (new) | Bus behavior: delivery, topic isolation, drop-on-full, unsubscribe, race |
| `internal/webhook/manager.go` (modify) | Optional `bus` field + `SetBus`; publish in `TriggerAlert` before outbound |
| `internal/webhook/manager_bus_test.go` (new) | `TriggerAlert` with bus emits one event; nil bus is safe |
| `internal/server/events.go` (new) | `Server.SubscribeBus(bus, topic)` — forward bus events → SSE hub |
| `internal/server/events_test.go` (new) | bus → SSE end-to-end: published event arrives as a `notifications` frame |
| `cmd/serve.go` (modify) | Construct the bus, `srv.SubscribeBus`, hand the bus to the webhook Manager |

---

## Task 1: events.Bus core

**Files:**
- Create: `internal/events/bus.go`
- Test: `internal/events/bus_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/events/bus_test.go
package events

import (
	"sync"
	"testing"
	"time"
)

func TestBus_PublishDeliversToSubscriber(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("notifications")
	defer unsub()
	b.Publish(Event{Topic: "notifications", Message: "hi"})
	select {
	case e := <-ch:
		if e.Message != "hi" {
			t.Fatalf("got message %q, want %q", e.Message, "hi")
		}
	case <-time.After(time.Second):
		t.Fatal("no event delivered")
	}
}

func TestBus_TopicIsolation(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("notifications")
	defer unsub()
	b.Publish(Event{Topic: "metrics", Message: "x"})
	select {
	case <-ch:
		t.Fatal("received event from a topic we did not subscribe to")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBus_DropOnFullDoesNotBlock(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("t")
	defer unsub()
	// Publish well past the buffer; a correct Bus never blocks the publisher.
	done := make(chan struct{})
	go func() {
		for i := 0; i < subBuffer+50; i++ {
			b.Publish(Event{Topic: "t"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a full subscriber buffer")
	}
	// Drained count must not exceed the buffer.
	got := 0
	for {
		select {
		case <-ch:
			got++
			if got > subBuffer {
				t.Fatalf("drained %d events, buffer is %d", got, subBuffer)
			}
		default:
			return
		}
	}
}

func TestBus_UnsubscribeStopsDeliveryAndClosesChannel(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("t")
	unsub()
	b.Publish(Event{Topic: "t"}) // must not panic on a removed subscriber
	if _, ok := <-ch; ok {
		t.Fatal("expected closed channel after unsubscribe")
	}
	unsub() // idempotent: second call must not panic
}

func TestBus_ConcurrentPublishSubscribe(t *testing.T) {
	// Run with -race. Hammers Publish/Subscribe/unsubscribe concurrently.
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := b.Subscribe("t")
			defer unsub()
			for j := 0; j < 200; j++ {
				b.Publish(Event{Topic: "t"})
				select {
				case <-ch:
				default:
				}
			}
		}()
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/events/ -v`
Expected: FAIL — `undefined: New`, `undefined: Event`, `undefined: subBuffer`.

- [ ] **Step 3: Write the implementation**

```go
// internal/events/bus.go

// Package events is a tiny in-process publish/subscribe bus. It is
// account-agnostic: producers (e.g. webhook.Manager) Publish events tagged with
// a topic, and in-process consumers (e.g. the serve daemon) Subscribe to a
// topic and forward events onward (e.g. into the SSE hub). Delivery is
// best-effort: a slow subscriber has frames dropped rather than blocking the
// publisher, mirroring internal/server.sseHub.
package events

import (
	"sync"
	"time"
)

// subBuffer is the per-subscriber channel buffer. Matches the SSE hub.
const subBuffer = 32

// Event is one published message. Field tags define the JSON shape that reaches
// the desktop via the SSE frame, so the UI can read `message`, `kind`, etc.
type Event struct {
	Topic   string         `json:"topic"`
	Account string         `json:"account,omitempty"`
	Kind    string         `json:"kind,omitempty"`
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Time    time.Time      `json:"time"`
}

// Bus fans events out to subscribers by topic. Safe for concurrent use.
type Bus struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{}
}

// New constructs an empty Bus.
func New() *Bus {
	return &Bus{subs: make(map[string]map[chan Event]struct{})}
}

// Publish delivers e to every subscriber of e.Topic. Best-effort: a subscriber
// whose buffer is full has this frame dropped rather than blocking the caller.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[e.Topic] {
		select {
		case ch <- e:
		default:
			// drop-on-full
		}
	}
}

// Subscribe registers a new subscriber for topic and returns its receive
// channel plus an idempotent unsubscribe function that removes and closes it.
func (b *Bus) Subscribe(topic string) (<-chan Event, func()) {
	ch := make(chan Event, subBuffer)
	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[chan Event]struct{})
	}
	b.subs[topic][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			b.mu.Lock()
			if set := b.subs[topic]; set != nil {
				delete(set, ch)
				if len(set) == 0 {
					delete(b.subs, topic)
				}
			}
			close(ch)
			b.mu.Unlock()
		})
	}
	return ch, unsub
}
```

- [ ] **Step 4: Run tests (incl. -race) to verify they pass**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/events/ -race -v`
Expected: PASS (all 5 tests, no race warnings).

- [ ] **Step 5: Commit**

```bash
git add internal/events/bus.go internal/events/bus_test.go
git commit -m "feat(events): add in-process pub/sub bus (drop-on-full)

Account-agnostic Bus with Publish/Subscribe and non-blocking drop-on-full
fan-out, mirroring the SSE hub. Foundation for ROAD-080 real-time notifications.

Refs: ROAD-080"
```

---

## Task 2: webhook.Manager publishes alerts to the bus

**Files:**
- Modify: `internal/webhook/manager.go` (struct `Manager`, func `TriggerAlert`)
- Test: `internal/webhook/manager_bus_test.go`

> Read `internal/webhook/manager.go` first. `Manager` is built by
> `NewManager(cf *cloudflare.API, accountID string)` and stores the account id in
> the `accountID` field. Add the bus field and a setter, then publish **before**
> the outbound POST loop in `TriggerAlert` (the per-webhook `sendNotification`
> calls) so alerts reach in-process subscribers even when outbound delivery fails
> or no webhooks are registered. Use `m.accountID` for `Event.Account`.

- [ ] **Step 1: Write the failing test**

```go
// internal/webhook/manager_bus_test.go
package webhook

import (
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/events"
)

func TestManager_TriggerAlertPublishesToBus(t *testing.T) {
	m := NewManager(nil, "acct-1") // no cf, no registered webhooks → outbound is a no-op
	bus := events.New()
	m.SetBus(bus)

	ch, unsub := bus.Subscribe("notifications")
	defer unsub()

	_ = m.TriggerAlert(&Alert{Name: "error-rate"}, 99, "error rate high", nil)

	select {
	case e := <-ch:
		if e.Kind != "alert" {
			t.Fatalf("kind = %q, want alert", e.Kind)
		}
		if e.Message != "error rate high" {
			t.Fatalf("message = %q", e.Message)
		}
		if e.Account != "acct-1" {
			t.Fatalf("account = %q, want acct-1", e.Account)
		}
	case <-time.After(time.Second):
		t.Fatal("no event published to bus")
	}
}

func TestManager_TriggerAlertNilBusIsSafe(t *testing.T) {
	m := NewManager(nil, "acct-1") // no SetBus
	if err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panicked with nil bus: %v", r)
			}
		}()
		_ = m.TriggerAlert(&Alert{Name: "x"}, 1, "msg", nil)
		return nil
	}(); err != nil {
		t.Fatal(err)
	}
}
```

> If `Alert` does not have a `Name` field, use the fields it actually has when
> constructing the test alert — read the struct at `manager.go`. The assertions
> on `Kind`/`Message`/`Account` are the contract; adapt only the alert literal.

- [ ] **Step 2: Run test to verify it fails**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/webhook/ -run TestManager_TriggerAlert -v`
Expected: FAIL — `m.SetBus undefined`.

- [ ] **Step 3: Implement**

In `manager.go`, add the import `"github.com/CosmoLabs-org/cosmoflare/internal/events"`, a field on `Manager`:

```go
	bus *events.Bus // optional in-process event sink (ROAD-080); nil = outbound only
```

a setter:

```go
// SetBus attaches an in-process event bus. When set, alert/webhook events are
// published to it in addition to the outbound POSTs. Safe to leave unset.
func (m *Manager) SetBus(b *events.Bus) { m.bus = b }
```

and, at the **top** of `TriggerAlert` (after `message`/`data` are in scope — they are parameters — but before the outbound per-webhook `sendNotification` loop), publish:

```go
	if m.bus != nil {
		m.bus.Publish(events.Event{
			Topic:   "notifications",
			Account: m.accountID, // the Manager's stored account-id field
			Kind:    "alert",
			Message: message,
			Data:    data,
			Time:    time.Now(),
		})
	}
```

Add `"time"` to imports if not present.

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/webhook/ -run TestManager_TriggerAlert -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/webhook/manager.go internal/webhook/manager_bus_test.go
git commit -m "feat(webhook): publish alerts to in-process event bus

TriggerAlert now publishes to an optional events.Bus before the outbound POST,
so in-process subscribers (the serve daemon) get alerts even when outbound
delivery fails. nil bus preserves today's outbound-only behavior.

Refs: ROAD-080"
```

---

## Task 3: Server forwards bus events to SSE

**Files:**
- Create: `internal/server/events.go`
- Test: `internal/server/events_test.go`

> Reference `internal/server/sse.go` for `s.Publish`, `s.Subscribers()`, and the
> `/events` handler, and `internal/server/sse_test.go` for the httptest SSE
> client + the `Subscribers()` wait pattern that defeats publish-before-subscribe.

- [ ] **Step 1: Write the failing test**

```go
// internal/server/events_test.go
package server

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/events"
)

func TestServer_SubscribeBusForwardsToSSE(t *testing.T) {
	s := New(Config{Token: "tok"})
	bus := events.New()
	stop := s.SubscribeBus(bus, "notifications")
	defer stop()

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/events?token=tok", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Wait until BOTH the SSE client and the bus-forwarder are registered.
	deadline := time.Now().Add(2 * time.Second)
	for s.Subscribers() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	bus.Publish(events.Event{Topic: "notifications", Kind: "alert", Message: "boom"})

	// Read until we see the event frame or time out.
	type result struct{ body string }
	got := make(chan result, 1)
	go func() {
		r := bufio.NewReader(resp.Body)
		var b strings.Builder
		for {
			line, err := r.ReadString('\n')
			b.WriteString(line)
			if strings.Contains(b.String(), `"boom"`) {
				got <- result{b.String()}
				return
			}
			if err != nil {
				got <- result{b.String()}
				return
			}
		}
	}()

	select {
	case res := <-got:
		if !strings.Contains(res.body, "event: notifications") || !strings.Contains(res.body, `"boom"`) {
			t.Fatalf("did not get forwarded notifications frame; got:\n%s", res.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for forwarded SSE frame")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -run TestServer_SubscribeBus -v`
Expected: FAIL — `s.SubscribeBus undefined`.

- [ ] **Step 3: Implement**

```go
// internal/server/events.go
package server

import (
	"sync"

	"github.com/CosmoLabs-org/cosmoflare/internal/events"
)

// SubscribeBus subscribes to topic on bus and forwards every event into this
// server's SSE hub under the same topic, until the returned stop func is called.
// Used by the serve daemon to turn in-process alert/webhook events into live
// SSE frames for the desktop.
func (s *Server) SubscribeBus(bus *events.Bus, topic string) (stop func()) {
	ch, unsub := bus.Subscribe(topic)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case e, ok := <-ch:
				if !ok {
					return
				}
				s.Publish(topic, e)
			case <-done:
				return
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			unsub()
		})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -race -v`
Expected: PASS (new test + existing SSE tests, no races).

- [ ] **Step 5: Commit**

```bash
git add internal/server/events.go internal/server/events_test.go
git commit -m "feat(server): forward in-process bus events to the SSE hub

Server.SubscribeBus drains an events.Bus topic and republishes each event as an
SSE frame on the same channel, with a stop func that ends the goroutine and
unsubscribes. This is the daemon-side consumer for ROAD-080.

Refs: ROAD-080"
```

---

## Task 4: Wire the bus into the serve daemon

**Files:**
- Modify: `cmd/serve.go` (func `runServe`)

> Read `cmd/serve.go`. `runServe` builds `srv := server.New(...)` and has a
> `ctx, cancel` for shutdown. Construct one bus, subscribe the server to it, and
> stop on shutdown. If/where a `webhook.Manager` is constructed for the daemon,
> call `mgr.SetBus(bus)` so its alerts flow to the SSE channel. (If no Manager is
> built in the daemon yet, leave a `// TODO(ROAD-080): mgr.SetBus(bus)` marker at
> the construction site and wire it when alert evaluation lands — the bus +
> subscription is the seam this task delivers.)

- [ ] **Step 1: Add the bus + subscription**

In `runServe`, after `srv := server.New(...)` and before the server starts serving:

```go
	// ROAD-080: in-process event bus → SSE notifications. Producers (webhook
	// alerts) publish to bus; the server forwards them to connected clients.
	bus := events.New()
	stopBus := srv.SubscribeBus(bus, "notifications")
	defer stopBus()
```

Add the import `"github.com/CosmoLabs-org/cosmoflare/internal/events"`.

- [ ] **Step 2: Verify build**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go build ./...`
Expected: exit 0.

- [ ] **Step 3: Verify full daemon test suite still green**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/events/ ./internal/webhook/ ./internal/server/ ./cmd/ -count=1`
Expected: all `ok`.

- [ ] **Step 4: Commit**

```bash
git add cmd/serve.go
git commit -m "feat(serve): wire in-process event bus into the daemon

The serve daemon now creates an events.Bus and forwards its notifications topic
to the SSE hub, so webhook alerts surface live in the desktop. Closes the
ROAD-080 seam end to end.

Refs: ROAD-080"
```

---

## Self-Review

**Spec coverage:** BR-01→P-01 (bus), BR-02→P-01 (Event type), BR-03→P-02 (Manager), BR-04→P-03+P-04 (server forward + daemon wiring), BR-05→tests in every task. All five brainstorm deliverables are covered.

**Placeholder scan:** Task 4 contains one intentional, conditional `TODO(ROAD-080)` marker — used only if no daemon-side `webhook.Manager` exists yet; the seam (bus + subscription + forwarding) is fully implemented and tested regardless. No other placeholders.

**Type consistency:** `Event` fields (`Topic/Account/Kind/Message/Data/Time`) and `subBuffer` are defined in Task 1 and referenced identically in Tasks 2–4. `SetBus`, `SubscribeBus(bus, topic)`, `Publish` signatures match across tasks. The one explicit adapt-point is the Manager's stored account-id field name and the `Alert` literal — flagged in Task 2 for the implementer to match against `manager.go`.

## Execution

Independent steps after Task 1: Task 2 (webhook) and Task 3 (server) are independent of each other; Task 4 depends on both. Suitable for a small GLM dispatch (Task 1 first, then Task 2 ∥ Task 3, then Task 4) or a single focused inline pass. Opus reviews every diff + re-runs tests before merge (S334 gate).
