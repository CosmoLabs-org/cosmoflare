package migration

import (
	"fmt"
	"testing"
	"time"
)

// TestTransferResultSuccess verifies a successful transferResult carries the
// object key with a nil error, as the result collector expects.
func TestTransferResultSuccess(t *testing.T) {
	t.Parallel()
	r := transferResult{Key: "a.jpg", Size: 1024, Err: nil}
	if r.Err != nil {
		t.Error("expected no error")
	}
	if r.Key != "a.jpg" {
		t.Errorf("Key = %q, want %q", r.Key, "a.jpg")
	}
}

// TestRetryBackoff verifies retryDelay grows 4x per retry attempt starting
// from one second (1s, 4s, 16s).
func TestRetryBackoff(t *testing.T) {
	t.Parallel()
	delays := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for i, expected := range delays {
		t.Run(fmt.Sprintf("attempt %d waits %v", i, expected), func(t *testing.T) {
			t.Parallel()
			if got := retryDelay(i); got != expected {
				t.Errorf("retryDelay(%d) = %v, want %v", i, got, expected)
			}
		})
	}
}

// TestTransferTimeout verifies transferTimeout enforces a five minute floor
// for small objects and scales past it for multi-gigabyte objects.
func TestTransferTimeout(t *testing.T) {
	t.Parallel()

	t.Run("small objects floor at 5 minutes", func(t *testing.T) {
		t.Parallel()
		small := transferTimeout(1024)
		if small < 5*time.Minute {
			t.Errorf("small object timeout should be at least 5m, got %v", small)
		}
	})
	t.Run("10GB object scales past 10 minutes", func(t *testing.T) {
		t.Parallel()
		large := transferTimeout(10 * 1024 * 1024 * 1024)
		if large < 10*time.Minute {
			t.Errorf("10GB object should have timeout > 10m, got %v", large)
		}
	})
}

// TestShouldUseMultipart verifies the multipart threshold: uploads below
// 100MB use the simple path, larger ones switch to multipart.
func TestShouldUseMultipart(t *testing.T) {
	t.Parallel()

	t.Run("50MB uses the simple upload path", func(t *testing.T) {
		t.Parallel()
		if shouldUseMultipart(50 * 1024 * 1024) {
			t.Error("50MB should not use multipart")
		}
	})
	t.Run("200MB uses multipart", func(t *testing.T) {
		t.Parallel()
		if !shouldUseMultipart(200 * 1024 * 1024) {
			t.Error("200MB should use multipart")
		}
	})
}
