package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// newTestConfigManagerWithSecrets returns a ConfigManager rooted in a per-test
// temp dir that uses the supplied secret store, so keychain-backed code paths
// can be exercised without touching the OS keychain.
func newTestConfigManagerWithSecrets(t *testing.T, secrets Secrets) *ConfigManager {
	t.Helper()
	cm := newTestConfigManager(t)
	cm.secrets = secrets
	return cm
}

// memSecrets is an in-memory Secrets implementation that reports Available
// and can be scripted to fail Set calls, letting tests exercise both the
// keychain-enabled and error branches of the secret storage layer.
type memSecrets struct {
	mu     sync.Mutex
	store  map[string]string
	setErr error
}

func newMemSecrets() *memSecrets {
	return &memSecrets{store: make(map[string]string)}
}

func (m *memSecrets) key(profile, field string) string {
	return profile + "\x00" + field
}

func (m *memSecrets) Available() bool { return true }

func (m *memSecrets) Get(profile, field string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.store[m.key(profile, field)]
	if !ok {
		return "", fmt.Errorf("secret not found")
	}
	return v, nil
}

func (m *memSecrets) Set(profile, field, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.store[m.key(profile, field)] = value
	return nil
}

func (m *memSecrets) Delete(profile, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, m.key(profile, field))
	return nil
}

// TestNewConfigManager verifies that NewConfigManager creates the config
// directory under the user's home directory, seeds the "default" current
// profile marker, and reports the expected config file path.
func TestNewConfigManager(t *testing.T) {
	// Not parallel: t.Setenv mutates the process-wide environment.
	home := t.TempDir()
	t.Setenv("HOME", home)

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	wantPath := filepath.Join(home, ".cosmoflare", "config.yaml")
	if cm.GetConfigPath() != wantPath {
		t.Errorf("GetConfigPath() = %q, want %q", cm.GetConfigPath(), wantPath)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare")); err != nil {
		t.Errorf("config directory should have been created: %v", err)
	}
	if cm.config.Current != "default" {
		t.Errorf("Current = %q, want %q", cm.config.Current, "default")
	}
	if got := cm.ListProfiles(); len(got) != 0 {
		t.Errorf("ListProfiles() = %v, want empty for fresh config", got)
	}
	if cm.KeychainAvailable() {
		t.Error("KeychainAvailable should be false under test (in-memory keychain)")
	}
}

// TestGetProfileRaw verifies that GetProfileRaw returns the stored profile
// without hydrating keychain sentinels, and errors for unknown profiles.
func TestGetProfileRaw(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	stored := &Profile{
		Name:      "test",
		AccountID: "12345678901234567890123456789012",
		APIToken:  keychainSentinel,
	}
	cm.config.Profiles["test"] = stored

	t.Run("existing profile returned unhydrated", func(t *testing.T) {
		t.Parallel()
		got, err := cm.GetProfileRaw("test")
		if err != nil {
			t.Fatalf("GetProfileRaw failed: %v", err)
		}
		if got != stored {
			t.Error("GetProfileRaw should return the stored profile pointer")
		}
		if got.APIToken != keychainSentinel {
			t.Errorf("APIToken = %q, want sentinel %q (raw must not hydrate)", got.APIToken, keychainSentinel)
		}
	})

	t.Run("missing profile errors", func(t *testing.T) {
		t.Parallel()
		_, err := cm.GetProfileRaw("nope")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("GetProfileRaw(nope) error = %v, want 'not found'", err)
		}
	})
}

// TestGetProfileNotFound verifies that GetProfile errors for a profile that
// does not exist in the config.
func TestGetProfileNotFound(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	_, err := cm.GetProfile("missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("GetProfile(missing) error = %v, want 'not found'", err)
	}
}

// TestListProfiles verifies that ListProfiles returns every stored profile
// name regardless of map iteration order.
func TestListProfiles(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		cm.config.Profiles[name] = &Profile{Name: name}
	}

	got := cm.ListProfiles()
	if len(got) != 3 {
		t.Fatalf("ListProfiles() = %v, want 3 entries", got)
	}
	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, name := range got {
		if !want[name] {
			t.Errorf("unexpected profile %q in ListProfiles output", name)
		}
	}
}

