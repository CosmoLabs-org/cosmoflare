---
title: "ROAD-020: Dashboard TUI Real Implementation Plan"
created: 2026-06-07T03:15:00-03:00
status: READY
brainstorm_ref: docs/brainstorming/2026-06-07-road020-dashboard-tui.md
origin: ROAD-020
deliverables:
  - P-01: DataSource interface and null backend
  - P-02: API DataSource backend with credential detection
  - P-03: Wire DataSource into DashboardModel
  - P-04: Live monitoring with auto-poll and deltas
  - P-05: Object list with pagination
  - P-06: Bucket create/delete with confirmation
  - P-07: Section-entry refresh and credential banner
  - P-08: CMD layer — interval flag and metrics absorption
  - P-09: Documentation and close-out
---

# ROAD-020: Dashboard TUI Real Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the existing Bubble Tea dashboard shell with live Cloudflare API data, real bucket CRUD, basic object listing, and live monitoring that absorbs `cmd/metrics.go`.

**Architecture:** A `DataSource` interface (`internal/tui/datasource.go`) abstracts API access with two implementations: `apiDataSource` (real API) and `nullDataSource` (no credentials). The existing `DashboardModel` gains a `DataSource` field and new message types for tiered refresh — auto-poll for monitoring, manual for everything else.

**Tech Stack:** Go, Bubble Tea, Lipgloss, `pkg/cosmoflare` (R2Client, WorkerService, KVService), `internal/config` (credential loading + keychain)

**Brainstorm ref:** `docs/brainstorming/2026-06-07-road020-dashboard-tui.md`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/tui/datasource.go` | **New.** `DataSource` interface, `apiDataSource`, `nullDataSource`, type conversion helpers |
| `internal/tui/datasource_test.go` | **New.** Tests for `nullDataSource` and type conversion |
| `internal/tui/model.go` | Add `DataSource`, monitoring state, object list state to `DashboardModel`. Remove `RealTimeStats`. Add new message types. |
| `internal/tui/update.go` | Wire monitoring tick, section-entry refresh, object pagination, bucket CRUD, `r`/`p` keys |
| `internal/tui/view.go` | Real `renderMonitoring` with deltas, real `renderObjectList` with pagination, credential banner |
| `internal/tui/dashboard.go` | Construct `DataSource` from credentials, accept interval config |
| `cmd/dashboard.go` | Add `--interval` flag, pass to `DashboardConfig` |
| `cmd/metrics.go` | Replace TUI model with thin wrapper launching dashboard at Monitoring section |

---

### Task 1: DataSource Interface and Null Backend

**Files:**
- Create: `internal/tui/datasource.go`
- Create: `internal/tui/datasource_test.go`

- [ ] **Step 1: Write the failing tests for nullDataSource**

```go
// internal/tui/datasource_test.go
package tui

import (
	"context"
	"testing"
)

func TestNullDataSourceAvailable(t *testing.T) {
	ds := &nullDataSource{}
	if ds.Available() {
		t.Error("nullDataSource should not be available")
	}
}

func TestNullDataSourceFetchBuckets(t *testing.T) {
	ds := &nullDataSource{}
	buckets, stats, err := ds.FetchBuckets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
	if stats.BucketCount != 0 {
		t.Errorf("expected 0 bucket count, got %d", stats.BucketCount)
	}
}

