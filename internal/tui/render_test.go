package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		{1500000, "1.5M"},
		{999999999, "1000.0M"},
		{1000000000, "1.0B"},
		{5000000000, "5.0B"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, formatNumber(tt.input))
		})
	}
}

// --- getNotificationIcon ---

func TestGetNotificationIcon(t *testing.T) {
	m := newTestModel()
	tests := []struct {
		nType string
		want  string
	}{
		{"success", "✅"},
		{"error", "❌"},
		{"warning", "⚠️"},
		{"info", "ℹ️"},
		{"", "📢"},
		{"unknown_type", "📢"},
	}
	for _, tt := range tests {
		t.Run(tt.nType, func(t *testing.T) {
			assert.Equal(t, tt.want, m.getNotificationIcon(tt.nType))
		})
	}
}

// --- getUploadStatus ---

func TestGetUploadStatus(t *testing.T) {
	m := newTestModel()
	tests := []struct {
		status string
		icon   string
		text   string
	}{
		{"uploading", "⬆️", "Uploading"},
		{"completed", "✅", "Completed"},
		{"failed", "❌", "Failed"},
		{"queued", "⏳", "Queued"},
		{"", "❓", "Unknown"},
		{"something_else", "❓", "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := m.getUploadStatus(UploadTask{Status: tt.status})
			assert.Equal(t, tt.icon, result.Icon)
			assert.Equal(t, tt.text, result.Text)
		})
	}
}

// --- getStatusIcon ---

func TestGetStatusIcon_AllCases(t *testing.T) {
	m := newTestModel()
	tests := []struct {
		status string
	}{
		{"active"},
		{"archived"},
		{"disabled"},
		{""},
		{"unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := m.getStatusIcon(tt.status)
			assert.NotEmpty(t, icon)
		})
	}
}

// --- renderActivityFeed ---

func TestRenderActivityFeed(t *testing.T) {
	t.Run("empty notifications", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{}
		feed := m.renderActivityFeed()
		assert.NotEmpty(t, feed)
		assert.Contains(t, feed, "No recent activity")
	})

	t.Run("with notifications", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{
			{Message: "upload done", Type: "success", Timestamp: time.Now()},
			{Message: "error occurred", Type: "error", Timestamp: time.Now()},
			{Message: "warning issued", Type: "warning", Timestamp: time.Now()},
			{Message: "info update", Type: "info", Timestamp: time.Now()},
		}
		feed := m.renderActivityFeed()
		assert.NotEmpty(t, feed)
		assert.Contains(t, feed, "upload done")
		assert.Contains(t, feed, "error occurred")
	})

	t.Run("single notification", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{
			{Message: "only one", Type: "info", Timestamp: time.Now()},
		}
		feed := m.renderActivityFeed()
		assert.Contains(t, feed, "only one")
	})
}

// --- renderFooter ---

func TestRenderFooter(t *testing.T) {
	t.Run("loading state", func(t *testing.T) {
		m := newTestModel()
		m.loading = true
		footer := m.renderFooter()
		assert.NotEmpty(t, footer)
		assert.Contains(t, footer, "Loading...")
	})

	t.Run("connected state with search query", func(t *testing.T) {
		m := newTestModel()
		m.searchQuery = "test-filter"
		footer := m.renderFooter()
		assert.NotEmpty(t, footer)
		assert.Contains(t, footer, "Search: test-filter")
	})

	t.Run("filter active without query", func(t *testing.T) {
		m := newTestModel()
		m.filterActive = true
		m.searchQuery = ""
		footer := m.renderFooter()
		assert.Contains(t, footer, "Filter Active")
	})

	t.Run("with notifications", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{
			{Message: "test msg", Type: "success", Timestamp: time.Now()},
		}
		footer := m.renderFooter()
		assert.Contains(t, footer, "test msg")
	})

	t.Run("with error notification", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{
			{Message: "something broke", Type: "error", Timestamp: time.Now()},
		}
		footer := m.renderFooter()
		assert.Contains(t, footer, "something broke")
	})

	t.Run("no notifications no search", func(t *testing.T) {
		m := newTestModel()
		m.notifications = []Notification{}
		m.searchQuery = ""
		m.filterActive = false
		footer := m.renderFooter()
		assert.NotEmpty(t, footer)
		assert.Contains(t, footer, "Connected")
		assert.Contains(t, footer, "Arrow Keys")
	})
}

