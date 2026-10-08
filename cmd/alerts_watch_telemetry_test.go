package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/alertspush"
)

// resetWatchState gives a test fresh watch cooldown memory and refs cache,
// restoring the package seams afterwards (watchFireState and the refs cache
// are package-level and would otherwise leak pages between tests).
func resetWatchState(t *testing.T) {
	t.Helper()
	origState, origRefsFn, origNow := watchFireState, telemetryRefsFn, telemetryNowFn
	watchFireState = webhook.NewFireState()
	telemetryRefsCache, telemetryZonesAttempt, telemetryDBAttempt = webhook.TelemetryRefs{}, time.Time{}, time.Time{}
	t.Cleanup(func() {
		watchFireState, telemetryRefsFn, telemetryNowFn = origState, origRefsFn, origNow
		telemetryRefsCache, telemetryZonesAttempt, telemetryDBAttempt = webhook.TelemetryRefs{}, time.Time{}, time.Time{}
	})
}

const watchZoneAndScriptRulesYAML = "rules:\n" +
	"  - name: uncached\n" +
	"    service: zone\n" +
	"    condition: zone-uncached-pct\n" +
	"    threshold: 60\n" +
	"    action: log\n" +
	"    target: /dev/null\n" +
	"    enabled: true\n" +
	"  - name: hot-cpu\n" +
	"    service: workers\n" +
	"    condition: worker-cpu\n" +
	"    threshold: 500\n" +
	"    action: log\n" +
	"    target: /dev/null\n" +
	"    enabled: true\n"

func churchesZoneRow() cosmoflare.ZoneCacheSummary {
	return cosmoflare.ZoneCacheSummary{ZoneID: "z1", Zone: "churches.app", ByStatus: map[string]uint64{
		"miss": 83, "bypass": 21, "dynamic": 939, "revalidated": 236, "expired": 4, "none": 197, "hit": 61,
	}}
}

func oneDeviceStore() *alertspush.Store {
	return &alertspush.Store{Subscriptions: []alertspush.Subscription{
		{Endpoint: "https://push.example/e1", P256dh: "k", Auth: "a"},
	}}
}

// TestWatchZoneCooldownAcrossCycles pins O3: across two watch cycles the
// zone rule pages once (1h cooldown kept in memory) while the script rule
// keeps re-paging every cycle.
func TestWatchZoneCooldownAcrossCycles(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	resetWatchState(t)
	watchWriteRule(t, rulesPath, watchZoneAndScriptRulesYAML)
	collectWatchMetricsFn = func(_ context.Context, _ []*cosmoflare.AlertRule) (webhook.EvalMetrics, error) {
		return webhook.EvalMetrics{
			Zones:   []cosmoflare.ZoneCacheSummary{churchesZoneRow()},
			Scripts: []cosmoflare.WorkersSummary{{Script: "api-proxy", Requests: 100, CPUP99: 812}},
		}, nil
	}
	sender := &watchFakeSender{}
	first, _, err := evaluateOnce(context.Background(), svc, oneDeviceStore(), sender)
	if err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	second, _, err := evaluateOnce(context.Background(), svc, oneDeviceStore(), sender)
	if err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if first != 2 || second != 1 {
		t.Fatalf("pushes per cycle = %d, %d; want 2 (zone + script) then 1 (script only)", first, second)
	}
}

// TestWatchTelemetryGapPages: a zone gap with a zone rule enabled pushes one
// telemetry-gap page.
func TestWatchTelemetryGapPages(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	resetWatchState(t)
	watchWriteRule(t, rulesPath, watchZoneAndScriptRulesYAML)
	collectWatchMetricsFn = func(_ context.Context, _ []*cosmoflare.AlertRule) (webhook.EvalMetrics, error) {
		return webhook.EvalMetrics{Gaps: map[string]string{"zone": "zones list: 403"}}, nil
	}
	sender := &watchFakeSender{}
	sent, _, err := evaluateOnce(context.Background(), svc, oneDeviceStore(), sender)
	if err != nil || sent != 1 {
		t.Fatalf("sent=%d err=%v, want one gap page", sent, err)
	}
	var p alertspush.Payload
	if err := json.Unmarshal(sender.lastEnvelope(t), &p); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if p.ID != "telemetry-gap/zone" || !strings.Contains(p.Detail, "403") {
		t.Errorf("gap payload = %+v", p)
	}
}

