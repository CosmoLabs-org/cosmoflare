package ux

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		info := ClassifyError(nil)
		assert.Equal(t, ErrorTypeUnknown, info.Type)
		assert.False(t, info.Retryable)
	})

	t.Run("network error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("connection refused"))
		assert.Equal(t, ErrorTypeNetwork, info.Type)
		assert.True(t, info.Retryable)
		assert.Equal(t, "NETWORK_ERROR", info.ErrorCode)
	})

	t.Run("timeout error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("context deadline exceeded"))
		assert.Equal(t, ErrorTypeTimeout, info.Type)
		assert.True(t, info.Retryable)
	})

	t.Run("permission error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("permission denied"))
		assert.Equal(t, ErrorTypePermission, info.Type)
		assert.False(t, info.Retryable)
	})

	t.Run("storage error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("no space left on device"))
		assert.Equal(t, ErrorTypeStorage, info.Type)
		assert.False(t, info.Retryable)
	})

	t.Run("auth error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("authentication failed"))
		assert.Equal(t, ErrorTypeAuthentication, info.Type)
		assert.False(t, info.Retryable)
	})

	t.Run("quota error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("rate limit exceeded"))
		assert.Equal(t, ErrorTypeQuota, info.Type)
		assert.True(t, info.Retryable)
	})

	t.Run("filesystem error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("no such file or directory"))
		assert.Equal(t, ErrorTypeFileSystem, info.Type)
		assert.False(t, info.Retryable)
	})

	t.Run("unknown error", func(t *testing.T) {
		info := ClassifyError(fmt.Errorf("something weird happened"))
		assert.Equal(t, ErrorTypeUnknown, info.Type)
		assert.False(t, info.Retryable)
	})
}

func TestDefaultRetryStrategy(t *testing.T) {
	s := DefaultRetryStrategy()
	assert.Equal(t, 3, s.MaxRetries)
	assert.Equal(t, time.Second, s.BaseDelay)
	assert.Equal(t, 30*time.Second, s.MaxDelay)
	assert.Equal(t, 2.0, s.BackoffFactor)
	assert.True(t, s.Jitter)
}

func TestNetworkRetryStrategy(t *testing.T) {
	s := NetworkRetryStrategy()
	assert.Equal(t, 5, s.MaxRetries)
	assert.Equal(t, 2*time.Second, s.BaseDelay)
}

func TestFileSystemRetryStrategy(t *testing.T) {
	s := FileSystemRetryStrategy()
	assert.Equal(t, 2, s.MaxRetries)
}

func TestRetryWithStrategy(t *testing.T) {
	t.Run("immediate success", func(t *testing.T) {
		callCount := 0
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
		}, func() error {
			callCount++
			return nil
		})

		assert.True(t, result.Success)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, 1, result.Attempts)
		assert.False(t, result.Retryed)
	})

	t.Run("retry then succeed", func(t *testing.T) {
		callCount := 0
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			callCount++
			if callCount < 3 {
				return fmt.Errorf("connection refused") // retryable
			}
			return nil
		})

		assert.True(t, result.Success)
		assert.Equal(t, 3, callCount)
		assert.True(t, result.Retryed)
	})

	t.Run("non-retryable error stops immediately", func(t *testing.T) {
		callCount := 0
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    5,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
		}, func() error {
			callCount++
			return fmt.Errorf("permission denied") // not retryable
		})

		assert.False(t, result.Success)
		assert.Equal(t, 1, callCount)
	})

	t.Run("nil strategy uses default", func(t *testing.T) {
		result := RetryWithStrategy(nil, func() error {
			return nil
		})
		assert.True(t, result.Success)
	})

	t.Run("exhausts all retries", func(t *testing.T) {
		callCount := 0
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    2,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			callCount++
			return fmt.Errorf("connection refused")
		})

		assert.False(t, result.Success)
		assert.Equal(t, 3, callCount) // 1 initial + 2 retries
		assert.NotNil(t, result.LastError)
	})
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := RetryWithBackoff(0, func() error { return nil })
		assert.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		err := RetryWithBackoff(0, func() error {
			return fmt.Errorf("always fails")
		})
		assert.Error(t, err)
	})
}

