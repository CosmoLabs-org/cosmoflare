package cosmoflare

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// statusError is a test error implementing httpStatusCarrier.
type statusError struct {
	status int
	msg    string
}

func (e *statusError) Error() string                { return e.msg }
func (e *statusError) StatusCode() int               { return e.status }
func (e *statusError) Unwrap() error                 { return nil }
func (e *statusError) Temporary() bool               { return false }
func (e *statusError) Timeout() bool                 { return false }

// smithyStyleError implements HTTPStatusCode() (AWS SDK pattern).
type smithyStyleError struct {
	status int
	msg    string
}

func (e *smithyStyleError) Error() string           { return e.msg }
func (e *smithyStyleError) HTTPStatusCode() int      { return e.status }
func (e *smithyStyleError) Unwrap() error           { return nil }

// timeoutNetError is a net.Error that is a timeout.
type timeoutNetError struct {
	msg string
}

func (e *timeoutNetError) Error() string      { return e.msg }
func (e *timeoutNetError) Timeout() bool      { return true }
func (e *timeoutNetError) Temporary() bool    { return true }

// --- Tests ---

func TestDo_SuccessOnFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return &statusError{status: http.StatusServiceUnavailable, msg: "503"}
		}
		return nil
	}, WithInitialDelay(1*time.Millisecond))
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_MaxRetriesExhausted(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return &statusError{status: http.StatusBadGateway, msg: "502"}
	}, WithMaxRetries(2), WithInitialDelay(1*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry exhausted after 3 attempts")
	assert.Equal(t, 3, calls)
}

func TestDo_NonRetryableError_StopsImmediately(t *testing.T) {
	calls := 0
	// 404 is not retryable.
	err := Do(context.Background(), func() error {
		calls++
		return &statusError{status: http.StatusNotFound, msg: "404"}
	})
	require.Error(t, err)
	// Should only be called once — no retry for non-transient errors.
	assert.Equal(t, 1, calls)
}

func TestDo_ContextCancellation_StopsRetries(t *testing.T) {
	calls := int32(0)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, func() error {
		atomic.AddInt32(&calls, 1)
		return &statusError{status: http.StatusInternalServerError, msg: "500"}
	}, WithInitialDelay(10*time.Millisecond), WithMaxRetries(20))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	// Should have stopped well before 20 attempts.
	assert.Less(t, atomic.LoadInt32(&calls), int32(20))
}

func TestDo_DelayCapsAtMaxDelay(t *testing.T) {
	var mu sync.Mutex
	timestamps := []time.Time{}
	calls := 0

	err := Do(context.Background(), func() error {
		mu.Lock()
		timestamps = append(timestamps, time.Now())
		mu.Unlock()
		calls++
		if calls < 4 {
			return &statusError{status: http.StatusServiceUnavailable, msg: "503"}
		}
		return nil
	}, WithInitialDelay(50*time.Millisecond), WithMaxDelay(100*time.Millisecond), WithMultiplier(10.0))

	assert.NoError(t, err)
	require.Len(t, timestamps, 4)

	mu.Lock()
	defer mu.Unlock()

	// delay between attempt 2 and 3 should be capped at MaxDelay (100ms),
	// not 50ms * 10^1 = 500ms.
	delay2to3 := timestamps[2].Sub(timestamps[1])
	assert.Less(t, delay2to3, 200*time.Millisecond, "delay should be capped at MaxDelay")
}

func TestDo_RetryOptions(t *testing.T) {
	cfg := defaultRetryConfig()
	WithMaxRetries(5)(&cfg)
	assert.Equal(t, 5, cfg.MaxRetries)

	WithInitialDelay(2 * time.Second)(&cfg)
	assert.Equal(t, 2*time.Second, cfg.InitialDelay)

	WithMaxDelay(1 * time.Minute)(&cfg)
	assert.Equal(t, 1*time.Minute, cfg.MaxDelay)

	WithMultiplier(3.5)(&cfg)
	assert.Equal(t, 3.5, cfg.Multiplier)

	WithJitter(true)(&cfg)
	assert.True(t, cfg.Jitter)
}

