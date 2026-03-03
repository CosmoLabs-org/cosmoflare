/*
Package tui provides the interactive terminal dashboard for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"fmt"
	"strings"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/charmbracelet/lipgloss"
)

// View renders the dashboard
func (m DashboardModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	// Main layout
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		m.renderMainContent(),
		m.renderFooter(),
	)

	// Center content if window is large enough
	if m.width > 80 {
		content = lipgloss.NewStyle().
			Width(m.width - 4).
			Padding(0, 2).
			Render(content)
	}

	return content
}

// renderHeader renders the dashboard header
func (m DashboardModel) renderHeader() string {
	title := titleStyle.Render(fmt.Sprintf("🎯 R2Go2 Dashboard - %s", m.getCurrentProfile()))

	// Add help indicator
	helpIndicator := ""
	if !m.showHelp {
		helpIndicator = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(" [F1] Help")
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, title, helpIndicator)
}

// renderMainContent renders the main content area
func (m DashboardModel) renderMainContent() string {
	if m.loading {
		return m.renderLoading()
	}

	switch m.currentSection {
	case SectionOverview:
		return m.renderOverview()
	case SectionBucketList:
		return m.renderBucketTable()
	case SectionObjectList:
		return m.renderObjectList()
	case SectionUpload:
		return m.renderUploadInterface()
	case SectionMonitoring:
		return m.renderMonitoring()
	case SectionSettings:
		return m.renderSettings()
	default:
		return m.renderOverview()
	}
}

// renderFooter renders the dashboard footer
func (m DashboardModel) renderFooter() string {
	// Status line
	statusText := fmt.Sprintf("📍 %s | Last Update: %s",
		m.getStatusText(),
		m.realTimeStats.LastUpdate.Format("2006-01-02 15:04:05"))

	// Navigation hint
	navHint := "Use Arrow Keys + Enter"

	// Search indicator
	searchIndicator := ""
	if m.searchQuery != "" {
		searchIndicator = fmt.Sprintf(" | Search: %s", m.searchQuery)
	} else if m.filterActive {
		searchIndicator = " | Filter Active"
	}

	// Notifications
	notificationText := ""
	if len(m.notifications) > 0 {
		latest := m.notifications[0]
		notificationText = fmt.Sprintf(" | %s %s", m.getNotificationIcon(latest.Type), latest.Message)
	}

	footer := lipgloss.JoinHorizontal(lipgloss.Left, statusText, navHint, searchIndicator, notificationText)

	return lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(footer)
}

// renderLoading renders a loading animation
func (m DashboardModel) renderLoading() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[m.loadingFrame%len(frames)]
	m.loadingFrame++

	loadingText := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(fmt.Sprintf("%s Loading R2Go2 Dashboard...", frame))

	return lipgloss.NewStyle().
		Height(m.height - 4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(loadingText)
}

// renderOverview renders the overview dashboard
func (m DashboardModel) renderOverview() string {
	var content strings.Builder

	// Usage statistics
	content.WriteString(m.renderUsageStats())
	content.WriteString("\n\n")

	// Bucket table
	content.WriteString(m.renderBucketTable())
	content.WriteString("\n\n")

	// Quick actions
	content.WriteString(m.renderQuickActions())

	return content.String()
}

// renderUsageStats renders storage usage statistics
func (m DashboardModel) renderUsageStats() string {
	percentage := float64(m.usageStats.TotalUsed) / float64(m.usageStats.TotalLimit) * 100

	title := headerStyle.Render("📊 Storage Usage")
	usageBar := m.createProgressBar(percentage, 100, "")

	usageText := fmt.Sprintf("%.1f GB / %.1f TB (%.1f%%)",
		float64(m.usageStats.TotalUsed)/(1024*1024*1024),
		float64(m.usageStats.TotalLimit)/(1024*1024*1024*1024),
		percentage)

	bucketText := fmt.Sprintf("🪣 Buckets: %d", m.usageStats.BucketCount)

	stats := lipgloss.JoinVertical(
		lipgloss.Left,
		usageBar,
		lipgloss.NewStyle().Foreground(textColor).Render(usageText),
		lipgloss.NewStyle().Foreground(textColor).Render(bucketText),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(stats),
	)
}

// renderBucketTable renders the bucket list table
func (m DashboardModel) renderBucketTable() string {
	if len(m.buckets) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("🚫 No buckets found. Press 'C' to create your first bucket.")
	}

	title := headerStyle.Render("🪣 Bucket Overview")

	// Table header
	headers := []string{"Bucket Name", "Size", "Objects", "Status"}
	headerRow := m.renderTableRow(headers, true)

	// Table rows
	var rows []string
	for i, bucket := range m.buckets {
		isSelected := i == m.selectedRow && m.currentSection == SectionBucketList
		rowData := []string{
			"🪣 " + bucket.Name,
			utils.FormatBytes(bucket.Size),
			formatNumber(bucket.ObjectCount),
			m.getStatusIcon(bucket.Status),
		}
		rows = append(rows, m.renderTableRow(rowData, isSelected))
	}

	// Combine with table borders
	tableContent := lipgloss.JoinVertical(lipgloss.Left, headerRow, strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(tableContent),
	)
}

// renderObjectList renders the object list for current bucket
func (m DashboardModel) renderObjectList() string {
	if m.currentBucket == nil {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("No bucket selected. Navigate to bucket list and select a bucket.")
	}

	title := headerStyle.Render(fmt.Sprintf("📁 Objects in %s", m.currentBucket.Name))
	content := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("🚫 Object listing functionality coming soon.")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content),
	)
}

// renderUploadInterface renders the file upload interface
func (m DashboardModel) renderUploadInterface() string {
	title := headerStyle.Render("📤 File Upload")

	var content strings.Builder

	// Upload queue
	content.WriteString(m.renderUploadQueue())
	content.WriteString("\n\n")

	// File selector
	content.WriteString(m.renderFileSelector())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content.String()),
	)
}

// renderMonitoring renders the monitoring dashboard
func (m DashboardModel) renderMonitoring() string {
	title := headerStyle.Render("📈 Real-Time Monitoring")

	var content strings.Builder

	// Real-time stats
	content.WriteString(m.renderRealTimeStats())
	content.WriteString("\n\n")

	// Activity feed
	content.WriteString(m.renderActivityFeed())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content.String()),
	)
}

// renderSettings renders the settings interface
func (m DashboardModel) renderSettings() string {
	title := headerStyle.Render("⚙️ Settings")

	content := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("🚫 Settings functionality coming soon.")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content),
	)
}

// renderQuickActions renders the quick actions section
func (m DashboardModel) renderQuickActions() string {
	title := headerStyle.Render("🎯 Quick Actions")

	actions := []string{
		"[C] Create Bucket     [U] Upload Files      [M] Monitor Mode",
		"[D] Delete Bucket     [S] Settings          [P] Profiles",
		"[L] List Objects      [A] Analytics         [Q] Quit",
	}

	actionsText := lipgloss.JoinVertical(lipgloss.Left, actions...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(actionsText),
	)
}

// renderHelp renders the help screen
func (m DashboardModel) renderHelp() string {
	title := titleStyle.Render("📚 R2Go2 Dashboard Help")

	helpContent := []string{
		"",
		"Navigation:",
		"  ↑/k, ↓/j        Move up/down in lists",
		"  ←/h, →/l        Switch between sections",
		"  Enter, Space     Select current item",
		"  Esc, q           Go back or quit",
		"  1-6              Jump to sections (1=Overview, 6=Settings)",
		"",
		"Quick Actions:",
		"  C                Create new bucket",
		"  U                Upload files",
		"  D                Delete selected bucket",
		"  M                Start monitoring mode",
		"  S                Open settings",
		"  P                Profile management",
		"  L                List objects in bucket",
		"  A                Analytics dashboard",
		"",
		"Search & Filter:",
		"  /                Start search",
		"  Esc              Cancel search",
		"  Enter            Apply search filter",
		"",
		"Function Keys:",
		"  F1               Toggle this help",
		"  F5               Refresh data",
		"  F10              Open settings",
		"",
		"Sections:",
		"  1. Overview      Dashboard summary and quick actions",
		"  2. Buckets       List and manage R2 buckets",
		"  3. Objects       Browse bucket contents",
		"  4. Upload        Upload files with progress tracking",
		"  5. Monitoring    Real-time statistics and activity",
		"  6. Settings      Configuration and preferences",
		"",
		"Press Esc, q, or F1 to return to dashboard",
	}

	content := lipgloss.JoinVertical(lipgloss.Left, helpContent...)

	return lipgloss.NewStyle().
		Padding(2, 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderUploadQueue renders the file upload queue
func (m DashboardModel) renderUploadQueue() string {
	if len(m.uploadQueue) == 0 {
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("📁 No files in upload queue. Press 'A' to add files.")
	}

	title := headerStyle.Render("📤 Upload Queue")

	var rows []string
	for _, upload := range m.uploadQueue {
		status := m.getUploadStatus(upload)
		progress := m.createProgressBar(float64(upload.Progress), 100, status.Icon)

		row := fmt.Sprintf("%s %s", progress, upload.FileName)
		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content),
	)
}

// renderFileSelector renders the file selection interface
func (m DashboardModel) renderFileSelector() string {
	title := headerStyle.Render("📁 File Selector")

	content := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("🚫 File selection functionality coming soon.")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content),
	)
}

// renderRealTimeStats renders real-time statistics
func (m DashboardModel) renderRealTimeStats() string {
	title := headerStyle.Render("🔥 Real-Time Activity")

	stats := m.realTimeStats

	// Create progress bars for upload/download rates
	uploadBar := m.createProgressBar(stats.UploadRate, 100, "⬆️")
	downloadBar := m.createProgressBar(stats.DownloadRate, 100, "⬇️")

	// Request rate indicator
	requestRate := fmt.Sprintf("🔄 %d req/min", stats.RequestsPerMin)

	statsContent := lipgloss.JoinVertical(
		lipgloss.Left,
		uploadBar,
		downloadBar,
		requestRate,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(statsContent),
	)
}

// renderActivityFeed renders the activity feed
func (m DashboardModel) renderActivityFeed() string {
	title := headerStyle.Render("📝 Activity Feed")

	if len(m.notifications) == 0 {
		content := lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("No recent activity.")
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			lipgloss.NewStyle().MarginTop(1).Render(content),
		)
	}

	var activities []string
	for _, notification := range m.notifications {
		activity := fmt.Sprintf("%s %s",
			m.getNotificationIcon(notification.Type),
			notification.Message)
		activities = append(activities, activity)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, activities...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.NewStyle().MarginTop(1).Render(content),
	)
}

// Helper rendering functions

func (m DashboardModel) renderTableRow(data []string, isHeader bool) string {
	var style lipgloss.Style
	if isHeader {
		style = tableHeader
	} else {
		style = tableRow
	}

	// Calculate column widths based on window size
	widths := m.calculateColumnWidths(len(data))

	var cells []string
	for i, cell := range data {
		cellStyle := style.Width(widths[i])
		cells = append(cells, cellStyle.Render(cell))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (m DashboardModel) calculateColumnWidths(numCols int) []int {
	if m.width <= 0 {
		// Default widths
		return []int{20, 12, 10, 10}
	}

	availableWidth := m.width - 4 // Account for padding
	baseWidth := availableWidth / numCols

	// Specific column allocations
	switch numCols {
	case 4:
		return []int{baseWidth + 10, baseWidth - 5, baseWidth - 5, baseWidth}
	default:
		widths := make([]int, numCols)
		for i := range widths {
			widths[i] = baseWidth
		}
		return widths
	}
}

func (m DashboardModel) createProgressBar(percentage, max float64, prefix string) string {
	if max <= 0 {
		max = 100
	}

	ratio := percentage / max
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	width := 20
	filled := int(float64(width) * ratio)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	text := fmt.Sprintf("%s %s %.1f%%", prefix, bar, ratio*100)

	return progressBar.Render(text)
}

// Status and formatting helpers

func (m DashboardModel) getStatusText() string {
	if m.loading {
		return "Loading..."
	}
	return "Connected"
}

func (m DashboardModel) getStatusIcon(status string) string {
	switch status {
	case "active":
		return statusActive.Render("✅ Active")
	case "archived":
		return "🗄️ Archived"
	case "disabled":
		return "❌ Disabled"
	default:
		return statusInactive.Render("❓ Unknown")
	}
}

func (m DashboardModel) getNotificationIcon(notificationType string) string {
	switch notificationType {
	case "success":
		return "✅"
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "📢"
	}
}

func (m DashboardModel) getUploadStatus(upload UploadTask) struct {
	Icon string
	Text string
} {
	switch upload.Status {
	case "uploading":
		return struct {
			Icon string
			Text string
		}{Icon: "⬆️", Text: "Uploading"}
	case "completed":
		return struct {
			Icon string
			Text string
		}{Icon: "✅", Text: "Completed"}
	case "failed":
		return struct {
			Icon string
			Text string
		}{Icon: "❌", Text: "Failed"}
	case "queued":
		return struct {
			Icon string
			Text string
		}{Icon: "⏳", Text: "Queued"}
	default:
		return struct {
			Icon string
			Text string
		}{Icon: "❓", Text: "Unknown"}
	}
}

// Utility functions

func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	} else if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else {
		return fmt.Sprintf("%.1fB", float64(n)/1000000000)
	}
}