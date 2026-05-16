package interactive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Easing functions
// ---------------------------------------------------------------------------

func TestEaseInOutCubic(t *testing.T) {
	t.Run("t=0 returns 0", func(t *testing.T) {
		result := easeInOutCubic(0)
		assert.Equal(t, 0.0, result)
	})

	t.Run("t=1 returns 1", func(t *testing.T) {
		result := easeInOutCubic(1)
		assert.Equal(t, 1.0, result)
	})

	t.Run("t=0.5 uses second branch", func(t *testing.T) {
		result := easeInOutCubic(0.5)
		assert.InDelta(t, 0.5, result, 0.01)
	})

	t.Run("t<0.5 produces smooth acceleration", func(t *testing.T) {
		result := easeInOutCubic(0.25)
		assert.InDelta(t, 0.0625, result, 0.01)
	})

	t.Run("t>0.5 produces output in [0,1]", func(t *testing.T) {
		result := easeInOutCubic(0.75)
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 1.0)
	})

	t.Run("negative input clamped to 0", func(t *testing.T) {
		result := easeInOutCubic(-0.1)
		assert.Equal(t, 0.0, result)
	})

	t.Run("input > 1 clamped to 1", func(t *testing.T) {
		result := easeInOutCubic(1.1)
		assert.Equal(t, 1.0, result)
	})

	t.Run("result never exceeds 1.0 across full range", func(t *testing.T) {
		for v := 0.0; v <= 1.0; v += 0.01 {
			result := easeInOutCubic(v)
			assert.GreaterOrEqual(t, result, 0.0, "for v=%v", v)
			assert.LessOrEqual(t, result, 1.0, "for v=%v", v)
		}
	})
}

func TestEaseOutQuad(t *testing.T) {
	t.Run("t=0 returns 0", func(t *testing.T) {
		result := easeOutQuad(0)
		assert.Equal(t, 0.0, result)
	})

	t.Run("t=1 returns 1", func(t *testing.T) {
		result := easeOutQuad(1)
		assert.Equal(t, 1.0, result)
	})

	t.Run("midpoint is above linear", func(t *testing.T) {
		result := easeOutQuad(0.5)
		assert.Greater(t, result, 0.5)
		assert.Less(t, result, 1.0)
	})
}

func TestEaseInQuad(t *testing.T) {
	t.Run("t=0 returns 0", func(t *testing.T) {
		result := easeInQuad(0)
		assert.Equal(t, 0.0, result)
	})

	t.Run("t=1 returns 1", func(t *testing.T) {
		result := easeInQuad(1)
		assert.Equal(t, 1.0, result)
	})

	t.Run("midpoint is below linear", func(t *testing.T) {
		result := easeInQuad(0.5)
		assert.Greater(t, result, 0.0)
		assert.Less(t, result, 0.5)
	})
}

// ---------------------------------------------------------------------------
// Animator creation
// ---------------------------------------------------------------------------

func TestNewAnimator(t *testing.T) {
	a := NewAnimator()
	assert.NotNil(t, a)
	assert.Equal(t, "standard", a.Style)
	assert.Equal(t, DefaultFrameRate, a.Speed)
	assert.False(t, a.Disabled)
	assert.Equal(t, 0, a.frameCount)
}

// ---------------------------------------------------------------------------
// ShowSpinner (disabled mode for testability)
// ---------------------------------------------------------------------------

func TestAnimator_ShowSpinner_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	// Should not panic and return immediately
	assert.NotPanics(t, func() {
		a.ShowSpinner("Test message", 100*time.Millisecond)
	})
}

func TestAnimator_ShowSpinner_VeryShortDuration(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true // Disable to avoid actual animation in tests

	assert.NotPanics(t, func() {
		a.ShowSpinner("Quick", 1*time.Millisecond)
	})
}

// ---------------------------------------------------------------------------
// ShowProgress (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_ShowProgress_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	steps := []string{"Step 1", "Step 2", "Step 3"}

	assert.NotPanics(t, func() {
		a.ShowProgress("Processing", steps)
	})
}

func TestAnimator_ShowProgress_EmptySteps(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowProgress("No steps", []string{})
	})
}

func TestAnimator_ShowProgress_SingleStep(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowProgress("Single", []string{"Only step"})
	})
}

// ---------------------------------------------------------------------------
// drawProgressBar
// ---------------------------------------------------------------------------

func TestAnimator_DrawProgressBar(t *testing.T) {
	a := NewAnimator()
	style := ProgressBarCharacters["standard"]

	t.Run("0% progress", func(t *testing.T) {
		// Just verify it doesn't panic
		assert.NotPanics(t, func() {
			a.drawProgressBar(0, style, "Test")
		})
	})

	t.Run("50% progress", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawProgressBar(0.5, style, "Half")
		})
	})

	t.Run("100% progress", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawProgressBar(1.0, style, "Complete")
		})
	})

	t.Run("boundary values", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawProgressBar(0.001, style, "Min")
			a.drawProgressBar(0.999, style, "Max")
		})
	})
}

