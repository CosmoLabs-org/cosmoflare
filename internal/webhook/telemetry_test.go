package webhook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// telemetryServer answers zone queries with one hit-heavy zone and D1
// queries with d1Body; zoneCalls counts zone requests.
func telemetryServer(t *testing.T, d1Body string, zoneCalls *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(string(body), "d1AnalyticsAdaptiveGroups"):
			w.Write([]byte(d1Body))
		default:
			atomic.AddInt32(zoneCalls, 1)
			w.Write([]byte(`{"data":{"viewer":{"zones":[{"zoneTag":"z1","httpRequestsAdaptiveGroups":[` +
				`{"count":900,"dimensions":{"cacheStatus":"dynamic"}},{"count":100,"dimensions":{"cacheStatus":"hit"}}]}]}}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func telemetryWindow() cosmoflare.AnalyticsWindow {
	end := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	return cosmoflare.AnalyticsWindow{Start: end.Add(-24 * time.Hour), End: end}
}

const d1OK = `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
	`{"sum":{"rowsRead":2945546702,"rowsWritten":7,"readQueries":73151},"dimensions":{"databaseId":"db-big"}}]}]}}}`

func TestCollectTelemetryMetricsHappyPath(t *testing.T) {
	t.Parallel()
	var zoneCalls int32
	srv := telemetryServer(t, d1OK, &zoneCalls)
	a := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	var m EvalMetrics
	CollectTelemetryMetrics(context.Background(), a, true, true, TelemetryRefs{
		Zones:   []cosmoflare.ZoneRef{{ID: "z1", Name: "churches.app"}},
		DBNames: map[string]string{"db-big": "mycarguide-db"},
	}, telemetryWindow(), &m)
	if len(m.Gaps) != 0 {
		t.Fatalf("Gaps = %v, want none", m.Gaps)
	}
	if len(m.Zones) != 1 || m.Zones[0].Zone != "churches.app" || m.Zones[0].ByStatus["dynamic"] != 900 {
		t.Errorf("Zones = %+v", m.Zones)
	}
	if len(m.D1) != 1 || m.D1[0].Name != "mycarguide-db" {
		t.Errorf("D1 = %+v, want named mycarguide-db", m.D1)
	}
}

// TestCollectTelemetryRefsFailureIsAGap: a failed zone list is a gap, never a
// silent skip; a failed D1 name list keeps D1 with database IDs (D8 fallback).
func TestCollectTelemetryRefsFailureIsAGap(t *testing.T) {
	t.Parallel()
	var zoneCalls int32
	srv := telemetryServer(t, d1OK, &zoneCalls)
	a := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	var m EvalMetrics
	CollectTelemetryMetrics(context.Background(), a, true, true, TelemetryRefs{
		ZonesErr:   errors.New("zones list: 403 missing Zone Read"),
		DBNamesErr: errors.New("d1 list: 403"),
	}, telemetryWindow(), &m)
	if !strings.Contains(m.Gaps["zone"], "403 missing Zone Read") {
		t.Errorf("Gaps[zone] = %q, want the zone list error", m.Gaps["zone"])
	}
	if atomic.LoadInt32(&zoneCalls) != 0 {
		t.Errorf("zone analytics requested %d times without refs, want 0", zoneCalls)
	}
	if _, gap := m.Gaps["d1"]; gap {
		t.Errorf("D1 name-list failure must not be a d1 gap: %v", m.Gaps)
	}
	if len(m.D1) != 1 || m.D1[0].Name != "db-big" {
		t.Errorf("D1 = %+v, want the row named by its database ID", m.D1)
	}
}

func TestCollectTelemetryAnalyticsFailureIsAGap(t *testing.T) {
	t.Parallel()
	var zoneCalls int32
	srv := telemetryServer(t, `{"errors":[{"message":"d1 dataset unavailable"}]}`, &zoneCalls)
	a := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	var m EvalMetrics
	CollectTelemetryMetrics(context.Background(), a, true, true, TelemetryRefs{
		Zones: []cosmoflare.ZoneRef{{ID: "z1", Name: "churches.app"}},
	}, telemetryWindow(), &m)
	if !strings.Contains(m.Gaps["d1"], "d1 dataset unavailable") {
		t.Errorf("Gaps[d1] = %q, want the analytics error", m.Gaps["d1"])
	}
	if len(m.Zones) != 1 {
		t.Errorf("zone collection must survive a D1 failure: %+v", m.Zones)
	}
}

func TestCollectTelemetryWantFlagsOff(t *testing.T) {
	t.Parallel()
	var zoneCalls int32
	srv := telemetryServer(t, d1OK, &zoneCalls)
	a := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	var m EvalMetrics
	CollectTelemetryMetrics(context.Background(), a, false, false, TelemetryRefs{
		Zones:    []cosmoflare.ZoneRef{{ID: "z1", Name: "churches.app"}},
		ZonesErr: errors.New("ignored when not wanted"),
	}, telemetryWindow(), &m)
	if len(m.Gaps) != 0 || len(m.Zones) != 0 || len(m.D1) != 0 || atomic.LoadInt32(&zoneCalls) != 0 {
		t.Errorf("nothing wanted → nothing collected, no gaps: %+v", m)
	}
}
