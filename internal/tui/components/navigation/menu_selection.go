/*
Package navigation provides professional menu selection with keyboard navigation

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package navigation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// MenuOption represents a single menu option
type MenuOption struct {
	ID       int
	Title    string
	Subtitle string
	Icon     string
	Enabled  bool
}

// FilterValue implements list.Item interface
func (m *MenuOption) FilterValue() string {
	return m.Title
}

// MenuModel represents the interactive menu selection model
type MenuModel struct {
	list          list.Model
	options       []MenuOption
	selectedIndex int
	width         int
	height        int
	viewport      int
	styles        *MenuStyles
	initialized   bool
}

// MenuStyles contains styling definitions for the menu
type MenuStyles struct {
	NormalTitle     lipgloss.Style
	SelectedTitle   lipgloss.Style
	NormalSubtitle  lipgloss.Style
	SelectedSubtitle lipgloss.Style
	NormalIcon      lipgloss.Style
	SelectedIcon    lipgloss.Style
	DisabledIcon    lipgloss.Style
	Border          lipgloss.Style
	SelectedBorder  lipgloss.Style
	HelpText        lipgloss.Style
}

// NewMenuModel creates a new menu selection model
func NewMenuModel(options []MenuOption, width, height int) *MenuModel {
	// Convert options to list items
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = &MenuOption{ID: opt.ID, Title: opt.Title, Subtitle: opt.Subtitle, Icon: opt.Icon, Enabled: opt.Enabled}
	}

	// Create list with custom delegate
	delegate := list.NewDefaultDelegate()
	listModel := list.New(items, delegate, width, height)
	listModel.SetShowStatusBar(false)
	listModel.SetFilteringEnabled(false)

	return &MenuModel{
		list:     listModel,
		options:  options,
		width:    width,
		height:   height,
		viewport: height,
		styles:   createDefaultMenuStyles(),
	}
}

// createDefaultMenuStyles creates clean, compact styling for the menu
func createDefaultMenuStyles() *MenuStyles {
	// Clean color palette
	textPrimary := lipgloss.Color("#FFFFFF")
	textSecondary := lipgloss.Color("#94A3B8")
	textMuted := lipgloss.Color("#64748B")
	accent := lipgloss.Color("#00D4AA")
	disabled := lipgloss.Color("#475569")

	return &MenuStyles{
		NormalTitle: lipgloss.NewStyle().
			Foreground(textSecondary),

		SelectedTitle: lipgloss.NewStyle().
			Foreground(textPrimary).
			Bold(true),

		NormalSubtitle: lipgloss.NewStyle().
			Foreground(textMuted),

		SelectedSubtitle: lipgloss.NewStyle().
			Foreground(textSecondary),

		NormalIcon: lipgloss.NewStyle().
			Foreground(textSecondary),

		SelectedIcon: lipgloss.NewStyle().
			Foreground(accent),

		DisabledIcon: lipgloss.NewStyle().
			Foreground(disabled),

		// No border - outer container has it
		Border: lipgloss.NewStyle().
			MarginTop(1),

		SelectedBorder: lipgloss.NewStyle(),

		HelpText: lipgloss.NewStyle().
			Foreground(textMuted).
			Align(lipgloss.Center),
	}
}

// Init initializes the menu model
func (m *MenuModel) Init() tea.Cmd {
	if !m.initialized {
		m.list.SetShowHelp(false)
		m.initialized = true
	}
	return nil
}

// Update handles keyboard input for menu navigation
func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp, tea.KeyDown:
			// Let the list handle arrow keys
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case tea.KeyEnter:
			// Handle selection
			if m.list.SelectedItem() != nil {
				selected := m.list.SelectedItem().(*MenuOption)
				if selected.Enabled {
					return m, func() tea.Msg {
						return MenuSelectionMsg{
							ID:    selected.ID,
							Title: selected.Title,
						}
					}
				}
			}

		case tea.KeyEsc:
			// Handle cancellation
			return m, tea.Quit

		case tea.KeyRunes:
			// Handle number shortcuts
			if len(msg.Runes) == 1 {
				num := msg.Runes[0]
				if num >= '1' && num <= '9' {
					optionIndex := int(num-'1') - 1
					if optionIndex >= 0 && optionIndex < len(m.options) {
						opt := m.options[optionIndex]
						if opt.Enabled {
							return m, func() tea.Msg {
								return MenuSelectionMsg{
									ID:    opt.ID,
									Title: opt.Title,
								}
							}
						}
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		// Handle window resizing
		if msg.Height != m.height {
			m.height = msg.Height
			m.viewport = msg.Height
			m.list.SetHeight(msg.Height)
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the menu
func (m *MenuModel) View() string {
	if !m.initialized {
		m.Init()
	}

	// Create content
	content := m.renderMenuContent()

	// Wrap in border
	return m.styles.Border.Render(content)
}

// renderMenuContent renders a clean, compact menu
func (m *MenuModel) renderMenuContent() string {
	var content []string

	// Get actual selected index from list
	selectedIndex := m.list.Index()

	// Render each menu item - compact single-line style
	for i, opt := range m.options {
		line := m.renderMenuItem(&opt, i == selectedIndex)
		content = append(content, line)
	}

	// Compact help footer
	content = append(content, "")
	content = append(content, m.renderHelpLine())

	return strings.Join(content, "\n")
}

// renderMenuItem renders a compact, single-line menu item
func (m *MenuModel) renderMenuItem(opt *MenuOption, isSelected bool) string {
	if !opt.Enabled {
		return m.renderDisabledMenuItem(opt)
	}

	// Selection indicator
	var indicator string
	var iconStyle, titleStyle, subtitleStyle lipgloss.Style

	if isSelected {
		indicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D4AA")).
			Bold(true).
			Render("▸ ")
		iconStyle = m.styles.SelectedIcon
		titleStyle = m.styles.SelectedTitle
		subtitleStyle = m.styles.SelectedSubtitle
	} else {
		indicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569")).
			Render("  ")
		iconStyle = m.styles.NormalIcon
		titleStyle = m.styles.NormalTitle
		subtitleStyle = m.styles.NormalSubtitle
	}

	// Number badge
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
	num := numStyle.Render(fmt.Sprintf("%d.", opt.ID))

	// Icon and title
	icon := iconStyle.Render(opt.Icon)
	title := titleStyle.Render(opt.Title)

	// Subtitle (dimmed, inline)
	subtitle := ""
	if opt.Subtitle != "" {
		subtitle = subtitleStyle.Render(" · " + opt.Subtitle)
	}

	return fmt.Sprintf("%s%s %s %s%s", indicator, num, icon, title, subtitle)
}

// renderDisabledMenuItem renders a disabled menu item
func (m *MenuModel) renderDisabledMenuItem(opt *MenuOption) string {
	num := m.styles.DisabledIcon.Render(fmt.Sprintf("  %d.", opt.ID))
	icon := m.styles.DisabledIcon.Render(opt.Icon)
	title := m.styles.DisabledIcon.Render(opt.Title)
	subtitle := ""
	if opt.Subtitle != "" {
		subtitle = m.styles.DisabledIcon.Render(" · " + opt.Subtitle)
	}
	return fmt.Sprintf("%s %s %s%s", num, icon, title, subtitle)
}

// renderHelpLine renders a compact help footer
func (m *MenuModel) renderHelpLine() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA")).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	return textStyle.Render("  ") +
		keyStyle.Render("↑↓") + textStyle.Render(" navigate") +
		sepStyle.Render(" · ") +
		keyStyle.Render("enter") + textStyle.Render(" select") +
		sepStyle.Render(" · ") +
		keyStyle.Render("1-5") + textStyle.Render(" quick") +
		sepStyle.Render(" · ") +
		keyStyle.Render("esc") + textStyle.Render(" exit")
}


// GetSelection returns the currently selected option
func (m *MenuModel) GetSelection() *MenuOption {
	if item := m.list.SelectedItem(); item != nil {
		return item.(*MenuOption)
	}
	return nil
}

// SetOptions updates the menu options
func (m *MenuModel) SetOptions(options []MenuOption) {
	m.options = options
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = &opt
	}
	m.list.SetItems(items)
}

// SetWidth updates the menu width
func (m *MenuModel) SetWidth(width int) {
	m.width = width
	m.list.SetWidth(width)
}

// SetHeight updates the menu height
func (m *MenuModel) SetHeight(height int) {
	m.height = height
	m.viewport = height
	m.list.SetHeight(height)
}

// MenuSelectionMsg is emitted when a menu selection is made
type MenuSelectionMsg struct {
	ID    int
	Title string
}