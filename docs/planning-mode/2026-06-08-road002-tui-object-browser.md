---
title: "ROAD-002: TUI Object Browser Implementation Plan"
created: 2026-06-08T01:00:00-03:00
status: READY
brainstorm_ref: docs/brainstorming/2026-06-07-road002-tui-object-browser.md
origin: ROAD-002
deliverables:
  - P-01: Library extension — ListResult.CommonPrefixes + ContinuationToken
  - P-02: DataSource interface extension — FetchObjects, HeadObject, DeleteObject
  - P-03: BrowserModel core — struct, pane focus, bucket list rendering
  - P-04: Folder navigation with prefix stack
  - P-05: Bottom detail panel and HeadObject modal
  - P-06: Object delete with two-keypress confirmation
  - P-07: Integration — wire BrowserModel into DashboardModel, remove SectionObjectList
---

# ROAD-002: TUI Object Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the basic flat object list with a split-pane file browser featuring folder navigation, a detail panel, and object delete — all within a self-contained `BrowserModel` sub-model.

**Architecture:** A `BrowserModel` in `internal/tui/browser.go` owns both panes (bucket list left, object list right), prefix-based folder navigation, and object actions. The parent `DashboardModel` delegates to it for the browser section. The `pkg/cosmoflare` library is extended with `CommonPrefixes` and `ContinuationToken` support. The `DataSource` interface gains `HeadObject`, `DeleteObject`, and a token-based `FetchObjects` with prefix.

**Tech Stack:** Go, Bubble Tea, Lipgloss, `pkg/cosmoflare` (R2Client), `internal/tui` (DataSource)

---

## File Structure

| File | Responsibility |
|------|---------------|
| `pkg/cosmoflare/types.go` | Add `CommonPrefixes` to `ListResult` |
| `pkg/cosmoflare/client.go` | Add `continuationToken` param to `ListObjects` in R2Client interface |
| `pkg/cosmoflare/storage.go` | Populate `CommonPrefixes`, pass `ContinuationToken` to S3 SDK |
| `internal/tui/datasource.go` | New `FetchObjects` signature, `ObjectListing`, `ObjectDetail`, `HeadObject`, `DeleteObject` |
| `internal/tui/browser.go` | **New.** `BrowserModel` — both panes, prefix stack, detail panel, actions |
| `internal/tui/browser_test.go` | **New.** Tests for browser logic |
| `internal/tui/model.go` | Remove `SectionObjectList`, add `browser BrowserModel`, remove old object fields |
| `internal/tui/update.go` | Delegate browser section to `BrowserModel.Update`, remove old object handlers |
| `internal/tui/view.go` | Browser section renders via `m.browser.View()`, remove old `renderObjectList`/`renderBucketTable` |

---

### Task 1: Library Extension — CommonPrefixes and ContinuationToken

**Files:**
- Modify: `pkg/cosmoflare/types.go:61-65`
- Modify: `pkg/cosmoflare/client.go:28`
- Modify: `pkg/cosmoflare/storage.go:104-144`

- [ ] **Step 1: Add CommonPrefixes field to ListResult**

In `pkg/cosmoflare/types.go`, add the field to the `ListResult` struct:

```go
type ListResult[T any] struct {
	Items          []T      `json:"items"`
	NextToken      string   `json:"next_token,omitempty"`
	IsTruncated    bool     `json:"is_truncated"`
	CommonPrefixes []string `json:"common_prefixes,omitempty"`
}
```

- [ ] **Step 2: Extend ListObjects interface with continuationToken parameter**

In `pkg/cosmoflare/client.go`, change the `ListObjects` signature in the `R2Client` interface:

```go
ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continuationToken string) (*ListResult[*Object], error)
```

- [ ] **Step 3: Update ListObjects implementation in storage.go**

In `pkg/cosmoflare/storage.go`, update the `ListObjects` method:

```go
func (c *client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continuationToken string) (*ListResult[*Object], error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("ListObjects", err.Error())
	}

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		params.Prefix = aws.String(prefix)
	}
	if delimiter != "" {
		params.Delimiter = aws.String(delimiter)
	}
	if maxKeys > 0 {
		params.MaxKeys = aws.Int32(maxKeys)
	}
	if continuationToken != "" {
		params.ContinuationToken = aws.String(continuationToken)
	}

	result, err := c.s3Client().ListObjectsV2(ctx, params)
	if err != nil {
		return nil, newError("ListObjects", "failed to list objects", err)
	}

	items := make([]*Object, 0, len(result.Contents))
	for _, obj := range result.Contents {
		items = append(items, &Object{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		})
	}

	prefixes := make([]string, 0, len(result.CommonPrefixes))
	for _, cp := range result.CommonPrefixes {
		prefixes = append(prefixes, aws.ToString(cp.Prefix))
	}

	return &ListResult[*Object]{
		Items:          items,
		NextToken:      aws.ToString(result.NextContinuationToken),
		IsTruncated:    aws.ToBool(result.IsTruncated),
		CommonPrefixes: prefixes,
	}, nil
}
```

