package ux

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultConfirmationOptions ---

func TestDefaultConfirmationOptions(t *testing.T) {
	opts := DefaultConfirmationOptions()
	assert.NotNil(t, opts)
	assert.Equal(t, ConfirmationTypeYesNo, opts.Type)
	assert.False(t, opts.Default)
	assert.Equal(t, 0, opts.Timeout)
	assert.False(t, opts.ShowDetails)
	assert.Equal(t, "Do you want to continue?", opts.Message)
}

// --- parseConfirmationInput ---

func TestParseConfirmationInput_YesNo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"empty input defaults to no", "", false, false, false},
		{"y accepts", "y", true, false, false},
		{"yes accepts", "yes", true, false, false},
		{"n rejects", "n", false, false, false},
		{"no rejects", "no", false, false, false},
		{"random input rejects", "random", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeYesNo)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_YesNoAll(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"y accepts", "y", true, false, false},
		{"yes accepts", "yes", true, false, false},
		{"n rejects", "n", false, false, false},
		{"no rejects", "no", false, false, false},
		{"a applies to all", "a", true, true, false},
		{"all applies to all", "all", true, true, false},
		{"l rejects (less)", "l", false, false, false},
		{"less rejects", "less", false, false, false},
		{"random input rejects", "random", false, false, false},
		{"empty input defaults to no", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeYesNoAll)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_YesNoCancel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"y accepts", "y", true, false, false},
		{"yes accepts", "yes", true, false, false},
		{"n rejects", "n", false, false, false},
		{"no rejects", "no", false, false, false},
		{"c cancels", "c", false, false, true},
		{"cancel cancels", "cancel", false, false, true},
		{"random input rejects", "random", false, false, false},
		{"empty input defaults to no", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeYesNoCancel)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_Continue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"continue accepts", "continue", true, false, false},
		{"c accepts", "c", true, false, false},
		{"cancel cancels", "cancel", false, false, true},
		{"random input rejects", "random", false, false, false},
		{"empty input defaults to no", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeContinue)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_Retry(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"retry accepts", "retry", true, false, false},
		{"r accepts", "r", true, false, false},
		{"cancel cancels", "cancel", false, false, true},
		{"random input rejects", "random", false, false, false},
		{"empty input defaults to no", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeRetry)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_Overwrite(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		answer    bool
		all       bool
		cancelled bool
	}{
		{"y accepts", "y", true, false, false},
		{"yes accepts", "yes", true, false, false},
		{"o overwrites", "o", true, false, false},
		{"overwrite overwrites", "overwrite", true, false, false},
		{"n skips", "n", false, false, false},
		{"no skips", "no", false, false, false},
		{"s skips", "s", false, false, false},
		{"skip skips", "skip", false, false, false},
		{"random input skips", "random", false, false, false},
		{"empty input defaults to no", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(tt.input, ConfirmationTypeOverwrite)
			assert.Equal(t, tt.answer, answer)
			assert.Equal(t, tt.all, all)
			assert.Equal(t, tt.cancelled, cancelled)
		})
	}
}

func TestParseConfirmationInput_UnknownType(t *testing.T) {
	// Unknown type falls through to default branch which treats it like YesNo
	answer, all, cancelled := parseConfirmationInput("y", ConfirmationType(99))
	assert.True(t, answer)
	assert.False(t, all)
	assert.False(t, cancelled)
}

// --- determineConfidence ---

