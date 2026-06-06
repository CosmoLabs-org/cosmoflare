package cosmoflare

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// WranglerConfig represents a parsed wrangler.toml file.
type WranglerConfig struct {
	Name              string                       `toml:"name" json:"name"`
	Main              string                       `toml:"main" json:"main,omitempty"`
	CompatibilityDate string                       `toml:"compatibility_date" json:"compatibility_date,omitempty"`
	CompatibilityFlags []string                    `toml:"compatibility_flags" json:"compatibility_flags,omitempty"`
	AccountID         string                       `toml:"account_id" json:"account_id,omitempty"`
	Route             string                       `toml:"route" json:"route,omitempty"`
	Routes            []string                     `toml:"routes" json:"routes,omitempty"`
	Workers_Dev       bool                         `toml:"workers_dev" json:"workers_dev,omitempty"`
	KVNamespaces      []WranglerKVNamespace        `toml:"kv_namespaces" json:"kv_namespaces,omitempty"`
	R2Buckets         []WranglerR2Bucket           `toml:"r2_buckets" json:"r2_buckets,omitempty"`
	D1Databases       []WranglerD1Database         `toml:"d1_databases" json:"d1_databases,omitempty"`
	Vars              map[string]string            `toml:"vars" json:"vars,omitempty"`
	Env               map[string]*WranglerEnvConfig `toml:"-" json:"env,omitempty"`
}

// WranglerKVNamespace represents a KV namespace binding in wrangler.toml.
type WranglerKVNamespace struct {
	Binding   string `toml:"binding" json:"binding"`
	ID        string `toml:"id" json:"id"`
	PreviewID string `toml:"preview_id" json:"preview_id,omitempty"`
}

// WranglerR2Bucket represents an R2 bucket binding in wrangler.toml.
type WranglerR2Bucket struct {
	Binding    string `toml:"binding" json:"binding"`
	BucketName string `toml:"bucket_name" json:"bucket_name"`
	PreviewBucketName string `toml:"preview_bucket_name" json:"preview_bucket_name,omitempty"`
}

// WranglerD1Database represents a D1 database binding in wrangler.toml.
type WranglerD1Database struct {
	Binding      string `toml:"binding" json:"binding"`
	DatabaseName string `toml:"database_name" json:"database_name"`
	DatabaseID   string `toml:"database_id" json:"database_id"`
}

// WranglerEnvConfig represents an environment-specific override in wrangler.toml.
type WranglerEnvConfig struct {
	Name              string                `toml:"name" json:"name,omitempty"`
	Main              string                `toml:"main" json:"main,omitempty"`
	CompatibilityDate string                `toml:"compatibility_date" json:"compatibility_date,omitempty"`
	Route             string                `toml:"route" json:"route,omitempty"`
	Routes            []string              `toml:"routes" json:"routes,omitempty"`
	Vars              map[string]string     `toml:"vars" json:"vars,omitempty"`
	KVNamespaces      []WranglerKVNamespace `toml:"kv_namespaces" json:"kv_namespaces,omitempty"`
	R2Buckets         []WranglerR2Bucket    `toml:"r2_buckets" json:"r2_buckets,omitempty"`
	D1Databases       []WranglerD1Database  `toml:"d1_databases" json:"d1_databases,omitempty"`
}

// WranglerImportResult is the output of ConvertToConfig — a YAML-friendly
// representation of the full .cosmoflare.yaml including D1 and profiles
// which the existing CosmoflareConfig (diff.go) does not yet cover.
type WranglerImportResult struct {
	Version  string                        `yaml:"version" json:"version"`
	Name     string                        `yaml:"name" json:"name"`
	Workers  map[string]WranglerWorkerYAML `yaml:"workers,omitempty" json:"workers,omitempty"`
	KV       *WranglerKVYAML               `yaml:"kv,omitempty" json:"kv,omitempty"`
	R2       *WranglerR2YAML               `yaml:"r2,omitempty" json:"r2,omitempty"`
	D1       *WranglerD1YAML               `yaml:"d1,omitempty" json:"d1,omitempty"`
	Profiles map[string]*WranglerProfileYAML `yaml:"profiles,omitempty" json:"profiles,omitempty"`
}

