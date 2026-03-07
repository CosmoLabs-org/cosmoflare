package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestModel() DashboardModel {
	m := initialModel()
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
	assert.Contains(t, view, "R2Go2")
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
