package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// d1RunSnapshot snapshots the D1 handler globals and restores them on cleanup.
func d1RunSnapshot(t *testing.T) {
	t.Helper()
	savedDry, savedJSON, savedLocal, savedRemote := DryRun, JSONOutput, d1Local, d1Remote
	savedSQL, savedForce := d1SQL, d1Force
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() {
		DryRun, JSONOutput, d1Local, d1Remote = savedDry, savedJSON, savedLocal, savedRemote
		d1SQL, d1Force = savedSQL, savedForce
		AccountID, APIToken = savedAccount, savedToken
	})
}

// TestRunD1ServiceError verifies list and get fail fast with the
// service-creation error when credentials are absent.
func TestRunD1ServiceError(t *testing.T) {
	d1RunSnapshot(t)
	JSONOutput = false
	DryRun = false
	AccountID = ""
	APIToken = ""

	t.Run("list", func(t *testing.T) {
		err := runD1List(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to create D1 service") {
			t.Errorf("expected service error, got %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		err := runD1Get(nil, []string{"db-uuid"})
		if err == nil || !strings.Contains(err.Error(), "failed to create D1 service") {
			t.Errorf("expected service error, got %v", err)
		}
	})
}

// TestRunD1Query_LocalAndRemoteConflict verifies passing both --local and
// --remote is rejected before any service or database access.
func TestRunD1Query_LocalAndRemoteConflict(t *testing.T) {
	d1RunSnapshot(t)
	JSONOutput = false
	d1SQL = "SELECT 1"
	d1Local = true
	d1Remote = true

	err := runD1Query(nil, []string{"db"})
	if err == nil || !strings.Contains(err.Error(), "use only one of --local or --remote") {
		t.Fatalf("expected local/remote conflict error, got %v", err)
	}
}

// TestRunD1Delete_DryRunDefault verifies the FEAT-020 wave-2 contract: an
// un-forced destructive delete runs dry by default — no prompt, no service
// call — and reports what would have been deleted.
func TestRunD1Delete_DryRunDefault(t *testing.T) {
	d1RunSnapshot(t)
	JSONOutput = false
	DryRun = false
	d1Force = false
	AccountID = ""
	APIToken = "" // dry-run must happen before service creation

	out := capturePrint(t, func() {
		if err := runD1Delete(nil, []string{"db-uuid"}); err != nil {
			t.Errorf("dry-run default should exit cleanly, got %v", err)
		}
	})

	if !strings.Contains(out, "DRY RUN: Would delete database") {
		t.Errorf("expected dry-run notice, got %q", out)
	}
}

// TestPrintD1Results_NoRows verifies results without rows report success and
// metadata instead of rendering an empty table.
func TestPrintD1Results_NoRows(t *testing.T) {
	savedJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = savedJSON }()

	out := capturePrint(t, func() {
		printD1Results([]*cosmoflare.D1QueryResult{{}})
	})

	if !strings.Contains(out, "no rows returned") {
		t.Errorf("expected no-rows notice, got %q", out)
	}
	if !strings.Contains(out, "Rows read: 0") {
		t.Errorf("expected zero metadata line, got %q", out)
	}
}

// TestPrintD1Results_WithRows verifies rows render as a table and multiple
// statements are printed in order.
func TestPrintD1Results_WithRows(t *testing.T) {
	savedJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = savedJSON }()

	results := []*cosmoflare.D1QueryResult{
		{Rows: []map[string]any{{"id": 1, "name": "alice"}, {"id": 2, "name": "bob"}}},
		{Rows: []map[string]any{{"count": 2}}},
	}

	out := capturePrint(t, func() { printD1Results(results) })

	for _, want := range []string{"alice", "bob", "count"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q: %q", want, out)
		}
	}
}
