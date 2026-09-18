package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestCountByLevel_MixedLevels verifies countByLevel counts only results at
// the requested level and tolerates empty slices.
func TestCountByLevel_MixedLevels(t *testing.T) {
	results := []cosmoflare.WranglerValidationResult{
		{Level: "error", Field: "name", Message: "missing"},
		{Level: "warning", Field: "main", Message: "missing entry point"},
		{Level: "error", Field: "kv_namespaces[0].id", Message: "missing id"},
		{Level: "warning", Field: "compatibility_date", Message: "missing"},
		{Level: "warning", Field: "vars", Message: "empty"},
	}

	cases := []struct {
		level string
		want  int
	}{
		{"error", 2},
		{"warning", 3},
		{"info", 0},
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			if got := countByLevel(results, tc.level); got != tc.want {
				t.Errorf("countByLevel(%s) = %d, want %d", tc.level, got, tc.want)
			}
		})
	}

	t.Run("nil slice", func(t *testing.T) {
		if got := countByLevel(nil, "error"); got != 0 {
			t.Errorf("countByLevel(nil) = %d, want 0", got)
		}
	})
}

// TestWranglerReportValidation_NoErrors verifies warnings-only validation
// results do not abort an import.
func TestWranglerReportValidation_NoErrors(t *testing.T) {
	savedJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = savedJSON }()

	results := []cosmoflare.WranglerValidationResult{
		{Level: "warning", Field: "main", Message: "missing entry point"},
	}
	if err := wranglerReportValidation(results); err != nil {
		t.Errorf("warnings-only results should pass, got %v", err)
	}
}

// TestWranglerReportValidation_ErrorsAbort verifies error-level results are
// printed and surfaced as the command's error.
func TestWranglerReportValidation_ErrorsAbort(t *testing.T) {
	savedJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = savedJSON }()

	results := []cosmoflare.WranglerValidationResult{
		{Level: "error", Field: "name", Message: "name is required"},
		{Level: "warning", Field: "main", Message: "missing entry point"},
	}

	out := capturePrint(t, func() {
		err := wranglerReportValidation(results)
		if err == nil || !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("expected validation-failed error, got %v", err)
		}
	})

	if !strings.Contains(out, "wrangler.toml has validation errors") {
		t.Errorf("error heading missing: %q", out)
	}
	if !strings.Contains(out, "name: name is required") {
		t.Errorf("error detail missing: %q", out)
	}
}

// TestWranglerPrintWarnings_ModeGated verifies warnings print in text mode
// and stay silent in --json mode.
func TestWranglerPrintWarnings_ModeGated(t *testing.T) {
	savedJSON := JSONOutput
	defer func() { JSONOutput = savedJSON }()

	results := []cosmoflare.WranglerValidationResult{
		{Level: "warning", Field: "main", Message: "missing entry point"},
		{Level: "error", Field: "name", Message: "ignored here"},
	}

	t.Run("text mode prints warnings only", func(t *testing.T) {
		JSONOutput = false
		out := capturePrint(t, func() { wranglerPrintWarnings(results) })
		if !strings.Contains(out, "main: missing entry point") {
			t.Errorf("warning line missing: %q", out)
		}
		if strings.Contains(out, "name is required") {
			t.Errorf("error-level result leaked into warnings: %q", out)
		}
	})

	t.Run("json mode is silent", func(t *testing.T) {
		JSONOutput = true
		out := capturePrint(t, func() { wranglerPrintWarnings(results) })
		if strings.TrimSpace(out) != "" {
			t.Errorf("json mode printed warnings: %q", out)
		}
	})
}

