package cmd

import (
	"context"
	"strings"
	"testing"
)

// TestGetBucketDomainService_ReturnsServiceWithCreds verifies the service
// factory wires the global credentials into a non-nil service — even with
// empty credentials, since the constructor never validates them.
func TestGetBucketDomainService_ReturnsServiceWithCreds(t *testing.T) {
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() { AccountID, APIToken = savedAccount, savedToken })

	AccountID, APIToken = "acct", "token"
	if svc := getBucketDomainService(); svc == nil {
		t.Fatal("expected non-nil bucket domain service with credentials set")
	}

	AccountID, APIToken = "", ""
	if svc := getBucketDomainService(); svc == nil {
		t.Fatal("expected non-nil bucket domain service even with empty credentials")
	}
}

// TestResolveZoneID_FailsWithoutCredentials verifies zone resolution aborts
// with a wrapped service error before any network call when credentials are
// empty.
func TestResolveZoneID_FailsWithoutCredentials(t *testing.T) {
	savedAccount, savedToken := AccountID, APIToken
	t.Cleanup(func() { AccountID, APIToken = savedAccount, savedToken })
	AccountID, APIToken = "", ""

	_, err := resolveZoneID(context.Background(), "cdn.example.com")
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}
