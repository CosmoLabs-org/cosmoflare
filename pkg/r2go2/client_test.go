package r2go2

import (
	"testing"
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
	WithBucket("my-bucket")(cfg)
	WithCacheControl(false)(cfg)
	WithRegion("us-east-1")(cfg)
	WithEndpoint("https://custom.r2.cloudflarestorage.com")(cfg)
	WithCredentials("ak", "sk")(cfg)
	WithDryRun(true)(cfg)
	WithAuditLog("/tmp/audit.log")(cfg)

	if cfg.profile != "my-profile" {
		t.Errorf("expected profile=my-profile, got %s", cfg.profile)
	}
	if cfg.bucket != "my-bucket" {
		t.Errorf("expected bucket=my-bucket, got %s", cfg.bucket)
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
	if !cfg.dryRun {
		t.Error("expected dryRun=true")
	}
	if cfg.auditLog != "/tmp/audit.log" {
		t.Errorf("unexpected auditLog: %s", cfg.auditLog)
	}
}
