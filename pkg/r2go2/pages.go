package r2go2

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// PagesProject represents a Cloudflare Pages project.
type PagesProject struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	SubDomain        string     `json:"subdomain"`
	Domains          []string   `json:"domains"`
	ProductionBranch string     `json:"production_branch"`
	CreatedOn        *time.Time `json:"created_on"`
}

// PagesDeployment represents a deployment within a Cloudflare Pages project.
type PagesDeployment struct {
	ID          string     `json:"id"`
	ShortID     string     `json:"short_id"`
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Environment string     `json:"environment"`
	URL         string     `json:"url"`
	CreatedOn   *time.Time `json:"created_on"`
	ModifiedOn  *time.Time `json:"modified_on"`
}

// PagesService implements Cloudflare Pages operations.
type PagesService struct {
	cf        *cloudflare.API
	accountID string
}

// NewPagesService creates a new Pages service client.
func NewPagesService(api *cloudflare.API, accountID string) (*PagesService, error) {
	if api == nil {
		return nil, validationError("NewPagesService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewPagesService", "account ID is required")
	}
	return &PagesService{cf: api, accountID: accountID}, nil
}

// NewPagesServiceFromCreds creates a PagesService from account ID and API token.
func NewPagesServiceFromCreds(accountID, apiToken string) (*PagesService, error) {
	if accountID == "" {
		return nil, validationError("NewPagesService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewPagesService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewPagesService", "failed to create Cloudflare API client", err)
	}
	return &PagesService{cf: cf, accountID: accountID}, nil
}

// List returns all Pages projects in the account.
func (s *PagesService) List(ctx context.Context) ([]*PagesProject, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListPagesProjects(ctx, rc, cloudflare.ListPagesProjectsParams{})
	if err != nil {
		return nil, newError("PagesService.List", "failed to list projects", err)
	}

	projects := make([]*PagesProject, 0, len(results))
	for _, p := range results {
		projects = append(projects, mapPagesProject(p))
	}
	return projects, nil
}

// Get retrieves a single Pages project by name.
func (s *PagesService) Get(ctx context.Context, projectName string) (*PagesProject, error) {
	if projectName == "" {
		return nil, validationError("PagesService.Get", "project name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetPagesProject(ctx, rc, projectName)
	if err != nil {
		return nil, newError("PagesService.Get", fmt.Sprintf("failed to get project %q", projectName), err)
	}

	return mapPagesProject(result), nil
}

// Create creates a new Pages project.
func (s *PagesService) Create(ctx context.Context, name, productionBranch string) (*PagesProject, error) {
	if name == "" {
		return nil, validationError("PagesService.Create", "project name is required")
	}
	if productionBranch == "" {
		return nil, validationError("PagesService.Create", "production branch is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.CreatePagesProject(ctx, rc, cloudflare.CreatePagesProjectParams{
		Name:             name,
		ProductionBranch: productionBranch,
	})
	if err != nil {
		return nil, newError("PagesService.Create", fmt.Sprintf("failed to create project %q", name), err)
	}

	return mapPagesProject(result), nil
}

// Delete deletes a Pages project by name.
func (s *PagesService) Delete(ctx context.Context, projectName string) error {
	if projectName == "" {
		return validationError("PagesService.Delete", "project name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeletePagesProject(ctx, rc, projectName)
	if err != nil {
		return newError("PagesService.Delete", fmt.Sprintf("failed to delete project %q", projectName), err)
	}
	return nil
}

// ListDeployments returns all deployments for a Pages project.
func (s *PagesService) ListDeployments(ctx context.Context, projectName string) ([]*PagesDeployment, error) {
	if projectName == "" {
		return nil, validationError("PagesService.ListDeployments", "project name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListPagesDeployments(ctx, rc, cloudflare.ListPagesDeploymentsParams{
		ProjectName: projectName,
	})
	if err != nil {
		return nil, newError("PagesService.ListDeployments", fmt.Sprintf("failed to list deployments for project %q", projectName), err)
	}

	deployments := make([]*PagesDeployment, 0, len(results))
	for _, d := range results {
		deployments = append(deployments, mapPagesDeployment(d))
	}
	return deployments, nil
}

// GetDeployment retrieves a single deployment by project name and deployment ID.
func (s *PagesService) GetDeployment(ctx context.Context, projectName, deploymentID string) (*PagesDeployment, error) {
	if projectName == "" {
		return nil, validationError("PagesService.GetDeployment", "project name is required")
	}
	if deploymentID == "" {
		return nil, validationError("PagesService.GetDeployment", "deployment ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetPagesDeploymentInfo(ctx, rc, projectName, deploymentID)
	if err != nil {
		return nil, newError("PagesService.GetDeployment", fmt.Sprintf("failed to get deployment %q for project %q", deploymentID, projectName), err)
	}

	return mapPagesDeployment(result), nil
}

// DeleteDeployment deletes a deployment from a Pages project.
func (s *PagesService) DeleteDeployment(ctx context.Context, projectName, deploymentID string) error {
	if projectName == "" {
		return validationError("PagesService.DeleteDeployment", "project name is required")
	}
	if deploymentID == "" {
		return validationError("PagesService.DeleteDeployment", "deployment ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeletePagesDeployment(ctx, rc, cloudflare.DeletePagesDeploymentParams{
		ProjectName:  projectName,
		DeploymentID: deploymentID,
	})
	if err != nil {
		return newError("PagesService.DeleteDeployment", fmt.Sprintf("failed to delete deployment %q for project %q", deploymentID, projectName), err)
	}
	return nil
}

// mapPagesProject converts a cloudflare.PagesProject to our PagesProject type.
func mapPagesProject(p cloudflare.PagesProject) *PagesProject {
	domains := p.Domains
	if domains == nil {
		domains = []string{}
	}
	return &PagesProject{
		ID:               p.ID,
		Name:             p.Name,
		SubDomain:        p.SubDomain,
		Domains:          domains,
		ProductionBranch: p.ProductionBranch,
		CreatedOn:        p.CreatedOn,
	}
}

// mapPagesDeployment converts a cloudflare.PagesProjectDeployment to our PagesDeployment type.
func mapPagesDeployment(d cloudflare.PagesProjectDeployment) *PagesDeployment {
	return &PagesDeployment{
		ID:          d.ID,
		ShortID:     d.ShortID,
		ProjectID:   d.ProjectID,
		ProjectName: d.ProjectName,
		Environment: d.Environment,
		URL:         d.URL,
		CreatedOn:   d.CreatedOn,
		ModifiedOn:  d.ModifiedOn,
	}
}
