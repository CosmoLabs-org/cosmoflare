package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// effMock spins the REST mock (workers list + KV namespaces) and the GraphQL
// mock (workers sums, KV sums, R2 storage) and wires an EfficiencyService.
type effMock struct {
	svc  *EfficiencyService
	rest *httptest.Server
	gql  *httptest.Server
}

func effWrite(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
}

// newEffMock spins the wave-1 mock. The variadic zones opt the service into
// the zone cache-coverage check; coverage queries hit the same shared gql
// server, so gqlBody must carry a viewer.zones array too when zones are set.
func newEffMock(t *testing.T, workersList []any, kvList []any, gqlBody string, zones ...*Zone) *effMock {
	t.Helper()
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/accounts/acct-eff/workers/scripts":
			effWrite(w, workersList)
		case r.URL.Path == "/accounts/acct-eff/storage/kv/namespaces":
			effWrite(w, kvList)
		default:
			t.Errorf("unexpected REST call %s %s", r.Method, r.URL.Path)
			effWrite(w, []any{})
		}
	}))
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, gqlBody)
	}))
	cf, _ := cloudflare.NewWithAPIToken("tok", cloudflare.BaseURL(rest.URL))
	workers, _ := NewWorkerService(cf, "acct-eff")
	kv, _ := NewKVService(cf, "acct-eff")
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	svc := NewEfficiencyService(workers, kv, analytics)
	if len(zones) > 0 {
		svc = NewEfficiencyServiceWithZones(workers, kv, analytics, nil, stubEffZoneLister{zones: zones})
	}
	return &effMock{svc: svc, rest: rest, gql: gql}
}

func effWindow() AnalyticsWindow {
	return AnalyticsWindow{Start: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)}
}

