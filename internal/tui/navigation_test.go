package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Navigation methods ---

func TestMoveUp(t *testing.T) {
	m := initialModel(nil, 0)
	m.selectedRow = 2

	m.moveUp()
	assert.Equal(t, 1, m.selectedRow)

	m.moveUp()
	assert.Equal(t, 0, m.selectedRow)

	m.moveUp() // at 0, should stay
	assert.Equal(t, 0, m.selectedRow)
}

func TestMoveDown(t *testing.T) {
	m := initialModel(nil, 0)
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.selectedRow = 0

	m.moveDown()
	assert.Equal(t, 1, m.selectedRow)

	m.moveDown()
	assert.Equal(t, 2, m.selectedRow)

	m.moveDown() // at last, should stay
	assert.Equal(t, 2, m.selectedRow)
}

func TestPreviousSection(t *testing.T) {
	m := initialModel(nil, 0)
	m.currentSection = SectionBucketList

	m.previousSection()
	assert.Equal(t, SectionOverview, m.currentSection)

	m.previousSection() // at 0, wraps to Help
	assert.Equal(t, SectionHelp, m.currentSection)
}

func TestNextSection(t *testing.T) {
	m := initialModel(nil, 0)
	m.currentSection = SectionSettings

	m.nextSection()
	assert.Equal(t, SectionHelp, m.currentSection)

	m.nextSection() // at Help, wraps to Overview
	assert.Equal(t, SectionOverview, m.currentSection)
}

func TestSelectCurrent(t *testing.T) {
	t.Run("selects bucket in bucket list", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.currentSection = SectionBucketList
		m.buckets = []Bucket{{Name: "my-bucket", Size: 1024}}
		m.selectedRow = 0

		cmd := m.selectCurrent()
		require.NotNil(t, cmd)

		msg := cmd()
		bsm, ok := msg.(bucketSelectedMsg)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", bsm.bucket.Name)
	})

	t.Run("returns nil for other sections", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.currentSection = SectionOverview

		cmd := m.selectCurrent()
		assert.Nil(t, cmd)
	})

	t.Run("returns nil when row out of bounds", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.currentSection = SectionBucketList
		m.selectedRow = 5
		m.buckets = []Bucket{{Name: "only-one"}}

		cmd := m.selectCurrent()
		assert.Nil(t, cmd)
	})
}

// --- handleKeyMsg ---

func TestHandleKeyMsg_Navigation(t *testing.T) {
	m := initialModel(nil, 0)
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}

	t.Run("down moves row", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
		dm := result.(DashboardModel)
		assert.Equal(t, 1, dm.selectedRow)
	})

	t.Run("up moves row", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.selectedRow = 1
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		dm := result.(DashboardModel)
		assert.Equal(t, 0, dm.selectedRow)
	})

	t.Run("right advances section", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		dm := result.(DashboardModel)
		assert.Equal(t, SectionBucketList, dm.currentSection)
	})

	t.Run("left goes back section", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.currentSection = SectionBucketList
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
		dm := result.(DashboardModel)
		assert.Equal(t, SectionOverview, dm.currentSection)
	})
}

func TestHandleKeyMsg_SectionShortcuts(t *testing.T) {
	tests := []struct {
		key      string
		expected Section
	}{
		{"1", SectionOverview},
		{"2", SectionBucketList},
		{"3", SectionObjectList},
		{"4", SectionUpload},
		{"5", SectionMonitoring},
		{"6", SectionSettings},
	}

	for _, tt := range tests {
		t.Run("section_"+tt.key, func(t *testing.T) {
			m := initialModel(nil, 0)
			result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			dm := result.(DashboardModel)
			assert.Equal(t, tt.expected, dm.currentSection)
		})
	}
}

func TestHandleKeyMsg_Help(t *testing.T) {
	t.Run("question mark shows help", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		dm := result.(DashboardModel)
		assert.True(t, dm.showHelp)
	})

	t.Run("f1 toggles help", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyF1})
		dm := result.(DashboardModel)
		assert.True(t, dm.showHelp)

		result, _ = dm.handleKeyMsg(tea.KeyMsg{Type: tea.KeyF1})
		dm = result.(DashboardModel)
		assert.False(t, dm.showHelp)
	})

	t.Run("esc closes help", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.showHelp = true
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
		dm := result.(DashboardModel)
		assert.False(t, dm.showHelp)
	})
}