func TestDo_JitterEnabled(t *testing.T) {
	// With jitter, the delay between retries should vary.
	// Run many times and check that not all delays are identical.
	delays := make([]time.Duration, 0, 50)
	for range 50 {
		var timestamps [2]time.Time
		calls := 0
		_ = Do(context.Background(), func() error {
			if calls < 2 {
				timestamps[calls] = time.Now()
			}
			calls++
			if calls < 2 {
				return &statusError{status: http.StatusServiceUnavailable, msg: "503"}
			}
			return nil
		}, WithInitialDelay(10*time.Millisecond), WithMaxRetries(1), WithJitter(true))
		if calls >= 2 {
			delays = append(delays, timestamps[1].Sub(timestamps[0]))
		}
	}
	// With jitter over 50 runs, we should see some variation.
	// Check that not all delays are exactly the same (within 1ms tolerance).
	if len(delays) >= 2 {
		unique := make(map[time.Duration]bool)
		for _, d := range delays {
			// Bucket to nearest millisecond for comparison.
			bucketed := d.Round(time.Millisecond)
			unique[bucketed] = true
		}
		// Jitter should produce at least 2 distinct delay buckets.
		assert.GreaterOrEqual(t, len(unique), 2, "jitter should produce varied delays")
	}
}

// --- isRetryable tests ---

func TestIsRetryable_NilError(t *testing.T) {
	assert.False(t, isRetryable(nil))
}

func TestIsRetryable_NetTimeout(t *testing.T) {
	err := &timeoutNetError{msg: "connection timed out"}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_NetOpError(t *testing.T) {
	err := &net.OpError{Op: "dial", Net: "tcp", Err: fmt.Errorf("connection refused")}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_HTTP429(t *testing.T) {
	err := &statusError{status: http.StatusTooManyRequests, msg: "rate limited"}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_HTTP500(t *testing.T) {
	err := &statusError{status: http.StatusInternalServerError, msg: "internal server error"}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_HTTP503(t *testing.T) {
	err := &statusError{status: http.StatusServiceUnavailable, msg: "service unavailable"}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_HTTP400_NotRetryable(t *testing.T) {
	err := &statusError{status: http.StatusBadRequest, msg: "bad request"}
	assert.False(t, isRetryable(err))
}

func TestIsRetryable_HTTP404_NotRetryable(t *testing.T) {
	err := &statusError{status: http.StatusNotFound, msg: "not found"}
	assert.False(t, isRetryable(err))
}

func TestIsRetryable_WrappedCloudflareError(t *testing.T) {
	// Simulate a wrapped error chain: outer error wraps a cloudflare-style status error.
	outer := fmt.Errorf("upload failed: %w", &statusError{status: http.StatusBadGateway, msg: "502"})
	assert.True(t, isRetryable(outer))
}

func TestIsRetryable_SmithyStyleError(t *testing.T) {
	err := &smithyStyleError{status: http.StatusInternalServerError, msg: "InternalError"}
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_R2QuotaError(t *testing.T) {
	err := quotaError("PutObject", "rate limit exceeded", nil)
	assert.True(t, isRetryable(err))
}

func TestIsRetryable_R2ValidationError_NotRetryable(t *testing.T) {
	err := validationError("PutObject", "invalid key")
	assert.False(t, isRetryable(err))
}

func TestIsRetryable_PlainError_NotRetryable(t *testing.T) {
	err := errors.New("something went wrong")
	assert.False(t, isRetryable(err))
}

func TestIsRetryable_WrappedNetTimeout(t *testing.T) {
	outer := fmt.Errorf("operation failed: %w", &timeoutNetError{msg: "timeout"})
	assert.True(t, isRetryable(outer))
}
