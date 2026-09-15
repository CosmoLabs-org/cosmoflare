package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
)

// workerDeploymentsPath builds the script-deployments path for a Worker.
func (s *WorkerService) workerDeploymentsPath(worker string) string {
	return fmt.Sprintf("/accounts/%s/workers/scripts/%s/deployments", s.accountID, worker)
}

// DeploymentList returns the deployment history of a Worker script, newest
// first as the API returns it.
func (s *WorkerService) DeploymentList(ctx context.Context, worker string) ([]WorkerDeployment, error) {
	if worker == "" {
		return nil, validationError("WorkerService.DeploymentList", "worker name is required")
	}
	var deployments []WorkerDeployment
	if err := s.rawRequest(ctx, http.MethodGet, s.workerDeploymentsPath(worker), nil, &deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

// DeploymentGet returns a single deployment of a Worker script.
func (s *WorkerService) DeploymentGet(ctx context.Context, worker, deploymentID string) (*WorkerDeployment, error) {
	if worker == "" {
		return nil, validationError("WorkerService.DeploymentGet", "worker name is required")
	}
	if deploymentID == "" {
		return nil, validationError("WorkerService.DeploymentGet", "deployment ID is required")
	}
	var deployment WorkerDeployment
	path := fmt.Sprintf("%s/%s", s.workerDeploymentsPath(worker), deploymentID)
	if err := s.rawRequest(ctx, http.MethodGet, path, nil, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Rollback re-points the Worker at the version deployed by the latest
// deployment created BEFORE the given deployment. The Cloudflare API has no
// dedicated rollback endpoint: rollback is a deployments PUT naming the
// earlier version. With an empty deploymentID the CURRENT deployment is
// used, rolling the Worker back to its predecessor.
func (s *WorkerService) Rollback(ctx context.Context, worker, deploymentID string) (*WorkerDeployment, error) {
	if worker == "" {
		return nil, validationError("WorkerService.Rollback", "worker name is required")
	}

	deployments, err := s.DeploymentList(ctx, worker)
	if err != nil {
		return nil, err
	}
	if len(deployments) == 0 {
		return nil, validationError("WorkerService.Rollback", "no deployments found for worker")
	}

	// Deployments arrive newest-first; index i+1 is the predecessor of i.
	idx := 0
	if deploymentID != "" {
		for i, d := range deployments {
			if d.ID == deploymentID {
				idx = i
				break
			}
		}
	}
	if idx+1 >= len(deployments) {
		return nil, validationError("WorkerService.Rollback", "no earlier deployment to roll back to")
	}
	target := deployments[idx+1]

	return s.VersionDeploy(ctx, worker, target.VersionID)
}
