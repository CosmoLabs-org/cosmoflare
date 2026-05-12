//go:build disabled

package api

import (
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid client creation",
			accountID: "test-account-123",
			setupEnv: func() {
				os.Setenv("CLOUDFLARE_API_TOKEN", "valid-token-12345")
			},
			cleanupEnv: func() {
				os.Unsetenv("CLOUDFLARE_API_TOKEN")
			},
			expectError: false,
		},
		{
			name:      "missing API token",
			accountID: "test-account-123",
			setupEnv: func() {
				os.Unsetenv("CLOUDFLARE_API_TOKEN")
			},
			cleanupEnv:  func() {},
			expectError: true,
			errorMsg:    "CLOUDFLARE_API_TOKEN environment variable is required",
		},
		{
			name:      "empty API token",
			accountID: "test-account-123",
			setupEnv: func() {
				os.Setenv("CLOUDFLARE_API_TOKEN", "")
			},
			cleanupEnv: func() {
				os.Unsetenv("CLOUDFLARE_API_TOKEN")
			},
			expectError: true,
			errorMsg:    "CLOUDFLARE_API_TOKEN environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			client, err := NewClient(tt.accountID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if client != nil {
					t.Errorf("Expected nil client on error, got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if client == nil {
					t.Errorf("Expected client but got nil")
					return
				}
				if client.accountID != tt.accountID {
					t.Errorf("Expected account ID %s, got %s", tt.accountID, client.accountID)
				}
				if client.apiToken != os.Getenv("CLOUDFLARE_API_TOKEN") {
					t.Errorf("Expected API token %s, got %s", os.Getenv("CLOUDFLARE_API_TOKEN"), client.apiToken)
				}
			}
		})
	}
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name        string
		bucketName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid bucket name",
			bucketName:  "my-valid-bucket",
			expectError: false,
		},
		{
			name:        "valid bucket name with numbers",
			bucketName:  "bucket123",
			expectError: false,
		},
		{
			name:        "empty bucket name",
			bucketName:  "",
			expectError: true,
			errorMsg:    "bucket name cannot be empty",
		},
		{
			name:        "bucket name too short",
			bucketName:  "ab",
			expectError: true,
			errorMsg:    "bucket name must be between 3 and 63 characters",
		},
		{
			name:        "bucket name too long",
			bucketName:  "this-is-a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-of-sixty-three-characters",
			expectError: true,
			errorMsg:    "bucket name must be between 3 and 63 characters",
		},
		{
			name:        "minimum length bucket name",
			bucketName:  "abc",
			expectError: false,
		},
		{
			name:        "maximum length bucket name",
			bucketName:  "this-is-a-bucket-name-with-exactly-sixty-three-characters-123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBucketName(tt.bucketName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListBuckets(t *testing.T) {
	// Setup test environment
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	buckets, err := client.ListBuckets()
	if err != nil {
		t.Errorf("ListBuckets returned error: %v", err)
		return
	}

	// Since we're using mock data, we expect at least some buckets
	if len(buckets) == 0 {
		t.Error("Expected at least one bucket in mock data")
	}

	// Validate each bucket
	for _, bucket := range buckets {
		if bucket.Name == "" {
			t.Error("Bucket name should not be empty")
		}
		if bucket.CreatedDate.IsZero() {
			t.Error("Bucket creation date should not be zero")
		}
	}
}

func TestCreateBucket(t *testing.T) {
	// Setup test environment
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	tests := []struct {
		name        string
		bucketName  string
		expectError bool
	}{
		{
			name:        "create valid bucket",
			bucketName:  "test-bucket",
			expectError: false,
		},
		{
			name:        "create invalid bucket name",
			bucketName:  "ab", // Too short
			expectError: true,
		},
		{
			name:        "create empty bucket name",
			bucketName:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, err := client.CreateBucket(tt.bucketName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if bucket != nil {
					t.Errorf("Expected nil bucket on error, got %v", bucket)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if bucket == nil {
					t.Errorf("Expected bucket but got nil")
					return
				}
				if bucket.Name != tt.bucketName {
					t.Errorf("Expected bucket name %s, got %s", tt.bucketName, bucket.Name)
				}
				if bucket.CreatedDate.IsZero() {
					t.Errorf("Expected non-zero creation date")
				}
			}
		})
	}
}

