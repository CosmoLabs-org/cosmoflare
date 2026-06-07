package cosmoflare

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

// --- Constructor validation ---

func TestNewExportServiceValidation(t *testing.T) {
	_, err := NewExportService(nil, "acct123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewExportService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewExportService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and accountID are empty")
	}
}

func TestNewExportServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewExportService(cf, "acct123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewExportServiceFromCredsValidation(t *testing.T) {
	_, err := NewExportServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewExportServiceFromCreds("acct123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewExportServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both accountID and apiToken are empty")
	}
}

func TestNewExportServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewExportServiceFromCreds("acct123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

// --- ValidateServices ---

func TestValidateServices_Valid(t *testing.T) {
	cases := [][]string{
		{"workers"},
		{"kv"},
		{"r2"},
		{"dns"},
		{"zones"},
		{"workers", "kv", "r2"},
		{"dns", "zones"},
	}
	for _, services := range cases {
		if err := ValidateServices(services); err != nil {
			t.Errorf("ValidateServices(%v) returned unexpected error: %v", services, err)
		}
	}
}

func TestValidateServices_Invalid(t *testing.T) {
	cases := [][]string{
		{"invalid"},
		{"workers", "bad"},
		{""},
	}
	for _, services := range cases {
		if err := ValidateServices(services); err == nil {
			t.Errorf("ValidateServices(%v) expected error, got nil", services)
		}
	}
}

// --- shouldExport ---

func TestShouldExport(t *testing.T) {
	// No filter = export all
	if !shouldExport(nil, "workers") {
		t.Error("nil filter should export all services")
	}
	if !shouldExport([]string{}, "workers") {
		t.Error("empty filter should export all services")
	}

	// With filter
	filter := []string{"workers", "kv"}
	if !shouldExport(filter, "workers") {
		t.Error("expected workers to be included")
	}
	if !shouldExport(filter, "kv") {
		t.Error("expected kv to be included")
	}
	if shouldExport(filter, "dns") {
		t.Error("expected dns to be excluded")
	}
}

// --- Marshal ---

func TestMarshal_YAML(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
		Services: ExportedServices{
			R2Buckets: []ExportedR2Bucket{
				{Name: "my-bucket", Location: "wnam"},
			},
		},
	}

	data, err := svc.Marshal(export, ExportFormatYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed ExportConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}
	if parsed.Version != 1 {
		t.Errorf("expected version=1, got %d", parsed.Version)
	}
	if len(parsed.Services.R2Buckets) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(parsed.Services.R2Buckets))
	}
	if parsed.Services.R2Buckets[0].Name != "my-bucket" {
		t.Errorf("expected bucket name=my-bucket, got %s", parsed.Services.R2Buckets[0].Name)
	}
}

func TestMarshal_JSON(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
		Services: ExportedServices{
			Workers: []ExportedWorker{
				{Name: "my-worker", ScriptSize: 1024},
			},
		},
	}

	data, err := svc.Marshal(export, ExportFormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed ExportConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed.Version != 1 {
		t.Errorf("expected version=1, got %d", parsed.Version)
	}
	if len(parsed.Services.Workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(parsed.Services.Workers))
	}
}

func TestMarshal_Nil(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	_, err := svc.Marshal(nil, ExportFormatYAML)
	if err == nil {
		t.Error("expected error when export is nil")
	}
}

func TestMarshal_UnsupportedFormat(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	export := &ExportConfig{Version: 1}
	_, err := svc.Marshal(export, "xml")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// --- ParseBytes ---

func TestParseBytes_JSON(t *testing.T) {
	input := `{
		"version": 1,
		"exported_at": "2026-06-06T00:00:00Z",
		"account_id": "acct123",
		"services": {
			"r2_buckets": [{"name": "test-bucket"}]
		}
	}`

	cfg, err := ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version=1, got %d", cfg.Version)
	}
	if cfg.AccountID != "acct123" {
		t.Errorf("expected account_id=acct123, got %s", cfg.AccountID)
	}
	if len(cfg.Services.R2Buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(cfg.Services.R2Buckets))
	}
	if cfg.Services.R2Buckets[0].Name != "test-bucket" {
		t.Errorf("expected bucket name=test-bucket, got %s", cfg.Services.R2Buckets[0].Name)
	}
}

