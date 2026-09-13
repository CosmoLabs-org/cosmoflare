package cosmoflare

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// UploadOption configures an upload operation.
type UploadOption func(*uploadConfig)

type uploadConfig struct {
	contentType      string
	cacheControl     string
	metadata         map[string]string
	partSize         int64
	concurrency      int
	progressCallback func(uploaded, total int64)
}

const (
	defaultPartSize    = 8 * 1024 * 1024 // 8MB
	defaultConcurrency = 3
	multipartThreshold = 100 * 1024 * 1024 // 100MB
)

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

// WithPartSize sets the part size for multipart uploads (default 8MB).
func WithPartSize(n int64) UploadOption {
	return func(c *uploadConfig) { c.partSize = n }
}

// WithConcurrency sets the number of concurrent part uploads (default 3).
func WithConcurrency(n int) UploadOption {
	return func(c *uploadConfig) { c.concurrency = n }
}

// WithProgressCallback sets a callback for upload progress reporting.
func WithProgressCallback(fn func(uploaded, total int64)) UploadOption {
	return func(c *uploadConfig) { c.progressCallback = fn }
}

// Upload uploads data to an R2 bucket.
// If size exceeds 100MB, it automatically delegates to MultipartUpload.
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
	if err := c.enforceUploadGuardrails("Upload", bucket, key, size); err != nil {
		return nil, err
	}

	cfg := &uploadConfig{}
	for _, o := range opts {
		o(cfg)
	}

	// Auto-threshold: delegate to multipart for large files
	if size > multipartThreshold {
		return c.MultipartUpload(ctx, bucket, key, reader, size, opts...)
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

// MultipartUpload uploads large files using the S3 multipart upload protocol.
// Files are split into parts and uploaded concurrently for better throughput.
func (c *client) MultipartUpload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("MultipartUpload", err.Error())
	}
	if key == "" {
		return nil, validationError("MultipartUpload", "object key is required")
	}
	if reader == nil {
		return nil, validationError("MultipartUpload", "reader is required")
	}
	if err := c.enforceUploadGuardrails("MultipartUpload", bucket, key, size); err != nil {
		return nil, err
	}

	cfg := &uploadConfig{
		partSize:    defaultPartSize,
		concurrency: defaultConcurrency,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Apply cache policy
	if c.cfg.cacheControl && cfg.cacheControl == "" {
		cfg.cacheControl = defaultCachePolicy(key)
	}

	// Initiate multipart upload
	createInput := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if cfg.contentType != "" {
		createInput.ContentType = aws.String(cfg.contentType)
	}
	if cfg.cacheControl != "" {
		createInput.CacheControl = aws.String(cfg.cacheControl)
	}
	if len(cfg.metadata) > 0 {
		createInput.Metadata = cfg.metadata
	}

	createResp, err := c.s3Client().CreateMultipartUpload(ctx, createInput)
	if err != nil {
		return nil, newError("MultipartUpload", "failed to initiate multipart upload", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Abort on any failure
	var completedParts []types.CompletedPart
	abort := func() {
		// BUG-043: abort with a context that survives cancellation of the
		// request context — aborting with the canceled ctx fails instantly
		// and leaks the incomplete upload (billed parts, 7-day retention).
		abortCtx, abortCancel := abortContext(ctx)
		defer abortCancel()
		c.s3Client().AbortMultipartUpload(abortCtx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
	}

	// Upload parts
	partSize := cfg.partSize
	numParts := (size + partSize - 1) / partSize
	if numParts == 0 {
		numParts = 1
	}

	var uploadedBytes int64
	type partResult struct {
		part  types.CompletedPart
		err   error
	}

	sem := make(chan struct{}, cfg.concurrency)
	results := make(chan partResult, numParts)

	for partNum := int64(1); partNum <= numParts; partNum++ {
		// Read this part's data (BUG-026: a reader shorter than the declared
		// size aborts the upload instead of storing a truncated object)
		buf, err := readUploadPart("MultipartUpload", reader, partSize, size, uploadedBytes, partNum)
		if err != nil {
			abort()
			return nil, err
		}
		partOffset := uploadedBytes
		partNumber := int32(partNum)

		sem <- struct{}{}
		go func(pn int32, data []byte, offset int64) {
			defer func() { <-sem }()
			resp, err := c.s3Client().UploadPart(ctx, &s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				UploadId:   aws.String(uploadID),
				PartNumber: aws.Int32(pn),
				Body:       bytes.NewReader(data),
			})
			if err != nil {
				results <- partResult{err: fmt.Errorf("part %d: %w", pn, err)}
				return
			}
			results <- partResult{part: types.CompletedPart{
				ETag:       resp.ETag,
				PartNumber: aws.Int32(pn),
			}}
		}(partNumber, buf, partOffset)

		uploadedBytes += int64(len(buf))

		if cfg.progressCallback != nil {
			cfg.progressCallback(uploadedBytes, size)
		}
	}

	// Collect results
	for i := int64(0); i < numParts; i++ {
		r := <-results
		if r.err != nil {
			abort()
			return nil, newError("MultipartUpload", "part upload failed", r.err)
		}
		completedParts = append(completedParts, r.part)
	}

	// Sort parts by part number (goroutine results arrive in non-deterministic order)
	sort.Slice(completedParts, func(i, j int) bool {
		return *completedParts[i].PartNumber < *completedParts[j].PartNumber
	})

	// Complete multipart upload
	completeResp, err := c.s3Client().CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		abort()
		return nil, newError("MultipartUpload", "failed to complete multipart upload", err)
	}

	etag := aws.ToString(completeResp.ETag)

	return &UploadResult{
		Key:       key,
		Bucket:    bucket,
		Size:      size,
		ETag:      etag,
		Uploaded:  time.Now().UTC(),
		Parts:     int(numParts),
	}, nil
}

// readUploadPart reads the next multipart part from reader, enforcing the
// declared total size (BUG-026).
//
// The part length is capped at the bytes remaining for the final part.
// io.EOF or io.ErrUnexpectedEOF while fewer than size bytes have been consumed
// is a validation error ("reader shorter than declared size"), so a truncated
// object can never be completed silently. A clean EOF exactly at the size
// boundary is a success.
func readUploadPart(op string, reader io.Reader, partSize, size, uploadedBytes, partNum int64) ([]byte, error) {
	thisPartSize := partSize
	if remaining := size - uploadedBytes; remaining < thisPartSize {
		thisPartSize = remaining
	}

	buf := make([]byte, thisPartSize)
	n, err := io.ReadFull(reader, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		total := uploadedBytes + int64(n)
		if total < size {
			return nil, validationError(op, fmt.Sprintf(
				"reader shorter than declared size (%d of %d bytes)", total, size))
		}
		// Clean EOF exactly at the size boundary: every declared byte is read.
		return buf[:n], nil
	}
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to read part %d", partNum), err)
	}
	return buf[:n], nil
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

