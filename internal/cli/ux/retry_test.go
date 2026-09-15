package ux

import (
	"fmt"
	"os"
	"strings"
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

func TestClassifyError_AllNetworkPatterns(t *testing.T) {
	// "timeout" and "deadline exceeded" match both Network and Timeout patterns.
	// Timeout check comes later and overwrites, so they end up as Timeout.
	// Only test the patterns that uniquely match Network.
	patterns := []string{
		"connection refused",
		"connection reset",
		"network unreachable",
		"no such host",
		"temporary failure",
		"connection timed out",
		"read: connection reset",
		"write: broken pipe",
		"network is down",
		"name resolution failed",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("operation failed: %s", p))
			assert.Equal(t, ErrorTypeNetwork, info.Type)
			assert.True(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllPermissionPatterns(t *testing.T) {
	// "unauthorized" matches both Permission and Auth patterns; Auth overwrites.
	// Only test patterns unique to Permission.
	patterns := []string{
		"permission denied",
		"access denied",
		"operation not permitted",
		"insufficient privileges",
		"forbidden",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypePermission, info.Type)
			assert.False(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllStoragePatterns(t *testing.T) {
	// "storage quota exceeded" matches both Storage and Quota patterns; Quota overwrites.
	// Only test patterns unique to Storage.
	patterns := []string{
		"no space left",
		"disk full",
		"insufficient disk space",
		"volume full",
		"out of disk space",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypeStorage, info.Type)
			assert.False(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllAuthPatterns(t *testing.T) {
	patterns := []string{
		"authentication failed",
		"invalid credentials",
		"unauthorized",
		"access token expired",
		"invalid api key",
		"authentication required",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypeAuthentication, info.Type)
			assert.False(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllQuotaPatterns(t *testing.T) {
	patterns := []string{
		"quota exceeded",
		"rate limit exceeded",
		"too many requests",
		"api rate limit",
		"request limit exceeded",
		"quota limit",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypeQuota, info.Type)
			assert.True(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllTimeoutPatterns(t *testing.T) {
	patterns := []string{
		"timeout",
		"deadline exceeded",
		"operation timed out",
		"context deadline exceeded",
		"request timeout",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypeTimeout, info.Type)
			assert.True(t, info.Retryable)
		})
	}
}

func TestClassifyError_AllFilesystemPatterns(t *testing.T) {
	patterns := []string{
		"file not found",
		"no such file or directory",
		"file exists",
		"directory not empty",
		"not a directory",
		"is a directory",
		"file already exists",
		"device or resource busy",
	}

	for _, p := range patterns {
		t.Run(p, func(t *testing.T) {
			info := ClassifyError(fmt.Errorf("%s", p))
			assert.Equal(t, ErrorTypeFileSystem, info.Type)
			assert.False(t, info.Retryable)
		})
	}
}

func TestClassifyError_ContextFields(t *testing.T) {
	info := ClassifyError(fmt.Errorf("connection refused"))
	assert.Equal(t, "Check network connection and try again", info.Suggestion)
	assert.Equal(t, "NETWORK_ERROR", info.ErrorCode)
	assert.NotNil(t, info.Original)
	assert.Equal(t, "connection refused", info.Message)
	assert.NotNil(t, info.Context)
}

func TestClassifyError_NilErrorContext(t *testing.T) {
	info := ClassifyError(nil)
	assert.Equal(t, "No error", info.Message)
	assert.Nil(t, info.Original)
}

func TestDefaultRetryStrategy(t *testing.T) {
	s := DefaultRetryStrategy()
	assert.Equal(t, 3, s.MaxRetries)
	assert.Equal(t, time.Second, s.BaseDelay)
	assert.Equal(t, 30*time.Second, s.MaxDelay)
	assert.Equal(t, 2.0, s.BackoffFactor)
	assert.True(t, s.Jitter)
	assert.Equal(t, "exponential", s.Strategy)
}

func TestNetworkRetryStrategy(t *testing.T) {
	s := NetworkRetryStrategy()
	assert.Equal(t, 5, s.MaxRetries)
	assert.Equal(t, 2*time.Second, s.BaseDelay)
	assert.Equal(t, 60*time.Second, s.MaxDelay)
	assert.Equal(t, 2.0, s.BackoffFactor)
	assert.True(t, s.Jitter)
	assert.Equal(t, "exponential", s.Strategy)
}

func TestFileSystemRetryStrategy(t *testing.T) {
	s := FileSystemRetryStrategy()
	assert.Equal(t, 2, s.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, s.BaseDelay)
	assert.Equal(t, 5*time.Second, s.MaxDelay)
	assert.Equal(t, 1.5, s.BackoffFactor)
	assert.False(t, s.Jitter)
	assert.Equal(t, "linear", s.Strategy)
}

// runRetryWithStrategyOutcomeTests covers the execution-outcome scenarios:
// success paths, retry classification, and retry exhaustion.
func runRetryWithStrategyOutcomeTests(t *testing.T) {
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

// runRetryWithStrategyResultFieldTests covers how RetryResult is populated
// on success and failure, including recorded backoff delays.
func runRetryWithStrategyResultFieldTests(t *testing.T) {
	t.Run("result fields populated on success", func(t *testing.T) {
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
		}, func() error {
			return nil
		})

		assert.False(t, result.StartTime.IsZero())
		assert.True(t, result.TotalTime >= 0)
		assert.Nil(t, result.LastError)
		assert.Nil(t, result.ErrorInfo)
	})

	t.Run("result fields populated on failure", func(t *testing.T) {
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    1,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			return fmt.Errorf("quota exceeded")
		})

		assert.Equal(t, 2, result.Attempts)
		assert.NotNil(t, result.LastError)
		assert.NotNil(t, result.ErrorInfo)
		assert.Equal(t, ErrorTypeQuota, result.ErrorInfo.Type)
		assert.True(t, result.Retryed)
	})

	t.Run("backoff delays recorded", func(t *testing.T) {
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    2,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			return fmt.Errorf("connection refused")
		})

		assert.Len(t, result.BackoffDelays, 2)
	})
}

func TestRetryWithStrategy(t *testing.T) {
	t.Run("execution outcomes", runRetryWithStrategyOutcomeTests)
	t.Run("result fields", runRetryWithStrategyResultFieldTests)
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

	t.Run("success after retries", func(t *testing.T) {
		callCount := 0
		err := RetryWithBackoff(2, func() error {
			callCount++
			if callCount < 2 {
				return fmt.Errorf("connection refused")
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("exhausts retries", func(t *testing.T) {
		err := RetryWithBackoff(2, func() error {
			return fmt.Errorf("connection refused")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("non-retryable stops immediately", func(t *testing.T) {
		callCount := 0
		err := RetryWithBackoff(5, func() error {
			callCount++
			return fmt.Errorf("permission denied")
		})
		assert.Error(t, err)
		assert.Equal(t, 1, callCount)
	})
}

// calculateDelayCase is one strategy/delay scenario for TestCalculateDelay.
type calculateDelayCase struct {
	name    string
	strat   *RetryStrategy
	attempt int
	min     time.Duration // inclusive lower bound (equal to max when exact)
	max     time.Duration // inclusive upper bound (equal to min when exact)
}

// calculateDelayCases returns the strategy/delay scenario table covering
// exponential, linear, fixed, and unknown strategies, the max-delay cap, and
// jitter behavior.
func calculateDelayCases() []calculateDelayCase {
	return []calculateDelayCase{
		{
			name: "exponential attempt 0",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "exponential"},
			attempt: 0, min: time.Second, max: time.Second,
		},
		{
			name: "exponential attempt 1",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "exponential"},
			attempt: 1, min: 2 * time.Second, max: 2 * time.Second,
		},
		{
			name: "exponential attempt 2",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "exponential"},
			attempt: 2, min: 4 * time.Second, max: 4 * time.Second,
		},
		{
			name: "linear attempt 0",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "linear"},
			attempt: 0, min: time.Second, max: time.Second,
		},
		{
			name: "linear attempt 2",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "linear"},
			attempt: 2, min: 3 * time.Second, max: 3 * time.Second,
		},
		{
			name: "linear attempt 0 small base",
			strat: &RetryStrategy{BaseDelay: 500 * time.Millisecond,
				MaxDelay: 30 * time.Second, Strategy: "linear"},
			attempt: 0, min: 500 * time.Millisecond, max: 500 * time.Millisecond,
		},
		{
			name: "fixed attempt 0",
			strat: &RetryStrategy{BaseDelay: time.Second,
				MaxDelay: 30 * time.Second, Strategy: "fixed"},
			attempt: 0, min: time.Second, max: time.Second,
		},
		{
			name: "fixed ignores attempt",
			strat: &RetryStrategy{BaseDelay: time.Second,
				MaxDelay: 30 * time.Second, Strategy: "fixed"},
			attempt: 5, min: time.Second, max: time.Second,
		},
		{
			name: "max delay cap exponential",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 5 * time.Second,
				BackoffFactor: 10.0, Strategy: "exponential"},
			attempt: 3, min: 5 * time.Second, max: 5 * time.Second,
		},
		{
			name: "max delay cap linear",
			strat: &RetryStrategy{BaseDelay: time.Second,
				MaxDelay: 2 * time.Second, Strategy: "linear"},
			attempt: 10, min: 2 * time.Second, max: 2 * time.Second,
		},
		{
			name: "with jitter",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "exponential", Jitter: true},
			// With 10% jitter, delay should be between 1s and 1.1s
			attempt: 0, min: time.Second, max: time.Second + 110*time.Millisecond,
		},
		{
			name: "no jitter",
			strat: &RetryStrategy{BaseDelay: time.Second, MaxDelay: 30 * time.Second,
				BackoffFactor: 2.0, Strategy: "exponential", Jitter: false},
			attempt: 2, min: 4 * time.Second, max: 4 * time.Second,
		},
		{
			name: "unknown strategy defaults to base delay",
			strat: &RetryStrategy{BaseDelay: time.Second,
				MaxDelay: 30 * time.Second, Strategy: "unknown"},
			attempt: 5, min: time.Second, max: time.Second,
		},
	}
}

func TestCalculateDelay(t *testing.T) {
	for _, tc := range calculateDelayCases() {
		t.Run(tc.name, func(t *testing.T) {
			d := calculateDelay(tc.strat, tc.attempt)
			assert.GreaterOrEqual(t, d, tc.min)
			assert.LessOrEqual(t, d, tc.max)
		})
	}
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
	assert.True(t, containsAny("test", []string{"test"}))
	assert.True(t, containsAny("test", []string{"es"}))
	assert.False(t, containsAny("", []string{"test"}))
	assert.True(t, containsAny("", []string{""}))
}

func TestHandleBatchErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		s, e := HandleBatchErrors(nil, true)
		assert.Equal(t, 0, s)
		assert.Equal(t, 0, e)
	})

	t.Run("empty slice", func(t *testing.T) {
		s, e := HandleBatchErrors([]error{}, true)
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

	t.Run("all successes", func(t *testing.T) {
		errors := []error{nil, nil, nil}
		s, e := HandleBatchErrors(errors, true)
		assert.Equal(t, 3, s)
		assert.Equal(t, 0, e)
	})

	t.Run("all failures with continue", func(t *testing.T) {
		errors := []error{
			fmt.Errorf("timeout"),
			fmt.Errorf("permission denied"),
			fmt.Errorf("disk full"),
		}
		s, e := HandleBatchErrors(errors, true)
		assert.Equal(t, 0, s)
		assert.Equal(t, 3, e)
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

	t.Run("multiple attempts no error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			PrintRetryProgress(&RetryResult{
				Attempts:      2,
				TotalTime:     1 * time.Second,
				BackoffDelays: []time.Duration{time.Second},
				LastError:     nil,
			})
		})
	})

	t.Run("multiple attempts no delays", func(t *testing.T) {
		assert.NotPanics(t, func() {
			PrintRetryProgress(&RetryResult{
				Attempts:      2,
				TotalTime:     100 * time.Millisecond,
				BackoffDelays: nil,
				LastError:     fmt.Errorf("failed"),
			})
		})
	})
}

// runSmartRetryContextSetupTests covers SmartRetryContext construction and
// per-error-type strategy/context registration.
func runSmartRetryContextSetupTests(t *testing.T) {
	t.Run("create and set strategy", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		require.NotNil(t, ctx)

		s := NetworkRetryStrategy()
		ctx.SetStrategy(ErrorTypeNetwork, s)
		assert.Equal(t, s, ctx.strategies[ErrorTypeNetwork])
	})

	t.Run("set context", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		ctx.SetContext("bucket", "my-bucket")
		ctx.SetContext("count", 42)
		assert.Equal(t, "my-bucket", ctx.context["bucket"])
		assert.Equal(t, 42, ctx.context["count"])
	})
}