// WranglerWorkerYAML represents a worker entry in the generated .cosmoflare.yaml.
type WranglerWorkerYAML struct {
	Name              string            `yaml:"name,omitempty" json:"name,omitempty"`
	Script            string            `yaml:"script,omitempty" json:"script,omitempty"`
	CompatibilityDate string            `yaml:"compatibility_date,omitempty" json:"compatibility_date,omitempty"`
	CompatibilityFlags []string         `yaml:"compatibility_flags,omitempty" json:"compatibility_flags,omitempty"`
	Route             string            `yaml:"route,omitempty" json:"route,omitempty"`
	Routes            []string          `yaml:"routes,omitempty" json:"routes,omitempty"`
	Vars              map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Bindings          *WranglerBindingsYAML `yaml:"bindings,omitempty" json:"bindings,omitempty"`
}

// WranglerBindingsYAML groups KV, R2, and D1 bindings for a worker.
type WranglerBindingsYAML struct {
	KV []WranglerBindingRef `yaml:"kv,omitempty" json:"kv,omitempty"`
	R2 []WranglerBindingRef `yaml:"r2,omitempty" json:"r2,omitempty"`
	D1 []WranglerBindingRef `yaml:"d1,omitempty" json:"d1,omitempty"`
}

// WranglerBindingRef references a resource binding by name and ID.
type WranglerBindingRef struct {
	Binding string `yaml:"binding" json:"binding"`
	ID      string `yaml:"id,omitempty" json:"id,omitempty"`
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
}

// WranglerKVYAML represents the KV section of the generated .cosmoflare.yaml.
type WranglerKVYAML struct {
	Namespaces []WranglerKVNamespaceYAML `yaml:"namespaces" json:"namespaces"`
}

// WranglerKVNamespaceYAML represents a KV namespace in the generated .cosmoflare.yaml.
type WranglerKVNamespaceYAML struct {
	Binding   string `yaml:"binding" json:"binding"`
	ID        string `yaml:"id" json:"id"`
	PreviewID string `yaml:"preview_id,omitempty" json:"preview_id,omitempty"`
}

// WranglerR2YAML represents the R2 section of the generated .cosmoflare.yaml.
type WranglerR2YAML struct {
	Buckets []WranglerR2BucketYAML `yaml:"buckets" json:"buckets"`
}

// WranglerR2BucketYAML represents an R2 bucket in the generated .cosmoflare.yaml.
type WranglerR2BucketYAML struct {
	Binding    string `yaml:"binding" json:"binding"`
	BucketName string `yaml:"bucket_name" json:"bucket_name"`
}

// WranglerD1YAML represents the D1 section of the generated .cosmoflare.yaml.
type WranglerD1YAML struct {
	Databases []WranglerD1DatabaseYAML `yaml:"databases" json:"databases"`
}

// WranglerD1DatabaseYAML represents a D1 database in the generated .cosmoflare.yaml.
type WranglerD1DatabaseYAML struct {
	Binding      string `yaml:"binding" json:"binding"`
	DatabaseName string `yaml:"database_name" json:"database_name"`
	DatabaseID   string `yaml:"database_id" json:"database_id"`
}

// WranglerProfileYAML represents an environment profile in the generated .cosmoflare.yaml.
type WranglerProfileYAML struct {
	Worker *WranglerWorkerYAML `yaml:"worker,omitempty" json:"worker,omitempty"`
}

// WranglerService provides wrangler.toml parsing and conversion.
type WranglerService struct{}

// NewWranglerService creates a new WranglerService.
func NewWranglerService() *WranglerService {
	return &WranglerService{}
}

// ParseToml reads and parses a wrangler.toml file from disk.
func (ws *WranglerService) ParseToml(path string) (*WranglerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read wrangler.toml: %w", err)
	}
	return ws.ParseTomlBytes(data)
}

// ParseTomlBytes parses wrangler.toml content from bytes.
func (ws *WranglerService) ParseTomlBytes(data []byte) (*WranglerConfig, error) {
	var cfg WranglerConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse wrangler.toml: %w", err)
	}

	// Parse [env.*] sections from raw map
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse wrangler.toml environments: %w", err)
	}

	if envMap, ok := raw["env"].(map[string]interface{}); ok {
		cfg.Env = make(map[string]*WranglerEnvConfig)
		for envName, envData := range envMap {
			envCfg, err := parseEnvConfig(envData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse env.%s: %w", envName, err)
			}
			cfg.Env[envName] = envCfg
		}
	}

	return &cfg, nil
}

