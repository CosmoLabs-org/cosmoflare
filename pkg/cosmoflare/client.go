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
// (the Cloudflare API client and the R2 S3 client). Otherwise the Cloudflare
// API control plane uses a client carrying the WithTimeout timeout (default
// 30s), while the S3 data plane gets a separate client with NO whole-request
// timeout — large transfers are bounded by context deadlines and SDK retries,
// not the API timeout (BUG-042).
func NewClient(opts ...ClientOption) (R2Client, error) {
	cfg := &clientConfig{
		region:     "auto",
		timeout:    30 * time.Second,
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

	// HTTP client shared by both transports.
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	// Cloudflare API client
	cfAPI, err := cloudflare.NewWithAPIToken(cfg.apiToken, cloudflare.HTTPClient(httpClient))
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

	// BUG-042: the S3 data plane must not carry the control-plane's
	// whole-request timeout (default 30s) — large transfers die mid-body.
	// Transfer limits come from context deadlines; retries come from the SDK.
	// An explicitly provided WithHTTPClient is honored on both transports
	// (caller's deliberate choice).
	dataPlaneClient := c.httpClient
	if c.cfg.httpClient == nil {
		dataPlaneClient = &http.Client{}
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(c.cfg.region),
		awsconfig.WithHTTPClient(dataPlaneClient),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			if accessKey != "" && secretKey != "" {
				return aws.Credentials{
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
				}, nil
			}
			// Use static credentials from the API token
			return aws.Credentials{
				AccessKeyID:     "r2-token",
				SecretAccessKey: c.apiToken,
				Source:          "Cloudflare-R2-API",
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
func (c *client) s3Client() *s3.Client { return c.s3 }
func (c *client) cfClient() *cloudflare.API { return c.cf }
