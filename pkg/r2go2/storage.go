package r2go2

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cloudflare/cloudflare-go"
)

// CreateBucket creates a new R2 bucket.
func (c *client) CreateBucket(ctx context.Context, name string) (*Bucket, error) {
	if err := validateBucketName(name); err != nil {
		return nil, validationError("CreateBucket", err.Error())
	}

	result, err := c.cfClient().CreateR2Bucket(ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.CreateR2BucketParameters{
		Name: name,
	})
	if err != nil {
		return nil, newError("CreateBucket", "failed to create bucket", err)
	}

	return &Bucket{
		Name:      result.Name,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// ListBuckets returns all R2 buckets in the account.
func (c *client) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	buckets, err := c.cfClient().ListR2Buckets(ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.ListR2BucketsParams{})
	if err != nil {
		return nil, newError("ListBuckets", "failed to list buckets", err)
	}

	result := make([]*Bucket, 0, len(buckets))
	for _, b := range buckets {
		bucket := &Bucket{
			Name: b.Name,
		}
		if b.CreationDate != nil {
			bucket.CreatedAt = *b.CreationDate
		}
		result = append(result, bucket)
	}
	return result, nil
}

// GetBucket returns details about a specific bucket.
func (c *client) GetBucket(ctx context.Context, name string) (*Bucket, error) {
	if err := validateBucketName(name); err != nil {
		return nil, validationError("GetBucket", err.Error())
	}

	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		if b.Name == name {
			return b, nil
		}
	}
	return nil, notFound("GetBucket", name, "", nil)
}

// DeleteBucket deletes an R2 bucket.
func (c *client) DeleteBucket(ctx context.Context, name string) error {
	if err := validateBucketName(name); err != nil {
		return validationError("DeleteBucket", err.Error())
	}

	err := c.cfClient().DeleteR2Bucket(ctx, cloudflare.AccountIdentifier(c.accountID), name)
	if err != nil {
		return newError("DeleteBucket", "failed to delete bucket", err)
	}
	return nil
}

// BucketExists checks whether a bucket exists in the account.
func (c *client) BucketExists(ctx context.Context, name string) (bool, error) {
	if err := validateBucketName(name); err != nil {
		return false, validationError("BucketExists", err.Error())
	}

	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		return false, err
	}
	for _, b := range buckets {
		if b.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// ListObjects lists objects in a bucket with optional prefix, delimiter, and max keys.
func (c *client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32) (*ListResult[*Object], error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("ListObjects", err.Error())
	}

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
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

	result, err := c.s3Client().ListObjectsV2(ctx, params)
	if err != nil {
		return nil, newError("ListObjects", "failed to list objects", err)
	}

	items := make([]*Object, 0, len(result.Contents))
	for _, obj := range result.Contents {
		items = append(items, &Object{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		})
	}

	return &ListResult[*Object]{
		Items:       items,
		NextToken:   aws.ToString(result.NextContinuationToken),
		IsTruncated: aws.ToBool(result.IsTruncated),
	}, nil
}

// GetObject downloads an object's content.
func (c *client) GetObject(ctx context.Context, bucket, key string) (*DownloadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("GetObject", err.Error())
	}
	if key == "" {
		return nil, validationError("GetObject", "object key is required")
	}

	result, err := c.s3Client().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, notFound("GetObject", bucket, key, err)
		}
		return nil, newError("GetObject", "failed to get object", err)
	}

	return &DownloadResult{
		Key:         key,
		Bucket:      bucket,
		Size:        aws.ToInt64(result.ContentLength),
		Content:     result.Body,
		ContentType: aws.ToString(result.ContentType),
		Metadata:    result.Metadata,
	}, nil
}

// HeadObject retrieves object metadata without downloading the body.
func (c *client) HeadObject(ctx context.Context, bucket, key string) (*HeadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("HeadObject", err.Error())
	}
	if key == "" {
		return nil, validationError("HeadObject", "object key is required")
	}

	result, err := c.s3Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, notFound("HeadObject", bucket, key, err)
		}
		return nil, newError("HeadObject", "failed to head object", err)
	}

	return &HeadResult{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
		ContentType:  aws.ToString(result.ContentType),
		CacheControl: aws.ToString(result.CacheControl),
		Metadata:     result.Metadata,
	}, nil
}

// DeleteObject deletes an object from a bucket.
func (c *client) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := validateBucketName(bucket); err != nil {
		return validationError("DeleteObject", err.Error())
	}
	if key == "" {
		return validationError("DeleteObject", "object key is required")
	}

	_, err := c.s3Client().DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return newError("DeleteObject", "failed to delete object", err)
	}
	return nil
}

// CopyObject copies an object between buckets (or within a bucket).
func (c *client) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) (*CopyResult, error) {
	if err := validateBucketName(srcBucket); err != nil {
		return nil, validationError("CopyObject", "source: "+err.Error())
	}
	if err := validateBucketName(dstBucket); err != nil {
		return nil, validationError("CopyObject", "destination: "+err.Error())
	}
	if srcKey == "" || dstKey == "" {
		return nil, validationError("CopyObject", "source and destination keys are required")
	}

	copySource := fmt.Sprintf("%s/%s", srcBucket, srcKey)
	result, err := c.s3Client().CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return nil, newError("CopyObject", "failed to copy object", err)
	}

	etag := ""
	if result.CopyObjectResult != nil {
		etag = aws.ToString(result.CopyObjectResult.ETag)
	}

	return &CopyResult{
		Key:       dstKey,
		SourceKey: srcKey,
		Bucket:    dstBucket,
		ETag:      etag,
	}, nil
}

// validateBucketName validates an R2 bucket name per S3 naming rules.
func validateBucketName(name string) error {
	if name == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters (got %d)", len(name))
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.') {
			return fmt.Errorf("bucket name contains invalid character: %q", ch)
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("bucket name cannot start or end with hyphen or dot")
	}
	return nil
}