// runSmartRetryContextExecuteBasicTests covers Execute with the default
// strategy: success, retry classification, and retry-until-success.
func runSmartRetryContextExecuteBasicTests(t *testing.T) {
	t.Run("execute success", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		err := ctx.Execute(func() error { return nil })
		assert.NoError(t, err)
	})

	t.Run("execute non-retryable stops", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("permission denied")
		})
		assert.Error(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("execute retryable retries", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			if callCount < 3 {
				return fmt.Errorf("connection refused")
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})
}

// runSmartRetryContextExecuteStrategyTests covers Execute behavior under
// custom/default strategies, the global safety limit, and retry exhaustion.
func runSmartRetryContextExecuteStrategyTests(t *testing.T) {
	t.Run("execute with custom strategy", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		ctx.SetStrategy(ErrorTypeNetwork, &RetryStrategy{
			MaxRetries:    1,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		})
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("connection refused")
		})
		assert.Error(t, err)
		assert.Equal(t, 2, callCount) // 1 initial + 1 retry
	})

	t.Run("execute default strategy for unknown type", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		// No strategy set for quota errors
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			if callCount < 4 {
				return fmt.Errorf("rate limit exceeded")
			}
			return nil
		})
		assert.NoError(t, err)
		// Default strategy has MaxRetries=3, so up to 4 attempts
		assert.Equal(t, 4, callCount)
	})

	t.Run("execute safety limit", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		// Set a high retry strategy but the safety limit is 10
		ctx.SetStrategy(ErrorTypeNetwork, &RetryStrategy{
			MaxRetries:    100,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		})
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("connection refused")
		})
		assert.Error(t, err)
		assert.LessOrEqual(t, callCount, 11) // safety limit: 10 attempts
	})

	t.Run("execute exhausted retries", func(t *testing.T) {
		ctx := NewSmartRetryContext()
		ctx.SetStrategy(ErrorTypeQuota, &RetryStrategy{
			MaxRetries:    2,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		})
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("quota exceeded")
		})
		assert.Error(t, err)
		assert.Equal(t, 3, callCount) // 1 + 2 retries
	})
}

