package interactive

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// linesFromInput converts a pipe-style input string to individual lines for mockReader.
func linesFromInput(s string) []string {
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// pipeStdin creates a pipe with input and returns cleanup that restores stdin.
func pipeStdin(t *testing.T, input string) (*os.File, func()) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	go func() {
		defer w.Close()
		io.WriteString(w, input)
	}()
	os.Stdin = r
	return r, func() {
		r.Close()
		os.Stdin = oldStdin
	}
}

// --- NewTutorialManager ---

func TestNewTutorialManager(t *testing.T) {
	tm := NewTutorialManager()
	require.NotNil(t, tm)
	require.NotNil(t, tm.state)
	require.NotNil(t, tm.config)
	require.NotNil(t, tm.tutorials)
}

func TestNewTutorialManager_DefaultState(t *testing.T) {
	tm := NewTutorialManager()
	assert.Equal(t, 0, tm.state.CurrentLesson)
	assert.Equal(t, 0, tm.state.TotalLessons)
	assert.Empty(t, tm.state.Completed)
	assert.Empty(t, tm.state.Skipped)
	assert.Equal(t, 0.0, tm.state.Progress)
	assert.False(t, tm.state.StartTime.IsZero())
}

func TestNewTutorialManager_DefaultConfig(t *testing.T) {
	tm := NewTutorialManager()
	assert.True(t, tm.config.AutoStart)
	assert.True(t, tm.config.AllowSkip)
	assert.True(t, tm.config.SaveProgress)
	assert.Equal(t, 5*time.Minute, tm.config.Timeout)
	assert.True(t, tm.config.ShowHints)
	assert.False(t, tm.config.VerboseMode)
	assert.True(t, tm.config.Interactive)
}

// --- createDefaultTutorials ---

func TestCreateDefaultTutorials(t *testing.T) {
	tutorials := createDefaultTutorials()
	require.Len(t, tutorials, 5)
}

func TestCreateDefaultTutorials_FirstTutorial(t *testing.T) {
	tutorials := createDefaultTutorials()
	first := tutorials[0]
	assert.Equal(t, 1, first.ID)
	assert.Equal(t, "Understanding Profiles", first.Title)
	assert.NotEmpty(t, first.Content)
	assert.Len(t, first.Actions, 2)
	assert.True(t, first.Interactive)
	assert.True(t, first.Required)
}

func TestCreateDefaultTutorials_LastTutorial(t *testing.T) {
	tutorials := createDefaultTutorials()
	last := tutorials[len(tutorials)-1]
	assert.Equal(t, 5, last.ID)
	assert.Equal(t, "Advanced Tips & Tricks", last.Title)
	assert.True(t, last.Interactive)
	assert.False(t, last.Required)
}

func TestCreateDefaultTutorials_AllHaveContent(t *testing.T) {
	tutorials := createDefaultTutorials()
	for i, tut := range tutorials {
		assert.NotEmpty(t, tut.Content, "tutorial %d", i+1)
		assert.NotEmpty(t, tut.Description, "tutorial %d", i+1)
		assert.NotEmpty(t, tut.Actions, "tutorial %d", i+1)
		assert.True(t, tut.ID > 0, "tutorial %d", i+1)
	}
}

func TestCreateDefaultTutorials_RequiredTutorials(t *testing.T) {
	tutorials := createDefaultTutorials()
	count := 0
	for _, tut := range tutorials {
		if tut.Required {
			count++
		}
	}
	assert.Equal(t, 3, count)
}

func TestCreateDefaultTutorials_ActionsHaveSkipAllowed(t *testing.T) {
	tutorials := createDefaultTutorials()
	for _, tut := range tutorials {
		for _, action := range tut.Actions {
			assert.True(t, action.SkipAllowed)
			assert.NotEmpty(t, action.Label)
			assert.NotEmpty(t, action.ID)
		}
	}
}

func TestCreateDefaultTutorials_ActionIDs(t *testing.T) {
	tutorials := createDefaultTutorials()
	expectedIDs := map[int][]string{
		1: {"view_profiles", "create_profile"},
		2: {"list_buckets", "create_bucket"},
		3: {"upload_file", "view_upload_help"},
		4: {"list_objects", "object_info"},
		5: {"try_switching", "view_help"},
	}
	for _, tut := range tutorials {
		ids, ok := expectedIDs[tut.ID]
		require.True(t, ok, "unexpected tutorial ID %d", tut.ID)
		require.Len(t, tut.Actions, len(ids))
		for i, action := range tut.Actions {
			assert.Equal(t, ids[i], action.ID)
		}
	}
}

