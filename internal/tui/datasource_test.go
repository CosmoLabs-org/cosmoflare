package tui

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNullDataSourceAvailable(t *testing.T) {
	ds := &nullDataSource{}
	if ds.Available() {
		t.Fatal("nullDataSource.Available() should return false")
	}
}

func TestNullDataSourceFetchBuckets(t *testing.T) {
	ds := &nullDataSource{}
	buckets, stats, err := ds.FetchBuckets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 0 {
		t.Fatalf("expected 0 buckets, got %d", len(buckets))
	}
	if stats.TotalUsed != 0 || stats.TotalLimit != 0 || stats.BucketCount != 0 {
		t.Fatalf("expected zero UsageStats, got %+v", stats)
	}
}

func TestNullDataSourceFetchObjects(t *testing.T) {
	ds := &nullDataSource{}
	listing, err := ds.FetchObjects(context.Background(), "test", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listing.Objects) != 0 {
		t.Errorf("expected 0 objects, got %d", len(listing.Objects))
	}
}

func TestNullDataSourceHeadObject(t *testing.T) {
	ds := &nullDataSource{}
	_, err := ds.HeadObject(context.Background(), "bucket", "key")
	if err == nil {
		t.Error("expected error from nullDataSource.HeadObject")
	}
}

func TestNullDataSourceDeleteObject(t *testing.T) {
	ds := &nullDataSource{}
	err := ds.DeleteObject(context.Background(), "bucket", "key")
	if err == nil {
		t.Error("expected error from nullDataSource.DeleteObject")
	}
}

func TestNullDataSourceFetchMetrics(t *testing.T) {
	before := time.Now()
	ds := &nullDataSource{}
	metrics, err := ds.FetchMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.R2.BucketCount != 0 || metrics.R2.TotalSize != 0 || metrics.R2.TotalObjects != 0 {
		t.Fatalf("expected zero R2 metrics, got %+v", metrics.R2)
	}
	if metrics.Workers.Count != 0 {
		t.Fatalf("expected zero Workers count, got %d", metrics.Workers.Count)
	}
	if metrics.KV.NamespaceCount != 0 {
		t.Fatalf("expected zero KV namespace count, got %d", metrics.KV.NamespaceCount)
	}
	if metrics.FetchedAt.Before(before) {
		t.Fatal("FetchedAt should be set to approximately now")
	}
}

func TestNullDataSourceCRUD(t *testing.T) {
	ds := &nullDataSource{}

	err := ds.CreateBucket(context.Background(), "test-bucket")
	if err == nil {
		t.Fatal("CreateBucket should return an error on nullDataSource")
	}
	if err.Error() != "no credentials configured — run cosmoflare setup" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	err = ds.DeleteBucket(context.Background(), "test-bucket")
	if err == nil {
		t.Fatal("DeleteBucket should return an error on nullDataSource")
	}
	if err.Error() != "no credentials configured — run cosmoflare setup" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestNewDataSourceNoCredentials(t *testing.T) {
	// Clear any env vars that might provide credentials.
	orig := map[string]string{
		"CLOUDFLARE_ACCOUNT_ID": os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		"CLOUDFLARE_API_TOKEN":  os.Getenv("CLOUDFLARE_API_TOKEN"),
	}
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	defer func() {
		for k, v := range orig {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	ds := newDataSource()
	if ds.Available() {
		t.Fatal("newDataSource() with no credentials should return a non-Available datasource")
	}
}