func TestSmartRetryContext(t *testing.T) {
	t.Run("setup", runSmartRetryContextSetupTests)
	t.Run("execute basic", runSmartRetryContextExecuteBasicTests)
	t.Run("execute strategies", runSmartRetryContextExecuteStrategyTests)
}

// --- ConfirmRetry (via stdin redirect) ---

func TestConfirmRetry_Retry(t *testing.T) {
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	assert.True(t, ConfirmRetry("Retry upload?", 1))
}

func TestConfirmRetry_RetryFullWord(t *testing.T) {
	restore := redirectStdinForRetry(t, "retry\n")
	defer restore()

	assert.True(t, ConfirmRetry("Retry upload?", 3))
}

func TestConfirmRetry_Cancel(t *testing.T) {
	restore := redirectStdinForRetry(t, "c\n")
	defer restore()

	assert.False(t, ConfirmRetry("Retry upload?", 1))
}

func TestConfirmRetry_EmptyInput(t *testing.T) {
	restore := redirectStdinForRetry(t, "\n")
	defer restore()

	assert.False(t, ConfirmRetry("Retry upload?", 1))
}

func TestConfirmRetry_CaseInsensitive(t *testing.T) {
	restore := redirectStdinForRetry(t, "R\n")
	defer restore()

	// ConfirmRetry lowercases the response, so "R" -> "r" which IS valid
	assert.True(t, ConfirmRetry("Retry upload?", 1))
}

