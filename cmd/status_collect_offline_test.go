package cmd

import (
	"context"
	"strings"
	"testing"
)

// buckStatusGlobals clears credentials for the duration of a test so every
// status collector hits its offline error path.
func buckStatusGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldTok := AccountID, APIToken
	// NewClient falls back to the CLOUDFLARE_* environment variables, so
	// clear them too (t.Setenv restores the originals afterwards).
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Cleanup(func() { AccountID, APIToken = oldAcct, oldTok })
	AccountID, APIToken = "", ""
}

// TestCollectStatusZones_OfflineError verifies the zones collector records
// the failure in both the report section and the shared error list.
func TestCollectStatusZones_OfflineError(t *testing.T) {
	buckStatusGlobals(t)

	report := &StatusReport{}
	var errs []string
	collectStatusZones(context.Background(), report, func(msg string) { errs = append(errs, msg) })

	if report.Zones.Count != 0 {
		t.Errorf("Zones.Count = %d, want 0 on failure", report.Zones.Count)
	}
	if report.Zones.Error == "" {
		t.Error("Zones.Error should be populated on failure")
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], "zones: ") {
		t.Errorf("addError calls = %v, want single 'zones: ' entry", errs)
	}
}

// TestCollectStatusWorkers_OfflineError verifies the workers collector
// records the failure in the report and the error list.
func TestCollectStatusWorkers_OfflineError(t *testing.T) {
	buckStatusGlobals(t)

	report := &StatusReport{}
	var errs []string
	collectStatusWorkers(context.Background(), report, func(msg string) { errs = append(errs, msg) })

	if report.Workers.Count != 0 {
		t.Errorf("Workers.Count = %d, want 0 on failure", report.Workers.Count)
	}
	if report.Workers.Error == "" {
		t.Error("Workers.Error should be populated on failure")
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], "workers: ") {
		t.Errorf("addError calls = %v, want single 'workers: ' entry", errs)
	}
}

// TestCollectStatusKV_OfflineError verifies the KV collector records the
// failure in the report and the error list.
func TestCollectStatusKV_OfflineError(t *testing.T) {
	buckStatusGlobals(t)

	report := &StatusReport{}
	var errs []string
	collectStatusKV(context.Background(), report, func(msg string) { errs = append(errs, msg) })

	if report.KV.Count != 0 {
		t.Errorf("KV.Count = %d, want 0 on failure", report.KV.Count)
	}
	if report.KV.Error == "" {
		t.Error("KV.Error should be populated on failure")
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], "kv: ") {
		t.Errorf("addError calls = %v, want single 'kv: ' entry", errs)
	}
}

// TestCollectStatusR2_OfflineError verifies the R2 collector records the
// failure in the report and the error list.
func TestCollectStatusR2_OfflineError(t *testing.T) {
	buckStatusGlobals(t)

	report := &StatusReport{}
	var errs []string
	collectStatusR2(context.Background(), report, func(msg string) { errs = append(errs, msg) })

	if report.Buckets.Count != 0 {
		t.Errorf("Buckets.Count = %d, want 0 on failure", report.Buckets.Count)
	}
	if report.Buckets.Error == "" {
		t.Error("Buckets.Error should be populated on failure")
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], "r2: ") {
		t.Errorf("addError calls = %v, want single 'r2: ' entry", errs)
	}
}

// TestGetR2ClientForStatus_MissingCreds verifies the R2 client helper used
// by the status dashboard fails fast without credentials.
func TestGetR2ClientForStatus_MissingCreds(t *testing.T) {
	buckStatusGlobals(t)

	if _, err := getR2ClientForStatus(); err == nil {
		t.Fatal("expected error creating R2 client without credentials")
	}
}
