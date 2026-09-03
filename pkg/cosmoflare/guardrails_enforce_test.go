package cosmoflare

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newGuardedTestClient builds a client whose endpoint points at a closed local
// port. Any upload that passes the guardrail check fails instantly with a
// transport error, so a guardrail error is distinguishable from a transport
// error and no test ever reaches the network.
func newGuardedTestClient(t *testing.T, cfg *ProjectConfig, opts ...ClientOption) R2Client {
	t.Helper()
	base := []ClientOption{
		WithAccountID("test-account-id"),
		WithAPIToken("test-api-token"),
		WithEndpoint("http://127.0.0.1:1"),
	}
	if cfg != nil {
		base = append(base, WithProjectConfig(cfg))
	}
	c, err := NewClient(append(base, opts...)...)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return c
}

// TestUpload_GuardrailBlockedKey_NoUpload proves that a key matching a
// blocked_keys pattern fails with a guardrail error before any upload attempt.
func TestUpload_GuardrailBlockedKey_NoUpload(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys: []string{".env"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	result, err := c.Upload(context.Background(), "my-bucket", ".env", strings.NewReader("SECRET=1"), 9)
	if err == nil {
		t.Fatal("Upload of blocked key .env succeeded, want guardrail rejection")
	}
	if result != nil {
		t.Errorf("Upload returned non-nil result alongside error: %+v", result)
	}
	msg := err.Error()
	if !strings.Contains(msg, "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", msg)
	}
	if !strings.Contains(msg, ".env") {
		t.Errorf("error does not mention the blocked key:\n%s", msg)
	}
}