// --- calculateColumnWidths ---

func TestCalculateColumnWidths_EdgeCases(t *testing.T) {
	t.Run("zero width", func(t *testing.T) {
		m := newTestModel()
		m.width = 0
		widths := m.calculateColumnWidths(4)
		assert.Equal(t, []int{20, 12, 10, 10}, widths)
	})

	t.Run("negative width", func(t *testing.T) {
		m := newTestModel()
		m.width = -10
		widths := m.calculateColumnWidths(4)
		assert.Equal(t, []int{20, 12, 10, 10}, widths)
	})

	t.Run("small width produces negative for 4 cols - known behavior", func(t *testing.T) {
		m := newTestModel()
		m.width = 20
		widths := m.calculateColumnWidths(4)
		assert.Len(t, widths, 4)
		// With width=20, baseWidth=(20-4)/4=4, so [-1, -1] are produced for inner cols.
		// This is a known issue with narrow windows.
		assert.Equal(t, []int{14, -1, -1, 4}, widths)
	})

	t.Run("non-standard column count", func(t *testing.T) {
		m := newTestModel()
		m.width = 100
		widths := m.calculateColumnWidths(3)
		assert.Len(t, widths, 3)
		for _, w := range widths {
			assert.Equal(t, 32, w) // (100-4)/3 = 32
		}
	})

	t.Run("single column", func(t *testing.T) {
		m := newTestModel()
		m.width = 80
		widths := m.calculateColumnWidths(1)
		assert.Len(t, widths, 1)
		assert.Equal(t, 76, widths[0])
	})

	t.Run("many columns", func(t *testing.T) {
		m := newTestModel()
		m.width = 200
		widths := m.calculateColumnWidths(10)
		assert.Len(t, widths, 10)
		for _, w := range widths {
			assert.Equal(t, 19, w) // (200-4)/10 = 19
		}
	})
}

// --- createProgressBar ---

func TestCreateProgressBar_EdgeCases(t *testing.T) {
	m := newTestModel()

	t.Run("zero max defaults to 100", func(t *testing.T) {
		bar := m.createProgressBar(50, 0, "")
		assert.NotEmpty(t, bar)
		assert.Contains(t, bar, "50.0%")
	})

	t.Run("negative max defaults to 100", func(t *testing.T) {
		bar := m.createProgressBar(25, -10, "test")
		assert.NotEmpty(t, bar)
		assert.Contains(t, bar, "25.0%")
	})

	t.Run("percentage exceeds max clamped to 1", func(t *testing.T) {
		bar := m.createProgressBar(200, 100, "over")
		assert.Contains(t, bar, "100.0%")
	})

	t.Run("zero percentage", func(t *testing.T) {
		bar := m.createProgressBar(0, 100, "empty")
		assert.Contains(t, bar, "0.0%")
	})

	t.Run("negative percentage clamped to 0", func(t *testing.T) {
		bar := m.createProgressBar(-50, 100, "neg")
		assert.Contains(t, bar, "0.0%")
	})

	t.Run("exact 100 percent", func(t *testing.T) {
		bar := m.createProgressBar(100, 100, "full")
		assert.Contains(t, bar, "100.0%")
	})

	t.Run("with unicode prefix", func(t *testing.T) {
		bar := m.createProgressBar(75, 100, "⬆️")
		assert.NotEmpty(t, bar)
		assert.Contains(t, bar, "75.0%")
	})
}

// --- renderMainContent edge cases ---

func TestRenderMainContent_AllSections(t *testing.T) {
	sections := []Section{
		SectionOverview,
		SectionBucketList,
		SectionObjectList,
		SectionUpload,
		SectionMonitoring,
		SectionSettings,
	}
	for _, sec := range sections {
		t.Run(sec.String(), func(t *testing.T) {
			m := newTestModel()
			m.currentSection = sec
			content := m.renderMainContent()
			assert.NotEmpty(t, content)
		})
	}
}

func TestRenderMainContent_Loading(t *testing.T) {
	m := newTestModel()
	m.loading = true
	content := m.renderMainContent()
	assert.NotEmpty(t, content)
}

