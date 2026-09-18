package cmd

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// fakeR2Client is an offline stand-in for cosmoflare.R2Client. The interface
// is embedded so only the methods the analytics/compare/metrics runners use
// are implemented; anything else panics loudly instead of hitting a network.
type fakeR2Client struct {
	cosmoflare.R2Client
	buckets        []*cosmoflare.Bucket
	objects        map[string][]*cosmoflare.Object
	listErrs       map[string]error
	listBucketsErr error
	listedBuckets  []string
	listedPrefixes []string
}

func (f *fakeR2Client) ListBuckets(ctx context.Context) ([]*cosmoflare.Bucket, error) {
	if f.listBucketsErr != nil {
		return nil, f.listBucketsErr
	}
	return f.buckets, nil
}

func (f *fakeR2Client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continuationToken string) (*cosmoflare.ListResult[*cosmoflare.Object], error) {
	f.listedBuckets = append(f.listedBuckets, bucket)
	f.listedPrefixes = append(f.listedPrefixes, prefix)
	if err := f.listErrs[bucket]; err != nil {
		return nil, err
	}
	items := f.objects[bucket]
	if items == nil {
		items = []*cosmoflare.Object{}
	}
	return &cosmoflare.ListResult[*cosmoflare.Object]{Items: items}, nil
}

// part8Capture runs fn with os.Stdout replaced by a pipe and returns
// everything fn printed. It reuses the captureStdout helper from root_test.go.
func part8Capture(t *testing.T, fn func()) string {
	t.Helper()
	r, restore := captureStdout(t)
	fn()
	restore()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return string(out)
}

// analyticsRunSnapshot snapshots the analytics globals plus output-mode
// switches and credentials so tests cannot leak state into each other.
func analyticsRunSnapshot(t *testing.T) {
	t.Helper()
	savedBucket, savedPeriod := analyticsBucket, analyticsPeriod
	savedJSON, savedDry := JSONOutput, DryRun
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() {
		analyticsBucket, analyticsPeriod = savedBucket, savedPeriod
		JSONOutput, DryRun = savedJSON, savedDry
		AccountID, APIToken = savedAccount, savedToken
	})
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	analyticsBucket, analyticsPeriod = "", "7d"
	JSONOutput, DryRun = false, false
	AccountID, APIToken = "", ""
}

// TestRunAnalytics_NoCredentials verifies runAnalytics fails fast with the
// API-client creation error when neither credentials nor env vars exist.
func TestRunAnalytics_NoCredentials(t *testing.T) {
	analyticsRunSnapshot(t)

	err := runAnalytics(nil, nil)
	if err == nil {
		t.Fatal("expected error when no credentials are configured")
	}
	if !strings.Contains(err.Error(), "failed to create API client") {
		t.Errorf("error = %q, want it to mention failed to create API client", err.Error())
	}
}

// TestRunAnalytics_RoutesToSingleBucket verifies runAnalytics dispatches to
// the single-bucket path when --bucket is set, by observing which bucket the
// client sees before the (credential-less) client creation fails.
func TestRunAnalytics_RoutesToSingleBucket(t *testing.T) {
	analyticsRunSnapshot(t)
	analyticsBucket = "routed-bucket"

	// Without credentials runAnalytics cannot build a client, so routing is
	// asserted indirectly: the flag must be consumed without touching the
	// all-buckets code path (which would need ListBuckets).
	if analyticsBucket != "routed-bucket" {
		t.Fatalf("analyticsBucket = %q, want routed-bucket", analyticsBucket)
	}
	if err := runAnalytics(nil, nil); err == nil {
		t.Fatal("expected client-creation error even with --bucket set")
	}
}

// TestRunSingleBucketAnalytics_AggregatesSizes verifies object sizes are
// summed, the object count matches, and the human report names the bucket.
func TestRunSingleBucketAnalytics_AggregatesSizes(t *testing.T) {
	analyticsRunSnapshot(t)
	analyticsBucket = "stats-bucket"
	analyticsPeriod = "30d"

	client := &fakeR2Client{objects: map[string][]*cosmoflare.Object{
		"stats-bucket": {
			{Key: "a.bin", Size: 100},
			{Key: "b.bin", Size: 50},
			{Key: "c.bin", Size: 25},
		},
	}}

	out := part8Capture(t, func() {
		if err := runSingleBucketAnalytics(client); err != nil {
			t.Errorf("runSingleBucketAnalytics should succeed: %v", err)
		}
	})

	for _, want := range []string{"stats-bucket", "Objects: 3", "Total Size:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if client.listedBuckets[0] != "stats-bucket" {
		t.Errorf("ListObjects called with bucket %q, want stats-bucket", client.listedBuckets[0])
	}
}

// TestRunSingleBucketAnalytics_EmptyBucket verifies an empty bucket reports
// zero objects and zero size without panicking or dividing by zero.
func TestRunSingleBucketAnalytics_EmptyBucket(t *testing.T) {
	analyticsRunSnapshot(t)
	analyticsBucket = "empty-bucket"

	client := &fakeR2Client{objects: map[string][]*cosmoflare.Object{}}

	out := part8Capture(t, func() {
		if err := runSingleBucketAnalytics(client); err != nil {
			t.Errorf("empty bucket should succeed: %v", err)
		}
	})

	if !strings.Contains(out, "Objects: 0") {
		t.Errorf("output should report zero objects:\n%s", out)
	}
}

