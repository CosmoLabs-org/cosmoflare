package r2go2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func workerMockSetup(handler http.HandlerFunc) (*WorkerService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWorkerService(cf, "acct-worker-123")
	return svc, server
}

func workerWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewWorkerServiceValidation(t *testing.T) {
	_, err := NewWorkerService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewWorkerService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewWorkerService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and accountID are empty")
	}
}

func TestNewWorkerServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewWorkerService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestWorkerOptions(t *testing.T) {
	cfg := &workerConfig{}
	WithWorkerCompatibilityDate("2024-01-01")(cfg)
	if cfg.compatibilityDate != "2024-01-01" {
		t.Errorf("expected compatibilityDate=2024-01-01, got %s", cfg.compatibilityDate)
	}

	bindings := []WorkerBinding{
		{Name: "MY_KV", Type: "kv", ID: "ns-123"},
		{Name: "MY_R2", Type: "r2", ID: "my-bucket"},
	}
	WithWorkerBindings(bindings)(cfg)
	if len(cfg.bindings) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(cfg.bindings))
	}
	if cfg.bindings[0].Name != "MY_KV" {
		t.Errorf("expected first binding name=MY_KV, got %s", cfg.bindings[0].Name)
	}

	tags := []string{"production", "v2"}
	WithWorkerTags(tags)(cfg)
	if len(cfg.tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(cfg.tags))
	}

	WithWorkerModule(true)(cfg)
	if !cfg.module {
		t.Error("expected module=true")
	}
}

func TestLogOptions(t *testing.T) {
	cfg := &logConfig{}
	WithLogLimit(50)(cfg)
	if cfg.limit != 50 {
		t.Errorf("expected limit=50, got %d", cfg.limit)
	}

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	WithLogSince(since)(cfg)
	if !cfg.since.Equal(since) {
		t.Errorf("expected since=%v, got %v", since, cfg.since)
	}
}

func TestWorkerTypes(t *testing.T) {
	w := &Worker{
		Name:        "my-worker",
		Modified:    time.Now(),
		Size:        1024,
		Runtime:     "workers",
		Script:      "export default {}",
		Bindings:    []WorkerBinding{{Name: "KV", Type: "kv", ID: "ns-1"}},
		Tags:        []string{"prod"},
		CompatibilityDate: "2024-01-01",
	}
	if w.Name != "my-worker" {
		t.Errorf("unexpected name: %s", w.Name)
	}
	if w.Size != 1024 {
		t.Errorf("unexpected size: %d", w.Size)
	}
	if len(w.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(w.Bindings))
	}
	if len(w.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(w.Tags))
	}
}

func TestWorkerSettingsType(t *testing.T) {
	s := WorkerSettings{
		CompatibilityDate: "2024-01-01",
		UsageModel:        "bundled",
		Bindings:          []WorkerBinding{{Name: "D1", Type: "d1", ID: "db-1"}},
	}
	if s.CompatibilityDate != "2024-01-01" {
		t.Errorf("unexpected compatibility date: %s", s.CompatibilityDate)
	}
	if s.UsageModel != "bundled" {
		t.Errorf("unexpected usage model: %s", s.UsageModel)
	}
	if len(s.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(s.Bindings))
	}
}

func TestLogEntryType(t *testing.T) {
	e := &LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "uncaught exception",
		Event:     "exception",
	}
	if e.Level != "error" {
		t.Errorf("unexpected level: %s", e.Level)
	}
}

func TestToCFBindings(t *testing.T) {
	bindings := []WorkerBinding{
		{Name: "MY_KV", Type: "kv", ID: "ns-123"},
		{Name: "MY_R2", Type: "r2", ID: "my-bucket"},
		{Name: "MY_D1", Type: "d1", ID: "db-456"},
		{Name: "MY_QUEUE", Type: "queue", ID: "my-queue"},
		{Name: "MY_SERVICE", Type: "service", ID: "other-worker"},
		{Name: "MY_VAR", Type: "plain_text", ID: "hello"},
		{Name: "MY_SECRET", Type: "secret_text", ID: "s3cret"},
	}

	result := toCFBindings(bindings)
	if len(result) != 7 {
		t.Fatalf("expected 7 bindings, got %d", len(result))
	}

	for name, binding := range result {
		switch name {
		case "MY_KV":
			if _, ok := binding.(cloudflare.WorkerKvNamespaceBinding); !ok {
				t.Errorf("expected WorkerKvNamespaceBinding for %s", name)
			}
		case "MY_R2":
			if _, ok := binding.(cloudflare.WorkerR2BucketBinding); !ok {
				t.Errorf("expected WorkerR2BucketBinding for %s", name)
			}
		case "MY_D1":
			if _, ok := binding.(cloudflare.WorkerD1DatabaseBinding); !ok {
				t.Errorf("expected WorkerD1DatabaseBinding for %s", name)
			}
		case "MY_QUEUE":
			if _, ok := binding.(cloudflare.WorkerQueueBinding); !ok {
				t.Errorf("expected WorkerQueueBinding for %s", name)
			}
		case "MY_SERVICE":
			if _, ok := binding.(cloudflare.WorkerServiceBinding); !ok {
				t.Errorf("expected WorkerServiceBinding for %s", name)
			}
		case "MY_VAR":
			if _, ok := binding.(cloudflare.WorkerPlainTextBinding); !ok {
				t.Errorf("expected WorkerPlainTextBinding for %s", name)
			}
		case "MY_SECRET":
			if _, ok := binding.(cloudflare.WorkerSecretTextBinding); !ok {
				t.Errorf("expected WorkerSecretTextBinding for %s", name)
			}
		}
	}
}

