package cosmoflare

import (
	"fmt"
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

// ── loadManifest edge cases ───────────────────────────────────────────────────

// TestLoadManifestMissingName verifies that a plugin.yaml without a name field is rejected.
func TestLoadManifestMissingName(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "no-name")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write manifest without a name field
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte("version: 1.0.0\ndescription: No name\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svc, _ := NewPluginService(dir)
	_, err := svc.loadManifest("no-name")
	if err == nil {
		t.Fatal("expected error for manifest missing 'name' field")
	}
}

// TestLoadManifestInvalidYAML verifies that malformed YAML in plugin.yaml is rejected.
func TestLoadManifestInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad-yaml")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write intentionally invalid YAML (tab character where not allowed)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte("name: bad\n\tversion: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	svc, _ := NewPluginService(dir)
	_, err := svc.loadManifest("bad-yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// TestLoadManifestNotFound verifies that a non-existent plugin returns an error.
func TestLoadManifestNotFound(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewPluginService(dir)
	_, err := svc.loadManifest("phantom")
	if err == nil {
		t.Fatal("expected error for missing plugin directory")
	}
}

// ── Get — path correctness ────────────────────────────────────────────────────

// TestPluginServiceGetReturnsCorrectPath verifies Get populates the Path field.
func TestPluginServiceGetReturnsCorrectPath(t *testing.T) {
	dir, svc := setupTestPlugin(t, "path-check")

	info, err := svc.Get("path-check")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "path-check")
	if info.Path != want {
		t.Errorf("PluginInfo.Path = %q, want %q", info.Path, want)
	}
}

// ── List — extra coverage ─────────────────────────────────────────────────────

// TestPluginServiceListMultiplePlugins verifies List returns all installed plugins.
func TestPluginServiceListMultiplePlugins(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewPluginService(dir)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		pluginDir := filepath.Join(dir, name)
		os.MkdirAll(pluginDir, 0755)
		manifest := "name: " + name + "\nversion: 1.0.0\ndescription: Plugin " + name + "\n"
		os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	}

	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(plugins))
	}
}

// TestPluginServiceListSkipsFiles verifies List ignores non-directory entries.
func TestPluginServiceListSkipsFiles(t *testing.T) {
	dir, svc := setupTestPlugin(t, "real-plugin")

	// Place a plain file alongside the plugin directory
	if err := os.WriteFile(filepath.Join(dir, "not-a-plugin.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	plugins, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin (file should be skipped), got %d", len(plugins))
	}
	if plugins[0].Manifest.Name != "real-plugin" {
		t.Errorf("plugin name = %q, want %q", plugins[0].Manifest.Name, "real-plugin")
	}
}

// ── Init — content verification ───────────────────────────────────────────────

// TestPluginServiceInitCommandsInManifest verifies Init writes correct commands in plugin.yaml.
func TestPluginServiceInitCommandsInManifest(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewPluginService(dir)

	_, err := svc.Init("cmd-check")
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := svc.loadManifest("cmd-check")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(manifest.Commands))
	}
	cmd := manifest.Commands[0]
	if cmd.Name != "cmd-check" {
		t.Errorf("command name = %q, want %q", cmd.Name, "cmd-check")
	}
	if cmd.Binary != "bin/cmd-check" {
		t.Errorf("command binary = %q, want %q", cmd.Binary, "bin/cmd-check")
	}
}

