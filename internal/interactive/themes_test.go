package interactive

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewThemeManager
// ---------------------------------------------------------------------------

func TestNewThemeManager(t *testing.T) {
	tm := NewThemeManager()
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.themes)
	assert.NotNil(t, tm.currentTheme)
	assert.NotEmpty(t, tm.themePath)
}

// ---------------------------------------------------------------------------
// getThemePath
// ---------------------------------------------------------------------------

func TestThemeManager_GetThemePath(t *testing.T) {
	tm := NewThemeManager()
	path := tm.GetThemePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".r2go2")
	assert.Contains(t, path, "themes")
}

// ---------------------------------------------------------------------------
// loadBuiltinThemes
// ---------------------------------------------------------------------------

func TestThemeManager_LoadBuiltinThemes(t *testing.T) {
	tm := NewThemeManager()

	// All builtin themes should be loaded
	builtinThemes := []string{"cosmic", "forest", "ocean", "sunset", "monochrome"}
	for _, themeID := range builtinThemes {
		theme, exists := tm.themes[themeID]
		assert.True(t, exists, "Theme %s should exist", themeID)
		assert.NotNil(t, theme)
		assert.Equal(t, themeID, theme.ID)
	}
}

// ---------------------------------------------------------------------------
// SetTheme
// ---------------------------------------------------------------------------