func TestDetermineConfidence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty input is low", "", "low"},
		{"single char is medium", "y", "medium"},
		{"two chars is low", "no", "low"},
		{"full word is high", "yes", "high"},
		{"long input is high", "absolutely", "high"},
		{"cancel is high", "cancel", "high"},
		{"overwrite is high", "overwrite", "high"},
		{"single char n is medium", "n", "medium"},
		{"three chars is high", "all", "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineConfidence(tt.input, ConfirmationTypeYesNo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- AutoConfirm ---

func TestAutoConfirm_True(t *testing.T) {
	result := AutoConfirm(true)
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
	assert.Equal(t, "auto", result.Confidence)
	assert.False(t, result.All)
	assert.False(t, result.Cancelled)
	assert.False(t, result.Timeout)
	assert.Equal(t, "", result.UserInput)
}

func TestAutoConfirm_False(t *testing.T) {
	result := AutoConfirm(false)
	assert.NotNil(t, result)
	assert.False(t, result.Answer)
	assert.Equal(t, "auto", result.Confidence)
	assert.False(t, result.All)
	assert.False(t, result.Cancelled)
	assert.False(t, result.Timeout)
	assert.Equal(t, "", result.UserInput)
}

// --- PromptConfirmation (via stdin redirect) ---

// redirectStdin redirects os.Stdin to read from the provided string.
// It returns a restore function that must be called to restore original stdin.
func redirectStdin(t *testing.T, input string) func() {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdin = r

	go func() {
		w.WriteString(input)
		w.Close()
	}()

	return func() {
		r.Close()
		os.Stdin = oldStdin
	}
}

func TestPromptConfirmation_NilOpts(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	result := PromptConfirmation(nil)
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
	assert.Equal(t, "y", result.UserInput)
}

func TestPromptConfirmation_YesInput(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
	assert.False(t, result.All)
	assert.False(t, result.Cancelled)
	assert.Equal(t, "yes", result.UserInput)
	assert.Equal(t, "high", result.Confidence)
}

func TestPromptConfirmation_NoInput(t *testing.T) {
	restore := redirectStdin(t, "no\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.False(t, result.Answer)
	assert.Equal(t, "no", result.UserInput)
}

func TestPromptConfirmation_EmptyInput(t *testing.T) {
	restore := redirectStdin(t, "\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.False(t, result.Answer)
	assert.Equal(t, "", result.UserInput)
	assert.Equal(t, "low", result.Confidence)
}

func TestPromptConfirmation_AllInput(t *testing.T) {
	restore := redirectStdin(t, "all\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNoAll,
		Message: "Apply to all?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
	assert.True(t, result.All)
	assert.False(t, result.Cancelled)
}

func TestPromptConfirmation_CancelInput(t *testing.T) {
	restore := redirectStdin(t, "cancel\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNoCancel,
		Message: "Proceed?",
	}
	result := PromptConfirmation(opts)
	assert.False(t, result.Answer)
	assert.True(t, result.Cancelled)
}

func TestPromptConfirmation_OverwriteYes(t *testing.T) {
	restore := redirectStdin(t, "o\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeOverwrite,
		Message: "Overwrite file.txt?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
	assert.False(t, result.Cancelled)
}

func TestPromptConfirmation_OverwriteSkip(t *testing.T) {
	restore := redirectStdin(t, "skip\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeOverwrite,
		Message: "Overwrite file.txt?",
	}
	result := PromptConfirmation(opts)
	assert.False(t, result.Answer)
}

func TestPromptConfirmation_ReaderError(t *testing.T) {
	oldStdin := os.Stdin
	// Create a pipe but close the read end immediately to cause an error
	r, w, err := os.Pipe()
	require.NoError(t, err)
	r.Close() // close read end to trigger error
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		w.Close()
	}()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Cancelled)
}

func TestPromptConfirmation_CaseInsensitive(t *testing.T) {
	restore := redirectStdin(t, "YES\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
	assert.Equal(t, "yes", result.UserInput)
}

func TestPromptConfirmation_WhitespaceHandling(t *testing.T) {
	restore := redirectStdin(t, "  y  \n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
	assert.Equal(t, "y", result.UserInput)
}

// --- printConfirmationMessage ---

func TestPrintConfirmationMessage_AllTypes(t *testing.T) {
	types := []ConfirmationType{
		ConfirmationTypeYesNo,
		ConfirmationTypeYesNoAll,
		ConfirmationTypeYesNoCancel,
		ConfirmationTypeContinue,
		ConfirmationTypeRetry,
		ConfirmationTypeOverwrite,
		ConfirmationType(99), // unknown type
	}

	for _, ct := range types {
		t.Run(fmt.Sprintf("type_%d", ct), func(t *testing.T) {
			assert.NotPanics(t, func() {
				printConfirmationMessage(&ConfirmationOptions{
					Type:    ct,
					Message: "Test message",
				})
			})
		})
	}
}

func TestPrintConfirmationMessage_WithWarning(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:    ConfirmationTypeYesNo,
			Message: "Delete everything?",
			Warning: "This action cannot be undone!",
		})
	})
}

func TestPrintConfirmationMessage_WithDetails(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:        ConfirmationTypeYesNo,
			Message:     "Delete files?",
			ShowDetails: true,
			Details:     "Files: a.txt, b.txt, c.txt",
		})
	})
}

func TestPrintConfirmationMessage_DefaultTrue(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:    ConfirmationTypeYesNo,
			Message: "Continue?",
			Default: true,
		})
	})
}

func TestPrintConfirmationMessage_OverwriteDefaultTrue(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:    ConfirmationTypeOverwrite,
			Message: "Overwrite?",
			Default: true,
		})
	})
}

