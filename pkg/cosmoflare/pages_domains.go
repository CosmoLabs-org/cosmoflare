package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// PagesCustomDomain represents a custom domain attached to a Pages project.
type PagesCustomDomain struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	VerificationStatus string `json:"verification_status"`
	ValidationStatus   string `json:"validation_status"`
}

// ListDomains returns all custom domains attached to a Pages project.
func (s *PagesService) ListDomains(ctx context.Context, project string) ([]*PagesCustomDomain, error) {
	const op = "PagesService.ListDomains"
	if project == "" {
		return nil, validationError(op, "project name is required")
	}

	results, err := s.cf.GetPagesDomains(ctx, cloudflare.PagesDomainsParameters{
		AccountID:   s.accountID,
		ProjectName: project,
	})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to list domains for project %q", project), err)
	}

	domains := make([]*PagesCustomDomain, 0, len(results))
	for _, d := range results {
		domains = append(domains, mapPagesDomain(d))
	}
	return domains, nil
}

// AttachDomain attaches a custom domain to a Pages project. The returned
// domain's Status/VerificationStatus reflect the Cloudflare validation
// state at the time of the call.
func (s *PagesService) AttachDomain(ctx context.Context, project, domain string) (*PagesCustomDomain, error) {
	const op = "PagesService.AttachDomain"
	if project == "" {
		return nil, validationError(op, "project name is required")
	}
	if domain == "" {
		return nil, validationError(op, "domain name is required")
	}

	result, err := s.cf.PagesAddDomain(ctx, cloudflare.PagesDomainParameters{
		AccountID:   s.accountID,
		ProjectName: project,
		DomainName:  domain,
	})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to attach domain %q to project %q", domain, project), err)
	}
	return mapPagesDomain(result), nil
}

// DetachDomain removes a custom domain from a Pages project.
func (s *PagesService) DetachDomain(ctx context.Context, project, domain string) error {
	const op = "PagesService.DetachDomain"
	if project == "" {
		return validationError(op, "project name is required")
	}
	if domain == "" {
		return validationError(op, "domain name is required")
	}

	err := s.cf.PagesDeleteDomain(ctx, cloudflare.PagesDomainParameters{
		AccountID:   s.accountID,
		ProjectName: project,
		DomainName:  domain,
	})
	if err != nil {
		return newError(op, fmt.Sprintf("failed to detach domain %q from project %q", domain, project), err)
	}
	return nil
}

func mapPagesDomain(d cloudflare.PagesDomain) *PagesCustomDomain {
	return &PagesCustomDomain{
		ID:                 d.ID,
		Name:               d.Name,
		Status:             d.Status,
		VerificationStatus: d.VerificationData.Status,
		ValidationStatus:   d.ValidationData.Status,
	}
}
