package cmd

import (
	"testing"
	"time"
)

func TestMetricsCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "metrics" {
			found = true
			break
		}
	}
	if !found {
		t.Error("metrics command not registered on root")
	}
}

func TestMetricsCmd_Metadata(t *testing.T) {
	if metricsCmd.Use != "metrics" {
		t.Errorf("expected Use 'metrics', got %q", metricsCmd.Use)
	}
	if metricsCmd.Short == "" {
		t.Error("metrics command should have a short description")
	}
}

func TestMetricsCmd_DefaultInterval(t *testing.T) {
	f := metricsCmd.Flags().Lookup("interval")
	if f == nil {
		t.Fatal("--interval flag not registered")
	}
	if f.DefValue != "30s" {
		t.Errorf("expected default interval '30s', got %q", f.DefValue)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range tests {
		got := formatSize(tc.bytes)
		if got != tc.want {
			t.Errorf("formatSize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestMetricsSnapshot_Fields(t *testing.T) {
	snap := metricsSnapshot{
		Timestamp:      time.Now(),
		R2Buckets:      3,
		R2TotalSize:    1048576,
		R2TotalObjects: 100,
		Workers:        5,
		KVNamespaces:   2,
	}
	if snap.R2Buckets != 3 {
		t.Errorf("expected 3 buckets, got %d", snap.R2Buckets)
	}
	if snap.Workers != 5 {
		t.Errorf("expected 5 workers, got %d", snap.Workers)
	}
}
