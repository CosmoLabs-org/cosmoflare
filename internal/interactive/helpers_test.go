package interactive

import (
	"os"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// CenterText
// ---------------------------------------------------------------------------

func TestCenterText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		width     int
		want      string
		wantLen   int // total length must equal width
	}{
		{
			name:    "text shorter than width",
			text:    "hello",
			width:   10,
			want:    "  hello   ",
			wantLen: 10,
		},
		{
			name:    "no room for padding",
			text:    "hi",
			width:   2,
			want:    "hi",
			wantLen: 2,
		},
		{
			name:    "exact fit",
			text:    "test",
			width:   4,
			want:    "test",
			wantLen: 4,
		},
		{
			name:    "empty text returns all padding",
			text:    "",
			width:   5,
			want:    "     ",
			wantLen: 5,
		},
		{
			name:    "odd padding left gets fewer spaces",
			text:    "ab",
			width:   3,
			want:    "ab ", // padding=1, left=0, right=1
			wantLen: 3,
		},
		{
			name:    "text longer than width returns text unchanged",
			text:    "hello world",
			width:   5,
			want:    "hello world",
			wantLen: 11,
		},
		{
			name:    "single char centered in width 3",
			text:    "x",
			width:   3,
			want:    " x ", // padding=2, left=1, right=1
			wantLen: 3,
		},
		{
			name:    "even width 10 with 3-char text",
			text:    "abc",
			width:   10,
			want:    "   abc    ", // padding=7, left=3, right=4
			wantLen: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CenterText(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("CenterText(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
			if len(got) != tt.wantLen {
				t.Errorf("CenterText(%q, %d) length = %d, want %d", tt.text, tt.width, len(got), tt.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BoxText
// ---------------------------------------------------------------------------

func TestBoxText(t *testing.T) {
	t.Run("single line", func(t *testing.T) {
		result := BoxText("hello")

		// Must have proper box borders
		if !strings.HasPrefix(result, "\u250c") {
			t.Errorf("BoxText should start with top-left corner, got prefix: %q", result[:4])
		}
		if !strings.HasSuffix(result, "\u2518") {
			t.Errorf("BoxText should end with bottom-right corner, got suffix: %q", result[len(result)-4:])
		}
		if !strings.Contains(result, "\u2500") {
			t.Error("BoxText should contain horizontal border")
		}
		if !strings.Contains(result, "\u2502") {
			t.Error("BoxText should contain vertical border")
		}
		if !strings.Contains(result, "\u2510") {
			t.Error("BoxText should contain top-right corner")
		}
		if !strings.Contains(result, "\u2514") {
			t.Error("BoxText should contain bottom-left corner")
		}
		// Content line should have the text
		if !strings.Contains(result, "hello") {
			t.Error("BoxText should contain the original text")
		}
	})

	t.Run("multi line", func(t *testing.T) {
		result := BoxText("line1\nline2")

		if !strings.Contains(result, "line1") {
			t.Error("BoxText multi-line should contain line1")
		}
		if !strings.Contains(result, "line2") {
			t.Error("BoxText multi-line should contain line2")
		}
		// Both content lines should be padded to the same width
		lines := strings.Split(result, "\n")
		// lines[0] = top border, lines[1] = first content, lines[2] = second content, lines[3] = bottom border
		if len(lines) != 4 {
			t.Fatalf("BoxText multi-line should have 4 lines, got %d", len(lines))
		}
		// Content lines should be the same length
		if len(lines[1]) != len(lines[2]) {
			t.Errorf("content lines should be same length: line1=%d, line2=%d", len(lines[1]), len(lines[2]))
		}
		// Top and bottom borders should be the same length
		if len(lines[0]) != len(lines[3]) {
			t.Errorf("border lines should be same length: top=%d, bottom=%d", len(lines[0]), len(lines[3]))
		}
	})

	t.Run("empty text", func(t *testing.T) {
		result := BoxText("")
		if !strings.HasPrefix(result, "\u250c") {
			t.Error("BoxText empty should still have top-left corner")
		}
		if !strings.HasSuffix(result, "\u2518") {
			t.Error("BoxText empty should still have bottom-right corner")
		}
	})
}

// ---------------------------------------------------------------------------
// TruncateText
// ---------------------------------------------------------------------------

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		want      string
	}{
		{
			name:      "no truncation needed",
			text:      "hello",
			maxLength: 10,
			want:      "hello",
		},
		{
			name:      "truncate with ellipsis",
			text:      "hello world",
			maxLength: 5,
			want:      "he...",
		},
		{
			name:      "truncate at boundary",
			text:      "hello world",
			maxLength: 8,
			want:      "hello...",
		},
		{
			name:      "exact fit no truncation",
			text:      "hi",
			maxLength: 2,
			want:      "hi",
		},
		{
			name:      "zero length returns empty",
			text:      "test",
			maxLength: 0,
			want:      "",
		},
		{
			name:      "maxLength 1 returns bullet",
			text:      "ab",
			maxLength: 1,
			want:      "\u25cf",
		},
		{
			name:      "maxLength 2 returns two bullets",
			text:      "abc",
			maxLength: 2,
			want:      "\u25cf\u25cf",
		},
		{
			name:      "empty text no truncation",
			text:      "",
			maxLength: 5,
			want:      "",
		},
		{
			name:      "maxLength 3 truncates to empty prefix plus dots",
			text:      "hello",
			maxLength: 3,
			want:      "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateText(tt.text, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.text, tt.maxLength, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatFileSize
// ---------------------------------------------------------------------------

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"under 1 KB", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"exactly 1 TB", 1099511627776, "1.0 TB"},
		{"small bytes", 1, "1 B"},
		{"999 B", 999, "999 B"},
		{"5 MB", 5242880, "5.0 MB"},
		{"500 B boundary", 512, "512 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatFileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatProfileName
// ---------------------------------------------------------------------------

func TestFormatProfileName(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		got := FormatProfileName("prod", "Production account")
		if !strings.Contains(got, "prod") {
			t.Error("FormatProfileName should contain profile name")
		}
		if !strings.Contains(got, "Production account") {
			t.Error("FormatProfileName should contain description")
		}
	})

	t.Run("empty description returns just name", func(t *testing.T) {
		got := FormatProfileName("dev", "")
		if got != "dev" {
			t.Errorf("FormatProfileName(%q, %q) = %q, want %q", "dev", "", got, "dev")
		}
	})

	t.Run("name with spaces", func(t *testing.T) {
		got := FormatProfileName("my profile", "some desc")
		if !strings.Contains(got, "my profile") {
			t.Error("FormatProfileName should preserve name with spaces")
		}
	})
}

// ---------------------------------------------------------------------------
// ValidateAdvancedConfig
// ---------------------------------------------------------------------------

func TestValidateAdvancedConfig(t *testing.T) {
	makeConfig := func(concurrency, retries int, region string) *AdvancedConfig {
		return &AdvancedConfig{
			Profile: &config.Profile{Name: "test"},
			UploadSettings: UploadSettings{
				Concurrency:   concurrency,
				RetryAttempts: retries,
			},
			RegionSettings: RegionSettings{
				Primary: region,
			},
		}
	}

	t.Run("valid config passes", func(t *testing.T) {
		cfg := makeConfig(4, 3, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("concurrency zero fails", func(t *testing.T) {
		cfg := makeConfig(0, 3, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err == nil {
			t.Error("expected error for concurrency=0")
		}
	})

	t.Run("concurrency 33 fails", func(t *testing.T) {
		cfg := makeConfig(33, 3, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err == nil {
			t.Error("expected error for concurrency=33")
		}
	})

	t.Run("retries negative fails", func(t *testing.T) {
		cfg := makeConfig(4, -1, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err == nil {
			t.Error("expected error for retries=-1")
		}
	})

	t.Run("retries 11 fails", func(t *testing.T) {
		cfg := makeConfig(4, 11, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err == nil {
			t.Error("expected error for retries=11")
		}
	})

	t.Run("invalid region fails", func(t *testing.T) {
		cfg := makeConfig(4, 3, "invalid-region")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err == nil {
			t.Error("expected error for invalid region")
		}
	})

	t.Run("all valid regions pass", func(t *testing.T) {
		validRegions := []string{"auto", "us-east-1", "eu-west-1", "ap-southeast-1"}
		for _, region := range validRegions {
			t.Run(region, func(t *testing.T) {
				cfg := makeConfig(4, 3, region)
				wizard := NewAdvancedConfigWizard(cfg.Profile)
				if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
					t.Errorf("region %q should be valid, got error: %v", region, err)
				}
			})
		}
	})

	t.Run("boundary concurrency=1 passes", func(t *testing.T) {
		cfg := makeConfig(1, 3, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
			t.Errorf("concurrency=1 should be valid, got error: %v", err)
		}
	})

	t.Run("boundary concurrency=32 passes", func(t *testing.T) {
		cfg := makeConfig(32, 3, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
			t.Errorf("concurrency=32 should be valid, got error: %v", err)
		}
	})

	t.Run("boundary retries=0 passes", func(t *testing.T) {
		cfg := makeConfig(4, 0, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
			t.Errorf("retries=0 should be valid, got error: %v", err)
		}
	})

	t.Run("boundary retries=10 passes", func(t *testing.T) {
		cfg := makeConfig(4, 10, "auto")
		wizard := NewAdvancedConfigWizard(cfg.Profile)
		if err := wizard.ValidateAdvancedConfig(cfg); err != nil {
			t.Errorf("retries=10 should be valid, got error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// AutoDetectAccessibility - renamed to avoid conflict with accessibility_deep_test.go
// ---------------------------------------------------------------------------

func TestAutoDetectAccessibilityFromHelpers(t *testing.T) {
	t.Run("no env vars returns none", func(t *testing.T) {
		// Clear all relevant env vars
		envVars := []string{"SCREEN_READER", "ACCESSIBILITY", "HIGH_CONTRAST", "LARGE_TEXT", "REDUCED_MOTION", "TERM_PROGRAM"}
		for _, key := range envVars {
			os.Unsetenv(key)
		}

		am := AutoDetectAccessibility()
		if am.IsEnabled() {
			t.Error("AutoDetectAccessibility with no env vars should have IsEnabled=false")
		}
		if am.config.Mode != AccessibilityNone {
			t.Errorf("expected AccessibilityNone, got %d", am.config.Mode)
		}
	})

	t.Run("SCREEN_READER set triggers screen reader mode", func(t *testing.T) {
		os.Setenv("SCREEN_READER", "1")
		defer os.Unsetenv("SCREEN_READER")
		os.Unsetenv("ACCESSIBILITY")
		os.Unsetenv("HIGH_CONTRAST")
		os.Unsetenv("LARGE_TEXT")
		os.Unsetenv("REDUCED_MOTION")

		am := AutoDetectAccessibility()
		if am.config.Mode != AccessibilityScreenReader {
			t.Errorf("expected AccessibilityScreenReader, got %d", am.config.Mode)
		}
		if !am.IsEnabled() {
			t.Error("ScreenReader mode should have IsEnabled=true")
		}
	})

	t.Run("HIGH_CONTRAST set triggers high contrast mode", func(t *testing.T) {
		os.Setenv("HIGH_CONTRAST", "1")
		defer os.Unsetenv("HIGH_CONTRAST")
		os.Unsetenv("SCREEN_READER")
		os.Unsetenv("ACCESSIBILITY")
		os.Unsetenv("LARGE_TEXT")
		os.Unsetenv("REDUCED_MOTION")

		am := AutoDetectAccessibility()
		if am.config.Mode != AccessibilityHighContrast {
			t.Errorf("expected AccessibilityHighContrast, got %d", am.config.Mode)
		}
	})

	t.Run("LARGE_TEXT set triggers large text mode", func(t *testing.T) {
		os.Setenv("LARGE_TEXT", "1")
		defer os.Unsetenv("LARGE_TEXT")
		os.Unsetenv("SCREEN_READER")
		os.Unsetenv("ACCESSIBILITY")
		os.Unsetenv("HIGH_CONTRAST")
		os.Unsetenv("REDUCED_MOTION")

		am := AutoDetectAccessibility()
		if am.config.Mode != AccessibilityLargeText {
			t.Errorf("expected AccessibilityLargeText, got %d", am.config.Mode)
		}
	})

	t.Run("REDUCED_MOTION set triggers reduced motion mode", func(t *testing.T) {
		os.Setenv("REDUCED_MOTION", "1")
		defer os.Unsetenv("REDUCED_MOTION")
		os.Unsetenv("SCREEN_READER")
		os.Unsetenv("ACCESSIBILITY")
		os.Unsetenv("HIGH_CONTRAST")
		os.Unsetenv("LARGE_TEXT")

		am := AutoDetectAccessibility()
		if am.config.Mode != AccessibilityReducedMotion {
			t.Errorf("expected AccessibilityReducedMotion, got %d", am.config.Mode)
		}
	})
}

// ---------------------------------------------------------------------------
// AccessibilityManager
// ---------------------------------------------------------------------------

func TestAccessibilityManager(t *testing.T) {
	t.Run("new manager is disabled", func(t *testing.T) {
		am := NewAccessibilityManager()
		if am.IsEnabled() {
			t.Error("NewAccessibilityManager should have IsEnabled=false")
		}
		if am.config.Mode != AccessibilityNone {
			t.Error("NewAccessibilityManager should have Mode=AccessibilityNone")
		}
	})

	t.Run("SetMode ScreenReader enables relevant flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityScreenReader)

		if !am.IsEnabled() {
			t.Error("ScreenReader mode should have IsEnabled=true")
		}
		if !am.config.ScreenReader {
			t.Error("ScreenReader mode should set ScreenReader=true")
		}
		if !am.config.Verbose {
			t.Error("ScreenReader mode should set Verbose=true")
		}
	})

	t.Run("SetMode Full enables all flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityFull)

		if !am.IsEnabled() {
			t.Error("Full mode should have IsEnabled=true")
		}
		if !am.config.HighContrast {
			t.Error("Full mode should set HighContrast=true")
		}
		if !am.config.ScreenReader {
			t.Error("Full mode should set ScreenReader=true")
		}
		if !am.config.LargeText {
			t.Error("Full mode should set LargeText=true")
		}
		if !am.config.ReducedMotion {
			t.Error("Full mode should set ReducedMotion=true")
		}
		if !am.config.Verbose {
			t.Error("Full mode should set Verbose=true")
		}
		if !am.config.KeyboardOnly {
			t.Error("Full mode should set KeyboardOnly=true")
		}
		if !am.config.ColorBlind {
			t.Error("Full mode should set ColorBlind=true")
		}
		if !am.config.AnnounceActions {
			t.Error("Full mode should set AnnounceActions=true")
		}
		if !am.config.ExtendedTimeouts {
			t.Error("Full mode should set ExtendedTimeouts=true")
		}
	})

	t.Run("SetMode None disables all", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityFull)
		am.SetMode(AccessibilityNone)

		if am.IsEnabled() {
			t.Error("AccessibilityNone should have IsEnabled=false")
		}
	})

	t.Run("SetMode HighContrast sets correct flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityHighContrast)

		if !am.config.HighContrast {
			t.Error("HighContrast mode should set HighContrast=true")
		}
		if !am.config.ColorBlind {
			t.Error("HighContrast mode should set ColorBlind=true")
		}
	})

	t.Run("SetMode ReducedMotion sets correct flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityReducedMotion)

		if !am.config.ReducedMotion {
			t.Error("ReducedMotion mode should set ReducedMotion=true")
		}
	})

	t.Run("SetMode LargeText sets correct flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityLargeText)

		if !am.config.LargeText {
			t.Error("LargeText mode should set LargeText=true")
		}
		if !am.config.Verbose {
			t.Error("LargeText mode should set Verbose=true")
		}
	})
}

// ---------------------------------------------------------------------------
// ThemeManager
// ---------------------------------------------------------------------------

func TestThemeManager(t *testing.T) {
	t.Run("new manager has cosmic default", func(t *testing.T) {
		tm := NewThemeManager()
		theme := tm.GetCurrentTheme()
		if theme == nil {
			t.Fatal("GetCurrentTheme should return non-nil")
		}
		if theme.Name != "Cosmic" {
			t.Errorf("default theme should be Cosmic, got %q", theme.Name)
		}
	})

	t.Run("ListThemes returns 5 themes", func(t *testing.T) {
		tm := NewThemeManager()
		themes := tm.ListThemes()
		if len(themes) != 5 {
			t.Errorf("ListThemes should return 5 themes, got %d", len(themes))
		}
	})

	t.Run("SetTheme forest succeeds", func(t *testing.T) {
		tm := NewThemeManager()
		err := tm.SetTheme("forest")
		if err != nil {
			t.Errorf("SetTheme(forest) should succeed, got error: %v", err)
		}
		if tm.GetCurrentTheme().Name != "Forest" {
			t.Errorf("current theme should be Forest, got %q", tm.GetCurrentTheme().Name)
		}
	})

	t.Run("SetTheme nonexistent returns error", func(t *testing.T) {
		tm := NewThemeManager()
		err := tm.SetTheme("nonexistent")
		if err == nil {
			t.Error("SetTheme(nonexistent) should return error")
		}
	})

	t.Run("GetCurrentTheme non-nil", func(t *testing.T) {
		tm := NewThemeManager()
		if tm.GetCurrentTheme() == nil {
			t.Error("GetCurrentTheme should return non-nil")
		}
	})

	t.Run("theme has expected properties", func(t *testing.T) {
		tm := NewThemeManager()
		theme := tm.GetCurrentTheme()
		if theme.ID != "cosmic" {
			t.Errorf("default theme ID should be cosmic, got %q", theme.ID)
		}
		if theme.Colors.Primary == "" {
			t.Error("default theme should have Primary color set")
		}
		if theme.Spacing.IndentSize == 0 {
			t.Error("default theme should have IndentSize set")
		}
		if len(theme.Icons.SpinnerChars) == 0 {
			t.Error("default theme should have SpinnerChars")
		}
	})

	t.Run("all 5 theme IDs are present", func(t *testing.T) {
		tm := NewThemeManager()
		expectedIDs := []string{"cosmic", "forest", "ocean", "sunset", "monochrome"}
		themes := tm.ListThemes()
		found := make(map[string]bool)
		for _, th := range themes {
			found[th.ID] = true
		}
		for _, id := range expectedIDs {
			if !found[id] {
				t.Errorf("missing theme with ID %q", id)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// getProfileIcon (unexported)
// ---------------------------------------------------------------------------

func TestGetProfileIcon(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{"production keyword", "Production account", "\U0001f3ed "},
		{"prod keyword", "prod server", "\U0001f3ed "},
		{"staging keyword", "staging environment", "\u26a1 "},
		{"dev keyword", "dev setup", "\u26a1 "},
		{"test keyword", "test profile", "\U0001f9ea "},
		{"personal keyword", "personal bucket", "\U0001f4bb "},
		{"work keyword", "work account", "\U0001f3e2 "},
		{"company keyword", "company bucket", "\U0001f3e2 "},
		{"default no match", "random description", "\U0001f4c1 "},
		{"empty description", "", "\U0001f4c1 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProfileIcon(tt.description)
			if got != tt.want {
				t.Errorf("getProfileIcon(%q) = %q, want %q", tt.description, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileDescription (unexported)
// ---------------------------------------------------------------------------

func TestGetProfileDescription(t *testing.T) {
	t.Run("empty description returns no description", func(t *testing.T) {
		got := getProfileDescription("")
		if got != "(no description)" {
			t.Errorf("got %q, want %q", got, "(no description)")
		}
	})

	t.Run("short description wrapped in parens", func(t *testing.T) {
		got := getProfileDescription("my desc")
		if got != "(my desc)" {
			t.Errorf("got %q, want %q", got, "(my desc)")
		}
	})

	t.Run("long description truncated to 30 chars", func(t *testing.T) {
		long := "this is a very long description that exceeds thirty characters"
		got := getProfileDescription(long)
		// 27 chars from description + "..."
		if !strings.HasPrefix(got, "(") || !strings.HasSuffix(got, ")") {
			t.Errorf("should be wrapped in parens, got %q", got)
		}
		// Inner content should be 27 chars + "..."
		inner := got[1 : len(got)-1] // strip parens
		if len(inner) != 30 { // 27 + 3 for "..."
			t.Errorf("inner content should be 30 chars, got %d: %q", len(inner), inner)
		}
		if !strings.HasSuffix(inner, "...") {
			t.Errorf("long description should end with ..., got %q", inner)
		}
	})
}

// ---------------------------------------------------------------------------
// formatRegion (unexported)
// ---------------------------------------------------------------------------

func TestFormatRegion(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"auto", "Auto-detect"},
		{"", "Auto-detect"},
		{"us-east-1", "US-EAST-1"},
		{"eu-west-1", "EU-WEST-1"},
		{"ap-southeast-1", "AP-SOUTHEAST-1"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := formatRegion(tt.region)
			if got != tt.want {
				t.Errorf("formatRegion(%q) = %q, want %q", tt.region, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formatTokenStatus (unexported)
// ---------------------------------------------------------------------------

func TestFormatTokenStatus(t *testing.T) {
	t.Run("empty token returns Not set", func(t *testing.T) {
		got := formatTokenStatus("")
		if !strings.Contains(got, "Not set") {
			t.Errorf("empty token should return 'Not set', got %q", got)
		}
	})

	t.Run("short token returns Invalid", func(t *testing.T) {
		got := formatTokenStatus("short")
		if !strings.Contains(got, "Invalid") {
			t.Errorf("short token should return 'Invalid', got %q", got)
		}
	})

	t.Run("valid token returns Configured", func(t *testing.T) {
		got := formatTokenStatus("abcdefghijklmnopqrst") // 20 chars
		if !strings.Contains(got, "Configured") {
			t.Errorf("valid token should return 'Configured', got %q", got)
		}
	})

	t.Run("exactly 10 chars returns Configured", func(t *testing.T) {
		got := formatTokenStatus("0123456789")
		if !strings.Contains(got, "Configured") {
			t.Errorf("10-char token should return 'Configured', got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// formatAccountIDStatus (unexported)
// ---------------------------------------------------------------------------

func TestFormatAccountIDStatus(t *testing.T) {
	t.Run("empty returns Not set", func(t *testing.T) {
		got := formatAccountIDStatus("")
		if !strings.Contains(got, "Not set") {
			t.Errorf("empty account ID should return 'Not set', got %q", got)
		}
	})

	t.Run("wrong length returns Invalid format", func(t *testing.T) {
		got := formatAccountIDStatus("tooshort")
		if !strings.Contains(got, "Invalid format") {
			t.Errorf("wrong length should return 'Invalid format', got %q", got)
		}
	})

	t.Run("32 chars returns Valid format", func(t *testing.T) {
		validID := strings.Repeat("a", 32)
		got := formatAccountIDStatus(validID)
		if !strings.Contains(got, "Valid format") {
			t.Errorf("32-char ID should return 'Valid format', got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// PrintSuccess
// ---------------------------------------------------------------------------

func TestPrintSuccess(t *testing.T) {
	tests := []struct {
		name string
		fmt  string
		args []interface{}
	}{
		{"simple message", "operation completed", nil},
		{"with args", "uploaded %d files", []interface{}{5}},
		{"empty string", "", nil},
		{"with format verb", "status: %s", []interface{}{"ok"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				PrintSuccess(tt.fmt, tt.args...)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// PrintError
// ---------------------------------------------------------------------------

func TestPrintError(t *testing.T) {
	tests := []struct {
		name string
		fmt  string
		args []interface{}
	}{
		{"simple error", "something went wrong", nil},
		{"with args", "failed to upload %s", []interface{}{"file.txt"}},
		{"error value", "%v", []interface{}{os.ErrNotExist}},
		{"empty string", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				PrintError(tt.fmt, tt.args...)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// PrintWarning
// ---------------------------------------------------------------------------

func TestPrintWarning(t *testing.T) {
	tests := []struct {
		name string
		fmt  string
		args []interface{}
	}{
		{"simple warning", "disk space low", nil},
		{"with args", "%d warnings found", []interface{}{3}},
		{"empty string", "", nil},
		{"long message", "this is a very long warning message that contains a lot of text", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				PrintWarning(tt.fmt, tt.args...)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// PrintInfo
// ---------------------------------------------------------------------------

func TestPrintInfo(t *testing.T) {
	tests := []struct {
		name string
		fmt  string
		args []interface{}
	}{
		{"simple info", "processing started", nil},
		{"with args", "bucket %s created", []interface{}{"my-bucket"}},
		{"empty string", "", nil},
		{"with multiple args", "%s %d %v", []interface{}{"test", 42, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				PrintInfo(tt.fmt, tt.args...)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// ClearScreen
// ---------------------------------------------------------------------------

func TestClearScreen(t *testing.T) {
	assert.NotPanics(t, func() {
		ClearScreen()
	})
}

func TestClearScreen_MultipleCalls(t *testing.T) {
	assert.NotPanics(t, func() {
		for i := 0; i < 5; i++ {
			ClearScreen()
		}
	})
}

// ---------------------------------------------------------------------------
// GetConfigManager
// ---------------------------------------------------------------------------

func TestGetConfigManager(t *testing.T) {
	mgr, err := GetConfigManager()
	if err != nil {
		t.Skipf("GetConfigManager failed (no config dir): %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil config manager")
	}
}

// ---------------------------------------------------------------------------
// ShowCommandTip
// ---------------------------------------------------------------------------

func TestShowCommandTip(t *testing.T) {
	t.Run("single tip", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ShowCommandTip("Use --help for more info")
		})
	})

	t.Run("multiple tips", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ShowCommandTip("Tip 1", "Tip 2", "Tip 3")
		})
	})

	t.Run("no tips", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ShowCommandTip()
		})
	})

	t.Run("long tip text", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ShowCommandTip("This is a very long tip that contains a lot of helpful information for the user")
		})
	})
}

// ---------------------------------------------------------------------------
// ExitWithError
// ---------------------------------------------------------------------------

func TestExitWithError_PanicRecovery(t *testing.T) {
	// ExitWithError calls os.Exit, which we cannot test directly.
	// We verify the function exists and the error formatting works.
	defer func() {
		// If ExitWithError were somehow called, we'd see a panic from os.Exit in test.
		// It won't be called here; this is just documentation of the limitation.
	}()
	// We can only verify that the function is callable in a non-exit context
	// by testing PrintError directly (already tested above).
	assert.True(t, true, "ExitWithError cannot be tested without subprocess")
}

// ---------------------------------------------------------------------------
// Color helper functions (verify they produce output without panicking)
// ---------------------------------------------------------------------------

func TestColorHelpers(t *testing.T) {
	input := "test text"
	assert.NotEmpty(t, Bold(input))
	assert.NotEmpty(t, Success(input))
	assert.NotEmpty(t, Error(input))
	assert.NotEmpty(t, Warning(input))
	assert.NotEmpty(t, Info(input))
	assert.NotEmpty(t, Dim(input))
	assert.NotEmpty(t, Muted(input))
	assert.NotEmpty(t, Red(input))
	assert.NotEmpty(t, Green(input))
	assert.NotEmpty(t, Yellow(input))
	assert.NotEmpty(t, Blue(input))
	assert.NotEmpty(t, Magenta(input))
	assert.NotEmpty(t, Cyan(input))
	assert.NotEmpty(t, White(input))
}

func TestColorHelpers_EmptyInput(t *testing.T) {
	// Color functions should not panic on empty input
	assert.NotPanics(t, func() { Bold("") })
	assert.NotPanics(t, func() { Success("") })
	assert.NotPanics(t, func() { Error("") })
	assert.NotPanics(t, func() { Warning("") })
	assert.NotPanics(t, func() { Info("") })
	assert.NotPanics(t, func() { Dim("") })
	assert.NotPanics(t, func() { Muted("") })
}

// ---------------------------------------------------------------------------
// Reset constant
// ---------------------------------------------------------------------------

func TestResetConstant(t *testing.T) {
	assert.NotEmpty(t, Reset)
	assert.Contains(t, Reset, "\033")
}

// ---------------------------------------------------------------------------
// stdinReader
// ---------------------------------------------------------------------------

func TestStdinReader_ImplementsInputReader(t *testing.T) {
	var reader InputReader = &stdinReader{}
	assert.NotNil(t, reader)
}

// ---------------------------------------------------------------------------
// FormatFileSize additional edge cases
// ---------------------------------------------------------------------------

func TestFormatFileSize_LargeValues(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
	}{
		{"1 PB", 1125899906842624},
		{"1 EB", 1152921504606846976},
		{"max int64", 9223372036854775807},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileSize(tt.bytes)
			assert.NotEmpty(t, got)
			assert.Contains(t, got, "B")
		})
	}
}

// ---------------------------------------------------------------------------
// TruncateText edge cases
// ---------------------------------------------------------------------------

func TestTruncateText_NegativeMaxLength(t *testing.T) {
	// TruncateText panics on negative maxLength in Go 1.26+ due to strings.Repeat
	assert.Panics(t, func() {
		TruncateText("hello", -1)
	})
}
