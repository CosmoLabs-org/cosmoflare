package fixtures

import (
	"fmt"
	"math/rand"
	"time"
)

// Sample configuration data for testing
var SampleConfig = map[string]interface{}{
	"version":         "1.0.0",
	"current_profile": "default",
	"profiles": map[string]interface{}{
		"default": map[string]interface{}{
			"name":        "Default Profile",
			"account_id":  "1234567890abcdef1234567890abcdef",
			"access_key":  "AKIAIOSFODNN7EXAMPLE",
			"secret_key":  "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"bucket":      "my-test-bucket",
			"endpoint":    "https://abc123.r2.cloudflarestorage.com",
			"region":      "auto",
			"created_at":  time.Now().Format(time.RFC3339),
		},
		"secondary": map[string]interface{}{
			"name":        "Secondary Profile",
			"account_id":  "fedcba0987654321fedcba0987654321",
			"access_key":  "AKIAI44QH8DHBEXAMPLE",
			"secret_key":  "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY",
			"bucket":      "my-other-bucket",
			"endpoint":    "https://def456.r2.cloudflarestorage.com",
			"region":      "auto",
			"created_at":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
	},
	"settings": map[string]interface{}{
		"theme":              "default",
		"tutorial_completed": false,
		"accessibility": map[string]interface{}{
			"screen_reader": false,
			"high_contrast": false,
			"large_text":    false,
		},
	},
}

// Sample themes for testing
var SampleThemes = map[string]interface{}{
	"default": map[string]interface{}{
		"name":        "Default",
		"description": "Default R2Go2 theme",
		"colors": map[string]interface{}{
			"primary":   "#007bff",
			"secondary": "#6c757d",
			"success":   "#28a745",
			"danger":    "#dc3545",
			"warning":   "#ffc107",
			"info":      "#17a2b8",
			"light":     "#f8f9fa",
			"dark":      "#343a40",
		},
		"background": map[string]interface{}{
			"main":      "#ffffff",
			"secondary": "#f8f9fa",
			"accent":    "#e9ecef",
		},
		"text": map[string]interface{}{
			"primary":   "#212529",
			"secondary": "#6c757d",
			"muted":     "#6c757d",
		},
	},
	"dark": map[string]interface{}{
		"name":        "Dark",
		"description": "Dark theme for low-light environments",
		"colors": map[string]interface{}{
			"primary":   "#0d6efd",
			"secondary": "#6c757d",
			"success":   "#198754",
			"danger":    "#dc3545",
			"warning":   "#ffc107",
			"info":      "#0dcaf0",
			"light":     "#f8f9fa",
			"dark":      "#212529",
		},
		"background": map[string]interface{}{
			"main":      "#212529",
			"secondary": "#343a40",
			"accent":    "#495057",
		},
		"text": map[string]interface{}{
			"primary":   "#ffffff",
			"secondary": "#adb5bd",
			"muted":     "#6c757d",
		},
	},
}

// Sample API responses
var SampleAPIResponses = map[string]interface{}{
	"accounts": []map[string]interface{}{
		{
			"id":     "1234567890abcdef1234567890abcdef",
			"name":   "Test Account",
			"status": "active",
		},
		{
			"id":     "fedcba0987654321fedcba0987654321",
			"name":   "Another Account",
			"status": "active",
		},
	},
	"buckets": []map[string]interface{}{
		{
			"name":          "test-bucket",
			"creation_date": "2024-01-01T00:00:00Z",
			"location": map[string]interface{}{
				"type": "region",
				"name": "auto",
			},
		},
		{
			"name":          "another-bucket",
			"creation_date": "2024-01-02T00:00:00Z",
			"location": map[string]interface{}{
				"type": "region",
				"name": "auto",
			},
		},
	},
	"objects": []map[string]interface{}{
		{
			"key":           "test-file.txt",
			"size":          1024,
			"last_modified": "2024-01-01T12:00:00Z",
			"etag":          "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			"key":           "folder/nested-file.txt",
			"size":          2048,
			"last_modified": "2024-01-01T13:00:00Z",
			"etag":          "098f6bcd4621d373cade4e832627b4f6",
		},
	},
}

// GenerateTestFile creates test file content of specified size
func GenerateTestFile(size int) []byte {
	content := make([]byte, size)
	rand.Seed(time.Now().UnixNano())
	rand.Read(content)
	return content
}

// GenerateTestFiles creates multiple test files with different sizes
func GenerateTestFiles(sizes []int) map[string][]byte {
	files := make(map[string][]byte)
	for i, size := range sizes {
		filename := fmt.Sprintf("test-file-%d.txt", i+1)
		files[filename] = GenerateTestFile(size)
	}
	return files
}

// GenerateTestDirectoryName creates a unique test directory name
func GenerateTestDirectoryName() string {
	return fmt.Sprintf("r2go2-test-%d", time.Now().Unix())
}

// GetSampleConfig returns a copy of the sample configuration
func GetSampleConfig() map[string]interface{} {
	config := make(map[string]interface{})
	for k, v := range SampleConfig {
		config[k] = v
	}
	return config
}

// GetSampleThemes returns a copy of the sample themes
func GetSampleThemes() map[string]interface{} {
	themes := make(map[string]interface{})
	for k, v := range SampleThemes {
		themes[k] = v
	}
	return themes
}

// GetSampleAPIResponse returns a sample API response by type
func GetSampleAPIResponse(responseType string) interface{} {
	if response, exists := SampleAPIResponses[responseType]; exists {
		return response
	}
	return nil
}

// Common test file sizes for performance testing
var TestFileSizes = []int{
	1024,        // 1KB
	10240,       // 10KB
	102400,      // 100KB
	1048576,     // 1MB
	10485760,    // 10MB
	104857600,   // 100MB
}

// Common test file names
var TestFileNames = []string{
	"small-file.txt",
	"medium-file.txt",
	"large-file.txt",
	"file-with-spaces.txt",
	"file-with-特殊字符.txt",
	"very-long-filename-that-tests-path-handling-capabilities.txt",
	"file.ext",
	"file.tar.gz",
	"file.with.many.dots.txt",
	"UPPERCASE.TXT",
	"lowercase.txt",
	"MixedCase.Txt",
}

// Common bucket names for testing
var TestBucketNames = []string{
	"test-bucket",
	"test-bucket-123",
	"my-test-bucket",
	"r2go2-test",
	"bucket-with-dashes",
	"bucket_with_underscores",
	"bucket123",
	"a", // minimum length
	"bucket-name-that-is-exactly-sixty-three-characters-long-123", // maximum length
}

// Error scenarios for testing
var ErrorScenarios = map[string]map[string]interface{}{
	"invalid_credentials": {
		"status_code": 401,
		"error":       "Invalid credentials",
		"message":     "The provided credentials are invalid",
	},
	"bucket_not_found": {
		"status_code": 404,
		"error":       "Bucket not found",
		"message":     "The specified bucket does not exist",
	},
	"access_denied": {
		"status_code": 403,
		"error":       "Access denied",
		"message":     "You do not have permission to perform this action",
	},
	"rate_limited": {
		"status_code": 429,
		"error":       "Rate limit exceeded",
		"message":     "Too many requests, please try again later",
	},
	"server_error": {
		"status_code": 500,
		"error":       "Internal server error",
		"message":     "An unexpected error occurred",
	},
}