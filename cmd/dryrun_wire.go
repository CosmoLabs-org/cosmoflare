package cmd

import (
	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// The cmdmanifest import is retained for callers composing paths from the
// registry; destructiveDryRun itself now delegates to destructive.go.
var _ = cmdmanifest.Load

// destructiveDryRun reports whether this invocation should execute in
// dry-run mode (FEAT-020 wave 2 consumer): registry entries flagged
// Destructive run dry unless the operator explicitly forces execution.
// Precedence: an explicit --dry-run always stays dry (even with --force,
// the safer reading wins); --force overrides only the destructive
// default; unregistered and non-destructive commands keep the operator's
// flag choice verbatim. The registry lookup delegates to the shared
// TASK-015 mechanism in destructive.go so runner-side and
// registration-side defaults can never drift apart.
func destructiveDryRun(cliPath []string, force bool) bool {
	if DryRun {
		return true
	}
	if force {
		return false
	}
	return destructiveByRegistry(cliPath)
}
