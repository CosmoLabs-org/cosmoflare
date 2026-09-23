/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var completionNoDescriptions bool

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion scripts for R2Go2.

Supported shells: bash, zsh, fish, powershell.

Completions cover all R2Go2 commands including:
  bucket    — create, list, get, update, delete, exists, import
  object    — upload, download, copy, delete, list, get, head
  worker    — deploy, list, logs, delete, rollback
  kv        — namespace create/delete/get/list, key get/put/delete/list
  config    — init, show, set, get
  auth      — login, logout, status
  backup    — create, restore, list
  copy      — object and bucket copy operations

Dynamic completions:
  Bucket names are auto-completed for bucket subcommands that accept
  a bucket-name argument (requires CLOUDFLARE_ACCOUNT_ID and
  CLOUDFLARE_API_TOKEN environment variables to be set).

To load completions:

Bash:
  $ source <(cosmoflare completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ cosmoflare completion bash > /etc/bash_completion.d/cosmoflare
  # macOS:
  $ cosmoflare completion bash > /usr/local/etc/bash_completion.d/cosmoflare

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ cosmoflare completion zsh > "${fpath[1]}/_r2go2"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ cosmoflare completion fish | source

  # To load completions for each session, execute once:
  $ cosmoflare completion fish > ~/.config/fish/completions/cosmoflare.fish

PowerShell:
  PS> cosmoflare completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> cosmoflare completion powershell > cosmoflare.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		includeDesc := !completionNoDescriptions
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			if includeDesc {
				cmd.Root().GenZshCompletion(os.Stdout)
			} else {
				cmd.Root().GenZshCompletionNoDesc(os.Stdout)
			}
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, includeDesc)
		case "powershell":
			if includeDesc {
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			} else {
				cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		}
	},
}

func init() {
	completionCmd.Flags().BoolVar(&completionNoDescriptions, "no-descriptions", false,
		"Disable completion descriptions (shorter output, supported: zsh, fish, powershell)")
}

// RegisterCompletionFlags sets up dynamic shell completions for commands
// that accept bucket names as positional arguments.
func RegisterCompletionFlags() {
	bucketNameCommands := []*cobra.Command{
		bucketGetCmd,
		bucketUpdateCmd,
		bucketDeleteCmd,
		bucketExistsCmd,
	}

	for _, c := range bucketNameCommands {
		c.ValidArgsFunction = completeBucketNames
	}
}

// completeBucketNames calls the R2 API to list buckets for shell completion.
// It gracefully returns no completions if credentials are not configured,
// so completion scripts still work in unauthenticated environments.
func completeBucketNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := newClientFromEnv()
	if err != nil {
		// Credentials not configured — return empty rather than an error
		// so the shell still shows other completions (flags, etc.)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, b := range buckets {
		names = append(names, b.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// bucketSubCommands is a helper that returns the list of bucket subcommand
// names, used by the bucket parent command for argument completion.
func bucketSubCommands() []string {
	return []string{"create", "list", "get", "update", "delete", "exists", "import"}
}