- [ ] **Step 4: Fix all callers of ListObjects to pass the new parameter**

Search all callers and add `""` as the continuation token:

```bash
grep -rn "\.ListObjects(" pkg/cosmoflare/ cmd/ internal/ --include="*.go" | grep -v "_test.go"
```

For each caller, append `""` as the last argument. Key callers:
- `internal/tui/datasource.go` — `a.r2.ListObjects(ctx, bucket, "", "", int32(objectsPerPage))` → add `, ""`
- `cmd/metrics.go` — `runMetricsJSON` function if it calls ListObjects
- `pkg/cosmoflare/terraform.go` — ListObjects call
- Any other callers found by grep

Also fix test files that call `ListObjects` (search `*_test.go` files).

- [ ] **Step 5: Run build and tests**

Run: `go build -o /dev/null . && go test ./pkg/cosmoflare/ -timeout 60s -count=1 2>&1 | tail -5`
Expected: Build success, tests pass (may timeout on full pkg suite — focus on compilation)

- [ ] **Step 6: Commit**

```bash
ccs commit --direct -m "feat(lib): add CommonPrefixes and ContinuationToken to ListObjects (ROAD-002 P-01)"
```

---

### Task 2: DataSource Interface Extension

**Files:**
- Modify: `internal/tui/datasource.go`
- Modify: `internal/tui/datasource_test.go`

- [ ] **Step 1: Write failing tests for new interface methods**

Add to `internal/tui/datasource_test.go`:

```go
func TestNullDataSourceHeadObject(t *testing.T) {
	ds := &nullDataSource{}
	_, err := ds.HeadObject(context.Background(), "bucket", "key")
	if err == nil {
		t.Error("expected error from nullDataSource.HeadObject")
	}
}

func TestNullDataSourceDeleteObject(t *testing.T) {
	ds := &nullDataSource{}
	err := ds.DeleteObject(context.Background(), "bucket", "key")
	if err == nil {
		t.Error("expected error from nullDataSource.DeleteObject")
	}
}

func TestNullDataSourceFetchObjectsWithPrefix(t *testing.T) {
	ds := &nullDataSource{}
	listing, err := ds.FetchObjects(context.Background(), "bucket", "prefix/", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listing.Dirs) != 0 {
		t.Errorf("expected 0 dirs, got %d", len(listing.Dirs))
	}
	if len(listing.Objects) != 0 {
		t.Errorf("expected 0 objects, got %d", len(listing.Objects))
	}
	if listing.HasMore {
		t.Error("expected HasMore=false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestNullDataSourceHead|TestNullDataSourceDelete|TestNullDataSourceFetchObjectsWithPrefix" -v`
Expected: FAIL — methods don't exist / signatures don't match

- [ ] **Step 3: Add new types to datasource.go**

Add after existing types:

```go
// ObjectListing holds the result of a prefix-scoped object listing.
type ObjectListing struct {
	Dirs      []string
	Objects   []ObjectItem
	NextToken string
	HasMore   bool
}

// ObjectDetail holds full metadata from HeadObject.
type ObjectDetail struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	ETag         string
	Metadata     map[string]string
}
```

Add `ETag string` field to `ObjectItem`:

```go
type ObjectItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	ETag         string
}
```

- [ ] **Step 4: Update DataSource interface**

Change `FetchObjects` signature and add new methods:

```go
type DataSource interface {
	FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error)
	FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error)
	FetchMetrics(ctx context.Context) (ServiceMetrics, error)
	HeadObject(ctx context.Context, bucket, key string) (*ObjectDetail, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	DeleteObject(ctx context.Context, bucket, key string) error
	Available() bool
}
```

- [ ] **Step 5: Update nullDataSource**

```go
func (n *nullDataSource) FetchObjects(_ context.Context, _, _, _ string) (ObjectListing, error) {
	return ObjectListing{}, nil
}

func (n *nullDataSource) HeadObject(_ context.Context, _, _ string) (*ObjectDetail, error) {
	return nil, fmt.Errorf("no credentials configured — run cosmoflare setup")
}

func (n *nullDataSource) DeleteObject(_ context.Context, _, _ string) error {
	return fmt.Errorf("no credentials configured — run cosmoflare setup")
}
```

