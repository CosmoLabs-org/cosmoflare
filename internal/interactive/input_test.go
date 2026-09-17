package interactive

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReader provides canned input lines for testing
type mockReader struct {
	inputs []string
	index  int
}

func (m *mockReader) ReadLine() (string, error) {
	if m.index >= len(m.inputs) {
		return "", io.EOF
	}
	line := m.inputs[m.index]
	m.index++
	return line, nil
}

func newMockReader(inputs ...string) *mockReader {
	return &mockReader{inputs: inputs}
}

// --- SetupWizard with InputReader ---

// TestSetupWizard_Step1_AuthMethod verifies SetupWizard behavior for the step1 case and auth...
func TestSetupWizard_Step1_AuthMethod(t *testing.T) {
	t.Run("default selection returns api_token", func(t *testing.T) {
		w := NewSetupWizard()
		w.Quiet = true
		w.Input = newMockReader("") // empty = default
		method, err := w.Step1_AuthMethod()
		require.NoError(t, err)
		assert.Equal(t, "api_token", method)
	})

	t.Run("selecting 1 returns api_token", func(t *testing.T) {
		w := NewSetupWizard()
		w.Quiet = true
		w.Input = newMockReader("1")
		method, err := w.Step1_AuthMethod()
		require.NoError(t, err)
		assert.Equal(t, "api_token", method)
	})

	t.Run("selecting 2 returns service_key", func(t *testing.T) {
		w := NewSetupWizard()
		w.Quiet = true
		w.Input = newMockReader("2")
		method, err := w.Step1_AuthMethod()
		require.NoError(t, err)
		assert.Equal(t, "service_key", method)
	})

	t.Run("selecting 3 returns env", func(t *testing.T) {
		w := NewSetupWizard()
		w.Quiet = true
		w.Input = newMockReader("3")
		method, err := w.Step1_AuthMethod()
		require.NoError(t, err)
		assert.Equal(t, "env", method)
	})

	t.Run("invalid then valid input", func(t *testing.T) {
		w := NewSetupWizard()
		w.Quiet = true
		w.Input = newMockReader("x", "2")
		method, err := w.Step1_AuthMethod()
		require.NoError(t, err)
		assert.Equal(t, "service_key", method)
	})
}

// --- PromptWithDefault with InputReader ---

// TestPromptWithReader verifies PromptWithReader behavior, one t.Run subtest per scenario.
func TestPromptWithReader(t *testing.T) {
	t.Run("returns user input", func(t *testing.T) {
		reader := newMockReader("hello")
		result, err := PromptWithReader("Enter name", "", reader)
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns default on empty input", func(t *testing.T) {
		reader := newMockReader("")
		result, err := PromptWithReader("Enter name", "default-val", reader)
		require.NoError(t, err)
		assert.Equal(t, "default-val", result)
	})
}

// --- SelectFromList with InputReader ---

// TestSelectFromListWithReader verifies SelectFromListWithReader behavior, one t.Run subtest per...
func TestSelectFromListWithReader(t *testing.T) {
	options := []string{"Apple", "Banana", "Cherry"}

	t.Run("selects by number", func(t *testing.T) {
		reader := newMockReader("2")
		idx, err := SelectFromListWithReader("Pick fruit:", options, -1, reader)
		require.NoError(t, err)
		assert.Equal(t, 1, idx) // 0-indexed
	})

	t.Run("returns default on empty input", func(t *testing.T) {
		reader := newMockReader("")
		idx, err := SelectFromListWithReader("Pick fruit:", options, 0, reader)
		require.NoError(t, err)
		assert.Equal(t, 0, idx)
	})

	t.Run("invalid then valid input", func(t *testing.T) {
		reader := newMockReader("9", "3")
		idx, err := SelectFromListWithReader("Pick fruit:", options, -1, reader)
		require.NoError(t, err)
		assert.Equal(t, 2, idx)
	})
}

// --- ConfirmYesNo with InputReader ---

// TestConfirmWithReader verifies ConfirmWithReader behavior, one t.Run subtest per scenario.
func TestConfirmWithReader(t *testing.T) {
	t.Run("empty input returns default yes", func(t *testing.T) {
		reader := newMockReader("")
		result := ConfirmWithReader("Continue?", true, reader)
		assert.True(t, result)
	})

	t.Run("empty input returns default no", func(t *testing.T) {
		reader := newMockReader("")
		result := ConfirmWithReader("Continue?", false, reader)
		assert.False(t, result)
	})

	t.Run("y returns true", func(t *testing.T) {
		reader := newMockReader("y")
		result := ConfirmWithReader("Continue?", false, reader)
		assert.True(t, result)
	})

	t.Run("yes returns true", func(t *testing.T) {
		reader := newMockReader("yes")
		result := ConfirmWithReader("Continue?", false, reader)
		assert.True(t, result)
	})

	t.Run("n returns false", func(t *testing.T) {
		reader := newMockReader("n")
		result := ConfirmWithReader("Continue?", true, reader)
		assert.False(t, result)
	})

	t.Run("no returns false", func(t *testing.T) {
		reader := newMockReader("no")
		result := ConfirmWithReader("Continue?", true, reader)
		assert.False(t, result)
	})
}