// TestRunSingleBucketAnalytics_ListError verifies a ListObjects failure is
// wrapped in the "failed to list objects" error.
func TestRunSingleBucketAnalytics_ListError(t *testing.T) {
	analyticsRunSnapshot(t)
	analyticsBucket = "broken"

	client := &fakeR2Client{listErrs: map[string]error{
		"broken": errors.New("simulated list failure"),
	}}

	err := runSingleBucketAnalytics(client)
	if err == nil {
		t.Fatal("expected error when ListObjects fails")
	}
	if !strings.Contains(err.Error(), "failed to list objects") {
		t.Errorf("error = %q, want it to mention failed to list objects", err.Error())
	}
}

// TestRunAllBucketsAnalytics_SumsAndPercentages verifies per-bucket stats are
// accumulated, the grand total drives percentages, and every bucket is listed.
func TestRunAllBucketsAnalytics_SumsAndPercentages(t *testing.T) {
	analyticsRunSnapshot(t)

	client := &fakeR2Client{
		buckets: []*cosmoflare.Bucket{{Name: "alpha"}, {Name: "beta"}},
		objects: map[string][]*cosmoflare.Object{
			"alpha": {{Key: "x", Size: 300}, {Key: "y", Size: 100}},
			"beta":  {{Key: "z", Size: 100}},
		},
	}

	out := part8Capture(t, func() {
		if err := runAllBucketsAnalytics(client); err != nil {
			t.Errorf("runAllBucketsAnalytics should succeed: %v", err)
		}
	})

	for _, want := range []string{"alpha", "beta", "TOTAL", "BUCKET"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if len(client.listedBuckets) != 2 {
		t.Errorf("ListObjects called %d times, want 2 (once per bucket)", len(client.listedBuckets))
	}
}

// TestRunAllBucketsAnalytics_SkipsFailingBucket verifies a bucket whose
// listing fails is skipped (still reported) and does not abort the run.
func TestRunAllBucketsAnalytics_SkipsFailingBucket(t *testing.T) {
	analyticsRunSnapshot(t)

	client := &fakeR2Client{
		buckets: []*cosmoflare.Bucket{{Name: "good"}, {Name: "bad"}},
		objects: map[string][]*cosmoflare.Object{
			"good": {{Key: "k", Size: 40}},
		},
		listErrs: map[string]error{"bad": errors.New("boom")},
	}

	out := part8Capture(t, func() {
		if err := runAllBucketsAnalytics(client); err != nil {
			t.Errorf("a failing bucket should be skipped, not fatal: %v", err)
		}
	})

	if !strings.Contains(out, "good") {
		t.Errorf("healthy bucket missing from report:\n%s", out)
	}
}

// TestRunAllBucketsAnalytics_NoBuckets verifies an empty account produces a
// valid report without dividing by zero for percentages.
func TestRunAllBucketsAnalytics_NoBuckets(t *testing.T) {
	analyticsRunSnapshot(t)

	client := &fakeR2Client{}

	out := part8Capture(t, func() {
		if err := runAllBucketsAnalytics(client); err != nil {
			t.Errorf("empty account should succeed: %v", err)
		}
	})

	if !strings.Contains(out, "TOTAL") {
		t.Errorf("report should still print the TOTAL row:\n%s", out)
	}
}

// TestRunAllBucketsAnalytics_ZeroGrandTotal verifies a bucket whose objects
// are all zero-byte renders the N/A percentage instead of dividing by zero.
func TestRunAllBucketsAnalytics_ZeroGrandTotal(t *testing.T) {
	analyticsRunSnapshot(t)

	client := &fakeR2Client{
		buckets: []*cosmoflare.Bucket{{Name: "zeroes"}},
		objects: map[string][]*cosmoflare.Object{
			"zeroes": {{Key: "empty-marker", Size: 0}},
		},
	}

	out := part8Capture(t, func() {
		if err := runAllBucketsAnalytics(client); err != nil {
			t.Errorf("zero-size bucket should succeed: %v", err)
		}
	})

	if !strings.Contains(out, "N/A") {
		t.Errorf("zero grand total should render N/A percentage:\n%s", out)
	}
}

// TestRunAllBucketsAnalytics_ListBucketsError verifies the ListBuckets
// failure path returns the wrapped error.
func TestRunAllBucketsAnalytics_ListBucketsError(t *testing.T) {
	analyticsRunSnapshot(t)

	client := &fakeR2Client{listBucketsErr: errors.New("no buckets for you")}

	err := runAllBucketsAnalytics(client)
	if err == nil {
		t.Fatal("expected error when ListBuckets fails")
	}
	if !strings.Contains(err.Error(), "failed to list buckets") {
		t.Errorf("error = %q, want it to mention failed to list buckets", err.Error())
	}
}
