package cosmoflare

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginCommand describes a single command exposed by a plugin.
type PluginCommand struct {
	Name        string `yaml:"name"        json:"name"`
	Description string `yaml:"description" json:"description"`
	Binary      string `yaml:"binary"      json:"binary"`
}

// PluginManifest represents the contents of a plugin.yaml file.
type PluginManifest struct {
	Name        string          `yaml:"name"        json:"name"`
	Version     string          `yaml:"version"     json:"version"`
	Description string          `yaml:"description" json:"description"`
	Author      string          `yaml:"author"      json:"author"`
	Commands    []PluginCommand `yaml:"commands"     json:"commands"`
}

// PluginInfo contains the manifest plus the installation path.
type PluginInfo struct {
	Manifest PluginManifest `json:"manifest"`
	Path     string         `json:"path"`
}

// PluginService manages cosmoflare community plugins.
// Plugins are stored in ~/.cosmoflare/plugins/<name>/ with a plugin.yaml manifest.
type PluginService struct {
	pluginsDir string

	// gitCloneFunc is the function used to clone git repos. Override in tests.
	gitCloneFunc func(url, dest string) error

	// execCommandFunc is the function used to execute plugin binaries. Override in tests.
	execCommandFunc func(binary string, args []string) ([]byte, error)
}

// NewPluginService creates a new PluginService.
// If pluginsDir is empty, defaults to ~/.cosmoflare/plugins.
func NewPluginService(pluginsDir string) (*PluginService, error) {
	if pluginsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, newError("PluginService.New", "failed to determine home directory", err)
		}
		pluginsDir = filepath.Join(home, ".cosmoflare", "plugins")
	}

	return &PluginService{
		pluginsDir:      pluginsDir,
		gitCloneFunc:    defaultGitClone,
		execCommandFunc: defaultExecCommand,
	}, nil
}

// defaultGitClone runs git clone to install a plugin from a URL.
func defaultGitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// defaultExecCommand runs a binary with arguments and returns the output.
func defaultExecCommand(binary string, args []string) ([]byte, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// PluginsDir returns the base plugins directory path.
func (s *PluginService) PluginsDir() string {
	return s.pluginsDir
}

// List returns all installed plugins by scanning the plugins directory.
func (s *PluginService) List() ([]PluginInfo, error) {
	entries, err := os.ReadDir(s.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginInfo{}, nil
		}
		return nil, newError("PluginService.List", "failed to read plugins directory", err)
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := s.loadManifest(entry.Name())
		if err != nil {
			// Skip directories without valid manifests
			continue
		}
		plugins = append(plugins, PluginInfo{
			Manifest: *manifest,
			Path:     filepath.Join(s.pluginsDir, entry.Name()),
		})
	}

	return plugins, nil
}

// Get returns a single plugin by name, or an error if not found.
func (s *PluginService) Get(name string) (*PluginInfo, error) {
	if name == "" {
		return nil, validationError("PluginService.Get", "plugin name is required")
	}

	manifest, err := s.loadManifest(name)
	if err != nil {
		return nil, err
	}

	return &PluginInfo{
		Manifest: *manifest,
		Path:     filepath.Join(s.pluginsDir, name),
	}, nil
}

// Install installs a plugin from a git URL or local path.
// For git URLs (containing "://" or ending in ".git"), it clones the repo.
// For local paths, it copies the directory contents.
func (s *PluginService) Install(source string) (*PluginInfo, error) {
	if source == "" {
		return nil, validationError("PluginService.Install", "source is required (git URL or local path)")
	}

	// Ensure plugins directory exists
	if err := os.MkdirAll(s.pluginsDir, 0755); err != nil {
		return nil, newError("PluginService.Install", "failed to create plugins directory", err)
	}

	if isGitURL(source) {
		return s.installFromGit(source)
	}
	return s.installFromLocal(source)
}

