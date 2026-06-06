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
