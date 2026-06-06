package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500")).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500"))
)

type selectModel struct {
	prompt    string
	options   []string
	cursor    int
	selected  int
	done      bool
	cancelled bool
}

func newSelectModel(prompt string, options []string, defaultIndex int) selectModel {
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	return selectModel{
		prompt:  prompt,
		options: options,
		cursor:  defaultIndex,
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
		switch msg.String() {
		case "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(promptStyle.Render(m.prompt))
	b.WriteString("\n\n")

	for i, option := range m.options {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("► "))
			b.WriteString(selectedStyle.Render(option))
		} else {
			b.WriteString("  ")
			b.WriteString(unselectedStyle.Render(option))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(defaultStyle.Render("↑/↓ navigate  •  enter select  •  esc cancel"))
	return b.String()
}

// SelectFromList runs an interactive list selection using Bubble Tea.
// Falls back to SelectFromListWithReader if stdin is not a terminal.
func SelectFromList(prompt string, options []string, defaultIndex int) (int, error) {
	m := newSelectModel(prompt, options, defaultIndex)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return SelectFromListWithReader(prompt, options, defaultIndex, DefaultInput())
	}
	final := result.(selectModel)
	if final.cancelled {
		return defaultIndex, fmt.Errorf("selection cancelled")
	}
	return final.selected, nil
}
