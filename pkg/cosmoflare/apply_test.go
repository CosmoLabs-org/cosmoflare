package cosmoflare

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Constructor validation ---

func TestNewApplyServiceValidation(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		apiToken  string
		wantErr   bool
	}{
		{"both empty", "", "", true},
		{"empty account ID", "", "tok-123", true},
		{"empty API token", "acc-123", "", true},
		{"valid", "acc-123", "tok-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewApplyService(tt.accountID, tt.apiToken)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if svc != nil {
					t.Error("expected nil service on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if svc == nil {
					t.Error("expected non-nil service")
				}
			}
		})
	}
}

func TestNewApplyServiceFields(t *testing.T) {
	svc, err := NewApplyService("acc-xyz", "tok-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acc-xyz" {
		t.Errorf("accountID = %q, want %q", svc.accountID, "acc-xyz")
	}
	if svc.apiToken != "tok-abc" {
		t.Errorf("apiToken = %q, want %q", svc.apiToken, "tok-abc")
	}
}

// --- ApplyAction constants ---

func TestApplyActionConstants(t *testing.T) {
	if ApplyCreate != "create" {
		t.Errorf("ApplyCreate = %q, want %q", ApplyCreate, "create")
	}
	if ApplyDelete != "delete" {
		t.Errorf("ApplyDelete = %q, want %q", ApplyDelete, "delete")
	}
	if ApplyUpdate != "update" {
		t.Errorf("ApplyUpdate = %q, want %q", ApplyUpdate, "update")
	}
	if ApplySkip != "skip" {
		t.Errorf("ApplySkip = %q, want %q", ApplySkip, "skip")
	}
}

// --- ApplyResult methods ---

func TestApplyResult_HasErrors(t *testing.T) {
	noErrors := &ApplyResult{
		Operations: []ApplyOperation{
			{Action: ApplyCreate, Status: ApplyStatusSuccess},
		},
	}
	if noErrors.HasErrors() {
		t.Error("expected no errors")
	}

	withErrors := &ApplyResult{
		Operations: []ApplyOperation{
			{Action: ApplyCreate, Status: ApplyStatusSuccess},
			{Action: ApplyDelete, Status: ApplyStatusFailed, Error: "something broke"},
		},
	}
	if !withErrors.HasErrors() {
		t.Error("expected errors")
	}
}

func TestApplyResult_Summary(t *testing.T) {
	result := &ApplyResult{
		Service: "workers",
		Operations: []ApplyOperation{
			{Action: ApplyCreate, Status: ApplyStatusSuccess},
			{Action: ApplyCreate, Status: ApplyStatusSuccess},
			{Action: ApplyDelete, Status: ApplyStatusFailed, Error: "fail"},
			{Action: ApplyUpdate, Status: ApplyStatusSkipped},
		},
	}
	s := result.Summary()
	if s.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", s.Succeeded)
	}
	if s.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.Failed)
	}
	if s.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", s.Skipped)
	}
	if s.Total != 4 {
		t.Errorf("Total = %d, want 4", s.Total)
	}
}

func TestApplyResult_SummaryEmpty(t *testing.T) {
	result := &ApplyResult{Service: "kv"}
	s := result.Summary()
	if s.Total != 0 {
		t.Errorf("Total = %d, want 0", s.Total)
	}
	if s.Succeeded != 0 {
		t.Errorf("Succeeded = %d, want 0", s.Succeeded)
	}
}

// --- ApplyOperation fields ---

func TestApplyOperation_Fields(t *testing.T) {
	op := ApplyOperation{
		Action:   ApplyCreate,
		Service:  "workers",
		Resource: "my-worker",
		Status:   ApplyStatusSuccess,
		Detail:   "deployed worker my-worker",
	}
	if op.Action != ApplyCreate {
		t.Errorf("Action = %q, want %q", op.Action, ApplyCreate)
	}
	if op.Service != "workers" {
		t.Errorf("Service = %q, want %q", op.Service, "workers")
	}
	if op.Resource != "my-worker" {
		t.Errorf("Resource = %q, want %q", op.Resource, "my-worker")
	}
	if op.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want %q", op.Status, ApplyStatusSuccess)
	}
	if op.Detail != "deployed worker my-worker" {
		t.Errorf("Detail = %q, want %q", op.Detail, "deployed worker my-worker")
	}
}