func TestNullDataSourceFetchObjects(t *testing.T) {
	ds := &nullDataSource{}
	objects, total, err := ds.FetchObjects(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("expected 0 objects, got %d", len(objects))
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
}

func TestNullDataSourceFetchMetrics(t *testing.T) {
	ds := &nullDataSource{}
	metrics, err := ds.FetchMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.R2.BucketCount != 0 {
		t.Errorf("expected 0 R2 buckets, got %d", metrics.R2.BucketCount)
	}
	if metrics.Workers.Count != 0 {
		t.Errorf("expected 0 workers, got %d", metrics.Workers.Count)
	}
}

func TestNullDataSourceCRUD(t *testing.T) {
	ds := &nullDataSource{}
	if err := ds.CreateBucket(context.Background(), "x"); err == nil {
		t.Error("expected error from nullDataSource.CreateBucket")
	}
	if err := ds.DeleteBucket(context.Background(), "x"); err == nil {
		t.Error("expected error from nullDataSource.DeleteBucket")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestNullDataSource -v`
Expected: FAIL — `nullDataSource` not defined

- [ ] **Step 3: Implement DataSource interface and nullDataSource**

```go
// internal/tui/datasource.go
package tui

import (
	"context"
	"fmt"
	"time"
)

// DataSource abstracts Cloudflare API access for the dashboard.
type DataSource interface {
	FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error)
	FetchObjects(ctx context.Context, bucket string, page int) ([]ObjectItem, int, error)
	FetchMetrics(ctx context.Context) (ServiceMetrics, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	Available() bool
}

// ObjectItem is the TUI display projection of a library Object.
type ObjectItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
}

// ServiceMetrics holds live counts for the Monitoring section.
type ServiceMetrics struct {
	R2        R2Metrics
	Workers   WorkersMetrics
	KV        KVMetrics
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

const objectsPerPage = 100

// nullDataSource returns empty data when no credentials are configured.
type nullDataSource struct{}

func (n *nullDataSource) Available() bool { return false }

func (n *nullDataSource) FetchBuckets(_ context.Context) ([]Bucket, UsageStats, error) {
	return []Bucket{}, UsageStats{}, nil
}

func (n *nullDataSource) FetchObjects(_ context.Context, _ string, _ int) ([]ObjectItem, int, error) {
	return []ObjectItem{}, 0, nil
}

func (n *nullDataSource) FetchMetrics(_ context.Context) (ServiceMetrics, error) {
	return ServiceMetrics{FetchedAt: time.Now()}, nil
}

func (n *nullDataSource) CreateBucket(_ context.Context, _ string) error {
	return fmt.Errorf("no credentials configured — run cosmoflare setup")
}

func (n *nullDataSource) DeleteBucket(_ context.Context, _ string) error {
	return fmt.Errorf("no credentials configured — run cosmoflare setup")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestNullDataSource -v`
Expected: PASS (all 5 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/datasource.go internal/tui/datasource_test.go
git commit -m "feat(tui): add DataSource interface and null backend (ROAD-020 P-01)"
```

---

### Task 2: API DataSource Backend

**Files:**
- Modify: `internal/tui/datasource.go` (append `apiDataSource`)

- [ ] **Step 1: Write the failing test for apiDataSource construction**

Add to `internal/tui/datasource_test.go`:

```go
func TestNewAPIDataSource(t *testing.T) {
	// Without valid credentials, newAPIDataSource should return a nullDataSource
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	ds := newDataSource()
	if ds.Available() {
		t.Error("expected unavailable DataSource with no credentials")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestNewAPIDataSource -v`
Expected: FAIL — `newDataSource` not defined

- [ ] **Step 3: Implement apiDataSource and newDataSource constructor**

Append to `internal/tui/datasource.go`:

```go
import (
	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// apiDataSource fetches live data from the Cloudflare API.
type apiDataSource struct {
	r2      cosmoflare.R2Client
	workers *cosmoflare.WorkerService
	kv      *cosmoflare.KVService
}

func (a *apiDataSource) Available() bool { return true }

func (a *apiDataSource) FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error) {
	apiBuckets, err := a.r2.ListBuckets(ctx)
	if err != nil {
		return nil, UsageStats{}, err
	}

	var buckets []Bucket
	var totalUsed int64
	for _, b := range apiBuckets {
		totalUsed += b.Size
		buckets = append(buckets, Bucket{
			Name:        b.Name,
			Size:        b.Size,
			ObjectCount: b.ObjectCount,
			Status:      "active",
			CreatedAt:   b.CreatedAt,
		})
	}

	stats := UsageStats{
		TotalUsed:   totalUsed,
		TotalLimit:  1024 * 1024 * 1024 * 1024, // 1TB default
		BucketCount: len(buckets),
	}
	return buckets, stats, nil
}

func (a *apiDataSource) FetchObjects(ctx context.Context, bucket string, page int) ([]ObjectItem, int, error) {
	result, err := a.r2.ListObjects(ctx, bucket, "", "", int32(objectsPerPage))
	if err != nil {
		return nil, 0, err
	}

	var items []ObjectItem
	for _, obj := range result.Items {
		items = append(items, ObjectItem{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
		})
	}
	return items, len(items), nil
}

func (a *apiDataSource) FetchMetrics(ctx context.Context) (ServiceMetrics, error) {
	metrics := ServiceMetrics{FetchedAt: time.Now()}

	buckets, err := a.r2.ListBuckets(ctx)
	if err == nil {
		metrics.R2.BucketCount = len(buckets)
		for _, b := range buckets {
			metrics.R2.TotalSize += b.Size
			metrics.R2.TotalObjects += b.ObjectCount
		}
	}

	if a.workers != nil {
		workers, err := a.workers.List(ctx)
		if err == nil {
			metrics.Workers.Count = len(workers)
		}
	}

	if a.kv != nil {
		namespaces, err := a.kv.ListNamespaces(ctx)
		if err == nil {
			metrics.KV.NamespaceCount = len(namespaces)
		}
	}

	return metrics, nil
}

func (a *apiDataSource) CreateBucket(ctx context.Context, name string) error {
	_, err := a.r2.CreateBucket(ctx, name)
	return err
}

func (a *apiDataSource) DeleteBucket(ctx context.Context, name string) error {
	return a.r2.DeleteBucket(ctx, name)
}

// newDataSource constructs the best available DataSource from credentials.
func newDataSource() DataSource {
	cm, err := config.NewConfigManager()
	if err != nil {
		return &nullDataSource{}
	}

	profile, err := cm.GetCurrent()
	if err != nil {
		profile = config.LoadFromEnvironment()
	}

	if profile.AccountID == "" || profile.APIToken == "" {
		return &nullDataSource{}
	}

	r2, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(profile.AccountID),
		cosmoflare.WithAPIToken(profile.APIToken),
	)
	if err != nil {
		return &nullDataSource{}
	}

	ds := &apiDataSource{r2: r2}

	if ws, err := cosmoflare.NewWorkerServiceFromCreds(profile.AccountID, profile.APIToken); err == nil {
		ds.workers = ws
	}
	if ks, err := cosmoflare.NewKVServiceFromCreds(profile.AccountID, profile.APIToken); err == nil {
		ds.kv = ks
	}

	return ds
}
```

Note: consolidate the import block at the top of `datasource.go` — the file now needs `context`, `fmt`, `time`, plus the config and cosmoflare imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestNew -v`
Expected: PASS

- [ ] **Step 5: Run full build**

Run: `go build -o build/cosmoflare .`
Expected: Success

- [ ] **Step 6: Commit**

```bash
git add internal/tui/datasource.go internal/tui/datasource_test.go
git commit -m "feat(tui): add API DataSource backend with credential auto-detection (ROAD-020 P-02)"
```

---

### Task 3: Wire DataSource into DashboardModel

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/dashboard.go`

- [ ] **Step 1: Add DataSource and monitoring state to DashboardModel**

In `internal/tui/model.go`, replace the `DashboardModel` struct with:

```go
// DashboardModel is the main TUI model
type DashboardModel struct {
	// Navigation state
	currentSection Section
	selectedRow     int
	cursor          int

	// Data source
	data DataSource

	// Data state
	buckets       []Bucket
	currentBucket *Bucket
	usageStats    UsageStats

	// Monitoring state
	metrics     ServiceMetrics
	prevMetrics ServiceMetrics
	pollInterval time.Duration
	pollPaused   bool
	skipNextPoll bool

	// Object list state
	objects      []ObjectItem
	objectPage   int
	objectTotal  int

	// UI state
	showHelp       bool
	notifications  []Notification
	searchQuery    string
	sortBy         SortField
	filterActive   bool
	loading        bool
	loadingFrame   int
	width          int
	height         int

	// Confirmation state
	confirmAction  string
	confirmTarget  string

	// Background operations
	backgroundTasks []BackgroundTask
	uploadQueue     []UploadTask

	// Settings
	currentProfile string
	theme          Theme

	// Command palette
	palette *palette.PaletteModel
}
```

- [ ] **Step 2: Remove the `RealTimeStats` type**

Delete the `RealTimeStats` struct (lines 72-80 of current `model.go`) and the `realTimeStats` field is already removed in step 1.

- [ ] **Step 3: Update `initialModel` to accept a DataSource**

Replace `initialModel()` with:

```go
func initialModel(ds DataSource, interval time.Duration) DashboardModel {
	theme := darkTheme
	InitializeStyles(theme)

	paletteCommands := palette.DefaultDashboardActions()
	p := palette.New(paletteCommands, 80, 24)

	return DashboardModel{
		data:            ds,
		buckets:         []Bucket{},
		currentSection:  SectionOverview,
		selectedRow:     0,
		loading:         ds.Available(),
		loadingFrame:    0,
		notifications:   []Notification{},
		backgroundTasks: []BackgroundTask{},
		uploadQueue:     []UploadTask{},
		currentProfile:  "default",
		theme:           theme,
		palette:         p,
		pollInterval:    interval,
		objects:         []ObjectItem{},
	}
}
```

- [ ] **Step 4: Update Init() to use DataSource**

Replace `Init()` with:

```go
func (m DashboardModel) Init() tea.Cmd {
	if !m.data.Available() {
		m.loading = false
		return nil
	}
	return loadDataCmd(m.data)
}
```

- [ ] **Step 5: Update `loadDataCmd` to accept DataSource**

Replace `loadDataCmd()` with:

```go
func loadDataCmd(ds DataSource) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		buckets, stats, err := ds.FetchBuckets(ctx)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to load data: %w", err)}
		}
		return dataLoadedMsg{buckets: buckets, stats: stats}
	}
}
```

- [ ] **Step 6: Update `dashboard.go` to construct DataSource**

Replace `RunDashboard()` in `internal/tui/dashboard.go`:

```go
func RunDashboard() error {
	return RunDashboardWithInterval(30 * time.Second)
}

func RunDashboardWithInterval(interval time.Duration) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
	}()

	ds := newDataSource()
	model := initialModel(ds, interval)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithOutput(os.Stderr),
	)

	_, err := p.Run()
	return err
}
```

Update `RunDashboardWithConfig` similarly — pass `config.Profile` to override credential detection:

```go
func RunDashboardWithConfig(config DashboardConfig) error {
	ds := newDataSource()
	interval := 30 * time.Second
	if config.Interval > 0 {
		interval = config.Interval
	}
	model := initialModel(ds, interval)
	model.currentProfile = config.Profile
	if config.Theme != "" {
		switch config.Theme {
		case "light":
			model.theme = lightTheme
		case "dark":
			model.theme = darkTheme
		}
		InitializeStyles(model.theme)
	}
	if config.Width > 0 {
		model.width = config.Width
	}
	if config.Height > 0 {
		model.height = config.Height
	}
	if config.StartSection >= 0 {
		model.currentSection = Section(config.StartSection)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
```

Add `Interval`, `StartSection` to `DashboardConfig`:

```go
type DashboardConfig struct {
	Profile      string
	Theme        string
	Width        int
	Height       int
	Debug        bool
	Interval     time.Duration
	StartSection int
}
```

- [ ] **Step 7: Fix `refreshDataCmd` to pass DataSource**

In `update.go`, change `refreshDataCmd()`:

```go
func refreshDataCmd(ds DataSource) tea.Cmd {
	return loadDataCmd(ds)
}
```

And update all callers in `handleKeyMsg` and `handlePaletteSelect` to pass `m.data`:
- `"f5"` case: `return m, refreshDataCmd(m.data)`
- `"Refresh Data"` case: `return m, refreshDataCmd(m.data)`

- [ ] **Step 8: Run build and tests**

Run: `go build -o build/cosmoflare . && go test ./internal/tui/ -timeout 30s`
Expected: Build success, all tests pass

- [ ] **Step 9: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/dashboard.go
git commit -m "refactor(tui): wire DataSource into DashboardModel (ROAD-020 P-03)"
```

---

### Task 4: Monitoring Section with Live Polling and Deltas

**Files:**
- Modify: `internal/tui/model.go` (add message type)
- Modify: `internal/tui/update.go` (handle monitoring tick)
- Modify: `internal/tui/view.go` (render monitoring panel)

- [ ] **Step 1: Add monitoringTickMsg and fetchMetricsMsg**

In `model.go`, add after the existing message types:

```go
type monitoringTickMsg struct{}

type metricsLoadedMsg struct {
	metrics ServiceMetrics
	err     error
}
```

- [ ] **Step 2: Update Init() to start monitoring tick when data is available**

```go
func (m DashboardModel) Init() tea.Cmd {
	if !m.data.Available() {
		return nil
	}
	return tea.Batch(
		loadDataCmd(m.data),
		tea.Tick(m.pollInterval, func(_ time.Time) tea.Msg {
			return monitoringTickMsg{}
		}),
	)
}
```

- [ ] **Step 3: Handle monitoringTickMsg in Update()**

In `update.go`, add cases in the `Update` switch:

```go
case monitoringTickMsg:
	if m.pollPaused || m.skipNextPoll {
		m.skipNextPoll = false
		return m, tea.Tick(m.pollInterval, func(_ time.Time) tea.Msg {
			return monitoringTickMsg{}
		})
	}
	return m, tea.Batch(
		fetchMetricsCmd(m.data),
		tea.Tick(m.pollInterval, func(_ time.Time) tea.Msg {
			return monitoringTickMsg{}
		}),
	)

case metricsLoadedMsg:
	if msg.err != nil {
		m.addNotification("Metrics error: "+msg.err.Error(), "error")
		if isRateLimited(msg.err) {
			m.skipNextPoll = true
			m.addNotification("Rate limited — skipping next refresh", "warning")
		}
		return m, nil
	}
	m.prevMetrics = m.metrics
	m.metrics = msg.metrics
	return m, nil
```

And remove the `realTimeUpdateMsg` case entirely. Add the helper:

```go
func fetchMetricsCmd(ds DataSource) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		metrics, err := ds.FetchMetrics(ctx)
		return metricsLoadedMsg{metrics: metrics, err: err}
	}
}

func isRateLimited(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit"))
}
```

Add `"strings"` to update.go imports.

- [ ] **Step 4: Add `r` key for manual refresh and `p` for pause in monitoring**

In `handleKeyMsg`, update the `"r"` and `"p"` cases:

```go
case "r":
	if m.data.Available() {
		switch m.currentSection {
		case SectionMonitoring:
			return m, fetchMetricsCmd(m.data)
		default:
			return m, refreshDataCmd(m.data)
		}
	}
case "p":
	if m.currentSection == SectionMonitoring {
		m.pollPaused = !m.pollPaused
		if m.pollPaused {
			m.addNotification("Polling paused", "info")
		} else {
			m.addNotification("Polling resumed", "info")
		}
		return m, nil
	}
```

- [ ] **Step 5: Rewrite renderMonitoring in view.go**

Replace `renderMonitoring()` with:

```go
func (m DashboardModel) renderMonitoring() string {
	if !m.data.Available() {
		return m.renderNoCreds()
	}

	title := headerStyle.Render("📈 Live Monitoring")

	r2Block := m.renderServiceBlock("R2 Storage",
		[]string{
			fmt.Sprintf("Buckets:  %d%s", m.metrics.R2.BucketCount, m.delta(m.prevMetrics.R2.BucketCount, m.metrics.R2.BucketCount)),
			fmt.Sprintf("Objects:  %d%s", m.metrics.R2.TotalObjects, m.delta(int(m.prevMetrics.R2.TotalObjects), int(m.metrics.R2.TotalObjects))),
			fmt.Sprintf("Size:     %s", formatBytes(m.metrics.R2.TotalSize)),
		})

	workersBlock := m.renderServiceBlock("Workers",
		[]string{
			fmt.Sprintf("Scripts:  %d%s", m.metrics.Workers.Count, m.delta(m.prevMetrics.Workers.Count, m.metrics.Workers.Count)),
		})

	kvBlock := m.renderServiceBlock("KV",
		[]string{
			fmt.Sprintf("Namespaces: %d%s", m.metrics.KV.NamespaceCount, m.delta(m.prevMetrics.KV.NamespaceCount, m.metrics.KV.NamespaceCount)),
		})

	var services string
	if m.width > 100 {
		services = lipgloss.JoinHorizontal(lipgloss.Top, r2Block, "  ", workersBlock, "  ", kvBlock)
	} else {
		services = lipgloss.JoinVertical(lipgloss.Left, r2Block, workersBlock, kvBlock)
	}

	status := m.renderPollStatus()

	return lipgloss.JoinVertical(lipgloss.Left, title, "", services, "", status)
}

func (m DashboardModel) renderServiceBlock(name string, lines []string) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(name)
	body := lipgloss.NewStyle().Foreground(textColor).Render(strings.Join(lines, "\n"))
	block := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(1, 2).
		Width(30).
		Render(block)
}

func (m DashboardModel) delta(prev, curr int) string {
	diff := curr - prev
	if diff == 0 || m.prevMetrics.FetchedAt.IsZero() {
		return ""
	}
	if diff > 0 {
		return lipgloss.NewStyle().Foreground(successColor).Render(fmt.Sprintf(" ↑%d", diff))
	}
	return lipgloss.NewStyle().Foreground(errorColor).Render(fmt.Sprintf(" ↓%d", -diff))
}

func (m DashboardModel) renderPollStatus() string {
	if m.pollPaused {
		return lipgloss.NewStyle().Foreground(warningColor).Render("⏸ Polling paused  |  p = resume  |  r = manual refresh")
	}
	lastUpdate := "never"
	if !m.metrics.FetchedAt.IsZero() {
		lastUpdate = m.metrics.FetchedAt.Format("15:04:05")
	}
	return lipgloss.NewStyle().Foreground(mutedColor).Render(
		fmt.Sprintf("Last: %s  |  Interval: %s  |  r = refresh  |  p = pause", lastUpdate, m.pollInterval))
}

func (m DashboardModel) renderNoCreds() string {
	return lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true).
		Padding(2, 4).
		Render("⚠ No credentials configured — press 6 for Settings or run cosmoflare setup")
}
```

Add `formatBytes` helper if not already present (check if `utils.FormatBytes` is available — it is, imported in the existing view.go). Use `utils.FormatBytes` instead of a local `formatBytes`.

- [ ] **Step 6: Run build and tests**

Run: `go build -o build/cosmoflare . && go test ./internal/tui/ -timeout 30s`
Expected: Build success, tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/view.go
git commit -m "feat(tui): add live monitoring with auto-poll and delta indicators (ROAD-020 P-04)"
```

