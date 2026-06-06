package cosmoflare

import (
	"strings"
	"testing"
)

// --- Constructor validation ---

func TestNewTerraformExporterValidation(t *testing.T) {
	_, err := NewTerraformExporter("", "token")
	if err == nil {
		t.Error("expected error when account ID is empty")
	}

	_, err = NewTerraformExporter("acct", "")
	if err == nil {
		t.Error("expected error when API token is empty")
	}

	_, err = NewTerraformExporter("", "")
	if err == nil {
		t.Error("expected error when both account ID and API token are empty")
	}
}

func TestNewTerraformExporterSuccess(t *testing.T) {
	te, err := NewTerraformExporter("acct123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if te.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", te.accountID)
	}
	if te.apiToken != "test-token" {
		t.Errorf("expected apiToken=test-token, got %s", te.apiToken)
	}
}

// --- sanitizeTerraformName ---

func TestSanitizeTerraformName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-worker", "my-worker"},
		{"my.worker.name", "my_worker_name"},
		{"123-start", "_123-start"},
		{"valid_name", "valid_name"},
		{"has spaces", "has_spaces"},
		{"special!@#chars", "special___chars"},
		{"", "_unnamed"},
		{"UPPER-case", "UPPER-case"},
		{"a/b/c", "a_b_c"},
		{"under_score-dash", "under_score-dash"},
	}

	for _, tt := range tests {
		got := sanitizeTerraformName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeTerraformName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- GenerateProviderHCL ---

func TestGenerateProviderHCL(t *testing.T) {
	hcl := GenerateProviderHCL("~> 4.0")

	if !strings.Contains(hcl, `source  = "cloudflare/cloudflare"`) {
		t.Error("provider HCL missing cloudflare source")
	}
	if !strings.Contains(hcl, `version = "~> 4.0"`) {
		t.Error("provider HCL missing version constraint")
	}
	if !strings.Contains(hcl, `provider "cloudflare"`) {
		t.Error("provider HCL missing provider block")
	}
	if !strings.Contains(hcl, `variable "cloudflare_account_id"`) {
		t.Error("provider HCL missing account_id variable")
	}
	if !strings.Contains(hcl, `variable "cloudflare_api_token"`) {
		t.Error("provider HCL missing api_token variable")
	}
	if !strings.Contains(hcl, "sensitive   = true") {
		t.Error("provider HCL api_token variable should be sensitive")
	}
	if !strings.Contains(hcl, `api_token = var.cloudflare_api_token`) {
		t.Error("provider HCL missing api_token reference in provider block")
	}
}

func TestGenerateProviderHCL_CustomVersion(t *testing.T) {
	hcl := GenerateProviderHCL("~> 5.2")
	if !strings.Contains(hcl, `version = "~> 5.2"`) {
		t.Errorf("expected custom version ~> 5.2 in output, got:\n%s", hcl)
	}
}

// --- GenerateImportHCL ---

func TestGenerateImportHCL(t *testing.T) {
	imports := []TerraformImportBlock{
		{To: "cloudflare_worker_script.my_worker", ID: "acct123/my-worker"},
		{To: "cloudflare_r2_bucket.my_bucket", ID: "acct123/my-bucket"},
	}

	hcl := GenerateImportHCL(imports)

	if !strings.Contains(hcl, "to = cloudflare_worker_script.my_worker") {
		t.Error("import HCL missing worker import 'to' field")
	}
	if !strings.Contains(hcl, `id = "acct123/my-worker"`) {
		t.Error("import HCL missing worker import ID")
	}
	if !strings.Contains(hcl, "to = cloudflare_r2_bucket.my_bucket") {
		t.Error("import HCL missing R2 bucket import 'to' field")
	}
	if !strings.Contains(hcl, `id = "acct123/my-bucket"`) {
		t.Error("import HCL missing R2 bucket import ID")
	}
	// Should have two import blocks
	if strings.Count(hcl, "import {") != 2 {
		t.Errorf("expected 2 import blocks, got %d", strings.Count(hcl, "import {"))
	}
}

func TestGenerateImportHCL_Empty(t *testing.T) {
	hcl := GenerateImportHCL([]TerraformImportBlock{})
	if strings.Contains(hcl, "import {") {
		t.Error("empty imports should produce no import blocks")
	}
}

// --- TerraformResource type ---

func TestTerraformResourceFields(t *testing.T) {
	r := TerraformResource{
		Type:       "cloudflare_worker_script",
		Name:       "my_worker",
		Attributes: map[string]string{"name": "my-worker"},
	}
	if r.Type != "cloudflare_worker_script" {
		t.Errorf("Type = %q, want cloudflare_worker_script", r.Type)
	}
	if r.Name != "my_worker" {
		t.Errorf("Name = %q, want my_worker", r.Name)
	}
	if r.Attributes["name"] != "my-worker" {
		t.Errorf("Attributes[name] = %q, want my-worker", r.Attributes["name"])
	}
}

// --- TerraformImportBlock type ---

func TestTerraformImportBlockFields(t *testing.T) {
	ib := TerraformImportBlock{
		To: "cloudflare_r2_bucket.assets",
		ID: "acct/assets",
	}
	if ib.To != "cloudflare_r2_bucket.assets" {
		t.Errorf("To = %q, want cloudflare_r2_bucket.assets", ib.To)
	}
	if ib.ID != "acct/assets" {
		t.Errorf("ID = %q, want acct/assets", ib.ID)
	}
}

// --- TerraformExportOption ---

func TestWithTerraformServices(t *testing.T) {
	cfg := &terraformExportConfig{}
	WithTerraformServices([]string{"workers", "dns"})(cfg)
	if len(cfg.services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(cfg.services))
	}
	if cfg.services[0] != "workers" || cfg.services[1] != "dns" {
		t.Errorf("services = %v, want [workers dns]", cfg.services)
	}
}

