package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// cliPathOf derives the command's manifest path from the live cobra tree:
// CommandPath() minus the root name. A rename in the tree flows through
// automatically — no hand-typed literals to forget. A nil cmd (direct
// runner invocations in tests bypass cobra wiring) yields a nil path, which
// every consumer treats as unregistered.
func cliPathOf(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	segments := strings.Fields(cmd.CommandPath())
	if len(segments) > 0 {
		// Drop the root command name; the rest is the manifest path.
		segments = segments[1:]
	}
	return segments
}

// cliPathOfCmdOr derives the manifest path from cmd, falling back to the
// command's known path when cmd is nil — direct runner invocations in
// tests (e.g. runD1Delete(nil, args)) skip cobra wiring but still need the
// registry-derived destructive default. The fallback is the path as a
// space-separated string (NOT the *cobra.Command: a runner referencing its
// own command variable is an initialization cycle).
func cliPathOfCmdOr(cmd *cobra.Command, pathFallback string) []string {
	if cmd != nil {
		return cliPathOf(cmd)
	}
	return strings.Fields(pathFallback)
}

// destructiveByRegistry reports whether the registry marks the given CLI
// path destructive. An unregistered path is never destructive.
func destructiveByRegistry(cliPath []string) bool {
	c, ok := cmdmanifest.Load().ResolveCLI(cliPath...)
	return ok && c.Destructive
}

// withDestructiveDefaults wraps a mutating runner so registry-flagged
// destructive commands execute dry unless --force. When the registry marks
// the command destructive and the operator passed neither --force nor
// --dry-run, the wrapper runs the runner with DryRun temporarily true —
// the runner's existing "if DryRun" branch prints its own preview, so a
// newly registered destructive command gets safe defaults with ZERO
// runner-side wiring to remember. Non-destructive and already-dry runs
// pass through untouched, and DryRun is always restored on exit.
func withDestructiveDefaults(cmd *cobra.Command, runE func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(invoked *cobra.Command, args []string) error {
		if !shouldDryByDefault(invoked) {
			return runE(invoked, args)
		}
		saved := DryRun
		DryRun = true
		defer func() { DryRun = saved }()
		return runE(invoked, args)
	}
}

// shouldDryByDefault reports whether the wrapper must inject a dry run:
// only when the invoked command is registry-destructive and the operator
// passed neither --dry-run (already dry) nor --force (explicit override).
// A missing "force" flag reads as force=false.
func shouldDryByDefault(cmd *cobra.Command) bool {
	if DryRun {
		return false
	}
	if force, err := cmd.Flags().GetBool("force"); err == nil && force {
		return false
	}
	return destructiveByRegistry(cliPathOf(cmd))
}

// Registry-destructive commands, wrapped once here: a rename or a newly
// authored destructive command only needs a registry entry to inherit the
// dry-by-default behavior — no runner-side wiring to remember.
func init() {
	for _, c := range []*cobra.Command{
		bucketDeleteCmd,
		workerDeleteCmd,
		kvNamespaceDeleteCmd,
		d1DeleteCmd,
		wafListDeleteCmd,
		hyperdriveDeleteCmd,
		accountMemberRemoveCmd,
		logpushJobDeleteCmd,
		lbPoolDeleteCmd,
		lbMonitorDeleteCmd,
		tunnelDeleteCmd,
		tunnelCleanupCmd,
		waitingRoomDeleteCmd,
		spectrumAppDeleteCmd,
		pageShieldPolicyDeleteCmd,
		turnstileWidgetDeleteCmd,
		webAnalyticsSiteDeleteCmd,
	} {
		if c != nil && c.RunE != nil {
			c.RunE = withDestructiveDefaults(c, c.RunE)
		}
	}
}