func TestParseBytes_YAML(t *testing.T) {
	input := `version: 1
exported_at: "2026-06-06T00:00:00Z"
account_id: acct123
services:
  kv_namespaces:
    - id: ns-123
      title: MY_KV
`

	cfg, err := ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version=1, got %d", cfg.Version)
	}
	if len(cfg.Services.KVNamespaces) != 1 {
		t.Fatalf("expected 1 KV namespace, got %d", len(cfg.Services.KVNamespaces))
	}
	if cfg.Services.KVNamespaces[0].Title != "MY_KV" {
		t.Errorf("expected title=MY_KV, got %s", cfg.Services.KVNamespaces[0].Title)
	}
}

func TestParseBytes_Invalid(t *testing.T) {
	_, err := ParseBytes([]byte("not valid anything }{]["))
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseBytes_MissingVersion(t *testing.T) {
	input := `account_id: acct123
services:
  r2_buckets: []
`
	_, err := ParseBytes([]byte(input))
	if err == nil {
		t.Error("expected error when version is missing")
	}
}

// --- WriteFile / ParseFile roundtrip ---

func TestWriteFileAndParseFile_YAML(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	dir := t.TempDir()
	path := filepath.Join(dir, "export.yaml")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
		Services: ExportedServices{
			R2Buckets: []ExportedR2Bucket{
				{Name: "bucket-a", Location: "enam"},
				{Name: "bucket-b"},
			},
			Zones: []ExportedZone{
				{ID: "zone-1", Name: "example.com", Status: "active"},
			},
		},
	}

	if err := svc.WriteFile(export, path, ExportFormatYAML); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if parsed.Version != 1 {
		t.Errorf("version mismatch: got %d", parsed.Version)
	}
	if len(parsed.Services.R2Buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(parsed.Services.R2Buckets))
	}
	if len(parsed.Services.Zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(parsed.Services.Zones))
	}
}