---

### Task 5: Object List with Pagination

**Files:**
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/view.go`

- [ ] **Step 1: Add object fetch message type**

In `model.go`, add:

```go
type objectsLoadedMsg struct {
	objects []ObjectItem
	total   int
	err     error
}
```

- [ ] **Step 2: Wire object fetching on bucket selection**

In `update.go`, update the `bucketSelectedMsg` handler:

```go
case bucketSelectedMsg:
	m.currentBucket = msg.bucket
	m.currentSection = SectionObjectList
	m.objectPage = 0
	m.objects = nil
	m.addNotification("Loading objects from "+msg.bucket.Name+"...", "info")
	return m, fetchObjectsCmd(m.data, msg.bucket.Name, 0)
```

Add the command:

```go
func fetchObjectsCmd(ds DataSource, bucket string, page int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		objects, total, err := ds.FetchObjects(ctx, bucket, page)
		return objectsLoadedMsg{objects: objects, total: total, err: err}
	}
}
```

Add the handler in `Update()`:

```go
case objectsLoadedMsg:
	if msg.err != nil {
		m.addNotification("Failed to load objects: "+msg.err.Error(), "error")
		return m, nil
	}
	m.objects = msg.objects
	m.objectTotal = msg.total
	return m, nil
```

- [ ] **Step 3: Add pagination keys and backspace navigation**

In `handleKeyMsg`, add cases:

```go
case "n":
	if m.currentSection == SectionObjectList && m.currentBucket != nil && len(m.objects) >= objectsPerPage {
		m.objectPage++
		return m, fetchObjectsCmd(m.data, m.currentBucket.Name, m.objectPage)
	}
