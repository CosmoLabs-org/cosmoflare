package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// workerSecretCmd is the "worker secret" group command. Registration is
// exported (registerWorkerSecretCmds) so the orchestrator wires it onto
// the worker command without this file editing cmd/worker.go.
var workerSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage Worker secrets",
	Long: `Manage secret bindings on Cloudflare Workers.

Secret values are write-only: they are uploaded to Cloudflare and never
displayed, logged, or returned by any command. Only secret NAMES appear
in output.

Commands:
  put     Create or update a single secret
  delete  Delete a secret
  list    List secret names bound to a Worker
  bulk    Upload many secrets from a JSON file

Examples:
  cosmoflare worker secret put my-worker API_TOKEN --value=...`,
}

var workerSecretPutCmd = &cobra.Command{
	Use:   "put WORKER KEY",
	Short: "Create or update a Worker secret",
	Long: `Create or update a single secret on a Worker.

The value comes from --value or, when omitted, from stdin (read with a
newline terminator). The value is never echoed back or included in any
output, including --json payloads and error messages.

Examples:
  cosmoflare worker secret put my-worker API_TOKEN --value='hunter2'
  printf '%s' 'hunter2' | cosmoflare worker secret put my-worker API_TOKEN`,
	RunE: runWorkerSecretPut,
}

var workerSecretDeleteCmd = &cobra.Command{
	Use:   "delete WORKER KEY",
	Short: "Delete a Worker secret",
	Long: `Delete a secret binding from a Worker.

Deletion is irreversible and immediately unbinds the secret from the
Worker. Use --force to skip the confirmation prompt.

Examples:
  cosmoflare worker secret delete my-worker API_TOKEN
  cosmoflare worker secret delete my-worker API_TOKEN --force`,
	RunE: runWorkerSecretDelete,
}

var workerSecretListCmd = &cobra.Command{
	Use:   "list WORKER",
	Short: "List Worker secrets",
	Long: `List the secret names bound to a Worker.

Cloudflare never returns secret values, so this lists names (and binding
types) only, sorted alphabetically.

Examples:
  cosmoflare worker secret list my-worker
  cosmoflare worker secret list my-worker --json`,
	RunE: runWorkerSecretList,
}

var workerSecretBulkCmd = &cobra.Command{
	Use:   "bulk WORKER FILE",
	Short: "Upload many Worker secrets from a JSON file",
	Long: `Upload every secret in a JSON object file to a Worker.

FILE must contain a JSON object mapping secret names to values, e.g.
  {"API_TOKEN": "...", "DB_PASSWORD": "..."}

Keys are uploaded in sorted order and each is reported individually; one
failing key does not abort the rest. Values are never printed — the
report contains names and per-key status only.

Examples:
  cosmoflare worker secret bulk my-worker secrets.json`,
	RunE: runWorkerSecretBulk,
}

var (
	workerSecretValue string
	workerSecretForce bool
)

// registerWorkerSecretCmds wires the "worker secret" command group onto
// the given parent command (the worker command).
func registerWorkerSecretCmds(parent *cobra.Command) {
	parent.AddCommand(workerSecretCmd)

	workerSecretCmd.AddCommand(workerSecretPutCmd)
	workerSecretCmd.AddCommand(workerSecretDeleteCmd)
	workerSecretCmd.AddCommand(workerSecretListCmd)
	workerSecretCmd.AddCommand(workerSecretBulkCmd)

	workerSecretPutCmd.Flags().StringVar(&workerSecretValue, "value", "", "Secret value (omit to read from stdin)")
	workerSecretDeleteCmd.Flags().BoolVar(&workerSecretForce, "force", false, "Skip confirmation prompt")
}

// parseSecretsBulkJSON decodes a JSON object of secret name -> value.
// It is a pure helper so tests can exercise the parsing rules directly.
func parseSecretsBulkJSON(data []byte) (map[string]string, error) {
	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("invalid JSON (expected an object of secret names to values): %w", err)
	}
	for name := range secrets {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("secret names must not be empty")
		}
	}
	return secrets, nil
}

