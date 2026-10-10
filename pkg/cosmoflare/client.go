package cosmoflare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// R2Client is the primary interface for R2 storage operations.
// It provides bucket CRUD, object CRUD, upload, download, and copy operations.
type R2Client interface {
	// Bucket operations
	CreateBucket(ctx context.Context, name string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]*Bucket, error)
	GetBucket(ctx context.Context, name string) (*Bucket, error)
	DeleteBucket(ctx context.Context, name string) error
	BucketExists(ctx context.Context, name string) (bool, error)

	// Object operations
	ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continuationToken string) (*ListResult[*Object], error)
	GetObject(ctx context.Context, bucket, key string) (*DownloadResult, error)
	HeadObject(ctx context.Context, bucket, key string) (*HeadResult, error)
	DeleteObject(ctx context.Context, bucket, key string) error

	// Upload / Download
	Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error)
	MultipartUpload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error)
	ResumableMultipartUpload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error)
	ResumeMultipartUpload(ctx context.Context, bucket, key string, reader io.ReadSeeker, size int64, opts ...UploadOption) (*UploadResult, error)
	Download(ctx context.Context, bucket, key string, opts ...DownloadOption) (*DownloadResult, error)

	// Copy
	CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) (*CopyResult, error)

	// Pre-signed URLs
	PresignGetObject(ctx context.Context, bucket, key string, expiresIn time.Duration) (string, error)

	// Metadata
	AccountID() string
	TestConnection(ctx context.Context) error
}

// client implements R2Client.
type client struct {
	cf         *cloudflare.API
	s3         *s3.Client
	accountID  string
	apiToken   string
	httpClient *http.Client
	cfg        *clientConfig
	guardrails *GuardrailChecker
}

// NewClient creates a new R2Client using functional options.
//
// Credentials are resolved with "options > env > profile" precedence:
// WithAccountID/WithAPIToken first, then the CLOUDFLARE_ACCOUNT_ID /
// CLOUDFLARE_API_TOKEN environment variables, then — only when an account
// ID or API token is still missing — the named profile from
// the machine config (~/.cosmoflare/config.yaml, legacy ~/.r2go2/config.yaml
// read for compatibility) selected via WithProfile.
//
// Transport policy: an explicit WithHTTPClient is used by both transports
// (the Cloudflare API client and the R2 S3 client) — the Cloudflare-side
// copy is wrapped with the knowledge endpoint registry (FEAT-044
// wrap-always; the S3 side keeps the caller's original client). Otherwise
// the Cloudflare API control plane uses a client carrying the WithTimeout
// timeout (default 30s) plus knowledge.Transport, while the S3 data plane
// gets a separate client with NO whole-request timeout and NO registry —
// large transfers are bounded by context deadlines and SDK retries, not
// the API timeout (BUG-042).
func NewClient(opts ...ClientOption) (R2Client, error) {
	cfg := &clientConfig{
		region:       "auto",
		timeout:      30 * time.Second,
		cacheControl: true,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Resolve account ID and API token: options > env > profile
	if cfg.accountID == "" {
		cfg.accountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	}
	if cfg.apiToken == "" {
		cfg.apiToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	}
	if cfg.profile != "" && (cfg.accountID == "" || cfg.apiToken == "") {
		profile, err := loadNamedProfile(cfg.profile)
		if err != nil {
			return nil, err
		}
		if cfg.accountID == "" {
			cfg.accountID = profile.AccountID
		}
		if cfg.apiToken == "" {
			cfg.apiToken = profile.APIToken
		}
	}
	if cfg.accountID == "" {
		return nil, validationError("NewClient", "CLOUDFLARE_ACCOUNT_ID is required (set env or use WithAccountID)")
	}
	if cfg.apiToken == "" {
		return nil, validationError("NewClient", "CLOUDFLARE_API_TOKEN is required (set env or use WithAPIToken)")
	}

	// Control-plane HTTP client (FEAT-044): it always carries the
	// knowledge endpoint registry. Default construction bakes it in; a
	// caller-supplied WithHTTPClient is WRAPPED, not replaced — the caller
	// keeps their tuning (timeout, dialer) and the registry still applies.
	// Bypass is documented: pass a client whose Transport is already a
	// bare RoundTripper of your own. A shallow copy avoids mutating a
	// caller-owned (possibly shared) http.Client in place; the original
	// stays reserved for the R2 data plane below.
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout, Transport: &RetryTransport{Base: &knowledge.Transport{}}}
	} else {
		wrapped := *httpClient
		wrapped.Transport = &RetryTransport{Base: &knowledge.Transport{Base: httpClient.Transport}}
		httpClient = &wrapped
	}

	// Cloudflare API client (built at the transport chokepoint)
	cfAPI, err := newCloudflareAPIWithClient(cfg.apiToken, httpClient)
	if err != nil {
		return nil, authError("NewClient", "failed to create Cloudflare API client", err)
	}

	// Guardrails are only active when a project config is attached; a nil
	// checker enforces nothing (previous behavior for library callers).
	var guardrails *GuardrailChecker
	if cfg.projectCfg != nil {
		guardrails = NewGuardrailChecker(cfg.projectCfg)
	}

	c := &client{
		cf:         cfAPI,
		accountID:  cfg.accountID,
		apiToken:   cfg.apiToken,
		httpClient: httpClient,
		cfg:        cfg,
		guardrails: guardrails,
	}

	// S3 client for object operations
	if err := c.initS3(); err != nil {
		return nil, fmt.Errorf("cosmoflare: init S3 client: %w", err)
	}

	return c, nil
}

