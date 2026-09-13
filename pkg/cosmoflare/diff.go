package cosmoflare

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/viper"
)

// DiffAction describes whether a resource will be added, removed, or modified.
type DiffAction string

const (
	DiffAdd    DiffAction = "add"
	DiffRemove DiffAction = "remove"
	DiffModify DiffAction = "modify"
)

// DiffEntry represents a single difference between local config and live state.
type DiffEntry struct {
	Action   DiffAction `json:"action"`
	Service  string     `json:"service"`
	Resource string     `json:"resource"`
	Detail   string     `json:"detail,omitempty"`
}

// DiffResult holds the full comparison between local config and live Cloudflare state.
type DiffResult struct {
	Service   string      `json:"service"`
	Additions []DiffEntry `json:"additions"`
	Deletions []DiffEntry `json:"deletions"`
	Changes   []DiffEntry `json:"changes"`
}

// TotalChanges returns the total number of changes across all categories.
func (r *DiffResult) TotalChanges() int {
	return len(r.Additions) + len(r.Deletions) + len(r.Changes)
}

// HasChanges returns true if there is at least one difference.
func (r *DiffResult) HasChanges() bool {
	return r.TotalChanges() > 0
}

// DiffSummary provides an aggregate view across all services.
type DiffSummary struct {
	Results    []DiffResult `json:"results"`
	TotalAdd   int          `json:"total_additions"`
	TotalDel   int          `json:"total_deletions"`
	TotalMod   int          `json:"total_modifications"`
	HasChanges bool         `json:"has_changes"`
}

// CosmoflareConfig represents the top-level .cosmoflare.yaml structure
// used for diff comparisons. This is intentionally decoupled from the
// init templates so the diff service can parse real-world configs.
type CosmoflareConfig struct {
	Name    string            `mapstructure:"name" json:"name"`
	Type    string            `mapstructure:"type" json:"type"`
	Workers map[string]WorkerConfig `mapstructure:"workers" json:"workers,omitempty"`
	R2      R2Config          `mapstructure:"r2" json:"r2,omitempty"`
	KV      KVConfig          `mapstructure:"kv" json:"kv,omitempty"`
	DNS     DNSConfig         `mapstructure:"dns" json:"dns,omitempty"`
}

// WorkerConfig describes a single Worker in .cosmoflare.yaml.
type WorkerConfig struct {
	Script            string `mapstructure:"script" json:"script"`
	CompatibilityDate string `mapstructure:"compatibility_date" json:"compatibility_date,omitempty"`
	Module            bool   `mapstructure:"module" json:"module,omitempty"`
}

// R2Config holds R2-specific configuration.
type R2Config struct {
	Buckets []R2BucketConfig `mapstructure:"buckets" json:"buckets,omitempty"`
}

// R2BucketConfig describes a single R2 bucket in the config.
type R2BucketConfig struct {
	Name     string `mapstructure:"name" json:"name"`
	Location string `mapstructure:"location" json:"location,omitempty"`
}

// KVConfig holds KV-specific configuration.
type KVConfig struct {
	Namespaces []KVNamespaceConfig `mapstructure:"namespaces" json:"namespaces,omitempty"`
}

// KVNamespaceConfig describes a single KV namespace in the config.
type KVNamespaceConfig struct {
	Title string `mapstructure:"title" json:"title"`
}

// DNSConfig holds DNS-specific configuration.
type DNSConfig struct {
	ZoneID  string          `mapstructure:"zone_id" json:"zone_id,omitempty"`
	Records []DNSRecordConfig `mapstructure:"records" json:"records,omitempty"`
}

// DNSRecordConfig describes a single DNS record in the config.
type DNSRecordConfig struct {
	Type    string `mapstructure:"type" json:"type"`
	Name    string `mapstructure:"name" json:"name"`
	Content string `mapstructure:"content" json:"content"`
	TTL     int    `mapstructure:"ttl" json:"ttl,omitempty"`
	Proxied bool   `mapstructure:"proxied" json:"proxied,omitempty"`
}

