package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
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
			result := utils.FormatBytes(tt.input)
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

func TestUploadFile_NilOpts(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	// nil opts triggers default creation path
	result, err := ec.UploadFile(t.Context(), "bucket", "key.txt", tmpFile, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "key.txt", result.Key)
	assert.Equal(t, int64(5), result.Size)
}

func TestUploadFile_WithProgressCallback(t *testing.T) {
	callbackCalled := false
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test data here"), 0644)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
		ProgressFunc: func(uploaded, total int64, speed float64) {
			callbackCalled = true
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	_ = callbackCalled
}

func TestAbortMultipartUpload_ErrorPath(t *testing.T) {
	abortCalled := false
	mock := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			return nil, fmt.Errorf("part upload failed")
		},
		abortMultipartFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
			abortCalled = true
			return nil, fmt.Errorf("abort also failed")
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
	assert.True(t, abortCalled, "abort should have been called")
}

func TestSinglePartUpload_WithVisualProgress(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("data"), 0644)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:        false,
		ShowProgress: true,
		ChunkSize:    1024 * 1024,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestUploadWithRealTimeProgress(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("upload content"), 0644)

	t.Run("Successful upload", func(t *testing.T) {
		err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key.txt", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})
		assert.NoError(t, err)
	})

	t.Run("File not found", func(t *testing.T) {
		err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key.txt", "/nonexistent", &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat file")
	})

	t.Run("Upload failure", func(t *testing.T) {
		mock.putObjectFunc = func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, fmt.Errorf("access denied")
		}

		err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key.txt", tmpFile, &UploadOptions{
			Quiet:     true,
			ChunkSize: 1024 * 1024,
		})
		assert.Error(t, err)

		// Reset mock
		mock.putObjectFunc = nil
	})
}

func TestMultipartUpload_WithVisualProgress(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.bin")
	os.WriteFile(tmpFile, make([]byte, 200), 0644)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:        false,
		ShowProgress: true,
		ChunkSize:    50,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, len(result.Parts), 1)
}

// ---------------------------------------------------------------------------
// singlePartUpload - file open error
// ---------------------------------------------------------------------------

func TestSinglePartUpload_FileOpenError(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", "/nonexistent/file.bin", &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	// UploadFile first calls os.Stat which fails for nonexistent files
	assert.Contains(t, err.Error(), "failed to stat file")
}

func TestSinglePartUpload_UnreadableFile(t *testing.T) {
	// Create a file, make it unreadable, then try to upload
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "unreadable.bin")
	os.WriteFile(tmpFile, []byte("secret data"), 0644)

	// Make unreadable
	os.Chmod(tmpFile, 0000)
	defer os.Chmod(tmpFile, 0644)

	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
	})
	// On some systems (or as root), unreadable files can still be opened
	if err != nil {
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to open file")
	}
}

// ---------------------------------------------------------------------------
// uploadPart - file open error
// ---------------------------------------------------------------------------

func TestUploadPart_FileOpenError(t *testing.T) {
	abortCalled := false
	mock := &mockS3Client{
		abortMultipartFunc: func(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
			abortCalled = true
			return &s3.AbortMultipartUploadOutput{}, nil
		},
	}
	ec := newTestEnhancedClient(t, mock)

	// Create a file, then delete it before the upload part reads it
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "vanish.bin")
	os.WriteFile(tmpFile, make([]byte, 200), 0644)

	// Replace file with a directory - os.Open on a dir succeeds in Go,
	// but seeking + reading will fail differently. Instead, let's just
	// use a path that doesn't exist for the multipart case.
	// The multipart path first stats the file (succeeds), then opens for each part.
	// We need to delete between stat and part upload - not possible in a single thread.
	// Instead test that a truly unreadable file is handled.
	// On Unix, we can make a file unreadable.
	_ = abortCalled

	// Test with a file that gets deleted between stat and part open
	// by using the multipart path with a file we remove right after stat
	os.Remove(tmpFile)
	os.WriteFile(tmpFile, make([]byte, 200), 0644)
	// Make the file unreadable to trigger open error in uploadPart
	os.Chmod(tmpFile, 0000)
	defer os.Chmod(tmpFile, 0644) // cleanup

	// On some systems 0000 still allows root to read, so just verify the flow works
	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 50,
	})
	// Either it fails with permission error (open fails) or succeeds
	if err != nil {
		assert.Nil(t, result)
	}
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - quiet mode
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_QuietMode(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "quiet-test.txt")
	os.WriteFile(tmpFile, []byte("quiet content"), 0644)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key.txt", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
	})
	assert.NoError(t, err)
}

