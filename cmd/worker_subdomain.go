package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Workers.dev subdomain get/set (FEAT-021). Registered onto the worker
// command tree by registerWorkerSubdomainCmds; this file deliberately
// does not edit cmd/worker.go.

var workerSubdomainCmd = &cobra.Command{
	Use:   "subdomain",
	Short: "Manage the account's workers.dev subdomain",
	Long: `View or set the workers.dev subdomain for your account.

Every Worker is reachable at <worker>.<subdomain>.workers.dev once the
subdomain is enabled. Each account has exactly one subdomain; setting a
new one renames it for all Workers at once (existing URLs change, so
renaming is disruptive for deployed traffic).

Subdomain names may contain lowercase letters, digits, and hyphens only.

Commands:
  get  Show the current subdomain
  set  Create or rename the subdomain

Examples:
  cosmoflare worker subdomain get
  cosmoflare worker subdomain set my-team`,
}

var workerSubdomainGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the account's workers.dev subdomain",
	Long: `Show the workers.dev subdomain currently configured for the account.

Workers are served at <worker-name>.<subdomain>.workers.dev.

Examples:
  cosmoflare worker subdomain get
  cosmoflare worker subdomain get --json`,
	Args: cobra.NoArgs,
	RunE: runWorkerSubdomainGet,
}

var workerSubdomainSetCmd = &cobra.Command{
	Use:   "set NAME",
	Short: "Create or rename the account's workers.dev subdomain",
	Long: `Create or rename the workers.dev subdomain for the account.

NAME may contain lowercase letters, digits, and hyphens only. Renaming
changes the public URL of every Worker on the account immediately, so
treat it as a breaking change for anything already deployed.

Examples:
  cosmoflare worker subdomain set my-team
  cosmoflare worker subdomain set my-team --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerSubdomainSet,
}

// registerWorkerSubdomainCmds wires the "worker subdomain" command group
// onto the given parent command (the worker command).
func registerWorkerSubdomainCmds(parent *cobra.Command) {
	parent.AddCommand(workerSubdomainCmd)
	workerSubdomainCmd.AddCommand(workerSubdomainGetCmd, workerSubdomainSetCmd)
}

// validateSubdomainName applies the workers.dev naming rules before any
// service is built, so bad input fails fast offline.
func validateSubdomainName(name string) error {
	if name == "" {
		return fmt.Errorf("subdomain is required")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return fmt.Errorf("subdomain may only contain lowercase letters, digits, and hyphens")
	}
	return nil
}

// printSubdomainInfo renders one WorkersSubdomainInfo for humans.
func printSubdomainInfo(info *cosmoflare.WorkersSubdomainInfo) {
	printInfo("Subdomain: %s", info.Subdomain)
	enabled := "no"
	if info.Enabled {
		enabled = "yes"
	}
	printInfo("Enabled:   %s", enabled)
	printInfo("Workers serve at <name>.%s.workers.dev", info.Subdomain)
}

func runWorkerSubdomainGet(cmd *cobra.Command, args []string) error {
	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}
	info, err := svc.SubdomainGet(cmd.Context())
	if err != nil {
		return outErr("failed to get workers.dev subdomain", err)
	}
	return outResult(info, func() {
		printSubdomainInfo(info)
	})
}

func runWorkerSubdomainSet(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("subdomain is required")
	}
	name := strings.TrimSpace(args[0])
	if err := validateSubdomainName(name); err != nil {
		return outErr("%s", err)
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}
	info, err := svc.SubdomainSet(cmd.Context(), name)
	if err != nil {
		return outErr("failed to set workers.dev subdomain", err)
	}
	return outPayload("Workers.dev subdomain set", func() any {
		return info
	}, func() {
		printSuccess("Subdomain set to %q", info.Subdomain)
		printSubdomainInfo(info)
	})
}
