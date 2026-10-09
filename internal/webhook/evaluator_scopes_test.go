package webhook

import (
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// churchesZone is the live churches.app cacheStatus breakdown captured
// 2026-10-08 (24h, eyeball): uncached 960 of 1344 known-status = 71.4%.
func churchesZone() cosmoflare.ZoneCacheSummary {
	return cosmoflare.ZoneCacheSummary{ZoneID: "z1", Zone: "churches.app", ByStatus: map[string]uint64{
		"miss": 83, "bypass": 21, "dynamic": 939, "revalidated": 236, "expired": 4, "none": 197, "hit": 61,
	}}
}

// healthyZone is fully cached: uncached 0%, miss ~9%.
func healthyZone() cosmoflare.ZoneCacheSummary {
	return cosmoflare.ZoneCacheSummary{ZoneID: "z2", Zone: "cdn.example", ByStatus: map[string]uint64{
		"hit": 1000, "miss": 100,
	}}
}

func scopedRule(name, service, condition string, threshold float64, exclude ...string) *cosmoflare.AlertRule {
	r := evalRule(name, condition, threshold)
	r.Service = service
	r.Exclude = exclude
	return r
}

func recordedIDs(rec *alertRecorder) []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	ids := make([]string, 0, len(rec.payloads))
	for _, p := range rec.payloads {
		ids = append(ids, p.Alert.ID)
	}
	return ids
}

func TestEvaluateZoneUncachedNamesZone(t *testing.T) {
	t.Parallel()
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("uncached", "zone", "zone-uncached-requests", 900))
	eval.Evaluate(EvalMetrics{Zones: []cosmoflare.ZoneCacheSummary{churchesZone(), healthyZone()}})
	ids := recordedIDs(rec)
	if len(ids) != 1 || ids[0] != "uncached/z1" {
		t.Fatalf("alert IDs = %v, want [uncached/z1] (keyed on the stable zone ID)", ids)
	}
	if p := rec.first(); !strings.Contains(p.Message, "churches.app") || !strings.Contains(p.Message, "960") {
		t.Errorf("message must name the zone and its uncached request count: %q", p.Message)
	}
}

func TestEvaluateZoneCacheMissPct(t *testing.T) {
	t.Parallel()
	// churches.app miss% = 87/384 = 22.7; cdn.example = 100/1100 = 9.1
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("misses", "zone", "zone-cache-miss-pct", 20))
	eval.Evaluate(EvalMetrics{Zones: []cosmoflare.ZoneCacheSummary{churchesZone(), healthyZone()}})
	if ids := recordedIDs(rec); len(ids) != 1 || ids[0] != "misses/z1" {
		t.Fatalf("alert IDs = %v, want [misses/z1]", ids)
	}
}

func TestEvaluateZoneFloorSkips(t *testing.T) {
	t.Parallel()
	tiny := cosmoflare.ZoneCacheSummary{ZoneID: "z3", Zone: "tiny.example", ByStatus: map[string]uint64{"dynamic": 40, "hit": 10}}
	eval, rec := newEvalEvaluator(t, time.Minute,
		scopedRule("misses", "zone", "zone-cache-miss-pct", 0.01),
	)
	eval.Evaluate(EvalMetrics{Zones: []cosmoflare.ZoneCacheSummary{tiny}})
	if ids := recordedIDs(rec); len(ids) != 0 {
		t.Fatalf("alert IDs = %v, want none (50 requests is under the %d floor)", ids, ZoneCacheFloor)
	}
}

func TestEvaluateExcludeSkipsScope(t *testing.T) {
	t.Parallel()
	other := churchesZone()
	other.ZoneID, other.Zone = "z9", "api.example"
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("uncached", "zone", "zone-uncached-requests", 900, "Churches.App"))
	eval.Evaluate(EvalMetrics{Zones: []cosmoflare.ZoneCacheSummary{churchesZone(), other}})
	if ids := recordedIDs(rec); len(ids) != 1 || ids[0] != "uncached/z9" {
		t.Fatalf("alert IDs = %v, want [uncached/z9] (churches.app excluded)", ids)
	}
}

func TestEvaluateD1HumanReadable(t *testing.T) {
	t.Parallel()
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("scan", "d1", "d1-rows-read", 1e9))
	eval.Evaluate(EvalMetrics{D1: []cosmoflare.D1RowsReadSummary{
		{DatabaseID: "db1", Name: "mycarguide-db", RowsRead: 2945546702},
		{DatabaseID: "db2", Name: "churches-db", RowsRead: 52332502},
	}})
	if ids := recordedIDs(rec); len(ids) != 1 || ids[0] != "scan/db1" {
		t.Fatalf("alert IDs = %v, want [scan/db1]", ids)
	}
	p := rec.first()
	if !strings.Contains(p.Message, "2.9B") || strings.Contains(p.Message, "e+09") {
		t.Errorf("d1 message must print 2.9B, not %%g: %q", p.Message)
	}
}