// --- shouldSkipTutorial ---

func TestShouldSkipTutorial_NotCompleted(t *testing.T) {
	tm := NewTutorialManager()
	assert.False(t, tm.shouldSkipTutorial(1))
	assert.False(t, tm.shouldSkipTutorial(5))
}

func TestShouldSkipTutorial_AlreadyCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.Completed = []int{1, 2, 3}
	assert.True(t, tm.shouldSkipTutorial(1))
	assert.False(t, tm.shouldSkipTutorial(4))
}

func TestShouldSkipTutorial_NonexistentID(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.Completed = []int{99}
	assert.False(t, tm.shouldSkipTutorial(1))
	assert.True(t, tm.shouldSkipTutorial(99))
}

// --- updateProgress ---

func TestUpdateProgress(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{1, 2, 3}
	tm.updateProgress()
	assert.InDelta(t, 0.6, tm.state.Progress, 0.001)
}

func TestUpdateProgress_NoneCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{}
	tm.updateProgress()
	assert.Equal(t, 0.0, tm.state.Progress)
}

func TestUpdateProgress_AllCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{1, 2, 3, 4, 5}
	tm.updateProgress()
	assert.InDelta(t, 1.0, tm.state.Progress, 0.001)
}

func TestUpdateProgress_ZeroTotalLessons(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 0
	tm.state.Completed = []int{}
	assert.NotPanics(t, func() { tm.updateProgress() })
}

// --- executeAction ---

func TestExecuteAction_AllActionIDs(t *testing.T) {
	ids := []string{
		"view_profiles", "create_profile", "list_buckets", "create_bucket",
		"upload_file", "list_objects", "try_switching", "view_help",
		"object_info", "view_upload_help", "unknown_action_id",
	}
	tm := NewTutorialManager()
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			err := tm.executeAction(TutorialAction{ID: id, Label: "Test " + id})
			assert.Nil(t, err)
		})
	}
}

// --- showTutorialCompletion ---

func TestShowTutorialCompletion(t *testing.T) {
	tests := []struct {
		name      string
		completed []int
		skipped   []int
	}{
		{"all completed", []int{1, 2, 3, 4, 5}, []int{}},
		{"partial", []int{1, 2}, []int{3, 4}},
		{"none skipped", []int{1, 2, 3}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTutorialManager()
			tm.state.TotalLessons = 5
			tm.state.Completed = tt.completed
			tm.state.Skipped = tt.skipped
			tm.state.StartTime = time.Now().Add(-30 * time.Second)
			tm.state.Progress = float64(len(tt.completed)) / float64(tm.state.TotalLessons)
			globalAnimator.Disabled = true
			defer func() { globalAnimator.Disabled = false }()
			assert.NotPanics(t, func() { tm.showTutorialCompletion() })
		})
	}
}

// --- ShowQuickStart ---

func TestShowQuickStart_Tutorials(t *testing.T) {
	tm := NewTutorialManager()
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { tm.ShowQuickStart() })
}

// --- StartTutorial with stdin ---

func TestStartTutorial_AllLessons(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader(linesFromInput("y\n1\n\n1\n\n1\n\n1\n\n1\n\n")...)
	err := tm.StartTutorial()
	assert.NoError(t, err)
	assert.Len(t, tm.state.Completed, 5)
	assert.Empty(t, tm.state.Skipped)
	assert.InDelta(t, 1.0, tm.state.Progress, 0.001)
}

func TestStartTutorial_SkipOptional(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader(linesFromInput("y\n1\n\n1\n\n1\n\ns\ns\n")...)
	err := tm.StartTutorial()
	assert.NoError(t, err)
	assert.Len(t, tm.state.Completed, 3)
	assert.Len(t, tm.state.Skipped, 2)
}

func TestStartTutorial_SelectAction2(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader(linesFromInput("y\n2\n\n2\n\n2\n\n2\n\n2\n\n")...)
	err := tm.StartTutorial()
	assert.NoError(t, err)
	assert.Len(t, tm.state.Completed, 5)
}

// --- showTutorialIntro with stdin ---

