package cmd

import (
	"testing"
	"time"
)

// --- Command registration ---

func TestListCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("listCmd not registered on rootCmd")
	}
}

func TestListCmd_Metadata(t *testing.T) {
	if listCmd.Use != "list" {
		t.Errorf("listCmd.Use = %q, want %q", listCmd.Use, "list")
	}
	if listCmd.Short == "" {
		t.Error("listCmd.Short is empty")
	}
}

// --- Run handler wired ---

func TestListCmd_HasRunHandler(t *testing.T) {
	if listCmd.Run == nil {
		t.Error("listCmd.Run is nil — no handler wired")
	}
}

// --- Args validation ---

func TestListCmd_ArgsNoArgs(t *testing.T) {
	// listCmd uses cobra.NoArgs — verify it rejects arguments
	if listCmd.Args == nil {
		t.Fatal("listCmd.Args validator is nil")
	}
	if err := listCmd.Args(listCmd, []string{}); err != nil {
		t.Errorf("expected no error with zero args, got: %v", err)
	}
	if err := listCmd.Args(listCmd, []string{"extra"}); err == nil {
		t.Error("expected error with extra args, got nil")
	}
}

// --- Helper functions ---

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		contains string
	}{
		{"minutes ago", time.Now().Add(-5 * time.Minute), "minute"},
		{"hours ago", time.Now().Add(-3 * time.Hour), "hour"},
		{"days ago", time.Now().Add(-7 * 24 * time.Hour), "day"},
		{"old date", time.Now().Add(-90 * 24 * time.Hour), "Created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.t)
			if result == "" {
				t.Error("formatTimeAgo returned empty string")
			}
			// All results should contain the expected substring
			if len(tt.contains) > 0 {
				found := false
				if len(result) >= len(tt.contains) {
					for i := 0; i <= len(result)-len(tt.contains); i++ {
						if result[i:i+len(tt.contains)] == tt.contains {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("formatTimeAgo(%v) = %q, want to contain %q", tt.t, result, tt.contains)
				}
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	if pluralize(1) != "" {
		t.Errorf("pluralize(1) = %q, want %q", pluralize(1), "")
	}
	if pluralize(0) != "s" {
		t.Errorf("pluralize(0) = %q, want %q", pluralize(0), "s")
	}
	if pluralize(5) != "s" {
		t.Errorf("pluralize(5) = %q, want %q", pluralize(5), "s")
	}
}