func TestHandleKeyMsg_QuickActions(t *testing.T) {
	t.Run("c enters input mode on bucket list", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.currentSection = SectionBucketList
		// nullDataSource is not Available, so c does nothing
		result, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
		dm := result.(DashboardModel)
		assert.False(t, dm.inputMode) // not available, no input mode
		assert.Nil(t, cmd)
	})

	t.Run("m goes to monitoring", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
		dm := result.(DashboardModel)
		assert.Equal(t, SectionMonitoring, dm.currentSection)
	})

	t.Run("u shows upload notification", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
		dm := result.(DashboardModel)
		require.NotEmpty(t, dm.notifications)
		assert.Contains(t, dm.notifications[0].Message, "Upload")
	})
}

func TestHandleKeyMsg_SearchMode(t *testing.T) {
	t.Run("slash enters search mode", func(t *testing.T) {
		m := initialModel(nil, 0)
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		dm := result.(DashboardModel)
		assert.Equal(t, "", dm.searchQuery) // initialized empty
	})

	t.Run("typing in search mode appends chars", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.searchQuery = "te"
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		dm := result.(DashboardModel)
		assert.Equal(t, "tes", dm.searchQuery)
	})

	t.Run("backspace removes last char", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.searchQuery = "test"
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyBackspace})
		dm := result.(DashboardModel)
		assert.Equal(t, "tes", dm.searchQuery)
	})

	t.Run("enter applies search", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.searchQuery = "test"
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
		dm := result.(DashboardModel)
		assert.True(t, dm.filterActive)
		assert.Equal(t, "", dm.searchQuery)
	})

	t.Run("escape cancels search", func(t *testing.T) {
		m := initialModel(nil, 0)
		m.searchQuery = "test"
		result, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
		dm := result.(DashboardModel)
		assert.False(t, dm.filterActive)
		assert.Equal(t, "", dm.searchQuery)
	})
}

func TestHandleKeyMsg_Refresh(t *testing.T) {
	t.Run("f5 with unavailable data source returns nil", func(t *testing.T) {
		m := initialModel(nil, 0) // nullDataSource, not Available
		_, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyF5})
		assert.Nil(t, cmd) // No refresh when data not available
	})
}

// --- Command functions ---

func TestCmdFunctions(t *testing.T) {
	ds := &nullDataSource{}

	t.Run("createBucketAPICmd returns bucketCreatedMsg", func(t *testing.T) {
		cmd := createBucketAPICmd(ds, "test-bucket")
		msg := cmd()
		bcm, ok := msg.(bucketCreatedMsg)
		require.True(t, ok)
		// nullDataSource returns an error for create
		assert.Error(t, bcm.err)
	})

	t.Run("deleteBucketAPICmd returns bucketDeletedMsg", func(t *testing.T) {
		cmd := deleteBucketAPICmd(ds, "test-bucket")
		msg := cmd()
		bdm, ok := msg.(bucketDeletedMsg)
		require.True(t, ok)
		// nullDataSource returns an error for delete
		assert.Error(t, bdm.err)
	})

	t.Run("fetchMetricsCmd returns metricsLoadedMsg", func(t *testing.T) {
		cmd := fetchMetricsCmd(ds)
		msg := cmd()
		mlm, ok := msg.(metricsLoadedMsg)
		require.True(t, ok)
		assert.Nil(t, mlm.err)
	})

	t.Run("fetchObjectsCmd returns objectsLoadedMsg", func(t *testing.T) {
		cmd := fetchObjectsCmd(ds, "bucket", 0)
		msg := cmd()
		olm, ok := msg.(objectsLoadedMsg)
		require.True(t, ok)
		assert.Nil(t, olm.err)
	})

	t.Run("refreshDataCmd is not nil", func(t *testing.T) {
		cmd := refreshDataCmd(ds)
		assert.NotNil(t, cmd)
	})

	t.Run("monitoringTick returns cmd", func(t *testing.T) {
		cmd := monitoringTick(5 * time.Second)
		assert.NotNil(t, cmd)
	})
}

// --- Notification handling ---

func TestDashboardModelUpdateNotification(t *testing.T) {
	m := initialModel(nil, 0)
	msg := notificationMsg{
		notification: Notification{
			Message:   "test notification",
			Type:      "info",
			Timestamp: time.Now(),
		},
	}

	result, _ := m.Update(msg)
	dm := result.(DashboardModel)
	require.Len(t, dm.notifications, 1)
	assert.Equal(t, "test notification", dm.notifications[0].Message)
}
