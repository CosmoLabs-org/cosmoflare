/*
Package tui provides the interactive terminal dashboard for Cosmoflare.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"context"
	"errors"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

const objectsPerPage = 100

// ObjectItem is a TUI display projection of a storage object.
type ObjectItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	ETag         string
}

// ObjectListing holds a page of objects returned by FetchObjects.
type ObjectListing struct {
	Dirs      []string
	Objects   []ObjectItem
	NextToken string
	HasMore   bool
}

// ObjectDetail holds full metadata for a single object (from HeadObject).
type ObjectDetail struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	ETag         string
	Metadata     map[string]string
}

// ServiceMetrics aggregates metrics across Cloudflare services.
type ServiceMetrics struct {
	R2      R2Metrics
	Workers WorkersMetrics
	KV      KVMetrics
	FetchedAt time.Time
}

// R2Metrics holds R2-specific usage numbers.
type R2Metrics struct {
	BucketCount  int
	TotalSize    int64
	TotalObjects int64
}

// WorkersMetrics holds Workers-specific usage numbers.
type WorkersMetrics struct {
	Count int
}

// KVMetrics holds KV-specific usage numbers.
type KVMetrics struct {
	NamespaceCount int
}

// DataSource abstracts Cloudflare API access for the dashboard.
type DataSource interface {
	FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error)
	FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error)
	FetchMetrics(ctx context.Context) (ServiceMetrics, error)
	HeadObject(ctx context.Context, bucket, key string) (*ObjectDetail, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	DeleteObject(ctx context.Context, bucket, key string) error
	Available() bool
}

// ---------------------------------------------------------------------------
// nullDataSource — returned when no credentials are configured.
// ---------------------------------------------------------------------------

type nullDataSource struct{}

func (n *nullDataSource) Available() bool { return false }

func (n *nullDataSource) FetchBuckets(_ context.Context) ([]Bucket, UsageStats, error) {
	return []Bucket{}, UsageStats{}, nil
}

func (n *nullDataSource) FetchObjects(_ context.Context, _, _, _ string) (ObjectListing, error) {
	return ObjectListing{}, nil
}

func (n *nullDataSource) HeadObject(_ context.Context, _, _ string) (*ObjectDetail, error) {
	return nil, errors.New("no credentials configured — run cosmoflare setup")
}

func (n *nullDataSource) DeleteObject(_ context.Context, _, _ string) error {
	return errors.New("no credentials configured — run cosmoflare setup")
}

func (n *nullDataSource) FetchMetrics(_ context.Context) (ServiceMetrics, error) {
	return ServiceMetrics{FetchedAt: time.Now()}, nil
}

func (n *nullDataSource) CreateBucket(_ context.Context, _ string) error {
	return errors.New("no credentials configured — run cosmoflare setup")
}

func (n *nullDataSource) DeleteBucket(_ context.Context, _ string) error {
	return errors.New("no credentials configured — run cosmoflare setup")
}

// ---------------------------------------------------------------------------
// apiDataSource — live Cloudflare API backend.
// ---------------------------------------------------------------------------

type apiDataSource struct {
	r2      cosmoflare.R2Client
	workers *cosmoflare.WorkerService
	kv      *cosmoflare.KVService
}

func (a *apiDataSource) Available() bool { return true }

func (a *apiDataSource) FetchBuckets(ctx context.Context) ([]Bucket, UsageStats, error) {
	cfBuckets, err := a.r2.ListBuckets(ctx)
	if err != nil {
		return nil, UsageStats{}, err
	}

	buckets := make([]Bucket, 0, len(cfBuckets))
	var totalUsed int64
	for _, b := range cfBuckets {
		buckets = append(buckets, Bucket{
			Name:        b.Name,
			Size:        b.Size,
			ObjectCount: b.ObjectCount,
			Status:      "active",
			CreatedAt:   b.CreatedAt,
		})
		totalUsed += b.Size
	}

	stats := UsageStats{
		TotalUsed:   totalUsed,
		TotalLimit:  1024 * 1024 * 1024 * 1024, // 1 TB default
		BucketCount: len(buckets),
	}

	return buckets, stats, nil
}

func (a *apiDataSource) FetchObjects(ctx context.Context, bucket, prefix, continuationToken string) (ObjectListing, error) {
	result, err := a.r2.ListObjects(ctx, bucket, prefix, "/", int32(objectsPerPage), continuationToken)
	if err != nil {
		return ObjectListing{}, err
	}

	items := make([]ObjectItem, 0, len(result.Items))
	for _, obj := range result.Items {
		items = append(items, ObjectItem{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
		})
	}

	return ObjectListing{
		Dirs:      result.CommonPrefixes,
		Objects:   items,
		NextToken: result.NextToken,
		HasMore:   result.IsTruncated,
	}, nil
}

func (a *apiDataSource) HeadObject(ctx context.Context, bucket, key string) (*ObjectDetail, error) {
	result, err := a.r2.HeadObject(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	return &ObjectDetail{
		Key:          result.Key,
		Size:         result.Size,
		LastModified: result.LastModified,
		ContentType:  result.ContentType,
		ETag:         result.ETag,
		Metadata:     result.Metadata,
	}, nil
}

func (a *apiDataSource) DeleteObject(ctx context.Context, bucket, key string) error {
	return a.r2.DeleteObject(ctx, bucket, key)
}

func (a *apiDataSource) FetchMetrics(ctx context.Context) (ServiceMetrics, error) {
	m := ServiceMetrics{FetchedAt: time.Now()}

	// R2 metrics — always available since r2 is non-nil.
	if cfBuckets, err := a.r2.ListBuckets(ctx); err == nil {
		m.R2.BucketCount = len(cfBuckets)
		for _, b := range cfBuckets {
			m.R2.TotalSize += b.Size
			m.R2.TotalObjects += b.ObjectCount
		}
	}

	// Workers metrics — may be nil if service creation failed.
	if a.workers != nil {
		if workers, err := a.workers.List(ctx); err == nil {
			m.Workers.Count = len(workers)
		}
	}

	// KV metrics — may be nil if service creation failed.
	if a.kv != nil {
		if namespaces, err := a.kv.ListNamespaces(ctx); err == nil {
			m.KV.NamespaceCount = len(namespaces)
		}
	}

	return m, nil
}

func (a *apiDataSource) CreateBucket(ctx context.Context, name string) error {
	_, err := a.r2.CreateBucket(ctx, name)
	return err
}

func (a *apiDataSource) DeleteBucket(ctx context.Context, name string) error {
	return a.r2.DeleteBucket(ctx, name)
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// newDataSource builds a DataSource from the user's configured credentials.
// If no credentials are found, it returns a nullDataSource.
func newDataSource() DataSource {
	var accountID, apiToken string

	// Try config file first.
	if cm, err := config.NewConfigManager(); err == nil {
		if profile, err := cm.GetCurrent(); err == nil {
			accountID = profile.AccountID
			apiToken = profile.APIToken
		}
	}

	// Fallback to environment variables.
	if accountID == "" || apiToken == "" {
		env := config.LoadFromEnvironment()
		if accountID == "" {
			accountID = env.AccountID
		}
		if apiToken == "" {
			apiToken = env.APIToken
		}
	}

	if accountID == "" || apiToken == "" {
		return &nullDataSource{}
	}

	r2, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(accountID),
		cosmoflare.WithAPIToken(apiToken),
	)
	if err != nil {
		return &nullDataSource{}
	}

	// Optional services — errors are non-fatal.
	ws, _ := cosmoflare.NewWorkerServiceFromCreds(accountID, apiToken)
	ks, _ := cosmoflare.NewKVServiceFromCreds(accountID, apiToken)

	return &apiDataSource{r2: r2, workers: ws, kv: ks}
}