func TestWriteFileAndParseFile_JSON(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	dir := t.TempDir()
	path := filepath.Join(dir, "export.json")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
		Services: ExportedServices{
			Workers: []ExportedWorker{
				{Name: "worker-1", ScriptSize: 2048, CompatibilityDate: "2024-01-01"},
			},
			KVNamespaces: []ExportedKVNamespace{
				{ID: "kv-1", Title: "MY_DATA"},
			},
		},
	}

	if err := svc.WriteFile(export, path, ExportFormatJSON); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(parsed.Services.Workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(parsed.Services.Workers))
	}
	if parsed.Services.Workers[0].CompatibilityDate != "2024-01-01" {
		t.Errorf("expected compat date 2024-01-01, got %s", parsed.Services.Workers[0].CompatibilityDate)
	}
	if len(parsed.Services.KVNamespaces) != 1 {
		t.Errorf("expected 1 KV namespace, got %d", len(parsed.Services.KVNamespaces))
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/export.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// --- Import validation ---

func TestImport_NilConfig(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	_, err := svc.Import(context.Background(), nil)
	if err == nil {
		t.Error("expected error when config is nil")
	}
}

func TestImport_UnsupportedVersion(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{Version: 99}
	_, err := svc.Import(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

// --- ExportConfig struct ---

func TestExportConfig_FullRoundtrip(t *testing.T) {
	priority := uint16(10)
	original := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T12:00:00Z",
		AccountID:  "acct-full-test",
		Services: ExportedServices{
			Workers: []ExportedWorker{
				{Name: "api-worker", ScriptSize: 4096, CompatibilityDate: "2024-03-01",
					Bindings: []WorkerBinding{{Name: "KV", Type: "kv", ID: "ns-1"}}},
			},
			KVNamespaces: []ExportedKVNamespace{
				{ID: "ns-1", Title: "CACHE"},
				{ID: "ns-2", Title: "SESSION"},
			},
			R2Buckets: []ExportedR2Bucket{
				{Name: "assets", Location: "wnam"},
				{Name: "backups"},
			},
			DNSRecords: []ExportedDNSRecord{
				{ZoneID: "z1", Type: "A", Name: "www", Content: "1.2.3.4", Proxied: true, TTL: 1},
				{ZoneID: "z1", Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600, Priority: &priority},
			},
			Zones: []ExportedZone{
				{ID: "z1", Name: "example.com", Status: "active"},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Parse back
	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.AccountID != original.AccountID {
		t.Errorf("account_id mismatch: %s vs %s", parsed.AccountID, original.AccountID)
	}
	if len(parsed.Services.Workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(parsed.Services.Workers))
	}
	if len(parsed.Services.Workers[0].Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(parsed.Services.Workers[0].Bindings))
	}
	if len(parsed.Services.KVNamespaces) != 2 {
		t.Errorf("expected 2 KV namespaces, got %d", len(parsed.Services.KVNamespaces))
	}
	if len(parsed.Services.R2Buckets) != 2 {
		t.Errorf("expected 2 R2 buckets, got %d", len(parsed.Services.R2Buckets))
	}
	if len(parsed.Services.DNSRecords) != 2 {
		t.Errorf("expected 2 DNS records, got %d", len(parsed.Services.DNSRecords))
	}
	if parsed.Services.DNSRecords[1].Priority == nil || *parsed.Services.DNSRecords[1].Priority != 10 {
		t.Error("expected MX record priority=10")
	}
	if len(parsed.Services.Zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(parsed.Services.Zones))
	}
}

// --- boolValue helper ---

func TestBoolValue(t *testing.T) {
	if boolValue(nil) != false {
		t.Error("boolValue(nil) should return false")
	}
	tr := true
	if boolValue(&tr) != true {
		t.Error("boolValue(&true) should return true")
	}
	fl := false
	if boolValue(&fl) != false {
		t.Error("boolValue(&false) should return false")
	}
}

// --- Export with service filter validation ---

func TestExport_InvalidServiceFilter(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	_, err := svc.Export(context.Background(), WithExportServices([]string{"invalid"}))
	if err == nil {
		t.Error("expected error for invalid service filter")
	}
}

// --- ImportOption helpers ---

func TestImportOptions(t *testing.T) {
	cfg := &importConfig{}

	WithImportDryRun(true)(cfg)
	if !cfg.dryRun {
		t.Error("expected dryRun=true")
	}

	WithImportMerge(true)(cfg)
	if !cfg.merge {
		t.Error("expected merge=true")
	}
}

// --- ExportOption helpers ---

func TestExportOptions(t *testing.T) {
	cfg := &exportConfig{}

	WithExportServices([]string{"r2", "dns"})(cfg)
	if len(cfg.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cfg.services))
	}

	WithExportFormat(ExportFormatJSON)(cfg)
	if cfg.format != ExportFormatJSON {
		t.Errorf("expected format=json, got %s", cfg.format)
	}
}

// --- YAML with empty services ---

func TestParseBytes_EmptyServices(t *testing.T) {
	input := `version: 1
exported_at: "2026-06-06T00:00:00Z"
account_id: acct123
services: {}
`
	cfg, err := ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version=1, got %d", cfg.Version)
	}
	if len(cfg.Services.Workers) != 0 {
		t.Errorf("expected 0 workers, got %d", len(cfg.Services.Workers))
	}
	if len(cfg.Services.R2Buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(cfg.Services.R2Buckets))
	}
}

// --- WriteFile to nonexistent directory ---

func TestWriteFile_BadPath(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	export := &ExportConfig{Version: 1}
	err := svc.WriteFile(export, "/nonexistent/dir/file.yaml", ExportFormatYAML)
	if err == nil {
		t.Error("expected error writing to nonexistent directory")
	}
}

// --- Temp file cleanup verification ---

func TestWriteFile_CreatesFile(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	dir := t.TempDir()
	path := filepath.Join(dir, "test-export.yaml")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
	}

	if err := svc.WriteFile(export, path, ExportFormatYAML); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("file is empty")
	}
}

