package cosmoflare

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to create a temporary plugins directory with a named plugin
func setupTestPlugin(t *testing.T, name string) (string, *PluginService) {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, name)
	binDir := filepath.Join(pluginDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifest := []byte(`name: ` + name + `
version: 1.0.0
description: Test plugin
author: tester
commands:
  - name: greet
    description: Say hello
    binary: bin/greet
`)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, svc
}

// TestPluginServiceNew verifies constructor with explicit and default paths.
func TestPluginServiceNew(t *testing.T) {
	t.Run("explicit path", func(t *testing.T) {
		svc, err := NewPluginService("/tmp/test-plugins")
		if err != nil {
			t.Fatal(err)
		}
		if svc.PluginsDir() != "/tmp/test-plugins" {
			t.Errorf("PluginsDir() = %q, want %q", svc.PluginsDir(), "/tmp/test-plugins")
		}
	})

	t.Run("default path uses home dir", func(t *testing.T) {
		svc, err := NewPluginService("")
		if err != nil {
			t.Fatal(err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".cosmoflare", "plugins")
		if svc.PluginsDir() != want {
			t.Errorf("PluginsDir() = %q, want %q", svc.PluginsDir(), want)
		}
	})
}

// TestPluginServiceListEmpty verifies List returns empty slice when no plugins exist.
func TestPluginServiceListEmpty(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

// TestPluginServiceListNonExistentDir verifies List returns empty when dir doesn't exist.
func TestPluginServiceListNonExistentDir(t *testing.T) {
	svc, err := NewPluginService(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

// TestPluginServiceListWithPlugins verifies List discovers installed plugins.
func TestPluginServiceListWithPlugins(t *testing.T) {
	_, svc := setupTestPlugin(t, "my-plugin")

	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.Name != "my-plugin" {
		t.Errorf("plugin name = %q, want %q", plugins[0].Manifest.Name, "my-plugin")
	}
	if plugins[0].Manifest.Version != "1.0.0" {
		t.Errorf("plugin version = %q, want %q", plugins[0].Manifest.Version, "1.0.0")
	}
	if plugins[0].Manifest.Author != "tester" {
		t.Errorf("plugin author = %q, want %q", plugins[0].Manifest.Author, "tester")
	}
	if len(plugins[0].Manifest.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(plugins[0].Manifest.Commands))
	}
}

// TestPluginServiceListSkipsInvalidPlugins verifies List skips dirs without valid manifests.
func TestPluginServiceListSkipsInvalidPlugins(t *testing.T) {
	dir, svc := setupTestPlugin(t, "valid-plugin")

	// Create an invalid plugin directory (no plugin.yaml)
	if err := os.MkdirAll(filepath.Join(dir, "broken-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 valid plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.Name != "valid-plugin" {
		t.Errorf("plugin name = %q, want %q", plugins[0].Manifest.Name, "valid-plugin")
	}
}

// TestPluginServiceGet verifies Get returns a specific plugin.
func TestPluginServiceGet(t *testing.T) {
	_, svc := setupTestPlugin(t, "test-get")

	info, err := svc.Get("test-get")
	if err != nil {
		t.Fatal(err)
	}
	if info.Manifest.Name != "test-get" {
		t.Errorf("plugin name = %q, want %q", info.Manifest.Name, "test-get")
	}
}

// TestPluginServiceGetNotFound verifies Get returns error for missing plugin.
func TestPluginServiceGetNotFound(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
}

// TestPluginServiceGetEmptyName verifies Get rejects empty name.
func TestPluginServiceGetEmptyName(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestPluginServiceRemove verifies Remove deletes a plugin.
func TestPluginServiceRemove(t *testing.T) {
	dir, svc := setupTestPlugin(t, "removable")

	// Verify it exists
	_, err := svc.Get("removable")
	if err != nil {
		t.Fatal(err)
	}

	// Remove it
	if err := svc.Remove("removable"); err != nil {
		t.Fatal(err)
	}

	// Verify it's gone
	if _, err := os.Stat(filepath.Join(dir, "removable")); !os.IsNotExist(err) {
		t.Error("plugin directory should be removed")
	}
}

// TestPluginServiceRemoveNotInstalled verifies Remove fails for missing plugin.
func TestPluginServiceRemoveNotInstalled(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error when removing nonexistent plugin")
	}
}

// TestPluginServiceRemoveEmptyName verifies Remove rejects empty name.
func TestPluginServiceRemoveEmptyName(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.Remove("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestPluginServiceInit verifies Init scaffolds a new plugin project.
func TestPluginServiceInit(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}

	pluginDir, err := svc.Init("my-new-plugin")
	if err != nil {
		t.Fatal(err)
	}

	// Verify directory was created
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		t.Fatal("plugin directory not created")
	}

	// Verify plugin.yaml exists and is valid
	manifest, err := svc.loadManifest("my-new-plugin")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "my-new-plugin" {
		t.Errorf("manifest name = %q, want %q", manifest.Name, "my-new-plugin")
	}
	if manifest.Version != "0.1.0" {
		t.Errorf("manifest version = %q, want %q", manifest.Version, "0.1.0")
	}

	// Verify bin/ directory exists
	binDir := filepath.Join(pluginDir, "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Error("bin directory not created")
	}

	// Verify README exists
	readmePath := filepath.Join(pluginDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md not created")
	}
}

// TestPluginServiceInitAlreadyExists verifies Init fails for existing plugin.
func TestPluginServiceInitAlreadyExists(t *testing.T) {
	_, svc := setupTestPlugin(t, "existing")

	_, err := svc.Init("existing")
	if err == nil {
		t.Fatal("expected error when initializing existing plugin")
	}
}

// TestPluginServiceInitInvalidName verifies Init rejects invalid names.
func TestPluginServiceInitInvalidName(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"valid-name", false},
		{"valid_name", false},
		{"ValidName123", false},
		{"", true},
		{"invalid name", true},   // space
		{"invalid/name", true},   // slash
		{"invalid.name", true},   // dot
		{"invalid@name", true},   // at sign
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Init(tc.name)
			if tc.wantErr && err == nil {
				t.Errorf("Init(%q) should have returned error", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Init(%q) returned unexpected error: %v", tc.name, err)
			}
			// Clean up for valid names so they don't interfere
			if !tc.wantErr {
				os.RemoveAll(filepath.Join(dir, tc.name))
			}
		})
	}
}

// TestPluginServiceInitEmptyName verifies Init rejects empty name.
func TestPluginServiceInitEmptyName(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Init("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestPluginServiceRunEmptyName verifies Run rejects empty name.
func TestPluginServiceRunEmptyName(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Run("", nil)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestPluginServiceRunNoCommands verifies Run fails when plugin has no commands.
func TestPluginServiceRunNoCommands(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "no-cmds")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("name: no-cmds\nversion: 1.0.0\ndescription: Test\n")
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Run("no-cmds", nil)
	if err == nil {
		t.Fatal("expected error when plugin has no commands")
	}
}

// TestPluginServiceRunBinaryNotFound verifies Run fails when binary doesn't exist.
func TestPluginServiceRunBinaryNotFound(t *testing.T) {
	_, svc := setupTestPlugin(t, "missing-bin")

	_, err := svc.Run("missing-bin", nil)
	if err == nil {
		t.Fatal("expected error when binary does not exist")
	}
}

// TestPluginServiceRunWithMockExec verifies Run calls the binary with args.
func TestPluginServiceRunWithMockExec(t *testing.T) {
	dir, svc := setupTestPlugin(t, "runnable")

	// Create a fake binary
	binPath := filepath.Join(dir, "runnable", "bin", "greet")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	// Mock the exec function
	var capturedBinary string
	var capturedArgs []string
	svc.execCommandFunc = func(binary string, args []string) ([]byte, error) {
		capturedBinary = binary
		capturedArgs = args
		return []byte("hello world"), nil
	}

	output, err := svc.Run("runnable", []string{"--name", "test"})
	if err != nil {
		t.Fatal(err)
	}

	if string(output) != "hello world" {
		t.Errorf("output = %q, want %q", string(output), "hello world")
	}
	if capturedBinary != binPath {
		t.Errorf("binary = %q, want %q", capturedBinary, binPath)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "--name" || capturedArgs[1] != "test" {
		t.Errorf("args = %v, want [--name test]", capturedArgs)
	}
}

// TestPluginServiceRunSubcommandRouting verifies Run routes to specific subcommands.
func TestPluginServiceRunSubcommandRouting(t *testing.T) {
	dir, svc := setupTestPlugin(t, "multi-cmd")

	// Add a second command to the manifest
	manifest := []byte(`name: multi-cmd
version: 1.0.0
description: Multi-command plugin
author: tester
commands:
  - name: greet
    description: Say hello
    binary: bin/greet
  - name: farewell
    description: Say goodbye
    binary: bin/farewell
`)
	if err := os.WriteFile(filepath.Join(dir, "multi-cmd", "plugin.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}

	// Create fake binaries
	for _, name := range []string{"greet", "farewell"} {
		binPath := filepath.Join(dir, "multi-cmd", "bin", name)
		if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho "+name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	var capturedBinary string
	svc.execCommandFunc = func(binary string, args []string) ([]byte, error) {
		capturedBinary = binary
		return []byte("ok"), nil
	}

	// Run with subcommand name "farewell"
	_, err := svc.Run("multi-cmd", []string{"farewell", "--flag"})
	if err != nil {
		t.Fatal(err)
	}

	wantBin := filepath.Join(dir, "multi-cmd", "bin", "farewell")
	if capturedBinary != wantBin {
		t.Errorf("binary = %q, want %q", capturedBinary, wantBin)
	}
}

// TestPluginServiceInstallLocal verifies Install from a local path.
func TestPluginServiceInstallLocal(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, err := NewPluginService(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a source plugin
	srcDir := t.TempDir()
	srcPlugin := filepath.Join(srcDir, "local-plugin")
	srcBin := filepath.Join(srcPlugin, "bin")
	if err := os.MkdirAll(srcBin, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("name: local-plugin\nversion: 2.0.0\ndescription: Local install test\nauthor: local\ncommands:\n  - name: run\n    description: Run it\n    binary: bin/run\n")
	if err := os.WriteFile(filepath.Join(srcPlugin, "plugin.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcBin, "run"), []byte("#!/bin/sh\necho hi"), 0755); err != nil {
		t.Fatal(err)
	}

	info, err := svc.Install(srcPlugin)
	if err != nil {
		t.Fatal(err)
	}

	if info.Manifest.Name != "local-plugin" {
		t.Errorf("installed plugin name = %q, want %q", info.Manifest.Name, "local-plugin")
	}
	if info.Manifest.Version != "2.0.0" {
		t.Errorf("installed plugin version = %q, want %q", info.Manifest.Version, "2.0.0")
	}

	// Verify the files were copied
	copiedBin := filepath.Join(pluginsDir, "local-plugin", "bin", "run")
	if _, err := os.Stat(copiedBin); os.IsNotExist(err) {
		t.Error("binary file was not copied to plugins directory")
	}
}

// TestPluginServiceInstallGitMocked verifies Install from a git URL using mock.
func TestPluginServiceInstallGitMocked(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, err := NewPluginService(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Mock git clone to just create the directory with a manifest
	svc.gitCloneFunc = func(url, dest string) error {
		if err := os.MkdirAll(filepath.Join(dest, "bin"), 0755); err != nil {
			return err
		}
		manifest := []byte("name: awesome\nversion: 1.0.0\ndescription: Awesome plugin\nauthor: community\ncommands:\n  - name: awesome\n    description: Be awesome\n    binary: bin/awesome\n")
		return os.WriteFile(filepath.Join(dest, "plugin.yaml"), manifest, 0644)
	}

	info, err := svc.Install("https://github.com/example/cosmoflare-awesome.git")
	if err != nil {
		t.Fatal(err)
	}

	if info.Manifest.Name != "awesome" {
		t.Errorf("installed plugin name = %q, want %q", info.Manifest.Name, "awesome")
	}
	if info.Path != filepath.Join(pluginsDir, "awesome") {
		t.Errorf("installed path = %q, want %q", info.Path, filepath.Join(pluginsDir, "awesome"))
	}
}

// TestPluginServiceInstallAlreadyExists verifies Install fails for existing plugin.
func TestPluginServiceInstallAlreadyExists(t *testing.T) {
	dir, svc := setupTestPlugin(t, "already-here")

	// Try to install from local path that has same name
	srcDir := t.TempDir()
	manifest := []byte("name: already-here\nversion: 1.0.0\ndescription: Duplicate\n")
	if err := os.WriteFile(filepath.Join(srcDir, "plugin.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Install(srcDir)
	if err == nil {
		t.Fatal("expected error when installing over existing plugin")
	}
	_ = dir
}

// TestPluginServiceInstallEmptySource verifies Install rejects empty source.
func TestPluginServiceInstallEmptySource(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewPluginService(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Install("")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

// TestPluginNameFromURL verifies URL-to-name extraction.
func TestPluginNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/cosmoflare-analytics.git", "analytics"},
		{"https://github.com/user/cosmoflare-analytics", "analytics"},
		{"https://github.com/user/my-plugin.git", "my-plugin"},
		{"https://github.com/user/my-plugin", "my-plugin"},
		{"git@github.com:user/cosmoflare-exporter.git", "exporter"},
	}
	for _, tc := range tests {
		got := pluginNameFromURL(tc.url)
		if got != tc.want {
			t.Errorf("pluginNameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

// TestIsGitURL verifies git URL detection.
func TestIsGitURL(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"https://github.com/user/repo.git", true},
		{"git://github.com/user/repo.git", true},
		{"ssh://git@github.com/user/repo.git", true},
		{"/local/path/to/plugin", false},
		{"./relative/path", false},
		{"repo.git", true}, // ends in .git
	}
	for _, tc := range tests {
		got := isGitURL(tc.source)
		if got != tc.want {
			t.Errorf("isGitURL(%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}

// TestCopyDir verifies directory copying.
func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0755)
	os.WriteFile(filepath.Join(src, "top.txt"), []byte("top"), 0644)
	os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("nested"), 0644)

	dst := filepath.Join(t.TempDir(), "copy")
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	// Verify files
	data, err := os.ReadFile(filepath.Join(dst, "top.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "top" {
		t.Errorf("top.txt content = %q, want %q", string(data), "top")
	}

	data, err = os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "nested" {
		t.Errorf("sub/nested.txt content = %q, want %q", string(data), "nested")
	}
}
