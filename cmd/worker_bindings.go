package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Worker bindings inspection (FEAT-021 wave 2). Reads the Worker record via
// WorkerService.Get, which populates the Bindings slice, and prints them as
// a table (plain mode) or the raw binding array (--json).

var workerBindingsCmd = &cobra.Command{
	Use:   "bindings <name>",
	Short: "List the bindings attached to a Worker",
	Long: `List the bindings attached to a Worker script.

Each binding connects the Worker to a resource: a KV namespace (kv), an
R2 bucket (r2), a D1 database (d1), a queue (queue), a plain variable
(var), or a secret (secret_text). Use --json for scripting.`,
	Example: `  # Show a Worker's bindings
  cosmoflare worker bindings api-gateway

  # Machine-readable output
  cosmoflare worker bindings api-gateway --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerBindings,
}

// registerWorkerBindingsCmds attaches the bindings command to the worker
// tree. Flag registration is guarded so re-registration (tests, repeated
// wiring) stays idempotent.
func registerWorkerBindingsCmds(parent *cobra.Command) {
	parent.AddCommand(workerBindingsCmd)
}

func runWorkerBindings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := applyResourcePrefix(args[0])

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	worker, err := svc.Get(cmd.Context(), name)
	if err != nil {
		return outErr(fmt.Sprintf("failed to get worker %q", name), err)
	}

	return outResult(worker.Bindings, func() {
		printBindingsTable(name, worker.Bindings)
	})
}

func printBindingsTable(name string, bindings []cosmoflare.WorkerBinding) {
	if len(bindings) == 0 {
		printInfo("No bindings attached to worker %q", name)
		return
	}
	printInfo("Bindings for worker %q:", name)
	for _, b := range bindings {
		printInfo("  %-24s  %-12s  %s", b.Name, b.Type, b.ID)
	}
	printInfo("Total: %d binding(s)", len(bindings))
}
