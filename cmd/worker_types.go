package cmd

import (
	"fmt"
	"os"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// Worker types .d.ts generation (FEAT-021 wave 3). Registered onto the
// worker command tree by registerWorkerTypesCmds; the .d.ts content itself
// comes from the pure generator in pkg/cosmoflare/worker_types.go, with
// live settings fetched via WorkerService.SettingsGet.

var workerTypesOut string

var workerTypesCmd = &cobra.Command{
	Use:   "types <name>",
	Short: "Generate a worker-configuration.d.ts from a Worker's bindings",
	Long: `Generate TypeScript environment types for a Worker.

Fetches the Worker's current settings (bindings) and renders a
worker-configuration.d.ts declaring the runtime interfaces (KVNamespace,
R2Bucket, D1Database, Queue, Fetcher) plus an Env interface typed from the
Worker's bindings. Output is stable: bindings are emitted sorted by name.

With no --out, the generated content is printed to stdout. With --out, it
is written to the given path (mode 0644).

Examples:
  cosmoflare worker types api-gateway
  cosmoflare worker types api-gateway --out=worker-configuration.d.ts
  cosmoflare worker types api-gateway --json`,
	Example: `  # Print the .d.ts to stdout
  cosmoflare worker types api-gateway

  # Write it to a file for tsc to pick up
  cosmoflare worker types api-gateway --out=worker-configuration.d.ts`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerTypes,
}

// registerWorkerTypesCmds wires the "worker types" command onto parent.
// Idempotent: safe to call multiple times (flags are guarded).
func registerWorkerTypesCmds(parent *cobra.Command) {
	parent.AddCommand(workerTypesCmd)
	if workerTypesCmd.Flags().Lookup("out") == nil {
		workerTypesCmd.Flags().StringVar(&workerTypesOut, "out", "", "Write the generated .d.ts to this path instead of stdout")
	}
}

func runWorkerTypes(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := applyResourcePrefix(args[0])

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	settings, err := svc.SettingsGet(cmd.Context(), name)
	if err != nil {
		return outErr(fmt.Sprintf("failed to fetch settings for worker %q", name), err)
	}

	content := cosmoflare.GenerateWorkerTypes(settings)

	if workerTypesOut == "" {
		return outResult(content, func() {
			fmt.Fprint(cmd.OutOrStdout(), content)
		})
	}

	if err := os.WriteFile(workerTypesOut, []byte(content), 0o644); err != nil {
		return outErr("failed to write types file", err)
	}
	return outPayload(fmt.Sprintf("Worker types written to %s", workerTypesOut), func() any {
		return map[string]string{"path": workerTypesOut}
	}, func() {
		printSuccess("Worker types written to %s", workerTypesOut)
	})
}
