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

// TestNewTutorialManager verifies the documented behavior of NewTutorialManager.
func TestNewTutorialManager(t *testing.T) {
	tm := NewTutorialManager()
	require.NotNil(t, tm)
	require.NotNil(t, tm.state)
	require.NotNil(t, tm.config)
	require.NotNil(t, tm.tutorials)
}

// TestNewTutorialManager_DefaultState verifies that NewTutorialManager handles the default state case.
func TestNewTutorialManager_DefaultState(t *testing.T) {
	tm := NewTutorialManager()
	assert.Equal(t, 0, tm.state.CurrentLesson)
	assert.Equal(t, 0, tm.state.TotalLessons)
	assert.Empty(t, tm.state.Completed)
	assert.Empty(t, tm.state.Skipped)
	assert.Equal(t, 0.0, tm.state.Progress)
	assert.False(t, tm.state.StartTime.IsZero())
}

// TestNewTutorialManager_DefaultConfig verifies that NewTutorialManager handles the default config...
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

// TestCreateDefaultTutorials verifies the documented behavior of createDefaultTutorials.
func TestCreateDefaultTutorials(t *testing.T) {
	tutorials := createDefaultTutorials()
	require.Len(t, tutorials, 5)
}

// TestCreateDefaultTutorials_FirstTutorial verifies that createDefaultTutorials handles the first...
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

// TestCreateDefaultTutorials_LastTutorial verifies that createDefaultTutorials handles the last...
func TestCreateDefaultTutorials_LastTutorial(t *testing.T) {
	tutorials := createDefaultTutorials()
	last := tutorials[len(tutorials)-1]
	assert.Equal(t, 5, last.ID)
	assert.Equal(t, "Advanced Tips & Tricks", last.Title)
	assert.True(t, last.Interactive)
	assert.False(t, last.Required)
}

// TestCreateDefaultTutorials_AllHaveContent verifies that createDefaultTutorials handles the all...
func TestCreateDefaultTutorials_AllHaveContent(t *testing.T) {
	tutorials := createDefaultTutorials()
	for i, tut := range tutorials {
		assert.NotEmpty(t, tut.Content, "tutorial %d", i+1)
		assert.NotEmpty(t, tut.Description, "tutorial %d", i+1)
		assert.NotEmpty(t, tut.Actions, "tutorial %d", i+1)
		assert.True(t, tut.ID > 0, "tutorial %d", i+1)
	}
}

// TestCreateDefaultTutorials_RequiredTutorials verifies that createDefaultTutorials handles the...
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

// TestCreateDefaultTutorials_ActionsHaveSkipAllowed verifies that createDefaultTutorials handles...
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

// TestCreateDefaultTutorials_ActionIDs verifies that createDefaultTutorials handles the action ids...
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

// TestShouldSkipTutorial_NotCompleted verifies that shouldSkipTutorial handles the not completed case.
func TestShouldSkipTutorial_NotCompleted(t *testing.T) {
	tm := NewTutorialManager()
	assert.False(t, tm.shouldSkipTutorial(1))
	assert.False(t, tm.shouldSkipTutorial(5))
}

// TestShouldSkipTutorial_AlreadyCompleted verifies that shouldSkipTutorial handles the already...
func TestShouldSkipTutorial_AlreadyCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.Completed = []int{1, 2, 3}
	assert.True(t, tm.shouldSkipTutorial(1))
	assert.False(t, tm.shouldSkipTutorial(4))
}

// TestShouldSkipTutorial_NonexistentID verifies that shouldSkipTutorial handles the nonexistent id...
func TestShouldSkipTutorial_NonexistentID(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.Completed = []int{99}
	assert.False(t, tm.shouldSkipTutorial(1))
	assert.True(t, tm.shouldSkipTutorial(99))
}

// --- updateProgress ---

// TestUpdateProgress verifies the documented behavior of updateProgress.
func TestUpdateProgress(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{1, 2, 3}
	tm.updateProgress()
	assert.InDelta(t, 0.6, tm.state.Progress, 0.001)
}

// TestUpdateProgress_NoneCompleted verifies that updateProgress handles the none completed case.
func TestUpdateProgress_NoneCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{}
	tm.updateProgress()
	assert.Equal(t, 0.0, tm.state.Progress)
}

// TestUpdateProgress_AllCompleted verifies that updateProgress handles the all completed case.
func TestUpdateProgress_AllCompleted(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 5
	tm.state.Completed = []int{1, 2, 3, 4, 5}
	tm.updateProgress()
	assert.InDelta(t, 1.0, tm.state.Progress, 0.001)
}

// TestUpdateProgress_ZeroTotalLessons verifies that updateProgress handles the zero total lessons...
func TestUpdateProgress_ZeroTotalLessons(t *testing.T) {
	tm := NewTutorialManager()
	tm.state.TotalLessons = 0
	tm.state.Completed = []int{}
	assert.NotPanics(t, func() { tm.updateProgress() })
}