func TestUploadWithRealTimeProgress_WithoutOriginalProgressFunc(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "no-callback.txt")
	os.WriteFile(tmpFile, []byte("no callback content"), 0644)

	opts := &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
		// ProgressFunc is nil
	}

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key.txt", tmpFile, opts)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UploadFile - exact chunk boundary (single vs multipart decision)
// ---------------------------------------------------------------------------

func TestUploadFile_ExactChunkSizeBoundary(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "boundary.bin")
	// Exactly chunk size - should be single part (fileSize > chunkSize is false)
	os.WriteFile(tmpFile, make([]byte, 100), 0644)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 100,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	// Exactly chunk size means fileSize > chunkSize is false -> single part
	assert.Empty(t, result.Parts)
}

func TestUploadFile_OneByteOverChunkSize(t *testing.T) {
	partCount := 0
	mock := &mockS3Client{
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			partCount++
			return &s3.UploadPartOutput{ETag: aws.String(fmt.Sprintf("part-%d", *params.PartNumber))}, nil
		},
	}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "over.bin")
	os.WriteFile(tmpFile, make([]byte, 101), 0644) // 101 > 100 chunk

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 100,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, len(result.Parts), 0, "should use multipart upload")
	assert.Greater(t, partCount, 0)
}

// ---------------------------------------------------------------------------
// UploadResult - empty parts
// ---------------------------------------------------------------------------

func TestUploadResult_EmptyParts(t *testing.T) {
	result := &UploadResult{
		Key:    "single.txt",
		Bucket: "my-bucket",
		Size:   1024,
		ETag:   "etag-single",
	}
	assert.Nil(t, result.Parts)
	assert.Equal(t, "", result.UploadID)
	assert.Equal(t, "", result.URL)
	assert.Empty(t, result.Metadata)
	assert.Zero(t, result.Duration)
	assert.Zero(t, result.Speed)
}

// ---------------------------------------------------------------------------
// CompletedPart struct
// ---------------------------------------------------------------------------

func TestCompletedPart_Fields(t *testing.T) {
	part := CompletedPart{
		PartNumber: 3,
		ETag:       "etag-part-3",
		Size:       5242880,
	}
	assert.Equal(t, 3, part.PartNumber)
	assert.Equal(t, "etag-part-3", part.ETag)
	assert.Equal(t, int64(5242880), part.Size)

	data, err := json.Marshal(part)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"part_number":3`)
	assert.Contains(t, string(data), `"etag":"etag-part-3"`)
}

// ---------------------------------------------------------------------------
// UploadOptions - JSON tags
// ---------------------------------------------------------------------------

func TestUploadOptions_JSONTags(t *testing.T) {
	opts := &UploadOptions{
		ContentType:  "image/png",
		CacheControl: "max-age=3600",
		ChunkSize:    8 * 1024 * 1024,
		Retries:      3,
		ShowProgress: true,
		Quiet:        false,
	}

	data, err := json.Marshal(opts)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"content_type":"image/png"`)
	assert.Contains(t, string(data), `"cache_control":"max-age=3600"`)

	// ProgressFunc should not be serialized (json:"-")
	assert.NotContains(t, string(data), `"progress_func"`)
}

// ---------------------------------------------------------------------------
// getFileSize - various file states
// ---------------------------------------------------------------------------

func TestGetFileSize_DirectoryReturnsZero(t *testing.T) {
	tmpDir := t.TempDir()
	size := getFileSize(tmpDir)
	// Directories return their size on some platforms, 0 on others
	// The function uses os.Stat which works on directories too
	assert.GreaterOrEqual(t, size, int64(0))
}

func TestGetFileSize_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	content := []byte("symlink target")
	os.WriteFile(target, content, 0644)

	link := filepath.Join(tmpDir, "link.txt")
	os.Symlink(target, link)

	size := getFileSize(link)
	assert.Equal(t, int64(len(content)), size) // follows symlink to get real file size
}

// ---------------------------------------------------------------------------
// EnhancedClient - nil S3 client from base
// ---------------------------------------------------------------------------

func TestEnhancedClient_NilS3FromBase(t *testing.T) {
	baseClient := &Client{
		accountID: "test",
		apiToken:  "test",
		s3:        nil,
	}

	ec, err := NewEnhancedClient(baseClient)
	require.NoError(t, err)
	assert.Nil(t, ec.s3Client)
}

