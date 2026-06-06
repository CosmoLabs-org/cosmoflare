//go:build integration

package cosmoflare

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestWorkersCRUD_Integration(t *testing.T) {
	workers := make(map[string]map[string]interface{})

	mux := http.NewServeMux()

	// List workers
	mux.HandleFunc("/accounts/test-account-id/workers/scripts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeCFError(w, 405, "method not allowed")
			return
		}
		result := make([]map[string]interface{}, 0, len(workers))
		for name, wk := range workers {
			wk["id"] = name
			result = append(result, wk)
		}
		writeCFJSON(w, result)
	})

	// Single worker operations
	mux.HandleFunc("/accounts/test-account-id/workers/scripts/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/accounts/test-account-id/workers/scripts/"):]
		switch r.Method {
		case http.MethodPut:
			workers[name] = map[string]interface{}{
				"id":          name,
				"modified_on": time.Now().Format(time.RFC3339),
				"size":        1024,
			}
			writeCFJSON(w, workers[name])
		case http.MethodGet:
			wk, exists := workers[name]
			if !exists {
				writeCFError(w, 404, "worker not found")
				return
			}
			writeCFJSON(w, wk)
		case http.MethodDelete:
			if _, exists := workers[name]; !exists {
				writeCFError(w, 404, "worker not found")
				return
			}
			delete(workers, name)
			writeCFJSON(w, nil)
		}
	})

	c, _ := newTestClientWithServer(t, mux)
	ctx := context.Background()

	ws := &WorkerService{cf: c.cf, accountID: c.accountID}

	t.Run("ListEmpty", func(t *testing.T) {
		list, err := ws.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 workers, got %d", len(list))
		}
	})

	t.Run("Deploy", func(t *testing.T) {
		script := "addEventListener('fetch', e => e.respondWith(new Response('ok')))"
		worker, err := ws.Deploy(ctx, "my-worker", strReader(script))
		if err != nil {
			t.Fatalf("Deploy failed: %v", err)
		}
		if worker.Name != "my-worker" {
			t.Errorf("expected worker name 'my-worker', got %q", worker.Name)
		}
	})

	t.Run("List", func(t *testing.T) {
		list, err := ws.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 worker, got %d", len(list))
		}
	})

	t.Run("Get", func(t *testing.T) {
		worker, err := ws.Get(ctx, "my-worker")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if worker.Name != "my-worker" {
			t.Errorf("expected 'my-worker', got %q", worker.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := ws.Delete(ctx, "my-worker")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		list, err := ws.List(ctx)
		if err != nil {
			t.Fatalf("List after delete failed: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 workers after delete, got %d", len(list))
		}
	})
}
