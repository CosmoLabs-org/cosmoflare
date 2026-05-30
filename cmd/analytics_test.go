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

// --- Long description content ---

func TestAnalyticsCmd_LongDescription(t *testing.T) {
	if analyticsCmd.Long == "" {
		t.Fatal("analyticsCmd.Long is empty")
	}
	// Must mention key usage topics
	keywords := []string{"bucket", "analytics", "period"}
	for _, kw := range keywords {
		found := false
		lower := analyticsCmd.Long
		for i := 0; i <= len(lower)-len(kw); i++ {
			match := true
			for j := 0; j < len(kw); j++ {
				c := lower[i+j]
				k := kw[j]
				// case-insensitive compare
				if c != k && c != k-32 && c != k+32 {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("analyticsCmd.Long should mention %q", kw)
		}
	}
}

// --- No subcommands ---

func TestAnalyticsCmd_NoSubcommands(t *testing.T) {
	if len(analyticsCmd.Commands()) != 0 {
		t.Errorf("analyticsCmd has %d subcommands, expected 0", len(analyticsCmd.Commands()))
	}
}

// --- BucketStat struct ---

func TestBucketStat_Fields(t *testing.T) {
	bs := BucketStat{
		Name:        "test-bucket",
		ObjectCount: 42,
		TotalSize:   1024,
		Percentage:  50.5,
	}
	if bs.Name != "test-bucket" {
		t.Errorf("BucketStat.Name = %q, want %q", bs.Name, "test-bucket")
	}
	if bs.ObjectCount != 42 {
		t.Errorf("BucketStat.ObjectCount = %d, want 42", bs.ObjectCount)
	}
	if bs.TotalSize != 1024 {
		t.Errorf("BucketStat.TotalSize = %d, want 1024", bs.TotalSize)
	}
	if bs.Percentage != 50.5 {
		t.Errorf("BucketStat.Percentage = %f, want 50.5", bs.Percentage)
	}
}

// --- AnalyticsResult struct ---

func TestAnalyticsResult_Fields(t *testing.T) {
	result := AnalyticsResult{
		Buckets: []BucketStat{
			{Name: "b1", ObjectCount: 10, TotalSize: 500},
			{Name: "b2", ObjectCount: 20, TotalSize: 1000},
		},
		Total:  BucketStat{Name: "TOTAL", ObjectCount: 30, TotalSize: 1500},
		Period: "30d",
	}
	if len(result.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(result.Buckets))
	}
	if result.Total.TotalSize != 1500 {
		t.Errorf("Total.TotalSize = %d, want 1500", result.Total.TotalSize)
	}
	if result.Period != "30d" {
		t.Errorf("Period = %q, want %q", result.Period, "30d")
	}
}

// --- Flag usage strings ---

func TestAnalyticsCmd_FlagUsageStrings(t *testing.T) {
	bucketFlag := analyticsCmd.Flags().Lookup("bucket")
	if bucketFlag == nil {
		t.Fatal("--bucket flag not found")
	}
	if bucketFlag.Usage == "" {
		t.Error("--bucket flag has empty usage string")
	}

	periodFlag := analyticsCmd.Flags().Lookup("period")
	if periodFlag == nil {
		t.Fatal("--period flag not found")
	}
	if periodFlag.Usage == "" {
		t.Error("--period flag has empty usage string")
	}
}

// --- AnalyticsResult with empty buckets ---

func TestAnalyticsResult_EmptyBuckets(t *testing.T) {
	result := AnalyticsResult{
		Buckets: []BucketStat{},
		Total:   BucketStat{Name: "TOTAL", ObjectCount: 0, TotalSize: 0},
		Period:  "7d",
	}
	if len(result.Buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(result.Buckets))
	}
	if result.Total.ObjectCount != 0 {
		t.Errorf("Total.ObjectCount = %d, want 0", result.Total.ObjectCount)
	}
}
