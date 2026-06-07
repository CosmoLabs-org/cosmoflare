package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManagerCreateAndGetProfile(t *testing.T) {
	dir := t.TempDir()
	cm := &ConfigManager{
		configPath: filepath.Join(dir, "config.yaml"),
		config: &Config{
			Profiles: make(map[string]*Profile),
			Current:  "default",
		},
		secrets: nil,
	}
	// Use a nil-safe approach: create a ConfigManager via constructor-like setup
	// but with a temp dir so we don't affect real config
	cm.secrets = newTestSecretStore()

	profile := &Profile{
		Name:      "test",
		AccountID: "12345678901234567890123456789012",
		APIToken:  "test-token-abcdef1234567890",
	}

	if err := cm.SetProfile(profile); err != nil {
		t.Fatalf("SetProfile failed: %v", err)
	}

	got, err := cm.GetProfile("test")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if got.AccountID != profile.AccountID {
		t.Errorf("AccountID = %q, want %q", got.AccountID, profile.AccountID)
	}
	if got.APIToken != profile.APIToken {
		t.Errorf("APIToken = %q, want %q", got.APIToken, profile.APIToken)
	}
}

func TestConfigManagerDeleteProfile(t *testing.T) {
	dir := t.TempDir()
	cm := &ConfigManager{
		configPath: filepath.Join(dir, "config.yaml"),
		config: &Config{
			Profiles: make(map[string]*Profile),
			Current:  "default",
		},
		secrets: newTestSecretStore(),
	}

	cm.config.Profiles["other"] = &Profile{Name: "other", AccountID: "12345678901234567890123456789012", APIToken: "tok"}
	cm.config.Profiles["default"] = &Profile{Name: "default", AccountID: "12345678901234567890123456789012", APIToken: "tok"}

	if err := cm.DeleteProfile("other"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	if cm.ProfileExists("other") {
		t.Error("profile should be deleted")
	}
}

func TestConfigManagerDeleteCurrentProfileFails(t *testing.T) {
	cm := &ConfigManager{
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		config: &Config{
			Profiles: map[string]*Profile{
				"active": {Name: "active"},
			},
			Current: "active",
		},
		secrets: newTestSecretStore(),
	}

	err := cm.DeleteProfile("active")
	if err == nil {
		t.Error("expected error when deleting current profile")
	}
}

func TestGetSecretField(t *testing.T) {
	p := &Profile{
		APIToken:  "tok",
		AccessKey: "ak",
		SecretKey: "sk",
	}
	tests := []struct {
		field string
		want  string
	}{
		{"api_token", "tok"},
		{"access_key", "ak"},
		{"secret_key", "sk"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := getSecretField(p, tt.field)
		if got != tt.want {
			t.Errorf("getSecretField(%q) = %q, want %q", tt.field, got, tt.want)
		}
	}
}

func TestSetSecretField(t *testing.T) {
	p := &Profile{}
	setSecretField(p, "api_token", "new-tok")
	setSecretField(p, "access_key", "new-ak")
	setSecretField(p, "secret_key", "new-sk")

	if p.APIToken != "new-tok" {
		t.Errorf("APIToken = %q, want %q", p.APIToken, "new-tok")
	}
	if p.AccessKey != "new-ak" {
		t.Errorf("AccessKey = %q, want %q", p.AccessKey, "new-ak")
	}
	if p.SecretKey != "new-sk" {
		t.Errorf("SecretKey = %q, want %q", p.SecretKey, "new-sk")
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"ab", "**"},
		{"abcd", "****"},
		{"abcdef", "ab**ef"},
		{"1234567890", "12******90"},
	}
	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.want {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	cm := &ConfigManager{
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		config:     &Config{Profiles: make(map[string]*Profile)},
		secrets:    newTestSecretStore(),
	}

	tests := []struct {
		name    string
		profile *Profile
		wantErr bool
	}{
		{"empty name", &Profile{}, true},
		{"missing account", &Profile{Name: "p", APIToken: "long-token-value-1234567890"}, true},
		{"missing token", &Profile{Name: "p", AccountID: "12345678901234567890123456789012"}, true},
		{"short token", &Profile{Name: "p", AccountID: "12345678901234567890123456789012", APIToken: "short"}, true},
		{"bad account length", &Profile{Name: "p", AccountID: "too-short", APIToken: "long-token-value-1234567890"}, true},
		{"valid", &Profile{Name: "p", AccountID: "12345678901234567890123456789012", APIToken: "long-token-value-1234567890"}, false},
	}

	for _, tt := range tests {
		err := cm.ValidateProfile(tt.profile)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: ValidateProfile() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestAutoDetectProfile(t *testing.T) {
	cm := &ConfigManager{
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		config:     &Config{Profiles: make(map[string]*Profile)},
		secrets:    newTestSecretStore(),
	}

	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	if cm.AutoDetectProfile() != nil {
		t.Error("expected nil profile with no env vars")
	}

	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-id")
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-token")

	p := cm.AutoDetectProfile()
	if p == nil {
		t.Fatal("expected auto-detected profile")
	}
	if p.AccountID != "test-id" {
		t.Errorf("AccountID = %q, want %q", p.AccountID, "test-id")
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-id")
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")
	t.Setenv("R2_ENDPOINT", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_REGION", "")

	p := LoadFromEnvironment()
	if p.AccountID != "env-id" {
		t.Errorf("AccountID = %q, want %q", p.AccountID, "env-id")
	}
	if p.APIToken != "env-token" {
		t.Errorf("APIToken = %q, want %q", p.APIToken, "env-token")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".cosmoflare", "config.yaml")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	cm := &ConfigManager{
		configPath: configPath,
		config: &Config{
			Profiles: map[string]*Profile{
				"test": {Name: "test", AccountID: "12345678901234567890123456789012", APIToken: "my-token"},
			},
			Current: "test",
		},
		secrets: newTestSecretStore(),
	}

	if err := cm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cm2 := &ConfigManager{
		configPath: configPath,
		config:     &Config{Profiles: make(map[string]*Profile)},
		secrets:    newTestSecretStore(),
	}
	if err := cm2.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cm2.config.Current != "test" {
		t.Errorf("Current = %q, want %q", cm2.config.Current, "test")
	}
}

func TestExportProfile(t *testing.T) {
	cm := &ConfigManager{
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		config: &Config{
			Profiles: map[string]*Profile{
				"test": {Name: "test", AccountID: "id123", APIToken: "tok456", Endpoint: "https://ep"},
			},
		},
		secrets: newTestSecretStore(),
	}

	out, err := cm.ExportProfile("test")
	if err != nil {
		t.Fatalf("ExportProfile failed: %v", err)
	}
	if !contains(out, "CLOUDFLARE_API_TOKEN") {
		t.Error("export should contain CLOUDFLARE_API_TOKEN")
	}
	if !contains(out, "R2_ENDPOINT") {
		t.Error("export should contain R2_ENDPOINT")
	}
}

func TestJSON(t *testing.T) {
	cm := &ConfigManager{
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		config: &Config{
			Profiles: map[string]*Profile{
				"p": {Name: "p", AccountID: "abc"},
			},
			Current: "p",
		},
		secrets: newTestSecretStore(),
	}

	out, err := cm.JSON()
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}
	if !contains(out, `"current"`) {
		t.Error("JSON should contain current field")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// newTestSecretStore returns a Secrets implementation that won't touch the real keychain.
func newTestSecretStore() Secrets {
	return &noopSecrets{}
}

type noopSecrets struct{}

func (n *noopSecrets) Available() bool                     { return false }
func (n *noopSecrets) Get(_, _ string) (string, error)     { return "", nil }
func (n *noopSecrets) Set(_, _, _ string) error            { return nil }
func (n *noopSecrets) Delete(_, _ string) error            { return nil }
