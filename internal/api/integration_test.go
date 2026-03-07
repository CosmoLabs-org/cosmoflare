package api

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_CompleteUploadFlow exercises the full upload stack:
// Client -> EnhancedClient -> UploadFile (single-part) -> verify result
func TestIntegration_CompleteUploadFlow(t *testing.T) {
	var capturedKey string
	var capturedBucket string
	var capturedBody []byte

	mock := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			capturedKey = aws.ToString(params.Key)
			capturedBucket = aws.ToString(params.Bucket)
			body, _ := io.ReadAll(params.Body)
			capturedBody = body
			return &s3.PutObjectOutput{ETag: aws.String("\"abc123\"")}, nil
		},
	}

	// Create client and enhanced client
	baseClient := &Client{accountID: "test-acct", apiToken: "test-token"}
	ec, err := NewEnhancedClientWithS3(baseClient, mock)
	require.NoError(t, err)

	// Create a test file
	tmpDir := t.TempDir()
	testContent := []byte("Hello from integration test! This is file content for upload.")
	testFile := filepath.Join(tmpDir, "integration-test.txt")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	// Upload via UploadFile (single part, small file)
	result, err := ec.UploadFile(t.Context(), "my-bucket", "uploads/test.txt", testFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 10 * 1024 * 1024, // 10MB - file is smaller, so single-part
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the mock received correct data
	assert.Equal(t, "uploads/test.txt", capturedKey)
	assert.Equal(t, "my-bucket", capturedBucket)
	assert.Equal(t, testContent, capturedBody)

	// Verify result fields
	assert.Equal(t, "uploads/test.txt", result.Key)
	assert.Equal(t, "my-bucket", result.Bucket)
	assert.Equal(t, int64(len(testContent)), result.Size)
	assert.NotEmpty(t, result.ETag)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// TestIntegration_MultipartUploadFlow exercises multipart upload:
// Client -> EnhancedClient -> UploadFile (multipart) -> verify parts
func TestIntegration_MultipartUploadFlow(t *testing.T) {
	var uploadedParts []int32

	mock := &mockS3Client{
		createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
			return &s3.CreateMultipartUploadOutput{
				UploadId: aws.String("integration-upload-id"),
			}, nil
		},
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			uploadedParts = append(uploadedParts, *params.PartNumber)
			return &s3.UploadPartOutput{
				ETag: aws.String("\"part-" + string(rune('0'+*params.PartNumber)) + "\""),
			}, nil
		},
		completeMultipartFunc: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
			// Verify all parts are included in completion
			assert.NotNil(t, params.MultipartUpload)
			assert.NotEmpty(t, params.MultipartUpload.Parts)
			return &s3.CompleteMultipartUploadOutput{
				ETag: aws.String("\"complete-etag\""),
			}, nil
		},
	}

	baseClient := &Client{accountID: "test-acct", apiToken: "test-token"}
	ec, err := NewEnhancedClientWithS3(baseClient, mock)
	require.NoError(t, err)

	// Create a file larger than chunk size to trigger multipart
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large-file.bin")
	largeContent := make([]byte, 3*1024*1024) // 3MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	require.NoError(t, os.WriteFile(testFile, largeContent, 0644))

	// Upload with small chunk size to force multipart
	result, err := ec.UploadFile(t.Context(), "data-bucket", "backups/large.bin", testFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024, // 1MB chunks → 3 parts
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify multipart was used
	assert.Equal(t, "backups/large.bin", result.Key)
	assert.Equal(t, "data-bucket", result.Bucket)
	assert.Equal(t, int64(len(largeContent)), result.Size)
	assert.Len(t, uploadedParts, 4, "should have uploaded 4 parts (3MB / 1MB + 1)")
	assert.Equal(t, []int32{1, 2, 3, 4}, uploadedParts)
	assert.NotEmpty(t, result.Parts, "result should include completed parts")
}

// TestIntegration_UploadWithMetadata verifies metadata is passed through to S3
func TestIntegration_UploadWithMetadata(t *testing.T) {
	var capturedContentType string
	var capturedMetadata map[string]string

	mock := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			capturedContentType = aws.ToString(params.ContentType)
			capturedMetadata = params.Metadata
			return &s3.PutObjectOutput{ETag: aws.String("\"meta-etag\"")}, nil
		},
	}

	ec := mustCreateClient(t, mock)
	tmpFile := createTestFile(t, "metadata test content")

	result, err := ec.UploadFile(t.Context(), "bucket", "doc.pdf", tmpFile, &UploadOptions{
		Quiet:       true,
		ChunkSize:   10 * 1024 * 1024,
		ContentType: "application/pdf",
		Metadata:    map[string]string{"author": "test", "version": "1.0"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "application/pdf", capturedContentType)
	assert.Equal(t, "test", capturedMetadata["author"])
	assert.Equal(t, "1.0", capturedMetadata["version"])
}

// TestIntegration_UploadFailureRecovery tests error propagation through the full stack
func TestIntegration_UploadFailureRecovery(t *testing.T) {
	t.Run("PutObject failure propagates", func(t *testing.T) {
		mock := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, &types.NoSuchBucket{Message: aws.String("bucket not found")}
			},
		}

		ec := mustCreateClient(t, mock)
		tmpFile := createTestFile(t, "fail content")

		result, err := ec.UploadFile(t.Context(), "nonexistent", "key.txt", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 10 * 1024 * 1024,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("multipart abort on part failure", func(t *testing.T) {
		var abortCalled bool
		mock := &mockS3Client{
			createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
				return &s3.CreateMultipartUploadOutput{UploadId: aws.String("fail-upload-id")}, nil
			},
			uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
				if *params.PartNumber == 2 {
					return nil, context.DeadlineExceeded
				}
				return &s3.UploadPartOutput{ETag: aws.String("\"ok\"")}, nil
			},
			abortMultipartFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
				abortCalled = true
				assert.Equal(t, "fail-upload-id", aws.ToString(params.UploadId))
				return &s3.AbortMultipartUploadOutput{}, nil
			},
		}

		ec := mustCreateClient(t, mock)
		tmpFile := createLargeTestFile(t, 3*1024*1024)

		_, err := ec.UploadFile(t.Context(), "bucket", "fail.bin", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})

		assert.Error(t, err)
		assert.True(t, abortCalled, "should abort multipart upload on failure")
	})
}

// Helpers

func mustCreateClient(t *testing.T, mock *mockS3Client) *EnhancedClient {
	t.Helper()
	baseClient := &Client{accountID: "test-acct", apiToken: "test-token"}
	ec, err := NewEnhancedClientWithS3(baseClient, mock)
	require.NoError(t, err)
	return ec
}

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	return tmpFile
}

func createLargeTestFile(t *testing.T, size int) string {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	tmpFile := filepath.Join(t.TempDir(), "large.bin")
	require.NoError(t, os.WriteFile(tmpFile, data, 0644))
	return tmpFile
}
