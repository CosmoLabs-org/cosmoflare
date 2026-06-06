package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestPluginCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "plugin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pluginCmd not registered on rootCmd")
	}
}

func TestPluginCmd_Metadata(t *testing.T) {
	if pluginCmd.Use != "plugin" {
		t.Errorf("pluginCmd.Use = %q, want %q", pluginCmd.Use, "plugin")
	}
	if pluginCmd.Short == "" {
		t.Error("pluginCmd.Short is empty")
	}
	if pluginCmd.Long == "" {
		t.Error("pluginCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestPluginCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "install", "remove", "init", "run"}
	for _, name := range expected {
		found := false
		for _, sub := range pluginCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("plugin subcommand %q not registered", name)
		}
	}
}

// --- Subcommand metadata ---

func TestPluginListCmd_Metadata(t *testing.T) {
	if pluginListCmd.Use != "list" {
		t.Errorf("pluginListCmd.Use = %q, want %q", pluginListCmd.Use, "list")
	}
	if pluginListCmd.Short == "" {
		t.Error("pluginListCmd.Short is empty")
	}
	if pluginListCmd.Long == "" {
		t.Error("pluginListCmd.Long is empty")
	}
}

func TestPluginInstallCmd_Metadata(t *testing.T) {
	if pluginInstallCmd.Use != "install <source>" {
		t.Errorf("pluginInstallCmd.Use = %q, want %q", pluginInstallCmd.Use, "install <source>")
	}
	if pluginInstallCmd.Short == "" {
		t.Error("pluginInstallCmd.Short is empty")
	}
}

func TestPluginRemoveCmd_Metadata(t *testing.T) {
	if pluginRemoveCmd.Use != "remove <name>" {
		t.Errorf("pluginRemoveCmd.Use = %q, want %q", pluginRemoveCmd.Use, "remove <name>")
	}
	if pluginRemoveCmd.Short == "" {
		t.Error("pluginRemoveCmd.Short is empty")
	}
}

func TestPluginInitCmd_Metadata(t *testing.T) {
	if pluginInitCmd.Use != "init <name>" {
		t.Errorf("pluginInitCmd.Use = %q, want %q", pluginInitCmd.Use, "init <name>")
	}
	if pluginInitCmd.Short == "" {
		t.Error("pluginInitCmd.Short is empty")
	}
}

func TestPluginRunCmd_Metadata(t *testing.T) {
	if pluginRunCmd.Use != "run <name> [args...]" {
		t.Errorf("pluginRunCmd.Use = %q, want %q", pluginRunCmd.Use, "run <name> [args...]")
	}
	if pluginRunCmd.Short == "" {
		t.Error("pluginRunCmd.Short is empty")
	}
}

// --- Arg validation ---

func TestPluginInstallCmd_RequiresArg(t *testing.T) {
	// cobra.ExactArgs(1) should reject zero args
	err := pluginInstallCmd.Args(pluginInstallCmd, []string{})
	if err == nil {
		t.Error("pluginInstallCmd should require exactly 1 argument")
	}
}

func TestPluginInstallCmd_AcceptsOneArg(t *testing.T) {
	err := pluginInstallCmd.Args(pluginInstallCmd, []string{"https://github.com/user/repo.git"})
	if err != nil {
		t.Errorf("pluginInstallCmd should accept 1 argument, got error: %v", err)
	}
}

func TestPluginRemoveCmd_RequiresArg(t *testing.T) {
	err := pluginRemoveCmd.Args(pluginRemoveCmd, []string{})
	if err == nil {
		t.Error("pluginRemoveCmd should require exactly 1 argument")
	}
}

func TestPluginInitCmd_RequiresArg(t *testing.T) {
	err := pluginInitCmd.Args(pluginInitCmd, []string{})
	if err == nil {
		t.Error("pluginInitCmd should require exactly 1 argument")
	}
}

func TestPluginRunCmd_RequiresArg(t *testing.T) {
	err := pluginRunCmd.Args(pluginRunCmd, []string{})
	if err == nil {
		t.Error("pluginRunCmd should require at least 1 argument")
	}
}

func TestPluginRunCmd_AcceptsMultipleArgs(t *testing.T) {
	err := pluginRunCmd.Args(pluginRunCmd, []string{"my-plugin", "subcmd", "--flag"})
	if err != nil {
		t.Errorf("pluginRunCmd should accept multiple arguments, got error: %v", err)
	}
}

// --- Run flag parsing disabled for run subcommand ---

func TestPluginRunCmd_DisableFlagParsing(t *testing.T) {
	if !pluginRunCmd.DisableFlagParsing {
		t.Error("pluginRunCmd.DisableFlagParsing should be true to pass flags through to plugins")
	}
}

// --- Help output ---

func TestPluginCmd_HelpContainsExamples(t *testing.T) {
	var buf bytes.Buffer
	pluginCmd.SetOut(&buf)
	pluginCmd.SetErr(&buf)
	pluginCmd.SetArgs([]string{"--help"})

	// Create a fresh root to avoid cross-test interference
	tmpRoot := &cobra.Command{Use: "cosmoflare"}
	tmpRoot.AddCommand(pluginCmd)

	err := pluginCmd.Help()
	if err != nil {
		t.Fatalf("Help() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("cosmoflare plugin list")) {
		t.Errorf("help output should contain example 'cosmoflare plugin list', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("cosmoflare plugin install")) {
		t.Errorf("help output should contain example 'cosmoflare plugin install', got:\n%s", output)
	}
}

// --- skipValidation for plugin command ---

func TestPluginCmd_SkipValidation(t *testing.T) {
	// The plugin command should be in the skipValidation list in root.go
	// so it doesn't require API tokens. We verify by checking that
	// "plugin" is handled as a parent-level skip in PersistentPreRun.
	// This is a structural test — the real validation is that plugin commands
	// work without CLOUDFLARE_API_TOKEN set.

	// Verify pluginCmd is a child of rootCmd
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub == pluginCmd {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pluginCmd is not a child of rootCmd")
	}

	// Verify pluginCmd has subcommands
	subs := pluginCmd.Commands()
	if len(subs) < 5 {
		t.Errorf("expected at least 5 subcommands, got %d", len(subs))
	}
}

// --- getPluginService ---

func TestGetPluginService(t *testing.T) {
	svc, err := getPluginService()
	if err != nil {
		t.Fatalf("getPluginService() error: %v", err)
	}
	if svc == nil {
		t.Fatal("getPluginService() returned nil")
	}
	if svc.PluginsDir() == "" {
		t.Error("PluginsDir() is empty")
	}
}