func TestToCFBindingsEmpty(t *testing.T) {
	result := toCFBindings(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 bindings for nil input, got %d", len(result))
	}
}

func TestWorkerDeployValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Deploy(nil, "", nil)
	if err == nil {
		t.Error("expected error when name is empty")
	}
	if !strings.Contains(err.Error(), "worker name is required") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = svc.Deploy(nil, "my-worker", nil)
	if err == nil {
		t.Error("expected error when script is nil")
	}
	if !strings.Contains(err.Error(), "script content is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorkerGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Get(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.Delete(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerLogsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Logs(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerUpdateSettingsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.UpdateSettings(nil, "", WorkerSettings{})
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

// --- httptest-based API mock tests ---

func TestWorkerDeployWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		now := time.Now()
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":          "my-worker",
				"script":      "export default { fetch() {} }",
				"size":        1024,
				"modified_on": now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	worker, err := svc.Deploy(ctx, "my-worker", strings.NewReader("export default { fetch() {} }"),
		WithWorkerCompatibilityDate("2024-01-01"),
		WithWorkerModule(true),
		WithWorkerTags([]string{"production"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if worker.Name != "my-worker" {
		t.Errorf("expected Name=my-worker, got %s", worker.Name)
	}
}

func TestWorkerListWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		now := time.Now()
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "worker-1", "modified_on": now.Format(time.RFC3339), "size": 2048},
				{"id": "worker-2", "modified_on": now.Format(time.RFC3339), "size": 4096},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	workers, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workers) != 2 {
		t.Fatalf("expected 2 workers, got %d", len(workers))
	}
	if workers[0].Name != "worker-1" {
		t.Errorf("expected first Name=worker-1, got %s", workers[0].Name)
	}
	if workers[1].Size != 4096 {
		t.Errorf("expected second Size=4096, got %d", workers[1].Size)
	}
}

func TestWorkerGetWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		// GetWorker reads the raw response body as the script content
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte("export default {}"))
	})
	defer server.Close()

	ctx := context.Background()
	worker, err := svc.Get(ctx, "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if worker.Name != "my-worker" {
		t.Errorf("expected Name=my-worker, got %s", worker.Name)
	}
	if worker.Script != "export default {}" {
		t.Errorf("expected Script content, got %s", worker.Script)
	}
}

func TestWorkerGetNotFound(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "worker not found"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestWorkerDeleteWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "worker-del"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Delete(ctx, "worker-del")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkerLogsWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"modified_on": now.Format(time.RFC3339),
				"size":        1024,
			},
			"meta": map[string]interface{}{
				"modified_on": now.Format(time.RFC3339),
				"size":        1024,
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	entries, err := svc.Logs(ctx, "my-worker", WithLogLimit(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) < 1 {
		t.Fatal("expected at least 1 log entry")
	}
	if entries[0].Level != "info" {
		t.Errorf("expected Level=info, got %s", entries[0].Level)
	}
}

func TestWorkerUpdateSettingsWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.UpdateSettings(ctx, "my-worker", WorkerSettings{
		CompatibilityDate: "2024-01-01",
		Bindings: []WorkerBinding{
			{Name: "MY_KV", Type: "kv", ID: "ns-123"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkerDeployAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid script"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Deploy(ctx, "bad-worker", strings.NewReader("bad script"))
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
}

func TestWorkerJSONMarshal(t *testing.T) {
	w := &Worker{
		Name: "json-worker", Size: 2048, Runtime: "workers",
		Bindings:    []WorkerBinding{{Name: "KV", Type: "kv", ID: "ns-1"}},
		Tags:        []string{"prod"},
		CompatibilityDate: "2024-01-01",
	}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Worker
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Name != "json-worker" {
		t.Errorf("Name mismatch: got %q", decoded.Name)
	}
	if len(decoded.Bindings) != 1 {
		t.Errorf("Bindings count mismatch: got %d", len(decoded.Bindings))
	}
}

func TestWorkerNewFromCredsSuccess(t *testing.T) {
	svc, err := NewWorkerServiceFromCreds("acct123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

// --- TailLogs tests ---

func TestTailLogsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.TailLogs(context.Background(), "", nil)
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestTailLogsReturnsChannel(t *testing.T) {
	callCount := 0
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"modified_on": time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := svc.TailLogs(ctx, "my-worker", &TailOptions{
		Interval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Should receive at least one entry before context cancels
	select {
	case entry, ok := <-ch:
		if !ok {
			t.Fatal("channel closed without sending entries")
		}
		if entry.Message == "" {
			t.Error("expected non-empty message")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for log entry")
	}
}

func TestTailLogsStopsOnContextCancel(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"modified_on": time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := svc.TailLogs(ctx, "my-worker", &TailOptions{
		Interval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read one entry
	<-ch

	// Cancel context
	cancel()

	// Channel should close soon
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // channel closed, test passes
			}
		case <-timeout:
			t.Fatal("channel did not close after context cancel")
		}
	}
}

func TestTailLogsLevelFilter(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"modified_on": time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := svc.TailLogs(ctx, "my-worker", &TailOptions{
		Interval: 100 * time.Millisecond,
		Level:    "error",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The metadata entries are "info" level, so with "error" filter we should
	// get no entries. Channel closes when context expires.
	select {
	case entry, ok := <-ch:
		if ok {
			t.Errorf("expected no entries with level=error filter, got: %+v", entry)
		}
	case <-ctx.Done():
		// Expected — no entries matched the filter
	}
}

func TestTailOptionsDefaults(t *testing.T) {
	opts := &TailOptions{}
	if opts.Interval != 0 {
		t.Errorf("expected zero interval before defaults applied, got %v", opts.Interval)
	}
}
