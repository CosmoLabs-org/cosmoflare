package cosmoflare

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// usageMockServer answers the four analytics datasets by sniffing the query
// body, mirroring the wave-1 collection-test pattern.
func usageMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	workersResp := `{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[` +
		`{"sum":{"requests":6000000,"errors":1,"subrequests":0,"cpuTimeUs":15000000000},"quantiles":{"cpuTimeP50":1,"cpuTimeP99":10},"dimensions":{"scriptName":"a","status":"ok"}},` +
		`{"sum":{"requests":2000000,"errors":0,"subrequests":0,"cpuTimeUs":5000000000},"quantiles":{"cpuTimeP50":1,"cpuTimeP99":10},"dimensions":{"scriptName":"b","status":"ok"}}` +
		`]}]}}}`
	doResp := `{"data":{"viewer":{"accounts":[{"durableObjectsInvocationsAdaptiveGroups":[` +
		`{"sum":{"requests":600000,"errors":0,"wallTime":640000000000},"dimensions":{"scriptName":"lobby","namespaceId":"aaaaaaaa1"}}` +
		`]}]}}}`
	storageResp := `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[` +
		`{"max":{"objectCount":10,"payloadSize":2000000000},"dimensions":{"bucketName":"media"}},` +
		`{"max":{"objectCount":10,"payloadSize":3000000000},"dimensions":{"bucketName":"backups"}}` +
		`]}]}}}`
	opsResp := `{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[` +
		`{"sum":{"requests":4000000},"dimensions":{"actionType":"GetObject","actionStatus":"success","bucketName":"media"}},` +
		`{"sum":{"requests":500000},"dimensions":{"actionType":"PutObject","actionStatus":"success","bucketName":"media"}},` +
		`{"sum":{"requests":200000},"dimensions":{"actionType":"ListObjects","actionStatus":"success","bucketName":"media"}}` +
		`]}]}}}`
	d1Resp := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
		`{"sum":{"rowsRead":1000000000,"rowsWritten":0,"readQueries":10},"dimensions":{"databaseId":"db-a"}},` +
		`{"sum":{"rowsRead":500000000,"rowsWritten":0,"readQueries":5},"dimensions":{"databaseId":"db-b"}}` +
		`]}]}}}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		q := string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(q, "workersInvocationsAdaptive"):
			w.Write([]byte(workersResp))
		case strings.Contains(q, "durableObjectsInvocationsAdaptiveGroups"):
			w.Write([]byte(doResp))
		case strings.Contains(q, "GetObject") || strings.Contains(q, "actionType"):
			w.Write([]byte(opsResp))
		case strings.Contains(q, "d1AnalyticsAdaptiveGroups"):
			w.Write([]byte(d1Resp))
		default:
			w.Write([]byte(storageResp))
		}
	}))
}

func usageDim(s *UsageSnapshot, id string) UsageDimension {
	for _, d := range s.Dimensions {
		if d.ID == id {
			return d
		}
	}
	return UsageDimension{ID: id}
}

func approx(t *testing.T, got, want float64, label string) {
	t.Helper()
	if d := got - want; d > 0.01 || d < -0.01 {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}

// TestCycleWindow pins the anchor semantics: 0/1 = calendar month UTC;
// anniversary anchors roll back a month before the anchor day.
func TestCycleWindow(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	s, e := CycleWindow(0, now)
	if s != time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) || e != time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) {
		t.Errorf("calendar cycle = %v..%v", s, e)
	}

	s, e = CycleWindow(20, now) // before the 20th → previous month's 20th
	if s != time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC) || e != time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) {
		t.Errorf("anchor-20 cycle = %v..%v", s, e)
	}

	s, e = CycleWindow(16, now) // on the anchor day → today's cycle
	if s != time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC) {
		t.Errorf("anchor-16 cycle start = %v", s)
	}

	s, e = CycleWindow(31, now) // clamp: months without day 31
	if s.Day() != 30 || s.Month() != time.September {
		t.Errorf("anchor-31 clamp = %v (want Sep 30)", s)
	}
}

// TestCollectUsage pins the pacing arithmetic end to end against fixed
// mocks: mid-calendar-month (15 of 31 days elapsed) with known usage on
// every dimension.
func TestCollectUsage(t *testing.T) {
	srv := usageMockServer(t)
	defer srv.Close()
	analytics := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))

	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	snap, err := CollectUsage(context.Background(), analytics, "paid", 0, now)
	if err != nil {
		t.Fatalf("CollectUsage: %v", err)
	}
	if snap.CycleStart != time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("cycle start = %v", snap.CycleStart)
	}
	approx(t, snap.DaysElapsed, 15, "days elapsed")
	approx(t, snap.DaysTotal, 31, "days total")

	wr := usageDim(snap, "workers.requests_monthly")
	approx(t, wr.Used, 8000000, "workers requests used")
	approx(t, wr.Limit, 10000000, "workers requests limit")
	approx(t, wr.Pct, 80, "workers requests pct")
	approx(t, wr.ProjectedPct, 80*31.0/15.0, "workers requests projected")

	wc := usageDim(snap, "workers.cpu_ms_monthly")
	approx(t, wc.Used, 20000000, "cpu ms used")
	approx(t, wc.Pct, 20000000.0/30000000*100, "cpu ms pct")

	dr := usageDim(snap, "do.requests_monthly")
	approx(t, dr.Used, 600000, "do requests used")
	approx(t, dr.Pct, 60, "do requests pct")

	dd := usageDim(snap, "do.duration_gb_s_monthly")
	// wallTime 640e9 µs = 640,000 s; billed memory 128 MB = 0.125 GB → 80,000 GB-s
	approx(t, dd.Used, 80000, "do duration GB-s")
	approx(t, dd.Pct, 20, "do duration pct")

	rs := usageDim(snap, "r2.storage_gb_monthly")
	approx(t, rs.Used, 5, "r2 storage GB")
	approx(t, rs.Pct, 50, "r2 storage pct")

	ca := usageDim(snap, "r2.class_a_monthly")
	approx(t, ca.Used, 700000, "class A used")
	approx(t, ca.Pct, 70, "class A pct")

	cb := usageDim(snap, "r2.class_b_monthly")
	approx(t, cb.Used, 4000000, "class B used")
	approx(t, cb.Pct, 40, "class B pct")
}

// TestCollectUsageUnknownPlanLimit limits resolve to zero (enterprise null
// tiers): dims render with Limit 0 and never NaN.
func TestCollectUsageUnknownPlanLimit(t *testing.T) {
	srv := usageMockServer(t)
	defer srv.Close()
	analytics := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))

	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	snap, err := CollectUsage(context.Background(), analytics, "enterprise", 0, now)
	if err != nil {
		t.Fatalf("CollectUsage: %v", err)
	}
	wr := usageDim(snap, "workers.requests_monthly")
	if wr.Limit != 0 || wr.Pct != 0 || wr.ProjectedPct != 0 {
		t.Errorf("unknown-plan dim = %+v, want zero limit/pct (never NaN)", wr)
	}
	if wr.Used == 0 {
		t.Error("usage should still be collected when the limit is unknown")
	}
}

// usageD1FailureMockServer is usageMockServer with the D1 branch replaced by
// a GraphQL error response, to prove D1 analytics failures are additive.
func usageD1FailureMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	workersResp := `{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[` +
		`{"sum":{"requests":6000000,"errors":1,"subrequests":0,"cpuTimeUs":15000000000},"quantiles":{"cpuTimeP50":1,"cpuTimeP99":10},"dimensions":{"scriptName":"a","status":"ok"}}` +
		`]}]}}}`
	doResp := `{"data":{"viewer":{"accounts":[{"durableObjectsInvocationsAdaptiveGroups":[` +
		`{"sum":{"requests":600000,"errors":0,"wallTime":640000000000},"dimensions":{"scriptName":"lobby","namespaceId":"aaaaaaaa1"}}` +
		`]}]}}}`
	storageResp := `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[` +
		`{"max":{"objectCount":10,"payloadSize":2000000000},"dimensions":{"bucketName":"media"}}` +
		`]}]}}}`
	opsResp := `{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[` +
		`{"sum":{"requests":4000000},"dimensions":{"actionType":"GetObject","actionStatus":"success","bucketName":"media"}}` +
		`]}]}}}`
	d1Resp := `{"errors":[{"message":"boom"}]}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		q := string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(q, "workersInvocationsAdaptive"):
			w.Write([]byte(workersResp))
		case strings.Contains(q, "durableObjectsInvocationsAdaptiveGroups"):
			w.Write([]byte(doResp))
		case strings.Contains(q, "GetObject") || strings.Contains(q, "actionType"):
			w.Write([]byte(opsResp))
		case strings.Contains(q, "d1AnalyticsAdaptiveGroups"):
			w.Write([]byte(d1Resp))
		default:
			w.Write([]byte(storageResp))
		}
	}))
}

