package interactive

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// helper to build a wizard with mock input and disabled animations
func newTestWizard(inputs ...string) (*AdvancedConfigWizard, *config.Profile) {
	profile := &config.Profile{Name: "test", Description: "test profile"}
	acw := NewAdvancedConfigWizard(profile)
	acw.Input = newMockReader(inputs...)
	return acw, profile
}

func disableAnimations(t *testing.T) {
	t.Helper()
	origDisabled := globalAnimator.Disabled
	t.Cleanup(func() { globalAnimator.Disabled = origDisabled })
	globalAnimator.Disabled = true
}

// ──────────────────────────────────────────────
// configureBucketSettings
// ──────────────────────────────────────────────

// TestConfigureBucketSettings verifies configureBucketSettings behavior, one t.Run subtest per...
func TestConfigureBucketSettings(t *testing.T) {
	disableAnimations(t)

	t.Run("bucket type", testBucketSettingsType)
	t.Run("retention days", testBucketSettingsRetention)
	t.Run("custom endpoint", testBucketSettingsEndpoint)
}

func testBucketSettingsType(t *testing.T) {
	t.Run("empty input sets standard type and no endpoint", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "standard", bs.Type)
		assert.Equal(t, 0, bs.RetentionDays) // function doesn't set a default; caller must pre-fill
		assert.Equal(t, "", bs.CustomEndpoint)
	})

	t.Run("select 1 yields standard type", func(t *testing.T) {
		acw, _ := newTestWizard("1", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "standard", bs.Type)
	})

	t.Run("select 2 yields performance type", func(t *testing.T) {
		acw, _ := newTestWizard("2", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "performance", bs.Type)
	})

	t.Run("select 3 yields cost type", func(t *testing.T) {
		acw, _ := newTestWizard("3", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "cost", bs.Type)
	})

	t.Run("invalid type defaults to standard with warning", func(t *testing.T) {
		acw, _ := newTestWizard("99", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "standard", bs.Type)
	})
}

func testBucketSettingsRetention(t *testing.T) {
	t.Run("valid retention days", func(t *testing.T) {
		acw, _ := newTestWizard("", "90", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, 90, bs.RetentionDays)
	})

	t.Run("zero retention days is accepted", func(t *testing.T) {
		acw, _ := newTestWizard("", "0", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, 0, bs.RetentionDays)
	})

	t.Run("invalid retention leaves value unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "abc", "")
		bs := BucketSettings{RetentionDays: 30}
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, 30, bs.RetentionDays) // unchanged from pre-filled value
	})

	t.Run("negative retention leaves value unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "-5", "")
		bs := BucketSettings{RetentionDays: 30}
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, 30, bs.RetentionDays)
	})
}

func testBucketSettingsEndpoint(t *testing.T) {
	t.Run("custom endpoint is stored", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "https://custom.example.com")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "https://custom.example.com", bs.CustomEndpoint)
	})

	t.Run("empty endpoint stays empty", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "")
		var bs BucketSettings
		err := acw.configureBucketSettings(&bs)
		require.NoError(t, err)
		assert.Equal(t, "", bs.CustomEndpoint)
	})
}

// ──────────────────────────────────────────────
// configureUploadSettings
// ──────────────────────────────────────────────

// TestConfigureUploadSettings verifies configureUploadSettings behavior, one t.Run subtest per...
func TestConfigureUploadSettings(t *testing.T) {
	disableAnimations(t)

	t.Run("defaults", testUploadSettingsDefaults)
	t.Run("concurrency", testUploadSettingsConcurrency)
	t.Run("chunk size", testUploadSettingsChunkSize)
	t.Run("retry attempts", testUploadSettingsRetries)
	t.Run("checksum", testUploadSettingsChecksum)
}

