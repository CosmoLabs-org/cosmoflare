package interactive

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Easing functions
// ---------------------------------------------------------------------------

// TestEaseInOutCubic verifies easeInOutCubic behavior, one t.Run subtest per scenario.
func TestEaseInOutCubic(t *testing.T) {
	t.Run("t=0 returns 0", func(t *testing.T) {
		result := easeInOutCubic(0)
		assert.Equal(t, 0.0, result)
	})

	t.Run("t=1 returns 1", func(t *testing.T) {
		result := easeInOutCubic(1)
		assert.Equal(t, 1.0, result)
	})

	t.Run("t=0.5 returns 0.5", func(t *testing.T) {
		result := easeInOutCubic(0.5)
		assert.InDelta(t, 0.5, result, 0.01)
	})

	t.Run("t<0.5 produces smooth acceleration", func(t *testing.T) {
		result := easeInOutCubic(0.25)
		assert.InDelta(t, 0.0625, result, 0.01)
	})

	t.Run("t>0.5 produces values in [0,1]", func(t *testing.T) {
		result := easeInOutCubic(0.75)
		assert.InDelta(t, 0.9375, result, 0.01)
		assert.LessOrEqual(t, result, 1.0)
	})

	t.Run("negative input clamped to 0", func(t *testing.T) {
		assert.Equal(t, 0.0, easeInOutCubic(-0.5))
	})

	t.Run("input > 1 clamped to 1", func(t *testing.T) {
		assert.Equal(t, 1.0, easeInOutCubic(1.5))
	})

	t.Run("result never exceeds 1.0 across full range", func(t *testing.T) {
		for v := 0.0; v <= 1.0; v += 0.01 {
			result := easeInOutCubic(v)
			assert.GreaterOrEqual(t, result, 0.0, "for v=%v", v)
			assert.LessOrEqual(t, result, 1.0, "for v=%v", v)
		}
	})
}

// TestEaseOutQuad verifies easeOutQuad behavior, one t.Run subtest per scenario.
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

// TestEaseInQuad verifies easeInQuad behavior, one t.Run subtest per scenario.
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

// TestNewAnimator verifies the documented behavior of NewAnimator.
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

// TestAnimator_ShowSpinner_Disabled verifies that Animator handles the show spinner case and...
func TestAnimator_ShowSpinner_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	// Should not panic and return immediately
	assert.NotPanics(t, func() {
		a.ShowSpinner("Test message", 100*time.Millisecond)
	})
}

// TestAnimator_ShowSpinner_VeryShortDuration verifies that Animator handles the show spinner case...
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

// TestAnimator_ShowProgress_Disabled verifies that Animator handles the show progress case and...
func TestAnimator_ShowProgress_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	steps := []string{"Step 1", "Step 2", "Step 3"}

	assert.NotPanics(t, func() {
		a.ShowProgress("Processing", steps)
	})
}

// TestAnimator_ShowProgress_EmptySteps verifies that Animator handles the show progress case and...
func TestAnimator_ShowProgress_EmptySteps(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowProgress("No steps", []string{})
	})
}

// TestAnimator_ShowProgress_SingleStep verifies that Animator handles the show progress case and...
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

// TestAnimator_DrawProgressBar verifies Animator behavior for the draw progress bar case, one...
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

// TestAnimator_AnimateTransition_Disabled verifies that Animator handles the animate transition...
func TestAnimator_AnimateTransition_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.AnimateTransition("From", "To", 100*time.Millisecond)
	})
}

// TestAnimator_AnimateTransition_ZeroDuration verifies that Animator handles the animate...
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

// TestAnimator_MixTexts verifies Animator behavior for the mix texts case, one t.Run subtest per...
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

// TestAnimator_ShowLoadingSkeleton_Disabled verifies that Animator handles the show loading...
func TestAnimator_ShowLoadingSkeleton_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	lines := []string{"Line 1", "Line 2", "Line 3"}

	assert.NotPanics(t, func() {
		a.ShowLoadingSkeleton(lines, 100*time.Millisecond)
	})
}

// TestAnimator_ShowLoadingSkeleton_EmptyLines verifies that Animator handles the show loading...
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

// TestAnimator_TypewriterEffect_Disabled verifies that Animator handles the typewriter effect case...
func TestAnimator_TypewriterEffect_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.TypewriterEffect("Hello, World!", 10*time.Millisecond)
	})
}

// TestAnimator_TypewriterEffect_EmptyString verifies that Animator handles the typewriter effect...
func TestAnimator_TypewriterEffect_EmptyString(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.TypewriterEffect("", 10*time.Millisecond)
	})
}

// TestAnimator_TypewriterEffect_Unicode verifies that Animator handles the typewriter effect case...
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

