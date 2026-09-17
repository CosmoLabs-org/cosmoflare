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
// TransitionManager creation
// ---------------------------------------------------------------------------

// TestNewTransitionManager verifies the documented behavior of NewTransitionManager.
func TestNewTransitionManager(t *testing.T) {
	tm := NewTransitionManager()
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.animator)
	assert.False(t, tm.Disabled)
	assert.Equal(t, "cosmic", tm.CurrentTheme)
}

// ---------------------------------------------------------------------------
// SetTheme
// ---------------------------------------------------------------------------

// TestTransitionManager_SetTheme verifies TransitionManager behavior for the set theme case, one...
func TestTransitionManager_SetTheme(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("set forest theme", func(t *testing.T) {
		tm.SetTheme("forest")
		assert.Equal(t, "forest", tm.CurrentTheme)
	})

	t.Run("set ocean theme", func(t *testing.T) {
		tm.SetTheme("ocean")
		assert.Equal(t, "ocean", tm.CurrentTheme)
	})

	t.Run("set custom theme", func(t *testing.T) {
		tm.SetTheme("custom")
		assert.Equal(t, "custom", tm.CurrentTheme)
	})

	t.Run("set empty theme", func(t *testing.T) {
		tm.SetTheme("")
		assert.Equal(t, "", tm.CurrentTheme)
	})
}

// ---------------------------------------------------------------------------
// Disable
// ---------------------------------------------------------------------------

// TestTransitionManager_Disable verifies that TransitionManager handles the disable case.
func TestTransitionManager_Disable(t *testing.T) {
	tm := NewTransitionManager()
	assert.False(t, tm.Disabled)
	assert.False(t, tm.animator.Disabled)

	tm.Disable()

	assert.True(t, tm.Disabled)
	assert.True(t, tm.animator.Disabled)
}

// ---------------------------------------------------------------------------
// Execute (disabled mode)
// ---------------------------------------------------------------------------

// TestTransitionManager_Execute_Disabled verifies TransitionManager behavior for the execute case...
func TestTransitionManager_Execute_Disabled(t *testing.T) {
	tm := NewTransitionManager()
	tm.Disable()

	from := TransitionState{
		Title:   "From",
		Content: []string{"Content 1"},
	}

	to := TransitionState{
		Title:   "To",
		Content: []string{"Content 2"},
	}

	transitionTypes := []TransitionType{
		TransitionFade,
		TransitionSlide,
		TransitionWipe,
		TransitionZoom,
		TransitionReplace,
	}

	for _, tt := range transitionTypes {
		t.Run(string(tt), func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.Execute(tt, from, to)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Execute with unknown transition type
// ---------------------------------------------------------------------------

// TestTransitionManager_Execute_UnknownType verifies that TransitionManager handles the execute...
func TestTransitionManager_Execute_UnknownType(t *testing.T) {
	tm := NewTransitionManager()
	tm.Disable()

	from := TransitionState{Title: "From"}
	to := TransitionState{Title: "To"}

	assert.NotPanics(t, func() {
		tm.Execute(TransitionType("unknown"), from, to)
	})
}

// ---------------------------------------------------------------------------
// renderInstant
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderInstant verifies that TransitionManager handles the render instant case.
func TestTransitionManager_RenderInstant(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:       "Test Title",
		Content:     []string{"Line 1", "Line 2"},
		Progress:    5,
		MaxProgress: 10,
	}

	assert.NotPanics(t, func() {
		tm.renderInstant(state)
	})
}

// TestTransitionManager_RenderInstant_NoProgress verifies that TransitionManager handles the...
func TestTransitionManager_RenderInstant_NoProgress(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:   "No Progress",
		Content: []string{"Content"},
	}

	assert.NotPanics(t, func() {
		tm.renderInstant(state)
	})
}

// TestTransitionManager_RenderInstant_EmptyContent verifies that TransitionManager handles the...
func TestTransitionManager_RenderInstant_EmptyContent(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:   "Empty",
		Content: []string{},
	}

	assert.NotPanics(t, func() {
		tm.renderInstant(state)
	})
}

