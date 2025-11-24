/*
Package tui provides the interactive terminal dashboard for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and updates the model
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case dataLoadedMsg:
		if msg.err != nil {
			m.addNotification("Failed to load data: "+msg.err.Error(), "error")
			m.loading = false
			return m, nil
		}
		m.buckets = msg.buckets
		m.usageStats = msg.stats
		m.loading = false
		m.addNotification("Data loaded successfully", "success")
		return m, nil

	case errorMsg:
		m.addNotification("Error: "+msg.err.Error(), "error")
		m.loading = false
		return m, nil

	case realTimeUpdateMsg:
		m.realTimeStats.LastUpdate = msg.timestamp
		// Simulate some real-time stats
		m.realTimeStats.UploadRate = 15.3 + float64(msg.timestamp.Second()%10)
		m.realTimeStats.DownloadRate = 8.7 + float64(msg.timestamp.Second()%8)
		m.realTimeStats.RequestsPerMin = 500 + msg.timestamp.Second()%100
		return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return realTimeUpdateMsg{timestamp: t}
		})

	case bucketSelectedMsg:
		m.currentBucket = msg.bucket
		m.currentSection = SectionObjectList
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
	// Handle search mode
	if m.searchQuery != "" {
		switch msg.String() {
		case "enter":
			// Apply search
			m.filterActive = true
			m.searchQuery = ""
			return m, nil
		case "escape":
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
	// Navigation
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "left", "h":
		m.previousSection()
	case "right", "l":
		m.nextSection()
	case "enter", " ":
		return m, m.selectCurrent()
	case "esc", "q":
		if m.showHelp {
			m.showHelp = false
		} else {
			return m, tea.Quit
		}

	// Quick actions
	case "c":
		return m, createBucketCmd()
	case "u":
		return m, uploadFileCmd()
	case "d":
		return m, deleteBucketCmd()
	case "m":
		return m, startMonitoringCmd()
	case "s":
		m.currentSection = SectionSettings
	case "p":
		m.currentSection = SectionSettings

	// Function keys
	case "f1":
		m.showHelp = !m.showHelp
	case "f5":
		return m, refreshDataCmd()
	case "f10":
		m.currentSection = SectionSettings

	// Search
	case "/":
		m.searchQuery = ""
		return m, nil

	// Help and quit
	case "?":
		m.showHelp = true

	// Section-specific shortcuts
	case "1":
		m.currentSection = SectionOverview
	case "2":
		m.currentSection = SectionBucketList
	case "3":
		m.currentSection = SectionObjectList
	case "4":
		m.currentSection = SectionUpload
	case "5":
		m.currentSection = SectionMonitoring
	case "6":
		m.currentSection = SectionSettings
	}

	return m, nil
}

// Command functions

func createBucketCmd() tea.Cmd {
	return func() tea.Msg {
		// For now, just show a notification
		return notificationMsg{
			notification: Notification{
				Message:   "Create bucket functionality coming soon",
				Type:      "info",
				Timestamp: time.Now(),
			},
		}
	}
}

func uploadFileCmd() tea.Cmd {
	return func() tea.Msg {
		// For now, just show a notification
		return notificationMsg{
			notification: Notification{
				Message:   "Upload file functionality coming soon",
				Type:      "info",
				Timestamp: time.Now(),
			},
		}
	}
}

func deleteBucketCmd() tea.Cmd {
	return func() tea.Msg {
		// For now, just show a notification
		return notificationMsg{
			notification: Notification{
				Message:   "Delete bucket functionality coming soon",
				Type:      "warning",
				Timestamp: time.Now(),
			},
		}
	}
}

func startMonitoringCmd() tea.Cmd {
	return func() tea.Msg {
		// For now, just show a notification
		return notificationMsg{
			notification: Notification{
				Message:   "Enhanced monitoring coming soon",
				Type:      "info",
				Timestamp: time.Now(),
			},
		}
	}
}

func refreshDataCmd() tea.Cmd {
	return loadDataCmd()
}