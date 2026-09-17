package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSetupTest validates that SetupTest creates the temp environment
// (TempDir, .r2go2 config dir, testdata dir), changes the working directory
// into it, and restores the original state via t.Cleanup once the test ends.
func TestSetupTest(t *testing.T) {
	origWd, err := os.Getwd()
	require.NoError(t, err)

	var cfg *TestConfig

	t.Run("creates environment and changes working directory", func(t *testing.T) {
		cfg = SetupTest(t)

		require.NotNil(t, cfg)
		require.DirExists(t, cfg.TempDir)
		require.DirExists(t, cfg.ConfigDir)
		require.DirExists(t, cfg.TestData)
		require.Equal(t, filepath.Join(cfg.TempDir, ".r2go2"), cfg.ConfigDir)
		require.Equal(t, filepath.Join(cfg.TempDir, "testdata"), cfg.TestData)

		// os.MkdirTemp may return a symlinked path (e.g. /var vs /private/var
		// on macOS) while os.Getwd returns the resolved path, so compare the
		// resolved form.
		resolvedTemp, err := filepath.EvalSymlinks(cfg.TempDir)
		require.NoError(t, err)

		wd, err := os.Getwd()
		require.NoError(t, err)
		require.Equal(t, resolvedTemp, wd, "working directory should be the temp dir")
	})

	// After the subtest completes its t.Cleanup must have run: the original
	// working directory is restored and the temp directory removed.
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, origWd, wd, "original working directory should be restored")

	require.False(t, FileExists(cfg.TempDir), "temp dir should be removed by cleanup")
}

// TestCreateTestFile validates that CreateTestFile writes the given content
// with 0644 permissions, creates new files, and overwrites existing ones.
func TestCreateTestFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("creates file with content", func(t *testing.T) {
		path := filepath.Join(dir, "hello.txt")
		CreateTestFile(t, path, "hello world")

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "hello world", string(data))

		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("empty content creates empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.txt")
		CreateTestFile(t, path, "")

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Empty(t, data)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(dir, "overwrite.txt")
		CreateTestFile(t, path, "old")
		CreateTestFile(t, path, "new")

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "new", string(data))
	})

	t.Run("creates nested path when directory exists", func(t *testing.T) {
		sub := filepath.Join(dir, "sub")
		require.NoError(t, os.Mkdir(sub, 0755))

		path := filepath.Join(sub, "nested.txt")
		CreateTestFile(t, path, "nested")
		require.True(t, FileExists(path))
	})
}

// TestCaptureOutput validates that stdout and stderr writes performed inside
// the captured function are returned as separate strings, and that a silent
// function yields empty captures.
func TestCaptureOutput(t *testing.T) {
	t.Run("captures stdout and stderr separately", func(t *testing.T) {
		stdout, stderr := CaptureOutput(func() {
			fmt.Println("to stdout")
			fmt.Fprintln(os.Stderr, "to stderr")
		})

		require.Contains(t, stdout, "to stdout")
		require.NotContains(t, stdout, "to stderr")
		require.Contains(t, stderr, "to stderr")
		require.NotContains(t, stderr, "to stdout")
	})

	t.Run("captures fmt print family on stdout", func(t *testing.T) {
		stdout, _ := CaptureOutput(func() {
			fmt.Printf("value=%d", 42)
		})
		require.Equal(t, "value=42", stdout)
	})

	t.Run("silent function yields empty output", func(t *testing.T) {
		stdout, stderr := CaptureOutput(func() {})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})

	t.Run("multiline writes preserve order", func(t *testing.T) {
		stdout, _ := CaptureOutput(func() {
			fmt.Println("first")
			fmt.Println("second")
			fmt.Println("third")
		})
		require.Equal(t, "first\nsecond\nthird\n", stdout)
	})

	t.Run("original streams are restored", func(t *testing.T) {
		before := os.Stdout
		CaptureOutput(func() {})
		require.Same(t, before, os.Stdout, "os.Stdout should be restored")
	})
}

