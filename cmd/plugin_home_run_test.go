package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buckPluginHome isolates the plugin service in a temporary HOME (the
// plugins directory resolves to $HOME/.cosmoflare/plugins) and snapshots
// the DryRun global.
func buckPluginHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldDry := DryRun
	t.Cleanup(func() { DryRun = oldDry })
	return home
}

// buckPluginManifest writes a minimal valid plugin.yaml into dir.
func buckPluginManifest(t *testing.T, dir, name string) {
	t.Helper()
	manifest := "name: " + name + "\n" +
		"version: 1.2.3\n" +
		"description: test plugin\n" +
		"commands:\n" +
		"  - name: " + name + "\n" +
		"    description: run " + name + "\n" +
		"    binary: bin/" + name + "\n"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRunPluginList_EmptyHome verifies listing with no installed plugins
// succeeds offline and prints the empty-state hint.
func TestRunPluginList_EmptyHome(t *testing.T) {
	buckPluginHome(t)
	DryRun = false

	if err := runPluginList(pluginListCmd, nil); err != nil {
		t.Fatalf("runPluginList on empty plugins dir: %v", err)
	}
}

// TestRunPluginList_AfterInstall shows an installed plugin is discoverable
// through the list runner.
func TestRunPluginList_AfterInstall(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false
	src := filepath.Join(home, "src-plug")
	buckPluginManifest(t, src, "listed-plug")

	if err := runPluginInstall(pluginInstallCmd, []string{src}); err != nil {
		t.Fatalf("runPluginInstall: %v", err)
	}
	if err := runPluginList(pluginListCmd, nil); err != nil {
		t.Fatalf("runPluginList after install: %v", err)
	}
}

// TestRunPluginInit_ScaffoldsProject verifies init creates the documented
// directory layout under the plugins directory.
func TestRunPluginInit_ScaffoldsProject(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false

	if err := runPluginInit(pluginInitCmd, []string{"my-plugin"}); err != nil {
		t.Fatalf("runPluginInit: %v", err)
	}
	dir := filepath.Join(home, ".cosmoflare", "plugins", "my-plugin")
	for _, rel := range []string{"plugin.yaml", "README.md", filepath.Join("bin")} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected scaffold entry %q: %v", rel, err)
		}
	}
}