func testUploadSettingsDefaults(t *testing.T) {
	t.Run("empty inputs leave values unchanged, checksum defaults to true", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "", "")
		us := UploadSettings{Concurrency: 4, ChunkSize: "8MB", RetryAttempts: 3}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 4, us.Concurrency)
		assert.Equal(t, "8MB", us.ChunkSize)
		assert.Equal(t, 3, us.RetryAttempts)
		assert.True(t, us.ChecksumEnabled) // ConfirmWithReader default=true
	})
}

func testUploadSettingsConcurrency(t *testing.T) {
	t.Run("valid concurrency 8", func(t *testing.T) {
		acw, _ := newTestWizard("8", "", "", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 8, us.Concurrency)
	})

	t.Run("concurrency 32 is accepted (upper bound)", func(t *testing.T) {
		acw, _ := newTestWizard("32", "", "", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 32, us.Concurrency)
	})

	t.Run("concurrency 33 is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("33", "", "", "")
		us := UploadSettings{Concurrency: 4}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 4, us.Concurrency)
	})

	t.Run("concurrency 0 is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("0", "", "", "")
		us := UploadSettings{Concurrency: 4}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 4, us.Concurrency)
	})

	t.Run("invalid concurrency string leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("abc", "", "", "")
		us := UploadSettings{Concurrency: 4}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 4, us.Concurrency)
	})

	t.Run("negative concurrency leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("-1", "", "", "")
		us := UploadSettings{Concurrency: 4}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 4, us.Concurrency)
	})
}

func testUploadSettingsChunkSize(t *testing.T) {
	t.Run("valid chunk size 16", func(t *testing.T) {
		acw, _ := newTestWizard("", "16", "", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "16MB", us.ChunkSize)
	})

	t.Run("chunk size 1 is accepted (lower bound)", func(t *testing.T) {
		acw, _ := newTestWizard("", "1", "", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "1MB", us.ChunkSize)
	})

	t.Run("chunk size 100 is accepted (upper bound)", func(t *testing.T) {
		acw, _ := newTestWizard("", "100", "", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "100MB", us.ChunkSize)
	})

	t.Run("chunk size 0 is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "0", "", "")
		us := UploadSettings{ChunkSize: "8MB"}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "8MB", us.ChunkSize)
	})

	t.Run("chunk size 101 is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "101", "", "")
		us := UploadSettings{ChunkSize: "8MB"}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "8MB", us.ChunkSize)
	})

	t.Run("invalid chunk size string leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "big", "", "")
		us := UploadSettings{ChunkSize: "8MB"}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, "8MB", us.ChunkSize)
	})
}

func testUploadSettingsRetries(t *testing.T) {
	t.Run("valid retries 5", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "5", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 5, us.RetryAttempts)
	})

	t.Run("retries 0 is accepted", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "0", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 0, us.RetryAttempts)
	})

	t.Run("retries 10 is accepted (upper bound)", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "10", "")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 10, us.RetryAttempts)
	})

	t.Run("retries 11 is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "11", "")
		us := UploadSettings{RetryAttempts: 3}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 3, us.RetryAttempts)
	})

	t.Run("negative retries is rejected, leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "-1", "")
		us := UploadSettings{RetryAttempts: 3}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 3, us.RetryAttempts)
	})

	t.Run("invalid retries string leaves unchanged", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "x", "")
		us := UploadSettings{RetryAttempts: 3}
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.Equal(t, 3, us.RetryAttempts)
	})
}

func testUploadSettingsChecksum(t *testing.T) {
	t.Run("checksum yes", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "", "y")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.True(t, us.ChecksumEnabled)
	})

	t.Run("checksum no overrides default", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "", "n")
		var us UploadSettings
		err := acw.configureUploadSettings(&us)
		require.NoError(t, err)
		assert.False(t, us.ChecksumEnabled)
	})
}

// ──────────────────────────────────────────────
// configureRegionSettings
// ──────────────────────────────────────────────