// TestSetCurrent verifies that SetCurrent persists the selected profile and
// refuses names that do not exist.
func TestSetCurrent(t *testing.T) {
	t.Parallel()

	t.Run("existing profile is set and persisted", func(t *testing.T) {
		t.Parallel()
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		cm := newTestConfigManagerAt(t, configPath)
		cm.config.Profiles["prod"] = &Profile{Name: "prod"}

		if err := cm.SetCurrent("prod"); err != nil {
			t.Fatalf("SetCurrent failed: %v", err)
		}
		if cm.config.Current != "prod" {
			t.Errorf("Current = %q, want %q", cm.config.Current, "prod")
		}

		cm2 := newTestConfigManagerAt(t, configPath)
		if err := cm2.load(); err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		if cm2.config.Current != "prod" {
			t.Errorf("persisted Current = %q, want %q", cm2.config.Current, "prod")
		}
	})

	t.Run("missing profile errors", func(t *testing.T) {
		t.Parallel()
		cm := newTestConfigManager(t)
		err := cm.SetCurrent("ghost")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("SetCurrent(ghost) error = %v, want 'not found'", err)
		}
	})
}

// TestGetCurrent verifies current-profile lookup: an empty Current marker
// errors, a missing target errors, and a valid target returns the profile.
func TestGetCurrent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		current string
		seed    map[string]*Profile
		wantErr bool
	}{
		{"no current set", "", nil, true},
		{"current points at missing profile", "ghost", nil, true},
		{"current profile exists", "real", map[string]*Profile{
			"real": {Name: "real", AccountID: "12345678901234567890123456789012"},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cm := newTestConfigManager(t)
			for name, p := range tt.seed {
				cm.config.Profiles[name] = p
			}
			cm.config.Current = tt.current

			got, err := cm.GetCurrent()
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCurrent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Name != "real" {
				t.Errorf("GetCurrent() name = %q, want %q", got.Name, "real")
			}
		})
	}
}

// TestGetSecretStore verifies that GetSecretStore exposes the manager's
// underlying secret store and that KeychainAvailable reflects its
// Available() result.
func TestGetSecretStore(t *testing.T) {
	t.Parallel()

	t.Run("unavailable store", func(t *testing.T) {
		t.Parallel()
		cm := newTestConfigManager(t)
		if cm.GetSecretStore() != cm.secrets {
			t.Error("GetSecretStore should return the manager's secret store")
		}
		if cm.KeychainAvailable() {
			t.Error("KeychainAvailable should be false with noop store")
		}
	})

	t.Run("available store", func(t *testing.T) {
		t.Parallel()
		cm := newTestConfigManagerWithSecrets(t, newMemSecrets())
		if !cm.KeychainAvailable() {
			t.Error("KeychainAvailable should be true with memory store")
		}
	})
}

// TestSanitizeForOutput verifies that SanitizeForOutput masks the account ID
// and access key, strips the API token and secret key entirely, and preserves
// non-secret metadata.
func TestSanitizeForOutput(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	cm.config.Current = "full"
	cm.config.Profiles["full"] = &Profile{
		Name:        "full",
		AccountID:   "12345678901234567890123456789012",
		APIToken:    "super-secret-token-value",
		Description: "prod account",
		Endpoint:    "https://example.r2.cloudflarestorage.com",
		AccessKey:   "AKIAIOSFODNN7EXAMPLE",
		SecretKey:   "wJalrXUtnFEMI",
		Region:      "auto",
	}

	sanitized := cm.SanitizeForOutput()

	if sanitized.Current != "full" {
		t.Errorf("Current = %q, want %q", sanitized.Current, "full")
	}
	sp, ok := sanitized.Profiles["full"]
	if !ok {
		t.Fatal("sanitized config should contain profile 'full'")
	}

	// Masked account ID: first 4 + 24 stars + last 4.
	wantAcct := "1234" + strings.Repeat("*", 24) + "9012"
	if sp.AccountID != wantAcct {
		t.Errorf("AccountID = %q, want %q", sp.AccountID, wantAcct)
	}
	// Masked access key: first 2 + stars + last 2.
	wantAccess := "AK" + strings.Repeat("*", 16) + "LE"
	if sp.AccessKey != wantAccess {
		t.Errorf("AccessKey = %q, want %q", sp.AccessKey, wantAccess)
	}
	if sp.APIToken != "" {
		t.Errorf("APIToken = %q, want empty (never included in output)", sp.APIToken)
	}
	if sp.SecretKey != "" {
		t.Errorf("SecretKey = %q, want empty (never included in output)", sp.SecretKey)
	}
	if sp.Description != "prod account" || sp.Endpoint != "https://example.r2.cloudflarestorage.com" || sp.Region != "auto" {
		t.Errorf("non-secret metadata not preserved: %+v", sp)
	}

	// The original config must be untouched.
	if cm.config.Profiles["full"].APIToken != "super-secret-token-value" {
		t.Error("SanitizeForOutput mutated the original config")
	}
}

