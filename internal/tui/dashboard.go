/*
Package tui provides the interactive terminal dashboard for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// RunDashboard starts the TUI dashboard
func RunDashboard() error {
	// Handle Ctrl+C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Let Bubble Tea handle the cleanup
	}()

	// Create the initial model
	model := initialModel()

	// Start the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithOutput(os.Stderr), // Use stderr to avoid interfering with stdout JSON output
	)

	// Run the program
	_, err := p.Run()
	return err
}

// RunDashboardWithConfig starts the TUI dashboard with custom configuration
func RunDashboardWithConfig(config DashboardConfig) error {
	// Create the initial model with custom config
	model := initialModel()
	model.currentProfile = config.Profile
	if config.Theme != "" {
		switch config.Theme {
		case "light":
			model.theme = lightTheme
		case "dark":
			model.theme = darkTheme
		}
		InitializeStyles(model.theme)
	}

	// Apply other config options
	if config.Width > 0 {
		model.width = config.Width
	}
	if config.Height > 0 {
		model.height = config.Height
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

// DashboardConfig holds configuration for the TUI dashboard
type DashboardConfig struct {
	Profile string
	Theme   string
	Width   int
	Height  int
	Debug   bool
}