case "backspace":
	if m.currentSection == SectionObjectList {
		m.currentSection = SectionBucketList
		m.currentBucket = nil
		m.objects = nil
		return m, nil
	}
```

Note: `p` for previous page in Object List conflicts with `p` for pause in Monitoring, but they're in different sections. The existing `handleKeyMsg` switch runs unconditionally — add section checks:

```go
case "p":
	if m.currentSection == SectionMonitoring {
		m.pollPaused = !m.pollPaused
		// ... (existing pause logic)
		return m, nil
	}
	if m.currentSection == SectionObjectList && m.objectPage > 0 {
		m.objectPage--
		return m, fetchObjectsCmd(m.data, m.currentBucket.Name, m.objectPage)
	}
```

- [ ] **Step 4: Rewrite renderObjectList**

Replace `renderObjectList()` in `view.go`:

```go
func (m DashboardModel) renderObjectList() string {
	if m.currentBucket == nil {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("No bucket selected. Navigate to Buckets (2) and press Enter.")
	}

	if !m.data.Available() {
		return m.renderNoCreds()
	}

	title := headerStyle.Render(fmt.Sprintf("📁 Objects in %s", m.currentBucket.Name))

	if m.objects == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "Loading...")
	}

	if len(m.objects) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			lipgloss.NewStyle().Foreground(mutedColor).Render("Bucket is empty."))
	}

	headers := []string{"Key", "Size", "Last Modified", "Content Type"}
	headerRow := m.renderTableRow(headers, true)

	var rows []string
	for _, obj := range m.objects {
		row := []string{
			truncate(obj.Key, 40),
			utils.FormatBytes(obj.Size),
			obj.LastModified.Format("2006-01-02 15:04"),
			obj.ContentType,
		}
		rows = append(rows, m.renderTableRow(row, false))
	}

	table := lipgloss.JoinVertical(lipgloss.Left, headerRow, strings.Join(rows, "\n"))

	pageInfo := lipgloss.NewStyle().Foreground(mutedColor).Render(
		fmt.Sprintf("Page %d  |  %d objects  |  n = next  |  p = prev  |  Backspace = back",
			m.objectPage+1, len(m.objects)))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", table, "", pageInfo)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

