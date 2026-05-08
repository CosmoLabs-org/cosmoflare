package r2go2

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadOption configures an upload operation.
type UploadOption func(*uploadConfig)

type uploadConfig struct {
	contentType  string
	cacheControl string
	metadata     map[string]string
}

// WithContentType sets the Content-Type header for the upload.
func WithContentType(ct string) UploadOption {
	return func(c *uploadConfig) { c.contentType = ct }
}

// WithUploadCacheControl sets the Cache-Control header for the upload.
func WithUploadCacheControl(cc string) UploadOption {
	return func(c *uploadConfig) { c.cacheControl = cc }
}

// WithMetadata sets custom metadata for the uploaded object.
func WithMetadata(m map[string]string) UploadOption {
	return func(c *uploadConfig) { c.metadata = m }
}

// Upload uploads data to an R2 bucket.
func (c *client) Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("Upload", err.Error())
	}
	if key == "" {
		return nil, validationError("Upload", "object key is required")
	}
	if reader == nil {
		return nil, validationError("Upload", "reader is required")
	}

	cfg := &uploadConfig{}
	for _, o := range opts {
		o(cfg)
	}

	// Apply cache policy if enabled
	if c.cfg.cacheControl && cfg.cacheControl == "" {
		cfg.cacheControl = defaultCachePolicy(key)
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}
	if cfg.contentType != "" {
		input.ContentType = aws.String(cfg.contentType)
	}
	if cfg.cacheControl != "" {
		input.CacheControl = aws.String(cfg.cacheControl)
	}
	if len(cfg.metadata) > 0 {
		input.Metadata = cfg.metadata
	}

	result, err := c.s3Client().PutObject(ctx, input)
	if err != nil {
		return nil, newError("Upload", "failed to upload object", err)
	}

	return &UploadResult{
		Key:       key,
		Bucket:    bucket,
		Size:      size,
		ETag:      aws.ToString(result.ETag),
		VersionID: aws.ToString(result.VersionId),
		Uploaded:  time.Now().UTC(),
	}, nil
}

// defaultCachePolicy returns a cache-control header based on file extension.
func defaultCachePolicy(key string) string {
	ext := extension(key)
	switch ext {
	case ".css", ".js":
		return "public, max-age=31536000, immutable"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico":
		return "public, max-age=86400"
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return "public, max-age=31536000, immutable"
	case ".html", ".htm":
		return "public, max-age=0, must-revalidate"
	default:
		return ""
	}
}

func extension(key string) string {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return key[i:]
		}
		if key[i] == '/' {
			break
		}
	}
	return ""
}

// detectContentType guesses content type from file extension.
func detectContentType(key string) string {
	ext := extension(key)
	ct, _ := mimeTypes[ext]
	return ct
}

var mimeTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".htm":  "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript",
	".json": "application/json",
	".xml":  "application/xml",
	".txt":  "text/plain; charset=utf-8",
	".csv":  "text/csv",
	".md":   "text/markdown; charset=utf-8",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".pdf":  "application/pdf",
	".zip":  "application/zip",
	".tar":  "application/x-tar",
	".gz":   "application/gzip",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".eot":   "application/vnd.ms-fontobject",
}

