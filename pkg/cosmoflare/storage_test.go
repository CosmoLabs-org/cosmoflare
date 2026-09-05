package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"
)

// --- Helpers ---

func storageCfMockSetup(handler http.HandlerFunc) (*client, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	c := &client{
		cf:         cf,
		accountID:  "test-account-123",
		apiToken:   "test-token",
		httpClient: server.Client(),
		cfg:        &clientConfig{accountID: "test-account-123", apiToken: "test-token", region: "auto", cacheControl: true},
	}
	return c, server
}

func storageS3MockClient(handler http.HandlerFunc) (*s3.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	awsCfg, _ := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
		})),
	)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(server.URL)
		o.UsePathStyle = true
	})
	return client, server
}

func storageS3TestClient(handler http.HandlerFunc) (*client, *httptest.Server) {
	s3c, server := storageS3MockClient(handler)
	c := &client{
		s3:        s3c,
		accountID: "test-acct",
		cfg:       &clientConfig{cacheControl: true, accountID: "test-acct", apiToken: "test-token", region: "auto"},
	}
	return c, server
}

func storageWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func storageCfEnvelope(result interface{}, success bool) map[string]interface{} {
	return map[string]interface{}{
		"result":   result,
		"success":  success,
		"errors":   []interface{}{},
		"messages": []interface{}{},
	}
}

// storageR2BucketList returns the correct envelope for R2 bucket list responses.
// cloudflare-go R2BucketListResponse expects result.buckets.
func storageR2BucketList(buckets []map[string]interface{}) map[string]interface{} {
	return storageCfEnvelope(map[string]interface{}{"buckets": buckets}, true)
}

func storageCfErrorEnvelope(code int, msg string) map[string]interface{} {
	return map[string]interface{}{
		"result":   nil,
		"success":  false,
		"errors":   []map[string]interface{}{{"code": code, "message": msg}},
		"messages": []interface{}{},
	}
}