- [ ] **Step 6: Update apiDataSource**

```go
func (a *apiDataSource) FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error) {
	result, err := a.r2.ListObjects(ctx, bucket, prefix, "/", int32(objectsPerPage), continuationToken)
	if err != nil {
		return ObjectListing{}, err
	}

	items := make([]ObjectItem, 0, len(result.Items))
	for _, obj := range result.Items {
		items = append(items, ObjectItem{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
		})
	}

	return ObjectListing{
		Dirs:      result.CommonPrefixes,
		Objects:   items,
		NextToken: result.NextToken,
		HasMore:   result.IsTruncated,
	}, nil
}

func (a *apiDataSource) HeadObject(ctx context.Context, bucket, key string) (*ObjectDetail, error) {
	result, err := a.r2.HeadObject(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	return &ObjectDetail{
		Key:          result.Key,
		Size:         result.Size,
		LastModified: result.LastModified,
		ContentType:  result.ContentType,
		ETag:         result.ETag,
		Metadata:     result.Metadata,
	}, nil
}

func (a *apiDataSource) DeleteObject(ctx context.Context, bucket, key string) error {
	return a.r2.DeleteObject(ctx, bucket, key)
}
```

- [ ] **Step 7: Fix existing callers**

Update all existing code that calls the old `FetchObjects(ctx, bucket, page)` to use the new signature `FetchObjects(ctx, bucket, "", "")`. Key locations:
- `internal/tui/update.go` — `fetchObjectsCmd` function
- `internal/tui/datasource_test.go` — existing `TestNullDataSourceFetchObjects`

Update `fetchObjectsCmd` in `update.go`:

```go
func fetchObjectsCmd(ds DataSource, bucket string, prefix string, token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		listing, err := ds.FetchObjects(ctx, bucket, prefix, token)
		return objectsLoadedMsg{listing: listing, err: err}
	}
}
```

Update `objectsLoadedMsg` in `model.go`:

```go
type objectsLoadedMsg struct {
	listing ObjectListing
	err     error
}
```

And its handler in `update.go` — adjust to use `msg.listing.Objects` and `msg.listing`.

- [ ] **Step 8: Run tests**

Run: `go build -o /dev/null . && go test ./internal/tui/ -run TestNullDataSource -v`
Expected: All pass

- [ ] **Step 9: Commit**

```bash
ccs commit --direct -m "feat(tui): extend DataSource with prefix, HeadObject, DeleteObject (ROAD-002 P-02)"
```

---

### Task 3: BrowserModel Core — Struct, Pane Focus, Bucket List

**Files:**
- Create: `internal/tui/browser.go`
- Create: `internal/tui/browser_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/tui/browser_test.go
package tui

import (
	"testing"
)

func TestBrowserModelInitialState(t *testing.T) {
	b := NewBrowserModel(nil)
	if !b.focusLeft {
		t.Error("initial focus should be on left pane")
	}
	if len(b.prefixStack) != 1 || b.prefixStack[0] != "" {
		t.Errorf("initial prefix stack should be [\"\"], got %v", b.prefixStack)
	}
}

func TestBrowserModelSetBuckets(t *testing.T) {
	b := NewBrowserModel(nil)
	buckets := []Bucket{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	b.SetBuckets(buckets)
	if len(b.buckets) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(b.buckets))
	}
}

func TestBrowserModelPaneFocus(t *testing.T) {
	b := NewBrowserModel(nil)
	if !b.focusLeft {
		t.Error("should start on left")
	}
	b.toggleFocus()
	if b.focusLeft {
		t.Error("should be on right after toggle")
	}
	b.toggleFocus()
	if !b.focusLeft {
		t.Error("should be on left after second toggle")
	}
}

func TestBrowserModelSetSize(t *testing.T) {
	b := NewBrowserModel(nil)
	b.SetSize(120, 40)
	if b.width != 120 || b.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", b.width, b.height)
	}
}

func TestBrowserModelIsWide(t *testing.T) {
	b := NewBrowserModel(nil)
	b.SetSize(120, 40)
	if !b.isWide() {
		t.Error("120 columns should be wide mode")
	}
	b.SetSize(80, 40)
	if b.isWide() {
		t.Error("80 columns should be narrow mode")
	}
}

func TestBrowserModelSelectedBucket(t *testing.T) {
	b := NewBrowserModel(nil)
	if b.SelectedBucket() != nil {
		t.Error("no buckets = nil selected")
	}
	b.SetBuckets([]Bucket{{Name: "test"}})
	sel := b.SelectedBucket()
	if sel == nil || sel.Name != "test" {
		t.Error("should return first bucket")
	}
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/tui/ -run TestBrowserModel -v`
Expected: FAIL — `NewBrowserModel` not defined