// parseEnvConfig converts a raw env map into a WranglerEnvConfig.
func parseEnvConfig(data interface{}) (*WranglerEnvConfig, error) {
	envMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", data)
	}

	cfg := &WranglerEnvConfig{}

	if v, ok := envMap["name"].(string); ok {
		cfg.Name = v
	}
	if v, ok := envMap["main"].(string); ok {
		cfg.Main = v
	}
	if v, ok := envMap["compatibility_date"].(string); ok {
		cfg.CompatibilityDate = v
	}
	if v, ok := envMap["route"].(string); ok {
		cfg.Route = v
	}
	if routes, ok := envMap["routes"].([]interface{}); ok {
		for _, r := range routes {
			if s, ok := r.(string); ok {
				cfg.Routes = append(cfg.Routes, s)
			}
		}
	}
	if vars, ok := envMap["vars"].(map[string]interface{}); ok {
		cfg.Vars = make(map[string]string)
		for k, v := range vars {
			cfg.Vars[k] = fmt.Sprintf("%v", v)
		}
	}

	// Parse kv_namespaces
	if kvList, ok := envMap["kv_namespaces"].([]interface{}); ok {
		for _, item := range kvList {
			if m, ok := item.(map[string]interface{}); ok {
				ns := WranglerKVNamespace{}
				if v, ok := m["binding"].(string); ok {
					ns.Binding = v
				}
				if v, ok := m["id"].(string); ok {
					ns.ID = v
				}
				if v, ok := m["preview_id"].(string); ok {
					ns.PreviewID = v
				}
				cfg.KVNamespaces = append(cfg.KVNamespaces, ns)
			}
		}
	}

	// Parse r2_buckets
	if r2List, ok := envMap["r2_buckets"].([]interface{}); ok {
		for _, item := range r2List {
			if m, ok := item.(map[string]interface{}); ok {
				b := WranglerR2Bucket{}
				if v, ok := m["binding"].(string); ok {
					b.Binding = v
				}
				if v, ok := m["bucket_name"].(string); ok {
					b.BucketName = v
				}
				cfg.R2Buckets = append(cfg.R2Buckets, b)
			}
		}
	}

	// Parse d1_databases
	if d1List, ok := envMap["d1_databases"].([]interface{}); ok {
		for _, item := range d1List {
			if m, ok := item.(map[string]interface{}); ok {
				db := WranglerD1Database{}
				if v, ok := m["binding"].(string); ok {
					db.Binding = v
				}
				if v, ok := m["database_name"].(string); ok {
					db.DatabaseName = v
				}
				if v, ok := m["database_id"].(string); ok {
					db.DatabaseID = v
				}
				cfg.D1Databases = append(cfg.D1Databases, db)
			}
		}
	}

	return cfg, nil
}