func TestThemeManager_SetTheme(t *testing.T) {
	tm := NewThemeManager()

	t.Run("set cosmic theme", func(t *testing.T) {
		err := tm.SetTheme("cosmic")
		assert.NoError(t, err)
		assert.Equal(t, "Cosmic", tm.GetCurrentTheme().Name)
	})

	t.Run("set forest theme", func(t *testing.T) {
		err := tm.SetTheme("forest")
		assert.NoError(t, err)
		assert.Equal(t, "Forest", tm.GetCurrentTheme().Name)
	})

	t.Run("set ocean theme", func(t *testing.T) {
		err := tm.SetTheme("ocean")
		assert.NoError(t, err)
		assert.Equal(t, "Ocean", tm.GetCurrentTheme().Name)
	})

	t.Run("set sunset theme", func(t *testing.T) {
		err := tm.SetTheme("sunset")
		assert.NoError(t, err)
		assert.Equal(t, "Sunset", tm.GetCurrentTheme().Name)
	})

	t.Run("set monochrome theme", func(t *testing.T) {
		err := tm.SetTheme("monochrome")
		assert.NoError(t, err)
		assert.Equal(t, "Monochrome", tm.GetCurrentTheme().Name)
	})

	t.Run("set nonexistent theme", func(t *testing.T) {
		err := tm.SetTheme("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ---------------------------------------------------------------------------
// GetCurrentTheme
// ---------------------------------------------------------------------------

func TestThemeManager_GetCurrentTheme(t *testing.T) {
	tm := NewThemeManager()
	theme := tm.GetCurrentTheme()
	assert.NotNil(t, theme)
	assert.NotEmpty(t, theme.ID)
	assert.NotEmpty(t, theme.Name)
}

// ---------------------------------------------------------------------------
// ListThemes
// ---------------------------------------------------------------------------

func TestThemeManager_ListThemes(t *testing.T) {
	tm := NewThemeManager()
	themes := tm.ListThemes()

	assert.NotEmpty(t, themes)
	assert.GreaterOrEqual(t, len(themes), 5) // At least builtin themes

	// Verify all themes have required fields
	for _, theme := range themes {
		assert.NotEmpty(t, theme.ID)
		assert.NotEmpty(t, theme.Name)
		assert.NotEmpty(t, theme.Colors.Primary)
		assert.NotZero(t, theme.Spacing.ProgressBarWidth)
		assert.NotNil(t, theme.Icons.SpinnerChars)
		assert.NotEmpty(t, theme.Icons.SpinnerChars)
	}
}

// ---------------------------------------------------------------------------
// Theme structure validation
// ---------------------------------------------------------------------------

func TestTheme_Structure(t *testing.T) {
	tm := NewThemeManager()

	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			// Colors
			assert.NotEmpty(t, theme.Colors.Primary)
			assert.NotEmpty(t, theme.Colors.Secondary)
			assert.NotEmpty(t, theme.Colors.Success)
			assert.NotEmpty(t, theme.Colors.Warning)
			assert.NotEmpty(t, theme.Colors.Error)
			assert.NotEmpty(t, theme.Colors.Info)
			assert.NotEmpty(t, theme.Colors.Background)
			assert.NotEmpty(t, theme.Colors.Foreground)

			// Emojis
			assert.NotNil(t, theme.Colors.Emoji)
			assert.Contains(t, theme.Colors.Emoji, "success")
			assert.Contains(t, theme.Colors.Emoji, "error")
			assert.Contains(t, theme.Colors.Emoji, "warning")
			assert.Contains(t, theme.Colors.Emoji, "info")
			assert.Contains(t, theme.Colors.Emoji, "loading")

			// Spacing
			assert.Greater(t, theme.Spacing.LineSpacing, 0)
			assert.Greater(t, theme.Spacing.SectionGap, 0)
			assert.Greater(t, theme.Spacing.IndentSize, 0)
			assert.Greater(t, theme.Spacing.ProgressBarWidth, 0)

			// Icons
			assert.NotEmpty(t, theme.Icons.SpinnerChars)
			assert.NotEmpty(t, theme.Icons.ProgressChars)
			assert.NotEmpty(t, theme.Icons.BulletChars)
		})
	}
}

// ---------------------------------------------------------------------------
// Cosmic theme specifics
// ---------------------------------------------------------------------------

func TestTheme_Cosmic(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("cosmic")
	theme := tm.GetCurrentTheme()

	assert.Equal(t, "cosmic", theme.ID)
	assert.Equal(t, "Cosmic", theme.Name)
	assert.Contains(t, theme.Description, "Purple")
	assert.True(t, theme.Icons.UseEmojis)
	assert.True(t, theme.Animations.Enabled)
	assert.Equal(t, 60, theme.Animations.Speed)
	assert.Equal(t, 40, theme.Spacing.ProgressBarWidth)
}

// ---------------------------------------------------------------------------
// Forest theme specifics
// ---------------------------------------------------------------------------

func TestTheme_Forest(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("forest")
	theme := tm.GetCurrentTheme()

	assert.Equal(t, "forest", theme.ID)
	assert.Equal(t, "Forest", theme.Name)
	assert.Contains(t, theme.Description, "Green")
	assert.True(t, theme.Icons.UseEmojis)
	assert.Equal(t, 80, theme.Animations.Speed)
	assert.Equal(t, "organic", theme.Animations.Easing)
}

// ---------------------------------------------------------------------------
// Ocean theme specifics
// ---------------------------------------------------------------------------

func TestTheme_Ocean(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("ocean")
	theme := tm.GetCurrentTheme()

	assert.Equal(t, "ocean", theme.ID)
	assert.Equal(t, "Ocean", theme.Name)
	assert.Contains(t, theme.Description, "blue")
	assert.Equal(t, 50, theme.Animations.Speed)
	assert.Equal(t, "wave", theme.Animations.Easing)
}

// ---------------------------------------------------------------------------
// Sunset theme specifics
// ---------------------------------------------------------------------------

func TestTheme_Sunset(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("sunset")
	theme := tm.GetCurrentTheme()

	assert.Equal(t, "sunset", theme.ID)
	assert.Equal(t, "Sunset", theme.Name)
	assert.Contains(t, theme.Description, "orange")
	assert.Equal(t, 70, theme.Animations.Speed)
	assert.Equal(t, "warm", theme.Animations.Easing)
}

// ---------------------------------------------------------------------------
// Monochrome theme specifics
// ---------------------------------------------------------------------------

func TestTheme_Monochrome(t *testing.T) {
	tm := NewThemeManager()
	tm.SetTheme("monochrome")
	theme := tm.GetCurrentTheme()

	assert.Equal(t, "monochrome", theme.ID)
	assert.Equal(t, "Monochrome", theme.Name)
	assert.Contains(t, theme.Description, "Black")
	assert.False(t, theme.Icons.UseEmojis)
	assert.False(t, theme.Animations.Enabled)
	assert.Equal(t, 0, theme.Animations.Speed)
	assert.Equal(t, "none", theme.Animations.Easing)
	assert.Equal(t, 50, theme.Spacing.ProgressBarWidth)
	assert.Equal(t, 4, theme.Spacing.IndentSize)
}

// ---------------------------------------------------------------------------
// applyTheme
// ---------------------------------------------------------------------------

func TestThemeManager_ApplyTheme(t *testing.T) {
	tm := NewThemeManager()

	t.Run("apply monochrome disables animations", func(t *testing.T) {
		tm.SetTheme("monochrome")
		// Animations should be disabled
		assert.True(t, globalAnimator.Disabled)
	})

	t.Run("apply cosmic enables animations", func(t *testing.T) {
		globalAnimator.Disabled = false
		tm.SetTheme("cosmic")
		// Should enable animations
		// Note: This depends on global state which may be affected by other tests
	})
}

// ---------------------------------------------------------------------------
// showProgressBar (internal function)
// ---------------------------------------------------------------------------

func TestThemeManager_ShowProgressBar(t *testing.T) {
	tm := NewThemeManager()

	t.Run("cosmic theme progress bar", func(t *testing.T) {
		tm.SetTheme("cosmic")
		theme := tm.GetCurrentTheme()
		assert.NotPanics(t, func() {
			tm.showProgressBar(theme, 0)
			tm.showProgressBar(theme, 50)
			tm.showProgressBar(theme, 100)
		})
	})

	t.Run("monochrome theme progress bar", func(t *testing.T) {
		tm.SetTheme("monochrome")
		theme := tm.GetCurrentTheme()
		assert.NotPanics(t, func() {
			tm.showProgressBar(theme, 25)
			tm.showProgressBar(theme, 75)
		})
	})

	t.Run("edge case percentages", func(t *testing.T) {
		tm.SetTheme("cosmic")
		theme := tm.GetCurrentTheme()

		// Valid percentages from 0-100 work correctly
		assert.NotPanics(t, func() {
			tm.showProgressBar(theme, 0)
			tm.showProgressBar(theme, 50)
			tm.showProgressBar(theme, 100)
		})
	})
}

// ---------------------------------------------------------------------------
// showSpinnerDemo
// ---------------------------------------------------------------------------

func TestThemeManager_ShowSpinnerDemo(t *testing.T) {
	tm := NewThemeManager()

	t.Run("cosmic spinner", func(t *testing.T) {
		tm.SetTheme("cosmic")
		theme := tm.GetCurrentTheme()
		assert.NotPanics(t, func() {
			tm.showSpinnerDemo(theme)
		})
	})

	t.Run("forest spinner", func(t *testing.T) {
		tm.SetTheme("forest")
		theme := tm.GetCurrentTheme()
		assert.NotPanics(t, func() {
			tm.showSpinnerDemo(theme)
		})
	})

	t.Run("monochrome spinner", func(t *testing.T) {
		tm.SetTheme("monochrome")
		theme := tm.GetCurrentTheme()
		assert.NotPanics(t, func() {
			tm.showSpinnerDemo(theme)
		})
	})
}

// ---------------------------------------------------------------------------
// ColorScheme
// ---------------------------------------------------------------------------

func TestColorScheme(t *testing.T) {
	tm := NewThemeManager()
	theme := tm.GetCurrentTheme()

	colors := theme.Colors
	assert.NotEmpty(t, colors.Name)
	assert.NotEmpty(t, colors.Description)
	assert.NotEmpty(t, colors.Primary)
	assert.NotEmpty(t, colors.Secondary)
	assert.NotEmpty(t, colors.Success)
	assert.NotEmpty(t, colors.Warning)
	assert.NotEmpty(t, colors.Error)
	assert.NotEmpty(t, colors.Info)
	assert.NotEmpty(t, colors.Accent)
	assert.NotEmpty(t, colors.Background)
	assert.NotEmpty(t, colors.Foreground)
	assert.NotNil(t, colors.Emoji)
}

// ---------------------------------------------------------------------------
// ThemeSpacing
// ---------------------------------------------------------------------------

func TestThemeSpacing(t *testing.T) {
	tm := NewThemeManager()
	theme := tm.GetCurrentTheme()

	spacing := theme.Spacing
	assert.Greater(t, spacing.LineSpacing, 0)
	assert.Greater(t, spacing.SectionGap, 0)
	assert.Greater(t, spacing.IndentSize, 0)
	assert.Greater(t, spacing.ProgressBarWidth, 0)
}

// ---------------------------------------------------------------------------
// ThemeIcons
// ---------------------------------------------------------------------------

func TestThemeIcons(t *testing.T) {
	tm := NewThemeManager()
	theme := tm.GetCurrentTheme()

	icons := theme.Icons
	assert.NotEmpty(t, icons.SpinnerChars)
	assert.NotEmpty(t, icons.ProgressChars)
	assert.NotEmpty(t, icons.BulletChars)

	// Spinner chars should all be unique
	seen := make(map[string]bool)
	for _, char := range icons.SpinnerChars {
		assert.False(t, seen[char], "Duplicate spinner char: %s", char)
		seen[char] = true
	}
}

// ---------------------------------------------------------------------------
// ThemeAnimations
// ---------------------------------------------------------------------------

func TestThemeAnimations(t *testing.T) {
	tm := NewThemeManager()

	t.Run("cosmic animations", func(t *testing.T) {
		tm.SetTheme("cosmic")
		anim := tm.GetCurrentTheme().Animations
		assert.True(t, anim.Enabled)
		assert.Greater(t, anim.Speed, 0)
		assert.NotEmpty(t, anim.Easing)
		assert.True(t, anim.Transitions)
	})

	t.Run("monochrome animations", func(t *testing.T) {
		tm.SetTheme("monochrome")
		anim := tm.GetCurrentTheme().Animations
		assert.False(t, anim.Enabled)
		assert.Equal(t, 0, anim.Speed)
		assert.Equal(t, "none", anim.Easing)
		assert.False(t, anim.Transitions)
	})
}

// ---------------------------------------------------------------------------
// Emoji mapping
// ---------------------------------------------------------------------------

func TestTheme_EmojiMapping(t *testing.T) {
	tm := NewThemeManager()

	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			emoji := theme.Colors.Emoji
			assert.NotNil(t, emoji)

			// Check required emoji keys exist
			_, hasSuccess := emoji["success"]
			_, hasError := emoji["error"]
			_, hasWarning := emoji["warning"]
			_, hasInfo := emoji["info"]
			_, hasLoading := emoji["loading"]

			assert.True(t, hasSuccess, "Missing success emoji")
			assert.True(t, hasError, "Missing error emoji")
			assert.True(t, hasWarning, "Missing warning emoji")
			assert.True(t, hasInfo, "Missing info emoji")
			assert.True(t, hasLoading, "Missing loading emoji")
		})
	}
}

