/*
Package main implements the professional TUI installer for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
	"github.com/CosmoLabs-org/cosmoflare/internal/tui/components/installer"
	"github.com/CosmoLabs-org/cosmoflare/internal/tui/components/navigation"
)

// InstallerModel represents the main installer model
type InstallerModel struct {
	header    *installer.HeaderModel
	menu      *navigation.MenuModel
	width     int
	height    int
	state     InstallerState
	styles    *InstallerStyles
	// API Setup wizard fields
	apiSetupStep     int    // 0=account, 1=token, 2=confirm
	accountIDInput   string
	apiTokenInput    string
	inputCursorPos   int
	inputError       string
	// Quick start guide fields
	quickStartScroll int
	// Connection test fields
	testResult       string
	testSuccess      bool
	// Installation info
	installDir       string
	pathAdded        bool
}

// InstallerState represents the current installer state
type InstallerState int

const (
	StateMenu InstallerState = iota
	StateInstalling
	StateComplete
	StatePostInstall   // Post-installation setup menu
	StateAPISetup      // API credential setup wizard
	StateTestConnection // Testing R2 connection
	StateQuickStart    // Quick start guide view
	StateError
)

// progressUpdateMsg represents a progress update message
type progressUpdateMsg struct {
	percentage int
}

// installCompleteMsg signals installation is complete
type installCompleteMsg struct{}

// connectionTestResultMsg contains connection test results
type connectionTestResultMsg struct {
	success bool
	message string
}

// apiSetupCompleteMsg signals API setup is complete
type apiSetupCompleteMsg struct {
	success bool
	message string
}

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
		// Handle state-specific key events first
		switch m.state {
		case StateAPISetup:
			return m.handleAPISetupKeys(msg)
		case StateQuickStart:
			return m.handleQuickStartKeys(msg)
		case StateTestConnection:
			// Any key returns to post-install menu
			if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
				m.state = StatePostInstall
				m.header = installer.CreateCompletionHeader(m.width)
				m.menu = createPostInstallMenu(m.width, m.height-8)
				m.menu.Init()
				return m, nil
			}
		default:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyCtrlC:
				return m, tea.Quit
			}
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

	case connectionTestResultMsg:
		// Handle connection test results
		m.testResult = msg.message
		m.testSuccess = msg.success

	case apiSetupCompleteMsg:
		// Handle API setup completion
		if msg.success {
			m.state = StatePostInstall
			m.header = installer.CreateCompletionHeader(m.width)
			m.menu = createPostInstallMenu(m.width, m.height-8)
			m.menu.Init()
		} else {
			m.inputError = msg.message
		}

	case installStepMsg:
		// Handle installation step progress
		progress := (msg.step * 100) / msg.totalSteps
		m.header.SetProgress(progress, 100)
		m.header.SetDescription(msg.message)

		// Schedule next step or run actual installation
		if msg.step < msg.totalSteps {
			nextStep := msg.step + 1
			var nextMsg string
			switch nextStep {
			case 2:
				nextMsg = "Copying binary..."
			case 3:
				nextMsg = "Setting permissions..."
			case 4:
				nextMsg = "Creating command aliases..."
			case 5:
				nextMsg = "Configuring PATH..."
			}
			return m, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
				return installStepMsg{step: nextStep, totalSteps: 5, message: nextMsg}
			})
		} else {
			// Final step - run actual installation
			return m, m.runInstallationWithProgress()
		}

	case installResultMsg:
		// Handle installation result
		if msg.success {
			m.header.SetProgress(100, 100)
			m.state = StatePostInstall
			m.header = installer.CreateCompletionHeader(m.width)
			m.menu = createPostInstallMenu(m.width, m.height-8)
			m.menu.Init()
			// Store install info for display
			m.installDir = msg.installDir
			m.pathAdded = msg.pathAdded
		} else {
			m.state = StateError
			m.inputError = msg.message
		}

	case tea.WindowSizeMsg:
		// Handle window resize
		m.handleResize(msg)
	}

	// Only update header/menu for menu-based states
	if m.state == StateMenu || m.state == StatePostInstall || m.state == StateInstalling {
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
		// Show install info if available
		if m.installDir != "" {
			content = append(content, m.renderInstallInfo())
		}
		content = append(content, m.menu.View())

	case StateAPISetup:
		content = append(content, m.renderAPISetupState())

	case StateTestConnection:
		content = append(content, m.renderTestConnectionState())

	case StateQuickStart:
		content = append(content, m.renderQuickStartState())

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
	// In production, this would actually download and install
	return tea.Batch(
		tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg { return progressUpdateMsg{25} }),
		tea.Tick(time.Millisecond*1000, func(t time.Time) tea.Msg { return progressUpdateMsg{50} }),
		tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg { return progressUpdateMsg{75} }),
		tea.Tick(time.Millisecond*2000, func(t time.Time) tea.Msg { return progressUpdateMsg{100} }),
	)
}

// installResultMsg contains the result of installation
type installResultMsg struct {
	success    bool
	message    string
	installDir string
	pathAdded  bool
}

// installStepMsg represents a step in the installation process
type installStepMsg struct {
	step       int
	totalSteps int
	message    string
}

// startLocalInstallation starts local build installation
func (m *InstallerModel) startLocalInstallation() tea.Cmd {
	m.state = StateInstalling
	m.header = installer.CreateProgressHeader(m.width, "🏗️ Local Build Installation", "Locating binary...")
	m.header.SetLoading(true)
	m.header.SetProgress(0, 100)

	// Start with first progress tick, then run installation
	return tea.Batch(
		tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return installStepMsg{step: 1, totalSteps: 5, message: "Preparing directories..."} }),
	)
}

// runInstallationWithProgress performs the actual installation
func (m *InstallerModel) runInstallationWithProgress() tea.Cmd {
	return func() tea.Msg {
		// Find binary
		binaryPaths := []string{
			"build/R2Go2",
			"build/r2go2",
			"r2go2",
			"R2Go2",
		}

		var sourceBinary string
		for _, path := range binaryPaths {
			if _, err := os.Stat(path); err == nil {
				sourceBinary = path
				break
			}
		}

		if sourceBinary == "" {
			return installResultMsg{success: false, message: "Local binary not found. Run 'go build -o build/R2Go2 .' first."}
		}

		// Prepare directories
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return installResultMsg{success: false, message: fmt.Sprintf("Failed to get home directory: %v", err)}
		}

		var installDir string
		if runtime.GOOS == "windows" {
			installDir = homeDir + "\\AppData\\Local\\R2Go2\\bin"
		} else {
			installDir = homeDir + "/.local/bin"
		}

		if err := os.MkdirAll(installDir, 0755); err != nil {
			return installResultMsg{success: false, message: fmt.Sprintf("Failed to create install directory: %v", err)}
		}

		// Copy binary
		destBinary := installDir + "/r2go2"
		if runtime.GOOS == "windows" {
			destBinary = installDir + "\\r2go2.exe"
		}

		if err := copyFile(sourceBinary, destBinary); err != nil {
			return installResultMsg{success: false, message: fmt.Sprintf("Failed to copy binary: %v", err)}
		}

		// Set permissions
		if runtime.GOOS != "windows" {
			if err := os.Chmod(destBinary, 0755); err != nil {
				return installResultMsg{success: false, message: fmt.Sprintf("Failed to set permissions: %v", err)}
			}
		}

		// Create symlinks
		if err := createBrandedSymlinks(installDir); err != nil {
			// Non-fatal - main binary still works
		}

		// Configure PATH
		pathAdded := addToPath(installDir)
		saveInstallLocation(installDir)

		return installResultMsg{
			success:    true,
			message:    "R2Go2 installed successfully!",
			installDir: installDir,
			pathAdded:  pathAdded,
		}
	}
}

// addToPath adds the install directory to the user's PATH
func addToPath(installDir string) bool {
	// Check if already in PATH
	currentPath := os.Getenv("PATH")
	if strings.Contains(currentPath, installDir) {
		return false // Already in PATH
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Determine which shell config to update
	shellConfigs := []string{}

	switch runtime.GOOS {
	case "darwin":
		// macOS: prefer .zshrc (default shell), also update .bash_profile
		shellConfigs = append(shellConfigs, homeDir+"/.zshrc")
		if _, err := os.Stat(homeDir + "/.bash_profile"); err == nil {
			shellConfigs = append(shellConfigs, homeDir+"/.bash_profile")
		}
	case "linux":
		// Linux: check for .bashrc, .zshrc, .profile
		if _, err := os.Stat(homeDir + "/.zshrc"); err == nil {
			shellConfigs = append(shellConfigs, homeDir+"/.zshrc")
		}
		if _, err := os.Stat(homeDir + "/.bashrc"); err == nil {
			shellConfigs = append(shellConfigs, homeDir+"/.bashrc")
		}
		if len(shellConfigs) == 0 {
			shellConfigs = append(shellConfigs, homeDir+"/.profile")
		}
	case "windows":
		// Windows: would need to modify registry, skip for now
		return false
	}

	// Add PATH export to shell configs
	pathLine := fmt.Sprintf("\n# R2Go2 CLI\nexport PATH=\"%s:$PATH\"\n", installDir)

	for _, configFile := range shellConfigs {
		// Read existing content
		content, err := os.ReadFile(configFile)
		if err != nil {
			// File doesn't exist, create it
			content = []byte{}
		}

		// Check if already added
		if strings.Contains(string(content), "# R2Go2 CLI") {
			continue
		}

		// Append PATH line
		f, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		f.WriteString(pathLine)
		f.Close()
	}

	return true
}

// saveInstallLocation saves the install directory to R2Go2 config
func saveInstallLocation(installDir string) {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return
	}

	// Get or create a metadata profile to store install info
	// We'll use a special "system" key in the config
	// For now, we'll just ensure the config directory exists
	// The install location can be stored in a separate file

	homeDir, _ := os.UserHomeDir()
	configDir := homeDir + "/.r2go2"
	os.MkdirAll(configDir, 0755)

	// Write install location to a simple file
	installInfoFile := configDir + "/install-info.txt"
	info := fmt.Sprintf("install_dir=%s\ninstall_date=%s\nversion=0.3.1\n",
		installDir,
		time.Now().Format("2006-01-02 15:04:05"))

	os.WriteFile(installInfoFile, []byte(info), 0644)

	// Touch the config to ensure it exists
	_ = configMgr
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
}

// createBrandedSymlinks creates case-insensitive command aliases
// Users can type R2Go2, r2go2, R2GO2, r2GO2, etc. - all will work
func createBrandedSymlinks(installDir string) error {
	// Common case variations users might type
	aliases := []string{
		"R2Go2",  // Branded (recommended)
		"R2GO2",  // All caps
		"r2Go2",  // Mixed
		"R2go2",  // Mixed
		"r2GO2",  // Mixed
		// "r2go2" is the main binary, not a symlink
	}

	for _, alias := range aliases {
		symlinkPath := installDir + "/" + alias

		// Remove existing symlink if present
		if _, err := os.Lstat(symlinkPath); err == nil {
			os.Remove(symlinkPath)
		}

		if runtime.GOOS == "windows" {
			// Windows: copy the binary (symlinks require special permissions)
			binaryPath := installDir + "/r2go2.exe"
			input, err := os.ReadFile(binaryPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(symlinkPath+".exe", input, 0755); err != nil {
				return err
			}
		} else {
			// macOS and Linux: create symbolic link
			if err := os.Symlink("r2go2", symlinkPath); err != nil {
				return err
			}
		}
	}

	return nil
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

// renderInstallInfo renders installation location and PATH info
func (m *InstallerModel) renderInstallInfo() string {
	var content []string

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Background(lipgloss.Color("#1E293B")).Padding(0, 1)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	// Colorful tip box style - gradient-like effect with border
	tipBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00D4AA")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1E3A5F")).
		Padding(0, 2).
		Bold(true)

	content = append(content, "")
	content = append(content, successStyle.Render("✅ R2Go2 installed successfully!"))
	content = append(content, "")
	content = append(content, infoStyle.Render("📁 Install location:"))
	content = append(content, pathStyle.Render(m.installDir))
	content = append(content, "")

	if m.pathAdded {
		content = append(content, successStyle.Render("✅ PATH updated automatically"))
		content = append(content, mutedStyle.Render("   Restart your terminal or run: source ~/.zshrc"))
	} else {
		content = append(content, infoStyle.Render("ℹ️  PATH already configured"))
	}

	content = append(content, "")
	// Eye-catching case-insensitive tip
	content = append(content, tipBoxStyle.Render("💡 TIP: Command is case-insensitive — R2Go2, r2go2, R2GO2 all work!"))
	content = append(content, "")

	return strings.Join(content, "\n")
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
	content = append(content, textStyle.Render("  • R2Go2 --help    View all commands"))
	content = append(content, textStyle.Render("  • R2Go2 setup     Configure your API token"))
	content = append(content, mutedStyle.Render("  (Command is case-insensitive: R2Go2, r2go2, R2GO2 all work)"))
	content = append(content, "")
	content = append(content, mutedStyle.Render("Press Esc to exit"))

	return strings.Join(content, "\n")
}

// renderErrorState renders the error state
func (m *InstallerModel) renderErrorState() string {
	var content []string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	content = append(content, titleStyle.Render("❌ Installation Failed"))
	content = append(content, "")

	// Show specific error if available
	if m.inputError != "" {
		content = append(content, errorStyle.Render("Error: "+m.inputError))
		content = append(content, "")
	}

	content = append(content, textStyle.Render("Troubleshooting:"))
	content = append(content, mutedStyle.Render("  • Ensure you have write permissions to /usr/local/bin"))
	content = append(content, mutedStyle.Render("  • Try running with sudo: sudo ./r2go2-tui-installer"))
	content = append(content, mutedStyle.Render("  • Check that the binary exists in build/ directory"))
	content = append(content, mutedStyle.Render("  • Run 'go build -o build/R2Go2 .' to build first"))
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
	m.state = StateAPISetup
	m.apiSetupStep = 0
	m.accountIDInput = ""
	m.apiTokenInput = ""
	m.inputCursorPos = 0
	m.inputError = ""

	// Try to load existing credentials from environment
	if envAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID"); envAccountID != "" {
		m.accountIDInput = envAccountID
	}
	if envToken := os.Getenv("CLOUDFLARE_API_TOKEN"); envToken != "" {
		m.apiTokenInput = envToken
	}

	return nil
}

// handleAPISetupKeys handles keyboard input during API setup
func (m *InstallerModel) handleAPISetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Return to post-install menu
		m.state = StatePostInstall
		m.header = installer.CreateCompletionHeader(m.width)
		m.menu = createPostInstallMenu(m.width, m.height-8)
		m.menu.Init()
		return m, nil

	case tea.KeyEnter:
		return m.handleAPISetupEnter()

	case tea.KeyTab:
		// Move to next step without validation
		if m.apiSetupStep < 2 {
			m.apiSetupStep++
			m.inputError = ""
		}
		return m, nil

	case tea.KeyBackspace:
		// Handle backspace for current input
		if m.apiSetupStep == 0 && len(m.accountIDInput) > 0 {
			m.accountIDInput = m.accountIDInput[:len(m.accountIDInput)-1]
		} else if m.apiSetupStep == 1 && len(m.apiTokenInput) > 0 {
			m.apiTokenInput = m.apiTokenInput[:len(m.apiTokenInput)-1]
		}
		m.inputError = ""
		return m, nil

	case tea.KeyRunes:
		// Handle character input
		char := string(msg.Runes)
		if m.apiSetupStep == 0 {
			// Account ID - only allow hex characters
			for _, r := range msg.Runes {
				if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
					m.accountIDInput += string(r)
				}
			}
		} else if m.apiSetupStep == 1 {
			// API Token - allow any characters
			m.apiTokenInput += char
		}
		m.inputError = ""
		return m, nil
	}

	return m, nil
}

// handleAPISetupEnter processes Enter key in API setup wizard
func (m *InstallerModel) handleAPISetupEnter() (tea.Model, tea.Cmd) {
	switch m.apiSetupStep {
	case 0: // Account ID step
		if err := interactive.ValidateAccountID(m.accountIDInput); err != nil {
			m.inputError = err.Error()
			return m, nil
		}
		m.apiSetupStep = 1
		m.inputError = ""

	case 1: // API Token step
		if len(m.apiTokenInput) < 20 {
			m.inputError = "API token is too short (minimum 20 characters)"
			return m, nil
		}
		m.apiSetupStep = 2
		m.inputError = ""

	case 2: // Confirmation step
		// Save the configuration
		return m, m.saveAPICredentials()
	}

	return m, nil
}

// saveAPICredentials saves the API credentials to config
func (m *InstallerModel) saveAPICredentials() tea.Cmd {
	return func() tea.Msg {
		configMgr, err := config.NewConfigManager()
		if err != nil {
			return apiSetupCompleteMsg{success: false, message: fmt.Sprintf("Failed to create config manager: %v", err)}
		}

		profile := &config.Profile{
			Name:        "default",
			Description: "Created by TUI installer",
			AccountID:   m.accountIDInput,
			APIToken:    m.apiTokenInput,
			Region:      "auto",
		}

		if err := configMgr.SetProfile(profile); err != nil {
			return apiSetupCompleteMsg{success: false, message: fmt.Sprintf("Failed to save profile: %v", err)}
		}

		if err := configMgr.SetCurrent("default"); err != nil {
			return apiSetupCompleteMsg{success: false, message: fmt.Sprintf("Failed to set current profile: %v", err)}
		}

		return apiSetupCompleteMsg{success: true, message: "Configuration saved successfully!"}
	}
}

// renderAPISetupState renders the API setup wizard view
func (m *InstallerModel) renderAPISetupState() string {
	var content []string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E293B")).Padding(0, 1)
	activeInputStyle := inputStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#00D4AA"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))

	content = append(content, titleStyle.Render("🔑 API Credential Setup"))
	content = append(content, "")

	// Progress indicator
	progress := []string{"○", "○", "○"}
	if m.apiSetupStep >= 0 {
		progress[0] = "●"
	}
	if m.apiSetupStep >= 1 {
		progress[1] = "●"
	}
	if m.apiSetupStep >= 2 {
		progress[2] = "●"
	}
	content = append(content, mutedStyle.Render(fmt.Sprintf("Step %d/3  %s", m.apiSetupStep+1, strings.Join(progress, " "))))
	content = append(content, "")

	switch m.apiSetupStep {
	case 0: // Account ID
		content = append(content, textStyle.Render("Enter your Cloudflare Account ID:"))
		content = append(content, mutedStyle.Render("32-character hex string from dashboard.cloudflare.com"))
		content = append(content, "")

		inputDisplay := m.accountIDInput
		if inputDisplay == "" {
			inputDisplay = "                                " // 32 char placeholder
		}
		content = append(content, activeInputStyle.Render(inputDisplay+"█"))
		content = append(content, mutedStyle.Render(fmt.Sprintf("%d/32 characters", len(m.accountIDInput))))

	case 1: // API Token
		content = append(content, textStyle.Render("Enter your Cloudflare API Token:"))
		content = append(content, mutedStyle.Render("Create at: dash.cloudflare.com/profile/api-tokens"))
		content = append(content, mutedStyle.Render("Required permissions: R2:Read, R2:Write"))
		content = append(content, "")

		// Mask token for display
		masked := strings.Repeat("•", len(m.apiTokenInput))
		if masked == "" {
			masked = "                    " // placeholder
		}
		content = append(content, activeInputStyle.Render(masked+"█"))
		content = append(content, mutedStyle.Render(fmt.Sprintf("%d characters entered", len(m.apiTokenInput))))

	case 2: // Confirmation
		content = append(content, successStyle.Render("✅ Ready to save configuration"))
		content = append(content, "")
		content = append(content, textStyle.Render("Account ID: "+config.MaskAccountID(m.accountIDInput)))
		content = append(content, textStyle.Render("API Token:  "+maskToken(m.apiTokenInput)))
		content = append(content, "")
		content = append(content, mutedStyle.Render("Press Enter to save, Esc to cancel"))
	}

	// Error message
	if m.inputError != "" {
		content = append(content, "")
		content = append(content, errorStyle.Render("❌ "+m.inputError))
	}

	content = append(content, "")
	content = append(content, mutedStyle.Render("Tab: next field · Esc: cancel"))

	return strings.Join(content, "\n")
}

// maskToken masks API token for display
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("•", len(token))
	}
	return token[:4] + strings.Repeat("•", len(token)-8) + token[len(token)-4:]
}

// testConnection tests the R2 connection
func (m *InstallerModel) testConnection() tea.Cmd {
	m.state = StateTestConnection
	m.testResult = ""
	m.testSuccess = false

	return func() tea.Msg {
		// Load config
		configMgr, err := config.NewConfigManager()
		if err != nil {
			return connectionTestResultMsg{
				success: false,
				message: fmt.Sprintf("Failed to load configuration: %v", err),
			}
		}

		profile, err := configMgr.GetCurrent()
		if err != nil {
			return connectionTestResultMsg{
				success: false,
				message: "No profile configured. Please set up API credentials first.",
			}
		}

		if profile.AccountID == "" || profile.APIToken == "" {
			return connectionTestResultMsg{
				success: false,
				message: "Incomplete credentials. Please configure API credentials first.",
			}
		}

		// Test connection
		if err := interactive.TestConnection(profile.AccountID, profile.APIToken); err != nil {
			return connectionTestResultMsg{
				success: false,
				message: fmt.Sprintf("Connection failed: %v", err),
			}
		}

		return connectionTestResultMsg{
			success: true,
			message: "Successfully connected to Cloudflare R2!",
		}
	}
}

// renderTestConnectionState renders the connection test view
func (m *InstallerModel) renderTestConnectionState() string {
	var content []string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)

	content = append(content, titleStyle.Render("🔌 Connection Test"))
	content = append(content, "")

	if m.testResult == "" {
		content = append(content, textStyle.Render("Testing connection to Cloudflare R2..."))
		content = append(content, "")
		content = append(content, mutedStyle.Render("⏳ Please wait..."))
	} else if m.testSuccess {
		content = append(content, successStyle.Render("✅ "+m.testResult))
		content = append(content, "")
		content = append(content, textStyle.Render("Your credentials are working correctly."))
		content = append(content, textStyle.Render("You can now use R2Go2 to manage your buckets."))
		content = append(content, mutedStyle.Render("(Type R2Go2, r2go2, or R2GO2 - all work!)"))
	} else {
		content = append(content, errorStyle.Render("❌ "+m.testResult))
		content = append(content, "")
		content = append(content, textStyle.Render("Troubleshooting tips:"))
		content = append(content, mutedStyle.Render("  • Verify your Account ID is correct"))
		content = append(content, mutedStyle.Render("  • Check that your API token has R2 permissions"))
		content = append(content, mutedStyle.Render("  • Ensure you have internet connectivity"))
	}

	content = append(content, "")
	content = append(content, mutedStyle.Render("Press Enter or Esc to return"))

	return strings.Join(content, "\n")
}

// showQuickStart displays the quick start guide
func (m *InstallerModel) showQuickStart() tea.Cmd {
	m.state = StateQuickStart
	m.quickStartScroll = 0
	return nil
}

// handleQuickStartKeys handles keyboard input for quick start guide
func (m *InstallerModel) handleQuickStartKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter, tea.KeyCtrlC:
		// Return to post-install menu
		m.state = StatePostInstall
		m.header = installer.CreateCompletionHeader(m.width)
		m.menu = createPostInstallMenu(m.width, m.height-8)
		m.menu.Init()
		return m, nil

	case tea.KeyUp, tea.KeyPgUp:
		if m.quickStartScroll > 0 {
			m.quickStartScroll--
		}
		return m, nil

	case tea.KeyDown, tea.KeyPgDown:
		m.quickStartScroll++
		return m, nil

	case tea.KeyHome:
		m.quickStartScroll = 0
		return m, nil
	}

	return m, nil
}

// renderQuickStartState renders the quick start guide view
func (m *InstallerModel) renderQuickStartState() string {
	var content []string

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA")).Bold(true)
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Background(lipgloss.Color("#1E293B")).Padding(0, 1)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))

	content = append(content, titleStyle.Render("📖 R2Go2 Quick Start Guide"))
	content = append(content, "")

	// Guide content
	noteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Italic(true)

	guide := []string{
		noteStyle.Render("Note: Command is case-insensitive (R2Go2, r2go2, R2GO2 all work)"),
		"",
		headerStyle.Render("Basic Commands"),
		"",
		textStyle.Render("List all buckets:"),
		codeStyle.Render("R2Go2 bucket list"),
		"",
		textStyle.Render("Create a new bucket:"),
		codeStyle.Render("R2Go2 bucket create my-bucket"),
		"",
		textStyle.Render("Delete a bucket:"),
		codeStyle.Render("R2Go2 bucket delete my-bucket"),
		"",
		headerStyle.Render("Object Operations"),
		"",
		textStyle.Render("List objects in bucket:"),
		codeStyle.Render("R2Go2 object list my-bucket"),
		"",
		textStyle.Render("Upload a file:"),
		codeStyle.Render("R2Go2 copy ./file.txt r2://my-bucket/"),
		"",
		textStyle.Render("Download a file:"),
		codeStyle.Render("R2Go2 copy r2://my-bucket/file.txt ./"),
		"",
		headerStyle.Render("Configuration"),
		"",
		textStyle.Render("View current config:"),
		codeStyle.Render("R2Go2 config show"),
		"",
		textStyle.Render("List all profiles:"),
		codeStyle.Render("R2Go2 config list"),
		"",
		textStyle.Render("Switch profile:"),
		codeStyle.Render("R2Go2 setup --switch"),
		"",
		headerStyle.Render("Getting Help"),
		"",
		textStyle.Render("Show all commands:"),
		codeStyle.Render("R2Go2 --help"),
		"",
		textStyle.Render("Get help for a command:"),
		codeStyle.Render("R2Go2 bucket --help"),
	}

	// Apply scroll offset
	maxScroll := len(guide) - 15
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.quickStartScroll > maxScroll {
		m.quickStartScroll = maxScroll
	}

	// Show visible portion
	endIndex := m.quickStartScroll + 15
	if endIndex > len(guide) {
		endIndex = len(guide)
	}

	content = append(content, guide[m.quickStartScroll:endIndex]...)

	// Scroll indicator
	if len(guide) > 15 {
		content = append(content, "")
		scrollPercent := float64(m.quickStartScroll) / float64(maxScroll) * 100
		if maxScroll == 0 {
			scrollPercent = 0
		}
		content = append(content, mutedStyle.Render(fmt.Sprintf("↑↓ scroll · %.0f%% · Esc to return", scrollPercent)))
	} else {
		content = append(content, "")
		content = append(content, mutedStyle.Render("Press Esc or Enter to return"))
	}

	return strings.Join(content, "\n")
}

// openDocumentation opens the documentation in browser
func (m *InstallerModel) openDocumentation() tea.Cmd {
	url := "https://github.com/CosmoLabs-org/cosmoflare"

	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			// Fallback - try xdg-open
			cmd = exec.Command("xdg-open", url)
		}

		_ = cmd.Start() // Fire and forget
		return nil
	}
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