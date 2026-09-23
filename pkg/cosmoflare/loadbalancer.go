package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// LoadBalancerOrigin is the public view of one origin inside a pool. Weight
// and per-origin headers exist in the API but are deliberately not exposed
// by the CLI yet — keep the surface minimal and honest.
type LoadBalancerOrigin struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// LoadBalancerPool is the public view of an account-scoped load balancer
// pool. Timestamps are rendered as RFC 3339 strings, empty when the API
// reports none. Healthy is a pointer: nil means the API has no verdict yet.
type LoadBalancerPool struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Enabled      bool                 `json:"enabled"`
	Monitor      string               `json:"monitor,omitempty"`
	Origins      []LoadBalancerOrigin `json:"origins"`
	CheckRegions []string             `json:"check_regions,omitempty"`
	CreatedOn    string               `json:"created_on,omitempty"`
	ModifiedOn   string               `json:"modified_on,omitempty"`
	Healthy      *bool                `json:"healthy,omitempty"`
}

// LoadBalancerPoolHealth is the public view of a pool's per-PoP health.
type LoadBalancerPoolHealth struct {
	PoolID string                                   `json:"pool_id"`
	Pop    map[string]LoadBalancerPoolPopHealthView `json:"pop_health,omitempty"`
}

// LoadBalancerPoolPopHealthView is the health of one pool as seen from one
// Cloudflare PoP.
type LoadBalancerPoolPopHealthView struct {
	Healthy bool                                      `json:"healthy"`
	Origins []map[string]LoadBalancerOriginHealthView `json:"origins,omitempty"`
}

// LoadBalancerOriginHealthView is the health verdict for one origin from one
// PoP's probe.
type LoadBalancerOriginHealthView struct {
	Healthy       bool   `json:"healthy"`
	FailureReason string `json:"failure_reason,omitempty"`
	ResponseCode  int    `json:"response_code,omitempty"`
}

// LoadBalancerMonitor is the public view of an account-scoped load balancer
// monitor (the health-check probe attached to pools).
type LoadBalancerMonitor struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Description   string `json:"description,omitempty"`
	Path          string `json:"path"`
	Port          uint16 `json:"port,omitempty"`
	ExpectedCodes string `json:"expected_codes,omitempty"`
	Interval      int    `json:"interval"`
	Retries       int    `json:"retries"`
	Timeout       int    `json:"timeout"`
	CreatedOn     string `json:"created_on,omitempty"`
	ModifiedOn    string `json:"modified_on,omitempty"`
}

// LoadBalancerPoolCreate carries the required fields for pool creation.
// Origins come from the CLI's --origins JSON; omitted "enabled" keys are
// treated as enabled (a pool of disabled origins is almost always a
// misconfiguration, mirroring the Logpush create default).
type LoadBalancerPoolCreate struct {
	Name        string
	Description string
	Monitor     string
	Origins     []LoadBalancerOrigin
}

// LoadBalancerPoolUpdate carries optional changes to an existing pool.
// Zero values leave the corresponding field untouched except Enabled, which
// is a pointer so "disable" is expressible; a nil Origins slice keeps the
// current origins.
type LoadBalancerPoolUpdate struct {
	Name        string
	Description string
	Monitor     string
	Enabled     *bool
	Origins     []LoadBalancerOrigin
}

// LoadBalancerMonitorCreate carries the fields for monitor creation. Zero
// Port means "the scheme's default port" (80/443); the caller documents the
// other sane defaults in the command help.
type LoadBalancerMonitorCreate struct {
	Type          string
	Description   string
	Path          string
	Port          uint16
	ExpectedCodes string
	Interval      int
	Retries       int
	Timeout       int
}

// LoadBalancerMonitorUpdate carries optional changes to an existing
// monitor. Zero values leave the corresponding field untouched.
type LoadBalancerMonitorUpdate struct {
	Type          string
	Description   string
	Path          string
	Port          uint16
	ExpectedCodes string
	Interval      int
	Retries       int
	Timeout       int
}