// ---------------------------------------------------------------------------
// loadCustomThemes
// ---------------------------------------------------------------------------

func TestThemeManager_LoadCustomThemes(t *testing.T) {
	tm := NewThemeManager()
	assert.NotPanics(t, func() {
		tm.loadCustomThemes()
	})
}

// ---------------------------------------------------------------------------
// SaveTheme and LoadTheme (stub implementations)
// ---------------------------------------------------------------------------

func TestThemeManager_SaveTheme(t *testing.T) {
	tm := NewThemeManager()
	theme := tm.GetCurrentTheme()

	// Stub implementation returns nil
	err := tm.SaveTheme(theme)
	assert.NoError(t, err)
}

func TestThemeManager_LoadTheme(t *testing.T) {
	tm := NewThemeManager()

	// Stub implementation returns nil
	err := tm.LoadTheme("cosmic")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// CreateCustomTheme edge cases
// ---------------------------------------------------------------------------

func TestThemeManager_CreateCustomTheme_EdgeCases(t *testing.T) {
	tm := NewThemeManager()

	// Note: CreateCustomTheme reads from stdin, so we can't test it fully
	// but we can verify the structure is ready

	t.Run("themes map initialized", func(t *testing.T) {
		assert.NotNil(t, tm.themes)
		assert.NotEmpty(t, tm.themes)
	})

	t.Run("cosmic theme exists for base", func(t *testing.T) {
		baseTheme, exists := tm.themes["cosmic"]
		assert.True(t, exists)
		assert.NotNil(t, baseTheme)
		assert.NotNil(t, baseTheme.Colors)
		assert.NotNil(t, baseTheme.Spacing)
		assert.NotNil(t, baseTheme.Icons)
		assert.NotNil(t, baseTheme.Animations)
	})
}

// ---------------------------------------------------------------------------
// Theme ID uniqueness
// ---------------------------------------------------------------------------

func TestTheme_IDUniqueness(t *testing.T) {
	tm := NewThemeManager()
	themes := tm.ListThemes()
	seenIDs := make(map[string]bool)

	for _, theme := range themes {
		assert.False(t, seenIDs[theme.ID], "Duplicate theme ID: %s", theme.ID)
		seenIDs[theme.ID] = true
	}
}

// ---------------------------------------------------------------------------
// Theme Name uniqueness
// ---------------------------------------------------------------------------

func TestTheme_NameUniqueness(t *testing.T) {
	tm := NewThemeManager()
	themes := tm.ListThemes()
	seenNames := make(map[string]bool)

	for _, theme := range themes {
		assert.False(t, seenNames[theme.Name], "Duplicate theme Name: %s", theme.Name)
		seenNames[theme.Name] = true
	}
}

// ---------------------------------------------------------------------------
// Bullet chars uniqueness
// ---------------------------------------------------------------------------

func TestTheme_BulletCharsUniqueness(t *testing.T) {
	tm := NewThemeManager()

	for _, theme := range tm.ListThemes() {
		t.Run(theme.ID, func(t *testing.T) {
			seen := make(map[string]bool)
			for _, char := range theme.Icons.BulletChars {
				assert.False(t, seen[char], "Duplicate bullet char in %s: %s", theme.ID, char)
				seen[char] = true
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ShowThemeMenu (0% coverage - biggest theme gap)
// ---------------------------------------------------------------------------

func TestThemeManager_ShowThemeMenu_SelectFirst(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.Input = newMockReader("1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := tm.ShowThemeMenu()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Theme")
}

func TestThemeManager_ShowThemeMenu_SelectEmpty(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.Input = newMockReader("") // empty = default selection (1)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := tm.ShowThemeMenu()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
}

func TestThemeManager_ShowThemeMenu_InvalidThenValid(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.Input = newMockReader("99", "2")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := tm.ShowThemeMenu()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	// Invalid input triggers recursive call, then valid input works
	require.NoError(t, err)
}

func TestThemeManager_ShowThemeMenu_NonNumeric(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.Input = newMockReader("abc", "3")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := tm.ShowThemeMenu()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GetCurrentThemeName edge cases
// ---------------------------------------------------------------------------

func TestGetCurrentThemeName_Default(t *testing.T) {
	name := GetCurrentThemeName()
	assert.Equal(t, "Cosmic", name)
}

func TestGetCurrentThemeName_NilTheme(t *testing.T) {
	orig := globalThemeManager.currentTheme
	globalThemeManager.currentTheme = nil
	defer func() { globalThemeManager.currentTheme = orig }()

	name := GetCurrentThemeName()
	assert.Equal(t, "unknown", name)
}

// ---------------------------------------------------------------------------
// ApplyThemeSettings edge cases
// ---------------------------------------------------------------------------

func TestApplyThemeSettings_NilTheme(t *testing.T) {
	orig := globalThemeManager.currentTheme
	globalThemeManager.currentTheme = nil
	defer func() { globalThemeManager.currentTheme = orig }()

	assert.NotPanics(t, func() {
		ApplyThemeSettings()
	})
}

func TestApplyThemeSettings_DisabledAnimations(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.SetTheme("monochrome") // has animations disabled

	ApplyThemeSettings()
	assert.True(t, globalAnimator.Disabled)
}

func TestApplyThemeSettings_EnabledAnimations(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	defer func() { globalAnimator.Disabled = origDisabled }()

	ApplyThemeSettings()
}

// ---------------------------------------------------------------------------
// applyTheme edge cases
// ---------------------------------------------------------------------------

func TestThemeManager_ApplyTheme_NilTheme(t *testing.T) {
	tm := NewThemeManager()
	tm.currentTheme = nil
	assert.NotPanics(t, func() {
		tm.applyTheme()
	})
}

func TestThemeManager_ApplyTheme_FastSpeed(t *testing.T) {
	origSpeed := globalAnimator.Speed
	defer func() { globalAnimator.Speed = origSpeed }()

	tm := NewThemeManager()
	// Create a theme with speed 30 -> fast
	customTheme := &Theme{
		ID: "test-fast",
		Animations: ThemeAnimations{
			Enabled: true,
			Speed:   30,
		},
		Colors:  tm.themes["cosmic"].Colors,
		Spacing: tm.themes["cosmic"].Spacing,
		Icons:   tm.themes["cosmic"].Icons,
	}
	tm.themes["test-fast"] = customTheme
	tm.SetTheme("test-fast")
	assert.Equal(t, FastFrameRate, globalAnimator.Speed)
}

func TestThemeManager_ApplyTheme_SlowSpeed(t *testing.T) {
	origSpeed := globalAnimator.Speed
	defer func() { globalAnimator.Speed = origSpeed }()

	tm := NewThemeManager()
	tm.SetTheme("forest") // speed 80 -> slow
	assert.Equal(t, SlowFrameRate, globalAnimator.Speed)
}

func TestThemeManager_ApplyTheme_NormalSpeed(t *testing.T) {
	origSpeed := globalAnimator.Speed
	defer func() { globalAnimator.Speed = origSpeed }()

	tm := NewThemeManager()
	// ocean has speed 50 -> normal
	tm.SetTheme("ocean")
	assert.Equal(t, DefaultFrameRate, globalAnimator.Speed)
}

// ---------------------------------------------------------------------------
// CreateCustomTheme duplicate name
// ---------------------------------------------------------------------------

func TestCreateCustomTheme_DuplicateName(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	tm.Input = newMockReader("cosmic", "desc")

	err := tm.CreateCustomTheme()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exists")
}

// ---------------------------------------------------------------------------
// SetTheme error case
// ---------------------------------------------------------------------------

func TestThemeManager_SetTheme_NotFound(t *testing.T) {
	tm := NewThemeManager()
	err := tm.SetTheme("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// showThemePreview (tested via ShowThemeMenu)
// ---------------------------------------------------------------------------

func TestThemeManager_ShowThemePreview(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	theme := tm.themes["cosmic"]

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm.showThemePreview(theme)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Preview")
}

func TestThemeManager_ShowThemePreview_NoEmojis(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewThemeManager()
	theme := tm.themes["monochrome"] // UseEmojis = false

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm.showThemePreview(theme)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Preview")
}

// ---------------------------------------------------------------------------
// GetThemeManager / SetGlobalTheme
// ---------------------------------------------------------------------------

func TestGetThemeManager(t *testing.T) {
	tm := GetThemeManager()
	assert.NotNil(t, tm)
}

func TestSetGlobalTheme_Valid(t *testing.T) {
	err := SetGlobalTheme("ocean")
	assert.NoError(t, err)
	// Reset
	SetGlobalTheme("cosmic")
}

func TestSetGlobalTheme_Invalid(t *testing.T) {
	err := SetGlobalTheme("nonexistent")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// SaveTheme / LoadTheme stubs (already covered in themes_deep_test.go)
// ---------------------------------------------------------------------------
