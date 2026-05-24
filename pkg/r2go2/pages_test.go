package r2go2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func pagesMockSetup(handler http.HandlerFunc) (*PagesService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewPagesService(cf, "account-test-123")
	return svc, server
}

func pagesWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewPagesServiceValidation(t *testing.T) {
	_, err := NewPagesService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewPagesService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewPagesServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewPagesService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewPagesServiceFromCredsValidation(t *testing.T) {
	_, err := NewPagesServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewPagesServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewPagesServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewPagesServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

// --- Create validation ---

func TestPagesCreateValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Create(context.Background(), "", "main")
	if err == nil {
		t.Error("expected error when name is empty")
	}

	_, err = svc.Create(context.Background(), "my-site", "")
	if err == nil {
		t.Error("expected error when production branch is empty")
	}
}

func TestPagesCreateSuccess(t *testing.T) {
	now := time.Now().UTC()
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                "proj-uuid-123",
				"name":              "my-site",
				"subdomain":         "my-site.pages.dev",
				"domains":           []string{"my-site.com"},
				"production_branch": "main",
				"created_on":        now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	project, err := svc.Create(context.Background(), "my-site", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != "proj-uuid-123" {
		t.Errorf("expected ID=proj-uuid-123, got %s", project.ID)
	}
	if project.Name != "my-site" {
		t.Errorf("expected Name=my-site, got %s", project.Name)
	}
	if project.ProductionBranch != "main" {
		t.Errorf("expected ProductionBranch=main, got %s", project.ProductionBranch)
	}
}

func TestPagesCreateError(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		pagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "my-site", "main")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

// --- List ---

func TestPagesListSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "p1", "name": "site-one", "subdomain": "site-one.pages.dev", "production_branch": "main"},
				{"id": "p2", "name": "site-two", "subdomain": "site-two.pages.dev", "production_branch": "master"},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 25, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	projects, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].ID != "p1" || projects[0].Name != "site-one" {
		t.Errorf("first project mismatch: %+v", projects[0])
	}
	if projects[1].ProductionBranch != "master" {
		t.Errorf("expected ProductionBranch=master, got %s", projects[1].ProductionBranch)
	}
}

func TestPagesListError(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		pagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

// --- Get ---

func TestPagesGetValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when project name is empty")
	}
}

func TestPagesGetSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":                "proj-get-123",
				"name":              "test-site",
				"subdomain":         "test-site.pages.dev",
				"domains":           []string{"test-site.com", "www.test-site.com"},
				"production_branch": "main",
			},
		})
	})
	defer server.Close()

	project, err := svc.Get(context.Background(), "test-site")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != "proj-get-123" {
		t.Errorf("expected ID=proj-get-123, got %s", project.ID)
	}
	if len(project.Domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(project.Domains))
	}
}

// --- Delete ---

func TestPagesDeleteValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error when project name is empty")
	}
}

func TestPagesDeleteSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}, "result": nil})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "my-site")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPagesDeleteError(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		pagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

// --- ListDeployments ---

func TestPagesListDeploymentsValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.ListDeployments(context.Background(), "")
	if err == nil {
		t.Error("expected error when project name is empty")
	}
}

func TestPagesListDeploymentsSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "d1", "project_name": "my-site", "environment": "production", "url": "https://abc.pages.dev"},
				{"id": "d2", "project_name": "my-site", "environment": "preview", "url": "https://def.pages.dev"},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 25, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	deployments, err := svc.ListDeployments(context.Background(), "my-site")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deployments) != 2 {
		t.Fatalf("expected 2 deployments, got %d", len(deployments))
	}
	if deployments[0].Environment != "production" {
		t.Errorf("expected Environment=production, got %s", deployments[0].Environment)
	}
	if deployments[1].URL != "https://def.pages.dev" {
		t.Errorf("expected URL=https://def.pages.dev, got %s", deployments[1].URL)
	}
}

