package error_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// TestConfigDeleteCurrentProfile ensures deleting the active profile is blocked
func TestConfigDeleteCurrentProfile(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	profile := &config.Profile{
		Name:      "active",
		AccountID: "1234567890abcdef1234567890abcdef",
		APIToken:  "active-token-1234567890",
	}
	require.NoError(t, cm.SetProfile(profile))
	require.NoError(t, cm.SetCurrent("active"))

	err = cm.DeleteProfile("active")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete current profile")
}

// TestConfigGetNonexistentProfile ensures looking up missing profiles fails cleanly
func TestConfigGetNonexistentProfile(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	_, err = cm.GetProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestConfigSwitchNonexistentProfile ensures switching to missing profile fails
func TestConfigSwitchNonexistentProfile(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	err = cm.SetCurrent("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestConfigDeleteNonexistentProfile ensures deleting missing profile fails
func TestConfigDeleteNonexistentProfile(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	err = cm.DeleteProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestConfigExportNonexistentProfile ensures exporting missing profile fails
func TestConfigExportNonexistentProfile(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/.r2go2", 0755))

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	_, err = cm.ExportProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestConfigValidationEdgeCases tests boundary conditions in validation
func TestConfigValidationEdgeCases(t *testing.T) {
	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	tests := []struct {
		name    string
		profile *config.Profile
		errMsg  string
	}{
		{
			name: "empty profile",
			profile: &config.Profile{},
			errMsg: "profile name is required",
		},
		{
			name: "name only",
			profile: &config.Profile{Name: "x"},
			errMsg: "account ID is required",
		},
		{
			name: "31 char account id",
			profile: &config.Profile{
				Name:      "almost",
				AccountID: "1234567890abcdef1234567890abcde",
				APIToken:  "valid-length-token-here",
			},
			errMsg: "32 characters",
		},
		{
			name: "33 char account id",
			profile: &config.Profile{
				Name:      "toolong",
				AccountID: "1234567890abcdef1234567890abcdef1",
				APIToken:  "valid-length-token-here",
			},
			errMsg: "32 characters",
		},
		{
			name: "9 char api token",
			profile: &config.Profile{
				Name:      "shorttok",
				AccountID: "1234567890abcdef1234567890abcdef",
				APIToken:  "123456789",
			},
			errMsg: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateProfile(tt.profile)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
