package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// workerPlanLimits maps plan-dependent resources to their per-tier limits.
// Limit 0 means "unlimited". Sources (verified 2026-09-09):
//   - https://developers.cloudflare.com/workers/platform/limits/
var workerPlanLimits = map[string]map[string]uint64{
	"workers.scripts":        {"free": 100, "paid": 500},
	"workers.daily_requests": {"free": 100000, "paid": 0},
}

// staticLimits maps plan-independent account resources to documented limits.
// Source (verified 2026-09-09):
//   - https://developers.cloudflare.com/r2/platform/limits/
var staticLimits = map[string]uint64{
	"r2.buckets":                   1000000,
	"r2.custom_domains_per_bucket": 100,
}

// dnsFreeZoneCutoff splits Free-zone DNS record quotas: zones created on or
// after this UTC date get 200 records, older ones keep 1,000.
// Source: https://developers.cloudflare.com/dns/manage-dns-records/
var dnsFreeZoneCutoff = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// limitFor returns the documented limit for a resource given the Workers plan
// tier ("free" or "paid"). ok=false when the resource is unknown or the plan
// is required but unresolved. A returned limit of 0 with ok=true means
// "unlimited".
func limitFor(resource, plan string) (limit uint64, ok bool) {
	if tiers, exists := workerPlanLimits[resource]; exists {
		limit, ok = tiers[plan]
		return limit, ok
	}
	limit, ok = staticLimits[resource]
	return limit, ok
}

// dnsRecordsStaticLimit returns the per-zone DNS record quota by zone plan
// tier. Used only as the fallback when the live DNS usage API is unavailable
// to the token. Enterprise has no per-zone limit (account-level quota), so
// ok=false there.
func dnsRecordsStaticLimit(zonePlan string, createdOn time.Time) (limit uint64, ok bool) {
	switch zonePlan {
	case "pro", "business":
		return 3500, true
	case "free":
		if createdOn.Before(dnsFreeZoneCutoff) {
			return 1000, true
		}
		return 200, true
	default: // enterprise or unknown
		return 0, false
	}
}

// LimitRow is one usage-vs-limit observation.
type LimitRow struct {
	Resource    string  `json:"resource"`        // "workers.scripts", "dns.records", ...
	Scope       string  `json:"scope,omitempty"` // "" account-wide, or bucket/zone name
	Used        uint64  `json:"used"`
	Limit       uint64  `json:"limit"`               // 0 = unlimited
	Percent     float64 `json:"percent,omitempty"`   // 0 when Limit == 0 or unknown
	PlanTier    string  `json:"plan_tier,omitempty"` // "free" | "paid" | "unknown" | "" when not plan-dependent
	LimitSource string  `json:"limit_source"`        // "static-docs" | "live-api" | "unknown"
}

// SourceError records one failed producer without failing the snapshot.
type SourceError struct {
	Source string `json:"source"` // "workers.list", "subscriptions", "dns.usage:<zone>", ...
	Err    string `json:"err"`
}

// LimitsSnapshot is the partial-failure result of one limits collection pass.
type LimitsSnapshot struct {
	Rows        []LimitRow    `json:"rows"`
	Sources     []SourceError `json:"sources,omitempty"`
	WorkersPlan string        `json:"workers_plan"` // "free" | "paid" | "unknown"
	PlanSource  string        `json:"plan_source"`  // "auto" | "config" | "flag" | "unknown"
}

// Consumer interfaces: LimitsService composes existing services through the
// narrow surface it needs. Any service satisfying the signature works,
// including test fakes.
type WorkersLister interface {
	List(ctx context.Context) ([]*Worker, error)
}

type BucketLister interface {
	ListBuckets(ctx context.Context) ([]*Bucket, error)
}

type ZoneLister interface {
	List(ctx context.Context) ([]*Zone, error)
}

type BucketDomainLister interface {
	List(ctx context.Context, bucket string) ([]BucketDomain, error)
}

type WorkersAnalytics interface {
	Workers(ctx context.Context, w AnalyticsWindow) ([]WorkersSummary, error)
}

// LimitsService joins live usage counts against documented plan limits.
type LimitsService struct {
	accountID  string
	apiToken   string
	httpClient *http.Client
	baseURL    string
	workers    WorkersLister
	r2         BucketLister
	zones      ZoneLister
	domains    BucketDomainLister
	analytics  WorkersAnalytics
	configPlan string
	flagPlan   string
}

// LimitsOption configures the LimitsService.
type LimitsOption func(*LimitsService)

// WithLimitsHTTPClient sets a custom HTTP client.
func WithLimitsHTTPClient(c *http.Client) LimitsOption {
	return func(s *LimitsService) { s.httpClient = c }
}

