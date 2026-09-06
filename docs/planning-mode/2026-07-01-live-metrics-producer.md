---
brainstorm_ref: docs/brainstorming/2026-07-01-live-metrics-producer.md
created: "2026-07-01T16:20:00+04:00"
deliverables:
    - id: P-01
      title: MetricsProducer core with subscriber gating and delta detection
    - id: P-02
      title: MetricsProducer tests with fakeSource
    - id: P-03
      title: cmd/serve.go wiring with --metrics-interval flag
    - id: P-04
      title: App.tsx SSE metrics → React Query cache bridge
last_review_content_hash: a671a3347c048b13b856707e405094c92fe307a80c85e5df9946a09085862d13
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-07-01-live-metrics-producer.md
last_reviewed: "2026-09-06T20:14:04.21166+04:00"
related_issues:
    - IDEA-037
status: COMPLETED
title: Live metrics producer implementation plan (IDEA-037)
updated: "2026-09-06T00:00:00+04:00"
---

# Live Metrics Producer Implementation Plan (IDEA-037)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a daemon-side goroutine that periodically polls the Cloudflare API and publishes full resource arrays to the SSE `metrics` channel, so the desktop dashboard stays live without manual refresh.

**Architecture:** A `MetricsProducer` in `internal/server/` owns a ticker, calls the existing `ServeSource` methods, and publishes via `Server.Publish("metrics", ...)`. It skips API calls when no SSE subscribers are connected. The React dashboard's SSE handler injects metrics frames into the React Query cache, so existing `useQuery` hooks re-render automatically.

**Tech Stack:** Go 1.26 (server), React 19 + TanStack Query (dashboard)

---

### Task 1: MetricsProducer core + tests (P-01, P-02)

**Files:**
- Create: `internal/server/metrics.go`
- Create: `internal/server/metrics_test.go`

**Existing code you need to know:**
- `Server.Publish(channel string, data any)` — fans out an SSE frame to all `/events` clients (`internal/server/sse.go:82`)
- `Server.Subscribers() int` — returns connected SSE client count (`internal/server/sse.go:71`)
- `ServeSource` interface — `Zones/R2Buckets/Workers/KV` each take `(ctx, profile string)` and return `(any, error)` (`internal/server/rest.go:18-29`)
- `fakeSource` in `internal/server/rest_test.go:14` — test double that satisfies `ServeSource`
- `newSourcedServer(t, src)` in `rest_test.go:42` — creates a Server with a fake source attached
- Tests run with: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -run <pattern> -v`

- [ ] **Step 1: Write the failing test for MetricsProducer**

Create `internal/server/metrics_test.go`:

```go
package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestMetricsProducer_PublishesOnTick(t *testing.T) {
	src := &fakeSource{
		zones:   []map[string]any{{"name": "example.com"}},
		r2:      []map[string]any{{"name": "bucket-1"}, {"name": "bucket-2"}},
		workers: []map[string]any{{"name": "worker-1"}},
		kv:      []map[string]any{{"name": "ns-1"}},
	}
	s, url := newSourcedServer(t, src)

	// Connect an SSE client so Subscribers() > 0.
	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)

	// Read one SSE frame from the stream.
	br := bufio.NewReader(resp.Body)
	var frame strings.Builder
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		frame.WriteString(line)
		if line == "\n" {
			break
		}
		if err != nil {
			break
		}
	}
	got := frame.String()
	if !strings.Contains(got, "event: metrics") {
		t.Fatalf("expected metrics event, got: %q", got)
	}

	// Parse the data line.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "data: ") {
			var payload MetricsSnapshot
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if payload.Profile != "" {
				t.Fatalf("profile = %q, want empty (default)", payload.Profile)
			}
			zones, ok := payload.Zones.([]any)
			if !ok || len(zones) != 1 {
				t.Fatalf("zones = %v, want 1-element array", payload.Zones)
			}
			return
		}
	}
	t.Fatal("no data line in frame")
}

func TestMetricsProducer_SkipsWhenNoSubscribers(t *testing.T) {
	src := &fakeSource{
		zones: []map[string]any{{"name": "z"}},
		r2:    []map[string]any{},
	}
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)

	// Wait two tick intervals with no subscriber.
	time.Sleep(150 * time.Millisecond)
	cancel()

	// fakeSource.seenProf is only set when a CF method is called.
	// If the producer correctly skipped, seenProf stays empty.
	if src.seenProf != "" {
		t.Fatalf("producer called source with no subscribers (profile=%q)", src.seenProf)
	}
}

