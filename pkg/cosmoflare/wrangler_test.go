package cosmoflare

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

const testWranglerToml = `
name = "my-worker"
main = "src/index.ts"
compatibility_date = "2024-09-01"
compatibility_flags = ["nodejs_compat"]
account_id = "abc123"
route = "example.com/*"
workers_dev = true

[vars]
API_URL = "https://api.example.com"
DEBUG = "false"

[[kv_namespaces]]
binding = "MY_KV"
id = "kv-namespace-id-1"
preview_id = "kv-preview-id-1"

[[kv_namespaces]]
binding = "CACHE_KV"
id = "kv-namespace-id-2"

[[r2_buckets]]
binding = "MY_BUCKET"
bucket_name = "my-production-bucket"

[[r2_buckets]]
binding = "ASSETS"
bucket_name = "static-assets"
preview_bucket_name = "static-assets-preview"

[[d1_databases]]
binding = "MY_DB"
database_name = "my-database"
database_id = "d1-database-id-1"

[env.staging]
name = "my-worker-staging"
route = "staging.example.com/*"

[env.staging.vars]
API_URL = "https://staging-api.example.com"
DEBUG = "true"

[[env.staging.kv_namespaces]]
binding = "MY_KV"
id = "kv-staging-id"

[env.production]
route = "example.com/*"
compatibility_date = "2024-10-01"
`

func TestParseTomlBytes_Basic(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if cfg.Name != "my-worker" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-worker")
	}
	if cfg.Main != "src/index.ts" {
		t.Errorf("Main = %q, want %q", cfg.Main, "src/index.ts")
	}
	if cfg.CompatibilityDate != "2024-09-01" {
		t.Errorf("CompatibilityDate = %q, want %q", cfg.CompatibilityDate, "2024-09-01")
	}
	if len(cfg.CompatibilityFlags) != 1 || cfg.CompatibilityFlags[0] != "nodejs_compat" {
		t.Errorf("CompatibilityFlags = %v, want [nodejs_compat]", cfg.CompatibilityFlags)
	}
	if cfg.Route != "example.com/*" {
		t.Errorf("Route = %q, want %q", cfg.Route, "example.com/*")
	}
}

func TestParseTomlBytes_KVNamespaces(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if len(cfg.KVNamespaces) != 2 {
		t.Fatalf("KVNamespaces count = %d, want 2", len(cfg.KVNamespaces))
	}
	if cfg.KVNamespaces[0].Binding != "MY_KV" {
		t.Errorf("KV[0].Binding = %q, want %q", cfg.KVNamespaces[0].Binding, "MY_KV")
	}
	if cfg.KVNamespaces[0].ID != "kv-namespace-id-1" {
		t.Errorf("KV[0].ID = %q, want %q", cfg.KVNamespaces[0].ID, "kv-namespace-id-1")
	}
	if cfg.KVNamespaces[0].PreviewID != "kv-preview-id-1" {
		t.Errorf("KV[0].PreviewID = %q, want %q", cfg.KVNamespaces[0].PreviewID, "kv-preview-id-1")
	}
	if cfg.KVNamespaces[1].Binding != "CACHE_KV" {
		t.Errorf("KV[1].Binding = %q, want %q", cfg.KVNamespaces[1].Binding, "CACHE_KV")
	}
}

func TestParseTomlBytes_R2Buckets(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if len(cfg.R2Buckets) != 2 {
		t.Fatalf("R2Buckets count = %d, want 2", len(cfg.R2Buckets))
	}
	if cfg.R2Buckets[0].Binding != "MY_BUCKET" {
		t.Errorf("R2[0].Binding = %q, want %q", cfg.R2Buckets[0].Binding, "MY_BUCKET")
	}
	if cfg.R2Buckets[0].BucketName != "my-production-bucket" {
		t.Errorf("R2[0].BucketName = %q, want %q", cfg.R2Buckets[0].BucketName, "my-production-bucket")
	}
	if cfg.R2Buckets[1].PreviewBucketName != "static-assets-preview" {
		t.Errorf("R2[1].PreviewBucketName = %q, want %q", cfg.R2Buckets[1].PreviewBucketName, "static-assets-preview")
	}
}

func TestParseTomlBytes_D1Databases(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if len(cfg.D1Databases) != 1 {
		t.Fatalf("D1Databases count = %d, want 1", len(cfg.D1Databases))
	}
	if cfg.D1Databases[0].Binding != "MY_DB" {
		t.Errorf("D1[0].Binding = %q, want %q", cfg.D1Databases[0].Binding, "MY_DB")
	}
	if cfg.D1Databases[0].DatabaseID != "d1-database-id-1" {
		t.Errorf("D1[0].DatabaseID = %q, want %q", cfg.D1Databases[0].DatabaseID, "d1-database-id-1")
	}
}

