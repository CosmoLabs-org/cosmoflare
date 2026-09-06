---
created: 2026-06-07T02:30:00-03:00
deliverables:
    - BR-01: DataSource interface with API and null backends
    - BR-02: Tiered refresh strategy (auto-poll monitoring, manual elsewhere)
    - BR-03: Live monitoring panel for R2, Workers, KV with delta indicators
    - BR-04: Real bucket list with CRUD and sort/filter
    - BR-05: Basic object list with pagination
    - BR-06: Graceful credential degradation
    - BR-07: Metrics command absorption into dashboard
last_review_content_hash: 6f31cbfc80ec7eb27152cd5682cf829340ef7239cf18fd28d68d37889bb86adc
last_review_findings: 0
last_review_ref: docs/brainstorming/2026-06-07-road020-dashboard-tui.md
last_reviewed: "2026-09-06T20:17:20.147964+04:00"
origin: ROAD-020
plan_ref: docs/planning-mode/2026-06-07-road020-dashboard-tui.md
roadmap: ROAD-020
status: approved
title: 'ROAD-020: Dashboard TUI Real Implementation'
---

# ROAD-020: Dashboard TUI Real Implementation

## Summary

Wire the existing Bubble Tea dashboard shell (`internal/tui/`) with live Cloudflare API data. The dashboard currently has full navigation, theming, command palette, and 7 sections — but all data is stubbed. This design adds a `DataSource` abstraction, real API fetching, tiered refresh, and absorbs `cmd/metrics.go` into the dashboard's Monitoring section.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Metrics integration | Absorb into dashboard | One unified TUI; `cosmoflare metrics` becomes a thin alias that auto-navigates to Monitoring |
| Object browsing scope | Basic list only | Simple paginated list per bucket; split-pane layout deferred to ROAD-002 |
| Missing credentials | Graceful degradation | Dashboard launches with empty panels and a banner; user can explore UI and configure from Settings |
| Services tracked | R2 + Workers + KV | Core operational services; DNS/Zones/SSL are "set and forget" — not worth polling |
| Refresh strategy | Tiered | Monitoring auto-polls at `--interval`; other sections refresh on navigation entry or manual `r` keypress |

## Architecture

### DataSource Interface

New file: `internal/tui/datasource.go`

```go
type DataSource interface {
    FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error)
    FetchObjects(ctx context.Context, bucket string, page int) ([]Object, int, error)
    FetchMetrics(ctx context.Context) (ServiceMetrics, error)
    CreateBucket(ctx context.Context, name string) error
    DeleteBucket(ctx context.Context, name string) error
    Available() bool
}
```

Two implementations:
- `apiDataSource` — wraps `pkg/cosmoflare` clients (R2Client, WorkerService, KVService). Constructed from `config.ConfigManager` credentials or environment variables.
- `nullDataSource` — returns empty results and `Available() == false`. Used when no credentials are configured.

**Type mapping note:** `internal/tui/model.go` already defines its own `Bucket` struct (with `Status` field) that differs from `pkg/cosmoflare.Bucket` (which has `Location`, `Tags`). The `DataSource` should return the existing TUI `Bucket` type, converting from the library type internally. Similarly, `Object` should project from `pkg/cosmoflare.Object` (7 fields) to a TUI-specific struct with the 4 display fields.

**API reality:** `R2Client.ListBuckets` returns only name and creation date per the S3 API. Size and object count require per-bucket `HeadBucket` or listing calls. The `FetchBuckets` implementation must make N+1 API calls (list + per-bucket stats) or accept that size/count may be unavailable on first load. Recommended: fetch the bucket list immediately, then hydrate size/count asynchronously via background commands — show "—" until data arrives.

### Data Types

```go
type Object struct {
    Key          string
    Size         int64
    LastModified time.Time
    ContentType  string
}

type ServiceMetrics struct {
    R2       R2Metrics
    Workers  WorkersMetrics
    KV       KVMetrics
    FetchedAt time.Time
}

type R2Metrics struct {
    BucketCount  int
    TotalSize    int64
    TotalObjects int64
}

type WorkersMetrics struct {
    Count int
}

type KVMetrics struct {
    NamespaceCount int
}
```

### Tiered Refresh

| Section | Trigger | Mechanism |
|---------|---------|-----------|
| Monitoring | Auto-poll at `--interval` (default 30s) | `tea.Tick` → `monitoringTickMsg` → `DataSource.FetchMetrics` |
| Overview | On navigation entry + manual `r` | `sectionEnteredMsg` → `DataSource.FetchBuckets` (for summary cards) |
| Bucket List | On navigation entry + manual `r` | Same as Overview |
| Object List | On bucket selection + `r` | `bucketSelectedMsg` → `DataSource.FetchObjects(bucket, page)` |

