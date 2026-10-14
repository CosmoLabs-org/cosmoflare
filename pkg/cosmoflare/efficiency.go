package cosmoflare

import (
	"context"
	"fmt"
	"time"
)

// EfficiencyFinding is one idle-resource finding: what was idle, why it
// counts as waste, the real measured evidence, and the fix.
type EfficiencyFinding struct {
	Resource string `json:"resource"`
	Check    string `json:"check"`
	Severity string `json:"severity"` // "warning" | "info"
	Evidence string `json:"evidence"`
	Fix      string `json:"fix"`
}

// EfficiencyReport is the full account efficiency audit for one window.
type EfficiencyReport struct {
	WindowStart         time.Time           `json:"window_start"`
	WindowEnd           time.Time           `json:"window_end"`
	Findings            []EfficiencyFinding `json:"findings"`
	WorkersChecked      int                 `json:"workers_checked"`
	KVNamespacesChecked int                 `json:"kv_namespaces_checked"`
	BucketsChecked      int                 `json:"buckets_checked"`
}

// EfficiencyWasteSummary counts findings per check with an honest dollar
// basis: wave 1 meters nothing beyond real usage, so idle request-priced
// resources are $0 directly and KV storage-byte attribution lands with
// wave-2 KV storage analytics.
type EfficiencyWasteSummary struct {
	IdleWorkers         int     `json:"idle_workers"`
	DeadKVNamespaces    int     `json:"dead_kv_namespaces"`
	EmptyR2Buckets      int     `json:"empty_r2_buckets"`
	EstimatedMonthlyUSD float64 `json:"estimated_monthly_usd"`
	Basis               string  `json:"basis"`
}

// EfficiencyService audits the account for idle resources by composing the
// Workers, KV, and Analytics services plus one new GraphQL dataset
// (kvOperationsAdaptiveGroups). It reports what is idle, with real measured
// evidence — never modeled numbers.
type EfficiencyService struct {
	workers   *WorkerService
	kv        *KVService
	analytics *AnalyticsService
	buckets   BucketLister // nil → empty-bucket checks are skipped
}

// NewEfficiencyService creates an EfficiencyService without R2 bucket
// enumeration: empty-bucket checks are skipped and BucketsChecked stays 0.
func NewEfficiencyService(workers *WorkerService, kv *KVService, analytics *AnalyticsService) *EfficiencyService {
	return &EfficiencyService{workers: workers, kv: kv, analytics: analytics}
}

// NewEfficiencyServiceWithBuckets creates an EfficiencyService that also
// enumerates R2 buckets for the empty-bucket check. BucketLister is the
// shared narrow consumer interface from limits.go — any service satisfying
// it works, including test fakes.
func NewEfficiencyServiceWithBuckets(workers *WorkerService, kv *KVService, analytics *AnalyticsService, buckets BucketLister) *EfficiencyService {
	return &EfficiencyService{workers: workers, kv: kv, analytics: analytics, buckets: buckets}
}

// NewEfficiencyServiceFromCreds creates an EfficiencyService from account ID
// and API token, wiring every backing service from the same credentials.
// Empty R2 checks need an R2 client — construct with
// NewEfficiencyServiceWithBuckets for the full audit.
func NewEfficiencyServiceFromCreds(accountID, apiToken string) (*EfficiencyService, error) {
	if accountID == "" {
		return nil, validationError("NewEfficiencyService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewEfficiencyService", "API token is required")
	}
	workers, err := NewWorkerServiceFromCreds(accountID, apiToken)
	if err != nil {
		return nil, err
	}
	kv, err := NewKVServiceFromCreds(accountID, apiToken)
	if err != nil {
		return nil, err
	}
	return &EfficiencyService{
		workers:   workers,
		kv:        kv,
		analytics: NewAnalyticsService(accountID, apiToken),
	}, nil
}

