/*
Package interactive provides advanced configuration wizard functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
)

// AdvancedConfig provides advanced configuration options
type AdvancedConfig struct {
	Profile              *config.Profile
	BucketSettings       BucketSettings
	UploadSettings       UploadSettings
	RegionSettings       RegionSettings
	AnalyticsEnabled     bool
	Theme                string
	AccessibilityEnabled bool
}

// BucketSettings defines bucket configuration
type BucketSettings struct {
	Type           string // standard, performance, cost
	DefaultRegion  string
	CustomEndpoint string
	RetentionDays  int
}

// UploadSettings defines upload preferences
type UploadSettings struct {
	Concurrency     int
	ChunkSize       string // MB
	RetryAttempts   int
	Timeout         time.Duration
	ChecksumEnabled bool
}

// RegionSettings defines region configuration
type RegionSettings struct {
	AutoDetect bool
	Primary    string
	Backup     string
}

// AdvancedConfigWizard handles advanced configuration
type AdvancedConfigWizard struct {
	profile *config.Profile
}

// NewAdvancedConfigWizard creates a new advanced config wizard
func NewAdvancedConfigWizard(profile *config.Profile) *AdvancedConfigWizard {
	return &AdvancedConfigWizard{
		profile: profile,
	}
}

// ShowAdvancedConfig displays the advanced configuration interface
func (acw *AdvancedConfigWizard) ShowAdvancedConfig() (*AdvancedConfig, error) {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🔧 Advanced Configuration"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	fmt.Println(Muted("Configure advanced settings for your profile."))
	fmt.Println(Muted("Press Enter to accept defaults."))
	fmt.Println()

	config := &AdvancedConfig{
		Profile: acw.profile,
		BucketSettings: BucketSettings{
			Type:           "standard",
			DefaultRegion:  "auto",
			CustomEndpoint: "",
			RetentionDays:  30,
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
			Backup:     "",
		},
		AnalyticsEnabled:     false,
		Theme:                "cosmic",
		AccessibilityEnabled: false,
	}

	// Step 1: Bucket Settings
	if err := acw.configureBucketSettings(&config.BucketSettings); err != nil {
		return nil, err
	}

	// Step 2: Upload Preferences
	if err := acw.configureUploadSettings(&config.UploadSettings); err != nil {
		return nil, err
	}

	// Step 3: Region Settings
	if err := acw.configureRegionSettings(&config.RegionSettings); err != nil {
		return nil, err
	}

	// Step 4: Additional Options
	if err := acw.configureAdditionalOptions(config); err != nil {
		return nil, err
	}

	return config, nil
}

// configureBucketSettings configures bucket settings
func (acw *AdvancedConfigWizard) configureBucketSettings(settings *BucketSettings) error {
	fmt.Println(Bold("📁 Default Bucket Settings"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	fmt.Println("Bucket Type Presets:")
	fmt.Println("  [1] Standard (default)      - General purpose storage")
	fmt.Println("  [2] High Performance         - Multiple regions, faster access")
	fmt.Println("  [3] Cost Optimized          - Single region, lower cost")
	fmt.Println()

	fmt.Printf("Select bucket type [1]: ")
	var bucketType string
	fmt.Scanln(&bucketType)
	bucketType = strings.TrimSpace(bucketType)

	switch bucketType {
	case "1", "":
		settings.Type = "standard"
	case "2":
		settings.Type = "performance"
	case "3":
		settings.Type = "cost"
	default:
		PrintWarning("Invalid selection. Using standard.")
		settings.Type = "standard"
	}

	fmt.Println()
	fmt.Printf("Default retention period in days [30]: ")
	var retention string
	fmt.Scanln(&retention)
	retention = strings.TrimSpace(retention)

	if retention != "" {
		if days, err := strconv.Atoi(retention); err == nil && days >= 0 {
			settings.RetentionDays = days
		} else {
			PrintWarning("Invalid retention days. Using default.")
		}
	}

	fmt.Println()
	fmt.Printf("Custom R2 endpoint (optional) []: ")
	var endpoint string
	fmt.Scanln(&endpoint)
	settings.CustomEndpoint = strings.TrimSpace(endpoint)

	fmt.Println()
	PrintSuccess("Bucket settings configured")
	fmt.Println()

	return nil
}

// configureUploadSettings configures upload preferences
func (acw *AdvancedConfigWizard) configureUploadSettings(settings *UploadSettings) error {
	fmt.Println(Bold("🚀 Upload Preferences"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	fmt.Printf("Concurrent uploads [4]: ")
	var concurrency string
	fmt.Scanln(&concurrency)
	concurrency = strings.TrimSpace(concurrency)

	if concurrency != "" {
		if count, err := strconv.Atoi(concurrency); err == nil && count > 0 && count <= 32 {
			settings.Concurrency = count
		} else {
			PrintWarning("Invalid concurrency. Using default (4).")
		}
	}

	fmt.Printf("Upload chunk size (MB) [8]: ")
	var chunkSize string
	fmt.Scanln(&chunkSize)
	chunkSize = strings.TrimSpace(chunkSize)

	if chunkSize != "" {
		if size, err := strconv.Atoi(chunkSize); err == nil && size >= 1 && size <= 100 {
			settings.ChunkSize = fmt.Sprintf("%dMB", size)
		} else {
			PrintWarning("Invalid chunk size. Using default (8MB).")
		}
	}

	fmt.Printf("Retry attempts [3]: ")
	var retries string
	fmt.Scanln(&retries)
	retries = strings.TrimSpace(retries)

	if retries != "" {
		if count, err := strconv.Atoi(retries); err == nil && count >= 0 && count <= 10 {
			settings.RetryAttempts = count
		} else {
			PrintWarning("Invalid retry attempts. Using default (3).")
		}
	}

	settings.ChecksumEnabled = ConfirmYesNo("Enable checksum verification for uploads?", true)

	fmt.Println()
	PrintSuccess("Upload preferences configured")
	fmt.Println()

	return nil
}

// configureRegionSettings configures region settings
func (acw *AdvancedConfigWizard) configureRegionSettings(settings *RegionSettings) error {
	fmt.Println(Bold("🌍 Region Selection"))
	fmt.Println(strings.Repeat("─", 20))
	fmt.Println()

	fmt.Println("Available Regions:")
	fmt.Println("  [1] Auto-detect (recommended) - Choose closest region")
	fmt.Println("  [2] us-east-1               - US East Coast")
	fmt.Println("  [3] eu-west-1               - Europe West")
	fmt.Println("  [4] ap-southeast-1          - Asia Pacific")
	fmt.Println()

	fmt.Printf("Select region [1]: ")
	var region string
	fmt.Scanln(&region)
	region = strings.TrimSpace(region)

	switch region {
	case "1", "":
		settings.AutoDetect = true
		settings.Primary = "auto"
	case "2":
		settings.AutoDetect = false
		settings.Primary = "us-east-1"
	case "3":
		settings.AutoDetect = false
		settings.Primary = "eu-west-1"
	case "4":
		settings.AutoDetect = false
		settings.Primary = "ap-southeast-1"
	default:
		PrintWarning("Invalid selection. Using auto-detect.")
		settings.AutoDetect = true
		settings.Primary = "auto"
	}

	fmt.Println()
	PrintSuccess("Region settings configured")
	fmt.Println()

	return nil
}

// configureAdditionalOptions configures additional options
func (acw *AdvancedConfigWizard) configureAdditionalOptions(config *AdvancedConfig) error {
	fmt.Println(Bold("⚙️  Additional Options"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	fmt.Println("Available Themes:")
	fmt.Println("  [1] Cosmic (default)    🌌 Purple and blue gradients")
	fmt.Println("  [2] Forest              🌲 Green and earth tones")
	fmt.Println("  [3] Ocean               🌊 Deep blues and teals")
	fmt.Println("  [4] Sunset              🌅 Warm oranges and reds")
	fmt.Println("  [5] Monochrome          ⚪ Black and white only")
	fmt.Println()

	fmt.Printf("Select theme [1]: ")
	var theme string
	fmt.Scanln(&theme)
	theme = strings.TrimSpace(theme)

	switch theme {
	case "1", "":
		config.Theme = "cosmic"
	case "2":
		config.Theme = "forest"
	case "3":
		config.Theme = "ocean"
	case "4":
		config.Theme = "sunset"
	case "5":
		config.Theme = "monochrome"
	default:
		PrintWarning("Invalid theme selection. Using cosmic.")
		config.Theme = "cosmic"
	}

	fmt.Println()
	config.AnalyticsEnabled = ConfirmYesNo("Enable anonymous usage reports? (helps improve R2Go2)", false)
	config.AccessibilityEnabled = ConfirmYesNo("Enable accessibility features?", false)

	fmt.Println()
	PrintSuccess("Advanced configuration completed!")
	fmt.Println()

	return nil
}

// ShowConfigurationSummary displays a summary of the configuration
func (acw *AdvancedConfigWizard) ShowConfigurationSummary(config *AdvancedConfig) {
	fmt.Println(Bold("📋 Configuration Summary"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	fmt.Printf("%s Profile:%s      %s\n", Bold, Reset, config.Profile.Name)
	fmt.Printf("%s Bucket Type:%s  %s\n", Bold, Reset, config.BucketSettings.Type)
	fmt.Printf("%s Region:%s       %s\n", Bold, Reset, config.RegionSettings.Primary)
	fmt.Printf("%s Uploads:%s      %s concurrent, %s chunks\n",
		Bold, Reset,
		fmt.Sprintf("%d", config.UploadSettings.Concurrency),
		config.UploadSettings.ChunkSize)
	fmt.Printf("%s Theme:%s        %s\n", Bold, Reset, config.Theme)
	fmt.Printf("%s Analytics:%s    %s\n", Bold, Reset, formatBool(config.AnalyticsEnabled))
	fmt.Printf("%s Accessibility:%s %s\n", Bold, Reset, formatBool(config.AccessibilityEnabled))

	if config.BucketSettings.RetentionDays > 0 {
		fmt.Printf("%s Retention:%s    %s days\n", Bold, Reset, fmt.Sprintf("%d", config.BucketSettings.RetentionDays))
	}

	if config.BucketSettings.CustomEndpoint != "" {
		fmt.Printf("%s Endpoint:%s     %s\n", Bold, Reset, config.BucketSettings.CustomEndpoint)
	}

	fmt.Println()
}

// SaveAdvancedConfig saves the advanced configuration to the profile
func (acw *AdvancedConfigWizard) SaveAdvancedConfig(config *AdvancedConfig, configMgr *config.ConfigManager) error {
	// Extend the profile with advanced settings
	// This would require extending the Profile struct to include advanced settings
	// For now, we'll store them in metadata
	profile := config.Profile

	// Store metadata in profile description for now (in a real implementation,
	// you'd extend the Profile struct to have a Metadata field and store:
	// - bucket_type, retention_days, custom_endpoint
	// - upload_concurrency, upload_chunk_size, retry_attempts, checksum_enabled
	// - region, auto_detect_region, theme, analytics_enabled, accessibility
	// For now, we just add a note to the description)
	profile.Description = fmt.Sprintf("%s [Advanced: %s]",
		profile.Description, config.Theme)

	return configMgr.SetProfile(profile)
}

// Helper functions

// ValidateAdvancedConfig validates the advanced configuration
func (acw *AdvancedConfigWizard) ValidateAdvancedConfig(config *AdvancedConfig) error {
	// Validate upload settings
	if config.UploadSettings.Concurrency <= 0 || config.UploadSettings.Concurrency > 32 {
		return fmt.Errorf("concurrency must be between 1 and 32")
	}

	// Validate upload settings
	if config.UploadSettings.RetryAttempts < 0 || config.UploadSettings.RetryAttempts > 10 {
		return fmt.Errorf("retry attempts must be between 0 and 10")
	}

	// Validate region
	validRegions := []string{"auto", "us-east-1", "eu-west-1", "ap-southeast-1"}
	validRegion := false
	for _, region := range validRegions {
		if config.RegionSettings.Primary == region {
			validRegion = true
			break
		}
	}
	if !validRegion {
		return fmt.Errorf("invalid region: %s", config.RegionSettings.Primary)
	}

	return nil
}