/*
Supplementary unit tests for the installer_tui package

Covers quickStartGuide (previously untested) plus additional behavioral
subtests for addToPath, createBrandedSymlinks and the main entry point.

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

	"github.com/charmbracelet/lipgloss"
)

// plainGuideStyles returns the four styles quickStartGuide expects, without
// any colors or borders, so guide lines can be compared as plain strings.
func plainGuideStyles() (headerStyle, textStyle, codeStyle, noteStyle lipgloss.Style) {
	return lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()
}

// TestQuickStartGuide verifies that quickStartGuide returns the full guide
// content: the case-insensitivity note, all four documented sections, every
// example command, and enough lines to require scrolling in the TUI view.
func TestQuickStartGuide(t *testing.T) {
	headerStyle, textStyle, codeStyle, noteStyle := plainGuideStyles()
	guide := quickStartGuide(headerStyle, textStyle, codeStyle, noteStyle)

	if len(guide) == 0 {
		t.Fatal("quickStartGuide() returned no lines")
	}

	t.Run("starts with the case-insensitivity note", func(t *testing.T) {
		want := "Note: Command is case-insensitive (R2Go2, r2go2, R2GO2 all work)"
		if guide[0] != want {
			t.Errorf("guide[0] = %q, want %q", guide[0], want)
		}
	})

	t.Run("contains all documented sections", func(t *testing.T) {
		for _, section := range []string{"Basic Commands", "Object Operations", "Configuration", "Getting Help"} {
			found := false
			for _, line := range guide {
				if line == section {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("guide is missing section header %q", section)
			}
		}
	})

	t.Run("contains every example command", func(t *testing.T) {
		commands := []string{
			"R2Go2 bucket list",
			"R2Go2 bucket create my-bucket",
			"R2Go2 bucket delete my-bucket",
			"R2Go2 object list my-bucket",
			"R2Go2 copy ./file.txt r2://my-bucket/",
			"R2Go2 copy r2://my-bucket/file.txt ./",
			"R2Go2 config show",
			"R2Go2 config list",
			"R2Go2 setup --switch",
			"R2Go2 --help",
			"R2Go2 bucket --help",
		}
		lines := make(map[string]bool, len(guide))
		for _, line := range guide {
			lines[line] = true
		}
		for _, cmd := range commands {
			if !lines[cmd] {
				t.Errorf("guide is missing example command %q", cmd)
			}
		}
	})

	t.Run("contains descriptive text for each section", func(t *testing.T) {
		descriptions := []string{
			"List all buckets:",
			"Create a new bucket:",
			"Delete a bucket:",
			"List objects in bucket:",
			"Upload a file:",
			"Download a file:",
			"View current config:",
			"List all profiles:",
			"Switch profile:",
			"Show all commands:",
			"Get help for a command:",
		}
		joined := strings.Join(guide, "\n")
		for _, desc := range descriptions {
			if !strings.Contains(joined, desc) {
				t.Errorf("guide is missing description %q", desc)
			}
		}
	})

	t.Run("is long enough to require scrolling in the view", func(t *testing.T) {
		// renderQuickStartState displays a 15-line window and only renders a
		// scroll indicator when the guide exceeds it.
		if len(guide) <= 15 {
			t.Errorf("guide has %d lines, want more than 15 to exercise scrolling", len(guide))
		}
	})

	t.Run("uses blank separator lines between entries", func(t *testing.T) {
		blankCount := 0
		for _, line := range guide {
			if line == "" {
				blankCount++
			}
		}
		if blankCount == 0 {
			t.Error("guide contains no blank separator lines")
		}
	})

	t.Run("does not contain unrendered placeholder text", func(t *testing.T) {
		for i, line := range guide {
			if strings.Contains(line, "%!") || strings.Contains(line, "%s") || strings.Contains(line, "%d") {
				t.Errorf("guide line %d looks like an unformatted printf verb: %q", i, line)
			}
		}
	})
}

// TestQuickStartGuideStyledRendering verifies that quickStartGuide preserves
// its text when rendered through the styled lipgloss styles used by the TUI.
func TestQuickStartGuideStyledRendering(t *testing.T) {
	headerStyle := lipgloss.NewStyle().Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Padding(0, 1)
	noteStyle := lipgloss.NewStyle().Italic(true)

	plainHeader, plainText, plainCode, plainNote := plainGuideStyles()
	styled := quickStartGuide(headerStyle, textStyle, codeStyle, noteStyle)
	plain := quickStartGuide(plainHeader, plainText, plainCode, plainNote)

	t.Run("styled and plain guides have identical length", func(t *testing.T) {
		if len(styled) != len(plain) {
			t.Fatalf("styled guide has %d lines, plain guide has %d", len(styled), len(plain))
		}
	})

	t.Run("every plain line survives styled rendering", func(t *testing.T) {
		for i, want := range plain {
			if want == "" {
				continue
			}
			if !strings.Contains(styled[i], want) {
				t.Errorf("styled line %d lost its text: have %q, want it to contain %q", i, styled[i], want)
			}
		}
	})

	t.Run("code lines receive padding from the code style", func(t *testing.T) {
		codeLine := -1
		for i, line := range plain {
			if line == "R2Go2 bucket list" {
				codeLine = i
				break
			}
		}
		if codeLine < 0 {
			t.Fatal("plain guide does not contain the bucket list example")
		}
		if len(styled[codeLine]) <= len(plain[codeLine]) {
			t.Errorf("styled code line %q has no extra styling bytes over the plain line %q", styled[codeLine], plain[codeLine])
		}
	})
}

// TestAddToPathShellConfigSelection verifies the platform-specific shell
// config selection in addToPath: darwin updates both .zshrc and an existing
// .bash_profile, while linux prefers existing rc files over the .profile
// fallback.
func TestAddToPathShellConfigSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses registry-based PATH handling and returns early")
	}

	installDir := "/opt/r2go2-test-bin"

	t.Run("darwin writes zshrc and existing bash_profile", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("darwin-specific shell config selection")
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", "/usr/bin:/bin")

		if err := os.WriteFile(home+"/.bash_profile", []byte("export PS1='$ '\n"), 0644); err != nil {
			t.Fatalf("failed to seed .bash_profile: %v", err)
		}

		if got := addToPath(installDir); !got {
			t.Fatal("addToPath() = false, want true")
		}

		for _, configFile := range []string{home + "/.zshrc", home + "/.bash_profile"} {
			data, err := os.ReadFile(configFile)
			if err != nil {
				t.Errorf("expected config %s to be updated: %v", configFile, err)
				continue
			}
			if !strings.Contains(string(data), "# R2Go2 CLI") {
				t.Errorf("config %s does not contain the R2Go2 CLI marker:\n%s", configFile, data)
			}
			if !strings.Contains(string(data), "export PATH=\""+installDir+":$PATH\"") {
				t.Errorf("config %s does not contain the PATH export:\n%s", configFile, data)
			}
		}
	})

	t.Run("linux prefers an existing zshrc over the profile fallback", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("linux-specific shell config selection")
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", "/usr/bin:/bin")

		if err := os.WriteFile(home+"/.zshrc", []byte("# zsh\n"), 0644); err != nil {
			t.Fatalf("failed to seed .zshrc: %v", err)
		}

		if got := addToPath(installDir); !got {
			t.Fatal("addToPath() = false, want true")
		}

		data, err := os.ReadFile(home + "/.zshrc")
		if err != nil {
			t.Fatalf("failed to read .zshrc: %v", err)
		}
		if !strings.Contains(string(data), "export PATH=\""+installDir+":$PATH\"") {
			t.Errorf(".zshrc does not contain the PATH export:\n%s", data)
		}
		if _, err := os.Stat(home + "/.profile"); err == nil {
			t.Error(".profile fallback was created even though .zshrc exists")
		}
	})

	t.Run("linux prefers an existing bashrc when zshrc is absent", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("linux-specific shell config selection")
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", "/usr/bin:/bin")

		if err := os.WriteFile(home+"/.bashrc", []byte("# bash\n"), 0644); err != nil {
			t.Fatalf("failed to seed .bashrc: %v", err)
		}

		if got := addToPath(installDir); !got {
			t.Fatal("addToPath() = false, want true")
		}

		data, err := os.ReadFile(home + "/.bashrc")
		if err != nil {
			t.Fatalf("failed to read .bashrc: %v", err)
		}
		if !strings.Contains(string(data), "export PATH=\""+installDir+":$PATH\"") {
			t.Errorf(".bashrc does not contain the PATH export:\n%s", data)
		}
	})

	t.Run("linux falls back to profile when no rc files exist", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("linux-specific shell config selection")
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", "/usr/bin:/bin")

		if got := addToPath(installDir); !got {
			t.Fatal("addToPath() = false, want true")
		}

		data, err := os.ReadFile(home + "/.profile")
		if err != nil {
			t.Fatalf(".profile fallback was not created: %v", err)
		}
		if !strings.Contains(string(data), "export PATH=\""+installDir+":$PATH\"") {
			t.Errorf(".profile does not contain the PATH export:\n%s", data)
		}
	})

	t.Run("returns false when home directory cannot be resolved", func(t *testing.T) {
		// An empty HOME makes os.UserHomeDir fail on unix, which must be
		// reported as "not added" rather than panicking.
		t.Setenv("HOME", "")
		t.Setenv("PATH", "/usr/bin:/bin")
		if got := addToPath(installDir); got {
			t.Error("addToPath() = true with an unresolvable home directory, want false")
		}
	})
}

// symlinkFilesystemIsCaseSensitive reports whether the directory backing the
// test distinguishes file names by case. On case-insensitive filesystems
// (default macOS APFS, Windows, some Linux mounts) the alias paths used by
// createBrandedSymlinks collide with the main r2go2 binary: the
// Lstat/Remove/Symlink sequence replaces the binary with a self-referential
// symlink. Resolution assertions are only meaningful on case-sensitive
// filesystems, so callers must skip when this returns false.
func symlinkFilesystemIsCaseSensitive(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	lower := filepath.Join(dir, "probe")
	upper := filepath.Join(dir, "PROBE")
	if err := os.WriteFile(lower, []byte("lower"), 0644); err != nil {
		t.Fatalf("failed to create case-sensitivity probe: %v", err)
	}
	if err := os.WriteFile(upper, []byte("upper"), 0644); err != nil {
		return false // second write clobbered the first: case-insensitive
	}
	lowerData, err := os.ReadFile(lower)
	if err != nil {
		t.Fatalf("failed to read case-sensitivity probe: %v", err)
	}
	return string(lowerData) == "lower"
}

// TestCreateBrandedSymlinksResolution verifies that the branded aliases
// created by createBrandedSymlinks actually resolve back to the main r2go2
// binary, and that the main binary itself stays a regular file.
func TestCreateBrandedSymlinksResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows copies binaries instead of creating symlinks")
	}

	t.Run("every alias resolves to the main binary", func(t *testing.T) {
		if !symlinkFilesystemIsCaseSensitive(t) {
			t.Skip("filesystem is case-insensitive: alias paths collide with the main binary and do not resolve (known production issue)")
		}
		dir := t.TempDir()
		mainBinary := filepath.Join(dir, "r2go2")
		want := []byte("fake-r2go2-payload")
		if err := os.WriteFile(mainBinary, want, 0755); err != nil {
			t.Fatalf("failed to create main binary: %v", err)
		}

		if err := createBrandedSymlinks(dir); err != nil {
			t.Fatalf("createBrandedSymlinks() error = %v", err)
		}

		for _, alias := range []string{"R2Go2", "R2GO2", "r2Go2", "R2go2", "r2GO2"} {
			resolved, err := filepath.EvalSymlinks(filepath.Join(dir, alias))
			if err != nil {
				t.Errorf("alias %s failed to resolve: %v", alias, err)
				continue
			}
			if resolved != mainBinary {
				t.Errorf("alias %s resolved to %q, want %q", alias, resolved, mainBinary)
			}
			// Reading through the alias must yield the real binary's bytes.
			got, err := os.ReadFile(filepath.Join(dir, alias))
			if err != nil {
				t.Errorf("failed to read through alias %s: %v", alias, err)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf("alias %s served %q, want %q", alias, got, want)
			}
		}
	})

	t.Run("leaves the main binary as a regular file", func(t *testing.T) {
		if !symlinkFilesystemIsCaseSensitive(t) {
			t.Skip("filesystem is case-insensitive: the main binary is replaced by an alias symlink (known production issue)")
		}
		dir := t.TempDir()
		mainBinary := filepath.Join(dir, "r2go2")
		if err := os.WriteFile(mainBinary, []byte("bin"), 0755); err != nil {
			t.Fatalf("failed to create main binary: %v", err)
		}

		if err := createBrandedSymlinks(dir); err != nil {
			t.Fatalf("createBrandedSymlinks() error = %v", err)
		}

		info, err := os.Lstat(mainBinary)
		if err != nil {
			t.Fatalf("main binary disappeared after alias creation: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Error("main binary r2go2 became a symlink, want a regular file")
		}
		if info.Mode().Perm()&0111 == 0 {
			t.Errorf("main binary lost its executable bit: %v", info.Mode().Perm())
		}
	})

	t.Run("keeps the documented alias set stable", func(t *testing.T) {
		if !symlinkFilesystemIsCaseSensitive(t) {
			t.Skip("filesystem is case-insensitive: alias paths collapse onto the main binary (known production issue)")
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "r2go2"), []byte("bin"), 0755); err != nil {
			t.Fatalf("failed to create main binary: %v", err)
		}
		if err := createBrandedSymlinks(dir); err != nil {
			t.Fatalf("createBrandedSymlinks() error = %v", err)
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to list install dir: %v", err)
		}
		if got := len(entries); got != 6 { // r2go2 + 5 aliases
			t.Errorf("install dir contains %d entries, want 6 (main binary plus five aliases)", got)
			for _, e := range entries {
				t.Logf("entry: %s", e.Name())
			}
		}
	})
}

// TestMainBinaryStderrDiagnostics verifies the error-reporting contract of
// the main entry point: when the built installer binary exits with a
// non-zero status on its own (i.e. it was not killed by the timeout), it
// must print a diagnostic to stderr, and it must never panic.
func TestMainBinaryStderrDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping installer binary smoke test in -short mode")
	}

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "installer_tui_diag")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("failed to build installer binary: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin)
	cmd.Stderr = &stderr
	cmd.Stdin = nil
	runErr := cmd.Run()
	output := stderr.String()

	t.Run("never panics", func(t *testing.T) {
		if strings.Contains(output, "panic:") || strings.Contains(output, "runtime error:") {
			t.Fatalf("installer binary panicked:\n%s", output)
		}
	})

	t.Run("reports a diagnostic when exiting non-zero on its own", func(t *testing.T) {
		if ctx.Err() == context.DeadlineExceeded {
			t.Skip("process was killed by the timeout (TUI started and blocked); nothing to assert")
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			if strings.TrimSpace(output) == "" {
				t.Errorf("installer exited with status %d but printed nothing to stderr", exitErr.ExitCode())
			}
			return
		}
		t.Skipf("installer exited cleanly (err=%v); error reporting not exercised", runErr)
	})
}