// ConvertToConfig transforms a WranglerConfig into a WranglerImportResult
// suitable for writing as .cosmoflare.yaml.
func (ws *WranglerService) ConvertToConfig(wc *WranglerConfig) *WranglerImportResult {
	cc := &WranglerImportResult{
		Version: "1",
		Name:    wc.Name,
		Workers: make(map[string]WranglerWorkerYAML),
	}

	// Main worker
	mainWorker := WranglerWorkerYAML{
		Name:              wc.Name,
		Script:            wc.Main,
		CompatibilityDate: wc.CompatibilityDate,
	}
	if len(wc.CompatibilityFlags) > 0 {
		mainWorker.CompatibilityFlags = wc.CompatibilityFlags
	}
	if wc.Route != "" {
		mainWorker.Route = wc.Route
	}
	if len(wc.Routes) > 0 {
		mainWorker.Routes = wc.Routes
	}
	if len(wc.Vars) > 0 {
		mainWorker.Vars = wc.Vars
	}

	// Attach bindings to the main worker
	bindings := &WranglerBindingsYAML{}
	hasBindings := false

	if len(wc.KVNamespaces) > 0 {
		hasBindings = true
		for _, kv := range wc.KVNamespaces {
			bindings.KV = append(bindings.KV, WranglerBindingRef{
				Binding: kv.Binding,
				ID:      kv.ID,
			})
		}
	}
	if len(wc.R2Buckets) > 0 {
		hasBindings = true
		for _, r2 := range wc.R2Buckets {
			bindings.R2 = append(bindings.R2, WranglerBindingRef{
				Binding: r2.Binding,
				Name:    r2.BucketName,
			})
		}
	}
	if len(wc.D1Databases) > 0 {
		hasBindings = true
		for _, d1 := range wc.D1Databases {
			bindings.D1 = append(bindings.D1, WranglerBindingRef{
				Binding: d1.Binding,
				ID:      d1.DatabaseID,
				Name:    d1.DatabaseName,
			})
		}
	}
	if hasBindings {
		mainWorker.Bindings = bindings
	}

	cc.Workers["main"] = mainWorker

	// KV namespaces (top-level)
	if len(wc.KVNamespaces) > 0 {
		cc.KV = &WranglerKVYAML{}
		for _, kv := range wc.KVNamespaces {
			cc.KV.Namespaces = append(cc.KV.Namespaces, WranglerKVNamespaceYAML{
				Binding:   kv.Binding,
				ID:        kv.ID,
				PreviewID: kv.PreviewID,
			})
		}
	}

	// R2 buckets (top-level)
	if len(wc.R2Buckets) > 0 {
		cc.R2 = &WranglerR2YAML{}
		for _, r2 := range wc.R2Buckets {
			cc.R2.Buckets = append(cc.R2.Buckets, WranglerR2BucketYAML{
				Binding:    r2.Binding,
				BucketName: r2.BucketName,
			})
		}
	}

	// D1 databases (top-level)
	if len(wc.D1Databases) > 0 {
		cc.D1 = &WranglerD1YAML{}
		for _, d1 := range wc.D1Databases {
			cc.D1.Databases = append(cc.D1.Databases, WranglerD1DatabaseYAML{
				Binding:      d1.Binding,
				DatabaseName: d1.DatabaseName,
				DatabaseID:   d1.DatabaseID,
			})
		}
	}

	// Environment profiles
	if len(wc.Env) > 0 {
		cc.Profiles = make(map[string]*WranglerProfileYAML)
		envNames := make([]string, 0, len(wc.Env))
		for name := range wc.Env {
			envNames = append(envNames, name)
		}
		sort.Strings(envNames)

		for _, envName := range envNames {
			env := wc.Env[envName]
			profile := &WranglerProfileYAML{
				Worker: &WranglerWorkerYAML{},
			}
			if env.Name != "" {
				profile.Worker.Name = env.Name
			}
			if env.Main != "" {
				profile.Worker.Script = env.Main
			}
			if env.CompatibilityDate != "" {
				profile.Worker.CompatibilityDate = env.CompatibilityDate
			}
			if env.Route != "" {
				profile.Worker.Route = env.Route
			}
			if len(env.Routes) > 0 {
				profile.Worker.Routes = env.Routes
			}
			if len(env.Vars) > 0 {
				profile.Worker.Vars = env.Vars
			}

			// Env-specific bindings
			envBindings := &WranglerBindingsYAML{}
			hasEnvBindings := false
			if len(env.KVNamespaces) > 0 {
				hasEnvBindings = true
				for _, kv := range env.KVNamespaces {
					envBindings.KV = append(envBindings.KV, WranglerBindingRef{
						Binding: kv.Binding,
						ID:      kv.ID,
					})
				}
			}
			if len(env.R2Buckets) > 0 {
				hasEnvBindings = true
				for _, r2 := range env.R2Buckets {
					envBindings.R2 = append(envBindings.R2, WranglerBindingRef{
						Binding: r2.Binding,
						Name:    r2.BucketName,
					})
				}
			}
			if len(env.D1Databases) > 0 {
				hasEnvBindings = true
				for _, d1 := range env.D1Databases {
					envBindings.D1 = append(envBindings.D1, WranglerBindingRef{
						Binding: d1.Binding,
						ID:      d1.DatabaseID,
						Name:    d1.DatabaseName,
					})
				}
			}
			if hasEnvBindings {
				profile.Worker.Bindings = envBindings
			}

			cc.Profiles[envName] = profile
		}
	}

	return cc
}