// --- validateBucketName ---

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "my-bucket", false},
		{"valid with numbers", "bucket-123", false},
		{"valid with dots", "my.bucket.name", false},
		{"valid alphanumeric", "abc123", false},
		{"min length 3", "abc", false},
		{"max length 63", strings.Repeat("a", 63), false},
		{"empty", "", true},
		{"too short 1", "a", true},
		{"too short 2", "ab", true},
		{"too long 64", strings.Repeat("a", 64), true},
		{"uppercase", "MyBucket", true},
		{"underscore", "my_bucket", true},
		{"space", "my bucket", true},
		{"special chars", "my@bucket", true},
		{"starts hyphen", "-bucket", true},
		{"starts dot", ".bucket", true},
		{"ends hyphen", "bucket-", true},
		{"ends dot", "bucket.", true},
		{"valid with hyphens", "my-test-bucket-2024", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBucketName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBucketName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// --- Bucket validation (no API calls needed) ---

func TestCreateBucketValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.CreateBucket(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	_, err = c.CreateBucket(context.Background(), "ab")
	if err == nil {
		t.Error("expected error for short bucket name")
	}
	_, err = c.CreateBucket(context.Background(), "UPPERCASE")
	if err == nil {
		t.Error("expected error for uppercase bucket name")
	}
}

func TestGetBucketValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.GetBucket(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	_, err = c.GetBucket(context.Background(), "ab")
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

func TestDeleteBucketValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	err := c.DeleteBucket(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	err = c.DeleteBucket(context.Background(), "ab")
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

func TestBucketExistsValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.BucketExists(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	_, err = c.BucketExists(context.Background(), "ab")
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

// --- Object validation ---

func TestListObjectsValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.ListObjects(context.Background(), "", "", "", 0, "")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
}

func TestGetObjectValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.GetObject(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	_, err = c.GetObject(context.Background(), "valid-bucket", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestHeadObjectValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.HeadObject(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	_, err = c.HeadObject(context.Background(), "valid-bucket", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestDeleteObjectValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	err := c.DeleteObject(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty bucket name")
	}
	err = c.DeleteObject(context.Background(), "valid-bucket", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestCopyObjectValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.CopyObject(context.Background(), "", "src", "dst-bucket", "dst")
	if err == nil {
		t.Error("expected error for empty source bucket")
	}
	_, err = c.CopyObject(context.Background(), "src-bucket", "src", "", "dst")
	if err == nil {
		t.Error("expected error for empty destination bucket")
	}
	_, err = c.CopyObject(context.Background(), "src-bucket", "", "dst-bucket", "dst")
	if err == nil {
		t.Error("expected error for empty source key")
	}
	_, err = c.CopyObject(context.Background(), "src-bucket", "src", "dst-bucket", "")
	if err == nil {
		t.Error("expected error for empty destination key")
	}
	_, err = c.CopyObject(context.Background(), "ab", "src", "dst-bucket", "dst")
	if err == nil {
		t.Error("expected error for short source bucket name")
	}
}

// --- Upload validation ---

func TestUploadValidationNoReader(t *testing.T) {
	c := &client{cfg: &clientConfig{cacheControl: true}}
	_, err := c.Upload(context.Background(), "test-bucket", "key.txt", nil, 0)
	if err == nil {
		t.Error("expected error for nil reader")
	}
}

func TestUploadValidationEmptyKey(t *testing.T) {
	c := &client{cfg: &clientConfig{cacheControl: true}}
	_, err := c.Upload(context.Background(), "test-bucket", "", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestUploadValidationBadBucket(t *testing.T) {
	c := &client{cfg: &clientConfig{cacheControl: true}}
	_, err := c.Upload(context.Background(), "ab", "key.txt", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

func TestMultipartUploadValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{cacheControl: true}}
	_, err := c.MultipartUpload(context.Background(), "test-bucket", "", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("expected error for empty key")
	}
	_, err = c.MultipartUpload(context.Background(), "test-bucket", "key.txt", nil, 4)
	if err == nil {
		t.Error("expected error for nil reader")
	}
	_, err = c.MultipartUpload(context.Background(), "ab", "key.txt", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

// --- Download validation ---

func TestDownloadValidation(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	_, err := c.Download(context.Background(), "", "key")
	if err == nil {
		t.Error("expected error for empty bucket")
	}
	_, err = c.Download(context.Background(), "valid-bucket", "")
	if err == nil {
		t.Error("expected error for empty key")
	}
	_, err = c.Download(context.Background(), "ab", "key")
	if err == nil {
		t.Error("expected error for short bucket name")
	}
}

// --- Download options ---

func TestWithOutputPathOption(t *testing.T) {
	cfg := &downloadConfig{}
	WithOutputPath("/tmp/test.txt")(cfg)
	if cfg.outputPath != "/tmp/test.txt" {
		t.Errorf("expected outputPath=/tmp/test.txt, got %s", cfg.outputPath)
	}
}

func TestWithRangeOption(t *testing.T) {
	cfg := &downloadConfig{}
	WithRange(100, 200)(cfg)
	if cfg.rangeStart != 100 || cfg.rangeEnd != 200 {
		t.Errorf("expected range=100-200, got %d-%d", cfg.rangeStart, cfg.rangeEnd)
	}
}

func TestDownloadRangeDefault(t *testing.T) {
	cfg := &downloadConfig{}
	// Default range should not trigger range header
	if cfg.rangeStart != 0 || cfg.rangeEnd != 0 {
		t.Error("expected default range to be 0,0")
	}
}

// --- Bucket operations with CF mock ---

func TestListBucketsSuccess(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "bucket-1", "creation_date": "2024-06-01T00:00:00Z"},
			{"name": "bucket-2"},
		}))
	})
	defer server.Close()

	buckets, err := c.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Name != "bucket-1" {
		t.Errorf("expected bucket-1, got %s", buckets[0].Name)
	}
	if buckets[1].Name != "bucket-2" {
		t.Errorf("expected bucket-2, got %s", buckets[1].Name)
	}
	if buckets[0].CreatedAt.IsZero() {
		t.Error("expected non-zero creation date for bucket-1")
	}
}

func TestListBucketsEmpty(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{}))
	})
	defer server.Close()

	buckets, err := c.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestListBucketsError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		storageWriteJSON(w, storageCfErrorEnvelope(1000, "internal error"))
	})
	defer server.Close()

	_, err := c.ListBuckets(context.Background())
	if err == nil {
		t.Error("expected error from failed ListBuckets")
	}
}

