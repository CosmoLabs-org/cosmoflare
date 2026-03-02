package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockS3Client implements S3API for testing
type mockS3Client struct {
	putObjectFunc             func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	createMultipartUploadFunc func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	uploadPartFunc            func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	completeMultipartFunc     func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	abortMultipartFunc        func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{ETag: aws.String("mock-etag")}, nil
}

func (m *mockS3Client) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	if m.createMultipartUploadFunc != nil {
		return m.createMultipartUploadFunc(ctx, params, optFns...)
	}
	return &s3.CreateMultipartUploadOutput{UploadId: aws.String("mock-upload-id")}, nil
}

func (m *mockS3Client) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	if m.uploadPartFunc != nil {
		return m.uploadPartFunc(ctx, params, optFns...)
	}
	return &s3.UploadPartOutput{ETag: aws.String(fmt.Sprintf("part-etag-%d", *params.PartNumber))}, nil
}

func (m *mockS3Client) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	if m.completeMultipartFunc != nil {
		return m.completeMultipartFunc(ctx, params, optFns...)
	}
	return &s3.CompleteMultipartUploadOutput{ETag: aws.String("complete-etag")}, nil
}

func (m *mockS3Client) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	if m.abortMultipartFunc != nil {
		return m.abortMultipartFunc(ctx, params, optFns...)
	}
	return &s3.AbortMultipartUploadOutput{}, nil
}

// newTestEnhancedClient creates an EnhancedClient with a mock S3 API
func newTestEnhancedClient(t *testing.T, mock *mockS3Client) *EnhancedClient {
	t.Helper()
	baseClient := &Client{
		accountID: "test-account",
		apiToken:  "test-token",
	}
	ec, err := NewEnhancedClientWithS3(baseClient, mock)
	require.NoError(t, err)
	return ec
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileSize(t *testing.T) {
	t.Run("Valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("hello world test content")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		size := getFileSize(tmpFile)
		assert.Equal(t, int64(len(content)), size)
	})

	t.Run("Nonexistent file returns 0", func(t *testing.T) {
		size := getFileSize("/nonexistent/path/file.txt")
		assert.Equal(t, int64(0), size)
	})

	t.Run("Empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(tmpFile, []byte{}, 0644)
		require.NoError(t, err)

		size := getFileSize(tmpFile)
		assert.Equal(t, int64(0), size)
	})
}

func TestNewEnhancedClient(t *testing.T) {
	t.Run("Nil base client returns error", func(t *testing.T) {
		ec, err := NewEnhancedClient(nil)
		assert.Error(t, err)
		assert.Nil(t, ec)
		assert.Contains(t, err.Error(), "base client cannot be nil")
	})

	t.Run("Valid base client", func(t *testing.T) {
		baseClient := &Client{
			accountID: "test-account",
			apiToken:  "test-token",
		}

		ec, err := NewEnhancedClient(baseClient)
		require.NoError(t, err)
		require.NotNil(t, ec)
		assert.Equal(t, baseClient, ec.Client)
	})

	t.Run("Inherits S3 client from base", func(t *testing.T) {
		baseClient := &Client{
			accountID: "test-account",
			apiToken:  "test-token",
			s3:        nil,
		}

		ec, err := NewEnhancedClient(baseClient)
		require.NoError(t, err)
		assert.Nil(t, ec.s3Client) // nil s3 from base
	})
}

