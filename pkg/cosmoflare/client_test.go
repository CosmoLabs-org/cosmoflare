package cosmoflare

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestR2ErrorInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "base error",
			err:     newError("ListBuckets", "something failed", nil),
			wantMsg: "r2go2: ListBuckets: something failed",
		},
		{
			name:    "not found error",
			err:     notFound("GetObject", "my-bucket", "key.txt", nil),
			wantMsg: "resource not found",
		},
		{
			name:    "auth error",
			err:     authError("NewClient", "bad token", nil),
			wantMsg: "bad token",
		},
		{
			name:    "validation error",
			err:     validationError("CreateBucket", "name too short"),
			wantMsg: "name too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() == "" {
				t.Error("error message should not be empty")
			}
		})
	}
}

func TestR2NotFoundError(t *testing.T) {
	err := notFound("GetObject", "my-bucket", "key.txt", nil)
	if err.Bucket != "my-bucket" {
		t.Errorf("expected bucket=my-bucket, got %s", err.Bucket)
	}
	if err.Key != "key.txt" {
		t.Errorf("expected key=key.txt, got %s", err.Key)
	}
	// Verify it satisfies the error interface via the base R2Error
	var _ error = err
}

func TestR2AuthError(t *testing.T) {
	err := authError("NewClient", "bad token", nil)
	if err.Op != "NewClient" {
		t.Errorf("expected op=NewClient, got %s", err.Op)
	}
	var _ error = err
}

func TestR2QuotaError(t *testing.T) {
	err := quotaError("CreateBucket", "limit exceeded", nil)
	if err.Message != "limit exceeded" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	var _ error = err
}

func TestR2AccessDeniedError(t *testing.T) {
	err := accessDenied("DeleteBucket", "my-bucket", "forbidden", nil)
	if err.Bucket != "my-bucket" {
		t.Errorf("expected bucket=my-bucket, got %s", err.Bucket)
	}
	var _ error = err
}

func TestR2ValidationError(t *testing.T) {
	err := validationError("CreateBucket", "name too short")
	if err.Message != "name too short" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	var _ error = err
}

func TestNewClientValidation(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	_, err := NewClient()
	if err == nil {
		t.Error("expected error when no credentials provided")
	}

	_, err = NewClient(WithAccountID("test123"))
	if err == nil {
		t.Error("expected error when api token missing")
	}

	_, err = NewClient(WithAPIToken("token123"))
	if err == nil {
		t.Error("expected error when account id missing")
	}
}

func TestClientOptions(t *testing.T) {
	cfg := &clientConfig{}
	WithProfile("my-profile")(cfg)
	WithCacheControl(false)(cfg)
	WithRegion("us-east-1")(cfg)
	WithEndpoint("https://custom.r2.cloudflarestorage.com")(cfg)
	WithCredentials("ak", "sk")(cfg)
	WithTimeout(42 * time.Second)(cfg)

	if cfg.profile != "my-profile" {
		t.Errorf("expected profile=my-profile, got %s", cfg.profile)
	}
	if cfg.cacheControl != false {
		t.Error("expected cacheControl=false")
	}
	if cfg.region != "us-east-1" {
		t.Errorf("expected region=us-east-1, got %s", cfg.region)
	}
	if cfg.endpoint != "https://custom.r2.cloudflarestorage.com" {
		t.Errorf("unexpected endpoint: %s", cfg.endpoint)
	}
	if cfg.accessKey != "ak" || cfg.secretKey != "sk" {
		t.Error("credentials not set correctly")
	}
	if cfg.timeout != 42*time.Second {
		t.Errorf("unexpected timeout: %s", cfg.timeout)
	}
}

// --- BUG-029: transport wiring, profile resolution, dead options ---