func TestPrintConfirmationMessage_DetailsHiddenWhenShowDetailsFalse(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:        ConfirmationTypeYesNo,
			Message:     "Continue?",
			ShowDetails: false,
			Details:     "Should not appear",
		})
	})
}

func TestPrintConfirmationMessage_EmptyWarning(t *testing.T) {
	assert.NotPanics(t, func() {
		printConfirmationMessage(&ConfirmationOptions{
			Type:    ConfirmationTypeYesNo,
			Message: "Continue?",
			Warning: "",
		})
	})
}

// --- Confirm (simple wrapper) ---

func TestConfirm_Yes(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, Confirm("Do you want to proceed?"))
}

func TestConfirm_No(t *testing.T) {
	restore := redirectStdin(t, "n\n")
	defer restore()

	assert.False(t, Confirm("Do you want to proceed?"))
}

func TestConfirm_Empty(t *testing.T) {
	restore := redirectStdin(t, "\n")
	defer restore()

	assert.False(t, Confirm("Do you want to proceed?"))
}

func TestConfirm_UnicodePrompt(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	assert.True(t, Confirm("Delete all files?"))
}

// --- ConfirmWithDetails ---

func TestConfirmWithDetails_Yes(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmWithDetails("Delete files?", "file1.txt\nfile2.txt"))
}

func TestConfirmWithDetails_No(t *testing.T) {
	restore := redirectStdin(t, "no\n")
	defer restore()

	assert.False(t, ConfirmWithDetails("Delete files?", "file1.txt\nfile2.txt"))
}

func TestConfirmWithDetails_EmptyDetails(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmWithDetails("Continue?", ""))
}

func TestConfirmWithDetails_UnicodeDetails(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	assert.True(t, ConfirmWithDetails("Delete?", "file-cafe.txt, file-naive.txt"))
}

func TestConfirmWithDetails_LongDetails(t *testing.T) {
	longDetails := strings.Repeat("detail line\n", 100)
	restore := redirectStdin(t, "n\n")
	defer restore()

	assert.False(t, ConfirmWithDetails("Delete?", longDetails))
}

// --- ConfirmDestructive ---

func TestConfirmDestructive_Yes(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmDestructive("delete all files", []string{"a.txt", "b.txt"}))
}

func TestConfirmDestructive_No(t *testing.T) {
	restore := redirectStdin(t, "n\n")
	defer restore()

	assert.False(t, ConfirmDestructive("delete all files", []string{"a.txt", "b.txt"}))
}

func TestConfirmDestructive_EmptyTargets(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmDestructive("nuke everything", nil))
}

func TestConfirmDestructive_ManyTargets(t *testing.T) {
	targets := make([]string, 50)
	for i := range targets {
		targets[i] = fmt.Sprintf("file%d.txt", i)
	}

	restore := redirectStdin(t, "no\n")
	defer restore()

	assert.False(t, ConfirmDestructive("delete files", targets))
}

func TestConfirmDestructive_ExactlyTenTargets(t *testing.T) {
	targets := make([]string, 10)
	for i := range targets {
		targets[i] = fmt.Sprintf("file%d.txt", i)
	}

	restore := redirectStdin(t, "yes\n")
	defer restore()

	assert.True(t, ConfirmDestructive("delete files", targets))
}

func TestConfirmDestructive_ElevenTargets(t *testing.T) {
	targets := make([]string, 11)
	for i := range targets {
		targets[i] = fmt.Sprintf("file%d.txt", i)
	}

	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmDestructive("delete files", targets))
}

func TestConfirmDestructive_UnicodeTargets(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmDestructive("delete", []string{"resume.txt", "naive.txt"}))
}

// --- ConfirmOverwrite ---

func TestConfirmOverwrite_Yes(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmOverwrite("output.txt", false))
}

func TestConfirmOverwrite_No(t *testing.T) {
	restore := redirectStdin(t, "n\n")
	defer restore()

	assert.False(t, ConfirmOverwrite("output.txt", false))
}

func TestConfirmOverwrite_OverwriteCmd(t *testing.T) {
	restore := redirectStdin(t, "o\n")
	defer restore()

	assert.True(t, ConfirmOverwrite("output.txt", true))
}

func TestConfirmOverwrite_Skip(t *testing.T) {
	restore := redirectStdin(t, "s\n")
	defer restore()

	assert.False(t, ConfirmOverwrite("output.txt", false))
}

