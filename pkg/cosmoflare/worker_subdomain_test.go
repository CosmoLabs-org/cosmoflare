package cosmoflare

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// Reuses workerMockSetup / workerWriteJSON from worker_test.go (same package).

func TestWorkerSubdomainGetWithMock(t *testing.T) {
	var gotMethod, gotPath string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"name": "my-team"},
		})
	})
	defer server.Close()

	info, err := svc.SubdomainGet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/subdomain") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if info.Subdomain != "my-team" {
		t.Errorf("expected subdomain my-team, got %s", info.Subdomain)
	}
}

func TestWorkerSubdomainSetWithMock(t *testing.T) {
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
			"result":  map[string]interface{}{"name": "renamed-team"},
		})
	})
	defer server.Close()

	info, err := svc.SubdomainSet(context.Background(), "renamed-team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/subdomain") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"name":"renamed-team"`) {
		t.Errorf("expected subdomain name in request body, got %s", gotBody)
	}
	if info.Subdomain != "renamed-team" {
		t.Errorf("expected subdomain renamed-team, got %s", info.Subdomain)
	}
}

// TestWorkerSubdomainSetValidationTable covers the format rules offline:
// empty names and anything outside [a-z0-9-] must be rejected before any
// network call is made.
func TestWorkerSubdomainSetValidationTable(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	cases := []struct {
		name      string
		subdomain string
		wantErr   string
	}{
		{"empty", "", "subdomain is required"},
		{"uppercase", "My-Team", "subdomain may only contain lowercase letters, digits, and hyphens"},
		{"underscore", "my_team", "subdomain may only contain lowercase letters, digits, and hyphens"},
		{"dot", "my.team", "subdomain may only contain lowercase letters, digits, and hyphens"},
		{"space", "my team", "subdomain may only contain lowercase letters, digits, and hyphens"},
		{"slash", "my/team", "subdomain may only contain lowercase letters, digits, and hyphens"},
		{"unicode", "tém", "subdomain may only contain lowercase letters, digits, and hyphens"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SubdomainSet(context.Background(), tc.subdomain)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}

	// Valid shapes must pass local validation (these calls would hit the
	// network, so only check the validator helper directly).
	for _, ok := range []string{"my-team", "team123", "a", "1-2-3"} {
		if err := validateSubdomainFormat(ok); err != nil {
			t.Errorf("expected %q to be valid, got %v", ok, err)
		}
	}
}

func TestWorkerSubdomainGetAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.SubdomainGet(context.Background())
	if err == nil {
		t.Fatal("expected error from API")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to get workers.dev subdomain") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}

func TestWorkerSubdomainSetAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1024, "message": "subdomain already taken"}},
		})
	})
	defer server.Close()

	_, err := svc.SubdomainSet(context.Background(), "taken-team")
	if err == nil {
		t.Fatal("expected error from API")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to set workers.dev subdomain") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}