// TestConfigureRegionSettings verifies configureRegionSettings behavior, one t.Run subtest per...
func TestConfigureRegionSettings(t *testing.T) {
	disableAnimations(t)

	t.Run("empty input defaults to auto-detect", func(t *testing.T) {
		acw, _ := newTestWizard("")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.True(t, rs.AutoDetect)
		assert.Equal(t, "auto", rs.Primary)
	})

	t.Run("select 1 yields auto-detect", func(t *testing.T) {
		acw, _ := newTestWizard("1")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.True(t, rs.AutoDetect)
		assert.Equal(t, "auto", rs.Primary)
	})

	t.Run("select 2 yields us-east-1", func(t *testing.T) {
		acw, _ := newTestWizard("2")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.False(t, rs.AutoDetect)
		assert.Equal(t, "us-east-1", rs.Primary)
	})

	t.Run("select 3 yields eu-west-1", func(t *testing.T) {
		acw, _ := newTestWizard("3")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.False(t, rs.AutoDetect)
		assert.Equal(t, "eu-west-1", rs.Primary)
	})

	t.Run("select 4 yields ap-southeast-1", func(t *testing.T) {
		acw, _ := newTestWizard("4")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.False(t, rs.AutoDetect)
		assert.Equal(t, "ap-southeast-1", rs.Primary)
	})

	t.Run("invalid selection defaults to auto-detect", func(t *testing.T) {
		acw, _ := newTestWizard("99")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.True(t, rs.AutoDetect)
		assert.Equal(t, "auto", rs.Primary)
	})

	t.Run("string input defaults to auto-detect", func(t *testing.T) {
		acw, _ := newTestWizard("hello")
		var rs RegionSettings
		err := acw.configureRegionSettings(&rs)
		require.NoError(t, err)
		assert.True(t, rs.AutoDetect)
		assert.Equal(t, "auto", rs.Primary)
	})
}

// ──────────────────────────────────────────────
// configureAdditionalOptions
// ──────────────────────────────────────────────

// TestConfigureAdditionalOptions verifies configureAdditionalOptions behavior, one t.Run subtest...
func TestConfigureAdditionalOptions(t *testing.T) {
	disableAnimations(t)

	t.Run("theme", testAdditionalOptionsTheme)
	t.Run("analytics", testAdditionalOptionsAnalytics)
	t.Run("accessibility", testAdditionalOptionsAccessibility)
}

func testAdditionalOptionsTheme(t *testing.T) {
	t.Run("empty input defaults to cosmic theme, analytics off, accessibility off", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "cosmic", cfg.Theme)
		assert.False(t, cfg.AnalyticsEnabled)
		assert.False(t, cfg.AccessibilityEnabled)
	})

	t.Run("theme 1 cosmic", func(t *testing.T) {
		acw, _ := newTestWizard("1", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "cosmic", cfg.Theme)
	})

	t.Run("theme 2 forest", func(t *testing.T) {
		acw, _ := newTestWizard("2", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "forest", cfg.Theme)
	})

	t.Run("theme 3 ocean", func(t *testing.T) {
		acw, _ := newTestWizard("3", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "ocean", cfg.Theme)
	})

	t.Run("theme 4 sunset", func(t *testing.T) {
		acw, _ := newTestWizard("4", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "sunset", cfg.Theme)
	})

	t.Run("theme 5 monochrome", func(t *testing.T) {
		acw, _ := newTestWizard("5", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "monochrome", cfg.Theme)
	})

	t.Run("invalid theme defaults to cosmic", func(t *testing.T) {
		acw, _ := newTestWizard("99", "", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.Equal(t, "cosmic", cfg.Theme)
	})
}

func testAdditionalOptionsAnalytics(t *testing.T) {
	t.Run("analytics yes", func(t *testing.T) {
		acw, _ := newTestWizard("", "y", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.True(t, cfg.AnalyticsEnabled)
	})

	t.Run("analytics no", func(t *testing.T) {
		acw, _ := newTestWizard("", "n", "")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.AnalyticsEnabled)
	})
}

