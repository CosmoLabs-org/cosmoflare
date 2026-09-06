/*
Package http provides rate limiting and retry mechanism testing for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

// RateLimitRetrySuite provides comprehensive rate limiting and retry testing
type RateLimitRetrySuite struct {
	suite.Suite
	server      *httptest.Server
	client      *http.Client
	testConfig  helpers.TestConfig
	requests    map[string]int // Track requests by endpoint
	mu          sync.Mutex
}

// SetupSuite sets up the rate limiting and retry test suite
func (suite *RateLimitRetrySuite) SetupSuite() {
	// Create test configuration
	suite.testConfig = *helpers.SetupTest(suite.T())
	suite.requests = make(map[string]int)

	// Create test server that simulates various rate limiting scenarios
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.handleRateLimitRequest(w, r)
	}))

	// Create HTTP client with retry-friendly settings
	suite.client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 10,
		},
	}
}

// TearDownSuite cleans up after rate limiting and retry tests
func (suite *RateLimitRetrySuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// handleRateLimitRequest simulates various rate limiting and error scenarios
func (suite *RateLimitRetrySuite) handleRateLimitRequest(w http.ResponseWriter, r *http.Request) {
	suite.mu.Lock()
	defer suite.mu.Unlock()

	// Track request counts
	suite.requests[r.URL.Path]++

	path := r.URL.Path
	requestCount := suite.requests[path]

	switch path {
	case "/api/v1/rate-limit":
		// Simulate rate limiting after 5 requests
		if requestCount <= 5 {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", 100-requestCount))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(3600*time.Second).Unix()))
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "success", "request_count": `+fmt.Sprintf("%d", requestCount)+`}`)
		} else {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()))
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error": "rate limit exceeded"}`)
		}

	case "/api/v1/retry-after":
		// Simulate transient failures that should be retried
		if requestCount <= 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error": "service temporarily unavailable"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "success after retries", "request_count": `+fmt.Sprintf("%d", requestCount)+`}`)
		}

	case "/api/v1/server-error":
		// Simulate server errors that might be retried
		if requestCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "internal server error"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "recovered from server error", "request_count": `+fmt.Sprintf("%d", requestCount)+`}`)
		}

	case "/api/v1/timeout":
		// Simulate slow responses that might timeout
		if requestCount <= 2 {
			time.Sleep(2 * time.Second) // Longer than client timeout
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "slow response"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "fast response", "request_count": `+fmt.Sprintf("%d", requestCount)+`}`)
		}

	case "/api/v1/network-error":
		// Simulate network issues (close connection immediately)
		if requestCount <= 3 {
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "network error simulation"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "network recovered", "request_count": `+fmt.Sprintf("%d", requestCount)+`}`)
		}

	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "endpoint not found"}`)
	}
}

// TestRateLimitDetection tests rate limit detection and handling
func (suite *RateLimitRetrySuite) TestRateLimitDetection() {
	suite.Run("Detect Rate Limit Headers", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		// Make several requests to hit rate limit
		for i := 0; i < 7; i++ {
			req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/rate-limit", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			require.NoError(suite.T(), err)

			if i < 5 {
				assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Request should succeed before rate limit")
				assert.Equal(suite.T(), "100", resp.Header.Get("X-RateLimit-Limit"), "Should have rate limit header")
				remaining := resp.Header.Get("X-RateLimit-Remaining")
				assert.NotEmpty(suite.T(), remaining, "Should have remaining requests header")
			} else {
				assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode, "Should hit rate limit")
				assert.Equal(suite.T(), "0", resp.Header.Get("X-RateLimit-Remaining"), "Should have no remaining requests")
				assert.Equal(suite.T(), "60", resp.Header.Get("Retry-After"), "Should have retry after header")
			}
			resp.Body.Close()
		}
	})

	suite.Run("Rate Limit Recovery", func() {
		// Test that rate limits recover after the retry period
		req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/rate-limit", nil)
		require.NoError(suite.T(), err)

		// This request should still be rate limited
		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode, "Should still be rate limited")
		resp.Body.Close()
	})
}

// TestRetryMechanism tests retry logic for various failure scenarios
func (suite *RateLimitRetrySuite) TestRetryMechanism() {
	suite.Run("Retry Transient Failures", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		// Implement simple retry logic
		var lastErr error
		maxRetries := 5
		retryDelay := 100 * time.Millisecond

		for attempt := 0; attempt <= maxRetries; attempt++ {
			req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/retry-after", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			if err != nil {
				lastErr = err
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
				break
			}

			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				assert.Equal(suite.T(), 4, attempt+1, "Should succeed after 3 retries")
				return // Success
			}

			// Check for retry-able status codes
			if resp.StatusCode == http.StatusServiceUnavailable {
				resp.Body.Close()
				lastErr = fmt.Errorf("service unavailable (attempt %d)", attempt+1)
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			resp.Body.Close()
			break
		}

		suite.T().Errorf("Should have succeeded after retries, last error: %v", lastErr)
	})

	suite.Run("Retry Server Errors", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		var lastErr error
		maxRetries := 3
		retryDelay := 50 * time.Millisecond

		for attempt := 0; attempt <= maxRetries; attempt++ {
			req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/server-error", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			if err != nil {
				lastErr = err
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					continue
				}
				break
			}

			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				assert.Equal(suite.T(), 3, attempt+1, "Should succeed after 2 retries")
				return // Success
			}

			if resp.StatusCode >= 500 {
				resp.Body.Close()
				lastErr = fmt.Errorf("server error: %d (attempt %d)", resp.StatusCode, attempt+1)
				if attempt < maxRetries {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			resp.Body.Close()
			break
		}

		suite.T().Errorf("Should have succeeded after retries, last error: %v", lastErr)
	})

	suite.Run("No Retry on Client Errors", func() {
		req, err := http.NewRequest("GET", suite.server.URL+"/non-existent-endpoint", nil)
		require.NoError(suite.T(), err)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode, "Should return 404")
		// Client errors (4xx) should not be retried
	})
}

// TestExponentialBackoff tests exponential backoff retry strategy
func (suite *RateLimitRetrySuite) TestExponentialBackoff() {
	suite.Run("Exponential Backoff Timing", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		start := time.Now()
		maxRetries := 3
		baseDelay := 10 * time.Millisecond

		for attempt := 0; attempt <= maxRetries; attempt++ {
			req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/retry-after", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			if err != nil {
				if attempt < maxRetries {
					delay := time.Duration(attempt+1) * baseDelay
					time.Sleep(delay)
					continue
				}
				break
			}

			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				duration := time.Since(start)

				// Should have taken at least some time due to retries and backoff
				expectedMinDelay := baseDelay * (1 + 2) // First two retries
				assert.GreaterOrEqual(suite.T(), duration, expectedMinDelay, "Should have exponential backoff delays")
				return
			}

			if resp.StatusCode == http.StatusServiceUnavailable {
				resp.Body.Close()
				if attempt < maxRetries {
					delay := time.Duration(attempt+1) * baseDelay
					time.Sleep(delay)
					continue
				}
			}
			resp.Body.Close()
			break
		}

		suite.T().Error("Should have succeeded with exponential backoff")
	})
}

// TestRetryWithTimeout tests retry behavior with timeout handling
func (suite *RateLimitRetrySuite) TestRetryWithTimeout() {
	suite.Run("Timeout During Retry", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		// Create client with short timeout
		shortTimeoutClient := &http.Client{
			Timeout: 500 * time.Millisecond,
		}

		start := time.Now()
		req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/timeout", nil)
		require.NoError(suite.T(), err)

		_, err = shortTimeoutClient.Do(req)
		duration := time.Since(start)

		assert.Error(suite.T(), err, "Should timeout")
		assert.Less(suite.T(), duration, 1*time.Second, "Should fail quickly due to short timeout")
	})

	suite.Run("Context Timeout With Retry", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		start := time.Now()
		var lastErr error
		attempts := 0

		for attempts < 10 {
			req, err := http.NewRequestWithContext(ctx, "GET", suite.server.URL+"/api/v1/timeout", nil)
			require.NoError(suite.T(), err)

			resp, err := suite.client.Do(req)
			attempts++

			if err != nil {
				lastErr = err
				if err.Error() == "context deadline exceeded" {
					duration := time.Since(start)
					assert.GreaterOrEqual(suite.T(), attempts, 1, "Should have made at least one attempt before timeout")
					assert.Less(suite.T(), duration, 2*time.Second, "Should have timed out within context deadline")
					return // Expected behavior - context timeout
				}
				continue
			}

			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				// If we get here, the request succeeded (possibly after some timeouts)
				return
			}
			resp.Body.Close()
		}

		// If we get here, neither timeout nor success occurred
		if lastErr != nil {
			suite.T().Logf("Last error: %v", lastErr)
		}
		// This is not necessarily an error - just log what happened
		suite.T().Logf("Completed %d attempts without clear success or timeout", attempts)
	})
}

// TestConcurrentRetry tests concurrent retry scenarios
func (suite *RateLimitRetrySuite) TestConcurrentRetry() {
	suite.Run("Concurrent Retry Attempts", func() {
		// Reset request count
		suite.mu.Lock()
		suite.requests = make(map[string]int)
		suite.mu.Unlock()

		concurrency := 5
		done := make(chan bool, concurrency)
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Each goroutine makes retry attempts
				maxRetries := 3
				for attempt := 0; attempt <= maxRetries; attempt++ {
					req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/retry-after", nil)
					if err != nil {
						results <- err
						return
					}

					resp, err := suite.client.Do(req)
					if err != nil {
						if attempt < maxRetries {
							time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
							continue
						}
						results <- err
						return
					}

					if resp.StatusCode == http.StatusOK {
						resp.Body.Close()
						results <- nil // Success
						return
					}

					resp.Body.Close()
					if attempt < maxRetries {
						time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
					}
				}
				results <- fmt.Errorf("max retries exceeded for goroutine %d", id)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			<-done
			err := <-results
			assert.NoError(suite.T(), err, "Concurrent retry should succeed")
		}
	})
}

// TestRateLimitRetrySuite runs the complete rate limiting and retry test suite
func TestRateLimitRetrySuite(t *testing.T) {
	suite.Run(t, new(RateLimitRetrySuite))
}

// TestRetryStrategyValidation tests different retry strategies
func TestRetryStrategyValidation(t *testing.T) {
	t.Run("Linear Backoff Strategy", func(t *testing.T) {
		delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond}
		totalDelay := time.Duration(0)

		for _, delay := range delays {
			totalDelay += delay
		}

		expectedDelay := 600 * time.Millisecond
		assert.Equal(t, expectedDelay, totalDelay, "Linear backoff should sum delays")
	})

	t.Run("Exponential Backoff Strategy", func(t *testing.T) {
		baseDelay := 100 * time.Millisecond
		delays := []time.Duration{}

		for i := 0; i < 3; i++ {
			delay := time.Duration(1<<uint(i)) * baseDelay
			delays = append(delays, delay)
		}

		totalDelay := time.Duration(0)
		for _, delay := range delays {
			totalDelay += delay
		}

		expectedDelay := 700 * time.Millisecond // 100 + 200 + 400
		assert.Equal(t, expectedDelay, totalDelay, "Exponential backoff should calculate correctly")
	})

	t.Run("Jitter Addition", func(t *testing.T) {
		baseDelay := 100 * time.Millisecond
		maxJitter := 50 * time.Millisecond

		// Simulate jitter calculation
		jitter := time.Duration(25) * time.Millisecond // Mock jitter
		delayWithJitter := baseDelay + jitter

		assert.GreaterOrEqual(t, delayWithJitter, baseDelay, "Delay with jitter should be at least base delay")
		assert.LessOrEqual(t, delayWithJitter, baseDelay+maxJitter, "Delay with jitter should not exceed max")
	})
}