package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestExportCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "export [file]" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("exportCmd not registered on rootCmd")
	}
}

func TestExportCmd_Metadata(t *testing.T) {
	if exportCmd.Use != "export [file]" {
		t.Errorf("exportCmd.Use = %q, want %q", exportCmd.Use, "export [file]")
	}
	if exportCmd.Short == "" {
		t.Error("exportCmd.Short is empty")
	}
	if exportCmd.Long == "" {
		t.Error("exportCmd.Long is empty")
	}
}

// --- RunE handler wired ---

func TestExportCmd_RunE(t *testing.T) {
	if exportCmd.RunE == nil {
		t.Error("exportCmd.RunE is nil")
	}
}

// --- Flag registration ---

func TestExportCmd_Flags(t *testing.T) {
	expected := []string{"services", "format"}
	for _, name := range expected {
		if exportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on exportCmd", name)
		}
	}
}

func TestExportCmd_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"services", ""},
		{"format", "yaml"},
	}
	for _, tc := range cases {
		f := exportCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- runExport arg validation ---

func TestRunExport_DefaultFile(t *testing.T) {
	// Without API creds, the service creation will fail,
	// but we can verify the function parses args correctly.
	oldID := AccountID
	oldToken := APIToken
	AccountID = ""
	APIToken = ""
	defer func() {
		AccountID = oldID
		APIToken = oldToken
	}()

	err := runExport(exportCmd, []string{})
	if err == nil {
		t.Error("expected error without API credentials")
	}
}

func TestRunExport_CustomFile(t *testing.T) {
	oldID := AccountID
	oldToken := APIToken
	AccountID = ""
	APIToken = ""
	defer func() {
		AccountID = oldID
		APIToken = oldToken
	}()

	err := runExport(exportCmd, []string{"custom.yaml"})
	if err == nil {
		t.Error("expected error without API credentials")
	}
}

func TestRunExport_InvalidFormat(t *testing.T) {
	oldFormat := exportFormat
	exportFormat = "xml"
	defer func() {
		exportFormat = oldFormat
	}()

	err := runExport(exportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if err.Error() != `unsupported format "xml"; use yaml or json` {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestRunExport_InvalidServices(t *testing.T) {
	oldServices := exportServices
	oldFormat := exportFormat
	exportServices = "workers,invalid_svc"
	exportFormat = "yaml"
	defer func() {
		exportServices = oldServices
		exportFormat = oldFormat
	}()

	err := runExport(exportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid service name")
	}
}

func TestRunExport_DryRun(t *testing.T) {
	oldDryRun := DryRun
	oldFormat := exportFormat
	oldServices := exportServices
	DryRun = true
	exportFormat = "yaml"
	exportServices = ""
	defer func() {
		DryRun = oldDryRun
		exportFormat = oldFormat
		exportServices = oldServices
	}()

	// Dry run should succeed without API creds
	err := runExport(exportCmd, []string{"test.yaml"})
	if err != nil {
		t.Errorf("dry run should succeed, got error: %v", err)
	}
}

func TestRunExport_DryRunJSON(t *testing.T) {
	oldDryRun := DryRun
	oldJSON := JSONOutput
	oldFormat := exportFormat
	oldServices := exportServices
	DryRun = true
	JSONOutput = true
	exportFormat = "json"
	exportServices = ""
	defer func() {
		DryRun = oldDryRun
		JSONOutput = oldJSON
		exportFormat = oldFormat
		exportServices = oldServices
	}()

	err := runExport(exportCmd, []string{"test.json"})
	if err != nil {
		t.Errorf("dry run JSON should succeed, got error: %v", err)
	}
}

// --- Help text contains examples ---

func TestExportCmd_HelpContainsExamples(t *testing.T) {
	if exportCmd.Long == "" {
		t.Fatal("exportCmd.Long is empty")
	}
	expected := []string{"cosmoflare export", "--services", "--format"}
	for _, s := range expected {
		found := false
		if len(exportCmd.Long) > 0 {
			for i := 0; i <= len(exportCmd.Long)-len(s); i++ {
				if exportCmd.Long[i:i+len(s)] == s {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("exportCmd.Long should contain %q", s)
		}
	}
}

// --- Verify export integrates correctly with cobra ---

func TestExportCmd_CobraStructure(t *testing.T) {
	var found *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "export" {
			found = sub
			break
		}
	}
	if found == nil {
		t.Fatal("export command not found on rootCmd")
	}
	if found.Args != nil {
		// Should accept optional args (file path)
		// cobra default is nil = accept any
	}
}

// --- Verify temp file write in dry run ---

func TestRunExport_DryRunDoesNotCreateFile(t *testing.T) {
	oldDryRun := DryRun
	oldFormat := exportFormat
	oldServices := exportServices
	DryRun = true
	exportFormat = "yaml"
	exportServices = ""
	defer func() {
		DryRun = oldDryRun
		exportFormat = oldFormat
		exportServices = oldServices
	}()

	dir := t.TempDir()
	path := filepath.Join(dir, "should-not-exist.yaml")
	_ = runExport(exportCmd, []string{path})

	if _, err := os.Stat(path); err == nil {
		t.Error("dry run should not create the output file")
	}
}