// TestWranglerImportPresenters verifies the dry-run and success payload
// helpers render the expected human-readable summary for a known config.
func TestWranglerImportPresenters(t *testing.T) {
	savedJSON, savedDry := JSONOutput, DryRun
	JSONOutput, DryRun = false, false
	defer func() { JSONOutput, DryRun = savedJSON, savedDry }()

	wc := &cosmoflare.WranglerConfig{Name: "my-worker", Main: "src/index.ts"}
	cc := &cosmoflare.WranglerImportResult{Version: "1"}

	t.Run("dry run previews without writing", func(t *testing.T) {
		out := capturePrint(t, func() {
			if err := wranglerImportDryRun("/tmp/wrangler.toml", "/tmp/.cosmoflare.yaml", wc, cc); err != nil {
				t.Errorf("dry-run presenter failed: %v", err)
			}
		})
		for _, want := range []string{"Would import", "Worker: my-worker", "Entry point: src/index.ts"} {
			if !strings.Contains(out, want) {
				t.Errorf("dry-run output missing %q: %q", want, out)
			}
		}
	})

	t.Run("success reports import result", func(t *testing.T) {
		out := capturePrint(t, func() {
			if err := wranglerImportSuccess("/tmp/wrangler.toml", "/tmp/.cosmoflare.yaml", wc); err != nil {
				t.Errorf("success presenter failed: %v", err)
			}
		})
		for _, want := range []string{"Imported", "Worker: my-worker"} {
			if !strings.Contains(out, want) {
				t.Errorf("success output missing %q: %q", want, out)
			}
		}
	})
}

// TestWranglerImportPresenters_ZeroConfig verifies the presenters tolerate a
// zero-value config (no name, no entry point) without panicking.
func TestWranglerImportPresenters_ZeroConfig(t *testing.T) {
	savedJSON, savedDry := JSONOutput, DryRun
	JSONOutput, DryRun = false, false
	defer func() { JSONOutput, DryRun = savedJSON, savedDry }()

	_ = capturePrint(t, func() {
		if err := wranglerImportDryRun("/in", "/out", &cosmoflare.WranglerConfig{}, &cosmoflare.WranglerImportResult{}); err != nil {
			t.Errorf("zero-config dry-run failed: %v", err)
		}
		if err := wranglerImportSuccess("/in", "/out", &cosmoflare.WranglerConfig{}); err != nil {
			t.Errorf("zero-config success failed: %v", err)
		}
	})
}

// TestRunWranglerDiff_MissingInputs verifies both input errors: a missing
// wrangler.toml and a missing .cosmoflare.yaml.
func TestRunWranglerDiff_MissingInputs(t *testing.T) {
	savedJSON, savedDry := JSONOutput, DryRun
	JSONOutput, DryRun = false, false
	defer func() { JSONOutput, DryRun = savedJSON, savedDry }()

	t.Run("missing wrangler toml", func(t *testing.T) {
		err := runWranglerDiff(nil, []string{filepath.Join(t.TempDir(), "nope", "wrangler.toml")})
		if err == nil || !strings.Contains(err.Error(), "failed to find wrangler.toml") {
			t.Errorf("expected find error, got %v", err)
		}
	})

	t.Run("missing cosmoflare yaml", func(t *testing.T) {
		dir := t.TempDir()
		tomlPath := filepath.Join(dir, "wrangler.toml")
		if err := os.WriteFile(tomlPath, []byte("name = \"w\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := runWranglerDiff(nil, []string{tomlPath})
		if err == nil || !strings.Contains(err.Error(), "no .cosmoflare.yaml found") {
			t.Errorf("expected missing-config error, got %v", err)
		}
	})
}

// TestRunWranglerDiff_AfterImport verifies diff succeeds once an import has
// produced the .cosmoflare.yaml, emitting the JSON diff payload in --json
// mode.
func TestRunWranglerDiff_AfterImport(t *testing.T) {
	savedJSON, savedDry := JSONOutput, DryRun
	savedOut, savedForce := wranglerOutputPath, wranglerForce
	defer func() {
		JSONOutput, DryRun = savedJSON, savedDry
		wranglerOutputPath, wranglerForce = savedOut, savedForce
	}()

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "wrangler.toml")
	if err := os.WriteFile(tomlPath, []byte("name = \"w\"\nmain = \"src/index.ts\"\ncompatibility_date = \"2024-01-01\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Import first (JSON off so the human path runs too).
	JSONOutput, DryRun, wranglerOutputPath, wranglerForce = false, false, "", true
	if err := runWranglerImport(nil, []string{tomlPath}); err != nil {
		t.Fatalf("import before diff failed: %v", err)
	}

	JSONOutput = true
	out := capturePrint(t, func() {
		if err := runWranglerDiff(nil, []string{tomlPath}); err != nil {
			t.Errorf("diff after import failed: %v", err)
		}
	})
	for _, want := range []string{`"wrangler_path"`, `"differences"`, `"diff_count"`} {
		if !strings.Contains(out, want) {
			t.Errorf("json diff payload missing %s: %q", want, out)
		}
	}
}
