/*
Package palette provides a VS Code-style command palette for the TUI dashboard

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package palette

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// PaletteSelectMsg is sent when the user selects a command.
type PaletteSelectMsg struct {
	Command Command
}

// PaletteDismissMsg is sent when the user dismisses the palette.
type PaletteDismissMsg struct{}

// maxVisible is the maximum number of results shown in the palette.
const maxVisible = 10

// PaletteModel is a Bubble Tea sub-model that renders a fuzzy-search
// command palette as a centered overlay.
type PaletteModel struct {
	input    textinput.Model
	commands []Command
	matches  []fuzzy.Match
	cursor   int
	visible  bool
	width    int
	height   int
}

// commandSource adapts []Command for sahilm/fuzzy.
type commandSource []Command

func (s commandSource) String(i int) string { return s[i].String() }
func (s commandSource) Len() int            { return len(s) }

// New creates a PaletteModel pre-loaded with the given commands.
func New(commands []Command, width, height int) *PaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.CharLimit = 120
	ti.Width = 50

	return &PaletteModel{
		input:    ti,
		commands: commands,
		width:    width,
		height:   height,
	}
}

// Show opens the palette, focuses the text input, and resets state.
func (m *PaletteModel) Show() {
	m.visible = true
	m.input.SetValue("")
	m.input.Focus()
	m.cursor = 0
	m.matches = nil
}

// Hide closes the palette.
func (m *PaletteModel) Hide() {
	m.visible = false
	m.input.Blur()
}

// IsVisible returns whether the palette is currently shown.
func (m *PaletteModel) IsVisible() bool {
	return m.visible
}

// SetSize updates the available dimensions for rendering.
func (m *PaletteModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init satisfies tea.Model (no-op; palette is driven by the parent).
func (m PaletteModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update processes key messages when the palette is visible.
func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.Hide()
			return m, func() tea.Msg { return PaletteDismissMsg{} }

		case tea.KeyEnter:
			selected := m.selectedCommand()
			if selected != nil {
				m.Hide()
				return m, func() tea.Msg { return PaletteSelectMsg{Command: *selected} }
			}
			return m, nil

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			max := m.resultCount() - 1
			if max < 0 {
				max = 0
			}
			if m.cursor < max {
				m.cursor++
			}
			return m, nil

		default:
			// Forward to text input for typing
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}

	return m, nil
}

// applyFilter runs fuzzy matching against the current input value.
func (m *PaletteModel) applyFilter() {
	query := m.input.Value()
	if query == "" {
		m.matches = nil
		m.cursor = 0
		return
	}

	// Filter out commands whose ContextFunc returns false
	available := m.availableCommands()
	m.matches = fuzzy.FindFrom(query, commandSource(available))
	if m.cursor >= len(m.matches) {
		m.cursor = 0
	}
}

// availableCommands returns commands that pass their ContextFunc filter.
func (m *PaletteModel) availableCommands() []Command {
	var out []Command
	for _, c := range m.commands {
		if c.ContextFunc == nil || c.ContextFunc() {
			out = append(out, c)
		}
	}
	return out
}

// resultCount returns the number of visible results.
func (m *PaletteModel) resultCount() int {
	if m.input.Value() == "" {
		available := m.availableCommands()
		if len(available) > maxVisible {
			return maxVisible
		}
		return len(available)
	}
	if len(m.matches) > maxVisible {
		return maxVisible
	}
	return len(m.matches)
}

// selectedCommand returns the command under the cursor, or nil.
func (m *PaletteModel) selectedCommand() *Command {
	if m.input.Value() == "" {
		available := m.availableCommands()
		if m.cursor < len(available) {
			return &available[m.cursor]
		}
		return nil
	}
	if m.cursor < len(m.matches) {
		available := m.availableCommands()
		idx := m.matches[m.cursor].Index
		if idx < len(available) {
			return &available[idx]
		}
	}
	return nil
}

// --- Rendering ---

// Styles
var (
	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5DADE2")).
			Padding(1, 2)

	resultNormalStyle = lipgloss.NewStyle().
				Padding(0, 1)

	resultSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#5DADE2")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)

	matchHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F39C12")).
				Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D")).
			Padding(1, 0, 0, 0)

	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D"))

	descriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#95A5A6"))
)

// View renders the palette as a centered overlay string.
func (m PaletteModel) View() string {
	if !m.visible {
		return ""
	}

	// Build the content
	var b strings.Builder

	// Text input
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")

	// Results
	count := 0
	if m.input.Value() == "" {
		// Show all available commands (up to maxVisible)
		available := m.availableCommands()
		for i, cmd := range available {
			if count >= maxVisible {
				break
			}
			b.WriteString(m.renderResult(i, cmd.Icon, cmd.Name, cmd.Description, nil))
			b.WriteString("\n")
			count++
		}
	} else {
		// Show fuzzy matches
		available := m.availableCommands()
		for i, match := range m.matches {
			if count >= maxVisible {
				break
			}
			cmd := available[match.Index]
			b.WriteString(m.renderResult(i, cmd.Icon, cmd.Name, cmd.Description, match.MatchedIndexes))
			b.WriteString("\n")
			count++
		}
	}

	if count == 0 {
		b.WriteString(categoryStyle.Render("  No matching commands"))
		b.WriteString("\n")
	}

	// Footer
	total := m.resultCount()
	b.WriteString(footerStyle.Render(fmt.Sprintf("  %d result(s)          Esc to close", total)))

	content := overlayStyle.Render(b.String())

	// Center the overlay
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}

	return content
}

// renderResult renders a single result row with optional match highlighting.
func (m *PaletteModel) renderResult(index int, icon, name, description string, matchedIndexes []int) string {
	style := resultNormalStyle
	if index == m.cursor {
		style = resultSelectedStyle
	}

	// Highlight matched characters in the name
	displayName := name
	if len(matchedIndexes) > 0 {
		displayName = highlightMatches(name, matchedIndexes)
	}

	desc := ""
	if description != "" {
		desc = " " + descriptionStyle.Render(description)
	}

	return style.Render(fmt.Sprintf("%s %s", icon, displayName)) + desc
}

// highlightMatches applies bold+color to matched character positions.
func highlightMatches(s string, indexes []int) string {
	// Build a set for O(1) lookup
	matchSet := make(map[int]bool, len(indexes))
	for _, idx := range indexes {
		matchSet[idx] = true
	}

	var b strings.Builder
	for i, ch := range s {
		if matchSet[i] {
			b.WriteString(matchHighlightStyle.Render(string(ch)))
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
