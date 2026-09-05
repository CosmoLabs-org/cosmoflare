package cosmoflare

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
	endpoint     string
	accessKey    string
	secretKey    string
	region       string
	httpClient   *http.Client
	timeout      time.Duration
	cacheControl bool
	projectCfg   *ProjectConfig
}

// WithProfile selects a named profile from ~/.r2go2/config.yaml. The profile
// supplies the account ID and API token only when they were not already set
// by an explicit option or the CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN
// environment variables ("options > env > profile" precedence).
func WithProfile(name string) ClientOption {
	return func(c *clientConfig) { c.profile = name }
}

// WithCacheControl enables automatic cache header injection on uploads.
func WithCacheControl(enabled bool) ClientOption {
	return func(c *clientConfig) { c.cacheControl = enabled }
}

// WithHTTPClient sets the HTTP client used for every API call this client
// makes, covering both the Cloudflare API and the R2 S3 endpoint. When set,
// it takes precedence over WithTimeout.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithTimeout sets the overall request timeout applied to every API call
// (default 30s). It is ignored when WithHTTPClient supplies an explicit
// client; set the timeout on that client instead.
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

// WithAccountID sets the Cloudflare account ID directly.
func WithAccountID(id string) ClientOption {
	return func(c *clientConfig) { c.accountID = id }
}

// WithProjectConfig attaches the project-level configuration
// (.cosmoflare.yaml / .r2go2.yaml) to the client. When the config declares
// guardrail rules (allowed_buckets, max_file_size, blocked_keys), every
// upload through this client is validated against them and rejected with an
// error before any data is sent. A nil config or a config with no guardrail
// rules enforces nothing.
func WithProjectConfig(cfg *ProjectConfig) ClientOption {
	return func(c *clientConfig) { c.projectCfg = cfg }
}

// WithAPIToken sets the Cloudflare API token directly.
func WithAPIToken(token string) ClientOption {
	return func(c *clientConfig) { c.apiToken = token }
}