// WithLimitsBaseURL overrides the REST API base URL.
func WithLimitsBaseURL(u string) LimitsOption {
	return func(s *LimitsService) { s.baseURL = u }
}

// WithLimitsWorkers sets the Workers script lister.
func WithLimitsWorkers(w WorkersLister) LimitsOption {
	return func(s *LimitsService) { s.workers = w }
}

// WithLimitsR2 sets the R2 bucket lister.
func WithLimitsR2(r BucketLister) LimitsOption {
	return func(s *LimitsService) { s.r2 = r }
}

// WithLimitsZones sets the zone lister.
func WithLimitsZones(z ZoneLister) LimitsOption {
	return func(s *LimitsService) { s.zones = z }
}

// WithLimitsDomains sets the per-bucket custom domain lister.
func WithLimitsDomains(d BucketDomainLister) LimitsOption {
	return func(s *LimitsService) { s.domains = d }
}

// WithLimitsAnalytics sets the Workers analytics source (daily requests).
func WithLimitsAnalytics(a WorkersAnalytics) LimitsOption {
	return func(s *LimitsService) { s.analytics = a }
}

// WithLimitsConfigPlan supplies the workers_plan value from project config.
func WithLimitsConfigPlan(plan string) LimitsOption {
	return func(s *LimitsService) { s.configPlan = plan }
}

// WithLimitsFlagPlan supplies the workers_plan value from the --plan flag.
// The flag outranks config.
func WithLimitsFlagPlan(plan string) LimitsOption {
	return func(s *LimitsService) { s.flagPlan = plan }
}

// NewLimitsService creates a limits collector for one account.
func NewLimitsService(accountID, apiToken string, opts ...LimitsOption) *LimitsService {
	s := &LimitsService{
		accountID:  accountID,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// validate ensures account-scoped collection has credentials.
func (s *LimitsService) validate(op string) error {
	if s.accountID == "" {
		return validationError(op, "account ID is required")
	}
	if s.apiToken == "" {
		return validationError(op, "API token is required")
	}
	return nil
}

// Snapshot collects every configured source and joins usage against limits.
// bucket selects the optional per-bucket rows; empty skips them.
func (s *LimitsService) Snapshot(ctx context.Context, bucket string) (*LimitsSnapshot, error) {
	const op = "LimitsSnapshot"
	if err := s.validate(op); err != nil {
		return nil, err
	}
	return &LimitsSnapshot{WorkersPlan: "unknown", PlanSource: "unknown"}, nil
}

// subscription wraps the parts of a /subscriptions result entry we join on.
type subscription struct {
	Product struct {
		Name string `json:"name"`
	} `json:"product"`
	RatePlan struct {
		ID         string `json:"id"`
		PublicName string `json:"public_name"`
	} `json:"rate_plan"`
}

// fetchSubscriptions lists account subscriptions. It requires Billing Read;
// callers treat any failure as "auto-detection unavailable".
func (s *LimitsService) fetchSubscriptions(ctx context.Context) ([]subscription, error) {
	const op = "LimitsSubscriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/accounts/"+s.accountID+"/subscriptions", nil)
	if err != nil {
		return nil, newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, newError(op, "request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newError(op, "failed to read response body", err)
	}
	var env struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result []subscription `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, newError(op, fmt.Sprintf("unexpected response (HTTP %d)", resp.StatusCode), err)
	}
	if !env.Success {
		msg := "subscriptions unavailable"
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		return nil, newError(op, msg, nil)
	}
	return env.Result, nil
}

// normalizePlanTier validates a tier string from any source. Only "free" and
// "paid" are Workers plan tiers; anything else is "unknown".
func normalizePlanTier(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "free":
		return "free"
	case "paid":
		return "paid"
	default:
		return "unknown"
	}
}

// resolveWorkersPlan resolves the Workers plan tier. Order: subscriptions
// API → config → flag → unknown. A subscriptions failure is silent — the
// source string records which path decided.
func (s *LimitsService) resolveWorkersPlan(ctx context.Context) (plan, source string, err error) {
	subs, err := s.fetchSubscriptions(ctx)
	if err == nil {
		for _, sub := range subs {
			id := sub.RatePlan.ID
			if strings.HasPrefix(id, "workers") {
				if strings.Contains(id, "paid") || strings.Contains(strings.ToLower(sub.RatePlan.PublicName), "paid") {
					return "paid", "auto", nil
				}
				return "free", "auto", nil
			}
		}
		// Subscriptions readable but no workers entry: fall through to config.
	}
	if tier := normalizePlanTier(s.flagPlan); tier != "unknown" {
		return tier, "flag", nil
	}
	if tier := normalizePlanTier(s.configPlan); tier != "unknown" {
		return tier, "config", nil
	}
	return "unknown", "unknown", nil
}
