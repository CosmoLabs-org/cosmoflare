package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Healthcheck represents a Cloudflare health check monitor.
type Healthcheck struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Address              string    `json:"address"`
	Type                 string    `json:"type"`                          // "HTTPS", "HTTP", "TCP"
	Status               string    `json:"status"`                        // "healthy", "unhealthy", "suspended", "unknown"
	Suspended            bool      `json:"suspended"`
	Interval             int       `json:"interval"`                      // seconds between checks
	Timeout              int       `json:"timeout"`                       // seconds before timeout
	Retries              int       `json:"retries"`
	Description          string    `json:"description,omitempty"`
	ConsecutiveSuccesses int       `json:"consecutive_successes,omitempty"`
	ConsecutiveFails     int       `json:"consecutive_fails,omitempty"`
	FailureReason        string    `json:"failure_reason,omitempty"`
	CreatedOn            time.Time `json:"created_on"`
	ModifiedOn           time.Time `json:"modified_on"`
}

// HealthcheckOption is a functional option for healthcheck operations.
type HealthcheckOption func(*healthcheckConfig)

type healthcheckConfig struct {
	interval             *int
	timeout              *int
	retries              *int
	suspended            *bool
	description          *string
	hcType               *string
	consecutiveSuccesses *int
	consecutiveFails     *int
}

// WithHealthcheckInterval sets the interval (seconds) between health checks.
func WithHealthcheckInterval(interval int) HealthcheckOption {
	return func(c *healthcheckConfig) { c.interval = &interval }
}

// WithHealthcheckTimeout sets the timeout (seconds) for each health check.
func WithHealthcheckTimeout(timeout int) HealthcheckOption {
	return func(c *healthcheckConfig) { c.timeout = &timeout }
}

// WithHealthcheckRetries sets the number of retries before marking unhealthy.
func WithHealthcheckRetries(retries int) HealthcheckOption {
	return func(c *healthcheckConfig) { c.retries = &retries }
}

// WithHealthcheckSuspended sets whether the health check is suspended.
func WithHealthcheckSuspended(suspended bool) HealthcheckOption {
	return func(c *healthcheckConfig) { c.suspended = &suspended }
}

// WithHealthcheckDescription sets a description for the health check.
func WithHealthcheckDescription(description string) HealthcheckOption {
	return func(c *healthcheckConfig) { c.description = &description }
}

// WithHealthcheckType sets the health check type ("HTTPS", "HTTP", "TCP").
func WithHealthcheckType(hcType string) HealthcheckOption {
	return func(c *healthcheckConfig) { c.hcType = &hcType }
}

// HealthcheckService implements health check operations.
// Healthchecks are zone-scoped — they use a zone ID, not an account ID.
type HealthcheckService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewHealthcheckService creates a new Healthcheck service client.
func NewHealthcheckService(api *cloudflare.API, zoneID string) (*HealthcheckService, error) {
	if api == nil {
		return nil, validationError("NewHealthcheckService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewHealthcheckService", "zone ID is required")
	}
	return &HealthcheckService{cf: api, zoneID: zoneID}, nil
}

// NewHealthcheckServiceFromCreds creates a HealthcheckService from zone ID and API token.
// Convenience helper for CLI usage.
func NewHealthcheckServiceFromCreds(zoneID, apiToken string) (*HealthcheckService, error) {
	if zoneID == "" {
		return nil, validationError("NewHealthcheckService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewHealthcheckService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewHealthcheckService", "failed to create Cloudflare API client", err)
	}
	return &HealthcheckService{cf: cf, zoneID: zoneID}, nil
}

// List returns all health checks for the zone.
func (s *HealthcheckService) List(ctx context.Context) ([]*Healthcheck, error) {
	results, err := s.cf.Healthchecks(ctx, s.zoneID)
	if err != nil {
		return nil, newError("HealthcheckService.List", "failed to list health checks", err)
	}

	checks := make([]*Healthcheck, 0, len(results))
	for _, h := range results {
		checks = append(checks, cfHealthcheckToHealthcheck(h))
	}
	return checks, nil
}

// Get retrieves a single health check by ID.
func (s *HealthcheckService) Get(ctx context.Context, id string) (*Healthcheck, error) {
	if id == "" {
		return nil, validationError("HealthcheckService.Get", "healthcheck ID is required")
	}

	h, err := s.cf.Healthcheck(ctx, s.zoneID, id)
	if err != nil {
		return nil, notFound("HealthcheckService.Get", "", id, err)
	}

	return cfHealthcheckToHealthcheck(h), nil
}

