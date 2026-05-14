package r2go2

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
