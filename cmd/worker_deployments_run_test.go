package cmd

import (
	"strings"
	"testing"
)

func TestWorkerDeploymentsListValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	if err := runWorkerDeploymentsList(workerDeploymentsListCmd, nil); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected arg validation error, got %v", err)
	}
}

func TestWorkerDeploymentsListFailsOffline(t *testing.T) {
	runGlobalsSnapshot(t)
	err := runWorkerDeploymentsList(workerDeploymentsListCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestWorkerDeploymentsViewValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	if err := runWorkerDeploymentsView(workerDeploymentsViewCmd, nil); err == nil {
		t.Fatal("expected arg-count validation error, got nil")
	}
}

func TestWorkerRollbackValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	if err := runWorkerRollback(workerRollbackCmd, nil); err == nil {
		t.Fatal("expected arg-count validation error, got nil")
	}
}

func TestWorkerRollbackFailsOffline(t *testing.T) {
	runGlobalsSnapshot(t)
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