// Validate checks a WranglerConfig for common issues and returns a list of warnings/errors.
func (ws *WranglerService) Validate(wc *WranglerConfig) []WranglerValidationResult {
	var results []WranglerValidationResult

	if wc.Name == "" {
		results = append(results, WranglerValidationResult{
			Level:   "error",
			Field:   "name",
			Message: "worker name is required",
		})
	}

	if wc.Main == "" {
		results = append(results, WranglerValidationResult{
			Level:   "warning",
			Field:   "main",
			Message: "no entry point (main) specified",
		})
	}

	if wc.CompatibilityDate == "" {
		results = append(results, WranglerValidationResult{
			Level:   "warning",
			Field:   "compatibility_date",
			Message: "no compatibility_date set; Cloudflare may use an old default",
		})
	}

	// Validate KV namespaces
	for i, kv := range wc.KVNamespaces {
		if kv.Binding == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("kv_namespaces[%d].binding", i),
				Message: "KV namespace binding name is required",
			})
		}
		if kv.ID == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("kv_namespaces[%d].id", i),
				Message: fmt.Sprintf("KV namespace %q is missing an ID", kv.Binding),
			})
		}
	}

	// Validate R2 buckets
	for i, r2 := range wc.R2Buckets {
		if r2.Binding == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("r2_buckets[%d].binding", i),
				Message: "R2 bucket binding name is required",
			})
		}
		if r2.BucketName == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("r2_buckets[%d].bucket_name", i),
				Message: fmt.Sprintf("R2 bucket %q is missing bucket_name", r2.Binding),
			})
		}
	}

	// Validate D1 databases
	for i, d1 := range wc.D1Databases {
		if d1.Binding == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("d1_databases[%d].binding", i),
				Message: "D1 database binding name is required",
			})
		}
		if d1.DatabaseID == "" {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   fmt.Sprintf("d1_databases[%d].database_id", i),
				Message: fmt.Sprintf("D1 database %q is missing database_id", d1.Binding),
			})
		}
	}

	// Check for duplicate bindings
	seen := make(map[string]string) // binding -> source
	for _, kv := range wc.KVNamespaces {
		if prev, ok := seen[kv.Binding]; ok {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   "bindings",
				Message: fmt.Sprintf("duplicate binding name %q (used by %s and kv_namespaces)", kv.Binding, prev),
			})
		}
		seen[kv.Binding] = "kv_namespaces"
	}
	for _, r2 := range wc.R2Buckets {
		if prev, ok := seen[r2.Binding]; ok {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   "bindings",
				Message: fmt.Sprintf("duplicate binding name %q (used by %s and r2_buckets)", r2.Binding, prev),
			})
		}
		seen[r2.Binding] = "r2_buckets"
	}
	for _, d1 := range wc.D1Databases {
		if prev, ok := seen[d1.Binding]; ok {
			results = append(results, WranglerValidationResult{
				Level:   "error",
				Field:   "bindings",
				Message: fmt.Sprintf("duplicate binding name %q (used by %s and d1_databases)", d1.Binding, prev),
			})
		}
		seen[d1.Binding] = "d1_databases"
	}

	return results
}

// WranglerValidationResult represents a single validation finding.
type WranglerValidationResult struct {
	Level   string `json:"level"`   // "error" or "warning"
	Field   string `json:"field"`   // dotted field path
	Message string `json:"message"` // human-readable message
}

// WranglerHasErrors returns true if any result has level "error".
func WranglerHasErrors(results []WranglerValidationResult) bool {
	for _, r := range results {
		if r.Level == "error" {
			return true
		}
	}
	return false
}

// MarshalWranglerImportYAML renders a WranglerImportResult as YAML bytes.
func MarshalWranglerImportYAML(cfg *WranglerImportResult) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cosmoflare config: %w", err)
	}
	return data, nil
}

