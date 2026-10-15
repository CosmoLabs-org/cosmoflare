package cosmoflare

import "strings"

// TTLVerdict grades one cached route's lifetime (FEAT-068).
type TTLVerdict struct {
	Route         string `json:"route"`
	KeyPrefix     string `json:"key_prefix"`
	TTLSeconds    int    `json:"ttl_seconds"`
	WritesPerDay  int64  `json:"writes_per_day"` // measured, database-wide
	Cadence       string `json:"cadence"`        // "rare" | "daily" | "hot"
	Covered       bool   `json:"invalidation_covered"`
	CoverageKnown bool   `json:"coverage_known"` // false when KeyPrefix == ""
	Verdict       string `json:"verdict"`        // "appropriate" | "raise" | "stale-risk" | "no-invalidation"
	Reason        string `json:"reason"`
}

// Cadence and verdict thresholds (named so tests can pin them).
const (
	TTLCadenceRareBelow = 1     // writes/day < 1 → rare
	TTLCadenceHotAbove  = 1000  // writes/day > 1000 → hot
	TTLStaleRiskAbove   = 3600  // uncovered TTL past 1h with non-rare cadence → stale-risk
	TTLRaiseBelow       = 21600 // covered + rare + TTL under 6h → raise
)

// TTLAsymmetryNote is appended to every verdict Reason: KV entries are
// purgeable from code; the Cache-Control layer is not.
const TTLAsymmetryNote = "KV TTLs can be purged by code, but the browser/edge Cache-Control layer cannot be targeted from a migration — keep Cache-Control max-age short (≤ 300s) even when KV TTLs are hours."

// InvalidationCovered reports whether any invalidation literal shares at
// least 2 leading colon-segments with the route's key prefix (Verified
// fact 4: depth 1 — the bare "api" bucket — never counts as coverage).
func InvalidationCovered(routePrefix string, invalidations []InvalidationCall) bool {
	if routePrefix == "" {
		return false
	}
	routeSegs := strings.Split(routePrefix, ":")
	for _, inv := range invalidations {
		if inv.KeyPattern == "" {
			continue
		}
		invSegs := strings.Split(inv.KeyPattern, ":")
		shared := 0
		for i := 0; i < len(routeSegs) && i < len(invSegs); i++ {
			if routeSegs[i] != invSegs[i] {
				break
			}
			shared++
		}
		if shared >= 2 {
			return true
		}
	}
	return false
}

// ttlCadence buckets the measured writes/day (int64 seam — sub-daily rates
// like 0.2 writes/day arrive as 0 and read as "rare", by design).
func ttlCadence(writesPerDay int64) string {
	switch {
	case writesPerDay < TTLCadenceRareBelow:
		return "rare"
	case writesPerDay > TTLCadenceHotAbove:
		return "hot"
	default:
		return "daily"
	}
}

// GradeTTLs grades every inventory route. writesPerDay is the measured
// database-wide D1 rows-written average (caller computes from DailyUsage).
func GradeTTLs(inv *ProjectCacheInventory, writesPerDay int64) []TTLVerdict {
	verdicts := make([]TTLVerdict, 0, len(inv.Routes))
	cadence := ttlCadence(writesPerDay)
	for _, route := range inv.Routes {
		v := TTLVerdict{
			Route:        route.Route,
			KeyPrefix:    route.KeyPrefix,
			TTLSeconds:   route.TTL,
			WritesPerDay: writesPerDay,
			Cadence:      cadence,
			Verdict:      "appropriate",
		}
		switch {
		case route.KeyPrefix == "":
			// Function-derived key: static analysis cannot prove or refute
			// coverage, so the verdict reports unknown and stays appropriate.
			v.CoverageKnown = false
			v.Reason = "coverage not statically determinable (function-derived key): judge at runtime whether anything purges this prefix. " + TTLAsymmetryNote
		default:
			v.CoverageKnown = true
			v.Covered = InvalidationCovered(route.KeyPrefix, inv.Invalidation)
			switch {
			case !v.Covered && route.TTL > TTLStaleRiskAbove && cadence != "rare":
				v.Verdict = "stale-risk"
				v.Reason = "nothing purges this prefix; entries can serve data up to TTL old after a write. " + TTLAsymmetryNote
			case !v.Covered && cadence != "rare" && route.TTL <= TTLStaleRiskAbove:
				v.Verdict = "no-invalidation"
				v.Reason = "safe today only because the TTL is short; adding invalidation is the upgrade path to longer TTLs. " + TTLAsymmetryNote
			case v.Covered && cadence == "rare" && route.TTL < TTLRaiseBelow:
				v.Verdict = "raise"
				v.Reason = "rare-change data with invalidation covered can safely take 6-24h. " + TTLAsymmetryNote
			default:
				v.Verdict = "appropriate"
				v.Reason = "coverage and cadence are consistent with this TTL. " + TTLAsymmetryNote
			}
		}
		verdicts = append(verdicts, v)
	}
	return verdicts
}
