package cosmoflare

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// ApplyAction describes what operation will be performed on a resource.
type ApplyAction string

const (
	ApplyCreate ApplyAction = "create"
	ApplyDelete ApplyAction = "delete"
	ApplyUpdate ApplyAction = "update"
	ApplySkip   ApplyAction = "skip"
)

// ApplyStatus describes the outcome of an apply operation.
type ApplyStatus string

const (
	ApplyStatusSuccess ApplyStatus = "success"
	ApplyStatusFailed  ApplyStatus = "failed"
	ApplyStatusSkipped ApplyStatus = "skipped"
)

// ApplyOperation represents a single resource mutation during apply.
type ApplyOperation struct {
	Action   ApplyAction `json:"action"`
	Service  string      `json:"service"`
	Resource string      `json:"resource"`
	Status   ApplyStatus `json:"status"`
	Detail   string      `json:"detail,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// ApplyResult holds the outcome of applying changes for a single service.
type ApplyResult struct {
	Service    string           `json:"service"`
	Operations []ApplyOperation `json:"operations"`
}

// HasErrors returns true if any operation failed.
func (r *ApplyResult) HasErrors() bool {
	for _, op := range r.Operations {
		if op.Status == ApplyStatusFailed {
			return true
		}
	}
	return false
}

// ApplyResultSummary provides counts for a single service apply result.
type ApplyResultSummary struct {
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Total     int `json:"total"`
}

// Summary returns counts of succeeded, failed, and skipped operations.
func (r *ApplyResult) Summary() ApplyResultSummary {
	s := ApplyResultSummary{Total: len(r.Operations)}
	for _, op := range r.Operations {
		switch op.Status {
		case ApplyStatusSuccess:
			s.Succeeded++
		case ApplyStatusFailed:
			s.Failed++
		case ApplyStatusSkipped:
			s.Skipped++
		}
	}
	return s
}

// ApplySummary provides an aggregate view across all services.
type ApplySummary struct {
	Results        []ApplyResult `json:"results"`
	TotalSucceeded int           `json:"total_succeeded"`
	TotalFailed    int           `json:"total_failed"`
	TotalSkipped   int           `json:"total_skipped"`
	HasErrors      bool          `json:"has_errors"`
}

// Aggregate computes totals from the per-service results.
func (s *ApplySummary) Aggregate() {
	s.TotalSucceeded = 0
	s.TotalFailed = 0
	s.TotalSkipped = 0
	s.HasErrors = false
	for _, r := range s.Results {
		sum := r.Summary()
		s.TotalSucceeded += sum.Succeeded
		s.TotalFailed += sum.Failed
		s.TotalSkipped += sum.Skipped
		if r.HasErrors() {
			s.HasErrors = true
		}
	}
}

// DiffToApplyOperations converts a DiffResult into a list of pending
// ApplyOperations. Each addition becomes a create, each deletion becomes
// a delete, and each change becomes an update. Status is left empty
// (to be filled after execution).
func DiffToApplyOperations(diff *DiffResult) []ApplyOperation {
	ops := make([]ApplyOperation, 0, diff.TotalChanges())

	for _, e := range diff.Additions {
		ops = append(ops, ApplyOperation{
			Action:   ApplyCreate,
			Service:  e.Service,
			Resource: e.Resource,
			Detail:   e.Detail,
		})
	}
	for _, e := range diff.Deletions {
		ops = append(ops, ApplyOperation{
			Action:   ApplyDelete,
			Service:  e.Service,
			Resource: e.Resource,
			Detail:   e.Detail,
		})
	}
	for _, e := range diff.Changes {
		ops = append(ops, ApplyOperation{
			Action:   ApplyUpdate,
			Service:  e.Service,
			Resource: e.Resource,
			Detail:   e.Detail,
		})
	}

	return ops
}

// ApplyService reconciles local .cosmoflare.yaml config against live
// Cloudflare state by computing a diff and executing the required changes.
type ApplyService struct {
	accountID string
	apiToken  string
	diff      *DiffService
	// deleteUnmanaged gates delete-by-omission (BUG-049): resources that
	// exist live but are absent from the config are only deleted when the
	// operator explicitly opted in. Defaults to false — cosmoflare has no
	// managed-resource tracking yet, so an unmanaged live resource must
	// never be destroyed just because the config does not mention it.
	deleteUnmanaged bool
}

// WithDeleteUnmanaged enables deletion of unmanaged resources (those present
// live but absent from the config) during apply. Without it, delete ops
// derived from diff omissions are skipped with an explanatory note.
func WithDeleteUnmanaged(enable bool) ApplyServiceOption {
	return func(a *ApplyService) { a.deleteUnmanaged = enable }
}

// gateUnmanagedDeletes rewrites delete ops to skipped when the operator has
// not opted into deleting unmanaged resources (BUG-049). Non-delete ops pass
// through untouched. Called before the dry-run preview so the preview shows
// the same skips a real run would perform.
func (a *ApplyService) gateUnmanagedDeletes(ops []ApplyOperation) []ApplyOperation {
	if a.deleteUnmanaged {
		return ops
	}
	for i := range ops {
		if ops[i].Action != ApplyDelete {
			continue
		}
		ops[i].Status = ApplyStatusSkipped
		ops[i].Detail = fmt.Sprintf(
			"resource %q exists but is not in config (unmanaged) — pass --delete-unmanaged to delete it",
			ops[i].Resource)
	}
	return ops
}

// ApplyServiceOption configures an ApplyService at construction time.
type ApplyServiceOption func(*ApplyService)

// NewApplyService creates a new ApplyService.
func NewApplyService(accountID, apiToken string, opts ...ApplyServiceOption) (*ApplyService, error) {
	if accountID == "" {
		return nil, validationError("NewApplyService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewApplyService", "API token is required")
	}

	diffSvc, err := NewDiffService(accountID, apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff service: %w", err)
	}

	a := &ApplyService{
		accountID: accountID,
		apiToken:  apiToken,
		diff:      diffSvc,
	}
	for _, o := range opts {
		o(a)
	}
	return a, nil
}

// ApplyAll computes diffs for all configured services and applies changes.
func (a *ApplyService) ApplyAll(ctx context.Context, cfg *CosmoflareConfig, dryRun bool) (*ApplySummary, error) {
	diffSummary, err := a.diff.CompareAll(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	summary := &ApplySummary{}

	for _, dr := range diffSummary.Results {
		if !dr.HasChanges() {
			continue
		}
		var result *ApplyResult
		var applyErr error

		switch dr.Service {
		case "workers":
			result, applyErr = a.applyWorkerChanges(ctx, cfg, &dr, dryRun)
		case "r2":
			result, applyErr = a.applyR2Changes(ctx, cfg, &dr, dryRun)
		case "kv":
			result, applyErr = a.applyKVChanges(ctx, cfg, &dr, dryRun)
		case "dns":
			result, applyErr = a.applyDNSChanges(ctx, cfg, &dr, dryRun)
		default:
			continue
		}

		if applyErr != nil {
			return nil, fmt.Errorf("failed to apply %s changes: %w", dr.Service, applyErr)
		}
		if result != nil {
			summary.Results = append(summary.Results, *result)
		}
	}

	summary.Aggregate()
	return summary, nil
}

// ApplyWorkers computes the worker diff and applies changes.
func (a *ApplyService) ApplyWorkers(ctx context.Context, cfg *CosmoflareConfig, dryRun bool) (*ApplyResult, error) {
	if len(cfg.Workers) == 0 {
		return &ApplyResult{Service: "workers"}, nil
	}

	dr, err := a.diff.CompareWorkers(ctx, cfg.Workers)
	if err != nil {
		return nil, fmt.Errorf("failed to compute workers diff: %w", err)
	}

	if !dr.HasChanges() {
		return &ApplyResult{Service: "workers"}, nil
	}

	return a.applyWorkerChanges(ctx, cfg, dr, dryRun)
}

// ApplyR2 computes the R2 diff and applies changes.
func (a *ApplyService) ApplyR2(ctx context.Context, cfg *CosmoflareConfig, dryRun bool) (*ApplyResult, error) {
	if len(cfg.R2.Buckets) == 0 {
		return &ApplyResult{Service: "r2"}, nil
	}

	dr, err := a.diff.CompareR2(ctx, cfg.R2)
	if err != nil {
		return nil, fmt.Errorf("failed to compute r2 diff: %w", err)
	}

	if !dr.HasChanges() {
		return &ApplyResult{Service: "r2"}, nil
	}

	return a.applyR2Changes(ctx, cfg, dr, dryRun)
}

// ApplyKV computes the KV diff and applies changes.
func (a *ApplyService) ApplyKV(ctx context.Context, cfg *CosmoflareConfig, dryRun bool) (*ApplyResult, error) {
	if len(cfg.KV.Namespaces) == 0 {
		return &ApplyResult{Service: "kv"}, nil
	}

	dr, err := a.diff.CompareKV(ctx, cfg.KV)
	if err != nil {
		return nil, fmt.Errorf("failed to compute kv diff: %w", err)
	}

	if !dr.HasChanges() {
		return &ApplyResult{Service: "kv"}, nil
	}

	return a.applyKVChanges(ctx, cfg, dr, dryRun)
}

// ApplyDNS computes the DNS diff and applies changes.
func (a *ApplyService) ApplyDNS(ctx context.Context, cfg *CosmoflareConfig, dryRun bool) (*ApplyResult, error) {
	if cfg.DNS.ZoneID == "" {
		return nil, validationError("ApplyService.ApplyDNS", "zone_id is required in dns config")
	}

	dr, err := a.diff.CompareDNS(ctx, cfg.DNS)
	if err != nil {
		return nil, fmt.Errorf("failed to compute dns diff: %w", err)
	}

	if !dr.HasChanges() {
		return &ApplyResult{Service: "dns"}, nil
	}

	return a.applyDNSChanges(ctx, cfg, dr, dryRun)
}

// --- Per-service apply implementations ---

func (a *ApplyService) applyWorkerChanges(ctx context.Context, cfg *CosmoflareConfig, dr *DiffResult, dryRun bool) (*ApplyResult, error) {
	ops := DiffToApplyOperations(dr)
	// BUG-049: gate delete-by-omission behind the explicit opt-in — before
	// the dry-run branch so previews show the same skips a real run makes.
	ops = a.gateUnmanagedDeletes(ops)
	result := &ApplyResult{Service: "workers", Operations: make([]ApplyOperation, 0, len(ops))}

	if dryRun {
		for _, op := range ops {
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("[dry-run] would %s %s", op.Action, op.Resource)
			result.Operations = append(result.Operations, op)
		}
		return result, nil
	}

	ws, err := NewWorkerServiceFromCreds(a.accountID, a.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}

	for _, op := range ops {
		switch op.Action {
		case ApplyCreate:
			wc, exists := cfg.Workers[op.Resource]
			if !exists {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("worker %q not found in config", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			scriptReader, err := openWorkerScript(wc.Script)
			if err != nil {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("failed to read script %q: %v", wc.Script, err)
				result.Operations = append(result.Operations, op)
				continue
			}
			var wopts []WorkerOption
			if wc.CompatibilityDate != "" {
				wopts = append(wopts, WithWorkerCompatibilityDate(wc.CompatibilityDate))
			}
			if wc.Module {
				wopts = append(wopts, WithWorkerModule(true))
			}
			_, deployErr := ws.Deploy(ctx, op.Resource, scriptReader, wopts...)
			scriptReader.Close()
			if deployErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = deployErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("deployed worker %q", op.Resource)
			}

		case ApplyDelete:
			if delErr := ws.Delete(ctx, op.Resource); delErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = delErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("deleted worker %q", op.Resource)
			}

		case ApplyUpdate:
			// For workers, update means re-deploy with new config
			wc, exists := cfg.Workers[op.Resource]
			if !exists {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("worker %q not found in config for update", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			scriptReader, err := openWorkerScript(wc.Script)
			if err != nil {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("failed to read script %q: %v", wc.Script, err)
				result.Operations = append(result.Operations, op)
				continue
			}
			var wopts []WorkerOption
			if wc.CompatibilityDate != "" {
				wopts = append(wopts, WithWorkerCompatibilityDate(wc.CompatibilityDate))
			}
			if wc.Module {
				wopts = append(wopts, WithWorkerModule(true))
			}
			_, deployErr := ws.Deploy(ctx, op.Resource, scriptReader, wopts...)
			scriptReader.Close()
			if deployErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = deployErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("updated worker %q", op.Resource)
			}

		default:
			op.Status = ApplyStatusSkipped
		}

		result.Operations = append(result.Operations, op)
	}

	return result, nil
}

func (a *ApplyService) applyR2Changes(ctx context.Context, _ *CosmoflareConfig, dr *DiffResult, dryRun bool) (*ApplyResult, error) {
	ops := DiffToApplyOperations(dr)
	// BUG-049: gate delete-by-omission behind the explicit opt-in — before
	// the dry-run branch so previews show the same skips a real run makes.
	ops = a.gateUnmanagedDeletes(ops)
	result := &ApplyResult{Service: "r2", Operations: make([]ApplyOperation, 0, len(ops))}

	if dryRun {
		for _, op := range ops {
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("[dry-run] would %s bucket %s", op.Action, op.Resource)
			result.Operations = append(result.Operations, op)
		}
		return result, nil
	}

	r2Client, err := NewClient(
		WithAccountID(a.accountID),
		WithAPIToken(a.apiToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 client: %w", err)
	}

	for _, op := range ops {
		switch op.Action {
		case ApplyCreate:
			_, createErr := r2Client.CreateBucket(ctx, op.Resource)
			if createErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = createErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("created bucket %q", op.Resource)
			}

		case ApplyDelete:
			if delErr := r2Client.DeleteBucket(ctx, op.Resource); delErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = delErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("deleted bucket %q", op.Resource)
			}

		default:
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("bucket %q: update not supported for R2 buckets", op.Resource)
		}

		result.Operations = append(result.Operations, op)
	}

	return result, nil
}

func (a *ApplyService) applyKVChanges(ctx context.Context, _ *CosmoflareConfig, dr *DiffResult, dryRun bool) (*ApplyResult, error) {
	ops := DiffToApplyOperations(dr)
	// BUG-049: gate delete-by-omission behind the explicit opt-in — before
	// the dry-run branch so previews show the same skips a real run makes.
	ops = a.gateUnmanagedDeletes(ops)
	result := &ApplyResult{Service: "kv", Operations: make([]ApplyOperation, 0, len(ops))}

	if dryRun {
		for _, op := range ops {
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("[dry-run] would %s namespace %s", op.Action, op.Resource)
			result.Operations = append(result.Operations, op)
		}
		return result, nil
	}

	kvSvc, err := NewKVServiceFromCreds(a.accountID, a.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV service: %w", err)
	}

	for _, op := range ops {
		switch op.Action {
		case ApplyCreate:
			_, createErr := kvSvc.CreateNamespace(ctx, op.Resource)
			if createErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = createErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("created namespace %q", op.Resource)
			}

		case ApplyDelete:
			// To delete a namespace we need its ID. List and find by title.
			namespaces, listErr := kvSvc.ListNamespaces(ctx)
			if listErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("failed to list namespaces to find ID: %v", listErr)
				result.Operations = append(result.Operations, op)
				continue
			}
			// BUG-048: a title collision makes first-match resolution delete
			// the WRONG namespace — refuse ambiguity instead of guessing.
			idx, idxErr := indexKVByTitle(namespaces)
			if idxErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = idxErr.Error()
				result.Operations = append(result.Operations, op)
				continue
			}
			nsID := idx[op.Resource]
			if nsID == "" {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("namespace %q not found for deletion", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			if delErr := kvSvc.DeleteNamespace(ctx, nsID); delErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = delErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("deleted namespace %q", op.Resource)
			}

		default:
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("namespace %q: update not supported for KV namespaces", op.Resource)
		}

		result.Operations = append(result.Operations, op)
	}

	return result, nil
}

func (a *ApplyService) applyDNSChanges(ctx context.Context, cfg *CosmoflareConfig, dr *DiffResult, dryRun bool) (*ApplyResult, error) {
	ops := DiffToApplyOperations(dr)
	// BUG-049: gate delete-by-omission behind the explicit opt-in — before
	// the dry-run branch so previews show the same skips a real run makes.
	ops = a.gateUnmanagedDeletes(ops)
	result := &ApplyResult{Service: "dns", Operations: make([]ApplyOperation, 0, len(ops))}

	if dryRun {
		for _, op := range ops {
			op.Status = ApplyStatusSkipped
			op.Detail = fmt.Sprintf("[dry-run] would %s record %s", op.Action, op.Resource)
			result.Operations = append(result.Operations, op)
		}
		return result, nil
	}

	dnsSvc, err := NewDNSServiceFromCreds(cfg.DNS.ZoneID, a.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS service: %w", err)
	}

	// Build lookup of live records for update/delete (need IDs)
	liveRecords, err := dnsSvc.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list live DNS records: %w", err)
	}

	liveMap := make(map[applyDNSKey]*DNSRecord, len(liveRecords))
	for _, r := range liveRecords {
		k := applyDNSKey{Type: r.Type, Name: r.Name, Content: r.Content}
		liveMap[k] = r
	}

	// Build config record lookup for create operations
	configMap := make(map[applyDNSKey]DNSRecordConfig, len(cfg.DNS.Records))
	for _, r := range cfg.DNS.Records {
		k := applyDNSKey{Type: r.Type, Name: r.Name, Content: r.Content}
		configMap[k] = r
	}

	for _, op := range ops {
		switch op.Action {
		case ApplyCreate:
			// Parse resource back to type+name+content
			rec, found := findConfigRecordFromResource(op.Resource, configMap)
			if !found {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("record %q not found in config", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			var dopts []DNSOption
			if rec.TTL != 0 {
				dopts = append(dopts, WithDNSTTL(rec.TTL))
			}
			dopts = append(dopts, WithDNSProxied(rec.Proxied))

			_, createErr := dnsSvc.Create(ctx, rec.Type, rec.Name, rec.Content, dopts...)
			if createErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = createErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("created %s record %q -> %q", rec.Type, rec.Name, rec.Content)
			}

		case ApplyDelete:
			liveRec := findLiveRecordFromResource(op.Resource, liveMap)
			if liveRec == nil {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("record %q not found in live state for deletion", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			if delErr := dnsSvc.Delete(ctx, liveRec.ID); delErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = delErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("deleted %s record %q", liveRec.Type, liveRec.Name)
			}

		case ApplyUpdate:
			liveRec := findLiveRecordFromResource(op.Resource, liveMap)
			if liveRec == nil {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("record %q not found in live state for update", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			rec, found := findConfigRecordFromResource(op.Resource, configMap)
			if !found {
				op.Status = ApplyStatusFailed
				op.Error = fmt.Sprintf("record %q not found in config for update", op.Resource)
				result.Operations = append(result.Operations, op)
				continue
			}
			var dopts []DNSOption
			if rec.TTL != 0 {
				dopts = append(dopts, WithDNSTTL(rec.TTL))
			}
			dopts = append(dopts, WithDNSProxied(rec.Proxied))

			_, updateErr := dnsSvc.Update(ctx, liveRec.ID, dopts...)
			if updateErr != nil {
				op.Status = ApplyStatusFailed
				op.Error = updateErr.Error()
			} else {
				op.Status = ApplyStatusSuccess
				op.Detail = fmt.Sprintf("updated %s record %q", rec.Type, rec.Name)
			}

		default:
			op.Status = ApplyStatusSkipped
		}

		result.Operations = append(result.Operations, op)
	}

	return result, nil
}

// openWorkerScript opens a worker script file for reading.
func openWorkerScript(path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("script path is empty")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open script %q: %w", path, err)
	}
	return f, nil
}

// applyDNSKey uniquely identifies a DNS record for apply lookups.
type applyDNSKey struct {
	Type    string
	Name    string
	Content string
}

// parseResourceToDNSKey parses a DNS resource string (format: "TYPE NAME CONTENT")
// into an applyDNSKey.
func parseResourceToDNSKey(resource string) (applyDNSKey, bool) {
	parts := strings.SplitN(resource, " ", 3)
	if len(parts) < 3 {
		return applyDNSKey{}, false
	}
	return applyDNSKey{Type: parts[0], Name: parts[1], Content: parts[2]}, true
}

// findConfigRecordFromResource parses a DNS resource string and looks it up
// in the config map.
func findConfigRecordFromResource(resource string, configMap map[applyDNSKey]DNSRecordConfig) (DNSRecordConfig, bool) {
	k, ok := parseResourceToDNSKey(resource)
	if !ok {
		return DNSRecordConfig{}, false
	}
	rec, found := configMap[k]
	return rec, found
}

// findLiveRecordFromResource parses a DNS resource string and looks it up
// in the live records map.
func findLiveRecordFromResource(resource string, liveMap map[applyDNSKey]*DNSRecord) *DNSRecord {
	k, ok := parseResourceToDNSKey(resource)
	if !ok {
		return nil
	}
	return liveMap[k]
}
