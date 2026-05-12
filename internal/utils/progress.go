package utils

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type TransferProgress struct {
	Total     int64
	Transfer  int64
	StartTime time.Time
	Speed     float64 // bytes per second
	Width     int
}

func NewTransferProgress(total int64) *TransferProgress {
	return &TransferProgress{
		Total:     total,
		StartTime: time.Now(),
		Width:     40,
	}
}

func (p *TransferProgress) Add(n int64) {
	p.Transfer += n
	elapsed := time.Since(p.StartTime).Seconds()
	if elapsed > 0 {
		p.Speed = float64(p.Transfer) / elapsed
	}
}

func (p *TransferProgress) Percent() float64 {
	if p.Total <= 0 {
		return 0
	}
	return float64(p.Transfer) / float64(p.Total) * 100
}

func (p *TransferProgress) ETA() time.Duration {
	if p.Speed <= 0 {
		return 0
	}
	remaining := p.Total - p.Transfer
	if remaining <= 0 {
		return 0
	}
	return time.Duration(float64(remaining)/p.Speed) * time.Second
}

func (p *TransferProgress) FormatBar() string {
	pct := p.Percent()
	filled := int(pct / 100 * float64(p.Width))
	if filled > p.Width {
		filled = p.Width
	}

	bar := strings.Repeat("=", filled)
	if filled < p.Width {
		bar += ">"
	}
	empty := p.Width - filled - 1
	if empty < 0 {
		empty = 0
	}

	return fmt.Sprintf("[%s%s] %5.1f%% | %s/s | ETA %s",
		bar,
		strings.Repeat(" ", empty),
		pct,
		FormatBytes(int64(p.Speed)),
		formatDuration(p.ETA()),
	)
}

func (p *TransferProgress) PrintTo(w io.Writer) {
	fmt.Fprintf(w, "\r%s", p.FormatBar())
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "--"
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	if m < 60 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

type ProgressWriter struct {
	io.Writer
	Progress *TransferProgress
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Progress.Add(int64(n))
	return n, err
}

type ProgressReader struct {
	io.Reader
	Progress *TransferProgress
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Progress.Add(int64(n))
	return n, err
}
