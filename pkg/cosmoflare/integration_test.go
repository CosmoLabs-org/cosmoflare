//go:build integration

package cosmoflare

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// Integration tests require real Cloudflare R2 credentials.
//
// Required environment variables:
//   - CLOUDFLARE_ACCOUNT_ID: Your Cloudflare account ID
//   - CLOUDFLARE_API_TOKEN: API token with R2 permissions
//
// Bucket naming: tests use prefix "r2go2-integration-" + timestamp
// Cleanup: tests attempt to clean up all created resources
//
// Run with: go test -tags=integration ./pkg/r2go2/

const integrationBucketPrefix = "r2go2-integration-"

func integrationClient(t *testing.T) R2Client {
	t.Helper()
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if accountID == "" || apiToken == "" {
		t.Skip("CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN must be set for integration tests")
	}
	client, err := NewClient(
		WithAccountID(accountID),
		WithAPIToken(apiToken),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client
}

func uniqueBucketName() string {
	return fmt.Sprintf("%s%d", integrationBucketPrefix, time.Now().UnixNano())
}

// TestIntegration_BucketLifecycle tests the full bucket CRUD lifecycle.
func TestIntegration_BucketLifecycle(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	bucketName := uniqueBucketName()
	t.Logf("Using bucket: %s", bucketName)

	// Create
	bucket, err := client.CreateBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}
	if bucket.Name != bucketName {
		t.Errorf("bucket.Name = %q, want %q", bucket.Name, bucketName)
	}
	t.Cleanup(func() {
		client.DeleteBucket(ctx, bucketName)
	})

	// Exists
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		t.Fatalf("BucketExists failed: %v", err)
	}
	if !exists {
		t.Error("BucketExists returned false after creation")
	}

	// Get
	got, err := client.GetBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("GetBucket failed: %v", err)
	}
	if got.Name != bucketName {
		t.Errorf("GetBucket name = %q, want %q", got.Name, bucketName)
	}

	// List
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	found := false
	for _, b := range buckets {
		if b.Name == bucketName {
			found = true
			break
		}
	}
	if !found {
		t.Error("bucket not found in ListBuckets result")
	}

	// Delete
	err = client.DeleteBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("DeleteBucket failed: %v", err)
	}

	// Verify deleted
	exists, err = client.BucketExists(ctx, bucketName)
	if err != nil {
		t.Fatalf("BucketExists after delete failed: %v", err)
	}
	if exists {
		t.Error("BucketExists returned true after deletion")
	}
}

// TestIntegration_ObjectLifecycle tests upload, get, head, copy, delete.
func TestIntegration_ObjectLifecycle(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	bucketName := uniqueBucketName()

	// Setup bucket
	_, err := client.CreateBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}
	t.Cleanup(func() {
		// Clean up objects first
		result, _ := client.ListObjects(ctx, bucketName, "", "", 0, "")
		if result != nil {
			for _, obj := range result.Items {
				client.DeleteObject(ctx, bucketName, obj.Key)
			}
		}
		client.DeleteBucket(ctx, bucketName)
	})

	// Upload
	content := []byte("Hello, R2Go2 integration test!")
	key := "test/hello.txt"
	uploadResult, err := client.Upload(ctx, bucketName, key, bytes.NewReader(content), int64(len(content)),
		WithContentType("text/plain"),
		WithMetadata(map[string]string{"author": "integration-test"}),
	)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if uploadResult.Key != key {
		t.Errorf("UploadResult.Key = %q, want %q", uploadResult.Key, key)
	}
	if uploadResult.Size != int64(len(content)) {
		t.Errorf("UploadResult.Size = %d, want %d", uploadResult.Size, len(content))
	}
	if uploadResult.ETag == "" {
		t.Error("UploadResult.ETag is empty")
	}

	// Head
	head, err := client.HeadObject(ctx, bucketName, key)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}
	if head.Size != int64(len(content)) {
		t.Errorf("HeadResult.Size = %d, want %d", head.Size, len(content))
	}
	if head.ContentType != "text/plain" {
		t.Errorf("HeadResult.ContentType = %q, want text/plain", head.ContentType)
	}
	if head.Metadata["author"] != "integration-test" {
		t.Errorf("HeadResult.Metadata[author] = %q, want integration-test", head.Metadata["author"])
	}

	// Get
	download, err := client.GetObject(ctx, bucketName, key)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	defer download.Content.Close()
	body, err := io.ReadAll(download.Content)
	if err != nil {
		t.Fatalf("reading download body failed: %v", err)
	}
	if !bytes.Equal(body, content) {
		t.Errorf("downloaded content = %q, want %q", string(body), string(content))
	}

	// Copy
	copyKey := "test/hello-copy.txt"
	copyResult, err := client.CopyObject(ctx, bucketName, key, bucketName, copyKey)
	if err != nil {
		t.Fatalf("CopyObject failed: %v", err)
	}
	if copyResult.Key != copyKey {
		t.Errorf("CopyResult.Key = %q, want %q", copyResult.Key, copyKey)
	}

	// Verify copy exists
	_, err = client.HeadObject(ctx, bucketName, copyKey)
	if err != nil {
		t.Fatalf("HeadObject on copied key failed: %v", err)
	}

	// Delete original
	err = client.DeleteObject(ctx, bucketName, key)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// Verify deleted (should get not found)
	_, err = client.GetObject(ctx, bucketName, key)
	if err == nil {
		t.Error("GetObject should fail after delete")
	}

	// Clean up copy
	client.DeleteObject(ctx, bucketName, copyKey)
}

