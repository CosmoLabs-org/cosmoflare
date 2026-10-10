package webhook

import (
	"context"
	"log"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TelemetryRefs carries the zone, D1 and KV name lists the scoped telemetry
// needs, plus the error from fetching each list. Callers own caching.
type TelemetryRefs struct {
	Zones      []cosmoflare.ZoneRef
	ZonesErr   error
	DBNames    map[string]string // D1 database ID → name
	DBNamesErr error
	KVNames    map[string]string // KV namespace ID → title (FEAT-066)
	KVNamesErr error
}

// CollectTelemetryMetrics augments m with zone cache, D1 rows-read and KV
// writes telemetry (FEAT-049, FEAT-066). It is additive (design D15): it
// never returns an error. Every failure that leaves a wanted scope blind
// lands in m.Gaps, so the evaluator pages a telemetry gap instead of the
// rules going silent:
//   - wantZones with a failed zone list, or a failed zone query → Gaps["zone"]
//   - wantD1 with a failed D1 query → Gaps["d1"]
//   - wantKV with a failed KV query → Gaps["kv"]
//
// A failed D1 or KV name list is NOT a gap: rows keep the ID as their name
// (design D8 fallback).
func CollectTelemetryMetrics(ctx context.Context, analytics *cosmoflare.AnalyticsService, wantZones, wantD1, wantKV bool, refs TelemetryRefs, w cosmoflare.AnalyticsWindow, m *EvalMetrics) {
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
		// refs.DBNamesErr is reported by the caller at fetch time (the watch
		// caches and backs off; logging here would repeat every cycle).
		rows, err := analytics.D1RowsRead(ctx, w)
		if err != nil {
			gap("d1", err)
		} else {
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

	if wantKV {
		rows, err := analytics.KVWrites(ctx, w)
		if err != nil {
			gap("kv", err)
		} else {
			for i := range rows {
				if name := refs.KVNames[rows[i].NamespaceID]; name != "" {
					rows[i].Name = name
				} else {
					rows[i].Name = rows[i].NamespaceID
				}
			}
			m.KV = rows
		}
	}
}

// CollectRuleTelemetry is the shared watch/check entry point: it derives
// which scopes the enabled rules need, asks refsFor for just those name
// lists, and collects. With no zone/d1/kv rule enabled nothing is fetched.
func CollectRuleTelemetry(ctx context.Context, analytics *cosmoflare.AnalyticsService, rules []*cosmoflare.AlertRule, refsFor func(wantZones, wantD1, wantKV bool) TelemetryRefs, w cosmoflare.AnalyticsWindow, m *EvalMetrics) {
	wantZones := RulesUseScope(rules, cosmoflare.ScopeZone)
	wantD1 := RulesUseScope(rules, cosmoflare.ScopeD1)
	wantKV := RulesUseScope(rules, cosmoflare.ScopeKV)
	if !wantZones && !wantD1 && !wantKV {
		return
	}
	CollectTelemetryMetrics(ctx, analytics, wantZones, wantD1, wantKV, refsFor(wantZones, wantD1, wantKV), w, m)
}