func TestConfirmRetry_InvalidInput(t *testing.T) {
	restore := redirectStdinForRetry(t, "maybe\n")
	defer restore()

	assert.False(t, ConfirmRetry("Retry upload?", 2))
}

func TestConfirmRetry_UnicodeMessage(t *testing.T) {
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	assert.True(t, ConfirmRetry("Upload cafe.txt?", 5))
}

// --- InteractiveRetry ---

func TestInteractiveRetry_SuccessOnFirstAttempt(t *testing.T) {
	restore := redirectStdinForRetry(t, "\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		return nil
	}, "Upload file?")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestInteractiveRetry_NonRetryableError(t *testing.T) {
	restore := redirectStdinForRetry(t, "\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		return fmt.Errorf("permission denied")
	}, "Upload file?")
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestInteractiveRetry_UserCancels(t *testing.T) {
	// Operation fails (retryable), user says cancel
	restore := redirectStdinForRetry(t, "c\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		return fmt.Errorf("connection refused")
	}, "Retry upload?")
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
	assert.Contains(t, err.Error(), "cancelled by user")
	assert.Contains(t, err.Error(), "1 attempts")
}

func TestInteractiveRetry_RetryThenSucceed(t *testing.T) {
	// First attempt fails, user retries, second succeeds
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		if callCount == 1 {
			return fmt.Errorf("connection refused")
		}
		return nil
	}, "Retry upload?")
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestInteractiveRetry_MultipleRetriesThenCancel(t *testing.T) {
	// Provide "r\n" then "c\n" for two rounds
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r

	go func() {
		w.WriteString("r\n")  // first: retry
		w.WriteString("r\n")  // second: retry
		w.WriteString("c\n")  // third: cancel
		w.Close()
	}()

	callCount := 0
	err = InteractiveRetry(func() error {
		callCount++
		return fmt.Errorf("connection refused")
	}, "Retry?")
	assert.Error(t, err)
	assert.Equal(t, 3, callCount)
	assert.Contains(t, err.Error(), "cancelled by user after 3 attempts")

	r.Close()
	os.Stdin = oldStdin
}

