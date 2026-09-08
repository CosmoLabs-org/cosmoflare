package cosmoflare

import (
	"context"
	"math"
	"net/http"
	"strings"
	"sync"
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
	rest       restClient
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
	return func(s *LimitsService) { s.rest.httpClient = c }
}

// WithLimitsBaseURL overrides the REST API base URL.
func WithLimitsBaseURL(u string) LimitsOption {
	return func(s *LimitsService) { s.rest.baseURL = u }
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

// NewLimitsService creates a limits collector for one account. Sources are
// injected via options; a source left unset is skipped by Snapshot.
func NewLimitsService(accountID, apiToken string, opts ...LimitsOption) *LimitsService {
	s := &LimitsService{
		accountID: accountID,
		rest:      newRESTClient(apiToken),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewLimitsServiceFromCreds composes the standard production sources over one
// credential pair — the constructor call sites should use so the collector
// cannot be accidentally wired inert. A source whose constructor fails is
// left unset (Snapshot skips it) rather than failing the whole collector.
func NewLimitsServiceFromCreds(accountID, apiToken string) *LimitsService {
	s := NewLimitsService(accountID, apiToken)
	if client, err := NewClient(WithAccountID(accountID), WithAPIToken(apiToken)); err == nil {
		s.r2 = client
	}
	if w, err := NewWorkerServiceFromCreds(accountID, apiToken); err == nil {
		s.workers = w
	}
	if z, err := NewZoneServiceFromCreds(accountID, apiToken); err == nil {
		s.zones = z
	}
	s.domains = NewBucketDomainService(accountID, apiToken)
	s.analytics = NewAnalyticsService(accountID, apiToken)
	return s
}

// validate ensures account-scoped collection has credentials.
func (s *LimitsService) validate(op string) error {
	if s.accountID == "" {
		return validationError(op, "account ID is required")
	}
	if s.rest.apiToken == "" {
		return validationError(op, "API token is required")
	}
	return nil
}

// percentOf computes used/limit as a percentage rounded to one decimal.
// Returns 0 for unlimited or unknown limits.
func percentOf(used, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return math.Round(float64(used)/float64(limit)*1000) / 10
}

// Snapshot collects every configured source and joins usage against limits.
// bucket selects optional per-bucket rows; empty skips them. Every source is
// independent: a failure records a SourceError and the snapshot continues.
// Per-zone DNS usage is fetched concurrently (bounded) — the calls are
// independent reads. Only zero rows + at least one error is a hard failure.
func (s *LimitsService) Snapshot(ctx context.Context, bucket string) (*LimitsSnapshot, error) {
	const op = "LimitsSnapshot"
	if err := s.validate(op); err != nil {
		return nil, err
	}

	snap := &LimitsSnapshot{}

	// The plan tier only influences plan-dependent rows; skip the
	// subscriptions round-trip entirely when no such source is wired.
	plan, source := "unknown", "unknown"
	if s.workers != nil || s.analytics != nil {
		plan, source = s.resolveWorkersPlan(ctx)
	}
	snap.WorkersPlan, snap.PlanSource = plan, source

	// workers.scripts
	if s.workers != nil {
		scripts, err := s.workers.List(ctx)
		if err != nil {
			snap.recordSourceError("workers.list", err)
		} else {
			snap.Rows = append(snap.Rows, countRow("workers.scripts", "", uint64(len(scripts)), plan))
		}
	}

	// workers.daily_requests (today UTC)
	if s.analytics != nil {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		summaries, err := s.analytics.Workers(ctx, AnalyticsWindow{Start: start, End: now})
		if err != nil {
			snap.recordSourceError("analytics.workers", err)
		} else {
			var reqs uint64
			for _, sum := range summaries {
				reqs += sum.Requests
			}
			snap.Rows = append(snap.Rows, countRow("workers.daily_requests", "", reqs, plan))
		}
	}

	// r2.buckets
	if s.r2 != nil {
		buckets, err := s.r2.ListBuckets(ctx)
		if err != nil {
			snap.recordSourceError("r2.list_buckets", err)
		} else {
			snap.Rows = append(snap.Rows, countRow("r2.buckets", "", uint64(len(buckets)), ""))
		}
	}

	// r2.custom_domains_per_bucket (only with --bucket)
	if bucket != "" && s.domains != nil {
		domains, err := s.domains.List(ctx, bucket)
		if err != nil {
			snap.recordSourceError("r2.bucket_domains:"+bucket, err)
		} else {
			snap.Rows = append(snap.Rows, countRow("r2.custom_domains_per_bucket", bucket, uint64(len(domains)), ""))
		}
	}

	// zones.count + dns.records per zone
	if s.zones != nil {
		zones, err := s.zones.List(ctx)
		if err != nil {
			snap.recordSourceError("zones.list", err)
		} else {
			snap.Rows = append(snap.Rows, LimitRow{
				Resource: "zones.count", Used: uint64(len(zones)), Limit: 0, LimitSource: "unknown", // no documented account cap
			})
			s.appendDNSRows(ctx, snap, zones)
		}
	}

	if len(snap.Rows) == 0 && len(snap.Sources) > 0 {
		return nil, newError(op, "all limit sources failed", nil)
	}
	return snap, nil
}

// countRow builds a usage-vs-documented-limit row. limitFor decides the
// limit and source: a known limit yields Limit/Percent/"static-docs"; an
// unknown one (unresolved plan or undocumented resource) yields
// Limit 0 with "unknown" — never fabricated.
func countRow(resource, scope string, used uint64, plan string) LimitRow {
	limit, ok := limitFor(resource, plan)
	src := "static-docs"
	if !ok {
		src = "unknown"
	}
	return LimitRow{
		Resource: resource, Scope: scope, Used: used, Limit: limit,
		Percent: percentOf(used, limit), PlanTier: plan, LimitSource: src,
	}
}

// dnsConcurrency bounds the parallel per-zone DNS usage fetches.
const dnsConcurrency = 8

// appendDNSRows appends one dns.records row per zone. Live usage comes from
// the DNS quota API (concurrent, bounded); a failed live call falls back to
// the static per-plan table with used 0 — honest, not fabricated. Enterprise
// zones (account-level quota) are skipped: no per-zone limit exists.
func (s *LimitsService) appendDNSRows(ctx context.Context, snap *LimitsSnapshot, zones []*Zone) {
	type dnsResult struct {
		row     LimitRow
		present bool
	}
	results := make([]dnsResult, len(zones))

	sem := make(chan struct{}, dnsConcurrency)
	var wg sync.WaitGroup
	for i, z := range zones {
		wg.Add(1)
		go func(idx int, zone *Zone) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			used, limit, src, err := s.dnsUsage(ctx, zone.ID)
			if err != nil {
				staticLimit, ok := dnsRecordsStaticLimit(zone.Plan.LegacyID, zone.CreatedOn)
				if !ok {
					return // enterprise: account-level quota, nothing per-zone to report
				}
				used, limit, src = 0, staticLimit, "static-docs"
			}
			results[idx] = dnsResult{row: LimitRow{
				Resource: "dns.records", Scope: zone.Name, Used: used, Limit: limit,
				Percent: percentOf(used, limit), LimitSource: src,
			}, present: true}
		}(i, z)
	}
	wg.Wait()

	for _, r := range results {
		if r.present {
			snap.Rows = append(snap.Rows, r.row)
		}
	}
}

// recordSourceError appends one producer failure.
func (s *LimitsSnapshot) recordSourceError(source string, err error) {
	s.Sources = append(s.Sources, SourceError{Source: source, Err: err.Error()})
}

// subscription wraps the parts of a /subscriptions result entry we join on.
type subscription struct {
	RatePlan struct {
		ID         string `json:"id"`
		PublicName string `json:"public_name"`
	} `json:"rate_plan"`
}

// fetchSubscriptions lists account subscriptions. It requires Billing Read;
// callers treat any failure as "auto-detection unavailable".
func (s *LimitsService) fetchSubscriptions(ctx context.Context) ([]subscription, error) {
	const op = "LimitsSubscriptions"
	var subs []subscription
	if err := s.rest.do(ctx, op, http.MethodGet, "/accounts/"+s.accountID+"/subscriptions", nil, &subs); err != nil {
		return nil, err
	}
	return subs, nil
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
// API → flag → config → unknown. A subscriptions failure is silent — the
// source string records which path decided.
func (s *LimitsService) resolveWorkersPlan(ctx context.Context) (plan, source string) {
	subs, err := s.fetchSubscriptions(ctx)
	if err == nil {
		for _, sub := range subs {
			id := sub.RatePlan.ID
			if strings.HasPrefix(id, "workers") {
				if strings.Contains(id, "paid") || strings.Contains(strings.ToLower(sub.RatePlan.PublicName), "paid") {
					return "paid", "auto"
				}
				return "free", "auto"
			}
		}
		// Subscriptions readable but no workers entry: fall through to config.
	}
	if tier := normalizePlanTier(s.flagPlan); tier != "unknown" {
		return tier, "flag"
	}
	if tier := normalizePlanTier(s.configPlan); tier != "unknown" {
		return tier, "config"
	}
	return "unknown", "unknown"
}

// dnsUsage fetches one zone's DNS record usage and quota from the live API.
// Endpoint and field names live-pinned 2026-09-09 against the API reference
// (dns → usage → zone → get): GET /zones/{id}/dns_records/usage returns
// result {record_usage, record_quota}. record_quota is null when an
// account-level quota applies — that yields an error so the caller falls
// back to the static per-plan table instead of treating null as zero.
// Requires DNS Read (or Zone DNS Settings Read) on the token.
func (s *LimitsService) dnsUsage(ctx context.Context, zoneID string) (used, limit uint64, source string, err error) {
	const op = "LimitsDNSUsage"
	var out struct {
		RecordUsage *uint64 `json:"record_usage"`
		RecordQuota *uint64 `json:"record_quota"`
	}
	if err := s.rest.do(ctx, op, http.MethodGet, "/zones/"+zoneID+"/dns_records/usage", nil, &out); err != nil {
		return 0, 0, "", err
	}
	if out.RecordUsage == nil || out.RecordQuota == nil {
		return 0, 0, "", newError(op, "dns usage response missing fields", nil)
	}
	return *out.RecordUsage, *out.RecordQuota, "live-api", nil
}
