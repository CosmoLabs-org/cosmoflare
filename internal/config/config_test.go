package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// newTestConfigManagerAt returns a ConfigManager that persists its config file
// at path and uses a no-op secret store, so tests never touch the user's real
// config file or OS keychain.
func newTestConfigManagerAt(t *testing.T, path string) *ConfigManager {
	t.Helper()
	return &ConfigManager{
		configPath: path,
		config: &Config{
			Profiles: make(map[string]*Profile),
			Current:  "default",
		},
		secrets: newTestSecretStore(),
	}
}

// newTestConfigManager returns a ConfigManager rooted in a per-test temp dir,
// ensuring each test writes to its own isolated config file.
func newTestConfigManager(t *testing.T) *ConfigManager {
	t.Helper()
	return newTestConfigManagerAt(t, filepath.Join(t.TempDir(), "config.yaml"))
}

// TestConfigManagerCreateAndGetProfile verifies that a profile added via
// SetProfile round-trips through GetProfile with its credentials intact.
func TestConfigManagerCreateAndGetProfile(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)

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

// TestConfigManagerDeleteProfile verifies that DeleteProfile removes a
// non-current profile from the config.
func TestConfigManagerDeleteProfile(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)

	// Seed two profiles: "default" remains current, "other" is deletable.
	for _, name := range []string{"default", "other"} {
		cm.config.Profiles[name] = &Profile{
			Name:      name,
			AccountID: "12345678901234567890123456789012",
			APIToken:  "tok",
		}
	}

	if err := cm.DeleteProfile("other"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	if cm.ProfileExists("other") {
		t.Error("profile should be deleted")
	}
}

// TestConfigManagerDeleteCurrentProfileFails verifies that DeleteProfile
// refuses to remove the profile currently selected as Current, since removing
// it would leave the config pointing at a missing profile.
func TestConfigManagerDeleteCurrentProfileFails(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	cm.config.Profiles["active"] = &Profile{Name: "active"}
	cm.config.Current = "active"

	err := cm.DeleteProfile("active")
	if err == nil {
		t.Error("expected error when deleting current profile")
	}
}

// TestGetSecretField verifies the field-name-to-Profile-attribute mapping
// used by the keychain layer, including that unknown field names yield an
// empty value rather than an error.
func TestGetSecretField(t *testing.T) {
	t.Parallel()
	p := &Profile{
		APIToken:  "tok",
		AccessKey: "ak",
		SecretKey: "sk",
	}
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"api token", "api_token", "tok"},
		{"access key", "access_key", "ak"},
		{"secret key", "secret_key", "sk"},
		{"unknown field yields empty string", "unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getSecretField(p, tt.field)
			if got != tt.want {
				t.Errorf("getSecretField(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

// TestSetSecretField verifies that each supported field name writes to the
// correct Profile attribute.
func TestSetSecretField(t *testing.T) {
	t.Parallel()
	p := &Profile{}
	setSecretField(p, "api_token", "new-tok")
	setSecretField(p, "access_key", "new-ak")
	setSecretField(p, "secret_key", "new-sk")

	fields := []struct {
		name string
		got  string
		want string
	}{
		{"api token", p.APIToken, "new-tok"},
		{"access key", p.AccessKey, "new-ak"},
		{"secret key", p.SecretKey, "new-sk"},
	}
	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			if f.got != f.want {
				t.Errorf("%s = %q, want %q", f.name, f.got, f.want)
			}
		})
	}
}

