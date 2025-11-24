/*
Package api provides enhanced R2 API client functionality for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	r2config "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"
)

// Client wraps the R2 API functionality with both Cloudflare API and S3 SDK
type Client struct {
	ctx         context.Context
	cf          *cloudflare.API
	s3          *s3.Client
	profile     *r2config.Profile
	accountID   string
	r2Endpoint  string
	httpClient  *http.Client
}

// ClientOptions represents options for creating a new client
type ClientOptions struct {
	Profile    *r2config.Profile
	AccountID  string
	APIToken   string
	HTTPClient *http.Client
}

// NewClient creates a new R2 API client
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("client options are required")
	}

	// Default HTTP client
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	client := &Client{
		ctx:        context.Background(),
		profile:    opts.Profile,
		accountID:  opts.AccountID,
		httpClient: httpClient,
	}

	// Initialize Cloudflare API client
	if opts.APIToken != "" {
		cfAPI, err := cloudflare.NewWithAPIToken(opts.APIToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
		}
		client.cf = cfAPI
	} else if opts.Profile != nil && opts.Profile.APIToken != "" {
		cfAPI, err := cloudflare.NewWithAPIToken(opts.Profile.APIToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
		}
		client.cf = cfAPI
	}

	// Initialize S3 client for object operations
	if err := client.initS3Client(); err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return client, nil
}

// NewClientFromProfile creates a new client from a profile
func NewClientFromProfile(profileName string) (*Client, error) {
	configMgr, err := r2config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	var profile *r2config.Profile
	if profileName != "" {
		profile, err = configMgr.GetProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("failed to get profile '%s': %w", profileName, err)
		}
	} else {
		profile, err = configMgr.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("failed to get current profile: %w", err)
		}
	}

	opts := &ClientOptions{
		Profile:   profile,
		AccountID: profile.AccountID,
		APIToken:  profile.APIToken,
	}

	return NewClient(opts)
}

// NewClientFromEnv creates a new client from environment variables
func NewClientFromEnv() (*Client, error) {
	profile := r2config.LoadFromEnvironment()

	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")

	if accountID == "" || apiToken == "" {
		return nil, fmt.Errorf("CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN environment variables are required")
	}

	opts := &ClientOptions{
		Profile:   profile,
		AccountID: accountID,
		APIToken:  apiToken,
	}

	return NewClient(opts)
}

// initS3Client initializes the S3 client for R2 operations
func (c *Client) initS3Client() error {
	// R2 endpoint pattern
	if c.r2Endpoint == "" {
		c.r2Endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.accountID)
	}

	// Override endpoint if provided in profile
	if c.profile != nil && c.profile.Endpoint != "" {
		c.r2Endpoint = c.profile.Endpoint
	}

	// AWS SDK v2 configuration
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			// Use profile credentials if available, otherwise get from R2
			if c.profile != nil && c.profile.AccessKey != "" && c.profile.SecretKey != "" {
				return aws.Credentials{
					AccessKeyID:     c.profile.AccessKey,
					SecretAccessKey: c.profile.SecretKey,
				}, nil
			}

			// Get R2 credentials using Cloudflare API
			return c.getR2Credentials()
		})),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with R2 endpoint
	c.s3 = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.r2Endpoint)
	})

	return nil
}

// getR2Credentials gets temporary R2 credentials using Cloudflare API
func (c *Client) getR2Credentials() (aws.Credentials, error) {
	if c.cf == nil {
		return aws.Credentials{}, fmt.Errorf("Cloudflare API client not initialized")
	}

	// Note: This is a simplified implementation
	// In practice, you'd use the Cloudflare API to get R2 credentials
	// For now, we'll return empty credentials and rely on the API token

	return aws.Credentials{
		AccessKeyID:     "r2-token", // Placeholder
		SecretAccessKey: c.profile.APIToken,
		Source:          "Cloudflare-R2-API",
	}, nil
}

// Bucket represents an R2 bucket with extended metadata
type Bucket struct {
	Name        string    `json:"name"`
	CreatedDate time.Time `json:"creation_date"`
	Size        int64     `json:"size,omitempty"`
	ObjectCount int64     `json:"object_count,omitempty"`
	Location    string    `json:"location,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// CreateBucket creates a new R2 bucket using Cloudflare API
func (c *Client) CreateBucket(name string) (*Bucket, error) {
	if err := validateBucketName(name); err != nil {
		return nil, fmt.Errorf("invalid bucket name: %w", err)
	}

	if c.cf == nil {
		return nil, fmt.Errorf("Cloudflare API client not initialized")
	}

	// Use Cloudflare API to create bucket
	result, err := c.cf.CreateR2Bucket(c.ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.CreateR2BucketParameters{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	bucket := &Bucket{
		Name:        result.Name,
		CreatedDate: time.Now().UTC(),
	}

	return bucket, nil
}

// ListBuckets lists all R2 buckets
func (c *Client) ListBuckets() ([]*Bucket, error) {
	if c.cf == nil {
		return nil, fmt.Errorf("Cloudflare API client not initialized")
	}

	buckets, err := c.cf.ListR2Buckets(c.ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.ListR2BucketsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var result []*Bucket
	for _, bucket := range buckets {
		result = append(result, &Bucket{
			Name:        bucket.Name,
			CreatedDate: bucket.Created, // Use Created instead of CreatedOn
		})
	}

	return result, nil
}

// GetBucket gets detailed information about a bucket
func (c *Client) GetBucket(name string) (*Bucket, error) {
	if err := validateBucketName(name); err != nil {
		return nil, fmt.Errorf("invalid bucket name: %w", err)
	}

	if c.cf == nil {
		return nil, fmt.Errorf("Cloudflare API client not initialized")
	}

	// Get bucket details from Cloudflare API
	// Note: Cloudflare API may not have detailed bucket info endpoint
	// We'll combine info from multiple sources

	bucket := &Bucket{
		Name:        name,
		CreatedDate: time.Time{}, // Would need to get from API
	}

	// Try to get additional info via S3 API
	if c.s3 != nil {
		// Get bucket location
		location, err := c.s3.GetBucketLocation(c.ctx, &s3.GetBucketLocationInput{
			Bucket: aws.String(name),
		})
		if err == nil {
			// Handle the location constraint (it might be nil for some buckets)
			if location.LocationConstraint != nil {
				bucket.Location = string(*location.LocationConstraint)
			} else {
				bucket.Location = "us-east-1" // Default location
			}
		}
	}

	return bucket, nil
}

// DeleteBucket deletes an R2 bucket
func (c *Client) DeleteBucket(name string) error {
	if err := validateBucketName(name); err != nil {
		return fmt.Errorf("invalid bucket name: %w", err)
	}

	if c.cf == nil {
		return fmt.Errorf("Cloudflare API client not initialized")
	}

	err := c.cf.DeleteR2Bucket(c.ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.DeleteR2BucketParameters{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// BucketExists checks if a bucket exists
func (c *Client) BucketExists(name string) (bool, error) {
	if err := validateBucketName(name); err != nil {
		return false, fmt.Errorf("invalid bucket name: %w", err)
	}

	buckets, err := c.ListBuckets()
	if err != nil {
		return false, err
	}

	for _, bucket := range buckets {
		if bucket.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// Object represents an R2 object
type Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	StorageClass string            `json:"storage_class"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ListObjects lists objects in a bucket
func (c *Client) ListObjects(bucketName string, prefix, delimiter string, maxKeys int32) ([]*Object, error) {
	if c.s3 == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}

	if prefix != "" {
		params.Prefix = aws.String(prefix)
	}
	if delimiter != "" {
		params.Delimiter = aws.String(delimiter)
	}
	if maxKeys > 0 {
		params.MaxKeys = aws.Int32(maxKeys)
	}

	result, err := c.s3.ListObjectsV2(c.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var objects []*Object
	for _, obj := range result.Contents {
		objects = append(objects, &Object{
			Key:          *obj.Key,
			Size:         obj.Size,
			LastModified: *obj.LastModified,
			ETag:         *obj.ETag,
			StorageClass: string(obj.StorageClass),
		})
	}

	return objects, nil
}

// GetObject gets an object from a bucket
func (c *Client) GetObject(bucketName, key string, rangeStart, rangeEnd int64) (io.ReadCloser, *Object, error) {
	if c.s3 == nil {
		return nil, nil, fmt.Errorf("S3 client not initialized")
	}

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}

	if rangeStart >= 0 && rangeEnd > rangeStart {
		params.Range = aws.String(fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	}

	result, err := c.s3.GetObject(c.ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	object := &Object{
		Key:          key,
		Size:         result.ContentLength,
		LastModified: *result.LastModified,
		ETag:         *result.ETag,
		Metadata:     result.Metadata,
	}

	return result.Body, object, nil
}

// HeadObject gets object metadata without downloading the content
func (c *Client) HeadObject(bucketName, key string) (*Object, error) {
	if c.s3 == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	result, err := c.s3.HeadObject(c.ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	object := &Object{
		Key:          key,
		Size:         result.ContentLength,
		LastModified: *result.LastModified,
		ETag:         *result.ETag,
		Metadata:     result.Metadata,
	}

	return object, nil
}

// DeleteObject deletes an object from a bucket
func (c *Client) DeleteObject(bucketName, key string) error {
	if c.s3 == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	_, err := c.s3.DeleteObject(c.ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// TestConnection tests the connection to Cloudflare API and R2
func (c *Client) TestConnection() error {
	if c.cf == nil {
		return fmt.Errorf("Cloudflare API client not initialized")
	}

	// Test Cloudflare API connection by trying to list accounts
	_, err := c.cf.Accounts(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Cloudflare API: %w", err)
	}

	// Test S3/R2 connection by listing buckets
	_, err = c.ListBuckets()
	if err != nil {
		return fmt.Errorf("failed to connect to R2: %w", err)
	}

	return nil
}

// GetAccountID returns the current account ID
func (c *Client) GetAccountID() string {
	return c.accountID
}

// GetProfile returns the current profile
func (c *Client) GetProfile() *r2config.Profile {
	return c.profile
}

// GetCloudflareClient returns the Cloudflare API client
func (c *Client) GetCloudflareClient() *cloudflare.API {
	return c.cf
}

// validateBucketName validates R2 bucket name according to AWS S3 naming rules
func validateBucketName(name string) error {
	if name == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters")
	}
	// Check for invalid characters
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			return fmt.Errorf("bucket name can only contain lowercase letters, numbers, hyphens, and dots")
		}
	}
	// Cannot start or end with hyphen or dot
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("bucket name cannot start or end with hyphen or dot")
	}
	return nil
}