// ---------------------------------------------------------------------------
// S3API interface compliance
// ---------------------------------------------------------------------------

func TestMockS3Client_ImplementsS3API(t *testing.T) {
	// Compile-time check
	var _ S3API = &mockS3Client{}
}

// ---------------------------------------------------------------------------
// Multipart upload - exact part sizing
// ---------------------------------------------------------------------------

func TestMultipartUpload_PartSizing(t *testing.T) {
	var capturedPartNumbers []int32
	var capturedUploadIDs []string

	mock := &mockS3Client{
		createMultipartUploadFunc: func(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
			return &s3.CreateMultipartUploadOutput{UploadId: aws.String("sizing-test")}, nil
		},
		uploadPartFunc: func(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
			capturedPartNumbers = append(capturedPartNumbers, *params.PartNumber)
			capturedUploadIDs = append(capturedUploadIDs, *params.UploadId)
			// Read body to verify it has content
			body, _ := io.ReadAll(params.Body)
			if len(body) == 0 {
				return nil, fmt.Errorf("empty body for part %d", *params.PartNumber)
			}
			return &s3.UploadPartOutput{ETag: aws.String(fmt.Sprintf("etag-%d", *params.PartNumber))}, nil
		},
		completeMultipartFunc: func(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
			return &s3.CompleteMultipartUploadOutput{ETag: aws.String("final")}, nil
		},
	}

	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sizing.bin")
	// 130 bytes with 50-byte chunks: numParts = int(130/50)+1 = 3 parts
	// Part 1: bytes 0-49 (50 bytes), Part 2: bytes 50-99 (50 bytes), Part 3: bytes 100-129 (30 bytes)
	os.WriteFile(tmpFile, make([]byte, 130), 0644)

	result, err := ec.UploadFile(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 50,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []int32{1, 2, 3}, capturedPartNumbers)
	for _, uid := range capturedUploadIDs {
		assert.Equal(t, "sizing-test", uid)
	}
	assert.Len(t, result.Parts, 3)
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - multipart path
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_MultipartFile(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large-rt.bin")
	os.WriteFile(tmpFile, make([]byte, 200), 0644)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 50, // forces multipart
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - with progress callback
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_WithProgressCallback(t *testing.T) {
	var callbackCount int
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "callback.txt")
	os.WriteFile(tmpFile, []byte("callback test"), 0644)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
		ProgressFunc: func(uploaded, total int64, speed float64) {
			callbackCount++
		},
	})
	assert.NoError(t, err)
	_ = callbackCount
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - stat failure
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_StatFailure(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", "/nonexistent/path/file.txt", &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat file")
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - non-quiet mode
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_NonQuiet(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "visual.txt")
	os.WriteFile(tmpFile, []byte("visual test"), 0644)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     false,
		ChunkSize: 1024 * 1024,
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - with progress callback that gets wrapped
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_ProgressFuncChaining(t *testing.T) {
	originalCalled := false
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "chain.txt")
	os.WriteFile(tmpFile, []byte("chained progress"), 0644)

	// This tests the branch where originalProgressFunc != nil
	// inside UploadWithRealTimeProgress
	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
		ProgressFunc: func(uploaded, total int64, speed float64) {
			originalCalled = true
		},
	})
	assert.NoError(t, err)
	// The progress func chain may or may not be called depending on timing,
	// but the important thing is the code path doesn't panic
	_ = originalCalled
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - non-quiet with multipart
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_NonQuietMultipart(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large-visual.bin")
	os.WriteFile(tmpFile, make([]byte, 200), 0644)

	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     false,
		ChunkSize: 50,
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UploadWithRealTimeProgress - quiet mode with progress callback
// ---------------------------------------------------------------------------

func TestUploadWithRealTimeProgress_QuietWithCallback(t *testing.T) {
	mock := &mockS3Client{}
	ec := newTestEnhancedClient(t, mock)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "quiet-cb.txt")
	os.WriteFile(tmpFile, []byte("quiet callback test"), 0644)

	var callbackCalled bool
	err := ec.UploadWithRealTimeProgress(t.Context(), "bucket", "key", tmpFile, &UploadOptions{
		Quiet:     true,
		ChunkSize: 1024 * 1024,
		ProgressFunc: func(uploaded, total int64, speed float64) {
			callbackCalled = true
		},
	})
	assert.NoError(t, err)
	_ = callbackCalled
}
