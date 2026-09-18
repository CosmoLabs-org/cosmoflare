package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// buckSyncGlobals snapshots the globals the sync printers read.
func buckSyncGlobals(t *testing.T) {
	t.Helper()
	oldJSON := JSONOutput
	t.Cleanup(func() { JSONOutput = oldJSON })
}

// TestPrintSyncPlan_AllActionTypes verifies the dry-run plan renderer
// handles every operation kind plus the summary line without error.
func TestPrintSyncPlan_AllActionTypes(t *testing.T) {
	buckSyncGlobals(t)

	cases := []struct {
		name string
		json bool
	}{
		{"human output", false},
		{"json output", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			JSONOutput = tc.json
			plan := &cosmoflare.SyncPlan{
				Direction: cosmoflare.SyncUp,
				Bucket:    "my-bucket",
				Prefix:    "assets/",
				LocalDir:  "./dist",
				Operations: []cosmoflare.SyncOp{
					{Action: cosmoflare.SyncOpUpload, Key: "a.txt", Reason: "new"},
					{Action: cosmoflare.SyncOpDownload, Key: "b.txt", Reason: "changed"},
					{Action: cosmoflare.SyncOpDelete, Key: "c.txt", Reason: "missing at source"},
					{Action: cosmoflare.SyncOpSkip, Key: "d.txt", Reason: "unchanged"},
				},
				Summary: cosmoflare.SyncPlanSummary{Uploads: 1, Downloads: 1, Deletes: 1, Skips: 1},
			}
			if err := printSyncPlan(plan); err != nil {
				t.Fatalf("printSyncPlan: %v", err)
			}
		})
	}
}

// TestPrintSyncPlan_ZeroValueDoesNotPanic verifies a zero-value plan (nil
// operations, empty strings) renders without panicking.
func TestPrintSyncPlan_ZeroValueDoesNotPanic(t *testing.T) {
	buckSyncGlobals(t)
	JSONOutput = false

	if err := printSyncPlan(&cosmoflare.SyncPlan{}); err != nil {
		t.Fatalf("printSyncPlan zero-value: %v", err)
	}
}

// TestPrintSyncResult_SuccessAndFailure verifies the result renderer
// handles clean, failed, and error-carrying executions.
func TestPrintSyncResult_SuccessAndFailure(t *testing.T) {
	buckSyncGlobals(t)

	cases := []struct {
		name   string
		result *cosmoflare.SyncResult
	}{
		{"clean success", &cosmoflare.SyncResult{Succeeded: 3, Skipped: 1}},
		{"with failures", &cosmoflare.SyncResult{Succeeded: 1, Failed: 2}},
		{"with error strings", &cosmoflare.SyncResult{Failed: 1, Errors: []string{"upload x.txt: connection reset"}}},
		{"zero value", &cosmoflare.SyncResult{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := &cosmoflare.SyncPlan{
				Direction: cosmoflare.SyncDown,
				Bucket:    "my-bucket",
				Prefix:    "",
				LocalDir:  "./backup",
			}
			if err := printSyncResult(plan, tc.result); err != nil {
				t.Fatalf("printSyncResult: %v", err)
			}
		})
	}
}

// TestNewR2StorageBackend_MissingCreds verifies the backend constructor
// propagates the client validation error when credentials are empty.
func TestNewR2StorageBackend_MissingCreds(t *testing.T) {
	// NewClient falls back to the CLOUDFLARE_* environment variables, so
	// clear them to prove the constructor itself rejects empty creds.
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	cases := []struct {
		name             string
		accountID, token string
	}{
		{"both empty", "", ""},
		{"missing token", "acct-1", ""},
		{"missing account", "", "tok-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newR2StorageBackend(tc.accountID, tc.token)
			if err == nil || !strings.Contains(err.Error(), "required") {
				t.Fatalf("expected credential validation error, got %v", err)
			}
		})
	}
}

// TestNewR2StorageBackend_ValidCreds verifies a backend is constructed
// offline from dummy credentials (client creation performs no I/O).
func TestNewR2StorageBackend_ValidCreds(t *testing.T) {
	backend, err := newR2StorageBackend("acct-1", "tok-1")
	if err != nil {
		t.Fatalf("newR2StorageBackend: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

// TestSyncDown_MissingArgsErrorMessages verifies the down runner's usage
// errors for zero and one argument.
func TestSyncDown_MissingArgsErrorMessages(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		err := runSyncDown(syncDownCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "bucket is required") {
			t.Fatalf("expected bucket-required error, got %v", err)
		}
	})
	t.Run("one arg", func(t *testing.T) {
		err := runSyncDown(syncDownCmd, []string{"my-bucket"})
		if err == nil || !strings.Contains(err.Error(), "local directory is required") {
			t.Fatalf("expected local-dir-required error, got %v", err)
		}
	})
}

// TestSyncUp_MissingArgsErrorMessages verifies the up runner's usage
// errors for zero and one argument.
func TestSyncUp_MissingArgsErrorMessages(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		err := runSyncUp(syncUpCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "local directory is required") {
			t.Fatalf("expected local-dir-required error, got %v", err)
		}
	})
	t.Run("one arg", func(t *testing.T) {
		err := runSyncUp(syncUpCmd, []string{"./dist"})
		if err == nil || !strings.Contains(err.Error(), "bucket is required") {
			t.Fatalf("expected bucket-required error, got %v", err)
		}
	})
}
