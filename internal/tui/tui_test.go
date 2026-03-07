package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSectionString(t *testing.T) {
	tests := []struct {
		section  Section
		expected string
	}{
		{SectionOverview, "Overview"},
		{SectionBucketList, "Buckets"},
		{SectionObjectList, "Objects"},
		{SectionUpload, "Upload"},
		{SectionMonitoring, "Monitoring"},
		{SectionSettings, "Settings"},
		{SectionHelp, "Help"},
		{Section(99), "Unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.section.String())
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()
	assert.Equal(t, SectionOverview, m.currentSection)
	assert.Equal(t, 0, m.selectedRow)
	assert.True(t, m.loading)
	assert.Equal(t, "default", m.currentProfile)
	assert.NotNil(t, m.buckets)
	assert.Empty(t, m.buckets)
}

func TestInitializeStyles(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		InitializeStyles(darkTheme)
	})
	assert.NotPanics(t, func() {
		InitializeStyles(lightTheme)
	})
}

func TestDashboardModelInit(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestDashboardModelUpdateWindowSize(t *testing.T) {
	m := initialModel()
	updatedModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Nil(t, cmd)

	dm := updatedModel.(DashboardModel)
	assert.Equal(t, 120, dm.width)
	assert.Equal(t, 40, dm.height)
}

func TestDashboardModelUpdateDataLoaded(t *testing.T) {
	m := initialModel()

	buckets := []Bucket{
		{Name: "test-bucket", Size: 1024, ObjectCount: 10, Status: "active"},
	}
	stats := UsageStats{TotalUsed: 1024, BucketCount: 1}

	updatedModel, _ := m.Update(dataLoadedMsg{buckets: buckets, stats: stats})
	dm := updatedModel.(DashboardModel)
	assert.False(t, dm.loading)
	assert.Len(t, dm.buckets, 1)
	assert.Equal(t, "test-bucket", dm.buckets[0].Name)
}

func TestDashboardModelUpdateDataLoadError(t *testing.T) {
	m := initialModel()

	updatedModel, _ := m.Update(dataLoadedMsg{err: assert.AnError})
	dm := updatedModel.(DashboardModel)
	assert.False(t, dm.loading)
	assert.Len(t, dm.notifications, 1)
	assert.Equal(t, "error", dm.notifications[0].Type)
}

func TestDashboardModelUpdateError(t *testing.T) {
	m := initialModel()

	updatedModel, _ := m.Update(errorMsg{err: assert.AnError})
	dm := updatedModel.(DashboardModel)
	assert.False(t, dm.loading)
}

func TestDashboardModelUpdateRealTimeStats(t *testing.T) {
	m := initialModel()

	now := time.Now()
	updatedModel, cmd := m.Update(realTimeUpdateMsg{timestamp: now})
	assert.NotNil(t, cmd) // Returns a tick command

	dm := updatedModel.(DashboardModel)
	assert.Equal(t, now, dm.realTimeStats.LastUpdate)
	assert.Greater(t, dm.realTimeStats.UploadRate, 0.0)
}

func TestDashboardModelUpdateBucketSelected(t *testing.T) {
	m := initialModel()

	bucket := &Bucket{Name: "selected", Size: 2048}
	updatedModel, _ := m.Update(bucketSelectedMsg{bucket: bucket})
	dm := updatedModel.(DashboardModel)
	assert.Equal(t, SectionObjectList, dm.currentSection)
	assert.Equal(t, "selected", dm.currentBucket.Name)
}

func TestDashboardModelUpdateUploadProgress(t *testing.T) {
	m := initialModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "test.txt", Status: "running", Progress: 0},
	}

	t.Run("progress update", func(t *testing.T) {
		updatedModel, _ := m.Update(uploadProgressMsg{taskID: "up-1", progress: 50})
		dm := updatedModel.(DashboardModel)
		assert.Equal(t, 50, dm.uploadQueue[0].Progress)
	})

	t.Run("completion", func(t *testing.T) {
		updatedModel, _ := m.Update(uploadProgressMsg{taskID: "up-1", progress: 100})
		dm := updatedModel.(DashboardModel)
		assert.Equal(t, "completed", dm.uploadQueue[0].Status)
	})

	t.Run("error", func(t *testing.T) {
		updatedModel, _ := m.Update(uploadProgressMsg{taskID: "up-1", err: assert.AnError})
		dm := updatedModel.(DashboardModel)
		assert.Equal(t, "failed", dm.uploadQueue[0].Status)
	})
}

func TestDashboardModelView(t *testing.T) {
	m := initialModel()
	m.width = 80
	m.height = 24
	m.loading = false

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "R2Go2")
}

func TestDashboardModelViewLoading(t *testing.T) {
	m := initialModel()
	m.width = 80
	m.height = 24
	m.loading = true

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestBucketStruct(t *testing.T) {
	b := Bucket{
		Name:        "my-bucket",
		Size:        1024 * 1024,
		ObjectCount: 100,
		Status:      "active",
		CreatedAt:   time.Now(),
	}
	assert.Equal(t, "my-bucket", b.Name)
	assert.Equal(t, int64(100), b.ObjectCount)
}

func TestNotification(t *testing.T) {
	n := Notification{
		Message:   "Test notification",
		Type:      "info",
		Timestamp: time.Now(),
	}
	assert.Equal(t, "info", n.Type)
}

func TestSortField(t *testing.T) {
	assert.Equal(t, SortField("name"), SortByName)
	assert.Equal(t, SortField("size"), SortBySize)
	assert.Equal(t, SortField("count"), SortByCount)
	assert.Equal(t, SortField("date"), SortByDate)
}
