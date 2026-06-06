package interactive

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	yesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Bold(true)
	noStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Bold(true)
)

type confirmModel struct {
	prompt     string
	defaultYes bool
	confirmed  bool
	done       bool
}

func newConfirmModel(prompt string, defaultYes bool) confirmModel {
	return confirmModel{
		prompt:     prompt,
		defaultYes: defaultYes,
		confirmed:  defaultYes,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "N":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		case "left", "h":
			m.confirmed = true
		case "right", "l":
			m.confirmed = false
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	yes := "  Yes  "
	no := "  No  "

	if m.confirmed {
		yes = yesStyle.Render("► Yes ")
		no = defaultStyle.Render("  No  ")
	} else {
		yes = defaultStyle.Render("  Yes ")
		no = noStyle.Render("► No  ")
	}

	return fmt.Sprintf("%s\n\n%s    %s\n\n%s",
		promptStyle.Render(m.prompt),
		yes, no,
		defaultStyle.Render("←/→ toggle  •  y/n  •  enter confirm"),
	)
}

// Confirm runs an interactive yes/no confirmation using Bubble Tea.
// Falls back to ConfirmWithReader if stdin is not a terminal.
func Confirm(prompt string, defaultYes bool) bool {
	m := newConfirmModel(prompt, defaultYes)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return ConfirmWithReader(prompt, defaultYes, DefaultInput())
	}
	final := result.(confirmModel)
	return final.confirmed
}