func TestShowTutorialIntro_Accept(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("y")
	assert.NotPanics(t, func() { tm.showTutorialIntro() })
}

func TestShowTutorialIntro_AllowSkipFalse(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("y")
	tm.config.AllowSkip = false
	assert.NotPanics(t, func() { tm.showTutorialIntro() })
}

// --- showTutorialLesson with stdin ---

func TestShowTutorialLesson_InteractiveSelectAction1(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("1", "")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{
			{ID: "view_profiles", Label: "Action 1", SkipAllowed: true},
			{ID: "create_profile", Label: "Action 2", SkipAllowed: true},
		},
	}
	err := tm.showTutorialLesson(tutorial)
	assert.NoError(t, err)
}

func TestShowTutorialLesson_InteractiveDefaultAction(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("", "")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.showTutorialLesson(tutorial)
	assert.NoError(t, err)
}

func TestShowTutorialLesson_InteractiveSkip(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("s")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.showTutorialLesson(tutorial)
	assert.Equal(t, ErrTutorialSkipped, err)
}

func TestShowTutorialLesson_NonInteractive(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 99, Title: "Non-Interactive", Description: "Test",
		Content: []string{"Line 1"}, Interactive: false, Actions: []TutorialAction{},
	}
	err := tm.showTutorialLesson(tutorial)
	assert.NoError(t, err)
}

func TestShowTutorialLesson_VerboseMode(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("", "")
	tm.config.VerboseMode = true
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 99, Title: "Verbose", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.showTutorialLesson(tutorial)
	assert.NoError(t, err)
}

// --- handleTutorialActions with stdin ---

func TestHandleTutorialActions_DefaultAction(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("", "")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.handleTutorialActions(tutorial)
	assert.NoError(t, err)
}

func TestHandleTutorialActions_SelectAction2(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("2", "")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{
			{ID: "a1", Label: "Action 1", SkipAllowed: true},
			{ID: "a2", Label: "Action 2", SkipAllowed: true},
		},
	}
	err := tm.handleTutorialActions(tutorial)
	assert.NoError(t, err)
}

func TestHandleTutorialActions_Skip(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("s")
	tm.config.AllowSkip = true
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.handleTutorialActions(tutorial)
	assert.Equal(t, ErrTutorialSkipped, err)
}

func TestHandleTutorialActions_InvalidThenValid(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("abc", "1", "")
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true,
		Actions: []TutorialAction{{ID: "a1", Label: "Action 1", SkipAllowed: true}},
	}
	err := tm.handleTutorialActions(tutorial)
	assert.NoError(t, err)
}

func TestHandleTutorialActions_NoActions(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.state.CurrentLesson = 1
	tm.state.TotalLessons = 1
	tutorial := Tutorial{
		ID: 1, Title: "Test", Description: "Test",
		Content: []string{"content"}, Interactive: true, Actions: []TutorialAction{},
	}
	err := tm.handleTutorialActions(tutorial)
	assert.NoError(t, err)
}

// --- Struct tests ---

func TestTutorialState_Fields(t *testing.T) {
	now := time.Now()
	state := TutorialState{
		CurrentLesson: 3, TotalLessons: 5,
		Completed: []int{1, 2}, Skipped: []int{4},
		Progress: 0.4, StartTime: now,
	}
	assert.Equal(t, 3, state.CurrentLesson)
	assert.Equal(t, 5, state.TotalLessons)
	assert.Equal(t, []int{1, 2}, state.Completed)
	assert.InDelta(t, 0.4, state.Progress, 0.001)
	assert.Equal(t, now, state.StartTime)
}

func TestTutorialAction_Handler(t *testing.T) {
	called := false
	action := TutorialAction{
		ID: "custom", Label: "Custom",
		Handler: func() error { called = true; return nil },
		SkipAllowed: false,
	}
	assert.NoError(t, action.Handler())
	assert.True(t, called)
}

func TestTutorialAction_HandlerError(t *testing.T) {
	action := TutorialAction{
		ID: "failing", Label: "Failing",
		Handler: func() error { return fmt.Errorf("boom") },
	}
	assert.Equal(t, "boom", action.Handler().Error())
}

func TestErrTutorialSkipped(t *testing.T) {
	assert.Equal(t, "tutorial skipped", ErrTutorialSkipped.Error())
}

// --- State mutation ---

