package cosmoflare

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_RequiredFields_Missing(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result when name and type are missing")
	}
	if len(result.Errors) < 2 {
		t.Fatalf("expected at least 2 errors, got %d", len(result.Errors))
	}

	foundName, foundType := false, false
	for _, e := range result.Errors {
		if e.Field == "name" {
			foundName = true
		}
		if e.Field == "type" {
			foundType = true
		}
	}
	if !foundName {
		t.Error("expected error for missing 'name' field")
	}
	if !foundType {
		t.Error("expected error for missing 'type' field")
	}
}

func TestValidate_RequiredFields_Present(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "my-project",
		Type: "application",
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid result, got %d errors: %v", len(result.Errors), result.Errors)
	}
}

func TestValidate_WorkerScriptExists(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "worker.js")
	if err := os.WriteFile(scriptPath, []byte("// worker"), 0644); err != nil {
		t.Fatal(err)
	}

	v := NewConfigValidator(tmpDir)
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		Workers: map[string]WorkerConfig{
			"my-worker": {Script: "worker.js"},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidate_WorkerScriptMissing(t *testing.T) {
	tmpDir := t.TempDir()

	v := NewConfigValidator(tmpDir)
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		Workers: map[string]WorkerConfig{
			"my-worker": {Script: "nonexistent.js"},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result when script does not exist")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Field != "workers.my-worker.script" {
		t.Errorf("expected field 'workers.my-worker.script', got %q", result.Errors[0].Field)
	}
}

func TestValidate_WorkerScriptEmpty(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		Workers: map[string]WorkerConfig{
			"my-worker": {Script: ""},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result when script is empty")
	}
}

func TestValidate_R2BucketNameValid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		R2: R2Config{
			Buckets: []R2BucketConfig{
				{Name: "my-bucket"},
				{Name: "assets-prod-2024"},
			},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_R2BucketNameInvalid(t *testing.T) {
	tests := []struct {
		name    string
		bucket  string
		wantMsg string
	}{
		{"too short", "ab", "must be 3-63 characters"},
		{"has dots", "my.bucket.name", "must not contain dots"},
		{"uppercase", "MyBucket", "must be lowercase"},
		{"starts with hyphen", "-bucket", "invalid"},
		{"empty name", "", "bucket name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewConfigValidator("")
			cfg := &CosmoflareConfig{
				Name: "test",
				Type: "app",
				R2: R2Config{
					Buckets: []R2BucketConfig{{Name: tt.bucket}},
				},
			}

			result := v.Validate(cfg)

			if result.Valid {
				t.Fatal("expected invalid result")
			}
		})
	}
}

func TestValidate_R2BucketDuplicate(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		R2: R2Config{
			Buckets: []R2BucketConfig{
				{Name: "my-bucket"},
				{Name: "my-bucket"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for duplicate bucket names")
	}

	foundDup := false
	for _, e := range result.Errors {
		if e.Message == `duplicate bucket name "my-bucket"` {
			foundDup = true
		}
	}
	if !foundDup {
		t.Error("expected duplicate bucket name error")
	}
}

func TestValidate_KVNamespaceValid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		KV: KVConfig{
			Namespaces: []KVNamespaceConfig{
				{Title: "MY-NAMESPACE"},
				{Title: "cache-v2"},
			},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_KVNamespaceInvalid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		KV: KVConfig{
			Namespaces: []KVNamespaceConfig{
				{Title: "invalid namespace!"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for namespace with spaces/special chars")
	}
}

func TestValidate_KVNamespaceDuplicate(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		KV: KVConfig{
			Namespaces: []KVNamespaceConfig{
				{Title: "my-ns"},
				{Title: "my-ns"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for duplicate namespace titles")
	}
}

func TestValidate_DNSRecordTypeValid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "A", Name: "example.com", Content: "1.2.3.4"},
				{Type: "AAAA", Name: "example.com", Content: "::1"},
				{Type: "CNAME", Name: "www", Content: "example.com"},
				{Type: "MX", Name: "example.com", Content: "mail.example.com"},
				{Type: "TXT", Name: "example.com", Content: "v=spf1 include:_spf.google.com ~all"},
			},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_DNSRecordTypeInvalid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "INVALID", Name: "example.com", Content: "1.2.3.4"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for bad DNS type")
	}
}

func TestValidate_DNSRecordDuplicate(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "A", Name: "example.com", Content: "1.2.3.4"},
				{Type: "A", Name: "example.com", Content: "5.6.7.8"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for duplicate A records")
	}
}

func TestValidate_DNSRecordDuplicate_MXAllowed(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "MX", Name: "example.com", Content: "mail1.example.com"},
				{Type: "MX", Name: "example.com", Content: "mail2.example.com"},
			},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid — multiple MX records should be allowed, got errors: %v", result.Errors)
	}
}