// ---------------------------------------------------------------------------
// renderWithAlpha
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderWithAlpha verifies TransitionManager behavior for the render with...
func TestTransitionManager_RenderWithAlpha(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:   "Alpha Test",
		Content: []string{"Content"},
	}

	testAlphas := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, alpha := range testAlphas {
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.renderWithAlpha(state, alpha)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// renderSlide
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderSlide verifies TransitionManager behavior for the render slide case,...
func TestTransitionManager_RenderSlide(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "To",
		Content: []string{"To Content"},
	}

	testProgress := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, progress := range testProgress {
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.renderSlide(from, to, progress)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// renderWipe
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderWipe verifies TransitionManager behavior for the render wipe case,...
func TestTransitionManager_RenderWipe(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "To",
		Content: []string{"To Content"},
	}

	testProgress := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, progress := range testProgress {
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.renderWipe(from, to, progress)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// renderWithScale
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderWithScale verifies TransitionManager behavior for the render with...
func TestTransitionManager_RenderWithScale(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:   "Scale Test",
		Content: []string{"Content Line 1", "Content Line 2"},
	}

	testScales := []float64{0.0, 0.25, 0.5, 0.75, 1.0, 1.5}

	for _, scale := range testScales {
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.renderWithScale(state, scale)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// renderWithOffset
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderWithOffset verifies TransitionManager behavior for the render with...
func TestTransitionManager_RenderWithOffset(t *testing.T) {
	tm := NewTransitionManager()

	state := TransitionState{
		Title:   "Offset Test",
		Content: []string{"Line 1", "Line 2", "Line 3"},
	}

	testOffsets := []int{-10, -5, 0, 5, 10}

	for _, offset := range testOffsets {
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				tm.renderWithOffset(state, offset)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// renderHeader
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderHeader verifies TransitionManager behavior for the render header...
func TestTransitionManager_RenderHeader(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("normal title", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderHeader("Test Title")
		})
	})

	t.Run("empty title", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderHeader("")
		})
	})

	t.Run("very long title", func(t *testing.T) {
		longTitle := strings.Repeat("A", 200)
		assert.NotPanics(t, func() {
			tm.renderHeader(longTitle)
		})
	})

	t.Run("unicode title", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderHeader("标题 Титул タイトル")
		})
	})
}

// ---------------------------------------------------------------------------
// renderContent
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderContent verifies TransitionManager behavior for the render content...
func TestTransitionManager_RenderContent(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("single line", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderContent([]string{"Single line"})
		})
	})

	t.Run("multiple lines", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderContent([]string{"Line 1", "Line 2", "Line 3"})
		})
	})

	t.Run("empty content", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderContent([]string{})
		})
	})

	t.Run("lines with special characters", func(t *testing.T) {
		content := []string{"Tab\there", "New\nLine", "Quote\"test"}
		assert.NotPanics(t, func() {
			tm.renderContent(content)
		})
	})
}

// ---------------------------------------------------------------------------
// renderProgress
// ---------------------------------------------------------------------------

// TestTransitionManager_RenderProgress verifies TransitionManager behavior for the render progress...
func TestTransitionManager_RenderProgress(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("zero progress", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderProgress(0, 100)
		})
	})

	t.Run("half progress", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderProgress(50, 100)
		})
	})

	t.Run("full progress", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderProgress(100, 100)
		})
	})

	t.Run("small max", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderProgress(2, 5)
		})
	})

	t.Run("current equals max", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tm.renderProgress(1, 1)
		})
	})
}

// ---------------------------------------------------------------------------
// scaleText
// ---------------------------------------------------------------------------

