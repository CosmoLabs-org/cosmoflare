---
title: "ROAD-002: TUI Object Browser with Split-Pane Layout"
created: 2026-06-07T04:30:00-03:00
status: approved
roadmap: ROAD-002
origin: ROAD-002
last_reviewed: 2026-06-08T00:30:00-03:00
last_review_ref: docs/independent-reviews/2026-06-08T00-30-00-road002-tui-object-browser.md
last_review_findings: 7
deliverables:
  - BR-01: BrowserModel sub-model with split-pane layout
  - BR-02: Pane focus management and narrow/wide mode switching
  - BR-03: Prefix-based folder navigation with stack
  - BR-04: Bottom detail panel with live metadata
  - BR-05: Object delete with two-keypress confirmation
  - BR-06: HeadObject modal for full metadata view
  - BR-07: DataSource interface extension (prefix, HeadObject, DeleteObject)
  - BR-08: Library extension — ListResult.CommonPrefixes
---

# ROAD-002: TUI Object Browser with Split-Pane Layout

## Summary

Replace the basic flat object list (shipped in ROAD-020) with a split-pane file browser. Left pane shows the bucket list, right pane shows objects at the current prefix level with folder navigation. A bottom detail panel shows metadata for the highlighted object. Implemented as a self-contained `BrowserModel` sub-model in `internal/tui/browser.go`.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Narrow terminal handling | Single-pane mode | Show only focused pane, switch with Tab. Preserves mental model without wasting vertical space on stacked layout. |
| Object actions | Browse + delete | Delete is the most common action. Copy, pre-signed URLs, download deferred. Two-keypress confirmation reuses existing pattern. |
| Folder navigation | Prefix-based hierarchy | Uses S3 `delimiter="/"` + `prefix` params. Natural file-manager UX for nested keys. |
| Metadata display | Bottom detail panel | 4-line strip always visible as cursor moves. Instant feedback without disrupting list flow. Modal only for HeadObject full details. |

## Architecture

### BrowserModel Sub-Model

New file: `internal/tui/browser.go`

`BrowserModel` is a self-contained Bubble Tea sub-model that owns both panes, focus state, the prefix stack, and object actions. The parent `DashboardModel` delegates to it when the bucket/object section is active.

```go
type BrowserModel struct {
    data       DataSource
    width      int
    height     int

    // Left pane: bucket list
    buckets    []Bucket
    bucketIdx  int

    // Right pane: object list at current prefix
    listing    *ObjectListing
    objectIdx  int
    objectPage int
    prefixStack []string

    // Pane focus
    focusLeft  bool

    // Detail panel
    detail     *ObjectDetail
    detailErr  error

    // Object actions
    confirmAction string
    confirmTarget string

    // Loading state
    loadingObjects bool
}
```

`BrowserModel` exposes:
- `Update(msg tea.Msg) (BrowserModel, tea.Cmd)` — handles all browser-specific keys and messages
- `View() string` — renders the split-pane or single-pane layout
- `SetSize(w, h int)` — propagates terminal size
- `SetBuckets(buckets []Bucket)` — receives bucket data from parent
- `SelectedBucket() *Bucket` — returns currently highlighted bucket (for parent to show in header, etc.)

### Layout: Wide vs Narrow

**Wide mode (width ≥ 100):**
```
┌─ Buckets ──────┐┌─ my-bucket > images/2026/ ──────────────────┐
│ ▸ my-bucket    ││ 📁 thumbnails/                              │
│   assets       ││ 📁 originals/                               │
│   logs         ││   photo-001.jpg    2.4 MB  2026-06-01 14:32│
│                ││ ▸ photo-002.jpg    1.8 MB  2026-06-01 14:33│
│                ││   photo-003.jpg    3.1 MB  2026-06-02 09:15│
│                ││─────────────────────────────────────────────│
│                ││ photo-002.jpg | 1.8 MB | 2026-06-01 14:33  │
│                ││ image/jpeg | standard | d=delete Enter=head │
└────────────────┘└─────────────────────────────────────────────┘
```

Left pane: ~30% width, right pane: ~70% width. Active pane has highlighted border. `Tab` switches focus.

**Narrow mode (width < 100):**
Shows only the focused pane full-width. `Tab` switches which pane is visible. A breadcrumb at the top shows context: `"my-bucket > images/2026/"` so the user knows where they are even when the bucket list is hidden.

### Folder Navigation

