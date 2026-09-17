package fixtures_test

import (
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/tests/fixtures"
)

// TestGenerateTestFile verifies that GenerateTestFile returns a byte slice whose
// length matches the requested size, including the zero-size edge case, and that
// the generated content actually contains random data instead of zeroed bytes.
func TestGenerateTestFile(t *testing.T) {
	t.Run("size zero returns an empty non-nil slice", func(t *testing.T) {
		content := fixtures.GenerateTestFile(0)
		assert.NotNil(t, content, "a zero-size file should still be a valid (empty) slice")
		assert.Empty(t, content, "a zero-size file must not contain any bytes")
	})

	t.Run("size one returns a single byte", func(t *testing.T) {
		content := fixtures.GenerateTestFile(1)
		require.Len(t, content, 1, "requested size 1, got %d bytes", len(content))
	})

	t.Run("size 1KB returns exactly the requested number of bytes", func(t *testing.T) {
		const size = 1024
		content := fixtures.GenerateTestFile(size)
		require.Len(t, content, size)
	})

	t.Run("size 1MB returns exactly the requested number of bytes", func(t *testing.T) {
		const size = 1024 * 1024
		content := fixtures.GenerateTestFile(size)
		require.Len(t, content, size)
	})

	t.Run("generated content is not zeroed", func(t *testing.T) {
		// With 4096 uniformly random bytes the probability of every byte being
		// zero is negligible, so this guards against a degenerate generator
		// returning an uninitialised buffer.
		content := fixtures.GenerateTestFile(4096)
		require.Len(t, content, 4096)

		nonZero := 0
		for _, b := range content {
			if b != 0 {
				nonZero++
			}
		}
		assert.Greater(t, nonZero, 0, "expected at least one non-zero byte in the random payload")
	})

	t.Run("negative size fails loudly with a panic", func(t *testing.T) {
		// GenerateTestFile has no error return, so an invalid size must not be
		// silently accepted. Document that a negative size panics instead of
		// producing a bogus file.
		defer func() {
			r := recover()
			require.NotNil(t, r, "expected a panic for a negative size, but GenerateTestFile returned normally")
		}()
		_ = fixtures.GenerateTestFile(-1)
		t.Fatal("unreachable: GenerateTestFile should have panicked")
	})
}

// TestGenerateTestFiles verifies that GenerateTestFiles builds one named entry
// per requested size, uses the documented test-file-N.txt naming scheme, honours
// empty and nil inputs, and keeps duplicate sizes under distinct names.
func TestGenerateTestFiles(t *testing.T) {
	t.Run("nil sizes returns an empty non-nil map", func(t *testing.T) {
		files := fixtures.GenerateTestFiles(nil)
		assert.NotNil(t, files, "a nil size list should still yield a usable (empty) map")
		assert.Empty(t, files)
	})

	t.Run("empty sizes returns an empty non-nil map", func(t *testing.T) {
		files := fixtures.GenerateTestFiles([]int{})
		assert.NotNil(t, files)
		assert.Empty(t, files)
	})

	t.Run("single size produces one correctly named file", func(t *testing.T) {
		files := fixtures.GenerateTestFiles([]int{16})
		require.Len(t, files, 1)
		content, ok := files["test-file-1.txt"]
		require.True(t, ok, "expected entry %q to exist, got keys %v", "test-file-1.txt", mapKeys(files))
		assert.Len(t, content, 16)
	})

	t.Run("multiple sizes produce sequentially numbered files", func(t *testing.T) {
		files := fixtures.GenerateTestFiles([]int{1, 1024, 2048})
		require.Len(t, files, 3)

		expected := map[string]int{
			"test-file-1.txt": 1,
			"test-file-2.txt": 1024,
			"test-file-3.txt": 2048,
		}
		for name, size := range expected {
			content, ok := files[name]
			require.True(t, ok, "expected entry %q to exist, got keys %v", name, mapKeys(files))
			assert.Lenf(t, content, size, "file %q should contain %d bytes", name, size)
		}
	})

	t.Run("duplicate sizes still get distinct file names", func(t *testing.T) {
		files := fixtures.GenerateTestFiles([]int{32, 32, 32})
		require.Len(t, files, 3)
		for _, name := range []string{"test-file-1.txt", "test-file-2.txt", "test-file-3.txt"} {
			assert.Contains(t, files, name, "expected a file named %q", name)
		}
	})

	t.Run("zero sizes produce empty file contents", func(t *testing.T) {
		files := fixtures.GenerateTestFiles([]int{0, 0})
		require.Len(t, files, 2)
		for name, content := range files {
			assert.NotNil(t, content, "file %q should be an empty slice, not nil", name)
			assert.Empty(t, content, "file %q requested with size 0", name)
		}
	})
}