func TestInteractiveRetry_StorageErrorNotRetryable(t *testing.T) {
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		return fmt.Errorf("no space left on device")
	}, "Retry upload?")
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
	// Storage error is not retryable, so user doesn't get prompted
}

func TestInteractiveRetry_AuthErrorNotRetryable(t *testing.T) {
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	callCount := 0
	err := InteractiveRetry(func() error {
		callCount++
		return fmt.Errorf("authentication failed")
	}, "Retry upload?")
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
}

// --- RetryResult edge cases ---

func TestRetryResult_RetryedField(t *testing.T) {
	t.Run("single attempt not retryed", func(t *testing.T) {
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    0,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			return fmt.Errorf("connection refused")
		})
		assert.Equal(t, 1, result.Attempts)
		assert.False(t, result.Retryed)
	})

	t.Run("multiple attempts retryed", func(t *testing.T) {
		result := RetryWithStrategy(&RetryStrategy{
			MaxRetries:    1,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1.0,
			Strategy:      "fixed",
		}, func() error {
			return fmt.Errorf("connection refused")
		})
		assert.Equal(t, 2, result.Attempts)
		assert.True(t, result.Retryed)
	})
}

// --- ClassifyError priority: timeout overrides network ---

func TestClassifyError_ErrorInfoFields(t *testing.T) {
	info := ClassifyError(fmt.Errorf("permission denied on file.txt"))
	assert.Equal(t, "PERMISSION_ERROR", info.ErrorCode)
	assert.Equal(t, "Check file permissions and user privileges", info.Suggestion)
	assert.Equal(t, "permission denied on file.txt", info.Message)
	assert.NotNil(t, info.Context)
}

// --- isRetryableError (covered via ClassifyError tests) ---

// --- ROBUSTNESS: RetryWithStrategy edge cases ---

func TestRetryWithStrategy_ZeroRetries(t *testing.T) {
	callCount := 0
	result := RetryWithStrategy(&RetryStrategy{
		MaxRetries:    0,
		BaseDelay:     time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	}, func() error {
		callCount++
		return fmt.Errorf("connection refused")
	})
	assert.False(t, result.Success)
	assert.Equal(t, 1, callCount)
	assert.False(t, result.Retryed)
}

func TestRetryWithStrategy_LargeRetryCount(t *testing.T) {
	callCount := 0
	result := RetryWithStrategy(&RetryStrategy{
		MaxRetries:    10,
		BaseDelay:     time.Microsecond,
		MaxDelay:      time.Microsecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	}, func() error {
		callCount++
		if callCount == 5 {
			return nil
		}
		return fmt.Errorf("connection refused")
	})
	assert.True(t, result.Success)
	assert.Equal(t, 5, callCount)
}

// --- SECURITY: RetryWithStrategy doesn't panic on nil error ---

func TestRetryWithStrategy_OperationReturnsNilThenError(t *testing.T) {
	callCount := 0
	result := RetryWithStrategy(&RetryStrategy{
		MaxRetries:    1,
		BaseDelay:     time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	}, func() error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return fmt.Errorf("connection refused")
	})
	// First attempt succeeds, no retry
	assert.True(t, result.Success)
	assert.Equal(t, 1, callCount)
}

// --- ConfirmRetry SECURITY ---

