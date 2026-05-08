package r2go2

import (
	"net/http"
	"time"
)

// ClientOption is a functional option for configuring an R2Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	accountID    string
	apiToken     string
	profile      string
	bucket       string
	endpoint     string
	accessKey    string
	secretKey    string
	region       string
	httpClient   *http.Client
	timeout      time.Duration
	cacheControl bool
	auditLog     string
	dryRun       bool
}

// WithProfile sets the named profile to load from ~/.r2go2/config.yaml.
func WithProfile(name string) ClientOption {
	return func(c *clientConfig) { c.profile = name }
}

// WithBucket sets a default bucket for operations.
func WithBucket(name string) ClientOption {
	return func(c *clientConfig) { c.bucket = name }
}

// WithCacheControl enables automatic cache header injection on uploads.
func WithCacheControl(enabled bool) ClientOption {
	return func(c *clientConfig) { c.cacheControl = enabled }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithTimeout sets the overall request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) { c.timeout = d }
}

// WithEndpoint sets a custom R2 endpoint (overrides default).
func WithEndpoint(url string) ClientOption {
	return func(c *clientConfig) { c.endpoint = url }
}

// WithCredentials sets explicit access key / secret key credentials.
func WithCredentials(accessKey, secretKey string) ClientOption {
	return func(c *clientConfig) {
		c.accessKey = accessKey
		c.secretKey = secretKey
	}
}

// WithRegion sets the AWS region (default: "auto").
func WithRegion(region string) ClientOption {
	return func(c *clientConfig) { c.region = region }
}

// WithAuditLog enables JSONL audit logging to the specified path.
func WithAuditLog(path string) ClientOption {
	return func(c *clientConfig) { c.auditLog = path }
}

// WithDryRun enables dry-run mode (no actual API calls).
func WithDryRun(enabled bool) ClientOption {
	return func(c *clientConfig) { c.dryRun = enabled }
}

// WithAccountID sets the Cloudflare account ID directly.
func WithAccountID(id string) ClientOption {
	return func(c *clientConfig) { c.accountID = id }
}

// WithAPIToken sets the Cloudflare API token directly.
func WithAPIToken(token string) ClientOption {
	return func(c *clientConfig) { c.apiToken = token }
}
