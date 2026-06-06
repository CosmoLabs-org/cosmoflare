/*
Package visual provides real-time R2 upload progress tracking

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package visual

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	bubbleProgress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// R2UploadState represents the state of an R2 upload
type R2UploadState struct {
	FileName        string
	Bucket          string
	ObjectKey       string
	TotalSize       int64
	UploadedBytes   int64
	Speed           float64
	ETA             time.Duration
	StartTime       time.Time
	LastUpdate      time.Time
	Status          UploadStatus
	Error           error
	Progress        float64
	PartsCompleted  int
	TotalParts      int
	UploadID        string
}

// UploadStatus represents the current status
type UploadStatus int

const (
	StatusQueued UploadStatus = iota
	StatusConnecting
	StatusUploading
	StatusPaused
	StatusCompleted
	StatusFailed
	StatusCancelled
)

// String returns the string representation of the status
func (s UploadStatus) String() string {
	switch s {
	case StatusQueued:
		return "Queued"
	case StatusConnecting:
		return "Connecting"
	case StatusUploading:
		return "Uploading"
	case StatusPaused:
		return "Paused"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// R2UploadProgress tracks multiple R2 uploads with rich visual feedback
type R2UploadProgress struct {
	uploads      []*R2UploadState
	activeUpload *R2UploadState
	mu           sync.RWMutex
	theme        *VisualTheme
	ctx          context.Context
	cancel       context.CancelFunc
	program      *tea.Program
}

// NewR2UploadProgress creates a new R2 upload progress tracker
func NewR2UploadProgress() *R2UploadProgress {
	ctx, cancel := context.WithCancel(context.Background())
	return &R2UploadProgress{
		uploads: make([]*R2UploadState, 0),
		theme:   DefaultTheme(),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddUpload adds a new upload to track
func (rup *R2UploadProgress) AddUpload(fileName, bucket, objectKey string, totalSize int64) *R2UploadState {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload := &R2UploadState{
		FileName:   fileName,
		Bucket:     bucket,
		ObjectKey:  objectKey,
		TotalSize:  totalSize,
		Status:     StatusQueued,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	rup.uploads = append(rup.uploads, upload)
	return upload
}

// StartUpload marks an upload as started
func (rup *R2UploadProgress) StartUpload(upload *R2UploadState) {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload.Status = StatusUploading
	upload.StartTime = time.Now()
	upload.LastUpdate = time.Now()
	rup.activeUpload = upload
}

// UpdateProgress updates the progress of an active upload
func (rup *R2UploadProgress) UpdateProgress(upload *R2UploadState, uploadedBytes int64, speed float64) {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload.UploadedBytes = uploadedBytes
	upload.Speed = speed
	upload.LastUpdate = time.Now()

	if upload.TotalSize > 0 {
		upload.Progress = float64(uploadedBytes) / float64(upload.TotalSize) * 100

		// Calculate ETA
		if speed > 0 {
			remaining := float64(upload.TotalSize-uploadedBytes) / (1024 * 1024)
			upload.ETA = time.Duration(remaining/speed) * time.Second
		}
	}

	upload.PartsCompleted = int(float64(uploadedBytes) / float64(upload.TotalSize) * float64(upload.TotalParts))
}

// CompleteUpload marks an upload as completed
func (rup *R2UploadProgress) CompleteUpload(upload *R2UploadState) {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload.Status = StatusCompleted
	upload.Progress = 100
	upload.UploadedBytes = upload.TotalSize
	upload.Speed = float64(upload.TotalSize) / time.Since(upload.StartTime).Seconds() / (1024 * 1024)
	upload.ETA = 0
}

// FailUpload marks an upload as failed
func (rup *R2UploadProgress) FailUpload(upload *R2UploadState, err error) {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload.Status = StatusFailed
	upload.Error = err
}

// CancelUpload cancels an upload
func (rup *R2UploadProgress) CancelUpload(upload *R2UploadState) {
	rup.mu.Lock()
	defer rup.mu.Unlock()

	upload.Status = StatusCancelled
}

// GetActiveUpload returns the currently active upload
func (rup *R2UploadProgress) GetActiveUpload() *R2UploadState {
	rup.mu.RLock()
	defer rup.mu.RUnlock()
	return rup.activeUpload
}

// StartLiveDisplay starts the live progress display
func (rup *R2UploadProgress) StartLiveDisplay() {
	m := NewR2ProgressModel(rup)
	program := tea.NewProgram(m)
	rup.program = program

	go func() {
		if _, err := program.Run(); err != nil {
			fmt.Printf("Error running progress display: %v\n", err)
		}
	}()
}

// StopLiveDisplay stops the live progress display
func (rup *R2UploadProgress) StopLiveDisplay() {
	if rup.program != nil {
		rup.cancel()
		rup.program.Quit()
	}
}

// ShowSimpleProgress displays a simple progress bar in terminal
func (rup *R2UploadProgress) ShowSimpleProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-rup.ctx.Done():
			return
		case <-ticker.C:
			rup.displaySimpleProgress()
		}
	}
}

func (rup *R2UploadProgress) displaySimpleProgress() {
	upload := rup.GetActiveUpload()
	if upload == nil {
		return
	}

	// Create styles
	primaryStyle := lipgloss.NewStyle().Foreground(rup.theme.Primary)
	secondaryStyle := lipgloss.NewStyle().Foreground(rup.theme.Secondary)

	// Clear line and show progress
	fmt.Printf("\r%s %s to %s/%s",
		rup.getStatusIcon(upload.Status),
		primaryStyle.Render(upload.FileName),
		secondaryStyle.Render(upload.Bucket),
		secondaryStyle.Render(upload.ObjectKey))

	if upload.Status == StatusUploading {
		bar := rup.createProgressBar(int(upload.Progress), 50)
		fmt.Printf(" %s %s %.1f MB/s ETA: %s",
			bar,
			utils.FormatBytes(upload.UploadedBytes),
			upload.Speed,
			rup.formatDuration(upload.ETA))
	}
}

func (rup *R2UploadProgress) getStatusIcon(status UploadStatus) string {
	switch status {
	case StatusQueued:
		return "⏳"
	case StatusConnecting:
		return "🔗"
	case StatusUploading:
		return "⬆️"
	case StatusPaused:
		return "⏸️"
	case StatusCompleted:
		return "✅"
	case StatusFailed:
		return "❌"
	case StatusCancelled:
		return "🚫"
	default:
		return "❓"
	}
}

func (rup *R2UploadProgress) createProgressBar(percentage, width int) string {
	primaryStyle := lipgloss.NewStyle().Foreground(rup.theme.Primary)
	secondaryStyle := lipgloss.NewStyle().Foreground(rup.theme.Secondary)

	filled := width * percentage / 100
	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)
	return primaryStyle.Render("["+bar) + secondaryStyle.Render(empty) + primaryStyle.Render("]")
}

func (rup *R2UploadProgress) formatDuration(d time.Duration) string {
	if d <= 0 {
		return "Done"
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), float64(int(d.Seconds())%60))
	}
	return fmt.Sprintf("%.0fh %.0fm", d.Hours(), float64(int(d.Minutes())%60))
}

// R2ProgressModel implements the Bubble Tea model for rich progress display
type R2ProgressModel struct {
	progress *R2UploadProgress
	prog     bubbleProgress.Model
	quitting bool
}

func NewR2ProgressModel(rp *R2UploadProgress) *R2ProgressModel {
	return &R2ProgressModel{
		progress: rp,
		prog:     bubbleProgress.New(bubbleProgress.WithDefaultGradient()),
		quitting: false,
	}
}

func (m *R2ProgressModel) Init() tea.Cmd {
	return nil
}

func (m *R2ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case bubbleProgress.FrameMsg:
		// Handle progress frame updates
		var cmds []tea.Cmd
		cmds = append(cmds, m.prog.SetPercent(m.getActiveProgress()))
		return m, tea.Batch(cmds...)
	}

	// Update progress model
	var cmd tea.Cmd
	progModel, cmd := m.prog.Update(msg)
	m.prog = progModel.(bubbleProgress.Model)
	return m, cmd
}

func (m *R2ProgressModel) View() string {
	if m.quitting {
		return ""
	}

	upload := m.progress.GetActiveUpload()
	if upload == nil {
		return "No active uploads"
	}

	title := lipgloss.NewStyle().
		Foreground(m.progress.theme.Primary).
		Bold(true).
		Render("🚀 R2 Upload Progress")

	var content strings.Builder

	// Header
	content.WriteString(title)
	content.WriteString("\n\n")

	// Current upload info
	content.WriteString(m.formatUploadInfo(upload))

	// Progress bar
	content.WriteString("\n\n")
	content.WriteString(m.prog.View())

	// Statistics
	content.WriteString("\n\n")
	content.WriteString(m.formatStatistics(upload))

	// Controls
	content.WriteString("\n\n")
	content.WriteString(m.formatControls())

	return lipgloss.NewStyle().
		Width(80).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.progress.theme.Primary).
		Padding(1, 2).
		Render(content.String())
}

func (m *R2ProgressModel) getActiveProgress() float64 {
	upload := m.progress.GetActiveUpload()
	if upload == nil {
		return 0
	}
	return upload.Progress
}

func (m *R2ProgressModel) formatUploadInfo(upload *R2UploadState) string {
	status := lipgloss.NewStyle().
		Foreground(m.getStatusColor(upload.Status)).
		Bold(true).
		Render(upload.Status.String())

	file := lipgloss.NewStyle().
		Foreground(m.progress.theme.Foreground).
		Render(upload.FileName)

	location := lipgloss.NewStyle().
		Foreground(m.progress.theme.Secondary).
		Render(fmt.Sprintf("%s/%s", upload.Bucket, upload.ObjectKey))

	return fmt.Sprintf(
		"Status: %s\nFile: %s\nLocation: %s",
		status, file, location,
	)
}

func (m *R2ProgressModel) formatStatistics(upload *R2UploadState) string {
	stats := lipgloss.NewStyle().
		Foreground(m.progress.theme.Foreground)

	parts := lipgloss.NewStyle().
		Foreground(m.progress.theme.Secondary)

	var content strings.Builder

	content.WriteString(stats.Render(fmt.Sprintf(
		"Size: %s / %s\n",
		utils.FormatBytes(upload.UploadedBytes),
		utils.FormatBytes(upload.TotalSize),
	)))

	if upload.Speed > 0 {
		content.WriteString(stats.Render(fmt.Sprintf(
			"Speed: %.1f MB/s\n",
			upload.Speed,
		)))
	}

	if upload.ETA > 0 {
		content.WriteString(stats.Render(fmt.Sprintf(
			"ETA: %s\n",
			m.formatDuration(upload.ETA),
		)))
	}

	if upload.TotalParts > 0 {
		content.WriteString(parts.Render(fmt.Sprintf(
			"Parts: %d / %d completed",
			upload.PartsCompleted,
			upload.TotalParts,
		)))
	}

	return content.String()
}

func (m *R2ProgressModel) formatControls() string {
	return lipgloss.NewStyle().
		Foreground(m.progress.theme.Secondary).
		Render("Press 'q' to cancel • 'Ctrl+C' to quit")
}

func (m *R2ProgressModel) getStatusColor(status UploadStatus) lipgloss.Color {
	switch status {
	case StatusQueued:
		return m.progress.theme.Info
	case StatusConnecting:
		return m.progress.theme.Warning
	case StatusUploading:
		return m.progress.theme.Primary
	case StatusPaused:
		return m.progress.theme.Warning
	case StatusCompleted:
		return m.progress.theme.Success
	case StatusFailed, StatusCancelled:
		return m.progress.theme.Error
	default:
		return m.progress.theme.Secondary
	}
}

func (m *R2ProgressModel) formatDuration(d time.Duration) string {
	if d <= 0 {
		return "Done"
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), float64(int(d.Seconds())%60))
	}
	return fmt.Sprintf("%.0fh %.0fm", d.Hours(), float64(int(d.Minutes())%60))
}

// R2ProgressReader wraps an io.Reader to show upload progress
type R2ProgressReader struct {
	reader      io.Reader
	progress    *R2UploadProgress
	upload      *R2UploadState
	bytesRead   int64
	lastUpdate  time.Time
	startTime   time.Time
}

// NewR2ProgressReader creates a new progress reader for R2 uploads
func NewR2ProgressReader(reader io.Reader, rp *R2UploadProgress, upload *R2UploadState) *R2ProgressReader {
	return &R2ProgressReader{
		reader:     reader,
		progress:   rp,
		upload:     upload,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

// Read implements io.Reader while tracking progress
func (rpr *R2ProgressReader) Read(p []byte) (n int, err error) {
	n, err = rpr.reader.Read(p)
	rpr.bytesRead += int64(n)

	// Update progress every 100ms
	now := time.Now()
	if now.Sub(rpr.lastUpdate) >= 100*time.Millisecond || err == io.EOF {
		speed := float64(rpr.bytesRead) / now.Sub(rpr.startTime).Seconds() / (1024 * 1024)
		rpr.progress.UpdateProgress(rpr.upload, rpr.bytesRead, speed)
		rpr.lastUpdate = now
	}

	if err == io.EOF {
		rpr.progress.CompleteUpload(rpr.upload)
	}

	return n, err
}

// CreateLiveUploadDisplay creates a live dashboard for multiple uploads
func CreateLiveUploadDisplay() func() {
	// This would create a more sophisticated dashboard
	return func() {
		rp := NewR2UploadProgress()
		rp.StartLiveDisplay()
		time.Sleep(10 * time.Second) // Demo
		rp.StopLiveDisplay()
	}
}

// Global convenience function for showing upload progress
func ShowR2UploadProgress(fileName, bucket, objectKey string, size int64) {
	rp := NewR2UploadProgress()
	upload := rp.AddUpload(fileName, bucket, objectKey, size)

	// Start with animated loading
	ShowSpinner(fmt.Sprintf("Preparing to upload %s", fileName), 2*time.Second)

	// Mark as started
	rp.StartUpload(upload)

	// Show simple progress
	rp.ShowSimpleProgress()
}