// Remove uninstalls a plugin by name.
func (s *PluginService) Remove(name string) error {
	if name == "" {
		return validationError("PluginService.Remove", "plugin name is required")
	}

	pluginDir := filepath.Join(s.pluginsDir, name)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return newError("PluginService.Remove", fmt.Sprintf("plugin %q is not installed", name), nil)
	}

	if err := os.RemoveAll(pluginDir); err != nil {
		return newError("PluginService.Remove", fmt.Sprintf("failed to remove plugin %q", name), err)
	}

	return nil
}

// Init scaffolds a new plugin project at the given path inside the plugins directory.
func (s *PluginService) Init(name string) (string, error) {
	if name == "" {
		return "", validationError("PluginService.Init", "plugin name is required")
	}

	// Validate plugin name (alphanumeric, hyphens, underscores)
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return "", validationError("PluginService.Init",
				fmt.Sprintf("invalid plugin name %q: only alphanumeric, hyphens, and underscores allowed", name))
		}
	}

	// Ensure plugins directory exists
	if err := os.MkdirAll(s.pluginsDir, 0755); err != nil {
		return "", newError("PluginService.Init", "failed to create plugins directory", err)
	}

	pluginDir := filepath.Join(s.pluginsDir, name)
	if _, err := os.Stat(pluginDir); err == nil {
		return "", newError("PluginService.Init", fmt.Sprintf("plugin %q already exists at %s", name, pluginDir), nil)
	}

	// Create directory structure
	binDir := filepath.Join(pluginDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", newError("PluginService.Init", "failed to create plugin directories", err)
	}

	// Write plugin.yaml manifest
	manifest := PluginManifest{
		Name:        name,
		Version:     "0.1.0",
		Description: fmt.Sprintf("A cosmoflare plugin: %s", name),
		Author:      "",
		Commands: []PluginCommand{
			{
				Name:        name,
				Description: fmt.Sprintf("Run %s", name),
				Binary:      fmt.Sprintf("bin/%s", name),
			},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return "", newError("PluginService.Init", "failed to marshal plugin.yaml", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return "", newError("PluginService.Init", "failed to write plugin.yaml", err)
	}

	// Write a README
	readme := fmt.Sprintf("# %s\n\nA cosmoflare community plugin.\n\n## Installation\n\n```bash\ncosmoflare plugin install %s\n```\n\n## Usage\n\n```bash\ncosmoflare plugin run %s\n```\n", name, name, name)
	readmePath := filepath.Join(pluginDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		return "", newError("PluginService.Init", "failed to write README.md", err)
	}

	return pluginDir, nil
}

// Run executes a plugin command by name with the given arguments.
func (s *PluginService) Run(name string, args []string) ([]byte, error) {
	if name == "" {
		return nil, validationError("PluginService.Run", "plugin name is required")
	}

	manifest, err := s.loadManifest(name)
	if err != nil {
		return nil, err
	}

	if len(manifest.Commands) == 0 {
		return nil, newError("PluginService.Run",
			fmt.Sprintf("plugin %q has no commands defined in plugin.yaml", name), nil)
	}

	// Find the command binary — use the first command by default,
	// or match by name if the first arg looks like a subcommand.
	var binaryPath string
	if len(args) > 0 {
		for _, cmd := range manifest.Commands {
			if cmd.Name == args[0] {
				binaryPath = filepath.Join(s.pluginsDir, name, cmd.Binary)
				args = args[1:] // consume the subcommand name
				break
			}
		}
	}
	if binaryPath == "" {
		// Default to first command
		binaryPath = filepath.Join(s.pluginsDir, name, manifest.Commands[0].Binary)
	}

	// Check binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, newError("PluginService.Run",
			fmt.Sprintf("plugin binary not found: %s (run 'go build' or check plugin.yaml)", binaryPath), nil)
	}

	// Check binary is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		return nil, newError("PluginService.Run", "failed to stat plugin binary", err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return nil, newError("PluginService.Run",
			fmt.Sprintf("plugin binary is not executable: %s (chmod +x)", binaryPath), nil)
	}

	output, err := s.execCommandFunc(binaryPath, args)
	if err != nil {
		return nil, newError("PluginService.Run",
			fmt.Sprintf("plugin %q exited with error", name), err)
	}

	return output, nil
}

