package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	limitsdata "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/limitsdata"
)

// catalogAliases maps stable Snapshot row ids to catalog ids where the
// research corpus renamed the resource (BR-03 corpus findings). Row ids are
// the CLI surface and stay unchanged; catalog ids are the SSOT vocabulary.
var catalogAliases = map[string]string{
	"workers.scripts":        "workers.scripts_per_account",
	"workers.daily_requests": "workers.requests_daily",
	"r2.buckets":             "r2.buckets_per_account",
	"dns.records":            "dns.records_zone",
}

// catalogID resolves a row id to its catalog id (identity when unaliased).
func catalogID(resource string) string {
	if id, ok := catalogAliases[resource]; ok {
		return id
	}
	return resource
}

// tierValue picks the tier field matching the plan. Unknown or empty plan
// resolves to "paid" (D2). Zone-specific tiers (pro/business) are handled by
// dnsRecordsStaticLimit, which reads the zone plan off the zone object.
func tierValue(t *limitsdata.Tiers, plan string) *json.Number {
	switch plan {
	case "free":
		return t.Free
	case "enterprise":
		return t.Enterprise
	default: // "paid" and any unresolved plan
		return t.Paid
	}
}

// limitFor returns the documented limit for a resource by looking it up in
// the embedded catalog (the single source of truth — BR-03). ok=false when
// the resource is not in the catalog or the tier value is null ("no
// published number" — unknown, never fabricated).
func limitFor(resource, plan string) (limit uint64, ok bool) {
	entry, found := limitsdata.Lookup(catalogID(resource))
	if !found || entry.Tiers == nil {
		return 0, false
	}
	n := tierValue(entry.Tiers, plan)
	if n == nil {
		return 0, false // null tier = no published number
	}
	v, err := strconv.ParseUint(n.String(), 10, 64)
	if err != nil {
		return 0, false // fractional/decimal value — not a countable quota
	}
	return v, true
}

// catalogVerifiedOn returns the catalog entry's verification timestamp for a
// row id ("" when absent or unverified) — the D3 freshness provenance.
func catalogVerifiedOn(resource string) string {
	entry, found := limitsdata.Lookup(catalogID(resource))
	if !found || entry.VerifiedOn == nil {
		return ""
	}
	return entry.VerifiedOn.UTC().Format(time.RFC3339)
}

// unverifiedZoneFallback is the DNS record quota for free zones created
// BEFORE dnsFreeZoneCutoff (2024-09-01). The corpus mentions the 1,000 value
// only in the dns.records_zone notes — the tiers carry 200 — so this stays a
// named, flagged local constant instead of a silent hardcode (BR-03 corpus
// findings: no verified entry ⇒ local constant, flagged unverified).
const unverifiedZoneFallback uint64 = 1000

