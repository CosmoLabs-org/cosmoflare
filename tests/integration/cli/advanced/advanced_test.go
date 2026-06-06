package advanced_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// setupTempConfig creates a temp home dir and returns a ConfigManager
func setupTempConfig(t *testing.T) *config.ConfigManager {
	t.Helper()
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)
	return cm
}

// TestConfigJSONOutput tests that JSON output is well-formed
func TestConfigJSONOutput(t *testing.T) {
	cm := setupTempConfig(t)

	profile := &config.Profile{
		Name:      "json-test",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "json-token-1234567890",
		Region:    "auto",
	}
	require.NoError(t, cm.SetProfile(profile))
	require.NoError(t, cm.SetCurrent("json-test"))

	output, err := cm.JSON()
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))

	assert.Equal(t, "json-test", parsed["current"])

	profiles, ok := parsed["profiles"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, profiles, "json-test")

	p := profiles["json-test"].(map[string]interface{})
	assert.Equal(t, "1234567890abcdef1234567890abcdef", p["account_id"])
}

// TestConfigAutoDetectFromEnv tests auto-detection of credentials from environment
func TestConfigAutoDetectFromEnv(t *testing.T) {
	cm := setupTempConfig(t)

	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "aaaabbbbccccddddaaaabbbbccccdddd")
	os.Setenv("CLOUDFLARE_API_TOKEN", "env-detected-token-1234567890")
	os.Setenv("R2_ENDPOINT", "https://env.r2.cloudflarestorage.com")
	t.Cleanup(func() {
		os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
		os.Unsetenv("CLOUDFLARE_API_TOKEN")
		os.Unsetenv("R2_ENDPOINT")
	})

	detected := cm.AutoDetectProfile()
	require.NotNil(t, detected)
	assert.Equal(t, "auto-detected", detected.Name)
	assert.Equal(t, "aaaabbbbccccddddaaaabbbbccccdddd", detected.AccountID)
	assert.Equal(t, "env-detected-token-1234567890", detected.APIToken)
	assert.Equal(t, "https://env.r2.cloudflarestorage.com", detected.Endpoint)
}

// TestConfigAutoDetectMissingEnv returns nil when env vars not set
func TestConfigAutoDetectMissingEnv(t *testing.T) {
	cm := setupTempConfig(t)

	// Ensure env vars are not set
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	os.Unsetenv("CLOUDFLARE_API_TOKEN")

	detected := cm.AutoDetectProfile()
	assert.Nil(t, detected)
}

// TestConfigPersistence tests that profiles survive save/load cycles
func TestConfigPersistence(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	// Create and save profiles
	cm1, err := config.NewConfigManager()
	require.NoError(t, err)

	for _, p := range []*config.Profile{
		{Name: "alpha", AccountID: "11111111111111111111111111111111", APIToken: "alpha-token-12345678901234567"},
		{Name: "beta", AccountID: "22222222222222222222222222222222", APIToken: "beta-token-123456789012345678"},
	} {
		require.NoError(t, cm1.SetProfile(p))
	}
	require.NoError(t, cm1.SetCurrent("beta"))

	// Reload config from disk
	cm2, err := config.NewConfigManager()
	require.NoError(t, err)

	current, err := cm2.GetCurrent()
	require.NoError(t, err)
	assert.Equal(t, "beta", current.Name)
	assert.Equal(t, "22222222222222222222222222222222", current.AccountID)

	alpha, err := cm2.GetProfile("alpha")
	require.NoError(t, err)
	assert.Equal(t, "alpha", alpha.Name)

	assert.True(t, cm2.ProfileExists("alpha"))
	assert.True(t, cm2.ProfileExists("beta"))
	assert.False(t, cm2.ProfileExists("gamma"))
}

// TestConfigFilePermissions ensures config file has restricted permissions
func TestConfigFilePermissions(t *testing.T) {
	cm := setupTempConfig(t)

	profile := &config.Profile{
		Name:      "perm-test",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "perm-token-1234567890",
	}
	require.NoError(t, cm.SetProfile(profile))

	info, err := os.Stat(cm.GetConfigPath())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "config file should be owner-readable only")
}

// TestConfigConcurrentReads tests that concurrent reads don't corrupt state
func TestConfigConcurrentReads(t *testing.T) {
	cm := setupTempConfig(t)

	profile := &config.Profile{
		Name:      "concurrent",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "concurrent-token-1234567890",
	}
	require.NoError(t, cm.SetProfile(profile))

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			assert.True(t, cm.ProfileExists("concurrent"))
			p, err := cm.GetProfile("concurrent")
			assert.NoError(t, err)
			assert.Equal(t, "concurrent", p.Name)
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestConfigMultipleProfilesIsolation ensures profiles don't leak into each other
func TestConfigMultipleProfilesIsolation(t *testing.T) {
	cm := setupTempConfig(t)

	profiles := map[string]*config.Profile{
		"prod":     {Name: "prod", AccountID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", APIToken: "prod-token-1234567890123456", Endpoint: "https://prod.r2.cloudflarestorage.com", Region: "us-east-1"},
		"staging":  {Name: "staging", AccountID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", APIToken: "stag-token-1234567890123456", Endpoint: "https://stag.r2.cloudflarestorage.com", Region: "eu-west-1"},
		"personal": {Name: "personal", AccountID: "cccccccccccccccccccccccccccccccc", APIToken: "pers-token-1234567890123456", AccessKey: "personal-access-key", SecretKey: "personal-secret-key"},
	}

	for _, p := range profiles {
		require.NoError(t, cm.SetProfile(p))
	}

	for name, expected := range profiles {
		got, err := cm.GetProfile(name)
		require.NoError(t, err)
		assert.Equal(t, expected.AccountID, got.AccountID, "profile %s should have correct AccountID", name)
		assert.Equal(t, expected.APIToken, got.APIToken, "profile %s should have correct APIToken", name)

		if expected.Endpoint != "" {
			assert.Equal(t, expected.Endpoint, got.Endpoint, "profile %s should have correct Endpoint", name)
		}
		if expected.AccessKey != "" {
			assert.Equal(t, expected.AccessKey, got.AccessKey, "profile %s should have correct AccessKey", name)
		}
	}
}
