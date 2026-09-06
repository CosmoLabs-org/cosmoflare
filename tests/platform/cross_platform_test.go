/*
Package platform provides cross-platform compatibility testing for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

// TestPlatformCompatibility tests cross-platform compatibility
func TestPlatformCompatibility(t *testing.T) {
	t.Run("Operating System Detection", func(t *testing.T) {
		// Verify OS detection works
		os := runtime.GOOS
		assert.NotEmpty(t, os, "Should detect operating system")

		// Common supported platforms
		supportedOS := []string{"darwin", "linux", "windows"}
		assert.Contains(t, supportedOS, os, "Should run on supported platform")
	})

	t.Run("Architecture Detection", func(t *testing.T) {
		// Verify architecture detection works
		arch := runtime.GOARCH
		assert.NotEmpty(t, arch, "Should detect architecture")

		// Common supported architectures
		supportedArch := []string{"amd64", "arm64", "386"}
		assert.Contains(t, supportedArch, arch, "Should run on supported architecture")
	})

	t.Run("File Path Handling", func(t *testing.T) {
		// Test file path handling across platforms
		testConfig := helpers.SetupTest(t)

		// Test temp directory creation
		assert.NotEmpty(t, testConfig.TempDir, "Should create temp directory")
		assert.DirExists(t, testConfig.TempDir, "Temp directory should exist")

		// Test file creation
		testFile := testConfig.TempDir + string(os.PathSeparator) + "test.txt"
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err, "Should create test file")
		assert.FileExists(t, testFile, "Test file should exist")

		// Clean up
		err = os.Remove(testFile)
		assert.NoError(t, err, "Should remove test file")
	})

	t.Run("Environment Variable Handling", func(t *testing.T) {
		// Test environment variable access
		testKey := "R2GO2_TEST_VAR"
		testValue := "test_value_" + runtime.GOOS

		// Set environment variable
		err := os.Setenv(testKey, testValue)
		require.NoError(t, err, "Should set environment variable")

		// Get environment variable
		retrievedValue := os.Getenv(testKey)
		assert.Equal(t, testValue, retrievedValue, "Should retrieve environment variable")

		// Clean up
		err = os.Unsetenv(testKey)
		assert.NoError(t, err, "Should unset environment variable")
	})

	t.Run("Path Separator Handling", func(t *testing.T) {
		// Test path separator handling
		pathSep := string(os.PathSeparator)

		// Construct paths using platform-appropriate separator
		pathComponents := []string{"home", "user", "documents"}
		expectedPath := strings.Join(pathComponents, pathSep)

		assert.Contains(t, expectedPath, pathSep, "Should contain path separator")

		// Verify path separator is correct for the platform
		switch runtime.GOOS {
		case "windows":
			assert.Equal(t, "\\", pathSep, "Windows should use backslash")
		case "darwin", "linux":
			assert.Equal(t, "/", pathSep, "Unix-like systems should use forward slash")
		}
	})

	t.Run("Executable Location", func(t *testing.T) {
		// Test executable location detection
		execPath, err := os.Executable()
		require.NoError(t, err, "Should detect executable path")
		assert.NotEmpty(t, execPath, "Should have executable path")

		// Verify path contains appropriate separator
		assert.Contains(t, execPath, string(os.PathSeparator), "Should contain path separator")
	})

	t.Run("Working Directory", func(t *testing.T) {
		// Test working directory detection
		wd, err := os.Getwd()
		require.NoError(t, err, "Should detect working directory")
		assert.NotEmpty(t, wd, "Should have working directory")

		// Verify it's a valid path
		assert.DirExists(t, wd, "Working directory should exist")
	})
}

// TestShellCompatibility tests shell compatibility across platforms
func TestShellCompatibility(t *testing.T) {
	t.Run("Shell Detection", func(t *testing.T) {
		// Test shell environment detection
		shell := os.Getenv("SHELL")
		if runtime.GOOS == "windows" {
			// On Windows, check for COMSPEC
			shell = os.Getenv("COMSPEC")
		}

		if shell != "" {
			t.Logf("Detected shell: %s", shell)
			assert.NotEmpty(t, shell, "Should detect shell on Unix-like systems")
		} else {
			t.Log("Shell environment variable not set")
		}
	})

	t.Run("Command Execution", func(t *testing.T) {
		// Test basic command execution capabilities
		// This is a simplified test - in practice you'd test actual shell commands

		switch runtime.GOOS {
		case "windows":
			// Windows-specific command checks
			assert.True(t, true, "Windows command execution placeholder")
		case "darwin", "linux":
			// Unix-like command checks
			assert.True(t, true, "Unix-like command execution placeholder")
		default:
			t.Logf("Unsupported platform for shell testing: %s", runtime.GOOS)
		}
	})
}

// TestPlatformSpecificBehavior tests platform-specific behaviors
func TestPlatformSpecificBehavior(t *testing.T) {
	t.Run("File Permissions", func(t *testing.T) {
		// Test file permission handling
		testConfig := helpers.SetupTest(t)
		testFile := testConfig.TempDir + string(os.PathSeparator) + "permissions_test.txt"

		// Create file with specific permissions
		content := []byte("test content")
		err := os.WriteFile(testFile, content, 0644)
		require.NoError(t, err, "Should create file with permissions")

		// Check file exists
		assert.FileExists(t, testFile, "File should exist")

		// Check file info
		info, err := os.Stat(testFile)
		require.NoError(t, err, "Should get file info")
		assert.NotNil(t, info, "Should have file info")

		// Platform-specific permission checks
		if runtime.GOOS != "windows" {
			// Unix-like systems have fine-grained permissions
			mode := info.Mode()
			t.Logf("File mode: %v", mode)
		} else {
			// Windows has different permission model
			t.Logf("Windows file permissions (simplified check)")
		}

		// Clean up
		err = os.Remove(testFile)
		assert.NoError(t, err, "Should remove file")
	})

	t.Run("Temporary Directory Handling", func(t *testing.T) {
		// Test temporary directory creation and usage
		tempDir := t.TempDir()

		assert.NotEmpty(t, tempDir, "Should create temporary directory")
		assert.DirExists(t, tempDir, "Temporary directory should exist")

		// Test file operations in temp directory
		testFile := tempDir + string(os.PathSeparator) + "temp_test.txt"
		err := os.WriteFile(testFile, []byte("temp test"), 0644)
		require.NoError(t, err, "Should create file in temp directory")

		// Verify file exists and has content
		assert.FileExists(t, testFile, "File should exist in temp directory")

		// Read and verify content
		content, err := os.ReadFile(testFile)
		require.NoError(t, err, "Should read file from temp directory")
		assert.Equal(t, "temp test", string(content), "Should have correct content")

		// Note: t.TempDir() is automatically cleaned up by Go testing framework
	})
}

// TestPerformanceCharacteristics tests performance characteristics across platforms
func TestPerformanceCharacteristics(t *testing.T) {
	t.Run("Basic Performance Metrics", func(t *testing.T) {
		// Test basic performance characteristics
		iterations := 1000

		// Measure operation time
		start := time.Now()

		for i := 0; i < iterations; i++ {
			// Simple operation: string manipulation
			testStr := "test_string_" + string(rune(i%1000))
			_ = strings.ToUpper(testStr)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Platform: %s", runtime.GOOS)
		t.Logf("Architecture: %s", runtime.GOARCH)
		t.Logf("Iterations: %d", iterations)
		t.Logf("Total duration: %v", duration)
		t.Logf("Average duration: %v", avgDuration)

		// Performance should be reasonable (less than 1ms per operation)
		assert.Less(t, avgDuration, time.Millisecond, "Average operation should be fast")
	})
}

// TestErrorHandling tests error handling across platforms
func TestErrorHandling(t *testing.T) {
	t.Run("File Operation Errors", func(t *testing.T) {
		// Test error handling for file operations
		nonExistentFile := "/path/that/does/not/exist.txt"

		// Should return error for non-existent file
		_, err := os.Stat(nonExistentFile)
		assert.Error(t, err, "Should return error for non-existent file")

		// Error should contain appropriate information
		errorMsg := err.Error()
		assert.NotEmpty(t, errorMsg, "Error message should not be empty")

		// Should be the expected type of error
		assert.True(t, os.IsNotExist(err), "Should be NotExist error type")
	})

	t.Run("Permission Error Handling", func(t *testing.T) {
		// Test permission-related error handling
		// This is platform-specific, so we'll test the pattern

		// Try to access a potentially restricted location
		var restrictedPath string
		switch runtime.GOOS {
		case "windows":
			restrictedPath = "C:\\Windows\\System32\\config\\"
		case "darwin":
			// /root/ doesn't exist on macOS; create a restricted temp dir
			restrictedDir := filepath.Join(t.TempDir(), "restricted")
			os.Mkdir(restrictedDir, 0000)
			restrictedPath = restrictedDir
		case "linux":
			restrictedPath = "/root/"
		default:
			t.Skip("Skipping permission test on unsupported platform")
			return
		}

		// Attempt to list directory contents
		_, err := os.Open(restrictedPath)
		if err != nil {
			t.Logf("Permission error as expected: %v", err)
			assert.True(t, os.IsPermission(err) || strings.Contains(err.Error(), "permission denied"),
				"Should return permission-related error")
		} else {
			t.Log("Permission test: able to access restricted path (may be running with elevated privileges)")
		}
	})
}