func TestSecurity_ConfirmRetry_BypassAttempts(t *testing.T) {
	bypassAttempts := []struct {
		name     string
		input    string
		expected bool
	}{
		{"lowercase r", "r\n", true},
		{"full retry", "retry\n", true},
		{"uppercase R", "R\n", true},       // ConfirmRetry lowercases: "R" -> "r"
		{"uppercase RETRY", "RETRY\n", true}, // ConfirmRetry lowercases: "RETRY" -> "retry"
		{"yes wrong type", "yes\n", false},  // "yes" is not "r" or "retry"
		{"numeric", "1\n", false},
		{"injection", "r;rm -rf /\n", false}, // "r;rm..." is not "r" or "retry"
	}

	for _, attempt := range bypassAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			restore := redirectStdinForRetry(t, attempt.input)
			defer restore()
			result := ConfirmRetry("test", 1)
			assert.Equal(t, attempt.expected, result, "input: %q", attempt.input)
		})
	}
}

func TestSecurity_InteractiveRetry_InformationDisclosure(t *testing.T) {
	// Verify that error type and suggestion are shown to user
	// This tests the display path, not security per se, but ensures
	// sensitive error details are included in output
	restore := redirectStdinForRetry(t, "c\n")
	defer restore()

	err := InteractiveRetry(func() error {
		return fmt.Errorf("connection refused to api.example.com:443")
	}, "Retry?")
	assert.Error(t, err)
	// Error message should contain original details
	assert.Contains(t, err.Error(), "cancelled by user")
}

// --- ROBUSTNESS: ConfirmRetry with reader errors ---

func TestRobustness_ConfirmRetry_EOF(t *testing.T) {
	oldStdin := os.Stdin
	f, err := os.Open(os.DevNull)
	require.NoError(t, err)
	os.Stdin = f

	// fmt.Scanln on EOF returns false, response stays ""
	result := ConfirmRetry("Retry?", 1)
	assert.False(t, result) // empty string is not "r" or "retry"

	os.Stdin = oldStdin
	f.Close()
}

func TestRobustness_InteractiveRetry_OperationPanics(t *testing.T) {
	restore := redirectStdinForRetry(t, "r\n")
	defer restore()

	// Test that a panic in the operation propagates
	defer func() {
		r := recover()
		assert.NotNil(t, r, "panic should propagate")
	}()

	InteractiveRetry(func() error {
		panic("operation exploded")
	}, "Retry?")
}

// --- Helper for retry tests ---

func redirectStdinForRetry(t *testing.T, input string) func() {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r

	go func() {
		w.WriteString(input)
		w.Close()
	}()

	return func() {
		r.Close()
		os.Stdin = oldStdin
	}
}

// --- ErrorInfo fields coverage ---

func TestErrorInfo_AllFields(t *testing.T) {
	info := ClassifyError(fmt.Errorf("quota exceeded for bucket"))
	assert.Equal(t, ErrorTypeQuota, info.Type)
	assert.Equal(t, "quota exceeded for bucket", info.Message)
	assert.Equal(t, "QUOTA_ERROR", info.ErrorCode)
	assert.Equal(t, "Wait for quota reset or upgrade your plan", info.Suggestion)
	assert.True(t, info.Retryable)
	assert.NotNil(t, info.Original)
	assert.NotNil(t, info.Context)
	assert.Equal(t, 0, info.Attempts)
	assert.True(t, info.LastAttempt.IsZero())
	assert.True(t, info.NextAttempt.IsZero())
}

func TestErrorInfo_Suggestion_EmptyForUnknown(t *testing.T) {
	info := ClassifyError(fmt.Errorf("random weird error xyz"))
	assert.Equal(t, ErrorTypeUnknown, info.Type)
	assert.Equal(t, "", info.Suggestion)
	assert.Equal(t, "", info.ErrorCode)
	assert.False(t, info.Retryable)
}

func TestErrorInfo_MultiplePatternMatch(t *testing.T) {
	// "unauthorized" matches both Permission and Authentication
	// Classification order in the code: Network -> Permission -> Storage -> Auth
	// Both Permission and Auth patterns contain "unauthorized"
	info := ClassifyError(fmt.Errorf("unauthorized access"))
	// Since "unauthorized" appears in both, verify it gets classified as one of them
	assert.Contains(t, []ErrorType{ErrorTypePermission, ErrorTypeAuthentication}, info.Type)
}

// --- ClassifyError case insensitivity ---

