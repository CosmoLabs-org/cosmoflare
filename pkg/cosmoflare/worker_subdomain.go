package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// WorkersSubdomainInfo is the public view of the account's workers.dev
// subdomain: the name and whether it is enabled for serving traffic.
//
// Note: cloudflare-go v0.116's WorkersSubdomain struct only decodes the
// "name" field of the API response, so Enabled is not currently
// populated from the wire and stays false. The field is kept in the
// public shape so callers do not have to change when the SDK catches up.
type WorkersSubdomainInfo struct {
	Subdomain string `json:"subdomain"`
	Enabled   bool   `json:"enabled"`
}

// SubdomainGet returns the account's workers.dev subdomain.
func (s *WorkerService) SubdomainGet(ctx context.Context) (*WorkersSubdomainInfo, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.WorkersGetSubdomain(ctx, rc)
	if err != nil {
		return nil, newError("WorkerService.SubdomainGet", "failed to get workers.dev subdomain", err)
	}
	return subdomainInfoFromAPI(resp), nil
}

// SubdomainSet creates (or renames) the account's workers.dev subdomain.
// The Cloudflare API treats "create" and "set" as the same PUT operation.
// The name must be non-empty and contain only lowercase letters, digits,
// and hyphens.
func (s *WorkerService) SubdomainSet(ctx context.Context, subdomain string) (*WorkersSubdomainInfo, error) {
	if err := validateSubdomainFormat(subdomain); err != nil {
		return nil, err
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.WorkersCreateSubdomain(ctx, rc, cloudflare.WorkersSubdomain{Name: subdomain})
	if err != nil {
		return nil, newError("WorkerService.SubdomainSet", fmt.Sprintf("failed to set workers.dev subdomain to %q", subdomain), err)
	}
	return subdomainInfoFromAPI(resp), nil
}

// validateSubdomainFormat enforces the workers.dev naming rules.
func validateSubdomainFormat(subdomain string) error {
	if subdomain == "" {
		return validationError("WorkerService.SubdomainSet", "subdomain is required")
	}
	for _, r := range subdomain {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return validationError("WorkerService.SubdomainSet", "subdomain may only contain lowercase letters, digits, and hyphens")
	}
	return nil
}

// subdomainInfoFromAPI maps the SDK struct into the public shape.
func subdomainInfoFromAPI(resp cloudflare.WorkersSubdomain) *WorkersSubdomainInfo {
	return &WorkersSubdomainInfo{
		Subdomain: resp.Name,
		Enabled:   false, // not decoded by cloudflare-go v0.116; see type doc
	}
}
