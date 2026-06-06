//go:build integration

package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

var kvIDCounter atomic.Int64

func TestKVNamespaceCRUD_Integration(t *testing.T) {
	namespaces := make(map[string]map[string]interface{})

	mux := http.NewServeMux()

	// List / Create namespaces
	mux.HandleFunc("/accounts/test-account-id/storage/kv/namespaces", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			result := make([]map[string]interface{}, 0, len(namespaces))
			for _, ns := range namespaces {
				result = append(result, ns)
			}
			writeCFJSONWithInfo(w, result, map[string]interface{}{
				"page":        1,
				"per_page":    20,
				"total_count": len(namespaces),
			})
		case http.MethodPost:
			var body struct {
				Title string `json:"title"`
			}
			if err := decodeJSON(r.Body, &body); err != nil {
				writeCFError(w, 400, "invalid body")
				return
			}
			id := fmt.Sprintf("ns-%d", kvIDCounter.Add(1))
			namespaces[id] = map[string]interface{}{
				"id":    id,
				"title": body.Title,
			}
			writeCFJSON(w, namespaces[id])
		}
	})

	// Single namespace operations
	mux.HandleFunc("/accounts/test-account-id/storage/kv/namespaces/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/accounts/test-account-id/storage/kv/namespaces/"):]
		// ID is the first path segment
		id := path
		if idx := indexOf(path, '/'); idx != -1 {
			id = path[:idx]
		}

		switch r.Method {
		case http.MethodGet:
			ns, exists := namespaces[id]
			if !exists {
				writeCFError(w, 404, "namespace not found")
				return
			}
			writeCFJSON(w, ns)
		case http.MethodDelete:
			if _, exists := namespaces[id]; !exists {
				writeCFError(w, 404, "namespace not found")
				return
			}
			delete(namespaces, id)
			writeCFJSON(w, nil)
		}
	})

	c, _ := newTestClientWithServer(t, mux)
	ctx := context.Background()

	kvs := &KVService{cf: c.cf, accountID: c.accountID}

	var createdID string

	t.Run("CreateNamespace", func(t *testing.T) {
		ns, err := kvs.CreateNamespace(ctx, "test-kv")
		if err != nil {
			t.Fatalf("CreateNamespace failed: %v", err)
		}
		if ns.Title != "test-kv" {
			t.Errorf("expected title 'test-kv', got %q", ns.Title)
		}
		if ns.ID == "" {
			t.Error("expected non-empty namespace ID")
		}
		createdID = ns.ID
	})

	t.Run("ListNamespaces", func(t *testing.T) {
		list, err := kvs.ListNamespaces(ctx)
		if err != nil {
			t.Fatalf("ListNamespaces failed: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 namespace, got %d", len(list))
		}
	})

	t.Run("GetNamespace", func(t *testing.T) {
		ns, err := kvs.GetNamespace(ctx, createdID)
		if err != nil {
			t.Fatalf("GetNamespace failed: %v", err)
		}
		if ns.ID != createdID {
			t.Errorf("expected ID %q, got %q", createdID, ns.ID)
		}
	})

	t.Run("DeleteNamespace", func(t *testing.T) {
		err := kvs.DeleteNamespace(ctx, createdID)
		if err != nil {
			t.Fatalf("DeleteNamespace failed: %v", err)
		}

		list, err := kvs.ListNamespaces(ctx)
		if err != nil {
			t.Fatalf("ListNamespaces after delete failed: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 namespaces after delete, got %d", len(list))
		}
	})
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