// Create creates a new health check in the zone.
// Default type is "HTTPS" if not specified via WithHealthcheckType.
func (s *HealthcheckService) Create(ctx context.Context, name, address string, opts ...HealthcheckOption) (*Healthcheck, error) {
	if name == "" {
		return nil, validationError("HealthcheckService.Create", "healthcheck name is required")
	}
	if address == "" {
		return nil, validationError("HealthcheckService.Create", "healthcheck address is required")
	}

	cfg := &healthcheckConfig{}
	for _, o := range opts {
		o(cfg)
	}

	hc := cloudflare.Healthcheck{
		Name:    name,
		Address: address,
	}

	// Default type to HTTPS if not specified
	if cfg.hcType != nil {
		hc.Type = *cfg.hcType
	} else {
		hc.Type = "HTTPS"
	}

	if cfg.interval != nil {
		hc.Interval = *cfg.interval
	}
	if cfg.timeout != nil {
		hc.Timeout = *cfg.timeout
	}
	if cfg.retries != nil {
		hc.Retries = *cfg.retries
	}
	if cfg.suspended != nil {
		hc.Suspended = *cfg.suspended
	}
	if cfg.description != nil {
		hc.Description = *cfg.description
	}
	if cfg.consecutiveSuccesses != nil {
		hc.ConsecutiveSuccesses = *cfg.consecutiveSuccesses
	}
	if cfg.consecutiveFails != nil {
		hc.ConsecutiveFails = *cfg.consecutiveFails
	}

	result, err := s.cf.CreateHealthcheck(ctx, s.zoneID, hc)
	if err != nil {
		return nil, newError("HealthcheckService.Create", fmt.Sprintf("failed to create health check %q", name), err)
	}

	return cfHealthcheckToHealthcheck(result), nil
}

// Update modifies an existing health check.
// Fetches the current state first to avoid sending zero-value fields as updates.
func (s *HealthcheckService) Update(ctx context.Context, id string, opts ...HealthcheckOption) (*Healthcheck, error) {
	if id == "" {
		return nil, validationError("HealthcheckService.Update", "healthcheck ID is required")
	}
	if len(opts) == 0 {
		return nil, validationError("HealthcheckService.Update", "at least one option is required")
	}

	cfg := &healthcheckConfig{}
	for _, o := range opts {
		o(cfg)
	}

	// Fetch current state so zero-value fields are not sent as intentional updates.
	hc, err := s.cf.Healthcheck(ctx, s.zoneID, id)
	if err != nil {
		return nil, notFound("HealthcheckService.Update", "", id, err)
	}

	if cfg.hcType != nil {
		hc.Type = *cfg.hcType
	}
	if cfg.interval != nil {
		hc.Interval = *cfg.interval
	}
	if cfg.timeout != nil {
		hc.Timeout = *cfg.timeout
	}
	if cfg.retries != nil {
		hc.Retries = *cfg.retries
	}
	if cfg.suspended != nil {
		hc.Suspended = *cfg.suspended
	}
	if cfg.description != nil {
		hc.Description = *cfg.description
	}
	if cfg.consecutiveSuccesses != nil {
		hc.ConsecutiveSuccesses = *cfg.consecutiveSuccesses
	}
	if cfg.consecutiveFails != nil {
		hc.ConsecutiveFails = *cfg.consecutiveFails
	}

	result, err := s.cf.UpdateHealthcheck(ctx, s.zoneID, id, hc)
	if err != nil {
		return nil, newError("HealthcheckService.Update", fmt.Sprintf("failed to update health check %q", id), err)
	}

	return cfHealthcheckToHealthcheck(result), nil
}

// Delete removes a health check by ID.
func (s *HealthcheckService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return validationError("HealthcheckService.Delete", "healthcheck ID is required")
	}

	err := s.cf.DeleteHealthcheck(ctx, s.zoneID, id)
	if err != nil {
		return newError("HealthcheckService.Delete", fmt.Sprintf("failed to delete health check %q", id), err)
	}
	return nil
}

// cfHealthcheckToHealthcheck converts a cloudflare.Healthcheck to our Healthcheck type.
func cfHealthcheckToHealthcheck(h cloudflare.Healthcheck) *Healthcheck {
	hc := &Healthcheck{
		ID:                   h.ID,
		Name:                 h.Name,
		Address:              h.Address,
		Type:                 h.Type,
		Status:               h.Status,
		Suspended:            h.Suspended,
		Interval:             h.Interval,
		Timeout:              h.Timeout,
		Retries:              h.Retries,
		Description:          h.Description,
		ConsecutiveSuccesses: h.ConsecutiveSuccesses,
		ConsecutiveFails:     h.ConsecutiveFails,
		FailureReason:        h.FailureReason,
	}

	if h.CreatedOn != nil {
		hc.CreatedOn = *h.CreatedOn
	}
	if h.ModifiedOn != nil {
		hc.ModifiedOn = *h.ModifiedOn
	}

	return hc
}
