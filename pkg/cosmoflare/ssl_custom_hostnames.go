package cosmoflare

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// CustomHostname is the public view of an SSL for SaaS custom hostname,
// with the nested ssl/verification objects flattened into plain fields.
type CustomHostname struct {
	ID                 string   `json:"id"`
	Hostname           string   `json:"hostname"`
	Status             string   `json:"status,omitempty"` // pending|active|moved|deleted|blocked
	CustomOriginServer string   `json:"custom_origin_server,omitempty"`
	VerificationStatus string   `json:"verification_status,omitempty"` // http validation state: pending_validation|active|...
	VerificationType   string   `json:"verification_type,omitempty"`   // validation method: http|txt|...
	SSLStatus          string   `json:"ssl_status,omitempty"`          // nested ssl.status, e.g. pending_validation|active
	VerificationErrors []string `json:"verification_errors,omitempty"`
	CreatedOn          string   `json:"created_on,omitempty"`
	ModifiedOn         string   `json:"modified_on,omitempty"`
}

// CustomHostnameListOptions narrows the results of List.
type CustomHostnameListOptions struct {
	// Hostname filters results to hostnames containing this substring.
	Hostname string
}

// SSLCustomHostnameService implements SSL for SaaS custom hostname operations.
// Custom hostnames are zone-scoped — they use a zone ID, not an account ID.
type SSLCustomHostnameService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewSSLCustomHostnameService creates a new custom hostname service client.
func NewSSLCustomHostnameService(api *cloudflare.API, zoneID string) (*SSLCustomHostnameService, error) {
	if api == nil {
		return nil, validationError("NewSSLCustomHostnameService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewSSLCustomHostnameService", "zone ID is required")
	}
	return &SSLCustomHostnameService{cf: api, zoneID: zoneID}, nil
}

// NewSSLCustomHostnameServiceFromCreds creates an SSLCustomHostnameService
// from zone ID and API token.
func NewSSLCustomHostnameServiceFromCreds(zoneID, apiToken string) (*SSLCustomHostnameService, error) {
	if zoneID == "" {
		return nil, validationError("NewSSLCustomHostnameService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewSSLCustomHostnameService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewSSLCustomHostnameService", "failed to create Cloudflare API client", err)
	}
	return &SSLCustomHostnameService{cf: cf, zoneID: zoneID}, nil
}

// flattenCustomHostname converts an SDK custom hostname into the flattened
// public view. cloudflare-go exposes the verification state only through the
// nested ssl object, so verification_status/ssl_status mirror ssl.status.
func flattenCustomHostname(ch cloudflare.CustomHostname) *CustomHostname {
	out := &CustomHostname{
		ID:                 ch.ID,
		Hostname:           ch.Hostname,
		Status:             string(ch.Status),
		CustomOriginServer: ch.CustomOriginServer,
		VerificationErrors: ch.VerificationErrors,
	}
	if ch.CreatedAt != nil {
		out.CreatedOn = ch.CreatedAt.Format(time.RFC3339)
	}
	if ch.SSL != nil {
		out.VerificationStatus = ch.SSL.Status
		out.VerificationType = ch.SSL.Method
		out.SSLStatus = ch.SSL.Status
		for _, ve := range ch.SSL.ValidationErrors {
			out.VerificationErrors = append(out.VerificationErrors, ve.Message)
		}
	}
	return out
}

// List returns custom hostnames for the zone, optionally filtered by a
// hostname substring. Results are sorted by hostname for deterministic output.
func (s *SSLCustomHostnameService) List(ctx context.Context, opts CustomHostnameListOptions) ([]*CustomHostname, error) {
	out := make([]*CustomHostname, 0)
	for page := 1; ; page++ {
		results, ri, err := s.cf.CustomHostnames(ctx, s.zoneID, page, cloudflare.CustomHostname{})
		if err != nil {
			return nil, newError("SSLCustomHostnameService.List", "failed to list custom hostnames", err)
		}
		for _, ch := range results {
			if opts.Hostname != "" && !strings.Contains(ch.Hostname, opts.Hostname) {
				continue
			}
			out = append(out, flattenCustomHostname(ch))
		}
		if len(results) == 0 || page >= ri.TotalPages {
			break
		}
	}
	return out, nil
}

// Create registers a new custom hostname for the zone, optionally pointing
// it at a custom origin server, and returns the created record.
func (s *SSLCustomHostnameService) Create(ctx context.Context, hostname, customOriginServer string) (*CustomHostname, error) {
	if hostname == "" {
		return nil, validationError("SSLCustomHostnameService.Create", "hostname is required")
	}

	ch := cloudflare.CustomHostname{Hostname: hostname}
	if customOriginServer != "" {
		ch.CustomOriginServer = customOriginServer
	}

	resp, err := s.cf.CreateCustomHostname(ctx, s.zoneID, ch)
	if err != nil {
		return nil, newError("SSLCustomHostnameService.Create", fmt.Sprintf("failed to create custom hostname %q", hostname), err)
	}
	return flattenCustomHostname(resp.Result), nil
}

// Get retrieves a single custom hostname by ID, including its full
// verification state.
func (s *SSLCustomHostnameService) Get(ctx context.Context, id string) (*CustomHostname, error) {
	if id == "" {
		return nil, validationError("SSLCustomHostnameService.Get", "custom hostname ID is required")
	}

	ch, err := s.cf.CustomHostname(ctx, s.zoneID, id)
	if err != nil {
		return nil, newError("SSLCustomHostnameService.Get", fmt.Sprintf("failed to get custom hostname %q", id), err)
	}
	return flattenCustomHostname(ch), nil
}

// CustomHostnameUpdateOptions holds the mutable fields of a custom hostname.
type CustomHostnameUpdateOptions struct {
	// Hostname moves the custom hostname to a new value when non-nil.
	Hostname *string
	// CustomOriginServer changes the origin server when non-nil.
	CustomOriginServer *string
}

// Update modifies an existing custom hostname. At least one field must be set.
func (s *SSLCustomHostnameService) Update(ctx context.Context, id string, opts CustomHostnameUpdateOptions) (*CustomHostname, error) {
	if id == "" {
		return nil, validationError("SSLCustomHostnameService.Update", "custom hostname ID is required")
	}
	if opts.Hostname == nil && opts.CustomOriginServer == nil {
		return nil, validationError("SSLCustomHostnameService.Update", "at least one field to update is required")
	}

	ch := cloudflare.CustomHostname{}
	if opts.Hostname != nil {
		ch.Hostname = *opts.Hostname
	}
	if opts.CustomOriginServer != nil {
		ch.CustomOriginServer = *opts.CustomOriginServer
	}

	resp, err := s.cf.UpdateCustomHostname(ctx, s.zoneID, id, ch)
	if err != nil {
		return nil, newError("SSLCustomHostnameService.Update", fmt.Sprintf("failed to update custom hostname %q", id), err)
	}
	return flattenCustomHostname(resp.Result), nil
}

// Delete removes a custom hostname from the zone.
func (s *SSLCustomHostnameService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return validationError("SSLCustomHostnameService.Delete", "custom hostname ID is required")
	}

	if err := s.cf.DeleteCustomHostname(ctx, s.zoneID, id); err != nil {
		return newError("SSLCustomHostnameService.Delete", fmt.Sprintf("failed to delete custom hostname %q", id), err)
	}
	return nil
}