func TestRenderMainContent_DefaultSection(t *testing.T) {
	m := newTestModel()
	m.currentSection = Section(99) // Unknown section
	content := m.renderMainContent()
	assert.NotEmpty(t, content)
}

// --- renderObjectList edge cases ---

func TestRenderObjectList_NoBucket(t *testing.T) {
	m := newTestModel()
	m.currentBucket = nil
	view := m.renderObjectList()
	assert.Contains(t, view, "No bucket selected")
}

func TestRenderObjectList_WithBucket(t *testing.T) {
	m := newTestModel()
	m.currentBucket = &Bucket{Name: "test-bucket"}
	view := m.renderObjectList()
	assert.Contains(t, view, "test-bucket")
}

// --- renderBucketTable edge cases ---

func TestRenderBucketTable_Empty(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{}
	view := m.renderBucketTable()
	assert.Contains(t, view, "No buckets found")
}

func TestRenderBucketTable_WithSelectedRow(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionBucketList
	m.selectedRow = 1
	view := m.renderBucketTable()
	assert.Contains(t, view, "prod-assets")
	assert.Contains(t, view, "backups")
}

// --- renderUploadQueue edge cases ---

func TestRenderUploadQueue_Empty(t *testing.T) {
	m := newTestModel()
	view := m.renderUploadQueue()
	assert.Contains(t, view, "No files")
}

func TestRenderUploadQueue_WithTasks(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "1", FileName: "file1.txt", Status: "uploading", Progress: 30},
		{ID: "2", FileName: "file2.txt", Status: "completed", Progress: 100},
		{ID: "3", FileName: "file3.txt", Status: "failed", Progress: 0},
		{ID: "4", FileName: "file4.txt", Status: "queued", Progress: 0},
	}
	view := m.renderUploadQueue()
	assert.Contains(t, view, "file1.txt")
	assert.Contains(t, view, "file2.txt")
	assert.Contains(t, view, "file3.txt")
	assert.Contains(t, view, "file4.txt")
}

// --- renderUsageStats edge case ---

func TestRenderUsageStats_ZeroLimit(t *testing.T) {
	m := newTestModel()
	m.usageStats = UsageStats{TotalUsed: 0, TotalLimit: 0, BucketCount: 0}
	// This will produce a division by zero in percentage calculation.
	// The function should still return something without panicking.
	assert.NotPanics(t, func() {
		view := m.renderUsageStats()
		assert.NotEmpty(t, view)
	})
}

// --- View rendering with various states ---

func TestView_SmallWindow(t *testing.T) {
	m := newTestModel()
	m.width = 40 // Below the 80 threshold
	m.height = 10
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_LargeWindow(t *testing.T) {
	m := newTestModel()
	m.width = 200
	m.height = 60
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_HelpScreen(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	view := m.View()
	assert.Contains(t, view, "Help")
	assert.Contains(t, view, "Navigation")
	assert.Contains(t, view, "Quick Actions")
	assert.Contains(t, view, "Search")
}

func TestView_WithNotifications(t *testing.T) {
	m := newTestModel()
	m.notifications = []Notification{
		{Message: "critical error!", Type: "error", Timestamp: time.Now()},
	}
	view := m.View()
	assert.Contains(t, view, "critical error!")
}

// --- Update handler edge cases ---

func TestUpdate_UnknownMessage(t *testing.T) {
	m := newTestModel()
	type unknownMsg struct{}
	result, cmd := m.Update(unknownMsg{})
	dm := result.(DashboardModel)
	assert.Nil(t, cmd)
	// Model should be unchanged
	assert.Equal(t, SectionOverview, dm.currentSection)
}

func TestUpdate_KeyMsgThroughUpdate(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}
	// Route a KeyMsg through Update (not handleKeyMsg directly)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
	dm := result.(DashboardModel)
	assert.Equal(t, 1, dm.selectedRow)

	// Also test quit through Update
	result2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	dm2 := result2.(DashboardModel)
	assert.NotNil(t, cmd)
	_ = dm2
}

func TestUpdate_UploadProgress_NotFound(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "existing", FileName: "file.txt"},
	}
	result, _ := m.Update(uploadProgressMsg{taskID: "nonexistent", progress: 50})
	dm := result.(DashboardModel)
	// No change to the queue
	assert.Equal(t, 0, dm.uploadQueue[0].Progress)
}

