package palette

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func testCommands() []Command {
	return []Command{
		{Name: "bucket create", Description: "Create a new R2 bucket", Category: "R2", Icon: "🪣"},
		{Name: "bucket list", Description: "List all buckets", Category: "R2", Icon: "🪣"},
		{Name: "bucket delete", Description: "Delete a bucket", Category: "R2", Icon: "🪣"},
		{Name: "dns list", Description: "List DNS records", Category: "DNS", Icon: "🌐"},
		{Name: "dns create", Description: "Create a DNS record", Category: "DNS", Icon: "🌐"},
		{Name: "config init", Description: "Initialize configuration", Category: "Settings", Icon: "⚙️"},
		{Name: "Go to Overview", Description: "Switch to overview", Category: "Navigation", Icon: "🏠"},
		{Name: "Quit", Description: "Exit the dashboard", Category: "Dashboard", Icon: "🚪"},
	}
}

func newTestPalette() *PaletteModel {
	return New(testCommands(), 80, 24)
}

func typeString(m *PaletteModel, s string) {
	for _, ch := range s {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		*m = updated
	}
}

// --- Tests ---

func TestNewPalette(t *testing.T) {
	p := newTestPalette()
	assert.NotNil(t, p)
	assert.False(t, p.IsVisible())
	assert.Equal(t, 8, len(p.commands))
	assert.Equal(t, 0, p.cursor)
}

func TestShowHide(t *testing.T) {
	p := newTestPalette()

	p.Show()
	assert.True(t, p.IsVisible())
	assert.Equal(t, "", p.input.Value())
	assert.Equal(t, 0, p.cursor)

	p.Hide()
	assert.False(t, p.IsVisible())
}

func TestShowResetsState(t *testing.T) {
	p := newTestPalette()
	p.Show()
	typeString(p, "bucket")
	p.cursor = 2

	// Show again — state must reset
	p.Show()
	assert.Equal(t, "", p.input.Value())
	assert.Equal(t, 0, p.cursor)
}

func TestEmptyQueryShowsAllCommands(t *testing.T) {
	p := newTestPalette()
	p.Show()

	// With empty query, resultCount should show all available (up to maxVisible)
	count := p.resultCount()
	assert.Equal(t, 8, count) // 8 test commands, all under maxVisible(10)
}

func TestFuzzyFiltering(t *testing.T) {
	p := newTestPalette()
	p.Show()

	typeString(p, "buc cre")
	require.True(t, len(p.matches) > 0, "expected at least one match")

	// The top match should be "bucket create"
	available := p.availableCommands()
	topCmd := available[p.matches[0].Index]
	assert.Equal(t, "bucket create", topCmd.Name)
}

func TestFuzzyFilteringPartialMatch(t *testing.T) {
	p := newTestPalette()
	p.Show()

	typeString(p, "dns")
	require.True(t, len(p.matches) >= 2, "expected at least 2 DNS matches")

	// All matches should be DNS commands
	available := p.availableCommands()
	for _, match := range p.matches {
		cmd := available[match.Index]
		assert.Contains(t, cmd.Name, "dns", "expected DNS command, got %s", cmd.Name)
	}
}

func TestNoMatches(t *testing.T) {
	p := newTestPalette()
	p.Show()

	typeString(p, "xyznonexistent")
	assert.Equal(t, 0, len(p.matches))
	assert.Equal(t, 0, p.resultCount())
}

func TestCursorNavigation(t *testing.T) {
	p := newTestPalette()
	p.Show()

	// Empty query — cursor navigates all commands
	// Down
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	*p = updated
	assert.Equal(t, 1, p.cursor)

	updated, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
	*p = updated
	assert.Equal(t, 2, p.cursor)

	// Up
	updated, _ = p.Update(tea.KeyMsg{Type: tea.KeyUp})
	*p = updated
	assert.Equal(t, 1, p.cursor)
}

func TestCursorStaysAtTop(t *testing.T) {
	p := newTestPalette()
	p.Show()
	p.cursor = 0

	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyUp})
	*p = updated
	assert.Equal(t, 0, p.cursor)
}

func TestCursorStaysAtBottom(t *testing.T) {
	p := newTestPalette()
	p.Show()

	// Move cursor to the end
	for i := 0; i < 20; i++ {
		updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyDown})
		*p = updated
	}

	// Should be clamped to last item
	assert.Equal(t, p.resultCount()-1, p.cursor)
}

func TestEnterSelectsCommand(t *testing.T) {
	p := newTestPalette()
	p.Show()

	// Select first command (bucket create)
	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	*p = updated
	assert.False(t, p.IsVisible(), "palette should hide after selection")
	require.NotNil(t, cmd)

	msg := cmd()
	selectMsg, ok := msg.(PaletteSelectMsg)
	require.True(t, ok, "expected PaletteSelectMsg, got %T", msg)
	assert.Equal(t, "bucket create", selectMsg.Command.Name)
}