- [ ] **Step 3: Implement BrowserModel core**

Create `internal/tui/browser.go`:

```go
package tui

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
)

const wideThreshold = 100

type BrowserModel struct {
	data   DataSource
	width  int
	height int

	buckets   []Bucket
	bucketIdx int

	listing     *ObjectListing
	objectIdx   int
	tokenStack  []string
	prefixStack []string

	focusLeft bool

	detail    *ObjectDetail
	detailErr error

	confirmAction string
	confirmTarget string

	showModal      bool
	modalDetail    *ObjectDetail
	modalErr       error

	loadingObjects bool
}

func NewBrowserModel(ds DataSource) BrowserModel {
	return BrowserModel{
		data:        ds,
		focusLeft:   true,
		prefixStack: []string{""},
		tokenStack:  []string{""},
	}
}

func (b *BrowserModel) SetSize(w, h int) { b.width = w; b.height = h }
func (b *BrowserModel) SetBuckets(buckets []Bucket) { b.buckets = buckets }
func (b BrowserModel) isWide() bool { return b.width >= wideThreshold }

func (b *BrowserModel) toggleFocus() { b.focusLeft = !b.focusLeft }

func (b BrowserModel) SelectedBucket() *Bucket {
	if len(b.buckets) == 0 || b.bucketIdx >= len(b.buckets) {
		return nil
	}
	return &b.buckets[b.bucketIdx]
}

func (b BrowserModel) currentPrefix() string {
	if len(b.prefixStack) == 0 {
		return ""
	}
	return b.prefixStack[len(b.prefixStack)-1]
}

func (b BrowserModel) breadcrumb() string {
	bucket := b.SelectedBucket()
	if bucket == nil {
		return ""
	}
	prefix := b.currentPrefix()
	if prefix == "" {
		return bucket.Name
	}
	return bucket.Name + " > " + prefix
}

func (b BrowserModel) totalRightItems() int {
	if b.listing == nil {
		return 0
	}
	return len(b.listing.Dirs) + len(b.listing.Objects)
}

func (b BrowserModel) selectedIsDir() bool {
	if b.listing == nil {
		return false
	}
	return b.objectIdx < len(b.listing.Dirs)
}

func (b BrowserModel) selectedObject() *ObjectItem {
	if b.listing == nil || b.selectedIsDir() {
		return nil
	}
	idx := b.objectIdx - len(b.listing.Dirs)
	if idx < 0 || idx >= len(b.listing.Objects) {
		return nil
	}
	return &b.listing.Objects[idx]
}

func (b BrowserModel) selectedDirName() string {
	if b.listing == nil || !b.selectedIsDir() {
		return ""
	}
	return b.listing.Dirs[b.objectIdx]
}
```

- [ ] **Step 4: Add View() method — renders left pane (bucket list)**

Append to `browser.go`:

```go
func (b BrowserModel) View() string {
	if b.showModal {
		return b.renderModal()
	}

	leftPane := b.renderLeftPane()
	rightPane := b.renderRightPane()

	if !b.isWide() {
		if b.focusLeft {
			return leftPane
		}
		header := lipgloss.NewStyle().Foreground(mutedColor).Render(b.breadcrumb())
		return lipgloss.JoinVertical(lipgloss.Left, header, rightPane)
	}

	leftWidth := b.width * 30 / 100
	rightWidth := b.width - leftWidth - 3

	leftBorder := lipgloss.RoundedBorder()
	rightBorder := lipgloss.RoundedBorder()
	leftColor := mutedColor
	rightColor := mutedColor
	if b.focusLeft {
		leftColor = primaryColor
	} else {
		rightColor = primaryColor
	}

	left := lipgloss.NewStyle().
		Border(leftBorder).BorderForeground(leftColor).
		Width(leftWidth).Height(b.height - 4).
		Render(leftPane)

	right := lipgloss.NewStyle().
		Border(rightBorder).BorderForeground(rightColor).
		Width(rightWidth).Height(b.height - 4).
		Render(rightPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (b BrowserModel) renderLeftPane() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(textColor).Render("Buckets")
	if len(b.buckets) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			lipgloss.NewStyle().Foreground(mutedColor).Render("No buckets."))
	}
	var rows []string
	for i, bucket := range b.buckets {
		indicator := "  "
		style := lipgloss.NewStyle().Foreground(textColor)
		if i == b.bucketIdx {
			indicator = "▸ "
			if b.focusLeft {
				style = lipgloss.NewStyle().Bold(true).Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF"))
			} else {
				style = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
			}
		}
		rows = append(rows, style.Render(indicator+bucket.Name))
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(rows, "\n"))
}

func (b BrowserModel) renderRightPane() string {
	bucket := b.SelectedBucket()
	if bucket == nil {
		return lipgloss.NewStyle().Foreground(mutedColor).Render("Select a bucket to browse.")
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(textColor).Render("📁 " + b.breadcrumb())

	if b.loadingObjects {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "Loading...")
	}

	if b.listing == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			lipgloss.NewStyle().Foreground(mutedColor).Render("Press Enter on a bucket to browse."))
	}

	if b.totalRightItems() == 0 {
		msg := "Empty"
		if b.currentPrefix() != "" {
			msg = "No objects at this prefix."
		}
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			lipgloss.NewStyle().Foreground(mutedColor).Render(msg))
	}

	var rows []string
	for i, dir := range b.listing.Dirs {
		lastSeg := path.Base(strings.TrimSuffix(dir, "/")) + "/"
		style := lipgloss.NewStyle().Foreground(textColor)
		if i == b.objectIdx && !b.focusLeft {
			style = lipgloss.NewStyle().Bold(true).Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF"))
		}
		rows = append(rows, style.Render("📁 "+lastSeg))
	}
	for i, obj := range b.listing.Objects {
		globalIdx := len(b.listing.Dirs) + i
		name := path.Base(obj.Key)
		size := utils.FormatBytes(obj.Size)
		date := obj.LastModified.Format("2006-01-02 15:04")
		line := fmt.Sprintf("  %-30s %8s  %s", truncate(name, 30), size, date)
		style := lipgloss.NewStyle().Foreground(textColor)
		if globalIdx == b.objectIdx && !b.focusLeft {
			style = lipgloss.NewStyle().Bold(true).Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF"))
		}
		rows = append(rows, style.Render(line))
	}

	list := strings.Join(rows, "\n")
	detail := b.renderDetailPanel()
	footer := b.renderBrowserFooter()

	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", detail, footer)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/tui/ -run TestBrowserModel -v`
Expected: All 6 pass

- [ ] **Step 6: Commit**

```bash
ccs commit --direct -m "feat(tui): add BrowserModel core with pane rendering (ROAD-002 P-03)"
```

---

### Task 4: Folder Navigation with Prefix Stack

**Files:**
- Modify: `internal/tui/browser.go` (add Update method)
- Modify: `internal/tui/browser_test.go`

- [ ] **Step 1: Write failing tests for prefix navigation**

Add to `browser_test.go`:

```go
func TestBrowserPrefixPush(t *testing.T) {
	b := NewBrowserModel(nil)
	b.pushPrefix("images/")
	if len(b.prefixStack) != 2 {
		t.Errorf("expected 2 entries, got %d", len(b.prefixStack))
	}
	if b.currentPrefix() != "images/" {
		t.Errorf("expected 'images/', got %q", b.currentPrefix())
	}
}

func TestBrowserPrefixPop(t *testing.T) {
	b := NewBrowserModel(nil)
	b.pushPrefix("images/")
	b.pushPrefix("images/2026/")
	b.popPrefix()
	if b.currentPrefix() != "images/" {
		t.Errorf("expected 'images/', got %q", b.currentPrefix())
	}
}

func TestBrowserPrefixPopAtRoot(t *testing.T) {
	b := NewBrowserModel(nil)
	b.popPrefix()
	if !b.focusLeft {
		t.Error("pop at root should switch focus to left pane")
	}
}

func TestBrowserTokenStack(t *testing.T) {
	b := NewBrowserModel(nil)
	if len(b.tokenStack) != 1 || b.tokenStack[0] != "" {
		t.Errorf("initial token stack should be [\"\"], got %v", b.tokenStack)
	}
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/tui/ -run TestBrowserPrefix -v`
Expected: FAIL — `pushPrefix`/`popPrefix` not defined

- [ ] **Step 3: Implement prefix navigation and Update()**

Add to `browser.go`:

```go
func (b *BrowserModel) pushPrefix(dir string) {
	b.prefixStack = append(b.prefixStack, dir)
	b.objectIdx = 0
	b.listing = nil
	b.tokenStack = []string{""}
}

func (b *BrowserModel) popPrefix() {
	if len(b.prefixStack) <= 1 {
		b.focusLeft = true
		return
	}
	b.prefixStack = b.prefixStack[:len(b.prefixStack)-1]
	b.objectIdx = 0
	b.listing = nil
	b.tokenStack = []string{""}
}

type browserObjectsMsg struct {
	listing ObjectListing
	err     error
}

type browserHeadMsg struct {
	detail *ObjectDetail
	err    error
}

type browserDeleteMsg struct {
	err error
}

func (b BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if b.showModal {
			if msg.String() == "esc" {
				b.showModal = false
				b.modalDetail = nil
				b.modalErr = nil
			}
			return b, nil
		}
		if b.confirmAction != "" {
			return b.handleConfirm(msg)
		}
		return b.handleKey(msg)

	case browserObjectsMsg:
		b.loadingObjects = false
		if msg.err != nil {
			return b, nil
		}
		b.listing = &msg.listing
		b.objectIdx = 0
		return b, nil

	case browserHeadMsg:
		b.showModal = true
		b.modalDetail = msg.detail
		b.modalErr = msg.err
		return b, nil

	case browserDeleteMsg:
		if msg.err != nil {
			return b, nil
		}
		bucket := b.SelectedBucket()
		if bucket != nil {
			return b, b.fetchObjects(bucket.Name, b.currentPrefix(), "")
		}
		return b, nil
	}

	return b, nil
}

func (b BrowserModel) handleKey(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		b.toggleFocus()
		return b, nil

	case "up", "k":
		if b.focusLeft {
			if b.bucketIdx > 0 {
				b.bucketIdx--
			}
		} else {
			if b.objectIdx > 0 {
				b.objectIdx--
			}
		}
		return b, nil

	case "down", "j":
		if b.focusLeft {
			if b.bucketIdx < len(b.buckets)-1 {
				b.bucketIdx++
			}
		} else {
			if b.objectIdx < b.totalRightItems()-1 {
				b.objectIdx++
			}
		}
		return b, nil

	case "enter":
		if b.focusLeft {
			bucket := b.SelectedBucket()
			if bucket != nil {
				b.focusLeft = false
				b.prefixStack = []string{""}
				b.tokenStack = []string{""}
				b.listing = nil
				b.objectIdx = 0
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, "", "")
			}
		} else {
			if b.selectedIsDir() {
				dir := b.selectedDirName()
				b.pushPrefix(dir)
				bucket := b.SelectedBucket()
				if bucket != nil {
					b.loadingObjects = true
					return b, b.fetchObjects(bucket.Name, dir, "")
				}
			} else if obj := b.selectedObject(); obj != nil {
				bucket := b.SelectedBucket()
				if bucket != nil {
					return b, b.headObject(bucket.Name, obj.Key)
				}
			}
		}
		return b, nil

	case "backspace":
		if !b.focusLeft {
			b.popPrefix()
			bucket := b.SelectedBucket()
			if bucket != nil && !b.focusLeft {
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), "")
			}
		}
		return b, nil

	case "r":
		bucket := b.SelectedBucket()
		if bucket != nil && b.data != nil && b.data.Available() {
			if b.focusLeft {
				return b, nil
			}
			b.loadingObjects = true
			return b, b.fetchObjects(bucket.Name, b.currentPrefix(), "")
		}
		return b, nil

	case "n":
		if !b.focusLeft && b.listing != nil && b.listing.HasMore {
			b.tokenStack = append(b.tokenStack, b.listing.NextToken)
			bucket := b.SelectedBucket()
			if bucket != nil {
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), b.listing.NextToken)
			}
		}
		return b, nil

	case "p":
		if !b.focusLeft && len(b.tokenStack) > 1 {
			b.tokenStack = b.tokenStack[:len(b.tokenStack)-1]
			token := b.tokenStack[len(b.tokenStack)-1]
			bucket := b.SelectedBucket()
			if bucket != nil {
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), token)
			}
		}
		return b, nil

	case "d":
		if !b.focusLeft {
			if obj := b.selectedObject(); obj != nil {
				b.confirmAction = "delete"
				b.confirmTarget = obj.Key
			}
		}
		return b, nil
	}

	return b, nil
}

func (b BrowserModel) handleConfirm(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		key := b.confirmTarget
		b.confirmAction = ""
		b.confirmTarget = ""
		bucket := b.SelectedBucket()
		if bucket != nil {
			return b, b.deleteObject(bucket.Name, key)
		}
	case "n", "esc":
		b.confirmAction = ""
		b.confirmTarget = ""
	}
	return b, nil
}

func (b BrowserModel) fetchObjects(bucket, prefix, token string) tea.Cmd {
	if b.data == nil || !b.data.Available() {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		listing, err := b.data.FetchObjects(ctx, bucket, prefix, token)
		return browserObjectsMsg{listing: listing, err: err}
	}
}

func (b BrowserModel) headObject(bucket, key string) tea.Cmd {
	if b.data == nil || !b.data.Available() {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		detail, err := b.data.HeadObject(ctx, bucket, key)
		return browserHeadMsg{detail: detail, err: err}
	}
}

func (b BrowserModel) deleteObject(bucket, key string) tea.Cmd {
	if b.data == nil || !b.data.Available() {
		return nil
	}
	return func() tea.Msg {
		err := b.data.DeleteObject(context.Background(), bucket, key)
		return browserDeleteMsg{err: err}
	}
}
```

