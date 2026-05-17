package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// --- Key handling in Update() for navigation ---

func TestUpdate_KeyUp(t *testing.T) {
	m := initialModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.selectedRow = 1

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	dm := result.(DashboardModel)
	assert.Equal(t, 0, dm.selectedRow)
}

func TestUpdate_KeyDown(t *testing.T) {
	m := initialModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	dm := result.(DashboardModel)
	assert.Equal(t, 1, dm.selectedRow)
}

func TestUpdate_KeyEnterSelectsBucket(t *testing.T) {
	m := initialModel()
	m.currentSection = SectionBucketList
	m.buckets = []Bucket{{Name: "pick-me", Size: 512}}
	m.selectedRow = 0

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	dm := result.(DashboardModel)
	assert.Equal(t, "pick-me", dm.currentBucket.Name)
}

func TestUpdate_KeyQ_Quits(t *testing.T) {
	m := initialModel()
	m.showHelp = false

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.NotNil(t, cmd)
}

func TestUpdate_KeyQ_ClosesHelp(t *testing.T) {
	m := initialModel()
	m.showHelp = true

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	dm := result.(DashboardModel)
	assert.False(t, dm.showHelp)
	assert.Nil(t, cmd)
}

func TestUpdate_Escape_QuitsFromMainView(t *testing.T) {
	m := initialModel()
	m.showHelp = false

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
}

func TestUpdate_Escape_ClosesHelp(t *testing.T) {
	m := initialModel()
	m.showHelp = true

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dm := result.(DashboardModel)
	assert.False(t, dm.showHelp)
	assert.Nil(t, cmd)
}

// --- getCurrentProfile ---

func TestGetCurrentProfile(t *testing.T) {
	m := initialModel()
	assert.Equal(t, "default", m.getCurrentProfile())

	m.currentProfile = "production"
	assert.Equal(t, "production", m.getCurrentProfile())
}

// --- moveUp / moveDown boundaries via Update ---

func TestUpdate_KeyUp_AtTop(t *testing.T) {
	m := initialModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}
	m.selectedRow = 0

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	dm := result.(DashboardModel)
	assert.Equal(t, 0, dm.selectedRow) // stays at top
}

func TestUpdate_KeyDown_AtBottom(t *testing.T) {
	m := initialModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}}
	m.selectedRow = 1

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	dm := result.(DashboardModel)
	assert.Equal(t, 1, dm.selectedRow) // stays at bottom
}

// --- Section navigation via key shortcuts ---

func TestUpdate_SectionShortcuts_ThroughUpdate(t *testing.T) {
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
		t.Run("key_"+tt.key, func(t *testing.T) {
			m := initialModel()
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			dm := result.(DashboardModel)
			assert.Equal(t, tt.expected, dm.currentSection)
		})
	}
}

// --- Vim-style navigation keys via Update ---

func TestUpdate_VimKeys_JK(t *testing.T) {
	m := initialModel()
	m.buckets = []Bucket{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.selectedRow = 0

	// j = down
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	dm := result.(DashboardModel)
	assert.Equal(t, 1, dm.selectedRow)

	// k = up
	result, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	dm = result.(DashboardModel)
	assert.Equal(t, 0, dm.selectedRow)
}

func TestUpdate_VimKeys_HL(t *testing.T) {
	m := initialModel()
	m.currentSection = SectionOverview

	// l = next section
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	dm := result.(DashboardModel)
	assert.Equal(t, SectionBucketList, dm.currentSection)

	// h = previous section
	result, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	dm = result.(DashboardModel)
	assert.Equal(t, SectionOverview, dm.currentSection)
}