func TestParseTomlBytes_Environments(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if len(cfg.Env) != 2 {
		t.Fatalf("Env count = %d, want 2", len(cfg.Env))
	}

	staging := cfg.Env["staging"]
	if staging == nil {
		t.Fatal("staging env is nil")
	}
	if staging.Name != "my-worker-staging" {
		t.Errorf("staging.Name = %q, want %q", staging.Name, "my-worker-staging")
	}
	if staging.Route != "staging.example.com/*" {
		t.Errorf("staging.Route = %q, want %q", staging.Route, "staging.example.com/*")
	}
	if len(staging.KVNamespaces) != 1 {
		t.Fatalf("staging.KVNamespaces count = %d, want 1", len(staging.KVNamespaces))
	}
	if staging.KVNamespaces[0].ID != "kv-staging-id" {
		t.Errorf("staging.KV[0].ID = %q, want %q", staging.KVNamespaces[0].ID, "kv-staging-id")
	}

	prod := cfg.Env["production"]
	if prod == nil {
		t.Fatal("production env is nil")
	}
	if prod.CompatibilityDate != "2024-10-01" {
		t.Errorf("production.CompatibilityDate = %q, want %q", prod.CompatibilityDate, "2024-10-01")
	}
}

func TestParseTomlBytes_Vars(t *testing.T) {
	ws := NewWranglerService()
	cfg, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("ParseTomlBytes failed: %v", err)
	}

	if cfg.Vars["API_URL"] != "https://api.example.com" {
		t.Errorf("Vars[API_URL] = %q, want %q", cfg.Vars["API_URL"], "https://api.example.com")
	}
	if cfg.Vars["DEBUG"] != "false" {
		t.Errorf("Vars[DEBUG] = %q, want %q", cfg.Vars["DEBUG"], "false")
	}
}

func TestParseTomlBytes_InvalidTOML(t *testing.T) {
	ws := NewWranglerService()
	_, err := ws.ParseTomlBytes([]byte(`[invalid toml {{`))
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}

func TestConvertToConfig_MainWorker(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	if cc.Version != "1" {
		t.Errorf("Version = %q, want %q", cc.Version, "1")
	}

	main, ok := cc.Workers["main"]
	if !ok {
		t.Fatal("workers.main not found")
	}
	if main.Name != "my-worker" {
		t.Errorf("main.Name = %q, want %q", main.Name, "my-worker")
	}
	if main.Script != "src/index.ts" {
		t.Errorf("main.Script = %q, want %q", main.Script, "src/index.ts")
	}
	if main.CompatibilityDate != "2024-09-01" {
		t.Errorf("main.CompatibilityDate = %q, want %q", main.CompatibilityDate, "2024-09-01")
	}
	if main.Route != "example.com/*" {
		t.Errorf("main.Route = %q, want %q", main.Route, "example.com/*")
	}
}

func TestConvertToConfig_KV(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	if cc.KV == nil {
		t.Fatal("KV section is nil")
	}
	if len(cc.KV.Namespaces) != 2 {
		t.Fatalf("KV.Namespaces count = %d, want 2", len(cc.KV.Namespaces))
	}
	if cc.KV.Namespaces[0].Binding != "MY_KV" {
		t.Errorf("KV[0].Binding = %q, want %q", cc.KV.Namespaces[0].Binding, "MY_KV")
	}
	if cc.KV.Namespaces[0].PreviewID != "kv-preview-id-1" {
		t.Errorf("KV[0].PreviewID = %q, want %q", cc.KV.Namespaces[0].PreviewID, "kv-preview-id-1")
	}
}

func TestConvertToConfig_R2(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	if cc.R2 == nil {
		t.Fatal("R2 section is nil")
	}
	if len(cc.R2.Buckets) != 2 {
		t.Fatalf("R2.Buckets count = %d, want 2", len(cc.R2.Buckets))
	}
	if cc.R2.Buckets[0].BucketName != "my-production-bucket" {
		t.Errorf("R2[0].BucketName = %q, want %q", cc.R2.Buckets[0].BucketName, "my-production-bucket")
	}
}