Add `"context"` and `"time"` to the import block.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/tui/ -run TestBrowser -v`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
ccs commit --direct -m "feat(tui): add browser folder navigation and Update handler (ROAD-002 P-04)"
```

---

### Task 5: Detail Panel and HeadObject Modal

**Files:**
- Modify: `internal/tui/browser.go` (add rendering methods)

- [ ] **Step 1: Add detail panel rendering**

Add to `browser.go`:

```go
func (b BrowserModel) renderDetailPanel() string {
	if b.focusLeft || b.totalRightItems() == 0 {
		return ""
	}

	if b.selectedIsDir() {
		dir := b.selectedDirName()
		lastSeg := path.Base(strings.TrimSuffix(dir, "/")) + "/"
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(mutedColor).
			Render(fmt.Sprintf("📁 %s — Enter to browse, Backspace to go up", lastSeg))
	}

	obj := b.selectedObject()
	if obj == nil {
		return ""
	}

	line1 := lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(obj.Key)
	line2 := lipgloss.NewStyle().Foreground(textColor).Render(
		fmt.Sprintf("%s | %s | %s",
			utils.FormatBytes(obj.Size),
			obj.LastModified.Format("2006-01-02 15:04:05"),
			truncateETag(obj.ETag)))
	ct := "—"
	if obj.ContentType != "" {
		ct = obj.ContentType
	}
	line3 := lipgloss.NewStyle().Foreground(mutedColor).Render(ct)
	line4 := lipgloss.NewStyle().Foreground(mutedColor).Render("d = delete | Enter = head details")

	panel := lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3, line4)
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(mutedColor).
		Render(panel)
}

func truncateETag(etag string) string {
	if len(etag) > 16 {
		return etag[:16] + "…"
	}
	return etag
}

func (b BrowserModel) renderBrowserFooter() string {
	if b.confirmAction == "delete" {
		return lipgloss.NewStyle().
			Foreground(errorColor).Bold(true).
			Render(fmt.Sprintf("Delete '%s'? (y/n)", path.Base(b.confirmTarget)))
	}

	hints := "Tab = switch pane | ↑↓ = navigate | Enter = open"
	if !b.focusLeft && b.listing != nil && b.listing.HasMore {
		hints += " | n = next page"
	}
	if !b.focusLeft && len(b.tokenStack) > 1 {
		hints += " | p = prev page"
	}
	return lipgloss.NewStyle().Foreground(mutedColor).Render(hints)
}

func (b BrowserModel) renderModal() string {
	if b.modalErr != nil {
		content := lipgloss.NewStyle().Foreground(errorColor).
			Render(fmt.Sprintf("❌ Failed to fetch details: %v", b.modalErr))
		footer := lipgloss.NewStyle().Foreground(mutedColor).Render("Esc = close")
		body := lipgloss.JoinVertical(lipgloss.Left, content, "", footer)
		return lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).BorderForeground(errorColor).
			Padding(1, 2).Width(60).
			Render(body)
	}

	if b.modalDetail == nil {
		return lipgloss.NewStyle().Foreground(mutedColor).Render("Loading...")
	}

	d := b.modalDetail
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Object Details"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Key:          %s", d.Key))
	lines = append(lines, fmt.Sprintf("Size:         %s", utils.FormatBytes(d.Size)))
	lines = append(lines, fmt.Sprintf("Modified:     %s", d.LastModified.Format("2006-01-02 15:04:05 MST")))
	lines = append(lines, fmt.Sprintf("Content-Type: %s", d.ContentType))
	lines = append(lines, fmt.Sprintf("ETag:         %s", d.ETag))

	if len(d.Metadata) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Custom Metadata:"))
		for k, v := range d.Metadata {
			lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(mutedColor).Render("Esc = close"))

	body := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).BorderForeground(primaryColor).
		Padding(1, 2).Width(70).
		Render(body)
}
```

- [ ] **Step 2: Run build**

Run: `go build -o /dev/null .`
Expected: Success

- [ ] **Step 3: Commit**

```bash
ccs commit --direct -m "feat(tui): add browser detail panel and HeadObject modal (ROAD-002 P-05)"
```

---

