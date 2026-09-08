package cosmoflare

import (
	"context"
	"io"
	"time"
)

type WorkerServicer interface {
	Deploy(ctx context.Context, name string, script io.Reader, opts ...WorkerOption) (*Worker, error)
	List(ctx context.Context) ([]*Worker, error)
	Get(ctx context.Context, name string) (*Worker, error)
	Delete(ctx context.Context, name string) error
	Logs(ctx context.Context, name string, opts ...LogOption) ([]*LogEntry, error)
	UpdateSettings(ctx context.Context, name string, settings WorkerSettings) error
}

type KVServicer interface {
	CreateNamespace(ctx context.Context, title string) (*KVNamespace, error)
	ListNamespaces(ctx context.Context) ([]*KVNamespace, error)
	GetNamespace(ctx context.Context, id string) (*KVNamespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	Put(ctx context.Context, namespaceID, key string, value io.Reader, opts ...KVOption) error
	Get(ctx context.Context, namespaceID, key string) ([]byte, error)
	Delete(ctx context.Context, namespaceID, key string) error
	ListKeys(ctx context.Context, namespaceID string, opts ...KVListOption) (*ListResult[*KVKey], error)
}

type DNSServicer interface {
	Create(ctx context.Context, recordType, name, content string, opts ...DNSOption) (*DNSRecord, error)
	List(ctx context.Context, opts ...DNSListOption) ([]*DNSRecord, error)
	Get(ctx context.Context, recordID string) (*DNSRecord, error)
	Update(ctx context.Context, recordID string, opts ...DNSOption) (*DNSRecord, error)
	Delete(ctx context.Context, recordID string) error
}

type ZoneServicer interface {
	Create(ctx context.Context, name string, zoneType string) (*Zone, error)
	List(ctx context.Context) ([]*Zone, error)
	Get(ctx context.Context, zoneID string) (*Zone, error)
	Delete(ctx context.Context, zoneID string) error
	GetSettings(ctx context.Context, zoneID string) ([]*ZoneSetting, error)
}

type SSLServicer interface {
	GetSSL(ctx context.Context) (*SSLStatus, error)
	UpdateSSL(ctx context.Context, value string) (*SSLStatus, error)
	GetVerification(ctx context.Context) ([]*SSLVerification, error)
	GetSettings(ctx context.Context) (*SSLSettings, error)
	UpdateSettings(ctx context.Context, opts ...SSLOption) error
}

type CacheServicer interface {
	PurgeAll(ctx context.Context) (*CachePurgeResult, error)
	PurgeByURLs(ctx context.Context, urls []string) (*CachePurgeResult, error)
	PurgeByTags(ctx context.Context, tags []string) (*CachePurgeResult, error)
	PurgeByHosts(ctx context.Context, hosts []string) (*CachePurgeResult, error)
	GetSettings(ctx context.Context) (*CacheSettings, error)
	UpdateSettings(ctx context.Context, opts ...CacheOption) error
}

type CORSServicer interface {
	GetCORSRules(ctx context.Context) ([]*CORSRule, error)
	SetCORSHeaders(ctx context.Context, opts ...CORSOption) (*CORSRule, error)
	RemoveCORSRule(ctx context.Context, name string) error
}

type FirewallServicer interface {
	List(ctx context.Context) ([]*FirewallFilterRule, error)
	Get(ctx context.Context, ruleID string) (*FirewallFilterRule, error)
	Create(ctx context.Context, expression, action, description string) ([]*FirewallFilterRule, error)
	Update(ctx context.Context, ruleID, expression, action, description string) (*FirewallFilterRule, error)
	Delete(ctx context.Context, ruleID string) error
}

type WAFServicer interface {
	ListPackages(ctx context.Context) ([]*WAFPackage, error)
	GetPackage(ctx context.Context, packageID string) (*WAFPackage, error)
	ListRules(ctx context.Context, packageID string) ([]*WAFRule, error)
	GetRule(ctx context.Context, packageID, ruleID string) (*WAFRule, error)
	UpdateRule(ctx context.Context, packageID, ruleID, mode string) (*WAFRule, error)
	ListAccessRules(ctx context.Context) ([]*FirewallRule, error)
	CreateAccessRule(ctx context.Context, target, value, mode, notes string) (*FirewallRule, error)
	DeleteAccessRule(ctx context.Context, ruleID string) error
}

type HealthcheckServicer interface {
	List(ctx context.Context) ([]*Healthcheck, error)
	Get(ctx context.Context, id string) (*Healthcheck, error)
	Create(ctx context.Context, name, address string, opts ...HealthcheckOption) (*Healthcheck, error)
	Update(ctx context.Context, id string, opts ...HealthcheckOption) (*Healthcheck, error)
	Delete(ctx context.Context, id string) error
}

type DomainServicer interface {
	List(ctx context.Context, opts DomainListOptions) ([]*DomainStatus, *Pagination, error)
	GetDetail(ctx context.Context, zoneID string) (*DomainDetail, error)
	EnrichWithHealth(ctx context.Context, domains []*DomainStatus)
}

type DoctorServicer interface {
	CheckDNSPropagation(ctx context.Context, domain string) (*DNSPropagationResult, error)
	CheckSSL(ctx context.Context, domain string) (*SSLProbeResult, error)
	CheckHTTP(ctx context.Context, domain string) (*HTTPProbeResult, error)
	CheckNameservers(ctx context.Context, domain string, expected []string) (*NSProbeResult, error)
	RunDiagnostics(ctx context.Context, domain string, expectedNS []string) (*DiagnosticReport, error)
}

// BucketDomainServicer manages R2 bucket custom domains.
type BucketDomainServicer interface {
	Attach(ctx context.Context, bucket string, req AttachBucketDomainRequest) (*BucketDomain, error)
	List(ctx context.Context, bucket string) ([]BucketDomain, error)
	Get(ctx context.Context, bucket, domain string) (*BucketDomain, error)
	Update(ctx context.Context, bucket, domain string, req UpdateBucketDomainRequest) (*BucketDomain, error)
	Detach(ctx context.Context, bucket, domain string) error
	Verify(ctx context.Context, bucket, domain string, timeout time.Duration) (*BucketDomain, error)
}

var _ BucketDomainServicer = (*BucketDomainService)(nil)

// Compile-time interface satisfaction checks
var (
	_ WorkerServicer      = (*WorkerService)(nil)
	_ KVServicer          = (*KVService)(nil)
	_ DNSServicer         = (*DNSService)(nil)
	_ ZoneServicer        = (*ZoneService)(nil)
	_ SSLServicer         = (*SSLService)(nil)
	_ CacheServicer       = (*CacheService)(nil)
	_ CORSServicer        = (*CORSService)(nil)
	_ FirewallServicer    = (*FirewallService)(nil)
	_ WAFServicer         = (*WAFService)(nil)
	_ HealthcheckServicer = (*HealthcheckService)(nil)
	_ DomainServicer      = (*DomainService)(nil)
	_ DoctorServicer      = (*DoctorService)(nil)
)