The right pane maintains a `prefixStack []string`:
- Start: `[""]` (bucket root)
- Enter a directory `"images/"`: push → `["", "images/"]`
- Enter deeper `"images/2026/"`: push → `["", "images/", "images/2026/"]`
- `Backspace`: pop → `["", "images/"]`
- `Backspace` at root: switch focus to left pane (bucket list)

The current prefix (top of stack) is passed to `DataSource.FetchObjects(ctx, bucket, prefix, "")` (empty token = first page).

Display entries in the right pane:
- **Directories** first: rendered with `📁` prefix, derived from `CommonPrefixes` in the API response. Show only the last segment (e.g., `"thumbnails/"` not `"images/thumbnails/"`). Sorted alphabetically.
- **Files** after: rendered with Size, Last Modified, Content Type. Sorted alphabetically.

### DataSource Interface Extension

`FetchObjects` signature changes:

```go
// Before (ROAD-020):
FetchObjects(ctx context.Context, bucket string, page int) ([]ObjectItem, int, error)

// After (token-based pagination):
FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error)
```

New types:
```go
type ObjectListing struct {
    Dirs      []string     // common prefixes ("images/", "logs/")
    Objects   []ObjectItem // files at this level
    NextToken string       // pass to next FetchObjects call for pagination
    HasMore   bool         // true if more pages exist
}
```

New methods:
```go
HeadObject(ctx context.Context, bucket, key string) (*ObjectDetail, error)
DeleteObject(ctx context.Context, bucket, key string) error
```

```go
type ObjectDetail struct {
    Key          string
    Size         int64
    LastModified time.Time
    ContentType  string
    ETag         string
    Metadata     map[string]string
}
```

**Library gap — CommonPrefixes:** The current `pkg/cosmoflare` `ListObjects` implementation discards `CommonPrefixes` from the S3 `ListObjectsV2` response. The `ListResult[*Object]` type has no field for them. **The implementation plan must extend `ListResult` with a `CommonPrefixes []string` field** and update `ListObjects` in `pkg/cosmoflare/storage.go` to populate it. Note: the S3 SDK type is `[]types.CommonPrefix` (each has a `.Prefix` field) — extraction requires `aws.ToString(cp.Prefix)` per entry, not direct assignment.

**Library gap — ContinuationToken:** The R2Client `ListObjects` method has no `ContinuationToken` parameter. The implementation plan must also extend the `ListObjects` signature (or add an overload) to accept an optional continuation token input. The S3 SDK's `ListObjectsV2Input` already supports `ContinuationToken`. This is a moderate library change (~15 lines across `client.go` interface + `storage.go` implementation).

**Library gap — HeadResult.StorageClass:** `HeadResult` has no `StorageClass` field. Drop `StorageClass` from `ObjectDetail` — it's not critical for the detail panel and avoids a library change. If needed later, extend `HeadResult`.

**Pagination:** S3 uses continuation tokens, not page numbers. Change `FetchObjects` to use a token-based approach:

```go
FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error)
```

`ObjectListing` gains `NextToken string` and `HasMore bool`. The `BrowserModel` stores `nextToken` and passes it on `n` key. Previous-page is not natively supported by S3 — the browser stores seen tokens in a `[]string` stack (one per page visited).

**ContentType availability:** S3 `ListObjectsV2` does not return `ContentType` per object. The detail panel shows ContentType only after `HeadObject` is fetched (when user presses `Enter`). The inline listing shows `"—"` for content type. `ObjectItem` gains an `ETag` field (available from `ListObjects` via the `Object` type).

The `apiDataSource` implementation:
- `FetchObjects`: calls `r2.ListObjects(ctx, bucket, prefix, "/", objectsPerPage)` with continuation token support. Extracts `CommonPrefixes` for directories and `Items` for files.
- `HeadObject`: calls `r2.HeadObject(ctx, bucket, key)`. Maps `*HeadResult` to `ObjectDetail` (without StorageClass).
- `DeleteObject`: calls `r2.DeleteObject(ctx, bucket, key)`.

The `nullDataSource`: returns empty `ObjectListing` for `FetchObjects`, returns error for `HeadObject` and `DeleteObject`.

### Bottom Detail Panel

A 4-line strip at the bottom of the right pane, always visible when an object (not a directory) is highlighted:
- Line 1: Full key path (untruncated)
- Line 2: Size (formatted) | Last Modified (full timestamp) | ETag
- Line 3: Content Type (shown as "—" until HeadObject fetched via Enter)
- Line 4: Action hints: `d = delete | Enter = head details`