// dnsFreeZoneCutoff splits Free-zone DNS record quotas: zones created on or
// after this UTC date get 200 records, older ones keep 1,000.
// Source: https://developers.cloudflare.com/dns/manage-dns-records/
var dnsFreeZoneCutoff = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// dnsRecordsStaticLimit returns the per-zone DNS record quota by zone plan
// tier, reading its numbers from the catalog entry dns.records_zone (D2:
// zones carry their plan on the zone object). Used only as the fallback when
// the live DNS usage API is unavailable to the token. Enterprise has no
// per-zone limit (tiers null — account-level quota), so ok=false there, as
// for unknown plans. Only the pre-cutoff free-zone 1,000 comes from the
// named unverifiedZoneFallback constant.
func dnsRecordsStaticLimit(zonePlan string, createdOn time.Time) (limit uint64, ok bool) {
	entry, found := limitsdata.Lookup("dns.records_zone")
	if !found || entry.Tiers == nil {
		return 0, false
	}
	var n *json.Number
	switch zonePlan {
	case "free":
		if createdOn.Before(dnsFreeZoneCutoff) {
			return unverifiedZoneFallback, true // corpus docs-silent — flagged above
		}
		n = entry.Tiers.Free
	case "pro":
		n = entry.Tiers.Pro
	case "business":
		n = entry.Tiers.Business
	default: // enterprise (null tiers) or unknown plan
		n = entry.Tiers.Enterprise
	}
	if n == nil {
		return 0, false
	}
	v, err := strconv.ParseUint(n.String(), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
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
	// VerifiedOn is the catalog entry's verification timestamp (RFC3339,
	// D3 freshness provenance). Empty for live-api limits and unverified
	// local constants.
	VerifiedOn string `json:"verified_on,omitempty"`
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
	PlanSource  string        `json:"plan_source"`  // "auto" | "override" | "config" | "flag" | "unknown"
	// Stale lists the catalog ids behind static-docs rows that are older
	// than 90 days or unverified (nil verified_on — D3 freshness warning).
	Stale []string `json:"stale,omitempty"`
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
	accountID     string
	rest          restClient
	workers       WorkersLister
	r2            BucketLister
	zones         ZoneLister
	domains       BucketDomainLister
	analytics     WorkersAnalytics
	configPlan    string
	flagPlan      string
	planOverrides map[string]string // per-service --plan overrides (D2)
	configPlans   map[string]string // .cosmoflare.yaml plans: map (D2)
	// subscriptions cache: Workers/R2/D1 resolve from ONE call (D2).
	subs       []subscription
	subsLoaded bool
	subsErr    error
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

// WithLimitsPlanOverrides supplies per-service plan tier overrides (D2),
// keyed by service name — e.g. {"d1": "paid", "r2": "free"} — from
// `--plan <service>=<tier>` flags. Precedence per service: subscriptions
// API → override → config plans map → unknown.
func WithLimitsPlanOverrides(m map[string]string) LimitsOption {
	return func(s *LimitsService) { s.planOverrides = m }
}

// WithLimitsConfigPlans supplies the plans: map from .cosmoflare.yaml (D2),
// keyed by service name. It ranks below the subscriptions API and the
// per-service --plan overrides.
func WithLimitsConfigPlans(m map[string]string) LimitsOption {
	return func(s *LimitsService) { s.configPlans = m }
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
	s.appendFreshness(snap)
	return snap, nil
}

// staleAfter is the D3 freshness horizon: catalog entries whose verified_on
// is older than this (or nil — the unverified local constants) are named in
// the snapshot's stale warning.
const staleAfter = 90 * 24 * time.Hour

// appendFreshness fills snap.Stale with the catalog ids behind static-docs
// rows that are older than 90 days or unverified (D3). Live-api rows are
// skipped: their limit value comes from the live quota API, not the catalog.
func (s *LimitsService) appendFreshness(snap *LimitsSnapshot) {
	cutoff := time.Now().UTC().Add(-staleAfter)
	seen := map[string]bool{}
	for _, row := range snap.Rows {
		if row.LimitSource != "static-docs" {
			continue
		}
		entry, ok := limitsdata.Lookup(catalogID(row.Resource))
		if !ok || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		if entry.VerifiedOn == nil || entry.VerifiedOn.Before(cutoff) {
			snap.Stale = append(snap.Stale, entry.ID)
		}
	}
}

// countRow builds a usage-vs-catalog-limit row. limitFor decides the
// limit and source: a known limit yields Limit/Percent/"static-docs" plus
// the catalog entry's verified_on; an unknown one (null tier or resource
// absent from the catalog) yields Limit 0 with "unknown" — never fabricated.
func countRow(resource, scope string, used uint64, plan string) LimitRow {
	limit, ok := limitFor(resource, plan)
	src := "static-docs"
	if !ok {
		src = "unknown"
	}
	return LimitRow{
		Resource: resource, Scope: scope, Used: used, Limit: limit,
		Percent: percentOf(used, limit), PlanTier: plan, LimitSource: src,
		VerifiedOn: catalogVerifiedOn(resource),
	}
}

// dnsConcurrency bounds the parallel per-zone DNS usage fetches.
const dnsConcurrency = 8

// appendDNSRows appends one dns.records row per zone. Live usage comes from
// the DNS quota API (concurrent, bounded); a failed live call falls back to
// the static per-plan table with used 0 — honest, not fabricated. Enterprise
// zones (account-level quota) are skipped: no per-zone limit exists.
//
// Fail-fast (FEAT-014): a 401/403 means the token lacks DNS Read — every
// zone would fail identically — so the first such error stops dispatching
// further zones and records ONE actionable source error instead of failing
// all N calls one by one. Other statuses (e.g. 404 on a zone without the
// endpoint) keep the per-zone static fallback.
func (s *LimitsService) appendDNSRows(ctx context.Context, snap *LimitsSnapshot, zones []*Zone) {
	type dnsResult struct {
		row     LimitRow
		present bool
	}
	results := make([]dnsResult, len(zones))

	var authFailed atomic.Bool
	sem := make(chan struct{}, dnsConcurrency)
	var wg sync.WaitGroup
	for i, z := range zones {
		if authFailed.Load() {
			break // permission already failed — don't start more calls
		}
		wg.Add(1)
		go func(idx int, zone *Zone) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if authFailed.Load() {
				return // queued behind the semaphore when the permission failure landed
			}

			used, limit, src, err := s.dnsUsage(ctx, zone.ID)
			if err != nil {
				if status := ErrorStatus(err); status == http.StatusUnauthorized || status == http.StatusForbidden {
					if authFailed.CompareAndSwap(false, true) {
						snap.recordSourceError("dns.usage", fmt.Errorf(
							"zone %s: %v — the token likely lacks DNS Read; skipping DNS usage for the remaining zones", zone.Name, err))
					}
					return
				}
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
	var subs []subscription
	if err := s.rest.do(ctx, op, http.MethodGet, "/accounts/"+s.accountID+"/subscriptions", nil, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

// normalizePlanTier validates a tier string from any source. Only "free" and
// "paid" are Workers plan tiers; anything else is "unknown".
func normalizePlanTier(plan string) string {
	switch normalizeCatalogPlan(plan) {
	case "":
		return "unknown"
	case "enterprise":
		// Not a self-serve Workers tier — rejected, not trusted.
		return "unknown"
	default:
		return normalizeCatalogPlan(plan)
	}
}

// normalizeCatalogPlan validates a tier string against the catalog's tier
// vocabulary (free | paid | enterprise — zone-only tiers are read off the
// zone object, never resolved here). Returns "" for anything invalid.
func normalizeCatalogPlan(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "free":
		return "free"
	case "paid":
		return "paid"
	case "enterprise":
		return "enterprise"
	default:
		return ""
	}
}

// cachedSubscriptions fetches the account subscriptions once per service
// lifetime (Workers/R2/D1 all resolve from the same call — D2).
func (s *LimitsService) cachedSubscriptions(ctx context.Context) ([]subscription, error) {
	if !s.subsLoaded {
		s.subs, s.subsErr = s.fetchSubscriptions(ctx)
		s.subsLoaded = true
	}
	return s.subs, s.subsErr
}

// tierFromSubscriptions resolves a service's tier from the cached
// subscriptions list: the entry whose rate-plan id starts with
// "<service>_" or whose product name matches. "paid" in the rate-plan id or
// public name means paid; anything else readable means free.
func (s *LimitsService) tierFromSubscriptions(ctx context.Context, service string) (string, bool) {
	subs, err := s.cachedSubscriptions(ctx)
	if err != nil {
		return "", false
	}
	for _, sub := range subs {
		id := sub.RatePlan.ID
		if strings.HasPrefix(id, service+"_") || sub.Product.Name == service {
			if strings.Contains(id, "paid") || strings.Contains(strings.ToLower(sub.RatePlan.PublicName), "paid") {
				return "paid", true
			}
			return "free", true
		}
	}
	return "", false
}

// resolveWorkersPlan resolves the Workers plan tier. Order (D2): cached
// subscriptions API → per-service --plan override → --plan flag →
// workers_plan config → config plans: map → unknown. A subscriptions
// failure is silent — the source string records which path decided.
func (s *LimitsService) resolveWorkersPlan(ctx context.Context) (plan, source string) {
	if tier, ok := s.tierFromSubscriptions(ctx, "workers"); ok {
		return tier, "auto"
	}
	if tier := normalizePlanTier(s.planOverrides["workers"]); tier != "unknown" {
		return tier, "override"
	}
	if tier := normalizePlanTier(s.flagPlan); tier != "unknown" {
		return tier, "flag"
	}
	if tier := normalizePlanTier(s.configPlan); tier != "unknown" {
		return tier, "config"
	}
	if tier := normalizeCatalogPlan(s.configPlans["workers"]); tier != "" {
		return tier, "config"
	}
	return "unknown", "unknown"
}

// planFor resolves the plan tier for one service (D2 per-service chain).
// Order: cached subscriptions API → per-service --plan override → config
// plans: map → "unknown". The subscriptions call is shared with the Workers
// resolution (one round-trip). The returned error is non-nil only when the
// subscriptions API failed — the tier is still the best-effort value, so
// callers may ignore the error for tier purposes.
func (s *LimitsService) planFor(ctx context.Context, service string) (tier string, err error) {
	if tier, ok := s.tierFromSubscriptions(ctx, service); ok {
		return tier, nil
	}
	_, subsErr := s.cachedSubscriptions(ctx)
	if tier := normalizeCatalogPlan(s.planOverrides[service]); tier != "" {
		return tier, subsErr
	}
	if tier := normalizeCatalogPlan(s.configPlans[service]); tier != "" {
		return tier, subsErr
	}
	return "unknown", subsErr
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