// --- ExportFormat constants ---

func TestExportFormat_Constants(t *testing.T) {
	if ExportFormatYAML != "yaml" {
		t.Errorf("expected ExportFormatYAML=yaml, got %q", ExportFormatYAML)
	}
	if ExportFormatJSON != "json" {
		t.Errorf("expected ExportFormatJSON=json, got %q", ExportFormatJSON)
	}
}

// --- WithExportFormat sets YAML ---

func TestExportOptions_WithYAMLFormat(t *testing.T) {
	cfg := &exportConfig{}
	WithExportFormat(ExportFormatYAML)(cfg)
	if cfg.format != ExportFormatYAML {
		t.Errorf("expected format=yaml, got %s", cfg.format)
	}
}

// --- WithExportServices with empty slice (export all) ---

func TestExportOptions_EmptyServices(t *testing.T) {
	cfg := &exportConfig{services: []string{"r2"}}
	WithExportServices([]string{})(cfg)
	if len(cfg.services) != 0 {
		t.Errorf("expected 0 services after setting empty, got %d", len(cfg.services))
	}
	// Empty filter means export all
	if !shouldExport(cfg.services, "workers") {
		t.Error("empty filter should export all services")
	}
}

// --- Multiple ExportOptions applied in sequence ---

func TestExportOptions_MultipleApplied(t *testing.T) {
	cfg := &exportConfig{}
	WithExportServices([]string{"r2", "kv"})(cfg)
	WithExportFormat(ExportFormatJSON)(cfg)
	WithExportServices([]string{"dns"})(cfg) // overwrite
	if len(cfg.services) != 1 || cfg.services[0] != "dns" {
		t.Errorf("expected services=[dns] after overwrite, got %v", cfg.services)
	}
	if cfg.format != ExportFormatJSON {
		t.Errorf("expected format=json, got %s", cfg.format)
	}
}

// --- WithImportDryRun false / WithImportMerge false ---

func TestImportOptions_FalseValues(t *testing.T) {
	cfg := &importConfig{dryRun: true, merge: true}
	WithImportDryRun(false)(cfg)
	if cfg.dryRun {
		t.Error("expected dryRun=false after WithImportDryRun(false)")
	}
	WithImportMerge(false)(cfg)
	if cfg.merge {
		t.Error("expected merge=false after WithImportMerge(false)")
	}
}

// --- Multiple ImportOptions applied in sequence ---

func TestImportOptions_BothSet(t *testing.T) {
	cfg := &importConfig{}
	WithImportDryRun(true)(cfg)
	WithImportMerge(true)(cfg)
	if !cfg.dryRun {
		t.Error("expected dryRun=true")
	}
	if !cfg.merge {
		t.Error("expected merge=true")
	}
}

// --- Marshal with empty-string format (treated as YAML) ---

func TestMarshal_EmptyFormatFallsBackToYAML(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
	}
	data, err := svc.Marshal(export, ExportFormat(""))
	if err != nil {
		t.Fatalf("unexpected error for empty format: %v", err)
	}
	// Result should be valid YAML
	var parsed ExportConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("expected YAML output for empty format, parse failed: %v", err)
	}
	if parsed.Version != 1 {
		t.Errorf("expected version=1, got %d", parsed.Version)
	}
}

// --- ValidateServices with empty slice is valid ---

func TestValidateServices_Empty(t *testing.T) {
	if err := ValidateServices([]string{}); err != nil {
		t.Errorf("ValidateServices(empty) should not error, got: %v", err)
	}
}

// --- ValidateServices with nil is valid ---

func TestValidateServices_Nil(t *testing.T) {
	if err := ValidateServices(nil); err != nil {
		t.Errorf("ValidateServices(nil) should not error, got: %v", err)
	}
}

// --- shouldExport: exact match required (no partial) ---

func TestShouldExport_ExactMatch(t *testing.T) {
	filter := []string{"workers"}
	if shouldExport(filter, "work") {
		t.Error("shouldExport should not match partial name 'work' against filter 'workers'")
	}
	if shouldExport(filter, "WORKERS") {
		t.Error("shouldExport should be case-sensitive")
	}
	if !shouldExport(filter, "workers") {
		t.Error("shouldExport should match exact name 'workers'")
	}
}

