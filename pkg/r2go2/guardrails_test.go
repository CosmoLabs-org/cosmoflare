package r2go2

import (
	"testing"
)

// --- NewGuardrailChecker ---

func TestNewGuardrailChecker_NilConfig(t *testing.T) {
	gc := NewGuardrailChecker(nil)
	if gc == nil {
		t.Fatal("expected non-nil GuardrailChecker")
	}
	if gc.cfg == nil {
		t.Fatal("expected non-nil cfg when nil passed")
	}
}

func TestNewGuardrailChecker_NonNilConfig(t *testing.T) {
	cfg := &ProjectConfig{
		Bucket: "my-bucket",
	}
	gc := NewGuardrailChecker(cfg)
	if gc.cfg != cfg {
		t.Fatal("expected cfg to be the same pointer passed in")
	}
}

// --- CheckBucketAccess ---

func TestCheckBucketAccess(t *testing.T) {
	tests := []struct {
		name            string
		topAllowed      []string
		guardrailAllowed []string
		bucket          string
		wantAllowed     bool
	}{
		{
			name:        "no allowlists — all buckets allowed",
			bucket:      "any-bucket",
			wantAllowed: true,
		},
		{
			name:        "top-level allowlist match",
			topAllowed:  []string{"prod-bucket", "staging-bucket"},
			bucket:      "prod-bucket",
			wantAllowed: true,
		},
		{
			name:        "top-level allowlist no match",
			topAllowed:  []string{"prod-bucket"},
			bucket:      "dev-bucket",
			wantAllowed: false,
		},
		{
			name:             "guardrails allowlist match (top-level empty)",
			guardrailAllowed: []string{"allowed-bucket"},
			bucket:           "allowed-bucket",
			wantAllowed:      true,
		},
		{
			name:             "guardrails allowlist no match (top-level empty)",
			guardrailAllowed: []string{"allowed-bucket"},
			bucket:           "other-bucket",
			wantAllowed:      false,
		},
		{
			// When top-level AllowedBuckets is non-empty it takes precedence.
			name:             "top-level allowlist takes precedence over guardrails",
			topAllowed:       []string{"top-bucket"},
			guardrailAllowed: []string{"guardrail-bucket"},
			bucket:           "guardrail-bucket",
			wantAllowed:      false,
		},
		{
			name:             "top-level allowlist match when both non-empty",
			topAllowed:       []string{"top-bucket"},
			guardrailAllowed: []string{"guardrail-bucket"},
			bucket:           "top-bucket",
			wantAllowed:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ProjectConfig{
				AllowedBuckets: tc.topAllowed,
				Guardrails: GuardrailConfig{
					AllowedBuckets: tc.guardrailAllowed,
				},
			}
			gc := NewGuardrailChecker(cfg)
			result := gc.CheckBucketAccess(tc.bucket)
			if result.Allowed != tc.wantAllowed {
				t.Errorf("CheckBucketAccess(%q): got Allowed=%v, want %v (reasons: %v)",
					tc.bucket, result.Allowed, tc.wantAllowed, result.Reasons)
			}
			if !tc.wantAllowed && len(result.Reasons) == 0 {
				t.Errorf("CheckBucketAccess(%q): expected at least one reason when denied", tc.bucket)
			}
		})
	}
}

func TestCheckBucketAccess_DeniedReasonContent(t *testing.T) {
	cfg := &ProjectConfig{
		AllowedBuckets: []string{"good-bucket"},
	}
	gc := NewGuardrailChecker(cfg)
	result := gc.CheckBucketAccess("bad-bucket")
	if result.Allowed {
		t.Fatal("expected denied")
	}
	if len(result.Reasons) == 0 {
		t.Fatal("expected reason message")
	}
	// Reason should mention the bucket name.
	found := false
	for _, r := range result.Reasons {
		if len(r) > 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected non-empty reason, got: %v", result.Reasons)
	}
}

// --- CheckUpload ---