func TestApplyOperation_WithError(t *testing.T) {
	op := ApplyOperation{
		Action:   ApplyDelete,
		Service:  "dns",
		Resource: "A example.com",
		Status:   ApplyStatusFailed,
		Error:    "API returned 403: forbidden",
	}
	if op.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want %q", op.Status, ApplyStatusFailed)
	}
	if op.Error == "" {
		t.Error("Error should not be empty for failed operation")
	}
}

// --- ApplyStatus constants ---

func TestApplyStatusConstants(t *testing.T) {
	if ApplyStatusSuccess != "success" {
		t.Errorf("ApplyStatusSuccess = %q, want %q", ApplyStatusSuccess, "success")
	}
	if ApplyStatusFailed != "failed" {
		t.Errorf("ApplyStatusFailed = %q, want %q", ApplyStatusFailed, "failed")
	}
	if ApplyStatusSkipped != "skipped" {
		t.Errorf("ApplyStatusSkipped = %q, want %q", ApplyStatusSkipped, "skipped")
	}
}

// --- ApplySummary aggregation ---

func TestApplySummary_Aggregation(t *testing.T) {
	summary := &ApplySummary{
		Results: []ApplyResult{
			{
				Service: "workers",
				Operations: []ApplyOperation{
					{Action: ApplyCreate, Status: ApplyStatusSuccess},
				},
			},
			{
				Service: "kv",
				Operations: []ApplyOperation{
					{Action: ApplyCreate, Status: ApplyStatusSuccess},
					{Action: ApplyCreate, Status: ApplyStatusFailed, Error: "fail"},
				},
			},
		},
	}
	summary.Aggregate()

	if summary.TotalSucceeded != 2 {
		t.Errorf("TotalSucceeded = %d, want 2", summary.TotalSucceeded)
	}
	if summary.TotalFailed != 1 {
		t.Errorf("TotalFailed = %d, want 1", summary.TotalFailed)
	}
	if !summary.HasErrors {
		t.Error("expected HasErrors to be true")
	}
}

func TestApplySummary_NoErrors(t *testing.T) {
	summary := &ApplySummary{
		Results: []ApplyResult{
			{
				Service: "workers",
				Operations: []ApplyOperation{
					{Action: ApplyCreate, Status: ApplyStatusSuccess},
				},
			},
		},
	}
	summary.Aggregate()

	if summary.TotalSucceeded != 1 {
		t.Errorf("TotalSucceeded = %d, want 1", summary.TotalSucceeded)
	}
	if summary.TotalFailed != 0 {
		t.Errorf("TotalFailed = %d, want 0", summary.TotalFailed)
	}
	if summary.HasErrors {
		t.Error("expected HasErrors to be false")
	}
}

// --- DiffToApplyOperations ---

func TestDiffToApplyOperations(t *testing.T) {
	diff := &DiffResult{
		Service: "workers",
		Additions: []DiffEntry{
			{Action: DiffAdd, Service: "workers", Resource: "api-worker", Detail: "new worker"},
		},
		Deletions: []DiffEntry{
			{Action: DiffRemove, Service: "workers", Resource: "old-worker", Detail: "not in config"},
		},
		Changes: []DiffEntry{
			{Action: DiffModify, Service: "workers", Resource: "changed-worker", Detail: "updated settings"},
		},
	}

	ops := DiffToApplyOperations(diff)
	if len(ops) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(ops))
	}

	// Check addition maps to create
	if ops[0].Action != ApplyCreate {
		t.Errorf("ops[0].Action = %q, want %q", ops[0].Action, ApplyCreate)
	}
	if ops[0].Resource != "api-worker" {
		t.Errorf("ops[0].Resource = %q, want %q", ops[0].Resource, "api-worker")
	}

	// Check deletion maps to delete
	if ops[1].Action != ApplyDelete {
		t.Errorf("ops[1].Action = %q, want %q", ops[1].Action, ApplyDelete)
	}
	if ops[1].Resource != "old-worker" {
		t.Errorf("ops[1].Resource = %q, want %q", ops[1].Resource, "old-worker")
	}

	// Check modification maps to update
	if ops[2].Action != ApplyUpdate {
		t.Errorf("ops[2].Action = %q, want %q", ops[2].Action, ApplyUpdate)
	}
	if ops[2].Resource != "changed-worker" {
		t.Errorf("ops[2].Resource = %q, want %q", ops[2].Resource, "changed-worker")
	}
}

