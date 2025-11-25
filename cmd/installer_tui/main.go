/*
Package main implements the professional TUI installer for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/tui/components/installer"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/tui/components/navigation"
)

// InstallerModel represents the main installer model
type InstallerModel struct {
	header    *installer.HeaderModel
	menu      *navigation.MenuModel
	width     int
	height    int
	state     InstallerState
	styles    *InstallerStyles
}

// InstallerState represents the current installer state
type InstallerState int

const (
	StateMenu InstallerState = iota
	StateInstalling
	StateComplete
	StatePostInstall // New: post-installation setup menu
	StateError
)

// progressUpdateMsg represents a progress update message
type progressUpdateMsg struct {
	percentage int
}

// installCompleteMsg signals installation is complete
type installCompleteMsg struct{}

// InstallerStyles contains styling for the installer
type InstallerStyles struct {
	Container lipgloss.Style
	Menu     lipgloss.Style
	Status   lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
}

// NewInstallerModel creates a new installer model
func NewInstallerModel() *InstallerModel {
	width := 80
	height := 25

	return &InstallerModel{
		header: installer.CreateWelcomeHeader(width),
		menu:   createMenuModel(width, height-8),
		width:  width,
		height: height,
		state:  StateMenu,
		styles: createInstallerStyles(width),
	}
}

// createInstallerStyles creates clean, minimal styling for the installer
func createInstallerStyles(width int) *InstallerStyles {
	success := lipgloss.Color("#10B981")
	warning := lipgloss.Color("#F59E0B")
	errorColor := lipgloss.Color("#EF4444")
	border := lipgloss.Color("#2D3748")

	return &InstallerStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(1, 2),

		Menu: lipgloss.NewStyle(),

		Status: lipgloss.NewStyle().
			Foreground(warning).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
	}
}

// Init initializes the installer model
func (m *InstallerModel) Init() tea.Cmd {
	m.header.Init()
	m.menu.Init()
	return nil
}

// Update handles installer updates
func (m *InstallerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		}

	case navigation.MenuSelectionMsg:
		// Handle menu selection based on current state
		if m.state == StatePostInstall {
			cmds = append(cmds, m.handlePostInstallSelection(msg))
		} else {
			cmds = append(cmds, m.handleMenuSelection(msg))
		}

	case installCompleteMsg:
		// Transition to post-install menu
		m.state = StatePostInstall
		m.header = installer.CreateCompletionHeader(m.width)
		m.menu = createPostInstallMenu(m.width, m.height-8)
		m.menu.Init()

	case progressUpdateMsg:
		// Handle progress updates
		m.header.SetProgress(msg.percentage, 100)
		// Check if installation complete
		if msg.percentage >= 100 {
			return m, func() tea.Msg { return installCompleteMsg{} }
		}

	case tea.WindowSizeMsg:
		// Handle window resize
		m.handleResize(msg)
	}

	// Update header
	headerModel, cmd := m.header.Update(msg)
	if headerModel != nil {
		m.header = headerModel.(*installer.HeaderModel)
		cmds = append(cmds, cmd)
	}

	// Update menu
	menuModel, cmd := m.menu.Update(msg)
	if menuModel != nil {
		m.menu = menuModel.(*navigation.MenuModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the installer
func (m *InstallerModel) View() string {
	var content []string

	switch m.state {
	case StateMenu:
		content = append(content, m.header.View())
		content = append(content, m.menu.View())

	case StateInstalling:
		content = append(content, m.renderInstallingState())

	case StateComplete:
		content = append(content, m.renderCompleteState())

	case StatePostInstall:
		content = append(content, m.header.View())
		content = append(content, m.menu.View())

	case StateError:
		content = append(content, m.renderErrorState())

	default:
		return ""
	}

	return m.styles.Container.Render(strings.Join(content, "\n"))
}

// handleMenuSelection processes menu selections
func (m *InstallerModel) handleMenuSelection(selection navigation.MenuSelectionMsg) tea.Cmd {
	switch selection.ID {
	case 1: // Download from GitHub
		return m.startGitHubInstallation()

	case 2: // Use local build
		return m.startLocalInstallation()

	case 3: // View requirements
		return m.showRequirements()

	case 4: // Show installation history
		return m.showInstallationHistory()

	case 5: // Exit
		return tea.Quit

	default:
		return nil
	}
}

// startGitHubInstallation starts GitHub-based installation
func (m *InstallerModel) startGitHubInstallation() tea.Cmd {
	m.state = StateInstalling
	m.header = installer.CreateProgressHeader(m.width, "📥 Downloading from GitHub", "Fetching the latest R2Go2 release...")
	m.header.SetLoading(true)

	// Simulate download progress with sequential ticks
	return tea.Batch(
		tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg { return progressUpdateMsg{25} }),
		tea.Tick(time.Millisecond*1000, func(t time.Time) tea.Msg { return progressUpdateMsg{50} }),
		tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg { return progressUpdateMsg{75} }),
		tea.Tick(time.Millisecond*2000, func(t time.Time) tea.Msg { return progressUpdateMsg{100} }),
	)
}

// startLocalInstallation starts local build installation
func (m *InstallerModel) startLocalInstallation() tea.Cmd {
	m.state = StateInstalling
	m.header = installer.CreateProgressHeader(m.width, "🏗️ Local Build Installation", "Building R2Go2 from source...")
	m.header.SetLoading(true)

	// Simulate build progress
	return tea.Batch(
		tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg { return progressUpdateMsg{20} }),
		tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg { return progressUpdateMsg{40} }),
		tea.Tick(time.Millisecond*3000, func(t time.Time) tea.Msg { return progressUpdateMsg{80} }),
		tea.Tick(time.Millisecond*4000, func(t time.Time) tea.Msg { return progressUpdateMsg{100} }),
	)
}

// showRequirements displays installation requirements
func (m *InstallerModel) showRequirements() tea.Cmd {
	return func() tea.Msg {
		m.state = StateMenu
		return nil
	}
}

// showInstallationHistory displays installation history
func (m *InstallerModel) showInstallationHistory() tea.Cmd {
	return func() tea.Msg {
		m.state = StateMenu
		return nil
	}
}

// handleResize handles window resizing
func (m *InstallerModel) handleResize(msg tea.WindowSizeMsg) {
	m.width, m.height = msg.Width, msg.Height
	m.header.SetWidth(m.width)
	m.menu.SetWidth(m.width)
	m.menu.SetHeight(m.height - 8)
	m.styles.Container.Width(m.width)
}

// renderInstallingState renders the installation progress state
func (m *InstallerModel) renderInstallingState() string {
	return m.header.View()
}

// renderCompleteState renders the completion state
func (m *InstallerModel) renderCompleteState() string {
	var content []string
	content = append(content, m.header.View())
	content = append(content, "")

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	content = append(content, accentStyle.Render("Next steps:"))
	content = append(content, textStyle.Render("  • r2go2 --help    View all commands"))
	content = append(content, textStyle.Render("  • r2go2 setup     Configure your API token"))
	content = append(content, "")
	content = append(content, mutedStyle.Render("Press Esc to exit"))

	return strings.Join(content, "\n")
}

// renderErrorState renders the error state
func (m *InstallerModel) renderErrorState() string {
	var content []string
	content = append(content, m.header.View())
	content = append(content, "")

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	content = append(content, warnStyle.Render("Troubleshooting:"))
	content = append(content, textStyle.Render("  • Check your internet connection"))
	content = append(content, textStyle.Render("  • Verify system requirements"))
	content = append(content, textStyle.Render("  • Try running the installer again"))
	content = append(content, "")
	content = append(content, mutedStyle.Render("Press Esc to exit"))

	return strings.Join(content, "\n")
}

// createMenuModel creates the menu model with professional options
func createMenuModel(width, height int) *navigation.MenuModel {
	options := []navigation.MenuOption{
		{
			ID:       1,
			Title:    "Download from GitHub",
			Subtitle: "Requires internet connection",
			Icon:     "🌐",
			Enabled:  true,
		},
		{
			ID:       2,
			Title:    "Use local build",
			Subtitle: "Development mode installation",
			Icon:     "🏗️",
			Enabled:  true,
		},
		{
			ID:       3,
			Title:    "View installation requirements",
			Subtitle: "Check system compatibility",
			Icon:     "📋",
			Enabled:  true,
		},
		{
			ID:       4,
			Title:    "Show installation history",
			Subtitle: "Previous installations",
			Icon:     "📜",
			Enabled:  true,
		},
		{
			ID:       5,
			Title:    "Exit installer",
			Subtitle: "Return to terminal",
			Icon:     "🚪",
			Enabled:  true,
		},
	}

	return navigation.NewMenuModel(options, width-4, height)
}

// createPostInstallMenu creates the post-installation setup menu
func createPostInstallMenu(width, height int) *navigation.MenuModel {
	options := []navigation.MenuOption{
		{
			ID:       1,
			Title:    "Configure API credentials",
			Subtitle: "Set up Cloudflare R2 access",
			Icon:     "🔑",
			Enabled:  true,
		},
		{
			ID:       2,
			Title:    "Test connection",
			Subtitle: "Verify R2 bucket access",
			Icon:     "🔌",
			Enabled:  true,
		},
		{
			ID:       3,
			Title:    "View quick start guide",
			Subtitle: "Learn basic commands",
			Icon:     "📖",
			Enabled:  true,
		},
		{
			ID:       4,
			Title:    "Open documentation",
			Subtitle: "Full usage reference",
			Icon:     "📚",
			Enabled:  true,
		},
		{
			ID:       5,
			Title:    "Exit to terminal",
			Subtitle: "Start using R2Go2",
			Icon:     "✅",
			Enabled:  true,
		},
	}

	return navigation.NewMenuModel(options, width-4, height)
}

// handlePostInstallSelection handles post-installation menu selections
func (m *InstallerModel) handlePostInstallSelection(selection navigation.MenuSelectionMsg) tea.Cmd {
	switch selection.ID {
	case 1: // Configure API credentials
		return m.startAPISetup()
	case 2: // Test connection
		return m.testConnection()
	case 3: // Quick start guide
		return m.showQuickStart()
	case 4: // Documentation
		return m.openDocumentation()
	case 5: // Exit
		return tea.Quit
	default:
		return nil
	}
}

// startAPISetup launches the API credential configuration
func (m *InstallerModel) startAPISetup() tea.Cmd {
	// TODO: Implement actual API setup wizard
	// For now, show a message that this will run r2go2 setup
	return tea.Sequence(
		tea.Printf("\n🔑 Launching API setup wizard...\n"),
		tea.Quit,
	)
}

// testConnection tests the R2 connection
func (m *InstallerModel) testConnection() tea.Cmd {
	// TODO: Implement actual connection test
	return nil
}

// showQuickStart displays the quick start guide
func (m *InstallerModel) showQuickStart() tea.Cmd {
	// TODO: Implement quick start view
	return nil
}

// openDocumentation opens the documentation
func (m *InstallerModel) openDocumentation() tea.Cmd {
	// TODO: Open docs in browser
	return nil
}

func main() {
	// Check terminal size
	if _, err := os.Stat("/dev/tty"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: TUI installer requires a terminal")
		os.Exit(1)
	}

	// Create and run the installer
	model := NewInstallerModel()
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running installer: %v", err)
		os.Exit(1)
	}
}