// TestAnimator_PulseText_Disabled verifies that Animator handles the pulse text case and behaves...
func TestAnimator_PulseText_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.PulseText("Important!", 1)
	})
}

// TestAnimator_PulseText_ZeroPulses verifies that Animator handles the pulse text case and handles...
func TestAnimator_PulseText_ZeroPulses(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.PulseText("No pulse", 0)
	})
}

// TestAnimator_PulseText_MultiplePulses verifies that Animator handles the pulse text case and...
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

// TestAnimator_ShowStepTransition_Disabled verifies that Animator handles the show step transition...
func TestAnimator_ShowStepTransition_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowStepTransition(1, 2, "Step One", "Step Two")
	})
}

// TestAnimator_ShowStepTransition_EdgeCases verifies that Animator handles the show step...
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

// TestAnimator_ShowSuccessAnimation_Disabled verifies that Animator handles the show success...
func TestAnimator_ShowSuccessAnimation_Disabled(t *testing.T) {
	a := NewAnimator()
	a.Disabled = true

	assert.NotPanics(t, func() {
		a.ShowSuccessAnimation("Operation completed!")
	})
}

// TestAnimator_ShowSuccessAnimation_EmptyMessage verifies that Animator handles the show success...
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

// TestAnimator_SetStyle verifies Animator behavior for the set style case, one t.Run subtest per...
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

// TestMax verifies max behavior, one t.Run subtest per scenario.
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

// TestSetAnimationStyle verifies the documented behavior of SetAnimationStyle.
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

// TestProgressBarCharacters verifies progress bar characters behavior, one t.Run subtest per scenario.
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

// TestSpinnerCharacters verifies the documented behavior of spinner characters.
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

// TestDotsSpinnerCharacters verifies the documented behavior of dots spinner characters.
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

// TestAnimationConstants verifies the documented behavior of animation constants.
func TestAnimationConstants(t *testing.T) {
	assert.Equal(t, 60*time.Millisecond, DefaultFrameRate)
	assert.Equal(t, 30*time.Millisecond, FastFrameRate)
	assert.Equal(t, 100*time.Millisecond, SlowFrameRate)
}

// ---------------------------------------------------------------------------
// AnimationState
// ---------------------------------------------------------------------------

// TestAnimationState verifies the documented behavior of AnimationState.
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

// TestAnimator_DrawPulsedText verifies Animator behavior for the draw pulsed text case, one t.Run...
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

// TestAnimator_DrawSkeleton verifies Animator behavior for the draw skeleton case, one t.Run...
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

// ---------------------------------------------------------------------------
// Enabled-path animation tests (stdout capture)
// ---------------------------------------------------------------------------

