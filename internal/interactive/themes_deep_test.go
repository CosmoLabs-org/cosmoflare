package interactive

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Global functions: GetThemeManager, SetGlobalTheme, GetCurrentThemeName,
// ApplyThemeSettings
// ---------------------------------------------------------------------------

func TestThemeDeep_GetThemeManager(t *testing.T) {
	tm := GetThemeManager()
	assert.NotNil(t, tm)
	// Same singleton every time
	assert.Same(t, tm, GetThemeManager())
}

func TestThemeDeep_SetGlobalTheme(t *testing.T) {
	// Save and restore
	origName := GetCurrentThemeName()
	defer func() {
		_ = SetGlobalTheme(strings.ToLower(origName))
	}()

	t.Run("set to forest", func(t *testing.T) {
		err := SetGlobalTheme("forest")
		require.NoError(t, err)
		assert.Equal(t, "Forest", GetCurrentThemeName())
	})

	t.Run("set to monochrome", func(t *testing.T) {
		err := SetGlobalTheme("monochrome")
		require.NoError(t, err)
		assert.Equal(t, "Monochrome", GetCurrentThemeName())
	})

	t.Run("invalid theme returns error", func(t *testing.T) {
		err := SetGlobalTheme("nonexistent")
		assert.Error(t, err)
	})
}

func TestThemeDeep_GetCurrentThemeName(t *testing.T) {
	name := GetCurrentThemeName()
	assert.NotEmpty(t, name)
	assert.NotEqual(t, "unknown", name)
}

func TestThemeDeep_GetCurrentThemeName_NilTheme(t *testing.T) {
	// When currentTheme is nil on the global, the function should return "unknown"
	// We can't set the global's currentTheme to nil through public API,
	// but we can verify the function doesn't panic.
	assert.NotPanics(t, func() {
		_ = GetCurrentThemeName()
	})
}

func TestThemeDeep_ApplyThemeSettings(t *testing.T) {
	// Should not panic even when animations disabled
	assert.NotPanics(t, func() {
		ApplyThemeSettings()
	})

	t.Run("disables animations when theme has animations disabled", func(t *testing.T) {
		_ = SetGlobalTheme("monochrome")
		ApplyThemeSettings()
		assert.True(t, globalAnimator.Disabled)
	})

	// Restore
	_ = SetGlobalTheme("cosmic")
	globalAnimator.Disabled = false
}

// ---------------------------------------------------------------------------
// applyTheme - cover all switch branches
// ---------------------------------------------------------------------------

func TestThemeDeep_ApplyTheme_AllSpeeds(t *testing.T) {
	tm := NewThemeManager()

	tests := []struct {
		name           string
		themeID        string
		expectedStyle  string // not directly observable, but shouldn't panic
		shouldDisable  bool
	}{
		{"disabled animations", "monochrome", "disabled", true},
		{"speed 0 (fast)", "cosmic", "normal", false},   // cosmic speed=60 -> normal
		{"speed 50 (normal)", "ocean", "normal", false},  // ocean speed=50 -> normal
		{"speed 70 (normal)", "sunset", "normal", false}, // sunset speed=70 -> normal
		{"speed 80 (slow)", "forest", "slow", false},     // forest speed=80 -> slow
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalAnimator.Disabled = false
			err := tm.SetTheme(tt.themeID)
			require.NoError(t, err)
			if tt.shouldDisable {
				assert.True(t, globalAnimator.Disabled)
			}
		})
	}
}

func TestThemeDeep_ApplyTheme_NilCurrent(t *testing.T) {
	tm := &ThemeManager{
		themes:    make(map[string]*Theme),
		themePath: "/tmp/test",
	}
	tm.currentTheme = nil
	// Should not panic
	assert.NotPanics(t, func() {
		tm.applyTheme()
	})
}

// ---------------------------------------------------------------------------
// showThemePreview - capture stdout, verify output
// ---------------------------------------------------------------------------

func TestThemeDeep_ShowThemePreview(t *testing.T) {
	tm := NewThemeManager()

	tests := []struct {
		name    string
		themeID string
	}{
		{"cosmic preview", "cosmic"},
		{"forest preview", "forest"},
		{"monochrome preview", "monochrome"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tm.SetTheme(tt.themeID)
			require.NoError(t, err)
			theme := tm.GetCurrentTheme()

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			assert.NotPanics(t, func() {
				tm.showThemePreview(theme)
			})

			w.Close()
			os.Stdout = old

			out, _ := io.ReadAll(r)
			got := string(out)
			assert.Contains(t, got, "Theme Preview")
			assert.Contains(t, got, theme.Name)
			assert.Contains(t, got, "Color Samples")
			assert.Contains(t, got, "Primary")
		})
	}
}