func TestEfficiencyIdleWorkers(t *testing.T) {
	// api-proxy has traffic; ghost-worker does not appear in analytics → idle.
	m := newEffMock(t,
		[]any{map[string]any{"id": "api-proxy"}, map[string]any{"id": "ghost-worker"}},
		[]any{},
		`{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[`+
			`{"sum":{"requests":5000,"errors":0,"subrequests":0},"dimensions":{"scriptName":"api-proxy"}}]}]}}}`,
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.WorkersChecked != 2 {
		t.Fatalf("WorkersChecked = %d, want 2", rep.WorkersChecked)
	}
	var idle []EfficiencyFinding
	for _, f := range rep.Findings {
		if f.Check == "idle-worker" {
			idle = append(idle, f)
		}
	}
	if len(idle) != 1 || idle[0].Resource != "worker:ghost-worker" {
		t.Fatalf("idle findings = %+v", idle)
	}
	if idle[0].Severity != "warning" || idle[0].Evidence == "" || idle[0].Fix == "" {
		t.Fatalf("finding lacks severity/evidence/fix: %+v", idle[0])
	}
}

func TestEfficiencyDeadKVNamespaces(t *testing.T) {
	// ns-live has traffic; ns-dead returns zeros; ns-absent never appears.
	m := newEffMock(t, []any{},
		[]any{map[string]any{"id": "ns-live", "title": "live"}, map[string]any{"id": "ns-dead", "title": "dead"}, map[string]any{"id": "ns-absent", "title": "absent"}},
		`{"data":{"viewer":{"accounts":[{"kvOperationsAdaptiveGroups":[`+
			`{"dimensions":{"namespaceId":"ns-live"},"sum":{"requests":105}},`+
			`{"dimensions":{"namespaceId":"ns-dead"},"sum":{"requests":0}}]}]}}}`,
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.KVNamespacesChecked != 3 {
		t.Fatalf("KVNamespacesChecked = %d, want 3", rep.KVNamespacesChecked)
	}
	dead := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Check == "dead-kv-namespace" {
			dead[f.Resource] = true
		}
	}
	// zero-rows AND absent rows are both dead
	if len(dead) != 2 || !dead["kv:ns-dead"] || !dead["kv:ns-absent"] {
		t.Fatalf("dead findings = %+v", dead)
	}
}

func TestEfficiencyEmptyR2Buckets(t *testing.T) {
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[`+
			`{"max":{"objectCount":42,"payloadSize":1000,"metadataSize":10,"uploadCount":0},"dimensions":{"datetime":"2026-10-14T00:00:00Z","bucketName":"has-things"}},`+
			`{"max":{"objectCount":0,"payloadSize":0,"metadataSize":0,"uploadCount":0},"dimensions":{"datetime":"2026-10-14T00:00:00Z","bucketName":"void"}}]}]}}}`)
	}))
	defer gql.Close()
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	buckets := stubBucketLister{names: []string{"has-things", "void", "never-seen"}}
	svc := NewEfficiencyServiceWithBuckets(nil, nil, analytics, buckets)
	rep, err := svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.BucketsChecked != 3 {
		t.Fatalf("BucketsChecked = %d, want 3", rep.BucketsChecked)
	}
	empty := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Check == "empty-r2-bucket" {
			empty[f.Resource] = true
		}
	}
	if len(empty) != 2 || !empty["r2:void"] || !empty["r2:never-seen"] {
		t.Fatalf("empty findings = %+v", empty)
	}
}

type stubBucketLister struct{ names []string }

func (s stubBucketLister) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	out := make([]*Bucket, 0, len(s.names))
	for _, n := range s.names {
		out = append(out, &Bucket{Name: n})
	}
	return out, nil
}

func TestEfficiencyCleanAccount(t *testing.T) {
	// everything busy → zero findings, no error
	m := newEffMock(t,
		[]any{map[string]any{"id": "busy"}},
		[]any{map[string]any{"id": "ns-live", "title": "live"}},
		`{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[`+
			`{"sum":{"requests":10,"errors":0,"subrequests":0},"dimensions":{"scriptName":"busy"}}],`+
			`"kvOperationsAdaptiveGroups":[{"dimensions":{"namespaceId":"ns-live"},"sum":{"requests":2}}]}],`+
			`"zones":[{"httpRequests1dGroups":[`+
			`{"sum":{"requests":4000,"cachedRequests":400},"dimensions":{"date":"2026-10-10"}}]}]}}}`,
		&Zone{ID: "z1", Name: "busy.example"},
	)
	rep, err := m.svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("want 0 findings, got %+v", rep.Findings)
	}
	if rep.ZonesChecked != 1 || len(rep.ZoneCoverage) != 1 || rep.ZoneCoverage[0].HitRatePct != 10 {
		t.Fatalf("busy zone coverage = checked %d, rows %+v", rep.ZonesChecked, rep.ZoneCoverage)
	}
}

func TestSummarizeWaste(t *testing.T) {
	rep := &EfficiencyReport{
		Findings: []EfficiencyFinding{
			{Resource: "worker:a", Check: "idle-worker", Severity: "warning"},
			{Resource: "worker:b", Check: "idle-worker", Severity: "warning"},
			{Resource: "kv:x", Check: "dead-kv-namespace", Severity: "warning"},
			{Resource: "r2:y", Check: "empty-r2-bucket", Severity: "info"},
		},
	}
	s := SummarizeWaste(rep)
	if s.IdleWorkers != 2 || s.DeadKVNamespaces != 1 || s.EmptyR2Buckets != 1 {
		t.Fatalf("summary = %+v", s)
	}
	if s.EstimatedMonthlyUSD != 0 {
		t.Fatalf("EstimatedMonthlyUSD = %v, want 0 (wave 1 meters only real usage)", s.EstimatedMonthlyUSD)
	}
	if s.Basis == "" {
		t.Fatal("Basis must explain the $0 attribution")
	}
}


// stubEffZoneLister is the efficiency tests' zone lister stub (named apart
// from limits_test.go's fakeZoneLister — one package, shared namespace).
type stubEffZoneLister struct{ zones []*Zone }

func (s stubEffZoneLister) List(ctx context.Context) ([]*Zone, error) { return s.zones, nil }

// TestEfficiencyLowCacheHit: a zone with real traffic but a near-zero
// edge-cache hit rate is a finding; busy-but-cached and quiet (below the
// 1k request floor) zones are not.
func TestEfficiencyLowCacheHit(t *testing.T) {
	t.Parallel()
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body := string(data)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(body, `"zoneTag":"z-low"`):
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[{"httpRequests1dGroups":[`+
				`{"sum":{"requests":2000,"cachedRequests":5},"dimensions":{"date":"2026-10-08"}}]}]}}}`)
		case strings.Contains(body, `"zoneTag":"z-good"`):
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[{"httpRequests1dGroups":[`+
				`{"sum":{"requests":5000,"cachedRequests":500},"dimensions":{"date":"2026-10-08"}}]}]}}}`)
		default:
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[]}}}`)
		}
	}))
	defer gql.Close()
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	zones := stubEffZoneLister{zones: []*Zone{
		{ID: "z-low", Name: "low.example"},
		{ID: "z-good", Name: "good.example"},
		{ID: "z-quiet", Name: "quiet.example"},
	}}
	svc := NewEfficiencyServiceWithZones(nil, nil, analytics, nil, zones)
	rep, err := svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.ZonesChecked != 3 {
		t.Fatalf("ZonesChecked = %d, want 3", rep.ZonesChecked)
	}
	if len(rep.ZoneCoverage) != 3 {
		t.Fatalf("ZoneCoverage rows = %d, want 3", len(rep.ZoneCoverage))
	}
	var findings []EfficiencyFinding
	for _, f := range rep.Findings {
		if f.Check == "low-cache-hit" {
			findings = append(findings, f)
		}
	}
	if len(findings) != 1 || findings[0].Resource != "zone:low.example" {
		t.Fatalf("low-cache-hit findings = %+v, want exactly zone:low.example", findings)
	}
	if findings[0].Severity != "info" || findings[0].Evidence == "" || findings[0].Fix == "" {
		t.Fatalf("finding lacks severity/evidence/fix: %+v", findings[0])
	}
	if !strings.Contains(findings[0].Evidence, "2000 requests") || !strings.Contains(findings[0].Evidence, "0.2%") {
		t.Fatalf("evidence must quote real numbers: %q", findings[0].Evidence)
	}
	// busy-but-cached zone stays on the report with its real hit rate
	var good *ZoneCacheCoverageSummary
	for i := range rep.ZoneCoverage {
		if rep.ZoneCoverage[i].ZoneID == "z-good" {
			good = &rep.ZoneCoverage[i]
		}
	}
	if good == nil || good.HitRatePct != 10 {
		t.Fatalf("good.example coverage row = %+v, want 10%% hit rate", good)
	}
}
