package cosmoflare

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloudflare/cloudflare-go"
)

// WorkerDomain is the public view of a Worker custom domain attachment:
// which hostname routes to which Worker script, inside which zone.
type WorkerDomain struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
	ZoneID   string `json:"zone_id"`
}

// DomainList returns every custom domain attached to a Worker in the
// account, sorted by hostname for deterministic output.
func (s *WorkerService) DomainList(ctx context.Context) ([]WorkerDomain, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	domains, err := s.cf.ListWorkersDomains(ctx, rc, cloudflare.ListWorkersDomainParams{})
	if err != nil {
		return nil, newError("WorkerService.DomainList", "failed to list worker domains", err)
	}

	out := make([]WorkerDomain, 0, len(domains))
	for _, d := range domains {
		out = append(out, WorkerDomain{
			ID:       d.ID,
			Hostname: d.Hostname,
			Service:  d.Service,
			ZoneID:   d.ZoneID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out, nil
}

// DomainAttach attaches hostname to the Worker script named service within
// the given zone and returns the resulting domain record.
func (s *WorkerService) DomainAttach(ctx context.Context, hostname, service, zoneID string) (*WorkerDomain, error) {
	if hostname == "" {
		return nil, validationError("WorkerService.DomainAttach", "hostname is required")
	}
	if service == "" {
		return nil, validationError("WorkerService.DomainAttach", "service name is required")
	}
	if zoneID == "" {
		return nil, validationError("WorkerService.DomainAttach", "zone ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	// cloudflare-go v0.116 rejects an empty Environment client-side, but the
	// domains endpoint only has one meaningful value, so pin it here.
	res, err := s.cf.AttachWorkersDomain(ctx, rc, cloudflare.AttachWorkersDomainParams{
		ZoneID:      zoneID,
		Hostname:    hostname,
		Service:     service,
		Environment: "production",
	})
	if err != nil {
		return nil, newError("WorkerService.DomainAttach", fmt.Sprintf("failed to attach domain %q to worker %q", hostname, service), err)
	}
	return &WorkerDomain{
		ID:       res.ID,
		Hostname: res.Hostname,
		Service:  res.Service,
		ZoneID:   res.ZoneID,
	}, nil
}

// DomainDetach removes a custom domain attachment by its domain ID. The
// hostname stops routing to the Worker immediately; the zone and DNS record
// are left untouched.
func (s *WorkerService) DomainDetach(ctx context.Context, domainID string) error {
	if domainID == "" {
		return validationError("WorkerService.DomainDetach", "domain ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DetachWorkersDomain(ctx, rc, domainID); err != nil {
		return newError("WorkerService.DomainDetach", fmt.Sprintf("failed to detach domain %q", domainID), err)
	}
	return nil
}
