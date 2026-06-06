package cosmoflare

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// TerraformResource represents a single Terraform resource block.
type TerraformResource struct {
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// TerraformImportBlock represents a terraform import block for state adoption.
type TerraformImportBlock struct {
	To string `json:"to"`
	ID string `json:"id"`
}

// TerraformExportResult holds the full export output.
type TerraformExportResult struct {
	ProviderHCL string                 `json:"provider_hcl"`
	Files       map[string]string      `json:"files"`
	Imports     []TerraformImportBlock `json:"imports"`
	Summary     TerraformExportSummary `json:"summary"`
}

// TerraformExportSummary contains counts of exported resources.
type TerraformExportSummary struct {
	Workers    int `json:"workers"`
	DNSRecords int `json:"dns_records"`
	R2Buckets  int `json:"r2_buckets"`
	KVSpaces   int `json:"kv_namespaces"`
	Zones      int `json:"zones"`
	Total      int `json:"total"`
}

// TerraformExportOption is a functional option for export operations.
type TerraformExportOption func(*terraformExportConfig)

type terraformExportConfig struct {
	services        []string
	providerVersion string
	format          string // "hcl" or "json"
}

// WithTerraformServices filters export to specific services.
func WithTerraformServices(services []string) TerraformExportOption {
	return func(c *terraformExportConfig) { c.services = services }
}

// WithTerraformProviderVersion sets the Cloudflare provider version constraint.
func WithTerraformProviderVersion(version string) TerraformExportOption {
	return func(c *terraformExportConfig) { c.providerVersion = version }
}

// WithTerraformFormat sets the output format (hcl or json).
func WithTerraformFormat(format string) TerraformExportOption {
	return func(c *terraformExportConfig) { c.format = format }
}

// TerraformExporter generates Terraform configuration from live Cloudflare state.
type TerraformExporter struct {
	accountID string
	apiToken  string
}

// NewTerraformExporter creates a new TerraformExporter.
func NewTerraformExporter(accountID, apiToken string) (*TerraformExporter, error) {
	if accountID == "" {
		return nil, validationError("NewTerraformExporter", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewTerraformExporter", "API token is required")
	}
	return &TerraformExporter{accountID: accountID, apiToken: apiToken}, nil
}

// Export fetches live Cloudflare resources and generates Terraform HCL files.
func (te *TerraformExporter) Export(ctx context.Context, opts ...TerraformExportOption) (*TerraformExportResult, error) {
	cfg := &terraformExportConfig{
		providerVersion: "~> 4.0",
		format:          "hcl",
	}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.format != "hcl" && cfg.format != "json" {
		return nil, validationError("TerraformExporter.Export", fmt.Sprintf("unsupported format %q, use hcl or json", cfg.format))
	}

	serviceFilter := make(map[string]bool)
	for _, s := range cfg.services {
		serviceFilter[strings.ToLower(s)] = true
	}
	exportAll := len(serviceFilter) == 0

	result := &TerraformExportResult{
		Files:   make(map[string]string),
		Imports: []TerraformImportBlock{},
	}

	// Provider config
	result.ProviderHCL = GenerateProviderHCL(cfg.providerVersion)
	result.Files["provider.tf"] = result.ProviderHCL

	// Workers
	if exportAll || serviceFilter["workers"] {
		hcl, imports, count, err := te.exportWorkers(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporting workers: %w", err)
		}
		if count > 0 {
			result.Files["workers.tf"] = hcl
			result.Imports = append(result.Imports, imports...)
			result.Summary.Workers = count
		}
	}

	// DNS records (requires zones)
	if exportAll || serviceFilter["dns"] {
		hcl, imports, count, err := te.exportDNS(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporting dns: %w", err)
		}
		if count > 0 {
			result.Files["dns.tf"] = hcl
			result.Imports = append(result.Imports, imports...)
			result.Summary.DNSRecords = count
		}
	}

	// R2 Buckets
	if exportAll || serviceFilter["r2"] {
		hcl, imports, count, err := te.exportR2(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporting r2: %w", err)
		}
		if count > 0 {
			result.Files["r2.tf"] = hcl
			result.Imports = append(result.Imports, imports...)
			result.Summary.R2Buckets = count
		}
	}

	// KV Namespaces
	if exportAll || serviceFilter["kv"] {
		hcl, imports, count, err := te.exportKV(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporting kv: %w", err)
		}
		if count > 0 {
			result.Files["kv.tf"] = hcl
			result.Imports = append(result.Imports, imports...)
			result.Summary.KVSpaces = count
		}
	}

	// Zones
	if exportAll || serviceFilter["zones"] {
		hcl, imports, count, err := te.exportZones(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporting zones: %w", err)
		}
		if count > 0 {
			result.Files["zones.tf"] = hcl
			result.Imports = append(result.Imports, imports...)
			result.Summary.Zones = count
		}
	}

	// Generate import.tf
	if len(result.Imports) > 0 {
		result.Files["import.tf"] = GenerateImportHCL(result.Imports)
	}

	result.Summary.Total = result.Summary.Workers + result.Summary.DNSRecords +
		result.Summary.R2Buckets + result.Summary.KVSpaces + result.Summary.Zones

	return result, nil
}

// GenerateImportBlocks returns import blocks without performing a full export.
func (te *TerraformExporter) GenerateImportBlocks(ctx context.Context, opts ...TerraformExportOption) ([]TerraformImportBlock, error) {
	result, err := te.Export(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return result.Imports, nil
}

// --- Internal export helpers ---

func (te *TerraformExporter) exportWorkers(ctx context.Context) (string, []TerraformImportBlock, int, error) {
	svc, err := NewWorkerServiceFromCreds(te.accountID, te.apiToken)
	if err != nil {
		return "", nil, 0, err
	}
	workers, err := svc.List(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	if len(workers) == 0 {
		return "", nil, 0, nil
	}

	var b strings.Builder
	b.WriteString("# Workers — generated by cosmoflare terraform export\n\n")
	var imports []TerraformImportBlock

	for _, w := range workers {
		tfName := sanitizeTerraformName(w.Name)
		b.WriteString(fmt.Sprintf("resource \"cloudflare_worker_script\" %q {\n", tfName))
		b.WriteString(fmt.Sprintf("  account_id = var.cloudflare_account_id\n"))
		b.WriteString(fmt.Sprintf("  name       = %q\n", w.Name))
		if w.CompatibilityDate != "" {
			b.WriteString(fmt.Sprintf("  compatibility_date = %q\n", w.CompatibilityDate))
		}
		b.WriteString(fmt.Sprintf("  content    = file(\"workers/%s.js\")\n", w.Name))
		b.WriteString("}\n\n")

		imports = append(imports, TerraformImportBlock{
			To: fmt.Sprintf("cloudflare_worker_script.%s", tfName),
			ID: fmt.Sprintf("%s/%s", te.accountID, w.Name),
		})
	}

	return b.String(), imports, len(workers), nil
}

func (te *TerraformExporter) exportDNS(ctx context.Context) (string, []TerraformImportBlock, int, error) {
	// Get zones first to find DNS records
	zoneSvc, err := NewZoneServiceFromCreds(te.accountID, te.apiToken)
	if err != nil {
		return "", nil, 0, err
	}
	zones, err := zoneSvc.List(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	if len(zones) == 0 {
		return "", nil, 0, nil
	}

	var b strings.Builder
	b.WriteString("# DNS Records — generated by cosmoflare terraform export\n\n")
	var imports []TerraformImportBlock
	total := 0

	for _, z := range zones {
		dnsSvc, err := NewDNSServiceFromCreds(z.ID, te.apiToken)
		if err != nil {
			return "", nil, 0, err
		}
		records, err := dnsSvc.List(ctx)
		if err != nil {
			return "", nil, 0, err
		}
		for _, r := range records {
			tfName := sanitizeTerraformName(fmt.Sprintf("%s_%s_%s", z.Name, r.Type, r.Name))
			b.WriteString(fmt.Sprintf("resource \"cloudflare_record\" %q {\n", tfName))
			b.WriteString(fmt.Sprintf("  zone_id = %q\n", z.ID))
			b.WriteString(fmt.Sprintf("  name    = %q\n", r.Name))
			b.WriteString(fmt.Sprintf("  type    = %q\n", r.Type))
			b.WriteString(fmt.Sprintf("  content = %q\n", r.Content))
			b.WriteString(fmt.Sprintf("  ttl     = %d\n", r.TTL))
			b.WriteString(fmt.Sprintf("  proxied = %t\n", r.Proxied))
			if r.Priority != nil {
				b.WriteString(fmt.Sprintf("  priority = %d\n", *r.Priority))
			}
			if r.Comment != "" {
				b.WriteString(fmt.Sprintf("  comment = %q\n", r.Comment))
			}
			b.WriteString("}\n\n")

			imports = append(imports, TerraformImportBlock{
				To: fmt.Sprintf("cloudflare_record.%s", tfName),
				ID: fmt.Sprintf("%s/%s", z.ID, r.ID),
			})
			total++
		}
	}

	return b.String(), imports, total, nil
}

func (te *TerraformExporter) exportR2(ctx context.Context) (string, []TerraformImportBlock, int, error) {
	r2Client, err := NewClient(
		WithAccountID(te.accountID),
		WithAPIToken(te.apiToken),
	)
	if err != nil {
		return "", nil, 0, err
	}
	buckets, err := r2Client.ListBuckets(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	if len(buckets) == 0 {
		return "", nil, 0, nil
	}

	var b strings.Builder
	b.WriteString("# R2 Buckets — generated by cosmoflare terraform export\n\n")
	var imports []TerraformImportBlock

	for _, bkt := range buckets {
		tfName := sanitizeTerraformName(bkt.Name)
		b.WriteString(fmt.Sprintf("resource \"cloudflare_r2_bucket\" %q {\n", tfName))
		b.WriteString(fmt.Sprintf("  account_id = var.cloudflare_account_id\n"))
		b.WriteString(fmt.Sprintf("  name       = %q\n", bkt.Name))
		if bkt.Location != "" {
			b.WriteString(fmt.Sprintf("  location   = %q\n", bkt.Location))
		}
		b.WriteString("}\n\n")

		imports = append(imports, TerraformImportBlock{
			To: fmt.Sprintf("cloudflare_r2_bucket.%s", tfName),
			ID: fmt.Sprintf("%s/%s", te.accountID, bkt.Name),
		})
	}

	return b.String(), imports, len(buckets), nil
}

func (te *TerraformExporter) exportKV(ctx context.Context) (string, []TerraformImportBlock, int, error) {
	svc, err := NewKVServiceFromCreds(te.accountID, te.apiToken)
	if err != nil {
		return "", nil, 0, err
	}
	namespaces, err := svc.ListNamespaces(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	if len(namespaces) == 0 {
		return "", nil, 0, nil
	}

	var b strings.Builder
	b.WriteString("# KV Namespaces — generated by cosmoflare terraform export\n\n")
	var imports []TerraformImportBlock

	for _, ns := range namespaces {
		tfName := sanitizeTerraformName(ns.Title)
		b.WriteString(fmt.Sprintf("resource \"cloudflare_workers_kv_namespace\" %q {\n", tfName))
		b.WriteString(fmt.Sprintf("  account_id = var.cloudflare_account_id\n"))
		b.WriteString(fmt.Sprintf("  title      = %q\n", ns.Title))
		b.WriteString("}\n\n")

		imports = append(imports, TerraformImportBlock{
			To: fmt.Sprintf("cloudflare_workers_kv_namespace.%s", tfName),
			ID: fmt.Sprintf("%s/%s", te.accountID, ns.ID),
		})
	}

	return b.String(), imports, len(namespaces), nil
}

func (te *TerraformExporter) exportZones(ctx context.Context) (string, []TerraformImportBlock, int, error) {
	svc, err := NewZoneServiceFromCreds(te.accountID, te.apiToken)
	if err != nil {
		return "", nil, 0, err
	}
	zones, err := svc.List(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	if len(zones) == 0 {
		return "", nil, 0, nil
	}

	var b strings.Builder
	b.WriteString("# Zones — generated by cosmoflare terraform export\n\n")
	var imports []TerraformImportBlock

	for _, z := range zones {
		tfName := sanitizeTerraformName(z.Name)
		b.WriteString(fmt.Sprintf("data \"cloudflare_zone\" %q {\n", tfName))
		b.WriteString(fmt.Sprintf("  account_id = var.cloudflare_account_id\n"))
		b.WriteString(fmt.Sprintf("  name       = %q\n", z.Name))
		b.WriteString("}\n\n")

		imports = append(imports, TerraformImportBlock{
			To: fmt.Sprintf("cloudflare_zone.%s", tfName),
			ID: z.ID,
		})
	}

	return b.String(), imports, len(zones), nil
}

// --- HCL generation helpers ---

// GenerateProviderHCL returns the Terraform provider configuration block.
func GenerateProviderHCL(providerVersion string) string {
	var b strings.Builder
	b.WriteString("# Cloudflare provider — generated by cosmoflare terraform export\n\n")
	b.WriteString("terraform {\n")
	b.WriteString("  required_providers {\n")
	b.WriteString("    cloudflare = {\n")
	b.WriteString("      source  = \"cloudflare/cloudflare\"\n")
	b.WriteString(fmt.Sprintf("      version = %q\n", providerVersion))
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n\n")
	b.WriteString("variable \"cloudflare_account_id\" {\n")
	b.WriteString("  description = \"Cloudflare Account ID\"\n")
	b.WriteString("  type        = string\n")
	b.WriteString("}\n\n")
	b.WriteString("variable \"cloudflare_api_token\" {\n")
	b.WriteString("  description = \"Cloudflare API Token\"\n")
	b.WriteString("  type        = string\n")
	b.WriteString("  sensitive   = true\n")
	b.WriteString("}\n\n")
	b.WriteString("provider \"cloudflare\" {\n")
	b.WriteString("  api_token = var.cloudflare_api_token\n")
	b.WriteString("}\n")
	return b.String()
}

// GenerateImportHCL returns import blocks for all resources.
func GenerateImportHCL(imports []TerraformImportBlock) string {
	var b strings.Builder
	b.WriteString("# Import blocks — generated by cosmoflare terraform export\n")
	b.WriteString("# Run: terraform plan -generate-config-out=generated.tf\n\n")
	for _, imp := range imports {
		b.WriteString(fmt.Sprintf("import {\n"))
		b.WriteString(fmt.Sprintf("  to = %s\n", imp.To))
		b.WriteString(fmt.Sprintf("  id = %q\n", imp.ID))
		b.WriteString("}\n\n")
	}
	return b.String()
}

// sanitizeTerraformName converts a string to a valid Terraform identifier.
// Terraform names must start with a letter or underscore and contain only
// letters, digits, underscores, and hyphens.
func sanitizeTerraformName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := re.ReplaceAllString(name, "_")
	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}
	if sanitized == "" {
		sanitized = "_unnamed"
	}
	return sanitized
}
