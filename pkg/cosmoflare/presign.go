package cosmoflare

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignGetObject generates a pre-signed URL for temporary download access.
func (c *client) PresignGetObject(ctx context.Context, bucket, key string, expiresIn time.Duration) (string, error) {
	if err := validateBucketName(bucket); err != nil {
		return "", validationError("PresignGetObject", err.Error())
	}
	if key == "" {
		return "", validationError("PresignGetObject", "object key is required")
	}
	if expiresIn <= 0 {
		return "", validationError("PresignGetObject", "expires duration must be positive")
	}

	presigner := s3.NewPresignClient(c.s3)
	result, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: ptrString(bucket),
		Key:    ptrString(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", newError("PresignGetObject", "failed to generate presigned URL", err)
	}

	return result.URL, nil
}

func ptrString(s string) *string { return &s }
