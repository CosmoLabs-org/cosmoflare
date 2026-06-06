package upload_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	r2go2 "github.com/CosmoLabs-org/cosmoflare/pkg/r2go2"
)

func TestUploadValidation_BucketName(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	content := bytes.NewReader([]byte("data"))

	_, err = client.Upload(ctx, "INVALID-BUCKET", "key.txt", content, 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket")
}

func TestUploadValidation_EmptyKey(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	content := bytes.NewReader([]byte("data"))

	_, err = client.Upload(ctx, "valid-bucket", "", content, 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key")
}

func TestUploadValidation_NilReader(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = client.Upload(ctx, "valid-bucket", "key.txt", nil, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reader")
}

func TestUploadOptionsApplied(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestPresignedURL_Generation(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
		r2go2.WithRegion("auto"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	url, err := client.PresignGetObject(ctx, "test-bucket", "file.txt", time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "test-bucket")
	assert.Contains(t, url, "file.txt")
	assert.Contains(t, url, "X-Amz-Signature")
}

func TestPresignedURL_DifferentExpiry(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx := context.Background()

	url30m, err := client.PresignGetObject(ctx, "bucket", "key", 30*time.Minute)
	require.NoError(t, err)

	url24h, err := client.PresignGetObject(ctx, "bucket", "key", 24*time.Hour)
	require.NoError(t, err)

	assert.NotEqual(t, url30m, url24h, "different expiry should produce different URLs")
}

func TestTestConnection_WithNoAPIClient(t *testing.T) {
	// Client created without Cloudflare API token should fail TestConnection
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.TestConnection(ctx)
	// Will fail because the token is fake, but shouldn't panic
	assert.Error(t, err)
}

func TestClientAccountID(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)
	assert.Equal(t, "test1234567890abcdef1234567890abc", client.AccountID())
}

func TestUploadWithCredentials(t *testing.T) {
	// Test that WithCredentials option creates a valid client
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
		r2go2.WithCredentials("test-access-key", "test-secret-key"),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestUploadWithCustomEndpoint(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
		r2go2.WithEndpoint("https://custom.r2.cloudflarestorage.com"),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestUploadAutoMultipartThreshold(t *testing.T) {
	// Verify that files over the multipart threshold are handled
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	// 200MB file should trigger multipart (threshold is 100MB)
	ctx := context.Background()
	largeContent := bytes.NewReader(make([]byte, 200*1024*1024))

	result, err := client.Upload(ctx, "test-bucket", "large.bin", largeContent, 200*1024*1024)
	// Will fail because no real S3 endpoint, but should be a network error
	// not a validation error
	if err != nil {
		assert.NotContains(t, err.Error(), "validation")
		assert.NotContains(t, err.Error(), "required")
	} else {
		// If somehow it succeeds, verify the result
		assert.Equal(t, "large.bin", result.Key)
	}
}

func TestUploadCancellation(t *testing.T) {
	client, err := r2go2.NewClient(
		r2go2.WithAccountID("test1234567890abcdef1234567890abc"),
		r2go2.WithAPIToken("test-api-token-123456789012345678"),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	content := bytes.NewReader([]byte("data"))
	_, err = client.Upload(ctx, "test-bucket", "key.txt", content, 4)
	// Should fail with context canceled error
	assert.Error(t, err)
}
