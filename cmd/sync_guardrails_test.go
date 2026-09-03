package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestEffectiveSyncExcludes_Defaults proves no explicit --exclude means the
// sensitive-file defaults apply.
func TestEffectiveSyncExcludes_Defaults(t *testing.T) {
	got := effectiveSyncExcludes(nil)
	want := []string{".git/", ".env", ".env.*", ".DS_Store"}
	if len(got) != len(want) {
		t.Fatalf("effectiveSyncExcludes(nil) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("effectiveSyncExcludes(nil)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestEffectiveSyncExcludes_ExplicitReplacesDefaults proves explicit excludes
// REPLACE the defaults (rsync-like semantics), not extend them.
func TestEffectiveSyncExcludes_ExplicitReplacesDefaults(t *testing.T) {
	explicit := []string{"*.log"}
	got := effectiveSyncExcludes(explicit)
	if len(got) != 1 || got[0] != "*.log" {
		t.Errorf("effectiveSyncExcludes(%v) = %v, want [*.log]", explicit, got)
	}
}

// TestScanLocalDir_DefaultExcludes_SkipSensitive proves the default exclude
// set keeps .git contents, env files, and .DS_Store out of sync uploads.
func TestScanLocalDir_DefaultExcludes_SkipSensitive(t *testing.T) {
	tmp := t.TempDir()
	files := []string{
		"index.html",
		".env",
		".env.local",
		".DS_Store",
		".git/config",
		".git/HEAD",
		".git/objects/ab/cdef",
	}
	for _, f := range files {
		path := filepath.Join(tmp, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := scanLocalDir(tmp, false, effectiveSyncExcludes(nil))
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}

	paths := make(map[string]bool, len(got))
	for _, f := range got {
		paths[f.RelPath] = true
	}

	if len(got) != 1 || !paths["index.html"] {
		t.Errorf("scanLocalDir with defaults returned %v, want only [index.html]", paths)
	}
}

// TestScanLocalDir_ExplicitExcludes_ReplaceDefaults proves a user-provided
// exclude list drops the defaults: .env is scanned again.
func TestScanLocalDir_ExplicitExcludes_ReplaceDefaults(t *testing.T) {
	tmp := t.TempDir()
	files := []string{"keep.txt", ".env", "debug.log"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmp, f), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := scanLocalDir(tmp, false, effectiveSyncExcludes([]string{"*.log"}))
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}

	paths := make(map[string]bool, len(got))
	for _, f := range got {
		paths[f.RelPath] = true
	}

	if !paths[".env"] {
		t.Errorf("explicit excludes must replace defaults: .env missing from %v", paths)
	}
	if paths["debug.log"] {
		t.Errorf("debug.log should be excluded by explicit pattern: %v", paths)
	}
	if !paths["keep.txt"] {
		t.Errorf("keep.txt missing from %v", paths)
	}
}

// TestScanLocalDir_NoExcludes_ReturnsEverything proves the raw scanner with
// no exclude list keeps its original behavior.
func TestScanLocalDir_NoExcludes_ReturnsEverything(t *testing.T) {
	tmp := t.TempDir()
	files := []string{"index.html", ".env", ".git/HEAD"}
	for _, f := range files {
		path := filepath.Join(tmp, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := scanLocalDir(tmp, false, nil)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("scanLocalDir with nil excludes returned %d files, want 3", len(got))
	}
}

// TestR2StorageBackend_UploadFile_GuardrailBlocked proves the sync upload
// path enforces guardrails loaded from the project config: a blocked key
// fails with a guardrail error and no upload is attempted.
func TestR2StorageBackend_UploadFile_GuardrailBlocked(t *testing.T) {
	dir := t.TempDir()

	configYAML := "guardrails:\n  blocked_keys:\n    - \".env\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := append(projectConfigOptions(dir), cosmoflare.WithEndpoint("http://127.0.0.1:1"))
	backend, err := newR2StorageBackend("test-account-id", "test-api-token", opts...)
	if err != nil {
		t.Fatalf("newR2StorageBackend returned error: %v", err)
	}

	err = backend.UploadFile(context.Background(), "my-bucket", ".env", filepath.Join(dir, ".env"))
	if err == nil {
		t.Fatal("UploadFile of blocked key .env succeeded, want guardrail rejection")
	}
	if !strings.Contains(err.Error(), "guardrail") {
		t.Errorf("error does not mention guardrails:\n%s", err.Error())
	}
}

// TestR2StorageBackend_UploadFile_GuardrailPasses proves a non-violating file
// is not stopped by policy: it proceeds and fails on transport (closed local
// endpoint), proving the guardrail is the only thing that stopped .env above.
func TestR2StorageBackend_UploadFile_GuardrailPasses(t *testing.T) {
	dir := t.TempDir()

	configYAML := "guardrails:\n  blocked_keys:\n    - \".env\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := append(projectConfigOptions(dir), cosmoflare.WithEndpoint("http://127.0.0.1:1"))
	backend, err := newR2StorageBackend("test-account-id", "test-api-token", opts...)
	if err != nil {
		t.Fatalf("newR2StorageBackend returned error: %v", err)
	}

	err = backend.UploadFile(context.Background(), "my-bucket", "ok.txt", filepath.Join(dir, "ok.txt"))
	if err == nil {
		t.Fatal("UploadFile unexpectedly succeeded against closed endpoint")
	}
	if strings.Contains(err.Error(), "guardrail") {
		t.Errorf("non-violating upload was blocked by guardrails:\n%s", err.Error())
	}
}

// TestProjectConfigOptions_MissingConfig proves a directory without a project
// config yields no options: guardrails are simply inactive, not an error.
// (The loaded-config path is proven end-to-end by the
// TestR2StorageBackend_UploadFile_Guardrail* tests above.)
func TestProjectConfigOptions_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	if got := projectConfigOptions(dir); len(got) != 0 {
		t.Errorf("projectConfigOptions on config-less dir = %v, want none", got)
	}
}