// TestIntegration_ObjectPagination tests listing with pagination.
func TestIntegration_ObjectPagination(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	bucketName := uniqueBucketName()

	// Setup bucket
	_, err := client.CreateBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}
	t.Cleanup(func() {
		result, _ := client.ListObjects(ctx, bucketName, "", "", 0, "")
		if result != nil {
			for _, obj := range result.Items {
				client.DeleteObject(ctx, bucketName, obj.Key)
			}
		}
		client.DeleteBucket(ctx, bucketName)
	})

	// Upload 15 objects
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("page/obj-%03d.txt", i)
		content := []byte(fmt.Sprintf("object %d", i))
		_, err := client.Upload(ctx, bucketName, key, bytes.NewReader(content), int64(len(content)))
		if err != nil {
			t.Fatalf("Upload %d failed: %v", i, err)
		}
	}

	// List with maxKeys=5
	result, err := client.ListObjects(ctx, bucketName, "page/", "", 5, "")
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if len(result.Items) != 5 {
		t.Errorf("len(Items) = %d, want 5", len(result.Items))
	}
	if !result.IsTruncated {
		t.Error("IsTruncated should be true with maxKeys=5 and 15 objects")
	}
	if result.NextToken == "" {
		t.Error("NextToken is empty but IsTruncated is true")
	}

	// Continue with next token
	result2, err := client.ListObjects(ctx, bucketName, "page/", "", 5, result.NextToken)
	if err != nil {
		t.Fatalf("ListObjects page 2 failed: %v", err)
	}
	if len(result2.Items) == 0 {
		t.Error("page 2 returned no items")
	}

	// Clean up
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("page/obj-%03d.txt", i)
		client.DeleteObject(ctx, bucketName, key)
	}
}

// TestIntegration_ErrorPaths tests error handling for missing resources.
func TestIntegration_ErrorPaths(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	// GetObject on non-existent key
	_, err := client.GetObject(ctx, "r2go2-nonexistent-bucket-xyz", "no-such-key.txt")
	if err == nil {
		t.Error("GetObject on missing bucket should fail")
	}

	// BucketExists on non-existent bucket
	exists, err := client.BucketExists(ctx, "r2go2-nonexistent-bucket-xyz")
	if err != nil {
		t.Fatalf("BucketExists on non-existent bucket errored: %v", err)
	}
	if exists {
		t.Error("BucketExists should return false for non-existent bucket")
	}
}

// TestIntegration_TestConnection verifies connection testing works.
func TestIntegration_TestConnection(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

// TestIntegration_MetadataRoundTrip verifies content type and metadata survive upload/download.
func TestIntegration_MetadataRoundTrip(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	bucketName := uniqueBucketName()

	_, err := client.CreateBucket(ctx, bucketName)
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}
	t.Cleanup(func() {
		result, _ := client.ListObjects(ctx, bucketName, "", "", 0, "")
		if result != nil {
			for _, obj := range result.Items {
				client.DeleteObject(ctx, bucketName, obj.Key)
			}
		}
		client.DeleteBucket(ctx, bucketName)
	})

	key := "metadata-test.json"
	content := []byte(`{"test": true}`)
	meta := map[string]string{
		"env":     "integration",
		"version": "v0.3.1",
	}

	_, err = client.Upload(ctx, bucketName, key, bytes.NewReader(content), int64(len(content)),
		WithContentType("application/json"),
		WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	head, err := client.HeadObject(ctx, bucketName, key)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	if head.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want application/json", head.ContentType)
	}
	if head.Metadata["env"] != "integration" {
		t.Errorf("Metadata[env] = %q, want integration", head.Metadata["env"])
	}
	if head.Metadata["version"] != "v0.3.1" {
		t.Errorf("Metadata[version] = %q, want v0.3.1", head.Metadata["version"])
	}

	client.DeleteObject(ctx, bucketName, key)
}

// TestIntegration_Validation tests input validation without hitting the API.
func TestIntegration_Validation(t *testing.T) {
	_, err := NewClient()
	if err == nil {
		t.Error("NewClient without credentials should fail")
	}
	if !strings.Contains(err.Error(), "CLOUDFLARE_ACCOUNT_ID") {
		t.Errorf("error should mention CLOUDFLARE_ACCOUNT_ID, got: %v", err)
	}

	_, err = NewClient(WithAccountID("test"))
	if err == nil {
		t.Error("NewClient without API token should fail")
	}
	if !strings.Contains(err.Error(), "CLOUDFLARE_API_TOKEN") {
		t.Errorf("error should mention CLOUDFLARE_API_TOKEN, got: %v", err)
	}
}