// recordingTransport intercepts HTTP requests before any network access and
// replies with a canned successful Cloudflare API response. If it is invoked,
// the HTTP client it belongs to was wired into the API client under test.
type recordingTransport struct {
	called int
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.called++
	// Envelope shaped for ListR2Buckets (R2BucketListResponse expects
	// result.buckets), which is what TestConnection calls.
	body := `{"success": true, "errors": [], "messages": [], "result": {"buckets": []}}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

// newTestClient builds a client with env credentials so tests can focus on
// the options under test.
func newTestClient(t *testing.T, opts ...ClientOption) *client {
	t.Helper()
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-token-12345")

	c, err := NewClient(opts...)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return c.(*client)
}

func TestNewClientWiresTimeoutToBothTransports(t *testing.T) {
	c := newTestClient(t, WithTimeout(7*time.Second))

	if c.httpClient == nil {
		t.Fatal("client httpClient is nil")
	}
	if c.httpClient.Timeout != 7*time.Second {
		t.Errorf("httpClient.Timeout = %v, want 7s", c.httpClient.Timeout)
	}

	s3HC, ok := c.s3.Options().HTTPClient.(*http.Client)
	if !ok {
		t.Fatalf("S3 client HTTP client is %T, want *http.Client", c.s3.Options().HTTPClient)
	}
	// BUG-042: WithTimeout bounds the CONTROL plane (Cloudflare API) only.
	// A whole-request timeout on the S3 data plane kills large transfers
	// mid-body; transfer limits come from context deadlines instead.
	if s3HC.Timeout != 0 {
		t.Errorf("S3 data-plane client carries whole-request timeout %v; WithTimeout must not bound whole transfers (BUG-042)", s3HC.Timeout)
	}
}

// BUG-042: the default S3 data-plane transport must not inherit the shared
// 30s whole-request timeout — any transfer slower than 30s dies mid-body.
// Context deadlines and SDK retries govern transfers instead.
func TestNewClientDefaultS3TransportHasNoRequestTimeout(t *testing.T) {
	c := newTestClient(t)

	if c.httpClient.Timeout != 30*time.Second {
		t.Fatalf("control-plane timeout = %v, want default 30s", c.httpClient.Timeout)
	}

	s3HC, ok := c.s3.Options().HTTPClient.(*http.Client)
	if !ok {
		t.Fatalf("S3 client HTTP client is %T, want *http.Client", c.s3.Options().HTTPClient)
	}
	if s3HC.Timeout != 0 {
		t.Errorf("default S3 data-plane client carries whole-request timeout %v — large transfers die mid-body (BUG-042)", s3HC.Timeout)
	}
}

func TestNewClientPassesHTTPClientToCloudflareAPI(t *testing.T) {
	rt := &recordingTransport{}
	hc := &http.Client{Transport: rt, Timeout: 10 * time.Second}
	c := newTestClient(t, WithHTTPClient(hc))

	if c.httpClient != hc {
		t.Error("custom HTTP client not stored on client")
	}

	s3HC, ok := c.s3.Options().HTTPClient.(*http.Client)
	if !ok || s3HC != hc {
		t.Errorf("S3 client HTTP client = %T, want the custom client passed via WithHTTPClient", c.s3.Options().HTTPClient)
	}

	// The Cloudflare API client must issue requests through the custom
	// client. The recording transport answers without any network access,
	// so TestConnection only succeeds when the transport was wired in.
	if err := c.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
	if rt.called == 0 {
		t.Error("custom transport was never used; WithHTTPClient is not wired into the Cloudflare API client")
	}
}

const testMachineConfigYAML = `profiles:
  prod:
    account_id: acc-prod
    api_token: tok-prod
  staging:
    account_id: acc-staging
    api_token: tok-staging
current: prod
`

// writeTestMachineConfig points HOME at a temp dir holding a machine config
// fixture at ~/.r2go2/config.yaml.
func writeTestMachineConfig(t *testing.T, yaml string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".r2go2")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0600); err != nil {
		t.Fatalf("failed to write machine config: %v", err)
	}
}

func TestNewClientProfileResolution(t *testing.T) {
	t.Run("profile supplies credentials when options and env are absent", func(t *testing.T) {
		writeTestMachineConfig(t, testMachineConfigYAML)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
		t.Setenv("CLOUDFLARE_API_TOKEN", "")

		c, err := NewClient(WithProfile("prod"))
		if err != nil {
			t.Fatalf("NewClient(WithProfile(prod)) failed: %v", err)
		}
		if got := c.AccountID(); got != "acc-prod" {
			t.Errorf("AccountID() = %q, want %q (from profile)", got, "acc-prod")
		}
	})

	t.Run("env overrides profile", func(t *testing.T) {
		writeTestMachineConfig(t, testMachineConfigYAML)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-account")
		t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")

		c, err := NewClient(WithProfile("prod"))
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if got := c.AccountID(); got != "env-account" {
			t.Errorf("AccountID() = %q, want %q (env must win over profile)", got, "env-account")
		}
	})

	t.Run("options override env and profile", func(t *testing.T) {
		writeTestMachineConfig(t, testMachineConfigYAML)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-account")
		t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")

		c, err := NewClient(WithProfile("prod"), WithAccountID("opt-account"), WithAPIToken("opt-token"))
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if got := c.AccountID(); got != "opt-account" {
			t.Errorf("AccountID() = %q, want %q (options must win)", got, "opt-account")
		}
	})

	t.Run("unknown profile errors", func(t *testing.T) {
		writeTestMachineConfig(t, testMachineConfigYAML)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
		t.Setenv("CLOUDFLARE_API_TOKEN", "")

		_, err := NewClient(WithProfile("missing"))
		if err == nil {
			t.Fatal("expected error for unknown profile")
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("error should name the missing profile, got: %v", err)
		}
	})
}

// TestClientConfigDeadFieldsRemoved pins the removal of the silent no-op
// config fields (BUG-029). WithBucket/WithAuditLog/WithDryRun were deleted
// together with these fields; their absence is enforced at compile time.
func TestClientConfigDeadFieldsRemoved(t *testing.T) {
	typ := reflect.TypeOf(clientConfig{})
	for _, name := range []string{"bucket", "auditLog", "dryRun"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("clientConfig still declares dead field %q (set by a deleted no-op option)", name)
		}
	}
}
