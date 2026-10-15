package cosmoflare

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// almostEqual checks if two float64 values are within epsilon of each other.
func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

const testEpsilon = 0.001

// ---------------------------------------------------------------------------
// NewCostService
// ---------------------------------------------------------------------------

// TestCostService_New TestCostService_New verifies that NewCostService
// accepts valid credentials and stores the account ID on the service.
func TestCostService_New(t *testing.T) {
	t.Parallel()
	svc, err := NewCostService("acc123", "tok456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil CostService")
	}
	if svc.accountID != "acc123" {
		t.Errorf("accountID = %q, want %q", svc.accountID, "acc123")
	}
}

// TestCostService_NewEmptyAccountID TestCostService_NewEmptyAccountID
// verifies that an empty account ID is rejected with an *R2ValidationError.
func TestCostService_NewEmptyAccountID(t *testing.T) {
	t.Parallel()
	_, err := NewCostService("", "tok456")
	if err == nil {
		t.Fatal("expected error for empty accountID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

// TestCostService_NewEmptyAPIToken TestCostService_NewEmptyAPIToken verifies
// that an empty API token is rejected with an *R2ValidationError.
func TestCostService_NewEmptyAPIToken(t *testing.T) {
	t.Parallel()
	_, err := NewCostService("acc123", "")
	if err == nil {
		t.Fatal("expected error for empty apiToken")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// Pricing constants
// ---------------------------------------------------------------------------

// TestCostPricingConstants TestCostPricingConstants pins the pricing
// constants to Cloudflare's published rates so accidental edits to billing
// math are caught in review.
func TestCostPricingConstants(t *testing.T) {
	t.Parallel()
	// Verify pricing constants match Cloudflare published rates
	if R2StoragePerGB != 0.015 {
		t.Errorf("R2StoragePerGB = %f, want 0.015", R2StoragePerGB)
	}
	if R2ClassAPerMillion != 4.50 {
		t.Errorf("R2ClassAPerMillion = %f, want 4.50", R2ClassAPerMillion)
	}
	if R2ClassBPerMillion != 0.36 {
		t.Errorf("R2ClassBPerMillion = %f, want 0.36", R2ClassBPerMillion)
	}
	if WorkersRequestsPerMillion != 0.50 {
		t.Errorf("WorkersRequestsPerMillion = %f, want 0.50", WorkersRequestsPerMillion)
	}
	if WorkersFreeRequests != 100000 {
		t.Errorf("WorkersFreeRequests = %d, want 100000", WorkersFreeRequests)
	}
	if KVReadsPerMillion != 0.50 {
		t.Errorf("KVReadsPerMillion = %f, want 0.50", KVReadsPerMillion)
	}
	if KVWritesPerMillion != 5.00 {
		t.Errorf("KVWritesPerMillion = %f, want 5.00", KVWritesPerMillion)
	}
	if KVStoragePerGB != 0.50 {
		t.Errorf("KVStoragePerGB = %f, want 0.50", KVStoragePerGB)
	}
}

// ---------------------------------------------------------------------------
// EstimateR2Cost
// ---------------------------------------------------------------------------

// TestCostService_EstimateR2Cost TestCostService_EstimateR2Cost verifies the
// R2 estimate math (storage, class A, class B, and their sum) for a usage
// fixture with non-trivial values.
func TestCostService_EstimateR2Cost(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	usage := R2Usage{
		StorageGB: 100.0,
		ClassAOps: 2000000,
		ClassBOps: 10000000,
	}

	est := svc.EstimateR2Cost(usage)

	// Storage: 100 * 0.015 = 1.50
	if !almostEqual(est.StorageCost, 1.50, testEpsilon) {
		t.Errorf("StorageCost = %f, want 1.50", est.StorageCost)
	}

	// Class A: 2M * 4.50/M = 9.00
	if !almostEqual(est.ClassACost, 9.00, testEpsilon) {
		t.Errorf("ClassACost = %f, want 9.00", est.ClassACost)
	}

	// Class B: 10M * 0.36/M = 3.60
	if !almostEqual(est.ClassBCost, 3.60, testEpsilon) {
		t.Errorf("ClassBCost = %f, want 3.60", est.ClassBCost)
	}

	// Total: 1.50 + 9.00 + 3.60 = 14.10
	if !almostEqual(est.TotalCost, 14.10, testEpsilon) {
		t.Errorf("TotalCost = %f, want 14.10", est.TotalCost)
	}
}

// TestCostService_EstimateR2Cost_Zero TestCostService_EstimateR2Cost_Zero
// verifies that zero R2 usage costs nothing.
func TestCostService_EstimateR2Cost_Zero(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	est := svc.EstimateR2Cost(R2Usage{})

	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", est.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// EstimateWorkersCost
// ---------------------------------------------------------------------------

// TestCostService_EstimateWorkersCost TestCostService_EstimateWorkersCost
// verifies that requests above the 100k free tier are billed at the
// per-million rate.
func TestCostService_EstimateWorkersCost(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	usage := WorkersUsage{
		Requests:  1100000, // 1.1M requests
		CPUTimeMs: 50000,
	}

	est := svc.EstimateWorkersCost(usage)

	// Billable: 1100000 - 100000 = 1000000 = 1M
	// Cost: 1M * 0.50/M = 0.50
	if !almostEqual(est.RequestsCost, 0.50, testEpsilon) {
		t.Errorf("RequestsCost = %f, want 0.50", est.RequestsCost)
	}
	if !almostEqual(est.TotalCost, 0.50, testEpsilon) {
		t.Errorf("TotalCost = %f, want 0.50", est.TotalCost)
	}
}

// TestCostService_EstimateWorkersCost_UnderFreeTier
// TestCostService_EstimateWorkersCost_UnderFreeTier verifies that usage below
// the free-tier threshold is not billed.
func TestCostService_EstimateWorkersCost_UnderFreeTier(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	usage := WorkersUsage{
		Requests:  50000, // under 100k free tier
		CPUTimeMs: 1000,
	}

	est := svc.EstimateWorkersCost(usage)

	if est.RequestsCost != 0 {
		t.Errorf("RequestsCost = %f, want 0 (under free tier)", est.RequestsCost)
	}
	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0 (under free tier)", est.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// EstimateKVCost
// ---------------------------------------------------------------------------

// TestCostService_EstimateKVCost TestCostService_EstimateKVCost verifies the
// KV estimate math for reads, writes, storage, and their sum.
func TestCostService_EstimateKVCost(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	usage := KVUsage{
		Reads:     5000000, // 5M reads
		Writes:    1000000, // 1M writes
		StorageGB: 2.0,
	}

	est := svc.EstimateKVCost(usage)

	// Reads: 5M * 0.50/M = 2.50
	if !almostEqual(est.ReadsCost, 2.50, testEpsilon) {
		t.Errorf("ReadsCost = %f, want 2.50", est.ReadsCost)
	}

	// Writes: 1M * 5.00/M = 5.00
	if !almostEqual(est.WritesCost, 5.00, testEpsilon) {
		t.Errorf("WritesCost = %f, want 5.00", est.WritesCost)
	}

	// Storage: 2.0 * 0.50 = 1.00
	if !almostEqual(est.StorageCost, 1.00, testEpsilon) {
		t.Errorf("StorageCost = %f, want 1.00", est.StorageCost)
	}

	// Total: 2.50 + 5.00 + 1.00 = 8.50
	if !almostEqual(est.TotalCost, 8.50, testEpsilon) {
		t.Errorf("TotalCost = %f, want 8.50", est.TotalCost)
	}
}

// TestCostService_EstimateKVCost_Zero TestCostService_EstimateKVCost_Zero
// verifies that zero KV usage costs nothing.
func TestCostService_EstimateKVCost_Zero(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	est := svc.EstimateKVCost(KVUsage{})

	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", est.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// EstimateTotal
// ---------------------------------------------------------------------------

// TestCostService_EstimateTotal TestCostService_EstimateTotal verifies that
// EstimateTotal populates all three per-service estimates, sums them into
// TotalMonthlyCost, and sets disclaimer and period metadata.
func TestCostService_EstimateTotal(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	r2Usage := R2Usage{StorageGB: 100, ClassAOps: 1000000, ClassBOps: 5000000}
	workersUsage := WorkersUsage{Requests: 1100000}
	kvUsage := KVUsage{Reads: 2000000, Writes: 500000, StorageGB: 1.0}

	total := svc.EstimateTotal(r2Usage, workersUsage, kvUsage)

	if total.R2 == nil {
		t.Fatal("expected non-nil R2 estimate")
	}
	if total.Workers == nil {
		t.Fatal("expected non-nil Workers estimate")
	}
	if total.KV == nil {
		t.Fatal("expected non-nil KV estimate")
	}

	// Verify total is sum of per-service totals
	expected := total.R2.TotalCost + total.Workers.TotalCost + total.KV.TotalCost
	if total.TotalMonthlyCost != expected {
		t.Errorf("TotalMonthlyCost = %f, want %f", total.TotalMonthlyCost, expected)
	}

	if total.Disclaimer == "" {
		t.Error("expected non-empty Disclaimer")
	}

	if total.Period == "" {
		t.Error("expected non-empty Period")
	}
}

// TestCostService_EstimateTotal_AllZero TestCostService_EstimateTotal_AllZero
// verifies that all-zero usage produces a zero total.
func TestCostService_EstimateTotal_AllZero(t *testing.T) {
	t.Parallel()
	svc, _ := NewCostService("acc123", "tok456")

	total := svc.EstimateTotal(R2Usage{}, WorkersUsage{}, KVUsage{})

	if total.TotalMonthlyCost != 0 {
		t.Errorf("TotalMonthlyCost = %f, want 0", total.TotalMonthlyCost)
	}
}

// ---------------------------------------------------------------------------
// CollectMeasuredUsage (measured cost inputs, D-cost FEAT-069/070)
// ---------------------------------------------------------------------------

// Dataset needles used to route the GraphQL fixtures below. They match the
// dataset names in the queries analytics.go actually sends.
const (
	costNeedleWorkers  = "workersInvocationsAdaptive"
	costNeedleR2Ops    = "r2OperationsAdaptiveGroups"
	costNeedleR2Stor   = "r2StorageAdaptiveGroups"
	costNeedleKV       = "kvOperationsAdaptiveGroups"
	costEmptyAccountOK = `{"data":{"viewer":{"accounts":[]}}}`
)

// costRoutingServer routes GraphQL queries by dataset name in the request
// body (analyticsServer pattern, extended): the handler inspects the query
// for each needle and replies with the matching fixture. Bodies of matched
// requests are recorded in seen when non-nil. Unmatched queries get an
// empty account.
func costRoutingServer(t *testing.T, routes map[string]string, seen map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body := string(data)
		w.Header().Set("Content-Type", "application/json")
		for needle, resp := range routes {
			if strings.Contains(body, needle) {
				if seen != nil {
					seen[needle] = body
				}
				w.Write([]byte(resp))
				return
			}
		}
		w.Write([]byte(costEmptyAccountOK))
	}))
}

// costSeamAnalytics redirects costAnalyticsFn to srv for the test's duration.
// The seam must be restored so parallel-unsafe globals never leak.
func costSeamAnalytics(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := costAnalyticsFn
	costAnalyticsFn = func(accountID, apiToken string) *AnalyticsService {
		return NewAnalyticsService(accountID, apiToken, WithAnalyticsBaseURL(srv.URL))
	}
	t.Cleanup(func() { costAnalyticsFn = prev })
}

// costMeasuredFixtures returns one fixture response per cost dataset with
// numbers chosen so the 7-day ×30/7 scaling is exact: Workers 700 requests
// → 3000, 7700 CPU ms → 33000; R2 490 Class A → 2100, 490 Class B → 2100
// (70 deletes count toward neither); KV 350 reads → 1500, 70 writes → 300;
// storage 7e9 bytes → 7 GB as a non-scaled gauge.
func costMeasuredFixtures() map[string]string {
	return map[string]string{
		costNeedleWorkers: `{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[` +
			`{"sum":{"requests":490,"errors":0,"subrequests":10,"cpuTimeUs":7000000},"quantiles":{"cpuTimeP50":1.0,"cpuTimeP99":5.0},"dimensions":{"scriptName":"api"}},` +
			`{"sum":{"requests":210,"errors":0,"subrequests":5,"cpuTimeUs":700000},"quantiles":{"cpuTimeP50":0.5,"cpuTimeP99":2.0},"dimensions":{"scriptName":"cron"}}` +
			`]}]}}}`,
		costNeedleR2Ops: `{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[` +
			`{"sum":{"requests":350},"dimensions":{"actionType":"PutObject","actionStatus":"success","bucketName":"media"}},` +
			`{"sum":{"requests":140},"dimensions":{"actionType":"ListObjects","actionStatus":"success","bucketName":"media"}},` +
			`{"sum":{"requests":490},"dimensions":{"actionType":"GetObject","actionStatus":"success","bucketName":"media"}},` +
			`{"sum":{"requests":70},"dimensions":{"actionType":"DeleteObject","actionStatus":"success","bucketName":"media"}}` +
			`]}]}}}`,
		costNeedleR2Stor: `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[` +
			`{"max":{"objectCount":10,"uploadCount":0,"payloadSize":4000000000,"metadataSize":100},"dimensions":{"bucketName":"media"}},` +
			`{"max":{"objectCount":5,"uploadCount":0,"payloadSize":3000000000,"metadataSize":50},"dimensions":{"bucketName":"backups"}}` +
			`]}]}}}`,
		costNeedleKV: `{"data":{"viewer":{"accounts":[{"kvOperationsAdaptiveGroups":[` +
			`{"dimensions":{"namespaceId":"ns1","actionType":"read"},"sum":{"requests":245}},` +
			`{"dimensions":{"namespaceId":"ns1","actionType":"write"},"sum":{"requests":70}},` +
			`{"dimensions":{"namespaceId":"ns2","actionType":"read"},"sum":{"requests":105}},` +
			`{"dimensions":{"namespaceId":"ns2","actionType":"list"},"sum":{"requests":7}}` +
			`]}]}}}`,
	}
}

// TestCollectMeasuredUsage covers sums, ClassA/B classification, the KV
// reads/writes split, exact ×30/7 scaling at 7d, the non-scaled storage
// gauge, ProjectedTo30d true@7d / false@30d, the days bounds, and the
// hard-error rule when one dataset fails.
func TestCollectMeasuredUsage(t *testing.T) {
	svc, err := NewCostService("acct", "tok")
	if err != nil {
		t.Fatalf("NewCostService: %v", err)
	}

	t.Run("sums-and-7d-scaling", func(t *testing.T) {
		seen := map[string]string{}
		srv := costRoutingServer(t, costMeasuredFixtures(), seen)
		costSeamAnalytics(t, srv)

		m, err := svc.CollectMeasuredUsage(context.Background(), 7)
		if err != nil {
			t.Fatalf("CollectMeasuredUsage(7): %v", err)
		}

		// All four datasets must have been queried.
		for _, needle := range []string{costNeedleWorkers, costNeedleR2Ops, costNeedleR2Stor, costNeedleKV} {
			if _, ok := seen[needle]; !ok {
				t.Errorf("dataset %s was never queried", needle)
			}
		}

		// Cumulative fields scale ×30/7 exactly.
		if m.Workers.Requests != 3000 {
			t.Errorf("Workers.Requests = %d, want 3000 (700 × 30/7)", m.Workers.Requests)
		}
		if m.Workers.CPUTimeMs != 33000 {
			t.Errorf("Workers.CPUTimeMs = %d, want 33000 (7700 ms × 30/7)", m.Workers.CPUTimeMs)
		}
		if m.R2.ClassAOps != 2100 {
			t.Errorf("R2.ClassAOps = %d, want 2100 (490 × 30/7)", m.R2.ClassAOps)
		}
		if m.R2.ClassBOps != 2100 {
			t.Errorf("R2.ClassBOps = %d, want 2100 (490 × 30/7)", m.R2.ClassBOps)
		}
		if m.KV.Reads != 1500 {
			t.Errorf("KV.Reads = %d, want 1500 (350 × 30/7)", m.KV.Reads)
		}
		if m.KV.Writes != 300 {
			t.Errorf("KV.Writes = %d, want 300 (70 × 30/7)", m.KV.Writes)
		}
		// Storage is a gauge: NOT scaled.
		if !almostEqual(m.R2.StorageGB, 7.0, testEpsilon) {
			t.Errorf("R2.StorageGB = %v, want 7 (gauge, never scaled)", m.R2.StorageGB)
		}
		// KV storage has no API source and stays 0.
		if m.KV.StorageGB != 0 {
			t.Errorf("KV.StorageGB = %v, want 0 (never modeled)", m.KV.StorageGB)
		}

		if !m.ProjectedTo30d {
			t.Error("ProjectedTo30d = false at 7d, want true")
		}

		// Window bounds: trailing 7 days ending now (tolerance 2 min).
		if d := time.Since(m.WindowStart); d < 7*24*time.Hour-2*time.Minute || d > 7*24*time.Hour+2*time.Minute {
			t.Errorf("WindowStart = %v, want ~7 days ago", m.WindowStart)
		}
		if d := time.Since(m.WindowEnd); d < -2*time.Minute || d > 2*time.Minute {
			t.Errorf("WindowEnd = %v, want ~now", m.WindowEnd)
		}
	})

	t.Run("30d-not-projected", func(t *testing.T) {
		srv := costRoutingServer(t, costMeasuredFixtures(), nil)
		costSeamAnalytics(t, srv)

		m, err := svc.CollectMeasuredUsage(context.Background(), 30)
		if err != nil {
			t.Fatalf("CollectMeasuredUsage(30): %v", err)
		}
		if m.ProjectedTo30d {
			t.Error("ProjectedTo30d = true at 30d, want false")
		}
		// scale = 30/30 = 1: fixture sums pass through unscaled.
		if m.Workers.Requests != 700 || m.R2.ClassAOps != 490 || m.R2.ClassBOps != 490 || m.KV.Reads != 350 || m.KV.Writes != 70 {
			t.Errorf("30d usage = %+v / %+v / %+v, want unscaled fixture sums", m.Workers, m.R2, m.KV)
		}
	})

	t.Run("days-bounds", func(t *testing.T) {
		hit := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit = true
			w.Write([]byte(costEmptyAccountOK))
		}))
		costSeamAnalytics(t, srv)

		for _, days := range []int{0, -1, 91, 1000} {
			m, err := svc.CollectMeasuredUsage(context.Background(), days)
			if err == nil {
				t.Fatalf("CollectMeasuredUsage(%d): want error, got nil", days)
			}
			if m != nil {
				t.Errorf("CollectMeasuredUsage(%d): want nil usage on error", days)
			}
			if _, ok := err.(*R2ValidationError); !ok {
				t.Errorf("CollectMeasuredUsage(%d): want *R2ValidationError, got %T", days, err)
			}
			if !strings.Contains(err.Error(), "CostService.CollectMeasuredUsage") {
				t.Errorf("error %q missing op CostService.CollectMeasuredUsage", err)
			}
		}
		if hit {
			t.Error("server was hit despite days validation failure")
		}
	})

	t.Run("dataset-failure-is-hard-error", func(t *testing.T) {
		fixtures := costMeasuredFixtures()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			body := string(data)
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(body, costNeedleKV):
				// One dataset answers 500 with a non-JSON body.
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			case strings.Contains(body, costNeedleWorkers):
				w.Write([]byte(fixtures[costNeedleWorkers]))
			case strings.Contains(body, costNeedleR2Ops):
				w.Write([]byte(fixtures[costNeedleR2Ops]))
			case strings.Contains(body, costNeedleR2Stor):
				w.Write([]byte(fixtures[costNeedleR2Stor]))
			default:
				w.Write([]byte(costEmptyAccountOK))
			}
		}))
		costSeamAnalytics(t, srv)

		m, err := svc.CollectMeasuredUsage(context.Background(), 7)
		if err == nil {
			t.Fatalf("want hard error when one dataset 500s, got usage %+v", m)
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error %q should carry the HTTP 500", err)
		}
		if m != nil {
			t.Errorf("usage must be nil on dataset failure, got %+v", m)
		}
	})
}