func TestConfirmOverwrite_NewerSource(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	assert.True(t, ConfirmOverwrite("file.txt", true))
}

func TestConfirmOverwrite_EmptyFilename(t *testing.T) {
	restore := redirectStdin(t, "no\n")
	defer restore()

	assert.False(t, ConfirmOverwrite("", false))
}

func TestConfirmOverwrite_UnicodeFilename(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmOverwrite("cafe-report.txt", true))
}

func TestConfirmOverwrite_LongFilename(t *testing.T) {
	restore := redirectStdin(t, "no\n")
	defer restore()

	longName := strings.Repeat("a", 500) + ".txt"
	assert.False(t, ConfirmOverwrite(longName, false))
}

// --- InteractiveConfirmation ---

func TestInteractiveConfirmation_NilOpts(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	result := InteractiveConfirmation(nil, 0)
	assert.NotNil(t, result)
	// PromptConfirmation with nil opts creates default opts, reads "y" from stdin
	assert.True(t, result.Answer)
}

func TestInteractiveConfirmation_WithOpts(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Proceed?",
	}
	result := InteractiveConfirmation(opts, 30)
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
}

// --- BatchConfirmation ---

func TestBatchConfirmation_Yes(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	result := BatchConfirmation([]string{"op1", "op2"}, "delete")
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
}

func TestBatchConfirmation_All(t *testing.T) {
	restore := redirectStdin(t, "all\n")
	defer restore()

	result := BatchConfirmation([]string{"op1", "op2"}, "upload")
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
	assert.True(t, result.All)
}

func TestBatchConfirmation_No(t *testing.T) {
	restore := redirectStdin(t, "n\n")
	defer restore()

	result := BatchConfirmation([]string{"op1"}, "download")
	assert.NotNil(t, result)
	assert.False(t, result.Answer)
}

func TestBatchConfirmation_EmptyOperations(t *testing.T) {
	restore := redirectStdin(t, "yes\n")
	defer restore()

	result := BatchConfirmation([]string{}, "delete")
	assert.NotNil(t, result)
	assert.True(t, result.Answer)
}

func TestBatchConfirmation_ManyOperations(t *testing.T) {
	ops := make([]string, 20)
	for i := range ops {
		ops[i] = fmt.Sprintf("upload file%d.txt", i)
	}

	restore := redirectStdin(t, "a\n")
	defer restore()

	result := BatchConfirmation(ops, "upload")
	assert.NotNil(t, result)
	assert.True(t, result.All)
}

// --- SECURITY: Confirmation bypass attempts ---

func TestSecurity_ConfirmationBypass_CraftedInput(t *testing.T) {
	bypassAttempts := []string{
		"Y",          // uppercase (should be lowered)
		"YES",        // uppercase (should be lowered)
		" Yes",       // leading space
		"y\nr\n",     // injection attempt
		"y; rm -rf /", // command injection
		"y\rmalicious", // carriage return
		"y\x00evil",   // null byte injection
	}

	for _, attempt := range bypassAttempts {
		t.Run(fmt.Sprintf("bypass_%q", attempt), func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(
				strings.ToLower(strings.TrimSpace(attempt)),
				ConfirmationTypeYesNo,
			)
			// After lower+trim, only actual "y"/"yes" should pass
			// Y->y, YES->yes (lowered), Yes->yes, etc.
			input := strings.ToLower(strings.TrimSpace(attempt))
			expected := input == "y" || input == "yes"
			assert.Equal(t, expected, answer)
			assert.False(t, all)
			assert.False(t, cancelled)
		})
	}
}

func TestSecurity_ConfirmationBypass_CancelBypass(t *testing.T) {
	// Ensure cancel can't be tricked into being "yes"
	answer, _, cancelled := parseConfirmationInput("cancel", ConfirmationTypeYesNoCancel)
	assert.False(t, answer)
	assert.True(t, cancelled)
}

func TestSecurity_ConfirmRetryBypass(t *testing.T) {
	// parseConfirmationInput does NOT lowercase - it requires exact lowercase match
	answer, _, _ := parseConfirmationInput("R", ConfirmationTypeRetry)
	assert.False(t, answer) // "R" != "r" and "R" != "retry"

	answer, _, _ = parseConfirmationInput("r", ConfirmationTypeRetry)
	assert.True(t, answer) // "r" is valid

	answer, _, _ = parseConfirmationInput("retry", ConfirmationTypeRetry)
	assert.True(t, answer) // "retry" is valid
}