// readWorkerSecretValue reads the secret value from stdin when --value is
// not given. The value is never echoed: only the prompt goes to stderr.
func readWorkerSecretValue(cmd *cobra.Command, key string) (string, error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Enter value for secret %q (ends with newline): ", key)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("failed to read secret value from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func runWorkerSecretPut(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	if len(args) < 2 {
		return outErrf("secret name is required")
	}
	worker, key := args[0], args[1]

	value := workerSecretValue
	if value == "" {
		v, err := readWorkerSecretValue(cmd, key)
		if err != nil {
			return outErr("failed to read secret value", err)
		}
		value = v
	}
	if value == "" {
		return outErrf("secret value is required (--value or stdin)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would set worker secret", func() any {
			return map[string]string{"worker": worker, "name": key}
		}, func() {
			printInfo("DRY RUN: Would set secret '%s' on worker '%s'", key, worker)
		})
	}

	if err := svc.SecretPut(context.Background(), worker, key, value); err != nil {
		return outErr("failed to set worker secret", err)
	}

	return outPayload("Worker secret set successfully", func() any {
		return map[string]string{"worker": worker, "name": key}
	}, func() {
		printSuccess("Secret '%s' set on worker '%s'", key, worker)
	})
}

func runWorkerSecretDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	if len(args) < 2 {
		return outErrf("secret name is required")
	}
	worker, key := args[0], args[1]

	if !workerSecretForce && !DryRun {
		fmt.Printf("Are you sure you want to delete secret '%s' from worker '%s'? [y/N]: ", key, worker)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Secret deletion cancelled")
			return nil
		}
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete worker secret", func() any {
			return map[string]string{"worker": worker, "name": key}
		}, func() {
			printInfo("DRY RUN: Would delete secret '%s' from worker '%s'", key, worker)
		})
	}

	if err := svc.SecretDelete(context.Background(), worker, key); err != nil {
		return outErr("failed to delete worker secret", err)
	}

	return outPayload("Worker secret deleted successfully", func() any {
		return map[string]string{"worker": worker, "name": key}
	}, func() {
		printSuccess("Secret '%s' deleted from worker '%s'", key, worker)
	})
}

func runWorkerSecretList(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	worker := args[0]

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	secrets, err := svc.SecretList(context.Background(), worker)
	if err != nil {
		return outErr("failed to list worker secrets", err)
	}

	return outResult(secrets, func() {
		if len(secrets) == 0 {
			printInfo("No secrets found for worker '%s'", worker)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE")
		for _, s := range secrets {
			fmt.Fprintf(w, "%s\t%s\n", s.Name, s.Type)
		}
		w.Flush()

		printInfo("Total: %d secret(s)", len(secrets))
	})
}

func runWorkerSecretBulk(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	if len(args) < 2 {
		return outErrf("secrets file is required")
	}
	worker, file := args[0], args[1]

	// Read and validate the file before building the service so bad input
	// fails fast offline (no credentials needed to see these errors).
	data, err := os.ReadFile(file)
	if err != nil {
		return outErr("failed to read secrets file", err)
	}

	secrets, err := parseSecretsBulkJSON(data)
	if err != nil {
		return outErr("failed to parse secrets file", err)
	}
	if len(secrets) == 0 {
		return outErrf("secrets file contains no entries")
	}

	names := make([]string, 0, len(secrets))
	for name := range secrets {
		names = append(names, name)
	}
	sort.Strings(names)

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would bulk set worker secrets", func() any {
			return map[string]any{"worker": worker, "secrets": names}
		}, func() {
			printInfo("DRY RUN: Would set %d secret(s) on worker '%s'", len(names), worker)
			for _, name := range names {
				printInfo("  %s", name)
			}
		})
	}

	results, err := svc.SecretsBulk(context.Background(), worker, secrets)
	if err != nil {
		return outErr("failed to bulk set worker secrets", err)
	}

	return outPayload("Worker secrets bulk upload complete", func() any {
		return map[string]any{"worker": worker, "results": results}
	}, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tERROR")
		failed := 0
		for _, res := range results {
			status := "ok"
			if !res.Success {
				status = "failed"
				failed++
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", res.Key, status, res.Error)
		}
		w.Flush()
		printSuccess("Bulk upload complete: %d/%d secret(s) set", len(results)-failed, len(results))
	})
}