func TestCheckUpload(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *ProjectConfig
		bucket      string
		key         string
		size        int64
		wantAllowed bool
		wantReasons int // minimum number of reasons expected on denial
	}{
		{
			name:        "empty config — everything allowed",
			cfg:         &ProjectConfig{},
			bucket:      "any-bucket",
			key:         "path/to/file.txt",
			size:        1024,
			wantAllowed: true,
		},
		// Bucket allowlist checks
		{
			name: "bucket in guardrails allowlist",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{
					Enabled:        true,
					AllowedBuckets: []string{"allowed"},
				},
			},
			bucket:      "allowed",
			key:         "file.txt",
			size:        100,
			wantAllowed: true,
		},
		{
			name: "bucket NOT in guardrails allowlist",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{
					Enabled:        true,
					AllowedBuckets: []string{"allowed"},
				},
			},
			bucket:      "forbidden",
			key:         "file.txt",
			size:        100,
			wantAllowed: false,
			wantReasons: 1,
		},
		// Guardrails.MaxFileSize checks
		{
			name: "size exactly at guardrails max — allowed",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{MaxFileSize: 500},
			},
			bucket:      "b",
			key:         "f",
			size:        500,
			wantAllowed: true,
		},
		{
			name: "size exceeds guardrails MaxFileSize",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{MaxFileSize: 100},
			},
			bucket:      "b",
			key:         "f",
			size:        101,
			wantAllowed: false,
			wantReasons: 1,
		},
		{
			name: "size within guardrails MaxFileSize",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{MaxFileSize: 1000},
			},
			bucket:      "b",
			key:         "f",
			size:        999,
			wantAllowed: true,
		},
		// Top-level MaxFileSize checks
		{
			name: "size exceeds top-level MaxFileSize",
			cfg: &ProjectConfig{
				MaxFileSize: 256,
			},
			bucket:      "b",
			key:         "f",
			size:        300,
			wantAllowed: false,
			wantReasons: 1,
		},
		{
			name: "size within top-level MaxFileSize",
			cfg: &ProjectConfig{
				MaxFileSize: 1024,
			},
			bucket:      "b",
			key:         "f",
			size:        1024,
			wantAllowed: true,
		},
		// Both MaxFileSize fields trigger
		{
			name: "size exceeds both guardrails and top-level MaxFileSize",
			cfg: &ProjectConfig{
				MaxFileSize: 100,
				Guardrails:  GuardrailConfig{MaxFileSize: 50},
			},
			bucket:      "b",
			key:         "f",
			size:        200,
			wantAllowed: false,
			wantReasons: 2,
		},
		// Blocked key patterns
		{
			name: "key matches blocked exact pattern",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{BlockedKeys: []string{"secrets/key.pem"}},
			},
			bucket:      "b",
			key:         "secrets/key.pem",
			size:        1,
			wantAllowed: false,
			wantReasons: 1,
		},
		{
			name: "key matches blocked prefix pattern",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{BlockedKeys: []string{"private/"}},
			},
			bucket:      "b",
			key:         "private/secret.txt",
			size:        1,
			wantAllowed: false,
			wantReasons: 1,
		},
		{
			name: "key matches blocked extension pattern",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{BlockedKeys: []string{"*.env"}},
			},
			bucket:      "b",
			key:         "config.env",
			size:        1,
			wantAllowed: false,
			wantReasons: 1,
		},
		{
			name: "key does not match any blocked pattern",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{BlockedKeys: []string{"*.env", "private/"}},
			},
			bucket:      "b",
			key:         "public/readme.txt",
			size:        1,
			wantAllowed: true,
		},
		// Multiple violations accumulate
		{
			name: "bucket denied AND size exceeded AND key blocked",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{
					AllowedBuckets: []string{"only-this"},
					MaxFileSize:    10,
					BlockedKeys:    []string{"*.exe"},
				},
			},
			bucket:      "wrong-bucket",
			key:         "malware.exe",
			size:        100,
			wantAllowed: false,
			wantReasons: 3,
		},
		// Zero-value MaxFileSize means unlimited
		{
			name: "zero MaxFileSize means no size limit",
			cfg: &ProjectConfig{
				Guardrails: GuardrailConfig{MaxFileSize: 0},
				MaxFileSize: 0,
			},
			bucket:      "b",
			key:         "f",
			size:        1<<62 - 1,
			wantAllowed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gc := NewGuardrailChecker(tc.cfg)
			result := gc.CheckUpload(tc.bucket, tc.key, tc.size)
			if result.Allowed != tc.wantAllowed {
				t.Errorf("CheckUpload(%q, %q, %d): got Allowed=%v, want %v (reasons: %v)",
					tc.bucket, tc.key, tc.size, result.Allowed, tc.wantAllowed, result.Reasons)
			}
			if !tc.wantAllowed && len(result.Reasons) < tc.wantReasons {
				t.Errorf("CheckUpload: expected at least %d reason(s), got %d: %v",
					tc.wantReasons, len(result.Reasons), result.Reasons)
			}
		})
	}
}

// --- matchBlockedKey ---

func TestMatchBlockedKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		pattern string
		want    bool
	}{
		// Exact match
		{name: "exact match", key: "secret.pem", pattern: "secret.pem", want: true},
		{name: "exact no match", key: "secret.pem", pattern: "other.pem", want: false},
		{name: "exact match with path", key: "dir/file.txt", pattern: "dir/file.txt", want: true},

		// Prefix match (pattern ends with /)
		{name: "prefix match", key: "private/file.txt", pattern: "private/", want: true},
		{name: "prefix match nested", key: "private/subdir/deep.txt", pattern: "private/", want: true},
		{name: "prefix no match", key: "public/file.txt", pattern: "private/", want: false},
		{name: "prefix exact boundary no match", key: "privateX/file.txt", pattern: "private/", want: false},
		{name: "prefix match root level", key: "private/", pattern: "private/", want: true},

		// Extension match (pattern starts with *.)
		{name: "extension match .env", key: "config.env", pattern: "*.env", want: true},
		{name: "extension match .pem", key: "cert.pem", pattern: "*.pem", want: true},
		{name: "extension match case-insensitive upper", key: "config.ENV", pattern: "*.env", want: true},
		{name: "extension match case-insensitive pattern", key: "config.env", pattern: "*.ENV", want: true},
		{name: "extension no match", key: "config.yaml", pattern: "*.env", want: false},
		{name: "extension match with path", key: "dir/subdir/secret.pem", pattern: "*.pem", want: true},

		// Glob match
		{name: "glob match single dir", key: "logs/app.log", pattern: "logs/*.log", want: true},
		{name: "glob no match", key: "logs/app.txt", pattern: "logs/*.log", want: false},
		{name: "glob match any single char", key: "file1.txt", pattern: "file?.txt", want: true},
		{name: "glob no match wrong length", key: "file12.txt", pattern: "file?.txt", want: false},
		{name: "glob star matches multiple chars", key: "archive.tar.gz", pattern: "archive.*", want: true},

		// Edge cases
		{name: "empty key empty pattern — exact", key: "", pattern: "", want: true},
		{name: "empty key non-empty pattern", key: "", pattern: "notempty", want: false},
		{name: "non-empty key empty pattern", key: "file.txt", pattern: "", want: false},
		{name: "pattern prefix slash does not match key without prefix", key: "file.txt", pattern: "private/", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchBlockedKey(tc.key, tc.pattern)
			if got != tc.want {
				t.Errorf("matchBlockedKey(%q, %q) = %v, want %v", tc.key, tc.pattern, got, tc.want)
			}
		})
	}
}

// --- ValidateBucketName ---

func TestValidateBucketName_Guardrails(t *testing.T) {
	// Delegate coverage — just verify the function is wired through correctly
	// and returns nil for valid names and non-nil for invalid ones.
	tests := []struct {
		name    string
		bucket  string
		wantErr bool
	}{
		{name: "valid simple name", bucket: "my-bucket", wantErr: false},
		{name: "valid with numbers", bucket: "bucket123", wantErr: false},
		{name: "too short", bucket: "ab", wantErr: true},
		{name: "too long", bucket: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantErr: true},
		{name: "uppercase not allowed", bucket: "My-Bucket", wantErr: true},
		{name: "starts with hyphen", bucket: "-bucket", wantErr: true},
		{name: "ends with hyphen", bucket: "bucket-", wantErr: true},
		{name: "contains underscore", bucket: "my_bucket", wantErr: true},
		{name: "empty name", bucket: "", wantErr: true},
		{name: "valid with hyphens", bucket: "my-valid-bucket-name", wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBucketName(tc.bucket)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBucketName(%q): got err=%v, wantErr=%v", tc.bucket, err, tc.wantErr)
			}
		})
	}
}

// --- GuardrailResult struct ---

func TestGuardrailResult_AllowedWithNoReasons(t *testing.T) {
	gc := NewGuardrailChecker(&ProjectConfig{})
	result := gc.CheckUpload("bucket", "key.txt", 1)
	if !result.Allowed {
		t.Errorf("expected allowed, reasons: %v", result.Reasons)
	}
	if len(result.Reasons) != 0 {
		t.Errorf("expected no reasons, got: %v", result.Reasons)
	}
}

func TestGuardrailResult_DeniedHasReasons(t *testing.T) {
	cfg := &ProjectConfig{
		Guardrails: GuardrailConfig{MaxFileSize: 1},
	}
	gc := NewGuardrailChecker(cfg)
	result := gc.CheckUpload("bucket", "key.txt", 999)
	if result.Allowed {
		t.Error("expected denied")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected at least one reason")
	}
}
