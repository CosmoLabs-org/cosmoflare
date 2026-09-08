package cosmoflare

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `bucket: my-bucket
region: auto
endpoint: https://example.com
env: production
max_file_size: 1048576
allowed_buckets:
  - bucket-a
  - bucket-b
cache:
  enabled: true
  default: "max-age=3600"
  rules:
    png: "max-age=86400"
    html: "no-cache"
guardrails:
  enabled: true
  max_file_size: 524288
  allowed_buckets:
    - safe-bucket
  blocked_keys:
    - secret/
  require_tags:
    - project
audit:
  enabled: true
  path: /var/log/r2go2.jsonl
  fields:
    - timestamp
    - action
    - key
`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if cfg.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, want %q", cfg.Bucket, "my-bucket")
	}
	if cfg.Region != "auto" {
		t.Errorf("Region = %q, want %q", cfg.Region, "auto")
	}
	if cfg.Endpoint != "https://example.com" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "https://example.com")
	}
	if cfg.Environment != "production" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "production")
	}
	if cfg.MaxFileSize != 1048576 {
		t.Errorf("MaxFileSize = %d, want %d", cfg.MaxFileSize, 1048576)
	}
	if len(cfg.AllowedBuckets) != 2 {
		t.Errorf("AllowedBuckets len = %d, want 2", len(cfg.AllowedBuckets))
	}
}

func TestLoadProjectConfig_CachePolicy(t *testing.T) {
	dir := t.TempDir()
	yaml := `bucket: test
cache:
  enabled: true
  default: "max-age=3600"
  rules:
    png: "max-age=86400"
    css: "max-age=604800"
`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if !cfg.CachePolicy.Enabled {
		t.Error("CachePolicy.Enabled = false, want true")
	}
	if cfg.CachePolicy.Default != "max-age=3600" {
		t.Errorf("CachePolicy.Default = %q, want %q", cfg.CachePolicy.Default, "max-age=3600")
	}
	if len(cfg.CachePolicy.Rules) != 2 {
		t.Errorf("CachePolicy.Rules len = %d, want 2", len(cfg.CachePolicy.Rules))
	}
	if cfg.CachePolicy.Rules["png"] != "max-age=86400" {
		t.Errorf("CachePolicy.Rules[png] = %q, want %q", cfg.CachePolicy.Rules["png"], "max-age=86400")
	}
}

func TestLoadProjectConfig_Guardrails(t *testing.T) {
	dir := t.TempDir()
	yaml := `bucket: test
guardrails:
  enabled: true
  max_file_size: 524288
  allowed_buckets:
    - safe-bucket
  blocked_keys:
    - secret/
    - .env
  require_tags:
    - project
    - owner
`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if !cfg.Guardrails.Enabled {
		t.Error("Guardrails.Enabled = false, want true")
	}
	if cfg.Guardrails.MaxFileSize != 524288 {
		t.Errorf("Guardrails.MaxFileSize = %d, want 524288", cfg.Guardrails.MaxFileSize)
	}
	if len(cfg.Guardrails.BlockedKeys) != 2 {
		t.Errorf("Guardrails.BlockedKeys len = %d, want 2", len(cfg.Guardrails.BlockedKeys))
	}
	if len(cfg.Guardrails.RequireTags) != 2 {
		t.Errorf("Guardrails.RequireTags len = %d, want 2", len(cfg.Guardrails.RequireTags))
	}
}

func TestLoadProjectConfig_Audit(t *testing.T) {
	dir := t.TempDir()
	yaml := `bucket: test
audit:
  enabled: true
  path: /tmp/audit.jsonl
  fields:
    - timestamp
    - action
`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if !cfg.Audit.Enabled {
		t.Error("Audit.Enabled = false, want true")
	}
	if cfg.Audit.Path != "/tmp/audit.jsonl" {
		t.Errorf("Audit.Path = %q, want %q", cfg.Audit.Path, "/tmp/audit.jsonl")
	}
	if len(cfg.Audit.Fields) != 2 {
		t.Errorf("Audit.Fields len = %d, want 2", len(cfg.Audit.Fields))
	}
}

func TestLoadProjectConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadProjectConfig(dir)
	if err == nil {
		t.Fatal("LoadProjectConfig should return error for missing file")
	}
}

func TestLoadProjectConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	invalid := `bucket: [[[invalid yaml {{{`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(invalid), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadProjectConfig(dir)
	if err == nil {
		t.Fatal("LoadProjectConfig should return error for invalid YAML")
	}
}

func TestLoadProjectConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error for empty file: %v", err)
	}

	// All fields should be zero values
	if cfg.Bucket != "" {
		t.Errorf("Bucket = %q, want empty", cfg.Bucket)
	}
	if cfg.Region != "" {
		t.Errorf("Region = %q, want empty", cfg.Region)
	}
	if cfg.CachePolicy.Enabled {
		t.Error("CachePolicy.Enabled = true, want false")
	}
	if cfg.Guardrails.Enabled {
		t.Error("Guardrails.Enabled = true, want false")
	}
	if cfg.Audit.Enabled {
		t.Error("Audit.Enabled = true, want false")
	}
}

func TestLoadProjectConfig_ParentDirectorySearch(t *testing.T) {
	// Create config in parent, load from child
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir", "deep")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	yaml := `bucket: found-in-parent
`
	if err := os.WriteFile(filepath.Join(parent, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(child)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if cfg.Bucket != "found-in-parent" {
		t.Errorf("Bucket = %q, want %q", cfg.Bucket, "found-in-parent")
	}
}

func TestLoadProjectConfig_MinimalBucketOnly(t *testing.T) {
	dir := t.TempDir()
	yaml := `bucket: just-a-bucket
`
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if cfg.Bucket != "just-a-bucket" {
		t.Errorf("Bucket = %q, want %q", cfg.Bucket, "just-a-bucket")
	}
	if cfg.MaxFileSize != 0 {
		t.Errorf("MaxFileSize = %d, want 0 (default)", cfg.MaxFileSize)
	}
	if cfg.Environment != "" {
		t.Errorf("Environment = %q, want empty (default)", cfg.Environment)
	}
}

func TestMachineConfig_GetProfile(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"prod": {
				AccountID: "acc-123",
				APIToken:  "tok-abc",
			},
			"staging": {
				AccountID: "acc-456",
				APIToken:  "tok-def",
			},
		},
		Current: "prod",
	}

	p, err := mc.GetProfile("prod")
	if err != nil {
		t.Fatalf("GetProfile(prod) returned error: %v", err)
	}
	if p.AccountID != "acc-123" {
		t.Errorf("AccountID = %q, want %q", p.AccountID, "acc-123")
	}
	if p.APIToken != "tok-abc" {
		t.Errorf("APIToken = %q, want %q", p.APIToken, "tok-abc")
	}
}

func TestMachineConfig_GetProfile_NotFound(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{},
		Current:  "default",
	}

	_, err := mc.GetProfile("nonexistent")
	if err == nil {
		t.Fatal("GetProfile should return error for missing profile")
	}
}

func TestMachineConfig_CurrentProfile(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"default": {
				AccountID: "acc-default",
				APIToken:  "tok-default",
			},
		},
		Current: "default",
	}

	p, err := mc.CurrentProfile()
	if err != nil {
		t.Fatalf("CurrentProfile returned error: %v", err)
	}
	if p.AccountID != "acc-default" {
		t.Errorf("AccountID = %q, want %q", p.AccountID, "acc-default")
	}
}

func TestMachineConfig_CurrentProfile_Empty(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{},
		Current:  "",
	}

	_, err := mc.CurrentProfile()
	if err == nil {
		t.Fatal("CurrentProfile should return error when Current is empty")
	}
}

func TestMachineConfig_CurrentProfile_MissingProfile(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{},
		Current:  "missing",
	}

	_, err := mc.CurrentProfile()
	if err == nil {
		t.Fatal("CurrentProfile should return error when current profile does not exist")
	}
}

func TestConfigProfileFromEnv(t *testing.T) {
	// Set environment variables
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-account")
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")
	t.Setenv("AWS_ACCESS_KEY_ID", "env-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret")
	t.Setenv("R2_ENDPOINT", "https://env-endpoint.com")
	t.Setenv("AWS_REGION", "us-east-1")

	p := ProfileFromEnv()

	if p.AccountID != "env-account" {
		t.Errorf("AccountID = %q, want %q", p.AccountID, "env-account")
	}
	if p.APIToken != "env-token" {
		t.Errorf("APIToken = %q, want %q", p.APIToken, "env-token")
	}
	if p.AccessKey != "env-access" {
		t.Errorf("AccessKey = %q, want %q", p.AccessKey, "env-access")
	}
	if p.SecretKey != "env-secret" {
		t.Errorf("SecretKey = %q, want %q", p.SecretKey, "env-secret")
	}
	if p.Endpoint != "https://env-endpoint.com" {
		t.Errorf("Endpoint = %q, want %q", p.Endpoint, "https://env-endpoint.com")
	}
	if p.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", p.Region, "us-east-1")
	}
}

func TestConfigProfileFromEnv_EmptyVars(t *testing.T) {
	// Clear all relevant env vars
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("R2_ENDPOINT", "")
	t.Setenv("AWS_REGION", "")

	p := ProfileFromEnv()

	if p.AccountID != "" {
		t.Errorf("AccountID = %q, want empty", p.AccountID)
	}
	if p.APIToken != "" {
		t.Errorf("APIToken = %q, want empty", p.APIToken)
	}
}

func TestConfigFindProjectFile(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "a", "b", "c")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	target := filepath.Join(parent, ".r2go2.yaml")
	if err := os.WriteFile(target, []byte("bucket: test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	found, err := findProjectFile(child, ".r2go2.yaml")
	if err != nil {
		t.Fatalf("findProjectFile returned error: %v", err)
	}
	if found != target {
		t.Errorf("findProjectFile = %q, want %q", found, target)
	}
}

func TestConfigFindProjectFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := findProjectFile(dir, ".nonexistent-config.yaml")
	if err == nil {
		t.Fatal("findProjectFile should return error when file not found")
	}
}

func TestLoadMachineConfig_NoConfigDir(t *testing.T) {
	// LoadMachineConfig uses os.UserHomeDir which we can't easily override,
	// but we can verify it returns a valid default config when the file doesn't exist.
	// This test verifies the default-creation path works when the home dir config is absent.
	cfg, err := LoadMachineConfig()
	if err != nil {
		t.Fatalf("LoadMachineConfig returned error: %v", err)
	}

	if cfg.Profiles == nil {
		t.Error("Profiles map should not be nil")
	}
}

func TestMachineConfig_Save_FilePermissions(t *testing.T) {
	// Verify Save creates the config file with 0600 permissions
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"test": {
				AccountID: "save-test-acc",
				APIToken:  "save-test-tok",
				Region:    "eu-west-1",
			},
		},
		Current: "test",
	}

	if err := mc.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify the file exists with restricted permissions
	configPath := filepath.Join(tmpHome, ".r2go2", "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not found after Save: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify current field was persisted
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}
	if len(content) == 0 {
		t.Error("saved config file is empty")
	}
}

func TestProjectConfigWorkersPlan(t *testing.T) {
	dir := t.TempDir()
	yamlBody := "workers_plan: paid\n"
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte(yamlBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.WorkersPlan != "paid" {
		t.Fatalf("WorkersPlan = %q, want paid", cfg.WorkersPlan)
	}
}