// TestTransitionManager_ScaleText verifies TransitionManager behavior for the scale text case, one...
func TestTransitionManager_ScaleText(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("scale >= 1.0 returns original", func(t *testing.T) {
		result := tm.scaleText("Hello World", 1.0)
		assert.Equal(t, "Hello World", result)

		result = tm.scaleText("Test", 1.5)
		assert.Equal(t, "Test", result)
	})

	t.Run("scale < 1.0 truncates text", func(t *testing.T) {
		result := tm.scaleText("Hello World", 0.5)
		// 11 * 0.5 = 5.5 -> 5 chars + "..." = 8
		assert.Equal(t, 8, len(result))
	})

	t.Run("very small scale", func(t *testing.T) {
		result := tm.scaleText("Hello", 0.1)
		// 5 * 0.1 = 0.5 -> 0 chars, but min is 3 for ellipsis
		assert.LessOrEqual(t, len(result), 4)
	})

	t.Run("empty string", func(t *testing.T) {
		result := tm.scaleText("", 0.5)
		assert.Equal(t, "", result)
	})

	t.Run("short text", func(t *testing.T) {
		result := tm.scaleText("Hi", 0.5)
		// 2 * 0.5 = 1 -> too short for ellipsis
		assert.LessOrEqual(t, len(result), 3)
	})
}

// ---------------------------------------------------------------------------
// SetupStepTransition
// ---------------------------------------------------------------------------

// TestNewSetupStepTransition verifies the documented behavior of NewSetupStepTransition.
func TestNewSetupStepTransition(t *testing.T) {
	sst := NewSetupStepTransition(5)
	assert.NotNil(t, sst.Manager)
	assert.NotNil(t, sst.Manager.animator)
	assert.Equal(t, 0, sst.Current)
	assert.Equal(t, 5, sst.Total)
}

// TestSetupStepTransition_NextStep verifies SetupStepTransition behavior for the next step case,...
func TestSetupStepTransition_NextStep(t *testing.T) {
	sst := NewSetupStepTransition(3)
	sst.Manager.Disable() // Disable for testing

	t.Run("first step", func(t *testing.T) {
		assert.NotPanics(t, func() {
			sst.NextStep("Step 1", "Content for step 1")
		})
		assert.Equal(t, 1, sst.Current)
	})

	t.Run("second step", func(t *testing.T) {
		assert.NotPanics(t, func() {
			sst.NextStep("Step 2", "Content for step 2")
		})
		assert.Equal(t, 2, sst.Current)
	})

	t.Run("final step", func(t *testing.T) {
		assert.NotPanics(t, func() {
			sst.NextStep("Step 3", "Content for step 3")
		})
		assert.Equal(t, 3, sst.Current)
	})
}

// TestSetupStepTransition_CompleteStep verifies that SetupStepTransition handles the complete step...
func TestSetupStepTransition_CompleteStep(t *testing.T) {
	sst := NewSetupStepTransition(3)
	sst.Manager.Disable() // Disable for testing

	assert.NotPanics(t, func() {
		sst.CompleteStep("Complete", "All done!")
	})
}

// ---------------------------------------------------------------------------
// fadeTransition
// ---------------------------------------------------------------------------

// TestTransitionManager_FadeTransition verifies that TransitionManager handles the fade transition...
func TestTransitionManager_FadeTransition(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "Fade From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "Fade To",
		Content: []string{"To Content"},
	}

	assert.NotPanics(t, func() {
		tm.fadeTransition(from, to)
	})
}

// ---------------------------------------------------------------------------
// slideTransition
// ---------------------------------------------------------------------------

// TestTransitionManager_SlideTransition verifies that TransitionManager handles the slide...
func TestTransitionManager_SlideTransition(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "Slide From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "Slide To",
		Content: []string{"To Content"},
	}

	assert.NotPanics(t, func() {
		tm.slideTransition(from, to)
	})
}

// ---------------------------------------------------------------------------
// wipeTransition
// ---------------------------------------------------------------------------

// TestTransitionManager_WipeTransition verifies that TransitionManager handles the wipe transition...
func TestTransitionManager_WipeTransition(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "Wipe From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "Wipe To",
		Content: []string{"To Content"},
	}

	assert.NotPanics(t, func() {
		tm.wipeTransition(from, to)
	})
}