// --- executeAction ---

// TestExecuteAction_AllActionIDs verifies executeAction behavior for the all action ids case, one...
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

// TestShowTutorialCompletion verifies showTutorialCompletion behavior, one t.Run subtest per scenario.
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

// TestShowQuickStart_Tutorials verifies that ShowQuickStart handles the tutorials case.
func TestShowQuickStart_Tutorials(t *testing.T) {
	tm := NewTutorialManager()
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { tm.ShowQuickStart() })
}

// --- StartTutorial with stdin ---

// TestStartTutorial_AllLessons verifies that StartTutorial handles the all lessons case.
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

// TestStartTutorial_SkipOptional verifies that StartTutorial handles the skip optional case.
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

// TestStartTutorial_SelectAction2 verifies that StartTutorial handles the select action2 case.
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

// TestShowTutorialIntro_Accept verifies that showTutorialIntro handles the accept case.
func TestShowTutorialIntro_Accept(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("y")
	assert.NotPanics(t, func() { tm.showTutorialIntro() })
}

// TestShowTutorialIntro_AllowSkipFalse verifies that showTutorialIntro handles the allow skip...
func TestShowTutorialIntro_AllowSkipFalse(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	tm := NewTutorialManager()
	tm.Input = newMockReader("y")
	tm.config.AllowSkip = false
	assert.NotPanics(t, func() { tm.showTutorialIntro() })
}

// --- showTutorialLesson with stdin ---

// TestShowTutorialLesson_InteractiveSelectAction1 verifies that showTutorialLesson handles the...
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

// TestShowTutorialLesson_InteractiveDefaultAction verifies that showTutorialLesson handles the...
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

// TestShowTutorialLesson_InteractiveSkip verifies that showTutorialLesson handles the interactive...
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

// TestShowTutorialLesson_NonInteractive verifies that showTutorialLesson handles the non...
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

// TestShowTutorialLesson_VerboseMode verifies that showTutorialLesson handles the verbose mode case.
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

// TestHandleTutorialActions_DefaultAction verifies that handleTutorialActions handles the default...
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

// TestHandleTutorialActions_SelectAction2 verifies that handleTutorialActions handles the select...
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

// TestHandleTutorialActions_Skip verifies that handleTutorialActions handles the skip case.
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

// TestHandleTutorialActions_InvalidThenValid verifies that handleTutorialActions re-prompts on...
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

// TestHandleTutorialActions_NoActions verifies that handleTutorialActions handles the no actions case.
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

// TestTutorialState_Fields verifies that TutorialState handles the fields case.
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

// TestTutorialAction_Handler verifies that TutorialAction handles the handler case.
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

// TestTutorialAction_HandlerError verifies that TutorialAction handles the handler error case.
func TestTutorialAction_HandlerError(t *testing.T) {
	action := TutorialAction{
		ID: "failing", Label: "Failing",
		Handler: func() error { return fmt.Errorf("boom") },
	}
	assert.Equal(t, "boom", action.Handler().Error())
}

// TestErrTutorialSkipped verifies the documented behavior of err tutorial skipped.
func TestErrTutorialSkipped(t *testing.T) {
	assert.Equal(t, "tutorial skipped", ErrTutorialSkipped.Error())
}

// --- State mutation ---

// TestTutorialManager_CompleteAll verifies that TutorialManager handles the complete all case.
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

// TestTutorialContent_NoANSIEscapeSequences verifies that tutorial content handles the no...
func TestTutorialContent_NoANSIEscapeSequences(t *testing.T) {
	tutorials := createDefaultTutorials()
	for _, tut := range tutorials {
		for _, line := range tut.Content {
			assert.NotContains(t, line, "\x1b", "tutorial %q", tut.Title)
		}
	}
}

// TestTutorialContent_ActionLabelsNotEmpty verifies that tutorial content handles the action...
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

// TestShowAccessibilityMenu_Options verifies ShowAccessibilityMenu behavior for the options case,...
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

// TestApplySettings_AllModes verifies applySettings behavior for the all modes case, one t.Run...
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

// TestShowCurrentSettings verifies the documented behavior of showCurrentSettings.
func TestShowCurrentSettings(t *testing.T) {
	am := NewAccessibilityManager()
	am.SetMode(AccessibilityFull)
	assert.NotPanics(t, func() { am.showCurrentSettings() })
}

// TestPrintAccessible_Modes verifies PrintAccessible behavior for the modes case, one t.Run...
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

// TestAnnounceToScreenReader verifies the documented behavior of announceToScreenReader.
func TestAnnounceToScreenReader(t *testing.T) {
	am := NewAccessibilityManager()
	assert.NotPanics(t, func() { am.announceToScreenReader("hello") })
}

// TestShowAccessibleMenu_StandardDefault verifies that ShowAccessibleMenu handles the standard...
func TestShowAccessibleMenu_StandardDefault(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B", "C"}, 0)
	assert.Equal(t, 0, got)
}

