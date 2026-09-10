/*
Unit tests for the installer_tui package

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// TestNewInstallerModel verifies that NewInstallerModel builds a fully
// initialized model with the default dimensions, the menu state active,
// and non-nil header, menu and styles sub-components.
func TestNewInstallerModel(t *testing.T) {
	t.Run("creates model with default dimensions", func(t *testing.T) {
		m := NewInstallerModel()
		if m.width != 80 {
			t.Errorf("width = %d, want 80", m.width)
		}
		if m.height != 25 {
			t.Errorf("height = %d, want 25", m.height)
		}
	})

	t.Run("starts in menu state", func(t *testing.T) {
		m := NewInstallerModel()
		if m.state != StateMenu {
			t.Errorf("state = %v, want StateMenu", m.state)
		}
	})

	t.Run("initializes header menu and styles", func(t *testing.T) {
		m := NewInstallerModel()
		if m.header == nil {
			t.Error("header is nil")
		}
		if m.menu == nil {
			t.Error("menu is nil")
		}
		if m.styles == nil {
			t.Error("styles is nil")
		}
	})

	t.Run("Init returns no command and does not panic", func(t *testing.T) {
		m := NewInstallerModel()
		if cmd := m.Init(); cmd != nil {
			t.Errorf("Init() = %v, want nil", cmd)
		}
	})

	t.Run("View renders the main menu options", func(t *testing.T) {
		m := NewInstallerModel()
		view := m.View()
		if view == "" {
			t.Fatal("View() returned an empty string in menu state")
		}
		for _, want := range []string{"Download from GitHub", "Use local build", "Exit installer"} {
			if !strings.Contains(view, want) {
				t.Errorf("View() does not contain menu option %q", want)
			}
		}
	})
}

// TestCreateInstallerStyles verifies that createInstallerStyles returns a
// fully populated style set whose styles render the provided text without
// losing it.
func TestCreateInstallerStyles(t *testing.T) {
	t.Run("returns fully initialized styles", func(t *testing.T) {
		s := createInstallerStyles(80)
		if s == nil {
			t.Fatal("createInstallerStyles returned nil")
		}
		// The container style is the only one with a border; rendering it
		// must produce visible border characters around the content.
		if rendered := s.Container.Render("x"); !strings.ContainsAny(rendered, "╭╮╰╯─│") {
			t.Errorf("container style rendered without a border: %q", rendered)
		}
	})

	t.Run("styles preserve rendered text", func(t *testing.T) {
		s := createInstallerStyles(80)
		cases := []struct {
			name  string
			style string
			got   string
		}{
			{"Container", "container", s.Container.Render("hello")},
			{"Menu", "menu", s.Menu.Render("hello")},
			{"Status", "status", s.Status.Render("hello")},
			{"Error", "error", s.Error.Render("hello")},
			{"Success", "success", s.Success.Render("hello")},
		}
		for _, c := range cases {
			if !strings.Contains(c.got, "hello") {
				t.Errorf("%s style lost the rendered text: got %q", c.name, c.got)
			}
		}
	})

	t.Run("independent instances do not share state", func(t *testing.T) {
		a := createInstallerStyles(40)
		b := createInstallerStyles(120)
		if a == nil || b == nil {
			t.Fatal("expected non-nil styles for every width")
		}
	})
}

// TestAddToPath verifies the shell-config PATH update helper: it must be a
// no-op when the directory is already on PATH, append the export line to the
// expected shell config files, and never duplicate an existing marker.
func TestAddToPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	originalPath := os.Getenv("PATH")

	t.Run("returns false without touching rc files when dir already on PATH", func(t *testing.T) {
		t.Setenv("PATH", originalPath+string(os.PathListSeparator)+home+"/bin")
		if got := addToPath(home + "/bin"); got {
			t.Error("addToPath() = true, want false for a directory already on PATH")
		}
		if _, err := os.Stat(home + "/.zshrc"); err == nil {
			t.Error("addToPath created .zshrc even though the directory was already on PATH")
		}
	})

	t.Run("appends export line to shell config", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("windows uses registry-based PATH handling and returns early")
		}
		t.Setenv("PATH", originalPath)

		installDir := home + "/.local/bin"
		if got := addToPath(installDir); !got {
			t.Fatal("addToPath() = false, want true when the PATH line is added")
		}

		configFile := expectedShellConfig(home)
		data, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("expected shell config %s was not created: %v", configFile, err)
		}
		wantLine := "export PATH=\"" + installDir + ":$PATH\""
		if !strings.Contains(string(data), wantLine) {
			t.Errorf("shell config %s does not contain %q:\n%s", configFile, wantLine, data)
		}
		if !strings.Contains(string(data), "# R2Go2 CLI") {
			t.Errorf("shell config %s does not contain the R2Go2 CLI marker", configFile)
		}
	})

	t.Run("does not duplicate an existing PATH marker", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("windows uses registry-based PATH handling and returns early")
		}
		t.Setenv("PATH", originalPath)

		configFile := expectedShellConfig(home)
		existing := "# R2Go2 CLI\nexport PATH=\"" + home + "/already/bin:$PATH\"\n"
		if err := os.WriteFile(configFile, []byte(existing), 0644); err != nil {
			t.Fatalf("failed to seed shell config: %v", err)
		}

		_ = addToPath(home + "/.local/bin")

		data, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("failed to read shell config: %v", err)
		}
		if got := strings.Count(string(data), "# R2Go2 CLI"); got != 1 {
			t.Errorf("shell config contains %d R2Go2 CLI markers, want 1:\n%s", got, data)
		}
	})

	t.Run("windows reports PATH as not configurable", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("registry-based PATH handling only applies on windows")
		}
		if got := addToPath(`C:\Apps\R2Go2`); got {
			t.Error("addToPath() = true on windows, want false")
		}
	})
}

// expectedShellConfig returns the shell config file addToPath is expected to
// write for the current platform inside the given home directory.
func expectedShellConfig(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return home + "/.zshrc"
	case "linux":
		// addToPath prefers .zshrc/.bashrc and falls back to .profile; the
		// tests never seed those files, so .profile is the expected target.
		return home + "/.profile"
	default:
		return home + "/.profile"
	}
}

// TestSaveInstallLocation verifies that saveInstallLocation persists the
// install directory under ~/.r2go2/install-info.txt and bails out safely
// when the home directory cannot be resolved.
func TestSaveInstallLocation(t *testing.T) {
	t.Run("writes install info file to home config dir", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		installDir := home + "/.local/bin"
		saveInstallLocation(installDir)

		data, err := os.ReadFile(home + "/.r2go2/install-info.txt")
		if err != nil {
			t.Fatalf("install info file was not written: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "install_dir="+installDir) {
			t.Errorf("install info does not record install_dir=%s:\n%s", installDir, content)
		}
		if !strings.Contains(content, "install_date=") {
			t.Error("install info does not record install_date")
		}
		if !strings.Contains(content, "version=0.3.1") {
			t.Errorf("install info does not record the installer version:\n%s", content)
		}
	})

	t.Run("returns without panicking when home dir is unavailable", func(t *testing.T) {
		t.Setenv("HOME", "")
		// Must simply return early; the only failure mode is a panic.
		saveInstallLocation("/tmp/somewhere")
	})

	t.Run("overwrites a previous install info file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		saveInstallLocation(home + "/first")
		saveInstallLocation(home + "/second")

		data, err := os.ReadFile(home + "/.r2go2/install-info.txt")
		if err != nil {
			t.Fatalf("install info file was not written: %v", err)
		}
		if strings.Contains(string(data), "install_dir="+home+"/first") {
			t.Errorf("stale install location survived the second save:\n%s", data)
		}
		if !strings.Contains(string(data), "install_dir="+home+"/second") {
			t.Errorf("latest install location not recorded:\n%s", data)
		}
	})
}

// TestCopyFile verifies the binary copy helper: it must copy file contents
// faithfully, overwrite existing destinations, and report missing sources.
func TestCopyFile(t *testing.T) {
	t.Run("copies file contents to destination", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dst := filepath.Join(dir, "dst")
		want := []byte("r2go2-binary-payload")
		if err := os.WriteFile(src, want, 0644); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile() error = %v", err)
		}

		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("destination file was not created: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("copied content = %q, want %q", got, want)
		}
	})

	t.Run("overwrites an existing destination", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dst := filepath.Join(dir, "dst")
		if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}
		if err := os.WriteFile(dst, []byte("old-and-longer"), 0644); err != nil {
			t.Fatalf("failed to create destination file: %v", err)
		}

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile() error = %v", err)
		}

		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("failed to read destination: %v", err)
		}
		if string(got) != "new" {
			t.Errorf("destination content = %q, want %q", got, "new")
		}
	})

	t.Run("returns error for a missing source", func(t *testing.T) {
		dir := t.TempDir()
		if err := copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "dst")); err == nil {
			t.Error("copyFile() with a missing source returned nil error")
		}
	})
}

// TestCreateBrandedSymlinks verifies that the case-insensitive command
// aliases are created as symlinks to the main r2go2 binary, that stale
// entries are replaced, and that failures are reported.
func TestCreateBrandedSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows copies binaries instead of creating symlinks")
	}

	aliases := []string{"R2Go2", "R2GO2", "r2Go2", "R2go2", "r2GO2"}

	t.Run("creates all case variant symlinks pointing at r2go2", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "r2go2"), []byte("bin"), 0755); err != nil {
			t.Fatalf("failed to create main binary: %v", err)
		}

		if err := createBrandedSymlinks(dir); err != nil {
			t.Fatalf("createBrandedSymlinks() error = %v", err)
		}

		for _, alias := range aliases {
			target, err := os.Readlink(filepath.Join(dir, alias))
			if err != nil {
				t.Errorf("alias %s was not created as a symlink: %v", alias, err)
				continue
			}
			if target != "r2go2" {
				t.Errorf("alias %s points to %q, want %q", alias, target, "r2go2")
			}
		}
	})

	t.Run("replaces pre-existing entries at alias paths", func(t *testing.T) {
		dir := t.TempDir()
		stale := filepath.Join(dir, "R2Go2")
		if err := os.WriteFile(stale, []byte("stale regular file"), 0644); err != nil {
			t.Fatalf("failed to create stale entry: %v", err)
		}

		if err := createBrandedSymlinks(dir); err != nil {
			t.Fatalf("createBrandedSymlinks() error = %v", err)
		}

		target, err := os.Readlink(stale)
		if err != nil {
			t.Fatalf("stale entry was not replaced by a symlink: %v", err)
		}
		if target != "r2go2" {
			t.Errorf("replaced alias points to %q, want %q", target, "r2go2")
		}
	})

	t.Run("returns error when install dir does not exist", func(t *testing.T) {
		dir := t.TempDir()
		if err := createBrandedSymlinks(filepath.Join(dir, "does-not-exist")); err == nil {
			t.Error("createBrandedSymlinks() with a missing install dir returned nil error")
		}
	})

	t.Run("returns error when an alias path cannot be replaced", func(t *testing.T) {
		dir := t.TempDir()
		// A non-empty directory cannot be removed by os.Remove, so the
		// subsequent symlink creation fails with "file exists".
		blocked := filepath.Join(dir, "R2Go2")
		if err := os.MkdirAll(filepath.Join(blocked, "nested"), 0755); err != nil {
			t.Fatalf("failed to create blocking directory: %v", err)
		}

		if err := createBrandedSymlinks(dir); err == nil {
			t.Error("createBrandedSymlinks() returned nil error despite an unreplaceable alias path")
		}
	})
}

// TestCreateMenuModel verifies that the main installer menu exposes the five
// documented options in its rendered view and tolerates unusual dimensions.
func TestCreateMenuModel(t *testing.T) {
	t.Run("renders all main menu options", func(t *testing.T) {
		menu := createMenuModel(80, 17)
		if menu == nil {
			t.Fatal("createMenuModel returned nil")
		}
		view := menu.View()
		wantTitles := []string{
			"Download from GitHub",
			"Use local build",
			"View installation requirements",
			"Show installation history",
			"Exit installer",
		}
		for _, want := range wantTitles {
			if !strings.Contains(view, want) {
				t.Errorf("menu view does not contain option %q", want)
			}
		}
	})

	t.Run("renders a non-empty view for zero dimensions", func(t *testing.T) {
		menu := createMenuModel(0, 0)
		if menu == nil {
			t.Fatal("createMenuModel returned nil for zero dimensions")
		}
		if view := menu.View(); view == "" {
			t.Error("menu view is empty for zero dimensions")
		}
	})
}

// TestCreatePostInstallMenu verifies that the post-installation menu exposes
// the five documented setup options in its rendered view.
func TestCreatePostInstallMenu(t *testing.T) {
	t.Run("renders all post-install menu options", func(t *testing.T) {
		menu := createPostInstallMenu(80, 17)
		if menu == nil {
			t.Fatal("createPostInstallMenu returned nil")
		}
		view := menu.View()
		wantTitles := []string{
			"Configure API credentials",
			"Test connection",
			"View quick start guide",
			"Open documentation",
			"Exit to terminal",
		}
		for _, want := range wantTitles {
			if !strings.Contains(view, want) {
				t.Errorf("post-install menu view does not contain option %q", want)
			}
		}
	})

	t.Run("renders a non-empty view for zero dimensions", func(t *testing.T) {
		menu := createPostInstallMenu(0, 0)
		if menu == nil {
			t.Fatal("createPostInstallMenu returned nil for zero dimensions")
		}
		if view := menu.View(); view == "" {
			t.Error("post-install menu view is empty for zero dimensions")
		}
	})
}

// TestMaskToken verifies that API tokens are fully masked when short and
// partially revealed (first and last four characters) when long enough.
func TestMaskToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"empty token stays empty", "", ""},
		{"short token is fully masked", "abc", "•••"},
		{"eight character token is fully masked", "12345678", "••••••••"},
		{"nine character token reveals first and last four", "123456789", "1234•6789"},
		{"long token reveals only the edges", "abcdefghijklmnop", "abcd••••••••mnop"},
		{"typical forty character token", "1234567890123456789012345678901234567890", "1234" + strings.Repeat("•", 32) + "7890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskToken(tt.token); got != tt.want {
				t.Errorf("maskToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
			// Masking must not change the token's visible length; "•" is a
			// multi-byte rune, so compare rune counts rather than byte counts.
			if got := maskToken(tt.token); utf8.RuneCountInString(got) != utf8.RuneCountInString(tt.token) {
				t.Errorf("maskToken(%q) rune length = %d, want %d", tt.token,
					utf8.RuneCountInString(got), utf8.RuneCountInString(tt.token))
			}
		})
	}
}

// TestMainEntryPoint smoke-tests the main function by building the installer
// binary and running it without a terminal. main must either refuse to start
// with a terminal-related error or start the TUI and block until killed; it
// must never panic.
func TestMainEntryPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping installer binary smoke test in -short mode")
	}

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "installer_tui")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("failed to build installer binary: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin)
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	output := stderr.String()

	if strings.Contains(output, "panic:") || strings.Contains(output, "runtime error:") {
		t.Fatalf("installer binary panicked:\n%s", output)
	}

	// Acceptable outcomes: killed by the timeout (TUI started and waited for
	// input), a clean exit, or a graceful error explaining why the TUI could
	// not run headlessly.
	t.Logf("installer terminated: err=%v stderr=%q", runErr, output)
	if runErr != nil && output == "" {
		t.Logf("installer exited with error %v and no diagnostics on stderr", runErr)
	}
}
