package helpers

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig holds configuration for test execution
type TestConfig struct {
	TempDir   string
	ConfigDir string
	TestData  string
}

// SetupTest creates a temporary test environment
func SetupTest(t *testing.T) *TestConfig {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "r2go2_test_*")
	require.NoError(t, err)

	configDir := filepath.Join(tempDir, ".r2go2")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	testData := filepath.Join(tempDir, "testdata")
	err = os.MkdirAll(testData, 0755)
	require.NoError(t, err)

	// Change to temp directory for test isolation
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Restore original directory after test
	t.Cleanup(func() {
		os.Chdir(originalWd)
		os.RemoveAll(tempDir)
	})

	return &TestConfig{
		TempDir:   tempDir,
		ConfigDir: configDir,
		TestData:  testData,
	}
}

// CreateTestFile creates a test file with specified content
func CreateTestFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// CaptureOutput captures stdout/stderr during function execution
func CaptureOutput(fn func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rStdout, wStdout, _ := os.Pipe()
	rStderr, wStderr, _ := os.Pipe()

	os.Stdout = wStdout
	os.Stderr = wStderr

	done := make(chan struct{})

	var stdoutBuf, stderrBuf bytes.Buffer

	go func() {
		io.Copy(&stdoutBuf, rStdout)
		io.Copy(&stderrBuf, rStderr)
		close(done)
	}()

	fn()

	wStdout.Close()
	wStderr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	<-done

	return stdoutBuf.String(), stderrBuf.String()
}

// AssertContains asserts that a string contains a substring
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	assert.Contains(t, haystack, needle)
}

// AssertNotContains asserts that a string does not contain a substring
func AssertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	assert.NotContains(t, haystack, needle)
}

// WaitFor waits for a condition to become true with timeout
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	start := time.Now()
	for time.Since(start) < timeout {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Fail(t, "Condition not met within timeout: %s", message)
}

// MockConfig creates a test configuration with sensible defaults
func MockConfig(configDir string) map[string]interface{} {
	return map[string]interface{}{
		"version":         "test-version",
		"current_profile": "test-profile",
		"profiles": map[string]interface{}{
			"test-profile": map[string]interface{}{
				"name":        "Test Profile",
				"account_id":  "test-account-id",
				"access_key":  "test-access-key",
				"secret_key":  "test-secret-key",
				"bucket":      "test-bucket",
				"endpoint":    "https://test.r2.cloudflarestorage.com",
				"region":      "auto",
				"created_at":  time.Now().Format(time.RFC3339),
			},
		},
		"settings": map[string]interface{}{
			"theme":           "default",
			"tutorial_completed": true,
			"accessibility": map[string]interface{}{
				"screen_reader": false,
				"high_contrast": false,
				"large_text":    false,
			},
		},
	}
}

// WithTimeout executes a function with a timeout
func WithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
		// Function completed in time
	case <-time.After(timeout):
		require.Fail(t, "Function timed out after %v", timeout)
	}
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CountFiles counts files in a directory recursively
func CountFiles(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// RandomString generates a random string of specified length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}