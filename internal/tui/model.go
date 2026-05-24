/*
Package tui provides the interactive terminal dashboard for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/tui/components/palette"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

// Section represents different sections of the dashboard
type Section int

const (
	SectionOverview Section = iota
	SectionBucketList
	SectionObjectList
	SectionUpload
	SectionMonitoring
	SectionSettings
	SectionHelp
)

// String returns the string representation of a section
func (s Section) String() string {
	switch s {
	case SectionOverview:
		return "Overview"
	case SectionBucketList:
		return "Buckets"
	case SectionObjectList:
		return "Objects"
	case SectionUpload:
		return "Upload"
	case SectionMonitoring:
		return "Monitoring"
	case SectionSettings:
		return "Settings"
	case SectionHelp:
		return "Help"
	default:
		return "Unknown"
	}
}

// Bucket represents an R2 bucket
type Bucket struct {
	Name        string
	Size        int64
	ObjectCount int64
	Status      string
	CreatedAt   time.Time
}

// UsageStats represents storage usage statistics
type UsageStats struct {
	TotalUsed   int64
	TotalLimit  int64
	BucketCount int
}

// RealTimeStats represents real-time activity statistics
type RealTimeStats struct {
	UploadRate       float64
	DownloadRate     float64
	RequestsPerMin   int
	LastUpdate       time.Time
	ActiveUploads    int
	ActiveDownloads  int
}

// Notification represents a system notification
type Notification struct {
	Message   string
	Type      string // "info", "success", "warning", "error"
	Timestamp time.Time
}

// BackgroundTask represents a background operation
type BackgroundTask struct {
	ID          string
	Type        string
	Description string
	Progress    int
	Status      string // "running", "completed", "failed"
}

// UploadTask represents a file upload operation
type UploadTask struct {
	ID          string
	FileName    string
	FilePath    string
	Bucket      string
	Progress    int
	Status      string
	StartTime   time.Time
	Error       error
}

// DashboardModel is the main TUI model
type DashboardModel struct {
	// Navigation state
	currentSection Section
	selectedRow     int
	cursor          int

	// Data state
	buckets       []Bucket
	currentBucket *Bucket
	usageStats    UsageStats
	realTimeStats RealTimeStats

	// UI state
	showHelp       bool
	notifications  []Notification
	searchQuery    string
	sortBy         SortField
	filterActive   bool
	loading        bool
	loadingFrame   int
	width          int
	height         int

	// Background operations
	backgroundTasks []BackgroundTask
	uploadQueue     []UploadTask

	// Settings
	currentProfile string
	theme          Theme

	// Command palette
	palette *palette.PaletteModel
}

// SortField represents sortable fields
type SortField string

const (
	SortByName  SortField = "name"
	SortBySize  SortField = "size"
	SortByCount SortField = "count"
	SortByDate  SortField = "date"
)

// Theme represents color scheme
type Theme struct {
	Primary   lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Muted     lipgloss.Color
	Background lipgloss.Color
	Text      lipgloss.Color
}

// Default themes
var (
	// Dark theme (default)
	darkTheme = Theme{
		Primary:   "#5DADE2", // Light blue
		Success:   "#2ECC71", // Green
		Warning:   "#F39C12", // Orange
		Error:     "#E74C3C", // Red
		Muted:     "#7F8C8D", // Gray
		Background: "#2C3E50", // Dark blue-gray
		Text:       "#ECF0F1", // Light gray
	}

	// Light theme
	lightTheme = Theme{
		Primary:   "#3498DB", // Blue
		Success:   "#2ECC71", // Green
		Warning:   "#F39C12", // Orange
		Error:     "#E74C3C", // Red
		Muted:     "#95A5A6", // Gray
		Background: "#FFFFFF", // White
		Text:       "#2C3E50", // Dark blue-gray
	}
)

// Color scheme variables (will be initialized with theme)
var (
	primaryColor   lipgloss.Color
	successColor   lipgloss.Color
	warningColor   lipgloss.Color
	errorColor     lipgloss.Color
	mutedColor     lipgloss.Color
	backgroundColor lipgloss.Color
	textColor      lipgloss.Color
)

// Styles (will be initialized with theme)
var (
	titleStyle     lipgloss.Style
	headerStyle    lipgloss.Style
	borderStyle    lipgloss.Style
	activeStyle    lipgloss.Style
	tableHeader    lipgloss.Style
	tableRow       lipgloss.Style
	tableRowActive lipgloss.Style
	statusActive   lipgloss.Style
	statusInactive lipgloss.Style
	progressBar    lipgloss.Style
)

// InitializeStyles sets up the visual styles based on the current theme
func InitializeStyles(theme Theme) {
	primaryColor = theme.Primary
	successColor = theme.Success
	warningColor = theme.Warning
	errorColor = theme.Error
	mutedColor = theme.Muted
	backgroundColor = theme.Background
	textColor = theme.Text

	// Define styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	activeStyle = lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	tableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#BDC3C7")).
		Padding(0, 1)

	tableRow = lipgloss.NewStyle().
		Padding(0, 1)

	tableRowActive = lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	statusActive = lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true)

	statusInactive = lipgloss.NewStyle().
		Foreground(mutedColor)

	progressBar = lipgloss.NewStyle().
		Foreground(primaryColor).
		Background(lipgloss.Color("#34495E"))
}

// initialModel creates the initial dashboard model
func initialModel() DashboardModel {
	theme := darkTheme // Use dark theme by default
	InitializeStyles(theme)

	// Build command palette with default dashboard actions
	paletteCommands := palette.DefaultDashboardActions()
	p := palette.New(paletteCommands, 80, 24)

	return DashboardModel{
		buckets:         []Bucket{},
		currentSection:  SectionOverview,
		selectedRow:     0,
		loading:         true,
		loadingFrame:    0,
		notifications:   []Notification{},
		backgroundTasks: []BackgroundTask{},
		uploadQueue:     []UploadTask{},
		currentProfile:  "default",
		theme:           theme,
		palette:         p,
		realTimeStats: RealTimeStats{
			LastUpdate: time.Now(),
		},
	}
}

// Init initializes the dashboard model
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		loadDataCmd(),
		tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return realTimeUpdateMsg{timestamp: t}
		}),
	)
}

// Messages for tea.Cmd

type dataLoadedMsg struct {
	buckets []Bucket
	stats   UsageStats
	err     error
}

type errorMsg struct {
	err error
}

type realTimeUpdateMsg struct {
	timestamp time.Time
}

type bucketSelectedMsg struct {
	bucket *Bucket
}

type uploadProgressMsg struct {
	taskID   string
	progress int
	err      error
}

type notificationMsg struct {
	notification Notification
}

// loadDataCmd loads initial data from the API
func loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		client, err := r2go2.NewClient()
		if err != nil {
			return errorMsg{fmt.Errorf("failed to create API client: %w", err)}
		}

		buckets, err := client.ListBuckets(context.Background())
		if err != nil {
			return errorMsg{fmt.Errorf("failed to list buckets: %w", err)}
		}

		// Convert to TUI bucket format
		var tuiBuckets []Bucket
		for _, bucket := range buckets {
			tuiBuckets = append(tuiBuckets, Bucket{
				Name:        bucket.Name,
				Size:        bucket.Size,
				ObjectCount: bucket.ObjectCount,
				Status:      "active",
				CreatedAt:   bucket.CreatedAt,
			})
		}

		// For now, create mock usage stats (will be enhanced later)
		var totalUsed int64
		for _, bucket := range tuiBuckets {
			totalUsed += bucket.Size
		}

		stats := UsageStats{
			TotalUsed:   totalUsed,
			TotalLimit:  1024 * 1024 * 1024 * 1024, // 1TB (default limit)
			BucketCount: len(tuiBuckets),
		}

		return dataLoadedMsg{
			buckets: tuiBuckets,
			stats:   stats,
		}
	}
}

// Navigation methods

func (m *DashboardModel) moveUp() {
	if m.selectedRow > 0 {
		m.selectedRow--
	}
}

func (m *DashboardModel) moveDown() {
	if m.selectedRow < len(m.buckets)-1 {
		m.selectedRow++
	}
}

func (m *DashboardModel) previousSection() {
	if m.currentSection > 0 {
		m.currentSection--
	} else {
		m.currentSection = SectionHelp // Wrap around
	}
}

func (m *DashboardModel) nextSection() {
	if m.currentSection < SectionHelp {
		m.currentSection++
	} else {
		m.currentSection = SectionOverview // Wrap around
	}
}

func (m *DashboardModel) selectCurrent() tea.Cmd {
	switch m.currentSection {
	case SectionBucketList:
		if m.selectedRow < len(m.buckets) {
			m.currentBucket = &m.buckets[m.selectedRow]
			return func() tea.Msg {
				return bucketSelectedMsg{bucket: m.currentBucket}
			}
		}
	case SectionObjectList:
		// Handle object selection
	}
	return nil
}

func (m *DashboardModel) getCurrentProfile() string {
	return m.currentProfile
}

func (m *DashboardModel) addNotification(message, notificationType string) {
	notification := Notification{
		Message:   message,
		Type:      notificationType,
		Timestamp: time.Now(),
	}
	m.notifications = append([]Notification{notification}, m.notifications...)

	// Keep only last 10 notifications
	if len(m.notifications) > 10 {
		m.notifications = m.notifications[:10]
	}
}