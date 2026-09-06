/*
Package tui provides the interactive terminal dashboard for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/CosmoLabs-org/cosmoflare/internal/tui/components/palette"
)

// Section represents different sections of the dashboard
type Section int

const (
	SectionOverview Section = iota
	SectionBucketList // Browser (buckets + objects in split-pane)
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
		return "Browser"
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

	// Data source
	data DataSource

	// Data state
	buckets       []Bucket
	currentBucket *Bucket
	usageStats    UsageStats

	// Monitoring state
	metrics      ServiceMetrics
	prevMetrics  ServiceMetrics
	pollInterval time.Duration
	pollPaused   bool
	skipNextPoll bool

	// Browser sub-model (split-pane bucket/object browser)
	browser BrowserModel

	// Input/confirmation state
	inputMode     bool
	inputBuffer   string
	confirmAction string
	confirmTarget string

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

// initialModel creates the initial dashboard model.
// If ds is nil a nullDataSource is used. If interval is zero a 30s default is applied.
func initialModel(ds DataSource, interval time.Duration) DashboardModel {
	if ds == nil {
		ds = &nullDataSource{}
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	theme := darkTheme // Use dark theme by default
	InitializeStyles(theme)

	// Build command palette with default dashboard actions
	paletteCommands := palette.DefaultDashboardActions()
	p := palette.New(paletteCommands, 80, 24)

	return DashboardModel{
		data:            ds,
		pollInterval:    interval,
		buckets:         []Bucket{},
		currentSection:  SectionOverview,
		selectedRow:     0,
		loading:         ds.Available(),
		loadingFrame:    0,
		notifications:   []Notification{},
		backgroundTasks: []BackgroundTask{},
		uploadQueue:     []UploadTask{},
		currentProfile:  "default",
		theme:           theme,
		palette:         p,
		browser:         NewBrowserModel(ds),
	}
}

// Init initializes the dashboard model
func (m DashboardModel) Init() tea.Cmd {
	if !m.data.Available() {
		return nil
	}
	return tea.Batch(
		loadDataCmd(m.data),
		monitoringTick(m.pollInterval),
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

type monitoringTickMsg struct{}

type metricsLoadedMsg struct {
	metrics ServiceMetrics
	err     error
}

type bucketCreatedMsg struct{ err error }

type bucketDeletedMsg struct{ err error }

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

// loadDataCmd loads initial data from the DataSource
func loadDataCmd(ds DataSource) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		buckets, stats, err := ds.FetchBuckets(ctx)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to load data: %w", err)}
		}

		return dataLoadedMsg{
			buckets: buckets,
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
	// Browser section handles its own Enter key via BrowserModel.Update.
	// Other sections can add selection logic here as needed.
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