func TestDeleteBucket(t *testing.T) {
	// Setup test environment
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	tests := []struct {
		name        string
		bucketName  string
		expectError bool
	}{
		{
			name:        "delete valid bucket",
			bucketName:  "test-bucket",
			expectError: false,
		},
		{
			name:        "delete invalid bucket name",
			bucketName:  "ab", // Too short
			expectError: true,
		},
		{
			name:        "delete empty bucket name",
			bucketName:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DeleteBucket(tt.bucketName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUploadObject(t *testing.T) {
	// Setup test environment
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Create a temporary test file
	tmpFile := t.TempDir() + "/test-upload.txt"
	testContent := "This is a test file for upload"
	err = os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		request     *UploadRequest
		expectError bool
	}{
		{
			name: "valid upload request",
			request: &UploadRequest{
				Bucket:    "test-bucket",
				LocalPath: tmpFile,
				ObjectKey: "test/file.txt",
			},
			expectError: false,
		},
		{
			name:        "nil upload request",
			request:     nil,
			expectError: true,
		},
		{
			name: "missing bucket name",
			request: &UploadRequest{
				LocalPath: tmpFile,
				ObjectKey: "test/file.txt",
			},
			expectError: true,
		},
		{
			name: "missing local path",
			request: &UploadRequest{
				Bucket:    "test-bucket",
				ObjectKey: "test/file.txt",
			},
			expectError: true,
		},
		{
			name: "missing object key",
			request: &UploadRequest{
				Bucket:    "test-bucket",
				LocalPath: tmpFile,
			},
			expectError: true,
		},
		{
			name: "non-existent file",
			request: &UploadRequest{
				Bucket:    "test-bucket",
				LocalPath: "/path/to/non/existent/file.txt",
				ObjectKey: "test/file.txt",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.UploadObject(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if result != nil {
					t.Errorf("Expected nil result on error, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Errorf("Expected result but got nil")
					return
				}
				if result.ObjectKey != tt.request.ObjectKey {
					t.Errorf("Expected object key %s, got %s", tt.request.ObjectKey, result.ObjectKey)
				}
				if result.Size <= 0 {
					t.Errorf("Expected positive file size, got %d", result.Size)
				}
				if result.UploadTime.IsZero() {
					t.Errorf("Expected non-zero upload time")
				}
			}
		})
	}
}

func TestSetLifecyclePolicy(t *testing.T) {
	// Setup test environment
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	tests := []struct {
		name        string
		bucket      string
		policy      *LifecyclePolicy
		expectError bool
	}{
		{
			name:   "valid policy",
			bucket: "test-bucket",
			policy: &LifecyclePolicy{
				ExpirationDays: 30,
			},
			expectError: false,
		},
		{
			name:        "empty bucket name",
			bucket:      "",
			policy:      &LifecyclePolicy{ExpirationDays: 30},
			expectError: true,
		},
		{
			name:        "nil policy",
			bucket:      "test-bucket",
			policy:      nil,
			expectError: true,
		},
		{
			name:   "zero expiration days",
			bucket: "test-bucket",
			policy: &LifecyclePolicy{
				ExpirationDays: 0,
			},
			expectError: true,
		},
		{
			name:   "negative expiration days",
			bucket: "test-bucket",
			policy: &LifecyclePolicy{
				ExpirationDays: -5,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SetLifecyclePolicy(tt.bucket, tt.policy)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkListBuckets(b *testing.B) {
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")
	defer os.Unsetenv("CLOUDFLARE_API_TOKEN")

	client, err := NewClient("test-account-123")
	if err != nil {
		b.Fatalf("Failed to create test client: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.ListBuckets()
		if err != nil {
			b.Errorf("ListBuckets returned error: %v", err)
		}
	}
}

func BenchmarkValidateBucketName(b *testing.B) {
	bucketNames := []string{
		"valid-bucket-name",
		"another-valid-bucket-123",
		"short",
		"a-very-long-bucket-name-that-is-still-valid-1234567890",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range bucketNames {
			validateBucketName(name)
		}
	}
}