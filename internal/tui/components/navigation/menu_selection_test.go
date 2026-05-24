package navigation

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOptions() []MenuOption {
	return []MenuOption{
		{ID: 1, Title: "Create Bucket", Subtitle: "New R2 bucket", Icon: "🪣", Enabled: true},
		{ID: 2, Title: "Upload Files", Subtitle: "Transfer data", Icon: "📤", Enabled: true},
		{ID: 3, Title: "View Stats", Subtitle: "Usage analytics", Icon: "📊", Enabled: true},
		{ID: 4, Title: "Settings", Subtitle: "Configuration", Icon: "⚙️", Enabled: true},
		{ID: 5, Title: "Coming Soon", Subtitle: "Future feature", Icon: "🔮", Enabled: false},
	}
}

func TestFilterValue(t *testing.T) {
	opt := &MenuOption{Title: "Test Option"}
	assert.Equal(t, "Test Option", opt.FilterValue())
}

func TestNewMenuModel(t *testing.T) {
	opts := newTestOptions()
	m := NewMenuModel(opts, 60, 20)
	require.NotNil(t, m)
	assert.Equal(t, 60, m.width)
	assert.Equal(t, 20, m.height)
	assert.Equal(t, 20, m.viewport)
	assert.False(t, m.initialized)
	assert.NotNil(t, m.styles)
	assert.Len(t, m.options, 5)
}

func TestCreateDefaultMenuStyles(t *testing.T) {
	s := createDefaultMenuStyles()
	require.NotNil(t, s)
	assert.NotNil(t, s.NormalTitle)
	assert.NotNil(t, s.SelectedTitle)
	assert.NotNil(t, s.NormalSubtitle)
	assert.NotNil(t, s.SelectedSubtitle)
	assert.NotNil(t, s.NormalIcon)
	assert.NotNil(t, s.SelectedIcon)
	assert.NotNil(t, s.DisabledIcon)
	assert.NotNil(t, s.Border)
	assert.NotNil(t, s.SelectedBorder)
	assert.NotNil(t, s.HelpText)
}

func TestMenuModel_Init(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)

	t.Run("first call initializes", func(t *testing.T) {
		cmd := m.Init()
		assert.Nil(t, cmd)
		assert.True(t, m.initialized)
	})

	t.Run("second call is no-op", func(t *testing.T) {
		cmd := m.Init()
		assert.Nil(t, cmd)
		assert.True(t, m.initialized)
	})
}

func TestMenuModel_Update_ArrowKeys(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.Init()

	t.Run("down key", func(t *testing.T) {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated := result.(*MenuModel)
		assert.NotNil(t, updated)
		// cmd may be nil when list is at boundary
	})

	t.Run("up key", func(t *testing.T) {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		updated := result.(*MenuModel)
		assert.NotNil(t, updated)
	})
}

func TestMenuModel_Update_Enter(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.Init()

	// First item should already be selected at index 0
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = result.(*MenuModel)

	// Execute the command to get the selection message
	msg := cmd()
	sel, ok := msg.(MenuSelectionMsg)
	require.True(t, ok)
	assert.Equal(t, 1, sel.ID)
	assert.Equal(t, "Create Bucket", sel.Title)
}