### Task 6: Integration — Wire BrowserModel into DashboardModel

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/view.go`
- Modify: test files as needed

- [ ] **Step 1: Remove SectionObjectList from Section enum**

In `model.go`, remove `SectionObjectList` from the `const` block and update `String()`. The new enum order:

```go
const (
	SectionOverview Section = iota
	SectionBucketList
	SectionUpload
	SectionMonitoring
	SectionSettings
	SectionHelp
)
```

Update `String()` — remove the `SectionObjectList` case.

- [ ] **Step 2: Add browser field, remove old object fields from DashboardModel**

Remove from `DashboardModel`:
- `objects []ObjectItem`
- `objectPage int`
- `objectTotal int`

Add:
- `browser BrowserModel`

In `initialModel`, initialize the browser:

```go
browser: NewBrowserModel(ds),
```

- [ ] **Step 3: Update Update() to delegate browser messages**

In `update.go`:
- Remove the `objectsLoadedMsg` handler (browser handles its own messages)
- Remove `fetchObjectsCmd` function (browser has its own)
- In the `bucketSelectedMsg` handler — remove object loading, instead pass to browser:
  ```go
  case bucketSelectedMsg:
      m.browser.SetBuckets(m.buckets)
      // Browser handles bucket selection via its own Enter key
      return m, nil
  ```
- In `dataLoadedMsg` handler, after setting `m.buckets`, propagate to browser:
  ```go
  m.browser.SetBuckets(msg.buckets)
  ```

- When `currentSection == SectionBucketList`, delegate key messages and browser messages to `m.browser.Update(msg)`:

```go
case tea.KeyMsg:
    if m.palette != nil && m.palette.IsVisible() {
        // palette handling unchanged
    }
    if m.currentSection == SectionBucketList {
        var cmd tea.Cmd
        m.browser, cmd = m.browser.Update(msg)
        return m, cmd
    }
    return m.handleKeyMsg(msg)

case browserObjectsMsg, browserHeadMsg, browserDeleteMsg:
    var cmd tea.Cmd
    m.browser, cmd = m.browser.Update(msg)
    return m, cmd
```

- [ ] **Step 4: Update view.go to render browser**

In `renderMainContent()`:
- Change `SectionBucketList` to render `m.browser.View()` instead of `m.renderBucketTable()`
- Remove `SectionObjectList` case entirely
- Remove `renderObjectList()` function
- Remove `renderBucketTable()` function (it's now in `browser.go` as `renderLeftPane`)

```go
case SectionBucketList:
    return m.browser.View()
```

- [ ] **Step 5: Propagate window size to browser**

In `Update()` `tea.WindowSizeMsg` handler, add:

```go
m.browser.SetSize(msg.Width, msg.Height)
```

- [ ] **Step 6: Fix section number key shortcuts**

Update the number key handlers in `handleKeyMsg` — remove `"3"` for ObjectList (browser handles this internally). Shift remaining numbers:
- `"1"` → Overview
- `"2"` → BucketList (browser)
- `"3"` → Upload
- `"4"` → Monitoring
- `"5"` → Settings

- [ ] **Step 7: Fix all test files referencing SectionObjectList**

Search: `grep -rn "SectionObjectList" internal/tui/ --include="*_test.go"`

For each reference:
- Remove tests that specifically test SectionObjectList behavior (it no longer exists)
- Update section number tests to use the new numbering
- Update any `initialModel` calls to pass DataSource

- [ ] **Step 8: Run build and full test suite**

Run: `go build -o /dev/null . && go test ./internal/tui/ ./cmd/ -timeout 60s`
Expected: Build success, all tests pass

- [ ] **Step 9: Commit**

```bash
ccs commit --direct -m "feat(tui): integrate BrowserModel, remove SectionObjectList (ROAD-002 P-06)"
```

---

### Task 7: Documentation and Close-Out

**Files:**
- Modify: `docs/USAGE.md`

- [ ] **Step 1: Update USAGE.md dashboard section**

Find the Dashboard section in `docs/USAGE.md`. Update the features list and keyboard shortcuts to reflect the browser:

Add under Features:
```markdown
- Split-pane object browser with folder navigation
- Bottom detail panel with object metadata
- Object deletion with two-keypress confirmation
```

Update keyboard shortcuts table to add browser keys (`Tab`, `Backspace`, `d`, `n`/`p` for objects, `Enter` for drill-down).

- [ ] **Step 2: Add changelog entry**

Run: `ccs changelog add "Add split-pane TUI object browser with folder navigation and object management (ROAD-002)" --type added`

- [ ] **Step 3: Update roadmap**

Run: `ccs roadmap update ROAD-002 --status completed`

- [ ] **Step 4: Commit**

```bash
ccs commit --direct -m "docs: update USAGE.md and close ROAD-002"
```
