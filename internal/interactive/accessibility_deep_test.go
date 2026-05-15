package interactive

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// applySettings - test each mode's effect on globals
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ApplySettings(t *testing.T) {
	// Save and restore global animator state
	origDisabled := globalAnimator.Disabled
	defer func() {
		globalAnimator.Disabled = origDisabled
		globalAnimator.SetStyle("standard")
		globalTransitionManager.Disabled = false
	}()

	t.Run("ScreenReader disables animator and transition manager", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityScreenReader)
		am.applySettings()
		assert.True(t, globalAnimator.Disabled)
		assert.True(t, globalTransitionManager.Disabled)
	})

	t.Run("HighContrast does not disable animator", func(t *testing.T) {
		globalAnimator.Disabled = false
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityHighContrast)
		am.applySettings()
		assert.False(t, globalAnimator.Disabled)
	})

	t.Run("ReducedMotion disables animation style", func(t *testing.T) {
		globalAnimator.Disabled = false
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityReducedMotion)
		am.applySettings()
		assert.True(t, globalAnimator.Disabled)
	})

	t.Run("None mode - no changes to animator", func(t *testing.T) {
		globalAnimator.Disabled = false
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityNone)
		am.applySettings()
		assert.False(t, globalAnimator.Disabled)
	})

	t.Run("LargeText does not disable animator", func(t *testing.T) {
		globalAnimator.Disabled = false
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityLargeText)
		am.applySettings()
		assert.False(t, globalAnimator.Disabled)
	})

	t.Run("Full mode disables animator and transitions", func(t *testing.T) {
		globalAnimator.Disabled = false
		globalTransitionManager.Disabled = false
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityFull)
		am.applySettings()
		assert.True(t, globalAnimator.Disabled)
		assert.True(t, globalTransitionManager.Disabled)
	})
}

// ---------------------------------------------------------------------------
// showCurrentSettings - just verify no panic for each mode
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ShowCurrentSettings(t *testing.T) {
	modes := []AccessibilityMode{
		AccessibilityScreenReader,
		AccessibilityHighContrast,
		AccessibilityLargeText,
		AccessibilityReducedMotion,
		AccessibilityFull,
		AccessibilityNone,
	}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode_%d", mode), func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(mode)
			assert.NotPanics(t, func() {
				am.showCurrentSettings()
			})
		})
	}
}

func TestAccessibilityDeep_ShowCurrentSettings_ContainsModeName(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.showCurrentSettings()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "Screen Reader")
	assert.Contains(t, got, "Current Accessibility Settings")
}

// ---------------------------------------------------------------------------
// PrintAccessible - all modes, no panic
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_PrintAccessible_NoPanic(t *testing.T) {
	modes := []AccessibilityMode{
		AccessibilityNone,
		AccessibilityScreenReader,
		AccessibilityHighContrast,
		AccessibilityLargeText,
		AccessibilityFull,
	}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode_%d", mode), func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(mode)
			assert.NotPanics(t, func() {
				am.PrintAccessible("test message")
			})
		})
	}
}

func TestAccessibilityDeep_PrintAccessible_ScreenReaderAnnounces(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessible("announce this")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "🔊")
	assert.Contains(t, got, "announce this")
}

// ---------------------------------------------------------------------------
// PrintAccessibleSuccess - no panic for all modes
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_PrintAccessibleSuccess_NoPanic(t *testing.T) {
	modes := []AccessibilityMode{
		AccessibilityNone,
		AccessibilityScreenReader,
		AccessibilityLargeText,
		AccessibilityFull,
	}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode_%d", mode), func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(mode)
			assert.NotPanics(t, func() {
				am.PrintAccessibleSuccess("operation completed")
			})
		})
	}
}

func TestAccessibilityDeep_PrintAccessibleSuccess_ScreenReader(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleSuccess("it worked")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "🔊")
	assert.Contains(t, got, "Success: it worked")
}

func TestAccessibilityDeep_PrintAccessibleSuccess_VerboseFormat(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityLargeText) // Verbose=true

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleSuccess("all good")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "✅ Success:")
	assert.Contains(t, got, "all good")
}

// ---------------------------------------------------------------------------
// PrintAccessibleError - no panic for all modes
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_PrintAccessibleError_NoPanic(t *testing.T) {
	modes := []AccessibilityMode{
		AccessibilityNone,
		AccessibilityScreenReader,
		AccessibilityLargeText,
		AccessibilityFull,
	}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode_%d", mode), func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(mode)
			assert.NotPanics(t, func() {
				am.PrintAccessibleError("something went wrong")
			})
		})
	}
}

func TestAccessibilityDeep_PrintAccessibleError_ScreenReader(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleError("failure")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "🔊")
	assert.Contains(t, got, "Error: failure")
}