func TestDiffToApplyOperations_Empty(t *testing.T) {
	diff := &DiffResult{Service: "workers"}
	ops := DiffToApplyOperations(diff)
	if len(ops) != 0 {
		t.Errorf("expected 0 operations, got %d", len(ops))
	}
}

// --- ApplyService has DiffService ---

func TestApplyServiceContainsDiffService(t *testing.T) {
	svc, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.diff == nil {
		t.Error("expected ApplyService to contain a DiffService")
	}
}

// --- BUG-049: apply must not delete resources it did not create by omission ---

func TestGateUnmanagedDeletes_DefaultSkipsDeletes(t *testing.T) {
	a := &ApplyService{} // deleteUnmanaged defaults false — NOT opted in
	ops := []ApplyOperation{
		{Action: ApplyDelete, Resource: "stray-bucket"},
		{Action: ApplyCreate, Resource: "new-bucket"},
	}

	gated := a.gateUnmanagedDeletes(ops)

	if gated[0].Status != ApplyStatusSkipped {
		t.Errorf("unmanaged delete must be skipped without --delete-unmanaged, got status %q", gated[0].Status)
	}
	if !strings.Contains(gated[0].Detail, "--delete-unmanaged") {
		t.Errorf("skip detail must tell the user how to opt in, got: %q", gated[0].Detail)
	}
	if gated[1].Status == ApplyStatusSkipped {
		t.Error("create ops must not be gated")
	}
}

func TestGateUnmanagedDeletes_OptInPreservesDeletes(t *testing.T) {
	a := &ApplyService{deleteUnmanaged: true}
	ops := []ApplyOperation{
		{Action: ApplyDelete, Resource: "stray-bucket"},
	}

	gated := a.gateUnmanagedDeletes(ops)

	if gated[0].Status == ApplyStatusSkipped {
		t.Error("opted-in deletes must be preserved")
	}
}

// --- ApplyAll / ApplyWorkers / ApplyR2 / ApplyKV / ApplyDNS orchestration ---
//
// These entry points always compute a live diff via the internal DiffService,
// which is not injectable with a mock server. Empty-config inputs exercise
// the early-return branches without touching the network. An already-canceled
// context makes the underlying Cloudflare/AWS SDK calls fail immediately
// (no real network I/O — the transport checks ctx.Err() before dialing),
// which deterministically exercises the error-propagation branches.

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestApplyAll_EmptyConfig(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary, err := a.ApplyAll(context.Background(), &CosmoflareConfig{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if len(summary.Results) != 0 {
		t.Errorf("expected 0 results for empty config, got %d", len(summary.Results))
	}
	if summary.HasErrors {
		t.Error("expected HasErrors=false for empty config")
	}
}

func TestApplyAll_DiffErrorPropagates(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"api": {Script: "worker.js"}}}
	_, err = a.ApplyAll(canceledContext(), cfg, false)
	if err == nil {
		t.Fatal("expected error when diff computation fails")
	}
	if !strings.Contains(err.Error(), "failed to compute diff") {
		t.Errorf("error = %v, want prefix about failed diff computation", err)
	}
}