// LoadBalancerService implements account-scoped load balancer pool and
// monitor operations on top of the typed cloudflare-go resources.
type LoadBalancerService struct {
	cf        *cloudflare.API
	accountID string
}

// NewLoadBalancerService creates a new load balancer service client.
func NewLoadBalancerService(api *cloudflare.API, accountID string) (*LoadBalancerService, error) {
	if api == nil {
		return nil, validationError("NewLoadBalancerService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewLoadBalancerService", "account ID is required")
	}
	return &LoadBalancerService{cf: api, accountID: accountID}, nil
}

// NewLoadBalancerServiceFromCreds creates a LoadBalancerService from account
// ID and API token.
func NewLoadBalancerServiceFromCreds(accountID, apiToken string) (*LoadBalancerService, error) {
	if accountID == "" {
		return nil, validationError("NewLoadBalancerServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewLoadBalancerServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewLoadBalancerServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &LoadBalancerService{cf: cf, accountID: accountID}, nil
}

// Create adds a new load balancer pool with the given origins. The returned
// pool carries the ID Cloudflare assigned.
func (s *LoadBalancerService) Create(ctx context.Context, opts LoadBalancerPoolCreate) (*LoadBalancerPool, error) {
	if opts.Name == "" {
		return nil, validationError("LoadBalancerService.Create", "pool name is required")
	}
	if len(opts.Origins) == 0 {
		return nil, validationError("LoadBalancerService.Create", "at least one origin is required")
	}
	if err := lbOriginsValid(opts.Origins); err != nil {
		return nil, err
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.Create", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	pool, err := s.cf.CreateLoadBalancerPool(ctx, rc, cloudflare.CreateLoadBalancerPoolParams{
		LoadBalancerPool: cloudflare.LoadBalancerPool{
			Name:        opts.Name,
			Description: opts.Description,
			Monitor:     opts.Monitor,
			Enabled:     true,
			Origins:     lbOriginsToCF(opts.Origins),
		},
	})
	if err != nil {
		return nil, newError("LoadBalancerService.Create", fmt.Sprintf("failed to create load balancer pool %q", opts.Name), err)
	}
	mapped := lbPoolFromCF(pool)
	return &mapped, nil
}

// List returns every load balancer pool in the account.
func (s *LoadBalancerService) List(ctx context.Context) ([]LoadBalancerPool, error) {
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.List", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	pools, err := s.cf.ListLoadBalancerPools(ctx, rc, cloudflare.ListLoadBalancerPoolParams{})
	if err != nil {
		return nil, newError("LoadBalancerService.List", "failed to list load balancer pools", err)
	}
	out := make([]LoadBalancerPool, 0, len(pools))
	for _, p := range pools {
		out = append(out, lbPoolFromCF(p))
	}
	return out, nil
}

// Get retrieves a single load balancer pool by its ID.
func (s *LoadBalancerService) Get(ctx context.Context, poolID string) (*LoadBalancerPool, error) {
	if poolID == "" {
		return nil, validationError("LoadBalancerService.Get", "pool ID is required")
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.Get", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	pool, err := s.cf.GetLoadBalancerPool(ctx, rc, poolID)
	if err != nil {
		return nil, newError("LoadBalancerService.Get", fmt.Sprintf("failed to get load balancer pool %s", poolID), err)
	}
	mapped := lbPoolFromCF(pool)
	return &mapped, nil
}

// Health retrieves the per-PoP health detail for a pool.
func (s *LoadBalancerService) Health(ctx context.Context, poolID string) (*LoadBalancerPoolHealth, error) {
	if poolID == "" {
		return nil, validationError("LoadBalancerService.Health", "pool ID is required")
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.Health", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	health, err := s.cf.GetLoadBalancerPoolHealth(ctx, rc, poolID)
	if err != nil {
		return nil, newError("LoadBalancerService.Health", fmt.Sprintf("failed to get health for load balancer pool %s", poolID), err)
	}
	mapped := lbPoolHealthFromCF(health)
	return &mapped, nil
}

// Update applies partial changes to an existing pool. The Cloudflare update
// replaces the whole pool, so the current state is fetched first and the
// non-zero update fields merged on top; the returned pool reflects the state
// the API reported for the update.
func (s *LoadBalancerService) Update(ctx context.Context, poolID string, opts LoadBalancerPoolUpdate) (*LoadBalancerPool, error) {
	if poolID == "" {
		return nil, validationError("LoadBalancerService.Update", "pool ID is required")
	}
	if opts.Origins != nil {
		if err := lbOriginsValid(opts.Origins); err != nil {
			return nil, err
		}
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.Update", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	current, err := s.cf.GetLoadBalancerPool(ctx, rc, poolID)
	if err != nil {
		return nil, newError("LoadBalancerService.Update", fmt.Sprintf("failed to fetch load balancer pool %s before update", poolID), err)
	}

	merged := current
	if opts.Name != "" {
		merged.Name = opts.Name
	}
	if opts.Description != "" {
		merged.Description = opts.Description
	}
	if opts.Monitor != "" {
		merged.Monitor = opts.Monitor
	}
	if opts.Enabled != nil {
		merged.Enabled = *opts.Enabled
	}
	if opts.Origins != nil {
		merged.Origins = lbOriginsToCF(opts.Origins)
	}

	pool, err := s.cf.UpdateLoadBalancerPool(ctx, rc, cloudflare.UpdateLoadBalancerPoolParams{LoadBalancer: merged})
	if err != nil {
		return nil, newError("LoadBalancerService.Update", fmt.Sprintf("failed to update load balancer pool %s", poolID), err)
	}
	mapped := lbPoolFromCF(pool)
	return &mapped, nil
}

// Delete permanently removes a load balancer pool. This is irreversible:
// any load balancer steering traffic to the pool starts failing over.
func (s *LoadBalancerService) Delete(ctx context.Context, poolID string) error {
	if poolID == "" {
		return validationError("LoadBalancerService.Delete", "pool ID is required")
	}
	if s.accountID == "" {
		return validationError("LoadBalancerService.Delete", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteLoadBalancerPool(ctx, rc, poolID); err != nil {
		return newError("LoadBalancerService.Delete", fmt.Sprintf("failed to delete load balancer pool %s", poolID), err)
	}
	return nil
}

// CreateMonitor adds a new load balancer monitor. The returned monitor
// carries the ID Cloudflare assigned.
func (s *LoadBalancerService) CreateMonitor(ctx context.Context, opts LoadBalancerMonitorCreate) (*LoadBalancerMonitor, error) {
	if opts.Type == "" {
		return nil, validationError("LoadBalancerService.CreateMonitor", "monitor type is required (http or https)")
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.CreateMonitor", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	monitor, err := s.cf.CreateLoadBalancerMonitor(ctx, rc, cloudflare.CreateLoadBalancerMonitorParams{
		LoadBalancerMonitor: cloudflare.LoadBalancerMonitor{
			Type:          opts.Type,
			Description:   opts.Description,
			Path:          opts.Path,
			Port:          opts.Port,
			ExpectedCodes: opts.ExpectedCodes,
			Interval:      opts.Interval,
			Retries:       opts.Retries,
			Timeout:       opts.Timeout,
		},
	})
	if err != nil {
		return nil, newError("LoadBalancerService.CreateMonitor", fmt.Sprintf("failed to create load balancer monitor of type %q", opts.Type), err)
	}
	mapped := lbMonitorFromCF(monitor)
	return &mapped, nil
}

// ListMonitors returns every load balancer monitor in the account.
func (s *LoadBalancerService) ListMonitors(ctx context.Context) ([]LoadBalancerMonitor, error) {
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.ListMonitors", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	monitors, err := s.cf.ListLoadBalancerMonitors(ctx, rc, cloudflare.ListLoadBalancerMonitorParams{})
	if err != nil {
		return nil, newError("LoadBalancerService.ListMonitors", "failed to list load balancer monitors", err)
	}
	out := make([]LoadBalancerMonitor, 0, len(monitors))
	for _, m := range monitors {
		out = append(out, lbMonitorFromCF(m))
	}
	return out, nil
}

// GetMonitor retrieves a single load balancer monitor by its ID.
func (s *LoadBalancerService) GetMonitor(ctx context.Context, monitorID string) (*LoadBalancerMonitor, error) {
	if monitorID == "" {
		return nil, validationError("LoadBalancerService.GetMonitor", "monitor ID is required")
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.GetMonitor", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	monitor, err := s.cf.GetLoadBalancerMonitor(ctx, rc, monitorID)
	if err != nil {
		return nil, newError("LoadBalancerService.GetMonitor", fmt.Sprintf("failed to get load balancer monitor %s", monitorID), err)
	}
	mapped := lbMonitorFromCF(monitor)
	return &mapped, nil
}

// UpdateMonitor applies partial changes to an existing monitor. Like pool
// update, the API call replaces the whole monitor, so the current state is
// fetched first and the non-zero update fields merged on top.
func (s *LoadBalancerService) UpdateMonitor(ctx context.Context, monitorID string, opts LoadBalancerMonitorUpdate) (*LoadBalancerMonitor, error) {
	if monitorID == "" {
		return nil, validationError("LoadBalancerService.UpdateMonitor", "monitor ID is required")
	}
	if s.accountID == "" {
		return nil, validationError("LoadBalancerService.UpdateMonitor", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	current, err := s.cf.GetLoadBalancerMonitor(ctx, rc, monitorID)
	if err != nil {
		return nil, newError("LoadBalancerService.UpdateMonitor", fmt.Sprintf("failed to fetch load balancer monitor %s before update", monitorID), err)
	}

	merged := current
	if opts.Type != "" {
		merged.Type = opts.Type
	}
	if opts.Description != "" {
		merged.Description = opts.Description
	}
	if opts.Path != "" {
		merged.Path = opts.Path
	}
	if opts.Port != 0 {
		merged.Port = opts.Port
	}
	if opts.ExpectedCodes != "" {
		merged.ExpectedCodes = opts.ExpectedCodes
	}
	if opts.Interval != 0 {
		merged.Interval = opts.Interval
	}
	if opts.Retries != 0 {
		merged.Retries = opts.Retries
	}
	if opts.Timeout != 0 {
		merged.Timeout = opts.Timeout
	}

	monitor, err := s.cf.UpdateLoadBalancerMonitor(ctx, rc, cloudflare.UpdateLoadBalancerMonitorParams{LoadBalancerMonitor: merged})
	if err != nil {
		return nil, newError("LoadBalancerService.UpdateMonitor", fmt.Sprintf("failed to update load balancer monitor %s", monitorID), err)
	}
	mapped := lbMonitorFromCF(monitor)
	return &mapped, nil
}

// DeleteMonitor permanently removes a load balancer monitor. Pools still
// referencing it keep their last health state until they are re-pointed.
func (s *LoadBalancerService) DeleteMonitor(ctx context.Context, monitorID string) error {
	if monitorID == "" {
		return validationError("LoadBalancerService.DeleteMonitor", "monitor ID is required")
	}
	if s.accountID == "" {
		return validationError("LoadBalancerService.DeleteMonitor", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteLoadBalancerMonitor(ctx, rc, monitorID); err != nil {
		return newError("LoadBalancerService.DeleteMonitor", fmt.Sprintf("failed to delete load balancer monitor %s", monitorID), err)
	}
	return nil
}

// lbOriginsValid rejects origin sets with missing names or addresses before
// any request is issued.
func lbOriginsValid(origins []LoadBalancerOrigin) error {
	for i, o := range origins {
		if o.Name == "" {
			return validationError("LoadBalancerService", fmt.Sprintf("origin %d is missing a name", i))
		}
		if o.Address == "" {
			return validationError("LoadBalancerService", fmt.Sprintf("origin %q is missing an address", o.Name))
		}
	}
	return nil
}

// lbOriginsToCF maps the public origin view to the cloudflare-go type.
func lbOriginsToCF(origins []LoadBalancerOrigin) []cloudflare.LoadBalancerOrigin {
	out := make([]cloudflare.LoadBalancerOrigin, 0, len(origins))
	for _, o := range origins {
		out = append(out, cloudflare.LoadBalancerOrigin{
			Name:    o.Name,
			Address: o.Address,
			Enabled: o.Enabled,
		})
	}
	return out
}

// lbPoolFromCF maps the cloudflare-go pool type to the public view,
// rendering timestamps as RFC 3339 strings and omitting zero times.
func lbPoolFromCF(p cloudflare.LoadBalancerPool) LoadBalancerPool {
	return LoadBalancerPool{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Enabled:      p.Enabled,
		Monitor:      p.Monitor,
		Origins:      lbOriginsFromCF(p.Origins),
		CheckRegions: p.CheckRegions,
		CreatedOn:    lbTimeString(p.CreatedOn),
		ModifiedOn:   lbTimeString(p.ModifiedOn),
		Healthy:      p.Healthy,
	}
}

// lbOriginsFromCF maps cloudflare-go origins to the public view.
func lbOriginsFromCF(origins []cloudflare.LoadBalancerOrigin) []LoadBalancerOrigin {
	out := make([]LoadBalancerOrigin, 0, len(origins))
	for _, o := range origins {
		out = append(out, LoadBalancerOrigin{
			Name:    o.Name,
			Address: o.Address,
			Enabled: o.Enabled,
		})
	}
	return out
}

// lbPoolHealthFromCF maps the cloudflare-go pool health type to the public
// view.
func lbPoolHealthFromCF(h cloudflare.LoadBalancerPoolHealth) LoadBalancerPoolHealth {
	out := LoadBalancerPoolHealth{PoolID: h.ID, Pop: map[string]LoadBalancerPoolPopHealthView{}}
	for pop, ph := range h.PopHealth {
		view := LoadBalancerPoolPopHealthView{Healthy: ph.Healthy}
		for _, originMap := range ph.Origins {
			mapped := map[string]LoadBalancerOriginHealthView{}
			for name, oh := range originMap {
				mapped[name] = LoadBalancerOriginHealthView{
					Healthy:       oh.Healthy,
					FailureReason: oh.FailureReason,
					ResponseCode:  oh.ResponseCode,
				}
			}
			view.Origins = append(view.Origins, mapped)
		}
		out.Pop[pop] = view
	}
	return out
}

// lbMonitorFromCF maps the cloudflare-go monitor type to the public view.
func lbMonitorFromCF(m cloudflare.LoadBalancerMonitor) LoadBalancerMonitor {
	return LoadBalancerMonitor{
		ID:            m.ID,
		Type:          m.Type,
		Description:   m.Description,
		Path:          m.Path,
		Port:          m.Port,
		ExpectedCodes: m.ExpectedCodes,
		Interval:      m.Interval,
		Retries:       m.Retries,
		Timeout:       m.Timeout,
		CreatedOn:     lbTimeString(m.CreatedOn),
		ModifiedOn:    lbTimeString(m.ModifiedOn),
	}
}

// lbTimeString renders an optional timestamp in RFC 3339 form, empty for nil.
func lbTimeString(ts *time.Time) string {
	if ts == nil {
		return ""
	}
	return ts.Format(time.RFC3339)
}
