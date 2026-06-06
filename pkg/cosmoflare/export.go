package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

// ExportFormat specifies the output format for config export.
type ExportFormat string

const (
	ExportFormatYAML ExportFormat = "yaml"
	ExportFormatJSON ExportFormat = "json"
)

// ExportConfig represents the full exported Cloudflare account configuration.
type ExportConfig struct {
	Version    int              `json:"version" yaml:"version"`
	ExportedAt string           `json:"exported_at" yaml:"exported_at"`
	AccountID  string           `json:"account_id" yaml:"account_id"`
	Services   ExportedServices `json:"services" yaml:"services"`
}

// ExportedServices contains all service configurations.
type ExportedServices struct {
	Workers      []ExportedWorker      `json:"workers,omitempty" yaml:"workers,omitempty"`
	KVNamespaces []ExportedKVNamespace `json:"kv_namespaces,omitempty" yaml:"kv_namespaces,omitempty"`
	R2Buckets    []ExportedR2Bucket    `json:"r2_buckets,omitempty" yaml:"r2_buckets,omitempty"`
	DNSRecords   []ExportedDNSRecord   `json:"dns_records,omitempty" yaml:"dns_records,omitempty"`
	Zones        []ExportedZone        `json:"zones,omitempty" yaml:"zones,omitempty"`
}

// ExportedWorker represents a Worker in an export file.
type ExportedWorker struct {
	Name              string           `json:"name" yaml:"name"`
	ScriptSize        int64            `json:"script_size" yaml:"script_size"`
	CompatibilityDate string           `json:"compatibility_date,omitempty" yaml:"compatibility_date,omitempty"`
	Bindings          []WorkerBinding  `json:"bindings,omitempty" yaml:"bindings,omitempty"`
}

// ExportedKVNamespace represents a KV namespace in an export file.
type ExportedKVNamespace struct {
	ID    string `json:"id" yaml:"id"`
	Title string `json:"title" yaml:"title"`
}

// ExportedR2Bucket represents an R2 bucket in an export file.
type ExportedR2Bucket struct {
	Name     string `json:"name" yaml:"name"`
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
}