func TestApplyWorkers_EmptyConfig(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := a.ApplyWorkers(context.Background(), &CosmoflareConfig{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Service != "workers" {
		t.Errorf("Service = %q, want %q", result.Service, "workers")
	}
	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestApplyWorkers_DiffErrorPropagates(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"api": {Script: "worker.js"}}}
	_, err = a.ApplyWorkers(canceledContext(), cfg, false)
	if err == nil {
		t.Fatal("expected error when workers diff computation fails")
	}
	if !strings.Contains(err.Error(), "failed to compute workers diff") {
		t.Errorf("error = %v, want prefix about failed workers diff", err)
	}
}

func TestApplyR2_EmptyConfig(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := a.ApplyR2(context.Background(), &CosmoflareConfig{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Service != "r2" {
		t.Errorf("Service = %q, want %q", result.Service, "r2")
	}
	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestApplyR2_DiffErrorPropagates(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{R2: R2Config{Buckets: []R2BucketConfig{{Name: "assets"}}}}
	_, err = a.ApplyR2(canceledContext(), cfg, false)
	if err == nil {
		t.Fatal("expected error when r2 diff computation fails")
	}
	if !strings.Contains(err.Error(), "failed to compute r2 diff") {
		t.Errorf("error = %v, want prefix about failed r2 diff", err)
	}
}

func TestApplyKV_EmptyConfig(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := a.ApplyKV(context.Background(), &CosmoflareConfig{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Service != "kv" {
		t.Errorf("Service = %q, want %q", result.Service, "kv")
	}
	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestApplyKV_DiffErrorPropagates(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{KV: KVConfig{Namespaces: []KVNamespaceConfig{{Title: "MY_KV"}}}}
	_, err = a.ApplyKV(canceledContext(), cfg, false)
	if err == nil {
		t.Fatal("expected error when kv diff computation fails")
	}
	if !strings.Contains(err.Error(), "failed to compute kv diff") {
		t.Errorf("error = %v, want prefix about failed kv diff", err)
	}
}

func TestApplyDNS_ZoneIDRequired(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = a.ApplyDNS(context.Background(), &CosmoflareConfig{}, false)
	if err == nil {
		t.Fatal("expected error when zone_id is missing")
	}
}

func TestApplyDNS_DiffErrorPropagates(t *testing.T) {
	a, err := NewApplyService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{DNS: DNSConfig{ZoneID: "zone-1"}}
	_, err = a.ApplyDNS(canceledContext(), cfg, false)
	if err == nil {
		t.Fatal("expected error when dns diff computation fails")
	}
	if !strings.Contains(err.Error(), "failed to compute dns diff") {
		t.Errorf("error = %v, want prefix about failed dns diff", err)
	}
}

// --- Per-service apply*Changes dry-run behavior ---

func TestApplyWorkerChanges_DryRun(t *testing.T) {
	a := &ApplyService{}
	dr := &DiffResult{
		Service:   "workers",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "workers", Resource: "new-worker"}},
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "workers", Resource: "old-worker"}},
		Changes:   []DiffEntry{{Action: DiffModify, Service: "workers", Resource: "changed-worker"}},
	}
	result, err := a.applyWorkerChanges(context.Background(), &CosmoflareConfig{}, dr, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(result.Operations))
	}
	for _, op := range result.Operations {
		if op.Status != ApplyStatusSkipped {
			t.Errorf("dry-run op %q status = %q, want skipped", op.Resource, op.Status)
		}
		if !strings.Contains(op.Detail, "[dry-run] would") {
			t.Errorf("dry-run op %q detail missing marker: %q", op.Resource, op.Detail)
		}
	}
}

// NOTE: gateUnmanagedDeletes runs before the dry-run branch and marks
// unmanaged deletes Skipped with a "--delete-unmanaged" hint in Detail, but
// the dry-run loop unconditionally overwrites Detail with a generic
// "[dry-run] would <action> <resource>" message — so a dry-run preview never
// actually surfaces the gate's explanation, contradicting the comment above
// gateUnmanagedDeletes ("so the preview shows the same skips a real run
// would perform"). Both a gated delete and an ungated delete render
// identically in dry-run output. Not fixed here per task scope — this test
// documents the current (surprising) behavior.
func TestApplyWorkerChanges_DryRunGatesUnmanagedDelete(t *testing.T) {
	a := &ApplyService{} // deleteUnmanaged defaults false
	dr := &DiffResult{
		Service:   "workers",
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "workers", Resource: "stray-worker"}},
	}
	result, err := a.applyWorkerChanges(context.Background(), &CosmoflareConfig{}, dr, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Operations[0].Status != ApplyStatusSkipped {
		t.Errorf("Status = %q, want skipped", result.Operations[0].Status)
	}
	if !strings.Contains(result.Operations[0].Detail, "[dry-run] would") {
		t.Errorf("Detail = %q, want generic dry-run marker (gate hint is masked)", result.Operations[0].Detail)
	}
}

func TestApplyR2Changes_DryRun(t *testing.T) {
	a := &ApplyService{}
	dr := &DiffResult{
		Service:   "r2",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "r2", Resource: "new-bucket"}},
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "r2", Resource: "old-bucket"}},
	}
	result, err := a.applyR2Changes(context.Background(), &CosmoflareConfig{}, dr, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(result.Operations))
	}
	for _, op := range result.Operations {
		if op.Status != ApplyStatusSkipped {
			t.Errorf("dry-run op %q status = %q, want skipped", op.Resource, op.Status)
		}
	}
}

