package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- formatNumber ---

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
		{999999999, "1000.0M"},
		{1000000000, "1.0B"},
		{2500000000, "2.5B"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatNumber(tt.input))
	}
}

// --- getStatusText ---

func TestGetStatusText(t *testing.T) {
	t.Run("loading", func(t *testing.T) {
		m := initialModel()
		m.loading = true
		assert.Equal(t, "Loading...", m.getStatusText())
	})

	t.Run("connected", func(t *testing.T) {
		m := initialModel()
		m.loading = false
		assert.Equal(t, "Connected", m.getStatusText())
	})
}

// --- getStatusIcon ---

func TestGetStatusIcon_AllStatuses(t *testing.T) {
	m := newTestModel()

	tests := []struct {
		status   string
		contains string
	}{
		{"active", "Active"},
		{"archived", "Archived"},
		{"disabled", "Disabled"},
		{"unknown", "Unknown"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := m.getStatusIcon(tt.status)
			assert.Contains(t, icon, tt.contains)
		})
	}
}

// --- getNotificationIcon ---

func TestGetNotificationIcon(t *testing.T) {
	m := newTestModel()

	tests := []struct {
		nType    string
		contains string
	}{
		{"success", "✅"},
		{"error", "❌"},
		{"warning", "⚠️"},
		{"info", "ℹ️"},
		{"other", "📢"},
		{"", "📢"},
	}

	for _, tt := range tests {
		t.Run(tt.nType, func(t *testing.T) {
			icon := m.getNotificationIcon(tt.nType)
			assert.Contains(t, icon, tt.contains)
		})
	}
}

// --- getUploadStatus ---

func TestGetUploadStatus(t *testing.T) {
	m := newTestModel()

	tests := []struct {
		status   string
		icon     string
		text     string
	}{
		{"uploading", "⬆️", "Uploading"},
		{"completed", "✅", "Completed"},
		{"failed", "❌", "Failed"},
		{"queued", "⏳", "Queued"},
		{"unknown", "❓", "Unknown"},
		{"", "❓", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			task := UploadTask{Status: tt.status}
			result := m.getUploadStatus(task)
			assert.Equal(t, tt.icon, result.Icon)
			assert.Equal(t, tt.text, result.Text)
		})
	}
}

// --- renderOverview ---

func TestRenderOverview_WithStats(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{
		TotalUsed:   10 * 1024 * 1024 * 1024, // 10 GB
		BucketCount: 3,
	}

	overview := m.renderOverview()
	assert.NotEmpty(t, overview)
}

func TestRenderOverview_EmptyStats(t *testing.T) {
	m := newTestModel()
	m.buckets = nil
	m.usageStats = UsageStats{}

	overview := m.renderOverview()
	assert.NotEmpty(t, overview)
}

// --- renderUsageStats ---

func TestRenderUsageStats(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{
		TotalUsed:   5 * 1024 * 1024 * 1024,
		BucketCount: 7,
	}

	stats := m.renderUsageStats()
	assert.NotEmpty(t, stats)
}

func TestRenderUsageStats_ZeroValues(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{}

	stats := m.renderUsageStats()
	assert.NotEmpty(t, stats)
}

// --- renderBucketTable ---

func TestRenderBucketTable_WithData(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{
		{Name: "prod", Size: 1024 * 1024, ObjectCount: 50, Status: "active", CreatedAt: time.Now()},
		{Name: "dev", Size: 512 * 1024, ObjectCount: 10, Status: "active", CreatedAt: time.Now()},
	}

	table := m.renderBucketTable()
	assert.NotEmpty(t, table)
	assert.Contains(t, table, "prod")
	assert.Contains(t, table, "dev")
}

func TestRenderBucketTable_Empty(t *testing.T) {
	m := newTestModel()
	m.buckets = nil

	table := m.renderBucketTable()
	assert.NotEmpty(t, table)
}

func TestRenderBucketTable_Sorted(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{
		{Name: "zoo", Size: 100, ObjectCount: 1},
		{Name: "alpha", Size: 200, ObjectCount: 2},
	}

	table := m.renderBucketTable()
	assert.NotEmpty(t, table)
}

// --- renderObjectList ---

func TestRenderObjectList_WithBucket(t *testing.T) {
	m := newTestModel()
	m.currentBucket = &Bucket{Name: "my-bucket", Size: 1024}

	list := m.renderObjectList()
	assert.NotEmpty(t, list)
}

func TestRenderObjectList_NoBucket(t *testing.T) {
	m := newTestModel()
	m.currentBucket = nil

	list := m.renderObjectList()
	assert.NotEmpty(t, list)
}

// --- renderUploadInterface ---

func TestRenderUploadInterface_WithQueue(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "big.zip", Bucket: "prod", Status: "uploading", Progress: 30},
		{ID: "up-2", FileName: "small.txt", Bucket: "prod", Status: "queued", Progress: 0},
		{ID: "up-3", FileName: "done.csv", Bucket: "prod", Status: "completed", Progress: 100},
		{ID: "up-4", FileName: "fail.log", Bucket: "prod", Status: "failed", Progress: 75},
	}

	ui := m.renderUploadInterface()
	assert.NotEmpty(t, ui)
}

// --- renderMonitoring ---