func TestUploadOptions_Defaults(t *testing.T) {
	t.Run("Default upload options", func(t *testing.T) {
		opts := &UploadOptions{}
		assert.False(t, opts.ShowProgress)
		assert.False(t, opts.Quiet)
		assert.Equal(t, int64(0), opts.ChunkSize)
		assert.Equal(t, 0, opts.Retries)
		assert.Empty(t, opts.ContentType)
		assert.Empty(t, opts.CacheControl)
		assert.Nil(t, opts.Metadata)
		assert.Nil(t, opts.ProgressFunc)
	})

	t.Run("Custom upload options", func(t *testing.T) {
		callbackCalled := false
		opts := &UploadOptions{
			ContentType:  "image/png",
			CacheControl: "max-age=3600",
			Metadata:     map[string]string{"key": "value"},
			ChunkSize:    16 * 1024 * 1024,
			ShowProgress: true,
			Quiet:        false,
			Retries:      5,
			ProgressFunc: func(uploaded, total int64, speed float64) {
				callbackCalled = true
			},
		}

		assert.Equal(t, "image/png", opts.ContentType)
		assert.Equal(t, "max-age=3600", opts.CacheControl)
		assert.Equal(t, int64(16*1024*1024), opts.ChunkSize)
		assert.True(t, opts.ShowProgress)
		assert.Equal(t, 5, opts.Retries)
		assert.Equal(t, "value", opts.Metadata["key"])

		// Test callback
		opts.ProgressFunc(100, 200, 1.5)
		assert.True(t, callbackCalled)
	})
}

func TestUploadResult_Struct(t *testing.T) {
	result := &UploadResult{
		Key:      "test/file.txt",
		Bucket:   "my-bucket",
		Size:     1024,
		ETag:     "etag-123",
		UploadID: "upload-456",
		Speed:    10.5,
		Parts: []CompletedPart{
			{PartNumber: 1, ETag: "part-1-etag", Size: 512},
			{PartNumber: 2, ETag: "part-2-etag", Size: 512},
		},
		Metadata: map[string]string{"type": "test"},
		URL:      "https://example.com/file.txt",
	}

	assert.Equal(t, "test/file.txt", result.Key)
	assert.Equal(t, "my-bucket", result.Bucket)
	assert.Equal(t, int64(1024), result.Size)
	assert.Equal(t, "etag-123", result.ETag)
	assert.Equal(t, "upload-456", result.UploadID)
	assert.Equal(t, 10.5, result.Speed)
	assert.Len(t, result.Parts, 2)
	assert.Equal(t, 1, result.Parts[0].PartNumber)
	assert.Equal(t, int64(512), result.Parts[0].Size)
	assert.Equal(t, "test", result.Metadata["type"])
	assert.Equal(t, "https://example.com/file.txt", result.URL)
}

func TestUploadFile_FileValidation(t *testing.T) {
	baseClient := &Client{
		accountID: "test-account",
		apiToken:  "test-token",
	}

	ec, err := NewEnhancedClient(baseClient)
	require.NoError(t, err)

	t.Run("Nonexistent file returns error", func(t *testing.T) {
		result, err := ec.UploadFile(
			t.Context(),
			"test-bucket",
			"test-key",
			"/nonexistent/file.txt",
			&UploadOptions{Quiet: true},
		)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to stat file")
	})

}

