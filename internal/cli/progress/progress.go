/*
Package progress provides real-time progress monitoring for R2Go2 operations

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package progress

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
)

// ProgressType defines different types of progress displays
type ProgressType int

const (
	ProgressTypeLinear ProgressType = iota
	ProgressTypeCircular
	ProgressTypePercentage
	ProgressTypeSpinner
)

// ProgressState represents the current state of a progress operation
type ProgressState int

const (
	StateProgress ProgressState = iota
	StateComplete
	StateError
	StateCancelled
)

// ProgressInfo contains detailed progress information
type ProgressInfo struct {
	Operation     string        `json:"operation"`
	Current       int64         `json:"current"`
	Total         int64         `json:"total"`
	Percentage    float64       `json:"percentage"`
	Speed         float64       `json:"speed_mbps"`
	ETA           time.Duration `json:"eta"`
	StartTime     time.Time     `json:"start_time"`
	LastUpdate    time.Time     `json:"last_update"`
	State         ProgressState `json:"state"`
	Message       string        `json:"message"`
	Error         error         `json:"error,omitempty"`
}

// ProgressBar defines the interface for progress display
type ProgressBar interface {
	Update(current, total int64)
	SetMessage(message string)
	SetSpeed(bytesPerSecond float64)
	Complete()
	Error(err error)
	Cancel()
	GetInfo() *ProgressInfo
	Start()
}

// EnhancedProgressBar is a feature-rich progress bar implementation
type EnhancedProgressBar struct {
	mu          sync.RWMutex
	info        *ProgressInfo
	bar         *pb.ProgressBar
	template    string
	showSpeed   bool
	showETA     bool
	colorOutput bool
}

// ProgressBarConfig contains configuration for progress bars
type ProgressBarConfig struct {
	Type         ProgressType
	Template     string
	ShowSpeed    bool
	ShowETA      bool
	ColorOutput  bool
	RefreshRate  time.Duration
	Width        int
	HideCursor   bool
}

// DefaultConfig returns a default progress bar configuration
func DefaultConfig() *ProgressBarConfig {
	return &ProgressBarConfig{
		Type:        ProgressTypeLinear,
		Template:    `{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{rtime . }}`,
		ShowSpeed:   true,
		ShowETA:     true,
		ColorOutput: true,
		RefreshRate: 100 * time.Millisecond,
		Width:       80,
		HideCursor:  true,
	}
}

// NewProgressBar creates a new enhanced progress bar
func NewProgressBar(config *ProgressBarConfig) ProgressBar {
	if config == nil {
		config = DefaultConfig()
	}

	bar := &EnhancedProgressBar{
		info: &ProgressInfo{
			State:      StateProgress,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		template:    config.Template,
		showSpeed:   config.ShowSpeed,
		showETA:     config.ShowETA,
		colorOutput: config.ColorOutput,
	}

	// Initialize the underlying progress bar
	bar.bar = pb.New64(0)
	bar.bar.SetTemplateString(config.Template)
	bar.bar.SetRefreshRate(config.RefreshRate)

	if config.Width > 0 {
		bar.bar.SetWidth(config.Width)
	}

	if config.HideCursor {
		hideCursor()
	}

	return bar
}

// NewLinearProgressBar creates a linear progress bar
func NewLinearProgressBar(total int64, message string) ProgressBar {
	config := DefaultConfig()
	config.Type = ProgressTypeLinear

	bar := NewProgressBar(config)
	bar.SetMessage(message)
	bar.Update(0, total)

	return bar
}

// NewCircularProgressBar creates a circular/spinner progress bar
func NewCircularProgressBar(message string) ProgressBar {
	config := DefaultConfig()
	config.Type = ProgressTypeCircular
	config.Template = `{{spinner .}} {{msg .}} {{percent .}}`

	bar := NewProgressBar(config)
	bar.SetMessage(message)

	return bar
}

// Update updates the progress with current values
func (p *EnhancedProgressBar) Update(current, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Update info
	p.info.Current = current
	p.info.Total = total
	p.info.LastUpdate = now

	if total > 0 {
		p.info.Percentage = float64(current) / float64(total) * 100
	} else {
		p.info.Percentage = 0
	}

	// Calculate speed if we have progress
	if p.showSpeed && !p.info.StartTime.IsZero() && current > 0 {
		elapsed := now.Sub(p.info.StartTime).Seconds()
		if elapsed > 0 {
			p.info.Speed = float64(current) / elapsed / (1024 * 1024) // MB/s

			// Calculate ETA
			if total > 0 && p.info.Speed > 0 {
				remaining := float64(total-current) / (1024 * 1024)
				p.info.ETA = time.Duration(remaining/p.info.Speed) * time.Second
			}
		}
	}

	// Update the underlying bar
	p.bar.SetCurrent(current)
	p.bar.SetTotal(total)

	// Update message if provided
	if p.info.Message != "" {
		p.bar.Set("msg", p.info.Message)
	}
}

// SetMessage updates the progress message
func (p *EnhancedProgressBar) SetMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.info.Message = message

	if p.bar != nil {
		p.bar.Set("msg", message)
	}
}

// SetSpeed updates the transfer speed manually
func (p *EnhancedProgressBar) SetSpeed(bytesPerSecond float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.info.Speed = bytesPerSecond / (1024 * 1024) // Convert to MB/s
}

// Complete marks the progress as completed
func (p *EnhancedProgressBar) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.info.State = StateComplete
	p.info.Percentage = 100

	if p.bar != nil {
		p.bar.SetCurrent(p.info.Total)
		p.bar.Finish()
	}

	showCursor()
}

// Error marks the progress as failed
func (p *EnhancedProgressBar) Error(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.info.State = StateError
	p.info.Error = err

	if p.bar != nil {
		p.bar.Finish()
	}

	if p.colorOutput && err != nil {
		color.Red("❌ Error: %v", err)
	}

	showCursor()
}

// Cancel marks the progress as cancelled
func (p *EnhancedProgressBar) Cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.info.State = StateCancelled

	if p.bar != nil {
		p.bar.Finish()
	}

	if p.colorOutput {
		color.Yellow("⚠️ Operation cancelled")
	}

	showCursor()
}

// GetInfo returns the current progress information
func (p *EnhancedProgressBar) GetInfo() *ProgressInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	info := *p.info
	return &info
}

// Start begins the progress display
func (p *EnhancedProgressBar) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.bar != nil {
		p.bar.Start()
	}
}

// ProgressReader wraps an io.Reader with progress tracking
type ProgressReader struct {
	reader    io.Reader
	bar       ProgressBar
	bytesRead int64
}

// NewProgressReader creates a new progress reader
func NewProgressReader(reader io.Reader, bar ProgressBar) *ProgressReader {
	return &ProgressReader{
		reader: reader,
		bar:    bar,
	}
}

// Read implements io.Reader
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.bytesRead += int64(n)

	// Update progress bar
	if pr.bar != nil {
		info := pr.bar.GetInfo()
		pr.bar.Update(pr.bytesRead, info.Total)
	}

	return n, err
}

// ProgressWriter wraps an io.Writer with progress tracking
type ProgressWriter struct {
	writer     io.Writer
	bar        ProgressBar
	bytesWrite int64
}

// NewProgressWriter creates a new progress writer
func NewProgressWriter(writer io.Writer, bar ProgressBar) *ProgressWriter {
	return &ProgressWriter{
		writer: writer,
		bar:    bar,
	}
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	pw.bytesWrite += int64(n)

	// Update progress bar
	if pw.bar != nil {
		info := pw.bar.GetInfo()
		pw.bar.Update(pw.bytesWrite, info.Total)
	}

	return n, err
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	bars  map[string]ProgressBar
	order []string
	mu    sync.RWMutex
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		bars: make(map[string]ProgressBar),
	}
}

// Add adds a new progress bar
func (mp *MultiProgress) Add(id string, bar ProgressBar) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.bars[id] = bar
	mp.order = append(mp.order, id)
}

// Update updates a specific progress bar
func (mp *MultiProgress) Update(id string, current, total int64) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if bar, exists := mp.bars[id]; exists {
		bar.Update(current, total)
	}
}

// Complete marks a specific progress bar as complete
func (mp *MultiProgress) Complete(id string) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if bar, exists := mp.bars[id]; exists {
		bar.Complete()
	}
}

// Error marks a specific progress bar as failed
func (mp *MultiProgress) Error(id string, err error) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if bar, exists := mp.bars[id]; exists {
		bar.Error(err)
	}
}

// GetAllInfo returns information about all progress bars
func (mp *MultiProgress) GetAllInfo() map[string]*ProgressInfo {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	result := make(map[string]*ProgressInfo)
	for id, bar := range mp.bars {
		result[id] = bar.GetInfo()
	}

	return result
}

// Utility functions

func hideCursor() {
	fmt.Print("\033[?25l")
}

func showCursor() {
	fmt.Print("\033[?25h")
}

// FormatBytes formats bytes into human readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration into human readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), float64(int(d.Seconds())%60))
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}