---
created: "2026-07-01T16:05:00+04:00"
deliverables:
    - description: Daemon goroutine polls CF API and publishes to SSE metrics channel, but only while SSE subscribers are connected.
      id: BR-01
      title: 'Decision: hybrid server-side polling with subscriber gating'
    - description: Full response arrays from existing 4 list calls forwarded directly — cache-compatible, no new CF API endpoints.
      id: BR-02
      title: 'Decision: extract usage hints from existing list responses'
    - description: Default 30s, user-tunable. 4 CF API calls per cycle, well within 1200 req/5min rate limit.
      id: BR-03
      title: 'Decision: configurable interval via --metrics-interval flag'
    - description: onEvent metrics handler calls queryClient.setQueryData, existing useQuery hooks re-render. Single source of truth.
      id: BR-04
      title: 'Decision: SSE updates React Query cache directly'
    - description: Full component design with lifecycle, payload shape, edge cases, and file scope.
      id: BR-05
      title: 'Design: MetricsProducer component specification'
id: BR-2026-07-01-live-metrics-producer
last_review_content_hash: a2a4cda49bb2ca1bbbfb1ed90942b5c646956ee8e23018e404f633934e864c36
last_review_findings: 0
last_review_ref: docs/brainstorming/2026-07-01-live-metrics-producer.md
last_reviewed: "2026-09-06T20:14:04.16862+04:00"
plan_ref: docs/planning-mode/2026-07-01-live-metrics-producer.md
related_issues:
    - IDEA-037
    - ROAD-080
status: validated
tags:
    - desktop
    - daemon
    - sse
    - metrics
    - polling
title: Live metrics producer for the SSE metrics channel (IDEA-037)
updated: "2026-07-01T16:05:00+04:00"
---

# Live metrics producer for the SSE metrics channel (IDEA-037)

## Problem

The Cosmoflare Desktop dashboard fetches resource counts (zones, R2 buckets,
workers, KV namespaces) via REST on initial load and profile switch. After that,
the data goes stale. The SSE `metrics` channel exists and the React client
subscribes to it, but nothing in the daemon publishes to it.

## Design Decisions

### BR-01: Hybrid server-side polling with subscriber gating

**Options considered:**
1. Server-side daemon goroutine (always polling)
2. Client-side `refetchInterval` on React Query hooks
3. Hybrid: daemon polls only while SSE subscribers are connected

**Chosen: Option 3 (hybrid).** Centralizes polling so multiple clients (desktop +
future mobile) share one set of API calls. Subscriber gating avoids wasted CF API
calls when the dashboard isn't open.

### BR-02: Counts + usage hints from existing list responses

**Options considered:**
1. Resource counts only (4 integers)
2. Counts + metadata already present in list responses (object counts, timestamps)
3. Counts + dedicated CF analytics API calls

**Chosen: Option 2.** Extract what the existing `ServeSource` list methods already
return without adding API calls. Keeps the door open for richer metrics later.

### BR-03: Configurable interval via --metrics-interval flag

**Options considered:**
1. Fixed 30s
2. Fixed 60s
3. Configurable with 30s default

**Chosen: Option 3.** `cosmoflare serve --metrics-interval 30s`. Each poll cycle
makes 4 CF API calls; at 30s that's 8 calls/min against a 1,200 req/5min limit.
Power users can tune the rate-limit budget.

### BR-04: SSE updates React Query cache directly

**Options considered:**
1. SSE `onEvent` callback updates React Query cache via `queryClient.setQueryData`
2. Separate `useMetrics` hook with its own state

**Chosen: Option 1.** Reuses the existing data flow. Dashboard code is untouched —
`useQuery` hooks re-render when the cache updates. Single source of truth per
query key. REST calls become initial-load only; SSE takes over for updates.

## Component Design (BR-05)

### MetricsProducer (`internal/server/metrics.go`)

A goroutine that periodically polls the CF API and publishes snapshots to the
SSE `metrics` channel.

**Lifecycle:**
- `Start(ctx context.Context)` launches the goroutine
- `ctx` cancellation stops the goroutine (via `cmd/serve.go`'s serve context)
- Goroutine owns a `time.Ticker` at the configured interval

**Subscriber gating:**
- Before each tick's API calls, checks `Server.Subscribers() > 0`
- If no subscribers, skips the cycle (no wasted CF API calls)

**Poll cycle:**
1. Call `ServeSource.Zones/R2Buckets/Workers/KV` for the default config profile
   (empty string → `resolveProfile("")` → current profile; v1 limitation)
2. Extract counts (and any usage hints) from responses
3. Compare against the last-published snapshot
4. If anything changed (or first tick): `Server.Publish("metrics", snapshot)`

**Payload shape:**
```json
{
  "profile": "default",
  "zones": [ ... full array from ServeSource.Zones ... ],
  "r2_buckets": [ ... full array from ServeSource.R2Buckets ... ],
  "workers": [ ... full array from ServeSource.Workers ... ],
  "kv_namespaces": [ ... full array from ServeSource.KV ... ]
}
```

The producer forwards the raw `ServeSource` response arrays directly. This
keeps the SSE payload cache-compatible with React Query's expected shape
(`Array.isArray(data) ? data.length`) without synthetic array conversion, and
preserves item-level details for future dashboard features.

**Error handling:**
- CF API error → log, publish nothing, retry next tick
- Does NOT flip `cfOnline` (that's the REST handlers' responsibility)

### Wiring (`cmd/serve.go`)

- New `--metrics-interval` flag (default `30s`, type `time.Duration`)
- After `srv.SetData(...)`: construct and start the producer
- Producer receives the serve `ctx` so it dies on shutdown

### Dashboard consumption (`desktop/src/App.tsx`)

- In the existing `onEvent` SSE callback, add a `metrics` channel case
- Parse the payload and call `queryClient.setQueryData` for each service key:
  `["/zones", profile]`, `["/r2/buckets", profile]`, etc.
- Payload arrays are cache-compatible — no conversion needed
- No changes to `Dashboard.tsx` or `sse.ts`

## File Scope

| File | Change |
|------|--------|
| `internal/server/metrics.go` | **NEW** — MetricsProducer struct + Start |
| `internal/server/metrics_test.go` | **NEW** — tests with fake ServeSource |
| `cmd/serve.go` | Add `--metrics-interval` flag, construct + start producer |
| `desktop/src/App.tsx` | Add `metrics` case to SSE onEvent handler |

**Unchanged:** `ServeSource` interface, `Dashboard.tsx`, `sse.go`, `sse.ts`

## Edge Cases

- **No SSE clients:** producer skips API calls
- **Profile switch:** REST fetch fires immediately (query key change); the
  metrics producer continues polling the default profile (it has no mechanism
  to learn the dashboard's selected profile). Tracking the active profile and
  multi-profile concurrent polling are v2.
- **CF API errors:** logged, skipped, retried next tick
- **Server.Close():** ctx cancellation stops the producer goroutine
- **Multiple connected clients:** one set of API calls, all clients receive the
  same SSE frame (hub fans out)

## Out of Scope

- Event bus integration (ROAD-080 `internal/events` not built yet)
- Multi-profile concurrent polling
- Dedicated CF analytics/usage API endpoints
- Dashboard UI changes (cards, charts, new widgets)