func testAdditionalOptionsAccessibility(t *testing.T) {
	t.Run("accessibility yes", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "y")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.True(t, cfg.AccessibilityEnabled)
	})

	t.Run("accessibility no", func(t *testing.T) {
		acw, _ := newTestWizard("", "", "n")
		cfg := &AdvancedConfig{}
		err := acw.configureAdditionalOptions(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.AccessibilityEnabled)
	})
}

// ──────────────────────────────────────────────
// ShowAdvancedConfig (full walkthrough)
// ──────────────────────────────────────────────

// TestShowAdvancedConfig verifies ShowAdvancedConfig behavior, one t.Run subtest per scenario.
func TestShowAdvancedConfig(t *testing.T) {
	disableAnimations(t)

	t.Run("all defaults (empty inputs)", func(t *testing.T) {
		// configureBucketSettings: type, retention, endpoint
		// configureUploadSettings: concurrency, chunk, retries, checksum
		// configureRegionSettings: region
		// configureAdditionalOptions: theme, analytics, accessibility
		acw, profile := newTestWizard("", "", "", "", "", "", "", "", "", "")
		result, err := acw.ShowAdvancedConfig()
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, profile, result.Profile)
		assert.Equal(t, "standard", result.BucketSettings.Type)
		assert.Equal(t, 30, result.BucketSettings.RetentionDays)
		assert.Equal(t, "", result.BucketSettings.CustomEndpoint)
		assert.Equal(t, 4, result.UploadSettings.Concurrency)
		assert.Equal(t, "8MB", result.UploadSettings.ChunkSize)
		assert.Equal(t, 3, result.UploadSettings.RetryAttempts)
		assert.True(t, result.UploadSettings.ChecksumEnabled)
		assert.True(t, result.RegionSettings.AutoDetect)
		assert.Equal(t, "auto", result.RegionSettings.Primary)
		assert.Equal(t, "cosmic", result.Theme)
		assert.False(t, result.AnalyticsEnabled)
		assert.False(t, result.AccessibilityEnabled)
	})

	t.Run("custom values throughout", func(t *testing.T) {
		// bucket: type=2 (performance), retention=60, endpoint=custom
		// upload: concurrency=16, chunk=32, retries=7, checksum=no
		// region: 3 (eu-west-1)
		// additional: theme=3 (ocean), analytics=yes, accessibility=yes
		acw, _ := newTestWizard(
			"2", "60", "https://r2.custom.com",
			"16", "32", "7", "n",
			"3",
			"3", "y", "y",
		)
		result, err := acw.ShowAdvancedConfig()
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "performance", result.BucketSettings.Type)
		assert.Equal(t, 60, result.BucketSettings.RetentionDays)
		assert.Equal(t, "https://r2.custom.com", result.BucketSettings.CustomEndpoint)
		assert.Equal(t, 16, result.UploadSettings.Concurrency)
		assert.Equal(t, "32MB", result.UploadSettings.ChunkSize)
		assert.Equal(t, 7, result.UploadSettings.RetryAttempts)
		assert.False(t, result.UploadSettings.ChecksumEnabled)
		assert.False(t, result.RegionSettings.AutoDetect)
		assert.Equal(t, "eu-west-1", result.RegionSettings.Primary)
		assert.Equal(t, "ocean", result.Theme)
		assert.True(t, result.AnalyticsEnabled)
		assert.True(t, result.AccessibilityEnabled)
	})
}

// ──────────────────────────────────────────────
// ShowConfigurationSummary
// ──────────────────────────────────────────────

