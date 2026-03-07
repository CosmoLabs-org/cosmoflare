package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func BenchmarkSinglePartUpload(b *testing.B) {
	mock := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return &s3.PutObjectOutput{ETag: aws.String("bench-etag")}, nil
		},
	}
	baseClient := &Client{accountID: "bench", apiToken: "bench"}
	ec, _ := NewEnhancedClientWithS3(baseClient, mock)

	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.bin")
	os.WriteFile(tmpFile, make([]byte, 1024), 0644)

	ctx := context.Background()
	opts := &UploadOptions{Quiet: true, ChunkSize: 1024 * 1024}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.UploadFile(ctx, "bucket", "key", tmpFile, opts)
	}
}

func BenchmarkMultipartUpload(b *testing.B) {
	mock := &mockS3Client{}
	baseClient := &Client{accountID: "bench", apiToken: "bench"}
	ec, _ := NewEnhancedClientWithS3(baseClient, mock)

	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.bin")
	os.WriteFile(tmpFile, make([]byte, 500), 0644)

	ctx := context.Background()
	opts := &UploadOptions{Quiet: true, ChunkSize: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.UploadFile(ctx, "bucket", "key", tmpFile, opts)
	}
}

func BenchmarkNewClient(b *testing.B) {
	opts := &ClientOptions{AccountID: "bench", APIToken: "bench"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewClient(opts)
	}
}