// TestRunPluginInit_DuplicateName verifies a second init with the same name
// is rejected instead of overwriting the existing scaffold.
func TestRunPluginInit_DuplicateName(t *testing.T) {
	buckPluginHome(t)
	DryRun = false

	if err := runPluginInit(pluginInitCmd, []string{"dupe"}); err != nil {
		t.Fatalf("first init: %v", err)
	}
	err := runPluginInit(pluginInitCmd, []string{"dupe"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already-exists error, got %v", err)
	}
}

// TestRunPluginInit_InvalidName verifies names with illegal characters are
// rejected by the validation rules before anything is created.
func TestRunPluginInit_InvalidName(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false

	for _, name := range []string{"bad name", "bad/name", "bad.name", "bad!"} {
		t.Run(name, func(t *testing.T) {
			err := runPluginInit(pluginInitCmd, []string{name})
			if err == nil || !strings.Contains(err.Error(), "invalid plugin name") {
				t.Fatalf("expected invalid-name error for %q, got %v", name, err)
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(home, ".cosmoflare", "plugins"))
	if err == nil && len(entries) != 0 {
		t.Errorf("invalid names must not create directories, found %d", len(entries))
	}
}

// TestRunPluginInit_DryRun verifies --dry-run scaffolds nothing.
func TestRunPluginInit_DryRun(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = true

	if err := runPluginInit(pluginInitCmd, []string{"ghost"}); err != nil {
		t.Fatalf("runPluginInit dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "plugins", "ghost")); !os.IsNotExist(err) {
		t.Errorf("dry-run must not create the plugin directory (stat err=%v)", err)
	}
}

// TestRunPluginInstall_LocalPath verifies installing from a local directory
// copies the manifest into the plugins directory.
func TestRunPluginInstall_LocalPath(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false
	src := filepath.Join(home, "src-plug")
	buckPluginManifest(t, src, "local-plug")

	if err := runPluginInstall(pluginInstallCmd, []string{src}); err != nil {
		t.Fatalf("runPluginInstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "plugins", "local-plug", "plugin.yaml")); err != nil {
		t.Fatalf("installed plugin.yaml missing: %v", err)
	}
}

// TestRunPluginInstall_MissingManifest verifies a source directory without
// plugin.yaml is rejected.
func TestRunPluginInstall_MissingManifest(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false
	empty := filepath.Join(home, "empty-src")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	err := runPluginInstall(pluginInstallCmd, []string{empty})
	if err == nil || !strings.Contains(err.Error(), "plugin.yaml") {
		t.Fatalf("expected missing-manifest error, got %v", err)
	}
}

// TestRunPluginInstall_DryRun verifies --dry-run installs nothing.
func TestRunPluginInstall_DryRun(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = true
	src := filepath.Join(home, "src-plug")
	buckPluginManifest(t, src, "dryrun-plug")

	if err := runPluginInstall(pluginInstallCmd, []string{src}); err != nil {
		t.Fatalf("runPluginInstall dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "plugins", "dryrun-plug")); !os.IsNotExist(err) {
		t.Errorf("dry-run must not install the plugin (stat err=%v)", err)
	}
}

// TestRunPluginRemove_NotInstalled verifies removing an unknown plugin
// fails with a clear error.
func TestRunPluginRemove_NotInstalled(t *testing.T) {
	buckPluginHome(t)
	DryRun = false

	err := runPluginRemove(pluginRemoveCmd, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected not-installed error, got %v", err)
	}
}

// TestRunPluginRemove_Installed verifies a scaffolded plugin can be removed
// and its directory disappears.
func TestRunPluginRemove_Installed(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false

	if err := runPluginInit(pluginInitCmd, []string{"temp-plug"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := runPluginRemove(pluginRemoveCmd, []string{"temp-plug"}); err != nil {
		t.Fatalf("runPluginRemove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "plugins", "temp-plug")); !os.IsNotExist(err) {
		t.Errorf("plugin directory should be gone after removal (stat err=%v)", err)
	}
}

// TestRunPluginRemove_DryRun verifies --dry-run keeps the plugin installed.
func TestRunPluginRemove_DryRun(t *testing.T) {
	home := buckPluginHome(t)
	DryRun = false
	if err := runPluginInit(pluginInitCmd, []string{"keepme"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	DryRun = true

	if err := runPluginRemove(pluginRemoveCmd, []string{"keepme"}); err != nil {
		t.Fatalf("runPluginRemove dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "plugins", "keepme")); err != nil {
		t.Fatalf("dry-run must keep the plugin installed: %v", err)
	}
}

// TestRunPluginRun_NotInstalled verifies running an unknown plugin fails
// through the runner's error wrapping.
func TestRunPluginRun_NotInstalled(t *testing.T) {
	buckPluginHome(t)

	err := runPluginRun(pluginRunCmd, []string{"missing-plug"})
	if err == nil || !strings.Contains(err.Error(), "plugin execution failed") {
		t.Fatalf("expected execution failure for unknown plugin, got %v", err)
	}
}

// TestRunPluginRun_MissingBinary verifies a scaffolded plugin without a
// compiled binary cannot be run.
func TestRunPluginRun_MissingBinary(t *testing.T) {
	buckPluginHome(t)
	DryRun = false
	if err := runPluginInit(pluginInitCmd, []string{"nobin"}); err != nil {
		t.Fatalf("init: %v", err)
	}

	err := runPluginRun(pluginRunCmd, []string{"nobin"})
	if err == nil {
		t.Fatal("expected error running a plugin whose binary does not exist")
	}
}

// TestRunPluginRun_EmptyArgsDoesNotPanic guards the arg slicing when only
// the plugin name is passed (no passthrough args).
func TestRunPluginRun_EmptyArgsDoesNotPanic(t *testing.T) {
	buckPluginHome(t)

	// Unknown plugin: must fail cleanly, never panic on args[1:].
	err := runPluginRun(pluginRunCmd, []string{"solo"})
	if err == nil {
		t.Fatal("expected error for unknown plugin")
	}
}
