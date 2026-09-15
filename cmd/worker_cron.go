package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Worker cron trigger management (FEAT-021). Cloudflare has no per-trigger
// CRUD: the schedule set on a script is replaced atomically, so every
// mutation here is a read-modify-replace. Registration is exported
// (registerWorkerCronCmds) so the orchestrator wires it onto the worker
// command without this file editing cmd/worker.go.

var workerCronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage Worker cron triggers",
	Long: `Manage cron trigger schedules on Cloudflare Workers.

The Cloudflare API replaces the FULL schedule set on every mutation —
there is no per-trigger create or delete. create/delete/update therefore
read the current set, modify it, and replace it atomically.

Commands:
  list    List the cron schedules attached to a Worker
  create  Append a cron expression to a Worker's schedule set
  delete  Remove a cron expression from a Worker's schedule set
  update  Swap one scheduled expression for another in a single replace

Examples:
  cosmoflare worker cron list my-worker
  cosmoflare worker cron create my-worker --expr "*/5 * * * *"
  cosmoflare worker cron delete my-worker --expr "*/5 * * * *"
  cosmoflare worker cron update my-worker --expr "*/5 * * * *" --new "*/10 * * * *"`,
}

var workerCronListCmd = &cobra.Command{
	Use:   "list WORKER",
	Short: "List Worker cron triggers",
	Long: `List the cron schedules attached to a Worker.

Each entry shows the cron expression and when it was last modified.

Examples:
  cosmoflare worker cron list my-worker
  cosmoflare worker cron list my-worker --json`,
	RunE: runWorkerCronList,
}

var workerCronCreateCmd = &cobra.Command{
	Use:   "create WORKER",
	Short: "Add a cron trigger to a Worker",
	Long: `Append a cron expression to a Worker's schedule set.

Duplicate expressions are rejected before any API call, so repeating a
create is a safe no-op. The existing schedules are preserved.

Examples:
  cosmoflare worker cron create my-worker --expr "*/5 * * * *"
  cosmoflare worker cron create my-worker --expr "0 12 * * 1" --json`,
	RunE: runWorkerCronCreate,
}

var workerCronDeleteCmd = &cobra.Command{
	Use:   "delete WORKER",
	Short: "Remove a cron trigger from a Worker",
	Long: `Remove a cron expression from a Worker's schedule set.

All other schedules are preserved. Deleting an expression that is not
scheduled is an error, not a silent no-op.

Examples:
  cosmoflare worker cron delete my-worker --expr "*/5 * * * *"
  cosmoflare worker cron delete my-worker --expr "*/5 * * * *" --json`,
	RunE: runWorkerCronDelete,
}

var workerCronUpdateCmd = &cobra.Command{
	Use:   "update WORKER",
	Short: "Swap one cron expression for another on a Worker",
	Long: `Replace one scheduled cron expression with another.

The old expression (--expr) is removed and the new one (--new) added in a
single API replace, so there is no window where the schedule is missing.
The old expression must currently be scheduled and the new one must not.

Examples:
  cosmoflare worker cron update my-worker --expr "*/5 * * * *" --new "*/10 * * * *"
  cosmoflare worker cron update my-worker --expr "*/5 * * * *" --new "0 12 * * 1" --json`,
	RunE: runWorkerCronUpdate,
}

var (
	workerCronExpr string
	workerCronNew  string
)

// registerWorkerCronCmds wires the "worker cron" command group onto the
// given parent command (the worker command).
func registerWorkerCronCmds(parent *cobra.Command) {
	parent.AddCommand(workerCronCmd)
	workerCronCmd.AddCommand(workerCronListCmd)
	workerCronCmd.AddCommand(workerCronCreateCmd)
	workerCronCmd.AddCommand(workerCronDeleteCmd)
	workerCronCmd.AddCommand(workerCronUpdateCmd)

	if workerCronCreateCmd.Flags().Lookup("expr") == nil {
		workerCronCreateCmd.Flags().StringVar(&workerCronExpr, "expr", "", "Cron expression to schedule (e.g. \"*/5 * * * *\")")
	}
	if workerCronDeleteCmd.Flags().Lookup("expr") == nil {
		workerCronDeleteCmd.Flags().StringVar(&workerCronExpr, "expr", "", "Cron expression to remove")
	}
	if workerCronUpdateCmd.Flags().Lookup("expr") == nil {
		workerCronUpdateCmd.Flags().StringVar(&workerCronExpr, "expr", "", "Currently scheduled cron expression to replace")
	}
	if workerCronUpdateCmd.Flags().Lookup("new") == nil {
		workerCronUpdateCmd.Flags().StringVar(&workerCronNew, "new", "", "New cron expression to schedule in its place")
	}
}

