package cosmoflare

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ProjectConfig represents a .r2go2.yaml project-level configuration.
type ProjectConfig struct {
	Bucket      string            `mapstructure:"bucket" json:"bucket"`
	Region      string            `mapstructure:"region" json:"region,omitempty"`
	Endpoint    string            `mapstructure:"endpoint" json:"endpoint,omitempty"`
	CachePolicy CachePolicyConfig `mapstructure:"cache" json:"cache,omitempty"`
	Guardrails  GuardrailConfig   `mapstructure:"guardrails" json:"guardrails,omitempty"`
	Audit       AuditConfig       `mapstructure:"audit" json:"audit,omitempty"`
	AllowedBuckets []string       `mapstructure:"allowed_buckets" json:"allowed_buckets,omitempty"`
	MaxFileSize   int64           `mapstructure:"max_file_size" json:"max_file_size,omitempty"`
	Environment   string          `mapstructure:"env" json:"env,omitempty"`
}

// CachePolicyConfig configures automatic cache header injection.
type CachePolicyConfig struct {
	Enabled bool              `mapstructure:"enabled" json:"enabled"`
	Rules   map[string]string `mapstructure:"rules" json:"rules,omitempty"`
	Default string            `mapstructure:"default" json:"default,omitempty"`
}

// GuardrailConfig configures upload validation and access scoping.
type GuardrailConfig struct {
	Enabled        bool     `mapstructure:"enabled" json:"enabled"`
	MaxFileSize    int64    `mapstructure:"max_file_size" json:"max_file_size,omitempty"`
	AllowedBuckets []string `mapstructure:"allowed_buckets" json:"allowed_buckets,omitempty"`
	BlockedKeys    []string `mapstructure:"blocked_keys" json:"blocked_keys,omitempty"`
	RequireTags    []string `mapstructure:"require_tags" json:"require_tags,omitempty"`
}

// AuditConfig configures JSONL audit logging.
type AuditConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Path    string `mapstructure:"path" json:"path,omitempty"`
	Fields  []string `mapstructure:"fields" json:"fields,omitempty"`
}

// MachineConfig represents ~/.r2go2/ user-level configuration.
type MachineConfig struct {
	Profiles map[string]*ProfileConfig `mapstructure:"profiles" json:"profiles"`
	Current  string                    `mapstructure:"current" json:"current"`
}

// ProfileConfig represents a single named profile in the machine config.
type ProfileConfig struct {
	AccountID   string `mapstructure:"account_id" json:"account_id"`
	APIToken    string `mapstructure:"api_token" json:"api_token"`
	AccessKey   string `mapstructure:"access_key,omitempty" json:"access_key,omitempty"`
	SecretKey   string `mapstructure:"secret_key,omitempty" json:"secret_key,omitempty"`
	Endpoint    string `mapstructure:"endpoint,omitempty" json:"endpoint,omitempty"`
	Region      string `mapstructure:"region,omitempty" json:"region,omitempty"`
	Description string `mapstructure:"description,omitempty" json:"description,omitempty"`
}

// LoadProjectConfig loads project configuration from the given directory,
// searching upward through parent directories. The canonical filename is
// .cosmoflare.yaml; the legacy .r2go2.yaml is still accepted when no
// .cosmoflare.yaml is found.
func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	path, err := findProjectFile(dir, ".cosmoflare.yaml")
	if err != nil {
		path, err = findProjectFile(dir, ".r2go2.yaml")
		if err != nil {
			return nil, err
		}
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	var cfg ProjectConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project config: %w", err)
	}

	return &cfg, nil
}

// LoadMachineConfig loads configuration from ~/.r2go2/config.yaml.
func LoadMachineConfig() (*MachineConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".r2go2", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &MachineConfig{
			Profiles: make(map[string]*ProfileConfig),
			Current:  "default",
		}, nil
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read machine config: %w", err)
	}

	var cfg MachineConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse machine config: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*ProfileConfig)
	}

	return &cfg, nil
}

// GetProfile returns a named profile from the machine config.
func (mc *MachineConfig) GetProfile(name string) (*ProfileConfig, error) {
	p, ok := mc.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// CurrentProfile returns the currently active profile.
func (mc *MachineConfig) CurrentProfile() (*ProfileConfig, error) {
	if mc.Current == "" {
		return nil, fmt.Errorf("no current profile set")
	}
	return mc.GetProfile(mc.Current)
}

// Save writes the machine config to disk.
func (mc *MachineConfig) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".r2go2")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	v := viper.New()
	v.Set("profiles", mc.Profiles)
	v.Set("current", mc.Current)

	configPath := filepath.Join(configDir, "config.yaml")
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write machine config: %w", err)
	}

	return os.Chmod(configPath, 0600)
}

// ProfileFromEnv creates a ProfileConfig from environment variables.
func ProfileFromEnv() *ProfileConfig {
	return &ProfileConfig{
		AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		APIToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Endpoint:  os.Getenv("R2_ENDPOINT"),
		Region:    os.Getenv("AWS_REGION"),
	}
}

// findProjectFile walks up from dir looking for filename.
func findProjectFile(dir, filename string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(abs, filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("%s not found in any parent directory", filename)
		}
		abs = parent
	}
}
