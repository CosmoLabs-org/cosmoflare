package interactive

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NewAccessibilityHelper
// ---------------------------------------------------------------------------

func TestNewAccessibilityHelper(t *testing.T) {
	helper := NewAccessibilityHelper()
	if helper == nil {
		t.Fatal("NewAccessibilityHelper should return non-nil")
	}
	if helper.manager == nil {
		t.Fatal("NewAccessibilityHelper.manager should be non-nil")
	}
	if helper.manager.config.Mode != AccessibilityNone {
		t.Errorf("default mode should be AccessibilityNone, got %d", helper.manager.config.Mode)
	}
}

// ---------------------------------------------------------------------------
// AccessibilityManager.getModeName (unexported, tested from same package)
// ---------------------------------------------------------------------------

func TestGetModeName(t *testing.T) {
	tests := []struct {
		name string
		mode AccessibilityMode
		want string
	}{
		{"AccessibilityNone returns Disabled", AccessibilityNone, "Disabled"},
		{"ScreenReader returns Screen Reader", AccessibilityScreenReader, "Screen Reader"},
		{"HighContrast returns High Contrast", AccessibilityHighContrast, "High Contrast"},
		{"LargeText returns Large Text", AccessibilityLargeText, "Large Text"},
		{"ReducedMotion returns Reduced Motion", AccessibilityReducedMotion, "Reduced Motion"},
		{"Full returns Full Accessibility", AccessibilityFull, "Full Accessibility"},
		{"Unknown mode (99) returns Disabled", AccessibilityMode(99), "Disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAccessibilityManager()
			am.SetMode(tt.mode)
			got := am.getModeName()
			if got != tt.want {
				t.Errorf("getModeName() with mode %d = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formatBool (unexported, tested from same package)
// ---------------------------------------------------------------------------

func TestFormatBool(t *testing.T) {
	t.Run("true contains Enabled", func(t *testing.T) {
		got := formatBool(true)
		if !strings.Contains(got, "Enabled") {
			t.Errorf("formatBool(true) = %q, want to contain 'Enabled'", got)
		}
	})

	t.Run("false contains Disabled", func(t *testing.T) {
		got := formatBool(false)
		if !strings.Contains(got, "Disabled") {
			t.Errorf("formatBool(false) = %q, want to contain 'Disabled'", got)
		}
	})

	t.Run("true and false produce different strings", func(t *testing.T) {
		tVal := formatBool(true)
		fVal := formatBool(false)
		if tVal == fVal {
			t.Errorf("formatBool(true) and formatBool(false) should differ, both = %q", tVal)
		}
	})
}

// ---------------------------------------------------------------------------
// ShowAccessibleProgress (verbose path)
// ---------------------------------------------------------------------------

func TestShowAccessibleProgressVerbose(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah := NewAccessibilityHelper()
	ah.manager.config.Verbose = true
	ah.manager.config.ScreenReader = false // only test verbose, not screen reader
	ah.ShowAccessibleProgress("Uploading", 0, 10)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	if !strings.Contains(got, "Uploading") {
		t.Errorf("verbose progress output should contain message, got %q", got)
	}
	if !strings.Contains(got, "0/10") {
		t.Errorf("verbose progress output should contain current/total, got %q", got)
	}
	if !strings.Contains(got, "0.0%") {
		t.Errorf("verbose progress output should contain percentage, got %q", got)
	}
	if !strings.Contains(got, "Progress:") {
		t.Errorf("verbose progress output should contain 'Progress:', got %q", got)
	}
}

func TestShowAccessibleProgressCompletion(t *testing.T) {
	// When current == total, a newline should be printed at the end.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah := NewAccessibilityHelper()
	ah.manager.config.Verbose = false
	ah.manager.config.ScreenReader = false
	ah.ShowAccessibleProgress("Done", 10, 10)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	// The non-verbose path uses \r, but when current==total it prints a newline
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("progress at completion should end with newline, got %q", got)
	}
	if !strings.Contains(got, "10/10") {
		t.Errorf("progress should show 10/10, got %q", got)
	}
}

func TestShowAccessibleProgressScreenReader(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah := NewAccessibilityHelper()
	ah.manager.config.ScreenReader = true
	ah.manager.config.Verbose = false
	ah.ShowAccessibleProgress("Downloading", 5, 20)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	got := string(out)

	if !strings.Contains(got, "Downloading") {
		t.Errorf("screen reader progress should announce message, got %q", got)
	}
	if !strings.Contains(got, "5 of 20 complete") {
		t.Errorf("screen reader progress should announce '5 of 20 complete', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// EnableAccessibilityMode / IsAccessibilityEnabled (global state)
// ---------------------------------------------------------------------------

func TestEnableAccessibilityModeAndIsEnabled(t *testing.T) {
	// Save and restore global state
	origMode := globalAccessibilityManager.config.Mode
	defer func() {
		globalAccessibilityManager.config.Mode = origMode
	}()

	t.Run("ScreenReader mode enables accessibility", func(t *testing.T) {
		EnableAccessibilityMode(AccessibilityScreenReader)
		if !IsAccessibilityEnabled() {
			t.Error("IsAccessibilityEnabled() should return true after EnableAccessibilityMode(ScreenReader)")
		}
	})

	t.Run("None mode disables accessibility", func(t *testing.T) {
		EnableAccessibilityMode(AccessibilityNone)
		if IsAccessibilityEnabled() {
			t.Error("IsAccessibilityEnabled() should return false after EnableAccessibilityMode(None)")
		}
	})

	t.Run("HighContrast mode enables accessibility", func(t *testing.T) {
		EnableAccessibilityMode(AccessibilityHighContrast)
		if !IsAccessibilityEnabled() {
			t.Error("IsAccessibilityEnabled() should return true after EnableAccessibilityMode(HighContrast)")
		}
	})

	t.Run("Full mode enables accessibility", func(t *testing.T) {
		EnableAccessibilityMode(AccessibilityFull)
		if !IsAccessibilityEnabled() {
			t.Error("IsAccessibilityEnabled() should return true after EnableAccessibilityMode(Full)")
		}
	})

	// Restore
	EnableAccessibilityMode(AccessibilityNone)
}

// ---------------------------------------------------------------------------
// GetAccessibilityManager
// ---------------------------------------------------------------------------

func TestGetAccessibilityManager(t *testing.T) {
	am := GetAccessibilityManager()
	if am == nil {
		t.Fatal("GetAccessibilityManager() should return non-nil")
	}
	// Should be the same instance every time
	am2 := GetAccessibilityManager()
	if am != am2 {
		t.Error("GetAccessibilityManager() should return the same instance")
	}
}

// ---------------------------------------------------------------------------
// TransitionType constants
// ---------------------------------------------------------------------------

func TestTransitionTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		typ   TransitionType
		value string
	}{
		{"Fade", TransitionFade, "fade"},
		{"Slide", TransitionSlide, "slide"},
		{"Wipe", TransitionWipe, "wipe"},
		{"Zoom", TransitionZoom, "zoom"},
		{"Replace", TransitionReplace, "replace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.value {
				t.Errorf("TransitionType %s = %q, want %q", tt.name, tt.typ, tt.value)
			}
		})
	}
}

func TestTransitionTypeStringValues(t *testing.T) {
	// All transition types should have distinct values
	types := []TransitionType{TransitionFade, TransitionSlide, TransitionWipe, TransitionZoom, TransitionReplace}
	seen := make(map[string]bool)
	for _, tt := range types {
		s := string(tt)
		if seen[s] {
			t.Errorf("duplicate TransitionType value: %q", s)
		}
		seen[s] = true
	}
	if len(seen) != 5 {
		t.Errorf("expected 5 distinct transition types, got %d", len(seen))
	}
}

// ---------------------------------------------------------------------------
// NewSetupWizardTransition
// ---------------------------------------------------------------------------

func TestNewSetupWizardTransition(t *testing.T) {
	swt := NewSetupWizardTransition()
	if swt == nil {
		t.Fatal("NewSetupWizardTransition() should return non-nil")
	}
	if swt.steps == nil {
		t.Error("steps should be initialized (non-nil slice)")
	}
	if len(swt.steps) != 0 {
		t.Errorf("steps should be empty, got %d items", len(swt.steps))
	}
	if swt.current != -1 {
		t.Errorf("current should be -1 (no step selected), got %d", swt.current)
	}
}

func TestSetupWizardTransitionAddStep(t *testing.T) {
	swt := NewSetupWizardTransition()
	swt.AddStep("Step 1", []string{"line1", "line2"})

	if len(swt.steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(swt.steps))
	}
	if swt.steps[0].Title != "Step 1" {
		t.Errorf("step title = %q, want %q", swt.steps[0].Title, "Step 1")
	}
	if len(swt.steps[0].Content) != 2 {
		t.Errorf("step content length = %d, want 2", len(swt.steps[0].Content))
	}
	if swt.steps[0].Content[0] != "line1" {
		t.Errorf("step content[0] = %q, want %q", swt.steps[0].Content[0], "line1")
	}
}

// ---------------------------------------------------------------------------
// ErrorType constants
// ---------------------------------------------------------------------------

func TestErrorTypeIotaValues(t *testing.T) {
	tests := []struct {
		name  string
		typ   ErrorType
		value int
	}{
		{"Network", ErrorTypeNetwork, 0},
		{"Auth", ErrorTypeAuth, 1},
		{"Config", ErrorTypeConfig, 2},
		{"Input", ErrorTypeInput, 3},
		{"Permission", ErrorTypePermission, 4},
		{"NotFound", ErrorTypeNotFound, 5},
		{"Validation", ErrorTypeValidation, 6},
		{"Unknown", ErrorTypeUnknown, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.typ) != tt.value {
				t.Errorf("ErrorType %s = %d, want %d", tt.name, tt.typ, tt.value)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ErrorContext struct
// ---------------------------------------------------------------------------

func TestErrorContextFields(t *testing.T) {
	t.Run("struct creation with all fields", func(t *testing.T) {
		err := fmt.Errorf("test error")
		ctx := ErrorContext{
			Error:        err,
			Type:         ErrorTypeNetwork,
			Operation:    "upload file",
			UserAction:   "check connection",
			Troubleshoot: []string{"step 1", "step 2"},
			NextSteps:    []string{"next 1"},
		}

		if ctx.Error == nil || ctx.Error.Error() != "test error" {
			t.Errorf("Error field not set correctly")
		}
		if ctx.Type != ErrorTypeNetwork {
			t.Errorf("Type = %d, want ErrorTypeNetwork (0)", ctx.Type)
		}
		if ctx.Operation != "upload file" {
			t.Errorf("Operation = %q, want %q", ctx.Operation, "upload file")
		}
		if ctx.UserAction != "check connection" {
			t.Errorf("UserAction = %q, want %q", ctx.UserAction, "check connection")
		}
		if len(ctx.Troubleshoot) != 2 {
			t.Errorf("Troubleshoot length = %d, want 2", len(ctx.Troubleshoot))
		}
		if len(ctx.NextSteps) != 1 {
			t.Errorf("NextSteps length = %d, want 1", len(ctx.NextSteps))
		}
	})

	t.Run("zero value defaults", func(t *testing.T) {
		ctx := ErrorContext{}

		if ctx.Error != nil {
			t.Error("zero-value ErrorContext should have nil Error")
		}
		if ctx.Type != ErrorTypeNetwork {
			// ErrorTypeNetwork is iota = 0, which is the zero value
			t.Errorf("zero-value Type should be ErrorTypeNetwork (0), got %d", ctx.Type)
		}
		if ctx.Operation != "" {
			t.Errorf("zero-value Operation should be empty, got %q", ctx.Operation)
		}
		if len(ctx.Troubleshoot) != 0 {
			t.Errorf("zero-value Troubleshoot should be empty")
		}
		if len(ctx.NextSteps) != 0 {
			t.Errorf("zero-value NextSteps should be empty")
		}
	})
}

// ---------------------------------------------------------------------------
// Error factory functions (pure logic: return ErrorContext structs)
// ---------------------------------------------------------------------------

func TestNetworkErrorFactory(t *testing.T) {
	err := fmt.Errorf("connection refused")
	ctx := NetworkError("upload", err)

	if ctx.Type != ErrorTypeNetwork {
		t.Errorf("NetworkError type = %d, want ErrorTypeNetwork", ctx.Type)
	}
	if ctx.Operation != "upload" {
		t.Errorf("NetworkError operation = %q, want %q", ctx.Operation, "upload")
	}
	if ctx.UserAction == "" {
		t.Error("NetworkError should set UserAction")
	}
	if len(ctx.Troubleshoot) == 0 {
		t.Error("NetworkError should provide Troubleshoot steps")
	}
	if len(ctx.NextSteps) == 0 {
		t.Error("NetworkError should provide NextSteps")
	}
}

func TestAuthErrorFactory(t *testing.T) {
	err := fmt.Errorf("invalid token")
	ctx := AuthError("authenticate", err)

	if ctx.Type != ErrorTypeAuth {
		t.Errorf("AuthError type = %d, want ErrorTypeAuth", ctx.Type)
	}
	if ctx.UserAction == "" {
		t.Error("AuthError should set UserAction")
	}
	if len(ctx.Troubleshoot) == 0 {
		t.Error("AuthError should provide Troubleshoot steps")
	}
}

func TestConfigErrorFactory(t *testing.T) {
	ctx := ConfigError("load config", fmt.Errorf("missing file"))

	if ctx.Type != ErrorTypeConfig {
		t.Errorf("ConfigError type = %d, want ErrorTypeConfig", ctx.Type)
	}
	if ctx.Operation != "load config" {
		t.Errorf("ConfigError operation = %q, want %q", ctx.Operation, "load config")
	}
}

func TestInputErrorFactory(t *testing.T) {
	ctx := InputError("parse bucket name", fmt.Errorf("empty string"))

	if ctx.Type != ErrorTypeInput {
		t.Errorf("InputError type = %d, want ErrorTypeInput", ctx.Type)
	}
}

func TestPermissionErrorFactory(t *testing.T) {
	ctx := PermissionError("delete bucket", fmt.Errorf("forbidden"))

	if ctx.Type != ErrorTypePermission {
		t.Errorf("PermissionError type = %d, want ErrorTypePermission", ctx.Type)
	}
	if len(ctx.Troubleshoot) == 0 {
		t.Error("PermissionError should provide Troubleshoot steps")
	}
}

func TestNotFoundErrorFactory(t *testing.T) {
	ctx := NotFoundError("get bucket", fmt.Errorf("not found"))

	if ctx.Type != ErrorTypeNotFound {
		t.Errorf("NotFoundError type = %d, want ErrorTypeNotFound", ctx.Type)
	}
}

func TestValidationErrorFactory(t *testing.T) {
	ctx := ValidationError("validate input", fmt.Errorf("bad format"))

	if ctx.Type != ErrorTypeValidation {
		t.Errorf("ValidationError type = %d, want ErrorTypeValidation", ctx.Type)
	}
	if ctx.UserAction == "" {
		t.Error("ValidationError should set UserAction")
	}
}

// ---------------------------------------------------------------------------
// TransitionState struct
// ---------------------------------------------------------------------------

func TestTransitionState(t *testing.T) {
	state := TransitionState{
		Title:       "Test Title",
		Content:     []string{"line1", "line2"},
		Progress:    5,
		MaxProgress: 10,
		Theme:       "cosmic",
	}

	if state.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", state.Title, "Test Title")
	}
	if len(state.Content) != 2 {
		t.Errorf("Content length = %d, want 2", len(state.Content))
	}
	if state.Progress != 5 {
		t.Errorf("Progress = %d, want 5", state.Progress)
	}
	if state.MaxProgress != 10 {
		t.Errorf("MaxProgress = %d, want 10", state.MaxProgress)
	}
	if state.Theme != "cosmic" {
		t.Errorf("Theme = %q, want %q", state.Theme, "cosmic")
	}
}

// ---------------------------------------------------------------------------
// ShowAccessibleMenu - screen reader mode (showScreenReaderMenu: 75%)
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_ShowAccessibleMenu_ScreenReader(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	ah.Input = newMockReader("2")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Test Menu", []string{"Option A", "Option B", "Option C"}, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 1, result) // 0-indexed, selected "2" = index 1
	assert.Contains(t, buf.String(), "Test Menu")
}

func TestAccessibilityHelper_ShowAccessibleMenu_ScreenReader_Default(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	ah.Input = newMockReader("") // empty = default

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Pick", []string{"A", "B"}, 1)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 1, result) // returns defaultIndex
}

func TestAccessibilityHelper_ShowAccessibleMenu_ScreenReader_InvalidThenValid(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)
	ah.Input = newMockReader("99", "1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Pick", []string{"A", "B"}, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 0, result)
}

// ---------------------------------------------------------------------------
// ShowAccessibleMenu - large text mode (showLargeTextMenu: 73.7%)
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_ShowAccessibleMenu_LargeText(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityLargeText)
	ah.Input = newMockReader("1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Large Menu", []string{"Big A", "Big B"}, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 0, result)
	assert.Contains(t, buf.String(), "Large Menu")
}

func TestAccessibilityHelper_ShowAccessibleMenu_LargeText_Default(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityLargeText)
	ah.Input = newMockReader("")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Menu", []string{"X", "Y"}, 1)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 1, result)
}

func TestAccessibilityHelper_ShowAccessibleMenu_LargeText_InvalidThenValid(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityLargeText)
	ah.Input = newMockReader("abc", "2")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Menu", []string{"A", "B"}, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 1, result)
}