func TestHumanCount(t *testing.T) {
	t.Parallel()
	for v, want := range map[float64]string{
		2945546702: "2.9B", 52332502: "52.3M", 3289: "3.3k", 522: "522", 0: "0",
	} {
		if got := humanCount(v); got != want {
			t.Errorf("humanCount(%v) = %q, want %q", v, got, want)
		}
	}
}

// TestEvaluateScopeCooldownPersists: the watch builds a fresh evaluator per
// cycle; a shared FireState plus the watch policy (default 0, zone/d1 1h)
// keeps zone pages hourly while script rules still page every cycle.
func TestEvaluateScopeCooldownPersists(t *testing.T) {
	t.Parallel()
	svc, _ := newEvalService(t,
		scopedRule("uncached", "zone", "zone-uncached-requests", 900),
		evalRule("hot-cpu", "worker-cpu", 500),
	)
	rec := &alertRecorder{}
	st := NewFireState()
	m := EvalMetrics{
		Zones:   []cosmoflare.ZoneCacheSummary{churchesZone()},
		Scripts: []cosmoflare.WorkersSummary{{Script: "api-proxy", Requests: 100, CPUP99: 812}},
	}
	base := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		eval := NewEvaluator(svc, newEvalManager(rec), 0)
		eval.UseState(st)
		eval.SetCooldownPolicy(0, map[string]time.Duration{"zone": time.Hour, "d1": time.Hour})
		now := base.Add(time.Duration(i) * time.Minute)
		eval.SetClock(func() time.Time { return now })
		eval.Evaluate(m)
	}
	var zone, script int
	for _, id := range recordedIDs(rec) {
		switch id {
		case "uncached/z1":
			zone++
		case "hot-cpu/api-proxy":
			script++
		}
	}
	if zone != 1 || script != 2 {
		t.Fatalf("zone pages = %d (want 1), script pages = %d (want 2): %v", zone, script, recordedIDs(rec))
	}

	// After the hour the zone pages again.
	eval := NewEvaluator(svc, newEvalManager(rec), 0)
	eval.UseState(st)
	eval.SetCooldownPolicy(0, map[string]time.Duration{"zone": time.Hour})
	later := base.Add(61 * time.Minute)
	eval.SetClock(func() time.Time { return later })
	eval.Evaluate(m)
	zone = 0
	for _, id := range recordedIDs(rec) {
		if id == "uncached/z1" {
			zone++
		}
	}
	if zone != 2 {
		t.Errorf("zone pages after 61 min = %d, want 2", zone)
	}
}

func TestEvaluateEscalationDoubling(t *testing.T) {
	t.Parallel()
	svc, _ := newEvalService(t, scopedRule("scan", "d1", "d1-rows-read", 1e9))
	rec := &alertRecorder{}
	st := NewFireState()
	base := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	for i, rows := range []uint64{1_000_000_000, 1_500_000_000, 2_100_000_000} {
		eval := NewEvaluator(svc, newEvalManager(rec), 0)
		eval.UseState(st)
		eval.SetCooldownPolicy(0, map[string]time.Duration{"d1": time.Hour})
		now := base.Add(time.Duration(i) * 10 * time.Minute)
		eval.SetClock(func() time.Time { return now })
		eval.Evaluate(EvalMetrics{D1: []cosmoflare.D1RowsReadSummary{{DatabaseID: "db1", Name: "mycarguide-db", RowsRead: rows}}})
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.payloads) != 2 {
		t.Fatalf("pages = %d, want 2 (first breach + escalation at ≥2×)", len(rec.payloads))
	}
	if !strings.HasPrefix(rec.payloads[1].Message, "escalated") {
		t.Errorf("second page must say escalated: %q", rec.payloads[1].Message)
	}
}

// TestEvaluateNoPolicyNoEscalation: serve's path (no policy) keeps its
// 15-min cooldown exactly — a doubled value inside it does not re-page.
func TestEvaluateNoPolicyNoEscalation(t *testing.T) {
	t.Parallel()
	eval, rec := newEvalEvaluator(t, 15*time.Minute, evalRule("hot-cpu", "worker-cpu", 500))
	base := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	now := base
	eval.SetClock(func() time.Time { return now })
	eval.Evaluate(EvalMetrics{Scripts: []cosmoflare.WorkersSummary{{Script: "a", Requests: 1, CPUP99: 600}}})
	now = base.Add(time.Minute)
	eval.Evaluate(EvalMetrics{Scripts: []cosmoflare.WorkersSummary{{Script: "a", Requests: 1, CPUP99: 1500}}})
	if ids := recordedIDs(rec); len(ids) != 1 {
		t.Fatalf("pages = %v, want 1 (no escalation without a scope policy)", ids)
	}
}