// TestShowAccessibleMenu_StandardSelect verifies that ShowAccessibleMenu handles the standard...
func TestShowAccessibleMenu_StandardSelect(t *testing.T) {
	_, cleanup := pipeStdin(t, "2\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B", "C"}, 0)
	assert.Equal(t, 1, got)
}

// TestShowAccessibleMenu_ScreenReaderMode verifies that ShowAccessibleMenu handles the screen...
func TestShowAccessibleMenu_ScreenReaderMode(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B"}, 0)
	assert.Equal(t, 0, got)
}

// TestShowAccessibleMenu_LargeTextMode verifies that ShowAccessibleMenu handles the large text...
func TestShowAccessibleMenu_LargeTextMode(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityLargeText)
	got := ah.ShowAccessibleMenu("Menu", []string{"A", "B"}, 1)
	assert.Equal(t, 1, got)
}

// TestGetAccessibleInput_NonSensitive verifies that GetAccessibleInput handles the non sensitive case.
func TestGetAccessibleInput_NonSensitive(t *testing.T) {
	_, cleanup := pipeStdin(t, "hello\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "hello", ah.GetAccessibleInput("Name:", false))
}

// TestGetAccessibleInput_Sensitive verifies that GetAccessibleInput handles the sensitive case.
func TestGetAccessibleInput_Sensitive(t *testing.T) {
	_, cleanup := pipeStdin(t, "secret\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "secret", ah.GetAccessibleInput("Password:", true))
}

// TestGetAccessibleInput_ScreenReader verifies that GetAccessibleInput routes output through the...
func TestGetAccessibleInput_ScreenReader(t *testing.T) {
	_, cleanup := pipeStdin(t, "val\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	assert.Equal(t, "val", ah.GetAccessibleInput("Enter:", false))
}

// TestGetAccessibleInput_Empty verifies that GetAccessibleInput handles the empty case.
func TestGetAccessibleInput_Empty(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	ah := NewAccessibilityHelper()
	assert.Equal(t, "", ah.GetAccessibleInput("Enter:", false))
}

// TestConfirmAccessibleYesNo verifies ConfirmAccessibleYesNo behavior, one t.Run subtest per scenario.
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

// TestConfirmYesNo verifies ConfirmYesNo behavior, one t.Run subtest per scenario.
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

// TestPauseAndWait verifies the documented behavior of PauseAndWait.
func TestPauseAndWait(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	assert.NotPanics(t, func() { PauseAndWait("Press Enter...") })
}

// TestPauseAndWait_DefaultMessage verifies that PauseAndWait handles the default message case.
func TestPauseAndWait_DefaultMessage(t *testing.T) {
	_, cleanup := pipeStdin(t, "\n")
	defer cleanup()
	assert.NotPanics(t, func() { PauseAndWait("") })
}

// TestCheckFirstRun exercises CheckFirstRun and asserts it completes without panicking.
func TestCheckFirstRun(t *testing.T) {
	isFirst, err := CheckFirstRun()
	if err != nil {
		t.Skipf("CheckFirstRun failed: %v", err)
	}
	_ = isFirst
}

// TestShowWelcomeForNewUser_Accept verifies that ShowWelcomeForNewUser handles the accept case.
func TestShowWelcomeForNewUser_Accept(t *testing.T) {
	_, cleanup := pipeStdin(t, "y\n")
	defer cleanup()
	assert.NotPanics(t, func() { ShowWelcomeForNewUser() })
}

// TestShowWelcomeForNewUser_Decline verifies that ShowWelcomeForNewUser handles the decline case.
func TestShowWelcomeForNewUser_Decline(t *testing.T) {
	_, cleanup := pipeStdin(t, "n\n")
	defer cleanup()
	assert.NotPanics(t, func() { ShowWelcomeForNewUser() })
}

// ===========================================================================
// Animation global functions
// ===========================================================================

// TestShowSpinner_Global verifies that ShowSpinner exercises the package-global instance safely.
func TestShowSpinner_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { ShowSpinner("Loading...", 1*time.Millisecond) })
}

// TestShowProgress_Global verifies that ShowProgress exercises the package-global instance safely.
func TestShowProgress_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { ShowProgress("Steps", []string{"s1", "s2"}) })
}

// TestAnimateTransition_Global verifies that AnimateTransition exercises the package-global...
func TestAnimateTransition_Global(t *testing.T) {
	globalAnimator.Disabled = true
	defer func() { globalAnimator.Disabled = false }()
	assert.NotPanics(t, func() { AnimateTransition("from", "to", 1*time.Millisecond) })
}

// TestSetAnimationStyle_Global verifies that SetAnimationStyle exercises the package-global...
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

// TestPromptWithDefault_Stdin verifies PromptWithDefault behavior for the stdin case, one t.Run...
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