// ---------------------------------------------------------------------------
// AnimateTransition (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_AnimateTransition_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.AnimateTransition("From", "To", 100*time.Millisecond)
	})
}

func TestAnimator_AnimateTransition_ZeroDuration(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.AnimateTransition("A", "B", 0)
	})
}

// ---------------------------------------------------------------------------
// mixTexts
// ---------------------------------------------------------------------------

func TestAnimator_MixTexts(t *testing.T) {
	a := NewAnimator()

	t.Run("equal length strings", func(t *testing.T) {
		result := a.mixTexts("hello", "world", 0.3, 0.7)
		// Result should be one of the characters at each position
		assert.Equal(t, 5, len(result))
	})

	t.Run("different length strings - padding", func(t *testing.T) {
		result := a.mixTexts("hi", "hello", 0.5, 0.5)
		// Should pad shorter string
		assert.Equal(t, 5, len(result))
	})

	t.Run("fade out dominant", func(t *testing.T) {
		result := a.mixTexts("from", "to", 0.9, 0.1)
		// Should favor from text
		assert.NotEmpty(t, result)
	})

	t.Run("fade in dominant", func(t *testing.T) {
		result := a.mixTexts("from", "to", 0.1, 0.9)
		// Should favor to text
		assert.NotEmpty(t, result)
	})

	t.Run("empty strings", func(t *testing.T) {
		result := a.mixTexts("", "", 0.5, 0.5)
		assert.Equal(t, "", result)
	})
}

// ---------------------------------------------------------------------------
// ShowLoadingSkeleton (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_ShowLoadingSkeleton_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	lines := []string{"Line 1", "Line 2", "Line 3"}

	assert.NotPanics(t, func() {
		a.ShowLoadingSkeleton(lines, 100*time.Millisecond)
	})
}

func TestAnimator_ShowLoadingSkeleton_EmptyLines(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowLoadingSkeleton([]string{}, 100*time.Millisecond)
	})
}

// ---------------------------------------------------------------------------
// TypewriterEffect (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_TypewriterEffect_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.TypewriterEffect("Hello, World!", 10*time.Millisecond)
	})
}

func TestAnimator_TypewriterEffect_EmptyString(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.TypewriterEffect("", 10*time.Millisecond)
	})
}

func TestAnimator_TypewriterEffect_Unicode(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.TypewriterEffect("Hello 世界 🌍", 10*time.Millisecond)
	})
}

// ---------------------------------------------------------------------------
// PulseText (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_PulseText_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.PulseText("Important!", 1)
	})
}

func TestAnimator_PulseText_ZeroPulses(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.PulseText("No pulse", 0)
	})
}

func TestAnimator_PulseText_MultiplePulses(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.PulseText("Multi pulse", 3)
	})
}

// ---------------------------------------------------------------------------
// ShowStepTransition (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_ShowStepTransition_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowStepTransition(1, 2, "Step One", "Step Two")
	})
}

func TestAnimator_ShowStepTransition_EdgeCases(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowStepTransition(0, 1, "", "")
		a.ShowStepTransition(99, 100, "99", "100")
	})
}

// ---------------------------------------------------------------------------
// ShowSuccessAnimation (disabled mode)
// ---------------------------------------------------------------------------

func TestAnimator_ShowSuccessAnimation_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowSuccessAnimation("Operation completed!")
	})
}

func TestAnimator_ShowSuccessAnimation_EmptyMessage(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowSuccessAnimation("")
	})
}

// ---------------------------------------------------------------------------
// SetStyle
// ---------------------------------------------------------------------------

func TestAnimator_SetStyle(t *testing.T) {
	a := NewAnimator()

	t.Run("fast style", func(t *testing.T) {
		a.SetStyle("fast")
		assert.Equal(t, FastFrameRate, a.Speed)
	})

	t.Run("slow style", func(t *testing.T) {
		a.SetStyle("slow")
		assert.Equal(t, SlowFrameRate, a.Speed)
	})

	t.Run("disabled style", func(t *testing.T) {
		a.SetStyle("disabled")
		assert.True(t, a.Disabled)
	})

	t.Run("unknown style defaults to standard", func(t *testing.T) {
		a := NewAnimator()
		a.SetStyle("unknown")
		assert.Equal(t, DefaultFrameRate, a.Speed)
		assert.False(t, a.Disabled)
	})
}

// ---------------------------------------------------------------------------
// max helper
// ---------------------------------------------------------------------------