// ---------------------------------------------------------------------------
// zoomTransition
// ---------------------------------------------------------------------------

// TestTransitionManager_ZoomTransition verifies that TransitionManager handles the zoom transition...
func TestTransitionManager_ZoomTransition(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "Zoom From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "Zoom To",
		Content: []string{"To Content"},
	}

	assert.NotPanics(t, func() {
		tm.zoomTransition(from, to)
	})
}

// ---------------------------------------------------------------------------
// replaceTransition
// ---------------------------------------------------------------------------

// TestTransitionManager_ReplaceTransition verifies that TransitionManager handles the replace...
func TestTransitionManager_ReplaceTransition(t *testing.T) {
	tm := NewTransitionManager()

	from := TransitionState{
		Title:   "Replace From",
		Content: []string{"From Content"},
	}

	to := TransitionState{
		Title:   "Replace To",
		Content: []string{"To Content"},
	}

	assert.NotPanics(t, func() {
		tm.replaceTransition(from, to)
	})
}

// ---------------------------------------------------------------------------
// SetupWizardTransition
// ---------------------------------------------------------------------------

// TestSetupWizardTransition_Next verifies SetupWizardTransition behavior for the next case, one...
func TestSetupWizardTransition_Next(t *testing.T) {
	globalTransitionManager.Disable()

	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"Content 1"})
	swt.AddStep("Step 2", []string{"Content 2"})
	swt.AddStep("Step 3", []string{"Content 3"})

	t.Run("initial current is -1", func(t *testing.T) {
		assert.Equal(t, -1, swt.current)
	})

	t.Run("first Next moves to step 0", func(t *testing.T) {
		swt.Next()
		assert.Equal(t, 0, swt.current)
	})

	t.Run("second Next moves to step 1", func(t *testing.T) {
		swt.Next()
		assert.Equal(t, 1, swt.current)
	})

	t.Run("third Next moves to step 2", func(t *testing.T) {
		swt.Next()
		assert.Equal(t, 2, swt.current)
	})

	t.Run("Next beyond last step increments past end", func(t *testing.T) {
		swt.Next()
		assert.Equal(t, 3, swt.current)
	})
}

// TestSetupWizardTransition_Show verifies SetupWizardTransition behavior for the show case, one...
func TestSetupWizardTransition_Show(t *testing.T) {
	globalTransitionManager.Disable()

	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"Content 1"})
	swt.AddStep("Step 2", []string{"Content 2"})

	t.Run("show valid step", func(t *testing.T) {
		assert.NotPanics(t, func() {
			swt.Show(0)
		})
		assert.Equal(t, 0, swt.current)
	})

	t.Run("show another valid step", func(t *testing.T) {
		assert.NotPanics(t, func() {
			swt.Show(1)
		})
		assert.Equal(t, 1, swt.current)
	})

	t.Run("show invalid index does nothing", func(t *testing.T) {
		originalCurrent := swt.current
		assert.NotPanics(t, func() {
			swt.Show(10)
		})
		assert.Equal(t, originalCurrent, swt.current)
	})

	t.Run("show negative index does nothing", func(t *testing.T) {
		originalCurrent := swt.current
		assert.NotPanics(t, func() {
			swt.Show(-1)
		})
		assert.Equal(t, originalCurrent, swt.current)
	})
}

// ---------------------------------------------------------------------------
// TransitionState
// ---------------------------------------------------------------------------

// TestTransitionState_Creation verifies that TransitionState handles the creation case.
func TestTransitionState_Creation(t *testing.T) {
	state := TransitionState{
		Title:       "Test State",
		Content:     []string{"Line 1", "Line 2"},
		Progress:    5,
		MaxProgress: 10,
		Theme:       "cosmic",
	}

	assert.Equal(t, "Test State", state.Title)
	assert.Len(t, state.Content, 2)
	assert.Equal(t, 5, state.Progress)
	assert.Equal(t, 10, state.MaxProgress)
	assert.Equal(t, "cosmic", state.Theme)
}

