package cosmoflare

import "time"

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
