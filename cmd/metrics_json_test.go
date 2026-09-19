package cmd

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// metricsRunSnapshot snapshots the metrics flags, output-mode switches, and
// credentials, and blanks the Cloudflare env vars so no test reads real creds.
func metricsRunSnapshot(t *testing.T) {
	t.Helper()
	savedInterval, savedWindow := metricsInterval, metricsWindow
	savedJSON, savedDry := JSONOutput, DryRun
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() {
		metricsInterval, metricsWindow = savedInterval, savedWindow
		JSONOutput, DryRun = savedJSON, savedDry
		AccountID, APIToken = savedAccount, savedToken
	})
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	JSONOutput, DryRun = false, false
	AccountID, APIToken = "", ""
}

// TestRunMetrics_JSONModeWithoutClient verifies the --json path returns the
// client-creation error when no credentials are available.
func TestRunMetrics_JSONModeWithoutClient(t *testing.T) {
	metricsRunSnapshot(t)
	JSONOutput = true

	err := runMetrics(nil, nil)
	if err == nil {
		t.Fatal("expected error when JSON mode cannot build a client")
	}
	if !strings.Contains(err.Error(), "failed to create client") {
		t.Errorf("error = %q, want it to mention failed to create client", err.Error())
	}
}

// TestRunMetricsJSON_AggregatesBucketTotals verifies bucket sizes and object
// counts are summed across buckets and emitted in the JSON snapshot.
func TestRunMetricsJSON_AggregatesBucketTotals(t *testing.T) {
	metricsRunSnapshot(t)
	metricsWindow = 24 * time.Hour

	client := &fakeR2Client{buckets: []*cosmoflare.Bucket{
		{Name: "one", Size: 100, ObjectCount: 4},
		{Name: "two", Size: 50, ObjectCount: 6},
	}}

	out := capturePrint(t, func() {
		if err := runMetricsJSON(client); err != nil {
			t.Errorf("runMetricsJSON should succeed: %v", err)
		}
	})

	var snap metricsSnapshot
	if err := json.Unmarshal([]byte(out), &snap); err != nil {
		t.Fatalf("output is not a metrics snapshot: %v\n%s", err, out)
	}
	if snap.R2Buckets != 2 {
		t.Errorf("R2Buckets = %d, want 2", snap.R2Buckets)
	}
	if snap.R2TotalSize != 150 {
		t.Errorf("R2TotalSize = %d, want 150", snap.R2TotalSize)
	}
	if snap.R2TotalObjects != 10 {
		t.Errorf("R2TotalObjects = %d, want 10", snap.R2TotalObjects)
	}
	// Without credentials the usage-analytics fan-out must record why the
	// analytics section is empty instead of silently omitting it.
	if snap.Errors["analytics"] == "" {
		t.Errorf("expected an 'analytics' entry in errors, got %v", snap.Errors)
	}
}

// TestRunMetricsJSON_NoBuckets verifies an empty account still prints a
// valid snapshot with zeroed totals.
func TestRunMetricsJSON_NoBuckets(t *testing.T) {
	metricsRunSnapshot(t)

	client := &fakeR2Client{}

	out := capturePrint(t, func() {
		if err := runMetricsJSON(client); err != nil {
			t.Errorf("empty account should succeed: %v", err)
		}
	})

	var snap metricsSnapshot
	if err := json.Unmarshal([]byte(out), &snap); err != nil {
		t.Fatalf("output is not a metrics snapshot: %v\n%s", err, out)
	}
	if snap.R2Buckets != 0 || snap.R2TotalSize != 0 || snap.R2TotalObjects != 0 {
		t.Errorf("expected zeroed totals, got %+v", snap)
	}
}

// TestRunMetricsJSON_ListBucketsError verifies the ListBuckets failure is
// surfaced as the fetch error rather than a partial snapshot.
func TestRunMetricsJSON_ListBucketsError(t *testing.T) {
	metricsRunSnapshot(t)

	client := &fakeR2Client{listBucketsErr: errMetricsStubList}

	err := runMetricsJSON(client)
	if err == nil {
		t.Fatal("expected error when ListBuckets fails")
	}
	if !strings.Contains(err.Error(), "failed to fetch metrics") {
		t.Errorf("error = %q, want it to mention failed to fetch metrics", err.Error())
	}
}

// errMetricsStubList is the sentinel error used by the fake R2 client above.
var errMetricsStubList = errStr("stub list failure")

// errMetricsStubList's concrete type: a tiny string error to avoid importing
// errors in this file for a single use.
type errStr string

func (e errStr) Error() string { return string(e) }

// TestFetchZoneHTTP_MissingCredentials verifies fetchZoneHTTP fails fast when
// the zone service cannot be built for lack of credentials.
func TestFetchZoneHTTP_MissingCredentials(t *testing.T) {
	metricsRunSnapshot(t)

	analytics := cosmoflare.NewAnalyticsService("", "")
	snap := &metricsSnapshot{}
	var mu sync.Mutex

	err := fetchZoneHTTP(nil, analytics, cosmoflare.AnalyticsWindow{}, snap, &mu)
	if err == nil {
		t.Fatal("expected error when zone service cannot be created")
	}
	if !strings.Contains(err.Error(), "account ID is required") {
		t.Errorf("error = %q, want it to mention account ID is required", err.Error())
	}
}
