package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func TestLimitFor(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		plan     string
		want     uint64
		wantOK   bool
	}{
		{"workers scripts free", "workers.scripts", "free", 100, true},
		{"workers scripts paid", "workers.scripts", "paid", 500, true},
		{"workers daily requests free", "workers.daily_requests", "free", 100000, true},
		{"workers daily requests paid is unlimited", "workers.daily_requests", "paid", 0, true},
		{"r2 buckets plan-independent", "r2.buckets", "", 1000000, true},
		{"r2 custom domains plan-independent", "r2.custom_domains_per_bucket", "paid", 100, true},
		{"workers resource with unknown plan", "workers.scripts", "unknown", 0, false},
		{"unknown resource", "nope", "free", 0, false},
		{"workers resource with empty plan", "workers.scripts", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := limitFor(tt.resource, tt.plan)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("limitFor(%q, %q) = (%d, %v), want (%d, %v)",
					tt.resource, tt.plan, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDNSRecordsStaticLimit(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		created string // ISO date; empty = zero time
		want    uint64
		wantOK  bool
	}{
		{"pro", "pro", "", 3500, true},
		{"business", "business", "", 3500, true},
		{"free before cutoff", "free", "2024-01-15", 1000, true},
		{"free on cutoff", "free", "2024-09-01", 200, true},
		{"free after cutoff", "free", "2025-06-01", 200, true},
		{"enterprise has no per-zone limit", "enterprise", "", 0, false},
		{"unknown plan", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := time.Time{}
			if tt.created != "" {
				parsed, err := time.Parse("2006-01-02", tt.created)
				if err != nil {
					t.Fatalf("bad fixture date %q: %v", tt.created, err)
				}
				created = parsed
			}
			got, ok := dnsRecordsStaticLimit(tt.plan, created)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("dnsRecordsStaticLimit(%q, %v) = (%d, %v), want (%d, %v)",
					tt.plan, created, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNewLimitsServiceDefaults(t *testing.T) {
	s := NewLimitsService("acct", "tok")
	if s.accountID != "acct" || s.rest.apiToken != "tok" {
		t.Fatal("credentials not stored")
	}
	if s.rest.httpClient == nil {
		t.Fatal("default HTTP client missing")
	}
	if s.rest.baseURL != "https://api.cloudflare.com/client/v4" {
		t.Fatalf("baseURL = %q", s.rest.baseURL)
	}
}

func TestNewLimitsServiceValidation(t *testing.T) {
	if _, err := NewLimitsService("", "tok").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty account ID must fail validation")
	} else if !strings.Contains(err.Error(), "account ID is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewLimitsService("acct", "").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty token must fail validation")
	} else if !strings.Contains(err.Error(), "API token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testLimitServer(t *testing.T, handler http.HandlerFunc) (*LimitsService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewLimitsService("acct", "tok", WithLimitsBaseURL(srv.URL)), srv
}

func TestResolveWorkersPlanAuto(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct/subscriptions" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "cdn"}, "rate_plan": {"id": "cdn_pro"}},
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_paid", "public_name": "Workers Paid"}}
		]}`)
	})
	plan, source := s.resolveWorkersPlan(context.Background())
	if plan != "paid" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (paid, auto)", plan, source)
	}
}

func TestResolveWorkersPlanAutoFree(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": [
			{"product": {"name": "workers"}, "rate_plan": {"id": "workers_free"}}
		]}`)
	})
	plan, source := s.resolveWorkersPlan(context.Background())
	if plan != "free" || source != "auto" {
		t.Fatalf("resolveWorkersPlan = (%q, %q), want (free, auto)", plan, source)
	}
}

func TestResolveWorkersPlanFallbackChain(t *testing.T) {
	// Subscriptions endpoint 403s (scoped token) → config → flag ordering.
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"success": false}`)
	})

	plan, source := s.resolveWorkersPlan(context.Background())
	if plan != "unknown" || source != "unknown" {
		t.Fatalf("no fallbacks: got (%q, %q), want (unknown, unknown)", plan, source)
	}

	s.configPlan = "paid"
	plan, source = s.resolveWorkersPlan(context.Background())
	if plan != "paid" || source != "config" {
		t.Fatalf("config fallback: got (%q, %q), want (paid, config)", plan, source)
	}

	s.flagPlan = "free"
	plan, source = s.resolveWorkersPlan(context.Background())
	if plan != "free" || source != "flag" {
		t.Fatalf("flag outranks config: got (%q, %q), want (free, flag)", plan, source)
	}
}

func TestResolveWorkersPlanInvalidValues(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	s.configPlan = "enterprise" // not a Workers tier — must be rejected, not trusted
	plan, source := s.resolveWorkersPlan(context.Background())
	if plan != "unknown" || source != "unknown" {
		t.Fatalf("invalid config plan: got (%q, %q), want (unknown, unknown)", plan, source)
	}
}

func TestDNSUsageLive(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/z1/dns_records/usage" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"success": true, "result": {"record_usage": 180, "record_quota": 200}}`)
	})
	used, limit, source, err := s.dnsUsage(context.Background(), "z1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 180 || limit != 200 || source != "live-api" {
		t.Fatalf("dnsUsage = (%d, %d, %q), want (180, 200, live-api)", used, limit, source)
	}
}