// TestShowConfigurationSummary verifies ShowConfigurationSummary behavior, one t.Run subtest per...
func TestShowConfigurationSummary(t *testing.T) {
	disableAnimations(t)

	t.Run("fully populated config does not panic", func(t *testing.T) {
		acw, profile := newTestWizard()
		cfg := &AdvancedConfig{
			Profile: profile,
			BucketSettings: BucketSettings{
				Type:           "performance",
				DefaultRegion:  "us-east-1",
				CustomEndpoint: "https://custom.example.com",
				RetentionDays:  90,
			},
			UploadSettings: UploadSettings{
				Concurrency:     8,
				ChunkSize:       "16MB",
				RetryAttempts:   5,
				Timeout:         5 * time.Minute,
				ChecksumEnabled: true,
			},
			RegionSettings: RegionSettings{
				AutoDetect: false,
				Primary:    "eu-west-1",
				Backup:     "us-east-1",
			},
			AnalyticsEnabled:     true,
			Theme:                "forest",
			AccessibilityEnabled: true,
		}
		// Should not panic
		acw.ShowConfigurationSummary(cfg)
	})

	t.Run("minimal config with zero retention and no endpoint", func(t *testing.T) {
		acw, profile := newTestWizard()
		cfg := &AdvancedConfig{
			Profile: profile,
			BucketSettings: BucketSettings{
				Type:           "standard",
				RetentionDays:  0,
				CustomEndpoint: "",
			},
			UploadSettings: UploadSettings{
				Concurrency:     4,
				ChunkSize:       "8MB",
				RetryAttempts:   3,
				Timeout:         5 * time.Minute,
				ChecksumEnabled: false,
			},
			RegionSettings: RegionSettings{
				AutoDetect: true,
				Primary:    "auto",
			},
			Theme: "cosmic",
		}
		acw.ShowConfigurationSummary(cfg)
	})
}

// ──────────────────────────────────────────────
// SaveAdvancedConfig
// ──────────────────────────────────────────────

// TestSaveAdvancedConfig verifies SaveAdvancedConfig behavior, one t.Run subtest per scenario.
func TestSaveAdvancedConfig(t *testing.T) {
	disableAnimations(t)

	t.Run("saves profile with theme in description", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		acw, profile := newTestWizard()
		cfg := &AdvancedConfig{
			Profile: profile,
			BucketSettings: BucketSettings{
				Type:          "standard",
				RetentionDays: 30,
			},
			UploadSettings: UploadSettings{
				Concurrency:     4,
				ChunkSize:       "8MB",
				RetryAttempts:   3,
				Timeout:         5 * time.Minute,
				ChecksumEnabled: true,
			},
			RegionSettings: RegionSettings{
				AutoDetect: true,
				Primary:    "auto",
			},
			Theme: "ocean",
		}

		cm, err := config.NewConfigManager()
		require.NoError(t, err)

		err = acw.SaveAdvancedConfig(cfg, cm)
		require.NoError(t, err)

		// Verify the description was updated with the theme
		assert.Contains(t, profile.Description, "ocean")
		assert.Contains(t, profile.Description, "Advanced")
	})

	t.Run("preserves existing description text", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		profile := &config.Profile{
			Name:        "preserved",
			Description: "original description",
		}
		acw := NewAdvancedConfigWizard(profile)
		acw.Input = newMockReader()

		cfg := &AdvancedConfig{
			Profile: profile,
			Theme:   "sunset",
		}

		cm, err := config.NewConfigManager()
		require.NoError(t, err)

		err = acw.SaveAdvancedConfig(cfg, cm)
		require.NoError(t, err)

		assert.Contains(t, profile.Description, "original description")
		assert.Contains(t, profile.Description, "sunset")
	})
}

// ──────────────────────────────────────────────
// ValidateAdvancedConfig
// ──────────────────────────────────────────────

// TestValidateAdvancedConfig_Boundaries verifies ValidateAdvancedConfig behavior for the...
func TestValidateAdvancedConfig_Boundaries(t *testing.T) {
	t.Run("valid configurations", testValidateBoundariesValid)
	t.Run("boundaries", testValidateBoundariesEdges)
	t.Run("concurrency errors", testValidateBoundariesConcurrency)
	t.Run("retry and region errors", testValidateBoundariesRetryRegion)
}