func TestApplyKVChanges_DryRun(t *testing.T) {
	a := &ApplyService{}
	dr := &DiffResult{
		Service:   "kv",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "kv", Resource: "new-ns"}},
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "kv", Resource: "old-ns"}},
	}
	result, err := a.applyKVChanges(context.Background(), &CosmoflareConfig{}, dr, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(result.Operations))
	}
	for _, op := range result.Operations {
		if op.Status != ApplyStatusSkipped {
			t.Errorf("dry-run op %q status = %q, want skipped", op.Resource, op.Status)
		}
	}
}

func TestApplyDNSChanges_DryRun(t *testing.T) {
	a := &ApplyService{}
	dr := &DiffResult{
		Service:   "dns",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "dns", Resource: "A new.example.com 1.2.3.4"}},
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "dns", Resource: "A old.example.com 5.6.7.8"}},
		Changes:   []DiffEntry{{Action: DiffModify, Service: "dns", Resource: "A changed.example.com 9.9.9.9"}},
	}
	result, err := a.applyDNSChanges(context.Background(), &CosmoflareConfig{}, dr, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(result.Operations))
	}
	for _, op := range result.Operations {
		if op.Status != ApplyStatusSkipped {
			t.Errorf("dry-run op %q status = %q, want skipped", op.Resource, op.Status)
		}
	}
}

// Non-dry-run: the client constructors themselves don't touch the network,
// but the first real API call inside these functions does. An already-
// canceled context fails that call deterministically without live network
// access, exercising the non-dry-run/error branches these functions take
// (BUG: R2/KV list-then-delete order means delete failures here surface as
// "failed to list" rather than a per-namespace delete error — noted, not
// fixed, per task instructions).

func TestApplyR2Changes_NonDryRunNetworkError(t *testing.T) {
	a := &ApplyService{accountID: "acc-123", apiToken: "tok-123"}
	dr := &DiffResult{
		Service:   "r2",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "r2", Resource: "new-bucket"}},
		Deletions: []DiffEntry{{Action: DiffRemove, Service: "r2", Resource: "old-bucket"}},
	}
	a.deleteUnmanaged = true // opt in so the delete path is actually attempted, not gated
	result, err := a.applyR2Changes(canceledContext(), &CosmoflareConfig{}, dr, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, op := range result.Operations {
		if op.Status != ApplyStatusFailed {
			t.Errorf("op %q status = %q, want failed (canceled context)", op.Resource, op.Status)
		}
	}
}

func TestApplyKVChanges_NonDryRunNetworkError(t *testing.T) {
	a := &ApplyService{accountID: "acc-123", apiToken: "tok-123"}
	dr := &DiffResult{
		Service:   "kv",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "kv", Resource: "new-ns"}},
	}
	result, err := a.applyKVChanges(canceledContext(), &CosmoflareConfig{}, dr, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Operations[0].Status != ApplyStatusFailed {
		t.Errorf("op status = %q, want failed (canceled context)", result.Operations[0].Status)
	}
}