// TestDNSUsageNullQuota pins the documented null-record_quota case: an
// account-level quota applies, so no per-zone limit is reportable and the
// caller must fall back rather than treat null as zero.
func TestDNSUsageNullQuota(t *testing.T) {
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": {"record_usage": 5, "record_quota": null}}`)
	})
	_, _, _, err := s.dnsUsage(context.Background(), "z1")
	if err == nil {
		t.Fatal("null record_quota must yield an error so the caller falls back")
	}
}

// TestDNSUsageFallbackToStatic exercises the fallback path end-to-end: the
// live endpoint 404s and the Snapshot-level static table answers. (The pure
// table itself is covered by TestDNSRecordsStaticLimit.)
func TestDNSUsageFallbackToStatic(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsZones(fakeZoneLister{zones: []*Zone{
			{ID: "z1", Name: "fallback.example", Plan: ZonePlan{LegacyID: "free"}, CreatedOn: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		}}),
	)
	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range snap.Rows {
		if row.Resource == "dns.records" && row.Scope == "fallback.example" {
			if row.Limit != 200 || row.LimitSource != "static-docs" || row.Used != 0 {
				t.Fatalf("fallback row = %+v, want limit=200 static-docs used=0", row)
			}
			return
		}
	}
	t.Fatal("dns.records fallback row missing")
}

// TestCFZoneToZoneLegacyID pins the real mapping: cloudflare-go's
// ZonePlan.LegacyID flows through cfZoneToZone into our ZonePlan.
func TestCFZoneToZoneLegacyID(t *testing.T) {
	in := cloudflare.Zone{ID: "z1", Name: "example.com"}
	in.Plan.LegacyID = "pro"
	in.Plan.Name = "Pro"
	in.Plan.ID = "plan-id"
	got := cfZoneToZone(in)
	if got.Plan.LegacyID != "pro" || got.Plan.Name != "Pro" || got.Plan.ID != "plan-id" {
		t.Fatalf("cfZoneToZone plan mapping = %+v", got.Plan)
	}
}

type fakeWorkersLister struct {
	n   int
	err error
}

func (f fakeWorkersLister) List(ctx context.Context) ([]*Worker, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*Worker, f.n), nil
}

type fakeBucketLister struct {
	n   int
	err error
}

func (f fakeBucketLister) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*Bucket, f.n), nil
}

type fakeZoneLister struct {
	zones []*Zone
	err   error
}

func (f fakeZoneLister) List(ctx context.Context) ([]*Zone, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.zones, nil
}

type fakeDomainLister struct {
	n   int
	err error
}

func (f fakeDomainLister) List(ctx context.Context, bucket string) ([]BucketDomain, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]BucketDomain, f.n), nil
}

type fakeWorkersAnalytics struct {
	sum []WorkersSummary
	err error
}

func (f fakeWorkersAnalytics) Workers(ctx context.Context, w AnalyticsWindow) ([]WorkersSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sum, nil
}

func snapshotFixture(t *testing.T, opts ...LimitsOption) *LimitsService {
	t.Helper()
	// Subscriptions: 403 → unknown plan path (tests must not depend on it).
	s, _ := testLimitServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	s.configPlan = "paid"
	for _, o := range opts {
		o(s)
	}
	return s
}

