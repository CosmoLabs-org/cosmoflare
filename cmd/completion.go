/*
Package cmd provides the Cobra CLI commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(r2go2 completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ r2go2 completion bash > /etc/bash_completion.d/r2go2
  # macOS:
  $ r2go2 completion bash > /usr/local/etc/bash_completion.d/r2go2

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ r2go2 completion zsh > "${fpath[1]}/_r2go2"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ r2go2 completion fish | source

  # To load completions for each session, execute once:
  $ r2go2 completion fish > ~/.config/fish/completions/r2go2.fish

PowerShell:
  PS> r2go2 completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> r2go2 completion powershell > r2go2.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}