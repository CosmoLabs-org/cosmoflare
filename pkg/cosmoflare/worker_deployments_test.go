package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// deploymentsEnvelope wraps a result in the Cloudflare API envelope shape.
func deploymentsEnvelope(result any) string {
	b, _ := json.Marshal(map[string]any{"success": true, "result": result})
	return string(b)
}

func TestWorkerDeploymentList(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/scripts/api/deployments") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(deploymentsEnvelope([]map[string]any{
			{"id": "dep-2", "version_id": "ver-2", "created_at": "2026-09-16T00:00:00Z"},
			{"id": "dep-1", "version_id": "ver-1", "created_at": "2026-09-15T00:00:00Z"},
		})))
	})

	out, err := svc.DeploymentList(context.Background(), "api")
	if err != nil {
		t.Fatalf("DeploymentList: %v", err)
	}
	if len(out) != 2 || out[0].ID != "dep-2" || out[1].VersionID != "ver-1" {
		t.Fatalf("unexpected deployments: %+v", out)
	}
}

func TestWorkerDeploymentListValidation(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, nil)
	if _, err := svc.DeploymentList(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name validation error, got %v", err)
	}
}

func TestWorkerDeploymentGet(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/deployments/dep-9") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(deploymentsEnvelope(map[string]any{"id": "dep-9", "version_id": "ver-9"})))
	})

	out, err := svc.DeploymentGet(context.Background(), "api", "dep-9")
	if err != nil {
		t.Fatalf("DeploymentGet: %v", err)
	}
	if out.ID != "dep-9" || out.VersionID != "ver-9" {
		t.Fatalf("unexpected deployment: %+v", out)
	}
}

func TestWorkerDeploymentGetValidation(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, nil)
	if _, err := svc.DeploymentGet(context.Background(), "api", ""); err == nil || !strings.Contains(err.Error(), "deployment ID is required") {
		t.Fatalf("expected deployment-ID validation error, got %v", err)
	}
}

func TestWorkerRollbackToPredecessor(t *testing.T) {
	var putBody map[string]any
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/deployments"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(deploymentsEnvelope([]map[string]any{
				{"id": "dep-2", "version_id": "ver-2", "created_at": time.Now().Format(time.RFC3339)},
				{"id": "dep-1", "version_id": "ver-1", "created_at": "2026-09-15T00:00:00Z"},
			})))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/deployments"):
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(deploymentsEnvelope(map[string]any{"id": "dep-3", "version_id": putBody["version_id"]})))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	out, err := svc.Rollback(context.Background(), "api", "")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if putBody["version_id"] != "ver-1" {
		t.Fatalf("rollback deployed %v, want ver-1", putBody["version_id"])
	}
	if out.ID != "dep-3" {
		t.Fatalf("unexpected new deployment: %+v", out)
	}
}

func TestWorkerRollbackNoEarlierDeployment(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(deploymentsEnvelope([]map[string]any{
			{"id": "dep-1", "version_id": "ver-1"},
		})))
	})

	if _, err := svc.Rollback(context.Background(), "api", ""); err == nil || !strings.Contains(err.Error(), "no earlier deployment") {
		t.Fatalf("expected no-earlier-deployment error, got %v", err)
	}
}
