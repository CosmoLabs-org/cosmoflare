package cosmoflare

import (
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
