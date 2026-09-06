/*
Package ux provides error handling and recovery mechanisms for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package ux

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// ErrorType defines different categories of errors
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypePermission
	ErrorTypeStorage
	ErrorTypeAuthentication
	ErrorTypeQuota
	ErrorTypeValidation
	ErrorTypeTimeout
	ErrorTypeUserCancel
	ErrorTypeFileSystem
	ErrorTypeUnknown
)

// RetryStrategy defines different retry strategies
type RetryStrategy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
	Strategy      string // "exponential", "linear", "fixed"
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Type         ErrorType
	Message      string
	Original     error
	Retryable    bool
	Suggestion   string
	ErrorCode    string
	Context      map[string]interface{}
	Attempts     int
	LastAttempt  time.Time
	NextAttempt  time.Time
}

// RetryResult contains the result of a retry attempt
type RetryResult struct {
	Success       bool
	Attempts      int
	TotalTime     time.Duration
	StartTime     time.Time
	LastError     error
	ErrorInfo     *ErrorInfo
	Retryed       bool
	BackoffDelays []time.Duration
}

// DefaultRetryStrategy returns a default retry strategy
func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		Strategy:      "exponential",
	}
}

// NetworkRetryStrategy returns a strategy optimized for network errors
func NetworkRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:    5,
		BaseDelay:     2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		Strategy:      "exponential",
	}
}

// FileSystemRetryStrategy returns a strategy optimized for file system errors
func FileSystemRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:    2,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        false,
		Strategy:      "linear",
	}
}

// ClassifyError classifies an error into different categories
func ClassifyError(err error) *ErrorInfo {
	if err == nil {
		return &ErrorInfo{
			Type:      ErrorTypeUnknown,
			Message:   "No error",
			Retryable: false,
		}
	}

	errStr := strings.ToLower(err.Error())
	info := &ErrorInfo{
		Type:       ErrorTypeUnknown,
		Message:    err.Error(),
		Original:   err,
		Retryable:  false,
		Suggestion: "",
		ErrorCode:  "",
		Context:    make(map[string]interface{}),
	}

	// Network errors
	if containsAny(errStr, []string{
		"connection refused", "connection reset", "network unreachable",
		"timeout", "deadline exceeded", "no such host",
		"temporary failure", "connection timed out", "read: connection reset",
		"write: broken pipe", "network is down", "name resolution failed",
	}) {
		info.Type = ErrorTypeNetwork
		info.Retryable = true
		info.Suggestion = "Check network connection and try again"
		info.ErrorCode = "NETWORK_ERROR"
	}

	// Permission errors
	if containsAny(errStr, []string{
		"permission denied", "access denied", "operation not permitted",
		"insufficient privileges", "unauthorized", "forbidden",
	}) {
		info.Type = ErrorTypePermission
		info.Retryable = false
		info.Suggestion = "Check file permissions and user privileges"
		info.ErrorCode = "PERMISSION_ERROR"
	}

	// Storage errors
	if containsAny(errStr, []string{
		"no space left", "disk full", "storage quota exceeded",
		"insufficient disk space", "volume full", "out of disk space",
	}) {
		info.Type = ErrorTypeStorage
		info.Retryable = false
		info.Suggestion = "Free up disk space and try again"
		info.ErrorCode = "STORAGE_ERROR"
	}

	// Authentication errors
	if containsAny(errStr, []string{
		"authentication failed", "invalid credentials", "unauthorized",
		"access token expired", "invalid api key", "authentication required",
	}) {
		info.Type = ErrorTypeAuthentication
		info.Retryable = false
		info.Suggestion = "Check authentication credentials and API tokens"
		info.ErrorCode = "AUTH_ERROR"
	}

	// Quota errors
	if containsAny(errStr, []string{
		"quota exceeded", "rate limit exceeded", "too many requests",
		"api rate limit", "request limit exceeded", "quota limit",
	}) {
		info.Type = ErrorTypeQuota
		info.Retryable = true
		info.Suggestion = "Wait for quota reset or upgrade your plan"
		info.ErrorCode = "QUOTA_ERROR"
	}

	// Timeout errors
	if containsAny(errStr, []string{
		"timeout", "deadline exceeded", "operation timed out",
		"context deadline exceeded", "request timeout",
	}) {
		info.Type = ErrorTypeTimeout
		info.Retryable = true
		info.Suggestion = "Increase timeout or try again later"
		info.ErrorCode = "TIMEOUT_ERROR"
	}

	// File system errors
	if containsAny(errStr, []string{
		"file not found", "no such file or directory", "file exists",
		"directory not empty", "not a directory", "is a directory",
		"file already exists", "device or resource busy",
	}) {
		info.Type = ErrorTypeFileSystem
		info.Retryable = false
		info.Suggestion = "Check file paths and permissions"
		info.ErrorCode = "FILESYSTEM_ERROR"
	}

	return info
}

// RetryWithStrategy retries an operation using the specified strategy
func RetryWithStrategy(strategy *RetryStrategy, operation func() error) *RetryResult {
	if strategy == nil {
		strategy = DefaultRetryStrategy()
	}

	startTime := time.Now()
	result := &RetryResult{
		Success:       false,
		Attempts:      0,
		StartTime:     startTime,
		BackoffDelays: make([]time.Duration, 0),
	}

	var lastError error
	var errorInfo *ErrorInfo

	for attempt := 0; attempt <= strategy.MaxRetries; attempt++ {
		result.Attempts++

		if attempt > 0 {
			// Calculate delay
			delay := calculateDelay(strategy, attempt-1)
			result.BackoffDelays = append(result.BackoffDelays, delay)

			// Wait before retry
			time.Sleep(delay)
		}

		// Execute operation
		err := operation()
		if err == nil {
			result.Success = true
			break
		}

		lastError = err
		errorInfo = ClassifyError(err)
		errorInfo.Attempts = result.Attempts

		// Check if we should retry
		if !errorInfo.Retryable {
			break
		}

		// Don't retry on the last attempt
		if attempt == strategy.MaxRetries {
			break
		}
	}

	result.LastError = lastError
	result.ErrorInfo = errorInfo
	result.TotalTime = time.Since(startTime)
	result.Retryed = result.Attempts > 1

	return result
}

// RetryWithBackoff retries an operation with exponential backoff
func RetryWithBackoff(maxRetries int, operation func() error) error {
	strategy := &RetryStrategy{
		MaxRetries:    maxRetries,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		Strategy:      "exponential",
	}

	result := RetryWithStrategy(strategy, operation)
	if result.Success {
		return nil
	}
	return result.LastError
}

// InteractiveRetry provides interactive retry with user confirmation
func InteractiveRetry(operation func() error, message string) error {
	attempt := 0
	for {
		attempt++
		err := operation()
		if err == nil {
			return nil
		}

		errorInfo := ClassifyError(err)

		// Show error information
		fmt.Printf("\n❌ Operation failed (attempt %d):\n", attempt)
		fmt.Printf("   Error: %s\n", err.Error())
		fmt.Printf("   Type: %s\n", errorTypeToString(errorInfo.Type))

		if errorInfo.Suggestion != "" {
			fmt.Printf("   💡 Suggestion: %s\n", errorInfo.Suggestion)
		}

		// Check if retryable
		if !errorInfo.Retryable {
			fmt.Printf("   ❌ This error is not retryable\n")
			return err
		}

		// Ask user if they want to retry
		if !ConfirmRetry(message, attempt) {
			return fmt.Errorf("operation cancelled by user after %d attempts", attempt)
		}
	}
}

// calculateDelay calculates the delay for a given attempt
func calculateDelay(strategy *RetryStrategy, attempt int) time.Duration {
	var delay time.Duration

	switch strategy.Strategy {
	case "exponential":
		delay = strategy.BaseDelay * time.Duration(math.Pow(strategy.BackoffFactor, float64(attempt)))
	case "linear":
		delay = strategy.BaseDelay * time.Duration(attempt+1)
	case "fixed":
		delay = strategy.BaseDelay
	default:
		delay = strategy.BaseDelay
	}

	// Apply maximum delay limit
	if delay > strategy.MaxDelay {
		delay = strategy.MaxDelay
	}

	// Add jitter if enabled
	if strategy.Jitter {
		jitter := rand.Float64() * 0.1 // 10% jitter
		delay = time.Duration(float64(delay) * (1 + jitter))
	}

	return delay
}

// ConfirmRetry asks the user if they want to retry
func ConfirmRetry(message string, attempt int) bool {
	fmt.Printf("\n%s (attempt %d)? [R]etry/[C]ancel: ", message, attempt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "r" || response == "retry"
}

// errorTypeToString converts ErrorType to string
func errorTypeToString(errorType ErrorType) string {
	switch errorType {
	case ErrorTypeNetwork:
		return "Network Error"
	case ErrorTypePermission:
		return "Permission Error"
	case ErrorTypeStorage:
		return "Storage Error"
	case ErrorTypeAuthentication:
		return "Authentication Error"
	case ErrorTypeQuota:
		return "Quota Error"
	case ErrorTypeValidation:
		return "Validation Error"
	case ErrorTypeTimeout:
		return "Timeout Error"
	case ErrorTypeUserCancel:
		return "User Cancelled"
	case ErrorTypeFileSystem:
		return "File System Error"
	default:
		return "Unknown Error"
	}
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// PrintRetryProgress prints retry progress information
func PrintRetryProgress(result *RetryResult) {
	if result.Attempts <= 1 {
		return
	}

	fmt.Printf("\n🔄 Retry Summary:\n")
	fmt.Printf("   Attempts: %d\n", result.Attempts)
	fmt.Printf("   Total Time: %v\n", result.TotalTime)

	if len(result.BackoffDelays) > 0 {
		fmt.Printf("   Backoff Delays: ")
		for i, delay := range result.BackoffDelays {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%v", delay)
		}
		fmt.Printf("\n")
	}

	if result.LastError != nil {
		errorInfo := ClassifyError(result.LastError)
		fmt.Printf("   Final Error: %v\n", errorInfo.Type)
	}
}

// HandleBatchErrors handles errors in batch operations
func HandleBatchErrors(errors []error, continueOnError bool) (int, int) {
	if len(errors) == 0 {
		return 0, 0
	}

	successCount := 0
	errorCount := 0

	fmt.Printf("\n📊 Error Summary:\n")

	for i, err := range errors {
		if err == nil {
			successCount++
			continue
		}

		errorCount++
		errorInfo := ClassifyError(err)

		fmt.Printf("   %d. %s: %s\n", i+1, errorTypeToString(errorInfo.Type), err.Error())

		if errorInfo.Suggestion != "" {
			fmt.Printf("      💡 %s\n", errorInfo.Suggestion)
		}

		if !continueOnError && i == 0 {
			fmt.Printf("\n❌ Batch operation stopped due to errors\n")
			return successCount, errorCount
		}
	}

	fmt.Printf("\nResults: %d successful, %d failed\n", successCount, errorCount)
	return successCount, errorCount
}

// SmartRetryContext provides context-aware retry logic
type SmartRetryContext struct {
	strategies map[ErrorType]*RetryStrategy
	context    map[string]interface{}
}

// NewSmartRetryContext creates a new smart retry context
func NewSmartRetryContext() *SmartRetryContext {
	return &SmartRetryContext{
		strategies: make(map[ErrorType]*RetryStrategy),
		context:    make(map[string]interface{}),
	}
}

// SetStrategy sets a retry strategy for a specific error type
func (src *SmartRetryContext) SetStrategy(errorType ErrorType, strategy *RetryStrategy) {
	src.strategies[errorType] = strategy
}

// SetContext sets context information
func (src *SmartRetryContext) SetContext(key string, value interface{}) {
	src.context[key] = value
}

// Execute executes an operation with smart retry logic
func (src *SmartRetryContext) Execute(operation func() error) error {
	var lastError error

	for attempt := 0; attempt < 10; attempt++ { // Safety limit
		err := operation()
		if err == nil {
			return nil
		}

		lastError = err
		errorInfo := ClassifyError(err)

		// Get strategy for this error type
		strategy, exists := src.strategies[errorInfo.Type]
		if !exists {
			strategy = DefaultRetryStrategy()
		}

		// Check if we've exceeded retries for this strategy
		if attempt >= strategy.MaxRetries {
			break
		}

		// Check if error is retryable
		if !errorInfo.Retryable {
			break
		}

		// Calculate and wait
		delay := calculateDelay(strategy, attempt)
		time.Sleep(delay)
	}

	return lastError
}