// TestMaskKey verifies secret masking for display: empty keys stay empty,
// keys of four characters or fewer are fully redacted, and longer keys expose
// only their first two and last two characters.
func TestMaskKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty key stays empty", "", ""},
		{"two char key fully masked", "ab", "**"},
		{"four char key fully masked", "abcd", "****"},
		{"six char key keeps first and last two", "abcdef", "ab**ef"},
		{"ten char key keeps first and last two", "1234567890", "12******90"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := maskKey(tt.input)
			if got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestValidateProfile verifies that profile validation rejects missing or
// malformed credentials (name, account ID length, token length) and accepts a
// fully valid profile.
func TestValidateProfile(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := cm.ValidateProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAutoDetectProfile verifies environment-based profile detection: with no
// environment credentials set no profile is returned, and once both the
// account ID and API token are present a profile populated from the
// environment is returned.
func TestAutoDetectProfile(t *testing.T) {
	// Not parallel: t.Setenv mutates the process-wide environment.
	cm := newTestConfigManager(t)

	t.Run("no env vars returns nil", func(t *testing.T) {
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
		t.Setenv("CLOUDFLARE_API_TOKEN", "")

		if cm.AutoDetectProfile() != nil {
			t.Error("expected nil profile with no env vars")
		}
	})

	t.Run("account and token set returns profile", func(t *testing.T) {
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-id")
		t.Setenv("CLOUDFLARE_API_TOKEN", "test-token")

		p := cm.AutoDetectProfile()
		if p == nil {
			t.Fatal("expected auto-detected profile")
		}
		if p.AccountID != "test-id" {
			t.Errorf("AccountID = %q, want %q", p.AccountID, "test-id")
		}
	})
}

// TestLoadFromEnvironment verifies that LoadFromEnvironment copies the
// essential Cloudflare credentials from the process environment onto a Profile.
func TestLoadFromEnvironment(t *testing.T) {
	// Not parallel: t.Setenv mutates the process-wide environment.
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-id")
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")
	// Optional fields are cleared so they do not leak from the host env.
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

// TestSaveAndLoad verifies that a config persisted with Save can be reloaded
// by a fresh ConfigManager via load, and that Save creates the config file's
// parent directory when it does not yet exist.
func TestSaveAndLoad(t *testing.T) {
	t.Parallel()
	// Nested path: the ".cosmoflare" parent directory does not exist yet, so
	// Save must create it itself.
	configPath := filepath.Join(t.TempDir(), ".cosmoflare", "config.yaml")

	cm := newTestConfigManagerAt(t, configPath)
	cm.config.Profiles["test"] = &Profile{
		Name:      "test",
		AccountID: "12345678901234567890123456789012",
		APIToken:  "my-token",
	}
	cm.config.Current = "test"

	if err := cm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cm2 := newTestConfigManagerAt(t, configPath)
	if err := cm2.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cm2.config.Current != "test" {
		t.Errorf("Current = %q, want %q", cm2.config.Current, "test")
	}
}

// TestExportProfile verifies that ExportProfile renders the required
// Cloudflare variables plus the optional R2 endpoint variable when one is
// configured on the profile.
func TestExportProfile(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	cm.config.Profiles["test"] = &Profile{
		Name:      "test",
		AccountID: "id123",
		APIToken:  "tok456",
		Endpoint:  "https://ep",
	}

	out, err := cm.ExportProfile("test")
	if err != nil {
		t.Fatalf("ExportProfile failed: %v", err)
	}
	for _, varName := range []string{"CLOUDFLARE_API_TOKEN", "R2_ENDPOINT"} {
		t.Run("exports "+varName, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(out, varName) {
				t.Errorf("export should contain %s", varName)
			}
		})
	}
}

// TestJSON verifies that JSON serialization of the config includes the
// current-profile marker field.
func TestJSON(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	cm.config.Profiles["p"] = &Profile{Name: "p", AccountID: "abc"}
	cm.config.Current = "p"

	out, err := cm.JSON()
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}
	if !strings.Contains(out, `"current"`) {
		t.Error(`JSON should contain "current" field`)
	}
}

// newTestSecretStore returns a Secrets implementation that won't touch the real keychain.
func newTestSecretStore() Secrets {
	return &noopSecrets{}
}

type noopSecrets struct{}

func (n *noopSecrets) Available() bool                 { return false }
func (n *noopSecrets) Get(_, _ string) (string, error) { return "", nil }
func (n *noopSecrets) Set(_, _, _ string) error        { return nil }
func (n *noopSecrets) Delete(_, _ string) error        { return nil }
