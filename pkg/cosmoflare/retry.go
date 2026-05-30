package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"time"
)

// RetryConfig holds the parameters for retry behavior.
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// RetryOption is a functional option for configuring retry behavior.
type RetryOption func(*RetryConfig)

// defaultRetryConfig returns sensible defaults.
func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) RetryOption {
	return func(c *RetryConfig) { c.MaxRetries = n }
}

// WithInitialDelay sets the initial delay before the first retry.
func WithInitialDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.InitialDelay = d }
}

// WithMaxDelay sets the upper bound for the exponential backoff delay.
func WithMaxDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.MaxDelay = d }
}

// WithMultiplier sets the backoff multiplier (default 2.0).
func WithMultiplier(m float64) RetryOption {
	return func(c *RetryConfig) { c.Multiplier = m }
}

// WithJitter enables random jitter (0-25% of delay) to spread retries.
func WithJitter(enabled bool) RetryOption {
	return func(c *RetryConfig) { c.Jitter = enabled }
}

// RetryableFunc is the function signature accepted by Do.
type RetryableFunc func() error

// Do executes fn with exponential backoff retry on transient errors.
// It respects context cancellation and returns the last error if all
// retries are exhausted.
func Do(ctx context.Context, fn RetryableFunc, opts ...RetryOption) error {
	cfg := defaultRetryConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context before each attempt.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retry aborted: context canceled after %d attempts: %w", attempt, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// If this was the last allowed attempt, return immediately.
		if attempt == cfg.MaxRetries {
			break
		}

		// Only retry on transient errors.
		if !isRetryable(lastErr) {
			return lastErr
		}

		// Calculate delay with exponential backoff.
		delay := time.Duration(float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt)))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		// Apply jitter: add random 0-25% of the delay.
		if cfg.Jitter {
			jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
			delay += jitter
		}

		// Wait with context awareness.
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry aborted: context canceled while waiting: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt.
		}
	}

	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxRetries+1, lastErr)
}

// isRetryable determines whether an error is transient and worth retrying.
// It checks for net.Error timeouts, and for errors carrying HTTP status
// codes that indicate server-side or rate-limit issues (429, >= 500).
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Network-level errors: timeouts, connection refused, etc.
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
	}

	// Check for HTTP status codes in wrapped errors.
	// This covers cloudflare-go errors, AWS SDK errors, and any error
	// that wraps an HTTP status code.
	if isHTTPStatusRetryable(err) {
		return true
	}

	// Check for our own quota/rate-limit error type.
	var quotaErr *R2QuotaError
	if errors.As(err, &quotaErr) {
		return true
	}

	// Connection refused and reset errors (common in transient failures).
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	return false
}

// httpStatusCarrier is an interface for errors that expose an HTTP status code.
// Both cloudflare-go and aws/smithy-go errors implement this pattern.
type httpStatusCarrier interface {
	StatusCode() int
}

// httpStatusError is an interface for errors that carry an HTTP response.
type httpStatusError interface {
	HTTPStatusCode() int
}

// isHTTPStatusRetryable checks if any wrapped error carries a retryable HTTP status.
func isHTTPStatusRetryable(err error) bool {
	for unwrapped := err; unwrapped != nil; {
		// cloudflare-go style: StatusCode() method.
		if sc, ok := unwrapped.(httpStatusCarrier); ok {
			return sc.StatusCode() == http.StatusTooManyRequests || sc.StatusCode() >= 500
		}
		// AWS SDK / smithy-go style: HTTPStatusCode() method.
		if sc, ok := unwrapped.(httpStatusError); ok {
			return sc.HTTPStatusCode() == http.StatusTooManyRequests || sc.HTTPStatusCode() >= 500
		}
		unwrapped = errors.Unwrap(unwrapped)
	}
	return false
}
