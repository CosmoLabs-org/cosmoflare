package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage cosmoflare community plugins",
	Long: `Manage cosmoflare community extensions (plugins).

Plugins extend cosmoflare with community-built commands. Each plugin lives in
~/.cosmoflare/plugins/<name>/ and contains a plugin.yaml manifest describing
its commands.

Commands:
  list      List installed plugins
  install   Install a plugin from a git URL or local path
  remove    Uninstall a plugin
  init      Scaffold a new plugin project
  run       Execute a plugin command

Plugin Manifest (plugin.yaml):
  name:        Plugin name (required)
  version:     Semantic version
  description: What the plugin does
  author:      Plugin author
  commands:    List of commands with name, description, and binary path

Examples:
  cosmoflare plugin list                                    # List installed plugins
  cosmoflare plugin list --json                             # List as JSON
  cosmoflare plugin install https://github.com/user/repo    # Install from git
  cosmoflare plugin install ./my-plugin                     # Install from local path
  cosmoflare plugin remove my-plugin                        # Uninstall a plugin
  cosmoflare plugin init my-plugin                          # Scaffold a new plugin
  cosmoflare plugin run my-plugin                           # Run a plugin
  cosmoflare plugin run my-plugin subcommand --flag value   # Run with args`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `List all installed cosmoflare plugins.

Scans ~/.cosmoflare/plugins/ for directories with valid plugin.yaml manifests
and displays plugin name, version, description, and command count.

Examples:
  cosmoflare plugin list         # Human-readable table
  cosmoflare plugin list --json  # Machine-readable JSON output`,
	RunE: runPluginList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a plugin from a git URL or local path",
	Long: `Install a cosmoflare plugin from a git repository or local directory.

For git URLs (containing "://" or ending in ".git"), the repo is cloned
into ~/.cosmoflare/plugins/. The plugin name is derived from the URL
(stripping "cosmoflare-" prefix if present).

For local paths, the directory is copied into the plugins folder. The
plugin name is read from the plugin.yaml manifest.

The source must contain a valid plugin.yaml file.

Examples:
  cosmoflare plugin install https://github.com/user/cosmoflare-analytics.git
  cosmoflare plugin install git@github.com:user/my-plugin.git
  cosmoflare plugin install ./my-local-plugin
  cosmoflare plugin install /path/to/plugin`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Uninstall a plugin",
	Long: `Remove an installed cosmoflare plugin.

Deletes the plugin directory from ~/.cosmoflare/plugins/<name>/.
This is a permanent action and cannot be undone.

Examples:
  cosmoflare plugin remove my-plugin
  cosmoflare plugin remove analytics`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginRemove,
}

var pluginInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Scaffold a new plugin project",
	Long: `Create a new cosmoflare plugin project with the standard directory structure.

Generates:
  ~/.cosmoflare/plugins/<name>/
    plugin.yaml    — Plugin manifest with metadata and commands
    bin/           — Directory for compiled binaries
    README.md      — Basic documentation template

Plugin names must contain only alphanumeric characters, hyphens, and underscores.

Examples:
  cosmoflare plugin init my-analytics
  cosmoflare plugin init custom-exporter`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInit,
}

var pluginRunCmd = &cobra.Command{
	Use:   "run <name> [args...]",
	Short: "Execute a plugin command",
	Long: `Run an installed plugin's command.

If the plugin defines multiple commands in plugin.yaml, you can specify
which command to run as the first argument after the plugin name. If no
subcommand is specified, the first command in the manifest is used.

Any additional arguments are passed through to the plugin binary.

Examples:
  cosmoflare plugin run my-plugin                  # Run default command
  cosmoflare plugin run my-plugin --verbose        # Pass flags to plugin
  cosmoflare plugin run multi-cmd subcommand       # Run specific subcommand
  cosmoflare plugin run analytics export --format csv`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runPluginRun,
}

func init() {
	rootCmd.AddCommand(pluginCmd)

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginInitCmd)
	pluginCmd.AddCommand(pluginRunCmd)
}

