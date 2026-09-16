package cosmoflare

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Constructor validation ---

func TestNewDiffServiceValidation(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		apiToken  string
		wantErr   bool
	}{
		{"both empty", "", "", true},
		{"empty account ID", "", "tok-123", true},
		{"empty API token", "acc-123", "", true},
		{"valid", "acc-123", "tok-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewDiffService(tt.accountID, tt.apiToken)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if svc != nil {
					t.Error("expected nil service on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if svc == nil {
					t.Error("expected non-nil service")
				}
			}
		})
	}
}

func TestNewDiffServiceFields(t *testing.T) {
	svc, err := NewDiffService("acc-xyz", "tok-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acc-xyz" {
		t.Errorf("accountID = %q, want %q", svc.accountID, "acc-xyz")
	}
	if svc.apiToken != "tok-abc" {
		t.Errorf("apiToken = %q, want %q", svc.apiToken, "tok-abc")
	}
}

// --- DiffResult methods ---

func TestDiffResult_TotalChanges(t *testing.T) {
	tests := []struct {
		name   string
		result DiffResult
		want   int
	}{
		{
			"empty",
			DiffResult{},
			0,
		},
		{
			"additions only",
			DiffResult{
				Additions: []DiffEntry{{Action: DiffAdd, Resource: "a"}},
			},
			1,
		},
		{
			"deletions only",
			DiffResult{
				Deletions: []DiffEntry{{Action: DiffRemove, Resource: "b"}},
			},
			1,
		},
		{
			"changes only",
			DiffResult{
				Changes: []DiffEntry{{Action: DiffModify, Resource: "c"}},
			},
			1,
		},
		{
			"mixed",
			DiffResult{
				Additions: []DiffEntry{{Action: DiffAdd, Resource: "a"}, {Action: DiffAdd, Resource: "b"}},
				Deletions: []DiffEntry{{Action: DiffRemove, Resource: "c"}},
				Changes:   []DiffEntry{{Action: DiffModify, Resource: "d"}, {Action: DiffModify, Resource: "e"}, {Action: DiffModify, Resource: "f"}},
			},
			6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.TotalChanges()
			if got != tt.want {
				t.Errorf("TotalChanges() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiffResult_HasChanges(t *testing.T) {
	empty := DiffResult{}
	if empty.HasChanges() {
		t.Error("empty result should not have changes")
	}

	withAdd := DiffResult{Additions: []DiffEntry{{Action: DiffAdd, Resource: "x"}}}
	if !withAdd.HasChanges() {
		t.Error("result with additions should have changes")
	}
}

// --- DiffSummary ---

func TestDiffSummary_Aggregation(t *testing.T) {
	summary := DiffSummary{
		Results: []DiffResult{
			{
				Service:   "workers",
				Additions: []DiffEntry{{Action: DiffAdd, Resource: "w1"}},
			},
			{
				Service:   "kv",
				Deletions: []DiffEntry{{Action: DiffRemove, Resource: "ns1"}, {Action: DiffRemove, Resource: "ns2"}},
			},
			{
				Service: "dns",
				Changes: []DiffEntry{{Action: DiffModify, Resource: "rec1"}},
			},
		},
		TotalAdd:   1,
		TotalDel:   2,
		TotalMod:   1,
		HasChanges: true,
	}

	if summary.TotalAdd != 1 {
		t.Errorf("TotalAdd = %d, want 1", summary.TotalAdd)
	}
	if summary.TotalDel != 2 {
		t.Errorf("TotalDel = %d, want 2", summary.TotalDel)
	}
	if summary.TotalMod != 1 {
		t.Errorf("TotalMod = %d, want 1", summary.TotalMod)
	}
	if !summary.HasChanges {
		t.Error("expected HasChanges to be true")
	}
}

// --- DiffAction constants ---

func TestDiffActionConstants(t *testing.T) {
	if DiffAdd != "add" {
		t.Errorf("DiffAdd = %q, want %q", DiffAdd, "add")
	}
	if DiffRemove != "remove" {
		t.Errorf("DiffRemove = %q, want %q", DiffRemove, "remove")
	}
	if DiffModify != "modify" {
		t.Errorf("DiffModify = %q, want %q", DiffModify, "modify")
	}
}

// --- DiffEntry fields ---

func TestDiffEntry_Fields(t *testing.T) {
	entry := DiffEntry{
		Action:   DiffAdd,
		Service:  "workers",
		Resource: "my-worker",
		Detail:   "worker \"my-worker\" exists in config but not deployed",
	}
	if entry.Action != DiffAdd {
		t.Errorf("Action = %q, want %q", entry.Action, DiffAdd)
	}
	if entry.Service != "workers" {
		t.Errorf("Service = %q, want %q", entry.Service, "workers")
	}
	if entry.Resource != "my-worker" {
		t.Errorf("Resource = %q, want %q", entry.Resource, "my-worker")
	}
	if entry.Detail == "" {
		t.Error("Detail should not be empty")
	}
}

// --- Config loading ---

func TestLoadCosmoflareConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadCosmoflareConfig(tmpDir)
	if err == nil {
		t.Error("expected error when .cosmoflare.yaml not found")
	}
}

// validFileConfigYAML is the full-featured config exercised by
// TestLoadCosmoflareConfig_ValidFile; keeping it at package level keeps the
// test function under the funlen limit without weakening assertions.
const validFileConfigYAML = `name: test-project
type: full

workers:
  api:
    script: src/index.js
    compatibility_date: "2024-01-01"
    module: true

r2:
  buckets:
    - name: assets
      location: enam

kv:
  namespaces:
    - title: MY_KV

dns:
  zone_id: zone-abc123
  records:
    - type: A
      name: www
      content: 1.2.3.4
      ttl: 300
      proxied: true
`

func TestLoadCosmoflareConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	if err := os.WriteFile(configPath, []byte(validFileConfigYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadCosmoflareConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("top-level name and type", func(t *testing.T) {
		if cfg.Name != "test-project" {
			t.Errorf("Name = %q, want %q", cfg.Name, "test-project")
		}
		if cfg.Type != "full" {
			t.Errorf("Type = %q, want %q", cfg.Type, "full")
		}
	})

	t.Run("worker api", func(t *testing.T) {
		if len(cfg.Workers) != 1 {
			t.Fatalf("expected 1 worker, got %d", len(cfg.Workers))
		}
		w, ok := cfg.Workers["api"]
		if !ok {
			t.Fatal("expected worker 'api' in config")
		}
		if w.Script != "src/index.js" {
			t.Errorf("Worker script = %q, want %q", w.Script, "src/index.js")
		}
		if !w.Module {
			t.Error("expected Worker module to be true")
		}
	})

	t.Run("r2 buckets", func(t *testing.T) {
		if len(cfg.R2.Buckets) != 1 {
			t.Fatalf("expected 1 R2 bucket, got %d", len(cfg.R2.Buckets))
		}
		if cfg.R2.Buckets[0].Name != "assets" {
			t.Errorf("R2 bucket name = %q, want %q", cfg.R2.Buckets[0].Name, "assets")
		}
	})

	t.Run("kv namespaces", func(t *testing.T) {
		if len(cfg.KV.Namespaces) != 1 {
			t.Fatalf("expected 1 KV namespace, got %d", len(cfg.KV.Namespaces))
		}
		if cfg.KV.Namespaces[0].Title != "MY_KV" {
			t.Errorf("KV namespace title = %q, want %q", cfg.KV.Namespaces[0].Title, "MY_KV")
		}
	})

	t.Run("dns zone and records", func(t *testing.T) {
		if cfg.DNS.ZoneID != "zone-abc123" {
			t.Errorf("DNS zone ID = %q, want %q", cfg.DNS.ZoneID, "zone-abc123")
		}
		if len(cfg.DNS.Records) != 1 {
			t.Fatalf("expected 1 DNS record, got %d", len(cfg.DNS.Records))
		}
		rec := cfg.DNS.Records[0]
		if rec.Type != "A" || rec.Name != "www" || rec.Content != "1.2.3.4" {
			t.Errorf("DNS record = %+v, want A www 1.2.3.4", rec)
		}
		if rec.TTL != 300 {
			t.Errorf("DNS record TTL = %d, want 300", rec.TTL)
		}
		if !rec.Proxied {
			t.Error("expected DNS record proxied to be true")
		}
	})
}

func TestLoadCosmoflareConfig_EmptyWorkers(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".cosmoflare.yaml")
	content := `name: minimal
type: r2

r2:
  buckets:
    - name: my-bucket
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadCosmoflareConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Workers) != 0 {
		t.Errorf("expected 0 workers, got %d", len(cfg.Workers))
	}
	if len(cfg.R2.Buckets) != 1 {
		t.Errorf("expected 1 R2 bucket, got %d", len(cfg.R2.Buckets))
	}
}

// --- joinStrings ---

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		sep   string
		want  string
	}{
		{"empty", nil, ", ", ""},
		{"single", []string{"a"}, ", ", "a"},
		{"multiple", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"dash sep", []string{"x", "y"}, "-", "x-y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinStrings(tt.parts, tt.sep)
			if got != tt.want {
				t.Errorf("joinStrings(%v, %q) = %q, want %q", tt.parts, tt.sep, got, tt.want)
			}
		})
	}
}

// --- CosmoflareConfig type structs ---

func TestCosmoflareConfig_WorkerConfig(t *testing.T) {
	wc := WorkerConfig{
		Script:            "worker.js",
		CompatibilityDate: "2024-06-01",
		Module:            true,
	}
	if wc.Script != "worker.js" {
		t.Errorf("Script = %q, want %q", wc.Script, "worker.js")
	}
	if wc.CompatibilityDate != "2024-06-01" {
		t.Errorf("CompatibilityDate = %q, want %q", wc.CompatibilityDate, "2024-06-01")
	}
	if !wc.Module {
		t.Error("expected Module to be true")
	}
}

func TestCosmoflareConfig_R2BucketConfig(t *testing.T) {
	bc := R2BucketConfig{Name: "images", Location: "wnam"}
	if bc.Name != "images" {
		t.Errorf("Name = %q, want %q", bc.Name, "images")
	}
	if bc.Location != "wnam" {
		t.Errorf("Location = %q, want %q", bc.Location, "wnam")
	}
}

func TestCosmoflareConfig_KVNamespaceConfig(t *testing.T) {
	kc := KVNamespaceConfig{Title: "MY_CACHE"}
	if kc.Title != "MY_CACHE" {
		t.Errorf("Title = %q, want %q", kc.Title, "MY_CACHE")
	}
}

func TestCosmoflareConfig_DNSRecordConfig(t *testing.T) {
	rc := DNSRecordConfig{
		Type:    "CNAME",
		Name:    "blog",
		Content: "blog.example.com",
		TTL:     3600,
		Proxied: true,
	}
	if rc.Type != "CNAME" {
		t.Errorf("Type = %q, want %q", rc.Type, "CNAME")
	}
	if rc.Name != "blog" {
		t.Errorf("Name = %q, want %q", rc.Name, "blog")
	}
	if rc.Content != "blog.example.com" {
		t.Errorf("Content = %q, want %q", rc.Content, "blog.example.com")
	}
	if rc.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", rc.TTL)
	}
	if !rc.Proxied {
		t.Error("expected Proxied to be true")
	}
}

// --- BUG-048: duplicate KV namespace titles must be refused, not guessed ---

func TestIndexKVByTitle_RefusesDuplicateTitles(t *testing.T) {
	namespaces := []*KVNamespace{
		{ID: "ns-aaa", Title: "prod-cache"},
		{ID: "ns-bbb", Title: "prod-cache"},
	}
	idx, err := indexKVByTitle(namespaces)
	if err == nil {
		t.Fatalf("expected ambiguity error for duplicate titles, got index %v", idx)
	}
	if !strings.Contains(err.Error(), "ns-aaa") || !strings.Contains(err.Error(), "ns-bbb") {
		t.Errorf("error must name both conflicting IDs so the user can disambiguate, got: %v", err)
	}
}

func TestIndexKVByTitle_UniqueTitles(t *testing.T) {
	namespaces := []*KVNamespace{
		{ID: "ns-aaa", Title: "prod-cache"},
		{ID: "ns-bbb", Title: "staging-cache"},
	}
	idx, err := indexKVByTitle(namespaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx["prod-cache"] != "ns-aaa" || idx["staging-cache"] != "ns-bbb" {
		t.Errorf("index = %v, want title->ID mapping", idx)
	}
}

// --- dnsKeyOf / dnsLiveIndex / dnsLocalIndex (pure helpers) ---

func TestDnsKeyOf(t *testing.T) {
	rec := DNSRecordConfig{Type: "A", Name: "www.example.com", Content: "1.2.3.4", TTL: 300, Proxied: true}
	key := dnsKeyOf(rec)
	if key.Type != "A" || key.Name != "www.example.com" || key.Content != "1.2.3.4" {
		t.Errorf("key = %+v, want {A www.example.com 1.2.3.4}", key)
	}
}

func TestDnsLiveIndex(t *testing.T) {
	records := []*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"},
		{Type: "AAAA", Name: "aaaa.example.com", Content: "::1"},
	}
	idx := dnsLiveIndex(records)
	if len(idx) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx))
	}
	if idx[dnsKey{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}] == nil {
		t.Error("expected A record indexed")
	}
}

func TestDnsLiveIndex_Empty(t *testing.T) {
	idx := dnsLiveIndex(nil)
	if len(idx) != 0 {
		t.Errorf("expected empty index, got %d entries", len(idx))
	}
}

func TestDnsLocalIndex(t *testing.T) {
	local := []DNSRecordConfig{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"},
		{Type: "CNAME", Name: "blog.example.com", Content: "example.com"},
	}
	idx := dnsLocalIndex(local)
	if len(idx) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx))
	}
	rec, ok := idx[dnsKey{Type: "CNAME", Name: "blog.example.com", Content: "example.com"}]
	if !ok {
		t.Fatal("expected CNAME record indexed")
	}
	if rec.Name != "blog.example.com" {
		t.Errorf("Name = %q, want %q", rec.Name, "blog.example.com")
	}
}

// --- dnsDiffAdditions / dnsDiffDeletions / dnsDiffModifications (pure helpers) ---

func TestDnsDiffAdditions(t *testing.T) {
	local := []DNSRecordConfig{
		{Type: "A", Name: "new.example.com", Content: "1.2.3.4"},
		{Type: "A", Name: "existing.example.com", Content: "5.6.7.8"},
	}
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "existing.example.com", Content: "5.6.7.8"}: {Type: "A", Name: "existing.example.com", Content: "5.6.7.8"},
	}
	result := &DiffResult{}
	dnsDiffAdditions(result, local, liveMap)
	if len(result.Additions) != 1 {
		t.Fatalf("expected 1 addition, got %d", len(result.Additions))
	}
	if result.Additions[0].Resource != "A new.example.com 1.2.3.4" {
		t.Errorf("Resource = %q, want %q", result.Additions[0].Resource, "A new.example.com 1.2.3.4")
	}
	if result.Additions[0].Action != DiffAdd {
		t.Errorf("Action = %q, want %q", result.Additions[0].Action, DiffAdd)
	}
}

func TestDnsDiffAdditions_NoneWhenAllLive(t *testing.T) {
	local := []DNSRecordConfig{{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}}
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1"},
	}
	result := &DiffResult{}
	dnsDiffAdditions(result, local, liveMap)
	if len(result.Additions) != 0 {
		t.Errorf("expected 0 additions, got %d", len(result.Additions))
	}
}

func TestDnsDiffDeletions(t *testing.T) {
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "stray.example.com", Content: "9.9.9.9"}: {Type: "A", Name: "stray.example.com", Content: "9.9.9.9"},
		{Type: "A", Name: "kept.example.com", Content: "1.1.1.1"}:  {Type: "A", Name: "kept.example.com", Content: "1.1.1.1"},
	}
	localMap := map[dnsKey]DNSRecordConfig{
		{Type: "A", Name: "kept.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "kept.example.com", Content: "1.1.1.1"},
	}
	result := &DiffResult{}
	dnsDiffDeletions(result, liveMap, localMap)
	if len(result.Deletions) != 1 {
		t.Fatalf("expected 1 deletion, got %d", len(result.Deletions))
	}
	if result.Deletions[0].Resource != "A stray.example.com 9.9.9.9" {
		t.Errorf("Resource = %q, want %q", result.Deletions[0].Resource, "A stray.example.com 9.9.9.9")
	}
	if result.Deletions[0].Action != DiffRemove {
		t.Errorf("Action = %q, want %q", result.Deletions[0].Action, DiffRemove)
	}
}

func TestDnsDiffDeletions_NoneWhenAllLocal(t *testing.T) {
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1"},
	}
	localMap := map[dnsKey]DNSRecordConfig{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1"},
	}
	result := &DiffResult{}
	dnsDiffDeletions(result, liveMap, localMap)
	if len(result.Deletions) != 0 {
		t.Errorf("expected 0 deletions, got %d", len(result.Deletions))
	}
}

func TestDnsDiffModifications_TTLChange(t *testing.T) {
	local := []DNSRecordConfig{{Type: "A", Name: "a.example.com", Content: "1.1.1.1", TTL: 600}}
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1", TTL: 300},
	}
	result := &DiffResult{}
	dnsDiffModifications(result, local, liveMap)
	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	if !strings.Contains(result.Changes[0].Detail, "ttl: 300 -> 600") {
		t.Errorf("Detail = %q, want ttl change mentioned", result.Changes[0].Detail)
	}
}

func TestDnsDiffModifications_ProxiedChange(t *testing.T) {
	local := []DNSRecordConfig{{Type: "A", Name: "a.example.com", Content: "1.1.1.1", Proxied: true}}
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1", Proxied: false},
	}
	result := &DiffResult{}
	dnsDiffModifications(result, local, liveMap)
	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	if !strings.Contains(result.Changes[0].Detail, "proxied: false -> true") {
		t.Errorf("Detail = %q, want proxied change mentioned", result.Changes[0].Detail)
	}
}

func TestDnsDiffModifications_NoneWhenUnchanged(t *testing.T) {
	local := []DNSRecordConfig{{Type: "A", Name: "a.example.com", Content: "1.1.1.1", TTL: 300, Proxied: true}}
	liveMap := map[dnsKey]*DNSRecord{
		{Type: "A", Name: "a.example.com", Content: "1.1.1.1"}: {Type: "A", Name: "a.example.com", Content: "1.1.1.1", TTL: 300, Proxied: true},
	}
	result := &DiffResult{}
	dnsDiffModifications(result, local, liveMap)
	if len(result.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(result.Changes))
	}
}

func TestDnsDiffModifications_NotLive(t *testing.T) {
	local := []DNSRecordConfig{{Type: "A", Name: "ghost.example.com", Content: "9.9.9.9", TTL: 300}}
	result := &DiffResult{}
	dnsDiffModifications(result, local, map[dnsKey]*DNSRecord{})
	if len(result.Changes) != 0 {
		t.Errorf("expected 0 changes for record not live, got %d", len(result.Changes))
	}
}

// --- CompareAll / CompareWorkers / CompareR2 / CompareKV / CompareDNS ---
//
// DiffService always constructs its own live client from credentials, with
// no seam for a mock server. An empty config short-circuits CompareAll
// without touching the network. An already-canceled context makes the
// underlying HTTP/AWS SDK calls fail immediately (no real network I/O) so
// the remaining Compare* functions can be exercised deterministically.

func TestCompareAll_EmptyConfig(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary, err := d.CompareAll(context.Background(), &CosmoflareConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if len(summary.Results) != 0 {
		t.Errorf("expected 0 results for empty config, got %d", len(summary.Results))
	}
	if summary.HasChanges {
		t.Error("expected HasChanges=false for empty config")
	}
}

func diffCanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCompareAll_WorkersErrorPropagates(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := &CosmoflareConfig{Workers: map[string]WorkerConfig{"api": {Script: "w.js"}}}
	_, err = d.CompareAll(diffCanceledContext(), cfg)
	if err == nil {
		t.Fatal("expected error when workers diff fails")
	}
	if !strings.Contains(err.Error(), "workers diff failed") {
		t.Errorf("error = %v, want prefix about workers diff failure", err)
	}
}

func TestCompareWorkers_NetworkError(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = d.CompareWorkers(diffCanceledContext(), map[string]WorkerConfig{"api": {Script: "w.js"}})
	if err == nil {
		t.Fatal("expected error when listing live workers fails")
	}
}

func TestCompareR2_NetworkError(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = d.CompareR2(diffCanceledContext(), R2Config{Buckets: []R2BucketConfig{{Name: "assets"}}})
	if err == nil {
		t.Fatal("expected error when listing live buckets fails")
	}
}

func TestCompareKV_NetworkError(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = d.CompareKV(diffCanceledContext(), KVConfig{Namespaces: []KVNamespaceConfig{{Title: "MY_KV"}}})
	if err == nil {
		t.Fatal("expected error when listing live namespaces fails")
	}
}

func TestCompareDNS_ZoneIDRequired(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = d.CompareDNS(context.Background(), DNSConfig{})
	if err == nil {
		t.Fatal("expected error when zone_id is missing")
	}
}

func TestCompareDNS_NetworkError(t *testing.T) {
	d, err := NewDiffService("acc-123", "tok-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = d.CompareDNS(diffCanceledContext(), DNSConfig{ZoneID: "zone-1"})
	if err == nil {
		t.Fatal("expected error when listing live DNS records fails")
	}
	if !strings.Contains(err.Error(), "failed to list live DNS records") {
		t.Errorf("error = %v, want prefix about failed live-record listing", err)
	}
}