func testValidateBoundariesValid(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: 3,
			},
			RegionSettings: RegionSettings{
				Primary: "auto",
			},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("all valid regions pass", func(t *testing.T) {
		acw, _ := newTestWizard()
		regions := []string{"auto", "us-east-1", "eu-west-1", "ap-southeast-1"}
		for _, region := range regions {
			cfg := &AdvancedConfig{
				UploadSettings: UploadSettings{
					Concurrency:   4,
					RetryAttempts: 3,
				},
				RegionSettings: RegionSettings{Primary: region},
			}
			err := acw.ValidateAdvancedConfig(cfg)
			assert.NoError(t, err, "region %s should be valid", region)
		}
	})
}

func testValidateBoundariesEdges(t *testing.T) {
	t.Run("concurrency boundary 1 is valid", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{Concurrency: 1},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("concurrency boundary 32 is valid", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{Concurrency: 32},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("retry boundary 0 is valid", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: 0,
			},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("retry boundary 10 is valid", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: 10,
			},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.NoError(t, err)
	})
}

func testValidateBoundariesConcurrency(t *testing.T) {
	t.Run("concurrency too low", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{Concurrency: 0},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency")
	})

	t.Run("concurrency too high", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{Concurrency: 33},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency")
	})

	t.Run("negative concurrency", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{Concurrency: -1},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency")
	})
}

func testValidateBoundariesRetryRegion(t *testing.T) {
	t.Run("retry attempts too high", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: 11,
			},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retry")
	})

	t.Run("negative retry attempts", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: -1,
			},
			RegionSettings: RegionSettings{Primary: "auto"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retry")
	})

	t.Run("invalid region", func(t *testing.T) {
		acw, _ := newTestWizard()
		cfg := &AdvancedConfig{
			UploadSettings: UploadSettings{
				Concurrency:   4,
				RetryAttempts: 3,
			},
			RegionSettings: RegionSettings{Primary: "invalid-region"},
		}
		err := acw.ValidateAdvancedConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid region")
	})
}

// ──────────────────────────────────────────────
// NewAdvancedConfigWizard constructor
// ──────────────────────────────────────────────

// TestNewAdvancedConfigWizard verifies NewAdvancedConfigWizard behavior, one t.Run subtest per...
func TestNewAdvancedConfigWizard(t *testing.T) {
	t.Run("sets profile and default input", func(t *testing.T) {
		profile := &config.Profile{Name: "my-profile"}
		acw := NewAdvancedConfigWizard(profile)
		assert.Equal(t, profile, acw.profile)
		assert.NotNil(t, acw.Input)
	})
}

// ---------------------------------------------------------------------------
// ShowAdvancedConfig full flow (77.8% gap)
// ---------------------------------------------------------------------------

// TestAdvancedConfigWizard_ShowAdvancedConfig_AllDefaults verifies that AdvancedConfigWizard...
func TestAdvancedConfigWizard_ShowAdvancedConfig_AllDefaults(t *testing.T) {
	disableAnimations(t)
	profile := &config.Profile{Name: "test-profile", Description: "test", Region: "auto"}
	acw, _ := newTestWizard(
		"",   // bucket type default (standard)
		"",   // retention default (30)
		"",   // endpoint default (empty)
		"",   // concurrency default (4)
		"",   // chunk size default (8)
		"",   // retries default (3)
		"y",  // checksum enabled
		"",   // region default (auto)
		"",   // theme default (cosmic)
		"n",  // analytics disabled
		"n",  // accessibility disabled
	)
	_ = profile

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg, err := acw.ShowAdvancedConfig()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "standard", cfg.BucketSettings.Type)
	assert.Equal(t, 30, cfg.BucketSettings.RetentionDays)
	assert.Equal(t, 4, cfg.UploadSettings.Concurrency)
	assert.Equal(t, "8MB", cfg.UploadSettings.ChunkSize)
	assert.Equal(t, "auto", cfg.RegionSettings.Primary)
	assert.Equal(t, "cosmic", cfg.Theme)
	assert.True(t, cfg.UploadSettings.ChecksumEnabled)
	assert.False(t, cfg.AnalyticsEnabled)
}