func TestWithTerraformProviderVersion(t *testing.T) {
	cfg := &terraformExportConfig{}
	WithTerraformProviderVersion("~> 5.0")(cfg)
	if cfg.providerVersion != "~> 5.0" {
		t.Errorf("providerVersion = %q, want ~> 5.0", cfg.providerVersion)
	}
}

func TestWithTerraformFormat(t *testing.T) {
	cfg := &terraformExportConfig{}
	WithTerraformFormat("json")(cfg)
	if cfg.format != "json" {
		t.Errorf("format = %q, want json", cfg.format)
	}
}

// --- Export format validation ---

func TestExportInvalidFormat(t *testing.T) {
	te, _ := NewTerraformExporter("acct123", "test-token")
	_, err := te.Export(nil, WithTerraformFormat("yaml"))
	if err == nil {
		t.Error("expected error for unsupported format yaml")
	}
}

// --- TerraformExportSummary ---

func TestTerraformExportSummaryZero(t *testing.T) {
	s := TerraformExportSummary{}
	if s.Total != 0 {
		t.Errorf("zero summary Total = %d, want 0", s.Total)
	}
}

func TestTerraformExportResultFields(t *testing.T) {
	r := TerraformExportResult{
		ProviderHCL: "terraform {}",
		Files:       map[string]string{"provider.tf": "terraform {}"},
		Imports:     []TerraformImportBlock{{To: "a.b", ID: "c"}},
		Summary:     TerraformExportSummary{Workers: 1, Total: 1},
	}
	if r.ProviderHCL != "terraform {}" {
		t.Errorf("ProviderHCL = %q", r.ProviderHCL)
	}
	if len(r.Files) != 1 {
		t.Errorf("Files count = %d, want 1", len(r.Files))
	}
	if r.Summary.Workers != 1 || r.Summary.Total != 1 {
		t.Errorf("Summary workers=%d total=%d, want 1/1", r.Summary.Workers, r.Summary.Total)
	}
}