// TestGenerateTestDirectoryName verifies that GenerateTestDirectoryName returns a
// name using the documented r2go2-test-<unix seconds> format whose timestamp is
// anchored to the current time.
func TestGenerateTestDirectoryName(t *testing.T) {
	t.Run("matches the documented naming scheme", func(t *testing.T) {
		name := fixtures.GenerateTestDirectoryName()
		require.NotEmpty(t, name)

		pattern := regexp.MustCompile(`^r2go2-test-\d+$`)
		assert.Regexp(t, pattern, name, "directory name %q does not follow r2go2-test-<unix>", name)
		assert.False(t, strings.ContainsAny(name, "/\\: *?\"<>|"), "name must stay filesystem safe")
	})

	t.Run("embedded timestamp is close to now", func(t *testing.T) {
		before := time.Now().Add(-time.Minute).Unix()
		name := fixtures.GenerateTestDirectoryName()
		after := time.Now().Add(time.Minute).Unix()

		raw := strings.TrimPrefix(name, "r2go2-test-")
		ts, err := strconv.ParseInt(raw, 10, 64)
		require.NoError(t, err, "timestamp portion %q must be numeric", raw)
		assert.GreaterOrEqual(t, ts, before, "timestamp must not be in the past")
		assert.LessOrEqual(t, ts, after, "timestamp must not be in the future")
	})

	t.Run("successive calls stay well-formed", func(t *testing.T) {
		// Uniqueness is deliberately not asserted: the name is derived from
		// Unix seconds, so two calls within the same second legitimately share
		// a name. Both must still be valid.
		pattern := regexp.MustCompile(`^r2go2-test-\d+$`)
		for i := 0; i < 5; i++ {
			assert.Regexp(t, pattern, fixtures.GenerateTestDirectoryName())
		}
	})
}