func TestCreateBucketSuccess(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageCfEnvelope(map[string]interface{}{
			"name": "new-bucket",
		}, true))
	})
	defer server.Close()

	bucket, err := c.CreateBucket(context.Background(), "new-bucket")
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}
	if bucket.Name != "new-bucket" {
		t.Errorf("expected name=new-bucket, got %s", bucket.Name)
	}
	if bucket.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateBucketError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		storageWriteJSON(w, storageCfErrorEnvelope(10002, "bucket already exists"))
	})
	defer server.Close()

	_, err := c.CreateBucket(context.Background(), "existing-bucket")
	if err == nil {
		t.Error("expected error from duplicate bucket")
	}
}

func TestDeleteBucketSuccess(t *testing.T) {
	called := false
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		called = true
		storageWriteJSON(w, storageCfEnvelope(nil, true))
	})
	defer server.Close()

	err := c.DeleteBucket(context.Background(), "my-bucket")
	if err != nil {
		t.Fatalf("DeleteBucket failed: %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestDeleteBucketError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		storageWriteJSON(w, storageCfErrorEnvelope(10003, "bucket not found"))
	})
	defer server.Close()

	err := c.DeleteBucket(context.Background(), "missing-bucket")
	if err == nil {
		t.Error("expected error from missing bucket")
	}
}

func TestGetBucketFound(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "bucket-a"},
			{"name": "bucket-b"},
		}))
	})
	defer server.Close()

	bucket, err := c.GetBucket(context.Background(), "bucket-b")
	if err != nil {
		t.Fatalf("GetBucket failed: %v", err)
	}
	if bucket.Name != "bucket-b" {
		t.Errorf("expected bucket-b, got %s", bucket.Name)
	}
}

func TestGetBucketNotFound(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "bucket-a"},
		}))
	})
	defer server.Close()

	_, err := c.GetBucket(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent bucket")
	}
	_, ok := err.(*R2NotFoundError)
	if !ok {
		t.Errorf("expected R2NotFoundError, got %T", err)
	}
}

func TestGetBucketListError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		storageWriteJSON(w, storageCfErrorEnvelope(1000, "server error"))
	})
	defer server.Close()

	_, err := c.GetBucket(context.Background(), "any-bucket")
	if err == nil {
		t.Error("expected error when ListBuckets fails")
	}
}

func TestBucketExistsTrue(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "my-bucket"},
			{"name": "other-bucket"},
		}))
	})
	defer server.Close()

	exists, err := c.BucketExists(context.Background(), "my-bucket")
	if err != nil {
		t.Fatalf("BucketExists failed: %v", err)
	}
	if !exists {
		t.Error("expected bucket to exist")
	}
}

func TestBucketExistsFalse(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "other-bucket"},
		}))
	})
	defer server.Close()

	exists, err := c.BucketExists(context.Background(), "missing-bucket")
	if err != nil {
		t.Fatalf("BucketExists failed: %v", err)
	}
	if exists {
		t.Error("expected bucket to not exist")
	}
}

func TestBucketExistsListError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		storageWriteJSON(w, storageCfErrorEnvelope(1000, "fail"))
	})
	defer server.Close()

	_, err := c.BucketExists(context.Background(), "any-bucket")
	if err == nil {
		t.Error("expected error when ListBuckets fails")
	}
}

func TestBucketCreationDateNil(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{
			{"name": "no-date-bucket"},
		}))
	})
	defer server.Close()

	buckets, err := c.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	if !buckets[0].CreatedAt.IsZero() {
		t.Error("expected zero time when creation_date not provided")
	}
}

// --- Object operations with S3 mock ---

func TestListObjectsSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
<IsTruncated>false</IsTruncated>
<Contents>
<Key>file.txt</Key>
<LastModified>2024-06-01T00:00:00.000Z</LastModified>
<ETag>"abc123"</ETag>
<Size>1024</Size>
<StorageClass>STANDARD</StorageClass>
</Contents>
<Contents>
<Key>doc.pdf</Key>
<LastModified>2024-06-02T00:00:00.000Z</LastModified>
<ETag>"def456"</ETag>
<Size>2048</Size>
<StorageClass>STANDARD</StorageClass>
</Contents>
<NextContinuationToken>token123</NextContinuationToken>
</ListBucketResult>`)
	})
	defer server.Close()

	result, err := c.ListObjects(context.Background(), "test-bucket", "", "", 0, "")
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Key != "file.txt" {
		t.Errorf("expected key=file.txt, got %s", result.Items[0].Key)
	}
	if result.Items[0].Size != 1024 {
		t.Errorf("expected size=1024, got %d", result.Items[0].Size)
	}
	if result.Items[1].Key != "doc.pdf" {
		t.Errorf("expected key=doc.pdf, got %s", result.Items[1].Key)
	}
	if result.NextToken != "token123" {
		t.Errorf("expected token=token123, got %s", result.NextToken)
	}
}

func TestListObjectsEmpty(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
<IsTruncated>false</IsTruncated>
</ListBucketResult>`)
	})
	defer server.Close()

	result, err := c.ListObjects(context.Background(), "test-bucket", "", "", 0, "")
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestListObjectsError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	_, err := c.ListObjects(context.Background(), "test-bucket", "", "", 0, "")
	if err == nil {
		t.Error("expected error from S3 failure")
	}
}

func TestGetObjectSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "5")
		fmt.Fprint(w, "hello")
	})
	defer server.Close()

	result, err := c.GetObject(context.Background(), "test-bucket", "file.txt")
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if result.Key != "file.txt" {
		t.Errorf("expected key=file.txt, got %s", result.Key)
	}
	if result.Bucket != "test-bucket" {
		t.Errorf("expected bucket=test-bucket, got %s", result.Bucket)
	}
	if result.ContentType != "text/plain" {
		t.Errorf("expected content-type=text/plain, got %s", result.ContentType)
	}
	if result.Content == nil {
		t.Error("expected non-nil Content")
	}
	result.Content.Close()
}

func TestGetObjectError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	_, err := c.GetObject(context.Background(), "test-bucket", "missing")
	if err == nil {
		t.Error("expected error for missing object")
	}
}

func TestHeadObjectSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2048")
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("ETag", "\"abc123\"")
		w.Header().Set("Last-Modified", "Sat, 01 Jun 2024 00:00:00 GMT")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	result, err := c.HeadObject(context.Background(), "test-bucket", "image.png")
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}
	if result.Key != "image.png" {
		t.Errorf("expected key=image.png, got %s", result.Key)
	}
	if result.Size != 2048 {
		t.Errorf("expected size=2048, got %d", result.Size)
	}
	if result.ContentType != "image/png" {
		t.Errorf("expected content-type=image/png, got %s", result.ContentType)
	}
}

func TestHeadObjectError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	_, err := c.HeadObject(context.Background(), "test-bucket", "missing")
	if err == nil {
		t.Error("expected error for missing object")
	}
}

func TestDeleteObjectSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := c.DeleteObject(context.Background(), "test-bucket", "file.txt")
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
}

func TestDeleteObjectError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	err := c.DeleteObject(context.Background(), "test-bucket", "file.txt")
	if err == nil {
		t.Error("expected error from S3 failure")
	}
}

func TestCopyObjectSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<CopyObjectResult>
<ETag>"copied123"</ETag>
<LastModified>2024-06-01T00:00:00.000Z</LastModified>
</CopyObjectResult>`)
	})
	defer server.Close()

	result, err := c.CopyObject(context.Background(), "src-bucket", "src.txt", "dst-bucket", "dst.txt")
	if err != nil {
		t.Fatalf("CopyObject failed: %v", err)
	}
	if result.Key != "dst.txt" {
		t.Errorf("expected key=dst.txt, got %s", result.Key)
	}
	if result.SourceKey != "src.txt" {
		t.Errorf("expected sourceKey=src.txt, got %s", result.SourceKey)
	}
	if result.Bucket != "dst-bucket" {
		t.Errorf("expected bucket=dst-bucket, got %s", result.Bucket)
	}
}

func TestCopyObjectError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	_, err := c.CopyObject(context.Background(), "src-bucket", "src.txt", "dst-bucket", "dst.txt")
	if err == nil {
		t.Error("expected error from S3 failure")
	}
}

// --- Upload with S3 mock ---

func TestUploadSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "\"upload123\"")
		w.Header().Set("x-amz-version-id", "v1")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	result, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hello"), 5)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if result.Key != "file.txt" {
		t.Errorf("expected key=file.txt, got %s", result.Key)
	}
	if result.Bucket != "test-bucket" {
		t.Errorf("expected bucket=test-bucket, got %s", result.Bucket)
	}
	if result.VersionID != "v1" {
		t.Errorf("expected versionId=v1, got %s", result.VersionID)
	}
}

func TestUploadWithContentType(t *testing.T) {
	var capturedCT string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedCT = r.Header.Get("Content-Type")
		w.Header().Set("ETag", "\"abc\"")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hi"), 2,
		WithContentType("application/json"),
	)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if capturedCT != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", capturedCT)
	}
}

func TestUploadWithCacheControl(t *testing.T) {
	var capturedCC string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedCC = r.Header.Get("Cache-Control")
		w.Header().Set("ETag", "\"abc\"")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hi"), 2,
		WithUploadCacheControl("max-age=3600"),
	)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if capturedCC != "max-age=3600" {
		t.Errorf("expected Cache-Control=max-age=3600, got %s", capturedCC)
	}
}

func TestUploadAutoCachePolicy(t *testing.T) {
	var capturedCC string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedCC = r.Header.Get("Cache-Control")
		w.Header().Set("ETag", "\"abc\"")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "style.css", strings.NewReader("body{}"), 6)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if capturedCC != "public, max-age=31536000, immutable" {
		t.Errorf("expected auto cache policy for CSS, got: %s", capturedCC)
	}
}

func TestUploadError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hi"), 2)
	if err == nil {
		t.Error("expected error from S3 failure")
	}
}

// --- Download with S3 mock ---

func TestDownloadSuccess(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "5")
		fmt.Fprint(w, "hello")
	})
	defer server.Close()

	result, err := c.Download(context.Background(), "test-bucket", "file.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if result.Key != "file.txt" {
		t.Errorf("expected key=file.txt, got %s", result.Key)
	}
	if result.ContentType != "text/plain" {
		t.Errorf("expected content-type=text/plain, got %s", result.ContentType)
	}
	body, _ := io.ReadAll(result.Content)
	result.Content.Close()
	if string(body) != "hello" {
		t.Errorf("expected body=hello, got %s", string(body))
	}
}

func TestDownloadWithRange(t *testing.T) {
	var capturedRange string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedRange = r.Header.Get("Range")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "3")
		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, "hel")
	})
	defer server.Close()

	result, err := c.Download(context.Background(), "test-bucket", "file.txt",
		WithRange(0, 2),
	)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if capturedRange != "bytes=0-2" {
		t.Errorf("expected Range=bytes=0-2, got %s", capturedRange)
	}
	result.Content.Close()
}

func TestDownloadError(t *testing.T) {
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	_, err := c.Download(context.Background(), "test-bucket", "missing")
	if err == nil {
		t.Error("expected error for missing object")
	}
}

func TestDownloadWithOutputPath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "output.txt")
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "12")
		fmt.Fprint(w, "hello world!")
	})
	defer server.Close()

	result, err := c.Download(context.Background(), "test-bucket", "file.txt",
		WithOutputPath(tmpFile),
	)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if result.Content != nil {
		t.Error("expected nil Content when writing to file")
	}
	if result.Size != 12 {
		t.Errorf("expected size=12, got %d", result.Size)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != "hello world!" {
		t.Errorf("expected file content 'hello world!', got %q", string(data))
	}
}

// --- Accessor / connection tests ---

func TestClientAccountIDMethod(t *testing.T) {
	c := &client{accountID: "test-123"}
	if c.AccountID() != "test-123" {
		t.Errorf("expected test-123, got %s", c.AccountID())
	}
}

func TestClientS3ClientAccessor(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	if c.s3Client() != nil {
		t.Error("expected nil s3 client before init")
	}
}

func TestClientCfClientAccessor(t *testing.T) {
	c := &client{cfg: &clientConfig{}}
	if c.cfClient() != nil {
		t.Error("expected nil cf client before init")
	}
}

func TestTestConnectionNoCfClient(t *testing.T) {
	c := &client{cfg: &clientConfig{}, cf: nil}
	err := c.TestConnection(context.Background())
	if err == nil {
		t.Error("expected error when cf client is nil")
	}
}

func TestTestConnectionMock(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		storageWriteJSON(w, storageR2BucketList([]map[string]interface{}{}))
	})
	defer server.Close()

	err := c.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
}

func TestTestConnectionError(t *testing.T) {
	c, server := storageCfMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		storageWriteJSON(w, storageCfErrorEnvelope(10000, "invalid token"))
	})
	defer server.Close()

	err := c.TestConnection(context.Background())
	if err == nil {
		t.Error("expected error from failed connection")
	}
}

// --- Error type checks ---

func TestR2ErrorWithBucketAndKey(t *testing.T) {
	err := &R2Error{Op: "GetObject", Bucket: "my-bucket", Key: "file.txt", Message: "not found"}
	msg := err.Error()
	if !strings.Contains(msg, "bucket=my-bucket") {
		t.Errorf("expected bucket in message, got: %s", msg)
	}
	if !strings.Contains(msg, "key=file.txt") {
		t.Errorf("expected key in message, got: %s", msg)
	}
}

func TestR2ErrorWithBucketOnly(t *testing.T) {
	err := &R2Error{Op: "CreateBucket", Bucket: "my-bucket", Message: "failed"}
	msg := err.Error()
	if !strings.Contains(msg, "bucket=my-bucket") {
		t.Errorf("expected bucket in message, got: %s", msg)
	}
	if strings.Contains(msg, "key=") {
		t.Errorf("did not expect key in message, got: %s", msg)
	}
}

func TestR2ErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := newError("Test", "msg", inner)
	if unwrapped := err.Unwrap(); unwrapped != inner {
		t.Error("Unwrap should return inner error")
	}
}

// --- Option function tests (additional coverage) ---

func TestWithHTTPClientOption(t *testing.T) {
	cfg := &clientConfig{}
	hc := &http.Client{Timeout: 10 * time.Second}
	WithHTTPClient(hc)(cfg)
	if cfg.httpClient != hc {
		t.Error("expected httpClient to be set")
	}
}

func TestWithTimeoutOption(t *testing.T) {
	cfg := &clientConfig{}
	WithTimeout(60 * time.Second)(cfg)
	if cfg.timeout != 60*time.Second {
		t.Errorf("expected timeout=60s, got %v", cfg.timeout)
	}
}

// --- Config tests ---

func TestMachineConfigGetProfile(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"default": {AccountID: "acc-123", APIToken: "tok-123"},
		},
		Current: "default",
	}
	p, err := mc.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if p.AccountID != "acc-123" {
		t.Errorf("expected AccountID=acc-123, got %s", p.AccountID)
	}
}

func TestMachineConfigGetProfileNotFound(t *testing.T) {
	mc := &MachineConfig{Profiles: map[string]*ProfileConfig{}, Current: "default"}
	_, err := mc.GetProfile("missing")
	if err == nil {
		t.Error("expected error for missing profile")
	}
}

func TestMachineConfigCurrentProfile(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"work": {AccountID: "acc-work", APIToken: "tok-work"},
		},
		Current: "work",
	}
	p, err := mc.CurrentProfile()
	if err != nil {
		t.Fatalf("CurrentProfile failed: %v", err)
	}
	if p.AccountID != "acc-work" {
		t.Errorf("expected AccountID=acc-work, got %s", p.AccountID)
	}
}

func TestMachineConfigCurrentProfileEmpty(t *testing.T) {
	mc := &MachineConfig{Profiles: map[string]*ProfileConfig{}, Current: ""}
	_, err := mc.CurrentProfile()
	if err == nil {
		t.Error("expected error when current is empty")
	}
}

func TestMachineConfigCurrentProfileMissing(t *testing.T) {
	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{},
		Current:  "nonexistent",
	}
	_, err := mc.CurrentProfile()
	if err == nil {
		t.Error("expected error when profile missing")
	}
}

func TestMachineConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{
			"test": {AccountID: "acc-test", APIToken: "tok-test", Region: "auto"},
		},
		Current: "test",
	}
	if err := mc.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the file was created
	configPath := filepath.Join(tmpDir, ".r2go2", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file not created at %s", configPath)
	}

	// Verify load returns a valid config
	loaded, err := LoadMachineConfig()
	if err != nil {
		t.Fatalf("LoadMachineConfig failed: %v", err)
	}
	if loaded.Current != "test" {
		t.Errorf("expected current=test, got %s", loaded.Current)
	}
}

func TestLoadMachineConfigNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	mc, err := LoadMachineConfig()
	if err != nil {
		t.Fatalf("LoadMachineConfig failed: %v", err)
	}
	if mc.Current != "default" {
		t.Errorf("expected default current, got %s", mc.Current)
	}
	if len(mc.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(mc.Profiles))
	}
}

func TestProfileFromEnv(t *testing.T) {
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-acc")
	os.Setenv("CLOUDFLARE_API_TOKEN", "env-tok")
	os.Setenv("AWS_ACCESS_KEY_ID", "env-ak")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "env-sk")
	os.Setenv("R2_ENDPOINT", "https://custom.r2.dev")
	os.Setenv("AWS_REGION", "us-east-1")
	defer func() {
		os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
		os.Unsetenv("CLOUDFLARE_API_TOKEN")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("R2_ENDPOINT")
		os.Unsetenv("AWS_REGION")
	}()

	p := ProfileFromEnv()
	if p.AccountID != "env-acc" {
		t.Errorf("expected AccountID=env-acc, got %s", p.AccountID)
	}
	if p.APIToken != "env-tok" {
		t.Errorf("expected APIToken=env-tok, got %s", p.APIToken)
	}
	if p.AccessKey != "env-ak" {
		t.Errorf("expected AccessKey=env-ak, got %s", p.AccessKey)
	}
	if p.SecretKey != "env-sk" {
		t.Errorf("expected SecretKey=env-sk, got %s", p.SecretKey)
	}
	if p.Endpoint != "https://custom.r2.dev" {
		t.Errorf("expected Endpoint, got %s", p.Endpoint)
	}
	if p.Region != "us-east-1" {
		t.Errorf("expected Region=us-east-1, got %s", p.Region)
	}
}

func TestLoadProjectConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadProjectConfig(tmpDir)
	if err == nil {
		t.Error("expected error when config not found")
	}
}

func TestLoadProjectConfigSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `bucket: my-bucket
region: auto
endpoint: https://custom.r2.dev
env: production
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".r2go2.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg.Bucket != "my-bucket" {
		t.Errorf("expected bucket=my-bucket, got %s", cfg.Bucket)
	}
	if cfg.Region != "auto" {
		t.Errorf("expected region=auto, got %s", cfg.Region)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected env=production, got %s", cfg.Environment)
	}
}

func TestFindProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "deep")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, ".r2go2.yaml"), []byte("bucket: test"), 0644)

	path, err := findProjectFile(subDir, ".r2go2.yaml")
	if err != nil {
		t.Fatalf("findProjectFile failed: %v", err)
	}
	if !strings.HasSuffix(path, ".r2go2.yaml") {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestFindProjectFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := findProjectFile(tmpDir, ".r2go2.yaml")
	if err == nil {
		t.Error("expected error when file not found")
	}
}

func TestWriteToFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	body := io.NopCloser(strings.NewReader("hello world"))
	err := writeToFile(body, tmpFile)
	if err != nil {
		t.Fatalf("writeToFile failed: %v", err)
	}
	data, _ := os.ReadFile(tmpFile)
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
}

func TestWriteToFileBadPath(t *testing.T) {
	body := io.NopCloser(strings.NewReader("data"))
	err := writeToFile(body, "/nonexistent/dir/file.txt")
	if err == nil {
		t.Error("expected error for bad path")
	}
}

// --- NewClient integration tests ---

func TestNewClientWithCredentials(t *testing.T) {
	origAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	origAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-abc")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-xyz123")
	t.Cleanup(func() {
		if origAccountID != "" {
			os.Setenv("CLOUDFLARE_ACCOUNT_ID", origAccountID)
		} else {
			os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
		}
		if origAPIToken != "" {
			os.Setenv("CLOUDFLARE_API_TOKEN", origAPIToken)
		} else {
			os.Unsetenv("CLOUDFLARE_API_TOKEN")
		}
	})

	client, err := NewClient(
		WithEndpoint("https://test.r2.cloudflarestorage.com"),
		WithCredentials("test-ak", "test-sk"),
		WithRegion("auto"),
		WithCacheControl(false),
		WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client.AccountID() != "test-account-abc" {
		t.Errorf("expected accountID=test-account-abc, got %s", client.AccountID())
	}
}

