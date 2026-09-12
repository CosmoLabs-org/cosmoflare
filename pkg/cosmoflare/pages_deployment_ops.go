package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// PagesDeploymentLogEntry represents a single line of a Pages deployment
// build log.
type PagesDeploymentLogEntry struct {
	Timestamp *time.Time `json:"timestamp"`
	Line      string     `json:"line"`
}

// PagesDeploymentLogs represents the build log for a Pages deployment.
type PagesDeploymentLogs struct {
	Total int                       `json:"total"`
	Data  []PagesDeploymentLogEntry `json:"data"`
}

// RetryDeployment re-runs a Pages deployment, typically used to retry a
// failed build. This triggers a new build using the same source; it does
// not modify existing deployments.
func (s *PagesService) RetryDeployment(ctx context.Context, project, deploymentID string) (*PagesDeployment, error) {
	const op = "PagesService.RetryDeployment"
	if project == "" {
		return nil, validationError(op, "project name is required")
	}
	if deploymentID == "" {
		return nil, validationError(op, "deployment ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.RetryPagesDeployment(ctx, rc, project, deploymentID)
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to retry deployment %q for project %q", deploymentID, project), err)
	}
	return mapPagesDeployment(result), nil
}

// GetDeploymentLogs returns the build log for a Pages deployment.
func (s *PagesService) GetDeploymentLogs(ctx context.Context, project, deploymentID string) (*PagesDeploymentLogs, error) {
	const op = "PagesService.GetDeploymentLogs"
	if project == "" {
		return nil, validationError(op, "project name is required")
	}
	if deploymentID == "" {
		return nil, validationError(op, "deployment ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetPagesDeploymentLogs(ctx, rc, cloudflare.GetPagesDeploymentLogsParams{
		ProjectName:  project,
		DeploymentID: deploymentID,
	})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to get logs for deployment %q of project %q", deploymentID, project), err)
	}

	entries := make([]PagesDeploymentLogEntry, 0, len(result.Data))
	for _, e := range result.Data {
		entries = append(entries, PagesDeploymentLogEntry{
			Timestamp: e.Timestamp,
			Line:      e.Line,
		})
	}
	return &PagesDeploymentLogs{
		Total: result.Total,
		Data:  entries,
	}, nil
}