// Report audits the account for idle resources over the window.
//
// Idle = absent or zero: a worker/namespace/bucket that never appears in
// the analytics response counts as zero usage — absence is never an error.
// Empty R2 detection uses the batched R2Storage analytics, never per-bucket
// object listing (one GraphQL query per dataset per run, no N+1).
func (s *EfficiencyService) Report(ctx context.Context, w AnalyticsWindow) (*EfficiencyReport, error) {
	rep := &EfficiencyReport{WindowStart: w.Start, WindowEnd: w.End}

	// Workers: listed scripts × per-script request sums. Absent from the
	// analytics rows or Requests==0 → idle.
	if s.workers != nil {
		scripts, err := s.workers.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list workers: %w", err)
		}
		sums, err := s.analytics.Workers(ctx, w)
		if err != nil {
			return nil, fmt.Errorf("workers analytics: %w", err)
		}
		byScript := map[string]uint64{}
		for _, sm := range sums {
			byScript[sm.Script] = sm.Requests
		}
		rep.WorkersChecked = len(scripts)
		for _, sc := range scripts {
			if byScript[sc.Name] == 0 {
				rep.Findings = append(rep.Findings, EfficiencyFinding{
					Resource: "worker:" + sc.Name,
					Check:    "idle-worker",
					Severity: "warning",
					Evidence: fmt.Sprintf("0 requests between %s and %s", w.Start.Format("2006-01-02"), w.End.Format("2006-01-02")),
					Fix:      fmt.Sprintf("delete it (cosmoflare worker delete %s) or remove its routes if retired", sc.Name),
				})
			}
		}
	}

	// KV: same absent-or-zero rule over reads+writes.
	if s.kv != nil {
		namespaces, err := s.kv.ListNamespaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("list kv namespaces: %w", err)
		}
		ops, err := s.analytics.KVOperations(ctx, w)
		if err != nil {
			return nil, fmt.Errorf("kv analytics: %w", err)
		}
		byNS := map[string]uint64{}
		for _, op := range ops {
			byNS[op.NamespaceID] = op.Requests
		}
		rep.KVNamespacesChecked = len(namespaces)
		for _, ns := range namespaces {
			if byNS[ns.ID] == 0 {
				rep.Findings = append(rep.Findings, EfficiencyFinding{
					Resource: "kv:" + ns.ID,
					Check:    "dead-kv-namespace",
					Severity: "warning",
					Evidence: fmt.Sprintf("0 KV operations between %s and %s", w.Start.Format("2006-01-02"), w.End.Format("2006-01-02")),
					Fix:      fmt.Sprintf("export anything worth keeping, then delete it (cosmoflare kv namespace delete %s) — keys survive until deleted", ns.ID),
				})
				continue
			}
		}
	}

	// R2: buckets (when a BucketLister is wired) absent from R2Storage rows
	// or with ObjectCount==0 → empty.
	if s.buckets != nil && s.analytics != nil {
		buckets, err := s.buckets.ListBuckets(ctx)
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		storage, err := s.analytics.R2Storage(ctx, w)
		if err != nil {
			return nil, fmt.Errorf("r2 analytics: %w", err)
		}
		byBucket := map[string]uint64{}
		for _, st := range storage {
			byBucket[st.Bucket] = st.ObjectCount
		}
		rep.BucketsChecked = len(buckets)
		for _, b := range buckets {
			if byBucket[b.Name] == 0 {
				rep.Findings = append(rep.Findings, EfficiencyFinding{
					Resource: "r2:" + b.Name,
					Check:    "empty-r2-bucket",
					Severity: "info",
					Evidence: fmt.Sprintf("0 objects at %s", w.End.Format("2006-01-02")),
					Fix:      fmt.Sprintf("delete it (cosmoflare bucket delete %s) — empty buckets cost nothing but clutter audits", b.Name),
				})
			}
		}
	}

	return rep, nil
}

// SummarizeWaste reduces a report to per-check counts with an honest dollar
// basis. EstimatedMonthlyUSD is 0.00 in wave 1: idle workers and empty
// buckets meter at $0 directly (requests/storage are the meter and both are
// zero) and per-namespace KV storage attribution needs wave-2 KV storage
// analytics — no guessed dollar figures, ever.
func SummarizeWaste(r *EfficiencyReport) *EfficiencyWasteSummary {
	s := &EfficiencyWasteSummary{
		Basis: "Idle request-priced resources meter $0 directly; per-namespace KV storage attribution lands with wave-2 KV storage analytics.",
	}
	for _, f := range r.Findings {
		switch f.Check {
		case "idle-worker":
			s.IdleWorkers++
		case "dead-kv-namespace":
			s.DeadKVNamespaces++
		case "empty-r2-bucket":
			s.EmptyR2Buckets++
		}
	}
	return s
}