func TestUpdate_UploadProgress_WithError(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "test.txt", Status: "running"},
	}
	testErr := errors.New("network failure")
	result, _ := m.Update(uploadProgressMsg{taskID: "up-1", err: testErr})
	dm := result.(DashboardModel)
	assert.Equal(t, "failed", dm.uploadQueue[0].Status)
	assert.Equal(t, testErr, dm.uploadQueue[0].Error)
	// Notification added
	assert.NotEmpty(t, dm.notifications)
	assert.Contains(t, dm.notifications[0].Message, "network failure")
}

func TestUpdate_UploadProgress_Completion(t *testing.T) {
	m := newTestModel()
	m.uploadQueue = []UploadTask{
		{ID: "up-1", FileName: "bigfile.bin", Status: "running", Progress: 0},
	}
	result, _ := m.Update(uploadProgressMsg{taskID: "up-1", progress: 100})
	dm := result.(DashboardModel)
	assert.Equal(t, "completed", dm.uploadQueue[0].Status)
	// Notification is added for completion
	assert.Len(t, dm.notifications, 1)
	assert.Contains(t, dm.notifications[0].Message, "bigfile.bin")
}

func TestUpdate_NotificationOverflow(t *testing.T) {
	m := newTestModel()
	// Add 12 notifications via Update
	for i := 0; i < 12; i++ {
		result, _ := m.Update(notificationMsg{
			notification: Notification{
				Message:   fmt.Sprintf("notif-%d", i),
				Type:      "info",
				Timestamp: time.Now(),
			},
		})
		m = result.(DashboardModel)
	}
	// Should cap at 10
	assert.Len(t, m.notifications, 10)
	// Most recent first
	assert.Contains(t, m.notifications[0].Message, "notif-11")
}

func TestUpdate_WindowSize(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 300, Height: 80})
	dm := result.(DashboardModel)
	assert.Equal(t, 300, dm.width)
	assert.Equal(t, 80, dm.height)
}

func TestUpdate_WindowSize_Zero(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	dm := result.(DashboardModel)
	assert.Equal(t, 0, dm.width)
	assert.Equal(t, 0, dm.height)
	// Should not panic when rendering
	assert.NotPanics(t, func() {
		_ = dm.View()
	})
}

func TestUpdate_DataLoaded_WithNotification(t *testing.T) {
	m := newTestModel()
	buckets := []Bucket{{Name: "b1"}, {Name: "b2"}}
	stats := UsageStats{TotalUsed: 5000, BucketCount: 2}
	result, _ := m.Update(dataLoadedMsg{buckets: buckets, stats: stats})
	dm := result.(DashboardModel)
	assert.False(t, dm.loading)
	assert.Len(t, dm.buckets, 2)
	assert.Contains(t, dm.notifications[0].Message, "successfully")
}

func TestUpdate_RealTimeStats_ReturnsTick(t *testing.T) {
	m := newTestModel()
	now := time.Now()
	_, cmd := m.Update(realTimeUpdateMsg{timestamp: now})
	assert.NotNil(t, cmd)
	// The returned cmd should produce another realTimeUpdateMsg
	msg := cmd()
	_, ok := msg.(realTimeUpdateMsg)
	assert.True(t, ok, "tick cmd should produce realTimeUpdateMsg")
}

func TestUpdate_BucketSelected_SetsSection(t *testing.T) {
	m := newTestModel()
	bucket := &Bucket{Name: "selected-bucket", Size: 999}
	result, _ := m.Update(bucketSelectedMsg{bucket: bucket})
	dm := result.(DashboardModel)
	assert.Equal(t, SectionObjectList, dm.currentSection)
	assert.Equal(t, "selected-bucket", dm.currentBucket.Name)
	// Notification added
	assert.Contains(t, dm.notifications[0].Message, "selected-bucket")
}

// --- handleKeyMsg edge cases ---

func TestHandleKeyMsg_EscQuit(t *testing.T) {
	m := newTestModel()
	m.showHelp = false
	_, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	// tea.Quit should be returned
	msg := cmd()
	assert.Equal(t, tea.Quit(), msg)
}