- [ ] **Step 5: Run build and tests**

Run: `go build -o build/cosmoflare . && go test ./internal/tui/ -timeout 30s`
Expected: Build success, tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/view.go
git commit -m "feat(tui): add object list with pagination (ROAD-020 P-05)"
```

---

### Task 6: Bucket CRUD (Create and Delete with Confirmation)

**Files:**
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/view.go`

- [ ] **Step 1: Add input mode state to model**

In `model.go`, add to `DashboardModel`:

```go
	// Input mode (for create bucket)
	inputMode   bool
	inputBuffer string
```

- [ ] **Step 2: Wire create and delete commands in handleKeyMsg**

Replace the existing `"c"` and `"d"` cases in `handleKeyMsg`:

```go
case "c":
	if m.currentSection == SectionBucketList && m.data.Available() {
		m.inputMode = true
		m.inputBuffer = ""
		return m, nil
	}
case "d":
	if m.currentSection == SectionBucketList && m.data.Available() && m.selectedRow < len(m.buckets) {
		m.confirmAction = "delete"
		m.confirmTarget = m.buckets[m.selectedRow].Name
		return m, nil
	}
case "y":
	if m.confirmAction == "delete" {
		name := m.confirmTarget
		m.confirmAction = ""
		m.confirmTarget = ""
		return m, deleteBucketAPICmd(m.data, name)
	}
```

