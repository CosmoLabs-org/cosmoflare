package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// DownloadOption configures a download operation.
type DownloadOption func(*downloadConfig)

type downloadConfig struct {
	outputPath string
	rangeStart int64
	rangeEnd   int64
}

// WithOutputPath writes the downloaded content to a local file.
func WithOutputPath(path string) DownloadOption {
	return func(c *downloadConfig) { c.outputPath = path }
}

// WithRange downloads a specific byte range.
func WithRange(start, end int64) DownloadOption {
	return func(c *downloadConfig) {
		c.rangeStart = start
		c.rangeEnd = end
	}
}

// Download downloads an object from R2.
func (c *client) Download(ctx context.Context, bucket, key string, opts ...DownloadOption) (*DownloadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("Download", err.Error())
	}
	if key == "" {
		return nil, validationError("Download", "object key is required")
	}

	cfg := &downloadConfig{}
	for _, o := range opts {
		o(cfg)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if cfg.rangeStart >= 0 && cfg.rangeEnd > cfg.rangeStart {
		input.Range = aws.String(fmt.Sprintf("bytes=%d-%d", cfg.rangeStart, cfg.rangeEnd))
	}

	result, err := c.s3Client().GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, notFound("Download", bucket, key, err)
		}
		return nil, newError("Download", "failed to download object", err)
	}

	// If output path specified, write to file and close body
	if cfg.outputPath != "" {
		if err := writeToFile(result.Body, cfg.outputPath); err != nil {
			return nil, newError("Download", "failed to write file", err)
		}
		return &DownloadResult{
			Key:         key,
			Bucket:      bucket,
			Size:        aws.ToInt64(result.ContentLength),
			ContentType: aws.ToString(result.ContentType),
			Metadata:    result.Metadata,
		}, nil
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

func writeToFile(body io.ReadCloser, path string) error {
	defer body.Close()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	return err
}
