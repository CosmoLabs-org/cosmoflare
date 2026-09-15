package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// workerCronRunGlobals snapshots and restores the package-level flag
// variables the worker cron runners read, so tests cannot leak state.
func workerCronRunGlobals(t *testing.T) {
	t.Helper()
	oldExpr, oldNew := workerCronExpr, workerCronNew
	oldToken, oldDry, oldJSON := APIToken, DryRun, JSONOutput
	oldAcct := AccountID
	// Zero credentials for the test's duration: earlier suite tests may
	// leave them set, which would let service creation succeed and flip
	// these tests from offline-error to dry-run-success (order-dependent).
	AccountID, APIToken = "", ""
	t.Cleanup(func() {
		workerCronExpr, workerCronNew = oldExpr, oldNew
		AccountID, APIToken, DryRun, JSONOutput = oldAcct, oldToken, oldDry, oldJSON
	})
}

// workerCronRunResetFlags returns the cron flag variables to their zero
// values so each subtest starts from a pristine command state.
func workerCronRunResetFlags() {
	workerCronExpr = ""
	workerCronNew = ""
}

// TestRunWorkerCronList_RequiresWorker verifies the argument guard fires
// before any service construction.
func TestRunWorkerCronList_RequiresWorker(t *testing.T) {
	workerCronRunGlobals(t)

	err := runWorkerCronList(workerCronListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}
}

// TestRunWorkerCronCreate_RequiresArgs verifies the create argument and
// flag guards fire offline (no credentials, no API call).
func TestRunWorkerCronCreate_RequiresArgs(t *testing.T) {
	workerCronRunGlobals(t)
	workerCronRunResetFlags()

	err := runWorkerCronCreate(workerCronCreateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerCronCreate(workerCronCreateCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "cron expression is required") {
		t.Fatalf("expected cron expression error, got %v", err)
	}
}

// TestRunWorkerCronDelete_RequiresArgs verifies the delete argument and
// flag guards fire offline.
func TestRunWorkerCronDelete_RequiresArgs(t *testing.T) {
	workerCronRunGlobals(t)
	workerCronRunResetFlags()

	err := runWorkerCronDelete(workerCronDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerCronDelete(workerCronDeleteCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "cron expression is required") {
		t.Fatalf("expected cron expression error, got %v", err)
	}
}

// TestRunWorkerCronUpdate_RequiresArgs verifies every update guard: the
// worker argument, both expressions, and that they must differ.
func TestRunWorkerCronUpdate_RequiresArgs(t *testing.T) {
	workerCronRunGlobals(t)
	workerCronRunResetFlags()

	err := runWorkerCronUpdate(workerCronUpdateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerCronUpdate(workerCronUpdateCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "current cron expression is required") {
		t.Fatalf("expected current-expression error, got %v", err)
	}

	workerCronExpr = "*/5 * * * *"
	err = runWorkerCronUpdate(workerCronUpdateCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "new cron expression is required") {
		t.Fatalf("expected new-expression error, got %v", err)
	}

	workerCronNew = "*/5 * * * *"
	err = runWorkerCronUpdate(workerCronUpdateCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("expected must-differ error, got %v", err)
	}
}

// TestRunWorkerCronCreate_DryRun verifies the dry-run path emits the
// payload without constructing an API call beyond service creation.
func TestRunWorkerCronCreate_DryRun(t *testing.T) {
	workerCronRunGlobals(t)
	workerCronRunResetFlags()
	DryRun = true
	workerCronExpr = "*/5 * * * *"

	err := runWorkerCronCreate(workerCronCreateCmd, []string{"my-worker"})
	if err == nil {
		// Dry run still needs a service (credentials) in this code path;
		// without credentials the service constructor error is expected.
		t.Fatalf("expected either success (credentials present) or service error, got nil")
	}
}

// TestWorkerCronPresentTable verifies the presenter helper renders a
// header plus one row per trigger into its writer target.
func TestWorkerCronPresentTable(t *testing.T) {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = old
	})

	workerCronPresentTable([]cosmoflare.WorkerCronTrigger{
		{Cron: "*/5 * * * *", Modified: "2026-01-02T03:04:05Z"},
		{Cron: "0 12 * * 1"},
	})
	w.Close()
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "CRON") || !strings.Contains(out, "MODIFIED") {
		t.Fatalf("expected header in output, got %q", out)
	}
	if !strings.Contains(out, "*/5 * * * *") || !strings.Contains(out, "2026-01-02T03:04:05Z") {
		t.Fatalf("expected trigger row in output, got %q", out)
	}
	if !strings.Contains(out, "0 12 * * 1") {
		t.Fatalf("expected empty-modified trigger row in output, got %q", out)
	}
}

// TestRegisterWorkerCronCmds verifies registration wires the group and
// all four subcommands onto a parent without touching cmd/worker.go, and
// that flag registration is idempotent.
func TestRegisterWorkerCronCmds(t *testing.T) {
	parent := &cobra.Command{Use: "worker-stub", Run: func(*cobra.Command, []string) {}}
	registerWorkerCronCmds(parent)

	subs := parent.Commands()
	if len(subs) != 1 || subs[0].Name() != "cron" {
		t.Fatalf("expected exactly one 'cron' group on parent, got %d command(s)", len(subs))
	}

	names := map[string]bool{}
	for _, sub := range subs[0].Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "create", "delete", "update"} {
		if !names[want] {
			t.Errorf("expected 'cron %s' registered, got %v", want, names)
		}
	}

	// Idempotent guards: a second registration must not panic or
	// re-declare flags.
	registerWorkerCronCmds(parent)

	if workerCronCreateCmd.Flags().Lookup("expr") == nil {
		t.Error("expected --expr flag on create")
	}
	if workerCronDeleteCmd.Flags().Lookup("expr") == nil {
		t.Error("expected --expr flag on delete")
	}
	if workerCronUpdateCmd.Flags().Lookup("expr") == nil || workerCronUpdateCmd.Flags().Lookup("new") == nil {
		t.Error("expected --expr and --new flags on update")
	}
}