func TestEvaluateTelemetryGapPagesOnce(t *testing.T) {
	t.Parallel()
	svc, _ := newEvalService(t, scopedRule("scan", "d1", "d1-rows-read", 1e9))
	rec := &alertRecorder{}
	st := NewFireState()
	base := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		eval := NewEvaluator(svc, newEvalManager(rec), 0)
		eval.UseState(st)
		eval.SetCooldownPolicy(0, map[string]time.Duration{"d1": time.Hour})
		now := base.Add(time.Duration(i) * time.Minute)
		eval.SetClock(func() time.Time { return now })
		eval.Evaluate(EvalMetrics{Gaps: map[string]string{"d1": "boom", "zone": "no zone rule enabled"}})
	}
	ids := recordedIDs(rec)
	if len(ids) != 1 || ids[0] != "telemetry-gap/d1" {
		t.Fatalf("pages = %v, want exactly [telemetry-gap/d1] (zone gap has no zone rule)", ids)
	}
	if p := rec.first(); p.Alert.Name != "telemetry-gap" || !strings.Contains(p.Message, "boom") {
		t.Errorf("gap page = name %q message %q", p.Alert.Name, p.Message)
	}
}

// TestUsageMaxSkipsD1 pins O16: D1 shows in the usage view but never drives
// usage-pct / usage-projected-pct (those re-page every cycle).
func TestUsageMaxSkipsD1(t *testing.T) {
	t.Parallel()
	dims := []cosmoflare.UsageDimension{
		{ID: "workers.requests_monthly", Limit: 100, Pct: 40, ProjectedPct: 60},
		{ID: "d1.rows_read_monthly", Limit: 25e9, Pct: 350, ProjectedPct: 900},
	}
	if got := usageMax(dims, false); got != 40 {
		t.Errorf("usageMax = %v, want 40 (d1 excluded)", got)
	}
	if got := usageMax(dims, true); got != 60 {
		t.Errorf("usageMax projected = %v, want 60 (d1 excluded)", got)
	}
}

// TestEvaluateD1KeyStableAcrossNameChange: when the D1 name list is down the
// row is named by its ID; when it recovers the name appears. The alert key is
// the database ID either way, so the hourly cooldown holds (review finding).
func TestEvaluateD1KeyStableAcrossNameChange(t *testing.T) {
	t.Parallel()
	svc, _ := newEvalService(t, scopedRule("scan", "d1", "d1-rows-read", 1e9))
	rec := &alertRecorder{}
	st := NewFireState()
	base := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	for i, name := range []string{"db1", "mycarguide-db"} {
		eval := NewEvaluator(svc, newEvalManager(rec), 0)
		eval.UseState(st)
		eval.SetCooldownPolicy(0, map[string]time.Duration{"d1": time.Hour})
		now := base.Add(time.Duration(i) * time.Minute)
		eval.SetClock(func() time.Time { return now })
		eval.Evaluate(EvalMetrics{D1: []cosmoflare.D1RowsReadSummary{{DatabaseID: "db1", Name: name, RowsRead: 2e9}}})
	}
	if ids := recordedIDs(rec); len(ids) != 1 || ids[0] != "scan/db1" {
		t.Fatalf("pages = %v, want exactly [scan/db1] (name change must not reset the cooldown)", ids)
	}
}

// TestEvaluateExcludeMatchesNameOrID: an exclude entry matches the display
// name or the stable ID, so excludes still work when names are unknown.
func TestEvaluateExcludeMatchesNameOrID(t *testing.T) {
	t.Parallel()
	rows := []cosmoflare.D1RowsReadSummary{
		{DatabaseID: "db1", Name: "db1", RowsRead: 2e9},           // name list down: ID fallback
		{DatabaseID: "db2", Name: "churches-db", RowsRead: 2e9},   // excluded by name
		{DatabaseID: "db3", Name: "analytics-db", RowsRead: 2e9}, // excluded by ID
		{DatabaseID: "db4", Name: "noble-db", RowsRead: 2e9},
	}
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("scan", "d1", "d1-rows-read", 1e9, "db1", "churches-db", "db3"))
	eval.Evaluate(EvalMetrics{D1: rows})
	if ids := recordedIDs(rec); len(ids) != 1 || ids[0] != "scan/db4" {
		t.Fatalf("pages = %v, want only [scan/db4]", ids)
	}
	if p := rec.first(); !strings.Contains(p.Message, "noble-db") {
		t.Errorf("message must show the display name: %q", p.Message)
	}
}

// TestPercentOneDecimal: % values print one decimal in page text (operator
// choice 2026-10-09); the raw value keeps full precision.
func TestPercentOneDecimal(t *testing.T) {
	t.Parallel()
	eval, rec := newEvalEvaluator(t, time.Minute, scopedRule("misses", "zone", "zone-cache-miss-pct", 20))
	eval.Evaluate(EvalMetrics{Zones: []cosmoflare.ZoneCacheSummary{churchesZone()}})
	p := rec.first()
	if p == nil {
		t.Fatal("no page")
	}
	if !strings.Contains(p.Message, "observed 22.7 meets threshold 20.0 (%)") {
		t.Errorf("message = %q, want 'observed 22.7 meets threshold 20.0 (%%)'", p.Message)
	}
	if p.Value < 22.65 || p.Value > 22.66 {
		t.Errorf("payload value = %v, want full precision 22.65625", p.Value)
	}
}
