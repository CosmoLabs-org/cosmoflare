/*
Package tui provides the interactive terminal dashboard for Cosmoflare

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// RunDashboard starts the TUI dashboard with a default 30s poll interval.
func RunDashboard() error {
	return RunDashboardWithInterval(30 * time.Second)
}

// RunDashboardWithInterval starts the TUI dashboard with the given poll interval.
func RunDashboardWithInterval(interval time.Duration) error {
	// Handle Ctrl+C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Let Bubble Tea handle the cleanup
	}()

	ds := newDataSource()
	model := initialModel(ds, interval)

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
func RunDashboardWithConfig(cfg DashboardConfig) error {
	ds := newDataSource()
	interval := cfg.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	model := initialModel(ds, interval)
	model.currentProfile = cfg.Profile
	if cfg.Theme != "" {
		switch cfg.Theme {
		case "light":
			model.theme = lightTheme
		case "dark":
			model.theme = darkTheme
		}
		InitializeStyles(model.theme)
	}

	// Apply other config options
	if cfg.Width > 0 {
		model.width = cfg.Width
	}
	if cfg.Height > 0 {
		model.height = cfg.Height
	}
	if cfg.StartSection > 0 {
		model.currentSection = Section(cfg.StartSection)
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
	Profile      string
	Theme        string
	Width        int
	Height       int
	Debug        bool
	Interval     time.Duration
	StartSection int
}