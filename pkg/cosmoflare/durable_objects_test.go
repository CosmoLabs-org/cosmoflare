package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func doMockSetup(handler http.HandlerFunc) (*DurableObjectsService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewDurableObjectsService(cf, "account-test-123")
	return svc, server
}

func doWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewDurableObjectsServiceValidation(t *testing.T) {
	_, err := NewDurableObjectsService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewDurableObjectsService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewDurableObjectsServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewDurableObjectsService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewDurableObjectsServiceFromCredsValidation(t *testing.T) {
	_, err := NewDurableObjectsServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewDurableObjectsServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewDurableObjectsServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewDurableObjectsServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

// --- Validation ---

func TestDurableObjectsListObjectsValidation(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	_, err := svc.ListObjects(context.Background(), "", ListObjectsOptions{})
	if err == nil {
		t.Error("expected error when namespace ID is empty")
	}
}

func TestDurableObjectsGetObjectValidation(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	_, err := svc.GetObject(context.Background(), "", "obj-1")
	if err == nil {
		t.Error("expected error when namespace ID is empty")
	}

	_, err = svc.GetObject(context.Background(), "ns-1", "")
	if err == nil {
		t.Error("expected error when object ID is empty")
	}
}

// --- ListNamespaces ---

func TestDurableObjectsListNamespacesSuccess(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/accounts/account-test-123/workers/durable_objects/namespaces" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		doWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ns-1", "name": "MyClass", "script": "my-worker"},
			},
		})
	})
	defer server.Close()

	namespaces, err := svc.ListNamespaces(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(namespaces) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(namespaces))
	}
	if namespaces[0].ID != "ns-1" || namespaces[0].Name != "MyClass" || namespaces[0].Script != "my-worker" {
		t.Errorf("unexpected namespace: %+v", namespaces[0])
	}
}

func TestDurableObjectsListNamespacesError(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		doWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "boom"}},
		})
	})
	defer server.Close()

	_, err := svc.ListNamespaces(context.Background())
	if err == nil {
		t.Error("expected error on API failure")
	}
}

// --- ListObjects ---

func TestDurableObjectsListObjectsSuccess(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-test-123/workers/durable_objects/namespaces/ns-1/objects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("expected limit=10, got %q", got)
		}
		if got := r.URL.Query().Get("cursor"); got != "abc" {
			t.Errorf("expected cursor=abc, got %q", got)
		}
		doWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "obj-1", "hasStoredData": true},
				{"id": "obj-2", "hasStoredData": false},
			},
			"result_info": map[string]interface{}{"cursor": "next-cursor"},
		})
	})
	defer server.Close()

	result, err := svc.ListObjects(context.Background(), "ns-1", ListObjectsOptions{Limit: 10, Cursor: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(result.Objects))
	}
	if result.NextCursor != "next-cursor" {
		t.Errorf("expected NextCursor=next-cursor, got %q", result.NextCursor)
	}
}

func TestDurableObjectsListObjectsNotFound(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		doWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "not found"}},
		})
	})
	defer server.Close()

	_, err := svc.ListObjects(context.Background(), "missing-ns", ListObjectsOptions{})
	if err == nil {
		t.Fatal("expected error for missing namespace")
	}
	if _, ok := err.(*R2NotFoundError); !ok {
		t.Errorf("expected R2NotFoundError, got %T: %v", err, err)
	}
}

// --- GetObject ---

func TestDurableObjectsGetObjectFound(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			doWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": []map[string]interface{}{
					{"id": "obj-1", "hasStoredData": true},
				},
				"result_info": map[string]interface{}{"cursor": "page-2"},
			})
			return
		}
		doWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "obj-2", "hasStoredData": false},
			},
		})
	})
	defer server.Close()

	detail, err := svc.GetObject(context.Background(), "ns-1", "obj-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.ID != "obj-2" || detail.NamespaceID != "ns-1" {
		t.Errorf("unexpected detail: %+v", detail)
	}
}

func TestDurableObjectsGetObjectNotFound(t *testing.T) {
	svc, server := doMockSetup(func(w http.ResponseWriter, r *http.Request) {
		doWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "obj-1", "hasStoredData": true},
			},
		})
	})
	defer server.Close()

	_, err := svc.GetObject(context.Background(), "ns-1", "does-not-exist")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if _, ok := err.(*R2NotFoundError); !ok {
		t.Errorf("expected R2NotFoundError, got %T: %v", err, err)
	}
}
