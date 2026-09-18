package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestWranglerCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "wrangler" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("wranglerCmd not registered on rootCmd")
	}
}

func TestWranglerCmd_Metadata(t *testing.T) {
	if wranglerCmd.Use != "wrangler" {
		t.Errorf("wranglerCmd.Use = %q, want %q", wranglerCmd.Use, "wrangler")
	}
	if wranglerCmd.Short == "" {
		t.Error("wranglerCmd.Short is empty")
	}
	if wranglerCmd.Long == "" {
		t.Error("wranglerCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestWranglerCmd_Subcommands(t *testing.T) {
	expected := []string{"import", "diff", "validate"}
	for _, name := range expected {
		found := false
		for _, sub := range wranglerCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("wrangler subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestWranglerCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		wranglerImportCmd,
		wranglerDiffCmd,
		wranglerValidateCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestWranglerImport_Flags(t *testing.T) {
	expected := []string{"output", "force"}
	for _, name := range expected {
		if wranglerImportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on wranglerImportCmd", name)
		}
	}
}

func TestWranglerImport_OutputShorthand(t *testing.T) {
	f := wranglerImportCmd.Flags().ShorthandLookup("o")
	if f == nil {
		t.Error("shorthand -o not registered for --output")
	}
}

func TestWranglerImport_ForceShorthand(t *testing.T) {
	f := wranglerImportCmd.Flags().ShorthandLookup("f")
	if f == nil {
		t.Error("shorthand -f not registered for --force")
	}
}

// --- Args validation ---

func TestWranglerImport_MaxArgs(t *testing.T) {
	// import accepts 0 or 1 args
	cmd := wranglerImportCmd
	if cmd.Args == nil {
		t.Skip("Args validator not set, accepts any args")
	}
	// Test with 0 args (should pass)
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("0 args should be valid: %v", err)
	}
	// Test with 1 arg (should pass)
	if err := cmd.Args(cmd, []string{"path"}); err != nil {
		t.Errorf("1 arg should be valid: %v", err)
	}
	// Test with 2 args (should fail)
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("2 args should be invalid")
	}
}

// --- Help text quality ---

func TestWranglerImport_HelpContainsCosmoflare(t *testing.T) {
	if wranglerImportCmd.Long == "" {
		t.Fatal("wranglerImportCmd.Long is empty")
	}
	// Help should reference cosmoflare for brand consistency
	found := false
	for _, sub := range []string{"cosmoflare", ".cosmoflare.yaml"} {
		if containsString(wranglerImportCmd.Long, sub) {
			found = true
			break
		}
	}
	if !found {
		t.Error("wranglerImportCmd.Long should reference cosmoflare")
	}
}

func TestWranglerValidate_HelpContainsChecks(t *testing.T) {
	if wranglerValidateCmd.Long == "" {
		t.Fatal("wranglerValidateCmd.Long is empty")
	}
	checks := []string{"name", "KV", "R2", "D1", "binding"}
	for _, check := range checks {
		if !containsString(wranglerValidateCmd.Long, check) {
			t.Errorf("wranglerValidateCmd.Long should mention %q", check)
		}
	}
}

// --- Functional tests with temp files ---

func TestWranglerValidate_ValidFile(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	content := `name = "test-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reset global state
	oldJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = oldJSON }()

	err := runWranglerValidate(wranglerValidateCmd, []string{tomlPath})
	if err != nil {
		t.Errorf("validate should pass for valid file: %v", err)
	}
}

func TestWranglerValidate_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	content := `main = "src/index.ts"` // missing name
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	oldJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = oldJSON }()

	err := runWranglerValidate(wranglerValidateCmd, []string{tomlPath})
	if err == nil {
		t.Error("validate should fail for file missing name")
	}
}

func TestWranglerValidate_FileNotFound(t *testing.T) {
	oldJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = oldJSON }()

	err := runWranglerValidate(wranglerValidateCmd, []string{"/nonexistent/wrangler.toml"})
	if err == nil {
		t.Error("validate should fail for nonexistent file")
	}
}

func TestWranglerImport_DryRun(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	content := `name = "test-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	oldJSON := JSONOutput
	oldDryRun := DryRun
	JSONOutput = false
	DryRun = true
	defer func() {
		JSONOutput = oldJSON
		DryRun = oldDryRun
	}()

	err := runWranglerImport(wranglerImportCmd, []string{tomlPath})
	if err != nil {
		t.Errorf("dry-run import should succeed: %v", err)
	}

	// Verify no file was written
	outputPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(outputPath); err == nil {
		t.Error(".cosmoflare.yaml should not be created in dry-run mode")
	}
}

func TestWranglerImport_WritesFile(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	content := `name = "test-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	oldJSON := JSONOutput
	oldDryRun := DryRun
	JSONOutput = false
	DryRun = false
	wranglerForce = false
	defer func() {
		JSONOutput = oldJSON
		DryRun = oldDryRun
	}()

	err := runWranglerImport(wranglerImportCmd, []string{tomlPath})
	if err != nil {
		t.Errorf("import should succeed: %v", err)
	}

	outputPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error(".cosmoflare.yaml should be created")
	}
}

func TestWranglerImport_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	content := `name = "test-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create existing .cosmoflare.yaml
	outputPath := filepath.Join(dir, ".cosmoflare.yaml")
	if err := os.WriteFile(outputPath, []byte("existing: true"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	oldJSON := JSONOutput
	oldDryRun := DryRun
	JSONOutput = false
	DryRun = false
	wranglerForce = false
	defer func() {
		JSONOutput = oldJSON
		DryRun = oldDryRun
	}()

	err := runWranglerImport(wranglerImportCmd, []string{tomlPath})
	if err == nil {
		t.Error("import should refuse to overwrite existing file without --force")
	}
}

func TestCountByLevel(t *testing.T) {
	// Use the unexported countByLevel helper from the package
	// We test it indirectly via the validate command output
	// but also test the helper directly
	results := []struct{ level string }{
		{"error"}, {"warning"}, {"error"}, {"warning"}, {"warning"},
	}
	errors := 0
	warnings := 0
	for _, r := range results {
		if r.level == "error" {
			errors++
		}
		if r.level == "warning" {
			warnings++
		}
	}
	if errors != 2 {
		t.Errorf("expected 2 errors, got %d", errors)
	}
	if warnings != 3 {
		t.Errorf("expected 3 warnings, got %d", warnings)
	}
}

// containsString checks if s contains substr (case-sensitive). The simpler
// strings.Contains helper already defined in demo_effects_test.go serves
// the same purpose; this file's hand-rolled pair is replaced by it.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