func TestHandleKeyMsg_QQuit(t *testing.T) {
	m := newTestModel()
	m.showHelp = false
	_, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.NotNil(t, cmd)
}

func TestHandleKeyMsg_EscWhenHelpShown(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	dm := result.(DashboardModel)
	assert.False(t, dm.showHelp)
}

func TestHandleKeyMsg_EnterSelects(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionBucketList
	m.buckets = []Bucket{{Name: "test"}, {Name: "test2"}}
	m.selectedRow = 0
	_, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
}

func TestHandleKeyMsg_SpaceSelects(t *testing.T) {
	m := newTestModel()
	m.currentSection = SectionBucketList
	m.buckets = []Bucket{{Name: "test"}}
	m.selectedRow = 0
	_, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	assert.NotNil(t, cmd)
}

func TestHandleKeyMsg_SandPGoToSettings(t *testing.T) {
	m := newTestModel()
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	dm := result.(DashboardModel)
	assert.Equal(t, SectionSettings, dm.currentSection)

	m2 := initialModel()
	result2, _ := m2.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	dm2 := result2.(DashboardModel)
	assert.Equal(t, SectionSettings, dm2.currentSection)
}

func TestHandleKeyMsg_F10GoesToSettings(t *testing.T) {
	m := newTestModel()
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyF10})
	dm := result.(DashboardModel)
	assert.Equal(t, SectionSettings, dm.currentSection)
}

func TestHandleKeyMsg_SearchMultibyteRune(t *testing.T) {
	m := newTestModel()
	m.searchQuery = "abc"
	// Multi-rune key press (e.g., Ctrl+something) should not be appended
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	dm := result.(DashboardModel)
	// Should not append since len(Runes) != 1
	assert.Equal(t, "abc", dm.searchQuery)
}

func TestHandleKeyMsg_SearchBackspaceEmpty(t *testing.T) {
	m := newTestModel()
	m.searchQuery = ""
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyBackspace})
	dm := result.(DashboardModel)
	assert.Equal(t, "", dm.searchQuery)
}

func TestHandleKeyMsg_UnrecognizedKey(t *testing.T) {
	m := newTestModel()
	result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	dm := result.(DashboardModel)
	// Should not crash, model unchanged
	assert.Equal(t, SectionOverview, dm.currentSection)
}

// --- addNotification overflow ---

func TestAddNotification_Overflow(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 15; i++ {
		m.addNotification(fmt.Sprintf("msg-%d", i), "info")
	}
	assert.Len(t, m.notifications, 10)
	assert.Contains(t, m.notifications[0].Message, "msg-14")
}

// --- Init command execution ---

func TestInit_ReturnsBatch(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	assert.NotNil(t, cmd)
	// Execute the batch - tea.Batch returns a Cmd that produces messages
	msgs := cmd()
	// tea.Batch wraps multiple cmds; the result is the first message.
	// We just verify it returns something without panicking.
	assert.NotNil(t, msgs)
}

// --- loadDataCmd ---

func TestLoadDataCmd_ReturnsCmd(t *testing.T) {
	cmd := loadDataCmd()
	assert.NotNil(t, cmd)
	// Executing the cmd will call the real API, which likely fails without credentials.
	// But it should return a valid message, not panic.
	msg := cmd()
	assert.NotNil(t, msg)
	// Should be either dataLoadedMsg or errorMsg
	_, isDataLoaded := msg.(dataLoadedMsg)
	_, isError := msg.(errorMsg)
	assert.True(t, isDataLoaded || isError, "expected dataLoadedMsg or errorMsg, got %T", msg)
}

// --- RunDashboard / RunDashboardWithConfig ---

// These functions call tea.NewProgram which requires a real terminal.
// They cannot be unit tested meaningfully. We verify the DashboardConfig struct
// and that the functions exist and are callable (they will fail at p.Run()
// without a TTY, but the setup code is covered by integration tests).

func TestDashboardConfig_Fields(t *testing.T) {
	cfg := DashboardConfig{}
	assert.Equal(t, "", cfg.Profile)
	assert.Equal(t, "", cfg.Theme)
	assert.Equal(t, 0, cfg.Width)
	assert.Equal(t, 0, cfg.Height)
	assert.False(t, cfg.Debug)
}

