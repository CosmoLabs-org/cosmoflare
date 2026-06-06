package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500")).Bold(true)
	defaultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

// textInputModel wraps bubbles/textinput for a single-line prompt.
type textInputModel struct {
	input        textinput.Model
	prompt       string
	defaultValue string
	value        string
	done         bool
	cancelled    bool
}

func newTextInputModel(prompt, defaultValue string) textInputModel {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.Focus()
	ti.Width = 40

	return textInputModel{
		input:        ti,
		prompt:       prompt,
		defaultValue: defaultValue,
	}
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.value = m.input.Value()
			if m.value == "" {
				m.value = m.defaultValue
			}
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(promptStyle.Render(m.prompt))
	if m.defaultValue != "" {
		b.WriteString(defaultStyle.Render(fmt.Sprintf(" [%s]", m.defaultValue)))
	}
	b.WriteString("\n")
	b.WriteString(m.input.View())
	return b.String()
}

// Prompt runs an interactive text prompt using Bubble Tea.
// Falls back to PromptWithReader if stdin is not a terminal.
func Prompt(prompt, defaultValue string) (string, error) {
	m := newTextInputModel(prompt, defaultValue)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return PromptWithReader(prompt, defaultValue, DefaultInput())
	}
	final := result.(textInputModel)
	if final.cancelled {
		return defaultValue, nil
	}
	return final.value, nil
}