// WriteWranglerImportYAML writes a WranglerImportResult to a .cosmoflare.yaml file.
func WriteWranglerImportYAML(cfg *WranglerImportResult, path string) error {
	data, err := MarshalWranglerImportYAML(cfg)
	if err != nil {
		return err
	}

	header := "# Generated by cosmoflare from wrangler.toml\n# See: cosmoflare wrangler import --help\n\n"
	return os.WriteFile(path, append([]byte(header), data...), 0644)
}

// WranglerDiffItem represents a difference between wrangler.toml and .cosmoflare.yaml.
type WranglerDiffItem struct {
	Field    string `json:"field"`
	Wrangler string `json:"wrangler"` // value from wrangler.toml
	Cosmo    string `json:"cosmo"`    // value from .cosmoflare.yaml
	Status   string `json:"status"`   // "added", "removed", "changed", "match"
}

// DiffWranglerConfigs compares a WranglerConfig against an existing WranglerImportResult
// and returns a list of differences.
func (ws *WranglerService) DiffWranglerConfigs(wc *WranglerConfig, existing *WranglerImportResult) []WranglerDiffItem {
	converted := ws.ConvertToConfig(wc)
	var diffs []WranglerDiffItem

	// Compare main worker
	cw, cwOk := converted.Workers["main"]
	ew, ewOk := existing.Workers["main"]

	if cwOk && !ewOk {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "workers.main",
			Wrangler: cw.Name,
			Cosmo:    "(missing)",
			Status:   "added",
		})
	} else if cwOk && ewOk {
		if cw.Name != ew.Name {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "workers.main.name",
				Wrangler: cw.Name,
				Cosmo:    ew.Name,
				Status:   "changed",
			})
		}
		if cw.Script != ew.Script {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "workers.main.script",
				Wrangler: cw.Script,
				Cosmo:    ew.Script,
				Status:   wranglerStatusOf(cw.Script, ew.Script),
			})
		}
		if cw.CompatibilityDate != ew.CompatibilityDate {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "workers.main.compatibility_date",
				Wrangler: cw.CompatibilityDate,
				Cosmo:    ew.CompatibilityDate,
				Status:   wranglerStatusOf(cw.CompatibilityDate, ew.CompatibilityDate),
			})
		}
		if cw.Route != ew.Route {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "workers.main.route",
				Wrangler: cw.Route,
				Cosmo:    ew.Route,
				Status:   wranglerStatusOf(cw.Route, ew.Route),
			})
		}
	}

	// Compare KV
	if converted.KV != nil && existing.KV == nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "kv",
			Wrangler: fmt.Sprintf("%d namespaces", len(converted.KV.Namespaces)),
			Cosmo:    "(missing)",
			Status:   "added",
		})
	} else if converted.KV == nil && existing.KV != nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "kv",
			Wrangler: "(none)",
			Cosmo:    fmt.Sprintf("%d namespaces", len(existing.KV.Namespaces)),
			Status:   "removed",
		})
	} else if converted.KV != nil && existing.KV != nil {
		if len(converted.KV.Namespaces) != len(existing.KV.Namespaces) {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "kv.namespaces",
				Wrangler: fmt.Sprintf("%d", len(converted.KV.Namespaces)),
				Cosmo:    fmt.Sprintf("%d", len(existing.KV.Namespaces)),
				Status:   "changed",
			})
		}
	}

	// Compare R2
	if converted.R2 != nil && existing.R2 == nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "r2",
			Wrangler: fmt.Sprintf("%d buckets", len(converted.R2.Buckets)),
			Cosmo:    "(missing)",
			Status:   "added",
		})
	} else if converted.R2 == nil && existing.R2 != nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "r2",
			Wrangler: "(none)",
			Cosmo:    fmt.Sprintf("%d buckets", len(existing.R2.Buckets)),
			Status:   "removed",
		})
	} else if converted.R2 != nil && existing.R2 != nil {
		if len(converted.R2.Buckets) != len(existing.R2.Buckets) {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "r2.buckets",
				Wrangler: fmt.Sprintf("%d", len(converted.R2.Buckets)),
				Cosmo:    fmt.Sprintf("%d", len(existing.R2.Buckets)),
				Status:   "changed",
			})
		}
	}

	// Compare D1
	if converted.D1 != nil && existing.D1 == nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "d1",
			Wrangler: fmt.Sprintf("%d databases", len(converted.D1.Databases)),
			Cosmo:    "(missing)",
			Status:   "added",
		})
	} else if converted.D1 == nil && existing.D1 != nil {
		diffs = append(diffs, WranglerDiffItem{
			Field:    "d1",
			Wrangler: "(none)",
			Cosmo:    fmt.Sprintf("%d databases", len(existing.D1.Databases)),
			Status:   "removed",
		})
	} else if converted.D1 != nil && existing.D1 != nil {
		if len(converted.D1.Databases) != len(existing.D1.Databases) {
			diffs = append(diffs, WranglerDiffItem{
				Field:    "d1.databases",
				Wrangler: fmt.Sprintf("%d", len(converted.D1.Databases)),
				Cosmo:    fmt.Sprintf("%d", len(existing.D1.Databases)),
				Status:   "changed",
			})
		}
	}

	// Compare profiles
	if len(converted.Profiles) > 0 || len(existing.Profiles) > 0 {
		allProfiles := make(map[string]bool)
		for k := range converted.Profiles {
			allProfiles[k] = true
		}
		for k := range existing.Profiles {
			allProfiles[k] = true
		}
		profileNames := make([]string, 0, len(allProfiles))
		for k := range allProfiles {
			profileNames = append(profileNames, k)
		}
		sort.Strings(profileNames)

		for _, name := range profileNames {
			_, inWrangler := converted.Profiles[name]
			_, inCosmo := existing.Profiles[name]
			if inWrangler && !inCosmo {
				diffs = append(diffs, WranglerDiffItem{
					Field:    fmt.Sprintf("profiles.%s", name),
					Wrangler: "present",
					Cosmo:    "(missing)",
					Status:   "added",
				})
			} else if !inWrangler && inCosmo {
				diffs = append(diffs, WranglerDiffItem{
					Field:    fmt.Sprintf("profiles.%s", name),
					Wrangler: "(none)",
					Cosmo:    "present",
					Status:   "removed",
				})
			}
		}
	}

	return diffs
}

