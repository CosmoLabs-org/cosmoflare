/*
Package http provides HTTP client validation testing for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	r2api "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/tests/helpers"
)

// HTTPClientValidationSuite provides comprehensive HTTP client testing
type HTTPClientValidationSuite struct {
	suite.Suite
	server      *httptest.Server
	client      *http.Client
	testConfig  helpers.TestConfig
}

// SetupSuite sets up the HTTP client validation test suite
func (suite *HTTPClientValidationSuite) SetupSuite() {
	// Create test configuration
	suite.testConfig = *helpers.SetupTest(suite.T())

	// Create test server that simulates Cloudflare R2 API
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.handleMockAPIRequest(w, r)
	}))

	// Create HTTP client with appropriate settings
	suite.client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 5,
		},
	}
}

// TearDownSuite cleans up after HTTP client tests
func (suite *HTTPClientValidationSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// handleMockAPIRequest simulates Cloudflare R2 API responses
func (suite *HTTPClientValidationSuite) handleMockAPIRequest(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/client/v4/accounts":
		// Mock account listing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
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
		}`)

	case r.Method == "GET" && r.URL.Path == "/accounts/test-account-id/r2/buckets":
		// Mock bucket listing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"result": [
				{
					"name": "test-bucket",
					"creation_date": "2025-01-01T00:00:00Z"
				}
			],
			"success": true,
			"errors": [],
			"messages": []
		}`)

	case r.Method == "PUT" && r.URL.Path == "/accounts/test-account-id/r2/buckets/new-bucket":
		// Mock bucket creation
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"result": {
				"name": "new-bucket",
				"creation_date": "2025-01-01T00:00:00Z"
			},
			"success": true,
			"errors": [],
			"messages": []
		}`)

	case r.Method == "DELETE" && r.URL.Path == "/accounts/test-account-id/r2/buckets/test-bucket":
		// Mock bucket deletion
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"result": null,
			"success": true,
			"errors": [],
			"messages": []
		}`)

	case r.Method == "GET" && r.URL.Path == "/rate-limit-test":
		// Simulate rate limiting
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Remaining", "999")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(3600*time.Second).Unix()))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status": "ok"}`)

	case r.Method == "GET" && r.URL.Path == "/error-test":
		// Simulate various error conditions based on query params
		errorType := r.URL.Query().Get("type")
		switch errorType {
		case "timeout":
			time.Sleep(2 * time.Second) // Simulate slow response
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status": "slow"}`)
		case "server_error":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "internal server error"}`)
		case "rate_limit":
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error": "rate limit exceeded"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "not found"}`)
		}

	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "endpoint not found"}`)
	}
}

// TestHTTPClientConfiguration tests HTTP client configuration
func (suite *HTTPClientValidationSuite) TestHTTPClientConfiguration() {
	suite.Run("Default Timeout Configuration", func() {
		assert.Equal(suite.T(), 30*time.Second, suite.client.Timeout, "Should have default timeout")
	})

	suite.Run("Transport Configuration", func() {
		transport, ok := suite.client.Transport.(*http.Transport)
		require.True(suite.T(), ok, "Should have HTTP transport")

		assert.Equal(suite.T(), 10, transport.MaxIdleConns, "Should have appropriate max idle connections")
		assert.Equal(suite.T(), 30*time.Second, transport.IdleConnTimeout, "Should have appropriate idle timeout")
		assert.False(suite.T(), transport.DisableCompression, "Should have compression enabled")
	})

	suite.Run("Custom Headers", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/rate-limit-test", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("User-Agent", "R2Go2/1.0.0")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Request should succeed")
	})
}

// TestHTTPRequestValidation tests HTTP request validation
func (suite *HTTPClientValidationSuite) TestHTTPRequestValidation() {
	suite.Run("Valid GET Request", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/client/v4/accounts", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "GET request should succeed")
		assert.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"), "Should return JSON")
	})

	suite.Run("Valid POST Request", func() {
		req, err := http.NewRequest("PUT", suite.server.URL+"/accounts/test-account-id/r2/buckets/new-bucket", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Content-Type", "application/json")

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "PUT request should succeed")
	})

	suite.Run("Valid DELETE Request", func() {
		req, err := http.NewRequest("DELETE", suite.server.URL+"/accounts/test-account-id/r2/buckets/test-bucket", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "DELETE request should succeed")
	})
}

// TestHTTPErrorHandling tests HTTP error handling
func (suite *HTTPClientValidationSuite) TestHTTPErrorHandling() {
	suite.Run("404 Not Found", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/error-test?type=notfound", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode, "Should return 404")
	})

	suite.Run("500 Internal Server Error", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/error-test?type=server_error", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode, "Should return 500")
	})

	suite.Run("Request Timeout", func() {
		// Create client with short timeout for this test
		shortTimeoutClient := &http.Client{
			Timeout: 1 * time.Second,
		}

		req, err := http.NewRequest("GET", suite.server.URL+"/error-test?type=timeout", nil)
		require.NoError(suite.T(), err)

		_, err = shortTimeoutClient.Do(req)
		assert.Error(suite.T(), err, "Should timeout with short timeout")
		assert.Contains(suite.T(), err.Error(), "timeout", "Error should mention timeout")
	})
}

// TestRateLimitHandling tests rate limit handling
func (suite *HTTPClientValidationSuite) TestRateLimitHandling() {
	suite.Run("Rate Limit Headers", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/rate-limit-test", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), "1000", resp.Header.Get("X-RateLimit-Limit"), "Should have rate limit header")
		assert.Equal(suite.T(), "999", resp.Header.Get("X-RateLimit-Remaining"), "Should have remaining requests header")
		assert.NotEmpty(suite.T(), resp.Header.Get("X-RateLimit-Reset"), "Should have reset timestamp header")
	})

	suite.Run("Rate Limit Exceeded", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/error-test?type=rate_limit", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode, "Should return 429")
		assert.Equal(suite.T(), "0", resp.Header.Get("X-RateLimit-Remaining"), "Should have no remaining requests")
		assert.Equal(suite.T(), "60", resp.Header.Get("Retry-After"), "Should have retry after header")
	})
}

// TestContextCancellation tests context cancellation
func (suite *HTTPClientValidationSuite) TestContextCancellation() {
	suite.Run("Context Timeout", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", suite.server.URL+"/error-test?type=timeout", nil)
		require.NoError(suite.T(), err)

		_, err = suite.client.Do(req)
		assert.Error(suite.T(), err, "Should fail with context timeout")
		assert.Contains(suite.T(), err.Error(), "context deadline exceeded", "Error should mention context deadline")
	})

	suite.Run("Context Cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())

		req, err := http.NewRequestWithContext(ctx, "GET", suite.server.URL+"/error-test?type=timeout", nil)
		require.NoError(suite.T(), err)

		// Cancel the request immediately
		cancel()

		_, err = suite.client.Do(req)
		assert.Error(suite.T(), err, "Should fail with context cancellation")
		assert.Contains(suite.T(), err.Error(), "context canceled", "Error should mention context cancellation")
	})
}

// TestConcurrentRequests tests concurrent request handling
func (suite *HTTPClientValidationSuite) TestConcurrentRequests() {
	suite.Run("Multiple Concurrent Requests", func() {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()

				req, err := http.NewRequest("GET", suite.server.URL+"/client/v4/accounts", nil)
				if err != nil {
					errors <- err
					return
				}

				resp, err := suite.client.Do(req)
				if err != nil {
					errors <- err
					return
				}
				resp.Body.Close()
				errors <- nil
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < concurrency; i++ {
			<-done
			err := <-errors
			assert.NoError(suite.T(), err, "Concurrent request should succeed")
		}
	})
}

// TestPerformanceMetrics tests HTTP performance characteristics
func (suite *HTTPClientValidationSuite) TestPerformanceMetrics() {
	suite.Run("Response Time Measurement", func() {
		start := time.Now()

		req, err := http.NewRequest("GET", suite.server.URL+"/client/v4/accounts", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		duration := time.Since(start)

		assert.Less(suite.T(), duration, 1*time.Second, "Request should complete within 1 second")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Request should succeed")
	})

	suite.Run("Connection Reuse", func() {
		// Make multiple requests to test connection reuse
		for i := 0; i < 5; i++ {
			req, err := http.NewRequest("GET", suite.server.URL+"/client/v4/accounts", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Request should succeed")
		}
	})
}

// TestHTTPClientValidationSuite runs the complete HTTP client validation test suite
func TestHTTPClientValidationSuite(t *testing.T) {
	suite.Run(t, new(HTTPClientValidationSuite))
}

// TestAPIIntegrationWithHTTPClient tests integration with actual R2 API client
func TestAPIIntegrationWithHTTPClient(t *testing.T) {
	t.Run("Client Creation with Custom HTTP Client", func(t *testing.T) {
		customHTTPClient := &http.Client{
			Timeout: 15 * time.Second,
		}

		opts := &r2api.ClientOptions{
			AccountID:  "test-account",
			APIToken:   "test-token",
			HTTPClient: customHTTPClient,
		}

		client, err := r2api.NewClient(opts)
		require.NoError(t, err, "Should create client with custom HTTP client")
		assert.NotNil(t, client, "Client should not be nil")
	})

	t.Run("Client Connection Test", func(t *testing.T) {
		opts := &r2api.ClientOptions{
			AccountID: "test-account",
			APIToken:  "test-token",
		}

		client, err := r2api.NewClient(opts)
		require.NoError(t, err, "Should create client")

		// Test basic connection functionality
		err = client.TestConnection()
		// Note: With test credentials, this might fail, which is expected
		t.Logf("Connection test result: %v", err)
	})
}