func TestMenuModel_Update_EnterDisabled(t *testing.T) {
	opts := newTestOptions()
	m := NewMenuModel(opts, 60, 20)
	m.Init()

	// Navigate to the disabled item (index 4)
	for i := 0; i < 4; i++ {
		m.list, _ = m.list.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Try to select disabled item
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := result.(*MenuModel)
	assert.NotNil(t, updated)
	assert.Nil(t, cmd) // No cmd for disabled item
}

func TestMenuModel_Update_Esc(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	msg := cmd()
	assert.Equal(t, tea.Quit(), msg)
}

func TestMenuModel_Update_NumberShortcuts(t *testing.T) {
	opts := newTestOptions()

	t.Run("number 1 selects first option (index 0)", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
		require.NotNil(t, cmd)
		updated := result.(*MenuModel)
		assert.NotNil(t, updated)
		msg := cmd()
		sel, ok := msg.(MenuSelectionMsg)
		require.True(t, ok)
		assert.Equal(t, 1, sel.ID)
	})

	t.Run("number 2 selects second option (index 1)", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		require.NotNil(t, cmd)
		updated := result.(*MenuModel)
		assert.NotNil(t, updated)
		msg := cmd()
		sel, ok := msg.(MenuSelectionMsg)
		require.True(t, ok)
		assert.Equal(t, 2, sel.ID)
	})

	t.Run("number 5 is disabled (no selection)", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
		assert.Nil(t, cmd)
	})

	t.Run("number 6 out of range", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
		_ = result
		assert.Nil(t, cmd)
	})

	t.Run("number 0 out of range", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
		assert.Nil(t, cmd)
	})

	t.Run("number 9 out of range", func(t *testing.T) {
		m := NewMenuModel(opts, 60, 20)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
		assert.Nil(t, cmd)
	})
}

func TestMenuModel_Update_WindowSize(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := result.(*MenuModel)
	assert.Equal(t, 30, updated.height)
	assert.Equal(t, 30, updated.viewport)
}

func TestMenuModel_Update_SameHeight(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	updated := result.(*MenuModel)
	assert.Equal(t, 20, updated.height)
}

func TestMenuModel_Update_UnknownMessage(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	type unknownMsg struct{}
	result, _ := m.Update(unknownMsg{})
	updated := result.(*MenuModel)
	assert.NotNil(t, updated)
	// cmd may be nil for unknown message types
}

func TestMenuModel_View(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	view := m.View()
	assert.NotEmpty(t, view)
	assert.True(t, m.initialized) // View calls Init if not initialized
}

func TestMenuModel_View_AutoInit(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	assert.False(t, m.initialized)
	view := m.View()
	assert.NotEmpty(t, view)
	assert.True(t, m.initialized)
}

func TestRenderMenuContent(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.Init()
	content := m.renderMenuContent()
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "Create Bucket")
	assert.Contains(t, content, "Upload Files")
	assert.Contains(t, content, "View Stats")
	assert.Contains(t, content, "Settings")
	assert.Contains(t, content, "Coming Soon")
	assert.Contains(t, content, "navigate")
	assert.Contains(t, content, "select")
}

func TestRenderMenuItem(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)

	t.Run("selected item", func(t *testing.T) {
		item := &MenuOption{ID: 1, Title: "Test", Subtitle: "Desc", Icon: "📦", Enabled: true}
		rendered := m.renderMenuItem(item, true)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "Test")
		assert.Contains(t, rendered, "Desc")
		assert.Contains(t, rendered, "▸") // selection indicator
	})

	t.Run("unselected item", func(t *testing.T) {
		item := &MenuOption{ID: 2, Title: "Other", Icon: "📁", Enabled: true}
		rendered := m.renderMenuItem(item, false)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "Other")
		assert.NotContains(t, rendered, "▸")
	})

	t.Run("disabled item delegates to renderDisabledMenuItem", func(t *testing.T) {
		item := &MenuOption{ID: 3, Title: "Disabled", Subtitle: "Nope", Icon: "🔒", Enabled: false}
		rendered := m.renderMenuItem(item, false)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "Disabled")
	})

	t.Run("item with empty subtitle", func(t *testing.T) {
		item := &MenuOption{ID: 4, Title: "NoSub", Subtitle: "", Icon: "📄", Enabled: true}
		rendered := m.renderMenuItem(item, true)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "NoSub")
	})
}

