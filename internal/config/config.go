/*
Package config provides configuration management for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Profiles map[string]*Profile `json:"profiles" yaml:"profiles"`
	Current  string              `json:"current" yaml:"current"`
}

// Profile represents a Cloudflare profile configuration
type Profile struct {
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	AccountID   string `json:"account_id" yaml:"account_id" mapstructure:"account_id"`
	APIToken    string `json:"api_token" yaml:"api_token" mapstructure:"api_token"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`
	// S3 compatible endpoint configuration
	Endpoint  string `json:"endpoint,omitempty" yaml:"endpoint,omitempty" mapstructure:"endpoint,omitempty"`
	AccessKey string `json:"access_key,omitempty" yaml:"access_key,omitempty" mapstructure:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty" yaml:"secret_key,omitempty" mapstructure:"secret_key,omitempty"`
	Region    string `json:"region,omitempty" yaml:"region,omitempty" mapstructure:"region,omitempty"`
}

// ConfigManager manages configuration operations
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".cosmoflare")
	configPath := filepath.Join(configDir, "config.yaml")

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cm := &ConfigManager{
		configPath: configPath,
		config: &Config{
			Profiles: make(map[string]*Profile),
			Current:  "default",
		},
	}

	// Load existing configuration if it exists
	if err := cm.load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cm, nil
}

// Load loads configuration from file
func (cm *ConfigManager) load() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Config doesn't exist, use defaults
		return nil
	}

	// Use viper for flexible config loading
	v := viper.New()
	v.SetConfigFile(cm.configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cm.config = &config
	return nil
}

// Save saves configuration to file
func (cm *ConfigManager) Save() error {
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tmpPath := cm.configPath + "~"
	data, err := yaml.Marshal(map[string]interface{}{
		"profiles": cm.config.Profiles,
		"current":  cm.config.Current,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, cm.configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize config file: %w", err)
	}

	return nil
}

// GetProfile returns a profile by name
func (cm *ConfigManager) GetProfile(name string) (*Profile, error) {
	if profile, exists := cm.config.Profiles[name]; exists {
		return profile, nil
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

// SetProfile adds or updates a profile
func (cm *ConfigManager) SetProfile(profile *Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	cm.config.Profiles[profile.Name] = profile
	return cm.Save()
}

// DeleteProfile removes a profile
func (cm *ConfigManager) DeleteProfile(name string) error {
	if !cm.ProfileExists(name) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// Don't allow deleting the current profile
	if cm.config.Current == name {
		return fmt.Errorf("cannot delete current profile '%s'", name)
	}

	delete(cm.config.Profiles, name)
	return cm.Save()
}

// ListProfiles returns all profile names
func (cm *ConfigManager) ListProfiles() []string {
	var profiles []string
	for name := range cm.config.Profiles {
		profiles = append(profiles, name)
	}
	return profiles
}

// ProfileExists checks if a profile exists
func (cm *ConfigManager) ProfileExists(name string) bool {
	_, exists := cm.config.Profiles[name]
	return exists
}

// SetCurrent sets the current profile
func (cm *ConfigManager) SetCurrent(name string) error {
	if !cm.ProfileExists(name) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	cm.config.Current = name
	return cm.Save()
}

// GetCurrent returns the current profile
func (cm *ConfigManager) GetCurrent() (*Profile, error) {
	if cm.config.Current == "" {
		return nil, fmt.Errorf("no current profile set")
	}
	return cm.GetProfile(cm.config.Current)
}

// ValidateProfile validates a profile configuration
func (cm *ConfigManager) ValidateProfile(profile *Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if profile.AccountID == "" {
		return fmt.Errorf("account ID is required")
	}
	if profile.APIToken == "" {
		return fmt.Errorf("API token is required")
	}

	// Basic token format validation
	if len(profile.APIToken) < 10 {
		return fmt.Errorf("API token appears to be invalid (too short)")
	}

	// Account ID format validation (should be 32 character hex string)
	if len(profile.AccountID) != 32 {
		return fmt.Errorf("account ID should be 32 characters long")
	}

	return nil
}

// GetConfigPath returns the config file path
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// ExportProfile exports a profile as environment variables
func (cm *ConfigManager) ExportProfile(name string) (string, error) {
	profile, err := cm.GetProfile(name)
	if err != nil {
		return "", err
	}

	var export strings.Builder
	export.WriteString(fmt.Sprintf("export CLOUDFLARE_API_TOKEN=\"%s\"\n", profile.APIToken))
	export.WriteString(fmt.Sprintf("export CLOUDFLARE_ACCOUNT_ID=\"%s\"\n", profile.AccountID))

	if profile.Endpoint != "" {
		export.WriteString(fmt.Sprintf("export R2_ENDPOINT=\"%s\"\n", profile.Endpoint))
	}
	if profile.AccessKey != "" {
		export.WriteString(fmt.Sprintf("export AWS_ACCESS_KEY_ID=\"%s\"\n", profile.AccessKey))
	}
	if profile.SecretKey != "" {
		export.WriteString(fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=\"%s\"\n", profile.SecretKey))
	}
	if profile.Region != "" {
		export.WriteString(fmt.Sprintf("export AWS_REGION=\"%s\"\n", profile.Region))
	}

	return export.String(), nil
}

// JSON returns the configuration as JSON
func (cm *ConfigManager) JSON() (string, error) {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	return string(data), nil
}

// AutoDetectProfile tries to auto-detect configuration from environment
func (cm *ConfigManager) AutoDetectProfile() *Profile {
	profile := &Profile{
		Name:       "auto-detected",
		AccountID:  os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		APIToken:   os.Getenv("CLOUDFLARE_API_TOKEN"),
		Endpoint:   os.Getenv("R2_ENDPOINT"),
		AccessKey:  os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Region:     os.Getenv("AWS_REGION"),
	}

	// Only return profile if we have the essentials
	if profile.AccountID != "" && profile.APIToken != "" {
		return profile
	}

	return nil
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() *Profile {
	return &Profile{
		AccountID:  os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		APIToken:   os.Getenv("CLOUDFLARE_API_TOKEN"),
		Endpoint:   os.Getenv("R2_ENDPOINT"),
		AccessKey:  os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Region:     os.Getenv("AWS_REGION"),
	}
}

// SanitizeForOutput returns a sanitized version of the config for output
func (cm *ConfigManager) SanitizeForOutput() *Config {
	sanitized := &Config{
		Current: cm.config.Current,
		Profiles: make(map[string]*Profile),
	}

	for name, profile := range cm.config.Profiles {
		sanitizedProfile := &Profile{
			Name:        profile.Name,
			AccountID:   utils.MaskAccountID(profile.AccountID),
			Description: profile.Description,
			Endpoint:    profile.Endpoint,
			AccessKey:   maskKey(profile.AccessKey),
			// Never include secret keys or API tokens in output
			Region: profile.Region,
		}
		sanitized.Profiles[name] = sanitizedProfile
	}

	return sanitized
}

// maskKey masks a secret key for display
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return key[:2] + strings.Repeat("*", len(key)-4) + key[len(key)-2:]
}

// MaskAccountID exports the masking function for use in other packages
func MaskAccountID(accountID string) string {
	return utils.MaskAccountID(accountID)
}

// MaskKey exports the maskKey function for use in other packages
func MaskKey(key string) string {
	return maskKey(key)
}