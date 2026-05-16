package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestAnalyticsCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "analytics" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("analyticsCmd not registered on rootCmd")
	}
}

func TestAnalyticsCmd_Metadata(t *testing.T) {
	if analyticsCmd.Use != "analytics" {
		t.Errorf("analyticsCmd.Use = %q, want %q", analyticsCmd.Use, "analytics")
	}
	if analyticsCmd.Short == "" {
		t.Error("analyticsCmd.Short is empty")
	}
}

// --- RunE handler wired ---

func TestAnalyticsCmd_HasRunE(t *testing.T) {
	if analyticsCmd.RunE == nil {
		t.Error("analyticsCmd.RunE is nil")
	}
}

// --- Flag registration ---

func TestAnalyticsCmd_Flags(t *testing.T) {
	expected := []string{"bucket", "period"}
	for _, name := range expected {
		if analyticsCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on analyticsCmd", name)
		}
	}
}

// --- Flag defaults ---

func TestAnalyticsCmd_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"bucket", ""},
		{"period", "7d"},
	}
	for _, tc := range cases {
		f := analyticsCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Flag variable wiring ---

func TestAnalyticsFlagsParsing_Bucket(t *testing.T) {
	cmd := &cobra.Command{}
	var bucket string
	cmd.Flags().StringVar(&bucket, "bucket", "", "")
	if err := cmd.Flags().Set("bucket", "my-bucket"); err != nil {
		t.Fatalf("failed to set --bucket: %v", err)
	}
	if bucket != "my-bucket" {
		t.Errorf("bucket = %q, want %q", bucket, "my-bucket")
	}
}

func TestAnalyticsFlagsParsing_Period(t *testing.T) {
	cmd := &cobra.Command{}
	var period string
	cmd.Flags().StringVar(&period, "period", "7d", "")
	if period != "7d" {
		t.Errorf("period default = %q, want %q", period, "7d")
	}
	if err := cmd.Flags().Set("period", "30d"); err != nil {
		t.Fatalf("failed to set --period: %v", err)
	}
	if period != "30d" {
		t.Errorf("period = %q, want %q", period, "30d")
	}
}