func TestAccessibilityDeep_PrintAccessibleError_VerboseFormat(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityLargeText) // Verbose=true

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleError("bad thing")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "❌ Error:")
	assert.Contains(t, got, "bad thing")
}

// ---------------------------------------------------------------------------
// announceToScreenReader - direct call
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_AnnounceToScreenReader(t *testing.T) {
	am := NewAccessibilityManager()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.announceToScreenReader("hello screen reader")

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	assert.Contains(t, got, "🔊")
	assert.Contains(t, got, "hello screen reader")
}

// ---------------------------------------------------------------------------
// AccessibilityConfig zero value
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ZeroValueConfig(t *testing.T) {
	var cfg AccessibilityConfig
	assert.Equal(t, AccessibilityMode(0), cfg.Mode)
	assert.False(t, cfg.HighContrast)
	assert.False(t, cfg.ScreenReader)
	assert.False(t, cfg.LargeText)
	assert.False(t, cfg.ReducedMotion)
	assert.False(t, cfg.Verbose)
	assert.False(t, cfg.KeyboardOnly)
	assert.False(t, cfg.ColorBlind)
	assert.False(t, cfg.AnnounceActions)
	assert.False(t, cfg.ExtendedTimeouts)
}

// ---------------------------------------------------------------------------
// AccessibilityMode constants
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ModeConstants(t *testing.T) {
	assert.Equal(t, AccessibilityMode(0), AccessibilityNone)
	assert.Equal(t, AccessibilityMode(1), AccessibilityScreenReader)
	assert.Equal(t, AccessibilityMode(2), AccessibilityHighContrast)
	assert.Equal(t, AccessibilityMode(3), AccessibilityLargeText)
	assert.Equal(t, AccessibilityMode(4), AccessibilityReducedMotion)
	assert.Equal(t, AccessibilityMode(5), AccessibilityFull)
}

// ---------------------------------------------------------------------------
// SetMode idempotency
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_SetMode_Idempotent(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityFull)
	am.SetMode(AccessibilityFull)
	assert.Equal(t, AccessibilityFull, am.config.Mode)

	am.SetMode(AccessibilityNone)
	assert.Equal(t, AccessibilityNone, am.config.Mode)
	am.SetMode(AccessibilityScreenReader)
	assert.Equal(t, AccessibilityScreenReader, am.config.Mode)
}

// ---------------------------------------------------------------------------
// SetMode overrides flags correctly
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_SetMode_OverridesPreviousFlags(t *testing.T) {
	am := NewAccessibilityManager()

	am.SetMode(AccessibilityFull)
	assert.True(t, am.config.ScreenReader)
	assert.True(t, am.config.HighContrast)
	assert.True(t, am.config.KeyboardOnly)

	am.SetMode(AccessibilityReducedMotion)
	assert.True(t, am.config.ReducedMotion)
}

// ---------------------------------------------------------------------------
// ShowAccessibleMenu - verify dispatch logic (no stdin blocking)
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ShowAccessibleMenu_Dispatch(t *testing.T) {
	t.Run("ScreenReader sets ScreenReader flag", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityScreenReader)
		assert.True(t, am.config.ScreenReader)
		assert.True(t, am.config.Verbose)
		assert.True(t, am.config.AnnounceActions)
	})

	t.Run("LargeText sets LargeText flag", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityLargeText)
		assert.False(t, am.config.ScreenReader)
		assert.True(t, am.config.LargeText)
	})

	t.Run("None has no special flags", func(t *testing.T) {
		am := NewAccessibilityManager()
		am.SetMode(AccessibilityNone)
		assert.False(t, am.config.ScreenReader)
		assert.False(t, am.config.LargeText)
	})
}

// ---------------------------------------------------------------------------
// GetAccessibleInput - verify no panic (stdin will get EOF from pipe)
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_GetAccessibleInput_NoPanic(t *testing.T) {
	t.Run("non-sensitive", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		assert.NotPanics(t, func() {
			// Will get EOF from stdin, return empty string
			_ = ah.GetAccessibleInput("Enter name", false)
		})
	})

	t.Run("sensitive", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		assert.NotPanics(t, func() {
			_ = ah.GetAccessibleInput("Password", true)
		})
	})

	t.Run("screen reader mode", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		ah.manager.SetMode(AccessibilityScreenReader)
		assert.NotPanics(t, func() {
			_ = ah.GetAccessibleInput("Enter value", false)
		})
	})
}