func TestMetricsProducer_StopsOnContextCancel(t *testing.T) {
	src := &fakeSource{zones: []map[string]any{{"name": "z"}}}
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)

	ctx, cancel := context.WithCancel(context.Background())
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)
	cancel()
	// If Start leaks a goroutine, the test's -race detector catches it.
	time.Sleep(100 * time.Millisecond)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -run TestMetricsProducer -v`
Expected: FAIL — `NewMetricsProducer` and `MetricsSnapshot` not defined.

- [ ] **Step 3: Write the MetricsProducer implementation**

Create `internal/server/metrics.go`:

```go
package server

import (
	"context"
	"log"
	"reflect"
	"time"
)

type MetricsSnapshot struct {
	Profile      string `json:"profile"`
	Zones        any    `json:"zones"`
	R2Buckets    any    `json:"r2_buckets"`
	Workers      any    `json:"workers"`
	KVNamespaces any    `json:"kv_namespaces"`
}

type MetricsProducer struct {
	srv      *Server
	src      ServeSource
	interval time.Duration
	last     *MetricsSnapshot
}

func NewMetricsProducer(srv *Server, src ServeSource, interval time.Duration) *MetricsProducer {
	return &MetricsProducer{srv: srv, src: src, interval: interval}
}

func (m *MetricsProducer) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *MetricsProducer) run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.srv.Subscribers() == 0 {
				continue
			}
			m.poll(ctx)
		}
	}
}

func (m *MetricsProducer) poll(ctx context.Context) {
	snap := MetricsSnapshot{Profile: ""}
	var failed bool

	if data, err := m.src.Zones(ctx, ""); err != nil {
		log.Printf("[metrics] zones: %v", err)
		failed = true
	} else {
		snap.Zones = data
	}
	if data, err := m.src.R2Buckets(ctx, ""); err != nil {
		log.Printf("[metrics] r2: %v", err)
		failed = true
	} else {
		snap.R2Buckets = data
	}
	if data, err := m.src.Workers(ctx, ""); err != nil {
		log.Printf("[metrics] workers: %v", err)
		failed = true
	} else {
		snap.Workers = data
	}
	if data, err := m.src.KV(ctx, ""); err != nil {
		log.Printf("[metrics] kv: %v", err)
		failed = true
	} else {
		snap.KVNamespaces = data
	}

	if failed {
		return
	}
	if m.last != nil && reflect.DeepEqual(*m.last, snap) {
		return
	}
	m.last = &snap
	m.srv.Publish("metrics", snap)
}
```

- [ ] **Step 4: Add missing imports to the test file**

Add to the import block of `metrics_test.go`:

```go
import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -run TestMetricsProducer -v`
Expected: 3 PASS

- [ ] **Step 6: Run the full server test suite**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ -v -timeout 30s`
Expected: all PASS, no regressions

- [ ] **Step 7: Commit**

```bash
git add internal/server/metrics.go internal/server/metrics_test.go
ccs commit --direct -m "feat(server): add MetricsProducer with subscriber gating and delta detection"
```

---

### Task 2: Wire MetricsProducer into cmd/serve.go (P-03)

**Files:**
- Modify: `cmd/serve.go:23-24` (flag vars), `cmd/serve.go:58-60` (init), `cmd/serve.go:98` (after SetData)

**Existing code you need to know:**
- `serveAddr` and `serveToken` are package-level flag vars (`cmd/serve.go:23-24`)
- `init()` registers flags on `serveCmd.Flags()` (`cmd/serve.go:57-61`)
- `srv.SetData(...)` is at line 98; the producer starts after this
- The serve context `ctx` is created at line 112-113

- [ ] **Step 1: Add the flag variable**

In `cmd/serve.go`, after line 24 (`serveToken`), add:

```go
serveMetricsInterval time.Duration // --metrics-interval
```

Add `"time"` to the import block (already present from BUG-022 fix).

- [ ] **Step 2: Register the flag in init()**

In `cmd/serve.go`, after the `serveCmd.Flags().StringVar(&serveToken, ...)` line, add:

```go
serveCmd.Flags().DurationVar(&serveMetricsInterval, "metrics-interval", 30*time.Second, "how often to poll Cloudflare for live metrics (0 to disable)")
```

- [ ] **Step 3: Start the producer after SetData**

In `cmd/serve.go`, after `srv.SetData(&serveAdapter{cm: cm})` (line 98), add:

```go
if serveMetricsInterval > 0 {
	mp := server.NewMetricsProducer(srv, &serveAdapter{cm: cm}, serveMetricsInterval)
	mp.Start(ctx)
}
```

Note: `ctx` is created at line 112 *after* the httpServer creation at line 100. The producer needs the serve context. Move the producer start to after line 121 (after the signal handler goroutine), before `errCh`. Also, extract the adapter to a local variable so it's reused (not created twice):

On line 98, change `srv.SetData(&serveAdapter{cm: cm})` to:

```go
adapter := &serveAdapter{cm: cm}
srv.SetData(adapter)
```

Then after the signal handler goroutine (line 121), before `errCh`:

```go
if serveMetricsInterval > 0 {
	mp := server.NewMetricsProducer(srv, adapter, serveMetricsInterval)
	mp.Start(ctx)
}
```

- [ ] **Step 4: Verify it builds**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go build -o /dev/null .`
Expected: exit 0

- [ ] **Step 5: Run cmd tests**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./cmd/ -timeout 60s`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/serve.go
ccs commit --direct -m "feat(serve): wire MetricsProducer with --metrics-interval flag"
```

---

### Task 3: Dashboard SSE → React Query cache bridge (P-04)

**Files:**
- Modify: `desktop/src/App.tsx:60-72` (onEvent handler)

**Existing code you need to know:**
- `queryClient` is a module-level `QueryClient` instance (`App.tsx:17`)
- The `useDaemonSSE` callback currently filters to `status` only (`App.tsx:63`)
- Dashboard `useQuery` keys are `["/zones", profile]`, `["/r2/buckets", profile]`, `["/workers", profile]`, `["/kv", profile]` (`Dashboard.tsx:44-49`)
- The `selected` state variable holds the current profile name (`App.tsx:33`)

- [ ] **Step 1: Define the MetricsPayload type**

In `App.tsx`, after the existing imports (line 12), add:

```typescript
interface MetricsPayload {
  profile: string;
  zones: unknown[];
  r2_buckets: unknown[];
  workers: unknown[];
  kv_namespaces: unknown[];
}
```

- [ ] **Step 2: Update the onEvent handler to process metrics frames**

Replace the `useDaemonSSE` call (lines 60-72) with:

```typescript
useDaemonSSE(
  endpoint,
  (e) => {
    if (e.channel === "status") {
      const s = e.data as StatusPayload;
      if (typeof s.systems_online === "boolean") setSystemsOnline(s.systems_online);
      if (typeof s.cloudflare_online === "boolean") setCloudflareOnline(s.cloudflare_online);
    } else if (e.channel === "metrics") {
      const m = e.data as MetricsPayload;
      const profile = m.profile || selected;
      if (m.zones) queryClient.setQueryData(["/zones", profile], m.zones);
      if (m.r2_buckets) queryClient.setQueryData(["/r2/buckets", profile], m.r2_buckets);
      if (m.workers) queryClient.setQueryData(["/workers", profile], m.workers);
      if (m.kv_namespaces) queryClient.setQueryData(["/kv", profile], m.kv_namespaces);
    }
  },
  (connected) => {
    setSystemsOnline(connected);
    if (!connected) setCloudflareOnline(false);
  }
);
```

- [ ] **Step 3: Run the frontend tests**

Run: `cd desktop && bun run test -- --run`
Expected: all vitest tests pass (15/15)

- [ ] **Step 4: Run type check**

Run: `cd desktop && bun run tsc --noEmit`
Expected: exit 0, no type errors

- [ ] **Step 5: Commit**

```bash
git add desktop/src/App.tsx
ccs commit --direct -m "feat(desktop): bridge SSE metrics channel to React Query cache"
```

---

### Task 4: Update USAGE.md + close IDEA-037

**Files:**
- Modify: `docs/USAGE.md` (add `--metrics-interval` to `serve` section)
- Close: `IDEA-037`

- [ ] **Step 1: Add --metrics-interval to USAGE.md**

Find the `cosmoflare serve` section in `docs/USAGE.md` and add the new flag to the options table:

```
--metrics-interval duration   How often to poll Cloudflare for live metrics (default 30s, 0 to disable)
```

- [ ] **Step 2: Close the idea**

Run: `ccs idea update IDEA-037 --status done`

- [ ] **Step 3: Final full test suite**

Run: `GOWORK=off COSMOFLARE_NO_KEYCHAIN=1 go test ./internal/server/ ./cmd/ -timeout 60s`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add docs/USAGE.md
ccs commit --direct -m "docs: add --metrics-interval to USAGE.md, close IDEA-037"
```