// DiffService compares local .cosmoflare.yaml configuration against live
// Cloudflare state. It is read-only and never modifies live resources.
type DiffService struct {
	accountID string
	apiToken  string
}

// NewDiffService creates a new DiffService.
func NewDiffService(accountID, apiToken string) (*DiffService, error) {
	if accountID == "" {
		return nil, validationError("NewDiffService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewDiffService", "API token is required")
	}
	return &DiffService{accountID: accountID, apiToken: apiToken}, nil
}

// LoadCosmoflareConfig loads a .cosmoflare.yaml from the given directory,
// searching upward through parent directories.
func LoadCosmoflareConfig(dir string) (*CosmoflareConfig, error) {
	path, err := findProjectFile(dir, ".cosmoflare.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to find .cosmoflare.yaml: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read .cosmoflare.yaml: %w", err)
	}

	var cfg CosmoflareConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .cosmoflare.yaml: %w", err)
	}

	return &cfg, nil
}

// CompareWorkers compares local Workers config against live state.
func (d *DiffService) CompareWorkers(ctx context.Context, local map[string]WorkerConfig) (*DiffResult, error) {
	result := &DiffResult{Service: "workers"}

	ws, err := NewWorkerServiceFromCreds(d.accountID, d.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}

	liveWorkers, err := ws.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list live workers: %w", err)
	}

	liveMap := make(map[string]bool, len(liveWorkers))
	for _, w := range liveWorkers {
		liveMap[w.Name] = true
	}

	// Find additions (in local, not live)
	localNames := make([]string, 0, len(local))
	for name := range local {
		localNames = append(localNames, name)
	}
	sort.Strings(localNames)
	for _, name := range localNames {
		if !liveMap[name] {
			result.Additions = append(result.Additions, DiffEntry{
				Action:   DiffAdd,
				Service:  "workers",
				Resource: name,
				Detail:   fmt.Sprintf("worker %q exists in config but not deployed", name),
			})
		}
	}

	// Find deletions (live, not in local)
	liveNames := make([]string, 0, len(liveMap))
	for name := range liveMap {
		liveNames = append(liveNames, name)
	}
	sort.Strings(liveNames)
	for _, name := range liveNames {
		if _, ok := local[name]; !ok {
			result.Deletions = append(result.Deletions, DiffEntry{
				Action:   DiffRemove,
				Service:  "workers",
				Resource: name,
				Detail:   fmt.Sprintf("worker %q is deployed but not in config", name),
			})
		}
	}

	return result, nil
}

// CompareR2 compares local R2 bucket config against live state.
func (d *DiffService) CompareR2(ctx context.Context, local R2Config) (*DiffResult, error) {
	result := &DiffResult{Service: "r2"}

	client, err := NewClient(
		WithAccountID(d.accountID),
		WithAPIToken(d.apiToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 client: %w", err)
	}

	liveBuckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list live buckets: %w", err)
	}

	liveMap := make(map[string]bool, len(liveBuckets))
	for _, b := range liveBuckets {
		liveMap[b.Name] = true
	}

	// Find additions (in local config, not live)
	for _, b := range local.Buckets {
		if !liveMap[b.Name] {
			result.Additions = append(result.Additions, DiffEntry{
				Action:   DiffAdd,
				Service:  "r2",
				Resource: b.Name,
				Detail:   fmt.Sprintf("bucket %q in config but does not exist", b.Name),
			})
		}
	}

	// Find deletions (live, not in local config)
	localMap := make(map[string]bool, len(local.Buckets))
	for _, b := range local.Buckets {
		localMap[b.Name] = true
	}
	liveNames := make([]string, 0, len(liveMap))
	for name := range liveMap {
		liveNames = append(liveNames, name)
	}
	sort.Strings(liveNames)
	for _, name := range liveNames {
		if !localMap[name] {
			result.Deletions = append(result.Deletions, DiffEntry{
				Action:   DiffRemove,
				Service:  "r2",
				Resource: name,
				Detail:   fmt.Sprintf("bucket %q exists but not in config", name),
			})
		}
	}

	return result, nil
}

