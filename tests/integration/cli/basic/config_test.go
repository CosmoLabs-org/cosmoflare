package basic_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// TestConfigInit creates a fresh config in a temp home directory
func TestConfigInit(t *testing.T) {
	tempDir := t.TempDir()
	configDir := tempDir + "/.cosmoflare"
	require.NoError(t, os.MkdirAll(configDir, 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)
	require.NotNil(t, cm)

	assert.Equal(t, configDir+"/config.yaml", cm.GetConfigPath())
}

// TestConfigProfileCRUD tests create, read, update, delete operations on profiles
func TestConfigProfileCRUD(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Create
	profile := &config.Profile{
		Name:      "production",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "prod-api-token-1234567890abcdef",
		Endpoint:  "https://prod.r2.cloudflarestorage.com",
		Region:    "auto",
	}
	require.NoError(t, cm.SetProfile(profile))

	// Read
	got, err := cm.GetProfile("production")
	require.NoError(t, err)
	assert.Equal(t, "production", got.Name)
	assert.Equal(t, "1234567890abcdef1234567890abcdef", got.AccountID)

	// Update
	profile.APIToken = "updated-token-1234567890123456"
	require.NoError(t, cm.SetProfile(profile))
	updated, err := cm.GetProfile("production")
	require.NoError(t, err)
	assert.Equal(t, "updated-token-1234567890123456", updated.APIToken)

	// Delete — need a different current profile
	other := &config.Profile{
		Name:      "staging",
		AccountID: "abcdef1234567890abcdef1234567890",
		APIToken:  "staging-token-12345678901234567",
	}
	require.NoError(t, cm.SetProfile(other))
	require.NoError(t, cm.SetCurrent("staging"))
	require.NoError(t, cm.DeleteProfile("production"))
	assert.False(t, cm.ProfileExists("production"))
}

// TestConfigMultiProfileSwitch tests switching between profiles
func TestConfigMultiProfileSwitch(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	profiles := []*config.Profile{
		{Name: "prod", AccountID: "aaaabbbbccccddddaaaabbbbccccdddd", APIToken: "prod-token-123456789012345678"},
		{Name: "staging", AccountID: "ddddccccbbbbaaaaddddccccbbbbaaaa", APIToken: "stag-token-123456789012345678"},
		{Name: "dev", AccountID: "11112222333344441111222233334444", APIToken: "devv-token-123456789012345678"},
	}

	for _, p := range profiles {
		require.NoError(t, cm.SetProfile(p))
	}

	// Switch to each profile and verify
	for _, p := range profiles {
		require.NoError(t, cm.SetCurrent(p.Name))
		current, err := cm.GetCurrent()
		require.NoError(t, err)
		assert.Equal(t, p.Name, current.Name)
		assert.Equal(t, p.AccountID, current.AccountID)
	}

	// Verify all profiles exist
	all := cm.ListProfiles()
	assert.Len(t, all, 3)
}

// TestConfigValidate tests profile validation
func TestConfigValidate(t *testing.T) {
	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	tests := []struct {
		name    string
		profile *config.Profile
		wantErr string
	}{
		{
			name: "valid profile",
			profile: &config.Profile{
				Name:      "valid",
				AccountID: "1234567890abcdef1234567890abcdef",
				APIToken:  "valid-api-token-123456789",
			},
			wantErr: "",
		},
		{
			name: "missing name",
			profile: &config.Profile{
				AccountID: "1234567890abcdef1234567890abcdef",
				APIToken:  "valid-api-token-123456789",
			},
			wantErr: "profile name is required",
		},
		{
			name: "missing account id",
			profile: &config.Profile{
				Name:     "no-account",
				APIToken: "valid-api-token-123456789",
			},
			wantErr: "account ID is required",
		},
		{
			name: "short account id",
			profile: &config.Profile{
				Name:      "short-id",
				AccountID: "12345678",
				APIToken:  "valid-api-token-123456789",
			},
			wantErr: "32 characters",
		},
		{
			name: "missing api token",
			profile: &config.Profile{
				Name:      "no-token",
				AccountID: "1234567890abcdef1234567890abcdef",
			},
			wantErr: "API token is required",
		},
		{
			name: "short api token",
			profile: &config.Profile{
				Name:      "short-token",
				AccountID: "1234567890abcdef1234567890abcdef",
				APIToken:  "short",
			},
			wantErr: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateProfile(tt.profile)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// TestConfigExport tests profile export to env vars
func TestConfigExport(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	profile := &config.Profile{
		Name:      "export-test",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "export-token-123456789",
		Endpoint:  "https://export.r2.cloudflarestorage.com",
		AccessKey: "export-access-key",
		SecretKey: "export-secret-key",
		Region:    "us-east-1",
	}
	require.NoError(t, cm.SetProfile(profile))

	export, err := cm.ExportProfile("export-test")
	require.NoError(t, err)
	assert.Contains(t, export, `export CLOUDFLARE_API_TOKEN="export-token-123456789"`)
	assert.Contains(t, export, `export CLOUDFLARE_ACCOUNT_ID="1234567890abcdef1234567890abcdef"`)
	assert.Contains(t, export, `export R2_ENDPOINT="https://export.r2.cloudflarestorage.com"`)
	assert.Contains(t, export, `export AWS_ACCESS_KEY_ID="export-access-key"`)
	assert.Contains(t, export, `export AWS_SECRET_ACCESS_KEY="export-secret-key"`)
	assert.Contains(t, export, `export AWS_REGION="us-east-1"`)
}

// TestConfigSanitization ensures secrets are masked in output
func TestConfigSanitization(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	profile := &config.Profile{
		Name:      "sanitized",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "secret-token-that-should-be-hidden",
		SecretKey: "super-secret-key",
	}
	require.NoError(t, cm.SetProfile(profile))

	sanitized := cm.SanitizeForOutput()
	got := sanitized.Profiles["sanitized"]
	require.NotNil(t, got)

	// Sensitive fields should be empty or masked
	assert.Empty(t, got.APIToken)
	assert.Empty(t, got.SecretKey)
	assert.Contains(t, got.AccountID, "*")
}