// TestTransitionState_ZeroValue verifies that TransitionState handles the zero value case.
func TestTransitionState_ZeroValue(t *testing.T) {
	var state TransitionState

	assert.Empty(t, state.Title)
	assert.Nil(t, state.Content)
	assert.Equal(t, 0, state.Progress)
	assert.Equal(t, 0, state.MaxProgress)
	assert.Empty(t, state.Theme)
}

// ---------------------------------------------------------------------------
// Transition
// ---------------------------------------------------------------------------

// TestTransition_Creation verifies that Transition handles the creation case.
func TestTransition_Creation(t *testing.T) {
	from := TransitionState{Title: "From"}
	to := TransitionState{Title: "To"}

	trans := Transition{
		Type:       TransitionFade,
		Duration:   500 * time.Millisecond,
		EasingFunc: easeInOutCubic,
		From:       from,
		To:         to,
	}

	assert.Equal(t, TransitionFade, trans.Type)
	assert.Equal(t, 500*time.Millisecond, trans.Duration)
	assert.NotNil(t, trans.EasingFunc)
	assert.Equal(t, "From", trans.From.Title)
	assert.Equal(t, "To", trans.To.Title)
}

// ---------------------------------------------------------------------------
// Global transition manager
// ---------------------------------------------------------------------------

// TestGlobalTransitionManager verifies the documented behavior of global transition manager.
func TestGlobalTransitionManager(t *testing.T) {
	assert.NotNil(t, globalTransitionManager)
	assert.NotNil(t, globalTransitionManager.animator)
}

// ---------------------------------------------------------------------------
// Edge cases and error handling
// ---------------------------------------------------------------------------

// TestTransitionManager_EdgeCases verifies TransitionManager behavior for the edge cases case, one...
func TestTransitionManager_EdgeCases(t *testing.T) {
	tm := NewTransitionManager()

	t.Run("nil content in renderInstant", func(t *testing.T) {
		state := TransitionState{
			Title:   "Test",
			Content: nil,
		}
		assert.NotPanics(t, func() {
			tm.renderInstant(state)
		})
	})

	t.Run("very long content", func(t *testing.T) {
		longContent := make([]string, 1000)
		for i := range longContent {
			longContent[i] = strings.Repeat("Line ", 50)
		}
		state := TransitionState{
			Title:   "Long",
			Content: longContent,
		}
		assert.NotPanics(t, func() {
			tm.renderInstant(state)
		})
	})

	t.Run("unicode in all fields", func(t *testing.T) {
		state := TransitionState{
			Title: "标题",
			Content: []string{
				"Содержание 1",
				"コンテンツ 2",
				"🎉🎊🎈",
			},
			Theme: "テーマ",
		}
		assert.NotPanics(t, func() {
			tm.renderInstant(state)
		})
	})
}

// ---------------------------------------------------------------------------
// Execute enabled-path tests with stdout capture
// ---------------------------------------------------------------------------

// TestTransitionManager_Execute_AllTypes_DisabledCapture verifies TransitionManager behavior for...
func TestTransitionManager_Execute_AllTypes_DisabledCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	tm := NewTransitionManager()
	from := TransitionState{Title: "From", Content: []string{"Line 1"}}
	to := TransitionState{Title: "To", Content: []string{"Line 2"}}

	for _, tt := range []struct {
		name string
		typ  TransitionType
	}{
		{"fade", TransitionFade},
		{"slide", TransitionSlide},
		{"wipe", TransitionWipe},
		{"zoom", TransitionZoom},
		{"replace", TransitionReplace},
		{"unknown_default", TransitionType("unknown")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tm.Execute(tt.typ, from, to)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			assert.True(t, buf.Len() > 0)
		})
	}
}