// TestPluginServiceInitREADMEContent verifies Init writes a README referencing the plugin name.
func TestPluginServiceInitREADMEContent(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewPluginService(dir)

	pluginDir, err := svc.Init("readme-test")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(pluginDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !pluginContains(content, "readme-test") {
		t.Error("README.md does not mention the plugin name")
	}
}

// pluginContains is a helper to check for a substring in plugin test assertions.
func pluginContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// ── Run — additional scenarios ────────────────────────────────────────────────

// TestPluginServiceRunFallsBackToFirstCommand verifies that an unrecognised first arg
// does not consume the arg and routes to the first command binary instead.
func TestPluginServiceRunFallsBackToFirstCommand(t *testing.T) {
	dir, svc := setupTestPlugin(t, "fallback-cmd")

	// Create the binary expected by the single "greet" command
	binPath := filepath.Join(dir, "fallback-cmd", "bin", "greet")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho hi"), 0755); err != nil {
		t.Fatal(err)
	}

	var capturedBinary string
	var capturedArgs []string
	svc.execCommandFunc = func(binary string, args []string) ([]byte, error) {
		capturedBinary = binary
		capturedArgs = args
		return []byte("hi"), nil
	}

	// "unknown" does not match any command name, so falls back to first command
	// and the arg is passed through as-is.
	_, err := svc.Run("fallback-cmd", []string{"unknown", "--flag"})
	if err != nil {
		t.Fatal(err)
	}

	if capturedBinary != binPath {
		t.Errorf("binary = %q, want %q", capturedBinary, binPath)
	}
	// The args should be passed unchanged (no subcommand consumed)
	if len(capturedArgs) != 2 {
		t.Errorf("args = %v, expected 2 elements [unknown --flag]", capturedArgs)
	}
}

// TestPluginServiceRunExecError verifies that an exec error is wrapped and returned.
func TestPluginServiceRunExecError(t *testing.T) {
	dir, svc := setupTestPlugin(t, "fail-exec")

	binPath := filepath.Join(dir, "fail-exec", "bin", "greet")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho fail"), 0755); err != nil {
		t.Fatal(err)
	}

	svc.execCommandFunc = func(binary string, args []string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 1")
	}

	_, err := svc.Run("fail-exec", nil)
	if err == nil {
		t.Fatal("expected error when exec fails")
	}
}

// TestPluginServiceRunNotInstalledPlugin verifies Run returns error for unknown plugin.
func TestPluginServiceRunNotInstalledPlugin(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewPluginService(dir)
	_, err := svc.Run("ghost-plugin", nil)
	if err == nil {
		t.Fatal("expected error for non-installed plugin")
	}
}

// ── Install — git failure cleanup ────────────────────────────────────────────

// TestPluginServiceInstallGitCloneFailure verifies partial clone is cleaned up on failure.
func TestPluginServiceInstallGitCloneFailure(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, _ := NewPluginService(pluginsDir)

	svc.gitCloneFunc = func(url, dest string) error {
		// Simulate partial clone by creating the directory then failing
		os.MkdirAll(dest, 0755)
		return fmt.Errorf("git clone failed: connection refused")
	}

	_, err := svc.Install("https://github.com/example/cosmoflare-broken.git")
	if err == nil {
		t.Fatal("expected error on git clone failure")
	}

	// Cleanup: the dest dir should have been removed
	destDir := filepath.Join(pluginsDir, "broken")
	if _, statErr := os.Stat(destDir); !os.IsNotExist(statErr) {
		t.Error("expected partial clone directory to be cleaned up after failure")
	}
}

// TestPluginServiceInstallGitAlreadyInstalled verifies Install fails when plugin already exists.
func TestPluginServiceInstallGitAlreadyInstalled(t *testing.T) {
	dir, svc := setupTestPlugin(t, "my-plugin")

	svc.gitCloneFunc = func(url, dest string) error {
		return nil // should never be called
	}

	_, err := svc.Install("https://github.com/example/cosmoflare-my-plugin.git")
	if err == nil {
		t.Fatal("expected error when plugin is already installed")
	}
	_ = dir
}

// ── Install local — error paths ───────────────────────────────────────────────

// TestPluginServiceInstallLocalMissingManifest verifies Install fails when source has no plugin.yaml.
func TestPluginServiceInstallLocalMissingManifest(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, _ := NewPluginService(pluginsDir)

	srcDir := t.TempDir() // no plugin.yaml inside

	_, err := svc.Install(srcDir)
	if err == nil {
		t.Fatal("expected error when source has no plugin.yaml")
	}
}

// TestPluginServiceInstallLocalInvalidYAML verifies Install fails for invalid plugin.yaml.
func TestPluginServiceInstallLocalInvalidYAML(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, _ := NewPluginService(pluginsDir)

	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "plugin.yaml"), []byte("name: bad\n\tversion: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Install(srcDir)
	if err == nil {
		t.Fatal("expected error for invalid plugin.yaml")
	}
}