// ExportedDNSRecord represents a DNS record in an export file.
type ExportedDNSRecord struct {
	ZoneID  string  `json:"zone_id" yaml:"zone_id"`
	Type    string  `json:"type" yaml:"type"`
	Name    string  `json:"name" yaml:"name"`
	Content string  `json:"content" yaml:"content"`
	Proxied bool    `json:"proxied" yaml:"proxied"`
	TTL     int     `json:"ttl" yaml:"ttl"`
	Priority *uint16 `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// ExportedZone represents a zone in an export file.
type ExportedZone struct {
	ID     string `json:"id" yaml:"id"`
	Name   string `json:"name" yaml:"name"`
	Status string `json:"status" yaml:"status"`
}

// ExportOption configures Export behavior.
type ExportOption func(*exportConfig)

type exportConfig struct {
	services []string
	format   ExportFormat
}

// WithExportServices filters which services to export.
// Valid values: "workers", "kv", "r2", "dns", "zones".
func WithExportServices(services []string) ExportOption {
	return func(c *exportConfig) { c.services = services }
}

// WithExportFormat sets the output format (yaml or json).
func WithExportFormat(format ExportFormat) ExportOption {
	return func(c *exportConfig) { c.format = format }
}

// ImportOption configures Import behavior.
type ImportOption func(*importConfig)

type importConfig struct {
	dryRun bool
	merge  bool
}

// WithImportDryRun enables dry-run mode (show what would change, don't apply).
func WithImportDryRun(dryRun bool) ImportOption {
	return func(c *importConfig) { c.dryRun = dryRun }
}

// WithImportMerge merges with existing configuration instead of replacing.
func WithImportMerge(merge bool) ImportOption {
	return func(c *importConfig) { c.merge = merge }
}

// ImportAction describes a single change that would be or was applied during import.
type ImportAction struct {
	Service  string `json:"service" yaml:"service"`
	Action   string `json:"action" yaml:"action"` // "create", "update", "skip"
	Resource string `json:"resource" yaml:"resource"`
	Detail   string `json:"detail,omitempty" yaml:"detail,omitempty"`
}

// ImportResult holds the outcome of an import operation.
type ImportResult struct {
	DryRun  bool           `json:"dry_run" yaml:"dry_run"`
	Actions []ImportAction `json:"actions" yaml:"actions"`
	Errors  []string       `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// ExportService handles full account configuration export and import.
type ExportService struct {
	cf        *cloudflare.API
	accountID string
}

// NewExportService creates a new ExportService.
func NewExportService(api *cloudflare.API, accountID string) (*ExportService, error) {
	if api == nil {
		return nil, validationError("NewExportService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewExportService", "account ID is required")
	}
	return &ExportService{cf: api, accountID: accountID}, nil
}

// NewExportServiceFromCreds creates an ExportService from account ID and API token.
func NewExportServiceFromCreds(accountID, apiToken string) (*ExportService, error) {
	if accountID == "" {
		return nil, validationError("NewExportService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewExportService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewExportService", "failed to create Cloudflare API client", err)
	}
	return &ExportService{cf: cf, accountID: accountID}, nil
}

// validServices is the set of recognized service names for filtering.
var validServices = map[string]bool{
	"workers": true,
	"kv":      true,
	"r2":      true,
	"dns":     true,
	"zones":   true,
}

// ValidateServices checks that all service names are recognized.
func ValidateServices(services []string) error {
	for _, s := range services {
		if !validServices[s] {
			return validationError("ValidateServices",
				fmt.Sprintf("unknown service %q; valid values: workers, kv, r2, dns, zones", s))
		}
	}
	return nil
}

// shouldExport returns true if the given service should be included.
func shouldExport(services []string, name string) bool {
	if len(services) == 0 {
		return true // no filter = export all
	}
	for _, s := range services {
		if s == name {
			return true
		}
	}
	return false
}

// Export gathers configuration from all (or filtered) Cloudflare services
// and returns an ExportConfig snapshot.
func (s *ExportService) Export(ctx context.Context, opts ...ExportOption) (*ExportConfig, error) {
	cfg := &exportConfig{format: ExportFormatYAML}
	for _, o := range opts {
		o(cfg)
	}

	if len(cfg.services) > 0 {
		if err := ValidateServices(cfg.services); err != nil {
			return nil, err
		}
	}

	export := &ExportConfig{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		AccountID:  s.accountID,
	}

	// Collect workers
	if shouldExport(cfg.services, "workers") {
		workers, err := s.exportWorkers(ctx)
		if err != nil {
			return nil, newError("ExportService.Export", "failed to export workers", err)
		}
		export.Services.Workers = workers
	}

	// Collect KV namespaces
	if shouldExport(cfg.services, "kv") {
		kvNamespaces, err := s.exportKVNamespaces(ctx)
		if err != nil {
			return nil, newError("ExportService.Export", "failed to export KV namespaces", err)
		}
		export.Services.KVNamespaces = kvNamespaces
	}

	// Collect R2 buckets
	if shouldExport(cfg.services, "r2") {
		buckets, err := s.exportR2Buckets(ctx)
		if err != nil {
			return nil, newError("ExportService.Export", "failed to export R2 buckets", err)
		}
		export.Services.R2Buckets = buckets
	}

	// Collect zones (needed before DNS)
	var zones []ExportedZone
	if shouldExport(cfg.services, "zones") || shouldExport(cfg.services, "dns") {
		var err error
		zones, err = s.exportZones(ctx)
		if err != nil {
			return nil, newError("ExportService.Export", "failed to export zones", err)
		}
		if shouldExport(cfg.services, "zones") {
			export.Services.Zones = zones
		}
	}

	// Collect DNS records (per zone)
	if shouldExport(cfg.services, "dns") {
		records, err := s.exportDNSRecords(ctx, zones)
		if err != nil {
			return nil, newError("ExportService.Export", "failed to export DNS records", err)
		}
		export.Services.DNSRecords = records
	}

	return export, nil
}

// Marshal serializes the ExportConfig in the given format.
func (s *ExportService) Marshal(export *ExportConfig, format ExportFormat) ([]byte, error) {
	if export == nil {
		return nil, validationError("ExportService.Marshal", "export config is nil")
	}
	switch format {
	case ExportFormatJSON:
		return json.MarshalIndent(export, "", "  ")
	case ExportFormatYAML, "":
		return yaml.Marshal(export)
	default:
		return nil, validationError("ExportService.Marshal",
			fmt.Sprintf("unsupported format %q; use yaml or json", format))
	}
}

// WriteFile serializes and writes the export to a file.
func (s *ExportService) WriteFile(export *ExportConfig, path string, format ExportFormat) error {
	data, err := s.Marshal(export, format)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ParseFile reads and parses an export file (auto-detects format from content).
func ParseFile(path string) (*ExportConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, newError("ParseFile", fmt.Sprintf("failed to read file %s", path), err)
	}
	return ParseBytes(data)
}

// ParseBytes parses export config from bytes (tries JSON first, then YAML).
func ParseBytes(data []byte) (*ExportConfig, error) {
	var cfg ExportConfig

	// Try JSON first
	if err := json.Unmarshal(data, &cfg); err == nil && cfg.Version > 0 {
		return &cfg, nil
	}

	// Try YAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, newError("ParseBytes", "failed to parse export file (tried JSON and YAML)", err)
	}
	if cfg.Version == 0 {
		return nil, validationError("ParseBytes", "invalid export file: missing version field")
	}
	return &cfg, nil
}