// TestTransitionManager_Execute_FullFadeAnimation verifies that TransitionManager handles the...
func TestTransitionManager_Execute_FullFadeAnimation(t *testing.T) {
	tm := NewTransitionManager()
	from := TransitionState{Title: "From"}
	to := TransitionState{Title: "To"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		tm.Execute(TransitionFade, from, to)
	}()
	<-done

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// TestTransitionManager_Execute_FullSlideAnimation verifies that TransitionManager handles the...
func TestTransitionManager_Execute_FullSlideAnimation(t *testing.T) {
	tm := NewTransitionManager()
	from := TransitionState{Title: "SlideFrom", Content: []string{"C1"}}
	to := TransitionState{Title: "SlideTo", Content: []string{"C2"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		tm.Execute(TransitionSlide, from, to)
	}()
	<-done

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// TestTransitionManager_Execute_FullWipeAnimation verifies that TransitionManager handles the...
func TestTransitionManager_Execute_FullWipeAnimation(t *testing.T) {
	tm := NewTransitionManager()
	from := TransitionState{Title: "WipeFrom"}
	to := TransitionState{Title: "WipeTo"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		tm.Execute(TransitionWipe, from, to)
	}()
	<-done

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// TestTransitionManager_Execute_FullZoomAnimation verifies that TransitionManager handles the...
func TestTransitionManager_Execute_FullZoomAnimation(t *testing.T) {
	tm := NewTransitionManager()
	from := TransitionState{Title: "ZoomFrom", Content: []string{"X"}}
	to := TransitionState{Title: "ZoomTo", Content: []string{"Y"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		tm.Execute(TransitionZoom, from, to)
	}()
	<-done

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// TestTransitionManager_Execute_FullReplaceAnimation verifies that TransitionManager handles the...
func TestTransitionManager_Execute_FullReplaceAnimation(t *testing.T) {
	tm := NewTransitionManager()
	from := TransitionState{Title: "ReplaceFrom"}
	to := TransitionState{Title: "ReplaceTo"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		tm.Execute(TransitionReplace, from, to)
	}()
	<-done

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// ---------------------------------------------------------------------------
// SetupStepTransition NextStep and CompleteStep
// ---------------------------------------------------------------------------

// TestSetupStepTransition_NextStepCapture verifies that SetupStepTransition handles the next step...
func TestSetupStepTransition_NextStepCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	sst := NewSetupStepTransition(3)
	assert.Equal(t, 0, sst.Current)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	sst.NextStep("Step 1", "Content 1")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 1, sst.Current)
}

// TestSetupStepTransition_CompleteStepCapture verifies that SetupStepTransition handles the...
func TestSetupStepTransition_CompleteStepCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	sst := NewSetupStepTransition(2)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	sst.CompleteStep("Done", "All finished")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// ---------------------------------------------------------------------------
// SetupWizardTransition
// ---------------------------------------------------------------------------

// TestSetupWizardTransition_AddAndShowCapture verifies that SetupWizardTransition handles the add...
func TestSetupWizardTransition_AddAndShowCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"Content 1"})
	swt.AddStep("Step 2", []string{"Content 2"})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	swt.Show(0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Step 1")
}

// TestSetupWizardTransition_NextCapture verifies that SetupWizardTransition handles the next...
func TestSetupWizardTransition_NextCapture(t *testing.T) {
	origDisabled := globalAnimator.Disabled
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = origDisabled }()

	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"C1"})
	swt.AddStep("Step 2", []string{"C2"})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	swt.Show(0)
	swt.Next()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.True(t, buf.Len() > 0)
}

// TestSetupWizardTransition_Show_InvalidIndexCapture verifies that SetupWizardTransition handles...
func TestSetupWizardTransition_Show_InvalidIndexCapture(t *testing.T) {
	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"C1"})

	assert.NotPanics(t, func() {
		swt.Show(-1)
		swt.Show(99)
	})
}

// TestSetupWizardTransition_Next_NoStepsCapture verifies that SetupWizardTransition handles the...
func TestSetupWizardTransition_Next_NoStepsCapture(t *testing.T) {
	swt := NewSetupWizardTransition()
	assert.NotPanics(t, func() {
		swt.Next()
	})
}
