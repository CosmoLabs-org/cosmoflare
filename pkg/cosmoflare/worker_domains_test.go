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

func TestWorkerDomainListWithMock(t *testing.T) {
	var gotMethod, gotPath string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "dom-2", "hostname": "zeta.example.com", "service": "api-worker", "zone_id": "zone-2"},
				{"id": "dom-1", "hostname": "alpha.example.com", "service": "web-worker", "zone_id": "zone-1"},
			},
		})
	})
	defer server.Close()

	domains, err := svc.DomainList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/domains") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	// Sorted deterministically by hostname.
	if domains[0].Hostname != "alpha.example.com" || domains[1].Hostname != "zeta.example.com" {
		t.Errorf("expected sorted hostnames, got %s then %s", domains[0].Hostname, domains[1].Hostname)
	}
	if domains[0].ID != "dom-1" || domains[0].Service != "web-worker" || domains[0].ZoneID != "zone-1" {
		t.Errorf("unexpected first domain: %+v", domains[0])
	}
}

func TestWorkerDomainAttachWithMock(t *testing.T) {
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
			"result": map[string]interface{}{
				"id":       "dom-new",
				"hostname": "app.example.com",
				"service":  "my-worker",
				"zone_id":  "zone-9",
			},
		})
	})
	defer server.Close()

	dom, err := svc.DomainAttach(context.Background(), "app.example.com", "my-worker", "zone-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/domains") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"hostname":"app.example.com"`) {
		t.Errorf("expected hostname in request body, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"service":"my-worker"`) {
		t.Errorf("expected service in request body, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"zone_id":"zone-9"`) {
		t.Errorf("expected zone_id in request body, got %s", gotBody)
	}
	if dom.ID != "dom-new" || dom.Hostname != "app.example.com" || dom.Service != "my-worker" || dom.ZoneID != "zone-9" {
		t.Errorf("unexpected returned domain: %+v", dom)
	}
}

func TestWorkerDomainDetachWithMock(t *testing.T) {
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

	if err := svc.DomainDetach(context.Background(), "dom-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/domains/dom-123") {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

// --- validation ---

func TestWorkerDomainAttachValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.DomainAttach(context.Background(), "", "my-worker", "zone-1")
	if err == nil || !strings.Contains(err.Error(), "hostname is required") {
		t.Fatalf("expected hostname error, got %v", err)
	}

	_, err = svc.DomainAttach(context.Background(), "app.example.com", "", "zone-1")
	if err == nil || !strings.Contains(err.Error(), "service name is required") {
		t.Fatalf("expected service-name error, got %v", err)
	}

	_, err = svc.DomainAttach(context.Background(), "app.example.com", "my-worker", "")
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

func TestWorkerDomainDetachValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.DomainDetach(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "domain ID is required") {
		t.Fatalf("expected domain ID error, got %v", err)
	}
}

// --- API error mapping ---

func TestWorkerDomainAttachAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1002, "message": "workers.api.domain_already_attached"}},
		})
	})
	defer server.Close()

	_, err := svc.DomainAttach(context.Background(), "taken.example.com", "my-worker", "zone-1")
	if err == nil {
		t.Fatal("expected error from API")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to attach domain") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}

func TestWorkerDomainListAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.DomainList(context.Background())
	if err == nil {
		t.Fatal("expected error from API")
	}
	if !strings.Contains(err.Error(), "failed to list worker domains") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}