// workerCronPresentTable renders the trigger list as a CRON/MODIFIED
// table. It is a presenter helper shared by the human-readable output of
// every cron subcommand.
func workerCronPresentTable(triggers []cosmoflare.WorkerCronTrigger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CRON\tMODIFIED")
	for _, t := range triggers {
		fmt.Fprintf(w, "%s\t%s\n", t.Cron, t.Modified)
	}
	w.Flush()
}

func runWorkerCronList(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	worker := args[0]

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	triggers, err := svc.CronList(context.Background(), worker)
	if err != nil {
		return outErr("failed to list worker cron triggers", err)
	}

	return outResult(triggers, func() {
		if len(triggers) == 0 {
			printInfo("No cron triggers found for worker '%s'", worker)
			return
		}

		workerCronPresentTable(triggers)
		printInfo("Total: %d cron trigger(s)", len(triggers))
	})
}

func runWorkerCronCreate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	worker := args[0]
	if workerCronExpr == "" {
		return outErrf("cron expression is required (--expr)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create worker cron trigger", func() any {
			return map[string]string{"worker": worker, "cron": workerCronExpr}
		}, func() {
			printInfo("DRY RUN: Would add cron '%s' to worker '%s'", workerCronExpr, worker)
		})
	}

	triggers, err := svc.CronCreate(context.Background(), worker, workerCronExpr)
	if err != nil {
		return outErr("failed to create worker cron trigger", err)
	}

	return outPayload("Worker cron trigger created successfully", func() any {
		return triggers
	}, func() {
		printSuccess("Cron '%s' added to worker '%s'", workerCronExpr, worker)
		workerCronPresentTable(triggers)
	})
}

func runWorkerCronDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	worker := args[0]
	if workerCronExpr == "" {
		return outErrf("cron expression is required (--expr)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete worker cron trigger", func() any {
			return map[string]string{"worker": worker, "cron": workerCronExpr}
		}, func() {
			printInfo("DRY RUN: Would remove cron '%s' from worker '%s'", workerCronExpr, worker)
		})
	}

	triggers, err := svc.CronDelete(context.Background(), worker, workerCronExpr)
	if err != nil {
		return outErr("failed to delete worker cron trigger", err)
	}

	return outPayload("Worker cron trigger deleted successfully", func() any {
		return triggers
	}, func() {
		printSuccess("Cron '%s' removed from worker '%s'", workerCronExpr, worker)
		if len(triggers) > 0 {
			workerCronPresentTable(triggers)
		}
	})
}

func runWorkerCronUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("worker name is required")
	}
	worker := args[0]
	if workerCronExpr == "" {
		return outErrf("current cron expression is required (--expr)")
	}
	if workerCronNew == "" {
		return outErrf("new cron expression is required (--new)")
	}
	if workerCronNew == workerCronExpr {
		return outErrf("new cron expression must differ from the current one (--new)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update worker cron trigger", func() any {
			return map[string]string{"worker": worker, "cron": workerCronExpr, "new": workerCronNew}
		}, func() {
			printInfo("DRY RUN: Would replace cron '%s' with '%s' on worker '%s'", workerCronExpr, workerCronNew, worker)
		})
	}

	// Read-modify-replace: the swap happens in one CronReplace call so the
	// schedule set is never missing the entry mid-update.
	current, err := svc.CronList(context.Background(), worker)
	if err != nil {
		return outErr("failed to list worker cron triggers", err)
	}
	found := false
	schedules := make([]string, 0, len(current)+1)
	for _, t := range current {
		if t.Cron == workerCronNew {
			return outErrf("cron %q already exists on worker %q", workerCronNew, worker)
		}
		if t.Cron == workerCronExpr {
			found = true
			schedules = append(schedules, workerCronNew)
			continue
		}
		schedules = append(schedules, t.Cron)
	}
	if !found {
		return outErrf("cron %q not found on worker %q", workerCronExpr, worker)
	}

	triggers, err := svc.CronReplace(context.Background(), worker, schedules)
	if err != nil {
		return outErr("failed to update worker cron triggers", err)
	}

	return outPayload("Worker cron trigger updated successfully", func() any {
		return triggers
	}, func() {
		printSuccess("Cron '%s' replaced with '%s' on worker '%s'", workerCronExpr, workerCronNew, worker)
		workerCronPresentTable(triggers)
	})
}