func TestApplyDNSChanges_NonDryRunNetworkError(t *testing.T) {
	a := &ApplyService{accountID: "acc-123", apiToken: "tok-123"}
	dr := &DiffResult{
		Service:   "dns",
		Additions: []DiffEntry{{Action: DiffAdd, Service: "dns", Resource: "A new.example.com 1.2.3.4"}},
	}
	cfg := &CosmoflareConfig{DNS: DNSConfig{ZoneID: "zone-1"}}
	_, err := a.applyDNSChanges(canceledContext(), cfg, dr, false)
	if err == nil {
		t.Fatal("expected error listing live DNS records with a canceled context")
	}
	if !strings.Contains(err.Error(), "failed to list live DNS records") {
		t.Errorf("error = %v, want prefix about failed live-record listing", err)
	}
}

// --- applyWorkerCreate / applyWorkerUpdate / applyWorkerDelete ---

func writeTestWorkerScript(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "worker.js")
	if err := os.WriteFile(path, []byte("export default { fetch() {} }"), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}
	return path
}

func TestApplyWorkerCreate_Success(t *testing.T) {
	ws, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "my-worker", "script": "x", "size": 10},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{
		"my-worker": {Script: writeTestWorkerScript(t), Module: true, CompatibilityDate: "2024-01-01"},
	}}
	op := ApplyOperation{Action: ApplyCreate, Resource: "my-worker"}
	result := a.applyWorkerCreate(context.Background(), ws, cfg, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
	if !strings.Contains(result.Detail, "my-worker") {
		t.Errorf("Detail = %q, want mention of worker name", result.Detail)
	}
}

func TestApplyWorkerCreate_MissingFromConfig(t *testing.T) {
	a := &ApplyService{}
	op := ApplyOperation{Action: ApplyCreate, Resource: "ghost-worker"}
	result := a.applyWorkerCreate(context.Background(), nil, &CosmoflareConfig{}, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in config") {
		t.Errorf("Error = %q, want mention of missing config", result.Error)
	}
}

func TestApplyWorkerCreate_ScriptOpenError(t *testing.T) {
	a := &ApplyService{}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"w": {Script: "/nonexistent/path/x.js"}}}
	op := ApplyOperation{Action: ApplyCreate, Resource: "w"}
	result := a.applyWorkerCreate(context.Background(), nil, cfg, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "failed to read script") {
		t.Errorf("Error = %q, want mention of script read failure", result.Error)
	}
}

func TestApplyWorkerCreate_DeployError(t *testing.T) {
	ws, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		// 400 (non-retriable) keeps this test fast — a 5xx response makes
		// cloudflare-go's built-in retry/backoff run for several seconds.
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "boom"}},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"w": {Script: writeTestWorkerScript(t)}}}
	op := ApplyOperation{Action: ApplyCreate, Resource: "w"}
	result := a.applyWorkerCreate(context.Background(), ws, cfg, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
}

func TestApplyWorkerUpdate_Success(t *testing.T) {
	ws, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "my-worker", "script": "x", "size": 10},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"my-worker": {Script: writeTestWorkerScript(t)}}}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "my-worker"}
	result := a.applyWorkerUpdate(context.Background(), ws, cfg, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
}

func TestApplyWorkerUpdate_MissingFromConfig(t *testing.T) {
	a := &ApplyService{}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "ghost-worker"}
	result := a.applyWorkerUpdate(context.Background(), nil, &CosmoflareConfig{}, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in config for update") {
		t.Errorf("Error = %q, want mention of missing config", result.Error)
	}
}

func TestApplyWorkerUpdate_ScriptOpenError(t *testing.T) {
	a := &ApplyService{}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"w": {Script: "/nonexistent/path/x.js"}}}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "w"}
	result := a.applyWorkerUpdate(context.Background(), nil, cfg, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
}