func TestCalculateDelay(t *testing.T) {
	t.Run("exponential", func(t *testing.T) {
		s := &RetryStrategy{
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Strategy:      "exponential",
		}
		d := calculateDelay(s, 0)
		assert.Equal(t, time.Second, d)

		d = calculateDelay(s, 1)
		assert.Equal(t, 2*time.Second, d)

		d = calculateDelay(s, 2)
		assert.Equal(t, 4*time.Second, d)
	})

	t.Run("linear", func(t *testing.T) {
		s := &RetryStrategy{
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Strategy:      "linear",
		}
		d := calculateDelay(s, 0)
		assert.Equal(t, time.Second, d)

		d = calculateDelay(s, 2)
		assert.Equal(t, 3*time.Second, d)
	})

	t.Run("fixed", func(t *testing.T) {
		s := &RetryStrategy{
			BaseDelay: time.Second,
			MaxDelay:  30 * time.Second,
			Strategy:  "fixed",
		}
		d := calculateDelay(s, 0)
		assert.Equal(t, time.Second, d)
		d = calculateDelay(s, 5)
		assert.Equal(t, time.Second, d)
	})

	t.Run("max delay cap", func(t *testing.T) {
		s := &RetryStrategy{
			BaseDelay:     time.Second,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 10.0,
			Strategy:      "exponential",
		}
		d := calculateDelay(s, 3)
		assert.Equal(t, 5*time.Second, d)
	})

	t.Run("with jitter", func(t *testing.T) {
		s := &RetryStrategy{
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Strategy:      "exponential",
			Jitter:        true,
		}
		d := calculateDelay(s, 0)
		// With 10% jitter, delay should be between 1s and 1.1s
		assert.GreaterOrEqual(t, d, time.Second)
		assert.LessOrEqual(t, d, time.Second+110*time.Millisecond)
	})
}

func TestErrorTypeToString(t *testing.T) {
	tests := []struct {
		input    ErrorType
		expected string
	}{
		{ErrorTypeNetwork, "Network Error"},
		{ErrorTypePermission, "Permission Error"},
		{ErrorTypeStorage, "Storage Error"},
		{ErrorTypeAuthentication, "Authentication Error"},
		{ErrorTypeQuota, "Quota Error"},
		{ErrorTypeValidation, "Validation Error"},
		{ErrorTypeTimeout, "Timeout Error"},
		{ErrorTypeUserCancel, "User Cancelled"},
		{ErrorTypeFileSystem, "File System Error"},
		{ErrorType(99), "Unknown Error"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, errorTypeToString(tt.input))
	}
}

func TestContainsAny(t *testing.T) {
	assert.True(t, containsAny("connection refused by server", []string{"refused", "denied"}))
	assert.False(t, containsAny("everything is fine", []string{"error", "fail"}))
	assert.False(t, containsAny("test", []string{}))
}

func TestHandleBatchErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		s, e := HandleBatchErrors(nil, true)
		assert.Equal(t, 0, s)
		assert.Equal(t, 0, e)
	})

	t.Run("mixed results with continue", func(t *testing.T) {
		errors := []error{
			nil,
			fmt.Errorf("connection refused"),
			nil,
			fmt.Errorf("permission denied"),
		}
		s, e := HandleBatchErrors(errors, true)
		assert.Equal(t, 2, s)
		assert.Equal(t, 2, e)
	})

	t.Run("stop on first error", func(t *testing.T) {
		errors := []error{
			fmt.Errorf("connection refused"),
			fmt.Errorf("another error"),
		}
		s, e := HandleBatchErrors(errors, false)
		assert.Equal(t, 0, s)
		assert.Equal(t, 1, e)
	})
}

func TestPrintRetryProgress(t *testing.T) {
	t.Run("single attempt no output", func(t *testing.T) {
		assert.NotPanics(t, func() {
			PrintRetryProgress(&RetryResult{Attempts: 1})
		})
	})

	t.Run("multiple attempts with details", func(t *testing.T) {
		assert.NotPanics(t, func() {
			PrintRetryProgress(&RetryResult{
				Attempts:      3,
				TotalTime:     5 * time.Second,
				BackoffDelays: []time.Duration{time.Second, 2 * time.Second},
				LastError:     fmt.Errorf("timeout"),
			})
		})
	})
}

func TestSmartRetryContext(t *testing.T) {
	ctx := NewSmartRetryContext()
	require.NotNil(t, ctx)

	s := NetworkRetryStrategy()
	ctx.SetStrategy(ErrorTypeNetwork, s)
	assert.Equal(t, s, ctx.strategies[ErrorTypeNetwork])
}
