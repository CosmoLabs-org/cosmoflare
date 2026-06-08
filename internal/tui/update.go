/*
Package tui provides the interactive terminal dashboard for Cosmoflare

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CosmoLabs-org/cosmoflare/internal/tui/components/palette"
)

// Update handles incoming messages and updates the model
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.palette != nil {
			m.palette.SetSize(msg.Width, msg.Height)
		}
		m.browser.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// When palette is visible, delegate all keys to it
		if m.palette != nil && m.palette.IsVisible() {
			updated, cmd := m.palette.Update(msg)
			*m.palette = updated
			return m, cmd
		}
		return m.handleKeyMsg(msg)

	case palette.PaletteSelectMsg:
		return m.handlePaletteSelect(msg)

	case palette.PaletteDismissMsg:
		// Palette already hidden itself; no-op
		return m, nil

	case dataLoadedMsg:
		if msg.err != nil {
			m.addNotification("Failed to load data: "+msg.err.Error(), "error")
			m.loading = false
			return m, nil
		}
		m.buckets = msg.buckets
		m.usageStats = msg.stats
		m.loading = false
		m.browser.SetBuckets(msg.buckets)
		m.addNotification("Data loaded successfully", "success")
		return m, nil

	case errorMsg:
		m.addNotification("Error: "+msg.err.Error(), "error")
		m.loading = false
		return m, nil

	case monitoringTickMsg:
		if m.pollPaused || m.skipNextPoll {
			m.skipNextPoll = false
			return m, monitoringTick(m.pollInterval)
		}
		return m, tea.Batch(fetchMetricsCmd(m.data), monitoringTick(m.pollInterval))

	case metricsLoadedMsg:
		if msg.err != nil {
			m.addNotification("Metrics error: "+msg.err.Error(), "error")
			if strings.Contains(msg.err.Error(), "429") {
				m.skipNextPoll = true
			}
			return m, nil
		}
		m.prevMetrics = m.metrics
		m.metrics = msg.metrics
		return m, nil

	case browserObjectsMsg, browserHeadMsg, browserDeleteMsg:
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd

	case bucketCreatedMsg:
		if msg.err != nil {
			m.addNotification("Failed to create bucket: "+msg.err.Error(), "error")
		} else {
			m.addNotification("Bucket created", "success")
		}
		return m, refreshDataCmd(m.data)

	case bucketDeletedMsg:
		if msg.err != nil {
			m.addNotification("Failed to delete bucket: "+msg.err.Error(), "error")
		} else {
			m.addNotification("Bucket deleted", "success")
		}
		return m, refreshDataCmd(m.data)

	case bucketSelectedMsg:
		// Browser handles bucket selection internally via BrowserModel.Update.
		m.currentBucket = msg.bucket
		m.addNotification("Selected bucket: "+msg.bucket.Name, "info")
		return m, nil

	case uploadProgressMsg:
		// Update upload progress
		for i, task := range m.uploadQueue {
			if task.ID == msg.taskID {
				if msg.err != nil {
					m.uploadQueue[i].Status = "failed"
					m.uploadQueue[i].Error = msg.err
					m.addNotification("Upload failed: "+msg.err.Error(), "error")
				} else if msg.progress >= 100 {
					m.uploadQueue[i].Status = "completed"
					m.addNotification("Upload completed: "+task.FileName, "success")
				} else {
					m.uploadQueue[i].Progress = msg.progress
				}
				break
			}
		}
		return m, nil

	case notificationMsg:
		m.notifications = append([]Notification{msg.notification}, m.notifications...)
		if len(m.notifications) > 10 {
			m.notifications = m.notifications[:10]
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg processes keyboard input
func (m DashboardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Input mode: creating a bucket
	if m.inputMode {
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.inputBuffer)
			m.inputMode = false
			m.inputBuffer = ""
			if name == "" {
				return m, nil
			}
			return m, createBucketAPICmd(m.data, name)
		case "esc":
			m.inputMode = false
			m.inputBuffer = ""
			return m, nil
		case "backspace":
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.inputBuffer += msg.String()
			}
			return m, nil
		}
	}

	// Confirmation mode: deleting a bucket
	if m.confirmAction != "" {
		switch msg.String() {
		case "y":
			target := m.confirmTarget
			m.confirmAction = ""
			m.confirmTarget = ""
			return m, deleteBucketAPICmd(m.data, target)
		case "n", "esc":
			m.confirmAction = ""
			m.confirmTarget = ""
			return m, nil
		default:
			return m, nil
		}
	}

	// Handle search mode
	if m.searchQuery != "" {
		switch msg.String() {
		case "enter":
			// Apply search
			m.filterActive = true
			m.searchQuery = ""
			return m, nil
		case "esc":
			// Cancel search
			m.searchQuery = ""
			m.filterActive = false
			return m, nil
		case "backspace":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}
			return m, nil
		default:
			// Add character to search query
			if len(msg.String()) == 1 {
				m.searchQuery += msg.String()
			}
			return m, nil
		}
	}

	// Normal key handling
	switch msg.String() {
	// Command palette
	case "ctrl+p":
		if m.palette != nil {
			m.palette.Show()
		}
		return m, nil

	// Navigation
	case "up", "k":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}
		m.moveUp()
	case "down", "j":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}
		m.moveDown()
	case "left", "h":
		m.previousSection()
	case "right", "l":
		m.nextSection()
	case "enter", " ":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}
		return m, m.selectCurrent()
	case "esc", "q":
		if m.showHelp {
			m.showHelp = false
		} else {
			return m, tea.Quit
		}

	// Refresh
	case "r":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}
		if m.data.Available() {
			if m.currentSection == SectionMonitoring {
				return m, fetchMetricsCmd(m.data)
			}
			return m, refreshDataCmd(m.data)
		}

	// Poll toggle / browser previous page
	case "p":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}
		if m.currentSection == SectionMonitoring {
			m.pollPaused = !m.pollPaused
			return m, nil
		}

	// Next page (browser handles its own)
	case "n":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}

	// Tab — browser focus toggle
	case "tab":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}

	// Create bucket
	case "c":
		if m.currentSection == SectionBucketList && m.data.Available() {
			m.inputMode = true
			m.inputBuffer = ""
			return m, nil
		}

	// Delete — browser handles object delete, dashboard handles bucket delete
	case "d":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}

	// Upload (placeholder)
	case "u":
		m.addNotification("Upload file functionality coming soon", "info")

	// Monitoring mode shortcut
	case "m":
		m.currentSection = SectionMonitoring

	// Settings
	case "s":
		m.currentSection = SectionSettings

	// Backspace: browser handles back navigation
	case "backspace":
		if m.currentSection == SectionBucketList {
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		}

	// Function keys
	case "f1":
		m.showHelp = !m.showHelp
	case "f5":
		if m.data.Available() {
			return m, refreshDataCmd(m.data)
		}
	case "f10":
		m.currentSection = SectionSettings

	// Search
	case "/":
		m.searchQuery = ""
		return m, nil

	// Help and quit
	case "?":
		m.showHelp = true

	// Section-specific shortcuts (with refresh on section change)
	case "1":
		prev := m.currentSection
		m.currentSection = SectionOverview
		if prev != SectionOverview && m.data.Available() {
			return m, refreshDataCmd(m.data)
		}
	case "2":
		prev := m.currentSection
		m.currentSection = SectionBucketList
		if prev != SectionBucketList && m.data.Available() {
			return m, refreshDataCmd(m.data)
		}
	case "3":
		m.currentSection = SectionUpload
	case "4":
		prev := m.currentSection
		m.currentSection = SectionMonitoring
		if prev != SectionMonitoring && m.data.Available() {
			return m, fetchMetricsCmd(m.data)
		}
	case "5":
		m.currentSection = SectionSettings
	}

	return m, nil
}

// Command functions

func monitoringTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg { return monitoringTickMsg{} })
}

func fetchMetricsCmd(ds DataSource) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		metrics, err := ds.FetchMetrics(ctx)
		return metricsLoadedMsg{metrics: metrics, err: err}
	}
}

func createBucketAPICmd(ds DataSource, name string) tea.Cmd {
	return func() tea.Msg {
		return bucketCreatedMsg{err: ds.CreateBucket(context.Background(), name)}
	}
}

func deleteBucketAPICmd(ds DataSource, name string) tea.Cmd {
	return func() tea.Msg {
		return bucketDeletedMsg{err: ds.DeleteBucket(context.Background(), name)}
	}
}

func refreshDataCmd(ds DataSource) tea.Cmd {
	return loadDataCmd(ds)
}

// handlePaletteSelect processes a command selected from the palette.
func (m DashboardModel) handlePaletteSelect(msg palette.PaletteSelectMsg) (tea.Model, tea.Cmd) {
	cmd := msg.Command

	// If the command has a custom Action, execute it
	if cmd.Action != nil {
		return m, cmd.Action()
	}

	// Handle built-in dashboard actions by name
	switch cmd.Name {
	case "Go to Overview":
		m.currentSection = SectionOverview
	case "Go to Buckets", "Go to Browser":
		m.currentSection = SectionBucketList
	case "Go to Upload":
		m.currentSection = SectionUpload
	case "Go to Monitoring":
		m.currentSection = SectionMonitoring
	case "Go to Settings":
		m.currentSection = SectionSettings
	case "Toggle Help":
		m.showHelp = !m.showHelp
	case "Refresh Data":
		return m, refreshDataCmd(m.data)
	case "Quit":
		return m, tea.Quit
	default:
		m.addNotification("Command: "+cmd.Name, "info")
	}

	return m, nil
}