func TestRenderDisabledMenuItem(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)

	t.Run("with subtitle", func(t *testing.T) {
		item := &MenuOption{ID: 1, Title: "Disabled", Subtitle: "Not available", Icon: "🔒", Enabled: false}
		rendered := m.renderDisabledMenuItem(item)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "Disabled")
		assert.Contains(t, rendered, "Not available")
	})

	t.Run("without subtitle", func(t *testing.T) {
		item := &MenuOption{ID: 2, Title: "NoDesc", Subtitle: "", Icon: "🚫", Enabled: false}
		rendered := m.renderDisabledMenuItem(item)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "NoDesc")
	})
}

func TestRenderHelpLine(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	help := m.renderHelpLine()
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "navigate")
	assert.Contains(t, help, "select")
	assert.Contains(t, help, "quick")
	assert.Contains(t, help, "exit")
}

func TestGetSelection(t *testing.T) {
	t.Run("with items", func(t *testing.T) {
		m := NewMenuModel(newTestOptions(), 60, 20)
		m.Init()
		sel := m.GetSelection()
		require.NotNil(t, sel)
		assert.Equal(t, "Create Bucket", sel.Title)
	})

	t.Run("empty list returns nil", func(t *testing.T) {
		m := NewMenuModel([]MenuOption{}, 60, 20)
		m.Init()
		sel := m.GetSelection()
		assert.Nil(t, sel)
	})
}

func TestSetOptions(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.Init()

	newOpts := []MenuOption{
		{ID: 10, Title: "New Option A", Icon: "🆕", Enabled: true},
		{ID: 20, Title: "New Option B", Icon: "🆕", Enabled: true},
	}
	m.SetOptions(newOpts)
	assert.Len(t, m.options, 2)
	assert.Equal(t, "New Option A", m.options[0].Title)

	content := m.renderMenuContent()
	assert.Contains(t, content, "New Option A")
	assert.Contains(t, content, "New Option B")
}

func TestSetWidth(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.SetWidth(100)
	assert.Equal(t, 100, m.width)
}

func TestSetHeight(t *testing.T) {
	m := NewMenuModel(newTestOptions(), 60, 20)
	m.SetHeight(40)
	assert.Equal(t, 40, m.height)
	assert.Equal(t, 40, m.viewport)
}

// --- ROBUSTNESS ---

func TestMenuRobustness_EmptyOptions(t *testing.T) {
	m := NewMenuModel([]MenuOption{}, 60, 20)
	assert.NotPanics(t, func() {
		view := m.View()
		assert.NotEmpty(t, view)
	})
}

func TestMenuRobustness_SingleOption(t *testing.T) {
	opts := []MenuOption{
		{ID: 1, Title: "Only One", Icon: "📌", Enabled: true},
	}
	m := NewMenuModel(opts, 60, 20)
	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Only One")
}

func TestMenuRobustness_AllDisabled(t *testing.T) {
	opts := []MenuOption{
		{ID: 1, Title: "Off1", Enabled: false},
		{ID: 2, Title: "Off2", Enabled: false},
	}
	m := NewMenuModel(opts, 60, 20)
	m.Init()
	// Enter on disabled item should return nil cmd
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestMenuRobustness_ZeroDimensions(t *testing.T) {
	opts := newTestOptions()
	m := NewMenuModel(opts, 0, 0)
	assert.NotPanics(t, func() {
		view := m.View()
		assert.NotEmpty(t, view)
	})
}

func TestMenuRobustness_UnicodeInOptions(t *testing.T) {
	opts := []MenuOption{
		{ID: 1, Title: "中文菜单", Subtitle: "描述", Icon: "🎯", Enabled: true},
		{ID: 2, Title: "Русский", Subtitle: "Описание", Icon: "🇷🇺", Enabled: true},
	}
	m := NewMenuModel(opts, 60, 20)
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestMenuRobustness_VeryLongTitle(t *testing.T) {
	opts := []MenuOption{
		{ID: 1, Title: "This is an extremely long menu option title that goes on and on", Icon: "📜", Enabled: true},
	}
	m := NewMenuModel(opts, 60, 20)
	assert.NotPanics(t, func() {
		view := m.View()
		assert.NotEmpty(t, view)
	})
}