// TestAssertContains validates the pass-through behavior of AssertContains:
// matching substrings keep the test green. The failure path intentionally
// fails the calling test and therefore cannot be asserted within this suite.
func TestAssertContains(t *testing.T) {
	t.Run("substring present", func(t *testing.T) {
		AssertContains(t, "hello cosmoflare", "cosmoflare")
	})

	t.Run("whole string equals needle", func(t *testing.T) {
		AssertContains(t, "exact", "exact")
	})

	t.Run("needle in middle of multiline haystack", func(t *testing.T) {
		AssertContains(t, "line1\nline2\nline3", "line2")
	})
}

// TestAssertNotContains validates that AssertNotContains keeps the test green
// when the needle is absent. The failure path intentionally fails the calling
// test and cannot be asserted within this suite.
func TestAssertNotContains(t *testing.T) {
	t.Run("substring absent", func(t *testing.T) {
		AssertNotContains(t, "hello cosmoflare", "goodbye")
	})

	t.Run("empty needle not asserted", func(t *testing.T) {
		// An empty needle is technically contained in everything; testify
		// would fail this case. Use a non-empty absent needle instead.
		AssertNotContains(t, "abc", "xyz")
	})
}

// TestWaitFor validates that WaitFor returns promptly when the condition is
// already true and polls until a condition becomes true within the timeout.
// The timeout failure path intentionally fails the calling test and cannot be
// asserted within this suite.
func TestWaitFor(t *testing.T) {
	t.Run("condition already true returns immediately", func(t *testing.T) {
		start := time.Now()
		WaitFor(t, func() bool { return true }, 5*time.Second, "should be immediately true")
		require.Less(t, time.Since(start), time.Second, "should not wait when condition is true")
	})

	t.Run("polls until condition becomes true", func(t *testing.T) {
		readyAt := time.Now().Add(80 * time.Millisecond)
		WaitFor(t, func() bool { return time.Now().After(readyAt) }, 5*time.Second, "condition should become true")
	})

	t.Run("condition is evaluated at least once", func(t *testing.T) {
		var calls int32
		WaitFor(t, func() bool {
			atomic.AddInt32(&calls, 1)
			return true
		}, 5*time.Second, "condition should be evaluated")
		require.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(1))
	})

	t.Run("condition flipping from false to true succeeds", func(t *testing.T) {
		var counter int32
		WaitFor(t, func() bool {
			return atomic.AddInt32(&counter, 1) >= 3
		}, 5*time.Second, "condition should flip true on third poll")
		require.GreaterOrEqual(t, atomic.LoadInt32(&counter), int32(3))
	})
}

// TestMockConfig validates the structure and default values of the generated
// configuration map and that each call returns a fresh, independently mutable
// instance.
func TestMockConfig(t *testing.T) {
	cfg := MockConfig("/tmp/ignored-config-dir")

	t.Run("top level defaults", func(t *testing.T) {
		require.Equal(t, "test-version", cfg["version"])
		require.Equal(t, "test-profile", cfg["current_profile"])
		require.Contains(t, cfg, "profiles")
		require.Contains(t, cfg, "settings")
	})

	t.Run("profile defaults", func(t *testing.T) {
		profiles, ok := cfg["profiles"].(map[string]interface{})
		require.True(t, ok)

		profile, ok := profiles["test-profile"].(map[string]interface{})
		require.True(t, ok)

		require.Equal(t, "Test Profile", profile["name"])
		require.Equal(t, "test-account-id", profile["account_id"])
		require.Equal(t, "test-access-key", profile["access_key"])
		require.Equal(t, "test-secret-key", profile["secret_key"])
		require.Equal(t, "test-bucket", profile["bucket"])
		require.Equal(t, "https://test.r2.cloudflarestorage.com", profile["endpoint"])
		require.Equal(t, "auto", profile["region"])
		require.NotEmpty(t, profile["created_at"], "created_at should be populated")

		_, err := time.Parse(time.RFC3339, profile["created_at"].(string))
		require.NoError(t, err, "created_at should be RFC3339 formatted")
	})

	t.Run("settings defaults", func(t *testing.T) {
		settings, ok := cfg["settings"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "default", settings["theme"])
		require.Equal(t, true, settings["tutorial_completed"])

		accessibility, ok := settings["accessibility"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, false, accessibility["screen_reader"])
		require.Equal(t, false, accessibility["high_contrast"])
		require.Equal(t, false, accessibility["large_text"])
	})

	t.Run("each call returns an independent instance", func(t *testing.T) {
		cfg["version"] = "mutated"
		fresh := MockConfig("")
		require.Equal(t, "test-version", fresh["version"], "mutation should not leak across calls")
	})
}