// getPluginService creates a PluginService with the default plugins directory.
func getPluginService() (*cosmoflare.PluginService, error) {
	return cosmoflare.NewPluginService("")
}

func runPluginList(cmd *cobra.Command, args []string) error {
	svc, err := getPluginService()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin service: %w", err)
	}

	plugins, err := svc.List()
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list plugins: %v", err))
		}
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("plugins listed", map[string]interface{}{
			"plugins": plugins,
			"count":   len(plugins),
			"path":    svc.PluginsDir(),
		})
	}

	if len(plugins) == 0 {
		printInfo("No plugins installed")
		printInfo("Install one: cosmoflare plugin install <git-url-or-path>")
		printInfo("Scaffold one: cosmoflare plugin init <name>")
		return nil
	}

	fmt.Printf("\nInstalled Plugins (%d)\n", len(plugins))
	fmt.Println(strings.Repeat("─", 60))

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tCOMMANDS\tDESCRIPTION")
	fmt.Fprintln(w, "────\t───────\t────────\t───────────")
	for _, p := range plugins {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			p.Manifest.Name,
			p.Manifest.Version,
			len(p.Manifest.Commands),
			p.Manifest.Description,
		)
	}
	w.Flush()

	fmt.Printf("\nPlugins directory: %s\n", svc.PluginsDir())
	return nil
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	source := args[0]

	svc, err := getPluginService()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin service: %w", err)
	}

	if DryRun {
		printWarning("DRY RUN: Would install plugin from %s", source)
		return nil
	}

	info, err := svc.Install(source)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to install plugin: %v", err))
		}
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("plugin installed", map[string]interface{}{
			"name":    info.Manifest.Name,
			"version": info.Manifest.Version,
			"path":    info.Path,
		})
	}

	printSuccess("Plugin %q v%s installed at %s", info.Manifest.Name, info.Manifest.Version, info.Path)
	if len(info.Manifest.Commands) > 0 {
		printInfo("Run it: cosmoflare plugin run %s", info.Manifest.Name)
	}
	return nil
}

func runPluginRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getPluginService()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin service: %w", err)
	}

	if DryRun {
		printWarning("DRY RUN: Would remove plugin %q", name)
		return nil
	}

	if err := svc.Remove(name); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to remove plugin: %v", err))
		}
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("plugin removed", map[string]interface{}{
			"name": name,
		})
	}

	printSuccess("Plugin %q removed", name)
	return nil
}

func runPluginInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getPluginService()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin service: %w", err)
	}

	if DryRun {
		printWarning("DRY RUN: Would scaffold plugin %q", name)
		return nil
	}

	pluginDir, err := svc.Init(name)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to scaffold plugin: %v", err))
		}
		return fmt.Errorf("failed to scaffold plugin: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("plugin scaffolded", map[string]interface{}{
			"name": name,
			"path": pluginDir,
		})
	}

	printSuccess("Plugin %q scaffolded at %s", name, pluginDir)
	fmt.Println()
	printInfo("Files created:")
	printInfo("  %s/plugin.yaml   — Plugin manifest", pluginDir)
	printInfo("  %s/bin/          — Place your compiled binary here", pluginDir)
	printInfo("  %s/README.md     — Plugin documentation", pluginDir)
	fmt.Println()
	printInfo("Next steps:")
	printInfo("  1. Edit plugin.yaml with your plugin's metadata")
	printInfo("  2. Build your binary and place it in bin/")
	printInfo("  3. Test: cosmoflare plugin run %s", name)
	return nil
}

func runPluginRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	var pluginArgs []string
	if len(args) > 1 {
		pluginArgs = args[1:]
	}

	svc, err := getPluginService()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin service: %w", err)
	}

	output, err := svc.Run(name, pluginArgs)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("plugin execution failed: %v", err))
		}
		return fmt.Errorf("plugin execution failed: %w", err)
	}

	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}
