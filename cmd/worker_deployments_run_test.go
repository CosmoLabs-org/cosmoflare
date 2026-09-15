package cmd

import (
	"strings"
	"testing"
)

// workerDeploymentsGlobals snapshots the credential globals the deployment
// commands read, so tests cannot leak state and cannot dial the API.
func workerDeploymentsGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldToken := AccountID, APIToken
	AccountID, APIToken = "", ""
	t.Cleanup(func() { AccountID, APIToken = oldAcct, oldToken })
}

func TestWorkerDeploymentsListValidation(t *testing.T) {
	workerDeploymentsGlobals(t)
	if err := runWorkerDeploymentsList(workerDeploymentsListCmd, nil); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected arg validation error, got %v", err)
	}
}

func TestWorkerDeploymentsListFailsOffline(t *testing.T) {
	workerDeploymentsGlobals(t)
	err := runWorkerDeploymentsList(workerDeploymentsListCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestWorkerDeploymentsViewValidation(t *testing.T) {
	workerDeploymentsGlobals(t)
	if err := runWorkerDeploymentsView(workerDeploymentsViewCmd, nil); err == nil {
		t.Fatal("expected arg-count validation error, got nil")
	}
}

func TestWorkerRollbackValidation(t *testing.T) {
	workerDeploymentsGlobals(t)
	if err := runWorkerRollback(workerRollbackCmd, nil); err == nil {
		t.Fatal("expected arg-count validation error, got nil")
	}
}

func TestWorkerRollbackFailsOffline(t *testing.T) {
	workerDeploymentsGlobals(t)
	err := runWorkerRollback(workerRollbackCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestRegisterWorkerDeploymentCmds(t *testing.T) {
	parent := workerCmd
	registerWorkerDeploymentCmds(parent)
	for _, name := range []string{"deployments", "rollback"} {
		found := false
		for _, c := range parent.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("worker tree missing %q subcommand", name)
		}
	}
}
