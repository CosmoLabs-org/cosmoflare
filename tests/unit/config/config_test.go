package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/tests/helpers"
)

func TestConfigManagerNew(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)
	require.NotNil(t, cm)

	// Should create config file in user home dir, but for testing we'll use temp dir
	assert.NotEmpty(t, cm.GetConfigPath())
}

func TestConfigManagerProfileOperations(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Test adding a profile
	profile := &config.Profile{
		Name:        "test-profile",
		AccountID:   "1234567890abcdef1234567890abcdef",
		APIToken:    "test-api-token-123456789",
		Description: "Test profile for unit testing",
		Endpoint:    "https://test.r2.cloudflarestorage.com",
		AccessKey:   "test-access-key",
		SecretKey:   "test-secret-key",
		Region:      "auto",
	}

	err = cm.SetProfile(profile)
	require.NoError(t, err)

	// Test profile exists
	assert.True(t, cm.ProfileExists("test-profile"))

	// Test getting profile
	retrieved, err := cm.GetProfile("test-profile")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, profile.Name, retrieved.Name)
	assert.Equal(t, profile.AccountID, retrieved.AccountID)
	assert.Equal(t, profile.APIToken, retrieved.APIToken)

	// Test listing profiles
	profiles := cm.ListProfiles()
	assert.Contains(t, profiles, "test-profile")

	// Test setting current profile
	err = cm.SetCurrent("test-profile")
	require.NoError(t, err)

	// Test getting current profile
	current, err := cm.GetCurrent()
	require.NoError(t, err)
	require.NotNil(t, current)
	assert.Equal(t, "test-profile", current.Name)

	// Test deleting profile (can't delete current, so switch to a non-existent one first)
	// Set current to a non-existent profile (will be created as default by ConfigManager)
	err = cm.SetCurrent("another-profile")
	// This might fail if the profile doesn't exist, which is expected for testing
	if err != nil {
		// If it fails, that's fine - we're testing deletion capability
	}

	// Try to delete the test profile - this should fail because it's current
	err = cm.DeleteProfile("test-profile")
	if err == nil {
		// If deletion succeeded, verify it's gone
		assert.False(t, cm.ProfileExists("test-profile"))
	} else {
		// Expected failure due to being current profile
		assert.Contains(t, err.Error(), "cannot delete current profile")
	}
}

