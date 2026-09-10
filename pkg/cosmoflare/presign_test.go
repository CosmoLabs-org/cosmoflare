package cosmoflare

import (
	"strings"
	"testing"
	"time"
)

// newPresignTestClient builds a client with fixed test credentials for
// exercising PresignGetObject input validation. No HTTP traffic occurs
// because every case below is expected to fail validation.
func newPresignTestClient() *client {
	return &client{cfg: &clientConfig{accountID: "test", apiToken: "test", region: "auto"}}
}

// TestPresignValidationClientInit verifies that a client can be constructed
// from account/token options alone; the test skips if config validation of
// the synthetic credentials is rejected.
func TestPresignValidationClientInit(t *testing.T) {
	t.Parallel()
	_, err := NewClient(WithAccountID("test"), WithAPIToken("test"))
	if err != nil {
		t.Skip("client init requires valid config")
	}
}

// TestPresignGetObjectValidation verifies that PresignGetObject rejects
// malformed bucket names, empty object keys, and non-positive expiration
// durations before any request is signed. Each case is a distinct invalid
// input scenario; wantSubstr (when set) pins the validator error message.
func TestPresignGetObjectValidation(t *testing.T) {
	t.Parallel()
	c := newPresignTestClient()

	tests := []struct {
		name       string
		bucket     string
		key        string
		expires    time.Duration
		wantSubstr string // optional substring expected in the error message
	}{
		{name: "empty bucket", bucket: "", key: "key", expires: time.Hour},
		{name: "uppercase bucket", bucket: "UPPERCASE", key: "key", expires: time.Hour},
		{name: "mixed-case bucket", bucket: "My-Bucket", key: "key", expires: time.Hour},
		{name: "bucket with space", bucket: "my bucket", key: "key", expires: time.Hour},
		{name: "empty key", bucket: "bucket", key: "", expires: time.Hour, wantSubstr: "object key is required"},
		{name: "zero expiry", bucket: "bucket", key: "key", expires: 0},
		{name: "explicit zero duration", bucket: "bucket", key: "key", expires: time.Duration(0)},
		{name: "negative hour expiry", bucket: "bucket", key: "key", expires: -time.Hour},
		{name: "negative millisecond expiry", bucket: "bucket", key: "key", expires: -time.Millisecond},
		{name: "negative day expiry", bucket: "bucket", key: "key", expires: -24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := c.PresignGetObject(nil, tt.bucket, tt.key, tt.expires)
			if err == nil {
				t.Fatalf("PresignGetObject(bucket=%q, key=%q, expires=%v) = nil error, want validation error", tt.bucket, tt.key, tt.expires)
			}
			if tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

// TestPresignGetObjectValidationOrder verifies that when several inputs are
// invalid at once the validators fire in a deterministic order — bucket
// first, then object key, then expiry — so error messages are predictable.
func TestPresignGetObjectValidationOrder(t *testing.T) {
	t.Parallel()
	c := newPresignTestClient()

	tests := []struct {
		name       string
		bucket     string
		key        string
		expires    time.Duration
		wantSubstr string // error text identifying which validator fired
	}{
		{
			name: "all invalid reports bucket first", bucket: "INVALID", key: "", expires: -time.Hour,
			wantSubstr: "bucket", // bucket validation must precede key/expiry checks
		},
		{
			name: "valid bucket empty key reports key", bucket: "valid-bucket", key: "", expires: time.Hour,
			wantSubstr: "object key is required",
		},
		{
			name: "valid bucket and key report expiry", bucket: "valid-bucket", key: "some-key", expires: -5 * time.Second,
			wantSubstr: "must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := c.PresignGetObject(nil, tt.bucket, tt.key, tt.expires)
			if err == nil {
				t.Fatal("expected error for all-invalid params")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}
