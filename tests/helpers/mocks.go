package helpers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MockCloudflareServer creates a mock Cloudflare R2 API server
func MockCloudflareServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock account endpoint
	mux.HandleFunc("/accounts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"result": [
				{
					"id": "test-account-id",
					"name": "Test Account",
					"status": "active"
				}
			],
			"success": true,
			"errors": [],
			"messages": []
		}`))
	})

	// Mock R2 buckets endpoint
	mux.HandleFunc("/accounts/test-account-id/r2/buckets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"result": [
					{
						"name": "test-bucket",
						"creation_date": "2024-01-01T00:00:00Z",
						"location": {
							"type": "region",
							"name": "auto"
						}
					}
				],
				"success": true,
				"errors": [],
				"messages": []
			}`))
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"result": {
					"name": "new-bucket",
					"creation_date": "2024-01-01T00:00:00Z"
				},
				"success": true,
				"errors": [],
				"messages": []
			}`))
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Mock bucket operations
	mux.HandleFunc("/accounts/test-account-id/r2/buckets/test-bucket", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"errors": [],
				"messages": []
			}`))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

// MockS3Client creates a mock S3 client for testing
type MockS3Client struct {
	Buckets map[string]bool
	Objects map[string][]string // bucket -> objects
}

// NewMockS3Client creates a new mock S3 client
func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		Buckets: make(map[string]bool),
		Objects: make(map[string][]string),
	}
}

// CreateBucket mocks the CreateBucket API call
func (m *MockS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	bucket := aws.ToString(params.Bucket)
	if _, exists := m.Buckets[bucket]; exists {
		return nil, fmt.Errorf("bucket already exists: %s", bucket)
	}

	m.Buckets[bucket] = true
	m.Objects[bucket] = []string{}

	return &s3.CreateBucketOutput{
		Location: aws.String("/" + bucket),
	}, nil
}

// ListBuckets mocks the ListBuckets API call
func (m *MockS3Client) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	var buckets []types.Bucket
	for bucket := range m.Buckets {
		buckets = append(buckets, types.Bucket{
			Name: aws.String(bucket),
			CreationDate: aws.Time(time.Now()),
		})
	}

	return &s3.ListBucketsOutput{
		Buckets: buckets,
	}, nil
}

// DeleteBucket mocks the DeleteBucket API call
func (m *MockS3Client) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	bucket := aws.ToString(params.Bucket)
	if _, exists := m.Buckets[bucket]; !exists {
		return nil, fmt.Errorf("bucket does not exist: %s", bucket)
	}

	delete(m.Buckets, bucket)
	delete(m.Objects, bucket)

	return &s3.DeleteBucketOutput{}, nil
}

// PutObject mocks the PutObject API call
func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	bucket := aws.ToString(params.Bucket)
	key := aws.ToString(params.Key)

	if _, exists := m.Buckets[bucket]; !exists {
		return nil, fmt.Errorf("bucket does not exist: %s", bucket)
	}

	m.Objects[bucket] = append(m.Objects[bucket], key)

	return &s3.PutObjectOutput{
		ETag: aws.String(fmt.Sprintf("\"%s\"", RandomString(32))),
	}, nil
}

// ListObjects mocks the ListObjects API call
func (m *MockS3Client) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	bucket := aws.ToString(params.Bucket)
	if _, exists := m.Buckets[bucket]; !exists {
		return nil, fmt.Errorf("bucket does not exist: %s", bucket)
	}

	var objects []types.Object
	for _, key := range m.Objects[bucket] {
		if params.Prefix != nil && !strings.HasPrefix(key, aws.ToString(params.Prefix)) {
			continue
		}

		objects = append(objects, types.Object{
			Key:          aws.String(key),
			LastModified: aws.Time(time.Now()),
			ETag:         aws.String(fmt.Sprintf("\"%s\"", RandomString(32))),
			Size:         aws.Int64(1024),
			StorageClass: types.ObjectStorageClassStandard,
		})
	}

	return &s3.ListObjectsOutput{
		Contents: objects,
	}, nil
}

// DeleteObject mocks the DeleteObject API call
func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	bucket := aws.ToString(params.Bucket)
	key := aws.ToString(params.Key)

	if _, exists := m.Buckets[bucket]; !exists {
		return nil, fmt.Errorf("bucket does not exist: %s", bucket)
	}

	objects := m.Objects[bucket]
	for i, objKey := range objects {
		if objKey == key {
			m.Objects[bucket] = append(objects[:i], objects[i+1:]...)
			break
		}
	}

	return &s3.DeleteObjectOutput{}, nil
}

// MockTerminal creates a mock terminal for TUI testing
type MockTerminal struct {
	Input  chan string
	Output []string
	Width  int
	Height int
}

// NewMockTerminal creates a new mock terminal
func NewMockTerminal(width, height int) *MockTerminal {
	return &MockTerminal{
		Input:  make(chan string, 100),
		Output: []string{},
		Width:  width,
		Height: height,
	}
}

// SendInput simulates user input
func (m *MockTerminal) SendInput(input string) {
	m.Input <- input
}

// CaptureOutput captures terminal output
func (m *MockTerminal) CaptureOutput(output string) {
	m.Output = append(m.Output, output)
}

// GetOutput returns captured output
func (m *MockTerminal) GetOutput() []string {
	return m.Output
}

// ClearOutput clears captured output
func (m *MockTerminal) ClearOutput() {
	m.Output = []string{}
}

// Close closes the mock terminal
func (m *MockTerminal) Close() {
	close(m.Input)
}