func TestConvertToConfig_D1(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	if cc.D1 == nil {
		t.Fatal("D1 section is nil")
	}
	if len(cc.D1.Databases) != 1 {
		t.Fatalf("D1.Databases count = %d, want 1", len(cc.D1.Databases))
	}
	if cc.D1.Databases[0].DatabaseID != "d1-database-id-1" {
		t.Errorf("D1[0].DatabaseID = %q, want %q", cc.D1.Databases[0].DatabaseID, "d1-database-id-1")
	}
}

func TestConvertToConfig_Profiles(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	if len(cc.Profiles) != 2 {
		t.Fatalf("Profiles count = %d, want 2", len(cc.Profiles))
	}

	staging := cc.Profiles["staging"]
	if staging == nil || staging.Worker == nil {
		t.Fatal("staging profile or worker is nil")
	}
	if staging.Worker.Name != "my-worker-staging" {
		t.Errorf("staging.Worker.Name = %q, want %q", staging.Worker.Name, "my-worker-staging")
	}
	if staging.Worker.Route != "staging.example.com/*" {
		t.Errorf("staging.Worker.Route = %q, want %q", staging.Worker.Route, "staging.example.com/*")
	}
}

func TestConvertToConfig_Bindings(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	main := cc.Workers["main"]

	if main.Bindings == nil {
		t.Fatal("main worker bindings is nil")
	}
	if len(main.Bindings.KV) != 2 {
		t.Errorf("main.Bindings.KV count = %d, want 2", len(main.Bindings.KV))
	}
	if len(main.Bindings.R2) != 2 {
		t.Errorf("main.Bindings.R2 count = %d, want 2", len(main.Bindings.R2))
	}
	if len(main.Bindings.D1) != 1 {
		t.Errorf("main.Bindings.D1 count = %d, want 1", len(main.Bindings.D1))
	}
}

func TestConvertToConfig_Vars(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	main := cc.Workers["main"]

	if main.Vars["API_URL"] != "https://api.example.com" {
		t.Errorf("Vars[API_URL] = %q, want %q", main.Vars["API_URL"], "https://api.example.com")
	}
}