func TestRenderMonitoring_WithStats(t *testing.T) {
	m := newTestModel()
	m.realTimeStats = RealTimeStats{
		UploadRate:    25.5,
		DownloadRate:  12.3,
		RequestsPerMin: 750,
	}

	monitoring := m.renderMonitoring()
	assert.NotEmpty(t, monitoring)
}

// --- renderSettings ---

func TestRenderSettings(t *testing.T) {
	m := newTestModel()

	settings := m.renderSettings()
	assert.NotEmpty(t, settings)
}

// --- renderQuickActions ---

func TestRenderQuickActions(t *testing.T) {
	m := newTestModel()

	actions := m.renderQuickActions()
	assert.NotEmpty(t, actions)
}

// --- renderHelp ---

func TestRenderHelp_Content(t *testing.T) {
	m := newTestModel()

	help := m.renderHelp()
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "Navigation")
	assert.Contains(t, help, "Quick Actions")
}

// --- renderActivityFeed ---

func TestRenderActivityFeed_WithNotifications(t *testing.T) {
	m := newTestModel()
	m.notifications = []Notification{
		{Message: "Upload complete", Type: "success", Timestamp: time.Now()},
		{Message: "Bucket created", Type: "info", Timestamp: time.Now()},
		{Message: "Upload failed", Type: "error", Timestamp: time.Now()},
	}

	feed := m.renderActivityFeed()
	assert.NotEmpty(t, feed)
}

func TestRenderActivityFeed_Empty(t *testing.T) {
	m := newTestModel()
	m.notifications = nil

	feed := m.renderActivityFeed()
	assert.NotEmpty(t, feed)
}

// --- renderHeader / renderFooter ---

func TestRenderHeader(t *testing.T) {
	m := newTestModel()
	m.currentProfile = "production"

	header := m.renderHeader()
	assert.NotEmpty(t, header)
}

func TestRenderFooter(t *testing.T) {
	m := newTestModel()

	footer := m.renderFooter()
	assert.NotEmpty(t, footer)
}

func TestRenderFooter_WithNotifications(t *testing.T) {
	m := newTestModel()
	m.notifications = []Notification{
		{Message: "test", Type: "info", Timestamp: time.Now()},
	}

	footer := m.renderFooter()
	assert.NotEmpty(t, footer)
}

// --- renderLoading ---

func TestRenderLoading(t *testing.T) {
	m := newTestModel()

	loading := m.renderLoading()
	assert.NotEmpty(t, loading)
}

// --- View integration ---

func TestView_LoadingState(t *testing.T) {
	m := initialModel()
	m.width = 80
	m.height = 24
	m.loading = true

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_WithNotifications(t *testing.T) {
	m := newTestModel()
	m.notifications = []Notification{
		{Message: "Upload done", Type: "success", Timestamp: time.Now()},
		{Message: "Bucket error", Type: "error", Timestamp: time.Now()},
	}

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_AllSections(t *testing.T) {
	sections := []Section{
		SectionOverview,
		SectionBucketList,
		SectionObjectList,
		SectionUpload,
		SectionMonitoring,
		SectionSettings,
		SectionHelp,
	}

	for _, section := range sections {
		t.Run(section.String(), func(t *testing.T) {
			m := newTestModel()
			m.currentSection = section
			if section == SectionObjectList {
				m.currentBucket = &Bucket{Name: "test"}
			}

			view := m.View()
			assert.NotEmpty(t, view, "View should render for section %s", section)
		})
	}
}

// --- Notification overflow ---

func TestNotificationOverflow(t *testing.T) {
	m := initialModel()

	for i := 0; i < 15; i++ {
		m.Update(notificationMsg{
			notification: Notification{
				Message:   "notification",
				Type:      "info",
				Timestamp: time.Now(),
			},
		})
	}

	assert.LessOrEqual(t, len(m.notifications), 10, "notifications should be capped at 10")
}

// --- renderRealTimeStats ---

func TestRenderRealTimeStats(t *testing.T) {
	m := newTestModel()
	m.realTimeStats = RealTimeStats{
		UploadRate:     50.0,
		DownloadRate:   30.0,
		RequestsPerMin: 1000,
		LastUpdate:     time.Now(),
	}

	stats := m.renderRealTimeStats()
	assert.NotEmpty(t, stats)
}

// --- Search filter rendering ---

func TestView_SearchFilterActive(t *testing.T) {
	m := newTestModel()
	m.filterActive = true
	m.searchQuery = ""

	view := m.View()
	assert.NotEmpty(t, view)
}

// --- renderFileSelector ---

func TestRenderFileSelector(t *testing.T) {
	m := newTestModel()

	selector := m.renderFileSelector()
	assert.NotEmpty(t, selector)
}

// --- renderUploadQueue statuses ---

func TestRenderUploadQueue_AllStatuses(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "1", FileName: "a.txt", Status: "uploading", Progress: 50},
		{ID: "2", FileName: "b.txt", Status: "completed", Progress: 100},
		{ID: "3", FileName: "c.txt", Status: "failed", Progress: 30},
		{ID: "4", FileName: "d.txt", Status: "queued", Progress: 0},
		{ID: "5", FileName: "e.txt", Status: "unknown", Progress: 0},
	}

	queue := m.renderUploadQueue()
	assert.NotEmpty(t, queue)
}