func TestTutorialManager_CompleteAll(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	for i := 1; i <= 5; i++ {
		tm.state.Completed = append(tm.state.Completed, i)
	}
	tm.updateProgress()
	assert.InDelta(t, 1.0, tm.state.Progress, 0.001)
	for i := 1; i <= 5; i++ {
		assert.True(t, tm.shouldSkipTutorial(i))
	}
}

// --- Content security ---

func TestTutorialContent_NoANSIEscapeSequences(t *testing.T) {
	tutorials := createDefaultTutorials()
	for _, tut := range tutorials {
		for _, line := range tut.Content {
			assert.NotContains(t, line, "\x1b", "tutorial %q", tut.Title)
		}
	}
}

func TestTutorialContent_ActionLabelsNotEmpty(t *testing.T) {
	tutorials := createDefaultTutorials()
	for _, tut := range tutorials {
		for _, action := range tut.Actions {
			assert.NotEmpty(t, strings.TrimSpace(action.Label))
		}
	}
}

// ===========================================================================
// Accessibility tests (stdin-mocked)
// ===========================================================================

func TestShowAccessibilityMenu_Options(t *testing.T) {
	tests := []struct {
		input   string
		wantMode AccessibilityMode
	}{
		{"1\n", AccessibilityScreenReader},
		{"2\n", AccessibilityHighContrast},
		{"3\n", AccessibilityLargeText},
		{"4\n", AccessibilityReducedMotion},
		{"5\n", AccessibilityFull},
		{"6\n", AccessibilityNone},
		{"\n", AccessibilityScreenReader},
		{"99\n", AccessibilityScreenReader},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, cleanup := pipeStdin(t, tt.input)
			defer cleanup()
			am := NewAccessibilityManager()
			err := am.ShowAccessibilityMenu()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMode, am.config.Mode)
		})
	}
}

func TestApplySettings_AllModes(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*AccessibilityManager)
	}{
		{"reduced motion", func(am *AccessibilityManager) { am.config.ReducedMotion = true }},
		{"screen reader", func(am *AccessibilityManager) { am.config.ScreenReader = true }},
		{"high contrast", func(am *AccessibilityManager) { am.config.HighContrast = true }},
		{"large text", func(am *AccessibilityManager) { am.config.LargeText = true }},
		{"no flags", func(am *AccessibilityManager) { am.config.Mode = AccessibilityNone }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAccessibilityManager()
			tt.setup(am)
			assert.NotPanics(t, func() { am.applySettings() })
		})
	}
}

func TestShowCurrentSettings(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityFull)
	assert.NotPanics(t, func() { am.showCurrentSettings() })
}

func TestPrintAccessible_Modes(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		screenReader bool
	}{
		{"non-verbose", false, false},
		{"verbose", true, false},
		{"screen reader", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAccessibilityManager()
			am.config.Verbose = tt.verbose
			am.config.ScreenReader = tt.screenReader
			assert.NotPanics(t, func() { am.PrintAccessible("test") })
			assert.NotPanics(t, func() { am.PrintAccessibleSuccess("ok") })
			assert.NotPanics(t, func() { am.PrintAccessibleError("fail") })
		})
	}
}

func TestAnnounceToScreenReader(t *testing.T) {
	am := NewAccessibilityManager()
	assert.NotPanics(t, func() { am.announceToScreenReader("hello") })
}

func TestShowAccessibleMenu_StandardDefault(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B", "C"}, 0)
	assert.Equal(t, 0, got)
}

func TestShowAccessibleMenu_StandardSelect(t *testing.T) {
	_, cleanup := pipeStdin(t, "2\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B", "C"}, 0)
	assert.Equal(t, 1, got)
}

func TestShowAccessibleMenu_ScreenReaderMode(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B"}, 0)
	assert.Equal(t, 0, got)
}

func TestShowAccessibleMenu_LargeTextMode(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityLargeText)
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B"}, 1)
	assert.Equal(t, 1, got)
}