// loadManifest reads and parses the plugin.yaml for a named plugin.
func (s *PluginService) loadManifest(name string) (*PluginManifest, error) {
	manifestPath := filepath.Join(s.pluginsDir, name, "plugin.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newError("PluginService.loadManifest",
				fmt.Sprintf("plugin %q not found (no plugin.yaml at %s)", name, manifestPath), nil)
		}
		return nil, newError("PluginService.loadManifest",
			fmt.Sprintf("failed to read plugin.yaml for %q", name), err)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, newError("PluginService.loadManifest",
			fmt.Sprintf("invalid plugin.yaml for %q: %v", name, err), nil)
	}

	if manifest.Name == "" {
		return nil, newError("PluginService.loadManifest",
			fmt.Sprintf("plugin.yaml for %q is missing required 'name' field", name), nil)
	}

	return &manifest, nil
}

// installFromGit clones a git repo into the plugins directory.
func (s *PluginService) installFromGit(url string) (*PluginInfo, error) {
	// Derive plugin name from URL
	name := pluginNameFromURL(url)
	if name == "" {
		return nil, validationError("PluginService.Install",
			fmt.Sprintf("cannot determine plugin name from URL %q", url))
	}

	destDir := filepath.Join(s.pluginsDir, name)
	if _, err := os.Stat(destDir); err == nil {
		return nil, newError("PluginService.Install",
			fmt.Sprintf("plugin %q is already installed at %s (remove first)", name, destDir), nil)
	}

	if err := s.gitCloneFunc(url, destDir); err != nil {
		// Clean up partial clone
		os.RemoveAll(destDir)
		return nil, newError("PluginService.Install",
			fmt.Sprintf("failed to clone %q", url), err)
	}

	// Verify plugin.yaml exists
	manifest, err := s.loadManifest(name)
	if err != nil {
		os.RemoveAll(destDir)
		return nil, newError("PluginService.Install",
			fmt.Sprintf("cloned repo does not contain a valid plugin.yaml"), err)
	}

	return &PluginInfo{
		Manifest: *manifest,
		Path:     destDir,
	}, nil
}

// installFromLocal copies a local plugin directory into the plugins directory.
func (s *PluginService) installFromLocal(source string) (*PluginInfo, error) {
	// Verify source exists and has plugin.yaml
	sourceManifest := filepath.Join(source, "plugin.yaml")
	data, err := os.ReadFile(sourceManifest)
	if err != nil {
		return nil, newError("PluginService.Install",
			fmt.Sprintf("source path %q does not contain a plugin.yaml", source), err)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, newError("PluginService.Install",
			fmt.Sprintf("invalid plugin.yaml at %q", source), err)
	}

	if manifest.Name == "" {
		return nil, validationError("PluginService.Install",
			"plugin.yaml is missing required 'name' field")
	}

	destDir := filepath.Join(s.pluginsDir, manifest.Name)
	if _, err := os.Stat(destDir); err == nil {
		return nil, newError("PluginService.Install",
			fmt.Sprintf("plugin %q is already installed (remove first)", manifest.Name), nil)
	}

	// Copy directory tree
	if err := copyDir(source, destDir); err != nil {
		os.RemoveAll(destDir)
		return nil, newError("PluginService.Install", "failed to copy plugin files", err)
	}

	return &PluginInfo{
		Manifest: manifest,
		Path:     destDir,
	}, nil
}

// isGitURL returns true if the source looks like a git URL.
func isGitURL(source string) bool {
	return strings.Contains(source, "://") || strings.HasSuffix(source, ".git")
}

// pluginNameFromURL extracts the repository name from a git URL.
func pluginNameFromURL(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")
	// Remove trailing slashes
	url = strings.TrimRight(url, "/")
	// Get last path component
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	// Remove cosmoflare- prefix if present (convention)
	name = strings.TrimPrefix(name, "cosmoflare-")
	return name
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}
