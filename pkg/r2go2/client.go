package r2go2

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

// R2Client is the primary interface for R2Go2 operations.
// It provides bucket CRUD, object CRUD, upload, download, and copy operations.
type R2Client interface {
	// Bucket operations
	CreateBucket(ctx context.Context, name string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]*Bucket, error)
	GetBucket(ctx context.Context, name string) (*Bucket, error)
	DeleteBucket(ctx context.Context, name string) error
	BucketExists(ctx context.Context, name string) (bool, error)

	// Object operations
	ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32) (*ListResult[*Object], error)
	GetObject(ctx context.Context, bucket, key string) (*DownloadResult, error)
	HeadObject(ctx context.Context, bucket, key string) (*HeadResult, error)
	DeleteObject(ctx context.Context, bucket, key string) error

	// Upload / Download
	Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error)
	Download(ctx context.Context, bucket, key string, opts ...DownloadOption) (*DownloadResult, error)

	// Copy
	CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) (*CopyResult, error)

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
}

// NewClient creates a new R2Client using functional options.
//
// By default it reads credentials from environment variables
// (CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_API_TOKEN). Use WithProfile,
// WithAccountID, WithAPIToken, or WithCredentials to override.
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
	if cfg.accountID == "" {
		return nil, validationError("NewClient", "CLOUDFLARE_ACCOUNT_ID is required (set env or use WithAccountID)")
	}
	if cfg.apiToken == "" {
		return nil, validationError("NewClient", "CLOUDFLARE_API_TOKEN is required (set env or use WithAPIToken)")
	}

	// HTTP client
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	// Cloudflare API client
	cfAPI, err := cloudflare.NewWithAPIToken(cfg.apiToken)
	if err != nil {
		return nil, authError("NewClient", "failed to create Cloudflare API client", err)
	}

	c := &client{
		cf:         cfAPI,
		accountID:  cfg.accountID,
		apiToken:   cfg.apiToken,
		httpClient: httpClient,
		cfg:        cfg,
	}

	// S3 client for object operations
	if err := c.initS3(); err != nil {
		return nil, fmt.Errorf("r2go2: init S3 client: %w", err)
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

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(c.cfg.region),
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