// TestUpload_GuardrailBlockedBucket proves the allowed_buckets allowlist is
// enforced on the upload path.
func TestUpload_GuardrailBlockedBucket(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			AllowedBuckets: []string{"prod"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.Upload(context.Background(), "dev", "index.html", strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("Upload to non-allowed bucket succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "allowed_buckets") {
		t.Errorf("error does not mention allowed_buckets:\n%s", err.Error())
	}
}

// TestUpload_GuardrailMaxFileSize proves max_file_size is enforced on the
// upload path.
func TestUpload_GuardrailMaxFileSize(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			MaxFileSize: 4,
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.Upload(context.Background(), "my-bucket", "big.bin", strings.NewReader("12345"), 5)
	if err == nil {
		t.Fatal("Upload exceeding max_file_size succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "max_file_size") {
		t.Errorf("error does not mention max_file_size:\n%s", err.Error())
	}
}

// TestUpload_GuardrailPasses_NonMatchingKey proves a key that violates no
// guardrail is not blocked by policy: the operation proceeds and fails with a
// transport error (closed local endpoint), never a guardrail error.
func TestUpload_GuardrailPasses_NonMatchingKey(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys:    []string{".env"},
			AllowedBuckets: []string{"my-bucket"},
			MaxFileSize:    1024,
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.Upload(context.Background(), "my-bucket", "index.html", strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("Upload unexpectedly succeeded against closed endpoint")
	}
	if strings.Contains(err.Error(), "guardrail") {
		t.Errorf("non-violating upload was blocked by guardrails:\n%s", err.Error())
	}
}

// TestUpload_NoProjectConfig_Unaffected proves library callers that attach no
// project config keep the previous behavior: no guardrail enforcement.
func TestUpload_NoProjectConfig_Unaffected(t *testing.T) {
	c := newGuardedTestClient(t, nil)

	_, err := c.Upload(context.Background(), "my-bucket", ".env", strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("Upload unexpectedly succeeded against closed endpoint")
	}
	if strings.Contains(err.Error(), "guardrail") {
		t.Errorf("upload without project config was blocked by guardrails:\n%s", err.Error())
	}
}

// TestMultipartUpload_GuardrailBlockedKey proves the multipart entry point is
// guarded too.
func TestMultipartUpload_GuardrailBlockedKey(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys: []string{".env"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.MultipartUpload(context.Background(), "my-bucket", ".env", strings.NewReader("data"), 4)
	if err == nil {
		t.Fatal("MultipartUpload of blocked key succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", err.Error())
	}
}

// TestResumableMultipartUpload_GuardrailBlockedKey proves the resumable
// multipart entry point rejects a blocked key before touching upload state.
func TestResumableMultipartUpload_GuardrailBlockedKey(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys: []string{".env"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.ResumableMultipartUpload(context.Background(), "my-bucket", ".env", strings.NewReader("data"), 4)
	if err == nil {
		t.Fatal("ResumableMultipartUpload of blocked key succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", err.Error())
	}
}

// TestResumeMultipartUpload_GuardrailBlockedKey proves the resume entry point
// rejects a blocked key before loading saved upload state.
func TestResumeMultipartUpload_GuardrailBlockedKey(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys: []string{".env"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.ResumeMultipartUpload(context.Background(), "my-bucket", ".env", strings.NewReader("data"), 4)
	if err == nil {
		t.Fatal("ResumeMultipartUpload of blocked key succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", err.Error())
	}
}

// TestUpload_LargeFileDelegation_GuardrailBlocked proves the auto-delegation
// from Upload to MultipartUpload cannot bypass guardrails.
func TestUpload_LargeFileDelegation_GuardrailBlocked(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{
			BlockedKeys: []string{"*.pem"},
		},
	}
	c := newGuardedTestClient(t, cfg)

	_, err := c.Upload(context.Background(), "my-bucket", "server.pem", strings.NewReader("data"), multipartThreshold+1)
	if err == nil {
		t.Fatal("Upload (multipart delegation) of blocked key succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", err.Error())
	}
}

// TestLoadProjectConfig_PrefersCosmoflareYAML proves guardrail rules written
// in .cosmoflare.yaml are loaded, with .r2go2.yaml kept as fallback.
func TestLoadProjectConfig_PrefersCosmoflareYAML(t *testing.T) {
	dir := t.TempDir()

	newStyle := []byte("bucket: from-cosmoflare\nguardrails:\n  blocked_keys:\n    - \".env\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), newStyle, 0644); err != nil {
		t.Fatal(err)
	}
	oldStyle := []byte("bucket: from-r2go2\n")
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), oldStyle, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}
	if cfg.Bucket != "from-cosmoflare" {
		t.Errorf("Bucket = %q, want %q (.cosmoflare.yaml must take precedence)", cfg.Bucket, "from-cosmoflare")
	}
	if len(cfg.Guardrails.BlockedKeys) != 1 || cfg.Guardrails.BlockedKeys[0] != ".env" {
		t.Errorf("Guardrails.BlockedKeys = %v, want [.env]", cfg.Guardrails.BlockedKeys)
	}
}

// TestLoadProjectConfig_FallsBackToR2go2YAML proves the legacy filename still
// loads when no .cosmoflare.yaml exists.
func TestLoadProjectConfig_FallsBackToR2go2YAML(t *testing.T) {
	dir := t.TempDir()

	oldStyle := []byte("bucket: from-r2go2\nguardrails:\n  blocked_keys:\n    - \".env\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".r2go2.yaml"), oldStyle, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}
	if cfg.Bucket != "from-r2go2" {
		t.Errorf("Bucket = %q, want %q", cfg.Bucket, "from-r2go2")
	}
	if len(cfg.Guardrails.BlockedKeys) != 1 || cfg.Guardrails.BlockedKeys[0] != ".env" {
		t.Errorf("Guardrails.BlockedKeys = %v, want [.env]", cfg.Guardrails.BlockedKeys)
	}
}

// TestIsExcluded_DirectoryPrefixPattern proves a trailing-slash exclude
// pattern skips everything nested under that directory, at any depth.
func TestIsExcluded_DirectoryPrefixPattern(t *testing.T) {
	cases := []struct {
		relPath  string
		excludes []string
		want     bool
	}{
		{".git/config", []string{".git/"}, true},
		{".git/objects/ab/cdef", []string{".git/"}, true},
		{"site/.git/HEAD", []string{".git/"}, false}, // prefix must match from the relative root
		{"gitt/config", []string{".git/"}, false},
		{"index.html", []string{".git/"}, false},
		{".env", []string{".env"}, true},
		{".env.local", []string{".env.*"}, true},
		{".DS_Store", []string{".DS_Store"}, true},
		{"nested/.DS_Store", []string{".DS_Store"}, true},
	}
	for _, tc := range cases {
		if got := isExcluded(tc.relPath, tc.excludes); got != tc.want {
			t.Errorf("isExcluded(%q, %v) = %v, want %v", tc.relPath, tc.excludes, got, tc.want)
		}
	}
}