func TestApplyWorkerDelete_Success(t *testing.T) {
	ws, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "my-worker"},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	op := ApplyOperation{Action: ApplyDelete, Resource: "my-worker"}
	result := a.applyWorkerDelete(context.Background(), ws, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
}

func TestApplyWorkerDelete_Error(t *testing.T) {
	ws, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "not found"}},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	op := ApplyOperation{Action: ApplyDelete, Resource: "ghost-worker"}
	result := a.applyWorkerDelete(context.Background(), ws, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
}

// --- applyDNSRecordCreate / applyDNSRecordDelete / applyDNSRecordUpdate ---

func TestApplyDNSRecordCreate_Success(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "rec1", "type": "A", "name": "test.example.com", "content": "1.2.3.4",
			},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	dctx := &applyDNSContext{
		dnsSvc: svc,
		configMap: map[applyDNSKey]DNSRecordConfig{
			key: {Type: "A", Name: "test.example.com", Content: "1.2.3.4", TTL: 300, Proxied: true},
		},
	}
	op := ApplyOperation{Action: ApplyCreate, Resource: "A test.example.com 1.2.3.4"}
	result := a.applyDNSRecordCreate(context.Background(), dctx, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
}

func TestApplyDNSRecordCreate_NotInConfig(t *testing.T) {
	a := &ApplyService{}
	dctx := &applyDNSContext{configMap: map[applyDNSKey]DNSRecordConfig{}}
	op := ApplyOperation{Action: ApplyCreate, Resource: "A ghost.example.com 9.9.9.9"}
	result := a.applyDNSRecordCreate(context.Background(), dctx, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in config") {
		t.Errorf("Error = %q, want mention of missing config record", result.Error)
	}
}

func TestApplyDNSRecordDelete_Success(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "rec1"},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	dctx := &applyDNSContext{
		dnsSvc: svc,
		liveMap: map[applyDNSKey]*DNSRecord{
			key: {ID: "rec1", Type: "A", Name: "test.example.com", Content: "1.2.3.4"},
		},
	}
	op := ApplyOperation{Action: ApplyDelete, Resource: "A test.example.com 1.2.3.4"}
	result := a.applyDNSRecordDelete(context.Background(), dctx, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
}

func TestApplyDNSRecordDelete_NotFoundLive(t *testing.T) {
	a := &ApplyService{}
	dctx := &applyDNSContext{liveMap: map[applyDNSKey]*DNSRecord{}}
	op := ApplyOperation{Action: ApplyDelete, Resource: "A ghost.example.com 9.9.9.9"}
	result := a.applyDNSRecordDelete(context.Background(), dctx, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in live state") {
		t.Errorf("Error = %q, want mention of missing live record", result.Error)
	}
}

func TestApplyDNSRecordUpdate_Success(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "rec1", "type": "A", "name": "test.example.com", "content": "1.2.3.4", "ttl": 600,
			},
		})
	})
	defer server.Close()

	a := &ApplyService{}
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	dctx := &applyDNSContext{
		dnsSvc: svc,
		liveMap: map[applyDNSKey]*DNSRecord{
			key: {ID: "rec1", Type: "A", Name: "test.example.com", Content: "1.2.3.4"},
		},
		configMap: map[applyDNSKey]DNSRecordConfig{
			key: {Type: "A", Name: "test.example.com", Content: "1.2.3.4", TTL: 600},
		},
	}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "A test.example.com 1.2.3.4"}
	result := a.applyDNSRecordUpdate(context.Background(), dctx, op)
	if result.Status != ApplyStatusSuccess {
		t.Errorf("Status = %q, want success; error: %s", result.Status, result.Error)
	}
}

func TestApplyDNSRecordUpdate_NotFoundLive(t *testing.T) {
	a := &ApplyService{}
	dctx := &applyDNSContext{liveMap: map[applyDNSKey]*DNSRecord{}}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "A ghost.example.com 9.9.9.9"}
	result := a.applyDNSRecordUpdate(context.Background(), dctx, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in live state for update") {
		t.Errorf("Error = %q, want mention of missing live record", result.Error)
	}
}