Add at the top of `handleKeyMsg`, before the existing search mode block:

```go
// Handle input mode (create bucket)
if m.inputMode {
	switch msg.String() {
	case "enter":
		name := m.inputBuffer
		m.inputMode = false
		m.inputBuffer = ""
		if name != "" {
			return m, createBucketAPICmd(m.data, name)
		}
		return m, nil
	case "esc":
		m.inputMode = false
		m.inputBuffer = ""
		return m, nil
	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.inputBuffer += msg.String()
		}
		return m, nil
	}
}

// Handle confirmation mode
if m.confirmAction != "" {
	switch msg.String() {
	case "y":
		name := m.confirmTarget
		m.confirmAction = ""
		m.confirmTarget = ""
		return m, deleteBucketAPICmd(m.data, name)
	case "n", "esc":
		m.confirmAction = ""
		m.confirmTarget = ""
		return m, nil
	default:
		return m, nil
	}
}
```

Add the command functions and message types:

```go
type bucketCreatedMsg struct{ err error }
type bucketDeletedMsg struct{ err error }

func createBucketAPICmd(ds DataSource, name string) tea.Cmd {
	return func() tea.Msg {
		err := ds.CreateBucket(context.Background(), name)
		return bucketCreatedMsg{err: err}
	}
}

func deleteBucketAPICmd(ds DataSource, name string) tea.Cmd {
	return func() tea.Msg {
		err := ds.DeleteBucket(context.Background(), name)
		return bucketDeletedMsg{err: err}
	}
}
```