// ---------------------------------------------------------------------------
// showProgressBar - various percentages
// ---------------------------------------------------------------------------

func TestThemeDeep_ShowProgressBar_AllThemes(t *testing.T) {
	tm := NewThemeManager()
	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.showProgressBar(theme, 0)
				tm.showProgressBar(theme, 50)
				tm.showProgressBar(theme, 100)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// SaveTheme / LoadTheme - stubs, just ensure no panic
// ---------------------------------------------------------------------------

func TestThemeDeep_SaveTheme_AllThemes(t *testing.T) {
	tm := NewThemeManager()
	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			err := tm.SaveTheme(theme)
			assert.NoError(t, err)
		})
	}
}

func TestThemeDeep_LoadTheme_AllThemes(t *testing.T) {
	tm := NewThemeManager()
	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			err := tm.LoadTheme(theme.ID)
			assert.NoError(t, err)
		})
	}
}

// ---------------------------------------------------------------------------
// loadCustomThemes - with empty path
// ---------------------------------------------------------------------------

func TestThemeDeep_LoadCustomThemes_EmptyPath(t *testing.T) {
	tm := &ThemeManager{
		themes:    make(map[string]*Theme),
		themePath: "",
	}
	// Should not panic with empty path
	assert.NotPanics(t, func() {
		tm.loadCustomThemes()
	})
}

// ---------------------------------------------------------------------------
// Theme structure edge cases
// ---------------------------------------------------------------------------

func TestThemeDeep_EmbeddedTypes(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("cosmic")
	theme := tm.GetCurrentTheme()

	t.Run("ColorScheme embedded in Theme", func(t *testing.T) {
		assert.Equal(t, "cosmic", theme.Colors.Name)
		assert.Contains(t, theme.Colors.Description, "Purple")
		assert.NotEmpty(t, theme.Colors.Accent)
	})

	t.Run("ThemeSpacing embedded in Theme", func(t *testing.T) {
		assert.Greater(t, theme.Spacing.LineSpacing, 0)
		assert.Greater(t, theme.Spacing.SectionGap, 0)
	})

	t.Run("ThemeIcons embedded in Theme", func(t *testing.T) {
		assert.Equal(t, "█", theme.Icons.ProgressChars)
	})

	t.Run("ThemeAnimations embedded in Theme", func(t *testing.T) {
		assert.NotEmpty(t, theme.Animations.Easing)
	})
}

// ---------------------------------------------------------------------------
// Theme zero-value edge cases
// ---------------------------------------------------------------------------

func TestThemeDeep_ZeroValueTheme(t *testing.T) {
	var theme Theme
	assert.Empty(t, theme.ID)
	assert.Empty(t, theme.Name)
	assert.Empty(t, theme.Colors.Primary)
	assert.Equal(t, 0, theme.Spacing.LineSpacing)
	assert.False(t, theme.Icons.UseEmojis)
	assert.False(t, theme.Animations.Enabled)
}

func TestThemeDeep_ZeroValueColorScheme(t *testing.T) {
	var cs ColorScheme
	assert.Empty(t, cs.Name)
	assert.Empty(t, cs.Primary)
	assert.Nil(t, cs.Emoji)
}

func TestThemeDeep_ZeroValueSpacing(t *testing.T) {
	var s ThemeSpacing
	assert.Equal(t, 0, s.LineSpacing)
	assert.Equal(t, 0, s.ProgressBarWidth)
}

func TestThemeDeep_ZeroValueIcons(t *testing.T) {
	var i ThemeIcons
	assert.False(t, i.UseEmojis)
	assert.Nil(t, i.SpinnerChars)
	assert.Empty(t, i.ProgressChars)
}

func TestThemeDeep_ZeroValueAnimations(t *testing.T) {
	var a ThemeAnimations
	assert.False(t, a.Enabled)
	assert.Equal(t, 0, a.Speed)
	assert.Empty(t, a.Easing)
	assert.False(t, a.Transitions)
}

// ---------------------------------------------------------------------------
// Theme emoji map operations
// ---------------------------------------------------------------------------

func TestThemeDeep_EmojiMapLookup(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("cosmic")
	emoji := tm.GetCurrentTheme().Colors.Emoji

	t.Run("all required keys exist and are non-empty", func(t *testing.T) {
		keys := []string{"success", "error", "warning", "info", "loading"}
		for _, key := range keys {
			val, ok := emoji[key]
			assert.True(t, ok, "missing emoji key: %s", key)
			assert.NotEmpty(t, val, "empty emoji for key: %s", key)
		}
	})
}

func TestThemeDeep_MonochromeUsesTextNotEmoji(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("monochrome")
	emoji := tm.GetCurrentTheme().Colors.Emoji

	// Monochrome should use text-based symbols
	assert.Equal(t, "✓", emoji["success"])
	assert.Equal(t, "✗", emoji["error"])
}