When a directory is highlighted: `"📁 <prefix> — Enter to browse, Backspace to go up"`

When the right pane is empty: detail panel hidden.

The detail panel shows data from the `ObjectItem` (already loaded). The `Enter` key fetches `HeadObject` for the full modal view (custom metadata, exact ETag, storage class).

### Object Actions

- `d` on an object → two-keypress confirmation in footer: `"Delete 'images/photo.jpg'? (y/n)"`. On `y`, calls `DataSource.DeleteObject(ctx, bucket, key)`. On success, refresh the listing. On failure, show error notification inline: `"❌ Failed to delete: <error>"`. Object stays in the list (next refresh will reflect actual state).
- `Enter` on an object → fetch `HeadObject`, show modal overlay with full metadata. On API error (timeout, 404), show error in the modal body: `"❌ Failed to fetch details: <error>"` with `Esc` to dismiss. `Esc` dismisses on success too.
- `Enter` on a directory → push prefix, fetch objects at new level.
- `Enter` on left pane (select bucket) → if API call to fetch objects fails, show error notification and keep the right pane empty with `"❌ Failed to load objects: <error>"`. Don't switch focus — user stays on bucket list.
- `Backspace` at root prefix in narrow mode → swap visible pane to bucket list (same as focus switch in wide mode, but affects visibility).

### Keyboard Shortcuts (Browser Section)

| Key | Context | Action |
|-----|---------|--------|
| `Tab` | Browser | Switch pane focus (wide: highlight, narrow: swap visible pane) |
| `Enter` | Left pane | Select bucket, load objects |
| `Enter` | Right pane, directory | Navigate into directory |
| `Enter` | Right pane, object | Show HeadObject modal |
| `Backspace` | Right pane, at root | Focus left pane |
| `Backspace` | Right pane, in prefix | Pop prefix stack (go up) |
| `d` | Right pane, object | Delete object (with confirmation) |
| `r` | Either pane | Refresh current view |
| `n`/`p` | Right pane | Next/previous page |
| `j`/`k` or `↑`/`↓` | Either pane | Navigate within pane |
| `Esc` | HeadObject modal | Dismiss modal |

## File Changes

| File | Change |
|------|--------|
| `pkg/cosmoflare/types.go` | Add `CommonPrefixes []string` field to `ListResult`. |
| `pkg/cosmoflare/client.go` | Extend `ListObjects` signature with optional `continuationToken` parameter (or add `ListObjectsOption` pattern). |
| `pkg/cosmoflare/storage.go` | Populate `CommonPrefixes` from S3 response (extract `.Prefix` from `[]types.CommonPrefix`). Pass `ContinuationToken` to S3 SDK. ~15 lines total. |
| `internal/tui/browser.go` | **New.** `BrowserModel` sub-model with dual-pane layout, prefix navigation, detail panel, object actions |
| `internal/tui/browser_test.go` | **New.** Tests for pane switching, prefix nav, narrow/wide mode, detail panel |
| `internal/tui/datasource.go` | Change `FetchObjects` to token-based pagination with prefix. Add `HeadObject` and `DeleteObject` methods. Add `ObjectListing` and `ObjectDetail` types. Add `ETag` to `ObjectItem`. Update both backends. |
| `internal/tui/datasource_test.go` | Update tests for new signatures |
| `internal/tui/model.go` | Add `browser BrowserModel` field. Remove `SectionObjectList` from Section enum (18 references across source + tests need updating). Remove object-specific fields (`objects`, `objectPage`, `objectTotal`). Renumber section constants so `SectionUpload` takes the old `SectionObjectList` slot. |
| `internal/tui/update.go` | Delegate `SectionBucketList` keys/messages to `browser.Update()`. Remove object-specific handlers. |
| `internal/tui/view.go` | `SectionBucketList` renders `m.browser.View()`. Remove `renderObjectList()` and `renderBucketTable()` (moved to browser). |

## Out of Scope

- Upload from browser
- Copy/move objects
- Pre-signed URL generation from browser
- File preview/download
- Drag-and-drop
- Multi-select operations
- Search/filter within object list (can be added later)

## Estimated Effort

~350-400 lines new in `browser.go`, ~80 lines datasource changes, ~100 lines model/update/view wiring (net reduction as object code moves to browser). Moderate-to-large.