func TestSnapshotHappyPath(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{n: 80}),
		WithLimitsR2(fakeBucketLister{n: 7}),
		WithLimitsZones(fakeZoneLister{zones: []*Zone{
			{ID: "z1", Name: "one.example", Plan: ZonePlan{LegacyID: "pro"}, CreatedOn: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}}),
		WithLimitsAnalytics(fakeWorkersAnalytics{sum: []WorkersSummary{{Requests: 42000}}}),
	)
	// DNS usage live endpoint serves 180/200 (pinned shape).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success": true, "result": {"record_usage": 180, "record_quota": 200}}`)
	}))
	t.Cleanup(srv.Close)
	s.rest.baseURL = srv.URL

	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.WorkersPlan != "paid" || snap.PlanSource != "config" {
		t.Fatalf("plan = (%q, %q), want (paid, config)", snap.WorkersPlan, snap.PlanSource)
	}

	byResource := map[string]LimitRow{}
	for _, row := range snap.Rows {
		byResource[row.Resource+"\x00"+row.Scope] = row
	}

	row := byResource["workers.scripts\x00"]
	if row.Used != 80 || row.Limit != 500 || row.Percent != 16 {
		t.Fatalf("workers.scripts = %+v, want used=80 limit=500 pct=16", row)
	}
	row = byResource["workers.daily_requests\x00"]
	if row.Used != 42000 || row.Limit != 0 || row.Percent != 0 {
		t.Fatalf("workers.daily_requests = %+v, want used=42000 limit=0(unlimited) pct=0", row)
	}
	row = byResource["r2.buckets\x00"]
	if row.Used != 7 || row.Limit != 1000000 {
		t.Fatalf("r2.buckets = %+v", row)
	}
	row = byResource["zones.count\x00"]
	if row.Used != 1 || row.Limit != 0 {
		t.Fatalf("zones.count = %+v, want informational used=1 limit=0", row)
	}
	row = byResource["dns.records\x00one.example"]
	if row.Used != 180 || row.Limit != 200 || row.Percent != 90 || row.LimitSource != "live-api" {
		t.Fatalf("dns.records = %+v, want 180/200=90%% live-api", row)
	}
	if len(snap.Sources) != 0 {
		t.Fatalf("unexpected source errors: %+v", snap.Sources)
	}
}

func TestSnapshotPartialFailure(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{err: errors.New("workers down")}),
		WithLimitsR2(fakeBucketLister{n: 3}),
		WithLimitsZones(fakeZoneLister{err: errors.New("zones down")}),
		WithLimitsAnalytics(fakeWorkersAnalytics{err: errors.New("analytics down")}),
	)
	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("partial failure must not fail the snapshot: %v", err)
	}
	if len(snap.Rows) == 0 {
		t.Fatal("healthy sources must still produce rows")
	}
	found := map[string]bool{}
	for _, se := range snap.Sources {
		found[se.Source] = true
	}
	if !found["workers.list"] || !found["zones.list"] || !found["analytics.workers"] {
		t.Fatalf("source errors = %+v, want workers.list, zones.list, analytics.workers", snap.Sources)
	}
}

func TestSnapshotAllSourcesFail(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsWorkers(fakeWorkersLister{err: errors.New("x")}),
		WithLimitsR2(fakeBucketLister{err: errors.New("x")}),
		WithLimitsZones(fakeZoneLister{err: errors.New("x")}),
		WithLimitsAnalytics(fakeWorkersAnalytics{err: errors.New("x")}),
	)
	if _, err := s.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("all sources failing must return an error")
	}
}

func TestSnapshotBucketScope(t *testing.T) {
	s := snapshotFixture(t,
		WithLimitsR2(fakeBucketLister{n: 1}),
		WithLimitsDomains(fakeDomainLister{n: 99}),
	)
	snap, err := s.Snapshot(context.Background(), "my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range snap.Rows {
		if row.Resource == "r2.custom_domains_per_bucket" {
			if row.Scope != "my-bucket" || row.Used != 99 || row.Limit != 100 || row.Percent != 99 {
				t.Fatalf("bucket row = %+v, want my-bucket 99/100=99%%", row)
			}
			return
		}
	}
	t.Fatal("r2.custom_domains_per_bucket row missing with --bucket set")
}

func TestSnapshotDNSStaticFallback(t *testing.T) {
	// Live DNS endpoint fails (testLimitServer returns 403 for every path);
	// zone is free + created 2025 → static 200.
	s := snapshotFixture(t,
		WithLimitsZones(fakeZoneLister{zones: []*Zone{
			{ID: "z1", Name: "new.example", Plan: ZonePlan{LegacyID: "free"}, CreatedOn: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)},
		}}),
	)
	snap, err := s.Snapshot(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range snap.Rows {
		if row.Resource == "dns.records" && row.Scope == "new.example" {
			if row.Limit != 200 || row.LimitSource != "static-docs" {
				t.Fatalf("dns fallback row = %+v, want limit=200 static-docs", row)
			}
			return
		}
	}
	t.Fatal("dns.records fallback row missing")
}
