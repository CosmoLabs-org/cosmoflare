package migration

import (
	"testing"
	"time"
)

func TestTransferResultSuccess(t *testing.T) {
	r := transferResult{Key: "a.jpg", Size: 1024, Err: nil}
	if r.Err != nil {
		t.Error("expected no error")
	}
	if r.Key != "a.jpg" {
		t.Errorf("Key = %q, want %q", r.Key, "a.jpg")
	}
}

func TestRetryBackoff(t *testing.T) {
	delays := []time.Duration{1 * time.Second, 4 * time.Second, 16 * time.Second}
	for i, expected := range delays {
		got := retryDelay(i)
		if got != expected {
			t.Errorf("retryDelay(%d) = %v, want %v", i, got, expected)
		}
	}
}

func TestTransferTimeout(t *testing.T) {
	small := transferTimeout(1024)
	if small < 5*time.Minute {
		t.Errorf("small object timeout should be at least 5m, got %v", small)
	}
	large := transferTimeout(10 * 1024 * 1024 * 1024)
	if large < 10*time.Minute {
		t.Errorf("10GB object should have timeout > 10m, got %v", large)
	}
}

func TestShouldUseMultipart(t *testing.T) {
	if shouldUseMultipart(50 * 1024 * 1024) {
		t.Error("50MB should not use multipart")
	}
	if !shouldUseMultipart(200 * 1024 * 1024) {
		t.Error("200MB should use multipart")
	}
}