func TestConvertToConfig_MinimalWorker(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(`name = "simple-worker"`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	main := cc.Workers["main"]

	if main.Name != "simple-worker" {
		t.Errorf("Name = %q, want %q", main.Name, "simple-worker")
	}
	if cc.KV != nil {
		t.Error("expected KV to be nil for minimal config")
	}
	if cc.R2 != nil {
		t.Error("expected R2 to be nil for minimal config")
	}
	if cc.D1 != nil {
		t.Error("expected D1 to be nil for minimal config")
	}
	if main.Bindings != nil {
		t.Error("expected Bindings to be nil for minimal config")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results := ws.Validate(wc)
	if WranglerHasErrors(results) {
		t.Errorf("expected no errors for valid config, got: %v", results)
	}
}

func TestValidate_MissingName(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(`main = "index.js"`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results := ws.Validate(wc)
	if !WranglerHasErrors(results) {
		t.Fatal("expected errors for missing name")
	}

	foundNameError := false
	for _, r := range results {
		if r.Field == "name" && r.Level == "error" {
			foundNameError = true
		}
	}
	if !foundNameError {
		t.Error("expected error for missing name field")
	}
}

func TestValidate_MissingKVID(t *testing.T) {
	ws := NewWranglerService()
	tomlData := `
name = "test"
main = "index.js"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "MY_KV"
id = ""
`
	wc, err := ws.ParseTomlBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results := ws.Validate(wc)
	if !WranglerHasErrors(results) {
		t.Fatal("expected errors for missing KV ID")
	}
}

func TestValidate_DuplicateBindings(t *testing.T) {
	ws := NewWranglerService()
	tomlData := `
name = "test"
main = "index.js"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "SHARED"
id = "kv-id-1"

[[r2_buckets]]
binding = "SHARED"
bucket_name = "my-bucket"
`
	wc, err := ws.ParseTomlBytes([]byte(tomlData))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results := ws.Validate(wc)
	if !WranglerHasErrors(results) {
		t.Fatal("expected errors for duplicate bindings")
	}

	foundDup := false
	for _, r := range results {
		if r.Field == "bindings" {
			foundDup = true
		}
	}
	if !foundDup {
		t.Error("expected duplicate binding error")
	}
}

func TestValidate_MissingMain(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(`name = "test"`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results := ws.Validate(wc)
	foundWarning := false
	for _, r := range results {
		if r.Field == "main" && r.Level == "warning" {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning for missing main")
	}
}

func TestMarshalWranglerImportYAML(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	data, err := MarshalWranglerImportYAML(cc)
	if err != nil {
		t.Fatalf("MarshalWranglerImportYAML failed: %v", err)
	}

	yamlStr := string(data)
	if len(yamlStr) == 0 {
		t.Fatal("marshaled YAML is empty")
	}

	// Verify round-trip: unmarshal back
	var roundtrip WranglerImportResult
	if err := yaml.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("failed to unmarshal round-trip YAML: %v", err)
	}
	if roundtrip.Workers["main"].Name != "my-worker" {
		t.Errorf("round-trip Name = %q, want %q", roundtrip.Workers["main"].Name, "my-worker")
	}
}

func TestParseToml_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wrangler.toml")
	if err := os.WriteFile(path, []byte(testWranglerToml), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ws := NewWranglerService()
	cfg, err := ws.ParseToml(path)
	if err != nil {
		t.Fatalf("ParseToml failed: %v", err)
	}
	if cfg.Name != "my-worker" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-worker")
	}
}

func TestParseToml_FileNotFound(t *testing.T) {
	ws := NewWranglerService()
	_, err := ws.ParseToml("/nonexistent/wrangler.toml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteWranglerImportYAML(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)

	dir := t.TempDir()
	outPath := filepath.Join(dir, ".cosmoflare.yaml")
	if err := WriteWranglerImportYAML(cc, outPath); err != nil {
		t.Fatalf("WriteWranglerImportYAML failed: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Fatal("output file is empty")
	}
	// Check header comment
	if content[:1] != "#" {
		t.Error("expected header comment in output")
	}
}

func TestDiffWranglerConfigs_NoDiffs(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	diffs := ws.DiffWranglerConfigs(wc, cc)

	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical configs, got %d: %v", len(diffs), diffs)
	}
}

func TestDiffWranglerConfigs_NameChanged(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := ws.ConvertToConfig(wc)
	main := cc.Workers["main"]
	main.Name = "different-name"
	cc.Workers["main"] = main

	diffs := ws.DiffWranglerConfigs(wc, cc)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for changed name")
	}

	found := false
	for _, d := range diffs {
		if d.Field == "workers.main.name" && d.Status == "changed" {
			found = true
		}
	}
	if !found {
		t.Error("expected workers.main.name changed diff")
	}
}

func TestDiffWranglerConfigs_MissingWorker(t *testing.T) {
	ws := NewWranglerService()
	wc, err := ws.ParseTomlBytes([]byte(testWranglerToml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cc := &WranglerImportResult{
		Version: "1",
		Workers: make(map[string]WranglerWorkerYAML),
	}

	diffs := ws.DiffWranglerConfigs(wc, cc)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for missing worker")
	}

	found := false
	for _, d := range diffs {
		if d.Field == "workers.main" && d.Status == "added" {
			found = true
		}
	}
	if !found {
		t.Error("expected workers.main added diff")
	}
}

func TestFindWranglerToml_InDir(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	if err := os.WriteFile(tomlPath, []byte(`name = "test"`), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	found, err := FindWranglerToml(dir)
	if err != nil {
		t.Fatalf("FindWranglerToml failed: %v", err)
	}
	if filepath.Base(found) != "wrangler.toml" {
		t.Errorf("found = %q, expected wrangler.toml", found)
	}
}

func TestFindWranglerToml_NotFound(t *testing.T) {
	_, err := FindWranglerToml("/nonexistent/path/wrangler.toml")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestFormatWranglerDiffTable_Empty(t *testing.T) {
	result := FormatWranglerDiffTable(nil)
	if result == "" {
		t.Fatal("expected non-empty result for no diffs")
	}
}

func TestFormatWranglerDiffTable_WithDiffs(t *testing.T) {
	diffs := []WranglerDiffItem{
		{Field: "workers.main.name", Wrangler: "a", Cosmo: "b", Status: "changed"},
	}
	result := FormatWranglerDiffTable(diffs)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestWranglerHasErrors(t *testing.T) {
	cases := []struct {
		name   string
		input  []WranglerValidationResult
		expect bool
	}{
		{"nil", nil, false},
		{"empty", []WranglerValidationResult{}, false},
		{"warning only", []WranglerValidationResult{{Level: "warning"}}, false},
		{"has error", []WranglerValidationResult{{Level: "error"}}, true},
		{"mixed", []WranglerValidationResult{{Level: "warning"}, {Level: "error"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WranglerHasErrors(tc.input); got != tc.expect {
				t.Errorf("WranglerHasErrors = %v, want %v", got, tc.expect)
			}
		})
	}
}