// Import applies an export configuration to the account.
// It creates resources that don't exist yet. With merge mode, it skips
// existing resources; without merge mode, it reports conflicts.
func (s *ExportService) Import(ctx context.Context, cfg *ExportConfig, opts ...ImportOption) (*ImportResult, error) {
	if cfg == nil {
		return nil, validationError("ExportService.Import", "export config is nil")
	}
	if cfg.Version != 1 {
		return nil, validationError("ExportService.Import",
			fmt.Sprintf("unsupported export version %d; expected 1", cfg.Version))
	}

	icfg := &importConfig{}
	for _, o := range opts {
		o(icfg)
	}

	result := &ImportResult{
		DryRun: icfg.dryRun,
	}

	// Import R2 buckets
	if len(cfg.Services.R2Buckets) > 0 {
		actions, errs := s.importR2Buckets(ctx, cfg.Services.R2Buckets, icfg)
		result.Actions = append(result.Actions, actions...)
		result.Errors = append(result.Errors, errs...)
	}

	// Import KV namespaces
	if len(cfg.Services.KVNamespaces) > 0 {
		actions, errs := s.importKVNamespaces(ctx, cfg.Services.KVNamespaces, icfg)
		result.Actions = append(result.Actions, actions...)
		result.Errors = append(result.Errors, errs...)
	}

	// Import DNS records
	if len(cfg.Services.DNSRecords) > 0 {
		actions, errs := s.importDNSRecords(ctx, cfg.Services.DNSRecords, icfg)
		result.Actions = append(result.Actions, actions...)
		result.Errors = append(result.Errors, errs...)
	}

	return result, nil
}

// --- internal export helpers ---

func (s *ExportService) exportWorkers(ctx context.Context) ([]ExportedWorker, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, _, err := s.cf.ListWorkers(ctx, rc, cloudflare.ListWorkersParams{})
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}

	workers := make([]ExportedWorker, 0, len(resp.WorkerList))
	for _, w := range resp.WorkerList {
		workers = append(workers, ExportedWorker{
			Name:       w.ID,
			ScriptSize: int64(w.Size),
		})
	}
	return workers, nil
}