// TestMaskAccountID verifies the exported account-ID masking wrapper against
// the underlying utils behavior: short IDs are fully redacted, longer ones
// keep their first and last four characters.
func TestMaskAccountID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty id stays empty", "", ""},
		{"short id fully masked", "abcd1234", "********"},
		{"long id keeps first and last four", "12345678901234567890123456789012", "1234" + strings.Repeat("*", 24) + "9012"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MaskAccountID(tt.input); got != tt.want {
				t.Errorf("MaskAccountID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMaskKeyExported verifies that the exported MaskKey wrapper matches the
// internal maskKey behavior.
func TestMaskKeyExported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty key stays empty", "", ""},
		{"short key fully masked", "key", "***"},
		{"long key keeps first and last two", "wJalrXUtnFEMI", "wJ*********MI"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MaskKey(tt.input); got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMigrateToKeychain verifies secret migration: it is refused when the
// keychain is unavailable, migrates plaintext secrets into the store leaving
// sentinels in the persisted config, skips profiles already using sentinels,
// and surfaces store errors.
func TestMigrateToKeychain(t *testing.T) {
	t.Run("unavailable keychain errors", func(t *testing.T) {
		t.Parallel()
		cm := newTestConfigManager(t)
		cm.config.Profiles["p"] = &Profile{Name: "p", APIToken: "tok"}

		count, err := cm.MigrateToKeychain()
		if err == nil || !strings.Contains(err.Error(), "not available") {
			t.Errorf("MigrateToKeychain() error = %v, want 'not available'", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("migrates plaintext secrets and persists sentinels", func(t *testing.T) {
		t.Parallel()
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		secrets := newMemSecrets()
		cm := newTestConfigManagerAt(t, configPath)
		cm.secrets = secrets
		cm.config.Profiles["plaintext"] = &Profile{
			Name:      "plaintext",
			AccountID: "12345678901234567890123456789012",
			APIToken:  "raw-token",
			AccessKey: "raw-access",
			SecretKey: "raw-secret",
		}
		// Already migrated: sentinels and empties must be skipped.
		cm.config.Profiles["done"] = &Profile{
			Name:     "done",
			APIToken: keychainSentinel,
		}

		count, err := cm.MigrateToKeychain()
		if err != nil {
			t.Fatalf("MigrateToKeychain failed: %v", err)
		}
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}

		// Secrets moved into the store.
		for field, want := range map[string]string{
			"api_token":  "raw-token",
			"access_key": "raw-access",
			"secret_key": "raw-secret",
		} {
			got, err := secrets.Get("plaintext", field)
			if err != nil || got != want {
				t.Errorf("store[%s] = %q (err %v), want %q", field, got, err, want)
			}
		}

		// In-memory profile now holds sentinels.
		p := cm.config.Profiles["plaintext"]
		if p.APIToken != keychainSentinel || p.AccessKey != keychainSentinel || p.SecretKey != keychainSentinel {
			t.Errorf("profile secrets = %q/%q/%q, want sentinels", p.APIToken, p.AccessKey, p.SecretKey)
		}

		// Sentinels persisted to disk.
		cm2 := newTestConfigManagerAt(t, configPath)
		if err := cm2.load(); err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		saved := cm2.config.Profiles["plaintext"]
		if saved == nil || saved.APIToken != keychainSentinel {
			t.Errorf("persisted APIToken = %+v, want sentinel", saved)
		}
	})

	t.Run("nothing to migrate returns zero", func(t *testing.T) {
		t.Parallel()
		cm := newTestConfigManagerWithSecrets(t, newMemSecrets())
		cm.config.Profiles["empty"] = &Profile{Name: "empty"}

		count, err := cm.MigrateToKeychain()
		if err != nil {
			t.Fatalf("MigrateToKeychain failed: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("store write error surfaces", func(t *testing.T) {
		t.Parallel()
		secrets := newMemSecrets()
		secrets.setErr = fmt.Errorf("keychain locked")
		cm := newTestConfigManagerWithSecrets(t, secrets)
		cm.config.Profiles["p"] = &Profile{Name: "p", APIToken: "tok"}

		count, err := cm.MigrateToKeychain()
		if err == nil || !strings.Contains(err.Error(), "keychain locked") {
			t.Errorf("MigrateToKeychain() error = %v, want store error", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})
}

// TestSetProfileWithKeychain verifies the store/hydrate round trip when the
// keychain is available: SetProfile swaps plaintext secrets for sentinels on
// disk, GetProfile hydrates them back, and GetProfileRaw leaves sentinels.
func TestSetProfileWithKeychain(t *testing.T) {
	t.Parallel()
	secrets := newMemSecrets()
	cm := newTestConfigManagerWithSecrets(t, secrets)

	if err := cm.SetProfile(&Profile{
		Name:      "sec",
		AccountID: "12345678901234567890123456789012",
		APIToken:  "plain-token",
		AccessKey: "plain-access",
	}); err != nil {
		t.Fatalf("SetProfile failed: %v", err)
	}

	// Persisted profile holds sentinels.
	raw, err := cm.GetProfileRaw("sec")
	if err != nil {
		t.Fatalf("GetProfileRaw failed: %v", err)
	}
	if raw.APIToken != keychainSentinel || raw.AccessKey != keychainSentinel {
		t.Errorf("raw secrets = %q/%q, want sentinels", raw.APIToken, raw.AccessKey)
	}

	// Hydrated profile returns the real values.
	got, err := cm.GetProfile("sec")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if got.APIToken != "plain-token" || got.AccessKey != "plain-access" {
		t.Errorf("hydrated secrets = %q/%q, want plaintext values", got.APIToken, got.AccessKey)
	}
}

// TestSetProfileEmptyName verifies that SetProfile rejects profiles without a
// name before touching storage.
func TestSetProfileEmptyName(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	if err := cm.SetProfile(&Profile{AccountID: "12345678901234567890123456789012"}); err == nil {
		t.Error("expected error when profile name is empty")
	}
}

// TestLoadMalformedYAML verifies that load returns an error when the config
// file contains invalid YAML instead of silently using defaults.
func TestLoadMalformedYAML(t *testing.T) {
	t.Parallel()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("profiles: [unclosed\n"), 0600); err != nil {
		t.Fatalf("writing malformed config failed: %v", err)
	}

	cm := newTestConfigManagerAt(t, configPath)
	if err := cm.load(); err == nil {
		t.Error("expected error loading malformed YAML config")
	}
}

// TestDeleteProfileRemovesKeychainSecrets verifies that deleting a profile
// also deletes its keychain entries.
func TestDeleteProfileRemovesKeychainSecrets(t *testing.T) {
	t.Parallel()
	secrets := newMemSecrets()
	cm := newTestConfigManagerWithSecrets(t, secrets)
	cm.config.Profiles["default"] = &Profile{Name: "default"}
	cm.config.Profiles["doomed"] = &Profile{
		Name:      "doomed",
		APIToken:  keychainSentinel,
		SecretKey: keychainSentinel,
	}
	if err := secrets.Set("doomed", "api_token", "tok"); err != nil {
		t.Fatalf("seeding secrets failed: %v", err)
	}

	if err := cm.DeleteProfile("doomed"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	if _, err := secrets.Get("doomed", "api_token"); err == nil {
		t.Error("keychain secret should have been deleted with the profile")
	}
}

// envTestProfiles seeds cm with a small set of named environment profiles.
func envTestProfiles(cm *ConfigManager) {
	cm.config.Profiles["prod"] = &Profile{
		Name:      "prod",
		AccountID: "12345678901234567890123456789012",
		APIToken:  "tok-prod-abcdef1234567890",
		PlanTier:  "paid",
	}
	cm.config.Profiles["dev"] = &Profile{
		Name:           "dev",
		AccountID:      "abcdef1234567890abcdef1234567890",
		APIToken:       "tok-dev-abcdef1234567890",
		PlanTier:       "free",
		ResourcePrefix: "dev-",
	}
}

// TestValidateEnv_Known verifies that ValidateEnv returns the named profile.
func TestValidateEnv_Known(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	envTestProfiles(cm)

	got, err := cm.ValidateEnv("prod")
	if err != nil {
		t.Fatalf("ValidateEnv(prod) failed: %v", err)
	}
	if got.AccountID != "12345678901234567890123456789012" {
		t.Errorf("AccountID = %q, want prod account", got.AccountID)
	}
	if got.PlanTier != "paid" {
		t.Errorf("PlanTier = %q, want %q", got.PlanTier, "paid")
	}
}

// TestValidateEnv_UnknownListsKnown verifies the actionable error message
// includes every known environment name.
func TestValidateEnv_UnknownListsKnown(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager(t)
	envTestProfiles(cm)

	_, err := cm.ValidateEnv("staging")
	if err == nil {
		t.Fatal("expected error for unknown environment")
	}
	want := `unknown environment "staging": known environments are`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q missing prefix %q", err.Error(), want)
	}
	for _, name := range []string{"prod", "dev"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q should list known environment %q", err.Error(), name)
		}
	}
}

// TestValidateEnv_PlanTier verifies free/paid pass and anything else fails.
func TestValidateEnv_PlanTier(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tier    string
		wantErr bool
	}{
		{"free", false},
		{"paid", false},
		{"enterprise", true},
		{"", false}, // unset is allowed
	}
	for _, tc := range cases {
		cm := newTestConfigManager(t)
		cm.config.Profiles["env"] = &Profile{
			Name:      "env",
			AccountID: "12345678901234567890123456789012",
			APIToken:  "tok-env-abcdef1234567890",
			PlanTier:  tc.tier,
		}
		_, err := cm.ValidateEnv("env")
		if gotErr := err != nil; gotErr != tc.wantErr {
			t.Errorf("PlanTier %q: err = %v, wantErr %v", tc.tier, err, tc.wantErr)
		}
	}
}

// TestValidateEnv_ResourcePrefix verifies the reserved field: set values must
// be non-empty and whitespace-free; unset is allowed.
func TestValidateEnv_ResourcePrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		prefix  string
		wantErr bool
	}{
		{"dev-", false},
		{"", false}, // unset is allowed
		{"   ", true},
		{"has space", true},
		{"tab\there", true},
	}
	for _, tc := range cases {
		cm := newTestConfigManager(t)
		cm.config.Profiles["env"] = &Profile{
			Name:           "env",
			AccountID:      "12345678901234567890123456789012",
			APIToken:       "tok-env-abcdef1234567890",
			ResourcePrefix: tc.prefix,
		}
		_, err := cm.ValidateEnv("env")
		if gotErr := err != nil; gotErr != tc.wantErr {
			t.Errorf("ResourcePrefix %q: err = %v, wantErr %v", tc.prefix, err, tc.wantErr)
		}
	}
}

// TestValidateEnv_NewFieldsRoundTrip verifies PlanTier and ResourcePrefix
// survive a Save followed by a fresh load from disk.
func TestValidateEnv_NewFieldsRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	cm := newTestConfigManagerAt(t, path)
	cm.config.Profiles["prod"] = &Profile{
		Name:           "prod",
		AccountID:      "12345678901234567890123456789012",
		APIToken:       "tok-prod-abcdef1234567890",
		PlanTier:       "paid",
		ResourcePrefix: "prod-",
	}
	if err := cm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	fresh := newTestConfigManagerAt(t, path)
	if err := fresh.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	got, err := fresh.ValidateEnv("prod")
	if err != nil {
		t.Fatalf("ValidateEnv after reload failed: %v", err)
	}
	if got.PlanTier != "paid" {
		t.Errorf("PlanTier = %q, want %q", got.PlanTier, "paid")
	}
	if got.ResourcePrefix != "prod-" {
		t.Errorf("ResourcePrefix = %q, want %q", got.ResourcePrefix, "prod-")
	}
}
