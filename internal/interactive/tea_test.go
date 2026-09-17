package interactive

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTextInputModel verifies text input model behavior, one t.Run subtest per scenario.
func TestTextInputModel(t *testing.T) {
	t.Run("init creates a model", func(t *testing.T) {
		m := newTextInputModel("Enter name", "default")
		if m.prompt != "Enter name" {
			t.Errorf("expected prompt 'Enter name', got %q", m.prompt)
		}
		if m.defaultValue != "default" {
			t.Errorf("expected default 'default', got %q", m.defaultValue)
		}
	})

	t.Run("enter with empty input returns default", func(t *testing.T) {
		m := newTextInputModel("Name", "fallback")
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		final := result.(textInputModel)
		if !final.done {
			t.Error("expected done after enter")
		}
		if final.value != "fallback" {
			t.Errorf("expected 'fallback', got %q", final.value)
		}
	})

	t.Run("escape cancels", func(t *testing.T) {
		m := newTextInputModel("Name", "")
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		final := result.(textInputModel)
		if !final.cancelled {
			t.Error("expected cancelled after esc")
		}
	})

	t.Run("view renders prompt", func(t *testing.T) {
		m := newTextInputModel("Test prompt", "def")
		view := m.View()
		if view == "" {
			t.Error("expected non-empty view")
		}
	})
}

// TestSelectModel verifies select model behavior, one t.Run subtest per scenario.
func TestSelectModel(t *testing.T) {
	options := []string{"Alpha", "Beta", "Gamma"}

	t.Run("init creates model at default index", func(t *testing.T) {
		m := newSelectModel("Pick one", options, 1)
		if m.cursor != 1 {
			t.Errorf("expected cursor at 1, got %d", m.cursor)
		}
	})

	t.Run("down moves cursor", func(t *testing.T) {
		m := newSelectModel("Pick", options, 0)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		final := result.(selectModel)
		if final.cursor != 1 {
			t.Errorf("expected cursor at 1 after down, got %d", final.cursor)
		}
	})

	t.Run("up moves cursor", func(t *testing.T) {
		m := newSelectModel("Pick", options, 2)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		final := result.(selectModel)
		if final.cursor != 1 {
			t.Errorf("expected cursor at 1 after up, got %d", final.cursor)
		}
	})

	t.Run("enter selects current option", func(t *testing.T) {
		m := newSelectModel("Pick", options, 0)
		m.cursor = 2
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		final := result.(selectModel)
		if !final.done {
			t.Error("expected done after enter")
		}
		if final.selected != 2 {
			t.Errorf("expected selected 2, got %d", final.selected)
		}
	})

	t.Run("cursor stays in bounds", func(t *testing.T) {
		m := newSelectModel("Pick", options, 0)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		final := result.(selectModel)
		if final.cursor != 0 {
			t.Errorf("expected cursor to stay at 0, got %d", final.cursor)
		}

		m = newSelectModel("Pick", options, 2)
		result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		final = result.(selectModel)
		if final.cursor != 2 {
			t.Errorf("expected cursor to stay at 2, got %d", final.cursor)
		}
	})
}

// TestConfirmModel verifies Confirm behavior for the model case, one t.Run subtest per scenario.
func TestConfirmModel(t *testing.T) {
	t.Run("y confirms", func(t *testing.T) {
		m := newConfirmModel("Continue?", false)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		final := result.(confirmModel)
		if !final.confirmed {
			t.Error("expected confirmed after 'y'")
		}
		if !final.done {
			t.Error("expected done after 'y'")
		}
	})

	t.Run("n denies", func(t *testing.T) {
		m := newConfirmModel("Continue?", true)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		final := result.(confirmModel)
		if final.confirmed {
			t.Error("expected not confirmed after 'n'")
		}
	})

	t.Run("enter uses default yes", func(t *testing.T) {
		m := newConfirmModel("Continue?", true)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		final := result.(confirmModel)
		if !final.confirmed {
			t.Error("expected confirmed with default yes")
		}
	})

	t.Run("enter uses default no", func(t *testing.T) {
		m := newConfirmModel("Continue?", false)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		final := result.(confirmModel)
		if final.confirmed {
			t.Error("expected not confirmed with default no")
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		m := newConfirmModel("Continue?", true)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		final := result.(confirmModel)
		if final.confirmed {
			t.Error("expected not confirmed after esc")
		}
	})
}