func (s *ExportService) exportKVNamespaces(ctx context.Context) ([]ExportedKVNamespace, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, _, err := s.cf.ListWorkersKVNamespaces(ctx, rc, cloudflare.ListWorkersKVNamespacesParams{})
	if err != nil {
		return nil, fmt.Errorf("list KV namespaces: %w", err)
	}

	namespaces := make([]ExportedKVNamespace, 0, len(resp))
	for _, ns := range resp {
		namespaces = append(namespaces, ExportedKVNamespace{
			ID:    ns.ID,
			Title: ns.Title,
		})
	}
	return namespaces, nil
}

func (s *ExportService) exportR2Buckets(ctx context.Context) ([]ExportedR2Bucket, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	buckets, err := s.cf.ListR2Buckets(ctx, rc, cloudflare.ListR2BucketsParams{})
	if err != nil {
		return nil, fmt.Errorf("list R2 buckets: %w", err)
	}

	result := make([]ExportedR2Bucket, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, ExportedR2Bucket{
			Name:     b.Name,
			Location: b.Location,
		})
	}
	return result, nil
}

func (s *ExportService) exportZones(ctx context.Context) ([]ExportedZone, error) {
	zones, err := s.cf.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}

	result := make([]ExportedZone, 0, len(zones))
	for _, z := range zones {
		result = append(result, ExportedZone{
			ID:     z.ID,
			Name:   z.Name,
			Status: z.Status,
		})
	}
	return result, nil
}

func (s *ExportService) exportDNSRecords(ctx context.Context, zones []ExportedZone) ([]ExportedDNSRecord, error) {
	var allRecords []ExportedDNSRecord
	for _, zone := range zones {
		rc := cloudflare.ZoneIdentifier(zone.ID)
		records, _, err := s.cf.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{})
		if err != nil {
			return nil, fmt.Errorf("list DNS records for zone %s: %w", zone.ID, err)
		}
		for _, r := range records {
			rec := ExportedDNSRecord{
				ZoneID:  zone.ID,
				Type:    r.Type,
				Name:    r.Name,
				Content: r.Content,
				Proxied: boolValue(r.Proxied),
				TTL:     r.TTL,
			}
			if r.Priority != nil {
				p := *r.Priority
				rec.Priority = &p
			}
			allRecords = append(allRecords, rec)
		}
	}
	return allRecords, nil
}

// --- internal import helpers ---

func (s *ExportService) importR2Buckets(ctx context.Context, buckets []ExportedR2Bucket, cfg *importConfig) ([]ImportAction, []string) {
	var actions []ImportAction
	var errs []string

	rc := cloudflare.AccountIdentifier(s.accountID)

	// Get existing buckets for merge mode
	var existingNames map[string]bool
	if cfg.merge {
		existingNames = make(map[string]bool)
		existing, err := s.cf.ListR2Buckets(ctx, rc, cloudflare.ListR2BucketsParams{})
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to list existing R2 buckets: %v", err))
			return actions, errs
		}
		for _, b := range existing {
			existingNames[b.Name] = true
		}
	}

	for _, b := range buckets {
		if cfg.merge && existingNames != nil && existingNames[b.Name] {
			actions = append(actions, ImportAction{
				Service:  "r2",
				Action:   "skip",
				Resource: b.Name,
				Detail:   "bucket already exists",
			})
			continue
		}

		if cfg.dryRun {
			actions = append(actions, ImportAction{
				Service:  "r2",
				Action:   "create",
				Resource: b.Name,
				Detail:   "would create R2 bucket",
			})
			continue
		}

		_, err := s.cf.CreateR2Bucket(ctx, rc, cloudflare.CreateR2BucketParameters{
			Name: b.Name,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to create R2 bucket %s: %v", b.Name, err))
			continue
		}
		actions = append(actions, ImportAction{
			Service:  "r2",
			Action:   "create",
			Resource: b.Name,
			Detail:   "created R2 bucket",
		})
	}
	return actions, errs
}

