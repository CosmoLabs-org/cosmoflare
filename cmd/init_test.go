package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("initCmd not registered on rootCmd")
	}
}

func TestInitCmd_Metadata(t *testing.T) {
	if initCmd.Use != "init" {
		t.Errorf("initCmd.Use = %q, want %q", initCmd.Use, "init")
	}
	if initCmd.Short == "" {
		t.Error("initCmd.Short is empty")
	}
}

func TestInitCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"yes", "false"},
		{"template", ""},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := initCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on initCmd", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestInitCmd_YesShorthand(t *testing.T) {
	f := initCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("--yes flag not found")
	}
	if f.Shorthand != "y" {
		t.Errorf("--yes shorthand = %q, want %q", f.Shorthand, "y")
	}
}

func TestInitScaffold_CreatesConfig(t *testing.T) {
	dir := t.TempDir()

	err := scaffoldProject(dir, "worker", false)
	if err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	configPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .cosmoflare.yaml to be created")
	}
}

func TestInitScaffold_WorkerTemplate(t *testing.T) {
	dir := t.TempDir()

	err := scaffoldProject(dir, "worker", false)
	if err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".cosmoflare.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("expected non-empty config file")
	}
}

func TestInitScaffold_PagesTemplate(t *testing.T) {
	dir := t.TempDir()

	err := scaffoldProject(dir, "pages", false)
	if err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	configPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .cosmoflare.yaml to be created")
	}
}

func TestInitScaffold_R2Template(t *testing.T) {
	dir := t.TempDir()

	err := scaffoldProject(dir, "r2", false)
	if err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	configPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .cosmoflare.yaml to be created")
	}
}

func TestInitScaffold_FullTemplate(t *testing.T) {
	dir := t.TempDir()

	err := scaffoldProject(dir, "full", false)
	if err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	configPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .cosmoflare.yaml to be created")
	}
}

func TestInitScaffold_WontOverwrite(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".cosmoflare.yaml")

	os.WriteFile(configPath, []byte("existing"), 0644)

	err := scaffoldProject(dir, "worker", false)
	if err == nil {
		t.Error("expected error when config already exists")
	}
}

func TestInitScaffold_OverwriteWhenForced(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".cosmoflare.yaml")

	os.WriteFile(configPath, []byte("existing"), 0644)

	err := scaffoldProject(dir, "worker", true)
	if err != nil {
		t.Fatalf("expected force overwrite to succeed, got: %v", err)
	}
}