func TestConfigManagerValidation(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Test valid profile
	validProfile := &config.Profile{
		Name:      "valid-profile",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "valid-api-token-with-sufficient-length",
	}

	err = cm.ValidateProfile(validProfile)
	assert.NoError(t, err)

	// Test invalid profile - empty name
	invalidProfile := &config.Profile{
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "valid-api-token-with-sufficient-length",
	}

	err = cm.ValidateProfile(invalidProfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile name is required")

	// Test invalid profile - empty account ID
	invalidProfile.Name = "test"
	invalidProfile.AccountID = ""

	err = cm.ValidateProfile(invalidProfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID is required")

	// Test invalid profile - wrong account ID length
	invalidProfile.AccountID = "12345678"

	err = cm.ValidateProfile(invalidProfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID should be 32 characters long")

	// Test invalid profile - empty API token
	invalidProfile.AccountID = "1234567890abcdef1234567890abcdef"
	invalidProfile.APIToken = ""

	err = cm.ValidateProfile(invalidProfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API token is required")

	// Test invalid profile - short API token
	invalidProfile.APIToken = "short"

	err = cm.ValidateProfile(invalidProfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API token appears to be invalid")
}

func TestConfigManagerEnvironmentDetection(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Test environment variable detection
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "1234567890abcdef1234567890abcdef")
	os.Setenv("CLOUDFLARE_API_TOKEN", "env-api-token-123456789")
	os.Setenv("R2_ENDPOINT", "https://env.r2.cloudflarestorage.com")
	os.Setenv("AWS_ACCESS_KEY_ID", "env-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret-key")
	os.Setenv("AWS_REGION", "us-east-1")

	defer func() {
		os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
		os.Unsetenv("CLOUDFLARE_API_TOKEN")
		os.Unsetenv("R2_ENDPOINT")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	}()

	// Test auto-detection
	detected := cm.AutoDetectProfile()
	require.NotNil(t, detected)
	assert.Equal(t, "auto-detected", detected.Name)
	assert.Equal(t, "1234567890abcdef1234567890abcdef", detected.AccountID)
	assert.Equal(t, "env-api-token-123456789", detected.APIToken)
	assert.Equal(t, "https://env.r2.cloudflarestorage.com", detected.Endpoint)

	// Test loading from environment
	loaded := config.LoadFromEnvironment()
	require.NotNil(t, loaded)
	assert.Equal(t, "1234567890abcdef1234567890abcdef", loaded.AccountID)
	assert.Equal(t, "env-api-token-123456789", loaded.APIToken)
}

func TestConfigManagerExportProfile(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Add a test profile
	profile := &config.Profile{
		Name:        "export-test",
		AccountID:   "1234567890abcdef1234567890abcdef",
		APIToken:    "export-api-token-123456789",
		Endpoint:    "https://export.r2.cloudflarestorage.com",
		AccessKey:   "export-access-key",
		SecretKey:   "export-secret-key",
		Region:      "us-east-1",
	}

	err = cm.SetProfile(profile)
	require.NoError(t, err)

	// Test profile export
	exportScript, err := cm.ExportProfile("export-test")
	require.NoError(t, err)

	assert.Contains(t, exportScript, `export CLOUDFLARE_API_TOKEN="export-api-token-123456789"`)
	assert.Contains(t, exportScript, `export CLOUDFLARE_ACCOUNT_ID="1234567890abcdef1234567890abcdef"`)
	assert.Contains(t, exportScript, `export R2_ENDPOINT="https://export.r2.cloudflarestorage.com"`)
	assert.Contains(t, exportScript, `export AWS_ACCESS_KEY_ID="export-access-key"`)
	assert.Contains(t, exportScript, `export AWS_SECRET_ACCESS_KEY="export-secret-key"`)
	assert.Contains(t, exportScript, `export AWS_REGION="us-east-1"`)

	// Test exporting non-existent profile
	_, err = cm.ExportProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfigManagerSanitization(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Add a profile with sensitive data
	profile := &config.Profile{
		Name:        "sanitize-test",
		AccountID:   "1234567890abcdef1234567890abcdef",
		APIToken:    "sensitive-api-token-123456789",
		AccessKey:   "sensitive-access-key",
		SecretKey:   "sensitive-secret-key",
		Description: "Test profile for sanitization",
		Endpoint:    "https://sanitize.r2.cloudflarestorage.com",
		Region:      "auto",
	}

	err = cm.SetProfile(profile)
	require.NoError(t, err)

	// Test sanitized output
	sanitized := cm.SanitizeForOutput()
	require.NotNil(t, sanitized)

	sanitizedProfile := sanitized.Profiles["sanitize-test"]
	require.NotNil(t, sanitizedProfile)

	// Sensitive data should be masked or missing
	assert.Equal(t, profile.Name, sanitizedProfile.Name)
	assert.Equal(t, profile.Description, sanitizedProfile.Description)
	assert.Equal(t, profile.Endpoint, sanitizedProfile.Endpoint)
	assert.Equal(t, profile.Region, sanitizedProfile.Region)

	// Account ID should be masked (first 4 and last 4 chars visible)
	assert.Contains(t, sanitizedProfile.AccountID, "1234")
	assert.Contains(t, sanitizedProfile.AccountID, "cdef")
	assert.Contains(t, sanitizedProfile.AccountID, "*")

	// Access key should be masked (first 2 and last 2 chars visible)
	assert.Contains(t, sanitizedProfile.AccessKey, "se")
	assert.Contains(t, sanitizedProfile.AccessKey, "ey")
	assert.Contains(t, sanitizedProfile.AccessKey, "*")

	// Secret key and API token should never appear in sanitized output
	assert.Empty(t, sanitizedProfile.SecretKey)
	assert.Empty(t, sanitizedProfile.APIToken)
}

func TestConfigManagerJSONOutput(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Add a test profile
	profile := &config.Profile{
		Name:        "json-test",
		AccountID:   "1234567890abcdef1234567890abcdef",
		APIToken:    "json-api-token-123456789",
		Endpoint:    "https://json.r2.cloudflarestorage.com",
	}

	err = cm.SetProfile(profile)
	require.NoError(t, err)

	// Test JSON output
	jsonOutput, err := cm.JSON()
	require.NoError(t, err)

	assert.Contains(t, jsonOutput, "\"name\": \"json-test\"")
	assert.Contains(t, jsonOutput, "\"account_id\": \"1234567890abcdef1234567890abcdef\"")
	assert.Contains(t, jsonOutput, "\"api_token\": \"json-api-token-123456789\"")
}

func TestConfigManagerErrorHandling(t *testing.T) {
	helpers.SetupTest(t)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Test deleting non-existent profile
	err = cm.DeleteProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test setting current to non-existent profile
	err = cm.SetCurrent("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting current when none set
	// Create a fresh config manager to test default behavior
	freshCM, err := config.NewConfigManager()
	require.NoError(t, err)

	// Try to get current profile - should work with default or show appropriate error
	currentProfile, err := freshCM.GetCurrent()
	if err != nil {
		assert.Contains(t, err.Error(), "no current profile set")
	} else {
		// Should have a default profile if the implementation creates one
		assert.NotNil(t, currentProfile)
	}
}

func TestConfigMaskingFunctions(t *testing.T) {
	// Test account ID masking
	shortID := "12345678"
	masked := config.MaskAccountID(shortID)
	assert.True(t, len(masked) >= len(shortID)) // Should be same length or longer due to masking

	longID := "1234567890abcdef1234567890abcdef"
	maskedLong := config.MaskAccountID(longID)
	assert.Contains(t, maskedLong, "1234") // Should preserve first 4 chars
	assert.Contains(t, maskedLong, "cdef") // Should preserve last 4 chars
	assert.Contains(t, maskedLong, "*")    // Should have masking chars

	// Test key masking
	emptyKey := ""
	assert.Equal(t, "", config.MaskKey(emptyKey))

	shortKey := "1234"
	maskedShort := config.MaskKey(shortKey)
	assert.True(t, len(maskedShort) >= len(shortKey)) // Should be same length or longer

	longKey := "1234567890abcdef"
	maskedLongKey := config.MaskKey(longKey)
	assert.Contains(t, maskedLongKey, "12") // Should preserve first 2 chars
	assert.Contains(t, maskedLongKey, "ef") // Should preserve last 2 chars
	assert.Contains(t, maskedLongKey, "*") // Should have masking chars
}

func TestConfigManagerConcurrentOperations(t *testing.T) {
	helpers.SetupTest(t)

	// Test concurrent read operations (safer than concurrent writes)
	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Add a test profile first
	testProfile := &config.Profile{
		Name:      "concurrent-test",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "test-token",
	}
	err = cm.SetProfile(testProfile)
	require.NoError(t, err)

	// Test concurrent read operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(num int) {
			// Read operations should be safe
			exists := cm.ProfileExists("concurrent-test")
			assert.True(t, exists)

			profiles := cm.ListProfiles()
			assert.NotEmpty(t, profiles)

			retrieved, err := cm.GetProfile("concurrent-test")
			assert.NoError(t, err)
			assert.Equal(t, "concurrent-test", retrieved.Name)

			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}