// indexKVByTitle builds a title→ID index over live KV namespaces and REFUSES
// ambiguous duplicate titles (BUG-048): with two namespaces sharing a title,
// every title-based match — including apply's delete resolution — is a coin
// flip that can destroy the wrong namespace. The error names both IDs so the
// user can delete one or match by ID instead.
func indexKVByTitle(namespaces []*KVNamespace) (map[string]string, error) {
	idx := make(map[string]string, len(namespaces))
	for _, ns := range namespaces {
		if prev, dup := idx[ns.Title]; dup {
			return nil, fmt.Errorf(
				"duplicate KV namespace title %q: namespaces %s and %s share it — delete one (or match by id) before diffing/applying",
				ns.Title, prev, ns.ID)
		}
		idx[ns.Title] = ns.ID
	}
	return idx, nil
}

// CompareKV compares local KV namespace config against live state.
func (d *DiffService) CompareKV(ctx context.Context, local KVConfig) (*DiffResult, error) {
	result := &DiffResult{Service: "kv"}

	kvSvc, err := NewKVServiceFromCreds(d.accountID, d.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV service: %w", err)
	}

	liveNamespaces, err := kvSvc.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list live KV namespaces: %w", err)
	}

	// BUG-048: refuse to diff against ambiguous duplicate titles — a title
	// collision makes downstream deletes destroy the wrong namespace.
	liveIdx, err := indexKVByTitle(liveNamespaces)
	if err != nil {
		return nil, validationError("DiffService.CompareKV", err.Error())
	}

	liveMap := make(map[string]bool, len(liveIdx))
	for title := range liveIdx {
		liveMap[title] = true
	}

	// Find additions (in local, not live)
	for _, ns := range local.Namespaces {
		if !liveMap[ns.Title] {
			result.Additions = append(result.Additions, DiffEntry{
				Action:   DiffAdd,
				Service:  "kv",
				Resource: ns.Title,
				Detail:   fmt.Sprintf("namespace %q in config but does not exist", ns.Title),
			})
		}
	}

	// Find deletions (live, not in local)
	localMap := make(map[string]bool, len(local.Namespaces))
	for _, ns := range local.Namespaces {
		localMap[ns.Title] = true
	}
	liveNames := make([]string, 0, len(liveMap))
	for name := range liveMap {
		liveNames = append(liveNames, name)
	}
	sort.Strings(liveNames)
	for _, name := range liveNames {
		if !localMap[name] {
			result.Deletions = append(result.Deletions, DiffEntry{
				Action:   DiffRemove,
				Service:  "kv",
				Resource: name,
				Detail:   fmt.Sprintf("namespace %q exists but not in config", name),
			})
		}
	}

	return result, nil
}