// LoadWranglerImportYAML reads an existing .cosmoflare.yaml file into the
// wrangler import result format.
func LoadWranglerImportYAML(path string) (*WranglerImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .cosmoflare.yaml: %w", err)
	}

	var cfg WranglerImportResult
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .cosmoflare.yaml: %w", err)
	}

	return &cfg, nil
}

// FindWranglerToml locates a wrangler.toml file. If path is empty, searches
// the current directory. Returns the resolved absolute path.
func FindWranglerToml(path string) (string, error) {
	if path == "" {
		path = "wrangler.toml"
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("wrangler.toml not found at %s", abs)
	}

	// If it's a directory, look for wrangler.toml inside
	if info.IsDir() {
		abs = filepath.Join(abs, "wrangler.toml")
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("wrangler.toml not found in %s", path)
		}
	}

	return abs, nil
}

// wranglerStatusOf returns a diff status string for two values.
func wranglerStatusOf(wrangler, cosmo string) string {
	if wrangler == "" && cosmo != "" {
		return "removed"
	}
	if wrangler != "" && cosmo == "" {
		return "added"
	}
	if wrangler != cosmo {
		return "changed"
	}
	return "match"
}

// FormatWranglerDiffTable formats diff results as a human-readable table.
func FormatWranglerDiffTable(diffs []WranglerDiffItem) string {
	if len(diffs) == 0 {
		return "No differences found. wrangler.toml and .cosmoflare.yaml are in sync."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d difference(s):\n\n", len(diffs)))
	sb.WriteString(fmt.Sprintf("  %-40s %-10s %-20s %-20s\n", "FIELD", "STATUS", "WRANGLER", "COSMOFLARE"))
	sb.WriteString(fmt.Sprintf("  %-40s %-10s %-20s %-20s\n", strings.Repeat("-", 40), strings.Repeat("-", 10), strings.Repeat("-", 20), strings.Repeat("-", 20)))

	for _, d := range diffs {
		sb.WriteString(fmt.Sprintf("  %-40s %-10s %-20s %-20s\n", d.Field, d.Status, d.Wrangler, d.Cosmo))
	}

	return sb.String()
}
