package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// workerDomainRunGlobals snapshots and restores the package-level flag
// variables the worker domain runners read, so tests cannot leak state.
func workerDomainRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldService, oldZone, oldForce := workerDomainService, workerDomainZone, workerDomainForce
	t.Cleanup(func() { workerDomainService, workerDomainZone, workerDomainForce = oldService, oldZone, oldForce })
}

// workerDomainRunResetFlags returns the domain flag variables to their zero
// values so each subtest starts from a pristine command state.
func workerDomainRunResetFlags() {
	workerDomainService = ""
	workerDomainZone = ""
	workerDomainForce = false
}

// TestRunWorkerDomainAttach_RequiresArgs verifies the argument and flag
// guards fire before any service construction or network access.
func TestRunWorkerDomainAttach_RequiresArgs(t *testing.T) {
	workerDomainRunGlobals(t)
	workerDomainRunResetFlags()

	err := runWorkerDomainAttach(workerDomainAttachCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "hostname is required") {
		t.Fatalf("expected hostname error, got %v", err)
	}

	err = runWorkerDomainAttach(workerDomainAttachCmd, []string{"app.example.com"})
	if err == nil || !strings.Contains(err.Error(), "service name is required") {
		t.Fatalf("expected service error, got %v", err)
	}

	workerDomainService = "my-worker"
	err = runWorkerDomainAttach(workerDomainAttachCmd, []string{"app.example.com"})
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone error, got %v", err)
	}
}

// TestRunWorkerDomainDetach_RequiresArgs verifies the argument guard.
func TestRunWorkerDomainDetach_RequiresArgs(t *testing.T) {
	workerDomainRunGlobals(t)
	workerDomainRunResetFlags()

	err := runWorkerDomainDetach(workerDomainDetachCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "domain ID is required") {
		t.Fatalf("expected domain ID error, got %v", err)
	}
}

// TestRunWorkerDomainDetach_BlankID verifies a whitespace-only domain ID is
// rejected rather than sent to the API.
func TestRunWorkerDomainDetach_BlankID(t *testing.T) {
	workerDomainRunGlobals(t)
	workerDomainRunResetFlags()

	err := runWorkerDomainDetach(workerDomainDetachCmd, []string{"  "})
	if err == nil || !strings.Contains(err.Error(), "domain ID is required") {
		t.Fatalf("expected domain ID error, got %v", err)
	}
}

// TestRunWorkerDomainList_OfflineNoCreds verifies that with empty
// credentials the runner fails at service construction, before any request.
func TestRunWorkerDomainList_OfflineNoCreds(t *testing.T) {
	workerDomainRunGlobals(t)
	APIToken = ""

	err := runWorkerDomainList(workerDomainListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected service construction error, got %v", err)
	}
}

// TestRunWorkerDomainAttach_OfflineNoCreds verifies flag validation happens
// before credentials are consulted: bad flags error even with no token, and
// valid flags stop at service construction with no token.
func TestRunWorkerDomainAttach_OfflineNoCreds(t *testing.T) {
	workerDomainRunGlobals(t)
	workerDomainRunResetFlags()
	APIToken = ""

	// Flags blank: validation fires first, independent of credentials.
	err := runWorkerDomainAttach(workerDomainAttachCmd, []string{"app.example.com"})
	if err == nil || !strings.Contains(err.Error(), "service name is required") {
		t.Fatalf("expected service error, got %v", err)
	}

	// Flags present: proceeds to service construction, which fails offline.
	workerDomainService = "my-worker"
	workerDomainZone = "zone-1"
	err = runWorkerDomainAttach(workerDomainAttachCmd, []string{"app.example.com"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected service construction error, got %v", err)
	}
}

// TestRegisterWorkerDomainCmds verifies registration wires the group and all
// three subcommands onto a parent without touching cmd/worker.go.
func TestRegisterWorkerDomainCmds(t *testing.T) {
	parent := &cobra.Command{Use: "worker-stub", Run: func(*cobra.Command, []string) {}}
	registerWorkerDomainCmds(parent)

	subs := parent.Commands()
	if len(subs) != 1 || subs[0].Name() != "domain" {
		t.Fatalf("expected exactly one 'domain' group on parent, got %d command(s)", len(subs))
	}

	names := map[string]bool{}
	for _, sub := range subs[0].Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "attach", "detach"} {
		if !names[want] {
			t.Errorf("expected 'domain %s' registered, got %v", want, names)
		}
	}

	// Flags must be registered exactly once even if registration re-runs.
	registerWorkerDomainCmds(parent)
	if workerDomainAttachCmd.Flags().Lookup("service") == nil {
		t.Error("expected --service flag on attach")
	}
	if workerDomainAttachCmd.Flags().Lookup("zone") == nil {
		t.Error("expected --zone flag on attach")
	}
	if workerDomainDetachCmd.Flags().Lookup("force") == nil {
		t.Error("expected --force flag on detach")
	}
}