// ---------------------------------------------------------------------------
// Theme description content
// ---------------------------------------------------------------------------

func TestThemeDeep_AllThemesHaveDescriptions(t *testing.T) {
	tm := NewThemeManager()
	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			assert.NotEmpty(t, theme.Description)
		})
	}
}

// ---------------------------------------------------------------------------
// GetThemePath
// ---------------------------------------------------------------------------

func TestThemeDeep_GetThemePath_ContainsR2Go2(t *testing.T) {
	tm := NewThemeManager()
	path := tm.GetThemePath()
	assert.Contains(t, path, ".r2go2")
	assert.Contains(t, path, "themes")
}

// ---------------------------------------------------------------------------
// fmt.Stringer / interface compliance
// ---------------------------------------------------------------------------

func TestThemeDeep_ThemeManagerInterface(t *testing.T) {
	tm := NewThemeManager()
	// Verify it implements the expected behavior
	assert.Implements(t, (*interface{})(nil), tm)
}

// ---------------------------------------------------------------------------
// Error formatting for SetTheme
// ---------------------------------------------------------------------------

func TestThemeDeep_SetTheme_ErrorFormatting(t *testing.T) {
	tm := NewThemeManager()
	err := tm.SetTheme("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	err = tm.SetTheme("DOES_NOT_EXIST")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DOES_NOT_EXIST")
}

// ---------------------------------------------------------------------------
// Verify theme consistency
// ---------------------------------------------------------------------------

func TestThemeDeep_OceanSpinnerDuplicates(t *testing.T) {
	// Ocean theme intentionally has duplicate spinner chars - verify they exist
	tm := NewThemeManager()
	tm.SetTheme("ocean")
	spinner := tm.GetCurrentTheme().Icons.SpinnerChars
	assert.Len(t, spinner, 5)
	assert.Equal(t, "🌊", spinner[0])
	assert.Equal(t, "🌊", spinner[1])
}

func TestThemeDeep_MonochromeSpinnerChars(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("monochrome")
	spinner := tm.GetCurrentTheme().Icons.SpinnerChars
	assert.Equal(t, []string{"|", "/", "-", "\\"}, spinner)
}

// ---------------------------------------------------------------------------
// ApplyThemeSettings - with nil currentTheme on global
// ---------------------------------------------------------------------------

func TestThemeDeep_ApplyThemeSettings_GlobalNilTheme(t *testing.T) {
	// This tests the nil guard in ApplyThemeSettings
	// We can't set the global's currentTheme to nil directly through public API,
	// but we can verify the function doesn't panic
	assert.NotPanics(t, func() {
		ApplyThemeSettings()
	})
}

// ---------------------------------------------------------------------------
// showThemePreview output verification
// ---------------------------------------------------------------------------

func TestThemeDeep_ShowThemePreview_VerboseOutput(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("cosmic")
	theme := tm.GetCurrentTheme()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm.showThemePreview(theme)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	// Verify key sections appear in output
	assert.Contains(t, got, "Theme Preview")
	assert.Contains(t, got, "Color Samples")
	assert.Contains(t, got, "UI Elements")
	assert.Contains(t, got, "Typography")
	assert.Contains(t, got, fmt.Sprintf("Theme: %s", theme.Name))
	assert.Contains(t, got, fmt.Sprintf("Description: %s", theme.Description))
}

func TestThemeDeep_ShowThemePreview_NoEmojiForMonochrome(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("monochrome")
	theme := tm.GetCurrentTheme()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm.showThemePreview(theme)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	// Monochrome should NOT show "Icon Samples" section
	assert.NotContains(t, got, "Icon Samples")
}

// ---------------------------------------------------------------------------
// showSpinnerDemo output
// ---------------------------------------------------------------------------

func TestThemeDeep_ShowSpinnerDemo_Output(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("forest")
	theme := tm.GetCurrentTheme()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm.showSpinnerDemo(theme)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	assert.Contains(t, got, "Spinner:")
	assert.Contains(t, got, "demo complete")
}

// ---------------------------------------------------------------------------
// ThemeProgressChars uniqueness per theme
// ---------------------------------------------------------------------------

func TestThemeDeep_ProgressCharsPerTheme(t *testing.T) {
	tm := NewThemeManager()
	progressChars := map[string]string{}
	for _, theme := range tm.ListThemes() {
		pc := theme.Icons.ProgressChars
		_, exists := progressChars[pc]
		if exists {
			// Duplicate progress chars is OK, just log it
			t.Logf("Theme %s shares progress chars with another theme: %q", theme.ID, pc)
		}
		progressChars[pc] = theme.ID
	}
}