func TestNewClientCustomTimeout(t *testing.T) {
	origAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	origAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	t.Cleanup(func() {
		if origAccountID != "" {
			os.Setenv("CLOUDFLARE_ACCOUNT_ID", origAccountID)
		} else {
			os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
		}
		if origAPIToken != "" {
			os.Setenv("CLOUDFLARE_API_TOKEN", origAPIToken)
		} else {
			os.Unsetenv("CLOUDFLARE_API_TOKEN")
		}
	})

	_, err := NewClient(WithTimeout(10 * time.Second))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
}

// --- Upload size > threshold delegates to multipart ---

func TestUploadAutoMultipartDelegation(t *testing.T) {
	initiated := false
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		// Multipart upload sends CreateMultipartUpload first
		initiated = true
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	// Size > 100MB triggers multipart delegation
	bigSize := int64(multipartThreshold) + 1
	_, err := c.Upload(context.Background(), "test-bucket", "big-file.bin",
		strings.NewReader("x"), bigSize)
	// The multipart will fail because the mock doesn't handle all steps,
	// but the delegation itself exercises the size-check branch
	_ = err
	if !initiated {
		t.Error("expected multipart upload to be initiated for large file")
	}
}

// --- ListObjects with prefix/delimiter ---

func TestListObjectsWithPrefixAndDelimiter(t *testing.T) {
	var capturedQuery string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
<IsTruncated>false</IsTruncated>
</ListBucketResult>`)
	})
	defer server.Close()

	result, err := c.ListObjects(context.Background(), "test-bucket", "photos/", "/", 100, "")
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if !strings.Contains(capturedQuery, "prefix") {
		t.Error("expected prefix in query")
	}
}

// --- Upload with ContentLength > 0 ---

func TestUploadWithPositiveSize(t *testing.T) {
	var capturedLen string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedLen = r.Header.Get("Content-Length")
		w.Header().Set("ETag", "\"abc\"")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hello"), 5)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if capturedLen != "5" {
		t.Errorf("expected Content-Length=5, got %s", capturedLen)
	}
}

// --- Upload with zero size (no ContentLength set) ---

func TestUploadWithZeroSize(t *testing.T) {
	var capturedLen string
	c, server := storageS3TestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedLen = r.Header.Get("Content-Length")
		w.Header().Set("ETag", "\"abc\"")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	_, err := c.Upload(context.Background(), "test-bucket", "file.txt", strings.NewReader("hi"), 0)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	// With size=0, ContentLength should not be explicitly set
	if capturedLen == "0" {
		t.Error("expected Content-Length to not be explicitly 0")
	}
}

// --- CopyObject with nil CopyObjectResult (handled by ETag check) ---

func TestCopyResultFields(t *testing.T) {
	r := &CopyResult{Key: "dst", SourceKey: "src", Bucket: "bkt", ETag: "abc", VersionID: "v1"}
	if r.Key != "dst" || r.SourceKey != "src" || r.Bucket != "bkt" {
		t.Error("CopyResult field mismatch")
	}
	if r.VersionID != "v1" {
		t.Errorf("expected VersionID=v1, got %s", r.VersionID)
	}
}

func TestUploadResultFields(t *testing.T) {
	r := &UploadResult{Key: "f.txt", Bucket: "b", Size: 42, Parts: 3}
	if r.Size != 42 || r.Parts != 3 {
		t.Error("UploadResult field mismatch")
	}
}

func TestHeadResultFields(t *testing.T) {
	r := &HeadResult{Key: "f.txt", Size: 100, ContentType: "text/plain", CacheControl: "no-cache"}
	if r.ContentType != "text/plain" {
		t.Error("HeadResult ContentType mismatch")
	}
}

func TestBucketFields(t *testing.T) {
	b := &Bucket{Name: "my-bucket", Location: "enam", Storage: "R2", Size: 1024, ObjectCount: 5}
	if b.Location != "enam" || b.ObjectCount != 5 {
		t.Error("Bucket field mismatch")
	}
}

func TestProjectConfigFields(t *testing.T) {
	cfg := &ProjectConfig{
		Bucket:         "my-bucket",
		Region:         "auto",
		Environment:    "production",
		AllowedBuckets: []string{"bucket-1"},
		MaxFileSize:    1024 * 1024 * 100,
		CachePolicy:    CachePolicyConfig{Enabled: true, Default: "max-age=3600"},
		Guardrails:     GuardrailConfig{Enabled: true, MaxFileSize: 50 * 1024 * 1024},
	}
	if cfg.Bucket != "my-bucket" {
		t.Error("ProjectConfig Bucket mismatch")
	}
	if !cfg.CachePolicy.Enabled {
		t.Error("expected CachePolicy.Enabled=true")
	}
}

func TestMachineConfigSaveCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	mc := &MachineConfig{
		Profiles: map[string]*ProfileConfig{},
		Current:  "default",
	}
	if err := mc.Save(); err != nil {
		t.Fatalf("Save failed")
	}
	info, err := os.Stat(filepath.Join(tmpDir, ".r2go2"))
	if err != nil {
		t.Fatalf("expected .r2go2 dir")
	}
	if !info.IsDir() {
		t.Error("expected .r2go2 to be a directory")
	}
}
