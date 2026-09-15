package cmd

import (
	"strings"
	"testing"
)

// workerBindingsGlobals snapshots the credential globals the bindings
// command reads, so tests cannot leak state and cannot dial the API
// (same zeroed-snapshot pattern as worker_deployments_run_test.go).
func workerBindingsGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldToken := AccountID, APIToken
	AccountID, APIToken = "", ""
	t.Cleanup(func() { AccountID, APIToken = oldAcct, oldToken })
}

func TestWorkerBindingsValidation(t *testing.T) {
	workerBindingsGlobals(t)
	if err := runWorkerBindings(workerBindingsCmd, nil); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected arg validation error, got %v", err)
	}
}

func TestWorkerBindingsFailsOffline(t *testing.T) {
	workerBindingsGlobals(t)
	err := runWorkerBindings(workerBindingsCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestRegisterWorkerBindingsCmds(t *testing.T) {
	parent := workerCmd
	registerWorkerBindingsCmds(parent)
	found := false
	for _, c := range parent.Commands() {
		if c.Name() == "bindings" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("worker tree missing \"bindings\" subcommand")
	}
}