Add handlers in `Update()`:

```go
case bucketCreatedMsg:
	if msg.err != nil {
		m.addNotification("Failed to create bucket: "+msg.err.Error(), "error")
	} else {
		m.addNotification("Bucket created", "success")
	}
	return m, refreshDataCmd(m.data)

case bucketDeletedMsg:
	if msg.err != nil {
		m.addNotification("Failed to delete bucket: "+msg.err.Error(), "error")
	} else {
		m.addNotification("Bucket deleted", "success")
	}
	return m, refreshDataCmd(m.data)
```

- [ ] **Step 3: Render input prompt and confirmation in footer**

In `view.go`, update `renderFooter()` to show inline prompts:

```go
func (m DashboardModel) renderFooter() string {
	if m.inputMode {
		return lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render(fmt.Sprintf("Create bucket: %s█  (Enter = confirm, Esc = cancel)", m.inputBuffer))
	}

	if m.confirmAction == "delete" {
		return lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Render(fmt.Sprintf("Delete bucket '%s'? (y/n)", m.confirmTarget))
	}

	// Default footer with section navigation hints
	return m.renderDefaultFooter()
}
```

If `renderFooter` doesn't exist yet, add it and call it from `View()`. Check the existing code — the current `View()` already calls `m.renderFooter()`, so update the existing function body.

- [ ] **Step 4: Run build and tests**

Run: `go build -o build/cosmoflare . && go test ./internal/tui/ -timeout 30s`
Expected: Build success, tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/view.go
git commit -m "feat(tui): add bucket create/delete with confirmation (ROAD-020 P-06)"
```

---

### Task 7: Section-Entry Refresh and Credential Banner

**Files:**
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/view.go`

- [ ] **Step 1: Add section-entry refresh trigger**

In `handleKeyMsg`, when switching sections via number keys, trigger a refresh:

```go
case "1":
	prev := m.currentSection
	m.currentSection = SectionOverview
	if prev != SectionOverview && m.data.Available() {
		return m, refreshDataCmd(m.data)
	}
case "2":
	prev := m.currentSection
	m.currentSection = SectionBucketList
	if prev != SectionBucketList && m.data.Available() {
		return m, refreshDataCmd(m.data)
	}
case "5":
	prev := m.currentSection
	m.currentSection = SectionMonitoring
	if prev != SectionMonitoring && m.data.Available() {
		return m, fetchMetricsCmd(m.data)
	}
```

Do the same for `left`/`right`/`h`/`l` section cycling — after `previousSection()`/`nextSection()`, check if `m.data.Available()` and fire the appropriate refresh.

- [ ] **Step 2: Add credential banner to Overview**

In `renderOverview()`, add at the top:

```go
func (m DashboardModel) renderOverview() string {
	if !m.data.Available() {
		return m.renderNoCreds()
	}

	var content strings.Builder
	content.WriteString(m.renderUsageStats())
	content.WriteString("\n\n")
	content.WriteString(m.renderBucketTable())
	content.WriteString("\n\n")
	content.WriteString(m.renderQuickActions())
	return content.String()
}
```

