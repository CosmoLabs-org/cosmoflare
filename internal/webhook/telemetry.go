package webhook

import (
	"context"
	"log"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TelemetryRefs carries the zone and D1 name lists the zone/d1 telemetry
// needs, plus the error from fetching each list. Callers own caching.
type TelemetryRefs struct {
	Zones      []cosmoflare.ZoneRef
	ZonesErr   error
	DBNames    map[string]string // D1 database ID → name
	DBNamesErr error
}

// CollectTelemetryMetrics augments m with zone cache and D1 rows-read
// telemetry (FEAT-049). It is additive (design D15): it never returns an
// error. Every failure that leaves a wanted scope blind lands in m.Gaps, so
// the evaluator pages a telemetry gap instead of the rules going silent:
//   - wantZones with a failed zone list, or a failed zone query → Gaps["zone"]
//   - wantD1 with a failed D1 query → Gaps["d1"]
//
// A failed D1 name list is NOT a gap: rows keep the database ID as their
// name (design D8 fallback).
func CollectTelemetryMetrics(ctx context.Context, analytics *cosmoflare.AnalyticsService, wantZones, wantD1 bool, refs TelemetryRefs, w cosmoflare.AnalyticsWindow, m *EvalMetrics) {
	if m == nil || analytics == nil {
		return
	}
	gap := func(scope string, err error) {
		if m.Gaps == nil {
			m.Gaps = make(map[string]string)
		}
		m.Gaps[scope] = err.Error()
		log.Printf("[alerts] %s telemetry unavailable, %s rules cannot fire this cycle: %v", scope, scope, err)
	}

	if wantZones {
		switch {
		case refs.ZonesErr != nil:
			gap("zone", refs.ZonesErr)
		case len(refs.Zones) > 0:
			zones, err := analytics.ZoneCache(ctx, refs.Zones, w)
			if err != nil {
				gap("zone", err)
			} else {
				m.Zones = zones
			}
		}
	}

	if wantD1 {
		if refs.DBNamesErr != nil {
			log.Printf("[alerts] D1 database list unavailable, d1 alerts name databases by ID: %v", refs.DBNamesErr)
		}
		rows, err := analytics.D1RowsRead(ctx, w)
		if err != nil {
			gap("d1", err)
			return
		}
		for i := range rows {
			if name := refs.DBNames[rows[i].DatabaseID]; name != "" {
				rows[i].Name = name
			} else {
				rows[i].Name = rows[i].DatabaseID
			}
		}
		m.D1 = rows
	}
}
