/*
Package installer provides professional installer TUI components

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package installer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// HeaderModel represents the professional installer header
type HeaderModel struct {
	Title       string
	Subtitle    string
	Description string
	Loading     bool
	Progress    int
	MaxProgress int
	Width       int
	styles      *HeaderStyles
	Animation   *HeaderAnimation
}

// HeaderAnimation handles header animations
type HeaderAnimation struct {
	frame     int
	lastTick  time.Time
	frames    []string
	active    bool
}

// HeaderStyles contains styling for the header
type HeaderStyles struct {
	Container   lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Description lipgloss.Style
	Border      lipgloss.Style
	InnerBorder lipgloss.Style
	Icon        lipgloss.Style
	Progress    lipgloss.Style
	Loading     lipgloss.Style
}

// NewHeaderModel creates a new header model
func NewHeaderModel(width int) *HeaderModel {
	return &HeaderModel{
		Title:       "🚀 R2Go2 Professional Installation",
		Subtitle:    "Transform your Cloudflare R2 management experience",
		Description: "💫 One command installation • Beautiful terminal dashboard",
		Loading:     false,
		Progress:    0,
		MaxProgress: 100,
		Width:       width,
		styles:      createDefaultHeaderStyles(width),
		Animation:   createHeaderAnimation(),
	}
}

// createDefaultHeaderStyles creates clean, compact styling for the header
func createDefaultHeaderStyles(width int) *HeaderStyles {
	accent := lipgloss.Color("#00D4AA")
	textPrimary := lipgloss.Color("#FFFFFF")
	textSecondary := lipgloss.Color("#94A3B8")
	textMuted := lipgloss.Color("#64748B")

	return &HeaderStyles{
		Container: lipgloss.NewStyle(), // No border - outer container has it

		Title: lipgloss.NewStyle().
			Foreground(textPrimary).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(accent),

		Description: lipgloss.NewStyle().
			Foreground(textSecondary),

		Border: lipgloss.NewStyle().
			Foreground(textMuted),

		InnerBorder: lipgloss.NewStyle(),

		Icon: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		Progress: lipgloss.NewStyle().
			Foreground(accent),

		Loading: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
	}
}

// createHeaderAnimation creates spinning animation
func createHeaderAnimation() *HeaderAnimation {
	return &HeaderAnimation{
		frame:    0,
		lastTick: time.Now(),
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		active:   false,
	}
}

// Init initializes the header model
func (h *HeaderModel) Init() tea.Cmd {
	h.Animation.lastTick = time.Now()
	return nil
}

// Update handles updates for the header model
func (h *HeaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.Width = msg.Width
	}

	if h.Loading {
		h.updateAnimation()
	}

	return h, nil
}

// updateAnimation updates the loading animation
func (h *HeaderModel) updateAnimation() {
	now := time.Now()
	if now.Sub(h.Animation.lastTick) > time.Millisecond*100 {
		h.Animation.frame = (h.Animation.frame + 1) % len(h.Animation.frames)
		h.Animation.lastTick = now
	}
}

// View renders the header
func (h *HeaderModel) View() string {
	content := h.renderHeaderContent()
	return h.styles.Container.Render(content)
}

// renderHeaderContent renders a clean, compact header
func (h *HeaderModel) renderHeaderContent() string {
	var content []string

	// Title line: icon + title
	titleLine := h.styles.Icon.Render("🚀") + " " + h.styles.Title.Render(h.Title)
	content = append(content, titleLine)

	// Subtitle (if set)
	if h.Subtitle != "" {
		content = append(content, h.styles.Subtitle.Render(h.Subtitle))
	}

	// Description (if set)
	if h.Description != "" {
		content = append(content, h.styles.Description.Render(h.Description))
	}

	// Progress bar (compact inline)
	if h.Progress > 0 && h.MaxProgress > 0 {
		content = append(content, h.renderProgressBar())
	}

	// Loading indicator
	if h.Loading && h.Progress == 0 {
		indicator := h.Animation.frames[h.Animation.frame]
		content = append(content, h.styles.Loading.Render(indicator+" Processing..."))
	}

	return strings.Join(content, "\n")
}

// renderProgressBar renders a gradient progress bar
func (h *HeaderModel) renderProgressBar() string {
	if h.MaxProgress <= 0 {
		return ""
	}

	progress := h.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > h.MaxProgress {
		progress = h.MaxProgress
	}

	percentage := float64(progress) / float64(h.MaxProgress)
	barWidth := 30
	filledWidth := int(float64(barWidth) * percentage)

	// Gradient colors from cyan to green to yellow
	gradientColors := []string{
		"#00D4AA", "#00D4AA", "#00D4AA", "#00D4AA", "#00D4AA", // Cyan
		"#10B981", "#10B981", "#10B981", "#10B981", "#10B981", // Green
		"#34D399", "#34D399", "#34D399", "#34D399", "#34D399", // Light green
		"#6EE7B7", "#6EE7B7", "#6EE7B7", "#6EE7B7", "#6EE7B7", // Lighter green
		"#A7F3D0", "#A7F3D0", "#A7F3D0", "#A7F3D0", "#A7F3D0", // Mint
		"#FCD34D", "#FCD34D", "#FCD34D", "#FCD34D", "#FCD34D", // Yellow/gold
	}

	// Build gradient filled portion
	var filled strings.Builder
	for i := 0; i < filledWidth; i++ {
		colorIdx := i
		if colorIdx >= len(gradientColors) {
			colorIdx = len(gradientColors) - 1
		}
		charStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColors[colorIdx]))
		filled.WriteString(charStyle.Render("█"))
	}

	// Empty portion
	empty := h.styles.Border.Render(strings.Repeat("░", barWidth-filledWidth))
	pct := fmt.Sprintf("%3d%%", int(percentage*100))

	bar := filled.String() + empty

	if h.Progress >= h.MaxProgress {
		checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
		return fmt.Sprintf("%s %s %s", bar, pct, checkStyle.Render("✓"))
	}
	return fmt.Sprintf("%s %s", bar, pct)
}

// SetLoading toggles loading state and animation
func (h *HeaderModel) SetLoading(loading bool) {
	h.Loading = loading
	h.Animation.active = loading
	if loading {
		h.Animation.lastTick = time.Now()
	}
}

// SetProgress updates the progress
func (h *HeaderModel) SetProgress(progress, max int) {
	h.Progress = progress
	h.MaxProgress = max
}

// SetTitle updates the header title
func (h *HeaderModel) SetTitle(title string) {
	h.Title = title
}

// SetSubtitle updates the header subtitle
func (h *HeaderModel) SetSubtitle(subtitle string) {
	h.Subtitle = subtitle
}

// SetDescription updates the header description
func (h *HeaderModel) SetDescription(description string) {
	h.Description = description
}

// SetWidth updates the header width
func (h *HeaderModel) SetWidth(width int) {
	h.Width = width
	h.styles.Container.Width(width)
}

// GetWidth returns the current header width
func (h *HeaderModel) GetWidth() int {
	return h.Width
}

// CreateWelcomeHeader creates a welcome header for the installer
func CreateWelcomeHeader(width int) *HeaderModel {
	header := NewHeaderModel(width)
	header.SetSubtitle("Professional installation and setup wizard")
	header.SetDescription("⚡ Fast setup • Expert configuration • Secure defaults")
	return header
}

// CreateProgressHeader creates a header with progress indicator
func CreateProgressHeader(width int, title, description string) *HeaderModel {
	header := NewHeaderModel(width)
	header.SetTitle(title)
	header.SetDescription(description)
	header.SetLoading(true)
	header.SetProgress(0, 100)
	return header
}

// CreateCompletionHeader creates a header for completion screen
func CreateCompletionHeader(width int) *HeaderModel {
	header := NewHeaderModel(width)
	header.SetTitle("✅ Installation Complete!")
	header.SetSubtitle("R2Go2 is ready to use")
	header.SetDescription("🎉 Thank you for choosing R2Go2 • Manage your Cloudflare R2 with style")
	header.SetProgress(100, 100)
	return header
}