// TestAdvancedConfigWizard_ShowAdvancedConfig_CustomBucket verifies that AdvancedConfigWizard...
func TestAdvancedConfigWizard_ShowAdvancedConfig_CustomBucket(t *testing.T) {
	disableAnimations(t)

	acw, _ := newTestWizard(
		"2",   // performance bucket
		"60",  // 60 days retention
		"",    // no custom endpoint
		"",    // concurrency default
		"16",  // 16MB chunks
		"5",   // 5 retries
		"n",   // no checksum
		"2",   // us-east-1
		"3",   // ocean theme
		"y",   // analytics enabled
		"y",   // accessibility enabled
	)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg, err := acw.ShowAdvancedConfig()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
	assert.Equal(t, "performance", cfg.BucketSettings.Type)
	assert.Equal(t, 60, cfg.BucketSettings.RetentionDays)
	assert.Equal(t, "16MB", cfg.UploadSettings.ChunkSize)
	assert.Equal(t, 5, cfg.UploadSettings.RetryAttempts)
	assert.False(t, cfg.UploadSettings.ChecksumEnabled)
	assert.Equal(t, "us-east-1", cfg.RegionSettings.Primary)
	assert.False(t, cfg.RegionSettings.AutoDetect)
	assert.Equal(t, "ocean", cfg.Theme)
	assert.True(t, cfg.AnalyticsEnabled)
	assert.True(t, cfg.AccessibilityEnabled)
}

// TestAdvancedConfigWizard_ShowAdvancedConfig_InvalidInputs verifies that AdvancedConfigWizard...
func TestAdvancedConfigWizard_ShowAdvancedConfig_InvalidInputs(t *testing.T) {
	disableAnimations(t)

	acw, _ := newTestWizard(
		"9",   // invalid bucket -> warning, uses standard
		"abc", // invalid retention -> warning, uses default
		"",    // endpoint
		"99",  // invalid concurrency (>32) -> warning
		"0",   // invalid chunk size (<1) -> warning
		"-1",  // invalid retries (<0) -> warning
		"y",   // checksum
		"9",   // invalid region -> warning, uses auto
		"9",   // invalid theme -> warning, uses cosmic
		"n",   // analytics
		"n",   // accessibility
	)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg, err := acw.ShowAdvancedConfig()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
	assert.Equal(t, "standard", cfg.BucketSettings.Type)
	assert.Equal(t, 4, cfg.UploadSettings.Concurrency)  // default
	assert.Equal(t, "8MB", cfg.UploadSettings.ChunkSize) // default
	assert.Equal(t, 3, cfg.UploadSettings.RetryAttempts)  // default
	assert.Equal(t, "auto", cfg.RegionSettings.Primary)
	assert.Equal(t, "cosmic", cfg.Theme)
}

// TestAdvancedConfigWizard_ShowAdvancedConfig_CostOptimized verifies that AdvancedConfigWizard...
func TestAdvancedConfigWizard_ShowAdvancedConfig_CostOptimized(t *testing.T) {
	disableAnimations(t)

	acw, _ := newTestWizard(
		"3",  // cost bucket
		"",   // retention
		"",   // endpoint
		"2",  // concurrency 2
		"",   // chunk default
		"",   // retries default
		"y",  // checksum
		"4",  // ap-southeast-1
		"5",  // monochrome
		"n",  // analytics
		"n",  // accessibility
	)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg, err := acw.ShowAdvancedConfig()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	require.NoError(t, err)
	assert.Equal(t, "cost", cfg.BucketSettings.Type)
	assert.Equal(t, 2, cfg.UploadSettings.Concurrency)
	assert.Equal(t, "ap-southeast-1", cfg.RegionSettings.Primary)
	assert.Equal(t, "monochrome", cfg.Theme)
}