Monitoring stores current + previous snapshots to compute deltas (↑3 buckets, ↓2 objects, etc.).

### Credential Handling

Startup:
1. `config.NewConfigManager()` → `GetCurrent()` (hydrates from keychain via FEAT-005)
2. Fallback: `config.LoadFromEnvironment()`
3. If both fail: `nullDataSource` — UI renders with banner: `"⚠ No credentials configured — press S for Settings or run cosmoflare setup"`

Per-section error handling:
- Each section fetches independently; one failure doesn't block others
- Errors render inline: `"❌ Failed to fetch Workers: 403 Forbidden"`
- Transient errors (5xx, timeouts) auto-retry on next cycle
- Rate limiting (429): skip the next poll cycle entirely and show `"⏳ Rate limited — skipping next refresh"` indicator. The tiered refresh operates per-section, not per-service, so per-service backoff is not practical — a simple skip-next is sufficient

## Sections

### Overview (landing page)
- Storage usage progress bar (R2 total used / account limit)
- Summary cards: bucket count, total objects, total storage, Workers deployed, KV namespaces
- Session notifications (errors, successes)
- Quick actions strip: Create Bucket, Upload File, View Metrics

### Bucket List
- Table columns: Name, Size, Object Count, Created Date, Status
- Cursor navigation, `Enter` drills into Object List
- `c` create bucket (inline text input at bottom of screen), `d` delete (two-keypress confirm: `d` then `y`/`n` prompt in footer), `r` refresh, `s` cycle sort, `/` search

### Object List (basic — ROAD-002 upgrades to split-pane)
- Flat list: Key, Size, Last Modified, Content Type
- Paginated: 100 objects per page, `n`/`p` for next/prev
- `Backspace` returns to Bucket List
- No upload, no split-pane — deferred to ROAD-002

### Monitoring (absorbs cmd/metrics.go)
- Three service blocks (side-by-side on wide terminals, stacked on narrow):
  - **R2:** bucket count, total size, total objects + deltas since last poll
  - **Workers:** deployed count + delta
  - **KV:** namespace count + delta
- Last refresh timestamp + countdown to next poll
- `r` force refresh, `p` pause/resume polling

### Settings, Help, Upload
Unchanged from current stubs.

## Keyboard Shortcuts

| Key | Context | Action |
|-----|---------|--------|
| `r` | Any section | Refresh current section |
| `p` | Monitoring | Pause/resume auto-poll |
| `Enter` | Bucket List | Drill into object list |
| `Backspace` | Object List | Back to bucket list |
| `c` | Bucket List | Create bucket |
| `d` | Bucket List | Delete bucket (confirm) |
| `/` | Bucket List | Search/filter |
| `s` | Bucket List | Cycle sort field |
| `n` | Object List | Next page |
| `p` | Object List | Previous page |
| `Tab`/`1-7` | Global | Section navigation (existing) |
| `F1` | Global | Help (existing) |
| `Ctrl+P` | Global | Command palette (existing) |
| `q` | Global | Quit (existing) |

## File Changes

| File | Change |
|------|--------|
| `internal/tui/datasource.go` | **New.** `DataSource` interface, `apiDataSource`, `nullDataSource` |
| `internal/tui/model.go` | Add `DataSource`, `pollInterval`, `pollPaused`, object list state. Remove hardcoded `RealTimeStats` |
| `internal/tui/update.go` | Wire `monitoringTickMsg`, per-section data messages, object pagination, bucket CRUD commands |
| `internal/tui/view.go` | Real `renderOverview` with summary cards, `renderMonitoring` with 3-service live panel + deltas, `renderObjectList` with pagination, credential banner |
| `internal/tui/dashboard.go` | Accept interval config, construct `DataSource` from credentials, pass to model |
| `cmd/dashboard.go` | Add `--interval` flag |
| `cmd/metrics.go` | Replace with thin wrapper launching dashboard at Monitoring section |

## Out of Scope

- Split-pane object browser (ROAD-002)
- Upload from dashboard UI
- Settings section wiring
- DNS/Zones/SSL/D1/Pages/Queues monitoring
- New TUI components

## Estimated Effort

~400-500 lines new (`datasource.go`), ~200 lines modified (`model.go`, `update.go`, `view.go`), ~50 lines cmd layer. Moderate.
