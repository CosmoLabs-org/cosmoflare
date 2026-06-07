package utils

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewTransferProgress(t *testing.T) {
	p := NewTransferProgress(1024)
	if p.Total != 1024 {
		t.Errorf("Total = %d, want 1024", p.Total)
	}
	if p.Transfer != 0 {
		t.Errorf("Transfer = %d, want 0", p.Transfer)
	}
	if p.Width != 40 {
		t.Errorf("Width = %d, want 40", p.Width)
	}
}

func TestTransferProgress_Add(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Add(100)
	if p.Transfer != 100 {
		t.Errorf("Transfer = %d, want 100", p.Transfer)
	}
	p.Add(200)
	if p.Transfer != 300 {
		t.Errorf("Transfer = %d, want 300", p.Transfer)
	}
}

func TestTransferProgress_Percent(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Add(250)
	if pct := p.Percent(); pct != 25.0 {
		t.Errorf("Percent() = %f, want 25.0", pct)
	}
}

func TestTransferProgress_PercentZeroTotal(t *testing.T) {
	p := NewTransferProgress(0)
	if pct := p.Percent(); pct != 0 {
		t.Errorf("Percent() with zero total = %f, want 0", pct)
	}
}

func TestTransferProgress_FormatBar(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Add(450)
	bar := p.FormatBar()
	if !strings.Contains(bar, "45.0%") {
		t.Errorf("FormatBar() should contain 45.0%%, got: %s", bar)
	}
	if !strings.HasPrefix(bar, "[") {
		t.Errorf("FormatBar() should start with [, got: %s", bar)
	}
	if !strings.Contains(bar, "ETA") {
		t.Errorf("FormatBar() should contain ETA, got: %s", bar)
	}
}

func TestProgressWriter(t *testing.T) {
	var buf bytes.Buffer
	p := NewTransferProgress(100)
	pw := &ProgressWriter{Writer: &buf, Progress: p}

	data := []byte("hello world")
	n, err := pw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	if p.Transfer != int64(len(data)) {
		t.Errorf("Transfer = %d, want %d", p.Transfer, len(data))
	}
}

func TestProgressReader(t *testing.T) {
	data := "hello world"
	r := strings.NewReader(data)
	p := NewTransferProgress(int64(len(data)))
	pr := &ProgressReader{Reader: r, Progress: p}

	buf := make([]byte, len(data))
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Read returned %d, want %d", n, len(data))
	}
	if p.Transfer != int64(len(data)) {
		t.Errorf("Transfer = %d, want %d", p.Transfer, len(data))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{0, "--"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m30s"},
		{3700 * time.Second, "1h1m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.expected)
		}
	}
}

func TestTransferProgress_ETA(t *testing.T) {
	p := NewTransferProgress(1000)
	// Before any transfer
	if eta := p.ETA(); eta != 0 {
		t.Errorf("ETA before transfer = %v, want 0", eta)
	}
}

func TestTransferProgress_ETA_WithSpeed(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Transfer = 500
	p.Speed = 100
	eta := p.ETA()
	if eta <= 0 {
		t.Errorf("ETA with active transfer should be positive, got %v", eta)
	}
	expected := 5 * time.Second
	if eta != expected {
		t.Errorf("ETA = %v, want %v (500 remaining / 100 bytes/s)", eta, expected)
	}
}

func TestTransferProgress_ETA_Complete(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Transfer = 1000
	p.Speed = 100
	eta := p.ETA()
	if eta != 0 {
		t.Errorf("ETA when complete should be 0, got %v", eta)
	}
}

func TestTransferProgress_ETA_OverTransferred(t *testing.T) {
	p := NewTransferProgress(1000)
	p.Transfer = 1200
	p.Speed = 100
	eta := p.ETA()
	if eta != 0 {
		t.Errorf("ETA when over-transferred should be 0, got %v", eta)
	}
}

func TestTransferProgress_PrintTo(t *testing.T) {
	p := NewTransferProgress(100)
	p.Transfer = 50
	p.Speed = 10
	var buf bytes.Buffer
	p.PrintTo(&buf)
	output := buf.String()
	if output == "" {
		t.Error("PrintTo should write non-empty output")
	}
	if !strings.HasPrefix(output, "\r") {
		t.Error("PrintTo should start with carriage return")
	}
	if !strings.Contains(output, "50.0%") {
		t.Errorf("PrintTo should show 50%%, got: %s", output)
	}
}

func TestTransferProgress_PrintTo_ZeroProgress(t *testing.T) {
	p := NewTransferProgress(100)
	var buf bytes.Buffer
	p.PrintTo(&buf)
	output := buf.String()
	if !strings.Contains(output, "0.0%") {
		t.Errorf("PrintTo at zero should show 0%%, got: %s", output)
	}
}

func TestTransferProgress_FullBar(t *testing.T) {
	p := NewTransferProgress(100)
	p.Add(100)
	bar := p.FormatBar()
	if !strings.Contains(bar, "100.0%") {
		t.Errorf("Full bar should show 100%%, got: %s", bar)
	}
}
