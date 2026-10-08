package cosmoflare

import (
	"context"
	"time"
)

// ZoneRef identifies a zone for batched zone analytics (FEAT-049).
type ZoneRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZoneCacheSummary counts one zone's eyeball requests per cacheStatus over a
// window (FEAT-049 wave 3a). Every status the API returns is kept, including
// statuses no ratio uses (none, ignored, …), so JSON output stays complete.
type ZoneCacheSummary struct {
	ZoneID   string            `json:"zone_id"`
	Zone     string            `json:"zone"`
	ByStatus map[string]uint64 `json:"by_status"`
}

// Cache status sets — design record D3, D4, D5, D13.
var (
	cacheEligibleStatuses = []string{"hit", "miss", "expired", "revalidated", "updating", "stale"}
	cacheMissStatuses     = []string{"miss", "expired"}
	cacheUncachedStatuses = []string{"dynamic", "bypass"}
)

// zoneCacheBatchSize is the documented cap on zones per zone-scoped query.
const zoneCacheBatchSize = 10

func (z ZoneCacheSummary) total(statuses []string) uint64 {
	var n uint64
	for _, st := range statuses {
		n += z.ByStatus[st]
	}
	return n
}

// MissPct returns (miss+expired) ÷ cache-eligible requests × 100. ok is false
// when the eligible count is zero or below floor (D6).
func (z ZoneCacheSummary) MissPct(floor uint64) (float64, bool) {
	eligible := z.total(cacheEligibleStatuses)
	if eligible == 0 || eligible < floor {
		return 0, false
	}
	return 100 * float64(z.total(cacheMissStatuses)) / float64(eligible), true
}

// UncachedPct returns (dynamic+bypass) ÷ known-status requests × 100, where
// known = eligible + dynamic + bypass. ok is false when known is zero or
// below floor (D4, D6).
func (z ZoneCacheSummary) UncachedPct(floor uint64) (float64, bool) {
	uncached := z.total(cacheUncachedStatuses)
	known := z.total(cacheEligibleStatuses) + uncached
	if known == 0 || known < floor {
		return 0, false
	}
	return 100 * float64(uncached) / float64(known), true
}

// ZoneCache returns per-zone eyeball request counts grouped by cacheStatus
// over the window. Zones are queried in batches of 10 with zoneTag_in — one
// request per batch, never one per zone. A zone with no traffic comes back
// with an empty ByStatus map. Results follow the input order.
func (s *AnalyticsService) ZoneCache(ctx context.Context, zones []ZoneRef, w AnalyticsWindow) ([]ZoneCacheSummary, error) {
	const op = "AnalyticsZoneCache"
	if s.apiToken == "" {
		return nil, validationError(op, "API token is required")
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($zoneTags: [String!], $start: Time!, $end: Time!) {
  viewer {
    zones(filter: {zoneTag_in: $zoneTags}) {
      zoneTag
      httpRequestsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end, requestSource: "eyeball"}) {
        count
        dimensions { cacheStatus }
      }
    }
  }
}`
	out := make([]ZoneCacheSummary, 0, len(zones))
	for i := 0; i < len(zones); i += zoneCacheBatchSize {
		end := i + zoneCacheBatchSize
		if end > len(zones) {
			end = len(zones)
		}
		batch := zones[i:end]
		tags := make([]string, len(batch))
		for j, z := range batch {
			tags[j] = z.ID
		}
		var resp struct {
			Viewer struct {
				Zones []struct {
					ZoneTag string `json:"zoneTag"`
					Groups  []struct {
						Count      uint64 `json:"count"`
						Dimensions struct {
							CacheStatus string `json:"cacheStatus"`
						} `json:"dimensions"`
					} `json:"httpRequestsAdaptiveGroups"`
				} `json:"zones"`
			} `json:"viewer"`
		}
		vars := map[string]any{
			"zoneTags": tags,
			"start":    w.Start.Format(time.RFC3339),
			"end":      w.End.Format(time.RFC3339),
		}
		if err := s.query(ctx, op, gql, vars, &resp); err != nil {
			return nil, err
		}
		byID := make(map[string]map[string]uint64, len(resp.Viewer.Zones))
		for _, z := range resp.Viewer.Zones {
			m := byID[z.ZoneTag]
			if m == nil {
				m = make(map[string]uint64)
				byID[z.ZoneTag] = m
			}
			for _, g := range z.Groups {
				m[g.Dimensions.CacheStatus] += g.Count
			}
		}
		for _, z := range batch {
			m := byID[z.ID]
			if m == nil {
				m = make(map[string]uint64)
			}
			out = append(out, ZoneCacheSummary{ZoneID: z.ID, Zone: z.Name, ByStatus: m})
		}
	}
	return out, nil
}