func TestGetAccessibleInput_NonSensitive(t *testing.T) {
	_, cleanup := pipeStdin(t, "hello\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "hello", ah.GetAccessibleInput("Name:", false))
}

func TestGetAccessibleInput_Sensitive(t *testing.T) {
	_, cleanup := pipeStdin(t, "secret\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "secret", ah.GetAccessibleInput("Password:", true))
}

func TestGetAccessibleInput_ScreenReader(t *testing.T) {
	_, cleanup := pipeStdin(t, "val\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	assert.Equal(t, "val", ah.GetAccessibleInput("Enter:", false))
}

func TestGetAccessibleInput_Empty(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "", ah.GetAccessibleInput("Enter:", false))
}

func TestConfirmAccessibleYesNo(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		verbose    bool
		want       bool
	}{
		{"default yes", "\n", true, false, true},
		{"default no", "\n", false, false, false},
		{"explicit yes", "y\n", false, false, true},
		{"explicit no", "n\n", true, false, false},
		{"verbose yes", "\n", true, true, true},
		{"verbose no", "\n", false, true, false},
		{"screen reader", "y\n", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := pipeStdin(t, tt.input)
			defer cleanup()
			ah := NewAccessibilityHelper()
			ah.manager.config.Verbose = tt.verbose
			if tt.name == "screen reader" {
				ah.manager.SetMode(AccessibilityScreenReader)
			}
			assert.Equal(t, tt.want, ah.ConfirmAccessibleYesNo("Continue?", tt.defaultYes))
		})
	}
}

// ===========================================================================
// Helpers stdin-mocked tests
// ===========================================================================

func TestConfirmYesNo(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{"default yes", "\n", true, true},
		{"default no", "\n", false, false},
		{"y", "y\n", false, true},
		{"n", "n\n", true, false},
		{"yes", "yes\n", false, true},
		{"no", "no\n", true, false},
		{"random", "maybe\n", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := pipeStdin(t, tt.input)
			defer cleanup()
			assert.Equal(t, tt.want, ConfirmYesNo("Q?", tt.defaultYes))
		})
	}
}

func TestPauseAndWait(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	assert.NotPanics(t, func() { PauseAndWait("Press Enter...") })
}

func TestPauseAndWait_DefaultMessage(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	assert.NotPanics(t, func() { PauseAndWait("") })
}

func TestCheckFirstRun(t *testing.T) {
	isFirst, err := CheckFirstRun()
	if err != nil {
		t.Skipf("CheckFirstRun failed: %v", err)
	}
	_ = isFirst
}

func TestShowWelcomeForNewUser_Accept(t *testing.T) {
	_, cleanup := pipeStdin(t, "y\n")
	defer cleanup()
	assert.NotPanics(t, func() { ShowWelcomeForNewUser() })
}

func TestShowWelcomeForNewUser_Decline(t *testing.T) {
	_, cleanup := pipeStdin(t, "n\n")
	defer cleanup()
	assert.NotPanics(t, func() { ShowWelcomeForNewUser() })
}

// ===========================================================================
// Animation global functions
// ===========================================================================

func TestShowSpinner_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { ShowSpinner("Loading...", 1*time.Millisecond) })
}

func TestShowProgress_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { ShowProgress("Steps", []string{"s1", "s2"}) })
}

func TestAnimateTransition_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { AnimateTransition("from", "to", 1*time.Millisecond) })
}

func TestSetAnimationStyle_Global(t *testing.T) {
	globalAnimator.Speed = DefaultFrameRate
	globalAnimator.Disabled = false
	SetAnimationStyle("fast")
	assert.Equal(t, FastFrameRate, globalAnimator.Speed)
	SetAnimationStyle("slow")
	assert.Equal(t, SlowFrameRate, globalAnimator.Speed)
	SetAnimationStyle("disabled")
	assert.True(t, globalAnimator.Disabled)
	globalAnimator.Disabled = false
	SetAnimationStyle("default")
	assert.Equal(t, DefaultFrameRate, globalAnimator.Speed)
}

// ===========================================================================
// First run stdin-mocked
// ===========================================================================


// ===========================================================================
// Setup stdin-mocked
// ===========================================================================

func TestPromptWithDefault_Stdin(t *testing.T) {
	tests := []struct {
		name  string
		input string
		def   string
		want  string
	}{
		{"accept default", "\n", "default", "default"},
		{"provide value", "custom\n", "default", "custom"},
		{"no default", "provided\n", "", "provided"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := pipeStdin(t, tt.input)
			defer cleanup()
			got, err := PromptWithDefault("Enter:", tt.def); assert.NoError(t, err); assert.Equal(t, tt.want, got)
		})
	}
}