// CompareDNS compares local DNS record config against live state.
func (d *DiffService) CompareDNS(ctx context.Context, local DNSConfig) (*DiffResult, error) {
	result := &DiffResult{Service: "dns"}

	if local.ZoneID == "" {
		return nil, validationError("DiffService.CompareDNS", "zone_id is required in dns config")
	}

	dnsSvc, err := NewDNSServiceFromCreds(local.ZoneID, d.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS service: %w", err)
	}

	liveRecords, err := dnsSvc.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list live DNS records: %w", err)
	}

	// Build lookup key: type+name+content uniquely identifies a record
	type dnsKey struct {
		Type    string
		Name    string
		Content string
	}

	liveMap := make(map[dnsKey]*DNSRecord, len(liveRecords))
	for _, r := range liveRecords {
		k := dnsKey{Type: r.Type, Name: r.Name, Content: r.Content}
		liveMap[k] = r
	}

	localMap := make(map[dnsKey]DNSRecordConfig, len(local.Records))
	for _, r := range local.Records {
		k := dnsKey{Type: r.Type, Name: r.Name, Content: r.Content}
		localMap[k] = r
	}

	// Find additions (in local, not live)
	for _, r := range local.Records {
		k := dnsKey{Type: r.Type, Name: r.Name, Content: r.Content}
		if _, exists := liveMap[k]; !exists {
			result.Additions = append(result.Additions, DiffEntry{
				Action:   DiffAdd,
				Service:  "dns",
				Resource: fmt.Sprintf("%s %s %s", r.Type, r.Name, r.Content),
				Detail:   fmt.Sprintf("%s record %q -> %q in config but not live", r.Type, r.Name, r.Content),
			})
		}
	}

	// Find deletions (live, not in local)
	for k, r := range liveMap {
		if _, exists := localMap[k]; !exists {
			result.Deletions = append(result.Deletions, DiffEntry{
				Action:   DiffRemove,
				Service:  "dns",
				Resource: fmt.Sprintf("%s %s %s", r.Type, r.Name, r.Content),
				Detail:   fmt.Sprintf("%s record %q -> %q is live but not in config", r.Type, r.Name, r.Content),
			})
		}
	}

	// Find modifications (same key, different settings)
	for _, r := range local.Records {
		k := dnsKey{Type: r.Type, Name: r.Name, Content: r.Content}
		live, exists := liveMap[k]
		if !exists {
			continue
		}
		// Check for TTL or proxied differences
		changes := []string{}
		if r.TTL != 0 && r.TTL != live.TTL {
			changes = append(changes, fmt.Sprintf("ttl: %d -> %d", live.TTL, r.TTL))
		}
		if r.Proxied != live.Proxied {
			changes = append(changes, fmt.Sprintf("proxied: %v -> %v", live.Proxied, r.Proxied))
		}
		if len(changes) > 0 {
			result.Changes = append(result.Changes, DiffEntry{
				Action:   DiffModify,
				Service:  "dns",
				Resource: fmt.Sprintf("%s %s %s", r.Type, r.Name, r.Content),
				Detail:   fmt.Sprintf("changes: %s", joinStrings(changes, ", ")),
			})
		}
	}

	return result, nil
}

// CompareAll runs comparisons for all configured services and returns a summary.
func (d *DiffService) CompareAll(ctx context.Context, cfg *CosmoflareConfig) (*DiffSummary, error) {
	summary := &DiffSummary{}

	if len(cfg.Workers) > 0 {
		result, err := d.CompareWorkers(ctx, cfg.Workers)
		if err != nil {
			return nil, fmt.Errorf("workers diff failed: %w", err)
		}
		summary.Results = append(summary.Results, *result)
	}

	if len(cfg.R2.Buckets) > 0 {
		result, err := d.CompareR2(ctx, cfg.R2)
		if err != nil {
			return nil, fmt.Errorf("r2 diff failed: %w", err)
		}
		summary.Results = append(summary.Results, *result)
	}

	if len(cfg.KV.Namespaces) > 0 {
		result, err := d.CompareKV(ctx, cfg.KV)
		if err != nil {
			return nil, fmt.Errorf("kv diff failed: %w", err)
		}
		summary.Results = append(summary.Results, *result)
	}

	if cfg.DNS.ZoneID != "" {
		result, err := d.CompareDNS(ctx, cfg.DNS)
		if err != nil {
			return nil, fmt.Errorf("dns diff failed: %w", err)
		}
		summary.Results = append(summary.Results, *result)
	}

	for _, r := range summary.Results {
		summary.TotalAdd += len(r.Additions)
		summary.TotalDel += len(r.Deletions)
		summary.TotalMod += len(r.Changes)
	}
	summary.HasChanges = (summary.TotalAdd + summary.TotalDel + summary.TotalMod) > 0

	return summary, nil
}

// joinStrings joins a string slice with a separator (avoids importing strings).
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}
