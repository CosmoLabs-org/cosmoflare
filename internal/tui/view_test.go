package tui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// availableTestDS is a DataSource that reports Available() == true but
// delegates all operations to the null implementation (returns empty data).
type availableTestDS struct{ nullDataSource }

func (a *availableTestDS) Available() bool { return true }

func (a *availableTestDS) FetchBuckets(_ context.Context) ([]Bucket, UsageStats, error) {
	return []Bucket{}, UsageStats{}, nil
}

func newTestModel() DashboardModel {
	m := initialModel(nil, 0)
	m.width = 120
	m.height = 40
	m.loading = false
	m.buckets = []Bucket{
		{Name: "prod-assets", Size: 1024 * 1024, ObjectCount: 50, Status: "active", CreatedAt: time.Now()},
		{Name: "backups", Size: 2048 * 1024, ObjectCount: 100, Status: "active", CreatedAt: time.Now()},
	}
	m.usageStats = UsageStats{TotalUsed: 3 * 1024 * 1024, BucketCount: 2}
	return m
}

// newAvailableTestModel returns a model with an Available data source.
func newAvailableTestModel() DashboardModel {
	m := initialModel(&availableTestDS{}, 30*time.Second)
	m.width = 120
	m.height = 40
	m.loading = false
	m.buckets = []Bucket{
		{Name: "prod-assets", Size: 1024 * 1024, ObjectCount: 50, Status: "active", CreatedAt: time.Now()},
		{Name: "backups", Size: 2048 * 1024, ObjectCount: 100, Status: "active", CreatedAt: time.Now()},
	}
	m.usageStats = UsageStats{TotalUsed: 3 * 1024 * 1024, BucketCount: 2}
	return m
}

func TestViewRendersOverview(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionOverview

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Cosmoflare")
}

func TestViewRendersBucketList(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionBucketList

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "prod-assets")
	assert.Contains(t, view, "backups")
}

func TestViewRendersObjectList(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionObjectList
	m.currentBucket = &Bucket{Name: "prod-assets"}

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestViewRendersUpload(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionUpload

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestViewRendersMonitoring(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionMonitoring

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestViewRendersSettings(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionSettings

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestViewRendersHelp(t *testing.T) {
	m := newTestModel()
	m.showHelp = true

	view := m.View()
	assert.NotEmpty(t, view)
	// Help screen should contain keyboard shortcuts
	assert.Contains(t, view, "Help")
}

func TestViewRendersUploadQueue(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionUpload
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "test.txt", Bucket: "prod", Status: "running", Progress: 50},
		{ID: "up-2", FileName: "data.csv", Bucket: "prod", Status: "completed", Progress: 100},
	}

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestRenderTableRow(t *testing.T) {
	m := newTestModel()
	row := m.renderTableRow([]string{"Col1", "Col2", "Col3"}, false)
	assert.NotEmpty(t, row)
}

func TestCalculateColumnWidths(t *testing.T) {
	m := newTestModel()
	widths := m.calculateColumnWidths(4)
	assert.Len(t, widths, 4)

	total := 0
	for _, w := range widths {
		assert.Greater(t, w, 0)
		total += w
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		expect string
	}{
		{"active", "●"},
		{"inactive", "○"},
		{"error", "✗"},
	}

	m := newTestModel()
	for _, tt := range tests {
		icon := m.getStatusIcon(tt.status)
		assert.NotEmpty(t, icon, "status %s should have an icon", tt.status)
	}
}

func TestCreateProgressBar_OverflowValues(t *testing.T) {
	m := newTestModel()

	t.Run("percentage exceeds max", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bar := m.createProgressBar(150, 100, "")
			assert.Contains(t, bar, "100.0%")
		})
	})

	t.Run("negative percentage", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bar := m.createProgressBar(-5, 100, "")
			assert.Contains(t, bar, "0.0%")
		})
	})

	t.Run("zero max", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = m.createProgressBar(50, 0, "")
		})
	})

	t.Run("negative max", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = m.createProgressBar(50, -10, "")
		})
	})

	t.Run("normal values", func(t *testing.T) {
		bar := m.createProgressBar(75, 100, "test")
		assert.NotEmpty(t, bar)
		assert.Contains(t, bar, "75.0%")
	})
}

func TestRenderUsageStats_ZeroDivision(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{TotalUsed: 1024, TotalLimit: 0, BucketCount: 2}

	assert.NotPanics(t, func() {
		stats := m.renderUsageStats()
		assert.NotEmpty(t, stats)
	})
}

// --- renderOverview with various stat combinations ---

func TestRenderOverview_ZeroValues(t *testing.T) {
	m := newAvailableTestModel()
	m.usageStats = UsageStats{TotalUsed: 0, TotalLimit: 0, BucketCount: 0}
	m.buckets = []Bucket{}

	overview := m.renderOverview()
	assert.NotEmpty(t, overview)
	assert.Contains(t, overview, "Storage Usage")
	assert.Contains(t, overview, "No buckets found")
}