// --- shouldExport: all five valid service names ---

func TestShouldExport_AllServices(t *testing.T) {
	services := []string{"workers", "kv", "r2", "dns", "zones"}
	for _, svc := range services {
		if !shouldExport(services, svc) {
			t.Errorf("shouldExport should include service %q when it's in the filter", svc)
		}
	}
}

// --- Import version 0 returns error ---

func TestImport_VersionZero(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{Version: 0}
	_, err := svc.Import(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for version=0")
	}
}

// --- Import with empty services (no-op) ---

func TestImport_EmptyServices(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{
		Version:   1,
		AccountID: "acct123",
		Services:  ExportedServices{}, // nothing to import
	}
	result, err := svc.Import(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for empty services, got %d", len(result.Actions))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors for empty services, got %d", len(result.Errors))
	}
}

// --- Import dry-run with R2 buckets (no HTTP call, checks Actions) ---

func TestImport_DryRun_R2Buckets(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{
		Version:   1,
		AccountID: "acct123",
		Services: ExportedServices{
			R2Buckets: []ExportedR2Bucket{
				{Name: "bucket-x"},
				{Name: "bucket-y"},
			},
		},
	}
	result, err := svc.Import(context.Background(), cfg, WithImportDryRun(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true in result")
	}
	if len(result.Actions) != 2 {
		t.Fatalf("expected 2 dry-run actions, got %d", len(result.Actions))
	}
	for _, action := range result.Actions {
		if action.Service != "r2" {
			t.Errorf("expected service=r2, got %s", action.Service)
		}
		if action.Action != "create" {
			t.Errorf("expected action=create, got %s", action.Action)
		}
	}
}

// --- Import dry-run with KV namespaces ---

func TestImport_DryRun_KVNamespaces(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{
		Version:   1,
		AccountID: "acct123",
		Services: ExportedServices{
			KVNamespaces: []ExportedKVNamespace{
				{ID: "ns-1", Title: "CACHE"},
				{ID: "ns-2", Title: "SESSION"},
			},
		},
	}
	result, err := svc.Import(context.Background(), cfg, WithImportDryRun(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true in result")
	}
	if len(result.Actions) != 2 {
		t.Fatalf("expected 2 dry-run actions for KV, got %d", len(result.Actions))
	}
	for _, action := range result.Actions {
		if action.Service != "kv" {
			t.Errorf("expected service=kv, got %s", action.Service)
		}
		if action.Action != "create" {
			t.Errorf("expected action=create, got %s", action.Action)
		}
	}
}

// --- Import dry-run with DNS records ---

func TestImport_DryRun_DNSRecords(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	priority := uint16(10)
	cfg := &ExportConfig{
		Version:   1,
		AccountID: "acct123",
		Services: ExportedServices{
			DNSRecords: []ExportedDNSRecord{
				{ZoneID: "zone-1", Type: "A", Name: "www", Content: "1.2.3.4", TTL: 300},
				{ZoneID: "zone-1", Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600, Priority: &priority},
			},
		},
	}
	result, err := svc.Import(context.Background(), cfg, WithImportDryRun(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true in result")
	}
	if len(result.Actions) != 2 {
		t.Fatalf("expected 2 dry-run actions for DNS, got %d", len(result.Actions))
	}
	for _, action := range result.Actions {
		if action.Service != "dns" {
			t.Errorf("expected service=dns, got %s", action.Service)
		}
		if action.Action != "create" {
			t.Errorf("expected action=create, got %s", action.Action)
		}
	}
}

// --- Import dry-run propagates DryRun to result ---

func TestImport_DryRunFalseByDefault(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{Version: 1, AccountID: "acct123"}
	result, err := svc.Import(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("expected DryRun=false when no option provided")
	}
}

// --- ExportedDNSRecord priority field nil vs set ---

func TestExportedDNSRecord_Priority(t *testing.T) {
	// nil priority
	rec := ExportedDNSRecord{
		ZoneID: "z1", Type: "A", Name: "api", Content: "5.6.7.8", TTL: 1,
	}
	if rec.Priority != nil {
		t.Error("expected nil priority for A record")
	}

	// non-nil priority
	p := uint16(20)
	mxRec := ExportedDNSRecord{
		ZoneID: "z1", Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600,
		Priority: &p,
	}
	if mxRec.Priority == nil || *mxRec.Priority != 20 {
		t.Errorf("expected priority=20, got %v", mxRec.Priority)
	}
}

// --- ExportedDNSRecord JSON round-trip preserves Priority ---

func TestExportedDNSRecord_JSONRoundtrip(t *testing.T) {
	p := uint16(5)
	original := ExportedDNSRecord{
		ZoneID: "z1", Type: "MX", Name: "mail", Content: "mx.example.com",
		Proxied: false, TTL: 3600, Priority: &p,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed ExportedDNSRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Priority == nil || *parsed.Priority != 5 {
		t.Errorf("expected priority=5 after roundtrip, got %v", parsed.Priority)
	}
	if parsed.ZoneID != "z1" {
		t.Errorf("expected zone_id=z1, got %s", parsed.ZoneID)
	}
}

// --- ExportedDNSRecord YAML round-trip ---

func TestExportedDNSRecord_YAMLRoundtrip(t *testing.T) {
	input := `zone_id: z2
type: CNAME
name: blog
content: example.com
proxied: true
ttl: 1
`
	var rec ExportedDNSRecord
	if err := yaml.Unmarshal([]byte(input), &rec); err != nil {
		t.Fatalf("yaml unmarshal error: %v", err)
	}
	if rec.ZoneID != "z2" {
		t.Errorf("expected zone_id=z2, got %s", rec.ZoneID)
	}
	if !rec.Proxied {
		t.Error("expected proxied=true")
	}
	if rec.Priority != nil {
		t.Error("expected nil priority for CNAME record")
	}
}

// --- WriteFile with nil export propagates error ---

func TestWriteFile_NilExport(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	err := svc.WriteFile(nil, path, ExportFormatYAML)
	if err == nil {
		t.Error("expected error when export is nil")
	}
}

// --- ParseBytes with all service types ---

func TestParseBytes_AllServiceTypes(t *testing.T) {
	input := `{
		"version": 1,
		"exported_at": "2026-06-06T00:00:00Z",
		"account_id": "acct-all",
		"services": {
			"workers": [{"name": "w1", "script_size": 512}],
			"kv_namespaces": [{"id": "kv1", "title": "KV_ONE"}],
			"r2_buckets": [{"name": "bucket1", "location": "enam"}],
			"dns_records": [{"zone_id": "z1", "type": "A", "name": "www", "content": "1.1.1.1", "proxied": false, "ttl": 300}],
			"zones": [{"id": "z1", "name": "example.com", "status": "active"}]
		}
	}`

	cfg, err := ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Services.Workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(cfg.Services.Workers))
	}
	if len(cfg.Services.KVNamespaces) != 1 {
		t.Errorf("expected 1 KV namespace, got %d", len(cfg.Services.KVNamespaces))
	}
	if len(cfg.Services.R2Buckets) != 1 {
		t.Errorf("expected 1 R2 bucket, got %d", len(cfg.Services.R2Buckets))
	}
	if len(cfg.Services.DNSRecords) != 1 {
		t.Errorf("expected 1 DNS record, got %d", len(cfg.Services.DNSRecords))
	}
	if len(cfg.Services.Zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(cfg.Services.Zones))
	}
	if cfg.Services.Workers[0].ScriptSize != 512 {
		t.Errorf("expected script_size=512, got %d", cfg.Services.Workers[0].ScriptSize)
	}
	if cfg.Services.R2Buckets[0].Location != "enam" {
		t.Errorf("expected location=enam, got %s", cfg.Services.R2Buckets[0].Location)
	}
}

// --- ImportResult struct fields ---

func TestImportResult_Fields(t *testing.T) {
	result := &ImportResult{
		DryRun: true,
		Actions: []ImportAction{
			{Service: "r2", Action: "create", Resource: "my-bucket", Detail: "would create R2 bucket"},
		},
		Errors: []string{"some error"},
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if len(result.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(result.Actions))
	}
	if result.Actions[0].Service != "r2" {
		t.Errorf("expected service=r2, got %s", result.Actions[0].Service)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

// --- ImportAction struct fields ---

func TestImportAction_Fields(t *testing.T) {
	action := ImportAction{
		Service:  "kv",
		Action:   "skip",
		Resource: "MY_NAMESPACE",
		Detail:   "namespace already exists",
	}
	if action.Service != "kv" {
		t.Errorf("expected service=kv, got %s", action.Service)
	}
	if action.Action != "skip" {
		t.Errorf("expected action=skip, got %s", action.Action)
	}
	if action.Detail != "namespace already exists" {
		t.Errorf("unexpected detail: %s", action.Detail)
	}
}

// --- ExportConfig ExportedAt field preserved in JSON roundtrip ---

func TestExportConfig_ExportedAt_Preserved(t *testing.T) {
	original := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-01-15T14:32:05Z",
		AccountID:  "acct-ts",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.ExportedAt != original.ExportedAt {
		t.Errorf("ExportedAt mismatch: got %q, want %q", parsed.ExportedAt, original.ExportedAt)
	}
}

// --- Import dry-run with all service types together ---

func TestImport_DryRun_AllServices(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	cfg := &ExportConfig{
		Version:   1,
		AccountID: "acct123",
		Services: ExportedServices{
			R2Buckets:    []ExportedR2Bucket{{Name: "b1"}},
			KVNamespaces: []ExportedKVNamespace{{ID: "kv1", Title: "NS1"}},
			DNSRecords:   []ExportedDNSRecord{{ZoneID: "z1", Type: "A", Name: "api", Content: "9.9.9.9", TTL: 60}},
		},
	}
	result, err := svc.Import(context.Background(), cfg, WithImportDryRun(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect 3 actions total (1 r2 + 1 kv + 1 dns)
	if len(result.Actions) != 3 {
		t.Errorf("expected 3 dry-run actions, got %d", len(result.Actions))
	}
	services := map[string]bool{}
	for _, a := range result.Actions {
		services[a.Service] = true
		if a.Action != "create" {
			t.Errorf("expected action=create, got %s for service %s", a.Action, a.Service)
		}
	}
	for _, expected := range []string{"r2", "kv", "dns"} {
		if !services[expected] {
			t.Errorf("missing dry-run action for service %s", expected)
		}
	}
}

// --- ParseFile roundtrip with DNS records ---

func TestParseFile_DNSRecords(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewExportService(cf, "acct123")

	dir := t.TempDir()
	path := filepath.Join(dir, "dns-export.json")

	priority := uint16(15)
	export := &ExportConfig{
		Version:    1,
		ExportedAt: "2026-06-06T00:00:00Z",
		AccountID:  "acct123",
		Services: ExportedServices{
			DNSRecords: []ExportedDNSRecord{
				{ZoneID: "z1", Type: "A", Name: "www", Content: "10.0.0.1", Proxied: true, TTL: 1},
				{ZoneID: "z1", Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600, Priority: &priority},
			},
			Zones: []ExportedZone{
				{ID: "z1", Name: "example.com", Status: "active"},
			},
		},
	}

	if err := svc.WriteFile(export, path, ExportFormatJSON); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(parsed.Services.DNSRecords) != 2 {
		t.Fatalf("expected 2 DNS records, got %d", len(parsed.Services.DNSRecords))
	}
	if parsed.Services.DNSRecords[0].Type != "A" {
		t.Errorf("expected first record type=A, got %s", parsed.Services.DNSRecords[0].Type)
	}
	if !parsed.Services.DNSRecords[0].Proxied {
		t.Error("expected first record proxied=true")
	}
	mx := parsed.Services.DNSRecords[1]
	if mx.Priority == nil || *mx.Priority != 15 {
		t.Errorf("expected MX priority=15, got %v", mx.Priority)
	}
}
