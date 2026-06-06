package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// TestValidateCmdRegistration verifies that validateCmd is registered on rootCmd
// and has the correct Use and Short fields.
func TestValidateCmdRegistration(t *testing.T) {
	if validateCmd == nil {
		t.Fatal("validateCmd is nil")
	}

	if validateCmd.Use != "validate [file]" {
		t.Errorf("validateCmd.Use = %q, want %q", validateCmd.Use, "validate [file]")
	}

	if validateCmd.Short == "" {
		t.Error("validateCmd.Short is empty")
	}

	// Verify it is registered on rootCmd
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub == validateCmd {
			found = true
			break
		}
	}
	if !found {
		t.Error("validateCmd is not registered on rootCmd")
	}
}

// TestValidateFlagDefaults verifies --strict and --fix default to false.
func TestValidateFlagDefaults(t *testing.T) {
	strictFlag := validateCmd.Flags().Lookup("strict")
	if strictFlag == nil {
		t.Fatal("--strict flag not registered")
	}
	if strictFlag.DefValue != "false" {
		t.Errorf("--strict default = %q, want %q", strictFlag.DefValue, "false")
	}

	fixFlag := validateCmd.Flags().Lookup("fix")
	if fixFlag == nil {
		t.Fatal("--fix flag not registered")
	}
	if fixFlag.DefValue != "false" {
		t.Errorf("--fix default = %q, want %q", fixFlag.DefValue, "false")
	}
}

// TestValidateFlagsParsing verifies that --strict and --fix flags can be parsed.
func TestValidateFlagsParsing(t *testing.T) {
	reset := func() {
		validateStrict = false
		validateFix = false
	}

	t.Run("strict flag sets validateStrict", func(t *testing.T) {
		reset()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&validateStrict, "strict", false, "")
		if err := cmd.Flags().Set("strict", "true"); err != nil {
			t.Fatalf("failed to set --strict: %v", err)
		}
		if !validateStrict {
			t.Error("validateStrict should be true after --strict=true")
		}
	})

	t.Run("fix flag sets validateFix", func(t *testing.T) {
		reset()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&validateFix, "fix", false, "")
		if err := cmd.Flags().Set("fix", "true"); err != nil {
			t.Fatalf("failed to set --fix: %v", err)
		}
		if !validateFix {
			t.Error("validateFix should be true after --fix=true")
		}
	})
}

// TestRunValidate_NoConfigFile tests validate when no config file exists.
func TestRunValidate_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runValidate(&cobra.Command{}, []string{})
	if err == nil {
		t.Fatal("expected error when no config file exists")
	}
}

// TestRunValidate_ValidConfig tests validate with a valid config file.
func TestRunValidate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
name: my-project
type: application
r2:
  buckets:
    - name: my-bucket
kv:
  namespaces:
    - title: my-cache
`
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Should succeed (no error)
	err = runValidate(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}
}

// TestRunValidate_InvalidConfig tests validate with an invalid config file.
func TestRunValidate_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
r2:
  buckets:
    - name: INVALID-BUCKET
`
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runValidate(&cobra.Command{}, []string{})
	if err == nil {
		t.Fatal("expected error for invalid config (missing name/type, bad bucket)")
	}
}

// TestRunValidate_SpecificFile tests validate with a file argument.
func TestRunValidate_SpecificFile(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
name: test
type: app
`
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Pass the directory as argument (LoadCosmoflareConfig searches for .cosmoflare.yaml)
	err := runValidate(&cobra.Command{}, []string{tmpDir})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestRunValidate_StrictMode tests that --strict causes warnings to fail.
func TestRunValidate_StrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
name: test
type: app
dns:
  records:
    - type: a
      name: example.com
      content: "1.2.3.4"
`
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Without strict — should pass (lowercase type is a warning, not error)
	validateStrict = false
	err = runValidate(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("expected no error without strict, got: %v", err)
	}

	// With strict — should fail
	validateStrict = true
	defer func() { validateStrict = false }()
	err = runValidate(&cobra.Command{}, []string{})
	if err == nil {
		t.Fatal("expected error in strict mode with warnings")
	}
}

// TestFileDir tests the fileDir helper function.
func TestFileDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/foo/bar/baz.yaml", "/foo/bar"},
		{"config.yaml", "."},
		{"/config.yaml", ""},
		{"a/b/c", "a/b"},
	}

	for _, tt := range tests {
		got := fileDir(tt.input)
		if got != tt.want {
			t.Errorf("fileDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