// TestWithTimeout validates that WithTimeout runs the supplied function and
// returns when it completes inside the timeout budget. The timeout failure
// path intentionally fails the calling test and cannot be asserted within
// this suite.
func TestWithTimeout(t *testing.T) {
	t.Run("function runs to completion", func(t *testing.T) {
		var ran int32
		WithTimeout(t, 5*time.Second, func() {
			atomic.StoreInt32(&ran, 1)
		})
		require.Equal(t, int32(1), atomic.LoadInt32(&ran), "function should have executed")
	})

	t.Run("function longer than instant still passes under generous timeout", func(t *testing.T) {
		var done int32
		WithTimeout(t, 5*time.Second, func() {
			time.Sleep(50 * time.Millisecond)
			atomic.StoreInt32(&done, 1)
		})
		require.Equal(t, int32(1), atomic.LoadInt32(&done))
	})

	t.Run("noop function completes", func(t *testing.T) {
		WithTimeout(t, time.Second, func() {})
	})
}

// TestFileExists validates FileExists for existing files, directories, and
// missing paths.
func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "present.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	t.Run("existing file", func(t *testing.T) {
		require.True(t, FileExists(file))
	})

	t.Run("existing directory also reports true", func(t *testing.T) {
		// FileExists only checks for not-not-exist, so directories count.
		require.True(t, FileExists(dir))
	})

	t.Run("missing path", func(t *testing.T) {
		require.False(t, FileExists(filepath.Join(dir, "no-such-file")))
	})

	t.Run("empty path", func(t *testing.T) {
		require.False(t, FileExists(""))
	})
}

// TestDirExists validates DirExists for existing directories, files (which
// must report false), and missing paths.
func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	t.Run("existing directory", func(t *testing.T) {
		require.True(t, DirExists(dir))
	})

	t.Run("existing file reports false", func(t *testing.T) {
		require.False(t, DirExists(file))
	})

	t.Run("missing path", func(t *testing.T) {
		require.False(t, DirExists(filepath.Join(dir, "no-such-dir")))
	})

	t.Run("nested existing directory", func(t *testing.T) {
		nested := filepath.Join(dir, "a", "b")
		require.NoError(t, os.MkdirAll(nested, 0755))
		require.True(t, DirExists(nested))
	})
}

// TestCountFiles validates recursive file counting including nested
// directories, empty directories, and the error path for a missing root.
func TestCountFiles(t *testing.T) {
	dir := t.TempDir()

	t.Run("empty directory counts zero", func(t *testing.T) {
		count, err := CountFiles(dir)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("counts files recursively ignoring directories", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "a", "b"), 0755))
		for _, rel := range []string{"root.txt", "a/one.txt", "a/two.txt", "a/b/three.txt"} {
			path := filepath.Join(dir, rel)
			require.NoError(t, os.WriteFile(path, []byte("data"), 0644))
		}

		count, err := CountFiles(dir)
		require.NoError(t, err)
		require.Equal(t, 4, count)
	})

	t.Run("missing directory returns error", func(t *testing.T) {
		count, err := CountFiles(filepath.Join(dir, "does-not-exist"))
		require.Error(t, err)
		require.Equal(t, 0, count)
	})
}

// TestRandomString validates that RandomString returns a string of the
// requested length composed only of alphanumeric characters, including the
// zero-length and larger-than-charset edge cases.
func TestRandomString(t *testing.T) {
	t.Run("returns requested length", func(t *testing.T) {
		for _, length := range []int{0, 1, 10, 32, 62} {
			require.Len(t, RandomString(length), length, "length %d", length)
		}
	})

	t.Run("uses only alphanumeric characters", func(t *testing.T) {
		s := RandomString(200)
		for _, r := range s {
			require.True(t,
				(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'),
				"unexpected rune %q in %q", r, s)
		}
	})

	t.Run("zero length returns empty string", func(t *testing.T) {
		require.Equal(t, "", RandomString(0))
	})

	t.Run("longer than charset wraps without panicking", func(t *testing.T) {
		s := RandomString(130)
		require.Len(t, s, 130)
	})
}