func TestEnterWithFilteredResults(t *testing.T) {
	p := newTestPalette()
	p.Show()
	typeString(p, "dns")

	// Move to second result
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	*p = updated

	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	*p = updated
	assert.False(t, p.IsVisible())
	require.NotNil(t, cmd)

	msg := cmd()
	selectMsg, ok := msg.(PaletteSelectMsg)
	require.True(t, ok)
	assert.Contains(t, selectMsg.Command.Name, "dns")
}

func TestEscDismisses(t *testing.T) {
	p := newTestPalette()
	p.Show()

	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	*p = updated
	assert.False(t, p.IsVisible(), "palette should hide on Esc")
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(PaletteDismissMsg)
	assert.True(t, ok, "expected PaletteDismissMsg, got %T", msg)
}

func TestContextFiltering(t *testing.T) {
	commands := []Command{
		{Name: "always visible", Description: "always here"},
		{Name: "context hidden", Description: "should be hidden", ContextFunc: func() bool { return false }},
		{Name: "context visible", Description: "should be visible", ContextFunc: func() bool { return true }},
	}

	p := New(commands, 80, 24)
	p.Show()

	available := p.availableCommands()
	assert.Equal(t, 2, len(available))

	names := []string{available[0].Name, available[1].Name}
	assert.Contains(t, names, "always visible")
	assert.Contains(t, names, "context visible")
	assert.NotContains(t, names, "context hidden")
}

func TestBuildRegistryFromCobra(t *testing.T) {
	root := &cobra.Command{Use: "r2go2", Short: "R2Go2 CLI"}

	bucket := &cobra.Command{Use: "bucket", Short: "Bucket operations"}
	bucket.AddCommand(&cobra.Command{Use: "create", Short: "Create a bucket"})
	bucket.AddCommand(&cobra.Command{Use: "list", Short: "List buckets"})
	bucket.AddCommand(&cobra.Command{Use: "delete", Short: "Delete a bucket", Hidden: true})

	dns := &cobra.Command{Use: "dns", Short: "DNS management"}
	dns.AddCommand(&cobra.Command{Use: "list", Short: "List DNS records"})

	root.AddCommand(bucket)
	root.AddCommand(dns)

	commands := BuildRegistryFromCobra(root)

	// Should have: bucket create, bucket list, dns list (delete is hidden)
	assert.Equal(t, 3, len(commands))

	names := make([]string, len(commands))
	for i, c := range commands {
		names[i] = c.Name
	}
	assert.Contains(t, names, "r2go2 bucket create")
	assert.Contains(t, names, "r2go2 bucket list")
	assert.Contains(t, names, "r2go2 dns list")
	assert.NotContains(t, names, "r2go2 bucket delete") // hidden
}

func TestBuildRegistryCategories(t *testing.T) {
	root := &cobra.Command{Use: "r2go2"}
	bucket := &cobra.Command{Use: "bucket", Short: "Bucket ops"}
	bucket.AddCommand(&cobra.Command{Use: "create", Short: "Create"})
	root.AddCommand(bucket)

	commands := BuildRegistryFromCobra(root)
	require.Equal(t, 1, len(commands))
	assert.Equal(t, "bucket", commands[0].Category)
	assert.Equal(t, "🪣", commands[0].Icon)
}

func TestDefaultDashboardActions(t *testing.T) {
	actions := DefaultDashboardActions()
	assert.True(t, len(actions) >= 5, "expected at least 5 default actions")

	names := make([]string, len(actions))
	for i, a := range actions {
		names[i] = a.Name
	}
	assert.Contains(t, names, "Go to Overview")
	assert.Contains(t, names, "Quit")
	assert.Contains(t, names, "Refresh Data")
}

func TestViewNotVisibleReturnsEmpty(t *testing.T) {
	p := newTestPalette()
	assert.Equal(t, "", p.View())
}

func TestViewVisibleRendersContent(t *testing.T) {
	p := newTestPalette()
	p.Show()

	view := p.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Esc to close")
	assert.Contains(t, view, "result(s)")
}

func TestHighlightMatches(t *testing.T) {
	result := highlightMatches("bucket", []int{0, 3})
	// All original characters must be present in the output
	assert.Contains(t, result, "u")
	assert.Contains(t, result, "c")
	assert.Contains(t, result, "e")
	assert.Contains(t, result, "t")
	// Result length must be >= original (equal without TTY, longer with ANSI codes)
	assert.GreaterOrEqual(t, len(result), len("bucket"))

	// Empty indexes should return the original string unchanged
	assert.Equal(t, "hello", highlightMatches("hello", nil))
	assert.Equal(t, "hello", highlightMatches("hello", []int{}))

	// All characters highlighted
	all := highlightMatches("ab", []int{0, 1})
	assert.Contains(t, all, "a")
	assert.Contains(t, all, "b")
}

func TestSetSize(t *testing.T) {
	p := newTestPalette()
	p.SetSize(120, 40)
	assert.Equal(t, 120, p.width)
	assert.Equal(t, 40, p.height)
}

func TestCommandString(t *testing.T) {
	c := Command{Name: "bucket create", Description: "Create a new bucket"}
	assert.Equal(t, "bucket create Create a new bucket", c.String())
}
