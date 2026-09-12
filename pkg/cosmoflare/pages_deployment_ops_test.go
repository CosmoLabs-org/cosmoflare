package cosmoflare

import (
	"context"
	"net/http"
	"testing"
)

// --- Validation ---

func TestPagesRetryDeploymentValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.RetryDeployment(context.Background(), "", "dep-1"); err == nil {
		t.Error("expected error when project is empty")
	}
	if _, err := svc.RetryDeployment(context.Background(), "my-site", ""); err == nil {
		t.Error("expected error when deployment ID is empty")
	}
}

func TestPagesGetDeploymentLogsValidation(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.GetDeploymentLogs(context.Background(), "", "dep-1"); err == nil {
		t.Error("expected error when project is empty")
	}
	if _, err := svc.GetDeploymentLogs(context.Background(), "my-site", ""); err == nil {
		t.Error("expected error when deployment ID is empty")
	}
}

// --- Success paths ---

func TestPagesRetryDeploymentSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/account-test-123/pages/projects/my-site/deployments/dep-1/retry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":           "dep-2",
				"project_name": "my-site",
				"environment":  "production",
			},
		})
	})
	defer server.Close()

	dep, err := svc.RetryDeployment(context.Background(), "my-site", "dep-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep.ID != "dep-2" {
		t.Errorf("expected new deployment ID dep-2, got %s", dep.ID)
	}
}

func TestPagesGetDeploymentLogsSuccess(t *testing.T) {
	svc, server := pagesMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/account-test-123/pages/projects/my-site/deployments/dep-1/history/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		pagesWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"total": 2,
				"data": []map[string]interface{}{
					{"ts": "2026-01-01T00:00:00Z", "line": "Building..."},
					{"ts": "2026-01-01T00:00:01Z", "line": "Build complete"},
				},
			},
		})
	})
	defer server.Close()

	logs, err := svc.GetDeploymentLogs(context.Background(), "my-site", "dep-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs.Total != 2 || len(logs.Data) != 2 {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	if logs.Data[1].Line != "Build complete" {
		t.Errorf("unexpected log line: %q", logs.Data[1].Line)
	}
}