- [ ] **Step 3: Update dashboard header to show "Cosmoflare" instead of "R2Go2"**

In `view.go`, update `renderHeader`:

```go
func (m DashboardModel) renderHeader() string {
	title := titleStyle.Render(fmt.Sprintf("🎯 Cosmoflare Dashboard — %s", m.getCurrentProfile()))
	// ...
}
```

- [ ] **Step 4: Run build and tests**

Run: `go build -o build/cosmoflare . && go test ./internal/tui/ -timeout 30s`
Expected: Build success, tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/tui/update.go internal/tui/view.go
git commit -m "feat(tui): add section-entry refresh and credential banner (ROAD-020 P-07)"
```

---

### Task 8: CMD Layer — Interval Flag and Metrics Absorption

**Files:**
- Modify: `cmd/dashboard.go`
- Modify: `cmd/metrics.go`

- [ ] **Step 1: Add --interval flag to dashboard command**

In `cmd/dashboard.go`:

```go
var dashboardInterval time.Duration

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch interactive TUI dashboard",
	Long: `Launch an interactive terminal dashboard for managing Cloudflare services.

Features:
  • Real-time monitoring of R2, Workers, and KV
  • Bucket management (create, delete, browse)
  • Object listing with pagination
  • Beautiful visual design with command palette

Examples:
  cosmoflare dashboard                    # Launch dashboard
  cosmoflare dashboard --interval 10s     # Poll every 10 seconds`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.RunDashboardWithInterval(dashboardInterval); err != nil {
			printError("Dashboard error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Flags().DurationVar(&dashboardInterval, "interval", 30*time.Second, "Monitoring poll interval")
}
```

Add `"time"` to imports.

- [ ] **Step 2: Replace metrics TUI with dashboard redirect**

Replace `cmd/metrics.go` `runMetrics` function:

```go
func runMetrics(cmd *cobra.Command, args []string) error {
	if JSONOutput {
		client, err := cosmoflare.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		return runMetricsJSON(client)
	}

	return tui.RunDashboardWithConfig(tui.DashboardConfig{
		Interval:     metricsInterval,
		StartSection: int(tui.SectionMonitoring),
	})
}
```

Add import for `"github.com/CosmoLabs-org/cosmoflare/internal/tui"`. Remove unused imports (`tea`, `lipgloss`, `strings`). Keep `runMetricsJSON` for `--json` mode. Remove `metricsModel`, `metricsSnapshot`, `tickMsg`, `snapshotMsg`, `errMsg`, `tickCmd`, `fetchMetrics`, `formatSize`, and all related View/Update/Init methods — they're replaced by the dashboard.

- [ ] **Step 3: Run build and full test suite**

Run: `go build -o build/cosmoflare . && go test ./cmd/ ./internal/tui/ -timeout 60s`
Expected: Build success, tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/dashboard.go cmd/metrics.go
git commit -m "feat(cmd): add --interval flag, absorb metrics into dashboard (ROAD-020 P-08)"
```

---

### Task 9: Update USAGE.md and Close Out

**Files:**
- Modify: `docs/USAGE.md`
- Modify: `CHANGELOG.md` (via `ccs changelog`)

- [ ] **Step 1: Update USAGE.md metrics section**

Find the metrics section in `docs/USAGE.md` and add a note:

```markdown
> **Note:** `cosmoflare metrics` now launches the full dashboard and navigates
> directly to the Monitoring section. Use `cosmoflare metrics --json` for
> machine-readable output without the TUI.
```

Add the dashboard section if not present:

```markdown
## Dashboard

Launch an interactive terminal dashboard for managing Cloudflare services.

### Launch dashboard
```bash
cosmoflare dashboard                    # Full dashboard
cosmoflare dashboard --interval 10s     # Custom poll interval
```

Features:
- Real-time monitoring of R2, Workers, and KV with delta indicators
- Bucket management (create, delete, browse objects)
- Tiered refresh: monitoring auto-polls, other sections refresh on navigation
- Graceful launch without credentials (banner with setup instructions)
- Command palette (Ctrl+P), keyboard navigation, dark/light themes
```

- [ ] **Step 2: Add changelog entry**

Run: `ccs changelog add "Add real-time dashboard TUI with live monitoring, bucket CRUD, and object browsing (ROAD-020)" --type added`

- [ ] **Step 3: Update roadmap**

Run: `ccs roadmap update ROAD-020 --status done`

- [ ] **Step 4: Commit**

```bash
git add docs/USAGE.md
git commit -m "docs: update USAGE.md and changelog for dashboard TUI (ROAD-020)"
```
