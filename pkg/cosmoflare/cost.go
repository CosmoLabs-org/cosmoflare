package cosmoflare

// ---------------------------------------------------------------------------
// Cloudflare Pricing Constants (as of 2025)
// ---------------------------------------------------------------------------

const (
	// R2 Storage pricing
	R2StoragePerGB    = 0.015 // $/GB/month
	R2ClassAPerMillion = 4.50  // $/million Class A operations (PUT, POST, LIST)
	R2ClassBPerMillion = 0.36  // $/million Class B operations (GET, HEAD)

	// Workers pricing (Paid plan)
	WorkersRequestsPerMillion = 0.50   // $/million requests after free tier
	WorkersFreeRequests       = 100000 // Free tier: 100K requests/day (used as monthly approximation)

	// KV pricing
	KVReadsPerMillion  = 0.50 // $/million reads
	KVWritesPerMillion = 5.00 // $/million writes
	KVStoragePerGB     = 0.50 // $/GB/month
)

// ---------------------------------------------------------------------------
// Usage structs — input to cost estimation
// ---------------------------------------------------------------------------

// R2Usage represents R2 storage usage metrics for cost estimation.
type R2Usage struct {
	StorageGB float64 `json:"storage_gb"`
	ClassAOps int64   `json:"class_a_ops"`
	ClassBOps int64   `json:"class_b_ops"`
}

// WorkersUsage represents Workers usage metrics for cost estimation.
type WorkersUsage struct {
	Requests  int64 `json:"requests"`
	CPUTimeMs int64 `json:"cpu_time_ms"`
}

// KVUsage represents KV usage metrics for cost estimation.
type KVUsage struct {
	Reads     int64   `json:"reads"`
	Writes    int64   `json:"writes"`
	StorageGB float64 `json:"storage_gb"`
}

// ---------------------------------------------------------------------------
// Cost estimate structs — output of cost estimation
// ---------------------------------------------------------------------------

// R2CostEstimate is the cost breakdown for R2 storage.
type R2CostEstimate struct {
	StorageCost float64 `json:"storage_cost"`
	ClassACost  float64 `json:"class_a_cost"`
	ClassBCost  float64 `json:"class_b_cost"`
	TotalCost   float64 `json:"total_cost"`
	Usage       R2Usage `json:"usage"`
}

// WorkersCostEstimate is the cost breakdown for Workers.
type WorkersCostEstimate struct {
	RequestsCost    float64      `json:"requests_cost"`
	BillableRequests int64       `json:"billable_requests"`
	TotalCost       float64      `json:"total_cost"`
	Usage           WorkersUsage `json:"usage"`
}

// KVCostEstimate is the cost breakdown for KV.
type KVCostEstimate struct {
	ReadsCost   float64 `json:"reads_cost"`
	WritesCost  float64 `json:"writes_cost"`
	StorageCost float64 `json:"storage_cost"`
	TotalCost   float64 `json:"total_cost"`
	Usage       KVUsage `json:"usage"`
}

// TotalCostEstimate is the aggregate cost estimate across all services.
type TotalCostEstimate struct {
	R2               *R2CostEstimate      `json:"r2"`
	Workers          *WorkersCostEstimate  `json:"workers"`
	KV               *KVCostEstimate       `json:"kv"`
	TotalMonthlyCost float64              `json:"total_monthly_cost"`
	Period           string               `json:"period"`
	Disclaimer       string               `json:"disclaimer"`
}

// ---------------------------------------------------------------------------
// CostService
// ---------------------------------------------------------------------------

// CostService estimates monthly Cloudflare costs based on usage patterns.
type CostService struct {
	accountID string
	apiToken  string
}

// NewCostService creates a new CostService. Both accountID and apiToken are
// required for potential future API-based usage fetching.
func NewCostService(accountID, apiToken string) (*CostService, error) {
	if accountID == "" {
		return nil, validationError("CostService.New", "accountID is required")
	}
	if apiToken == "" {
		return nil, validationError("CostService.New", "apiToken is required")
	}
	return &CostService{
		accountID: accountID,
		apiToken:  apiToken,
	}, nil
}

// EstimateR2Cost calculates the estimated monthly cost for R2 storage.
func (s *CostService) EstimateR2Cost(usage R2Usage) *R2CostEstimate {
	storageCost := usage.StorageGB * R2StoragePerGB
	classACost := float64(usage.ClassAOps) / 1_000_000 * R2ClassAPerMillion
	classBCost := float64(usage.ClassBOps) / 1_000_000 * R2ClassBPerMillion

	return &R2CostEstimate{
		StorageCost: storageCost,
		ClassACost:  classACost,
		ClassBCost:  classBCost,
		TotalCost:   storageCost + classACost + classBCost,
		Usage:       usage,
	}
}

// EstimateWorkersCost calculates the estimated monthly cost for Workers.
func (s *CostService) EstimateWorkersCost(usage WorkersUsage) *WorkersCostEstimate {
	billable := usage.Requests - int64(WorkersFreeRequests)
	if billable < 0 {
		billable = 0
	}

	requestsCost := float64(billable) / 1_000_000 * WorkersRequestsPerMillion

	return &WorkersCostEstimate{
		RequestsCost:     requestsCost,
		BillableRequests: billable,
		TotalCost:        requestsCost,
		Usage:            usage,
	}
}

// EstimateKVCost calculates the estimated monthly cost for KV.
func (s *CostService) EstimateKVCost(usage KVUsage) *KVCostEstimate {
	readsCost := float64(usage.Reads) / 1_000_000 * KVReadsPerMillion
	writesCost := float64(usage.Writes) / 1_000_000 * KVWritesPerMillion
	storageCost := usage.StorageGB * KVStoragePerGB

	return &KVCostEstimate{
		ReadsCost:   readsCost,
		WritesCost:  writesCost,
		StorageCost: storageCost,
		TotalCost:   readsCost + writesCost + storageCost,
		Usage:       usage,
	}
}

// EstimateTotal calculates the aggregate estimated monthly cost across all services.
func (s *CostService) EstimateTotal(r2 R2Usage, workers WorkersUsage, kv KVUsage) *TotalCostEstimate {
	r2Est := s.EstimateR2Cost(r2)
	workersEst := s.EstimateWorkersCost(workers)
	kvEst := s.EstimateKVCost(kv)

	return &TotalCostEstimate{
		R2:               r2Est,
		Workers:          workersEst,
		KV:               kvEst,
		TotalMonthlyCost: r2Est.TotalCost + workersEst.TotalCost + kvEst.TotalCost,
		Period:           "30d",
		Disclaimer:       "Estimates are approximate and based on Cloudflare published pricing. Actual costs may vary based on your plan, contract, and usage patterns.",
	}
}