// TestCollectMeasuredUsageStorageGauge verifies the R2 storage gauge is read
// over a trailing 24h window (not the cost window): the r2StorageAdaptiveGroups
// request must carry start≈now-24h / end≈now while the cumulative datasets
// carry the full trailing-days window.
func TestCollectMeasuredUsageStorageGauge(t *testing.T) {
	svc, err := NewCostService("acct", "tok")
	if err != nil {
		t.Fatalf("NewCostService: %v", err)
	}
	seen := map[string]string{}
	srv := costRoutingServer(t, costMeasuredFixtures(), seen)
	costSeamAnalytics(t, srv)

	if _, err := svc.CollectMeasuredUsage(context.Background(), 7); err != nil {
		t.Fatalf("CollectMeasuredUsage: %v", err)
	}

	spanOf := func(needle string) time.Duration {
		t.Helper()
		body, ok := seen[needle]
		if !ok {
			t.Fatalf("dataset %s was never queried", needle)
		}
		var req struct {
			Variables struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"variables"`
		}
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			t.Fatalf("request body for %s not JSON: %v\n%s", needle, err, body)
		}
		start, err := time.Parse(time.RFC3339, req.Variables.Start)
		if err != nil {
			t.Fatalf("bad start %q: %v", req.Variables.Start, err)
		}
		end, err := time.Parse(time.RFC3339, req.Variables.End)
		if err != nil {
			t.Fatalf("bad end %q: %v", req.Variables.End, err)
		}
		return end.Sub(start)
	}

	// Tolerance 2 min: the window is built from time.Now() inside the callee.
	got := spanOf(costNeedleR2Stor)
	if got < 24*time.Hour-2*time.Minute || got > 24*time.Hour+2*time.Minute {
		t.Errorf("R2 storage window span = %v, want ~24h (gauge, trailing day)", got)
	}
	cum := spanOf(costNeedleWorkers)
	if cum < 7*24*time.Hour-2*time.Minute || cum > 7*24*time.Hour+2*time.Minute {
		t.Errorf("Workers window span = %v, want ~7d (trailing days)", cum)
	}
}