// TestGetSampleConfig verifies that GetSampleConfig exposes a shallow copy of the
// package-level SampleConfig: every top-level section is present with the
// documented values, and mutating the returned map never leaks back into the
// shared fixture.
func TestGetSampleConfig(t *testing.T) {
	t.Run("returns all documented top-level keys", func(t *testing.T) {
		config := fixtures.GetSampleConfig()
		require.NotNil(t, config)
		for _, key := range []string{"version", "current_profile", "profiles", "settings"} {
			assert.Contains(t, config, key, "sample config should expose the %q section", key)
		}
	})

	t.Run("returns the documented scalar values", func(t *testing.T) {
		config := fixtures.GetSampleConfig()
		require.NotNil(t, config)
		assert.Equal(t, "1.0.0", config["version"])
		assert.Equal(t, "default", config["current_profile"])
	})

	t.Run("profiles section contains both sample profiles", func(t *testing.T) {
		config := fixtures.GetSampleConfig()
		require.NotNil(t, config)

		profiles, ok := config["profiles"].(map[string]interface{})
		require.True(t, ok, "profiles should be a nested map, got %T", config["profiles"])
		assert.Contains(t, profiles, "default")
		assert.Contains(t, profiles, "secondary")

		def, ok := profiles["default"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Default Profile", def["name"])
		assert.Equal(t, "my-test-bucket", def["bucket"])
	})

	t.Run("returns a copy, not the package-level map", func(t *testing.T) {
		config := fixtures.GetSampleConfig()
		require.NotNil(t, config)
		assert.NotEqual(t,
			reflect.ValueOf(fixtures.SampleConfig).Pointer(),
			reflect.ValueOf(config).Pointer(),
			"callers must receive a copy of the sample config, not the package-level map")
	})

	t.Run("mutating the copy leaves the shared fixture untouched", func(t *testing.T) {
		config := fixtures.GetSampleConfig()
		require.NotNil(t, config)

		config["version"] = "tampered"
		delete(config, "settings")

		shared := fixtures.GetSampleConfig()
		assert.Equal(t, "1.0.0", shared["version"], "mutating a copy must not alter the shared sample config")
		assert.Contains(t, shared, "settings", "deleting a key on a copy must not alter the shared sample config")
	})
}

// TestGetSampleThemes verifies that GetSampleThemes returns a shallow copy of the
// package-level SampleThemes containing the documented default and dark themes,
// and that copy mutations do not affect the shared fixture.
func TestGetSampleThemes(t *testing.T) {
	t.Run("returns the documented themes", func(t *testing.T) {
		themes := fixtures.GetSampleThemes()
		require.NotNil(t, themes)
		assert.Contains(t, themes, "default")
		assert.Contains(t, themes, "dark")
		assert.Len(t, themes, 2, "only the default and dark sample themes are documented")
	})

	t.Run("default theme carries name, description and color groups", func(t *testing.T) {
		themes := fixtures.GetSampleThemes()
		require.NotNil(t, themes)

		def, ok := themes["default"].(map[string]interface{})
		require.True(t, ok, "theme should be a nested map, got %T", themes["default"])
		assert.Equal(t, "Default", def["name"])
		assert.Equal(t, "Default R2Go2 theme", def["description"])

		for _, group := range []string{"colors", "background", "text"} {
			assert.Contains(t, def, group, "default theme should define a %q group", group)
		}

		colors, ok := def["colors"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "#007bff", colors["primary"])
		assert.Equal(t, "#dc3545", colors["danger"])
	})

	t.Run("dark theme uses a dark background", func(t *testing.T) {
		themes := fixtures.GetSampleThemes()
		require.NotNil(t, themes)

		dark, ok := themes["dark"].(map[string]interface{})
		require.True(t, ok)
		background, ok := dark["background"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "#212529", background["main"])
	})

	t.Run("returns a copy, not the package-level map", func(t *testing.T) {
		themes := fixtures.GetSampleThemes()
		require.NotNil(t, themes)
		assert.NotEqual(t,
			reflect.ValueOf(fixtures.SampleThemes).Pointer(),
			reflect.ValueOf(themes).Pointer(),
			"callers must receive a copy of the sample themes, not the package-level map")
	})

	t.Run("mutating the copy leaves the shared fixture untouched", func(t *testing.T) {
		themes := fixtures.GetSampleThemes()
		require.NotNil(t, themes)

		themes["default"] = "tampered"
		delete(themes, "dark")

		shared := fixtures.GetSampleThemes()
		assert.Contains(t, shared, "default", "mutating a copy must not alter the shared sample themes")
		assert.Contains(t, shared, "dark", "deleting a key on a copy must not alter the shared sample themes")
	})
}

// TestGetSampleAPIResponse verifies that GetSampleAPIResponse resolves every
// documented response type to the expected payload and returns nil for unknown
// or empty response types.
func TestGetSampleAPIResponse(t *testing.T) {
	t.Run("accounts returns the two sample accounts", func(t *testing.T) {
		response := fixtures.GetSampleAPIResponse("accounts")
		accounts, ok := response.([]map[string]interface{})
		require.True(t, ok, "accounts response should be []map[string]interface{}, got %T", response)
		require.Len(t, accounts, 2)

		assert.Equal(t, "1234567890abcdef1234567890abcdef", accounts[0]["id"])
		assert.Equal(t, "Test Account", accounts[0]["name"])
		assert.Equal(t, "active", accounts[0]["status"])
		assert.Equal(t, "Another Account", accounts[1]["name"])
	})

	t.Run("buckets returns the two sample buckets", func(t *testing.T) {
		response := fixtures.GetSampleAPIResponse("buckets")
		buckets, ok := response.([]map[string]interface{})
		require.True(t, ok, "buckets response should be []map[string]interface{}, got %T", response)
		require.Len(t, buckets, 2)

		assert.Equal(t, "test-bucket", buckets[0]["name"])
		assert.Equal(t, "2024-01-01T00:00:00Z", buckets[0]["creation_date"])

		location, ok := buckets[1]["location"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "auto", location["name"])
	})

	t.Run("objects returns the two sample objects", func(t *testing.T) {
		response := fixtures.GetSampleAPIResponse("objects")
		objects, ok := response.([]map[string]interface{})
		require.True(t, ok, "objects response should be []map[string]interface{}, got %T", response)
		require.Len(t, objects, 2)

		assert.Equal(t, "test-file.txt", objects[0]["key"])
		assert.Equal(t, 1024, objects[0]["size"])
		assert.Equal(t, "folder/nested-file.txt", objects[1]["key"])
		assert.Equal(t, 2048, objects[1]["size"])
	})

	t.Run("unknown response type returns nil", func(t *testing.T) {
		assert.Nil(t, fixtures.GetSampleAPIResponse("does-not-exist"))
	})

	t.Run("empty response type returns nil", func(t *testing.T) {
		assert.Nil(t, fixtures.GetSampleAPIResponse(""))
	})
}

// mapKeys returns the keys of files in a stable, human-readable form so that
// assertion failures show which filenames were actually generated.
func mapKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