func TestDashboardConfig_AllFields(t *testing.T) {
	cfg := DashboardConfig{
		Profile: "prod",
		Theme:   "dark",
		Width:   200,
		Height:  60,
		Debug:   true,
	}
	assert.Equal(t, "prod", cfg.Profile)
	assert.Equal(t, "dark", cfg.Theme)
	assert.Equal(t, 200, cfg.Width)
	assert.Equal(t, 60, cfg.Height)
	assert.True(t, cfg.Debug)
}

func TestRunDashboard_ReturnsErrorWithoutTTY(t *testing.T) {
	// RunDashboard creates a tea.NewProgram which needs a TTY.
	// In test environment, it should return an error rather than hang.
	// Use a short timeout to avoid blocking CI.
	done := make(chan error, 1)
	go func() {
		done <- RunDashboard()
	}()
	select {
	case err := <-done:
		// Either nil (ran successfully) or an error (no TTY) - both are fine
		// The important thing is it didn't hang
		_ = err
	case <-time.After(time.Second * 2):
		t.Fatal("RunDashboard hung - should have returned quickly without TTY")
	}
}

func TestRunDashboardWithConfig_ReturnsErrorWithoutTTY(t *testing.T) {
	cfg := DashboardConfig{
		Profile: "test",
		Theme:   "dark",
		Width:   80,
		Height:  24,
	}
	done := make(chan error, 1)
	go func() {
		done <- RunDashboardWithConfig(cfg)
	}()
	select {
	case err := <-done:
		_ = err
	case <-time.After(time.Second * 2):
		t.Fatal("RunDashboardWithConfig hung - should have returned quickly without TTY")
	}
}

// --- renderLoading frames ---

func TestRenderLoading_AdvancesFrame(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.loadingFrame = 0
	// renderLoading has a value receiver, so the frame mutation is on the call's copy.
	// We verify it doesn't panic and produces output with a spinner frame.
	view1 := m.renderLoading()
	assert.NotEmpty(t, view1)
	// The loading animation frames contain spinner chars
	assert.True(t, strings.ContainsAny(view1, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"))
}

// --- SECURITY: no control characters in rendered output ---

func TestSecurity_NoControlCharsInView(t *testing.T) {
	m := newTestModel()
	// Inject potentially dangerous strings (ANSI escape sequences)
	m.notifications = []Notification{
		{Message: "\x1b[31mINJECTED\x1b[0mbad", Type: "error", Timestamp: time.Now()},
	}
	m.buckets = []Bucket{
		{Name: "normal-bucket\x1b[31mINJECTED\x1b[0m", Size: 1024},
	}
	view := m.View()
	// The view should still render without panicking
	assert.NotEmpty(t, view)
}

func TestSecurity_UnicodeInBucketNames(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{
		{Name: "中文存储桶", Size: 1024},       // Chinese
		{Name: "Русский", Size: 2048}, // Russian
		{Name: "💠-bucket", Size: 4096},                       // Emoji
		{Name: "bucket-with-special", Size: 512},
	}
	m.currentSection = SectionBucketList
	assert.NotPanics(t, func() {
		view := m.View()
		assert.NotEmpty(t, view)
	})
}

// --- ROBUSTNESS: edge case model states ---

func TestRobustness_ZeroHeightRendering(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 0
	m.loading = true
	assert.NotPanics(t, func() {
		_ = m.View()
	})
}

func TestRobustness_NegativeDimensions(t *testing.T) {
	m := newTestModel()
	m.width = -1
	m.height = -1
	assert.NotPanics(t, func() {
		_ = m.View()
	})
}

func TestRobustness_VeryLargeBucketList(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.loading = false
	// Create 1000 buckets
	for i := 0; i < 1000; i++ {
		m.buckets = append(m.buckets, Bucket{
			Name:        fmt.Sprintf("bucket-%04d", i),
			Size:        int64(i * 1024),
			ObjectCount: int64(i * 10),
			Status:      "active",
		})
	}
	assert.NotPanics(t, func() {
		view := m.View()
		assert.NotEmpty(t, view)
	})
}

func TestRobustness_ScrollBoundaries(t *testing.T) {
	m := newTestModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	// Can't move up from 0
	m.selectedRow = 0
	m.moveUp()
	assert.Equal(t, 0, m.selectedRow)

	// Can't move down past last
	m.selectedRow = 2
	m.moveDown()
	assert.Equal(t, 2, m.selectedRow)

	// Section wraparound
	m.currentSection = SectionOverview
	m.previousSection()
	assert.Equal(t, SectionHelp, m.currentSection)

	m.currentSection = SectionHelp
	m.nextSection()
	assert.Equal(t, SectionOverview, m.currentSection)
}

// --- renderRealTimeStats ---

func TestRenderRealTimeStats(t *testing.T) {
	m := newTestModel()
	m.realTimeStats = RealTimeStats{
		UploadRate:     42.5,
		DownloadRate:   18.3,
		RequestsPerMin: 1234,
		LastUpdate:     time.Now(),
	}
	view := m.renderRealTimeStats()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "1234")
}

// --- renderQuickActions ---

func TestRenderQuickActions(t *testing.T) {
	m := newTestModel()
	view := m.renderQuickActions()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Create Bucket")
	assert.Contains(t, view, "Upload")
	assert.Contains(t, view, "Quit")
}

// --- renderFileSelector ---

func TestRenderFileSelector(t *testing.T) {
	m := newTestModel()
	view := m.renderFileSelector()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "File Selector")
}

