/*
Package security provides comprehensive input validation and injection prevention testing

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

// InputValidationTestSuite provides comprehensive input validation testing
type InputValidationTestSuite struct {
	suite.Suite
	server     *httptest.Server
	testConfig helpers.TestConfig
}

// SetupSuite sets up the input validation test suite
func (suite *InputValidationTestSuite) SetupSuite() {
	// Create test configuration
	suite.testConfig = *helpers.SetupTest(suite.T())

	// Create test server that simulates input validation scenarios
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.handleInputValidationRequest(w, r)
	}))
}

// TearDownSuite cleans up after input validation tests
func (suite *InputValidationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// handleInputValidationRequest simulates various input validation scenarios
func (suite *InputValidationTestSuite) handleInputValidationRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()

	switch path {
	case "/validate/bucket-name":
		suite.handleBucketNameValidation(w, query.Get("name"))
	case "/validate/object-key":
		suite.handleObjectKeyValidation(w, query.Get("key"))
	case "/validate/metadata":
		suite.handleMetadataValidation(w, r)
	case "/upload/test":
		suite.handleFileUploadTest(w, r)
	case "/search/test":
		suite.handleSearchTest(w, r)
	case "/api/test":
		suite.handleAPITest(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "endpoint not found"}`)
	}
}

// handleBucketNameValidation validates bucket names
func (suite *InputValidationTestSuite) handleBucketNameValidation(w http.ResponseWriter, bucketName string) {
	// Bucket name validation rules (Cloudflare R2)
	if bucketName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name cannot be empty"}`)
		return
	}

	if len(bucketName) < 3 || len(bucketName) > 63 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name must be between 3 and 63 characters"}`)
		return
	}

	// Check for invalid patterns (basic validation already done above)

	// Must be lowercase
	if strings.ToLower(bucketName) != bucketName {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name must be lowercase"}`)
		return
	}

	// Check for invalid characters (only lowercase letters, numbers, hyphens, and periods allowed)
	for _, ch := range bucketName {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.') {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "bucket name contains invalid character: %c"}`, ch)
			return
		}
	}

	// Cannot start or end with period or hyphen
	if strings.HasPrefix(bucketName, ".") || strings.HasSuffix(bucketName, ".") ||
		strings.HasPrefix(bucketName, "-") || strings.HasSuffix(bucketName, "-") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name cannot start or end with period or hyphen"}`)
		return
	}

	// Check for consecutive periods
	if strings.Contains(bucketName, "..") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name cannot contain consecutive periods"}`)
		return
	}

	// Check for IP address format
	if isIPAddressFormat(bucketName) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bucket name cannot be formatted as IP address"}`)
		return
	}

	// Check for reserved prefixes
	reservedPrefixes := []string{"xn--", "sthree-"}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(bucketName, prefix) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "bucket name has reserved prefix"}`)
			return
		}
	}

	// If all validations pass
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"valid": true, "bucket_name": "%s"}`, bucketName)
}

// handleObjectKeyValidation validates object keys
func (suite *InputValidationTestSuite) handleObjectKeyValidation(w http.ResponseWriter, objectKey string) {
	if objectKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "object key cannot be empty"}`)
		return
	}

	if len(objectKey) > 1024 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "object key too long (max 1024 characters)"}`)
		return
	}

	// Check for dangerous characters
	dangerousChars := []string{"\x00", "\r", "\n"}
	for _, char := range dangerousChars {
		if strings.Contains(objectKey, char) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "object key contains invalid characters"}`)
			return
		}
	}

	// Check for path traversal attempts
	if strings.Contains(objectKey, "../") || strings.Contains(objectKey, "..\\") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "object key contains path traversal attempt"}`)
		return
	}

	// Check for absolute path attempts
	if strings.HasPrefix(objectKey, "/") || strings.HasPrefix(objectKey, "\\") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "object key cannot be absolute path"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"valid": true, "object_key": "%s"}`, sanitizeJSONString(objectKey))
}

// handleMetadataValidation validates metadata
func (suite *InputValidationTestSuite) handleMetadataValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, `{"error": "method not allowed"}`)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "failed to read request body"}`)
		return
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(body, &metadata); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "invalid JSON format"}`)
		return
	}

	// Validate metadata size
	if len(body) > 2048 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "metadata too large (max 2KB)"}`)
		return
	}

	// Check for dangerous content in decoded metadata values
	for _, val := range metadata {
		if strVal, ok := val.(string); ok {
			if containsInjectionPatterns(strVal) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `{"error": "metadata contains potentially dangerous content"}`)
				return
			}
		}
	}

	// Validate each metadata key
	for key := range metadata {
		if len(key) > 256 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "metadata key too long"}`)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"valid": true, "message": "metadata is valid"}`)
}

// handleFileUploadTest tests file upload security
func (suite *InputValidationTestSuite) handleFileUploadTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, `{"error": "method not allowed"}`)
		return
	}

	// Check content length
	if r.ContentLength > 100*1024*1024 { // 100MB limit
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		fmt.Fprint(w, `{"error": "file too large"}`)
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")
	dangerousTypes := []string{
		"application/octet-stream",
		"application/x-executable",
		"application/x-msdownload",
	}

	for _, dangerousType := range dangerousTypes {
		if strings.Contains(contentType, dangerousType) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprint(w, `{"error": "file type not allowed"}`)
			return
		}
	}

	// Simulate file processing
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"uploaded": true, "message": "file upload test successful"}`)
}

// handleSearchTest tests search input validation
func (suite *InputValidationTestSuite) handleSearchTest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "search query cannot be empty"}`)
		return
	}

	if len(query) > 1000 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "search query too long"}`)
		return
	}

	// Check for injection patterns (SQL, XSS, path traversal, command injection)
	if containsInjectionPatterns(query) || containsSQLInjectionPatterns(query) || containsXSSPatterns(query) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "search query contains potentially dangerous content"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"search_results": [], "query": "%s"}`, sanitizeJSONString(query))
}

// handleAPITest tests API input validation
func (suite *InputValidationTestSuite) handleAPITest(w http.ResponseWriter, r *http.Request) {
	// Validate request headers
	for key, values := range r.Header {
		if len(key) > 100 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "header name too long"}`)
			return
		}

		for _, value := range values {
			if len(value) > 8192 { // 8KB per header value
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `{"error": "header value too long"}`)
				return
			}
		}
	}

	// Validate URL length
	if len(r.URL.String()) > 8192 {
		w.WriteHeader(http.StatusRequestURITooLong)
		fmt.Fprint(w, `{"error": "URL too long"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message": "API input validation successful"}`)
}

// TestBucketNameValidation tests bucket name input validation
func (suite *InputValidationTestSuite) TestBucketNameValidation() {
	suite.Run("Valid Bucket Names", func() {
		validNames := []string{
			"my-bucket",
			"test-bucket-123",
			"a.b.c",
			"my-test-bucket",
			"bucket123",
			"the-quick-brown-fox",
		}

		for _, name := range validNames {
			resp, err := http.Get(fmt.Sprintf("%s/validate/bucket-name?name=%s", suite.server.URL, url.QueryEscape(name)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid bucket name should be accepted: %s", name)
		}
	})

	suite.Run("Invalid Bucket Names", func() {
		invalidNames := []string{
			"",                          // Empty
			"ab",                        // Too short
			strings.Repeat("a", 64),     // Too long
			"My-Bucket",                 // Uppercase
			"my_bucket",                 // Underscore
			"my..bucket",                // Consecutive periods
			".mybucket",                 // Starts with period
			"mybucket.",                 // Ends with period
			"192.168.1.1",              // IP format
			"xn--invalid",               // Reserved prefix
			"sthree-test",               // Reserved prefix
			"my bucket",                 // Space
			"my/bucket",                 // Slash
		}

		for _, name := range invalidNames {
			resp, err := http.Get(fmt.Sprintf("%s/validate/bucket-name?name=%s", suite.server.URL, url.QueryEscape(name)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Invalid bucket name should be rejected: %s", name)
		}
	})
}

// TestObjectKeyValidation tests object key input validation
func (suite *InputValidationTestSuite) TestObjectKeyValidation() {
	suite.Run("Valid Object Keys", func() {
		validKeys := []string{
			"file.txt",
			"path/to/file.txt",
			"documents/2025/report.pdf",
			"user-uploads/image.jpg",
			"测试文件.txt",                // Unicode
			"file-with-dashes.txt",
			"file_with_underscores.txt",
			"file123.txt",
		}

		for _, key := range validKeys {
			resp, err := http.Get(fmt.Sprintf("%s/validate/object-key?key=%s", suite.server.URL, url.QueryEscape(key)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid object key should be accepted: %s", key)
		}
	})

	suite.Run("Invalid Object Keys", func() {
		invalidKeys := []string{
			"",                          // Empty
			strings.Repeat("a", 1025),  // Too long
			"../etc/passwd",             // Path traversal
			"..\\windows\\system32",     // Windows path traversal
			"/absolute/path",            // Absolute path
			"\\absolute\\windows\\path", // Windows absolute path
			"file\x00.txt",              // Null byte
			"file\r.txt",                // Carriage return
			"file\n.txt",                // Newline
		}

		for _, key := range invalidKeys {
			resp, err := http.Get(fmt.Sprintf("%s/validate/object-key?key=%s", suite.server.URL, url.QueryEscape(key)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Invalid object key should be rejected: %q", key)
		}
	})
}

// TestMetadataValidation tests metadata input validation
func (suite *InputValidationTestSuite) TestMetadataValidation() {
	suite.Run("Valid Metadata", func() {
		validMetadata := map[string]interface{}{
			"content-type":  "image/jpeg",
			"user-id":       "12345",
			"upload-date":   "2025-01-15",
			"custom-field":  "custom-value",
			"unicode-test":  "测试值",
		}

		metadataJSON, err := json.Marshal(validMetadata)
		require.NoError(suite.T(), err)

		resp, err := http.Post(suite.server.URL+"/validate/metadata", "application/json", bytes.NewReader(metadataJSON))
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid metadata should be accepted")
	})

	suite.Run("Invalid Metadata", func() {
		invalidMetadataCases := []struct {
			name        string
			metadata    map[string]interface{}
			expectError bool
		}{
			{
				name: "Too Large Metadata",
				metadata: map[string]interface{}{
					"large-field": strings.Repeat("a", 2049),
				},
				expectError: true,
			},
			{
				name: "SQL Injection",
				metadata: map[string]interface{}{
					"query": "SELECT * FROM users WHERE id = 1; DROP TABLE users; --",
				},
				expectError: true,
			},
			{
				name: "XSS Attempt",
				metadata: map[string]interface{}{
					"html": "<script>alert('xss')</script>",
				},
				expectError: true,
			},
			{
				name: "Long Key Name",
				metadata: map[string]interface{}{
					strings.Repeat("a", 257): "value",
				},
				expectError: true,
			},
		}

		for _, tc := range invalidMetadataCases {
			metadataJSON, err := json.Marshal(tc.metadata)
			require.NoError(suite.T(), err)

			resp, err := http.Post(suite.server.URL+"/validate/metadata", "application/json", bytes.NewReader(metadataJSON))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			if tc.expectError {
				assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Invalid metadata should be rejected: %s", tc.name)
			} else {
				assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid metadata should be accepted: %s", tc.name)
			}
		}
	})
}

// TestSearchInputValidation tests search query validation
func (suite *InputValidationTestSuite) TestSearchInputValidation() {
	suite.Run("Valid Search Queries", func() {
		validQueries := []string{
			"test query",
			"file.txt",
			"2025-01-15",
			"user-uploads",
			"document type:pdf",
			"测试查询", // Unicode
			"file-name_123",
		}

		for _, query := range validQueries {
			resp, err := http.Get(fmt.Sprintf("%s/search/test?q=%s", suite.server.URL, url.QueryEscape(query)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid search query should be accepted: %s", query)
		}
	})

	suite.Run("Invalid Search Queries", func() {
		invalidQueries := []string{
			"",                                      // Empty
			strings.Repeat("a", 1001),             // Too long
			"'; DROP TABLE users; --",             // SQL injection
			"<script>alert('xss')</script>",       // XSS
			"../../etc/passwd",                    // Path traversal
		}

		for _, query := range invalidQueries {
			resp, err := http.Get(fmt.Sprintf("%s/search/test?q=%s", suite.server.URL, url.QueryEscape(query)))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Invalid search query should be rejected: %s", query)
		}
	})
}

// TestFileUploadSecurity tests file upload security
func (suite *InputValidationTestSuite) TestFileUploadSecurity() {
	suite.Run("Valid File Upload", func() {
		// Simulate a small text file upload
		fileContent := "This is a test file for upload"
		req, err := http.NewRequest("POST", suite.server.URL+"/upload/test", strings.NewReader(fileContent))
		require.NoError(suite.T(), err)

		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(fileContent)))

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Valid file upload should be accepted")
	})

	suite.Run("Invalid File Upload", func() {
		// Simulate an executable file upload
		req, err := http.NewRequest("POST", suite.server.URL+"/upload/test", strings.NewReader("fake executable content"))
		require.NoError(suite.T(), err)

		req.Header.Set("Content-Type", "application/x-executable")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnsupportedMediaType, resp.StatusCode, "Executable file should be rejected")
	})
}

// TestInputValidationTestSuite runs the complete input validation test suite
func TestInputValidationTestSuite(t *testing.T) {
	suite.Run(t, new(InputValidationTestSuite))
}

// TestSecurityValidationFunctions tests security validation helper functions
func TestSecurityValidationFunctions(t *testing.T) {
	t.Run("Injection Pattern Detection", func(t *testing.T) {
		injectionStrings := []string{
			"'; DROP TABLE users; --",
			"1' OR '1'='1",
			"<script>alert('xss')</script>",
			"../../../etc/passwd",
			"cmd.exe /c dir",
			"$(whoami)",
			"`id`",
		}

		for _, injection := range injectionStrings {
			assert.True(t, containsInjectionPatterns(injection), "Should detect injection pattern: %s", injection)
		}

		safeStrings := []string{
			"normal text",
			"file.txt",
			"document-2025",
			"user_uploads",
			"测试文档",
		}

		for _, safe := range safeStrings {
			assert.False(t, containsInjectionPatterns(safe), "Should not detect injection in safe string: %s", safe)
		}
	})

	t.Run("SQL Injection Detection", func(t *testing.T) {
		sqlInjectionStrings := []string{
			"SELECT * FROM users",
			"'; DROP TABLE users; --",
			"1' UNION SELECT * FROM passwords",
			"INSERT INTO users VALUES",
			"UPDATE users SET",
			"DELETE FROM users",
		}

		for _, injection := range sqlInjectionStrings {
			assert.True(t, containsSQLInjectionPatterns(injection), "Should detect SQL injection: %s", injection)
		}
	})

	t.Run("XSS Detection", func(t *testing.T) {
		xssStrings := []string{
			"<script>alert('xss')</script>",
			"<img src=x onerror=alert('xss')>",
			"javascript:alert('xss')",
			"<svg onload=alert('xss')>",
			"<iframe src=javascript:alert('xss')>",
		}

		for _, xss := range xssStrings {
			assert.True(t, containsXSSPatterns(xss), "Should detect XSS: %s", xss)
		}
	})
}

// Helper functions for security testing

// isIPAddressFormat checks if string looks like IP address
func isIPAddressFormat(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}

		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}

// containsInjectionPatterns checks for common injection patterns
func containsInjectionPatterns(s string) bool {
	patterns := []string{
		"<script",
		"</script>",
		"javascript:",
		"onerror=",
		"onload=",
		"' or ",
		"' and ",
		"1'='1",
		" DROP ",
		" DELETE ",
		" INSERT ",
		" UPDATE ",
		" SELECT ",
		" UNION ",
		"../",
		"..\\",
		"cmd.exe",
		"/bin/sh",
		"powershell",
		"$(",
		"`",
	}

	lower := strings.ToLower(s)
	for _, pattern := range patterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// containsSQLInjectionPatterns checks for SQL injection patterns
func containsSQLInjectionPatterns(s string) bool {
	patterns := []string{
		"' or ",
		"' and ",
		" or '",
		" and '",
		"1'='1",
		"1=1",
		" drop ",
		" delete ",
		" insert ",
		" update ",
		" select ",
		" union ",
		" --",
		" /*",
		" */",
		"xp_",
		"sp_",
	}

	// Also detect SQL keywords at start of string
	startPatterns := []string{
		"select ", "insert ", "update ", "delete ", "drop ", "union ",
	}

	lower := strings.ToLower(s)
	for _, pattern := range startPatterns {
		if strings.HasPrefix(lower, pattern) {
			return true
		}
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// containsXSSPatterns checks for XSS patterns
func containsXSSPatterns(s string) bool {
	patterns := []string{
		"<script",
		"</script>",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onclick=",
		"onmouseover=",
		"<iframe",
		"<object",
		"<embed",
		"<form",
		"<input",
		"alert(",
		"confirm(",
		"prompt(",
	}

	lower := strings.ToLower(s)
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// sanitizeJSONString escapes special JSON characters
func sanitizeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\b", "\\b")
	s = strings.ReplaceAll(s, "\f", "\\f")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}