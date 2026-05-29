package r2go2

import (
	"math"
	"testing"
)

// almostEqual checks if two float64 values are within epsilon of each other.
func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

const testEpsilon = 0.001

// ---------------------------------------------------------------------------
// NewCostService
// ---------------------------------------------------------------------------

func TestCostService_New(t *testing.T) {
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

func TestCostService_NewEmptyAccountID(t *testing.T) {
	_, err := NewCostService("", "tok456")
	if err == nil {
		t.Fatal("expected error for empty accountID")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestCostService_NewEmptyAPIToken(t *testing.T) {
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

func TestCostPricingConstants(t *testing.T) {
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

func TestCostService_EstimateR2Cost(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	usage := R2Usage{
		StorageGB:    100.0,
		ClassAOps:    2000000,
		ClassBOps:    10000000,
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

func TestCostService_EstimateR2Cost_Zero(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	est := svc.EstimateR2Cost(R2Usage{})

	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", est.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// EstimateWorkersCost
// ---------------------------------------------------------------------------

func TestCostService_EstimateWorkersCost(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	usage := WorkersUsage{
		Requests:     1100000, // 1.1M requests
		CPUTimeMs:    50000,
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

func TestCostService_EstimateWorkersCost_UnderFreeTier(t *testing.T) {
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

func TestCostService_EstimateKVCost(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	usage := KVUsage{
		Reads:     5000000,  // 5M reads
		Writes:    1000000,  // 1M writes
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

func TestCostService_EstimateKVCost_Zero(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	est := svc.EstimateKVCost(KVUsage{})

	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", est.TotalCost)
	}
}

// ---------------------------------------------------------------------------
// EstimateTotal
// ---------------------------------------------------------------------------

func TestCostService_EstimateTotal(t *testing.T) {
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

func TestCostService_EstimateTotal_AllZero(t *testing.T) {
	svc, _ := NewCostService("acc123", "tok456")

	total := svc.EstimateTotal(R2Usage{}, WorkersUsage{}, KVUsage{})

	if total.TotalMonthlyCost != 0 {
		t.Errorf("TotalMonthlyCost = %f, want 0", total.TotalMonthlyCost)
	}
}