// TestWatchRulesGetPassedToCollector: the collector receives the rule list
// so it can decide which telemetry to fetch.
func TestWatchRulesGetPassedToCollector(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	resetWatchState(t)
	watchWriteRule(t, rulesPath, watchZoneAndScriptRulesYAML)
	var got []*cosmoflare.AlertRule
	collectWatchMetricsFn = func(_ context.Context, rules []*cosmoflare.AlertRule) (webhook.EvalMetrics, error) {
		got = rules
		return webhook.EvalMetrics{}, nil
	}
	if _, _, err := evaluateOnce(context.Background(), svc, &alertspush.Store{}, &watchFakeSender{}); err != nil {
		t.Fatalf("evaluateOnce: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("collector got %d rules, want 2", len(got))
	}
}

func TestCachedTelemetryRefs(t *testing.T) {
	resetWatchState(t)
	now := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	telemetryNowFn = func() time.Time { return now }
	calls := 0
	failNext := false
	telemetryRefsFn = func(_ context.Context, wantZones, wantD1 bool) webhook.TelemetryRefs {
		calls++
		if failNext {
			return webhook.TelemetryRefs{ZonesErr: errors.New("zones list: 503"), DBNamesErr: errors.New("d1 list: 503")}
		}
		return webhook.TelemetryRefs{
			Zones:   []cosmoflare.ZoneRef{{ID: "z1", Name: "churches.app"}},
			DBNames: map[string]string{"db1": "mycarguide-db"},
		}
	}

	refs := cachedTelemetryRefs(context.Background(), true, true)
	if calls != 1 || len(refs.Zones) != 1 {
		t.Fatalf("first fetch: calls=%d refs=%+v", calls, refs)
	}
	cachedTelemetryRefs(context.Background(), true, true)
	if calls != 1 {
		t.Errorf("within TTL: calls=%d, want 1 (cached)", calls)
	}

	// Past the TTL a failing fetch keeps the last good lists (no gap).
	now = now.Add(16 * time.Minute)
	failNext = true
	refs = cachedTelemetryRefs(context.Background(), true, true)
	if calls != 2 || refs.ZonesErr != nil || len(refs.Zones) != 1 || refs.DBNames["db1"] != "mycarguide-db" {
		t.Fatalf("keep-last-good: calls=%d refs=%+v", calls, refs)
	}

	// Wanting a scope the cache never fetched forces a fetch even within TTL.
	resetWatchState(t)
	telemetryNowFn = func() time.Time { return now }
	calls = 0
	failNext = true
	telemetryRefsFn = func(_ context.Context, wantZones, wantD1 bool) webhook.TelemetryRefs {
		calls++
		return webhook.TelemetryRefs{ZonesErr: errors.New("zones list: 403")}
	}
	refs = cachedTelemetryRefs(context.Background(), true, false)
	if refs.ZonesErr == nil {
		t.Fatalf("no last-good zones: the error must surface (becomes a gap), got %+v", refs)
	}
}

func TestRulesNeedScope(t *testing.T) {
	rules := []*cosmoflare.AlertRule{
		{Name: "a", Condition: "zone-uncached-pct", Enabled: false},
		{Name: "b", Condition: "worker-cpu", Enabled: true},
		{Name: "c", Condition: "d1-rows-read", Enabled: true},
	}
	if webhook.RulesUseScope(rules, "zone") {
		t.Error("disabled zone rule must not count")
	}
	if !webhook.RulesUseScope(rules, "d1") {
		t.Error("enabled d1 rule must count")
	}
}

func TestAlertsWatchHelpMentionsTelemetry(t *testing.T) {
	for _, want := range []string{"zone cache", "D1 rows read", "once an hour", "telemetry gap"} {
		if !strings.Contains(alertsWatchCmd.Long, want) {
			t.Errorf("alerts watch help missing %q", want)
		}
	}
}

func TestServeZoneD1Warning(t *testing.T) {
	if msg := serveZoneD1Warning([]*cosmoflare.AlertRule{{Name: "b", Condition: "worker-cpu", Enabled: true}}); msg != "" {
		t.Errorf("no zone/d1 rule → no warning, got %q", msg)
	}
	msg := serveZoneD1Warning([]*cosmoflare.AlertRule{{Name: "scan", Condition: "d1-rows-read", Enabled: true}})
	if !strings.Contains(msg, "alerts watch") || !strings.Contains(msg, "scan") {
		t.Errorf("warning must name the rule and point at alerts watch: %q", msg)
	}
}

// TestCachedTelemetryRefsBackoff pins the review fix: a list that keeps
// failing (e.g. a token without D1:Read) is retried every 5 min, not every
// cycle, and a refresh fetches only the lists that are due.
func TestCachedTelemetryRefsBackoff(t *testing.T) {
	resetWatchState(t)
	now := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	telemetryNowFn = func() time.Time { return now }
	var calls [][2]bool
	telemetryRefsFn = func(_ context.Context, wantZones, wantD1 bool) webhook.TelemetryRefs {
		calls = append(calls, [2]bool{wantZones, wantD1})
		refs := webhook.TelemetryRefs{DBNamesErr: errors.New("d1 list: 403")}
		if wantZones {
			refs.Zones = []cosmoflare.ZoneRef{{ID: "z1", Name: "churches.app"}}
		}
		return refs
	}

	refs := cachedTelemetryRefs(context.Background(), true, true)
	if len(calls) != 1 || refs.DBNamesErr == nil || len(refs.Zones) != 1 {
		t.Fatalf("first: calls=%v refs=%+v", calls, refs)
	}
	now = now.Add(time.Minute)
	refs = cachedTelemetryRefs(context.Background(), true, true)
	if len(calls) != 1 {
		t.Fatalf("1 min later: calls=%v, want no refetch (failed list backs off 5 min)", calls)
	}
	if refs.DBNamesErr == nil {
		t.Error("the cached D1 list error must still be reported while backing off")
	}
	now = now.Add(5 * time.Minute)
	cachedTelemetryRefs(context.Background(), true, true)
	if len(calls) != 2 || calls[1] != [2]bool{false, true} {
		t.Fatalf("6 min later: calls=%v, want one retry of the D1 list only (zones still fresh)", calls)
	}
}