func TestValidate_DNSRecordMissingFields(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID:  "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{{}},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for DNS record with no fields")
	}
	if len(result.Errors) < 3 {
		t.Fatalf("expected at least 3 errors (type, name, content), got %d", len(result.Errors))
	}
}

func TestValidate_DNSTTLOutOfRange(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "A", Name: "example.com", Content: "1.2.3.4", TTL: 30},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for TTL=30")
	}
}

func TestValidate_DNSTTLAutoValid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "A", Name: "example.com", Content: "1.2.3.4", TTL: 1},
			},
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid for TTL=1 (auto), got errors: %v", result.Errors)
	}
}

func TestValidate_ZoneIDValid(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
		},
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_ZoneIDInvalid(t *testing.T) {
	tests := []struct {
		name   string
		zoneID string
	}{
		{"too short", "abcdef"},
		{"uppercase", "ABCDEF0123456789ABCDEF0123456789"},
		{"non-hex chars", "zzzzzz0123456789abcdef012345678z"},
		{"too long", "abcdef0123456789abcdef01234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewConfigValidator("")
			cfg := &CosmoflareConfig{
				Name: "test",
				Type: "app",
				DNS: DNSConfig{
					ZoneID: tt.zoneID,
				},
			}

			result := v.Validate(cfg)

			foundZoneErr := false
			for _, e := range result.Errors {
				if e.Field == "dns.zone_id" {
					foundZoneErr = true
				}
			}
			if !foundZoneErr {
				t.Errorf("expected zone_id validation error for %q", tt.zoneID)
			}
		})
	}
}

func TestValidateSSLMode(t *testing.T) {
	valid := []string{"off", "flexible", "full", "strict"}
	for _, m := range valid {
		if err := ValidateSSLMode(m); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", m, err)
		}
	}

	invalid := []string{"", "none", "super", "invalid-mode"}
	for _, m := range invalid {
		if err := ValidateSSLMode(m); err == nil {
			t.Errorf("expected %q to be invalid, got nil error", m)
		}
	}
}

func TestValidatePortNumber(t *testing.T) {
	valid := []int{1, 80, 443, 8080, 65535}
	for _, p := range valid {
		if err := ValidatePortNumber(p); err != nil {
			t.Errorf("expected port %d to be valid, got error: %v", p, err)
		}
	}

	invalid := []int{0, -1, 65536, 100000}
	for _, p := range invalid {
		if err := ValidatePortNumber(p); err == nil {
			t.Errorf("expected port %d to be invalid, got nil error", p)
		}
	}
}

func TestValidate_DNSRecordTypeLowercase(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			ZoneID: "abcdef0123456789abcdef0123456789",
			Records: []DNSRecordConfig{
				{Type: "a", Name: "example.com", Content: "1.2.3.4"},
			},
		},
	}

	result := v.Validate(cfg)

	// Lowercase type is technically valid but should warn
	if result.HasErrors() {
		t.Fatal("lowercase DNS type should not be an error")
	}
	if !result.HasWarnings() {
		t.Fatal("expected warning for lowercase DNS type")
	}
	if !result.Warnings[0].Fixable {
		t.Error("expected the lowercase type warning to be fixable")
	}
}

func TestValidate_R2BucketNameFixable(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		R2: R2Config{
			Buckets: []R2BucketConfig{
				{Name: "my.bucket.name"},
			},
		},
	}

	result := v.Validate(cfg)

	if result.Valid {
		t.Fatal("expected invalid result for bucket with dots")
	}
	if !result.Errors[0].Fixable {
		t.Error("expected the dot error to be fixable")
	}
}

func TestValidate_DNSZoneMissingWithRecords(t *testing.T) {
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
		DNS: DNSConfig{
			Records: []DNSRecordConfig{
				{Type: "A", Name: "example.com", Content: "1.2.3.4"},
			},
		},
	}

	result := v.Validate(cfg)

	// Should be valid (zone_id missing is a warning, not error)
	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}
	if !result.HasWarnings() {
		t.Fatal("expected warning about missing zone_id")
	}
}

func TestValidationResult_TotalFindings(t *testing.T) {
	r := &ValidationResult{
		Errors:   []ValidationFinding{{}, {}},
		Warnings: []ValidationFinding{{}},
		Info:     []ValidationFinding{{}, {}, {}},
	}

	if got := r.TotalFindings(); got != 6 {
		t.Errorf("expected 6 total findings, got %d", got)
	}
}

func TestValidate_EmptyConfig_WorkersKVR2DNS(t *testing.T) {
	// A minimal valid config with no services should pass
	v := NewConfigValidator("")
	cfg := &CosmoflareConfig{
		Name: "test",
		Type: "app",
	}

	result := v.Validate(cfg)

	if !result.Valid {
		t.Fatalf("expected valid result for minimal config, got errors: %v", result.Errors)
	}
	if result.TotalFindings() != 0 {
		t.Errorf("expected 0 findings, got %d", result.TotalFindings())
	}
}