func (s *ExportService) importKVNamespaces(ctx context.Context, namespaces []ExportedKVNamespace, cfg *importConfig) ([]ImportAction, []string) {
	var actions []ImportAction
	var errs []string

	rc := cloudflare.AccountIdentifier(s.accountID)

	// Get existing namespaces for merge mode
	var existingTitles map[string]bool
	if cfg.merge {
		existingTitles = make(map[string]bool)
		existing, _, err := s.cf.ListWorkersKVNamespaces(ctx, rc, cloudflare.ListWorkersKVNamespacesParams{})
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to list existing KV namespaces: %v", err))
			return actions, errs
		}
		for _, ns := range existing {
			existingTitles[ns.Title] = true
		}
	}

	for _, ns := range namespaces {
		if cfg.merge && existingTitles != nil && existingTitles[ns.Title] {
			actions = append(actions, ImportAction{
				Service:  "kv",
				Action:   "skip",
				Resource: ns.Title,
				Detail:   "namespace already exists",
			})
			continue
		}

		if cfg.dryRun {
			actions = append(actions, ImportAction{
				Service:  "kv",
				Action:   "create",
				Resource: ns.Title,
				Detail:   "would create KV namespace",
			})
			continue
		}

		_, err := s.cf.CreateWorkersKVNamespace(ctx, rc, cloudflare.CreateWorkersKVNamespaceParams{
			Title: ns.Title,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to create KV namespace %s: %v", ns.Title, err))
			continue
		}
		actions = append(actions, ImportAction{
			Service:  "kv",
			Action:   "create",
			Resource: ns.Title,
			Detail:   "created KV namespace",
		})
	}
	return actions, errs
}

func (s *ExportService) importDNSRecords(ctx context.Context, records []ExportedDNSRecord, cfg *importConfig) ([]ImportAction, []string) {
	var actions []ImportAction
	var errs []string

	// Group records by zone
	byZone := make(map[string][]ExportedDNSRecord)
	for _, r := range records {
		byZone[r.ZoneID] = append(byZone[r.ZoneID], r)
	}

	for zoneID, zoneRecords := range byZone {
		rc := cloudflare.ZoneIdentifier(zoneID)

		// Get existing records for merge mode
		var existingRecords map[string]bool
		if cfg.merge {
			existingRecords = make(map[string]bool)
			existing, _, err := s.cf.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{})
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to list existing DNS records for zone %s: %v", zoneID, err))
				continue
			}
			for _, r := range existing {
				key := fmt.Sprintf("%s:%s:%s", r.Type, r.Name, r.Content)
				existingRecords[key] = true
			}
		}

		for _, rec := range zoneRecords {
			recordKey := fmt.Sprintf("%s:%s:%s", rec.Type, rec.Name, rec.Content)
			displayName := fmt.Sprintf("%s %s -> %s (zone %s)", rec.Type, rec.Name, rec.Content, zoneID)

			if cfg.merge && existingRecords != nil && existingRecords[recordKey] {
				actions = append(actions, ImportAction{
					Service:  "dns",
					Action:   "skip",
					Resource: displayName,
					Detail:   "record already exists",
				})
				continue
			}

			if cfg.dryRun {
				actions = append(actions, ImportAction{
					Service:  "dns",
					Action:   "create",
					Resource: displayName,
					Detail:   "would create DNS record",
				})
				continue
			}

			proxied := rec.Proxied
			params := cloudflare.CreateDNSRecordParams{
				Type:    rec.Type,
				Name:    rec.Name,
				Content: rec.Content,
				TTL:     rec.TTL,
				Proxied: &proxied,
			}
			if rec.Priority != nil {
				params.Priority = rec.Priority
			}

			_, err := s.cf.CreateDNSRecord(ctx, rc, params)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to create DNS record %s: %v", displayName, err))
				continue
			}
			actions = append(actions, ImportAction{
				Service:  "dns",
				Action:   "create",
				Resource: displayName,
				Detail:   "created DNS record",
			})
		}
	}
	return actions, errs
}

// boolValue safely dereferences a *bool, returning false if nil.
func boolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