// TestPluginServiceInstallLocalMissingName verifies Install fails when plugin.yaml has no name.
func TestPluginServiceInstallLocalMissingName(t *testing.T) {
	pluginsDir := t.TempDir()
	svc, _ := NewPluginService(pluginsDir)

	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "plugin.yaml"), []byte("version: 1.0.0\ndescription: No name\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Install(srcDir)
	if err == nil {
		t.Fatal("expected error when plugin.yaml is missing 'name' field")
	}
}

// ── pluginNameFromURL — edge cases ────────────────────────────────────────────

// TestPluginNameFromURLEdgeCases verifies edge-case URL patterns.
func TestPluginNameFromURLEdgeCases(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// Trailing slash is stripped
		{"https://github.com/user/cosmoflare-trim/", "trim"},
		// No path after host — last component is the domain itself
		{"https://github.com", "github.com"},
		// cosmoflare- prefix is stripped only once
		{"https://github.com/user/cosmoflare-cosmoflare-double.git", "cosmoflare-double"},
		// SSH SCP-style without ://
		{"git@github.com:user/cosmoflare-ssh.git", "ssh"},
	}
	for _, tc := range tests {
		got := pluginNameFromURL(tc.url)
		if got != tc.want {
			t.Errorf("pluginNameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

// ── PluginsDir — accessor ─────────────────────────────────────────────────────

// TestPluginServicePluginsDirAccessor verifies PluginsDir returns the configured path.
func TestPluginServicePluginsDirAccessor(t *testing.T) {
	want := "/custom/plugins/path"
	svc, err := NewPluginService(want)
	if err != nil {
		t.Fatal(err)
	}
	if svc.PluginsDir() != want {
		t.Errorf("PluginsDir() = %q, want %q", svc.PluginsDir(), want)
	}
}

// ── PluginManifest / PluginCommand struct fields ──────────────────────────────

// TestPluginManifestAllFields verifies all fields survive a roundtrip through loadManifest.
func TestPluginManifestAllFields(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "full-manifest")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := `name: full-manifest
version: 3.2.1
description: Full manifest test
author: CosmoLabs
commands:
  - name: cmd1
    description: First command
    binary: bin/cmd1
  - name: cmd2
    description: Second command
    binary: bin/cmd2
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	svc, _ := NewPluginService(dir)
	manifest, err := svc.loadManifest("full-manifest")
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Name != "full-manifest" {
		t.Errorf("Name = %q, want %q", manifest.Name, "full-manifest")
	}
	if manifest.Version != "3.2.1" {
		t.Errorf("Version = %q, want %q", manifest.Version, "3.2.1")
	}
	if manifest.Description != "Full manifest test" {
		t.Errorf("Description = %q, want %q", manifest.Description, "Full manifest test")
	}
	if manifest.Author != "CosmoLabs" {
		t.Errorf("Author = %q, want %q", manifest.Author, "CosmoLabs")
	}
	if len(manifest.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(manifest.Commands))
	}
	if manifest.Commands[0].Name != "cmd1" || manifest.Commands[0].Binary != "bin/cmd1" {
		t.Errorf("Commands[0] = %+v", manifest.Commands[0])
	}
	if manifest.Commands[1].Name != "cmd2" || manifest.Commands[1].Binary != "bin/cmd2" {
		t.Errorf("Commands[1] = %+v", manifest.Commands[1])
	}
}

// ── isGitURL — additional cases ───────────────────────────────────────────────

// TestIsGitURLAdditionalCases covers HTTP (non-S) and plain .git suffix.
func TestIsGitURLAdditionalCases(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"http://github.com/user/repo", true},  // http:// contains "://"
		{"file:///local/repo", true},            // file:// contains "://"
		{"just-a-name", false},                  // plain name
		{"path/to/dir", false},                  // relative path, no ://, no .git
		{"cosmoflare-plugin.git", true},         // ends in .git
	}
	for _, tc := range tests {
		got := isGitURL(tc.source)
		if got != tc.want {
			t.Errorf("isGitURL(%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}