func TestSecurity_OverwriteBypass(t *testing.T) {
	// Ensure only valid overwrite commands work
	answer, _, _ := parseConfirmationInput("delete", ConfirmationTypeOverwrite)
	assert.False(t, answer) // "delete" is not a valid overwrite command

	answer, _, _ = parseConfirmationInput("o", ConfirmationTypeOverwrite)
	assert.True(t, answer) // "o" is valid
}

// --- ROBUSTNESS: Edge cases ---

func TestRobustness_ConfirmationWithSpecialChars(t *testing.T) {
	specialInputs := []string{
		"\t",
		"\n\n",
		"   ",
		"!@#$%",
		"<script>alert(1)</script>",
		"' OR '1'='1",
		"\x1b[31m y \x1b[0m", // ANSI escape codes
	}

	for _, input := range specialInputs {
		t.Run(fmt.Sprintf("special_%q", input), func(t *testing.T) {
			answer, all, cancelled := parseConfirmationInput(input, ConfirmationTypeYesNo)
			// All special inputs should be rejected
			assert.False(t, answer, "special input should not be accepted: %q", input)
			assert.False(t, all)
			assert.False(t, cancelled)
		})
	}
}

func TestRobustness_PromptConfirmationWithClosedReader(t *testing.T) {
	oldStdin := os.Stdin
	// Use /dev/null to simulate EOF immediately
	f, err := os.Open(os.DevNull)
	require.NoError(t, err)
	os.Stdin = f

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	// Reading from /dev/null returns EOF immediately
	// bufio.ReadString('\n') returns io.EOF
	assert.True(t, result.Cancelled)

	os.Stdin = oldStdin
	f.Close()
}

func TestRobustness_ConfirmDestructiveWithSpecialCharsInTargets(t *testing.T) {
	restore := redirectStdin(t, "no\n")
	defer restore()

	targets := []string{
		"file; rm -rf /",
		"$(whoami)",
		"`cat /etc/passwd`",
		"../../etc/shadow",
	}
	assert.False(t, ConfirmDestructive("delete", targets))
}

func TestRobustness_ConfirmOverwritePathTraversal(t *testing.T) {
	restore := redirectStdin(t, "n\n")
	defer restore()

	assert.False(t, ConfirmOverwrite("../../etc/passwd", false))
}

func TestRobustness_ConfirmWithDetailsEmptyPrompt(t *testing.T) {
	restore := redirectStdin(t, "y\n")
	defer restore()

	assert.True(t, ConfirmWithDetails("", ""))
}

func TestRobustness_ConfirmWithVeryLongMessage(t *testing.T) {
	longMsg := strings.Repeat("Delete ", 1000)
	restore := redirectStdin(t, "no\n")
	defer restore()

	assert.False(t, Confirm(longMsg))
}

// --- IO error handling ---

func TestRobustness_PromptConfirmation_IOError(t *testing.T) {
	oldStdin := os.Stdin
	// Use a read-closed pipe to simulate IO error
	r, w, _ := os.Pipe()
	w.Close()
	r.Close()
	os.Stdin = r

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Cancelled)

	os.Stdin = oldStdin
}

// --- Test with io.EOF via limited reader ---

func TestRobustness_PromptConfirmation_EOFReader(t *testing.T) {
	oldStdin := os.Stdin
	f, err := os.Open(os.DevNull)
	require.NoError(t, err)
	os.Stdin = f

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Cancelled)

	os.Stdin = oldStdin
	f.Close()
}

func TestRobustness_PromptConfirmation_MultipleLines(t *testing.T) {
	// Only first line should be read
	restore := redirectStdin(t, "y\nno\n")
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.True(t, result.Answer)
}

func TestRobustness_PromptConfirmation_VeryLongInput(t *testing.T) {
	longInput := strings.Repeat("a", 10000) + "\n"
	restore := redirectStdin(t, longInput)
	defer restore()

	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeYesNo,
		Message: "Continue?",
	}
	result := PromptConfirmation(opts)
	assert.False(t, result.Answer) // long garbage input -> not "y"/"yes"
}

// --- parseConfirmationInput edge: two-char boundary ---

func TestDetermineConfidence_BoundaryCases(t *testing.T) {
	// len == 2 should be "low" (not > 2, not == 1)
	assert.Equal(t, "low", determineConfidence("ab", ConfirmationTypeYesNo))
	assert.Equal(t, "low", determineConfidence("12", ConfirmationTypeYesNo))

	// len == 3 should be "high" (> 2)
	assert.Equal(t, "high", determineConfidence("abc", ConfirmationTypeYesNo))
}