func TestRenderOverview_LargeNumbers(t *testing.T) {
	m := newAvailableTestModel()
	m.usageStats = UsageStats{
		TotalUsed:   500 * 1024 * 1024 * 1024 * 1024, // 500 TB
		TotalLimit:  1024 * 1024 * 1024 * 1024 * 1024, // 1 PB
		BucketCount: 150,
	}

	overview := m.renderOverview()
	assert.NotEmpty(t, overview)
	assert.Contains(t, overview, "Storage Usage")
	assert.Contains(t, overview, "Buckets: 150")
}

func TestRenderOverview_WithPercentage(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{
		TotalUsed:   512 * 1024 * 1024 * 1024, // 512 GB = 0.5 TB
		TotalLimit:  1024 * 1024 * 1024 * 1024, // 1 TB
		BucketCount: 3,
	}

	stats := m.renderUsageStats()
	assert.NotEmpty(t, stats)
	assert.Contains(t, stats, "50.0%")
}

func TestRenderOverview_NearFullUsage(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{
		TotalUsed:   1023 * 1024 * 1024 * 1024, // 1023 GB (near 1 TB limit)
		TotalLimit:  1024 * 1024 * 1024 * 1024, // 1 TB
		BucketCount: 5,
	}

	stats := m.renderUsageStats()
	assert.Contains(t, stats, "99.9%")
}

// --- renderBucketTable with various states ---

func TestRenderBucketTable_MultipleStatuses(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{
		{Name: "active-bucket", Size: 1024, ObjectCount: 10, Status: "active"},
		{Name: "archived-bucket", Size: 2048, ObjectCount: 20, Status: "archived"},
		{Name: "disabled-bucket", Size: 512, ObjectCount: 5, Status: "disabled"},
		{Name: "unknown-bucket", Size: 256, ObjectCount: 1, Status: "unknown"},
	}
	m.currentSection = SectionBucketList

	view := m.renderBucketTable()
	assert.Contains(t, view, "active-bucket")
	assert.Contains(t, view, "archived-bucket")
	assert.Contains(t, view, "disabled-bucket")
	assert.Contains(t, view, "unknown-bucket")
}

func TestRenderBucketTable_LargeObjectCount(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{
		{Name: "big-bucket", Size: 10 * 1024 * 1024 * 1024, ObjectCount: 5000000000, Status: "active"},
	}
	m.currentSection = SectionBucketList

	view := m.renderBucketTable()
	assert.Contains(t, view, "big-bucket")
}

// --- renderUploadQueue with specific states ---

func TestRenderUploadQueue_InProgressUpload(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "large-file.bin", Status: "uploading", Progress: 45, StartTime: time.Now()},
	}

	view := m.renderUploadQueue()
	assert.Contains(t, view, "large-file.bin")
	assert.Contains(t, view, "45.0%")
}

func TestRenderUploadQueue_CompletedUpload(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "done-file.txt", Status: "completed", Progress: 100, StartTime: time.Now()},
	}

	view := m.renderUploadQueue()
	assert.Contains(t, view, "done-file.txt")
	assert.Contains(t, view, "100.0%")
}

func TestRenderUploadQueue_FailedUpload(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "failed.dat", Status: "failed", Progress: 30, StartTime: time.Now()},
	}

	view := m.renderUploadQueue()
	assert.Contains(t, view, "failed.dat")
}

func TestRenderUploadQueue_QueuedUpload(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "queued.tar.gz", Status: "queued", Progress: 0},
	}

	view := m.renderUploadQueue()
	assert.Contains(t, view, "queued.tar.gz")
}

func TestRenderUploadQueue_MultipleTasks(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "uploading.txt", Status: "uploading", Progress: 75},
		{ID: "up-2", FileName: "completed.txt", Status: "completed", Progress: 100},
		{ID: "up-3", FileName: "failed.txt", Status: "failed", Progress: 50},
		{ID: "up-4", FileName: "queued.txt", Status: "queued", Progress: 0},
	}

	view := m.renderUploadQueue()
	assert.Contains(t, view, "uploading.txt")
	assert.Contains(t, view, "completed.txt")
	assert.Contains(t, view, "failed.txt")
	assert.Contains(t, view, "queued.txt")
}

// --- renderMonitoring with various metric states ---

func TestRenderMonitoring_ZeroMetrics(t *testing.T) {
	m := newAvailableTestModel()
	m.metrics = ServiceMetrics{FetchedAt: time.Now()}
	view := m.renderMonitoring()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "R2 Storage")
}

func TestRenderMonitoring_WithMetrics(t *testing.T) {
	m := newAvailableTestModel()
	m.metrics = ServiceMetrics{
		R2:        R2Metrics{BucketCount: 5, TotalSize: 1024 * 1024, TotalObjects: 9999},
		Workers:   WorkersMetrics{Count: 12},
		KV:        KVMetrics{NamespaceCount: 3},
		FetchedAt: time.Now(),
	}
	view := m.renderMonitoring()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "R2 Storage")
	assert.Contains(t, view, "Workers")
	assert.Contains(t, view, "KV")
}

func TestRenderMonitoring_NoCreds(t *testing.T) {
	m := newTestModel()
	// nullDataSource is not Available
	view := m.renderMonitoring()
	assert.Contains(t, view, "No credentials configured")
}
