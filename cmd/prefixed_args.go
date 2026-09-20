package cmd

import (
	"github.com/spf13/cobra"
)

// cobra.Args-level resource-prefix mechanism (TASK-011 pilot).
//
// Historically every runner that takes a resource NAME as its first
// positional remembered to wrap the read: `applyResourcePrefix(args[0])`.
// The declarations below move that convention to command definition so
// runners read args[0] bare and a new command cannot forget the scoping.

// prefixedResourceArgs wraps a cobra positional-args validator so the
// first argument is treated as a profile-prefixed resource name: the
// prefix is applied (idempotently) before the wrapped validator runs.
// Runners then read args[0] bare — the convention is declared once at
// command definition instead of remembered per runner.
//
// The args slice is rewritten in place: cobra passes the very same slice
// to the Args validator and then to RunE, so the rewrite is exactly what
// the runner observes. Direct runner calls (tests, internal reuse) keep
// receiving whatever args their caller built, unmodified.
func prefixedResourceArgs(wrapped cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			args[0] = applyResourcePrefix(args[0])
		}
		return wrapped(cmd, args)
	}
}

// prefixedFlag applies the profile prefix to a name sourced from a flag
// rather than a positional (e.g. worker routes --script): call it at the
// top of the runner, before any use of the flag value. It returns the
// prefixed flag value; an unknown or unset flag yields the empty string,
// which applyResourcePrefix leaves empty ("not provided", not a resource
// named after the bare prefix).
func prefixedFlag(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}
	return applyResourcePrefix(v)
}
