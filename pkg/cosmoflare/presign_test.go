package cosmoflare

import (
	"testing"
	"time"
)

func TestPresignValidationEmptyBucket(t *testing.T) {
	_, err := NewClient(WithAccountID("test"), WithAPIToken("test"))
	if err != nil {
		t.Skip("client init requires valid config")
	}
}

func TestPresignValidationEmptyKey(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "bucket", "", time.Hour)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestPresignValidationZeroExpires(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "bucket", "key", 0)
	if err == nil {
		t.Fatal("expected error for zero expires")
	}
}

func TestPresignValidationNegativeExpires(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "bucket", "key", -1*time.Hour)
	if err == nil {
		t.Fatal("expected error for negative expires")
	}
}

func TestPresignValidationInvalidBucketName(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "UPPERCASE", "key", time.Hour)
	if err == nil {
		t.Fatal("expected error for invalid bucket name")
	}
}

func TestPresignValidationExpiredDuration(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	// Very small negative duration
	_, err := c.PresignGetObject(nil, "bucket", "key", -1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for negative millisecond duration")
	}

	// Large negative duration
	_, err = c.PresignGetObject(nil, "bucket", "key", -24*time.Hour)
	if err == nil {
		t.Fatal("expected error for -24h duration")
	}
}

func TestPresignValidationZeroDurationExplicit(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "bucket", "key", time.Duration(0))
	if err == nil {
		t.Fatal("expected error for explicit zero duration")
	}
}

func TestPresignValidationEmptyBucketRejectsEmpty(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	_, err := c.PresignGetObject(nil, "", "key", time.Hour)
	if err == nil {
		t.Fatal("expected error for empty bucket name")
	}
}

func TestPresignValidationBucketWithSpecialChars(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	// Buckets with uppercase should fail validation
	_, err := c.PresignGetObject(nil, "My-Bucket", "key", time.Hour)
	if err == nil {
		t.Fatal("expected error for bucket name with uppercase chars")
	}

	// Buckets with spaces should fail
	_, err = c.PresignGetObject(nil, "my bucket", "key", time.Hour)
	if err == nil {
		t.Fatal("expected error for bucket name with spaces")
	}
}

func TestPresignValidationKeyVariants(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	// Empty key must be rejected
	_, err := c.PresignGetObject(nil, "bucket", "", time.Hour)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !containsSubstr(err.Error(), "object key is required") {
		t.Errorf("expected 'object key is required' error, got: %s", err.Error())
	}

	// Multiple empty-string variants: all should fail key validation
	for _, key := range []string{"", ""} {
		_, err := c.PresignGetObject(nil, "bucket", key, time.Hour)
		if err == nil {
			t.Fatalf("expected error for key %q", key)
		}
	}
}

func TestPresignValidationCombinations(t *testing.T) {
	cfg := &clientConfig{accountID: "test", apiToken: "test", region: "auto"}
	c := &client{cfg: cfg}

	// All invalid params: bucket validation runs first
	_, err := c.PresignGetObject(nil, "INVALID", "", -time.Hour)
	if err == nil {
		t.Fatal("expected error for all-invalid params")
	}

	// Valid bucket, empty key: key validation should trigger
	_, err = c.PresignGetObject(nil, "valid-bucket", "", time.Hour)
	if err == nil {
		t.Fatal("expected error for empty key with valid bucket")
	}
	if !containsSubstr(err.Error(), "object key is required") {
		t.Errorf("expected key validation error, got: %s", err.Error())
	}

	// Valid bucket, valid key, negative duration: duration validation should trigger
	_, err = c.PresignGetObject(nil, "valid-bucket", "some-key", -5*time.Second)
	if err == nil {
		t.Fatal("expected error for negative duration with valid bucket and key")
	}
	if !containsSubstr(err.Error(), "must be positive") {
		t.Errorf("expected duration validation error, got: %s", err.Error())
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