// TestCollectUsageD1Dimension pins the D1 rows-read pacing dimension: the
// per-database summaries sum into one monthly account-wide figure.
func TestCollectUsageD1Dimension(t *testing.T) {
	srv := usageMockServer(t)
	defer srv.Close()
	analytics := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))

	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	snap, err := CollectUsage(context.Background(), analytics, "paid", 0, now)
	if err != nil {
		t.Fatalf("CollectUsage: %v", err)
	}
	d := usageDim(snap, "d1.rows_read_monthly")
	approx(t, d.Used, 1.5e9, "d1 rows read used")
	approx(t, d.Limit, 2.5e10, "d1 rows read limit")
}

// TestCollectUsageD1FailureIsAdditive pins design D15: a D1 analytics failure
// must not fail CollectUsage or drop the Workers/DO/R2 pacing rows — the D1
// dimension is simply omitted.
func TestCollectUsageD1FailureIsAdditive(t *testing.T) {
	srv := usageD1FailureMockServer(t)
	defer srv.Close()
	analytics := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))

	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	snap, err := CollectUsage(context.Background(), analytics, "paid", 0, now)
	if err != nil {
		t.Fatalf("CollectUsage: %v", err)
	}
	if usageDim(snap, "workers.requests_monthly").Used == 0 {
		t.Error("workers.requests_monthly must remain present when D1 analytics fail")
	}
	for _, d := range snap.Dimensions {
		if d.ID == "d1.rows_read_monthly" {
			t.Error("d1.rows_read_monthly must be omitted when D1 analytics fail")
		}
	}
}
