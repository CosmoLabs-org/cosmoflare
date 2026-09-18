package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// workerBindingsSnapshot zeroes the credential globals so service
// construction fails deterministically, restoring them afterwards.
func workerBindingsSnapshot(t *testing.T) {
	t.Helper()
	origAccount, origToken := AccountID, APIToken
	origJSON := JSONOutput
	t.Cleanup(func() {
		AccountID, APIToken = origAccount, origToken
		JSONOutput = origJSON
	})
	AccountID, APIToken = "", ""
	JSONOutput = false
}

// TestRunWorkerBindings_NameRequired verifies the direct-call guard rejects
// an empty argument list before any service work.
func TestRunWorkerBindings_NameRequired(t *testing.T) {
	workerBindingsSnapshot(t)

	err := runWorkerBindings(workerBindingsCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected name-required error, got %v", err)
	}
}

// TestRunWorkerBindings_MissingCreds verifies the runner fails fast with the
// service construction error when credentials are absent.
func TestRunWorkerBindings_MissingCreds(t *testing.T) {
	workerBindingsSnapshot(t)

	err := runWorkerBindings(workerBindingsCmd, []string{"api-gateway"})
	if err == nil {
		t.Fatal("runWorkerBindings with empty credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create worker service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create worker service")
	}
}

// TestPrintBindingsTable_NoBindings verifies the empty case prints the
// no-bindings notice instead of an empty table.
func TestPrintBindingsTable_NoBindings(t *testing.T) {
	out := capturePrint(t, func() {
		printBindingsTable("api-gateway", nil)
	})

	if !strings.Contains(out, `No bindings attached to worker "api-gateway"`) {
		t.Errorf("expected no-bindings notice, got %q", out)
	}
}

// TestPrintBindingsTable_RendersRows verifies the table lists each binding's
// name, type, and target ID, plus the total count.
func TestPrintBindingsTable_RendersRows(t *testing.T) {
	bindings := []cosmoflare.WorkerBinding{
		{Name: "CACHE", Type: "kv", ID: "ns-cache"},
		{Name: "DB", Type: "d1", ID: "db-uuid"},
	}

	out := capturePrint(t, func() {
		printBindingsTable("api-gateway", bindings)
	})

	for _, want := range []string{"CACHE", "kv", "ns-cache", "DB", "d1", "db-uuid", "Total: 2 binding(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("bindings table missing %q: %q", want, out)
		}
	}
}