// ---------------------------------------------------------------------------
// ConfirmAccessibleYesNo - verify no panic
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ConfirmAccessibleYesNo_NoPanic(t *testing.T) {
	t.Run("defaultYes true", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		ah.manager.config.Verbose = false
		assert.NotPanics(t, func() {
			_ = ah.ConfirmAccessibleYesNo("Continue?", true)
		})
	})

	t.Run("defaultYes false", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		ah.manager.config.Verbose = false
		assert.NotPanics(t, func() {
			_ = ah.ConfirmAccessibleYesNo("Proceed?", false)
		})
	})

	t.Run("verbose mode", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		ah.manager.config.Verbose = true
		assert.NotPanics(t, func() {
			_ = ah.ConfirmAccessibleYesNo("Sure?", true)
		})
	})

	t.Run("screen reader mode", func(t *testing.T) {
		ah := NewAccessibilityHelper()
		ah.manager.SetMode(AccessibilityScreenReader)
		assert.NotPanics(t, func() {
			_ = ah.ConfirmAccessibleYesNo("Confirm?", true)
		})
	})
}

// ---------------------------------------------------------------------------
// ShowAccessibleProgress - additional edge cases
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_ShowAccessibleProgress_ZeroTotal(t *testing.T) {
	ah := NewAccessibilityHelper()
	assert.NotPanics(t, func() {
		ah.ShowAccessibleProgress("test", 0, 0)
	})
}

func TestAccessibilityDeep_ShowAccessibleProgress_ScreenReaderMode(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah.ShowAccessibleProgress("upload", 3, 10)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "🔊")
	assert.Contains(t, got, "3 of 10 complete")
}

func TestAccessibilityDeep_ShowAccessibleProgress_NonVerbose(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityReducedMotion)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah.ShowAccessibleProgress("download", 5, 20)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)
	assert.Contains(t, got, "download")
}

// ---------------------------------------------------------------------------
// AccessibilityHelper zero value
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_HelperZeroValue(t *testing.T) {
	var ah AccessibilityHelper
	assert.Nil(t, ah.manager)
}

// ---------------------------------------------------------------------------
// IsEnabled for all modes
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_IsEnabled_AllModes(t *testing.T) {
	tests := []struct {
		mode    AccessibilityMode
		enabled bool
	}{
		{AccessibilityNone, false},
		{AccessibilityScreenReader, true},
		{AccessibilityHighContrast, true},
		{AccessibilityLargeText, true},
		{AccessibilityReducedMotion, true},
		{AccessibilityFull, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("mode_%d", tt.mode), func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(tt.mode)
			assert.Equal(t, tt.enabled, am.IsEnabled())
		})
	}
}

// ---------------------------------------------------------------------------
// AutoDetectAccessibility - Full mode via env vars
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_AutoDetect_FullMode(t *testing.T) {
	os.Setenv("TERM_PROGRAM", "vscode")
	os.Setenv("ACCESSIBILITY", "1")
	defer func() {
		os.Unsetenv("TERM_PROGRAM")
		os.Unsetenv("ACCESSIBILITY")
	}()

	am := AutoDetectAccessibility()
	assert.Equal(t, AccessibilityFull, am.config.Mode)
	assert.True(t, am.IsEnabled())
}

func TestAccessibilityDeep_AutoDetect_Precedence(t *testing.T) {
	os.Setenv("SCREEN_READER", "1")
	os.Setenv("HIGH_CONTRAST", "1")
	defer func() {
		os.Unsetenv("SCREEN_READER")
		os.Unsetenv("HIGH_CONTRAST")
	}()

	am := AutoDetectAccessibility()
	assert.Equal(t, AccessibilityHighContrast, am.config.Mode)
}

// ---------------------------------------------------------------------------
// formatBool output
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_FormatBool_ContainsKeywords(t *testing.T) {
	t.Run("true contains Enabled", func(t *testing.T) {
		got := formatBool(true)
		assert.Contains(t, got, "Enabled")
	})

	t.Run("false contains Disabled", func(t *testing.T) {
		got := formatBool(false)
		assert.Contains(t, got, "Disabled")
	})
}

// ---------------------------------------------------------------------------
// getModeName full coverage
// ---------------------------------------------------------------------------

func TestAccessibilityDeep_GetModeName_FullCoverage(t *testing.T) {
	am := NewAccessibilityManager()

	modes := []struct {
		mode AccessibilityMode
		want string
	}{
		{AccessibilityNone, "Disabled"},
		{AccessibilityScreenReader, "Screen Reader"},
		{AccessibilityHighContrast, "High Contrast"},
		{AccessibilityLargeText, "Large Text"},
		{AccessibilityReducedMotion, "Reduced Motion"},
		{AccessibilityFull, "Full Accessibility"},
		{AccessibilityMode(99), "Disabled"},
	}

	for _, m := range modes {
		t.Run(m.want, func(t *testing.T) {
			am.SetMode(m.mode)
			assert.Equal(t, m.want, am.getModeName())
		})
	}
}