func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{-1, -5, -1},
		{0, 0, 0},
		{100, 100, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := max(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Global functions
// ---------------------------------------------------------------------------

func TestSetAnimationStyle(t *testing.T) {
	originalSpeed := globalAnimator.Speed
	originalDisabled := globalAnimator.Disabled

	defer func() {
		globalAnimator.Speed = originalSpeed
		globalAnimator.Disabled = originalDisabled
	}()

	SetAnimationStyle("fast")
	assert.Equal(t, FastFrameRate, globalAnimator.Speed)

	SetAnimationStyle("disabled")
	assert.True(t, globalAnimator.Disabled)
}

// ---------------------------------------------------------------------------
// ProgressBarCharacters
// ---------------------------------------------------------------------------

func TestProgressBarCharacters(t *testing.T) {
	t.Run("standard style exists", func(t *testing.T) {
		style, ok := ProgressBarCharacters["standard"]
		assert.True(t, ok)
		assert.NotEmpty(t, style.Empty)
		assert.NotEmpty(t, style.Fill)
		assert.NotEmpty(t, style.Start)
		assert.NotEmpty(t, style.End)
		assert.NotEmpty(t, style.Head)
	})

	t.Run("blocks style exists", func(t *testing.T) {
		style, ok := ProgressBarCharacters["blocks"]
		assert.True(t, ok)
		assert.Equal(t, "░", style.Empty)
		assert.Equal(t, "▓", style.Fill)
	})

	t.Run("dots style exists", func(t *testing.T) {
		style, ok := ProgressBarCharacters["dots"]
		assert.True(t, ok)
		assert.Equal(t, " ", style.Empty)
		assert.Equal(t, "●", style.Fill)
	})

	t.Run("arrows style exists", func(t *testing.T) {
		style, ok := ProgressBarCharacters["arrows"]
		assert.True(t, ok)
		assert.Equal(t, "-", style.Empty)
		assert.Equal(t, ">", style.Fill)
	})
}

// ---------------------------------------------------------------------------
// SpinnerCharacters
// ---------------------------------------------------------------------------

func TestSpinnerCharacters(t *testing.T) {
	assert.NotEmpty(t, SpinnerCharacters)
	assert.Greater(t, len(SpinnerCharacters), 5)

	// All characters should be unique
	seen := make(map[rune]bool)
	for _, s := range SpinnerCharacters {
		r := []rune(s)[0]
		if seen[r] {
			t.Errorf("Duplicate spinner character: %c", r)
		}
		seen[r] = true
	}
}

func TestDotsSpinnerCharacters(t *testing.T) {
	assert.NotEmpty(t, DotsSpinnerCharacters)
	assert.Greater(t, len(DotsSpinnerCharacters), 3)

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, s := range DotsSpinnerCharacters {
		if seen[s] {
			t.Errorf("Duplicate dots spinner character: %s", s)
		}
		seen[s] = true
	}
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestAnimationConstants(t *testing.T) {
	assert.Equal(t, 60*time.Millisecond, DefaultFrameRate)
	assert.Equal(t, 30*time.Millisecond, FastFrameRate)
	assert.Equal(t, 100*time.Millisecond, SlowFrameRate)
}

// ---------------------------------------------------------------------------
// AnimationState
// ---------------------------------------------------------------------------

func TestAnimationState(t *testing.T) {
	state := AnimationState{
		Progress: 0.5,
		Message:  "Loading...",
		Speed:    100 * time.Millisecond,
		Complete: false,
	}

	assert.Equal(t, 0.5, state.Progress)
	assert.Equal(t, "Loading...", state.Message)
	assert.Equal(t, 100*time.Millisecond, state.Speed)
	assert.False(t, state.Complete)

	state.Complete = true
	assert.True(t, state.Complete)
}

// ---------------------------------------------------------------------------
// drawPulsedText
// ---------------------------------------------------------------------------

func TestAnimator_DrawPulsedText(t *testing.T) {
	a := NewAnimator()

	t.Run("high intensity", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawPulsedText("Test", 0.9)
		})
	})

	t.Run("low intensity", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawPulsedText("Test", 0.1)
		})
	})

	t.Run("mid intensity", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawPulsedText("Test", 0.5)
		})
	})

	t.Run("empty string", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawPulsedText("", 0.7)
		})
	})

	t.Run("boundary intensities", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawPulsedText("Test", 0.0)
			a.drawPulsedText("Test", 1.0)
		})
	})
}

// ---------------------------------------------------------------------------
// drawSkeleton
// ---------------------------------------------------------------------------

func TestAnimator_DrawSkeleton(t *testing.T) {
	a := NewAnimator()

	t.Run("single line", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawSkeleton([]string{"Line 1"}, 0)
		})
	})

	t.Run("multiple lines", func(t *testing.T) {
		assert.NotPanics(t, func() {
			a.drawSkeleton([]string{"L1", "L2", "L3"}, 0)
		})
	})

	t.Run("different frames", func(t *testing.T) {
		lines := []string{"Test"}
		for i := 0; i < 4; i++ {
			assert.NotPanics(t, func() {
				a.drawSkeleton(lines, i)
			})
		}
	})
}

// renderWithOffset is tested in transitions_test.go