// ---------------------------------------------------------------------------
// ShowAccessibleMenu - standard mode (showStandardMenu: 88.2%)
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_ShowAccessibleMenu_Standard_InvalidThenValid(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityNone)
	ah.Input = newMockReader("99", "1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.ShowAccessibleMenu("Std Menu", []string{"A", "B"}, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, 0, result)
}

// ---------------------------------------------------------------------------
// GetAccessibleInput
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_GetAccessibleInput(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.Input = newMockReader("test input")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.GetAccessibleInput("Enter value", false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, "test input", result)
}

func TestAccessibilityHelper_GetAccessibleInput_Sensitive(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.Input = newMockReader("secret123")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := ah.GetAccessibleInput("Password", true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Equal(t, "secret123", result)
}

// ---------------------------------------------------------------------------
// ConfirmAccessibleYesNo
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_ConfirmAccessibleYesNo(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)

	tests := []struct {
		name      string
		input     string
		defaultYes bool
		want      bool
	}{
		{"y with default no", "y", false, true},
		{"empty with default yes", "", true, true},
		{"n with default yes", "n", true, false},
		{"empty with default no", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah.Input = newMockReader(tt.input)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			result := ah.ConfirmAccessibleYesNo("Confirm?", tt.defaultYes)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// ShowAccessibleProgress
// ---------------------------------------------------------------------------

func TestAccessibilityHelper_ShowAccessibleProgress(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah.ShowAccessibleProgress("Loading", 5, 10)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Loading")
}

func TestAccessibilityHelper_ShowAccessibleProgress_Complete(t *testing.T) {
	ah := NewAccessibilityHelper()
	ah.manager.SetMode(AccessibilityScreenReader)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ah.ShowAccessibleProgress("Done", 10, 10)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "100")
}

// ---------------------------------------------------------------------------
// AutoDetectAccessibility
// ---------------------------------------------------------------------------

func TestAutoDetectAccessibility_ScreenReaderEnv(t *testing.T) {
	t.Setenv("SCREEN_READER", "1")
	mgr := AutoDetectAccessibility()
	assert.True(t, mgr.IsEnabled())
	assert.True(t, mgr.config.ScreenReader)
}

func TestAutoDetectAccessibility_HighContrastEnv(t *testing.T) {
	t.Setenv("HIGH_CONTRAST", "1")
	mgr := AutoDetectAccessibility()
	assert.True(t, mgr.IsEnabled())
	assert.True(t, mgr.config.HighContrast)
}

func TestAutoDetectAccessibility_LargeTextEnv(t *testing.T) {
	t.Setenv("LARGE_TEXT", "1")
	mgr := AutoDetectAccessibility()
	assert.True(t, mgr.IsEnabled())
	assert.True(t, mgr.config.LargeText)
}

func TestAutoDetectAccessibility_ReducedMotionEnv(t *testing.T) {
	t.Setenv("REDUCED_MOTION", "1")
	mgr := AutoDetectAccessibility()
	assert.True(t, mgr.IsEnabled())
	assert.True(t, mgr.config.ReducedMotion)
}

func TestAutoDetectAccessibility_VSCodeFull(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("ACCESSIBILITY", "1")
	mgr := AutoDetectAccessibility()
	assert.Equal(t, AccessibilityFull, mgr.config.Mode)
}

func TestAutoDetectAccessibility_NoEnvs(t *testing.T) {
	mgr := AutoDetectAccessibility()
	assert.False(t, mgr.IsEnabled())
}

// ---------------------------------------------------------------------------
// Global accessibility functions
// ---------------------------------------------------------------------------

func TestEnableAccessibilityMode(t *testing.T) {
	origMode := globalAccessibilityManager.config.Mode
	defer func() { globalAccessibilityManager.config.Mode = origMode }()

	EnableAccessibilityMode(AccessibilityHighContrast)
	assert.True(t, IsAccessibilityEnabled())
}

// ---------------------------------------------------------------------------
// PrintAccessible methods
// ---------------------------------------------------------------------------

func TestPrintAccessible_Verbose(t *testing.T) {
	am := NewAccessibilityManager()
	am.config.Verbose = true

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessible("Test message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Test message")
}

func TestPrintAccessibleSuccess(t *testing.T) {
	am := NewAccessibilityManager()
	am.config.ScreenReader = true
	am.config.Verbose = true

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleSuccess("It worked")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "It worked")
}

func TestPrintAccessibleError(t *testing.T) {
	am := NewAccessibilityManager()
	am.config.ScreenReader = true
	am.config.Verbose = true

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.PrintAccessibleError("Something failed")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "Something failed")
}