// initS3 sets up the AWS S3 SDK v2 client pointed at the R2 endpoint.
func (c *client) initS3() error {
	endpoint := c.cfg.endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.accountID)
	}

	accessKey := c.cfg.accessKey
	secretKey := c.cfg.secretKey

	// BUG-p3Y31ZQ: R2's S3 gateway only accepts 32-char Access Key IDs from
	// R2 API tokens; a Cloudflare API token cannot sign S3 requests. The old
	// fallback fabricated an 8-char AccessKeyID, so every token-only user's
	// data-plane call died in "Credential access key has length 8, should
	// be 32".
	const r2AccessKeyLen = 32
	switch {
	case accessKey != "" && secretKey != "":
		if len(accessKey) != r2AccessKeyLen {
			return validationError("NewClient",
				fmt.Sprintf("R2 access_key must be %d characters (got %d) — use the Access Key ID from an R2 API token (Cloudflare dashboard → R2 → Manage R2 API Tokens)", r2AccessKeyLen, len(accessKey)))
		}
	case accessKey != "" || secretKey != "":
		return validationError("NewClient",
			"R2 S3 credentials are incomplete: set both access_key and secret_key, or neither (control-plane-only use needs no S3 keys)")
	}

	// BUG-042: the S3 data plane must not carry the control-plane's
	// whole-request timeout (default 30s) — large transfers die mid-body.
	// Transfer limits come from context deadlines; retries come from the SDK.
	// An explicitly provided WithHTTPClient is honored on both transports
	// (caller's deliberate choice), but the data plane gets the caller's
	// ORIGINAL client: c.httpClient is the knowledge-wrapped control-plane
	// copy, and the registry never applies to S3 transfers (FEAT-044).
	dataPlaneClient := c.cfg.httpClient
	if dataPlaneClient == nil {
		dataPlaneClient = &http.Client{}
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(c.cfg.region),
		awsconfig.WithHTTPClient(dataPlaneClient),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			if accessKey == "" {
				// Token-only config: the control plane works, but the S3 data
				// plane cannot — a Cloudflare API token cannot sign SigV4
				// requests. Fail the first object operation with how to fix
				// instead of the gateway's opaque InvalidArgument.
				return aws.Credentials{}, validationError("S3",
					"R2 object operations need an R2 API token's Access Key ID (32 chars) and Secret Access Key — a Cloudflare API token cannot sign S3 requests. Create an R2 API token (Cloudflare dashboard → R2 → Manage R2 API Tokens → Create API token, Object Read & Write), then set access_key/secret_key in ~/.r2go2/config.yaml or export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
			}
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	c.s3 = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return nil
}

func (c *client) AccountID() string { return c.accountID }

// loadNamedProfile resolves a profile by name from the machine config
// (~/.r2go2/config.yaml). It returns a validation error naming the profile
// when the config cannot be read or the profile does not exist.
func loadNamedProfile(name string) (*ProfileConfig, error) {
	mc, err := LoadMachineConfig()
	if err != nil {
		return nil, validationError("NewClient",
			fmt.Sprintf("failed to read machine config for profile %q: %v", name, err))
	}
	profile, err := mc.GetProfile(name)
	if err != nil {
		return nil, validationError("NewClient",
			fmt.Sprintf("profile %q not found in ~/.r2go2/config.yaml (create it with `cosmoflare account add` or use WithAccountID/WithAPIToken)", name))
	}
	return profile, nil
}

func (c *client) TestConnection(ctx context.Context) error {
	if c.cf == nil {
		return authError("TestConnection", "Cloudflare API client not initialized", nil)
	}
	_, err := c.cf.ListR2Buckets(ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.ListR2BucketsParams{})
	if err != nil {
		return newError("TestConnection", "failed to connect to R2", err)
	}
	return nil
}

// Convenience: s3Client returns the underlying S3 client for CopyObject.
func (c *client) s3Client() *s3.Client      { return c.s3 }
func (c *client) cfClient() *cloudflare.API { return c.cf }