// TestAnimator_ShowSpinner_Enabled verifies that Animator handles the show spinner case and...
func TestAnimator_ShowSpinner_Enabled(t *testing.T) {
	a := NewAnimator()
	a.Speed = 10 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowSpinner("Loading...", 15*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Loading")
}

// TestAnimator_ShowProgress_Enabled verifies that Animator handles the show progress case and...
func TestAnimator_ShowProgress_Enabled(t *testing.T) {
	a := NewAnimator()
	a.Speed = 1 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowProgress("Processing", []string{"Step 1"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	assert.Contains(t, output, "Processing")
	assert.Contains(t, output, "Step 1")
}

// TestAnimator_ShowProgress_Enabled_MultipleSteps verifies that Animator handles the show progress...
func TestAnimator_ShowProgress_Enabled_MultipleSteps(t *testing.T) {
	a := NewAnimator()
	a.Speed = 1 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowProgress("Multi", []string{"A", "B", "C"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Multi")
}

// TestAnimator_AnimateTransition_Enabled verifies that Animator handles the animate transition...
func TestAnimator_AnimateTransition_Enabled(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.AnimateTransition("From", "To", 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "To")
}

// TestAnimator_AnimateTransition_Enabled_DifferentLengths verifies that Animator handles the...
func TestAnimator_AnimateTransition_Enabled_DifferentLengths(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.AnimateTransition("Short", "LongerText", 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "LongerText")
}

// TestAnimator_ShowLoadingSkeleton_Enabled verifies that Animator handles the show loading...
func TestAnimator_ShowLoadingSkeleton_Enabled(t *testing.T) {
	a := NewAnimator()
	a.Speed = 1 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowLoadingSkeleton([]string{"Loading..."}, 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Loading")
}

// TestAnimator_ShowLoadingSkeleton_Enabled_MultipleLines verifies that Animator handles the show...
func TestAnimator_ShowLoadingSkeleton_Enabled_MultipleLines(t *testing.T) {
	a := NewAnimator()
	a.Speed = 1 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowLoadingSkeleton([]string{"Line 1", "Line 2"}, 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Line")
}

// TestAnimator_TypewriterEffect_Enabled verifies that Animator handles the typewriter effect case...
func TestAnimator_TypewriterEffect_Enabled(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.TypewriterEffect("Hi", 1*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Hi")
}

// TestAnimator_TypewriterEffect_Enabled_LongText verifies that Animator handles the typewriter...
func TestAnimator_TypewriterEffect_Enabled_LongText(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.TypewriterEffect("Hello World", 1*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Hello")
}

// TestAnimator_PulseText_Enabled verifies that Animator handles the pulse text case and behaves...
func TestAnimator_PulseText_Enabled(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.PulseText("Test", 1)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Test")
}

// TestAnimator_ShowStepTransition_Enabled verifies that Animator handles the show step transition...
func TestAnimator_ShowStepTransition_Enabled(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowStepTransition(1, 2, "Step One", "Step Two")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Step")
}

// TestAnimator_ShowSuccessAnimation_Enabled verifies that Animator handles the show success...
func TestAnimator_ShowSuccessAnimation_Enabled(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowSuccessAnimation("Done!")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Done")
}

// TestAnimator_ShowSuccessAnimation_Enabled_SpecialChars verifies that Animator handles the show...
func TestAnimator_ShowSuccessAnimation_Enabled_SpecialChars(t *testing.T) {
	a := NewAnimator()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowSuccessAnimation("✅ Success!")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// ---------------------------------------------------------------------------
// Global convenience function tests
// ---------------------------------------------------------------------------

// TestShowSpinner_GlobalCapture verifies that ShowSpinner captures output through the...
func TestShowSpinner_GlobalCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	origSpeed := globalAnimator.Speed
	globalAnimator.Disabled = true
	defer func() {
		globalAnimator.Disabled = origDisabled
		globalAnimator.Speed = origSpeed
	}()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowSpinner("Global spinner", 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Global spinner")
}

// TestShowProgress_GlobalCapture verifies that ShowProgress captures output through the...
func TestShowProgress_GlobalCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowProgress("Global progress", []string{"Step"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Global progress")
}

// TestAnimateTransition_GlobalCapture verifies that AnimateTransition captures output through the...
func TestAnimateTransition_GlobalCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	AnimateTransition("A", "B", 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "B")
}

// ---------------------------------------------------------------------------
// mixTexts additional edge cases
// ---------------------------------------------------------------------------

// TestAnimator_MixTexts_EdgeCases verifies Animator behavior for the mix texts case and edge cases...
func TestAnimator_MixTexts_EdgeCases(t *testing.T) {
	a := NewAnimator()

	t.Run("to shorter than from", func(t *testing.T) {
		result := a.mixTexts("hello world", "hi", 0.5, 0.5)
		assert.NotEmpty(t, result)
	})

	t.Run("both single char", func(t *testing.T) {
		result := a.mixTexts("a", "b", 0.5, 0.5)
		assert.Equal(t, 1, len(result))
	})

	t.Run("unicode strings", func(t *testing.T) {
		result := a.mixTexts("Hello 世界", "Bye 🌍", 0.5, 0.5)
		assert.NotEmpty(t, result)
	})
}

// ---------------------------------------------------------------------------
// drawProgressBar additional cases
// ---------------------------------------------------------------------------

// TestAnimator_DrawProgressBar_AllStyles verifies Animator behavior for the draw progress bar case...
func TestAnimator_DrawProgressBar_AllStyles(t *testing.T) {
	a := NewAnimator()

	for name, style := range ProgressBarCharacters {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				a.drawProgressBar(0.5, style, "Test")
			})
		})
	}
}

// ---------------------------------------------------------------------------
// ShowSpinner and ShowProgress with empty messages
// ---------------------------------------------------------------------------

// TestAnimator_ShowSpinner_Enabled_EmptyMessage verifies that Animator handles the show spinner...
func TestAnimator_ShowSpinner_Enabled_EmptyMessage(t *testing.T) {
	a := NewAnimator()
	a.Speed = 10 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowSpinner("", 10*time.Millisecond)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	// Should still produce output (the checkmark)
	assert.True(t, buf.Len() > 0)
}

// TestAnimator_ShowProgress_Enabled_EmptySteps verifies that Animator handles the show progress...
func TestAnimator_ShowProgress_Enabled_EmptySteps(t *testing.T) {
	a := NewAnimator()
	a.Speed = 1 * time.Millisecond

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	a.ShowProgress("Test", []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Test")
}

// ---------------------------------------------------------------------------
// StringsContains helper check
// ---------------------------------------------------------------------------

// TestStringsContains verifies the documented behavior of strings contains.
func TestStringsContains(t *testing.T) {
	assert.True(t, strings.Contains("hello world", "world"))
	assert.False(t, strings.Contains("hello", "world"))
}
