package cmd

import (
	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// destructiveDryRun reports whether this invocation should execute in
// dry-run mode (FEAT-020 wave 2 consumer): registry entries flagged
// Destructive run dry unless the operator explicitly forces execution.
// Precedence: an explicit --dry-run always stays dry (even with --force,
// the safer reading wins); --force overrides only the destructive
// default; unregistered and non-destructive commands keep the operator's
// flag choice verbatim.
func destructiveDryRun(cliPath []string, force bool) bool {
	if DryRun {
		return true
	}
	if force {
		return false
	}
	c, ok := cmdmanifest.Load().ResolveCLI(cliPath...)
	return ok && c.Destructive
}