func TestSinglePartUpload_WithMock(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	t.Run("Successful single part upload", func(t *testing.T) {
		putCalled := false
		mock.putObjectFunc = func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			putCalled = true
			assert.Equal(t, "test-bucket", *params.Bucket)
			assert.Equal(t, "test-key.txt", *params.Key)
			assert.Equal(t, "text/plain", *params.ContentType)
			assert.Equal(t, "max-age=300", *params.CacheControl)
			assert.Equal(t, "val", params.Metadata["key"])
			return &s3.PutObjectOutput{ETag: aws.String("upload-etag")}, nil
		}

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(tmpFile, []byte("hello world"), 0644)

		result, err := ec.UploadFile(t.Context(), "test-bucket", "test-key.txt", tmpFile, &UploadOptions{
			Quiet:        true,
			ChunkSize:    1024 * 1024,
			ContentType:  "text/plain",
			CacheControl: "max-age=300",
			Metadata:     map[string]string{"key": "val"},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, putCalled)
		assert.Equal(t, "test-key.txt", result.Key)
		assert.Equal(t, "test-bucket", result.Bucket)
		assert.Equal(t, int64(11), result.Size)
		assert.Equal(t, "upload-etag", result.ETag)
		assert.NotZero(t, result.Duration)
	})

	t.Run("PutObject error propagates", func(t *testing.T) {
		mock.putObjectFunc = func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, fmt.Errorf("access denied")
		}

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(tmpFile, []byte("data"), 0644)

		result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to upload object")
	})

	t.Run("Default content type when not specified", func(t *testing.T) {
		mock.putObjectFunc = func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			assert.Equal(t, "application/octet-stream", *params.ContentType)
			return &s3.PutObjectOutput{ETag: aws.String("etag")}, nil
		}

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.bin")
		os.WriteFile(tmpFile, []byte("binary"), 0644)

		result, err := ec.UploadFile(t.Context(), "b", "k", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestMultipartUpload_WithMock(t *testing.T) {
	t.Run("Successful multipart upload", func(t *testing.T) {
		partCount := 0
		mock := &mockS3Client{
			createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				return &s3.CreateMultipartUploadOutput{UploadId: aws.String("mp-upload-123")}, nil
			},
			uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
				partCount++
				return &s3.UploadPartOutput{ETag: aws.String(fmt.Sprintf("part-%d", *params.PartNumber))}, nil
			},
			completeMultipartFunc: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
				assert.Equal(t, "mp-upload-123", *params.UploadId)
				return &s3.CompleteMultipartUploadOutput{ETag: aws.String("final-etag")}, nil
			},
		}

		ec := newTestEnhancedClient(t, mock)

		// Create file larger than chunk size
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "large.bin")
		content := make([]byte, 200)
		for i := range content {
			content[i] = byte(i % 256)
		}
		os.WriteFile(tmpFile, content, 0644)

		result, err := ec.UploadFile(t.Context(), "test-bucket", "large-key", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 50, // Forces multipart with 200-byte file
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "large-key", result.Key)
		assert.Equal(t, "test-bucket", result.Bucket)
		assert.Equal(t, int64(200), result.Size)
		assert.Equal(t, "final-etag", result.ETag)
		assert.Equal(t, "mp-upload-123", result.UploadID)
		assert.Greater(t, partCount, 1, "Should have uploaded multiple parts")
		assert.Len(t, result.Parts, partCount)
	})

	t.Run("CreateMultipartUpload failure", func(t *testing.T) {
		mock := &mockS3Client{
			createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
				return nil, fmt.Errorf("bucket not found")
			},
		}
		ec := newTestEnhancedClient(t, mock)

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.bin")
		os.WriteFile(tmpFile, make([]byte, 200), 0644)

		result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 50,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create multipart upload")
	})

	t.Run("UploadPart failure triggers abort", func(t *testing.T) {
		abortCalled := false
		mock := &mockS3Client{
			uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
				return nil, fmt.Errorf("network timeout")
			},
			abortMultipartFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
				abortCalled = true
				assert.Equal(t, "mock-upload-id", *params.UploadId)
				return &s3.AbortMultipartUploadOutput{}, nil
			},
		}
		ec := newTestEnhancedClient(t, mock)

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.bin")
		os.WriteFile(tmpFile, make([]byte, 200), 0644)

		result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 50,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to upload part")
		assert.True(t, abortCalled, "Should have called AbortMultipartUpload on failure")
	})

	t.Run("CompleteMultipartUpload failure triggers abort", func(t *testing.T) {
		abortCalled := false
		mock := &mockS3Client{
			completeMultipartFunc: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
				return nil, fmt.Errorf("completion failed")
			},
			abortMultipartFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
				abortCalled = true
				return &s3.AbortMultipartUploadOutput{}, nil
			},
		}
		ec := newTestEnhancedClient(t, mock)

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.bin")
		os.WriteFile(tmpFile, make([]byte, 200), 0644)

		result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 50,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to complete multipart upload")
		assert.True(t, abortCalled)
	})
}

func TestNewEnhancedClientWithS3(t *testing.T) {
	t.Run("Nil base client returns error", func(t *testing.T) {
		ec, err := NewEnhancedClientWithS3(nil, &mockS3Client{})
		assert.Error(t, err)
		assert.Nil(t, ec)
	})

	t.Run("Valid base client and S3 API", func(t *testing.T) {
		mock := &mockS3Client{}
		baseClient := &Client{accountID: "test"}
		ec, err := NewEnhancedClientWithS3(baseClient, mock)
		require.NoError(t, err)
		require.NotNil(t, ec)
		assert.Equal(t, mock, ec.s3Client)
	})
}
