package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// Reuses workerMockSetup / workerWriteJSON from worker_test.go (same package).

func TestSecretPutWithMock(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"name": "API_TOKEN", "type": "secret_text"},
		})
	})
	defer server.Close()

	if err := svc.SecretPut(context.Background(), "my-worker", "API_TOKEN", "super-secret-value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/scripts/my-worker/secrets") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"name":"API_TOKEN"`) {
		t.Errorf("expected secret name in request body, got %s", gotBody)
	}
}

func TestSecretDeleteWithMock(t *testing.T) {
	var gotMethod, gotPath string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  nil,
		})
	})
	defer server.Close()

	if err := svc.SecretDelete(context.Background(), "my-worker", "API_TOKEN"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/scripts/my-worker/secrets/API_TOKEN") {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

func TestSecretListWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"name": "DB_PASSWORD", "type": "secret_text"},
				{"name": "API_TOKEN", "type": "secret_text"},
			},
		})
	})
	defer server.Close()

	secrets, err := svc.SecretList(context.Background(), "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(secrets))
	}
	// Sorted deterministically by name.
	if secrets[0].Name != "API_TOKEN" || secrets[1].Name != "DB_PASSWORD" {
		t.Errorf("expected sorted names, got %s then %s", secrets[0].Name, secrets[1].Name)
	}
	if secrets[0].Type != "secret_text" {
		t.Errorf("unexpected type: %s", secrets[0].Type)
	}
}

func TestSecretsBulkWithMock(t *testing.T) {
	seen := map[string]bool{}
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		seen[req.Name] = true
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"name": req.Name, "type": "secret_text"},
		})
	})
	defer server.Close()

	results, err := svc.SecretsBulk(context.Background(), "my-worker", map[string]string{
		"API_TOKEN":   "tok",
		"DB_PASSWORD": "pw",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, res := range results {
		if !res.Success {
			t.Errorf("expected key %s to succeed, got error: %s", res.Key, res.Error)
		}
		if res.Error != "" {
			t.Errorf("expected empty error for %s, got %q", res.Key, res.Error)
		}
	}
	if !seen["API_TOKEN"] || !seen["DB_PASSWORD"] {
		t.Errorf("expected both keys uploaded, seen=%v", seen)
	}
}

func TestSecretsBulkOneFailingKey(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "BAD_KEY" {
			w.WriteHeader(http.StatusBadRequest)
			workerWriteJSON(w, map[string]interface{}{
				"success": false,
				"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid secret name"}},
			})
			return
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"name": req.Name, "type": "secret_text"},
		})
	})
	defer server.Close()

	results, err := svc.SecretsBulk(context.Background(), "my-worker", map[string]string{
		"BAD_KEY":       "nope",
		"GOOD_KEY":      "fine",
		"ALSO_GOOD_KEY": "fine",
	})
	if err != nil {
		t.Fatalf("bulk must not abort on a failing key: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	byKey := map[string]SecretBulkResult{}
	for _, res := range results {
		byKey[res.Key] = res
	}
	if byKey["BAD_KEY"].Success {
		t.Error("expected BAD_KEY to be recorded as failed")
	}
	if byKey["BAD_KEY"].Error == "" {
		t.Error("expected non-empty error for BAD_KEY")
	}
	if !byKey["GOOD_KEY"].Success || !byKey["ALSO_GOOD_KEY"].Success {
		t.Errorf("expected other keys to succeed after the failure: %+v", byKey)
	}
}

// --- validation ---

func TestSecretPutValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.SecretPut(context.Background(), "", "KEY", "val")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}

	err = svc.SecretPut(context.Background(), "my-worker", "", "val")
	if err == nil || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("expected secret-name error, got %v", err)
	}

	err = svc.SecretPut(context.Background(), "my-worker", "KEY", "")
	if err == nil || !strings.Contains(err.Error(), "secret value is required") {
		t.Fatalf("expected secret-value error, got %v", err)
	}
}

func TestSecretDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.SecretDelete(context.Background(), "", "KEY")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}

	err = svc.SecretDelete(context.Background(), "my-worker", "")
	if err == nil || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("expected secret-name error, got %v", err)
	}
}

func TestSecretListValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.SecretList(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}
}

func TestSecretsBulkValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.SecretsBulk(context.Background(), "", map[string]string{"KEY": "val"})
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}

	_, err = svc.SecretsBulk(context.Background(), "my-worker", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "at least one secret is required") {
		t.Fatalf("expected empty-map error, got %v", err)
	}
}

// --- API error mapping ---

func TestSecretPutAPIError(t *testing.T) {
	const secretValue = "super-secret-value-xyz"
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid secret name"}},
		})
	})
	defer server.Close()

	err := svc.SecretPut(context.Background(), "bad-worker", "API_TOKEN", secretValue)
	if err == nil {
		t.Fatal("expected error from API")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to set secret") {
		t.Errorf("expected wrapped message, got %v", err)
	}
	// Security invariant: the secret value must never leak into errors.
	if strings.Contains(err.Error(), secretValue) {
		t.Error("secret value leaked into error message")
	}
}

func TestSecretDeleteAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "not found"}},
		})
	})
	defer server.Close()

	if err := svc.SecretDelete(context.Background(), "my-worker", "GONE"); err == nil {
		t.Fatal("expected error from API")
	}
}
