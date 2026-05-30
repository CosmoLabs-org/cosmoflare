package cosmoflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// NewTemplateService
// ---------------------------------------------------------------------------

func TestTemplateService_New(t *testing.T) {
	svc := NewTemplateService()
	if svc == nil {
		t.Fatal("expected non-nil TemplateService")
	}
}

// ---------------------------------------------------------------------------
// ListTemplates
// ---------------------------------------------------------------------------

func TestTemplateService_ListTemplates_ReturnsAll(t *testing.T) {
	svc := NewTemplateService()
	templates := svc.ListTemplates()

	expected := []string{"api", "static", "fullstack", "cron", "queue"}
	if len(templates) != len(expected) {
		t.Fatalf("expected %d templates, got %d", len(expected), len(templates))
	}

	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("template %q not found in list", name)
		}
	}
}

func TestTemplateService_ListTemplates_HasRequiredFields(t *testing.T) {
	svc := NewTemplateService()
	templates := svc.ListTemplates()

	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("template has empty Name")
		}
		if tmpl.Description == "" {
			t.Errorf("template %q has empty Description", tmpl.Name)
		}
		if len(tmpl.Services) == 0 {
			t.Errorf("template %q has no Services", tmpl.Name)
		}
		if len(tmpl.Files) == 0 {
			t.Errorf("template %q has no Files", tmpl.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// GetTemplate
// ---------------------------------------------------------------------------

func TestTemplateService_GetTemplate_Found(t *testing.T) {
	svc := NewTemplateService()

	tmpl, err := svc.GetTemplate("api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.Name != "api" {
		t.Errorf("expected name %q, got %q", "api", tmpl.Name)
	}
}

func TestTemplateService_GetTemplate_NotFound(t *testing.T) {
	svc := NewTemplateService()

	_, err := svc.GetTemplate("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention template name, got: %v", err)
	}
}

func TestTemplateService_GetTemplate_EmptyName(t *testing.T) {
	svc := NewTemplateService()

	_, err := svc.GetTemplate("")
	if err == nil {
		t.Fatal("expected error for empty template name")
	}
}

// ---------------------------------------------------------------------------
// CreateFromTemplate
// ---------------------------------------------------------------------------

func TestTemplateService_CreateFromTemplate_WritesFiles(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	result, err := svc.CreateFromTemplate("api", dir, "my-api", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TemplateName != "api" {
		t.Errorf("expected TemplateName %q, got %q", "api", result.TemplateName)
	}
	if result.ProjectName != "my-api" {
		t.Errorf("expected ProjectName %q, got %q", "my-api", result.ProjectName)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("expected files to be created")
	}

	// Verify .cosmoflare.yaml exists
	configPath := filepath.Join(dir, ".cosmoflare.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected .cosmoflare.yaml to be created")
	}

	// Verify wrangler.toml exists
	wranglerPath := filepath.Join(dir, "wrangler.toml")
	if _, err := os.Stat(wranglerPath); os.IsNotExist(err) {
		t.Error("expected wrangler.toml to be created")
	}
}

func TestTemplateService_CreateFromTemplate_ProjectNameInConfig(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	_, err := svc.CreateFromTemplate("api", dir, "test-project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".cosmoflare.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "test-project") {
		t.Error("config should contain the project name")
	}
}

func TestTemplateService_CreateFromTemplate_NoOverwriteByDefault(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	// Create a file that the template would write
	os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte("existing"), 0644)

	_, err := svc.CreateFromTemplate("api", dir, "my-api", false)
	if err == nil {
		t.Fatal("expected error when files exist and force=false")
	}
}

func TestTemplateService_CreateFromTemplate_ForceOverwrite(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte("existing"), 0644)

	_, err := svc.CreateFromTemplate("api", dir, "my-api", true)
	if err != nil {
		t.Fatalf("expected force to succeed, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".cosmoflare.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) == "existing" {
		t.Error("file should have been overwritten")
	}
}

func TestTemplateService_CreateFromTemplate_UnknownTemplate(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	_, err := svc.CreateFromTemplate("bogus", dir, "proj", false)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestTemplateService_CreateFromTemplate_EmptyDir(t *testing.T) {
	svc := NewTemplateService()

	_, err := svc.CreateFromTemplate("api", "", "proj", false)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestTemplateService_CreateFromTemplate_AllTemplates(t *testing.T) {
	svc := NewTemplateService()
	templates := svc.ListTemplates()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			dir := t.TempDir()
			result, err := svc.CreateFromTemplate(tmpl.Name, dir, "test-"+tmpl.Name, false)
			if err != nil {
				t.Fatalf("failed to create template %q: %v", tmpl.Name, err)
			}
			if len(result.FilesCreated) == 0 {
				t.Errorf("template %q created no files", tmpl.Name)
			}

			// Every template must produce .cosmoflare.yaml
			configPath := filepath.Join(dir, ".cosmoflare.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Errorf("template %q did not create .cosmoflare.yaml", tmpl.Name)
			}

			// Every template must produce wrangler.toml
			wranglerPath := filepath.Join(dir, "wrangler.toml")
			if _, err := os.Stat(wranglerPath); os.IsNotExist(err) {
				t.Errorf("template %q did not create wrangler.toml", tmpl.Name)
			}
		})
	}
}

func TestTemplateService_CreateFromTemplate_CreatesSubdirectories(t *testing.T) {
	svc := NewTemplateService()
	dir := t.TempDir()

	_, err := svc.CreateFromTemplate("api", dir, "my-api", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// API template should create src/ directory
	srcDir := filepath.Join(dir, "src")
	info, err := os.Stat(srcDir)
	if os.IsNotExist(err) {
		t.Error("expected src/ directory to be created")
	} else if !info.IsDir() {
		t.Error("src/ should be a directory")
	}
}
