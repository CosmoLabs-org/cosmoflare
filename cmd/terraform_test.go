package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestTerraformCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "terraform" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("terraformCmd not registered on rootCmd")
	}
}

func TestTerraformCmd_Metadata(t *testing.T) {
	if terraformCmd.Use != "terraform" {
		t.Errorf("terraformCmd.Use = %q, want %q", terraformCmd.Use, "terraform")
	}
	if terraformCmd.Short == "" {
		t.Error("terraformCmd.Short is empty")
	}
	if terraformCmd.Long == "" {
		t.Error("terraformCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestTerraformCmd_Subcommands(t *testing.T) {
	expected := []string{"export", "import-block"}
	for _, name := range expected {
		found := false
		for _, sub := range terraformCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("terraform subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestTerraformCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		terraformExportCmd,
		terraformImportBlockCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestTerraformExport_Flags(t *testing.T) {
	expected := []string{"services", "format", "provider-version"}
	for _, name := range expected {
		if terraformExportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on terraformExportCmd", name)
		}
	}
}

func TestTerraformExport_FlagDefaults(t *testing.T) {
	f := terraformExportCmd.Flags()

	format := f.Lookup("format")
	if format.DefValue != "hcl" {
		t.Errorf("--format default = %q, want %q", format.DefValue, "hcl")
	}

	pv := f.Lookup("provider-version")
	if pv.DefValue != "~> 4.0" {
		t.Errorf("--provider-version default = %q, want %q", pv.DefValue, "~> 4.0")
	}
}

func TestTerraformImportBlock_Flags(t *testing.T) {
	expected := []string{"services"}
	for _, name := range expected {
		if terraformImportBlockCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on terraformImportBlockCmd", name)
		}
	}
}

// --- Help text ---

func TestTerraformExport_HelpContainsExamples(t *testing.T) {
	long := terraformExportCmd.Long
	if long == "" {
		t.Fatal("terraformExportCmd.Long is empty")
	}
	examples := []string{
		"cosmoflare terraform export",
		"--services",
		"--provider-version",
	}
	for _, ex := range examples {
		found := false
		for _, line := range []string{long} {
			if contains(line, ex) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("help text missing example/keyword %q", ex)
		}
	}
}

func TestTerraformImportBlock_HelpContainsExamples(t *testing.T) {
	long := terraformImportBlockCmd.Long
	if long == "" {
		t.Fatal("terraformImportBlockCmd.Long is empty")
	}
	if !contains(long, "cosmoflare terraform import-block") {
		t.Error("import-block help missing usage example")
	}
}

// contains checks if s contains substr (simple helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Runner behavior (offline) ---

// terraformRunGlobals snapshots and zeroes the credentials and terraform
// flag variables the runners read, so tests cannot leak state.
func terraformRunGlobals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldServices, oldFormat, oldVersion := tfServices, tfFormat, tfProviderVersion
	AccountID, APIToken = "", ""
	tfServices, tfFormat, tfProviderVersion = nil, "hcl", "~> 4.0"
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
		tfServices, tfFormat, tfProviderVersion = oldServices, oldFormat, oldVersion
	})
}

// TestGetTerraformExporter validates credential handling in the exporter
// factory: both fields are required and valid values yield an exporter.
func TestGetTerraformExporter(t *testing.T) {
	terraformRunGlobals(t)

	t.Run("missing account ID", func(t *testing.T) {
		AccountID, APIToken = "", "token"
		exp, err := getTerraformExporter()
		if err == nil || !strings.Contains(err.Error(), "account ID is required") {
			t.Fatalf("expected account ID error, got %v", err)
		}
		if exp != nil {
			t.Error("expected nil exporter on error")
		}
	})

	t.Run("missing API token", func(t *testing.T) {
		AccountID, APIToken = "account", ""
		exp, err := getTerraformExporter()
		if err == nil || !strings.Contains(err.Error(), "API token is required") {
			t.Fatalf("expected token error, got %v", err)
		}
		if exp != nil {
			t.Error("expected nil exporter on error")
		}
	})

	t.Run("valid credentials", func(t *testing.T) {
		AccountID, APIToken = "account", "token"
		exp, err := getTerraformExporter()
		if err != nil {
			t.Fatalf("expected exporter, got error: %v", err)
		}
		if exp == nil {
			t.Fatal("expected non-nil exporter")
		}
	})
}

// TestRunTerraformExport_NoCreds verifies the export runner fails at
// exporter construction when credentials are absent.
func TestRunTerraformExport_NoCreds(t *testing.T) {
	terraformRunGlobals(t)

	err := runTerraformExport(terraformExportCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create terraform exporter") {
		t.Fatalf("expected exporter construction error, got %v", err)
	}
}

// TestRunTerraformExport_DryRun verifies the dry-run path short-circuits
// after exporter construction and emits the payload without exporting.
func TestRunTerraformExport_DryRun(t *testing.T) {
	terraformRunGlobals(t)
	AccountID, APIToken = "account", "token"
	DryRun = true
	JSONOutput = false
	tfServices = []string{"workers", "dns"}

	if err := runTerraformExport(terraformExportCmd, []string{t.TempDir()}); err != nil {
		t.Fatalf("runTerraformExport(DryRun) returned error: %v", err)
	}
}

// TestRunTerraformExport_DryRunJSON verifies the dry-run path in JSON mode.
func TestRunTerraformExport_DryRunJSON(t *testing.T) {
	terraformRunGlobals(t)
	AccountID, APIToken = "account", "token"
	DryRun = true
	JSONOutput = true

	if err := runTerraformExport(terraformExportCmd, nil); err != nil {
		t.Fatalf("runTerraformExport(DryRun+JSON) returned error: %v", err)
	}
}

// TestRunTerraformImportBlock_NoCreds verifies the import-block runner fails
// at exporter construction when credentials are absent.
func TestRunTerraformImportBlock_NoCreds(t *testing.T) {
	terraformRunGlobals(t)

	err := runTerraformImportBlock(terraformImportBlockCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create terraform exporter") {
		t.Fatalf("expected exporter construction error, got %v", err)
	}
}

// TestRunTerraformImportBlock_DryRun verifies the import-block dry-run path
// short-circuits after exporter construction.
func TestRunTerraformImportBlock_DryRun(t *testing.T) {
	terraformRunGlobals(t)
	AccountID, APIToken = "account", "token"
	DryRun = true
	JSONOutput = false

	if err := runTerraformImportBlock(terraformImportBlockCmd, nil); err != nil {
		t.Fatalf("runTerraformImportBlock(DryRun) returned error: %v", err)
	}
}