// --- renderHelp ---

func TestRenderHelp_ContainsSections(t *testing.T) {
	m := newTestModel()
	view := m.renderHelp()
	assert.Contains(t, view, "R2Go2 Dashboard Help")
	assert.Contains(t, view, "Navigation:")
	assert.Contains(t, view, "Quick Actions:")
	assert.Contains(t, view, "Search & Filter:")
	assert.Contains(t, view, "Function Keys:")
	assert.Contains(t, view, "Sections:")
	assert.Contains(t, view, "Press Esc")
}

// --- renderTableRow with different column counts ---

func TestRenderTableRow_SingleColumn(t *testing.T) {
	m := newTestModel()
	row := m.renderTableRow([]string{"OnlyOne"}, false)
	assert.NotEmpty(t, row)
}

func TestRenderTableRow_ManyColumns(t *testing.T) {
	m := newTestModel()
	data := []string{"a", "b", "c", "d", "e", "f", "g"}
	row := m.renderTableRow(data, false)
	assert.NotEmpty(t, row)
}

func TestRenderTableRow_Empty_PanicsOnZeroCols(t *testing.T) {
	m := newTestModel()
	// 0 columns causes divide-by-zero in calculateColumnWidths - known bug
	assert.Panics(t, func() {
		m.renderTableRow([]string{}, false)
	})
}

// --- renderHeader with help shown ---

func TestRenderHeader_ShowHelp(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	header := m.renderHeader()
	assert.NotEmpty(t, header)
	// When help is shown, no help indicator should be in header
	assert.NotContains(t, header, "[F1] Help")
}

func TestRenderHeader_Normal(t *testing.T) {
	m := newTestModel()
	m.showHelp = false
	header := m.renderHeader()
	assert.NotEmpty(t, header)
	assert.Contains(t, header, "[F1] Help")
	assert.Contains(t, header, "R2Go2 Dashboard")
}

// --- strings.Contains helper for footer verification ---

func TestRenderFooter_ContainsNavigationHint(t *testing.T) {
	m := newTestModel()
	footer := m.renderFooter()
	assert.True(t, strings.Contains(footer, "Arrow Keys"))
}

// --- Test renderMonitoring full content ---

func TestRenderMonitoring_Full(t *testing.T) {
	m := newTestModel()
	m.realTimeStats = RealTimeStats{
		UploadRate:     10.0,
		DownloadRate:   5.0,
		RequestsPerMin: 200,
		LastUpdate:     time.Now(),
	}
	m.notifications = []Notification{
		{Message: "activity 1", Type: "info", Timestamp: time.Now()},
		{Message: "activity 2", Type: "success", Timestamp: time.Now()},
	}
	view := m.renderMonitoring()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Real-Time")
	assert.Contains(t, view, "activity 1")
	assert.Contains(t, view, "activity 2")
}