// ---------------------------------------------------------------------------
// ShowConfigurationSummary
// ---------------------------------------------------------------------------

// TestAdvancedConfigWizard_ShowConfigurationSummary verifies that AdvancedConfigWizard handles the...
func TestAdvancedConfigWizard_ShowConfigurationSummary(t *testing.T) {
	disableAnimations(t)
	profile := &config.Profile{Name: "summary-test", Description: "test", Region: "auto"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		Profile: profile,
		BucketSettings: BucketSettings{
			Type:          "performance",
			RetentionDays: 60,
		},
		UploadSettings: UploadSettings{
			Concurrency: 8,
			ChunkSize:   "16MB",
		},
		RegionSettings: RegionSettings{
			Primary: "us-east-1",
		},
		Theme:                "ocean",
		AnalyticsEnabled:     true,
		AccessibilityEnabled: false,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	acw.ShowConfigurationSummary(cfg)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "summary-test")
	assert.Contains(t, buf.String(), "performance")
}

// TestAdvancedConfigWizard_ShowConfigurationSummary_WithEndpoint verifies that...
func TestAdvancedConfigWizard_ShowConfigurationSummary_WithEndpoint(t *testing.T) {
	disableAnimations(t)
	profile := &config.Profile{Name: "ep-test", Description: "test", Region: "auto"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		Profile: profile,
		BucketSettings: BucketSettings{
			Type:           "standard",
			CustomEndpoint: "https://custom.r2.cloudflarestorage.com",
		},
		UploadSettings: UploadSettings{
			Concurrency: 4,
			ChunkSize:   "8MB",
		},
		RegionSettings: RegionSettings{
			Primary: "auto",
		},
		Theme: "cosmic",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	acw.ShowConfigurationSummary(cfg)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	assert.Contains(t, buf.String(), "custom.r2")
}

// ---------------------------------------------------------------------------
// ValidateAdvancedConfig
// ---------------------------------------------------------------------------

// TestAdvancedConfigWizard_ValidateAdvancedConfig_Valid verifies that AdvancedConfigWizard handles...
func TestAdvancedConfigWizard_ValidateAdvancedConfig_Valid(t *testing.T) {
	profile := &config.Profile{Name: "test"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		UploadSettings: UploadSettings{
			Concurrency:   4,
			RetryAttempts: 3,
		},
		RegionSettings: RegionSettings{
			Primary: "auto",
		},
	}

	err := acw.ValidateAdvancedConfig(cfg)
	assert.NoError(t, err)
}

// TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidConcurrency verifies that...
func TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidConcurrency(t *testing.T) {
	profile := &config.Profile{Name: "test"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		UploadSettings: UploadSettings{
			Concurrency:   50,
			RetryAttempts: 3,
		},
		RegionSettings: RegionSettings{
			Primary: "auto",
		},
	}

	err := acw.ValidateAdvancedConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency")
}

// TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidRetries verifies that...
func TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidRetries(t *testing.T) {
	profile := &config.Profile{Name: "test"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		UploadSettings: UploadSettings{
			Concurrency:   4,
			RetryAttempts: 20,
		},
		RegionSettings: RegionSettings{
			Primary: "auto",
		},
	}

	err := acw.ValidateAdvancedConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry")
}

// TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidRegion verifies that AdvancedConfigWizard...
func TestAdvancedConfigWizard_ValidateAdvancedConfig_InvalidRegion(t *testing.T) {
	profile := &config.Profile{Name: "test"}
	acw := NewAdvancedConfigWizard(profile)

	cfg := &AdvancedConfig{
		UploadSettings: UploadSettings{
			Concurrency:   4,
			RetryAttempts: 3,
		},
		RegionSettings: RegionSettings{
			Primary: "invalid-region",
		},
	}

	err := acw.ValidateAdvancedConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "region")
}
