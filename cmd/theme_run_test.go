/*
Package cmd provides the theme run-function tests

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// themeRunGlobals snapshots and restores the theme flag globals and resets
// the global theme manager back to the default theme after each test.
func themeRunGlobals(t *testing.T) {
	t.Helper()
	oldList, oldSet := themeList, themeSet
	oldCreate, oldInfo := themeCreate, themeInfo
	t.Cleanup(func() {
		themeList, themeSet = oldList, oldSet
		themeCreate, themeInfo = oldCreate, oldInfo
		// runTheme mutates the shared global theme manager; restore it so
		// other tests that assert on the default theme are not affected.
		_ = interactive.SetGlobalTheme("cosmic")
	})
}

// themeResetFlags clears the theme flag globals to their zero values.
func themeResetFlags() {
	themeList = false
	themeSet = ""
	themeCreate = false
	themeInfo = false
}

// TestRunTheme_List verifies the --list branch enumerates the builtin themes
// and reports the current one without error.
func TestRunTheme_List(t *testing.T) {
	themeRunGlobals(t)
	themeResetFlags()
	themeList = true

	if err := runTheme(themeCmd, nil); err != nil {
		t.Fatalf("theme --list should succeed: %v", err)
	}
}

// TestRunTheme_ListMarksCurrent verifies each builtin theme appears in the
// manager's listing and exactly one matches the current theme name.
func TestRunTheme_ListMarksCurrent(t *testing.T) {
	tm := interactive.GetThemeManager()
	themes := tm.ListThemes()
	if len(themes) < 5 {
		t.Fatalf("expected at least 5 builtin themes, got %d", len(themes))
	}
	current := tm.GetCurrentTheme()
	if current == nil {
		t.Fatal("current theme must not be nil by default")
	}
	marked := 0
	for _, th := range themes {
		if th.Name == current.Name {
			marked++
		}
	}
	if marked != 1 {
		t.Fatalf("expected exactly 1 theme matching current name, got %d", marked)
	}
}

// TestRunTheme_SetValid verifies --set with each builtin theme name succeeds
// and updates the manager's current theme.
func TestRunTheme_SetValid(t *testing.T) {
	for _, name := range []string{"cosmic", "forest", "ocean", "sunset", "monochrome"} {
		t.Run(name, func(t *testing.T) {
			themeRunGlobals(t)
			themeResetFlags()
			themeSet = name

			if err := runTheme(themeCmd, nil); err != nil {
				t.Fatalf("theme --set=%s should succeed: %v", name, err)
			}
			if got := interactive.GetThemeManager().GetCurrentTheme().ID; got != name {
				t.Fatalf("current theme ID = %q, want %q", got, name)
			}
		})
	}
}

// TestRunTheme_SetInvalid verifies --set with an unknown theme surfaces the
// manager's not-found error.
func TestRunTheme_SetInvalid(t *testing.T) {
	themeRunGlobals(t)
	themeResetFlags()
	themeSet = "no-such-theme"

	err := runTheme(themeCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "theme 'no-such-theme' not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

// TestRunTheme_Info verifies the --info branch prints the current theme
// details without error.
func TestRunTheme_Info(t *testing.T) {
	themeRunGlobals(t)
	themeResetFlags()
	themeInfo = true

	if err := runTheme(themeCmd, nil); err != nil {
		t.Fatalf("theme --info should succeed: %v", err)
	}
}