func TestPagesListDeploymentsError(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		pagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.ListDeployments(context.Background(), "my-site")
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

// --- GetDeployment ---

func TestPagesGetDeploymentValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.GetDeployment(context.Background(), "", "deploy-123")
	if err == nil {
		t.Error("expected error when project name is empty")
	}

	_, err = svc.GetDeployment(context.Background(), "my-site", "")
	if err == nil {
		t.Error("expected error when deployment ID is empty")
	}
}

func TestPagesGetDeploymentSuccess(t *testing.T) {
	now := time.Now().UTC()
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":           "deploy-get-123",
				"short_id":     "abc123",
				"project_id":   "proj-123",
				"project_name": "my-site",
				"environment":  "production",
				"url":          "https://abc123.pages.dev",
				"created_on":   now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	deployment, err := svc.GetDeployment(context.Background(), "my-site", "deploy-get-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deployment.ID != "deploy-get-123" {
		t.Errorf("expected ID=deploy-get-123, got %s", deployment.ID)
	}
	if deployment.Environment != "production" {
		t.Errorf("expected Environment=production, got %s", deployment.Environment)
	}
}

// --- DeleteDeployment ---

func TestPagesDeleteDeploymentValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteDeployment(context.Background(), "", "deploy-123")
	if err == nil {
		t.Error("expected error when project name is empty")
	}

	err = svc.DeleteDeployment(context.Background(), "my-site", "")
	if err == nil {
		t.Error("expected error when deployment ID is empty")
	}
}

func TestPagesDeleteDeploymentSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		pagesWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	err := svc.DeleteDeployment(context.Background(), "my-site", "deploy-del-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPagesDeleteDeploymentError(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		pagesWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.DeleteDeployment(context.Background(), "my-site", "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

// --- Mapper tests ---

func TestMapPagesProject(t *testing.T) {
	now := time.Now().UTC()
	p := mapPagesProject(cloudflare.PagesProject{
		ID:               "test-id",
		Name:             "test-project",
		SubDomain:        "test-project.pages.dev",
		Domains:          []string{"example.com"},
		ProductionBranch: "main",
		CreatedOn:        &now,
	})

	if p.ID != "test-id" {
		t.Errorf("expected ID=test-id, got %s", p.ID)
	}
	if p.Name != "test-project" {
		t.Errorf("expected Name=test-project, got %s", p.Name)
	}
	if p.SubDomain != "test-project.pages.dev" {
		t.Errorf("expected SubDomain=test-project.pages.dev, got %s", p.SubDomain)
	}
	if len(p.Domains) != 1 || p.Domains[0] != "example.com" {
		t.Errorf("expected Domains=[example.com], got %v", p.Domains)
	}
	if p.CreatedOn == nil || !p.CreatedOn.Equal(now) {
		t.Error("CreatedOn mismatch")
	}
}

func TestMapPagesProjectNilDomains(t *testing.T) {
	p := mapPagesProject(cloudflare.PagesProject{
		ID:   "test-id",
		Name: "test-project",
	})

	if p.Domains == nil {
		t.Error("expected Domains to be empty slice, got nil")
	}
	if len(p.Domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(p.Domains))
	}
}

func TestMapPagesDeployment(t *testing.T) {
	now := time.Now().UTC()
	d := mapPagesDeployment(cloudflare.PagesProjectDeployment{
		ID:          "deploy-id",
		ShortID:     "abc",
		ProjectID:   "proj-id",
		ProjectName: "my-site",
		Environment: "production",
		URL:         "https://abc.pages.dev",
		CreatedOn:   &now,
		ModifiedOn:  &now,
	})

	if d.ID != "deploy-id" {
		t.Errorf("expected ID=deploy-id, got %s", d.ID)
	}
	if d.ShortID != "abc" {
		t.Errorf("expected ShortID=abc, got %s", d.ShortID)
	}
	if d.ProjectName != "my-site" {
		t.Errorf("expected ProjectName=my-site, got %s", d.ProjectName)
	}
	if d.Environment != "production" {
		t.Errorf("expected Environment=production, got %s", d.Environment)
	}
	if d.CreatedOn == nil || !d.CreatedOn.Equal(now) {
		t.Error("CreatedOn mismatch")
	}
}
