package cosmoflare

import (
	"context"
	"net/http"
	"testing"
)

// --- Validation ---

func TestPagesListEnvVarsValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.ListEnvVars(context.Background(), "", "production"); err == nil {
		t.Error("expected error when project is empty")
	}
	if _, err := svc.ListEnvVars(context.Background(), "my-site", "staging"); err == nil {
		t.Error("expected error for unknown env value")
	}
}

func TestPagesSetEnvVarsValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.SetEnvVars(context.Background(), "", "production", []PagesEnvVar{{Key: "K", Value: "V", Type: "plain"}}); err == nil {
		t.Error("expected error when project is empty")
	}
	if err := svc.SetEnvVars(context.Background(), "my-site", "staging", []PagesEnvVar{{Key: "K", Value: "V", Type: "plain"}}); err == nil {
		t.Error("expected error for unknown env value")
	}
	if err := svc.SetEnvVars(context.Background(), "my-site", "production", nil); err == nil {
		t.Error("expected error when no vars provided")
	}
	if err := svc.SetEnvVars(context.Background(), "my-site", "production", []PagesEnvVar{{Key: "", Value: "V", Type: "plain"}}); err == nil {
		t.Error("expected error when key is empty")
	}
	if err := svc.SetEnvVars(context.Background(), "my-site", "production", []PagesEnvVar{{Key: "K", Value: "V", Type: "bogus"}}); err == nil {
		t.Error("expected error for unknown var type")
	}
}

func TestPagesDeleteEnvVarValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.DeleteEnvVar(context.Background(), "", "production", "K"); err == nil {
		t.Error("expected error when project is empty")
	}
	if err := svc.DeleteEnvVar(context.Background(), "my-site", "staging", "K"); err == nil {
		t.Error("expected error for unknown env value")
	}
	if err := svc.DeleteEnvVar(context.Background(), "my-site", "production", ""); err == nil {
		t.Error("expected error when key is empty")
	}
}

// --- ListEnvVars success, secrets masked ---

func TestPagesListEnvVarsSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":   "proj-uuid-123",
				"name": "my-site",
				"deployment_configs": map[string]interface{}{
					"production": map[string]interface{}{
						"env_vars": map[string]interface{}{
							"API_URL": map[string]interface{}{"value": "https://api.example.com", "type": "plain_text"},
							"SECRET_KEY": map[string]interface{}{
								"value": "",
								"type":  "secret_text",
							},
						},
					},
					"preview": map[string]interface{}{},
				},
			},
		})
	})
	defer server.Close()

	vars, err := svc.ListEnvVars(context.Background(), "my-site", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(vars))
	}

	byKey := map[string]PagesEnvVar{}
	for _, v := range vars {
		byKey[v.Key] = v
	}
	if byKey["API_URL"].Value != "https://api.example.com" || byKey["API_URL"].Type != "plain" {
		t.Errorf("unexpected API_URL var: %+v", byKey["API_URL"])
	}
	if byKey["SECRET_KEY"].Type != "secret" {
		t.Errorf("expected SECRET_KEY type=secret, got %+v", byKey["SECRET_KEY"])
	}
	if byKey["SECRET_KEY"].Value != "" {
		t.Errorf("secret value must never be echoed, got %q", byKey["SECRET_KEY"].Value)
	}
}

// --- SetEnvVars success ---

func TestPagesSetEnvVarsSuccess(t *testing.T) {
	getCalled := false
	patchCalled := false
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCalled = true
			pagesWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": map[string]interface{}{
					"id":                  "my-site",
					"name":                "my-site",
					"deployment_configs":  map[string]interface{}{"production": map[string]interface{}{}, "preview": map[string]interface{}{}},
					"production_branch":   "main",
					"build_config":        map[string]interface{}{},
					"latest_deployment":   map[string]interface{}{},
					"canonical_deployment": map[string]interface{}{},
				},
			})
		case http.MethodPatch:
			patchCalled = true
			if r.URL.Path != "/accounts/account-test-123/pages/projects/my-site" {
				t.Errorf("unexpected PATCH path: %s", r.URL.Path)
			}
			pagesWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result":  map[string]interface{}{"id": "my-site", "name": "my-site"},
			})
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	})
	defer server.Close()

	err := svc.SetEnvVars(context.Background(), "my-site", "production", []PagesEnvVar{
		{Key: "API_URL", Value: "https://api.example.com", Type: "plain"},
		{Key: "SECRET_KEY", Value: "shh", Type: "secret"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !getCalled {
		t.Error("expected GetPagesProject to be called before PATCH")
	}
	if !patchCalled {
		t.Error("expected UpdatePagesProject (PATCH) to be called")
	}
}

// --- DeleteEnvVar success ---

func TestPagesDeleteEnvVarSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pagesWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": map[string]interface{}{
					"id":   "my-site",
					"name": "my-site",
					"deployment_configs": map[string]interface{}{
						"production": map[string]interface{}{
							"env_vars": map[string]interface{}{
								"API_URL": map[string]interface{}{"value": "https://api.example.com", "type": "plain_text"},
							},
						},
						"preview": map[string]interface{}{},
					},
				},
			})
		case http.MethodPatch:
			pagesWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result":  map[string]interface{}{"id": "my-site", "name": "my-site"},
			})
		}
	})
	defer server.Close()

	err := svc.DeleteEnvVar(context.Background(), "my-site", "production", "API_URL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
