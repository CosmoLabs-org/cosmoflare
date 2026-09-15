package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// Worker deployment history + rollback (FEAT-021). Registered onto the
// worker command tree by registerWorkerDeploymentCmds; deployments and
// versions share the raw-REST client in pkg/cosmoflare/worker_versions.go.

var workerDeploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "Inspect Worker deployment history",
	Long: `Inspect the deployment history of a Worker script.

Each deployment records one rollout of a script version. Use
"cosmoflare worker rollback" to re-point the Worker at an earlier
deployment's version.`,
	Example: `  # List deployments (newest first)
  cosmoflare worker deployments list api-gateway

  # Inspect one deployment
  cosmoflare worker deployments view api-gateway dep-123`,
}

var workerDeploymentsListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "List Worker deployments",
	Long: `List the deployment history of a Worker, newest first.

Each entry carries the deployment ID, the version it rolled out, when it
was created, and who created it. Pipe through --json for scripting.`,
	Example: `  cosmoflare worker deployments list api-gateway --json`,
	Args:    cobra.ExactArgs(1),
	RunE:    runWorkerDeploymentsList,
}

var workerDeploymentsViewCmd = &cobra.Command{
	Use:   "view <name> <deployment-id>",
	Short: "View one Worker deployment",
	Long: `Show the full record of a single Worker deployment: the version it
rolled out, creation time, source, and author.`,
	Example: `  cosmoflare worker deployments view api-gateway dep-123`,
	Args:    cobra.ExactArgs(2),
	RunE:    runWorkerDeploymentsView,
}

var workerRollbackCmd = &cobra.Command{
	Use:   "rollback <name> [deployment-id]",
	Short: "Roll a Worker back to an earlier deployment",
	Long: `Roll a Worker back by re-deploying the version of an earlier
deployment.

With no deployment ID, the Worker rolls back to the version deployed
BEFORE the current deployment (the common "undo my last deploy" case).
With an ID, it rolls back to that deployment's version.`,
	Example: `  # Undo the last deploy
  cosmoflare worker rollback api-gateway

  # Roll back to a specific deployment's version
  cosmoflare worker rollback api-gateway dep-123`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runWorkerRollback,
}

func registerWorkerDeploymentCmds(parent *cobra.Command) {
	parent.AddCommand(workerDeploymentsCmd, workerRollbackCmd)
	workerDeploymentsCmd.AddCommand(workerDeploymentsListCmd, workerDeploymentsViewCmd)
}

func runWorkerDeploymentsList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}
	deployments, err := svc.DeploymentList(cmd.Context(), args[0])
	if err != nil {
		return outErr(fmt.Sprintf("failed to list deployments for worker %q", args[0]), err)
	}
	return outResult(deployments, func() {
		if len(deployments) == 0 {
			printInfo("No deployments found for worker %q", args[0])
			return
		}
		printInfo("Deployments for worker %q (newest first):", args[0])
		for _, d := range deployments {
			printInfo("  %s  version=%s  created=%s", d.ID, d.VersionID, d.CreatedAt.Format(time.RFC3339))
		}
	})
}

func runWorkerDeploymentsView(cmd *cobra.Command, args []string) error {
	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}
	deployment, err := svc.DeploymentGet(cmd.Context(), args[0], args[1])
	if err != nil {
		return outErr(fmt.Sprintf("failed to get deployment %s", args[1]), err)
	}
	return outResult(deployment, func() {
		printInfo("Deployment %s", deployment.ID)
		printInfo("  Version:  %s", deployment.VersionID)
		printInfo("  Created:  %s", deployment.CreatedAt.Format(time.RFC3339))
		if deployment.Source != "" {
			printInfo("  Source:   %s", deployment.Source)
		}
		if deployment.AuthorEmail != "" {
			printInfo("  Author:   %s", deployment.AuthorEmail)
		}
	})
}

func runWorkerRollback(cmd *cobra.Command, args []string) error {
	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}
	deploymentID := ""
	if len(args) == 2 {
		deploymentID = args[1]
	}
	deployment, err := svc.Rollback(cmd.Context(), args[0], deploymentID)
	if err != nil {
		return outErr(fmt.Sprintf("failed to roll back worker %q", args[0]), err)
	}
	return outPayload(fmt.Sprintf("Rolled back worker %q to version %s", args[0], deployment.VersionID), func() any {
		return deployment
	}, func() {
		printSuccess("Rolled back worker %q to version %s (deployment %s)", args[0], deployment.VersionID, deployment.ID)
	})
}