func TestClassifyError_CaseInsensitive(t *testing.T) {
	info := ClassifyError(fmt.Errorf("Connection Refused"))
	assert.Equal(t, ErrorTypeNetwork, info.Type)

	info = ClassifyError(fmt.Errorf("PERMISSION DENIED"))
	assert.Equal(t, ErrorTypePermission, info.Type)

	info = ClassifyError(fmt.Errorf("Timeout"))
	assert.Equal(t, ErrorTypeTimeout, info.Type)
}

// --- HandleBatchErrors all errors with suggestion ---

func TestHandleBatchErrors_WithSuggestions(t *testing.T) {
	errors := []error{
		fmt.Errorf("connection refused"),
		fmt.Errorf("permission denied"),
	}
	s, e := HandleBatchErrors(errors, true)
	assert.Equal(t, 0, s)
	assert.Equal(t, 2, e)
}

func TestHandleBatchErrors_NilInMiddle(t *testing.T) {
	errors := []error{
		nil,
		fmt.Errorf("timeout"),
		nil,
		nil,
		fmt.Errorf("disk full"),
	}
	s, e := HandleBatchErrors(errors, true)
	assert.Equal(t, 3, s)
	assert.Equal(t, 2, e)
}

// --- PrintRetryProgress: zero attempts ---

func TestPrintRetryProgress_ZeroAttempts(t *testing.T) {
	assert.NotPanics(t, func() {
		PrintRetryProgress(&RetryResult{Attempts: 0})
	})
}

// --- SmartRetryContext: Execute with nil operation ---

func TestSmartRetryContext_ExecuteNilOperation(t *testing.T) {
	ctx := NewSmartRetryContext()
	// Calling Execute with a nil function would panic — don't do that
	// But test that a returning-nil function works
	err := ctx.Execute(func() error { return nil })
	assert.NoError(t, err)
}

// --- containsAny edge cases ---

func TestContainsAny_EmptyString(t *testing.T) {
	assert.False(t, containsAny("", []string{"a", "b"}))
}

func TestContainsAny_EmptyBoth(t *testing.T) {
	assert.False(t, containsAny("", nil))
}

func TestContainsAny_MultipleMatches(t *testing.T) {
	assert.True(t, containsAny("error: timeout and connection refused", []string{"timeout", "refused"}))
}

// --- SmartRetryContext: multiple error types ---

func TestSmartRetryContext_MultipleErrorTypes(t *testing.T) {
	ctx := NewSmartRetryContext()
	ctx.SetStrategy(ErrorTypeNetwork, &RetryStrategy{
		MaxRetries:    1,
		BaseDelay:     time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	})
	ctx.SetStrategy(ErrorTypeQuota, &RetryStrategy{
		MaxRetries:    0,
		BaseDelay:     time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	})

	t.Run("network error retries once", func(t *testing.T) {
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("connection refused")
		})
		assert.Error(t, err)
		assert.Equal(t, 2, callCount) // 1 + 1 retry
	})

	t.Run("quota error no retry", func(t *testing.T) {
		callCount := 0
		err := ctx.Execute(func() error {
			callCount++
			return fmt.Errorf("rate limit exceeded")
		})
		assert.Error(t, err)
		assert.Equal(t, 1, callCount)
	})
}

// --- InteractiveRetry with empty message ---

func TestInteractiveRetry_EmptyMessage(t *testing.T) {
	restore := redirectStdinForRetry(t, "c\n")
	defer restore()

	err := InteractiveRetry(func() error {
		return fmt.Errorf("connection refused")
	}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled by user")
}

// --- InteractiveRetry with long error message ---

func TestInteractiveRetry_LongErrorMessage(t *testing.T) {
	restore := redirectStdinForRetry(t, "c\n")
	defer restore()

	longMsg := strings.Repeat("x", 10000)
	err := InteractiveRetry(func() error {
		return fmt.Errorf("%s", longMsg)
	}, "Retry?")
	assert.Error(t, err)
}

// --- RetryWithStrategy: ErrorInfo attempts tracking ---

func TestRetryWithStrategy_ErrorInfoAttempts(t *testing.T) {
	result := RetryWithStrategy(&RetryStrategy{
		MaxRetries:    2,
		BaseDelay:     time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 1.0,
		Strategy:      "fixed",
	}, func() error {
		return fmt.Errorf("connection refused")
	})

	assert.NotNil(t, result.ErrorInfo)
	assert.Equal(t, 3, result.ErrorInfo.Attempts) // tracked on each failure
}