func TestApplyDNSRecordUpdate_NotFoundConfig(t *testing.T) {
	a := &ApplyService{}
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	dctx := &applyDNSContext{
		liveMap: map[applyDNSKey]*DNSRecord{
			key: {ID: "rec1", Type: "A", Name: "test.example.com", Content: "1.2.3.4"},
		},
		configMap: map[applyDNSKey]DNSRecordConfig{},
	}
	op := ApplyOperation{Action: ApplyUpdate, Resource: "A test.example.com 1.2.3.4"}
	result := a.applyDNSRecordUpdate(context.Background(), dctx, op)
	if result.Status != ApplyStatusFailed {
		t.Errorf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.Error, "not found in config for update") {
		t.Errorf("Error = %q, want mention of missing config record", result.Error)
	}
}

// --- workerDeployOptions / dnsRecordOptions / openWorkerScript / parseResourceToDNSKey ---

func TestWorkerDeployOptions(t *testing.T) {
	opts := workerDeployOptions(WorkerConfig{CompatibilityDate: "2024-01-01", Module: true})
	if len(opts) != 2 {
		t.Errorf("expected 2 options, got %d", len(opts))
	}

	none := workerDeployOptions(WorkerConfig{})
	if len(none) != 0 {
		t.Errorf("expected 0 options for zero-value config, got %d", len(none))
	}
}

func TestDNSRecordOptions(t *testing.T) {
	opts := dnsRecordOptions(DNSRecordConfig{TTL: 300, Proxied: true})
	if len(opts) != 2 {
		t.Errorf("expected 2 options (ttl + proxied), got %d", len(opts))
	}

	noTTL := dnsRecordOptions(DNSRecordConfig{Proxied: false})
	if len(noTTL) != 1 {
		t.Errorf("expected 1 option (proxied only, ttl omitted when zero), got %d", len(noTTL))
	}
}

func TestOpenWorkerScript_EmptyPath(t *testing.T) {
	_, err := openWorkerScript("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpenWorkerScript_NotFound(t *testing.T) {
	_, err := openWorkerScript("/nonexistent/path/script.js")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestOpenWorkerScript_Success(t *testing.T) {
	path := writeTestWorkerScript(t)
	rc, err := openWorkerScript(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
}

func TestParseResourceToDNSKey(t *testing.T) {
	key, ok := parseResourceToDNSKey("A test.example.com 1.2.3.4")
	if !ok {
		t.Fatal("expected successful parse")
	}
	if key.Type != "A" || key.Name != "test.example.com" || key.Content != "1.2.3.4" {
		t.Errorf("key = %+v, want {A test.example.com 1.2.3.4}", key)
	}

	_, ok = parseResourceToDNSKey("malformed")
	if ok {
		t.Error("expected parse failure for malformed resource")
	}
}

func TestFindConfigRecordFromResource(t *testing.T) {
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	configMap := map[applyDNSKey]DNSRecordConfig{key: {Type: "A", Name: "test.example.com", Content: "1.2.3.4"}}

	rec, found := findConfigRecordFromResource("A test.example.com 1.2.3.4", configMap)
	if !found {
		t.Fatal("expected record to be found")
	}
	if rec.Name != "test.example.com" {
		t.Errorf("Name = %q, want %q", rec.Name, "test.example.com")
	}

	_, found = findConfigRecordFromResource("malformed", configMap)
	if found {
		t.Error("expected no match for malformed resource")
	}
}

func TestFindLiveRecordFromResource(t *testing.T) {
	key := applyDNSKey{Type: "A", Name: "test.example.com", Content: "1.2.3.4"}
	liveMap := map[applyDNSKey]*DNSRecord{key: {ID: "rec1", Type: "A", Name: "test.example.com", Content: "1.2.3.4"}}

	rec := findLiveRecordFromResource("A test.example.com 1.2.3.4", liveMap)
	if rec == nil || rec.ID != "rec1" {
		t.Errorf("expected record with ID=rec1, got %+v", rec)
	}

	rec = findLiveRecordFromResource("malformed", liveMap)
	if rec != nil {
		t.Error("expected nil for malformed resource")
	}
}
